//go:build linux

package cli_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/procrastivity/lds/internal/adoption"
)

func TestReviewReportsFindingsSkipsAndRepairWithoutWrites(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	files := map[string]string{
		"README.md":     "# Demo\n[guide](docs/guide.md#install)\n[missing](docs/missing.md)\n[web](https://example.invalid)\n[query](docs/guide.md?raw=1)\n[alias](docs/alias.md)\n[escape](../outside.md)\n",
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
			files["docs/guide.md"] = "# Guide\n## Install\n"
			if err := os.WriteFile(filepath.Join(root, "docs/guide.md"), []byte(files["docs/guide.md"]), 0o640); err != nil {
				t.Fatal(err)
			}
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
		wantFindings := []adoption.Finding{
			{Code: "validation.local-anchor-missing", Path: "README.md", Line: 2, Target: "docs/guide.md#install", Message: "local heading anchor is missing: docs/guide.md#install"},
			{Code: "validation.local-target-missing", Path: "README.md", Line: 3, Target: "docs/missing.md", Message: "local file target is missing: docs/missing.md"},
		}
		if repaired {
			wantExit, wantStatus, wantFiles, wantFindings = 0, "no-findings", 4, []adoption.Finding{}
		}
		if got.exitCode != wantExit || report.Status != wantStatus || len(report.Inventory.Files) != wantFiles || len(report.Inventory.Links) != 5 || !reflect.DeepEqual(report.Validation.Findings, wantFindings) {
			t.Fatalf("incorrect evidence/status: %+v, %+v", got, report)
		}
		if !repaired && parseErrorEnvelope(t, got.stderr).Error.Code != "validation.local-anchor-missing" || repaired && got.stderr != "" {
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
		if !reflect.DeepEqual(codes, map[string]int{"inventory.external-skipped": 1, "inventory.symlink-skipped": 1, "validation.query-skipped": 1, "validation.unsafe-link-skipped": 2}) {
			t.Fatalf("skips were hidden, duplicated or promoted: %+v", report.Validation.Skipped)
		}
		if len(report.Limitations) == 0 || !strings.Contains(strings.Join(report.Limitations, " "), "not assessed") || !strings.Contains(strings.Join(report.Limitations, " "), "not an atomic snapshot") {
			t.Fatalf("missing semantic/concurrency limitations: %v", report.Limitations)
		}
		human := run(t, env, "review", root)
		for _, text := range []string{"README.md", "sha256:", "Local-link findings", "Skipped checks", "validation.query-skipped", "not assessed"} {
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

func TestReviewCIWrapperSummariesAndExitStatuses(t *testing.T) {
	wrapper, err := filepath.Abs("../../contrib/review-documentation-ci")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		exit, wantExit int
		invalid        string
	}{
		{name: "clean"},
		{name: "findings", exit: 1, wantExit: 1},
		{name: "usage-empty", exit: 2, wantExit: 2, invalid: "empty"},
		{name: "refused-malformed", exit: 3, wantExit: 3, invalid: "malformed"},
		{name: "io-empty", exit: 4, wantExit: 4, invalid: "empty"},
		{name: "false-success", wantExit: 4, invalid: "malformed"},
		{name: "summary-disabled"},
		{name: "summary-inside-root"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, scratch, home := t.TempDir(), t.TempDir(), t.TempDir()
			bin := filepath.Join(root, "result/bin/lds")
			if err := os.MkdirAll(filepath.Dir(bin), 0o700); err != nil {
				t.Fatal(err)
			}
			// Simulate the reviewed command's public stdout/stderr/exit boundary,
			// including failures that cannot reliably be induced via real I/O.
			if err := os.WriteFile(bin, []byte("#!/usr/bin/env bash\ncat \"$CI_FIXTURE\"\nprintf '%s\\n' 'diagnostic <raw>' >&2\nexit \"$CI_EXIT\"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			findings, status := []adoption.Finding{}, "no-findings"
			if tc.exit == 1 {
				status = "findings"
				findings = append(findings, adoption.Finding{Path: "docs/<img>.md", Line: 7, Code: "validation.local-target-missing", Message: "</pre><script>bad</script>\n::error::injected"})
			}
			payload, err := json.Marshal(map[string]any{
				"schemaVersion": 1, "operation": "review", "targetRoot": root, "status": status,
				"inventory": adoption.Inventory{Files: []adoption.File{{Path: "README.md"}, {Path: "docs/guide.md"}}},
				"validation": adoption.Validation{Findings: findings, Skipped: []adoption.Skip{
					{Code: "inventory.external-skipped", Reason: "external URL not fetched"},
					{Code: "inventory.external-skipped", Reason: "external URL not fetched"},
					{Code: "validation.fragment-skipped", Reason: "unsupported <heading>"},
					{Code: "validation.fragment-skipped", Reason: "source-line reference"},
				}},
				"limitations": []string{"Prose accuracy is not assessed."},
			})
			if err != nil {
				t.Fatal(err)
			}
			switch tc.invalid {
			case "empty":
				payload = nil
			case "malformed":
				payload = []byte("<unusable>")
			}
			fixture, summary := filepath.Join(scratch, "fixture.json"), filepath.Join(scratch, "summary.md")
			if err := os.WriteFile(fixture, payload, 0o600); err != nil {
				t.Fatal(err)
			}
			if tc.name == "summary-inside-root" {
				summary = filepath.Join(root, "README.md")
			} else if tc.name != "summary-disabled" {
				if err := os.WriteFile(summary, []byte("Existing job notes.\n\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("bash", wrapper)
			cmd.Dir = root
			cmd.Env = []string{
				"PATH=" + os.Getenv("PATH"), "HOME=" + home, "GITHUB_WORKSPACE=" + root,
				"RUNNER_TEMP=" + scratch, "CI_FIXTURE=" + fixture, "CI_EXIT=" + strconv.Itoa(tc.exit),
			}
			if tc.name != "summary-disabled" {
				cmd.Env = append(cmd.Env, "GITHUB_STEP_SUMMARY="+summary)
			}
			before := assessmentTree(t, root)
			output, err := cmd.CombinedOutput()
			code := 0
			if err != nil {
				exit, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatal(err)
				}
				code = exit.ExitCode()
			}
			if code != tc.wantExit || !strings.Contains(string(output), fmt.Sprintf("LDS exit status: %d;", tc.exit)) {
				t.Fatalf("lost original exit status: %d (%v), %s", code, err, output)
			}
			captures, err := filepath.Glob(filepath.Join(scratch, "lds-review.*/review.json"))
			if err != nil || len(captures) != 1 {
				t.Fatalf("missing external capture: %v (%v)", captures, err)
			}
			captured, err := os.ReadFile(captures[0])
			if err != nil || string(captured) != string(payload) || assessmentTree(t, root) != before {
				t.Fatal("wrapper changed raw JSON or wrote inside the inspected root")
			}
			if tc.name == "summary-disabled" || tc.name == "summary-inside-root" {
				if _, err := os.Stat(summary); !os.IsNotExist(err) {
					t.Fatalf("unexpected summary write: %v", err)
				}
				return
			}
			data, err := os.ReadFile(summary)
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			if !strings.HasPrefix(text, "Existing job notes.\n\n") || !strings.Contains(text, fmt.Sprintf("LDS exit status: **%d**", tc.exit)) {
				t.Fatalf("summary lost existing notes or actual LDS status: %s", text)
			}
			if tc.invalid != "" {
				if !strings.Contains(text, "No usable review result") || strings.Contains(text, "No findings means") {
					t.Fatalf("unusable output presented as clean: %s", text)
				}
				return
			}
			for _, want := range []string{"2 Markdown files", "4 skipped checks", "2 × [&quot;inventory.external-skipped&quot;]", "1 × [&quot;validation.fragment-skipped&quot;] &quot;source-line reference&quot;", "external URL not fetched", "unsupported &lt;heading&gt;", "Prose accuracy is not assessed."} {
				if !strings.Contains(text, want) {
					t.Errorf("summary omitted %q: %s", want, text)
				}
			}
			if tc.exit == 1 && (!strings.Contains(text, "&quot;docs/&lt;img&gt;.md&quot;:7") || !strings.Contains(text, "&lt;/pre&gt;&lt;script&gt;bad&lt;/script&gt;\\n::error::injected")) {
				t.Fatalf("finding lost its location or safe display: %s", text)
			}
			if strings.Contains(text, "<script>") || strings.Contains(text, "\n::error::") {
				t.Fatalf("untrusted finding became active markup or a workflow command: %s", text)
			}
		})
	}
}
