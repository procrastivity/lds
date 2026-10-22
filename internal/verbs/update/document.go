package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func resolveDocument() (asset.Resolved, error) {
	document, err := asset.Resolve("flows/update.md")
	if err != nil {
		return document, ldserr.New("internal.update-document-unavailable", "could not resolve the live update flow")
	}
	data := document.Bytes()
	if len(data) > 64<<10 || !utf8.Valid(data) || strings.TrimSpace(string(data)) == "" {
		return document, ldserr.New("validation.update-document-invalid", "update flow must be nonempty UTF-8 Markdown of at most 64 KiB")
	}
	for _, r := range string(data) {
		if unicode.Is(unicode.Cf, r) || unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return document, ldserr.New("validation.update-document-invalid", "update flow contains unsafe controls")
		}
	}
	return document, nil
}

func provenance(document asset.Resolved) flowAsset {
	sum := sha256.Sum256(document.Bytes())
	return flowAsset{"flows/update.md", document.Source.String(), hex.EncodeToString(sum[:])}
}

// DocumentCommand reads the shared live update flow without executing it.
func DocumentCommand(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "update", Short: "read the shared update flow (does not execute updates)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			document, err := resolveDocument()
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
			}{1, "update", "preparatory", "unavailable-for-execution", provenance(document), string(document.Bytes())})
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","availability","asset","content"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"update"},"status":{"const":"preparatory"},"availability":{"const":"unavailable-for-execution"},"asset":{"type":"object"},"content":{"type":"string"}}}`))
	return cmd
}
