package reconcile

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/manifest"
	core "github.com/procrastivity/lds/internal/reconcile"
	"github.com/procrastivity/lds/internal/surface"
)

// JournalCommand registers explicit journal read and caller-authored operations.
func JournalCommand(streams *iostreams.Streams) *cobra.Command {
	group := &cobra.Command{Use: "journal", Short: "read or explicitly update the caller-owned review journal"}
	read := &cobra.Command{
		Use:   "read",
		Short: "validate and project the named journal without writing it",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pair, _ := cmd.Flags().GetString("pair")
			journal, _ := cmd.Flags().GetString("journal")
			response, err := core.ReadJournal(core.JournalInput{PairPath: pair, JournalPath: journal})
			if err != nil {
				return err
			}
			return writeJournalResponse(streams, response)
		},
	}
	addPairJournalFlags(read)
	annotateJournalLeaf(read)
	group.AddCommand(read)

	for _, operation := range []string{"record", "decide", "reconsider"} {
		operation := operation
		leaf := &cobra.Command{
			Use:   operation,
			Short: journalDescription(operation),
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				pair, _ := cmd.Flags().GetString("pair")
				journal, _ := cmd.Flags().GetString("journal")
				scan, _ := cmd.Flags().GetString("scan")
				request, _ := cmd.Flags().GetString("request")
				response, err := core.WriteJournal(core.JournalInput{
					PairPath: pair, JournalPath: journal, ScanPath: scan, RequestPath: request, Operation: operation,
				})
				if err != nil {
					return err
				}
				return writeJournalResponse(streams, response)
			},
		}
		addPairJournalFlags(leaf)
		leaf.Flags().String("scan", "", "absolute path to the exact saved v1 scan JSON")
		leaf.Flags().String("request", "", "absolute path to the caller-authored v1 operation request JSON")
		markJournalFlagsRequired(leaf, "scan", "request")
		annotateJournalLeaf(leaf)
		group.AddCommand(leaf)
	}
	return group
}

func addPairJournalFlags(command *cobra.Command) {
	command.Flags().String("pair", "", "absolute path to the v1 pair descriptor")
	command.Flags().String("journal", "", "absolute path to the caller-owned journal")
	markJournalFlagsRequired(command, "pair", "journal")
}

func markJournalFlagsRequired(command *cobra.Command, names ...string) {
	for _, name := range names {
		if err := command.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
}

func annotateJournalLeaf(command *cobra.Command) {
	surface.Annotate(command, surface.Plumbing)
	manifest.SetOutputSchema(command, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","ok","journalPath","pairId","revision","snapshots","candidates","events"],"properties":{"schemaVersion":{"const":1},"operation":{"enum":["reconcile journal read","reconcile journal record","reconcile journal decide","reconcile journal reconsider"]},"ok":{"const":true},"journalPath":{"type":"string"},"pairId":{"type":"string"},"revision":{"type":"integer","minimum":0},"snapshots":{"type":"object","required":["left","right"]},"candidates":{"type":"array","items":{"type":"object","required":["candidateId","candidateKey","snapshots","evidenceDigest","evidence","correspondence","provenance","assessment","authority","decision","lastDecision","lastEventSeq"]}},"events":{"type":"array","items":{"type":"object"}},"replayed":{"type":"boolean"}}}`))
}

func journalDescription(operation string) string {
	switch operation {
	case "record":
		return "append a reviewed correspondence and exact scan evidence"
	case "decide":
		return "append an explicit authority and outcome decision"
	case "reconsider":
		return "explicitly supersede a prior decision against a fresh scan"
	default:
		return "update the caller-owned review journal"
	}
}

func writeJournalResponse(streams *iostreams.Streams, response core.JournalResponse) error {
	encoded, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(streams.Out, string(encoded))
	return err
}
