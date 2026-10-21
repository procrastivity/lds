package docs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

func assessCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "assess <root>", Short: "collect maintenance evidence and reviewer proposals without editing the target",
		Long: "Create one new assessment report outside an explicit target. The brief declares additional source paths/hashes, reviewer-authored proposals and unresolved questions. This deterministic operation does not infer impacts, call a provider, execute project commands or edit target files. Report creation requires Linux, existing safe parents and a new output path.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			brief, _ := cmd.Flags().GetString("brief")
			output, _ := cmd.Flags().GetString("output")
			result, err := adoption.Assess(args[0], brief, output)
			if err != nil {
				var structured *ldserr.Error
				switch {
				case errors.As(err, &structured):
					return structured
				case errors.Is(err, adoption.ErrUnsafeTarget):
					return ldserr.New("refusal.unsafe-target", err.Error())
				case errors.Is(err, os.ErrNotExist):
					return ldserr.New("validation.assessment-path-missing", err.Error())
				default:
					return ldserr.New("internal.assessment-failed", err.Error())
				}
			}
			if cliflags.FromContext(cmd.Context()).JSON {
				data, err := json.Marshal(result)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(streams.Out, string(data))
				return err
			}
			_, err = fmt.Fprintf(streams.Out, "Assessment report: %s; %d link finding(s), %d reviewer proposal(s). Review required; no target edits.\n", result.ReportPath, len(result.Validation.Findings), len(result.Brief.Proposals))
			return err
		},
	}
	cmd.Flags().String("brief", "", "path to the strict v1 review brief JSON")
	cmd.Flags().String("output", "", "absolute path to a new Markdown report outside the target (existing parent required)")
	for _, name := range []string{"brief", "output"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","targetRoot","targetIdentity","reportPath","briefSha256","brief","inventory","inventoryExitCode","validation","validationExitCode","evidence"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"docs assess"},"status":{"const":"review-required"},"targetRoot":{"type":"string"},"targetIdentity":{"type":"string"},"reportPath":{"type":"string"},"briefSha256":{"type":"string"},"brief":{"type":"object"},"inventory":{"type":"object"},"inventoryExitCode":{"const":0},"validation":{"type":"object"},"validationExitCode":{"enum":[0,1]},"evidence":{"type":"array"}}}`))
	return cmd
}
