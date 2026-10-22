//go:build linux

package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/procrastivity/lds/internal/buildinfo"
	"github.com/procrastivity/lds/internal/cli"
	"github.com/procrastivity/lds/internal/iostreams"
	core "github.com/procrastivity/lds/internal/reconcile"
)

func TestGuidedJournalLifecycleAndExplicitProofClaims(t *testing.T) {
	setup := newJournalCLISetup(t)
	before := assessmentTree(t, setup.leftRoot) + assessmentTree(t, setup.rightRoot)
	stages := []struct {
		operation string
		answers   []string
	}{
		{"record", []string{
			"1", "fixture-reviewer", "Review an asymmetric correspondence", "corr:reviewed", "split", "left:topic", "right:a,right:b",
			"1", "Explicit citation, not semantic proof", "bounded", "reviewed-fixture", "bounded", "low", "Independent reviewer judgment",
		}},
		{"decide", []string{
			"1", "fixture-reviewer", "Wait for an explicit condition", "separate-owner", "declared", "right", "right-to-left",
			"deferred", "trigger:later", "condition", "A separate owner requests reconsideration",
		}},
		{"reconsider", []string{
			"1", "fixture-reviewer", "Record intent, not an executed transfer", "trigger:now", "event", "Owner request received", "no",
			"new-declarer", "declared", "left", "left-to-right", "intent", "left-to-right",
		}},
		{"reconsider", []string{
			"1", "fixture-reviewer", "Record an external proof claim only", "trigger:proof", "event", "External verifier supplied a report", "yes",
			"corr:replacement", "merged", "left:a,left:b", "right:topic", "", "replacement-reviewed", "untrusted", "not-assessed",
			"proof-owner", "declared", "right", "right-to-left", "applied", "right-to-left",
			"https://example.invalid/external-proof.json", strings.Repeat("a", 64), strings.Repeat("c", 40), strings.Repeat("d", 40),
			"right.md", "left.md", "external-verifier", "2026-10-20T12:13:14Z", "Disposable attributed proof; URI is not fetched",
		}},
	}
	for index, stage := range stages {
		response, preview, terminal, exit := driveGuidedJournal(t, setup, stage.operation, stage.answers, func(preview core.JournalPreview) string {
			return stage.operation + " " + preview.PlanDigest + "\n"
		})
		if exit != 0 {
			t.Fatalf("%s: exit=%d terminal=%s", stage.operation, exit, terminal)
		}
		if response.Revision != index+1 || len(response.Events) != index+1 || response.Replayed == nil || *response.Replayed {
			t.Fatalf("unexpected journal response: %+v", response)
		}
		var event struct {
			Request json.RawMessage `json:"request"`
		}
		if err := json.Unmarshal(response.Events[index], &event); err != nil || !jsonValuesEqualCLI(event.Request, preview.Request) {
			t.Fatalf("stored request differs from the approved request: %s / %s, %v", event.Request, preview.Request, err)
		}
		candidate := response.Candidates[0]
		switch index {
		case 0:
			if candidate.Correspondence.Relation != "split" || len(candidate.Correspondence.RightMembers) != 2 || len(candidate.Provenance) != 1 || candidate.Assessment.Confidence.Level != "low" || candidate.Decision.State != "pending-review" || string(candidate.Authority) != `{"status":"not-declared"}` {
				t.Fatalf("record inferred a decision or lost explicit correspondence: %+v", candidate)
			}
		case 1:
			if candidate.Decision.State != "deferred" || candidate.Decision.Trigger == nil || candidate.Decision.Trigger.ID != "trigger:later" {
				t.Fatalf("decision lost explicit deferral: %+v", candidate)
			}
		case 2:
			if candidate.Decision.State != "intent" || candidate.Decision.Direction != "left-to-right" || candidate.Decision.AppliedProof != nil || candidate.Correspondence.Key != "corr:reviewed" || candidate.LastDecision.Seq != 3 {
				t.Fatalf("reconsider promoted intent or lost prior correspondence: %+v", candidate)
			}
		case 3:
			proof := candidate.Decision.AppliedProof
			if proof == nil || proof.Source.ProjectID != "right" || proof.Destination.ProjectID != "left" || proof.Source.Revision != strings.Repeat("c", 40) || proof.Destination.Revision != strings.Repeat("d", 40) || proof.Verifier != "external-verifier" || candidate.Correspondence.Key != "corr:replacement" || candidate.Correspondence.Relation != "merged" || candidate.LastDecision.Seq != 4 {
				t.Fatalf("external proof or replacement differs from explicit answers: %+v", candidate)
			}
		}
	}
	if after := assessmentTree(t, setup.leftRoot) + assessmentTree(t, setup.rightRoot); after != before {
		t.Fatal("guided journal lifecycle modified a source project")
	}
}

func TestGuidedJournalFreshApprovalAndStaleRefusals(t *testing.T) {
	for _, name := range []string{"decline", "bare-digest", "wrong-operation", "eof", "source-changed", "pair-changed", "journal-created"} {
		t.Run(name, func(t *testing.T) {
			setup := newJournalCLISetup(t)
			answers := []string{"1", "fixture-reviewer", "Fixture review", "corr:explicit", "one-to-one", "left:item", "right:item", "", "reviewed", "bounded", "not-assessed"}
			expected := assessmentTree(t, setup.base)
			_, _, terminal, exit := driveGuidedJournal(t, setup, "record", answers, func(preview core.JournalPreview) string {
				decision := "record " + preview.PlanDigest + "\n"
				switch name {
				case "decline":
					decision = "decline\n"
				case "bare-digest":
					decision = preview.PlanDigest + "\n"
				case "wrong-operation":
					decision = "decide " + preview.PlanDigest + "\n"
				case "eof":
					decision = "\x04"
				case "source-changed":
					if err := os.WriteFile(filepath.Join(setup.leftRoot, "left.md"), []byte("Preserve this newer evidence\n"), 0o644); err != nil {
						t.Fatal(err)
					}
				case "pair-changed":
					data, err := os.ReadFile(setup.pairPath)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(setup.pairPath, append(data, '\n'), 0o600); err != nil {
						t.Fatal(err)
					}
				case "journal-created":
					if err := os.WriteFile(setup.journal, []byte("Preserve another writer's journal"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				expected = assessmentTree(t, setup.base)
				return decision
			})
			if exit == 0 || !strings.Contains(terminal, "refusal.") {
				t.Fatalf("expected an explicit refusal, exit=%d terminal=%s", exit, terminal)
			}
			if name == "decline" || name == "bare-digest" || name == "wrong-operation" || name == "eof" {
				if assessmentTree(t, setup.base) != expected {
					t.Fatal("unapproved operation created or changed a file, including a journal lock")
				}
			} else {
				// A refused apply may leave the established sibling lock, but
				// must never alter either source or another writer's journal.
				if err := os.Remove(setup.journal + ".lock"); err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				if assessmentTree(t, setup.base) != expected {
					t.Fatal("stale refusal changed current source, pair or journal bytes")
				}
			}
		})
	}
}

func TestGuidedJournalNoninteractiveRefusesBeforeReadingInputs(t *testing.T) {
	for _, operation := range []string{"record", "decide", "reconsider"} {
		r := run(t, nil, "reconcile", "journal", operation, "--pair", "/nonexistent/pair.json", "--left-root", "/nonexistent/left", "--right-root", "/nonexistent/right", "--map", "/nonexistent/map.md", "--journal", "/nonexistent/journal.json", "--json")
		assertJournalError(t, r, 3, "refusal.interactive-required")
	}
}

func driveGuidedJournal(t *testing.T, setup journalCLISetup, operation string, answers []string, confirmation func(core.JournalPreview) string) (core.JournalResponse, core.JournalPreview, string, int) {
	t.Helper()
	master, slave := assessmentPTY(t)
	var stdout bytes.Buffer
	streams := &iostreams.Streams{In: slave, Out: &stdout, Err: slave}
	command := cli.NewRootCommand(streams, buildinfo.Info{})
	command.SetArgs([]string{"reconcile", "journal", operation, "--pair", setup.pairPath, "--left-root", setup.leftRoot, "--right-root", setup.rightRoot, "--map", filepath.Join(setup.rightRoot, "maps", "a.md"), "--journal", setup.journal, "--json"})
	done := make(chan int, 1)
	go func() { done <- cli.Execute(command, streams) }()
	terminal := assessmentWait(t, master, "Candidate number")
	if _, err := unix.Write(int(master.Fd()), []byte(strings.Join(answers, "\n")+"\n")); err != nil {
		t.Fatal(err)
	}
	terminal += assessmentWait(t, master, "Confirmation: ")
	start := strings.Index(terminal, "Complete journal preview (review every field; still no writes):")
	if start < 0 {
		t.Fatal("no complete preview was displayed")
	}
	start += len("Complete journal preview (review every field; still no writes):")
	var preview core.JournalPreview
	if err := json.NewDecoder(strings.NewReader(terminal[start:])).Decode(&preview); err != nil {
		t.Fatalf("decode displayed preview: %v / %s", err, terminal[start:])
	}
	if !regexp.MustCompile(`^sha256:[a-f0-9]{64}$`).MatchString(preview.PlanDigest) {
		t.Fatalf("invalid approval digest %q", preview.PlanDigest)
	}
	if _, err := unix.Write(int(master.Fd()), []byte(confirmation(preview))); err != nil {
		t.Fatal(err)
	}
	var exit int
	select {
	case exit = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("guided journal did not terminate after fresh input")
	}
	terminal += assessmentDrain(t, master)
	var response core.JournalResponse
	if exit == 0 {
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("stdout must contain exactly the success JSON: %v / %s", err, stdout.Bytes())
		}
	} else if stdout.Len() != 0 {
		t.Fatalf("refused journal must leave stdout empty: %s", stdout.Bytes())
	}
	return response, preview, terminal, exit
}
