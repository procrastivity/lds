package adoption

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"syscall"
)

// Finding describes a missing local file target in a scoped Markdown file.
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

// Validate checks only supported local file targets in scoped Markdown.
// Fragments are not checked; external, query-bearing, unsafe and other
// unsupported targets are reported as skips rather than successes.
func Validate(root string) (Validation, error) {
	return validateWithOverlay(root, nil)
}

func validateWithOverlay(root string, overlay map[string]string) (Validation, error) {
	result := Validation{Findings: []Finding{}, Skipped: []Skip{}}
	inventory, err := inspectWithOverlay(root, overlay)
	if err != nil {
		return result, err
	}
	result.Skipped = append(result.Skipped, inventory.Skipped...)
	dir, err := openRoot(root)
	if err != nil {
		return result, fmt.Errorf("open target root: %w", err)
	}
	defer func() { _ = dir.Close() }()
	for _, link := range inventory.Links {
		target := link.Target
		hasFragment := false
		if strings.Contains(target, "?") {
			result.Skipped = append(result.Skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.query-skipped", Reason: "query-bearing link is not checked"})
			continue
		}
		if i := strings.IndexByte(target, '#'); i >= 0 {
			target = target[:i]
			hasFragment = true
		}
		if path.IsAbs(target) || strings.HasPrefix(target, "\\") {
			result.Skipped = append(result.Skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.unsafe-link-skipped", Reason: "absolute local target is not followed"})
			continue
		}
		candidate := path.Clean(path.Join(path.Dir(link.Source), target))
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
			result.Skipped = append(result.Skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.fragment-skipped", Reason: "fragment anchor is not checked"})
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
