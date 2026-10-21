// Package update hosts offline, human-reviewed existing-file updates.
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
	Plan          adoption.UpdatePlan    `json:"plan"`
	Apply         *adoption.UpdateResult `json:"apply"`
}

// Command displays the full deterministic plan and requires fresh terminal
// approval before passing the exact displayed artifact to the update boundary.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "update <root>", Short: "review and confirm explicit existing-file documentation updates offline",
		Long: "Review caller-authored full-content replacements against the original docs collect packet. Requires Linux and a terminal. Displays the complete deterministic plan; only fresh `apply sha256:<displayed digest>` approval authorizes the existing update boundary. No model, credentials, creates, automatic retries or noninteractive approval bypass.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			streams := &iostreams.Streams{In: cmd.InOrStdin(), Out: streams.Out, Err: streams.Err}
			if !streams.IsInteractive() {
				return ldserr.New("refusal.interactive-required", "lds update requires a Linux terminal; no target write was made")
			}
			observed, _ := cmd.Flags().GetString("observed")
			request, _ := cmd.Flags().GetString("request")
			return runWorkflow(args[0], observed, request, streams, cliflags.FromContext(cmd.Context()).JSON)
		},
	}
	cmd.Flags().String("observed", "", "path to the unchanged original docs collect JSON packet")
	cmd.Flags().String("request", "", "path to schema-1 caller-authored full-content edit requests")
	for _, name := range []string{"observed", "request"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","plan","apply"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"update"},"status":{"enum":["applied","refused","partial","indeterminate"]},"plan":{"type":"object"},"apply":{"type":["object","null"]}}}`))
	return cmd
}

func runWorkflow(root, observed, request string, streams *iostreams.Streams, jsonOut bool) error {
	plan, planErr := adoption.PlanUpdates(root, observed, request)
	if plan.SchemaVersion != 1 {
		return planErr
	}
	displayed, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(streams.Err, "Complete deterministic update plan (review every field; no writes have occurred):\n%s\n", displayed); err != nil {
		return err
	}
	result := workflowResult{SchemaVersion: 1, Operation: "update", Status: "refused", Plan: plan}
	finish := func(outcome error) error {
		if jsonOut {
			err = json.NewEncoder(streams.Out).Encode(result)
		} else {
			replaced := 0
			if result.Apply != nil {
				replaced = len(result.Apply.Replaced)
			}
			_, err = fmt.Fprintf(streams.Out, "update %s; %d replaced; plan %s\n", result.Status, replaced, plan.PlanDigest)
		}
		if err != nil {
			return ldserr.New("internal.update-output", "could not write the final result; inspect the target before considering another update")
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
	applied, applyErr := adoption.ApplyUpdatesReviewed(root, displayed, plan.PlanDigest)
	result.Apply, result.Status = &applied, applied.Status
	return finish(applyErr)
}
