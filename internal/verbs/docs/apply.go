package docs

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

func applyCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply <root>",
		Short: "apply only the explicitly confirmed reviewed documentation plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			facts, _ := cmd.Flags().GetString("facts")
			plan, _ := cmd.Flags().GetString("plan")
			confirm, _ := cmd.Flags().GetString("confirm")
			result, applyErr := adoption.Apply(args[0], facts, plan, confirm, cliflags.FromContext(cmd.Context()).ProjectRoot)
			if result.SchemaVersion == 1 {
				if cliflags.FromContext(cmd.Context()).JSON {
					data, err := json.Marshal(result)
					if err != nil {
						return ldserr.New("internal.apply-failed", err.Error())
					}
					if _, err := fmt.Fprintln(streams.Out, string(data)); err != nil {
						return ldserr.New("internal.apply-failed", err.Error())
					}
				} else if _, err := fmt.Fprintf(streams.Out, "docs apply %s; %d created, %d identical untouched\n", result.Status, len(result.Created), len(result.IdenticalUntouched)); err != nil {
					return ldserr.New("internal.apply-failed", err.Error())
				}
			}
			if applyErr == nil {
				return nil
			}
			var structured *adoption.ApplyError
			if errors.As(applyErr, &structured) {
				return ldserr.New(structured.Code, structured.Message)
			}
			return ldserr.New("internal.apply-failed", applyErr.Error())
		},
	}
	cmd.Flags().String("facts", "", "path to the owner-approved JSON fact bundle")
	cmd.Flags().String("plan", "", "path to the exact reviewed docs plan JSON")
	cmd.Flags().String("confirm", "", "explicitly confirmed sha256 digest of the reviewed plan")
	for _, name := range []string{"facts", "plan", "confirm"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","planDigest","status","created","identicalUntouched","unattempted","failed","preserved"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"docs apply"},"planDigest":{"type":"string"},"status":{"enum":["applied","current","refused","partial"]},"created":{"type":"array","items":{"type":"object","required":["path","sha256"]}},"identicalUntouched":{"type":"array","items":{"type":"object","required":["path","sha256"]}},"unattempted":{"type":"array","items":{"type":"object","required":["path"]}},"failed":{"type":["object","null"]},"preserved":{"type":"array"}}}`))
	return cmd
}
