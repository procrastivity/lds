//go:build linux

package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
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
		{[]string{"facts", "--resume", "draft.json", "--json"}, 3},
		{[]string{"facts", "--review", "draft.json", "--json"}, 3},
		{[]string{"facts", "--resume", "draft.json", "--review", "draft.json"}, 2},
	} {
		got := run(t, nil, tc.args...)
		if got.exitCode != tc.exit || got.stdout != "" {
			t.Fatalf("args=%v exit=%d stdout=%q stderr=%q", tc.args, got.exitCode, got.stdout, got.stderr)
		}
	}
}

func TestFactsResumePreservesAssertionsAndResetsApproval(t *testing.T) {
	for _, mode := range []string{"edit", "unchanged", "cancel", "stale"} {
		t.Run(mode, func(t *testing.T) {
			bundle := factsFixture(t)
			bundle.Limitations = "Keep this longer, multiline assertion.\nStill unverified outside the disposable fixture."
			original, err := json.Marshal(bundle)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			path := filepath.Join(root, "saved.json")
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			master, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs([]string{"facts", "--resume", path, "--json"})
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			assessmentWait(t, master, "Field> ")
			if mode == "edit" {
				for _, edit := range []struct{ field, value string }{{"Purpose", "A deliberately different purpose"}, {"OutputEvidence", ""}} {
					if _, err := unix.Write(int(master.Fd()), []byte(edit.field+"\n")); err != nil {
						t.Fatal(err)
					}
					assessmentWait(t, master, "\n> ")
					if _, err := unix.Write(int(master.Fd()), []byte(edit.value+"\n")); err != nil {
						t.Fatal(err)
					}
					assessmentWait(t, master, "Field> ")
				}
				bundle.Purpose, bundle.OutputEvidence = "A deliberately different purpose", ""
			}
			decision := "done\n"
			wantExit := 0
			if mode == "cancel" {
				decision, wantExit = ":cancel\n", 3
			}
			if mode == "stale" {
				original = append(original, '\n')
				if err := os.WriteFile(path, original, 0o600); err != nil {
					t.Fatal(err)
				}
				wantExit = 3
			}
			if _, err := unix.Write(int(master.Fd()), []byte(decision)); err != nil {
				t.Fatal(err)
			}
			select {
			case exit := <-done:
				if exit != wantExit {
					t.Fatalf("exit=%d terminal=%s", exit, assessmentDrain(t, master))
				}
			case <-time.After(5 * time.Second):
				t.Fatal("resume did not finish")
			}
			if wantExit == 0 {
				bundle.Approval.Approver, bundle.Approval.Status = "", "pending"
				bundle.Approval.ApprovedFacts = []string{}
				var got adoption.FactBundle
				if err := json.Unmarshal(stdout.Bytes(), &got); err != nil || !reflect.DeepEqual(got, bundle) {
					t.Fatalf("resume lost untouched assertions, invented evidence or retained approval: %s (%v)", stdout.String(), err)
				}
			} else if stdout.Len() != 0 {
				t.Fatalf("refused resume emitted a bundle: %s", stdout.String())
			}
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(actual, original) {
				t.Fatal("resume overwrote the saved input")
			}
		})
	}
}

func TestFactsReviewRequiresFreshExactApproval(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENCODE_API_KEY", "")
	for _, mode := range []string{"approve", "decline", "bare-digest", "wrong-digest", "queued", "eof", "changed-bytes", "replaced-inode"} {
		t.Run(mode, func(t *testing.T) {
			bundle := factsFixture(t)
			if mode == "approve" {
				bundle.Approval.Approver, bundle.Approval.Status = "", "pending"
				bundle.Approval.ApprovedFacts = []string{}
			}
			original, err := json.Marshal(bundle)
			if err != nil {
				t.Fatal(err)
			}
			root, inputs := t.TempDir(), t.TempDir()
			path := filepath.Join(inputs, "facts.json")
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			bundle.Approval.Approver, bundle.Approval.Status = "Test-only fresh reviewer", "approved"
			bundle.Approval.ApprovedFacts = []string{"DisplayName", "Purpose", "CurrentCapability", "VerifiedUserOperation"}
			expected, err := json.MarshalIndent(bundle, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			expected = append(expected, '\n')
			digest := fmt.Sprintf("sha256:%x", sha256.Sum256(expected))
			approval := "approve facts " + digest + "\n"
			master, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs([]string{"facts", "--review", path, "--json"})
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			assessmentWait(t, master, "Reviewer name (required):")
			input := bundle.Approval.Approver + "\n"
			if mode == "queued" {
				input += approval
			}
			if _, err := unix.Write(int(master.Fd()), []byte(input)); err != nil {
				t.Fatal(err)
			}
			terminal := assessmentWait(t, master, "Type decline or anything else to refuse.")
			if !strings.Contains(strings.ReplaceAll(terminal, "\r\n", "\n"), string(expected)) || !strings.Contains(terminal, "approve facts "+digest) {
				t.Fatal("facts approval did not display the exact full candidate and its independently computed digest")
			}
			switch mode {
			case "decline":
				approval = "decline\n"
			case "bare-digest":
				approval = digest + "\n"
			case "wrong-digest":
				approval = "approve facts sha256:" + strings.Repeat("0", 64) + "\n"
			case "queued", "eof":
				approval = "\x04"
			case "changed-bytes":
				original = append(original, '\n')
				if err := os.WriteFile(path, original, 0o600); err != nil {
					t.Fatal(err)
				}
			case "replaced-inode":
				replacement := filepath.Join(inputs, "replacement.json")
				if err := os.WriteFile(replacement, original, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(replacement, path); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := unix.Write(int(master.Fd()), []byte(approval)); err != nil {
				t.Fatal(err)
			}
			select {
			case exit := <-done:
				if mode == "approve" {
					if exit != 0 || !bytes.Equal(stdout.Bytes(), expected) {
						t.Fatalf("incorrect approved artifact: exit=%d stdout=%s", exit, stdout.String())
					}
					output := filepath.Join(inputs, "approved.json")
					if err := os.WriteFile(output, stdout.Bytes(), 0o600); err != nil {
						t.Fatal(err)
					}
					if rendered, err := adoption.Render(root, output, ""); err != nil || len(rendered) != 4 {
						t.Fatalf("approved artifact incompatible with adoption: %v", err)
					}
				} else if exit != 3 || stdout.Len() != 0 {
					t.Fatalf("approval bypass or missing stale refusal: exit=%d stdout=%s terminal=%s", exit, stdout.String(), assessmentDrain(t, master))
				}
			case <-time.After(5 * time.Second):
				t.Fatal("facts review did not finish")
			}
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(actual, original) {
				t.Fatal("facts review overwrote the saved input")
			}
			if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
				t.Fatal("facts approval or rendering wrote adoption outputs")
			}
		})
	}
}

func TestFactsSavedInputsRefuseAmbiguityAndIncompleteApproval(t *testing.T) {
	valid, err := json.Marshal(factsFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"unknown", "citation", "duplicate", "case", "missing", "null", "approval-null", "type", "trailing", "control", "invalid-utf8", "oversize", "encoded-oversize", "symlink", "fifo"} {
		t.Run(mode, func(t *testing.T) {
			flag, wantExit := "--resume", 1
			var fields map[string]any
			if err := json.Unmarshal(valid, &fields); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "unknown":
				fields["OutputEvidence"], flag = "", "--review"
			case "citation":
				fields["OperationSource"], flag = "../outside.sh", "--review"
			case "case":
				fields["purpose"] = fields["Purpose"]
			case "missing":
				delete(fields, "Purpose")
			case "null":
				fields["Purpose"] = nil
			case "approval-null":
				fields["approval"] = nil
			case "type":
				fields["Purpose"] = 42
			case "control":
				fields["Purpose"] = "unsafe\x1btext"
			case "encoded-oversize":
				fields["Purpose"] = strings.Repeat("<", 50000) // JSON escaping exceeds the bound although the input fits.
			}
			data, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "duplicate":
				data = append([]byte(`{"Purpose":"conflicting",`), data[1:]...)
			case "trailing":
				data = append(data, []byte("{}")...)
			case "invalid-utf8":
				data = bytes.Replace(data, []byte(`"Purpose":"`), []byte("\"Purpose\":\"\xff"), 1)
			case "encoded-oversize":
				data = bytes.ReplaceAll(data, []byte(`\u003c`), []byte("<"))
				if len(data) >= 256<<10 {
					t.Fatal("encoded-output test must fit the input bound")
				}
			case "oversize":
				data = []byte(strings.Repeat(" ", (256<<10)+1))
			}
			path := filepath.Join(t.TempDir(), "facts.json")
			switch mode {
			case "symlink":
				if err := os.Symlink("missing", path); err != nil {
					t.Fatal(err)
				}
				wantExit = 3
			case "fifo":
				if err := unix.Mkfifo(path, 0o600); err != nil {
					t.Fatal(err)
				}
				wantExit = 3
			default:
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			_, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs([]string{"facts", flag, path, "--json"})
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			select {
			case exit := <-done:
				if exit != wantExit || stdout.Len() != 0 {
					t.Fatalf("invalid saved input accepted: exit=%d stdout=%q", exit, stdout.String())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("invalid saved input blocked or requested approval")
			}
		})
	}
}

func factsFixture(t *testing.T) adoption.FactBundle {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "asset", "testdata", "adoption", "greenfield", "facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bundle adoption.FactBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	return bundle
}
