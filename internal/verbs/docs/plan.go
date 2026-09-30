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

func planCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan <root>",
		Short: "preview documentation adoption without writing to the target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			factsPath, err := cmd.Flags().GetString("facts")
			if err != nil {
				return err
			}
			result, planErr := adoption.Plan(args[0], factsPath)
			if result.SchemaVersion == 1 {
				if cliflags.FromContext(cmd.Context()).JSON {
					data, err := json.Marshal(result)
					if err != nil {
						return ldserr.New("internal.plan-failed", err.Error())
					}
					if _, err := fmt.Fprintln(streams.Out, string(data)); err != nil {
						return ldserr.New("internal.plan-failed", err.Error())
					}
				} else if _, err := fmt.Fprintf(streams.Out, "%d output(s); veto=%t; %d finding(s); %d skipped check(s)\n", len(result.Outputs), result.Veto, len(result.Findings), len(result.Skipped)); err != nil {
					return ldserr.New("internal.plan-failed", err.Error())
				}
			}
			if planErr == nil {
				return nil
			}
			var refusal *adoption.PlanRefusal
			if errors.As(planErr, &refusal) {
				return ldserr.New(refusal.Code, refusal.Message)
			}
			switch {
			case errors.Is(planErr, adoption.ErrInvalidRenderInput):
				return ldserr.New("validation.render-invalid", planErr.Error())
			case errors.Is(planErr, adoption.ErrUnsafeTarget):
				return ldserr.New("refusal.unsafe-target", planErr.Error())
			case errors.Is(planErr, os.ErrNotExist):
				return ldserr.New("validation.target-root-missing", planErr.Error())
			default:
				return ldserr.New("internal.plan-failed", planErr.Error())
			}
		},
	}
	cmd.Flags().String("facts", "", "path to the owner-approved JSON fact bundle")
	if err := cmd.MarkFlagRequired("facts"); err != nil {
		panic(err)
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","targetRoot","targetIdentity","planDigest","factsDigest","outputs","necessaryParentDirectories","preserved","reportOnly","veto","findings","skipped"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"docs plan"},"targetRoot":{"type":"string"},"targetIdentity":{"type":"string"},"planDigest":{"type":["string","null"]},"factsDigest":{"type":"string"},"outputs":{"type":"array","items":{"type":"object","required":["path","classification","reason"],"properties":{"path":{"type":"string"},"classification":{"enum":["create","identical-untouched","different-collision","refusal"]},"reason":{"type":"string"},"content":{"type":"string"},"contentSha256":{"type":"string"},"asset":{"type":"object","required":["relativeName","source","sha256"]},"existingSha256":{"type":"string"},"existingContent":{"type":"string"},"existingContentBase64":{"type":"string"},"existingContentEncoding":{"type":"string"},"existingProvenance":{"type":"string"},"affectedLinks":{"type":"array"}}}},"necessaryParentDirectories":{"type":"array","items":{"type":"string"}},"preserved":{"type":"array","items":{"type":"object","required":["path","sha256","size"]}},"reportOnly":{"type":"array","items":{"type":"object","required":["path","message"]}},"veto":{"type":"boolean"},"findings":{"type":"array","items":{"type":"object","required":["code","message"]}},"skipped":{"type":"array","items":{"type":"object","required":["source","code","reason"]}}}}`))
	return cmd
}
