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
		Args: reportArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			input := reportInput(cmd)
			var result core.ReportResult
			var err error
			observed, _ := cmd.Flags().GetString("observed")
			if cmd.Flags().Changed("observed") {
				draft, _ := cmd.Flags().GetString("draft")
				receipt, _ := cmd.Flags().GetString("receipt")
				result, err = core.ReportWithObservation(input, observed, draft, receipt)
			} else {
				result, err = core.Report(input)
			}
			if err != nil {
				return err
			}
			return writeReportResult(cmd, streams, result)
		},
	}
	reportFlags(cmd)
	cmd.Flags().String("observed", "", "external unchanged original reconcile collect JSON (requires --draft and --receipt)")
	cmd.Flags().String("draft", "", "external strict model proposal JSON; never write authority")
	cmd.Flags().String("receipt", "", "external attributed provider/flow/consent receipt JSON")
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(reportOutputSchema))
	return cmd
}

func reportFlags(cmd *cobra.Command) {
	cmd.Flags().String("pair", "", "absolute path to an external strict v1 pair descriptor")
	cmd.Flags().String("left-root", "", "absolute path to the clean pinned left Git worktree")
	cmd.Flags().String("right-root", "", "absolute path to the clean pinned right Git worktree")
	cmd.Flags().StringArray("map", nil, "absolute path to a tracked raw map file (repeatable, at most 16)")
	cmd.Flags().String("journal", "", "optional absolute path to an existing external read-only journal")
	cmd.Flags().String("review", "", "absolute path to an external strict v1 review brief")
	cmd.Flags().String("output", "", "new absolute external private Markdown report path")
	markJournalFlagsRequired(cmd, "pair", "left-root", "right-root", "map", "review", "output")
}

func reportInput(cmd *cobra.Command) core.ReportInput {
	pair, _ := cmd.Flags().GetString("pair")
	left, _ := cmd.Flags().GetString("left-root")
	right, _ := cmd.Flags().GetString("right-root")
	maps, _ := cmd.Flags().GetStringArray("map")
	journal, _ := cmd.Flags().GetString("journal")
	review, _ := cmd.Flags().GetString("review")
	output, _ := cmd.Flags().GetString("output")
	return core.ReportInput{Input: core.Input{PairPath: pair, LeftRoot: left, RightRoot: right, MapPaths: maps, Journal: journal}, ReviewPath: review, OutputPath: output}
}

func reportArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return err
	}
	count := 0
	for _, name := range []string{"observed", "draft", "receipt"} {
		if cmd.Flags().Changed(name) {
			count++
		}
	}
	if count != 0 && count != 3 {
		return fmt.Errorf("--observed, --draft and --receipt must be supplied together")
	}
	return nil
}

func writeReportResult(cmd *cobra.Command, streams *iostreams.Streams, result core.ReportResult) error {
	if cliflags.FromContext(cmd.Context()).JSON {
		return json.NewEncoder(streams.Out).Encode(result)
	}
	mode := "offline"
	if result.Assistance {
		mode = "assisted advice"
	}
	_, err := fmt.Fprintf(streams.Out, "Review required: %s\nSHA-256: %s\n%d candidates; %s, no source or journal edits.\n", result.ReportPath, result.ReportSHA256, result.Candidates, mode)
	return err
}

func collectCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "collect", Short: "capture the original bounded reconciliation report evidence without publishing", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			packet, err := core.CollectReport(reportInput(cmd))
			if err != nil {
				return err
			}
			return json.NewEncoder(streams.Out).Encode(packet)
		},
	}
	reportFlags(cmd)
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","collector","inputs","review","content","scan","scanSha256","rootIdentities","outputParentIdentity","excluded"],"properties":{"schemaVersion":{"const":1},"collector":{"const":"lds/reconcile-report/v1"},"review":{"type":"object"},"scan":{"type":"object"},"content":{"type":"array"}}}`))
	return cmd
}

const reportOutputSchema = `{"type":"object","required":["schemaVersion","operation","status","reportPath","reportSha256","pairId","snapshots","candidates","assistance"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"reconcile report"},"status":{"const":"review-required"},"reportPath":{"type":"string"},"reportSha256":{"type":"string","pattern":"^[0-9a-f]{64}$"},"pairId":{"type":"string"},"snapshots":{"type":"object"},"candidates":{"type":"integer","minimum":0},"assistance":{"type":"boolean"}}}`

// RootCommand retains offline reports and adds an explicit assisted form;
// journal operations remain separate and provider-free.
func RootCommand(streams *iostreams.Streams) *cobra.Command {
	group := &cobra.Command{Use: "reconcile", Short: "review cross-project evidence and explicitly record external journal claims"}
	group.AddCommand(assistedReportCommand(streams))
	group.AddCommand(guidedJournalCommand(streams))
	return group
}
