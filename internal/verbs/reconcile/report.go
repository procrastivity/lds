package reconcile

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/manifest"
	core "github.com/procrastivity/lds/internal/reconcile"
	"github.com/procrastivity/lds/internal/surface"
)

// ReportCommand is the same offline operation in porcelain and plumbing.
func ReportCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "report", Short: "publish a private offline reconciliation review report",
		Long: "Build a Linux-only external review report from explicit clean pinned Git roots, maps, a strict review brief and an optional existing journal. No provider, project commands, source edits or journal writes. Output must be a new absolute path outside both roots and Git common directories; parents must already exist. Inputs and output parents must have no symlink components.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pair, _ := cmd.Flags().GetString("pair")
			left, _ := cmd.Flags().GetString("left-root")
			right, _ := cmd.Flags().GetString("right-root")
			maps, _ := cmd.Flags().GetStringArray("map")
			journal, _ := cmd.Flags().GetString("journal")
			review, _ := cmd.Flags().GetString("review")
			output, _ := cmd.Flags().GetString("output")
			result, err := core.Report(core.ReportInput{
				Input:      core.Input{PairPath: pair, LeftRoot: left, RightRoot: right, MapPaths: maps, Journal: journal},
				ReviewPath: review, OutputPath: output,
			})
			if err != nil {
				return err
			}
			if cliflags.FromContext(cmd.Context()).JSON {
				return json.NewEncoder(streams.Out).Encode(result)
			}
			_, err = fmt.Fprintf(streams.Out, "Review required: %s\nSHA-256: %s\n%d candidates; offline, no source or journal edits.\n", result.ReportPath, result.ReportSHA256, result.Candidates)
			return err
		},
	}
	cmd.Flags().String("pair", "", "absolute path to an external strict v1 pair descriptor")
	cmd.Flags().String("left-root", "", "absolute path to the clean pinned left Git worktree")
	cmd.Flags().String("right-root", "", "absolute path to the clean pinned right Git worktree")
	cmd.Flags().StringArray("map", nil, "absolute path to a tracked raw map file (repeatable, at most 16)")
	cmd.Flags().String("journal", "", "optional absolute path to an existing external read-only journal")
	cmd.Flags().String("review", "", "absolute path to an external strict v1 review brief")
	cmd.Flags().String("output", "", "new absolute external private Markdown report path")
	markJournalFlagsRequired(cmd, "pair", "left-root", "right-root", "map", "review", "output")
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","reportPath","reportSha256","pairId","snapshots","candidates","assistance"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"reconcile report"},"status":{"const":"review-required"},"reportPath":{"type":"string"},"reportSha256":{"type":"string","pattern":"^[0-9a-f]{64}$"},"pairId":{"type":"string"},"snapshots":{"type":"object"},"candidates":{"type":"integer","minimum":0},"assistance":{"const":false}}}`))
	return cmd
}

// RootCommand exposes the offline human-facing entry point without giving it
// a model surface or duplicating the qualified plumbing implementation.
func RootCommand(streams *iostreams.Streams) *cobra.Command {
	group := &cobra.Command{Use: "reconcile", Short: "review cross-project evidence and explicitly record external journal claims"}
	report := ReportCommand(streams)
	surface.Alias(report, "plumbing reconcile report")
	group.AddCommand(report)
	group.AddCommand(guidedJournalCommand(streams))
	return group
}
