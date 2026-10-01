// Package reconcile registers deterministic reconciliation plumbing.
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

// Command registers reconciliation scan and journal plumbing.
func Command(streams *iostreams.Streams) *cobra.Command {
	group := &cobra.Command{
		Use:   "reconcile",
		Short: "scan pinned project evidence and review its caller-owned journal",
	}
	scan := &cobra.Command{
		Use:   "scan",
		Short: "scan a strict pair descriptor and pinned Git snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pairPath, _ := cmd.Flags().GetString("pair")
			leftRoot, _ := cmd.Flags().GetString("left-root")
			rightRoot, _ := cmd.Flags().GetString("right-root")
			mapPaths, _ := cmd.Flags().GetStringArray("map")
			journal, _ := cmd.Flags().GetString("journal")

			response, err := core.Scan(core.Input{
				PairPath:  pairPath,
				LeftRoot:  leftRoot,
				RightRoot: rightRoot,
				MapPaths:  mapPaths,
				Journal:   journal,
			})
			if err != nil {
				return err
			}
			encoded, err := json.Marshal(response)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(streams.Out, string(encoded))
			return err
		},
	}
	scan.Flags().String("pair", "", "absolute path to the v1 pair descriptor")
	scan.Flags().String("left-root", "", "absolute path to the pinned left Git worktree root")
	scan.Flags().String("right-root", "", "absolute path to the pinned right Git worktree root")
	scan.Flags().StringArray("map", nil, "absolute path to a tracked raw map file (repeatable)")
	scan.Flags().String("journal", "", "optional absolute path to an existing read-only journal")
	for _, name := range []string{"pair", "left-root", "right-root", "map"} {
		if err := scan.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	surface.Annotate(scan, surface.Plumbing)
	manifest.SetOutputSchema(scan, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","ok","pairId","pairPath","pairSha256","leftRoot","rightRoot","snapshots","maps","candidates"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"reconcile scan"},"ok":{"const":true},"pairId":{"type":"string"},"pairPath":{"type":"string"},"pairSha256":{"type":"string","pattern":"^[0-9a-f]{64}$"},"leftRoot":{"type":"string"},"rightRoot":{"type":"string"},"journalPath":{"type":"string"},"snapshots":{"type":"object"},"maps":{"type":"array","items":{"type":"object","required":["projectId","revision","path","sha256"],"properties":{"projectId":{"type":"string"},"revision":{"type":"string"},"path":{"type":"string"},"sha256":{"type":"string","pattern":"^[0-9a-f]{64}$"}}}},"candidates":{"type":"array","items":{"type":"object","required":["candidateId","candidateKey","evidenceDigest","evidence","left","right","confidence","evidenceStatus"],"properties":{"candidateId":{"type":"string","pattern":"^cand-v1-[0-9a-f]{64}$"},"candidateKey":{"type":"string"},"evidenceDigest":{"type":["string","null"]},"evidence":{"type":"array"},"left":{"type":"array"},"right":{"type":"array"},"confidence":{"const":"not-assessed"},"evidenceStatus":{"enum":["not-compared","new","unchanged","changed","out-of-scope"]},"journalRevision":{"type":"integer"},"journalEvidenceDigest":{"type":["string","null"]},"lastDecision":{"type":["object","null"]}}}}}}`))
	group.AddCommand(scan)
	group.AddCommand(JournalCommand(streams))
	return group
}
