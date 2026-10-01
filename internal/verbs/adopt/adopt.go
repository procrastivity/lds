// Package adopt registers the read-only, preparation-only adoption document.
package adopt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

const documentAsset = "flows/adopt.md"

// ErrInvalidDocument marks a missing or unsafe shared adoption-flow asset.
var ErrInvalidDocument = errors.New("invalid adoption document")

// ResolveDocument returns the live, validated shared adoption-flow asset.
// The root workflow and the plumbing reader consume this same source.
func ResolveDocument() (asset.Resolved, error) {
	resolved, err := asset.Resolve(documentAsset)
	if err != nil {
		return asset.Resolved{}, fmt.Errorf("resolve %s: %w", documentAsset, err)
	}
	if !validDocument(resolved.Bytes()) {
		return asset.Resolved{}, fmt.Errorf("%w: asset %q must be non-empty UTF-8 Markdown without unsafe control characters", ErrInvalidDocument, documentAsset)
	}
	return resolved, nil
}

// Command constructs the read-only `lds plumbing adopt` document verb.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adopt",
		Short: "read the preparation-only adoption document (does not run adoption)",
		Long:  "Print the live, preparation-only adoption flow document. This command does not run adoption, inspect a target, plan, request confirmation, or write project files.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolved, err := ResolveDocument()
			if err != nil {
				if errors.Is(err, ErrInvalidDocument) {
					return ldserr.New("validation.adopt-document-invalid", err.Error())
				}
				return ldserr.New("internal.adopt-document-unavailable", err.Error())
			}
			content := resolved.Bytes()

			if cliflags.FromContext(cmd.Context()).JSON {
				sum := sha256.Sum256(content)
				payload := struct {
					SchemaVersion int    `json:"schemaVersion"`
					Operation     string `json:"operation"`
					Status        string `json:"status"`
					Availability  string `json:"availability"`
					Asset         struct {
						RelativeName string `json:"relativeName"`
						Source       string `json:"source"`
						SHA256       string `json:"sha256"`
					} `json:"asset"`
					Content string `json:"content"`
				}{
					SchemaVersion: 1,
					Operation:     "adopt",
					Status:        "preparatory",
					Availability:  "unavailable-for-execution",
					Content:       string(content),
				}
				payload.Asset.RelativeName = documentAsset
				payload.Asset.Source = resolved.Source.String()
				payload.Asset.SHA256 = hex.EncodeToString(sum[:])
				data, err := json.Marshal(payload)
				if err != nil {
					return ldserr.New("internal.adopt-document-encode", err.Error())
				}
				if _, err := fmt.Fprintln(streams.Out, string(data)); err != nil {
					return ldserr.New("internal.adopt-document-write", err.Error())
				}
				return nil
			}
			_, err = streams.Out.Write(content)
			return err
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","status","availability","asset","content"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"adopt"},"status":{"const":"preparatory"},"availability":{"const":"unavailable-for-execution"},"asset":{"type":"object","required":["relativeName","source","sha256"],"properties":{"relativeName":{"const":"flows/adopt.md"},"source":{"enum":["override","default","embedded"]},"sha256":{"type":"string"}}},"content":{"type":"string"}}}`))
	return cmd
}

func validDocument(content []byte) bool {
	if len(content) == 0 || !utf8.Valid(content) || strings.TrimSpace(string(content)) == "" {
		return false
	}
	for _, r := range string(content) {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}
