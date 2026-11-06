// Package review presents bounded offline documentation checks to humans.
package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

type reviewResult struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Operation     string              `json:"operation"`
	TargetRoot    string              `json:"targetRoot"`
	Status        string              `json:"status"`
	Inventory     adoption.Inventory  `json:"inventory"`
	Validation    adoption.Validation `json:"validation"`
	Limitations   []string            `json:"limitations"`
}

// Command composes existing inventory and local-target validation without
// prompts, provider requests, project commands or report/target writes.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "review <root>", Short: "review scoped documentation inventory, local-link findings and unchecked areas offline",
		Long: "Run read-only Linux documentation inspection for an explicit root. Prints files, local-link findings, skipped checks and limitations, or the same evidence as JSON. No credential, terminal, provider, project commands or file writes. Exit 0 means no supported missing local targets, not complete, correct or current documentation; semantic consistency and code alignment are not assessed.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := filepath.Abs(args[0])
			if err != nil {
				return ldserr.New("validation.target-root-invalid", err.Error())
			}
			inventory, validation, err := adoption.Review(root)
			if err != nil {
				switch {
				case errors.Is(err, adoption.ErrUnsafeTarget):
					return ldserr.New("refusal.unsafe-target", err.Error())
				case errors.Is(err, os.ErrNotExist):
					return ldserr.New("validation.target-root-missing", err.Error())
				default:
					return ldserr.New("internal.review-failed", err.Error())
				}
			}
			result := reviewResult{
				SchemaVersion: 1, Operation: "review", TargetRoot: root, Status: "no-findings",
				Inventory: inventory, Validation: validation,
				Limitations: []string{
					"Inventory covers only root README.md and regular Markdown below docs/ and engineering/; absent optional tracks are not findings.",
					"Local files and bounded ASCII ATX heading anchors in inventoried Markdown are checked. External URLs, queries and unsupported fragments/Markdown remain unchecked; see skipped checks.",
					"Documentation completeness, prose accuracy, semantic/cross-track consistency, code alignment and runtime behavior are not assessed.",
					"Observations are not an atomic snapshot; concurrent changes may affect results. No findings does not mean complete, correct or current documentation.",
				},
			}
			if len(validation.Findings) != 0 {
				result.Status = "findings"
			}
			if cliflags.FromContext(cmd.Context()).JSON {
				err = json.NewEncoder(streams.Out).Encode(result)
			} else {
				_, err = fmt.Fprintln(streams.Out, humanReport(result))
			}
			if err != nil {
				return ldserr.New("internal.review-output", err.Error())
			}
			if len(validation.Findings) != 0 {
				return ldserr.New(validation.Findings[0].Code, fmt.Sprintf("%d local file or heading target(s) missing", len(validation.Findings)))
			}
			return nil
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{"type":"object","required":["schemaVersion","operation","targetRoot","status","inventory","validation","limitations"],"properties":{"schemaVersion":{"const":1},"operation":{"const":"review"},"targetRoot":{"type":"string"},"status":{"enum":["findings","no-findings"]},"inventory":{"type":"object","required":["files","links","skipped"]},"validation":{"type":"object","required":["findings","skipped"]},"limitations":{"type":"array","minItems":1,"items":{"type":"string"}}}}`))
	return cmd
}

func humanReport(result reviewResult) string {
	lines := []string{
		"Documentation review: " + result.Status,
		fmt.Sprintf("Root: %q", result.TargetRoot),
		fmt.Sprintf("%d Markdown file(s), %d local link(s), %d finding(s), %d skipped check(s)", len(result.Inventory.Files), len(result.Inventory.Links), len(result.Validation.Findings), len(result.Validation.Skipped)),
		"", "Scoped Markdown files:",
	}
	if len(result.Inventory.Files) == 0 {
		lines = append(lines, "  None observed; documentation completeness is not assessed.")
	}
	for _, file := range result.Inventory.Files {
		lines = append(lines, fmt.Sprintf("  %q (%d bytes, sha256:%s)", file.Path, file.Size, file.SHA256))
	}
	lines = append(lines, "", "Local-link findings:")
	if len(result.Validation.Findings) == 0 {
		lines = append(lines, "  No supported missing local targets observed; unchecked areas remain.")
	}
	for _, finding := range result.Validation.Findings {
		lines = append(lines, fmt.Sprintf("  %q:%d [%s] %q", finding.Path, finding.Line, finding.Code, finding.Message))
	}
	lines = append(lines, "", "Skipped checks:")
	if len(result.Validation.Skipped) == 0 {
		lines = append(lines, "  None reported by the bounded parser; see limitations below.")
	}
	for _, skip := range result.Validation.Skipped {
		location := fmt.Sprintf("%q", skip.Source)
		if skip.Line != 0 {
			location += fmt.Sprintf(":%d", skip.Line)
		}
		lines = append(lines, fmt.Sprintf("  %s [%s] %q", location, skip.Code, skip.Reason))
	}
	lines = append(lines, "", "Limitations (not verified):")
	for _, limitation := range result.Limitations {
		lines = append(lines, "  - "+limitation)
	}
	return strings.Join(lines, "\n")
}
