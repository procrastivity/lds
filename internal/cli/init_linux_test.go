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

func TestInitFreshReviewAndCreateOnlyEffects(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENCODE_API_KEY", "")
	fixture := filepath.Join("..", "asset", "testdata", "adoption", "greenfield")
	factsBytes, err := os.ReadFile(filepath.Join(fixture, "facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"apply", "current", "partial-tree", "decline", "bare-digest", "wrong-digest", "queued-approval", "eof", "facts-changed", "target-changed", "template-changed", "collision", "incomplete-facts", "unsafe-parent"} {
		t.Run(name, func(t *testing.T) {
			root, inputs, projectRoot := t.TempDir(), t.TempDir(), t.TempDir()
			write := func(path, content string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
					t.Fatal(err)
				}
			}
			facts := filepath.Join(inputs, "facts.json")
			write(facts, string(factsBytes))
			write(filepath.Join(root, "README.md"), "# Keep my root signpost\n")
			write(filepath.Join(root, "source.txt"), "preserve exact source bytes\n")
			template := filepath.Join(projectRoot, ".lds/assets/templates/adoption/docs/README.md")
			write(template, "# Project-local {{.DisplayName}}\n")
			paths := []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"}
			want := map[string]string{"docs/README.md": "# Project-local Wordcount\n"}
			for _, path := range paths[1:] {
				data, err := os.ReadFile(filepath.Join(fixture, "expected", path))
				if err != nil {
					t.Fatal(err)
				}
				want[path] = string(data)
			}
			if name == "current" || name == "partial-tree" {
				for _, path := range paths {
					if name == "current" || path == "docs/README.md" {
						write(filepath.Join(root, path), want[path])
					}
				}
			}
			vetoCode := ""
			switch name {
			case "collision":
				write(filepath.Join(root, "engineering/README.md"), "# Existing, different engineering guide\n")
				vetoCode = "refusal.output-collision"
			case "incomplete-facts":
				write(facts, "{}")
				vetoCode = "refusal.incomplete-facts"
			case "unsafe-parent":
				if err := os.Symlink(inputs, filepath.Join(root, "docs")); err != nil {
					t.Fatal(err)
				}
				vetoCode = "refusal.unsafe-target"
			}
			plan, planErr := adoption.Plan(root, facts, projectRoot)
			if (planErr != nil) != (vetoCode != "") {
				t.Fatalf("unexpected preflight: %v", planErr)
			}
			expectedPlan, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			before := assessmentTree(t, root)
			inputsBefore := assessmentTree(t, inputs)
			master, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs([]string{"init", root, "--facts", facts, "--project-root", projectRoot, "--json"})
			if name == "queued-approval" {
				if _, err := unix.Write(int(master.Fd()), []byte("apply "+*plan.PlanDigest+"\n")); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			var terminal string
			if vetoCode != "" {
				terminal = assessmentWait(t, master, `"code":"`+vetoCode+`"`)
				if strings.Contains(terminal, "To authorize") {
					t.Fatal("vetoed preflight requested approval")
				}
			} else {
				terminal = assessmentWait(t, master, "Type decline (without a digest)")
				if assessmentTree(t, root) != before {
					t.Fatal("initialization wrote before fresh approval")
				}
				switch name {
				case "facts-changed":
					write(facts, string(factsBytes)+"\n")
					inputsBefore = assessmentTree(t, inputs)
				case "target-changed":
					write(filepath.Join(root, "README.md"), "# Changed during review\n")
					before = assessmentTree(t, root)
				case "template-changed":
					write(template, "# Unreviewed {{.DisplayName}}\n")
				}
				decision := "apply " + *plan.PlanDigest + "\n"
				switch name {
				case "decline":
					decision = "decline\n"
				case "bare-digest":
					decision = *plan.PlanDigest + "\n"
				case "wrong-digest":
					decision = "apply sha256:" + strings.Repeat("0", 64) + "\n"
				case "queued-approval", "eof":
					decision = "\x04"
				}
				if _, err := unix.Write(int(master.Fd()), []byte(decision)); err != nil {
					t.Fatal(err)
				}
			}
			success := name == "apply" || name == "current" || name == "partial-tree"
			wantExit := 3
			if success {
				wantExit = 0
			}
			select {
			case exit := <-done:
				terminal += assessmentDrain(t, master)
				if exit != wantExit {
					t.Fatalf("exit=%d want=%d terminal=%s", exit, wantExit, terminal)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("init did not finish")
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
				t.Fatalf("display omitted or changed plan fields: %v", err)
			}
			var result struct {
				SchemaVersion     int
				Operation, Status string
				Plan              json.RawMessage
				Apply             *adoption.ApplyResult
			}
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.SchemaVersion != 1 || result.Operation != "init" || !bytes.Equal(result.Plan, expectedPlan) {
				t.Fatalf("invalid final result: %s (%v)", stdout.String(), err)
			}
			if success {
				created := 4
				status := "applied"
				switch name {
				case "current":
					created, status = 0, "current"
				case "partial-tree":
					created = 3
				}
				if result.Apply == nil || result.Status != status || result.Apply.Status != status || len(result.Apply.Created) != created || len(result.Apply.IdenticalUntouched) != 4-created {
					t.Fatalf("incorrect effects: %s", stdout.String())
				}
				for _, path := range paths {
					data, err := os.ReadFile(filepath.Join(root, path))
					if err != nil || string(data) != want[path] {
						t.Fatalf("lost exact expected bytes for %s: %q (%v)", path, data, err)
					}
					if name != "current" && (name != "partial-tree" || path != "docs/README.md") {
						if err := os.Remove(filepath.Join(root, path)); err != nil {
							t.Fatal(err)
						}
					}
				}
				// Remove only newly necessary parents to compare every preexisting byte/mode.
				for i := len(plan.NecessaryParentDirectories) - 1; i >= 0; i-- {
					if err := os.Remove(filepath.Join(root, plan.NecessaryParentDirectories[i])); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				code := vetoCode
				if code == "" {
					code = "refusal.confirmation-declined"
				}
				stale := strings.HasSuffix(name, "-changed")
				if stale {
					code = "refusal.plan-stale"
				}
				if result.Status != "refused" || (result.Apply != nil) != stale || !strings.Contains(terminal, `"code":"`+code+`"`) {
					t.Fatalf("incorrect refusal: %s / %s", stdout.String(), terminal)
				}
			}
			if assessmentTree(t, root) != before || assessmentTree(t, inputs) != inputsBefore {
				t.Fatal("unreviewed writes, changed existing metadata, or scratch artifacts")
			}
		})
	}
}
