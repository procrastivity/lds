// Package adoption contains the read-only target inspection used by docs plumbing.
package adoption

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

// File describes one regular Markdown file in the bounded target scope.
type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}

// Link describes a supported local inline or reference Markdown link.
type Link struct {
	Source string `json:"source"`
	Line   int    `json:"line"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

// Skip explains a target or syntax that was deliberately not checked.
type Skip struct {
	Source string `json:"source"`
	Line   int    `json:"line,omitempty"`
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

// Inventory is the sorted read-only file and link report.
type Inventory struct {
	Files   []File `json:"files"`
	Links   []Link `json:"links"`
	Skipped []Skip `json:"skipped"`
}

var (
	// This is a deliberately bounded subset of Markdown, not a full parser.
	linkPattern       = regexp.MustCompile(`(!?)\[([^]\n]+)\](?:\(([^)\n]*)\)|\[([^]\n]*)\])`)
	definitionPattern = regexp.MustCompile(`^\s{0,3}\[([^]\n]+)\]:\s*(\S+)`)
	titledTarget      = regexp.MustCompile(`^(\S+)(?:\s+(?:"[^"]*"|'[^']*'))?$`)
)

// ErrUnsafeTarget marks a path that cannot be traversed without following a symlink.
var ErrUnsafeTarget = errors.New("unsafe target path")

// Inspect reads only root README.md and regular Markdown below engineering/
// and docs/. It never follows a symlink, including a symlink at the root.
func Inspect(root string) (Inventory, error) {
	result := Inventory{Files: []File{}, Links: []Link{}, Skipped: []Skip{}}
	dir, err := openRoot(root)
	if err != nil {
		return result, fmt.Errorf("inspect target root: %w", err)
	}
	defer func() { _ = dir.Close() }()
	var visit func(*os.File, string, string) error
	visit = func(parent *os.File, name, rel string) error {
		file, err := openChild(parent, name)
		if errors.Is(err, os.ErrNotExist) {
			return nil // optional layers and README are absent on fresh targets
		}
		if errors.Is(err, ErrUnsafeTarget) {
			result.Skipped = append(result.Skipped, Skip{Source: rel, Code: "inventory.symlink-skipped", Reason: "symlink not followed"})
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect %s: %w", rel, err)
		}
		defer func() { _ = file.Close() }()
		info, err := file.Stat()
		if err != nil {
			return fmt.Errorf("stat %s: %w", rel, err)
		}
		if rel == "README.md" && info.IsDir() {
			result.Skipped = append(result.Skipped, Skip{Source: rel, Code: "inventory.not-file-skipped", Reason: "root README.md is not a regular file"})
			return nil
		}
		if info.IsDir() {
			entries, err := file.ReadDir(-1)
			if err != nil {
				return fmt.Errorf("read %s: %w", rel, err)
			}
			for _, entry := range entries {
				if err := visit(file, entry.Name(), rel+"/"+entry.Name()); err != nil {
					return err
				}
			}
			return nil
		}
		if rel == "engineering" || rel == "docs" {
			result.Skipped = append(result.Skipped, Skip{Source: rel, Code: "inventory.not-directory-skipped", Reason: "documentation layer is not a directory"})
			return nil
		}
		if !info.Mode().IsRegular() || !strings.HasSuffix(strings.ToLower(rel), ".md") {
			return nil
		}
		data, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		result.Files = append(result.Files, File{Path: rel, SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Size: len(data)})
		links, skipped := extractLinks(rel, string(data))
		result.Links = append(result.Links, links...)
		result.Skipped = append(result.Skipped, skipped...)
		return nil
	}
	for _, rel := range []string{"README.md", "engineering", "docs"} {
		if err := visit(dir, rel, rel); err != nil {
			return result, err
		}
	}
	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].Path < result.Files[j].Path })
	sort.SliceStable(result.Links, func(i, j int) bool {
		a, b := result.Links[i], result.Links[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Line < b.Line // same-line links retain their source order
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

func extractLinks(source, content string) ([]Link, []Skip) {
	lines := proseLines(content)
	definitions := map[string]string{}
	for _, line := range lines {
		if m := definitionPattern.FindStringSubmatch(line); m != nil {
			key := strings.ToLower(strings.Join(strings.Fields(m[1]), " "))
			if _, exists := definitions[key]; !exists {
				definitions[key] = m[2]
			}
		}
	}
	links := []Link{}
	skipped := []Skip{}
	for lineNumber, line := range lines {
		if definitionPattern.MatchString(line) {
			continue
		}
		for _, span := range linkPattern.FindAllStringIndex(line, -1) {
			if escapedBracket(line, span[0]) {
				continue
			}
			m := linkPattern.FindStringSubmatch(line[span[0]:span[1]])
			target, kind := m[3], "inline"
			if m[4] != "" || strings.HasSuffix(m[0], "[]") {
				kind = "reference"
				key := m[4]
				if key == "" {
					key = m[2]
				}
				target = definitions[strings.ToLower(strings.Join(strings.Fields(key), " "))]
				if target == "" {
					skipped = append(skipped, Skip{Source: source, Line: lineNumber + 1, Code: "inventory.reference-unresolved", Reason: "reference definition not found"})
					continue
				}
			}
			if m[1] == "!" {
				skipped = append(skipped, Skip{Source: source, Line: lineNumber + 1, Code: "inventory.image-skipped", Reason: "image targets are not checked as links"})
				continue
			}
			if kind == "inline" {
				parts := titledTarget.FindStringSubmatch(target)
				if parts == nil || strings.Contains(parts[1], "(") {
					skipped = append(skipped, Skip{Source: source, Line: lineNumber + 1, Code: "inventory.unsupported-link", Reason: "inline destination syntax is not supported"})
					continue
				}
				target = parts[1]
			}
			if strings.HasPrefix(target, "<") || strings.HasSuffix(target, ">") {
				if len(target) < 3 || target[0] != '<' || target[len(target)-1] != '>' || strings.ContainsAny(target[1:len(target)-1], "<>") {
					skipped = append(skipped, Skip{Source: source, Line: lineNumber + 1, Code: "inventory.unsupported-link", Reason: "angle-bracket destination syntax is not supported"})
					continue
				}
				target = target[1 : len(target)-1]
			}
			if strings.HasPrefix(target, "#") || target == "" {
				skipped = append(skipped, Skip{Source: source, Line: lineNumber + 1, Code: "inventory.fragment-only-skipped", Reason: "fragment-only target is not checked"})
				continue
			}
			if strings.Contains(target, ":") || strings.HasPrefix(target, "//") {
				skipped = append(skipped, Skip{Source: source, Line: lineNumber + 1, Code: "inventory.external-skipped", Reason: "external URL is not fetched"})
				continue
			}
			links = append(links, Link{Source: source, Line: lineNumber + 1, Target: target, Kind: kind})
		}
	}
	return links, skipped
}

func escapedBracket(line string, start int) bool {
	if line[start] == '!' {
		start++
	}
	backslashes := 0
	for i := start - 1; i >= 0 && line[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

// proseLines excludes fenced code from link extraction. It deliberately does
// not claim to recognize every Markdown construct; unsupported syntax is
// outside the bounded inline/reference-link contract.
func proseLines(content string) []string {
	lines := strings.Split(content, "\n")
	var fence byte
	var fenceLength int
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if fence == 0 && (strings.HasPrefix(line, "\t") || len(line)-len(trimmed) >= 4) {
			lines[i] = ""
			continue
		}
		if len(line)-len(trimmed) <= 3 && len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
			run := 0
			for run < len(trimmed) && trimmed[run] == trimmed[0] {
				run++
			}
			if run >= 3 && (fence == 0 || fence == trimmed[0] && run >= fenceLength && strings.TrimSpace(trimmed[run:]) == "") {
				if fence == 0 {
					fence, fenceLength = trimmed[0], run
				} else {
					fence, fenceLength = 0, 0
				}
				lines[i] = ""
				continue
			}
		}
		if fence != 0 {
			lines[i] = ""
		} else {
			lines[i] = withoutCodeSpans(line)
		}
	}
	return lines
}

func withoutCodeSpans(line string) string {
	var out strings.Builder
	for i := 0; i < len(line); {
		if line[i] != '`' {
			out.WriteByte(line[i])
			i++
			continue
		}
		end := i
		for end < len(line) && line[end] == '`' {
			end++
		}
		closing := -1
		for j := end; j < len(line); j++ {
			if line[j] != '`' {
				continue
			}
			k := j
			for k < len(line) && line[k] == '`' {
				k++
			}
			if k-j == end-i {
				closing = k
				break
			}
			j = k - 1
		}
		if closing < 0 {
			out.WriteString(line[i:end])
			i = end
		} else {
			out.WriteByte(' ')
			i = closing
		}
	}
	return out.String()
}
