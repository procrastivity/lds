//go:build linux

package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/procrastivity/lds/internal/buildinfo"
	"github.com/procrastivity/lds/internal/cli"
	"github.com/procrastivity/lds/internal/iostreams"
	core "github.com/procrastivity/lds/internal/reconcile"
)

func TestAssistedReconciliationConsentBaselineAndReport(t *testing.T) {
	for _, name := range []string{"report", "selected-map-citation", "questions-only", "decline", "queued-send", "changed-review", "changed-source", "changed-journal", "flow-before", "flow-response", "unselected-citation", "reviewed-claim"} {
		t.Run(name, func(t *testing.T) {
			setup := newJournalCLISetup(t)
			t.Setenv("OPENCODE_API_KEY", "mock-only-not-a-credential")
			xdg := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdg)
			write := func(path, data string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			review := filepath.Join(setup.base, "review.json")
			write(review, `{"schemaVersion":1,"purpose":"Review this pair","reviewer":"human-fixture","at":"2026-10-22T00:00:00Z","selections":[{"side":"left","path":"left.md"},{"side":"right","path":"maps/a.md"}],"claims":[],"questions":["Human question"]}`)
			write(setup.journal, `{"schemaVersion":1,"pairId":"fixture:journal-cli:v1","leftProjectId":"left","rightProjectId":"right","revision":0,"events":[]}`)
			flow := filepath.Join(xdg, "lds/flows/reconcile.md")
			if err := os.MkdirAll(filepath.Dir(flow), 0o700); err != nil {
				t.Fatal(err)
			}
			write(flow, "# Live reconciliation flow\nReport advice only.\n")
			output := filepath.Join(setup.base, "report.md")
			flags := []string{"--pair", setup.pairPath, "--left-root", setup.leftRoot, "--right-root", setup.rightRoot, "--map", filepath.Join(setup.rightRoot, "maps/a.md"), "--review", review, "--output", output, "--journal", setup.journal, "--json"}
			expected := assessmentTree(t, setup.base)
			var calls atomic.Int32
			var system, user string
			previous := http.DefaultTransport
			http.DefaultTransport = assessmentTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				var request struct {
					Model    string
					Messages []struct{ Role, Content string }
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					return nil, err
				}
				if request.Model != "deepseek-v4-flash" || len(request.Messages) != 2 {
					t.Fatal("unexpected provider request")
				}
				system, user = request.Messages[0].Content, request.Messages[1].Content
				for _, forbidden := range []string{setup.base, xdg, "right fixture evidence", "mock-only-not-a-credential", `"events"`} {
					if strings.Contains(user, forbidden) {
						t.Errorf("outbound host/unselected/journal disclosure %q", forbidden)
					}
				}
				var payload struct {
					Flow    string
					Scan    core.Response
					Content []struct{ Side, Path, Text, SHA256 string }
				}
				if err := json.Unmarshal([]byte(user), &payload); err != nil {
					return nil, err
				}
				if !strings.Contains(payload.Flow, "Live reconciliation flow") || len(payload.Content) != 2 || payload.Content[0].Text != "left fixture evidence\n" || payload.Content[1].Text != "map a\n" {
					t.Error("selected content/live flow lost")
				}
				candidate := payload.Scan.Candidates[0]
				path := "left.md"
				if name == "selected-map-citation" {
					path = "maps/a.md"
				}
				if name == "unselected-citation" {
					path = "right.md"
				}
				var row core.Evidence
				for _, evidence := range candidate.Evidence {
					if evidence.Key == path {
						row = evidence
					}
				}
				if row.Key == "" {
					t.Fatal("missing citation fixture")
				}
				claim := core.ReviewClaim{CandidateID: candidate.CandidateID, Status: "proposed", Correspondence: core.Correspondence{Key: "model:topic", LeftMembers: []string{"left:topic"}, RightMembers: []string{"right:topic"}, Relation: "one-to-one"}, Rationale: "Review this proposed relationship", Provenance: []core.Provenance{{Kind: row.Kind, ProjectID: row.ProjectID, Revision: &row.Revision, Path: row.Key, SHA256: &row.SHA256, Claim: "Captured text", Trust: "bounded"}}, Assessment: core.Assessment{Code: "needs-review", Confidence: core.Confidence{Level: "not-assessed"}, Trust: "bounded"}, Conflicts: []string{}, Questions: []string{}}
				if name == "reviewed-claim" {
					claim.Status = "reviewed"
				}
				draft := core.ReportDraft{SchemaVersion: 1, Claims: []core.ReviewClaim{claim}, Questions: []string{"Who declares authority?"}}
				if name == "questions-only" {
					draft.Claims = []core.ReviewClaim{}
				}
				if name == "changed-source" {
					write(filepath.Join(setup.leftRoot, "left.md"), "Changed while provider worked\n")
				}
				if name == "changed-journal" {
					data, _ := os.ReadFile(setup.journal)
					write(setup.journal, string(data)+"\n")
				}
				if name == "flow-response" {
					write(flow, "# Changed after sending\n")
				}
				expected = assessmentTree(t, setup.base)
				advice, _ := json.Marshal(draft)
				body, _ := json.Marshal(map[string]any{"model": "deepseek-v4-flash", "choices": []any{map[string]any{"message": map[string]any{"content": string(advice)}}}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
			})
			t.Cleanup(func() { http.DefaultTransport = previous })
			master, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs(append([]string{"reconcile", "report", "--assist"}, flags...))
			if name == "queued-send" {
				if _, err := unix.Write(int(master.Fd()), []byte("send\n")); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			terminal := assessmentWait(t, master, "or anything else to decline:")
			if !strings.Contains(terminal, "left fixture evidence") || !strings.Contains(terminal, "Complete user payload") || !strings.Contains(terminal, "System instructions") {
				t.Fatal("incomplete disclosure before consent")
			}
			decision := "send\n"
			if name == "decline" {
				decision = "decline\n"
			}
			if name == "queued-send" {
				decision = "\x04"
			}
			if name == "changed-review" {
				data, _ := os.ReadFile(review)
				write(review, string(data)+"\n")
				expected = assessmentTree(t, setup.base)
			}
			if name == "flow-before" {
				write(flow, "# Changed before send\n")
			}
			if _, err := unix.Write(int(master.Fd()), []byte(decision)); err != nil {
				t.Fatal(err)
			}
			var exit int
			select {
			case exit = <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("report did not terminate")
			}
			terminal += assessmentDrain(t, master)
			wantCalls := int32(1)
			if name == "decline" || name == "queued-send" || name == "changed-review" || name == "flow-before" {
				wantCalls = 0
			}
			if calls.Load() != wantCalls {
				t.Fatalf("calls=%d want=%d", calls.Load(), wantCalls)
			}
			if name == "report" || name == "selected-map-citation" || name == "questions-only" {
				var result core.ReportResult
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || exit != 0 || !result.Assistance || result.Status != "review-required" {
					t.Fatalf("report exit=%d result=%s error=%v terminal=%s", exit, stdout.Bytes(), err, terminal)
				}
				data, err := os.ReadFile(output)
				if err != nil {
					t.Fatal(err)
				}
				_, appendix, ok := strings.Cut(string(data), "## Provenance and selected content\n\n")
				if !ok {
					t.Fatal("missing appendix")
				}
				lines := strings.Split(strings.TrimSpace(appendix), "\n")
				var packet core.ReportEvidence
				if err := json.Unmarshal([]byte(strings.Join(lines[1:len(lines)-1], "\n")), &packet); err != nil {
					t.Fatal(err)
				}
				hash := func(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
				if result.ReportSHA256 != hash(data) || packet.Assistance == nil || packet.Assistance.SystemSHA256 != hash([]byte(system)) || packet.Assistance.UserSHA256 != hash([]byte(user)) || packet.Assistance.UserBytes != len(user) || packet.Advice == nil || packet.Review.Questions[0] != "Human question" || packet.Journal == nil || packet.Journal.Revision != 0 {
					t.Fatal("report receipt/advice/history not tied to actual request")
				}
				if name == "questions-only" && len(packet.Advice.Claims) != 0 {
					t.Fatal("empty advice invented a claim")
				}
				info, _ := os.Stat(output)
				if info.Mode().Perm() != 0o600 {
					t.Fatal("report not private")
				}
				if err := os.Remove(output); err != nil {
					t.Fatal(err)
				}
			} else if exit == 0 || stdout.Len() != 0 {
				t.Fatalf("expected refusal without report: exit=%d stdout=%s terminal=%s", exit, stdout.Bytes(), terminal)
			}
			if assessmentTree(t, setup.base) != expected {
				t.Fatal("unexpected source/journal/lock/output effects")
			}
		})
	}
}
