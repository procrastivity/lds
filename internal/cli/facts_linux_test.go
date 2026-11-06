//go:build linux

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/buildinfo"
	"github.com/procrastivity/lds/internal/cli"
	"github.com/procrastivity/lds/internal/iostreams"
)

func TestFactsQuestionnaireProducesOnlyUnapprovedAssertions(t *testing.T) {
	t.Setenv("OPENCODE_API_KEY", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, mode := range []string{"complete", "unknowns", "byte-limit"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "must-not-execute")
			answers := map[string]string{
				"DisplayName": "A different \"name\" — café", "DisplayNameSource": "identity.md:2",
				"Purpose": "Count things", "PurposeSource": "purpose.txt:3",
				"CurrentCapability": "One supported operation", "CurrentCapabilitySource": "current.go:4",
				"Prerequisites": "A trusted shell", "OperationName": "Record", "OperationSummary": "Records literal text",
				"Invocation": "touch " + marker, "Input": "sample.txt", "OutputEvidence": "source-backed test-only observation",
				"OperationSource": "operation.sh:5", "Limitations": "No real project was inspected",
			}
			if mode == "unknowns" {
				answers["CurrentCapability"], answers["OutputEvidence"] = "", ""
			}
			if mode == "byte-limit" {
				answers["Purpose"] = strings.Repeat("é", 1024) // Exactly 2048 bytes, not 2048 characters.
			}
			before := assessmentTree(t, root)
			master, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			// An irrelevant, nonexistent selector must not cause project/asset reads.
			args := []string{"facts", "--project-root", filepath.Join(root, "nonexistent")}
			if mode != "unknowns" {
				args = append(args, "--json")
			}
			command.SetArgs(args)
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			seen := map[string]bool{}
			for range len(answers) {
				terminal := assessmentWait(t, master, "\n> ")
				lines := strings.Split(strings.TrimSpace(terminal), "\n")
				key, _, ok := strings.Cut(lines[len(lines)-2], " — ")
				answer, exists := answers[key]
				if !ok || !exists || seen[key] {
					t.Fatalf("unexpected or repeated question: %s", terminal)
				}
				seen[key] = true
				if _, err := unix.Write(int(master.Fd()), []byte(answer+"\n")); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case code := <-done:
				if code != 0 {
					t.Fatalf("exit=%d; terminal=%s", code, assessmentDrain(t, master))
				}
			case <-time.After(5 * time.Second):
				t.Fatal("facts did not finish")
			}
			var got map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatalf("not a single raw fact bundle: %s (%v)", stdout.String(), err)
			}
			want := map[string]any{"approval": map[string]any{"approver": "", "approvedFacts": []any{}, "status": "pending"}}
			for key, value := range answers {
				want[key] = value
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("changed, inferred or approved assertions: got=%v want=%v", got, want)
			}
			factsPath := filepath.Join(t.TempDir(), "draft.json")
			if err := os.WriteFile(factsPath, stdout.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := adoption.Render(root, factsPath, ""); !errors.Is(err, adoption.ErrInvalidRenderInput) {
				t.Fatalf("render accepted an unapproved draft: %v", err)
			}
			if mode == "complete" {
				// Disposable simulated approval verifies compatibility with the existing
				// consumer; the questionnaire itself never emits this authorization.
				got["approval"] = map[string]any{"approver": "test-only simulated approver", "approvedFacts": []string{"DisplayName", "Purpose", "CurrentCapability", "VerifiedUserOperation"}, "status": "test-only simulated approval; not a real project authorization"}
				approved, err := json.Marshal(got)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(factsPath, approved, 0o600); err != nil {
					t.Fatal(err)
				}
				outputs, err := adoption.Render(root, factsPath, "")
				if err != nil || len(outputs) != 4 {
					t.Fatalf("incompatible facts draft: outputs=%d, err=%v", len(outputs), err)
				}
			}
			if assessmentTree(t, root) != before {
				t.Fatal("questionnaire or render wrote to the project or executed the invocation")
			}
		})
	}
}

func TestFactsCancelledOrInvalidInputEmitsNoDraft(t *testing.T) {
	for _, tc := range []struct {
		name, input, code string
		exit              int
	}{
		{"cancel", ":cancel\n", "refusal.facts-cancelled", 3},
		{"eof", "\x04", "refusal.facts-cancelled", 3},
		{"over-byte-limit", strings.Repeat("é", 1024) + "x\n", "validation.facts-answer", 1},
		{"buffer-limit", strings.Repeat("x", 3000) + "\n", "validation.facts-answer", 1},
		{"invalid-utf8", "\xff\n", "validation.facts-answer", 1},
		{"control", "one\ttwo\n", "validation.facts-answer", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			master, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs([]string{"facts", "--json"})
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			assessmentWait(t, master, "\n> ")
			if _, err := unix.Write(int(master.Fd()), []byte("A partially answered draft\n")); err != nil {
				t.Fatal(err)
			}
			assessmentWait(t, master, "\n> ")
			if _, err := unix.Write(int(master.Fd()), []byte(tc.input)); err != nil {
				t.Fatal(err)
			}
			select {
			case exit := <-done:
				terminal := assessmentDrain(t, master)
				if exit != tc.exit || stdout.Len() != 0 || !strings.Contains(terminal, `"code":"`+tc.code+`"`) {
					t.Fatalf("exit=%d stdout=%q terminal=%s", exit, stdout.String(), terminal)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("invalid facts input did not finish")
			}
		})
	}
}

func TestFactsRefusesNoninteractiveAndUnexpectedArguments(t *testing.T) {
	for _, tc := range []struct {
		args []string
		exit int
	}{
		{[]string{"facts", "--json"}, 3},
		{[]string{"facts", "/not/a/target", "--json"}, 2},
		{[]string{"facts", "--yes", "--json"}, 2},
	} {
		got := run(t, nil, tc.args...)
		if got.exitCode != tc.exit || got.stdout != "" {
			t.Fatalf("args=%v exit=%d stdout=%q stderr=%q", tc.args, got.exitCode, got.stdout, got.stderr)
		}
	}
}
