package adoption

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func FuzzReviewMarkdown(f *testing.F) {
	for _, data := range [][]byte{
		{},
		[]byte("# Anchor\n# Anchor-1\n# Anchor\n"),
		[]byte("```md\n[hidden](missing.md)\n```\n[ref][id]\n[id]: known.md#a-b\n"),
		[]byte("# Café\n<a id=\"anchor\"></a>\n[x](#%FF)\n"),
		[]byte("[escape](../../outside.md)\n[alias](alias.md#anchor)\n"),
		[]byte("\\`[x](missing.md)\\`\n[image](<known.md#a-b>)\n"),
		[]byte("\xff\x00\r\n"), bytes.Repeat([]byte("# Anchor\n"), 455),
	} {
		f.Add(data)
	}
	root, outside := f.TempDir(), f.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		f.Fatal(err)
	}
	for name, content := range map[string]string{
		filepath.Join(outside, "sentinel.md"): "# Outside\n",
		filepath.Join(root, "docs/known.md"):  "# A B\n## A B-1\n# A B\n",
		filepath.Join(root, "README.md"): "[encoded](docs/known.md#a%2Db)\n" +
			"[collision](docs/known.md#a-b-2)\n[absent](docs/known.md#absent)\n" +
			"[missing](docs/missing.md)\n[external](https://example.invalid/)\n" +
			"[escape](../outside.md)\n[alias](docs/alias.md)\n[input](docs/input.md#anchor)\n",
	} {
		if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
			f.Fatal(err)
		}
	}
	alias := filepath.Join(root, "docs/alias.md")
	sentinel := filepath.Join(outside, "sentinel.md")
	if err := os.Symlink(sentinel, alias); err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			t.Skip()
		}
		if err := os.WriteFile(filepath.Join(root, "docs/input.md"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		before := snapshotRenderTarget(t, root)
		inventory, validation, err := Review(root)
		if !reflect.DeepEqual(before, snapshotRenderTarget(t, root)) {
			t.Fatal("review changed target or linked sentinel bytes")
		}
		if target, err := os.Readlink(alias); err != nil || target != sentinel {
			t.Fatal("review changed the symlink")
		}
		// Arbitrary links can produce filesystem errors (e.g. NAME_MAX).
		// Even those runs must leave the disposable tree unchanged.
		if err != nil {
			if !errors.Is(err, syscall.ENAMETOOLONG) && !errors.Is(err, syscall.EINVAL) {
				t.Fatalf("unexpected review error: %v", err)
			}
			return
		}
		if len(inventory.Files) != 3 {
			t.Fatalf("inventory followed a symlink or lost a document: %+v", inventory.Files)
		}
		for _, file := range inventory.Files {
			if file.Path == "docs/input.md" && (file.Size != len(data) || file.SHA256 != fmt.Sprintf("%x", sha256.Sum256(data))) {
				t.Fatal("inventory did not preserve input byte identity")
			}
		}
		findings, skips := map[int]string{}, map[int]string{}
		for _, finding := range validation.Findings {
			if finding.Path == "README.md" && finding.Line < 8 {
				findings[finding.Line] = finding.Code
			}
		}
		for _, skip := range validation.Skipped {
			if skip.Source == "README.md" && skip.Line < 8 {
				skips[skip.Line] = skip.Code
			}
		}
		if !reflect.DeepEqual(findings, map[int]string{3: "validation.local-anchor-missing", 4: "validation.local-target-missing"}) ||
			!reflect.DeepEqual(skips, map[int]string{5: "inventory.external-skipped", 6: "validation.unsafe-link-skipped", 7: "validation.unsafe-link-skipped"}) {
			t.Fatalf("unrelated input changed supported checks or honest skips: findings=%v skips=%v", findings, skips)
		}
	})
}

func TestValidateLocalFilesAndHonestSkips(t *testing.T) {
	root := t.TempDir()
	write := func(name, data string) {
		t.Helper()
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("engineering/decisions/0001.md", "# Heading\n")
	write("notes.md", "# Heading\n")
	write("docs/links.md", "[existing](../engineering/decisions/0001.md#heading)\n"+
		"[missing](reference/missing.md#install)\n"+
		"[external](https://example.invalid/docs)\n"+
		"[fragment](#overview)\n# Overview\n"+
		"[escape](../../outside.md)\n"+
		"[query](reference/usage.md?version=1)\n"+
		"[symlink](reference/alias.md)\n"+
		"[directory](../engineering/decisions/)\n"+
		"[outside inventory](../notes.md#heading)\n")
	if err := os.MkdirAll(filepath.Join(root, "docs/reference"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "engineering/decisions/0001.md"), filepath.Join(root, "docs/reference/alias.md")); err != nil {
		t.Fatal(err)
	}
	got, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []Finding{{Code: "validation.local-target-missing", Path: "docs/links.md", Line: 2, Target: "reference/missing.md#install", Message: "local file target is missing: reference/missing.md#install"}}
	if !reflect.DeepEqual(got.Findings, want) {
		t.Fatalf("findings = %+v, want %+v", got.Findings, want)
	}
	wantCodes := []string{"inventory.external-skipped", "validation.unsafe-link-skipped", "validation.query-skipped", "validation.unsafe-link-skipped", "validation.not-file-skipped", "validation.fragment-skipped", "inventory.symlink-skipped"}
	var codes []string
	for _, skip := range got.Skipped {
		codes = append(codes, skip.Code)
	}
	if !reflect.DeepEqual(codes, wantCodes) {
		t.Fatalf("skip codes = %v, want %v; skips=%+v", codes, wantCodes, got.Skipped)
	}
}

func TestValidateFreshAndPartialTrees(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"", "engineering"} {
		if dir != "" {
			if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		got, err := Validate(root)
		if err != nil || got.Findings == nil || got.Skipped == nil || len(got.Findings)+len(got.Skipped) != 0 {
			t.Fatalf("validate with optional layer %q = %+v, err=%v", dir, got, err)
		}
	}
}

func TestValidateHeadingAnchorsAndHonestUnsupportedTargets(t *testing.T) {
	for _, tc := range []struct {
		name, content           string
		valid, missing, skipped []string
	}{
		{"duplicate collisions", "# Run\n## Run-1\n### Run\n# Run\n", []string{"run", "run-1", "run-2", "run-3"}, []string{"run-4", "run-1-1"}, nil},
		{"numbered heading collision", "# Run\n## Run\n# Run-1\n", []string{"run", "run-1", "run-1-1"}, []string{"run-2"}, nil},
		{"many duplicates", strings.Repeat("# Run\n", 20000), []string{"run", "run-19999"}, []string{"run-20000"}, nil},
		{"plain and code", "# API `read_file`!\r\n  ## A  B ###\n### Hello, World!\nUse `<root>` as a placeholder.\n~~~sh\n# Hidden\n~~~\n", []string{"api-read_file", "a--b", "hello-world"}, []string{"hidden", "api-read-file", "a-b"}, nil},
		{"invalid fence is prose", "```not`a`fence\n# Visible\n", []string{"visible"}, []string{"hidden"}, nil},
		{"encoding and case", "# Some Header\n", []string{"some%2Dheader"}, []string{"Some-header", "some+header", "some?header", "some:header"}, []string{"", "%GG", "%00", "%1b", "%FF"}},
		{"source line references", "# L8\n", []string{"l8"}, []string{"L8-other", "l8-l14"}, []string{"L8", "L8-L14", "L%38-L14"}},
		{"formatted heading", "# Run\n## *Run*\n", nil, nil, []string{"run", "run-1", "missing"}},
		{"complex code span", "# Run\n## ``Run``\n", nil, nil, []string{"run", "run-1"}},
		{"tab in heading", "# Run\n## A\tB\n", nil, nil, []string{"run", "ab"}},
		{"unicode heading", "# Café\n", nil, nil, []string{"caf%C3%A9", "missing"}},
		{"html anchors", "# Run\n<a id=\"custom\"></a>\n", nil, nil, []string{"run", "custom", "missing"}},
		{"html beside escaped backticks", "# Run\n\\`<a id=\"custom\"></a>\\`\n", nil, nil, []string{"run", "custom"}},
		{"setext heading", "Run\n===\n", nil, nil, []string{"run", "missing"}},
		{"nested heading", "# Run\n> ## Quoted\n", nil, nil, []string{"run", "quoted"}},
		{"nested setext heading", "# Run\n> Quoted\n> ===\n", nil, nil, []string{"run", "quoted"}},
		{"indented heading", "# Run\n    # Ambiguous\n", nil, nil, []string{"run", "ambiguous"}},
		{"invalid utf8", "# Run\n\xff\n", nil, nil, []string{"run"}},
		{"size boundary", "# Limit\n" + strings.Repeat("x", (1<<20)-8), []string{"limit"}, nil, nil},
		{"over size boundary", "# Limit\n" + strings.Repeat("x", (1<<20)-7), nil, nil, []string{"limit"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "docs/guide.md"), []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			var links strings.Builder
			for _, group := range [][]string{tc.valid, tc.missing, tc.skipped} {
				for _, fragment := range group {
					links.WriteString("[section](docs/guide.md#" + fragment + ")\n")
				}
			}
			links.WriteString("# Root\n[self](#root)\n[absent](#absent)\n")
			if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(links.String()), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := Validate(root)
			if err != nil || len(got.Findings) != len(tc.missing)+1 || len(got.Skipped) != len(tc.skipped) {
				t.Fatalf("validation=%+v err=%v", got, err)
			}
			wantTargets := append(append([]string{}, tc.missing...), "#absent")
			for i, finding := range got.Findings {
				want := wantTargets[i]
				if i < len(tc.missing) {
					want = "docs/guide.md#" + want
				}
				if finding.Code != "validation.local-anchor-missing" || finding.Path != "README.md" || finding.Target != want {
					t.Fatalf("finding=%+v want target=%s", finding, want)
				}
			}
			for _, skip := range got.Skipped {
				if skip.Code != "validation.fragment-skipped" || skip.Reason == "" {
					t.Fatalf("unsupported target not explained: %+v", skip)
				}
			}
		})
	}
}
