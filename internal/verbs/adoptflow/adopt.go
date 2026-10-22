// Package adoptflow registers the interactive, provider-assisted root form.
// It is deliberately separate from internal/verbs/adopt, whose plumbing
// command only reads the shared flow document.
package adoptflow

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/asset"
	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
	adoptverb "github.com/procrastivity/lds/internal/verbs/adopt"
)

const apiKeyEnv = "OPENCODE_API_KEY"

type workflowResult struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Operation     string                `json:"operation"`
	Status        string                `json:"status"`
	TargetRoot    string                `json:"targetRoot"`
	FlowAsset     flowAsset             `json:"flowAsset"`
	Advisory      string                `json:"advisory"`
	Inventory     adoption.Inventory    `json:"inventory"`
	Validation    adoption.Validation   `json:"validation"`
	Plan          adoption.PlanResult   `json:"plan"`
	Apply         *adoption.ApplyResult `json:"apply"`
}

type flowAsset struct {
	RelativeName string `json:"relativeName"`
	Source       string `json:"source"`
	SHA256       string `json:"sha256"`
}

// Command constructs the human-facing, provider-assisted root `lds adopt`.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adopt <root>",
		Short: "review and create the four approved documentation outputs",
		Long: "Interactively review create-only documentation adoption for an explicit target. " +
			"Requires a complete owner-supplied --facts JSON bundle and a terminal. " +
			"The live flow and supplied facts plus scoped inventory/link metadata are sent to OpenCode Go " +
			"(deepseek-v4-flash); target source files are not sent. The model has no tools and cannot approve or write. " +
			"The complete deterministic plan is displayed, and only a fresh exact-digest human decision can authorize creates.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractive(cmd.InOrStdin(), streams.Err) {
				return ldserr.New("refusal.interactive-required", "lds adopt requires an interactive terminal; no provider request or target write was made")
			}
			key, ok := os.LookupEnv(apiKeyEnv)
			if !ok || key == "" {
				return ldserr.New("refusal.provider-credential-unavailable", "selected OpenCode Go credential is unavailable; no provider request or target write was made")
			}
			factsPath, _ := cmd.Flags().GetString("facts")
			return runWorkflow(cmd.Context(), args[0], factsPath, key, cmd.InOrStdin(), streams,
				cliflags.FromContext(cmd.Context()).JSON, true, requestAdvisory)
		},
	}
	cmd.Flags().String("facts", "", "path to the complete owner-supplied JSON fact bundle")
	if err := cmd.MarkFlagRequired("facts"); err != nil {
		panic(err)
	}
	surface.Annotate(cmd, surface.LLM)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","targetRoot","flowAsset","advisory","inventory","validation","plan","apply"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"adopt"},"status":{"enum":["applied","current","refused","partial"]},"targetRoot":{"type":"string"},"flowAsset":{"type":"object","required":["relativeName","source","sha256"]},"advisory":{"type":"string"},"inventory":{"type":"object","required":["files","links","skipped"]},"validation":{"type":"object","required":["findings","skipped"]},"plan":{"type":"object","required":["planDigest","outputs","veto"]},"apply":{"type":["object","null"]}}}`))
	return cmd
}

func runWorkflow(ctx context.Context, root, factsPath, key string, input io.Reader, streams *iostreams.Streams, jsonOut, interactive bool, advise providerCall) error {
	return runWorkflowWithApprovalInput(ctx, root, factsPath, key, input, streams, jsonOut, interactive, advise, func() (*bufio.Reader, error) {
		return freshApprovalReader(input)
	})
}

func runWorkflowWithApprovalInput(ctx context.Context, root, factsPath, key string, input io.Reader, streams *iostreams.Streams, jsonOut, interactive bool, advise providerCall, approvalInput func() (*bufio.Reader, error)) error {
	if !interactive {
		return ldserr.New("refusal.interactive-required", "lds adopt requires an interactive terminal; no provider request or target write was made")
	}
	if key == "" {
		return ldserr.New("refusal.provider-credential-unavailable", "selected OpenCode Go credential is unavailable; no provider request or target write was made")
	}
	if input == nil || streams == nil || streams.Out == nil || streams.Err == nil {
		return ldserr.New("internal.adopt-stream-unavailable", "interactive adoption streams are unavailable")
	}
	reader := bufio.NewReader(input)

	projectRoot := cliflags.FromContext(ctx).ProjectRoot
	document, err := adoptverb.ResolveDocument(projectRoot)
	if err != nil {
		var structured *ldserr.Error
		if errors.As(err, &structured) && structured.Code == "refusal.project-asset" {
			return structured
		}
		if errors.Is(err, adoptverb.ErrInvalidDocument) {
			return ldserr.New("validation.adopt-document-invalid", err.Error())
		}
		return ldserr.New("internal.adopt-document-unavailable", err.Error())
	}
	facts, err := os.ReadFile(factsPath)
	if err != nil {
		return ldserr.New("validation.render-invalid", "owner-supplied facts file cannot be read")
	}
	inventory, err := adoption.Inspect(root)
	if err != nil {
		return targetReadError("inventory", err)
	}
	validation, err := adoption.Validate(root)
	if err != nil {
		return targetReadError("validate", err)
	}
	if _, err := adoption.Render(root, factsPath, projectRoot); err != nil {
		if errors.Is(err, adoption.ErrProjectAsset) {
			return ldserr.New("refusal.project-asset", err.Error())
		}
		if errors.Is(err, adoption.ErrInvalidRenderInput) {
			return ldserr.New("validation.render-invalid", err.Error())
		}
		if errors.Is(err, adoption.ErrUnsafeTarget) {
			return ldserr.New("refusal.unsafe-target", err.Error())
		}
		if errors.Is(err, os.ErrNotExist) {
			return ldserr.New("validation.target-root-missing", err.Error())
		}
		return ldserr.New("internal.render-failed", err.Error())
	}
	plan, planErr := adoption.Plan(root, factsPath, projectRoot)
	if planErr != nil {
		var refusal *adoption.PlanRefusal
		if !errors.As(planErr, &refusal) {
			if errors.Is(planErr, adoption.ErrProjectAsset) {
				return ldserr.New("refusal.project-asset", planErr.Error())
			}
			if errors.Is(planErr, adoption.ErrInvalidRenderInput) {
				return ldserr.New("validation.render-invalid", planErr.Error())
			}
			if errors.Is(planErr, adoption.ErrUnsafeTarget) {
				return ldserr.New("refusal.unsafe-target", planErr.Error())
			}
			if errors.Is(planErr, os.ErrNotExist) {
				return ldserr.New("validation.target-root-missing", planErr.Error())
			}
			return ldserr.New("internal.plan-failed", planErr.Error())
		}
		result := makeWorkflowResult(root, document, inventory, validation, plan, "", nil)
		result.Status = "refused"
		planBytes, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return ldserr.New("internal.adopt-plan-encode", "could not encode the complete refused plan")
		}
		if err := presentPlan(streams.Err, planBytes); err != nil {
			return ldserr.New("internal.adopt-output-failed", err.Error())
		}
		if err := emitResult(streams.Out, result, jsonOut); err != nil {
			return ldserr.New("internal.adopt-output-failed", err.Error())
		}
		return ldserr.New(refusal.Code, refusal.Message)
	}
	factsDigest := sha256.Sum256(facts)
	if plan.FactsDigest != "sha256:"+hex.EncodeToString(factsDigest[:]) {
		return ldserr.New("refusal.plan-stale", "owner-supplied facts changed during review; start a fresh adoption review")
	}
	currentDocument, err := adoptverb.ResolveDocument(projectRoot)
	if err != nil || currentDocument.Source != document.Source || currentDocument.Path != document.Path || !bytes.Equal(currentDocument.Bytes(), document.Bytes()) {
		return ldserr.New("refusal.flow-changed", "the live adoption flow changed during review; start a fresh adoption review")
	}
	if plan.Veto || plan.PlanDigest == nil {
		return ldserr.New("refusal.plan-ineligible", "the deterministic plan is not eligible for application")
	}
	planBytes, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return ldserr.New("internal.adopt-plan-encode", "could not encode the complete plan for review")
	}

	evidence := aggregateEvidence(inventory, validation, plan)
	if _, err := fmt.Fprintln(streams.Err, "This request sends the live flow, owner-supplied facts, and bounded inventory/link metadata to OpenCode Go. No target source-file contents are sent. Type exactly `send` to continue:"); err != nil {
		return ldserr.New("internal.adopt-output-failed", err.Error())
	}
	if !readExactLine(reader, "send") {
		return ldserr.New("refusal.provider-transfer-declined", "provider transfer was declined; the target was not changed")
	}
	advisory, err := advise(ctx, key, string(document.Bytes()), facts, evidence)
	if err != nil {
		return ldserr.New("internal.adopt-provider-failed", "selected OpenCode Go request failed or returned an unexpected response; no target write was made")
	}
	if !safeAdvisory(advisory) {
		return ldserr.New("internal.adopt-provider-failed", "selected OpenCode Go returned unsafe advisory text; no target write was made")
	}
	if err := presentAdvisory(streams.Err, advisory); err != nil {
		return ldserr.New("internal.adopt-output-failed", err.Error())
	}
	if err := presentPlan(streams.Err, planBytes); err != nil {
		return ldserr.New("internal.adopt-output-failed", err.Error())
	}
	approvalReader, err := approvalInput()
	if err != nil {
		return ldserr.New("refusal.approval-input-unavailable", "fresh terminal input could not be established after displaying the plan; no target write was made")
	}
	if _, err := fmt.Fprintf(streams.Err, "To authorize only this displayed plan, type exactly: apply %s\n", *plan.PlanDigest); err != nil {
		return ldserr.New("internal.adopt-output-failed", err.Error())
	}
	if !readExactLine(approvalReader, "apply "+*plan.PlanDigest) {
		result := makeWorkflowResult(root, document, inventory, validation, plan, advisory, nil)
		result.Status = "refused"
		if err := emitResult(streams.Out, result, jsonOut); err != nil {
			return ldserr.New("internal.adopt-output-failed", err.Error())
		}
		return ldserr.New("refusal.confirmation-declined", "fresh exact-digest confirmation was not received; no target write was made")
	}
	currentDocument, err = adoptverb.ResolveDocument(projectRoot)
	if err != nil || currentDocument.Source != document.Source || currentDocument.Path != document.Path || !bytes.Equal(currentDocument.Bytes(), document.Bytes()) {
		return ldserr.New("refusal.flow-changed", "the live adoption flow changed after review; no target write was made; start a fresh adoption review")
	}

	applyResult, applyErr := adoption.ApplyReviewed(root, factsPath, planBytes, *plan.PlanDigest, projectRoot)
	result := makeWorkflowResult(root, document, inventory, validation, plan, advisory, &applyResult)
	result.Status = applyResult.Status
	if err := emitResult(streams.Out, result, jsonOut); err != nil {
		return ldserr.New("internal.adopt-output-failed", err.Error())
	}
	if applyErr == nil {
		return nil
	}
	var structured *adoption.ApplyError
	if errors.As(applyErr, &structured) {
		return ldserr.New(structured.Code, structured.Message)
	}
	return ldserr.New("internal.apply-failed", "application stopped unexpectedly; inspect observed result and re-run preflight")
}

func targetReadError(operation string, err error) *ldserr.Error {
	switch {
	case errors.Is(err, adoption.ErrUnsafeTarget):
		return ldserr.New("refusal.unsafe-target", err.Error())
	case errors.Is(err, os.ErrNotExist):
		return ldserr.New("validation.target-root-missing", err.Error())
	default:
		return ldserr.New("internal."+operation+"-failed", err.Error())
	}
}

func aggregateEvidence(inventory adoption.Inventory, validation adoption.Validation, plan adoption.PlanResult) providerEvidence {
	classes := map[string]int{
		"create": 0, "different-collision": 0, "identical-untouched": 0, "refusal": 0,
	}
	for _, output := range plan.Outputs {
		classes[output.Classification]++
	}
	return providerEvidence{
		InventoryFiles: len(inventory.Files), InventoryLinks: len(inventory.Links),
		InventorySkipped: len(inventory.Skipped), ValidationFindings: len(validation.Findings),
		ValidationSkipped: len(validation.Skipped), PlanFindings: len(plan.Findings),
		PlanSkipped: len(plan.Skipped), OutputClassifications: classes, PlanVeto: plan.Veto,
	}
}

func makeWorkflowResult(root string, document asset.Resolved, inventory adoption.Inventory, validation adoption.Validation, plan adoption.PlanResult, advisory string, apply *adoption.ApplyResult) workflowResult {
	assetBytes := document.Bytes()
	sum := sha256.Sum256(assetBytes)
	return workflowResult{
		SchemaVersion: 1, Operation: "adopt", Status: "reviewed", TargetRoot: root,
		FlowAsset: flowAsset{RelativeName: "flows/adopt.md", Source: document.Source.String(), SHA256: hex.EncodeToString(sum[:])},
		Advisory:  advisory, Inventory: inventory, Validation: validation, Plan: plan, Apply: apply,
	}
}

func presentAdvisory(w io.Writer, advisory string) error {
	if _, err := fmt.Fprintln(w, "OpenCode Go advisory only — not verified facts, a decision, or approval:"); err != nil {
		return err
	}
	for _, line := range strings.Split(advisory, "\n") {
		if _, err := fmt.Fprintf(w, "  | %s\n", line); err != nil {
			return err
		}
	}
	return nil
}

func presentPlan(w io.Writer, data []byte) error {
	if _, err := fmt.Fprintln(w, "Complete deterministic plan (review every field; no writes have occurred):"); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, string(data))
	return err
}

func emitResult(w io.Writer, result workflowResult, jsonOut bool) error {
	if jsonOut {
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(data))
		return err
	}
	if result.Apply == nil {
		if result.Plan.PlanDigest == nil {
			_, err := fmt.Fprintln(w, "Adoption plan refused; no files were created. Plan digest: unavailable.")
			return err
		}
		_, err := fmt.Fprintf(w, "Adoption plan refused; no files were created. Plan digest: %s\n", *result.Plan.PlanDigest)
		return err
	}
	_, err := fmt.Fprintf(w, "Adoption %s; %d file(s) created, %d identical and untouched. Plan: %s\n", result.Apply.Status, len(result.Apply.Created), len(result.Apply.IdenticalUntouched), result.Apply.PlanDigest)
	return err
}

func readExactLine(input *bufio.Reader, expected string) bool {
	line, err := input.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line == expected
}
