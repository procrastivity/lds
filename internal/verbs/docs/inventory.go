// Package docs registers deterministic documentation plumbing beneath lds plumbing docs.
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

// Command registers the deterministic documentation plumbing leaves.
func Command(streams *iostreams.Streams) *cobra.Command {
	docs := &cobra.Command{Use: "docs", Short: "inspect and adopt documentation in an explicit target"}
	inventory := &cobra.Command{
		Use: "inventory <root>", Short: "list scoped Markdown files and supported local links",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := adoption.Inspect(args[0])
			if err != nil {
				if errors.Is(err, adoption.ErrUnsafeTarget) {
					return ldserr.New("refusal.unsafe-target", err.Error())
				}
				if errors.Is(err, os.ErrNotExist) {
					return ldserr.New("validation.target-root-missing", err.Error())
				}
				return ldserr.New("internal.inventory-failed", err.Error())
			}
			if cliflags.FromContext(cmd.Context()).JSON {
				b, err := json.Marshal(struct {
					SchemaVersion int    `json:"schemaVersion"`
					Operation     string `json:"operation"`
					adoption.Inventory
				}{1, "docs inventory", result})
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(streams.Out, string(b))
				return err
			}
			_, err = fmt.Fprintf(streams.Out, "%d Markdown file(s), %d local link(s), %d skipped check(s)\n", len(result.Files), len(result.Links), len(result.Skipped))
			return err
		},
	}
	surface.Annotate(inventory, surface.Plumbing)
	manifest.SetOutputSchema(inventory, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","files","links","skipped"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"docs inventory"},"files":{"type":"array","items":{"type":"object","required":["path","sha256","size"]}},"links":{"type":"array","items":{"type":"object","required":["source","line","target","kind"]}},"skipped":{"type":"array","items":{"type":"object","required":["source","code","reason"]}}}}`))
	docs.AddCommand(inventory)
	docs.AddCommand(validateCommand(streams))
	docs.AddCommand(renderCommand(streams))
	docs.AddCommand(planCommand(streams))
	docs.AddCommand(applyCommand(streams))
	docs.AddCommand(assessCommand(streams))
	docs.AddCommand(collectCommand(streams))
	docs.AddCommand(updatePlanCommand(streams))
	docs.AddCommand(updateApplyCommand(streams))
	return docs
}
