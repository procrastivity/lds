package reconcile

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/asset"
	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

type flowAsset struct {
	RelativeName string `json:"relativeName"`
	Source       string `json:"source"`
	SHA256       string `json:"sha256"`
}

func resolveFlow(projectRoot string) (asset.Resolved, error) {
	var document asset.Resolved
	var err error
	if projectRoot == "" {
		document, err = asset.Resolve("flows/reconcile.md")
	} else {
		document, err = asset.ResolveIn(projectRoot, "flows/reconcile.md")
		if err != nil {
			return document, ldserr.New("refusal.project-asset", fmt.Sprintf("refusing selected project asset: %v", err))
		}
	}
	if err != nil {
		return document, ldserr.New("internal.reconcile-document-unavailable", "could not resolve the live reconciliation flow")
	}
	data := document.Bytes()
	if len(data) > 64<<10 || !utf8.Valid(data) || strings.TrimSpace(string(data)) == "" {
		return document, ldserr.New("validation.reconcile-document-invalid", "reconciliation flow must be nonempty UTF-8 Markdown of at most 64 KiB")
	}
	for _, r := range string(data) {
		if unicode.Is(unicode.Cf, r) || unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return document, ldserr.New("validation.reconcile-document-invalid", "reconciliation flow contains unsafe controls")
		}
	}
	return document, nil
}

func flowProvenance(document asset.Resolved) flowAsset {
	return flowAsset{"flows/reconcile.md", document.Source.String(), reportHash(document.Bytes())}
}

func flowCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "flow", Short: "read the shared reconciliation report flow (does not execute it)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			document, err := resolveFlow(cliflags.FromContext(cmd.Context()).ProjectRoot)
			if err != nil {
				return err
			}
			if !cliflags.FromContext(cmd.Context()).JSON {
				_, err := streams.Out.Write(document.Bytes())
				return err
			}
			return json.NewEncoder(streams.Out).Encode(struct {
				SchemaVersion int       `json:"schemaVersion"`
				Operation     string    `json:"operation"`
				Status        string    `json:"status"`
				Availability  string    `json:"availability"`
				Asset         flowAsset `json:"asset"`
				Content       string    `json:"content"`
			}{1, "reconcile", "preparatory", "unavailable-for-execution", flowProvenance(document), string(document.Bytes())})
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","availability","asset","content"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"reconcile"},"status":{"const":"preparatory"},"availability":{"const":"unavailable-for-execution"},"asset":{"type":"object"},"content":{"type":"string"}}}`))
	return cmd
}
