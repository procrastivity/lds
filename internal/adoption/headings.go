package adoption

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxHeadingBytes = 1 << 20

var (
	atxHeading      = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*)|$)`)
	closingHashes   = regexp.MustCompile(`[ \t]+#+[ \t]*$`)
	setextUnderline = regexp.MustCompile(`^ {0,3}(?:=+|-+)[ \t]*$`)
	containerHeader = regexp.MustCompile(`^[ \t]*(?:>[ \t]*|[-+*][ \t]+|[0-9]+[.)][ \t]+)+(?:#{1,6}(?:[ \t]|$)|[=-]+[ \t]*$)`)
	htmlMarkup      = regexp.MustCompile(`<[A-Za-z!/?]|-->`)
	lineFragment    = regexp.MustCompile(`^L[0-9]+(?:-L[0-9]+)?$`)
)

type headingIndex struct {
	anchors map[string]bool
	reason  string
}

// readHeadings reads only an inventoried Markdown target, with no-follow
// traversal and a separate byte limit. Update previews use their overlay, not
// the on-disk before bytes. Drift from the inventoried bytes remains unchecked.
func readHeadings(root *os.File, file File, overlay map[string]string) headingIndex {
	current := root
	for _, component := range strings.Split(file.Path, "/") {
		next, err := openChild(current, component)
		if current != root {
			_ = current.Close()
		}
		if err != nil {
			return headingIndex{reason: "Markdown target could not be read safely"}
		}
		current = next
	}
	defer func() { _ = current.Close() }()
	info, err := current.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return headingIndex{reason: "Markdown target is no longer a regular file"}
	}
	var data []byte
	if content, ok := overlay[file.Path]; ok {
		data = []byte(content)
	} else {
		data, err = io.ReadAll(io.LimitReader(current, maxHeadingBytes+1))
		if err != nil {
			return headingIndex{reason: "Markdown target could not be read safely"}
		}
	}
	if len(data) > maxHeadingBytes || !utf8.Valid(data) {
		return headingIndex{reason: "heading checks require UTF-8 Markdown of at most 1 MiB"}
	}
	if sha256Hex(data) != file.SHA256 {
		return headingIndex{reason: "Markdown target changed since inventory"}
	}
	return indexHeadings(string(data))
}

// indexHeadings implements a conservative GitHub-style ASCII ATX subset.
// An unsupported heading can change duplicate numbering; custom HTML can
// introduce other anchors. Either makes the entire target index uncertain.
func indexHeadings(content string) headingIndex {
	index := headingIndex{anchors: map[string]bool{}}
	suffixes := map[string]int{}
	previous := ""
	for _, line := range blockLines(content) {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)
		if htmlMarkup.MatchString(withoutCodeSpans(line)) ||
			strings.Contains(line, "\\`") && htmlMarkup.MatchString(line) || containerHeader.MatchString(line) ||
			(strings.HasPrefix(line, "\t") || len(line)-len(strings.TrimLeft(line, " ")) >= 4) && atxHeading.MatchString(trimmed) ||
			setextUnderline.MatchString(trimmed) && previous != "" {
			return headingIndex{reason: "HTML, container, indented or Setext headings are outside the supported ATX subset"}
		}
		match := atxHeading.FindStringSubmatch(line)
		if match == nil {
			previous = trimmed
			continue
		}
		previous = ""
		title := strings.TrimSpace(closingHashes.ReplaceAllString(" "+match[2], ""))
		base, supported := headingSlug(title)
		if !supported {
			return headingIndex{reason: "heading text uses unsupported Unicode, formatting, entities or code-span syntax"}
		}
		anchor := base
		for index.anchors[anchor] {
			suffixes[base]++
			anchor = fmt.Sprintf("%s-%d", base, suffixes[base])
		}
		index.anchors[anchor] = true
	}
	return index
}

func headingSlug(title string) (string, bool) {
	var slug strings.Builder
	code := false
	for i := 0; i < len(title); i++ {
		c := title[i]
		if c >= 0x80 || c < 0x20 {
			return "", false
		}
		if c == '`' {
			if i+1 < len(title) && title[i+1] == '`' || !code && (i+1 == len(title) || title[i+1] == ' ') || code && (i == 0 || title[i-1] == ' ') {
				return "", false
			}
			code = !code
			continue
		}
		if !code && strings.ContainsRune(`*_[]~\<&`, rune(c)) {
			return "", false
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		switch {
		case c == ' ':
			slug.WriteByte('-')
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_':
			slug.WriteByte(c)
		}
	}
	return slug.String(), !code
}
