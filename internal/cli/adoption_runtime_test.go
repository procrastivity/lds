package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/procrastivity/lds/internal/adoption"
)

// These cases execute the built CLI against disposable target copies. The
// expected output bytes and input hashes come from independently authored
// fixtures, not from the runtime renderer or plan.
func TestDocsAdoptionRuntime_PreservationMatrix(t *testing.T) {
	fixture := filepath.Join("..", "asset", "testdata", "adoption")
	data, err := os.ReadFile(filepath.Join(fixture, "preservation", "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var matrix struct {
		ApprovedOutputs []string `json:"approvedOutputs"`
		SharedBaseline  struct {
			TargetFiles []runtimeFixtureFile `json:"targetFiles"`
		} `json:"sharedBaseline"`
		Cases []runtimeScenario `json:"cases"`
	}
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatal(err)
	}
	if len(matrix.Cases) != 8 || len(matrix.ApprovedOutputs) != 4 {
		t.Fatalf("preservation matrix has %d cases and %d outputs", len(matrix.Cases), len(matrix.ApprovedOutputs))
	}
	matrix.Cases = append([]runtimeScenario{{
		Name: "greenfield", Classifications: []string{"create", "create", "create", "create"},
	}}, matrix.Cases...)
	facts := filepath.Join(fixture, "greenfield", "facts.json")
	for _, scenario := range matrix.Cases {
		t.Run(scenario.Name, func(t *testing.T) {
			if len(scenario.Classifications) != len(matrix.ApprovedOutputs) {
				t.Fatalf("%d classifications for %d outputs", len(scenario.Classifications), len(matrix.ApprovedOutputs))
			}
			root := t.TempDir()
			for _, file := range matrix.SharedBaseline.TargetFiles {
				copyRuntimeFixture(t, fixture, root, file, strings.TrimPrefix(file.Path, "target/"))
			}
			for _, file := range scenario.OverlayFiles {
				copyRuntimeFixture(t, fixture, root, file, file.Path)
			}
			before := renderTargetSnapshot(t, root)
			env := []string{"XDG_CONFIG_HOME=" + t.TempDir()}
			planRun := run(t, env, "plumbing", "docs", "plan", root, "--facts", facts, "--json")
			wantExit := 0
			if scenario.VetoAllCreates {
				wantExit = 3
			}
			if planRun.exitCode != wantExit || (wantExit == 0 && planRun.stderr != "") {
				t.Fatalf("plan exit=%d stderr=%q, want %d", planRun.exitCode, planRun.stderr, wantExit)
			}
			if wantExit == 3 {
				code := "refusal.legacy-docs"
				if strings.Contains(scenario.Name, "output") {
					code = "refusal.output-collision"
				}
				if got := parseErrorEnvelope(t, planRun.stderr).Error.Code; got != code {
					t.Fatalf("plan refusal code=%s, want %s", got, code)
				}
			}
			var plan adoption.PlanResult
			if err := json.Unmarshal([]byte(planRun.stdout), &plan); err != nil || plan.PlanDigest == nil || plan.TargetIdentity == "" || len(plan.Outputs) != 4 || plan.Veto != scenario.VetoAllCreates {
				t.Fatalf("plan result=%+v decode=%v", plan, err)
			}
			wantClasses := map[string]string{}
			for i, path := range matrix.ApprovedOutputs {
				wantClasses[path] = scenario.Classifications[i]
			}
			seenPaths := map[string]bool{}
			for _, output := range plan.Outputs {
				classification, approved := wantClasses[output.Path]
				if !approved || seenPaths[output.Path] || output.Classification != classification {
					t.Errorf("%s classified %s, want %s", output.Path, output.Classification, wantClasses[output.Path])
				}
				seenPaths[output.Path] = true
			}
			if len(seenPaths) != len(wantClasses) {
				t.Errorf("plan paths = %v, want exact approved paths %v", seenPaths, matrix.ApprovedOutputs)
			}
			if want := runtimePreserved(before); !reflect.DeepEqual(plan.Preserved, want) {
				t.Errorf("plan preservation evidence = %+v, want %+v", plan.Preserved, want)
			}
			if !reflect.DeepEqual(before, renderTargetSnapshot(t, root)) {
				t.Fatal("read-only plan changed target")
			}
			planPath := filepath.Join(t.TempDir(), "reviewed.json")
			if err := os.WriteFile(planPath, []byte(planRun.stdout), 0o600); err != nil {
				t.Fatal(err)
			}
			appliedRun := run(t, env, "plumbing", "docs", "apply", root, "--facts", facts, "--plan", planPath, "--confirm", *plan.PlanDigest, "--json")
			if appliedRun.exitCode != wantExit || (wantExit == 0 && appliedRun.stderr != "") {
				t.Fatalf("apply exit=%d stderr=%q stdout=%q, want %d", appliedRun.exitCode, appliedRun.stderr, appliedRun.stdout, wantExit)
			}
			var applied adoption.ApplyResult
			if err := json.Unmarshal([]byte(appliedRun.stdout), &applied); err != nil || applied.PlanDigest != *plan.PlanDigest {
				t.Fatalf("apply result=%+v decode=%v", applied, err)
			}
			if scenario.VetoAllCreates {
				if got := parseErrorEnvelope(t, appliedRun.stderr).Error.Code; got != "refusal.plan-veto" {
					t.Errorf("apply refusal code=%s, want refusal.plan-veto", got)
				}
				if applied.Status != "refused" || len(applied.Created) != 0 || len(applied.Preserved) != 0 || !reflect.DeepEqual(before, renderTargetSnapshot(t, root)) {
					t.Fatalf("veto wrote target: %+v", applied)
				}
				verifyRuntimeInputHashes(t, root, matrix.SharedBaseline.TargetFiles, scenario.OverlayFiles)
				return
			}
			wantCreates := 4
			wantStatus := "applied"
			if scenario.Name == "already-generated-identical" {
				wantCreates, wantStatus = 0, "current"
			}
			if applied.Status != wantStatus || len(applied.Created) != wantCreates || len(applied.IdenticalUntouched) != 4-wantCreates || len(applied.Unattempted) != 0 || applied.Failed != nil {
				t.Fatalf("apply actual effects = %+v", applied)
			}
			verifyRuntimeTarget(t, fixture, root, matrix.ApprovedOutputs, before, wantCreates)
			verifyRuntimeReport(t, fixture, applied, matrix.ApprovedOutputs)
			if want := runtimePreserved(before); !reflect.DeepEqual(applied.Preserved, want) {
				t.Errorf("apply preservation evidence = %+v, want %+v", applied.Preserved, want)
			}
			verifyRuntimeInputHashes(t, root, matrix.SharedBaseline.TargetFiles, scenario.OverlayFiles)
			if scenario.Name == "greenfield" || scenario.Name == "mixed-root-readme" {
				command := exec.Command("sh", "word-count.sh", "count", "sample.txt")
				command.Dir = root
				stdout, err := command.Output()
				if err != nil || string(stdout) != "Word count: 4\n" {
					t.Fatalf("documented operation stdout=%q err=%v", stdout, err)
				}
			}
			// A second review of the resulting tree is an identical no-op.
			postApply := renderTargetSnapshot(t, root)
			secondPlan := run(t, env, "plumbing", "docs", "plan", root, "--facts", facts, "--json")
			var identical adoption.PlanResult
			if secondPlan.exitCode != 0 || json.Unmarshal([]byte(secondPlan.stdout), &identical) != nil || identical.PlanDigest == nil {
				t.Fatalf("second plan = %+v", secondPlan)
			}
			if want := runtimePreserved(postApply); !reflect.DeepEqual(identical.Preserved, want) {
				t.Errorf("identical plan preservation evidence = %+v, want %+v", identical.Preserved, want)
			}
			if err := os.WriteFile(planPath, []byte(secondPlan.stdout), 0o600); err != nil {
				t.Fatal(err)
			}
			secondRun := run(t, env, "plumbing", "docs", "apply", root, "--facts", facts, "--plan", planPath, "--confirm", *identical.PlanDigest, "--json")
			var current adoption.ApplyResult
			if secondRun.exitCode != 0 || json.Unmarshal([]byte(secondRun.stdout), &current) != nil || current.Status != "current" || len(current.Created) != 0 || len(current.IdenticalUntouched) != 4 || !reflect.DeepEqual(postApply, renderTargetSnapshot(t, root)) {
				t.Fatalf("identical rerun changed target: run=%+v result=%+v", secondRun, current)
			}
			verifyRuntimeReport(t, fixture, current, matrix.ApprovedOutputs)
			if want := runtimePreserved(postApply); !reflect.DeepEqual(current.Preserved, want) {
				t.Errorf("identical apply preservation evidence = %+v, want %+v", current.Preserved, want)
			}
		})
	}
}

type runtimeFixtureFile struct {
	Path   string `json:"path"`
	Source string `json:"source"`
	SHA256 string `json:"sha256"`
}

type runtimeScenario struct {
	Name            string               `json:"name"`
	OverlayFiles    []runtimeFixtureFile `json:"overlayFiles"`
	Classifications []string             `json:"classifications"`
	VetoAllCreates  bool                 `json:"vetoAllCreates"`
}

func copyRuntimeFixture(t *testing.T, fixture, root string, file runtimeFixtureFile, relative string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(file.Source)))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != file.SHA256 {
		t.Fatalf("independent fixture hash for %s = %s, want %s", file.Source, got, file.SHA256)
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func verifyRuntimeTarget(t *testing.T, fixture, root string, outputs []string, before map[string]string, creates int) {
	t.Helper()
	after := renderTargetSnapshot(t, root)
	added := []string{}
	for key, original := range before {
		if current, exists := after[key]; !exists || current != original {
			t.Errorf("preserved %s changed bytes", key)
		}
	}
	allowedNewDirs := map[string]bool{}
	for _, output := range outputs {
		for parent := filepath.Dir(output); parent != "."; parent = filepath.Dir(parent) {
			allowedNewDirs[filepath.ToSlash(parent)] = true
		}
	}
	for key := range after {
		if _, existed := before[key]; !existed {
			switch {
			case strings.HasPrefix(key, "file:"):
				added = append(added, strings.TrimPrefix(key, "file:"))
			case strings.HasPrefix(key, "dir:") && allowedNewDirs[strings.TrimPrefix(key, "dir:")]:
			default:
				t.Errorf("unapproved target path added: %s", key)
			}
		}
	}
	sort.Strings(added)
	if len(added) != creates {
		t.Fatalf("new file paths = %v, want %d approved creates", added, creates)
	}
	wantAdded := []string{}
	for _, output := range outputs {
		want, err := os.ReadFile(filepath.Join(fixture, "greenfield", "expected", filepath.FromSlash(output)))
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(output)))
		if err != nil || !reflect.DeepEqual(actual, want) {
			t.Errorf("actual %s differs from independent expected bytes: %v", output, err)
		}
		if _, existed := before["file:"+output]; !existed {
			wantAdded = append(wantAdded, output)
		}
	}
	sort.Strings(wantAdded)
	if !reflect.DeepEqual(added, wantAdded) {
		t.Fatalf("new files = %v, want approved paths %v", added, wantAdded)
	}
}

func verifyRuntimeReport(t *testing.T, fixture string, applied adoption.ApplyResult, outputs []string) {
	t.Helper()
	want := make(map[string]string, len(outputs))
	for _, path := range outputs {
		data, err := os.ReadFile(filepath.Join(fixture, "greenfield", "expected", filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		want[path] = hex.EncodeToString(sum[:])
	}
	seen := map[string]bool{}
	for _, entry := range append(applied.Created, applied.IdenticalUntouched...) {
		hash, approved := want[entry.Path]
		if !approved || entry.SHA256 != hash || seen[entry.Path] {
			t.Errorf("reported path/hash mismatch or duplicate: %+v", entry)
		}
		seen[entry.Path] = true
	}
	if len(seen) != len(want) {
		t.Errorf("reported paths %v, want all approved paths %v", seen, outputs)
	}
}

func runtimePreserved(snapshot map[string]string) []adoption.PreservedPath {
	preserved := []adoption.PreservedPath{}
	for key, contents := range snapshot {
		if !strings.HasPrefix(key, "file:") {
			continue
		}
		sum := sha256.Sum256([]byte(contents))
		preserved = append(preserved, adoption.PreservedPath{
			Path: strings.TrimPrefix(key, "file:"), SHA256: hex.EncodeToString(sum[:]), Size: len(contents),
		})
	}
	sort.Slice(preserved, func(i, j int) bool { return preserved[i].Path < preserved[j].Path })
	return preserved
}

func verifyRuntimeInputHashes(t *testing.T, root string, baseline, overlays []runtimeFixtureFile) {
	t.Helper()
	for _, file := range append(baseline, overlays...) {
		path := strings.TrimPrefix(file.Path, "target/")
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Errorf("preserved input %s: %v", path, err)
			continue
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != file.SHA256 {
			t.Errorf("preserved input %s hash = %s, want %s", path, got, file.SHA256)
		}
	}
}
