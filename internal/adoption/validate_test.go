package adoption

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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
	write("engineering/decisions/0001.md", "# Decision\n")
	write("docs/links.md", "[existing](../engineering/decisions/0001.md#heading)\n"+
		"[missing](reference/missing.md#install)\n"+
		"[external](https://example.invalid/docs)\n"+
		"[fragment](#overview)\n"+
		"[escape](../../outside.md)\n"+
		"[query](reference/usage.md?version=1)\n"+
		"[symlink](reference/alias.md)\n"+
		"[directory](../engineering/decisions/)\n")
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
	wantCodes := []string{"validation.fragment-skipped", "inventory.nonlocal-skipped", "inventory.nonlocal-skipped", "validation.unsafe-link-skipped", "validation.query-skipped", "validation.unsafe-link-skipped", "validation.not-file-skipped", "inventory.symlink-skipped"}
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
