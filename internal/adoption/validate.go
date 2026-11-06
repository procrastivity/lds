package adoption

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"
)

// Finding describes a missing local file or supported heading target.
type Finding struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Line    int    `json:"line,omitempty"`
	Target  string `json:"target,omitempty"`
	Message string `json:"message"`
}

// Validation contains the bounded link findings and explicitly skipped checks.
type Validation struct {
	Findings []Finding `json:"findings"`
	Skipped  []Skip    `json:"skipped"`
}

// Validate checks supported local files and bounded heading fragments in
// inventoried Markdown. Unsupported syntax remains an explicit skip.
func Validate(root string) (Validation, error) {
	return validateWithOverlay(root, nil)
}

// Review inventories once and checks the local links from that same inventory.
// Neither the inventory nor the subsequent target checks form an atomic snapshot.
func Review(root string) (Inventory, Validation, error) {
	inventory, err := Inspect(root)
	if err != nil {
		return inventory, Validation{Findings: []Finding{}, Skipped: []Skip{}}, err
	}
	validation, err := validateInventory(root, inventory, nil)
	return inventory, validation, err
}

func validateWithOverlay(root string, overlay map[string]string) (Validation, error) {
	inventory, err := inspectWithOverlay(root, overlay)
	if err != nil {
		return Validation{Findings: []Finding{}, Skipped: []Skip{}}, err
	}
	return validateInventory(root, inventory, overlay)
}

func validateInventory(root string, inventory Inventory, overlay map[string]string) (Validation, error) {
	result := Validation{Findings: []Finding{}, Skipped: []Skip{}}
	result.Skipped = append(result.Skipped, inventory.Skipped...)
	dir, err := openRoot(root)
	if err != nil {
		return result, fmt.Errorf("open target root: %w", err)
	}
	defer func() { _ = dir.Close() }()
	files := map[string]File{}
	for _, file := range inventory.Files {
		files[file.Path] = file
	}
	indexes := map[string]headingIndex{}
	for _, link := range inventory.Links {
		target := link.Target
		fragment := ""
		hasFragment := false
		if i := strings.IndexByte(target, '#'); i >= 0 {
			fragment = target[i+1:]
			target = target[:i]
			hasFragment = true
		}
		if strings.Contains(target, "?") {
			result.Skipped = append(result.Skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.query-skipped", Reason: "query-bearing link is not checked"})
			continue
		}
		if path.IsAbs(target) || strings.HasPrefix(target, "\\") {
			result.Skipped = append(result.Skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.unsafe-link-skipped", Reason: "absolute local target is not followed"})
			continue
		}
		candidate := path.Clean(path.Join(path.Dir(link.Source), target))
		if target == "" {
			candidate = link.Source
		}
		if candidate == ".." || strings.HasPrefix(candidate, "../") {
			result.Skipped = append(result.Skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.unsafe-link-skipped", Reason: "target escapes the explicit root"})
			continue
		}
		regular, err := regularTarget(dir, candidate)
		switch {
		case errors.Is(err, ErrUnsafeTarget):
			result.Skipped = append(result.Skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.unsafe-link-skipped", Reason: "symlink target is not followed"})
		case errors.Is(err, os.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
			result.Findings = append(result.Findings, Finding{Code: "validation.local-target-missing", Path: link.Source, Line: link.Line, Target: link.Target, Message: "local file target is missing: " + link.Target})
		case err != nil:
			return result, fmt.Errorf("check %s from %s: %w", candidate, link.Source, err)
		case !regular:
			result.Skipped = append(result.Skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.not-file-skipped", Reason: "local target is not a regular file"})
		case hasFragment:
			file, scoped := files[candidate]
			decoded, decodeErr := url.PathUnescape(fragment)
			reason := ""
			switch {
			case !scoped:
				reason = "heading target is outside inventoried Markdown scope"
			case decodeErr != nil || decoded == "" || !utf8.ValidString(decoded) || strings.IndexFunc(decoded, unicode.IsControl) >= 0:
				reason = "empty, malformed or control-bearing fragment is not checked"
			case lineFragment.MatchString(decoded):
				reason = "source-line fragment is outside heading checks"
			default:
				index, cached := indexes[candidate]
				if !cached {
					index = readHeadings(dir, file, overlay)
					indexes[candidate] = index
				}
				reason = index.reason
				if reason == "" && !index.anchors[decoded] {
					result.Findings = append(result.Findings, Finding{Code: "validation.local-anchor-missing", Path: link.Source, Line: link.Line, Target: link.Target, Message: "local heading anchor is missing: " + link.Target})
				}
			}
			if reason != "" {
				result.Skipped = append(result.Skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.fragment-skipped", Reason: reason})
			}
		}
	}
	sort.Slice(result.Findings, func(i, j int) bool {
		a, b := result.Findings[i], result.Findings[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Code < b.Code
	})
	sort.Slice(result.Skipped, func(i, j int) bool {
		a, b := result.Skipped[i], result.Skipped[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Code < b.Code
	})
	return result, nil
}

func regularTarget(root *os.File, relative string) (bool, error) {
	current := root
	for _, component := range strings.Split(relative, "/") {
		next, err := openChild(current, component)
		if current != root {
			_ = current.Close()
		}
		if err != nil {
			return false, err
		}
		current = next
	}
	defer func() { _ = current.Close() }()
	info, err := current.Stat()
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
}
