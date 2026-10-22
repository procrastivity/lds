package reconcile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func reviewFixture(t *testing.T) (Input, string) {
	t.Helper()
	base := t.TempDir()
	left, right := filepath.Join(base, "left"), filepath.Join(base, "right")
	leftRev := initGitFixture(t, left, map[string][]byte{"left.md": []byte("left\n")})
	rightRev := initGitFixture(t, right, map[string][]byte{"right.md": []byte("right\n"), "map.json": []byte("{}\n")})
	pairPath := filepath.Join(base, "pair.json")
	writePair(t, pairPath, "fixture:review", leftRev, rightRev,
		[]string{"left.md"}, []string{"right.md"},
		[]candidateInput{{"topic:review", []string{"left.md"}, []string{"right.md"}}})
	return Input{PairPath: pairPath, LeftRoot: left, RightRoot: right, MapPaths: []string{filepath.Join(right, "map.json")}}, filepath.Join(base, "journal.json")
}

func reviewRequest(t *testing.T, scan Response, operation string, revision int) []byte {
	t.Helper()
	candidate := scan.Candidates[0]
	request := map[string]any{
		"schemaVersion": 1, "requestId": "fixture-" + operation,
		"expectedJournalRevision": revision, "candidateId": candidate.CandidateID,
		"expectedEvidenceDigest": *candidate.EvidenceDigest, "actor": "fixture-reviewer",
		"at": "2026-10-01T00:00:01Z", "rationale": "Synthetic offline review only.",
	}
	switch operation {
	case "record":
		request["correspondence"] = Correspondence{Key: "fixture:relation", LeftMembers: []string{"L"}, RightMembers: []string{"R"}, Relation: "one-to-one"}
		request["provenance"] = []Provenance{{Kind: "synthetic", ProjectID: "synthetic", Path: "fixture:relation", Claim: "Synthetic test claim.", Trust: "synthetic"}}
		request["assessment"] = Assessment{Code: "fixture-review", Confidence: Confidence{Level: "not-assessed"}, Trust: "synthetic"}
	case "decide":
		request["authority"] = Authority{Owner: "fixture-owner", Status: "unresolved", PermittedDirections: []string{}}
		request["outcome"] = Outcome{State: "unresolved"}
	case "reconsider":
		request["supersedesSeq"] = 2
		request["trigger"] = Trigger{ID: "fixture:review", Kind: "condition", Description: "Fixture owner requests review."}
		request["authority"] = Authority{Owner: "fixture-owner", Status: "unresolved", PermittedDirections: []string{}}
		request["outcome"] = Outcome{State: "unresolved"}
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestJournalReviewLifecycleAndReadOnlyPreview(t *testing.T) {
	input, path := reviewFixture(t)
	input.Journal = filepath.Join(t.TempDir(), "nonexistent.json") // Cannot bypass the fresh scan.
	review, err := PrepareJournalReview(input, path)
	if err != nil {
		t.Fatal(err)
	}
	if review.Scan().JournalPath != "" || review.Journal().Revision != 0 {
		t.Fatal("review captured journal comparison instead of clean source scan")
	}
	scanCopy := review.Scan()
	scanCopy.Candidates[0].Evidence[0].Key = "tampered"
	journalCopy := review.Journal()
	journalCopy.Events = append(journalCopy.Events, json.RawMessage(`{}`))
	if review.Scan().Candidates[0].Evidence[0].Key == "tampered" || len(review.Journal().Events) != 0 {
		t.Fatal("read projections alias captured state")
	}
	request := append([]byte(" \n"), reviewRequest(t, review.Scan(), "record", 0)...)
	preview, err := review.Preview("record", request)
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := json.Marshal(preview)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip JournalPreview
	if err := json.Unmarshal(serialized, &roundTrip); err != nil || !reflect.DeepEqual(preview, roundTrip) {
		t.Fatalf("displayed preview did not round trip: %v", err)
	}
	if preview.JournalExists || preview.JournalSHA256 != "" || preview.PairSHA256 != review.Scan().PairSHA256 {
		t.Fatalf("unexpected baseline identity: %+v", preview)
	}
	if digest, err := journalPlanDigest(preview); err != nil || digest != preview.PlanDigest {
		t.Fatalf("digest mismatch: %s, %v", digest, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("preview created journal: %v", err)
	}
	if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("preview created lock: %v", err)
	}
	// Mutating returned slices or raw request cannot change the approved copy.
	bad := preview
	bad.Request = append(json.RawMessage(nil), preview.Request...)
	bad.Request[1] = 'X'
	assertCode(t, func() error { _, e := review.Apply(bad, preview.PlanDigest); return e }(), "refusal.reconcile-stale-evidence")
	assertCode(t, func() error { _, e := review.Apply(preview, "sha256:wrong"); return e }(), "refusal.reconcile-stale-evidence")
	forged := copyPreview(preview)
	forged.Scan.Candidates[0].CandidateKey = "topic:forged"
	forged.PlanDigest, _ = journalPlanDigest(forged)
	assertCode(t, func() error { _, e := review.Apply(forged, forged.PlanDigest); return e }(), "refusal.reconcile-stale-evidence")
	response, err := review.Apply(roundTrip, roundTrip.PlanDigest)
	if err != nil || response.Revision != 1 || len(response.Events) != 1 || response.Replayed == nil || *response.Replayed {
		t.Fatalf("record response: %+v, %v", response, err)
	}
	for seq, step := range []struct{ operation, eventType string }{{"decide", "decision.recorded"}, {"reconsider", "decision.reconsidered"}} {
		current, err := PrepareJournalReview(input, path)
		if err != nil {
			t.Fatal(err)
		}
		journalCopy := current.Journal()
		journalCopy.Events[0][0] = 'X'
		journalCopy.Candidates[0].Evidence[0].Key = "forged"
		if current.Journal().Events[0][0] == 'X' || current.Journal().Candidates[0].Evidence[0].Key == "forged" {
			t.Fatal("journal projection aliases captured event or evidence")
		}
		request := reviewRequest(t, current.Scan(), step.operation, seq+1)
		preview, err := current.Preview(step.operation, request)
		if err != nil {
			t.Fatal(err)
		}
		if !preview.JournalExists || preview.JournalSHA256 == "" || preview.Journal.Revision != seq+1 {
			t.Fatalf("incorrect existing baseline: %+v", preview)
		}
		response, err = current.Apply(preview, preview.PlanDigest)
		if err != nil || response.Revision != seq+2 {
			t.Fatalf("%s: %+v, %v", step.operation, response, err)
		}
		var event journalEvent
		if err := json.Unmarshal(response.Events[seq+1], &event); err != nil || event.Type != step.eventType || !jsonValuesEqual(event.Request, request) || event.Seq != seq+2 {
			t.Fatalf("%s event: %+v, %v", step.operation, event, err)
		}
	}
	var first journalEvent
	_ = json.Unmarshal(response.Events[0], &first)
	if first.Type != "candidate.recorded" || !jsonValuesEqual(first.Request, request) || response.Candidates[0].LastDecision.Seq != 3 {
		t.Fatalf("lifecycle event or final candidate incorrect: %+v", response)
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	state, err := parseJournal(disk, response.PairID, response.Snapshots.Left.ProjectID, response.Snapshots.Right.ProjectID)
	if err != nil || state.Revision != 3 || !reflect.DeepEqual(response.Events, state.Events) {
		t.Fatalf("persisted lifecycle mismatch: %v", err)
	}
}

func TestJournalReviewRefusesStaleApproval(t *testing.T) {
	for _, mutation := range []string{"source", "map", "pair", "missing-to-present", "same-revision-swap"} {
		t.Run(mutation, func(t *testing.T) {
			input, path := reviewFixture(t)
			if mutation == "same-revision-swap" {
				// A valid zero-event baseline, changed later without changing revision.
				pair, err := loadPair(input.PairPath)
				if err != nil {
					t.Fatal(err)
				}
				doc := journalDocument{SchemaVersion: 1, PairID: *pair.file.PairID, LeftProjectID: *pair.file.Left.ProjectID, RightProjectID: *pair.file.Right.ProjectID, Revision: 0, Events: []json.RawMessage{}}
				data, _ := json.Marshal(doc)
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			review, err := PrepareJournalReview(input, path)
			if err != nil {
				t.Fatal(err)
			}
			preview, err := review.Preview("record", reviewRequest(t, review.Scan(), "record", 0))
			if err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "source":
				err = os.WriteFile(filepath.Join(input.LeftRoot, "left.md"), []byte("dirty\n"), 0o644)
			case "map":
				err = os.WriteFile(input.MapPaths[0], []byte("dirty map\n"), 0o644)
			case "pair":
				original, readErr := os.ReadFile(input.PairPath)
				if readErr != nil {
					t.Fatal(readErr)
				}
				err = os.WriteFile(input.PairPath, append(original, '\n'), 0o644)
			case "missing-to-present":
				err = os.WriteFile(path, []byte("occupied"), 0o600)
			case "same-revision-swap":
				original, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				err = os.WriteFile(path, append(original, '\n'), 0o600)
			}
			if err != nil {
				t.Fatal(err)
			}
			assertCode(t, func() error { _, e := review.Apply(preview, preview.PlanDigest); return e }(), "refusal.reconcile-stale-evidence")
		})
	}
}

func TestJournalReviewProtectsSnapshotAndGitLockBoundaries(t *testing.T) {
	input, _ := reviewFixture(t)
	for _, target := range []string{
		filepath.Join(input.LeftRoot, "journal.json"),
		filepath.Join(input.RightRoot, ".git", "journal.json"),
	} {
		_, err := PrepareJournalReview(input, target)
		assertCode(t, err, "refusal.reconcile-write-target")
		if _, err := os.Stat(target + ".lock"); !os.IsNotExist(err) {
			t.Fatalf("protected journal lock created: %v", err)
		}
	}
}
