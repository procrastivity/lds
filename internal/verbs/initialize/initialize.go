// Package initialize hosts provider-free, human-reviewed documentation setup.
package initialize

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

type workflowResult struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Operation     string                `json:"operation"`
	Status        string                `json:"status"`
	Plan          adoption.PlanResult   `json:"plan"`
	Apply         *adoption.ApplyResult `json:"apply"`
}

// Command reviews the existing create-only plan without model assistance.
// It does not change the adoption contract or add another write boundary.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <root>",
		Short: "review and initialize two-track documentation offline without overwriting files",
		Long: "Initialize an existing explicit project root with the four adoption outputs using a complete owner-approved --facts bundle. " +
			"Requires a Linux terminal: displays the full deterministic plan and accepts only fresh apply-digest confirmation. " +
			"No provider, credentials, inferred facts, root README edits, migrations, retries or noninteractive approval bypass. " +
			"Identical outputs remain untouched; any differing occupied output vetoes all creates. Late failure may be partial without rollback.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			streams := &iostreams.Streams{In: cmd.InOrStdin(), Out: streams.Out, Err: streams.Err}
			if !streams.IsInteractive() {
				return ldserr.New("refusal.interactive-required", "lds init requires a Linux terminal; no target write was made")
			}
			facts, _ := cmd.Flags().GetString("facts")
			flags := cliflags.FromContext(cmd.Context())
			return runWorkflow(args[0], facts, flags.ProjectRoot, streams, flags.JSON)
		},
	}
	cmd.Flags().String("facts", "", "path to the complete owner-approved adoption JSON fact bundle")
	if err := cmd.MarkFlagRequired("facts"); err != nil {
		panic(err)
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","plan","apply"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"init"},"status":{"enum":["applied","current","refused","partial"]},"plan":{"type":"object","required":["operation","planDigest","outputs","veto"],"properties":{"operation":{"const":"docs plan"}}},"apply":{"type":["object","null"]}}}`))
	return cmd
}

func runWorkflow(root, facts, projectRoot string, streams *iostreams.Streams, jsonOut bool) error {
	plan, planErr := adoption.Plan(root, facts, projectRoot)
	if plan.SchemaVersion != 1 {
		return workflowError(planErr)
	}
	result := workflowResult{SchemaVersion: 1, Operation: "init", Status: "refused", Plan: plan}
	displayed, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(streams.Err, "Complete deterministic initialization plan (review every field; no writes have occurred):\n%s\n", displayed); err != nil {
		return err
	}
	finish := func(outcome error) error {
		if err := writeResult(streams, result, jsonOut); err != nil {
			return err
		}
		return workflowError(outcome)
	}
	if planErr != nil {
		return finish(planErr)
	}
	reader, err := streams.FreshInput()
	if err != nil {
		return finish(ldserr.New("refusal.approval-input-unavailable", "fresh terminal input could not be established after displaying the plan; no target write was made"))
	}
	if _, err := fmt.Fprintf(streams.Err, "To authorize only this displayed plan, type exactly: apply %s\nType decline (without a digest), or anything else, to refuse.\n", *plan.PlanDigest); err != nil {
		return err
	}
	line, err := reader.ReadString('\n')
	if err != nil || strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r") != "apply "+*plan.PlanDigest {
		return finish(ldserr.New("refusal.confirmation-declined", "fresh exact-digest confirmation was not received; no target write was made"))
	}
	applied, applyErr := adoption.ApplyReviewed(root, facts, displayed, *plan.PlanDigest, projectRoot)
	result.Apply, result.Status = &applied, applied.Status
	return finish(applyErr)
}

func writeResult(streams *iostreams.Streams, result workflowResult, jsonOut bool) error {
	var err error
	if jsonOut {
		err = json.NewEncoder(streams.Out).Encode(result)
	} else {
		created, untouched := 0, 0
		if result.Apply != nil {
			created, untouched = len(result.Apply.Created), len(result.Apply.IdenticalUntouched)
		}
		_, err = fmt.Fprintf(streams.Out, "init %s; %d created, %d identical untouched\n", result.Status, created, untouched)
	}
	if err != nil {
		return ldserr.New("internal.init-output", "could not write the final result; inspect the target before considering another initialization")
	}
	return nil
}

func workflowError(err error) error {
	if err == nil {
		return nil
	}
	var structured *ldserr.Error
	var refusal *adoption.PlanRefusal
	var applyErr *adoption.ApplyError
	switch {
	case errors.As(err, &structured):
		return structured
	case errors.As(err, &refusal):
		return ldserr.New(refusal.Code, refusal.Message)
	case errors.As(err, &applyErr):
		return ldserr.New(applyErr.Code, applyErr.Message)
	case errors.Is(err, adoption.ErrProjectAsset):
		return ldserr.New("refusal.project-asset", err.Error())
	case errors.Is(err, adoption.ErrInvalidRenderInput):
		return ldserr.New("validation.render-invalid", err.Error())
	case errors.Is(err, adoption.ErrUnsafeTarget):
		return ldserr.New("refusal.unsafe-target", err.Error())
	case errors.Is(err, os.ErrNotExist):
		return ldserr.New("validation.target-root-missing", err.Error())
	default:
		return ldserr.New("internal.init-failed", err.Error())
	}
}
