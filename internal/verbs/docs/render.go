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

func renderCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render <root>",
		Short: "render approved adoption documentation without writing to the target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			factsPath, err := cmd.Flags().GetString("facts")
			if err != nil {
				return err
			}
			outputs, err := adoption.Render(args[0], factsPath)
			if err != nil {
				switch {
				case errors.Is(err, adoption.ErrInvalidRenderInput):
					return ldserr.New("validation.render-invalid", err.Error())
				case errors.Is(err, adoption.ErrUnsafeTarget):
					return ldserr.New("refusal.unsafe-target", err.Error())
				case errors.Is(err, os.ErrNotExist):
					return ldserr.New("validation.target-root-missing", err.Error())
				default:
					return ldserr.New("internal.render-failed", err.Error())
				}
			}
			if cliflags.FromContext(cmd.Context()).JSON {
				payload := struct {
					SchemaVersion int                     `json:"schemaVersion"`
					Operation     string                  `json:"operation"`
					Outputs       []adoption.RenderOutput `json:"outputs"`
				}{1, "docs render", outputs}
				data, err := json.Marshal(payload)
				if err != nil {
					return ldserr.New("internal.render-failed", err.Error())
				}
				if _, err := fmt.Fprintln(streams.Out, string(data)); err != nil {
					return ldserr.New("internal.render-failed", err.Error())
				}
				return nil
			}
			_, err = fmt.Fprintf(streams.Out, "%d adoption document(s) rendered; target unchanged\n", len(outputs))
			return err
		},
	}
	cmd.Flags().String("facts", "", "path to the owner-approved JSON fact bundle")
	if err := cmd.MarkFlagRequired("facts"); err != nil {
		panic(err)
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","outputs"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"docs render"},"outputs":{"type":"array","items":{"type":"object","required":["path","content","sha256","asset"],"properties":{"path":{"type":"string"},"content":{"type":"string"},"sha256":{"type":"string"},"asset":{"type":"object","required":["relativeName","source","sha256"]}}}}}}`))
	return cmd
}
