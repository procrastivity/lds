package adoption

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const planFixtureDir = "../asset/testdata/adoption"

type preservationFixture struct {
	ApprovedOutputs []string `json:"approvedOutputs"`
	SharedBaseline  struct {
		TargetFiles []struct {
			Path   string `json:"path"`
			Source string `json:"source"`
			SHA256 string `json:"sha256"`
		} `json:"targetFiles"`
	} `json:"sharedBaseline"`
	Cases []struct {
		Name         string `json:"name"`
		OverlayFiles []struct {
			Path   string `json:"path"`
			Source string `json:"source"`
			SHA256 string `json:"sha256"`
		} `json:"overlayFiles"`
		Classifications  []string `json:"classifications"`
		VetoAllCreates   bool     `json:"vetoAllCreates"`
		NecessaryParents []string `json:"necessaryParentDirectories"`
		ReportOnly       []string `json:"reportOnly"`
	} `json:"cases"`
}

func TestPlan_PreservationFixtureCases(t *testing.T) {
	useIsolatedAssetDirectories(t)
	fixture := readPreservationFixture(t)
	if len(fixture.Cases) != 8 || len(fixture.ApprovedOutputs) != 4 {
		t.Fatalf("fixture has %d cases and %d output paths; want eight and four", len(fixture.Cases), len(fixture.ApprovedOutputs))
	}
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			root := createPreservationTarget(t, fixture, tc.OverlayFiles)
			factsPath := filepath.Join(planFixtureDir, "greenfield", "facts.json")
			if _, facts, err := readRenderFacts(factsPath); err != nil || facts.OutputEvidence == "" || facts.OperationSource == "" || facts.Invocation == "" {
				t.Fatalf("shared greenfield operation evidence is not complete: facts=%+v err=%v", facts, err)
			}
			before := snapshotPlanTarget(t, root)
			plan, err := Plan(root, factsPath, "")
			if tc.VetoAllCreates {
				var refusal *PlanRefusal
				if !errors.As(err, &refusal) {
					t.Fatalf("Plan error = %v, want policy refusal", err)
				}
			} else if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			if plan.SchemaVersion != 1 || plan.Operation != "docs plan" || plan.TargetRoot != root || plan.PlanDigest == nil {
				t.Fatalf("plan header/digest = %+v", plan.PlanProjection)
			}
			if got, err := CanonicalPlanDigest(plan); err != nil || got != *plan.PlanDigest {
				t.Fatalf("plan digest = %v (%v), want %q", got, err, *plan.PlanDigest)
			}
			if plan.Veto != tc.VetoAllCreates || len(plan.Outputs) != len(fixture.ApprovedOutputs) {
				t.Fatalf("veto/output count = %t/%d", plan.Veto, len(plan.Outputs))
			}
			wantClassifications := make(map[string]string, len(fixture.ApprovedOutputs))
			for i, outputPath := range fixture.ApprovedOutputs {
				wantClassifications[outputPath] = tc.Classifications[i]
			}
			for _, output := range plan.Outputs {
				if want := wantClassifications[output.Path]; output.Classification != want {
					t.Errorf("%s classification = %q, want %q", output.Path, output.Classification, want)
				}
				if output.Classification == "create" && (output.Content == "" || output.ContentSHA256 != sha256Hex([]byte(output.Content)) || output.Asset == nil) {
					t.Errorf("%s create lacks exact rendered bytes/hash/asset evidence: %+v", output.Path, output)
				}
			}
			if tc.NecessaryParents != nil {
				wantParents := append([]string(nil), tc.NecessaryParents...)
				sort.Strings(wantParents)
				if !reflect.DeepEqual(plan.NecessaryParentDirectories, wantParents) {
					t.Errorf("necessary parents = %v, want %v", plan.NecessaryParentDirectories, wantParents)
				}
			}
			if tc.Name == "mixed-root-readme" && (len(plan.ReportOnly) != 1 || plan.ReportOnly[0].Path != "README.md") {
				t.Errorf("report-only notices = %+v, want root README signpost only", plan.ReportOnly)
			}
			if tc.Name != "mixed-root-readme" && len(plan.ReportOnly) != 0 {
				t.Errorf("unexpected report-only notices: %+v", plan.ReportOnly)
			}
			for _, baseline := range fixture.SharedBaseline.TargetFiles {
				path := strings.TrimPrefix(baseline.Path, "target/")
				preserved, ok := preservedByPath(plan.Preserved, path)
				if !ok || preserved.SHA256 != baseline.SHA256 {
					t.Errorf("baseline %s preservation = %+v, present=%t; want hash %s", path, preserved, ok, baseline.SHA256)
				}
			}
			for _, overlay := range tc.OverlayFiles {
				preserved, ok := preservedByPath(plan.Preserved, overlay.Path)
				if !ok || preserved.SHA256 != overlay.SHA256 {
					t.Errorf("overlay %s preservation = %+v, present=%t; want hash %s", overlay.Path, preserved, ok, overlay.SHA256)
				}
			}
			if tc.Name == "one-different-approved-output" || tc.Name == "edited-generated-looking-output" {
				collision := findPlanOutput(t, plan, "docs/reference/usage.md")
				if collision.Classification != "different-collision" || collision.ExistingSHA256 == "" || collision.ExistingContent == "" || collision.ExistingProvenance == "" || len(collision.AffectedLinks) == 0 {
					t.Errorf("collision review evidence = %+v", collision)
				}
			}
			if after := snapshotPlanTarget(t, root); !reflect.DeepEqual(before, after) {
				t.Fatalf("Plan changed target bytes/tree\nbefore=%v\nafter=%v", before, after)
			}
		})
	}
}

func TestPlan_IncompleteFactsRefusesEveryOutputWithoutDigest(t *testing.T) {
	useIsolatedAssetDirectories(t)
	data, err := os.ReadFile(filepath.Join(planFixtureDir, "greenfield", "facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	fields["OutputEvidence"] = ""
	data, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	factsPath := filepath.Join(t.TempDir(), "incomplete.json")
	if err := os.WriteFile(factsPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(t.TempDir(), factsPath, "")
	var refusal *PlanRefusal
	if !errors.As(err, &refusal) || refusal.Code != planIncompleteFacts {
		t.Fatalf("Plan error = %v, want incomplete-facts refusal", err)
	}
	if plan.PlanDigest != nil || !plan.Veto || len(plan.Outputs) != 4 || len(plan.Findings) == 0 {
		t.Fatalf("incomplete facts result = %+v", plan)
	}
	for _, output := range plan.Outputs {
		if output.Classification != "refusal" || output.Content != "" || output.ContentSHA256 != "" || output.Asset != nil {
			t.Errorf("incomplete facts output = %+v", output)
		}
	}
}

func TestPlan_LinkDiagnosticsAreVirtualAndDoNotVeto(t *testing.T) {
	useIsolatedAssetDirectories(t)
	fixture := readPreservationFixture(t)
	root := createPreservationTarget(t, fixture, nil)
	facts := filepath.Join(planFixtureDir, "greenfield", "facts.json")
	baseline, err := Plan(root, facts, "")
	if err != nil {
		t.Fatalf("baseline Plan: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	markdown := "# Local links\n[missing](reference/missing.md#install)\n[external](https://example.invalid/docs)\n[anchor](#overview)\n"
	if err := os.WriteFile(filepath.Join(root, "docs", "legacy-links.md"), []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := Plan(root, facts, "")
	if err != nil {
		t.Fatalf("Plan with diagnostics: %v", err)
	}
	if after.Veto || len(after.Findings) != 1 || after.Findings[0].Path != "docs/legacy-links.md" || after.Findings[0].Line != 2 {
		t.Fatalf("link findings/veto = %+v/%t", after.Findings, after.Veto)
	}
	var external, anchor bool
	for _, skipped := range after.Skipped {
		external = external || skipped.Code == "inventory.external-skipped" && strings.Contains(skipped.Reason, "not fetched")
		anchor = anchor || skipped.Code == "validation.fragment-skipped" && strings.Contains(skipped.Reason, "not checked")
	}
	if !external || !anchor {
		t.Fatalf("external/anchor skips not reported distinctly: %+v", after.Skipped)
	}
	if after.PlanDigest == nil || baseline.PlanDigest == nil || *after.PlanDigest == *baseline.PlanDigest {
		t.Fatalf("diagnostics did not change digest: before=%v after=%v", baseline.PlanDigest, after.PlanDigest)
	}
	// Both navigation links from the proposed hubs resolve through the
	// virtual overlay even though none of those files exists on disk.
	for _, finding := range after.Findings {
		if finding.Path == "docs/README.md" || finding.Path == "engineering/README.md" {
			t.Errorf("prospective navigation link reported missing: %+v", finding)
		}
	}
}

func TestCanonicalJSON_RFC8785AcceptanceVector(t *testing.T) {
	input := []byte(`{"veto":false,"schemaVersion":1,"operation":"docs plan"}`)
	canonical, err := canonicalJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"operation":"docs plan","schemaVersion":1,"veto":false}`
	if string(canonical) != want {
		t.Fatalf("canonical JSON = %s, want %s", canonical, want)
	}
	sum := sha256.Sum256(canonical)
	if got := hex.EncodeToString(sum[:]); got != "009aebfc79bf51ce3fd5fdbd951ec4977e840e053267adc26a1db37558ef1f64" {
		t.Fatalf("RFC 8785 vector SHA-256 = %s", got)
	}
	ordered, err := canonicalJSON([]byte(`{"":1,"𐀀":2,"html":"<>&","line":"a\nb"}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(ordered), `"𐀀"`) > strings.Index(string(ordered), `""`) || strings.Contains(string(ordered), `\u003c`) {
		t.Fatalf("canonical UTF-16 key/string encoding = %s", ordered)
	}
	controls, err := canonicalJSON([]byte(`{"text":"\u0001\u007f\u0085"}`))
	if err != nil || string(controls) != "{\"text\":\"\\u0001\x7f\u0085\"}" {
		t.Fatalf("RFC 8785 escapes only U+0000..U+001F: %q, %v", controls, err)
	}
}

func TestPlanDigestCommitsToEveryDisplayedField(t *testing.T) {
	useIsolatedAssetDirectories(t)
	fixture := readPreservationFixture(t)
	root := createPreservationTarget(t, fixture, nil)
	base, err := Plan(root, filepath.Join(planFixtureDir, "greenfield", "facts.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*PlanResult)
	}{
		{"schema version", func(p *PlanResult) { p.SchemaVersion++ }},
		{"operation", func(p *PlanResult) { p.Operation += " changed" }},
		{"root", func(p *PlanResult) { p.TargetRoot += "/changed" }},
		{"root identity", func(p *PlanResult) { p.TargetIdentity += ":changed" }},
		{"facts", func(p *PlanResult) { p.FactsDigest += "x" }},
		{"output path", func(p *PlanResult) { p.Outputs[0].Path += ".changed" }},
		{"output content", func(p *PlanResult) { p.Outputs[0].Content += "x" }},
		{"output hash", func(p *PlanResult) { p.Outputs[0].ContentSHA256 += "x" }},
		{"asset name", func(p *PlanResult) { p.Outputs[0].Asset.RelativeName += ".changed" }},
		{"asset source", func(p *PlanResult) { p.Outputs[0].Asset.Source += ".changed" }},
		{"asset hash", func(p *PlanResult) { p.Outputs[0].Asset.SHA256 += "x" }},
		{"classification", func(p *PlanResult) { p.Outputs[0].Classification = "refusal" }},
		{"reason", func(p *PlanResult) { p.Outputs[0].Reason += "x" }},
		{"existing hash", func(p *PlanResult) { p.Outputs[0].ExistingSHA256 = "existing" }},
		{"existing content", func(p *PlanResult) { p.Outputs[0].ExistingContent = "existing" }},
		{"existing content base64", func(p *PlanResult) { p.Outputs[0].ExistingContentBase64 = "AA==" }},
		{"existing content encoding", func(p *PlanResult) { p.Outputs[0].ExistingContentEncoding = "base64" }},
		{"existing provenance", func(p *PlanResult) { p.Outputs[0].ExistingProvenance = "unknown" }},
		{"affected links", func(p *PlanResult) {
			p.Outputs[0].AffectedLinks = append(p.Outputs[0].AffectedLinks, Link{Source: "docs/x.md", Line: 1, Target: "missing.md", Kind: "inline"})
		}},
		{"parents", func(p *PlanResult) { p.NecessaryParentDirectories = append(p.NecessaryParentDirectories, "new/") }},
		{"existing parent identity", func(p *PlanResult) {
			p.ExistingParentDirectories = append(p.ExistingParentDirectories, ParentIdentity{Path: "docs/", Identity: "other"})
		}},
		{"preserved path", func(p *PlanResult) { p.Preserved[0].Path += ".changed" }},
		{"preserved hash", func(p *PlanResult) { p.Preserved[0].SHA256 += "x" }},
		{"preserved size", func(p *PlanResult) { p.Preserved[0].Size++ }},
		{"report only", func(p *PlanResult) {
			p.ReportOnly = append(p.ReportOnly, ReportOnlyNotice{Path: "README.md", Message: "notice"})
		}},
		{"veto", func(p *PlanResult) { p.Veto = true }},
		{"finding code", func(p *PlanResult) {
			p.Findings = append(p.Findings, PlanFinding{Code: "new.code", Message: "missing"})
		}},
		{"finding location", func(p *PlanResult) {
			p.Findings = append(p.Findings, PlanFinding{Code: "new.code", Path: "docs/x.md", Line: 2, Target: "target.md", Message: "missing"})
		}},
		{"finding message", func(p *PlanResult) {
			p.Findings = append(p.Findings, PlanFinding{Code: "new.code", Message: "changed"})
		}},
		{"skipped source", func(p *PlanResult) {
			p.Skipped = append(p.Skipped, Skip{Source: "docs/x.md", Code: "new.code", Reason: "unchecked"})
		}},
		{"skipped line", func(p *PlanResult) {
			p.Skipped = append(p.Skipped, Skip{Source: "docs/x.md", Line: 2, Code: "new.code", Reason: "unchecked"})
		}},
		{"skipped code and reason", func(p *PlanResult) {
			p.Skipped = append(p.Skipped, Skip{Source: "docs/x.md", Code: "new.code", Reason: "changed"})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cloned := clonePlan(t, base)
			tc.mutate(&cloned)
			got, err := CanonicalPlanDigest(cloned)
			if err != nil {
				t.Fatal(err)
			}
			if got == *base.PlanDigest {
				t.Fatalf("changing %s did not change digest", tc.name)
			}
		})
	}
	changedDigest := clonePlan(t, base)
	changedValue := "changed self digest"
	changedDigest.PlanDigest = &changedValue
	got, err := CanonicalPlanDigest(changedDigest)
	if err != nil || got != *base.PlanDigest {
		t.Fatalf("digest included itself: got=%q err=%v, want %q", got, err, *base.PlanDigest)
	}
}

func readPreservationFixture(t *testing.T) preservationFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(planFixtureDir, "preservation", "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture preservationFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func createPreservationTarget(t *testing.T, fixture preservationFixture, overlays []struct {
	Path   string `json:"path"`
	Source string `json:"source"`
	SHA256 string `json:"sha256"`
},
) string {
	t.Helper()
	root := t.TempDir()
	files := make([]struct {
		path, source, wantHash string
	}, 0, len(fixture.SharedBaseline.TargetFiles)+len(overlays))
	for _, file := range fixture.SharedBaseline.TargetFiles {
		files = append(files, struct {
			path, source, wantHash string
		}{strings.TrimPrefix(file.Path, "target/"), filepath.Join("greenfield", "target", filepath.Base(file.Source)), file.SHA256})
	}
	for _, file := range overlays {
		files = append(files, struct {
			path, source, wantHash string
		}{file.Path, filepath.FromSlash(file.Source), file.SHA256})
	}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(planFixtureDir, file.source))
		if err != nil {
			t.Fatalf("read fixture source %s: %v", file.source, err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != file.wantHash {
			t.Fatalf("fixture %s hash=%s, want %s", file.source, got, file.wantHash)
		}
		destination := filepath.Join(root, filepath.FromSlash(file.path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func snapshotPlanTarget(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			files["dir:"+filepath.ToSlash(relative)] = ""
			return nil
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		files["file:"+filepath.ToSlash(relative)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func preservedByPath(paths []PreservedPath, wanted string) (PreservedPath, bool) {
	for _, item := range paths {
		if item.Path == wanted {
			return item, true
		}
	}
	return PreservedPath{}, false
}

func findPlanOutput(t *testing.T, plan PlanResult, wanted string) PlanOutput {
	t.Helper()
	for _, output := range plan.Outputs {
		if output.Path == wanted {
			return output
		}
	}
	t.Fatalf("plan has no output %s", wanted)
	return PlanOutput{}
}

func clonePlan(t *testing.T, plan PlanResult) PlanResult {
	t.Helper()
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	var result PlanResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestPlan_VirtualOverlayDoesNotWrite(t *testing.T) {
	useIsolatedAssetDirectories(t)
	fixture := readPreservationFixture(t)
	root := createPreservationTarget(t, fixture, nil)
	before := snapshotPlanTarget(t, root)
	plan, err := Plan(root, filepath.Join(planFixtureDir, "greenfield", "facts.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, snapshotPlanTarget(t, root)) {
		t.Fatal("Plan wrote files or directories to target")
	}
	for _, output := range plan.Outputs {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(output.Path))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("planned output %s exists after read-only plan (stat err=%v)", output.Path, err)
		}
	}
}

func TestPlan_RequiresVerifiedOwnerOperationEvidence(t *testing.T) {
	useIsolatedAssetDirectories(t)
	data, err := os.ReadFile(filepath.Join(planFixtureDir, "greenfield", "facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	var facts renderFactBundle
	if err := json.Unmarshal(data, &facts); err != nil {
		t.Fatal(err)
	}
	if facts.OutputEvidence == "" || facts.OperationSource == "" || facts.Invocation != "sh word-count.sh count sample.txt" {
		t.Fatalf("greenfield fixture is missing concrete operation evidence: %+v", facts)
	}
	command := exec.Command("sh", "word-count.sh", "count", "sample.txt")
	command.Dir = filepath.Join(planFixtureDir, "greenfield", "target")
	output, err := command.CombinedOutput()
	if err != nil || string(output) != "Word count: 4\n" {
		t.Fatalf("verified fixture operation output=%q err=%v", output, err)
	}
}

func TestPlan_FactsDigestHashesExactInputBytes(t *testing.T) {
	useIsolatedAssetDirectories(t)
	factsPath := filepath.Join(planFixtureDir, "greenfield", "facts.json")
	root := t.TempDir()
	data, err := os.ReadFile(factsPath)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(root, factsPath, "")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if plan.FactsDigest != "sha256:"+hex.EncodeToString(sum[:]) {
		t.Fatalf("facts digest = %s, want hash of exact file bytes", plan.FactsDigest)
	}
	if len(data) == 0 {
		t.Fatal("fixture facts unexpectedly empty")
	}
}

func TestPlan_ReplacedByteIdenticalRootChangesDigest(t *testing.T) {
	useIsolatedAssetDirectories(t)
	parent := t.TempDir()
	root := filepath.Join(parent, "target")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	const sample = "alpha beta gamma delta\n"
	if err := os.WriteFile(filepath.Join(root, "sample.txt"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	facts := filepath.Join(planFixtureDir, "greenfield", "facts.json")
	first, err := Plan(root, facts, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, filepath.Join(parent, "old-target")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.txt"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := Plan(root, facts, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.TargetIdentity == "" || second.TargetIdentity == "" || first.TargetIdentity == second.TargetIdentity || *first.PlanDigest == *second.PlanDigest {
		t.Fatalf("replaced target retained review identity/digest: %s/%s and %s/%s", first.TargetIdentity, *first.PlanDigest, second.TargetIdentity, *second.PlanDigest)
	}
	if !reflect.DeepEqual(first.Preserved, second.Preserved) || !reflect.DeepEqual(first.Outputs, second.Outputs) {
		t.Fatal("replacement regression did not retain byte-identical reviewed file evidence")
	}
}

func TestPlan_ReplacedEmptyOutputParentChangesDigest(t *testing.T) {
	useIsolatedAssetDirectories(t)
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.Mkdir(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	facts := filepath.Join(planFixtureDir, "greenfield", "facts.json")
	first, err := Plan(root, facts, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(docs, filepath.Join(root, "docs.old")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	second, err := Plan(root, facts, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.TargetIdentity != second.TargetIdentity || len(first.ExistingParentDirectories) != 1 || len(second.ExistingParentDirectories) != 1 || first.ExistingParentDirectories[0].Identity == second.ExistingParentDirectories[0].Identity || *first.PlanDigest == *second.PlanDigest {
		t.Fatalf("replacement parent retained reviewed digest: before=%+v after=%+v", first.ExistingParentDirectories, second.ExistingParentDirectories)
	}
	if !reflect.DeepEqual(first.Outputs, second.Outputs) || !reflect.DeepEqual(first.Preserved, second.Preserved) {
		t.Fatal("parent replacement changed file bytes/classifications instead of only parent identity")
	}
}

func TestPlan_RecognizesInstalledLDSUnderDocs(t *testing.T) {
	for _, tc := range []struct{ name, path, content string }{
		{"installed guide", "docs/DOCUMENTATION-GUIDE.md", "# Documentation Guide\n\nThis guide documents the project's installed Layered Documentation System (LDS) configuration.\n\n| `DOCS_DIR` | The project documentation root — `docs/` in this repo |\n"},
		{"accepted decision", "docs/decisions/0001-adopt-layered-documentation-system.md", "# ADR-0001: Adopt the Layered Documentation System\n\n**Status**: accepted\n\n## Decision\n\nAdopt the Layered Documentation System (LDS) with six layers.\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			useIsolatedAssetDirectories(t)
			root := t.TempDir()
			file := filepath.Join(root, filepath.FromSlash(tc.path))
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			plan, err := Plan(root, filepath.Join(planFixtureDir, "greenfield", "facts.json"), "")
			var refusal *PlanRefusal
			if !errors.As(err, &refusal) || refusal.Code != planLegacyDocs || !plan.Veto || plan.PlanDigest == nil {
				t.Fatalf("installed LDS under docs/ not refused: error=%v plan=%+v", err, plan.PlanProjection)
			}
			for _, output := range plan.Outputs {
				if output.Classification != "refusal" {
					t.Errorf("legacy target proposes %s as %s", output.Path, output.Classification)
				}
			}
		})
	}
}
