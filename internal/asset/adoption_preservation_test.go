package asset

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	rootassets "github.com/procrastivity/lds/assets"
)

const adoptionFixtureRoot = "testdata/adoption"

type preservationFixture struct {
	ApprovedOutputs []string `json:"approvedOutputs"`
	SharedBaseline  struct {
		Fixture     string        `json:"fixture"`
		Facts       string        `json:"facts"`
		TargetFiles []fixtureFile `json:"targetFiles"`
	} `json:"sharedBaseline"`
	Cases []preservationCase `json:"cases"`
}

type preservationCase struct {
	Name                       string        `json:"name"`
	OverlayFiles               []fixtureFile `json:"overlayFiles"`
	Preflight                  string        `json:"preflight"`
	Classifications            []string      `json:"classifications"`
	ProposedCreates            []string      `json:"proposedCreates"`
	VetoAllCreates             bool          `json:"vetoAllCreates"`
	ReportOnly                 []string      `json:"reportOnly"`
	AffectedLinks              []string      `json:"affectedLinks"`
	Provenance                 string        `json:"provenance"`
	NecessaryParentDirectories []string      `json:"necessaryParentDirectories"`
	AbsentLayers               []string      `json:"absentLayers"`
	AbsentLayersAreError       bool          `json:"absentLayersAreError"`
	PredictedWrites            *int          `json:"predictedWrites"`
	Preserved                  []fixtureFile `json:"preserved"`
}

type fixtureFile struct {
	Path   string `json:"path"`
	Source string `json:"source"`
	SHA256 string `json:"sha256"`
}

func TestAdoptionPreservationFixtures(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(adoptionFixtureRoot, "preservation", "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture preservationFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode preservation cases: %v", err)
	}

	outputs := []string{
		"engineering/README.md",
		"engineering/product/overview.md",
		"docs/README.md",
		"docs/reference/usage.md",
	}
	if !equalStrings(fixture.ApprovedOutputs, outputs) {
		t.Fatalf("approved outputs = %v, want %v", fixture.ApprovedOutputs, outputs)
	}
	wantBaselineFiles := []string{"target/sample.txt", "target/word-count.sh"}
	if fixture.SharedBaseline.Fixture != "greenfield" || fixture.SharedBaseline.Facts != "greenfield/facts.json" || len(fixture.SharedBaseline.TargetFiles) != len(wantBaselineFiles) {
		t.Fatalf("shared baseline = %+v, want the Step 7 greenfield target and facts", fixture.SharedBaseline)
	}
	baselineContents := make(map[string][]byte, len(fixture.SharedBaseline.TargetFiles))
	for i, file := range fixture.SharedBaseline.TargetFiles {
		if file.Path != wantBaselineFiles[i] || !safeRelativePath(file.Path) {
			t.Fatalf("shared baseline target file[%d] = %q, want %q", i, file.Path, wantBaselineFiles[i])
		}
		content := readAdoptionFixture(t, file.Source)
		assertFixtureHash(t, file, content)
		baselineContents[file.Path] = content
	}
	var facts greenfieldFacts
	readJSONFixture(t, filepath.Join(adoptionFixtureRoot, filepath.FromSlash(fixture.SharedBaseline.Facts)), &facts)
	if facts.DisplayName == "" || facts.Purpose == "" || facts.CurrentCapability == "" || facts.DisplayNameSource == "" || facts.PurposeSource == "" || facts.CurrentCapabilitySource == "" ||
		facts.Prerequisites == "" || facts.OperationName == "" || facts.OperationSummary == "" || facts.Invocation == "" || facts.Input == "" || facts.OutputEvidence == "" || facts.OperationSource == "" || facts.Limitations == "" {
		t.Fatalf("shared greenfield facts lack required product or verified-operation evidence: %+v", facts)
	}
	if facts.Approval.Approver != "Simulated fixture owner" || !strings.Contains(facts.Approval.Status, "test-only") ||
		!equalStrings(facts.Approval.ApprovedFacts, []string{"DisplayName", "Purpose", "CurrentCapability", "VerifiedUserOperation"}) ||
		facts.Invocation != "sh word-count.sh count sample.txt" || !strings.Contains(facts.OperationSource, "word-count.sh") ||
		!strings.Contains(facts.OperationSource, "sample.txt") || !strings.Contains(facts.OutputEvidence, "Word count: 4") ||
		!strings.Contains(facts.OutputEvidence, "exit status was 0") {
		t.Fatalf("shared baseline must carry test-only approval and sourced/observed operation evidence: %+v", facts)
	}
	renderedOutputs := renderAdoptionOutputs(t, facts)

	want := map[string]struct {
		preflight       string
		classifications []string
		creates         []string
		veto            bool
	}{
		"mixed-root-readme": {
			preflight: "eligible", classifications: []string{"create", "create", "create", "create"}, creates: outputs,
		},
		"recognizable-legacy-under-docs": {
			preflight: "refusal", classifications: []string{"refusal", "refusal", "refusal", "refusal"}, veto: true,
		},
		"partial-legacy-under-docs": {
			preflight: "refusal", classifications: []string{"refusal", "refusal", "refusal", "refusal"}, veto: true,
		},
		"ambiguous-legacy-under-docs": {
			preflight: "refusal", classifications: []string{"refusal", "refusal", "refusal", "refusal"}, veto: true,
		},
		"partial-engineering-missing-layers": {
			preflight: "eligible", classifications: []string{"create", "create", "create", "create"}, creates: outputs,
		},
		"one-different-approved-output": {
			preflight: "collision", classifications: []string{"create", "create", "create", "different-collision"},
			creates: outputs[:3], veto: true,
		},
		"already-generated-identical": {
			preflight: "eligible-no-op", classifications: []string{"identical-untouched", "identical-untouched", "identical-untouched", "identical-untouched"},
		},
		"edited-generated-looking-output": {
			preflight: "collision", classifications: []string{"create", "create", "create", "different-collision"},
			creates: outputs[:3], veto: true,
		},
	}
	if len(fixture.Cases) != len(want) {
		t.Fatalf("fixture has %d cases, want %d", len(fixture.Cases), len(want))
	}

	seen := make(map[string]bool)
	for _, scenario := range fixture.Cases {
		scenario := scenario
		t.Run(scenario.Name, func(t *testing.T) {
			contract, ok := want[scenario.Name]
			if !ok || seen[scenario.Name] {
				t.Fatalf("unexpected or duplicate scenario %q", scenario.Name)
			}
			seen[scenario.Name] = true
			if scenario.Preflight != contract.preflight || !equalStrings(scenario.Classifications, contract.classifications) ||
				!equalStrings(scenario.ProposedCreates, contract.creates) || scenario.VetoAllCreates != contract.veto {
				t.Fatalf("preflight = (%q, %v, creates %v, veto %t), want (%q, %v, creates %v, veto %t)",
					scenario.Preflight, scenario.Classifications, scenario.ProposedCreates, scenario.VetoAllCreates,
					contract.preflight, contract.classifications, contract.creates, contract.veto)
			}
			if scenario.AbsentLayersAreError {
				t.Fatal("missing LDS layers alone must not be a refusal condition")
			}
			if len(scenario.OverlayFiles) == 0 || len(scenario.Preserved) != len(scenario.OverlayFiles) {
				t.Fatalf("overlay and preserved file sets must match: overlays=%v preserved=%v", scenario.OverlayFiles, scenario.Preserved)
			}
			for _, file := range fixture.SharedBaseline.TargetFiles {
				current := readAdoptionFixture(t, file.Source)
				assertFixtureHash(t, file, current)
				if !bytes.Equal(baselineContents[file.Path], current) {
					t.Fatalf("shared baseline target file %q changed in scenario %q", file.Path, scenario.Name)
				}
			}

			initial := make(map[string][]byte, len(scenario.OverlayFiles))
			initialFiles := make(map[string]fixtureFile, len(scenario.OverlayFiles))
			for _, file := range scenario.OverlayFiles {
				if !safeRelativePath(file.Path) || initial[file.Path] != nil {
					t.Fatalf("unsafe or duplicate target path %q", file.Path)
				}
				content := readAdoptionFixture(t, file.Source)
				assertFixtureHash(t, file, content)
				initial[file.Path] = content
				initialFiles[file.Path] = file
			}
			preservedPaths := make(map[string]bool, len(scenario.Preserved))
			for _, file := range scenario.Preserved {
				before, ok := initial[file.Path]
				original := initialFiles[file.Path]
				if !ok || preservedPaths[file.Path] || file.Source != original.Source || file.SHA256 != original.SHA256 {
					t.Fatalf("preserved path %q is absent from initial tree", file.Path)
				}
				preservedPaths[file.Path] = true
				preserved := readAdoptionFixture(t, file.Source)
				assertFixtureHash(t, file, preserved)
				if !bytes.Equal(before, preserved) {
					t.Fatalf("preserved path %q differs byte-for-byte from its initial contents", file.Path)
				}
			}

			switch scenario.Name {
			case "mixed-root-readme":
				if len(scenario.ReportOnly) != 1 || scenario.ReportOnly[0] != "README.md: root signpost opportunity only; never an output" {
					t.Fatalf("root README handling = %v, want report-only signpost", scenario.ReportOnly)
				}
				if !bytes.Contains(initial["README.md"], []byte(facts.Invocation)) || !bytes.Contains(initial["README.md"], []byte("## Contributors")) {
					t.Fatal("mixed README fixture must contain both user and contributor material")
				}
			case "one-different-approved-output", "edited-generated-looking-output":
				if !scenario.VetoAllCreates || len(scenario.AffectedLinks) != 1 || scenario.Provenance == "" {
					t.Fatalf("collision must veto creates and record provenance/affected links: %+v", scenario)
				}
				file := scenario.OverlayFiles[0]
				occupied := readAdoptionFixture(t, file.Source)
				generated := renderedOutputs[file.Path]
				if bytes.Equal(occupied, generated) {
					t.Fatal("collision fixture must have bytes distinct from its approved generated output")
				}
				linkParts := strings.SplitN(scenario.AffectedLinks[0], ": ", 2)
				if len(linkParts) != 2 || linkParts[0] != file.Path || !bytes.Contains(occupied, []byte(linkParts[1])) {
					t.Fatalf("affected link %q is not present in occupied %q bytes", scenario.AffectedLinks[0], file.Path)
				}
			case "already-generated-identical":
				if scenario.PredictedWrites == nil || *scenario.PredictedWrites != 0 {
					t.Fatalf("identical rerun must predict zero writes: %v", scenario.PredictedWrites)
				}
				for i, file := range scenario.OverlayFiles {
					if file.Path != outputs[i] {
						t.Fatalf("identical rerun input path[%d] = %q, want %q", i, file.Path, outputs[i])
					}
					if !bytes.Equal(readAdoptionFixture(t, file.Source), renderedOutputs[file.Path]) {
						t.Fatalf("pre-existing %q must be byte-identical to current rendered template output", file.Path)
					}
				}
			case "partial-engineering-missing-layers":
				wantParents := []string{"engineering/product/", "docs/", "docs/reference/"}
				if !equalStrings(scenario.NecessaryParentDirectories, wantParents) {
					t.Fatalf("necessary parent directories = %v, want %v", scenario.NecessaryParentDirectories, wantParents)
				}
				wantAbsentLayers := []string{"engineering/product/", "engineering/architecture/", "engineering/specs/", "engineering/reference/", "engineering/behaviors/", "engineering/implementation/"}
				if !equalStrings(scenario.AbsentLayers, wantAbsentLayers) {
					t.Fatalf("absent layers = %v, want %v", scenario.AbsentLayers, wantAbsentLayers)
				}
			case "partial-legacy-under-docs":
				wantAbsentLayers := []string{"docs/product/", "docs/architecture/", "docs/specs/", "docs/reference/", "docs/features/", "docs/implementation/"}
				if !equalStrings(scenario.AbsentLayers, wantAbsentLayers) {
					t.Fatalf("absent layers = %v, want %v", scenario.AbsentLayers, wantAbsentLayers)
				}
			case "ambiguous-legacy-under-docs":
				if scenario.Preflight != "refusal" || !scenario.VetoAllCreates || len(scenario.ProposedCreates) != 0 {
					t.Fatalf("ambiguous docs legacy must refuse without proposing creates: %+v", scenario)
				}
			}
		})
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("missing preservation scenario %q", name)
		}
	}
}

func renderAdoptionOutputs(t *testing.T, facts greenfieldFacts) map[string][]byte {
	t.Helper()
	cases := []struct {
		assetPath string
		output    string
	}{
		{"templates/adoption/engineering/README.md", "engineering/README.md"},
		{"templates/adoption/engineering/product/overview.md", "engineering/product/overview.md"},
		{"templates/adoption/docs/README.md", "docs/README.md"},
		{"templates/adoption/docs/reference/usage.md", "docs/reference/usage.md"},
	}
	rendered := make(map[string][]byte, len(cases))
	for _, file := range cases {
		assetBytes, err := rootassets.FS.ReadFile(file.assetPath)
		if err != nil {
			t.Fatalf("read adoption template %q: %v", file.assetPath, err)
		}
		tmpl, err := template.New(file.assetPath).Parse(string(assetBytes))
		if err != nil {
			t.Fatalf("parse adoption template %q: %v", file.assetPath, err)
		}
		var output bytes.Buffer
		if err := tmpl.Execute(&output, facts); err != nil {
			t.Fatalf("render adoption template %q: %v", file.assetPath, err)
		}
		rendered[file.output] = output.Bytes()
	}
	return rendered
}

func readAdoptionFixture(t *testing.T, source string) []byte {
	t.Helper()
	if !safeRelativePath(source) {
		t.Fatalf("unsafe fixture source path %q", source)
	}
	content, err := os.ReadFile(filepath.Join(adoptionFixtureRoot, filepath.FromSlash(source)))
	if err != nil {
		t.Fatalf("read fixture source %q: %v", source, err)
	}
	return content
}

func assertFixtureHash(t *testing.T, file fixtureFile, content []byte) {
	t.Helper()
	got := sha256.Sum256(content)
	if hex.EncodeToString(got[:]) != file.SHA256 {
		t.Fatalf("SHA-256 for %q (%s) = %x, want %s", file.Path, file.Source, got, file.SHA256)
	}
}

func safeRelativePath(path string) bool {
	clean := filepath.Clean(filepath.FromSlash(path))
	return path != "" && !filepath.IsAbs(clean) && clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}
