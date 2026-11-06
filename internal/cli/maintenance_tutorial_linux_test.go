//go:build linux

package cli_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/procrastivity/lds/internal/adoption"
)

func TestOfflineMaintenanceTutorial(t *testing.T) {
	repository, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	document, err := os.ReadFile(filepath.Join(repository, "docs/tutorials/maintain-documentation.md"))
	if err != nil {
		t.Fatal(err)
	}
	blocks := regexp.MustCompile("(?ms)^```sh\n(.*?)^```\n").FindAllStringSubmatch(string(document), -1)
	names := []string{"prepare", "init", "initial-review", "broken-review", "collect", "assess", "request", "update", "final-review", "cleanup"}
	if len(blocks) != len(names) {
		t.Fatalf("tutorial has %d shell blocks; review the connected assertions for its %d steps", len(blocks), len(names))
	}
	home, scratch := t.TempDir(), t.TempDir()
	state := filepath.Join(scratch, "paths")
	// Minimal child environment: no provider credentials, real home, XDG
	// overrides or harness settings. TestMain's binary has embedded assets.
	env := []string{
		"PATH=" + filepath.Dir(binPath) + ":" + os.Getenv("PATH"), "HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "TMPDIR=" + scratch,
		"LC_ALL=C.UTF-8", "TUTORIAL_STATE=" + state,
	}
	var project, review string
	const after = "# User documentation\n\n[Usage](reference/usage.md)\n\n" +
		"The usage page documents the fixture's word-count command.\n\n" +
		"[External reference](https://example.com).\n"
	readJSON := func(t *testing.T, name string, target any) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(review, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, target); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	checkDiagnostics := func(t *testing.T, got adoption.Validation, broken bool, skips int) {
		t.Helper()
		wantFindings := 0
		if broken {
			wantFindings = 1
		}
		if len(got.Findings) != wantFindings || len(got.Skipped) != skips {
			t.Fatalf("unexpected diagnostics: %+v", got)
		}
		if broken {
			finding := got.Findings[0]
			if finding.Code != "validation.local-target-missing" || finding.Path != "docs/README.md" || finding.Line != 5 || finding.Target != "missing-notes.md" {
				t.Fatalf("lost deliberate missing-file finding: %+v", finding)
			}
		}
		if skips == 1 {
			skip := got.Skipped[0]
			if skip.Code != "inventory.external-skipped" || skip.Source != "docs/README.md" || skip.Line != 7 {
				t.Fatalf("external URL was not honestly skipped: %+v", skip)
			}
		}
	}
	for i, name := range names {
		if !t.Run(name, func(t *testing.T) {
			block := blocks[i][1]
			if i == 0 {
				// Preserve shell variables between separate exact-block executions.
				block += "\nprintf '%s\\n' \"$project\" \"$review\" > \"$TUTORIAL_STATE\"\n"
			}
			before, operation := "", ""
			if project != "" {
				before = assessmentTree(t, project)
			}
			if name == "init" || name == "update" {
				operation = name
			}
			output, displayed := runTutorialBlock(t, repository, env, block, project, operation)
			if project != "" && operation == "" && name != "broken-review" && name != "cleanup" && assessmentTree(t, project) != before {
				t.Fatal("read-only tutorial step changed the project")
			}
			switch name {
			case "prepare":
				if output != "Word count: 4\n" {
					t.Fatalf("fixture operation output: %q", output)
				}
				data, err := os.ReadFile(state)
				if err != nil {
					t.Fatal(err)
				}
				paths := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
				if len(paths) != 2 || paths[0] == paths[1] {
					t.Fatalf("expected separate disposable project and review paths: %q", data)
				}
				for _, path := range paths {
					if filepath.Dir(path) != scratch {
						t.Fatalf("tutorial escaped its disposable parent: %s", path)
					}
				}
				project, review = paths[0], paths[1]
				env = append(env, "project="+project, "review="+review)
			case "init", "update":
				var outcome struct {
					SchemaVersion     int
					Operation, Status string
					Plan, Apply       json.RawMessage
				}
				if err := json.Unmarshal([]byte(output), &outcome); err != nil || outcome.SchemaVersion != 1 || outcome.Operation != name || outcome.Status != "applied" {
					t.Fatalf("unexpected workflow result: %s (%v)", output, err)
				}
				var compact bytes.Buffer
				if err := json.Compact(&compact, outcome.Plan); err != nil || !bytes.Equal(compact.Bytes(), displayed) {
					t.Fatal("applied a different plan from the full terminal display")
				}
				if name == "init" {
					var applied adoption.ApplyResult
					if err := json.Unmarshal(outcome.Apply, &applied); err != nil || applied.Status != "applied" || len(applied.Created) != 4 {
						t.Fatalf("initialization did not create four outputs: %s", outcome.Apply)
					}
				} else {
					var plan adoption.UpdatePlan
					if err := json.Unmarshal(displayed, &plan); err != nil || len(plan.Edits) != 1 || plan.Edits[0].Path != "docs/README.md" || plan.Edits[0].After != after {
						t.Fatalf("update lost the independently authored full replacement: %s", displayed)
					}
					checkDiagnostics(t, adoption.Validation{Findings: plan.Findings, Skipped: plan.Skipped}, true, 1)
					var applied adoption.UpdateResult
					if err := json.Unmarshal(outcome.Apply, &applied); err != nil || applied.Status != "applied" || len(applied.Replaced) != 1 || applied.Replaced[0].Path != "docs/README.md" {
						t.Fatalf("update did not report one selected replacement: %s", outcome.Apply)
					}
				}
			case "initial-review", "broken-review", "final-review":
				var got struct {
					Status     string
					Inventory  adoption.Inventory
					Validation adoption.Validation
				}
				readJSON(t, name+".json", &got)
				broken, skips, status := name == "broken-review", 1, "no-findings"
				if broken {
					status = "findings"
				}
				if name == "initial-review" {
					skips = 0
				}
				if got.Status != status || len(got.Inventory.Files) != 4 {
					t.Fatalf("unexpected review: %+v", got)
				}
				checkDiagnostics(t, got.Validation, broken, skips)
				if name == "final-review" {
					data, err := os.ReadFile(filepath.Join(project, "docs/README.md"))
					if err != nil || string(data) != after || len(data) != 154 {
						t.Fatalf("final bytes differ from the tutorial contract: %q (%v)", data, err)
					}
					if !strings.HasPrefix(output, "Replacement matches all 154 requested bytes; other collected files unchanged.\n") {
						t.Fatal("tutorial's preservation assertions did not complete")
					}
				}
			case "collect", "assess":
				var got adoption.AssessmentResult
				file := "collected.json"
				if name == "assess" {
					file = "assessment.json"
				}
				readJSON(t, file, &got)
				if got.Status != "review-required" || got.ValidationExitCode != 1 || len(got.Evidence) != 6 {
					t.Fatalf("lost original evidence and validation outcome: %+v", got)
				}
				checkDiagnostics(t, got.Validation, true, 1)
				for _, item := range got.Evidence {
					data, err := os.ReadFile(filepath.Join(project, item.Path))
					if err != nil || !item.Exists || string(data) != item.Content || item.SHA256 != fmt.Sprintf("%x", sha256.Sum256(data)) {
						t.Fatalf("evidence did not preserve exact source bytes: %s (%v)", item.Path, err)
					}
				}
				if name == "collect" && (got.ReportPath != "" || len(got.Brief.Proposals)+len(got.Brief.Questions) != 0) {
					t.Fatal("collection inferred advice or published a report")
				}
				if name == "assess" {
					info, err := os.Stat(got.ReportPath)
					if err != nil || got.ReportPath != filepath.Join(review, "assessment.md") || info.Mode().Perm() != 0o600 || len(got.Brief.Proposals) != 1 || len(got.Brief.Questions) != 1 {
						t.Fatalf("assessment lost private external authored advice: %+v (%v)", got, err)
					}
				}
			case "cleanup":
				for _, path := range []string{project, review} {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Fatalf("tutorial did not clean up %s: %v", path, err)
					}
				}
			}
		}) {
			t.FailNow() // Later steps depend on this exact same fixture.
		}
	}
}

func runTutorialBlock(t *testing.T, repository string, env []string, block, project, operation string) (string, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// Each shell shares the tutorial's initial private umask and variables.
	cmd := exec.CommandContext(ctx, "bash", "--noprofile", "--norc", "-eu", "-c", "umask 077\n"+block)
	cmd.Dir, cmd.Env = repository, env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if operation == "" {
		if err := cmd.Run(); err != nil {
			t.Fatalf("tutorial shell block: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
		}
		return stdout.String(), nil
	}
	before := assessmentTree(t, project)
	master, slave := assessmentPTY(t)
	cmd.Stdin, cmd.Stderr = slave, slave
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	terminal := assessmentWait(t, master, "Type decline (without a digest)")
	start := strings.Index(terminal, "{\r\n")
	if start < 0 {
		t.Fatal("tutorial workflow did not display its full plan")
	}
	var displayed json.RawMessage
	if err := json.NewDecoder(strings.NewReader(terminal[start:])).Decode(&displayed); err != nil {
		t.Fatal(err)
	}
	var plan struct {
		Operation, PlanDigest string
		Veto                  bool
	}
	if err := json.Unmarshal(displayed, &plan); err != nil || plan.Veto || len(plan.PlanDigest) != 71 || !strings.HasPrefix(plan.PlanDigest, "sha256:") {
		t.Fatalf("tutorial has no eligible displayed plan: %s (%v)", displayed, err)
	}
	if assessmentTree(t, project) != before {
		t.Fatal("tutorial workflow wrote before the fresh fixture decision")
	}
	// Test-only simulated decisions, after each full display, for these private
	// fixtures. Never a human approval, production bypass or pilot result.
	if _, err := unix.Write(int(master.Fd()), []byte("apply "+plan.PlanDigest+"\n")); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("tutorial %s: %v\nstdout=%s\nterminal=%s%s", operation, err, stdout.String(), terminal, assessmentDrain(t, master))
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, displayed); err != nil {
		t.Fatal(err)
	}
	return stdout.String(), compact.Bytes()
}
