//go:build linux

package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

type assessmentTransport func(*http.Request) (*http.Response, error)

func (f assessmentTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAssessmentCommand_ConsentBoundDraftAndPreservation(t *testing.T) {
	for _, tc := range []struct {
		name, decision, mutation, reply, code string
		early                                 bool
		calls, exit                           int
	}{
		{name: "draft-report", decision: "send", calls: 1},
		{name: "decline-before-transfer", decision: "decline", code: "refusal.provider-transfer-declined", exit: 3},
		{name: "discard-pretyped-send", decision: "decline", early: true, code: "refusal.provider-transfer-declined", exit: 3},
		{name: "changed-source-before-transfer", decision: "send", mutation: "before", code: "refusal.assessment-stale", exit: 3},
		{name: "changed-document-after-transfer", decision: "send", mutation: "during", calls: 1, code: "refusal.assessment-stale", exit: 3},
		{name: "changed-flow-root-after-transfer", decision: "send", mutation: "flow", calls: 1, code: "refusal.flow-changed", exit: 3},
		{name: "invalid-draft", decision: "send", reply: "invalid", calls: 1, code: "validation.assessment-brief", exit: 1},
		{name: "model-cannot-expand-scope", decision: "send", reply: "scope", calls: 1, code: "refusal.assessment-scope-changed", exit: 3},
		{name: "model-cannot-cite-unread-source", decision: "send", reply: "citation", calls: 1, code: "validation.assessment-brief", exit: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENCODE_API_KEY", "test-only-not-a-credential")
			xdg := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdg)
			root, reports := t.TempDir(), t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "docs"), 0o750); err != nil {
				t.Fatal(err)
			}
			write := func(name, data string) {
				t.Helper()
				if err := os.WriteFile(name, []byte(data), 0o640); err != nil {
					t.Fatal(err)
				}
			}
			guide := filepath.Join(root, "docs", "usage.md")
			source := filepath.Join(root, "command.txt")
			write(guide, "# Usage\nexample --old\n[details](missing.md) [web](https://example.invalid)\n")
			write(source, "supported option: --new\n")
			write(filepath.Join(root, "unselected.txt"), "UNSELECTED PRIVATE MARKER\n")
			expected := assessmentTree(t, root)
			var expectedAfterReply string
			projectRoot, flowSource := "", "override"
			flow := filepath.Join(xdg, "lds", "flows", "assess.md")
			if tc.name == "draft-report" || tc.mutation == "flow" {
				projectRoot, flowSource = t.TempDir(), "project"
				if err := os.MkdirAll(filepath.Dir(flow), 0o700); err != nil {
					t.Fatal(err)
				}
				write(flow, "# Shadowed user flow\n")
				flow = filepath.Join(projectRoot, ".lds/assets/flows/assess.md")
				if tc.mutation == "flow" {
					selected := filepath.Join(t.TempDir(), "selected")
					if err := os.Symlink(projectRoot, selected); err != nil {
						t.Fatal(err)
					}
					projectRoot = selected
				}
			}
			if err := os.MkdirAll(filepath.Dir(flow), 0o700); err != nil {
				t.Fatal(err)
			}
			write(flow, "# Live assessment override\nCollect evidence and draft unverified advice.\n")
			output := filepath.Join(reports, "report.md")
			var calls atomic.Int32
			previous := http.DefaultTransport
			http.DefaultTransport = assessmentTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				var request struct {
					Model     string                           `json:"model"`
					MaxTokens int                              `json:"max_tokens"`
					Stream    bool                             `json:"stream"`
					Messages  []struct{ Role, Content string } `json:"messages"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					return nil, err
				}
				if r.Method != "POST" || r.URL.String() != "https://opencode.ai/zen/go/v1/chat/completions" || request.Model != "deepseek-v4-flash" || request.MaxTokens != 4096 || request.Stream || len(request.Messages) != 2 {
					t.Error("unexpected assessment request contract")
				}
				var payload struct {
					Flow     string                        `json:"flow"`
					Brief    adoption.AssessmentBrief      `json:"brief"`
					Evidence []adoption.AssessmentEvidence `json:"evidence"`
				}
				if err := json.Unmarshal([]byte(request.Messages[1].Content), &payload); err != nil {
					return nil, err
				}
				if !strings.Contains(payload.Flow, "Live assessment override") || len(payload.Evidence) != 2 || payload.Evidence[0].Content != "supported option: --new\n" || payload.Evidence[1].Path != "docs/usage.md" {
					t.Error("request lost the exact live flow or scoped source/document bytes")
				}
				for _, forbidden := range []string{root, reports, "UNSELECTED PRIVATE MARKER", "test-only-not-a-credential"} {
					if strings.Contains(request.Messages[1].Content, forbidden) {
						t.Errorf("request disclosed host metadata or unselected input %q", forbidden)
					}
				}
				payload.Brief.Proposals = []adoption.AssessmentProposal{{Path: "docs/usage.md", Summary: "Review the documented option", Evidence: []string{"command.txt", "docs/usage.md"}, ProposedText: "Use example --new."}}
				payload.Brief.Questions = []string{"Should the missing details link be removed?"}
				if tc.reply == "scope" {
					payload.Brief.Reason = "Model substituted a different task"
				}
				if tc.reply == "citation" {
					payload.Brief.Proposals[0].Evidence = []string{"unselected.txt"}
				}
				advice, err := json.Marshal(payload.Brief)
				if err != nil {
					return nil, err
				}
				if tc.reply == "invalid" {
					advice = []byte("Not the required JSON")
				}
				if assessmentTree(t, root) != expected {
					t.Error("assessment changed target before the provider response")
				}
				if tc.mutation == "during" {
					write(guide, "# Changed while drafting; preserve me.\n")
					expectedAfterReply = assessmentTree(t, root)
				}
				if tc.mutation == "flow" {
					// Retarget the selected root with byte-identical flow content:
					// checking just source="project" and bytes must not pass.
					replacement := t.TempDir()
					destination := filepath.Join(replacement, ".lds/assets/flows/assess.md")
					if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
						t.Fatal(err)
					}
					write(destination, "# Live assessment override\nCollect evidence and draft unverified advice.\n")
					if err := os.Remove(projectRoot); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(replacement, projectRoot); err != nil {
						t.Fatal(err)
					}
				}
				body, err := json.Marshal(map[string]any{"model": "deepseek-v4-flash", "choices": []any{map[string]any{"message": map[string]string{"content": string(advice)}}}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{}, Request: r}, err
			})
			t.Cleanup(func() { http.DefaultTransport = previous })
			master, slave := assessmentPTY(t)
			var stdout bytes.Buffer
			streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs([]string{"assess", root, "--reason", "Review option rename", "--audience", "CLI users", "--source", "command.txt", "--output", output, "--project-root", projectRoot, "--json"})
			if tc.early {
				if _, err := unix.Write(int(master.Fd()), []byte("send\n")); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan int, 1)
			go func() { done <- cli.Execute(command, streams) }()
			terminal := assessmentWait(t, master, "After reviewing this transfer")
			if !strings.Contains(terminal, `supported option: --new\n`) || !strings.Contains(terminal, "System instructions:") || !strings.Contains(terminal, "Live assessment override") {
				t.Fatal("consent prompt omitted complete selected payload or system instructions")
			}
			if assessmentTree(t, root) != expected {
				t.Error("assessment changed target before consent")
			}
			if tc.mutation == "before" {
				write(source, "Changed before transfer; preserve me.\n")
				expected = assessmentTree(t, root)
			}
			if _, err := unix.Write(int(master.Fd()), []byte(tc.decision+"\n")); err != nil {
				t.Fatal(err)
			}
			select {
			case exit := <-done:
				terminal += assessmentDrain(t, master)
				if exit != tc.exit || int(calls.Load()) != tc.calls {
					t.Fatalf("exit=%d calls=%d; want %d/%d; terminal=%s", exit, calls.Load(), tc.exit, tc.calls, terminal)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("assessment did not finish after consent")
			}
			if tc.code != "" {
				if stdout.Len() != 0 || !strings.Contains(terminal, `"code":"`+tc.code+`"`) {
					t.Fatalf("refusal lost its error/no-success contract: stdout=%s terminal=%s", stdout.String(), terminal)
				}
				if _, err := os.Stat(output); !os.IsNotExist(err) {
					t.Fatalf("refusal wrote report: %v", err)
				}
			} else {
				var result struct {
					Status, Provider, Model string
					FlowAsset               struct{ Source, SHA256 string }
					Assessment              adoption.AssessmentResult
				}
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				flowBytes, err := os.ReadFile(flow)
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(flowBytes)
				if result.Status != "review-required" || result.Provider != "OpenCode Go" || result.Model != "deepseek-v4-flash" || result.FlowAsset.Source != flowSource || result.FlowAsset.SHA256 != hex.EncodeToString(sum[:]) || result.Assessment.Brief.Proposals[0].ProposedText != "Use example --new." || result.Assessment.ValidationExitCode != 1 || len(result.Assessment.Validation.Skipped) != 1 {
					t.Fatalf("report lost advice, provenance, findings or skips: %+v", result)
				}
				report, err := os.ReadFile(output)
				if err != nil || !bytes.Contains(report, []byte("Use example --new.")) || !bytes.Contains(report, []byte("may be model-authored")) {
					t.Fatalf("missing or mislabeled advice in report: %v", err)
				}
				if info, err := os.Stat(output); err != nil || info.Mode().Perm() != 0o600 {
					t.Fatalf("report must be private: %v %v", info, err)
				}
			}
			if expectedAfterReply != "" {
				expected = expectedAfterReply
			}
			if after := assessmentTree(t, root); expected != after {
				t.Fatalf("assessment changed target: expected=%s after=%s", expected, after)
			}
		})
	}
}

func TestAssessmentCommand_PreflightNeverTransfers(t *testing.T) {
	for _, tc := range []struct{ name, code string }{
		{"noninteractive", "refusal.interactive-required"},
		{"inside-target", "refusal.assessment-output"},
		{"occupied-report", "refusal.assessment-output"},
		{"oversized-transfer", "refusal.assessment-transfer-too-large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENCODE_API_KEY", "test-only-not-a-credential")
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			root, reports := t.TempDir(), t.TempDir()
			output := filepath.Join(reports, "report.md")
			content := "# Review\n"
			if tc.name == "oversized-transfer" {
				content += strings.Repeat("x", 512*1024)
			}
			if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(content), 0o640); err != nil {
				t.Fatal(err)
			}
			if tc.name == "inside-target" {
				output = filepath.Join(root, "report.md")
			}
			if tc.name == "occupied-report" {
				if err := os.WriteFile(output, []byte("preserve prior report"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			before, reportsBefore := assessmentTree(t, root), assessmentTree(t, reports)
			previous := http.DefaultTransport
			calls := 0
			http.DefaultTransport = assessmentTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return nil, fmt.Errorf("preflight must not reach transport")
			})
			t.Cleanup(func() { http.DefaultTransport = previous })
			var stdout, stderr bytes.Buffer
			streams := &iostreams.Streams{In: strings.NewReader("send\n"), Out: &stdout, Err: &stderr}
			var master *os.File
			if tc.name != "noninteractive" {
				var slave *os.File
				master, slave = assessmentPTY(t)
				streams.In, streams.Err = slave, slave
			}
			command := cli.NewRootCommand(streams, buildinfo.Info{})
			command.SetArgs([]string{"assess", root, "--reason", "Review", "--audience", "Maintainers", "--output", output, "--json"})
			exit := cli.Execute(command, streams)
			terminal := stderr.String()
			if master != nil {
				terminal = assessmentDrain(t, master)
			}
			if exit != 3 || calls != 0 || stdout.Len() != 0 || !strings.Contains(terminal, `"code":"`+tc.code+`"`) {
				t.Fatalf("preflight: exit=%d calls=%d stdout=%s stderr=%s", exit, calls, stdout.String(), terminal)
			}
			if assessmentTree(t, root) != before || assessmentTree(t, reports) != reportsBefore {
				t.Fatal("preflight changed target or external report tree")
			}
		})
	}
}

func assessmentTree(t *testing.T, root string) string {
	t.Helper()
	state := map[string]string{}
	err := filepath.WalkDir(root, func(name string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(name)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			value += fmt.Sprintf(":%x", sha256.Sum256(data))
		}
		state[rel] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assessmentPTY(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	fd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	master := os.NewFile(uintptr(fd), "assessment-pty-master")
	t.Cleanup(func() { _ = master.Close() })
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	number, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = slave.Close() })
	if err := unix.SetNonblock(fd, true); err != nil {
		t.Fatal(err)
	}
	return master, slave
}

func assessmentDrain(t *testing.T, master *os.File) string {
	t.Helper()
	var output strings.Builder
	buffer := make([]byte, 8192)
	for {
		n, err := unix.Read(int(master.Fd()), buffer)
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK || n == 0 {
			return output.String()
		}
		if err != nil {
			t.Fatal(err)
		}
		output.Write(buffer[:n])
	}
}

func assessmentWait(t *testing.T, master *os.File, expected string) string {
	t.Helper()
	var output string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		output += assessmentDrain(t, master)
		if strings.Contains(output, expected) {
			return output
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q; terminal=%s", expected, output)
	return ""
}
