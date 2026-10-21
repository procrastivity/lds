//go:build linux

package adoption

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestUpdateReplacePreservesReviewedModeAndOwnership(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(root, "docs", "note.md")
	if err := os.WriteFile(name, []byte("before\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	before, parents, err := updateRead(root, "docs/note.md")
	if err != nil {
		t.Fatal(err)
	}
	if before.Content != "before\n" || before.Mode != 0o640 || before.Metadata != updateMetadata || len(parents) != 1 || parents[0].Path != "docs/" {
		t.Fatalf("unexpected state: %+v, %+v", before, parents)
	}
	called := false
	result, err := updateReplace(root, "docs/note.md", before, "after\n", parents, func() error { called = true; return nil })
	if err != nil || !called || !result.Renamed || result.Phase != "complete" || result.Scratch != "" || result.Observed == nil {
		t.Fatalf("replacement: %+v, %v", result, err)
	}
	got, _, err := updateRead(root, "docs/note.md")
	if err != nil || got.Content != "after\n" || got.Mode != before.Mode || got.UID != before.UID || got.GID != before.GID || got.Identity == before.Identity || got != *result.Observed {
		t.Fatalf("observed: %+v, %v", got, err)
	}
}

func TestUpdateRefusesHardlinksAndXattrs(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, "README.md")
	if err := os.WriteFile(name, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Link(name, alias); err != nil {
		t.Fatal(err)
	}
	if _, _, err := updateRead(root, "README.md"); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("hardlink not refused: %v", err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := unix.Setxattr(name, "user.lds-test", []byte("value"), 0); errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EOPNOTSUPP) {
		t.Skip("filesystem does not support user xattrs")
	} else if err != nil {
		t.Fatal(err)
	}
	if _, _, err := updateRead(root, "README.md"); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("xattr not refused: %v", err)
	}
}

func TestUpdateCallbackFailureCleansOnlyStage(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, "README.md")
	if err := os.WriteFile(name, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, parents, err := updateRead(root, "README.md")
	if err != nil {
		t.Fatal(err)
	}
	result, err := updateReplace(root, "README.md", before, "after", parents, func() error { return ErrUnsafeTarget })
	if !errors.Is(err, ErrUnsafeTarget) || result.Renamed || result.Scratch != "" || result.Phase != "recheck-callback" {
		t.Fatalf("callback failure: %+v, %v", result, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "README.md" {
		t.Fatalf("stage leaked: %+v, %v", entries, err)
	}
	data, err := os.ReadFile(name)
	if err != nil || string(data) != "before" {
		t.Fatalf("target changed: %q, %v", data, err)
	}
	if _, _, err := updateRead(root, "../README.md"); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("escape allowed: %v", err)
	}
	if _, _, err := updateRead(root, "x\\README.md"); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("backslash allowed: %v", err)
	}
}

func TestUpdateRefusesChangedAncestorAndSymlinkLeaf(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "docs")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(parent, "note.md")
	if err := os.WriteFile(name, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, parents, err := updateRead(root, "docs/note.md")
	if err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(root, "old-docs")
	if err := os.Rename(parent, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := updateReplace(root, "docs/note.md", before, "after", parents, func() error { t.Fatal("callback reached"); return nil })
	if !errors.Is(err, ErrUnsafeTarget) || result.Renamed || result.Scratch != "" {
		t.Fatalf("ancestor replacement not refused: %+v, %v", result, err)
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(old, "note.md"), name); err != nil {
		t.Fatal(err)
	}
	if _, _, err := updateRead(root, "docs/note.md"); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("symlink leaf not refused: %v", err)
	}
}
