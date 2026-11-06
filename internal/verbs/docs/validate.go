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

func validateCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "validate <root>", Short: "check local file targets and supported Markdown heading anchors",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := adoption.Validate(args[0])
			if err != nil {
				if errors.Is(err, adoption.ErrUnsafeTarget) {
					return ldserr.New("refusal.unsafe-target", err.Error())
				}
				if errors.Is(err, os.ErrNotExist) {
					return ldserr.New("validation.target-root-missing", err.Error())
				}
				return ldserr.New("internal.validation-failed", err.Error())
			}
			if cliflags.FromContext(cmd.Context()).JSON {
				b, err := json.Marshal(struct {
					SchemaVersion int    `json:"schemaVersion"`
					Operation     string `json:"operation"`
					adoption.Validation
				}{1, "docs validate", result})
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintln(streams.Out, string(b)); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(streams.Out, "%d finding(s), %d skipped check(s)\n", len(result.Findings), len(result.Skipped)); err != nil {
					return err
				}
			}
			if len(result.Findings) != 0 {
				return ldserr.New(result.Findings[0].Code, fmt.Sprintf("%d local file or heading target(s) missing", len(result.Findings)))
			}
			return nil
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","findings","skipped"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"docs validate"},"findings":{"type":"array","items":{"type":"object","required":["code","path","message"]}},"skipped":{"type":"array","items":{"type":"object","required":["source","code","reason"]}}}}`))
	return cmd
}
