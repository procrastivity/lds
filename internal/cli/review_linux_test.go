//go:build linux

package cli_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/procrastivity/lds/internal/adoption"
)

func TestReviewReportsFindingsSkipsAndRepairWithoutWrites(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	files := map[string]string{
		"README.md":     "# Demo\n[guide](docs/guide.md#not-checked)\n[missing](docs/missing.md)\n[web](https://example.invalid)\n[query](docs/guide.md?raw=1)\n[alias](docs/alias.md)\n[escape](../outside.md)\n",
		"docs/guide.md": "# Guide\n", "engineering/decision.md": "# Decision\n",
		"source.go": "PRIVATE UNSELECTED CONTENT\n",
	}
	for name, content := range files {
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	secret := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(secret, []byte("PRIVATE OUTSIDE CONTENT\n[missing](outside-missing.md)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "docs/alias.md")); err != nil {
		t.Fatal(err)
	}
	env := []string{"OPENCODE_API_KEY=", "ANTHROPIC_API_KEY=", "OPENAI_API_KEY="}
	for _, repaired := range []bool{false, true} {
		if repaired {
			if err := os.WriteFile(filepath.Join(root, "docs/missing.md"), []byte("# Repaired\n"), 0o640); err != nil {
				t.Fatal(err)
			}
			files["docs/missing.md"] = "# Repaired\n"
		}
		before, outsideBefore := assessmentTree(t, root), assessmentTree(t, outside)
		got := run(t, env, "review", root, "--json")
		var report struct {
			SchemaVersion                 int
			Operation, TargetRoot, Status string
			Inventory                     adoption.Inventory
			Validation                    adoption.Validation
			Limitations                   []string
		}
		if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || report.SchemaVersion != 1 || report.Operation != "review" || report.TargetRoot != root {
			t.Fatalf("invalid review report: %+v (%v)", got, err)
		}
		wantExit, wantStatus, wantFiles := 1, "findings", 3
		wantFindings := []adoption.Finding{{Code: "validation.local-target-missing", Path: "README.md", Line: 3, Target: "docs/missing.md", Message: "local file target is missing: docs/missing.md"}}
		if repaired {
			wantExit, wantStatus, wantFiles, wantFindings = 0, "no-findings", 4, []adoption.Finding{}
		}
		if got.exitCode != wantExit || report.Status != wantStatus || len(report.Inventory.Files) != wantFiles || len(report.Inventory.Links) != 5 || !reflect.DeepEqual(report.Validation.Findings, wantFindings) {
			t.Fatalf("incorrect evidence/status: %+v, %+v", got, report)
		}
		if !repaired && parseErrorEnvelope(t, got.stderr).Error.Code != "validation.local-target-missing" || repaired && got.stderr != "" {
			t.Fatalf("lost diagnostic exit/error: %+v", got)
		}
		for _, file := range report.Inventory.Files {
			content, exists := files[file.Path]
			if !exists || file.Path == "source.go" || file.Size != len(content) || file.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(content))) {
				t.Fatalf("incorrect or out-of-scope inventory: %+v", file)
			}
		}
		codes := map[string]int{}
		for _, skip := range report.Validation.Skipped {
			codes[skip.Code]++
		}
		if !reflect.DeepEqual(codes, map[string]int{"inventory.external-skipped": 1, "inventory.symlink-skipped": 1, "validation.fragment-skipped": 1, "validation.query-skipped": 1, "validation.unsafe-link-skipped": 2}) {
			t.Fatalf("skips were hidden, duplicated or promoted: %+v", report.Validation.Skipped)
		}
		if len(report.Limitations) == 0 || !strings.Contains(strings.Join(report.Limitations, " "), "not assessed") || !strings.Contains(strings.Join(report.Limitations, " "), "not an atomic snapshot") {
			t.Fatalf("missing semantic/concurrency limitations: %v", report.Limitations)
		}
		human := run(t, env, "review", root)
		for _, text := range []string{"README.md", "sha256:", "Local-link findings", "Skipped checks", "validation.fragment-skipped", "not assessed"} {
			if !strings.Contains(human.stdout, text) {
				t.Errorf("human report omitted %q: %s", text, human.stdout)
			}
		}
		if human.exitCode != wantExit || strings.Contains(human.stdout+got.stdout, "PRIVATE") {
			t.Fatalf("wrong human exit or content disclosure: %+v", human)
		}
		if again := run(t, env, "review", root, "--json"); again.stdout != got.stdout || again.exitCode != got.exitCode {
			t.Fatal("unchanged target produced nondeterministic evidence")
		}
		if assessmentTree(t, root) != before || assessmentTree(t, outside) != outsideBefore {
			t.Fatal("review changed target or symlink destination")
		}
	}
}

func TestReviewEmptyScopeAndUnsafeRoots(t *testing.T) {
	root := t.TempDir()
	got := run(t, nil, "review", root, "--json")
	if got.exitCode != 0 || got.stderr != "" || !strings.Contains(got.stdout, `"status":"no-findings"`) || !strings.Contains(got.stdout, `"files":[]`) || !strings.Contains(got.stdout, `"findings":[]`) || !strings.Contains(got.stdout, "Documentation completeness") {
		t.Fatalf("empty scope falsely implied completeness or failed: %+v", got)
	}
	human := run(t, nil, "review", root)
	if human.exitCode != 0 || !strings.Contains(human.stdout, "None observed") || !strings.Contains(human.stdout, "not assessed") {
		t.Fatalf("empty human report hid lack of coverage: %+v", human)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, code string
		exit       int
	}{
		{filepath.Join(root, "absent"), "validation.target-root-missing", 1},
		{alias, "refusal.unsafe-target", 3},
	} {
		got := run(t, nil, "review", tc.path, "--json")
		if got.exitCode != tc.exit || got.stdout != "" || parseErrorEnvelope(t, got.stderr).Error.Code != tc.code {
			t.Fatalf("invalid root emitted a report or wrong error: %+v", got)
		}
	}
	if usage := run(t, nil, "review", "--json"); usage.exitCode != 2 || usage.stdout != "" {
		t.Fatalf("review must require an explicit root: %+v", usage)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("empty or refused review wrote files: %v, %v", entries, err)
	}
}
