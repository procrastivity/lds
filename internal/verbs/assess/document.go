// Package assess implements the human assessment workflow and its live reader.
package assess

import (
	"crypto/sha256"
	"encoding/hex"
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

func resolveDocument(projectRoot string) (asset.Resolved, error) {
	var document asset.Resolved
	var err error
	if projectRoot == "" {
		document, err = asset.Resolve("flows/assess.md")
	} else {
		document, err = asset.ResolveIn(projectRoot, "flows/assess.md")
		if err != nil {
			return document, ldserr.New("refusal.project-asset", fmt.Sprintf("refusing selected project asset: %v", err))
		}
	}
	if err != nil {
		return document, ldserr.New("internal.assess-document-unavailable", "could not resolve the live assessment flow")
	}
	data := document.Bytes()
	if len(data) > 64*1024 || !utf8.Valid(data) || strings.TrimSpace(string(data)) == "" {
		return document, ldserr.New("validation.assess-document-invalid", "assessment flow must be nonempty UTF-8 Markdown, at most 64 KiB")
	}
	for _, r := range string(data) {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return document, ldserr.New("validation.assess-document-invalid", "assessment flow contains unsafe control characters")
		}
	}
	return document, nil
}

func provenance(document asset.Resolved) flowAsset {
	sum := sha256.Sum256(document.Bytes())
	return flowAsset{RelativeName: "flows/assess.md", Source: document.Source.String(), SHA256: hex.EncodeToString(sum[:])}
}

// DocumentCommand reads the live shared flow; it never executes an assessment.
func DocumentCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "assess", Short: "read the shared assessment flow (does not run assessment)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			document, err := resolveDocument(cliflags.FromContext(cmd.Context()).ProjectRoot)
			if err != nil {
				return err
			}
			if !cliflags.FromContext(cmd.Context()).JSON {
				_, err := streams.Out.Write(document.Bytes())
				return err
			}
			data, err := json.Marshal(struct {
				SchemaVersion int       `json:"schemaVersion"`
				Operation     string    `json:"operation"`
				Status        string    `json:"status"`
				Availability  string    `json:"availability"`
				Asset         flowAsset `json:"asset"`
				Content       string    `json:"content"`
			}{1, "assess", "preparatory", "unavailable-for-execution", provenance(document), string(document.Bytes())})
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(streams.Out, string(data))
			return err
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","availability","asset","content"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"assess"},"status":{"const":"preparatory"},"availability":{"const":"unavailable-for-execution"},"asset":{"type":"object"},"content":{"type":"string"}}}`))
	return cmd
}
