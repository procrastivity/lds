//go:build linux

package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/buildinfo"
	"github.com/procrastivity/lds/internal/cli"
	"github.com/procrastivity/lds/internal/iostreams"
)

func TestUpdateCommandFreshReviewAndExactEffects(t *testing.T) {
	for _, name := range []string{"apply", "decline", "wrong-digest", "bare-digest", "queued-approval", "eof", "stale", "request-changed", "veto"} {
		t.Run(name, func(t *testing.T) {
			root, inputs := t.TempDir(), t.TempDir()
			write := func(path, content string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
					t.Fatal(err)
				}
			}
			readme := filepath.Join(root, "README.md")
			write(readme, "# Before\nOld text.\n")
			write(filepath.Join(root, "source.txt"), "Supported wording.\n")
			observed, err := adoption.CollectAssessment(root, "Review wording", "Maintainers", []string{"source.txt"})
			if err != nil {
				t.Fatal(err)
			}
			request := adoption.UpdateRequest{SchemaVersion: 1, Edits: []adoption.UpdateProposal{{
				Path: "README.md", After: "# After 😀\nNew wording.\n", Rationale: "Correct the wording", Citations: []string{"README.md", "source.txt"},
			}}}
			if name == "veto" {
				request.Edits[0].Path = "absent.md"
			}
			observedBytes, err := json.Marshal(observed)
			if err != nil {
				t.Fatal(err)
			}
			requestBytes, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			observedPath, requestPath := filepath.Join(inputs, "observed.json"), filepath.Join(inputs, "request.json")
			write(observedPath, string(observedBytes))
			write(requestPath, string(requestBytes))
			plan, err := adoption.PlanUpdates(root, observedPath, requestPath)
			if (err != nil) != (name == "veto") {
				t.Fatal(err)
			}
			expectedPlan, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			before := assessmentTree(t, root)
			master, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs([]string{"update", root, "--observed", observedPath, "--request", requestPath, "--json"})
			if name == "queued-approval" {
				if _, err := unix.Write(int(master.Fd()), []byte("apply "+plan.PlanDigest+"\n")); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			var terminal string
			if name != "veto" {
				terminal = assessmentWait(t, master, "Type decline (without a digest)")
				if assessmentTree(t, root) != before {
					t.Fatal("target changed before fresh approval")
				}
				if name == "stale" {
					write(filepath.Join(root, "source.txt"), "Changed during review; preserve me.\n")
					before = assessmentTree(t, root)
				}
				if name == "request-changed" {
					write(requestPath, strings.ReplaceAll(string(requestBytes), "New wording.", "UNREVIEWED replacement."))
				}
				decision := "apply " + plan.PlanDigest + "\n"
				switch name {
				case "decline":
					decision = "decline\n"
				case "wrong-digest":
					decision = "apply sha256:" + strings.Repeat("0", 64) + "\n"
				case "bare-digest":
					decision = plan.PlanDigest + "\n"
				case "queued-approval", "eof":
					decision = "\x04"
				}
				if _, err := unix.Write(int(master.Fd()), []byte(decision)); err != nil {
					t.Fatal(err)
				}
			} else {
				terminal = assessmentWait(t, master, `"code":"refusal.update-veto"`)
				if strings.Contains(terminal, "To authorize") {
					t.Fatal("veto prompted for approval")
				}
			}
			wantExit := 3
			if name == "apply" || name == "request-changed" {
				wantExit = 0
			}
			select {
			case exit := <-done:
				terminal += assessmentDrain(t, master)
				if exit != wantExit {
					t.Fatalf("exit=%d want=%d terminal=%s", exit, wantExit, terminal)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("update did not finish")
			}
			start := strings.Index(terminal, "{\r\n")
			if start < 0 {
				t.Fatal("complete plan was not displayed")
			}
			var displayed json.RawMessage
			if err := json.NewDecoder(strings.NewReader(terminal[start:])).Decode(&displayed); err != nil {
				t.Fatal(err)
			}
			var compact bytes.Buffer
			if err := json.Compact(&compact, displayed); err != nil || !bytes.Equal(compact.Bytes(), expectedPlan) {
				t.Fatalf("display omitted or changed full deterministic plan: %v", err)
			}
			var result struct {
				SchemaVersion     int
				Operation, Status string
				Plan              json.RawMessage
				Apply             *adoption.UpdateResult
			}
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.SchemaVersion != 1 || result.Operation != "update" || !bytes.Equal(result.Plan, expectedPlan) {
				t.Fatalf("invalid final envelope: %s (%v)", stdout.String(), err)
			}
			if wantExit == 0 {
				data, err := os.ReadFile(readme)
				info, statErr := os.Stat(readme)
				if err != nil || statErr != nil || string(data) != request.Edits[0].After || info.Mode().Perm() != 0o640 || result.Status != "applied" || result.Apply == nil || len(result.Apply.Replaced) != 1 || result.Apply.Replaced[0].Path != "README.md" {
					t.Fatalf("lost exact reviewed content/mode/effects: %s %v %v", data, err, result.Apply)
				}
				write(readme, "# Before\nOld text.\n")
			} else if result.Status != "refused" || (result.Apply != nil) != (name == "stale") {
				t.Fatalf("incorrect refusal effects: %+v", result)
			}
			if after := assessmentTree(t, root); after != before {
				t.Fatalf("unapproved changes or scratch: before=%s after=%s", before, after)
			}
		})
	}
}
