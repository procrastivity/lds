// Package update hosts human-reviewed existing-file updates.
package update

import (
	"encoding/json"
	"fmt"
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
	SchemaVersion int                    `json:"schemaVersion"`
	Operation     string                 `json:"operation"`
	Status        string                 `json:"status"`
	Plan          *adoption.UpdatePlan   `json:"plan"`
	Apply         *adoption.UpdateResult `json:"apply"`
	FlowAsset     *flowAsset             `json:"flowAsset,omitempty"`
	Provider      string                 `json:"provider,omitempty"`
	Model         string                 `json:"model,omitempty"`
	Draft         *adoption.UpdateDraft  `json:"draft,omitempty"`
}

// Command displays the full deterministic plan and requires fresh terminal
// approval before passing the exact displayed artifact to the update boundary.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "update <root>", Short: "review exact documentation replacements, offline or with explicit drafting assistance",
		Long: "Default mode reviews authored replacements against the original docs collect packet without credentials or a model. Explicit --assist mode asks fresh send consent for one OpenCode Go draft restricted to --document paths. Both require Linux, a terminal and separate fresh apply-digest approval. No creates, retries or noninteractive approval bypass.",
		Args: updateArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			streams := &iostreams.Streams{In: cmd.InOrStdin(), Out: streams.Out, Err: streams.Err}
			if !streams.IsInteractive() {
				return ldserr.New("refusal.interactive-required", "lds update requires a Linux terminal; no target write was made")
			}
			assist, _ := cmd.Flags().GetBool("assist")
			if assist {
				return runAssistedCommand(cmd, args[0], streams, cliflags.FromContext(cmd.Context()).JSON)
			}
			observed, _ := cmd.Flags().GetString("observed")
			request, _ := cmd.Flags().GetString("request")
			return runWorkflow(args[0], observed, request, streams, cliflags.FromContext(cmd.Context()).JSON)
		},
	}
	cmd.Flags().String("observed", "", "path to the unchanged original docs collect JSON packet")
	cmd.Flags().String("request", "", "path to schema-1 caller-authored full-content edit requests")
	cmd.Flags().Bool("assist", false, "opt into one consent-gated model draft, not replacement approval")
	cmd.Flags().String("reason", "", "explicit review question for assisted mode")
	cmd.Flags().String("audience", "", "intended readers and task for assisted mode")
	cmd.Flags().StringArray("document", nil, "explicit existing relative Markdown path eligible for a draft (repeatable)")
	cmd.Flags().StringArray("source", nil, "explicit relative UTF-8 source file to disclose (repeatable)")
	surface.Annotate(cmd, surface.LLM)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","plan","apply"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"update"},"status":{"enum":["applied","refused","partial","indeterminate","review-required"]},"plan":{"type":["object","null"]},"apply":{"type":["object","null"]},"flowAsset":{"type":"object"},"provider":{"const":"OpenCode Go"},"model":{"const":"deepseek-v4-flash"},"draft":{"type":"object"}}}`))
	return cmd
}

func runWorkflow(root, observed, request string, streams *iostreams.Streams, jsonOut bool) error {
	plan, planErr := adoption.PlanUpdates(root, observed, request)
	if plan.SchemaVersion != 1 {
		return planErr
	}
	return reviewUpdate(root, workflowResult{SchemaVersion: 1, Operation: "update", Status: "refused", Plan: &plan}, planErr, streams, jsonOut, nil)
}

// reviewUpdate shares the same write approval for authored and drafted edits.
func reviewUpdate(root string, result workflowResult, planErr error, streams *iostreams.Streams, jsonOut bool, recheck func() error) error {
	plan := result.Plan
	displayed, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(streams.Err, "Complete deterministic update plan (review every field; no writes have occurred):\n%s\n", displayed); err != nil {
		return err
	}
	finish := func(outcome error) error {
		if err := writeResult(streams, result, jsonOut); err != nil {
			return err
		}
		return outcome
	}
	if planErr != nil {
		return finish(planErr)
	}
	reader, err := streams.FreshInput()
	if err != nil {
		return finish(ldserr.New("refusal.approval-input-unavailable", "fresh terminal input could not be established after displaying the plan; no target write was made"))
	}
	if _, err := fmt.Fprintf(streams.Err, "To authorize only this displayed plan, type exactly: apply %s\nType decline (without a digest), or anything else, to refuse.\n", plan.PlanDigest); err != nil {
		return err
	}
	line, err := reader.ReadString('\n')
	if err != nil || strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r") != "apply "+plan.PlanDigest {
		return finish(ldserr.New("refusal.confirmation-declined", "fresh exact-digest confirmation was not received; no target write was made"))
	}
	if recheck != nil {
		if err := recheck(); err != nil {
			return finish(err)
		}
	}
	applied, applyErr := adoption.ApplyUpdatesReviewed(root, displayed, plan.PlanDigest)
	result.Apply, result.Status = &applied, applied.Status
	return finish(applyErr)
}

func writeResult(streams *iostreams.Streams, result workflowResult, jsonOut bool) error {
	var err error
	if jsonOut {
		err = json.NewEncoder(streams.Out).Encode(result)
	} else if result.Plan == nil {
		_, err = fmt.Fprintln(streams.Out, "update review-required; no replacements proposed; questions remain unverified advice")
	} else {
		replaced := 0
		if result.Apply != nil {
			replaced = len(result.Apply.Replaced)
		}
		_, err = fmt.Fprintf(streams.Out, "update %s; %d replaced; plan %s\n", result.Status, replaced, result.Plan.PlanDigest)
	}
	if err != nil {
		return ldserr.New("internal.update-output", "could not write the final result; inspect the target before considering another update")
	}
	return nil
}
