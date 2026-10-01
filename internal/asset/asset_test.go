package asset

import (
	"os"
	"path/filepath"
	"testing"

	rootassets "github.com/procrastivity/lds/assets"
)

func TestResolveAdoptionTemplateResolutionOrderAndShadowing(t *testing.T) {
	originalExecutable := executable
	t.Cleanup(func() { executable = originalExecutable })

	paths := []string{
		"flows/adopt.md",
		"templates/adoption/engineering/README.md",
		"templates/adoption/engineering/product/overview.md",
		"templates/adoption/docs/README.md",
		"templates/adoption/docs/reference/usage.md",
	}
	sources := []struct {
		name       string
		wantSource Source
	}{
		{name: "override", wantSource: SourceOverride},
		{name: "installed-default", wantSource: SourceDefault},
		{name: "embedded", wantSource: SourceEmbedded},
	}

	for _, relativeName := range paths {
		t.Run(relativeName, func(t *testing.T) {
			embedded, err := rootassets.FS.ReadFile(relativeName)
			if err != nil {
				t.Fatalf("read embedded source: %v", err)
			}

			for _, source := range sources {
				t.Run(source.name, func(t *testing.T) {
					xdg := t.TempDir()
					t.Setenv("XDG_CONFIG_HOME", xdg)
					prefix := t.TempDir()
					fakeExe := fakeExecutable(t, prefix)
					executable = func() (string, error) { return fakeExe, nil }

					defaultPath := filepath.Join(prefix, "share", "lds", "assets", relativeName)
					overridePath := filepath.Join(xdg, "lds", relativeName)
					expected := string(embedded)
					switch source.wantSource {
					case SourceOverride:
						writeResolutionFile(t, defaultPath, "DEFAULT-ONLY:"+relativeName+"\n")
						expected = "OVERRIDE-ONLY:" + relativeName + "\n"
						writeResolutionFile(t, overridePath, expected)
					case SourceDefault:
						expected = "INSTALLED-DEFAULT:" + relativeName + "\n"
						writeResolutionFile(t, defaultPath, expected)
					}

					got, err := Resolve(relativeName)
					if err != nil {
						t.Fatalf("Resolve: %v", err)
					}
					if got.Source != source.wantSource || string(got.Bytes()) != expected {
						t.Fatalf("Resolve = (%s, %q), want (%s, %q)", got.Source, got.Bytes(), source.wantSource, expected)
					}
				})
			}
		})
	}
}

func fakeExecutable(t *testing.T, prefix string) string {
	t.Helper()
	path := filepath.Join(prefix, "bin", "lds")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fake bin dir: %v", err)
	}
	if err := os.WriteFile(path, nil, 0o755); err != nil {
		t.Fatalf("create fake executable: %v", err)
	}
	return path
}

func writeResolutionFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %q: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}
