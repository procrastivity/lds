package docs

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

func updatePlanCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "update-plan <root>", Short: "review exact replacements of explicitly selected existing Markdown files",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			observed, _ := cmd.Flags().GetString("observed")
			request, _ := cmd.Flags().GetString("request")
			plan, planErr := adoption.PlanUpdates(args[0], observed, request)
			if plan.SchemaVersion == 1 {
				if cliflags.FromContext(cmd.Context()).JSON {
					if err := json.NewEncoder(streams.Out).Encode(plan); err != nil {
						return ldserr.New("internal.update-output", err.Error())
					}
				} else if _, err := fmt.Fprintf(streams.Out, "docs update-plan %s; %d edit(s); veto=%t; use --json to review the complete artifact\n", plan.PlanDigest, len(plan.Edits), plan.Veto); err != nil {
					return ldserr.New("internal.update-output", err.Error())
				}
			}
			return planErr
		},
	}
	cmd.Flags().String("observed", "", "path to the unchanged original docs collect JSON packet")
	cmd.Flags().String("request", "", "path to schema-1 explicit full-content edit requests")
	for _, name := range []string{"observed", "request"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","targetRoot","targetIdentity","observation","edits","veto","findings","skipped","planDigest"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"docs update-plan"},"targetRoot":{"type":"string"},"targetIdentity":{"type":"string"},"observation":{"type":"object"},"edits":{"type":"array","items":{"type":"object","required":["path","after","rationale","citations","before","afterSha256","afterBytes","ancestors","afterObservationSha256"]}},"veto":{"type":"boolean"},"findings":{"type":"array"},"skipped":{"type":"array"},"planDigest":{"type":"string"}}}`))
	return cmd
}

func updateApplyCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "update-apply <root>", Short: "replace only existing files bound to a freshly confirmed update-plan digest",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, _ := cmd.Flags().GetString("plan")
			confirm, _ := cmd.Flags().GetString("confirm")
			result, applyErr := adoption.ApplyUpdates(args[0], plan, confirm)
			if cliflags.FromContext(cmd.Context()).JSON {
				if err := json.NewEncoder(streams.Out).Encode(result); err != nil {
					return ldserr.New("internal.update-output", err.Error())
				}
			} else if _, err := fmt.Fprintf(streams.Out, "docs update-apply %s; %d replaced, %d observed untouched, %d unattempted\n", result.Status, len(result.Replaced), len(result.Untouched), len(result.Unattempted)); err != nil {
				return ldserr.New("internal.update-output", err.Error())
			}
			return applyErr
		},
	}
	cmd.Flags().String("plan", "", "path to the complete reviewed docs update-plan JSON artifact")
	cmd.Flags().String("confirm", "", "freshly human-confirmed sha256 digest of this exact update plan")
	for _, name := range []string{"plan", "confirm"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","planDigest","status","replaced","untouched","unattempted","failed","scratch","cleanupError"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"docs update-apply"},"planDigest":{"type":"string"},"status":{"enum":["applied","refused","partial","indeterminate"]},"replaced":{"type":"array"},"untouched":{"type":"array"},"unattempted":{"type":"array","items":{"type":"string"}},"failed":{"type":["object","null"]},"scratch":{"type":"string"},"cleanupError":{"type":"string"}}}`))
	return cmd
}
