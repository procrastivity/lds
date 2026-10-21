package docs

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

func collectCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "collect <root>", Short: "capture assessment evidence and source hashes without writes",
		Long: "Read scoped documentation and explicitly named regular UTF-8 source files into an assessment packet. Computes hashes and repeats collection without a provider, proposed edits or report creation. JSON contains complete selected file contents; store it privately outside the target.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reason, _ := cmd.Flags().GetString("reason")
			audience, _ := cmd.Flags().GetString("audience")
			sources, _ := cmd.Flags().GetStringArray("source")
			result, err := adoption.CollectAssessment(args[0], reason, audience, sources)
			if err != nil {
				return assessmentError(err)
			}
			if cliflags.FromContext(cmd.Context()).JSON {
				data, err := json.Marshal(result)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(streams.Out, string(data))
				return err
			}
			_, err = fmt.Fprintf(streams.Out, "Collected %d evidence file(s), %d link finding(s). No report or target edits; use --json for the complete private review packet.\n", len(result.Evidence), len(result.Validation.Findings))
			return err
		},
	}
	cmd.Flags().String("reason", "", "change or review question to assess")
	cmd.Flags().String("audience", "", "intended readers and their task")
	cmd.Flags().StringArray("source", nil, "explicit target-relative source file to include (repeatable; full contents captured)")
	for _, name := range []string{"reason", "audience"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(assessmentOutputSchema))
	return cmd
}
