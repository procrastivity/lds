package asset_test

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	rootassets "github.com/procrastivity/lds/assets"
	"github.com/procrastivity/lds/internal/asset"
)

const shippedName = "flows/adopt.md"

func projectAsset(root, name string) string {
	return filepath.Join(root, ".lds", "assets", filepath.FromSlash(name))
}

func writeAsset(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func checkResolved(t *testing.T, got asset.Resolved, err error, source asset.Source, path, data string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != source || got.Path != path || string(got.Bytes()) != data {
		t.Fatalf("resolved = (%s, %q, %q), want (%s, %q, %q)", got.Source, got.Path, got.Bytes(), source, path, data)
	}
}

func canonical(t *testing.T, path string) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveInExactRootsAndFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, b := t.TempDir(), t.TempDir()
	localA, localB := projectAsset(a, shippedName), projectAsset(b, shippedName)
	xdg := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "lds", shippedName)
	writeAsset(t, localA, "A")
	writeAsset(t, localB, "B")
	writeAsset(t, xdg, "user")
	for _, tc := range []struct{ root, content, path string }{
		{a, "A", localA}, {b, "B", localB}, {a, "A", localA},
	} {
		got, err := asset.ResolveIn(tc.root, shippedName)
		checkResolved(t, got, err, asset.SourceProject, canonical(t, tc.path), tc.content)
	}
	got, err := asset.Resolve(shippedName)
	checkResolved(t, got, err, asset.SourceOverride, xdg, "user")
	got, err = asset.ResolveIn("", shippedName)
	checkResolved(t, got, err, asset.SourceOverride, xdg, "user")

	// Relative selected roots and symlink aliases resolve to canonical provenance.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(cwd, a)
	if err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(rel, shippedName)
	checkResolved(t, got, err, asset.SourceProject, canonical(t, localA), "A")
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(a, alias); err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(alias, shippedName)
	checkResolved(t, got, err, asset.SourceProject, canonical(t, localA), "A")

	// The project asset is opaque, even when empty or invalid for its consumer.
	for _, data := range []string{"", "[invalid: yaml"} {
		writeAsset(t, localA, data)
		got, err = asset.ResolveIn(a, shippedName)
		checkResolved(t, got, err, asset.SourceProject, canonical(t, localA), data)
	}
	if err := os.Remove(localA); err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(a, shippedName)
	checkResolved(t, got, err, asset.SourceOverride, xdg, "user")
	// No discovery from a child directory, and no state cached across calls.
	child := filepath.Join(b, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(child, shippedName)
	checkResolved(t, got, err, asset.SourceOverride, xdg, "user")
	if err := os.Remove(xdg); err != nil {
		t.Fatal(err)
	}
	embedded, err := rootassets.FS.ReadFile(shippedName)
	if err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(a, shippedName)
	checkResolved(t, got, err, asset.SourceEmbedded, "", string(embedded))
	got, err = asset.ResolveIn(b, shippedName)
	checkResolved(t, got, err, asset.SourceProject, canonical(t, localB), "B")
}

func TestResolveInNamesAndRoots(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	for _, name := range []string{"local-only.md", "assets.go", "flows", "CONTRACT.md"} {
		writeAsset(t, projectAsset(root, name), "project")
		xdg := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "lds", name)
		writeAsset(t, xdg, "user")
		if _, err := asset.ResolveIn(root, name); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("ResolveIn(%q) = %v, want unknown shipped name", name, err)
		}
		got, err := asset.Resolve(name)
		checkResolved(t, got, err, asset.SourceOverride, xdg, "user")
	}
	for _, name := range []string{"", ".", "..", "../flows/adopt.md", "/flows/adopt.md", "flows/../adopt.md", "./flows/adopt.md", "flows//adopt.md", "flows/./adopt.md", "flows/", `flows\adopt.md`, `C:/x`, `//server/x`, "x\x00y"} {
		if _, err := asset.ResolveIn(root, name); !errors.Is(err, fs.ErrInvalid) {
			t.Errorf("ResolveIn(%q) = %v, want invalid name", name, err)
		}
	}
	if _, err := asset.ResolveIn(filepath.Join(root, "missing"), shippedName); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing selected root: %v", err)
	}
	fileRoot := filepath.Join(root, "file")
	writeAsset(t, fileRoot, "not directory")
	if _, err := asset.ResolveIn(fileRoot, shippedName); !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("file selected root: %v", err)
	}
	for i, source := range []asset.Source{asset.SourceOverride, asset.SourceDefault, asset.SourceEmbedded, asset.SourceProject} {
		if int(source) != i || source.String() != []string{"override", "default", "embedded", "project"}[i] {
			t.Errorf("source %d = (%d, %q)", i, source, source.String())
		}
	}
}

func TestResolveInKeepsStrictOverrideErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	override := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "lds", shippedName)
	writeAsset(t, override, "user")
	for _, kind := range []string{"directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			if err := os.Remove(override); err != nil {
				t.Fatal(err)
			}
			if kind == "directory" {
				if err := os.Mkdir(override, 0o755); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Symlink(filepath.Join(root, "missing-target"), override); err != nil {
					t.Fatal(err)
				}
			}
			for _, resolve := range []struct {
				name string
				call func() (asset.Resolved, error)
			}{
				{"Resolve", func() (asset.Resolved, error) { return asset.Resolve(shippedName) }},
				{"ResolveIn empty root", func() (asset.Resolved, error) { return asset.ResolveIn("", shippedName) }},
				{"ResolveIn selected root", func() (asset.Resolved, error) { return asset.ResolveIn(root, shippedName) }},
			} {
				if _, err := resolve.call(); err == nil || !strings.Contains(err.Error(), "is not a regular file") {
					t.Errorf("%s: %v, want strict override error", resolve.name, err)
				}
			}
			if _, present, err := asset.ReadOverride(shippedName); present || err == nil || !strings.Contains(err.Error(), "is not a regular file") {
				t.Errorf("ReadOverride: present=%t, err=%v", present, err)
			}
		})
		if err := os.Remove(override); err != nil {
			t.Fatal(err)
		}
		writeAsset(t, override, "user")
	}
}

func TestResolveInRejectsSymlinksAndNonregularPaths(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeAsset(t, filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "lds", shippedName), "fallback must not mask refusal")
	for _, component := range []string{".lds", ".lds/assets", ".lds/assets/flows", ".lds/assets/" + shippedName} {
		for _, kind := range []string{"inside link", "outside link", "broken link", "nonregular"} {
			t.Run(component+"/"+kind, func(t *testing.T) {
				root := t.TempDir()
				path := filepath.Join(root, filepath.FromSlash(component))
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if kind == "nonregular" {
					if component == ".lds/assets/"+shippedName {
						if err := os.Mkdir(path, 0o755); err != nil {
							t.Fatal(err)
						}
					} else {
						writeAsset(t, path, "file instead of directory")
					}
				} else {
					target := filepath.Join(root, "target")
					if kind == "outside link" {
						target = filepath.Join(t.TempDir(), "target")
					}
					if kind != "broken link" {
						remaining := strings.TrimPrefix(".lds/assets/"+shippedName, component+"/")
						if component == ".lds/assets/"+shippedName {
							remaining = ""
						}
						writeAsset(t, filepath.Join(target, filepath.FromSlash(remaining)), "linked content")
					}
					if err := os.Symlink(target, path); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := asset.ResolveIn(root, shippedName); !errors.Is(err, fs.ErrInvalid) {
					t.Fatalf("unsafe project path: %v, want invalid", err)
				}
			})
		}
	}
}

// Run with an actual installed layout so public API tests do not modify the
// package's executable hook or depend on an installation on the host.
func TestResolveInInstalledOnlyAndPrecedence(t *testing.T) {
	if os.Getenv("LDS_ASSET_TEST_CHILD") != t.Name() {
		prefix := canonical(t, t.TempDir())
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		binary, err := os.ReadFile(exe)
		if err != nil {
			t.Fatal(err)
		}
		bin := filepath.Join(prefix, "bin", "lds")
		if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(bin, binary, 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(bin, "-test.run=^"+t.Name()+"$")
		cmd.Env = append(os.Environ(), "LDS_ASSET_TEST_CHILD="+t.Name(), "LDS_ASSET_TEST_PREFIX="+prefix)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("installed-layout child: %v\n%s", err, out)
		}
		return
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	prefix := os.Getenv("LDS_ASSET_TEST_PREFIX")
	root := t.TempDir()
	local := projectAsset(root, shippedName)
	xdg := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "lds", shippedName)
	share := filepath.Join(prefix, "share", "lds", "assets", shippedName)
	writeAsset(t, local, "project")
	writeAsset(t, xdg, "user")
	writeAsset(t, share, "installed")
	got, err := asset.ResolveIn(root, shippedName)
	checkResolved(t, got, err, asset.SourceProject, canonical(t, local), "project")
	if err := os.Remove(local); err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(root, shippedName)
	checkResolved(t, got, err, asset.SourceOverride, xdg, "user")
	if err := os.Remove(xdg); err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(root, shippedName)
	checkResolved(t, got, err, asset.SourceDefault, share, "installed")
	if err := os.Remove(share); err != nil {
		t.Fatal(err)
	}
	embedded, err := rootassets.FS.ReadFile(shippedName)
	if err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(root, shippedName)
	checkResolved(t, got, err, asset.SourceEmbedded, "", string(embedded))

	const installedOnly = "templates/installed-only.txt"
	installed := filepath.Join(prefix, "share", "lds", "assets", installedOnly)
	writeAsset(t, installed, "installed-only")
	local = projectAsset(root, installedOnly)
	writeAsset(t, local, "project for installed-only name")
	got, err = asset.ResolveIn(root, installedOnly)
	checkResolved(t, got, err, asset.SourceProject, canonical(t, local), "project for installed-only name")
	if err := os.Remove(local); err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(root, installedOnly)
	checkResolved(t, got, err, asset.SourceDefault, installed, "installed-only")
}
