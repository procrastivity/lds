package adoption

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/procrastivity/lds/internal/asset"
)

type preservationOverlayFile struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

func TestRender_PreservationComparisonsUseCurrentTemplates(t *testing.T) {
	useIsolatedAssetDirectories(t)
	fixtureRoot := filepath.Join("..", "asset", "testdata", "adoption")
	greenfield := filepath.Join(fixtureRoot, "greenfield")
	outputs, err := Render(filepath.Join(greenfield, "target"), filepath.Join(greenfield, "facts.json"))
	if err != nil {
		t.Fatalf("render shared greenfield facts: %v", err)
	}
	wantPaths := []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"}
	if len(outputs) != len(wantPaths) {
		t.Fatalf("Render returned %d outputs, want %d", len(outputs), len(wantPaths))
	}
	rendered := make(map[string][]byte, len(outputs))
	for i, output := range outputs {
		if output.Path != wantPaths[i] {
			t.Fatalf("rendered path[%d] = %q, want %q", i, output.Path, wantPaths[i])
		}
		if output.Asset.Source != asset.SourceEmbedded.String() {
			t.Fatalf("rendered %q from %q, want current embedded asset", output.Path, output.Asset.Source)
		}
		rendered[output.Path] = []byte(output.Content)
	}

	data, err := os.ReadFile(filepath.Join(fixtureRoot, "preservation", "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases struct {
		Cases []struct {
			Name         string                    `json:"name"`
			OverlayFiles []preservationOverlayFile `json:"overlayFiles"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("decode preservation cases: %v", err)
	}

	seenIdentical, seenCollision, seenEdited := false, false, false
	for _, scenario := range cases.Cases {
		switch scenario.Name {
		case "already-generated-identical":
			seenIdentical = true
			if len(scenario.OverlayFiles) != len(rendered) {
				t.Fatalf("identical overlay has %d files, want %d", len(scenario.OverlayFiles), len(rendered))
			}
			for _, file := range scenario.OverlayFiles {
				current, ok := rendered[file.Path]
				if !ok {
					t.Fatalf("identical overlay has unapproved output path %q", file.Path)
				}
				occupied := readPreservationOverlay(t, fixtureRoot, file.Source)
				if !bytes.Equal(occupied, current) {
					t.Errorf("identical overlay %q differs from a fresh render of current assets", file.Path)
				}
			}
		case "one-different-approved-output":
			seenCollision = true
			assertCurrentUsageDiffers(t, fixtureRoot, rendered, scenario.OverlayFiles)
		case "edited-generated-looking-output":
			seenEdited = true
			assertCurrentUsageDiffers(t, fixtureRoot, rendered, scenario.OverlayFiles)
		}
	}
	if !seenIdentical || !seenCollision || !seenEdited {
		t.Fatalf("preservation cases missing: identical=%t collision=%t edited=%t", seenIdentical, seenCollision, seenEdited)
	}
}

func assertCurrentUsageDiffers(t *testing.T, fixtureRoot string, rendered map[string][]byte, files []preservationOverlayFile) {
	t.Helper()
	if len(files) != 1 || files[0].Path != "docs/reference/usage.md" {
		t.Fatalf("usage collision overlay = %+v, want one occupied docs/reference/usage.md", files)
	}
	occupied := readPreservationOverlay(t, fixtureRoot, files[0].Source)
	current, ok := rendered[files[0].Path]
	if !ok {
		t.Fatalf("collision overlay has unapproved output path %q", files[0].Path)
	}
	if bytes.Equal(occupied, current) {
		t.Fatalf("occupied %q bytes must differ from a fresh render of current assets", files[0].Path)
	}
}

func readPreservationOverlay(t *testing.T, fixtureRoot, source string) []byte {
	t.Helper()
	path := filepath.Clean(filepath.Join(fixtureRoot, filepath.FromSlash(source)))
	if !filepath.IsLocal(filepath.FromSlash(source)) {
		t.Fatalf("unsafe preservation overlay source %q", source)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read preservation overlay %q: %v", source, err)
	}
	return data
}
