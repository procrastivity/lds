package adoption

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const applyFactsPath = "../asset/testdata/adoption/greenfield/facts.json"

func TestApply_CreatesOnlyReviewedOutputsAndIdenticalRerunIsCurrent(t *testing.T) {
	useIsolatedAssetDirectories(t)
	root := t.TempDir()
	plan, planPath := writeApplyPlan(t, root, applyFactsPath)
	result, err := Apply(root, applyFactsPath, planPath, *plan.PlanDigest)
	if err != nil || result.Status != "applied" || len(result.Created) != 4 || len(result.Unattempted) != 0 {
		t.Fatalf("first Apply = %+v, %v", result, err)
	}
	for _, created := range result.Created {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(created.Path)))
		if err != nil || sha256Hex(data) != created.SHA256 {
			t.Fatalf("created %s bytes/hash=%q/%s err=%v", created.Path, data, created.SHA256, err)
		}
	}
	second, secondPlan := writeApplyPlan(t, root, applyFactsPath)
	current, err := Apply(root, applyFactsPath, secondPlan, *second.PlanDigest)
	if err != nil || current.Status != "current" || len(current.Created) != 0 || len(current.IdenticalUntouched) != 4 {
		t.Fatalf("identical Apply = %+v, %v", current, err)
	}
}

func TestApplyReviewedUsesRawDisplayedFieldsAndDigest(t *testing.T) {
	useIsolatedAssetDirectories(t)
	acceptedRoot := t.TempDir()
	plan, err := Plan(acceptedRoot, applyFactsPath)
	if err != nil || plan.PlanDigest == nil {
		t.Fatalf("Plan: digest=%v err=%v", plan.PlanDigest, err)
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := ApplyReviewed(acceptedRoot, applyFactsPath, raw, *plan.PlanDigest)
	if err != nil || accepted.Status != "applied" || len(accepted.Created) != 4 {
		t.Fatalf("exact in-memory reviewed bytes = %+v, %v", accepted, err)
	}

	changedRoot := t.TempDir()
	changedPlan, err := Plan(changedRoot, applyFactsPath)
	if err != nil || changedPlan.PlanDigest == nil {
		t.Fatalf("second Plan: digest=%v err=%v", changedPlan.PlanDigest, err)
	}
	changedBytes, err := json.Marshal(changedPlan)
	if err != nil {
		t.Fatal(err)
	}
	changedBytes = bytes.Replace(changedBytes, []byte(`"veto":false`), []byte(`"veto":true`), 1)
	if !bytes.Contains(changedBytes, []byte(`"veto":true`)) {
		t.Fatal("asymmetric reviewed-field mutation did not alter the raw plan")
	}
	refused, err := ApplyReviewed(changedRoot, applyFactsPath, changedBytes, *changedPlan.PlanDigest)
	assertApplyRefused(t, refused, err, "refusal.plan-invalid")
	assertNoApprovedOutputs(t, changedRoot)
}

func TestApply_DeclineTamperAndStaleInputsWriteNothing(t *testing.T) {
	t.Run("declined", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		root := t.TempDir()
		plan, p := writeApplyPlan(t, root, applyFactsPath)
		result, err := Apply(root, applyFactsPath, p, "sha256:not-confirmed")
		assertApplyRefused(t, result, err, "refusal.confirmation-declined")
		assertNoApprovedOutputs(t, root)
		_ = plan
	})
	t.Run("tampered plan", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		root := t.TempDir()
		plan, p := writeApplyPlan(t, root, applyFactsPath)
		plan.Outputs[0].Content += "tampered"
		data, _ := json.Marshal(plan)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		result, err := Apply(root, applyFactsPath, p, *plan.PlanDigest)
		assertApplyRefused(t, result, err, "refusal.plan-invalid")
		assertNoApprovedOutputs(t, root)
	})
	t.Run("case-folded displayed field", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		root := t.TempDir()
		plan, p := writeApplyPlan(t, root, applyFactsPath)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte(`"targetRoot":`), []byte(`"TargetRoot":`), 1)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		result, err := Apply(root, applyFactsPath, p, *plan.PlanDigest)
		assertApplyRefused(t, result, err, "refusal.plan-invalid")
		assertNoApprovedOutputs(t, root)
	})
	t.Run("changed facts", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		root := t.TempDir()
		facts := filepath.Join(t.TempDir(), "facts.json")
		data, err := os.ReadFile(applyFactsPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(facts, data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, p := writeApplyPlan(t, root, facts)
		if err := os.WriteFile(facts, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		result, err := Apply(root, facts, p, planDigestFromFile(t, p))
		assertApplyRefused(t, result, err, "refusal.plan-stale")
		assertNoApprovedOutputs(t, root)
	})
	t.Run("changed diagnostics", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		root := t.TempDir()
		_, p := writeApplyPlan(t, root, applyFactsPath)
		if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "docs", "links.md"), []byte("[missing](absent.md)\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		result, err := Apply(root, applyFactsPath, p, planDigestFromFile(t, p))
		assertApplyRefused(t, result, err, "refusal.plan-stale")
		assertNoApprovedOutputs(t, root)
	})
	t.Run("stale preserved bytes are not reported as observed", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		root := t.TempDir()
		sample := filepath.Join(root, "sample.txt")
		if err := os.WriteFile(sample, []byte("before\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		plan, p := writeApplyPlan(t, root, applyFactsPath)
		if err := os.WriteFile(sample, []byte("after\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		result, err := Apply(root, applyFactsPath, p, *plan.PlanDigest)
		assertApplyRefused(t, result, err, "refusal.plan-stale")
		if len(result.Preserved) != 1 || result.Preserved[0].SHA256 != sha256Hex([]byte("after\n")) {
			t.Fatalf("refusal reported stale preservation evidence: %+v", result.Preserved)
		}
		assertNoApprovedOutputs(t, root)
	})
	t.Run("replaced target", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		parent := t.TempDir()
		root := filepath.Join(parent, "target")
		if err := os.Mkdir(root, 0o755); err != nil {
			t.Fatal(err)
		}
		_, p := writeApplyPlan(t, root, applyFactsPath)
		retired := root + ".old"
		if err := os.Rename(root, retired); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(root, 0o755); err != nil {
			t.Fatal(err)
		}
		result, err := Apply(root, applyFactsPath, p, planDigestFromFile(t, p))
		assertApplyRefused(t, result, err, "refusal.plan-stale")
		assertNoApprovedOutputs(t, root)
	})
}

func TestApply_ChangedAssetAndUnsafeSymlinkPathsRefuse(t *testing.T) {
	t.Run("changed effective asset", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		root := t.TempDir()
		_, p := writeApplyPlan(t, root, applyFactsPath)
		assetOverride := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "lds", "templates", "adoption", "docs", "README.md")
		if err := os.MkdirAll(filepath.Dir(assetOverride), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(assetOverride, []byte("# changed template\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		result, err := Apply(root, applyFactsPath, p, planDigestFromFile(t, p))
		assertApplyRefused(t, result, err, "refusal.plan-stale")
		assertNoApprovedOutputs(t, root)
	})
	t.Run("output symlink", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		root := t.TempDir()
		outside := filepath.Join(t.TempDir(), "outside.md")
		if err := os.WriteFile(outside, []byte("preserve"), 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(root, "docs", "reference", "usage.md")
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, link); err != nil {
			t.Fatal(err)
		}
		plan, err := Plan(root, applyFactsPath)
		var refusal *PlanRefusal
		if !errors.As(err, &refusal) || !plan.Veto {
			t.Fatalf("symlink plan=%+v err=%v", plan, err)
		}
		data, err := os.ReadFile(outside)
		if err != nil || string(data) != "preserve" {
			t.Fatalf("symlink target changed: %q %v", data, err)
		}
	})
	t.Run("ancestor symlink", func(t *testing.T) {
		useIsolatedAssetDirectories(t)
		root := t.TempDir()
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(root, "docs")); err != nil {
			t.Fatal(err)
		}
		plan, err := Plan(root, applyFactsPath)
		var refusal *PlanRefusal
		if !errors.As(err, &refusal) || !plan.Veto {
			t.Fatalf("ancestor-symlink plan=%+v err=%v", plan, err)
		}
		if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
			t.Fatalf("ancestor target was modified: entries=%v err=%v", entries, err)
		}
	})
}

func TestApply_ReportsPartialEffectsWithoutRollback(t *testing.T) {
	useIsolatedAssetDirectories(t)
	root := t.TempDir()
	plan, p := writeApplyPlan(t, root, applyFactsPath)
	original := createOutputFile
	calls := 0
	createOutputFile = func(target *createTarget, rel string, data []byte) error {
		calls++
		if calls == 1 {
			return original(target, rel, data)
		}
		return errors.New("injected")
	}
	t.Cleanup(func() { createOutputFile = original })
	result, err := Apply(root, applyFactsPath, p, *plan.PlanDigest)
	var applyErr *ApplyError
	if !errors.As(err, &applyErr) || applyErr.Code != "internal.apply-failed" || result.Status != "partial" || len(result.Created) != 1 || result.Failed == nil || len(result.Unattempted) != 2 {
		t.Fatalf("partial Apply = %+v, %v", result, err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.Created[0].Path)))
	if err != nil || sha256Hex(data) != result.Created[0].SHA256 {
		t.Fatalf("first effect missing/hash mismatch: %v", err)
	}
}

func TestApply_AncestorReplacementAfterReviewDoesNotFollowSymlink(t *testing.T) {
	useIsolatedAssetDirectories(t)
	root := t.TempDir()
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel.md")
	if err := os.WriteFile(sentinel, []byte("outside bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, p := writeApplyPlan(t, root, applyFactsPath)
	original := createOutputFile
	createOutputFile = func(target *createTarget, rel string, data []byte) error {
		if rel == "docs/README.md" {
			if err := os.Symlink(outside, filepath.Join(root, "docs")); err != nil {
				return err
			}
		}
		return original(target, rel, data)
	}
	t.Cleanup(func() { createOutputFile = original })
	result, err := Apply(root, applyFactsPath, p, *plan.PlanDigest)
	var applyErr *ApplyError
	if !errors.As(err, &applyErr) || result.Status != "refused" || result.Failed == nil || len(result.Created) != 0 || len(result.Unattempted) != 3 {
		t.Fatalf("ancestor swap result=%+v err=%v", result, err)
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "outside bytes\n" {
		t.Fatalf("symlink target changed: %q err=%v", data, err)
	}
}

func TestApply_RefusesRegularParentReplacementAtWriteBoundary(t *testing.T) {
	useIsolatedAssetDirectories(t)
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, p := writeApplyPlan(t, root, applyFactsPath)
	original := createOutputFile
	createOutputFile = func(target *createTarget, rel string, data []byte) error {
		if err := os.Rename(filepath.Join(root, "docs"), filepath.Join(root, "docs.old")); err != nil {
			return err
		}
		if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
			return err
		}
		return original(target, rel, data)
	}
	t.Cleanup(func() { createOutputFile = original })
	result, err := Apply(root, applyFactsPath, p, *plan.PlanDigest)
	assertApplyRefused(t, result, err, "refusal.plan-stale")
	assertNoApprovedOutputs(t, root)
	if entries, err := os.ReadDir(filepath.Join(root, "docs.old")); err != nil || len(entries) != 0 {
		t.Fatalf("replaced ancestor was written: %v %v", entries, err)
	}
}

func TestApply_RefusesNamedRootReplacementAtWriteBoundary(t *testing.T) {
	useIsolatedAssetDirectories(t)
	parent := t.TempDir()
	root := filepath.Join(parent, "target")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	plan, p := writeApplyPlan(t, root, applyFactsPath)
	original := createOutputFile
	createOutputFile = func(target *createTarget, rel string, data []byte) error {
		if err := os.Rename(root, root+".old"); err != nil {
			return err
		}
		if err := os.Mkdir(root, 0o755); err != nil {
			return err
		}
		return original(target, rel, data)
	}
	t.Cleanup(func() { createOutputFile = original })
	result, err := Apply(root, applyFactsPath, p, *plan.PlanDigest)
	assertApplyRefused(t, result, err, "refusal.plan-stale")
	assertNoApprovedOutputs(t, root)
	assertNoApprovedOutputs(t, root+".old")
}

func TestApply_LateOccupiedOutputIsPolicyCollision(t *testing.T) {
	useIsolatedAssetDirectories(t)
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, p := writeApplyPlan(t, root, applyFactsPath)
	original := createOutputFile
	createOutputFile = func(target *createTarget, rel string, data []byte) error {
		if rel == "docs/README.md" {
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte("occupied\n"), 0o644); err != nil {
				return err
			}
		}
		return original(target, rel, data)
	}
	t.Cleanup(func() { createOutputFile = original })
	result, err := Apply(root, applyFactsPath, p, *plan.PlanDigest)
	assertApplyRefused(t, result, err, "refusal.output-collision")
	if result.Failed == nil || result.Failed.Path != "docs/README.md" || result.Failed.SHA256 != sha256Hex([]byte("occupied\n")) || len(result.Unattempted) != 3 {
		t.Fatalf("late occupied path result = %+v", result)
	}
	occupied, err := os.ReadFile(filepath.Join(root, "docs", "README.md"))
	if err != nil || string(occupied) != "occupied\n" {
		t.Fatalf("occupied bytes overwritten: %q %v", occupied, err)
	}
}

func writeApplyPlan(t *testing.T, root, facts string) (PlanResult, string) {
	t.Helper()
	plan, err := Plan(root, facts)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return plan, file
}

func planDigestFromFile(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var plan PlanResult
	if err := json.Unmarshal(data, &plan); err != nil || plan.PlanDigest == nil {
		t.Fatalf("decode plan: %v", err)
	}
	return *plan.PlanDigest
}

func assertApplyRefused(t *testing.T, result ApplyResult, err error, code string) {
	t.Helper()
	var applyErr *ApplyError
	if !errors.As(err, &applyErr) || applyErr.Code != code || result.Status != "refused" || len(result.Created) != 0 {
		t.Fatalf("Apply refusal = %+v, %v; want %s", result, err, code)
	}
}

func assertNoApprovedOutputs(t *testing.T, root string) {
	t.Helper()
	for _, name := range []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(name))); err == nil {
			t.Errorf("refused apply created %s", name)
		}
	}
}
