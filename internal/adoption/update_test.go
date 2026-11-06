//go:build linux

package adoption

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/procrastivity/lds/internal/ldserr"
)

func updateFixture(t *testing.T) (string, AssessmentResult, UpdateRequest) {
	t.Helper()
	root := t.TempDir()
	for name, content := range map[string]string{
		"README.md": "# Original\n[missing](absent.txt)\n", "docs/z.md": "# Usage\nOld usage.\n",
		"docs/unselected.md": "Generated-looking text is not write authority.\n", "command.txt": "mode=current\n",
	} {
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	observed, err := CollectAssessment(root, "Review usage", "maintainers", []string{"README.md", "command.txt"})
	if err != nil {
		t.Fatal(err)
	}
	return root, observed, UpdateRequest{SchemaVersion: 1, Edits: []UpdateProposal{
		{Path: "docs/z.md", After: "# New usage\n[missing](new-missing.md)\n", Rationale: "Describe current usage", Citations: []string{"command.txt"}},
		{Path: "README.md", After: "# Revised 😀\u0085\n[usage](docs/z.md#usage)\n", Rationale: "Point to current instructions", Citations: []string{"command.txt", "README.md"}},
	}}
}

func updateJSONFile(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(name, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestUpdatePlanAndApplyExactFilesWithCompletedObservationOverlay(t *testing.T) {
	root, observed, request := updateFixture(t)
	before := snapshotPlanTarget(t, root)
	plan, err := PlanUpdates(root, updateJSONFile(t, observed), updateJSONFile(t, request))
	if err != nil || plan.Veto || len(plan.Edits) != 2 || len(plan.Findings) != 1 || plan.Edits[0].Path != "README.md" || !reflect.DeepEqual(before, snapshotPlanTarget(t, root)) {
		t.Fatalf("read-only sorted plan with non-vetoing link findings: %+v, %v", plan, err)
	}
	planPath := updateJSONFile(t, plan)
	result, err := ApplyUpdates(root, planPath, plan.PlanDigest)
	if err != nil || result.Status != "applied" || len(result.Replaced) != 2 || len(result.Untouched) != 0 || len(result.Unattempted) != 0 || result.Failed != nil || result.Scratch != "" {
		t.Fatalf("apply with evolving inventory/link findings: %+v, %v", result, err)
	}
	for _, edit := range plan.Edits {
		state, _, err := updateRead(root, edit.Path)
		if err != nil || state.Content != edit.After || state.Mode != edit.Before.Mode || state.UID != edit.Before.UID || state.GID != edit.Before.GID || state.Identity == edit.Before.Identity {
			t.Fatalf("exact reviewed file/metadata %s: %+v, %v", edit.Path, state, err)
		}
		before["file:"+edit.Path] = edit.After
	}
	if !reflect.DeepEqual(before, snapshotPlanTarget(t, root)) {
		t.Fatal("apply changed unselected files or left scratch")
	}
	validation, err := Validate(root)
	if err != nil || len(validation.Findings) != 2 || validation.Findings[0].Code != "validation.local-anchor-missing" || validation.Findings[0].Target != "docs/z.md#usage" || validation.Findings[1].Target != "new-missing.md" {
		t.Fatalf("heading rename and file findings must match the completed observation overlay: %+v, %v", validation, err)
	}
	result, err = ApplyUpdates(root, planPath, plan.PlanDigest)
	assertUpdateRefusal(t, result, err, "refusal.update-stale")
}

func TestUpdateRefusesChangedOriginalEvidenceAndMetadata(t *testing.T) {
	for _, change := range []string{"selected bytes", "mode", "identity", "source", "inventory", "link finding", "ancestor"} {
		t.Run(change, func(t *testing.T) {
			root, observed, request := updateFixture(t)
			plan, err := PlanUpdatesReviewed(root, observed, request)
			if err != nil {
				t.Fatal(err)
			}
			planPath := updateJSONFile(t, plan)
			switch change {
			case "mode":
				err = os.Chmod(filepath.Join(root, "README.md"), 0o600)
			case "identity":
				name := filepath.Join(root, "README.md")
				err = os.Rename(name, filepath.Join(root, "old-readme"))
				if err == nil {
					err = os.WriteFile(name, []byte(observed.Evidence[0].Content), 0o640)
				}
			case "ancestor":
				err = os.Rename(filepath.Join(root, "docs"), filepath.Join(root, "old-docs"))
			default:
				name := map[string]string{"selected bytes": "README.md", "source": "command.txt", "inventory": "docs/new.md", "link finding": "absent.txt"}[change]
				err = os.WriteFile(filepath.Join(root, name), []byte("changed\n"), 0o640)
			}
			if err != nil {
				t.Fatal(err)
			}
			before := snapshotPlanTarget(t, root)
			result, err := ApplyUpdates(root, planPath, plan.PlanDigest)
			assertUpdateRefusal(t, result, err, "refusal.update-stale")
			if !reflect.DeepEqual(before, snapshotPlanTarget(t, root)) {
				t.Fatal("stale refusal changed target or staged scratch")
			}
		})
	}
}

func TestUpdatePlanVetoesUncapturedUnsafeAndIdenticalSelections(t *testing.T) {
	for _, selection := range []string{"docs/new.md", "command.txt", "../README.md", "docs/../README.md", "README.md"} {
		t.Run(selection, func(t *testing.T) {
			root, observed, request := updateFixture(t)
			request.Edits = request.Edits[:1]
			request.Edits[0].Path = selection
			if selection == "README.md" {
				request.Edits[0].After = "# Original\n[missing](absent.txt)\n"
			}
			before := snapshotPlanTarget(t, root)
			plan, err := PlanUpdatesReviewed(root, observed, request)
			if err == nil || !plan.Veto || plan.PlanDigest == "" {
				t.Fatalf("expected reviewable veto: %+v, %v", plan, err)
			}
			result, err := ApplyUpdates(root, updateJSONFile(t, plan), plan.PlanDigest)
			assertUpdateRefusal(t, result, err, "refusal.update-veto")
			if !reflect.DeepEqual(before, snapshotPlanTarget(t, root)) {
				t.Fatal("veto changed target")
			}
		})
	}
}

func TestUpdateRequiresFreshConfirmationAndExactRawArtifact(t *testing.T) {
	root, observed, request := updateFixture(t)
	plan, err := PlanUpdatesReviewed(root, observed, request)
	if err != nil {
		t.Fatal(err)
	}
	before := snapshotPlanTarget(t, root)
	result, err := ApplyUpdates(root, updateJSONFile(t, plan), "decline")
	assertUpdateRefusal(t, result, err, "refusal.update-confirmation")
	data, _ := json.Marshal(plan)
	for name, raw := range map[string][]byte{
		"unknown":            bytes.Replace(data, []byte(`"veto":false`), []byte(`"extra":1,"veto":false`), 1),
		"alias":              bytes.Replace(data, []byte(`"targetRoot":`), []byte(`"TargetRoot":`), 1),
		"duplicate":          bytes.Replace(data, []byte(`"veto":false`), []byte(`"veto":false,"veto":false`), 1),
		"null":               bytes.Replace(data, []byte(`"ancestors":[]`), []byte(`"ancestors":null`), 1),
		"tampered rationale": bytes.Replace(data, []byte("Point to current instructions"), []byte("Changed review rationale"), 1),
		"invalid surrogate":  bytes.Replace(data, []byte("Point to current instructions"), []byte(`\ud800`), 1),
		"trailing JSON":      append(append([]byte{}, data...), []byte(` {}`)...),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "reviewed.json")
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			result, err := ApplyUpdates(root, path, plan.PlanDigest)
			assertUpdateRefusal(t, result, err, "refusal.update-plan-invalid")
			result, err = ApplyUpdatesReviewed(root, raw, plan.PlanDigest)
			assertUpdateRefusal(t, result, err, "refusal.update-plan-invalid")
		})
	}
	if !reflect.DeepEqual(before, snapshotPlanTarget(t, root)) {
		t.Fatal("invalid or declined plan wrote target")
	}
}

func assertUpdateRefusal(t *testing.T, result UpdateResult, err error, code string) {
	t.Helper()
	var structured *ldserr.Error
	if !errors.As(err, &structured) || structured.Code != code || result.Status != "refused" || len(result.Replaced) != 0 || result.Scratch != "" {
		t.Fatalf("expected %s with no replacement/scratch: %+v, %v", code, result, err)
	}
}

func TestUpdateStopsAfterPartialOrUncertainReplacement(t *testing.T) {
	for _, failure := range []string{"source after first", "after rename"} {
		t.Run(failure, func(t *testing.T) {
			root, observed, request := updateFixture(t)
			plan, err := PlanUpdatesReviewed(root, observed, request)
			if err != nil {
				t.Fatal(err)
			}
			realReplace := replaceUpdateFile
			t.Cleanup(func() { replaceUpdateFile = realReplace })
			replaceUpdateFile = func(root, path string, before UpdateFileState, after string, parents []ParentIdentity, check func() error) (UpdateReplacement, error) {
				if path == "docs/z.md" && failure == "source after first" {
					if err := os.WriteFile(filepath.Join(root, "command.txt"), []byte("late source change\n"), 0o640); err != nil {
						t.Fatal(err)
					}
				}
				actual, err := realReplace(root, path, before, after, parents, check)
				if path == "README.md" && failure == "after rename" && err == nil {
					actual.Observed, actual.Phase = nil, "observe"
					return actual, errors.New("simulated unreadable post-rename state")
				}
				return actual, err
			}
			result, err := ApplyUpdates(root, updateJSONFile(t, plan), plan.PlanDigest)
			if err == nil || result.Failed == nil || result.Scratch != "" || result.CleanupError != "" {
				t.Fatalf("expected reported failure: %+v, %v", result, err)
			}
			readme, _ := os.ReadFile(filepath.Join(root, "README.md"))
			pending, _ := os.ReadFile(filepath.Join(root, "docs/z.md"))
			if string(readme) != plan.Edits[0].After || string(pending) != plan.Edits[1].Before.Content {
				t.Fatal("partial failure rolled back the first edit or applied the pending edit")
			}
			if failure == "source after first" && (result.Status != "partial" || len(result.Replaced) != 1 || len(result.Untouched) != 1) {
				t.Fatalf("partial effects: %+v", result)
			}
			if failure == "after rename" && (result.Status != "indeterminate" || len(result.Replaced) != 0 || len(result.Unattempted) != 1 || len(result.Untouched) != 1 || result.Untouched[0].Path != "docs/z.md") {
				t.Fatalf("uncertain renamed file was claimed preserved: %+v", result)
			}
			for name := range snapshotPlanTarget(t, root) {
				if strings.Contains(name, ".lds-update-") {
					t.Fatal("owned scratch was not cleaned")
				}
			}
		})
	}
}

func TestUpdateDoesNotApplyOrCleanSubstitutedScratch(t *testing.T) {
	root, observed, request := updateFixture(t)
	plan, err := PlanUpdatesReviewed(root, observed, request)
	if err != nil {
		t.Fatal(err)
	}
	realReplace := replaceUpdateFile
	t.Cleanup(func() { replaceUpdateFile = realReplace })
	var foreign string
	replaceUpdateFile = func(root, path string, before UpdateFileState, after string, parents []ParentIdentity, check func() error) (UpdateReplacement, error) {
		return realReplace(root, path, before, after, parents, func() error {
			entries, err := os.ReadDir(root)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".lds-update-") {
					foreign = filepath.Join(root, entry.Name())
					if err := os.Rename(foreign, filepath.Join(t.TempDir(), "held-stage")); err != nil {
						return err
					}
					if err := os.WriteFile(foreign, []byte("foreign bytes"), 0o600); err != nil {
						return err
					}
				}
			}
			return check()
		})
	}
	result, err := ApplyUpdates(root, updateJSONFile(t, plan), plan.PlanDigest)
	if err == nil || result.Status != "partial" || len(result.Replaced) != 0 || len(result.Untouched) != 2 || result.Scratch == "" || result.CleanupError == "" {
		t.Fatalf("foreign scratch must remain, with no selected replacement: %+v, %v", result, err)
	}
	data, err := os.ReadFile(foreign)
	if err != nil || string(data) != "foreign bytes" || filepath.Join(root, result.Scratch) != foreign {
		t.Fatalf("foreign scratch was removed or unreported: %q, %v", data, err)
	}
}
