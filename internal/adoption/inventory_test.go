package adoption

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInspectScopedFilesLinksAndSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	write := func(path, data string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("README.md", "[guide](engineering/product/overview.md)\n[z](z.md) [a](a.md)\n")
	write("notes.md", "[not scoped](missing.md)\n")
	write("engineering/product/overview.md", "[usage][  Guide ]\n[external](https://example.invalid)\n[anchor](#intro)\n\n[guide]: ../../docs/reference/usage.md#run\n")
	write("docs/reference/usage.md", "# Usage\n")
	write("docs/notes.txt", "not Markdown")
	if err := os.WriteFile(filepath.Join(outside, "private.md"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "docs", "linked")); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{"README.md", "docs/reference/usage.md", "engineering/product/overview.md"}
	var paths []string
	for _, file := range got.Files {
		paths = append(paths, file.Path)
	}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("files = %v, want %v", paths, wantPaths)
	}
	content := "[usage][  Guide ]\n[external](https://example.invalid)\n[anchor](#intro)\n\n[guide]: ../../docs/reference/usage.md#run\n"
	if got.Files[2].SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(content))) || got.Files[2].Size != len(content) {
		t.Fatalf("overview metadata = %+v", got.Files[2])
	}
	wantLinks := []Link{
		{Source: "README.md", Line: 1, Target: "engineering/product/overview.md", Kind: "inline"},
		{Source: "README.md", Line: 2, Target: "z.md", Kind: "inline"},
		{Source: "README.md", Line: 2, Target: "a.md", Kind: "inline"},
		{Source: "engineering/product/overview.md", Line: 1, Target: "../../docs/reference/usage.md#run", Kind: "reference"},
		{Source: "engineering/product/overview.md", Line: 3, Target: "#intro", Kind: "inline"},
	}
	if !reflect.DeepEqual(got.Links, wantLinks) {
		t.Fatalf("links = %+v, want %+v", got.Links, wantLinks)
	}
	if len(got.Skipped) != 2 || got.Skipped[0].Source != "docs/linked" || got.Skipped[0].Code != "inventory.symlink-skipped" || got.Skipped[1].Line != 2 {
		t.Fatalf("skipped = %+v, want symlink and external link", got.Skipped)
	}
}

func TestInspectFreshTargetAndSymlinkRoot(t *testing.T) {
	root := t.TempDir()
	got, err := Inspect(root)
	if err != nil || got.Files == nil || got.Links == nil || got.Skipped == nil || len(got.Files)+len(got.Links)+len(got.Skipped) != 0 {
		t.Fatalf("fresh target = %+v, err %v; want nonnil empty arrays", got, err)
	}
	link := filepath.Join(t.TempDir(), "root")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(link); err == nil {
		t.Fatal("symlinked target root must not be followed")
	}
}

func TestExtractLinksDoesNotTreatCodeAsDocumentationLinks(t *testing.T) {
	content := "`[not](missing-a.md)` [real](present.md)\n" +
		"`` ` [not](missing-b.md) ``\n" +
		"~~~sh\n[not](missing-c.md)\n[code-ref]: missing-d.md\n~~~\n" +
		"[ref][code-ref]\n"
	links, skips := extractLinks("docs/guide.md", content)
	want := []Link{{Source: "docs/guide.md", Line: 1, Target: "present.md", Kind: "inline"}}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %+v, want only real prose link %+v", links, want)
	}
	if len(skips) != 1 || skips[0].Line != 7 || skips[0].Code != "inventory.reference-unresolved" {
		t.Fatalf("skips = %+v, want unresolved reference on line 7", skips)
	}
}

func TestExtractLinksRejectsFalseLocalTargets(t *testing.T) {
	content := "    [indented](missing-a.md)\n" +
		"~~~go\n~~~not-a-closing-fence\n[still code](missing-b.md)\n~~~\n" +
		"[titled](present.md \"User guide\") \\[literal](missing-c.md)\n" +
		"[unsupported](nested(path).md)\n"
	links, skips := extractLinks("docs/guide.md", content)
	want := []Link{{Source: "docs/guide.md", Line: 6, Target: "present.md", Kind: "inline"}}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %+v, want %+v", links, want)
	}
	if len(skips) != 1 || skips[0].Line != 7 || skips[0].Code != "inventory.unsupported-link" {
		t.Fatalf("skips = %+v, want one unsupported nested destination", skips)
	}
}

func TestExtractLinksNormalizesAngleDestinations(t *testing.T) {
	content := "[angle](<present.md>) [reference][a]\n[a]: <present.md>\n[unsupported](<broken.md)\n"
	links, skips := extractLinks("docs/links.md", content)
	want := []Link{
		{Source: "docs/links.md", Line: 1, Target: "present.md", Kind: "inline"},
		{Source: "docs/links.md", Line: 1, Target: "present.md", Kind: "reference"},
	}
	if !reflect.DeepEqual(links, want) {
		t.Fatalf("links = %+v, want %+v", links, want)
	}
	if len(skips) != 1 || skips[0].Line != 3 || skips[0].Code != "inventory.unsupported-link" {
		t.Fatalf("skips = %+v, want malformed angle destination skip", skips)
	}
}

func TestInspectRefusesSymlinkInTargetAncestor(t *testing.T) {
	actual := t.TempDir()
	project := filepath.Join(actual, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "README.md"), []byte("private\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(actual, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(filepath.Join(alias, "project")); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("symlink ancestor error = %v, want unsafe target refusal", err)
	}
}
