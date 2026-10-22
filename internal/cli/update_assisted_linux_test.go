//go:build linux

package cli_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/buildinfo"
	"github.com/procrastivity/lds/internal/cli"
	"github.com/procrastivity/lds/internal/iostreams"
)

func TestAssistedUpdateConsentScopeAndFreshApply(t *testing.T) {
	for _, name := range []string{"apply", "decline-send", "decline-apply", "queued-send", "queued-apply", "stale-send", "stale-response", "stale-apply", "flow-response", "flow-apply", "unselected-document", "uncaptured-citation", "malformed", "no-edits"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("OPENCODE_API_KEY", "mock-only-not-a-credential")
			root, xdg := t.TempDir(), t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdg)
			write := func(path, value string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(value), 0o640); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.MkdirAll(filepath.Join(root, "docs"), 0o750); err != nil {
				t.Fatal(err)
			}
			write(filepath.Join(root, "README.md"), "# Before\nUse the old spelling.\n")
			write(filepath.Join(root, "docs/other.md"), "Collected but not eligible for this draft.\n")
			write(filepath.Join(root, "command.txt"), "Supported spelling: --new\n")
			write(filepath.Join(root, "private.txt"), "UNSELECTED PRIVATE MARKER\n")
			flowPath := filepath.Join(xdg, "lds/flows/update.md")
			if err := os.MkdirAll(filepath.Dir(flowPath), 0o700); err != nil {
				t.Fatal(err)
			}
			write(flowPath, "# Live update flow\nRequire separate human approval.\n")
			before := assessmentTree(t, root)
			draft := adoption.UpdateDraft{SchemaVersion: 1, Edits: []adoption.UpdateProposal{{Path: "README.md", After: "# After 😀\nUse --new.\n", Rationale: "Match captured source", Citations: []string{"README.md", "command.txt"}}}, Questions: []string{"Is the source complete?"}}
			observed, err := adoption.CollectAssessment(root, "Review spelling", "Maintainers", []string{"command.txt"})
			if err != nil {
				t.Fatal(err)
			}
			plan, err := adoption.PlanUpdatesReviewed(root, observed, adoption.UpdateRequest{SchemaVersion: 1, Edits: draft.Edits})
			if err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			previous := http.DefaultTransport
			http.DefaultTransport = assessmentTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				var request struct {
					Model    string                           `json:"model"`
					Messages []struct{ Role, Content string } `json:"messages"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					return nil, err
				}
				if request.Model != "deepseek-v4-flash" || len(request.Messages) != 2 || strings.Contains(request.Messages[1].Content, root) || strings.Contains(request.Messages[1].Content, "UNSELECTED PRIVATE MARKER") {
					t.Error("outbound payload leaked host metadata/unselected content or changed model")
				}
				var payload struct {
					Flow     string                        `json:"flow"`
					Selected []string                      `json:"selectedDocuments"`
					Evidence []adoption.AssessmentEvidence `json:"evidence"`
				}
				if err := json.Unmarshal([]byte(request.Messages[1].Content), &payload); err != nil || len(payload.Selected) != 1 || payload.Selected[0] != "README.md" || !strings.Contains(payload.Flow, "Live update flow") || len(payload.Evidence) != 3 {
					t.Error("outbound scope/flow/evidence was not preserved")
				}
				if name == "stale-response" {
					write(filepath.Join(root, "command.txt"), "Changed during provider response\n")
					before = assessmentTree(t, root)
				}
				if name == "flow-response" {
					write(flowPath, "# Changed flow\n")
				}
				if name == "unselected-document" {
					draft.Edits[0].Path = "docs/other.md"
				}
				if name == "uncaptured-citation" {
					draft.Edits[0].Citations = []string{"private.txt"}
				}
				if name == "no-edits" {
					draft.Edits = []adoption.UpdateProposal{}
				}
				content, _ := json.Marshal(draft)
				if name == "malformed" {
					content = []byte(`{"schemaVersion":1,"edits":[],"questions":[],"approve":true}`)
				}
				body, _ := json.Marshal(map[string]any{"model": "deepseek-v4-flash", "choices": []any{map[string]any{"message": map[string]any{"content": string(content)}}}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
			})
			t.Cleanup(func() { http.DefaultTransport = previous })
			master, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs([]string{"update", root, "--assist", "--reason", "Review spelling", "--audience", "Maintainers", "--document", "README.md", "--source", "command.txt", "--json"})
			if name == "queued-send" {
				if _, err := unix.Write(int(master.Fd()), []byte("send\n")); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			terminal := assessmentWait(t, master, "or anything else to decline:")
			if name == "stale-send" {
				write(filepath.Join(root, "command.txt"), "Changed before sending\n")
				before = assessmentTree(t, root)
			}
			consent := "send\n"
			if name == "decline-send" {
				consent = "decline\n"
			}
			if name == "queued-send" {
				consent = "\x04"
			}
			if name == "queued-apply" {
				consent += "apply " + plan.PlanDigest + "\n"
			}
			if _, err := unix.Write(int(master.Fd()), []byte(consent)); err != nil {
				t.Fatal(err)
			}
			needsApply := name == "apply" || name == "decline-apply" || name == "queued-apply" || name == "stale-apply" || name == "flow-apply"
			if needsApply {
				terminal += assessmentWait(t, master, "Type decline (without a digest)")
				if assessmentTree(t, root) != before {
					t.Fatal("send or draft wrote target before apply approval")
				}
				if name == "stale-apply" {
					write(filepath.Join(root, "command.txt"), "Changed during plan review\n")
					before = assessmentTree(t, root)
				}
				if name == "flow-apply" {
					write(flowPath, "# Changed after plan\n")
				}
				approval := "apply " + plan.PlanDigest + "\n"
				if name == "decline-apply" {
					approval = "decline\n"
				}
				if name == "queued-apply" {
					approval = "\x04"
				}
				if _, err := unix.Write(int(master.Fd()), []byte(approval)); err != nil {
					t.Fatal(err)
				}
			}
			var exit int
			select {
			case exit = <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("update did not terminate")
			}
			terminal += assessmentDrain(t, master)
			wantCalls := int32(1)
			if name == "decline-send" || name == "queued-send" || name == "stale-send" {
				wantCalls = 0
			}
			if calls.Load() != wantCalls {
				t.Fatalf("calls=%d want=%d", calls.Load(), wantCalls)
			}
			if name == "apply" {
				if exit != 0 {
					t.Fatalf("apply exit=%d terminal=%s", exit, terminal)
				}
				data, _ := os.ReadFile(filepath.Join(root, "README.md"))
				if string(data) != "# After 😀\nUse --new.\n" {
					t.Fatalf("wrong replacement %q", data)
				}
				// Independently require the exact tree we expected, including preserved neighbors/modes.
				write(filepath.Join(root, "README.md"), "# Before\nUse the old spelling.\n")
			} else if name == "no-edits" {
				var result struct {
					Status      string
					Plan, Apply any
					Draft       adoption.UpdateDraft
				}
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || exit != 0 || result.Status != "review-required" || result.Plan != nil || result.Apply != nil || len(result.Draft.Questions) != 1 {
					t.Fatalf("empty advice: exit=%d result=%s err=%v", exit, stdout.Bytes(), err)
				}
			} else if exit == 0 {
				t.Fatalf("expected refusal/validation, terminal=%s", terminal)
			}
			if assessmentTree(t, root) != before {
				t.Fatal("unexpected target effects")
			}
		})
	}
}
