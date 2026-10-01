package reconcile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/procrastivity/lds/internal/ldserr"
)

const (
	journalPairID        = "fixture:lds-bds252:synthetic-split:v1"
	journalKey           = "fixture:route-decomposition"
	journalLeftID        = "fixture-left"
	journalRightID       = "fixture-right"
	journalLeftRev       = "1111111111111111111111111111111111111111"
	journalRightRev      = "2222222222222222222222222222222222222222"
	journalNextRightRev  = "3333333333333333333333333333333333333333"
	journalCandidateID   = "cand-v1-aed9fd6fc71e2ccff638d2e91a338b80ba6761a929d4b44a67dd26d242d81af6"
	journalInitialDigest = "sha256:2d0a164bdc0f44c1fef7f872600c66b0e8045c955b21dac04b693db282522f65"
	journalChangedDigest = "sha256:355c5ec79246cf43663edf541b695a8f89e70b93ad60c1dc8cefe6795120506b"
)

func TestJournalReplayLifecycleAndComparisonStatuses(t *testing.T) {
	initial := syntheticJournalObservation(journalRightRev, "a9fcf28638ddc43b9122fb54f764bd6feab099b694e6eef24d19bb8c33fbb462")
	if initial.EvidenceDigest != journalInitialDigest || candidateID(journalPairID, journalKey) != journalCandidateID {
		t.Fatalf("independent initial fixture vectors changed: id=%s digest=%s", candidateID(journalPairID, journalKey), initial.EvidenceDigest)
	}
	changed := syntheticJournalObservation(journalNextRightRev, "4444444444444444444444444444444444444444444444444444444444444444")
	if changed.EvidenceDigest != journalChangedDigest {
		t.Fatalf("independent changed fixture vector = %s, want %s", changed.EvidenceDigest, journalChangedDigest)
	}

	record := makeRecordEvent(1, "candidate.recorded", "fixture-record-1", initial)
	decision := makeDecisionEvent(2, "fixture-decide-2", initial, Authority{
		Owner:                  "fixture-owner",
		Status:                 "declared",
		AuthoritativeProjectID: stringPointer(journalRightID),
		PermittedDirections:    []string{"right-to-left"},
	}, Outcome{State: "deferred", Trigger: &Trigger{
		ID: "fixture:revisit", Kind: "condition", Description: "A fixture owner requests a future review.",
	}})
	observed := makeRecordEvent(3, "evidence.observed", "fixture-observe-3", changed)
	stateAfterObserve := parseTestJournal(t, record, decision, observed)
	candidate := stateAfterObserve.Candidates[journalCandidateID]
	if candidate.EvidenceDigest != journalChangedDigest || candidate.Decision.State != "deferred" || candidate.LastDecision == nil || candidate.LastDecision.Seq != 2 || candidate.LastDecision.EvidenceDigest != journalInitialDigest {
		t.Fatalf("observation did not preserve the prior decision digest and outcome: %+v", candidate)
	}

	current := testCandidate(initial, journalCandidateID)
	compared := compareJournal([]Candidate{testCandidate(changed, journalCandidateID)}, stateAfterObserve)
	if len(compared) != 1 || compared[0].EvidenceStatus != "unchanged" || string(compared[0].JournalEvidenceDigest) != `"`+journalChangedDigest+`"` {
		t.Fatalf("current observed digest comparison = %+v", compared)
	}
	var savedDecision LastDecision
	if err := json.Unmarshal(compared[0].LastDecision, &savedDecision); err != nil {
		t.Fatal(err)
	}
	if savedDecision.Seq != 2 || savedDecision.EvidenceDigest != journalInitialDigest || savedDecision.Outcome.State != "deferred" {
		t.Fatalf("comparison falsely advanced the decision: %+v", savedDecision)
	}
	changedComparison := compareJournal([]Candidate{current}, stateAfterObserve)
	if changedComparison[0].EvidenceStatus != "changed" {
		t.Fatalf("stale current evidence status=%q, want changed", changedComparison[0].EvidenceStatus)
	}
	newObservation := observationWithKey(initial, "new:key")
	newComparison := compareJournal([]Candidate{testCandidate(newObservation, candidateID(journalPairID, "new:key"))}, stateAfterObserve)
	if len(newComparison) != 2 || newComparison[0].CandidateID >= newComparison[1].CandidateID {
		t.Fatalf("journal/current union was not ID-sorted or status-classified: %+v", newComparison)
	}
	var journalOnly, newlyScoped Candidate
	for _, item := range newComparison {
		if item.CandidateID == journalCandidateID {
			journalOnly = item
		} else {
			newlyScoped = item
		}
	}
	if journalOnly.EvidenceStatus != "out-of-scope" || newlyScoped.EvidenceStatus != "new" {
		t.Fatalf("journal/current statuses = %q/%q, want out-of-scope/new", journalOnly.EvidenceStatus, newlyScoped.EvidenceStatus)
	}
	if journalOnly.EvidenceDigest != nil || journalOnly.Evidence == nil || len(journalOnly.Evidence) != 0 || journalOnly.Left == nil || len(journalOnly.Left) != 0 || journalOnly.Right == nil || len(journalOnly.Right) != 0 || string(journalOnly.JournalEvidenceDigest) != `"`+journalChangedDigest+`"` {
		t.Fatalf("out-of-scope projection fabricated current evidence: %+v", journalOnly)
	}
	var outOfScopeDecision LastDecision
	if err := json.Unmarshal(journalOnly.LastDecision, &outOfScopeDecision); err != nil {
		t.Fatal(err)
	}
	if outOfScopeDecision.Seq != 2 || outOfScopeDecision.EvidenceDigest != journalInitialDigest || outOfScopeDecision.Outcome.State != "deferred" {
		t.Fatalf("out-of-scope projection failed to preserve the decision's older evidence basis: %+v", outOfScopeDecision)
	}

	newKeyObservation := observationWithKey(initial, "brand:new")
	newOnly := compareJournal([]Candidate{testCandidate(newKeyObservation, candidateID(journalPairID, "brand:new"))}, &journalState{Revision: 3, Candidates: map[string]*journalCandidate{}})
	if len(newOnly) != 1 || newOnly[0].EvidenceStatus != "new" || string(newOnly[0].JournalEvidenceDigest) != "null" || string(newOnly[0].LastDecision) != "null" || newOnly[0].JournalRevision == nil || *newOnly[0].JournalRevision != 3 {
		t.Fatalf("new-candidate projection = %+v", newOnly)
	}

	reconsider := makeReconsiderEvent(4, "fixture-reconsider-4", changed, 2)
	stateAfterReconsider := parseTestJournal(t, record, decision, observed, reconsider)
	final := stateAfterReconsider.Candidates[journalCandidateID]
	if final.Decision.State != "unresolved" || final.LastDecision == nil || final.LastDecision.Seq != 4 || final.LastDecision.EvidenceDigest != journalChangedDigest || final.LastEventSeq != 4 {
		t.Fatalf("reconsideration did not explicitly supersede the decision: %+v", final)
	}
	if _, err := parseJournal(testJournalBytes(record, decision, makeReconsiderEvent(3, "wrong-sequence", changed, 1)), journalPairID, journalLeftID, journalRightID); err == nil {
		t.Fatal("reconsideration with a non-latest supersedesSeq was accepted")
	}
}

func TestJournalRejectsMalformedHistoryAndAppliedProofDirection(t *testing.T) {
	observation := syntheticJournalObservation(journalRightRev, "a9fcf28638ddc43b9122fb54f764bd6feab099b694e6eef24d19bb8c33fbb462")
	record := makeRecordEvent(1, "candidate.recorded", "fixture-record-1", observation)
	decision := makeDecisionEvent(2, "fixture-decide-2", observation, Authority{
		Owner:                  "fixture-owner",
		Status:                 "declared",
		AuthoritativeProjectID: stringPointer(journalRightID),
		PermittedDirections:    []string{"left-to-right"},
	}, Outcome{State: "applied", Direction: "left-to-right", AppliedProof: &AppliedProof{
		URI: "https://example.invalid/fixture-transfer.json", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Source:      ProofProject{ProjectID: journalRightID, Revision: journalRightRev},
		Destination: ProofProject{ProjectID: journalLeftID, Revision: journalLeftRev},
		Scope:       ProofScope{SourcePaths: []string{"source.md"}, DestinationPaths: []string{"destination.md"}},
		Verifier:    "fixture-verifier", VerifiedAt: "2026-10-01T00:00:02Z", Rationale: "Synthetic malformed direction fixture.",
	}})
	if _, err := parseJournal(testJournalBytes(record, decision), journalPairID, journalLeftID, journalRightID); err == nil {
		t.Fatal("persisted applied proof with reversed source/destination was accepted")
	} else {
		assertCode(t, err, "validation.reconcile-evidence-invalid")
	}

	badCases := []struct {
		name   string
		events []json.RawMessage
	}{
		{"unknown event type", []json.RawMessage{mutateEvent(t, record, func(event map[string]any) { event["type"] = "future.event" })}},
		{"unknown event field", []json.RawMessage{mutateEvent(t, record, func(event map[string]any) { event["unexpected"] = true })}},
		{"duplicate request id", []json.RawMessage{record, makeRecordEvent(2, "evidence.observed", "fixture-record-1", observation)}},
		{"tampered evidence digest", []json.RawMessage{mutateEvent(t, record, func(event map[string]any) {
			identity := event["scanIdentity"].(map[string]any)
			identity["evidenceDigest"] = "sha256:" + stringsRepeat("f", 64)
		})}},
		{"unknown payload field", []json.RawMessage{mutateEvent(t, record, func(event map[string]any) {
			event["payload"].(map[string]any)["guess"] = true
		})}},
	}
	for _, test := range badCases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseJournal(testJournalBytes(test.events...), journalPairID, journalLeftID, journalRightID); err == nil {
				t.Fatal("malformed journal history was accepted")
			}
		})
	}
}

func TestJournalRejectsNullRevisionFields(t *testing.T) {
	empty := []byte(`{"schemaVersion":1,"pairId":"fixture:lds-bds252:synthetic-split:v1","leftProjectId":"fixture-left","rightProjectId":"fixture-right","revision":0,"events":[]}`)
	if _, err := parseJournal(empty, journalPairID, journalLeftID, journalRightID); err != nil {
		t.Fatalf("valid zero-revision journal: %v", err)
	}
	observation := syntheticJournalObservation(journalRightRev, "a9fcf28638ddc43b9122fb54f764bd6feab099b694e6eef24d19bb8c33fbb462")
	record := makeRecordEvent(1, "candidate.recorded", "fixture-record-1", observation)
	nullRequestRevision := mutateEvent(t, record, func(event map[string]any) {
		event["request"].(map[string]any)["expectedJournalRevision"] = nil
	})
	for _, data := range [][]byte{
		[]byte(strings.Replace(string(empty), `"revision":0`, `"revision":null`, 1)),
		testJournalBytes(nullRequestRevision),
	} {
		_, err := parseJournal(data, journalPairID, journalLeftID, journalRightID)
		assertCode(t, err, "validation.reconcile-evidence-invalid")
	}
}

func TestScanJournalComparisonIsReadOnlyAndProjectsOutOfScope(t *testing.T) {
	base := t.TempDir()
	leftRoot := filepath.Join(base, "left")
	rightRoot := filepath.Join(base, "right")
	leftRevision := initGitFixture(t, leftRoot, map[string][]byte{"docs/left.md": []byte("left fixture\n")})
	rightRevision := initGitFixture(t, rightRoot, map[string][]byte{
		"docs/right.md":  []byte("right fixture\n"),
		"maps/review.md": []byte("fixture review map\n"),
	})
	pairPath := filepath.Join(base, "pair.json")
	writePair(t, pairPath, "fixture:journal-scan:v1", leftRevision, rightRevision,
		[]string{"docs/left.md"}, []string{"docs/right.md"},
		[]candidateInput{{"topic:fixture", []string{"docs/left.md"}, []string{"docs/right.md"}}},
	)
	journalPath := filepath.Join(base, "journal.json")
	input := Input{
		PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot,
		MapPaths: []string{filepath.Join(rightRoot, "maps", "review.md")},
	}
	initial, err := Scan(input)
	if err != nil {
		t.Fatalf("initial scan: %v", err)
	}
	if len(initial.Candidates) != 1 || initial.Candidates[0].EvidenceStatus != "not-compared" {
		t.Fatalf("initial scan projection = %+v", initial.Candidates)
	}

	identityBytes, err := json.Marshal(initial)
	if err != nil {
		t.Fatal(err)
	}
	scanBytes := append(identityBytes, '\n')
	journalBytes := scanRecordJournal(t, initial, initial.Candidates[0], scanBytes)
	if err := os.WriteFile(journalPath, journalBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	leftBefore := snapshotFixtureFiles(t, leftRoot)
	rightBefore := snapshotFixtureFiles(t, rightRoot)
	journalBefore, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}

	input.Journal = journalPath
	unchanged, err := Scan(input)
	if err != nil {
		t.Fatalf("scan with recorded journal: %v", err)
	}
	if len(unchanged.Candidates) != 1 || unchanged.Candidates[0].EvidenceStatus != "unchanged" || digestValue(unchanged.Candidates[0].EvidenceDigest) != digestValue(initial.Candidates[0].EvidenceDigest) {
		t.Fatalf("same observed digest was not classified unchanged: %+v", unchanged.Candidates)
	}
	if unchanged.JournalPath != journalPath || unchanged.Candidates[0].JournalRevision == nil || *unchanged.Candidates[0].JournalRevision != 1 || string(unchanged.Candidates[0].LastDecision) != "null" {
		t.Fatalf("record-only journal projection = %+v", unchanged.Candidates[0])
	}

	writePair(t, pairPath, "fixture:journal-scan:v1", leftRevision, rightRevision,
		[]string{}, []string{}, []candidateInput{},
	)
	outOfScope, err := Scan(input)
	if err != nil {
		t.Fatalf("scan after scope removal: %v", err)
	}
	if len(outOfScope.Candidates) != 1 || outOfScope.Candidates[0].CandidateID != unchanged.Candidates[0].CandidateID || outOfScope.Candidates[0].EvidenceStatus != "out-of-scope" || outOfScope.Candidates[0].EvidenceDigest != nil || outOfScope.Candidates[0].Evidence == nil || len(outOfScope.Candidates[0].Evidence) != 0 || string(outOfScope.Candidates[0].JournalEvidenceDigest) != `"`+digestValue(initial.Candidates[0].EvidenceDigest)+`"` {
		t.Fatalf("journal-only candidate fabricated current evidence: %+v", outOfScope.Candidates)
	}
	leftAfter := snapshotFixtureFiles(t, leftRoot)
	rightAfter := snapshotFixtureFiles(t, rightRoot)
	journalAfter, err := os.ReadFile(journalPath)
	if err != nil || !equalByteMaps(leftBefore, leftAfter) || !equalByteMaps(rightBefore, rightAfter) || string(journalBefore) != string(journalAfter) {
		t.Fatalf("journal comparison modified input bytes: err=%v leftEqual=%t rightEqual=%t journalEqual=%t", err, equalByteMaps(leftBefore, leftAfter), equalByteMaps(rightBefore, rightAfter), string(journalBefore) == string(journalAfter))
	}

	missingJournal := filepath.Join(base, "missing-journal.json")
	input.Journal = missingJournal
	_, err = Scan(input)
	assertCode(t, err, "validation.reconcile-evidence-invalid")
	if _, err := os.Stat(missingJournal); !os.IsNotExist(err) {
		t.Fatalf("scan initialized a missing journal: stat error=%v", err)
	}
	input.Journal = filepath.Join(leftRoot, "docs", "left.md")
	_, err = Scan(input)
	assertCode(t, err, "refusal.reconcile-write-target")
}

func TestJournalRechecksCanonicalParentBeforeReplacement(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	parent := filepath.Join(base, "journal")
	for _, dir := range []string{source, parent} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(parent, "journal.json")
	lockPath := target + ".lock"
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	dir := openTestJournalDirectory(t, parent)
	defer func() {
		if err := lock.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := checkJournalPathStillSafe(target, lockPath, dir, lock, []string{source}); err != nil {
		t.Fatalf("unchanged canonical journal parent refused: %v", err)
	}
	moved := filepath.Join(source, "journal")
	if err := os.Rename(parent, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, parent); err != nil {
		t.Fatal(err)
	}
	// The open lock and its path still name the same inode. Rechecking only
	// that inode and the target's lexical prefix misses this parent alias.
	assertCode(t, checkJournalPathStillSafe(target, lockPath, dir, lock, []string{source}), "refusal.reconcile-write-target")
}

func TestJournalPersistenceAnchorsIOAcrossParentPivots(t *testing.T) {
	for _, phase := range []string{"create", "rename", "cleanup", "directory-sync"} {
		t.Run(phase, func(t *testing.T) {
			base := t.TempDir()
			parent, source, moved := filepath.Join(base, "journal"), filepath.Join(base, "source"), filepath.Join(base, "moved-journal")
			for _, path := range []string{parent, source} {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			const name = "journal.json"
			old, next, protected := []byte("old caller journal\n"), []byte("new complete caller journal\n"), []byte("protected source journal\n")
			for path, data := range map[string][]byte{filepath.Join(parent, name): old, filepath.Join(source, name): protected} {
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			dir := openTestJournalDirectory(t, parent)
			shadow := ""
			pivot := func() error {
				if phase == "cleanup" {
					entries, err := os.ReadDir(parent)
					if err != nil {
						return err
					}
					for _, entry := range entries {
						if strings.HasPrefix(entry.Name(), ".lds-journal-") {
							shadow = entry.Name()
						}
					}
					if shadow == "" {
						return errors.New("test expected a sibling temp before rename")
					}
					if err := os.WriteFile(filepath.Join(source, shadow), protected, 0o600); err != nil {
						return err
					}
				}
				if err := os.Rename(parent, moved); err != nil {
					return err
				}
				if err := os.Symlink(source, parent); err != nil {
					return err
				}
				if phase == "cleanup" {
					return errors.New("injected pre-rename failure after pivot")
				}
				return nil
			}
			hooks := journalWriteHooks{}
			switch phase {
			case "create":
				hooks.beforeCreateTemp = pivot
			case "rename", "cleanup":
				hooks.beforeRename = pivot
			case "directory-sync":
				hooks.beforeDirectorySync = pivot
			}
			err := persistJournal(dir, name, next, hooks)
			want := next
			if phase == "cleanup" {
				assertCode(t, err, "internal.reconcile-journal-write")
				want = old
			} else if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(moved, name))
			if err != nil || string(got) != string(want) {
				t.Fatalf("anchored journal bytes = %q, want %q: %v", got, want, err)
			}
			entries, err := os.ReadDir(moved)
			if err != nil || len(entries) != 1 || entries[0].Name() != name {
				t.Fatalf("temp leaked outside anchored directory: %v %v", entries, err)
			}
			got, err = os.ReadFile(filepath.Join(source, name))
			if err != nil || string(got) != string(protected) {
				t.Fatalf("parent pivot changed source bytes: %q %v", got, err)
			}
			if shadow != "" {
				got, err = os.ReadFile(filepath.Join(source, shadow))
				if err != nil || string(got) != string(protected) {
					t.Fatalf("temp cleanup followed pivot into source: %q %v", got, err)
				}
			}
			sourceEntries, err := os.ReadDir(source)
			wantCount := 1
			if shadow != "" {
				wantCount++
			}
			if err != nil || len(sourceEntries) != wantCount {
				t.Fatalf("temp creation followed pivot into source: %v %v", sourceEntries, err)
			}
		})
	}
}

func openTestJournalDirectory(t *testing.T, path string) *os.File {
	t.Helper()
	dir, err := openJournalDirectory(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := dir.Close(); err != nil {
			t.Error(err)
		}
	})
	return dir
}

func TestJournalAtomicReplacementFailureAndIndeterminateRecovery(t *testing.T) {
	base := t.TempDir()
	journalPath := filepath.Join(base, "journal.json")
	oldEvent := makeRecordEvent(1, "candidate.recorded", "atomic-record-1", syntheticJournalObservation(journalRightRev, "a9fcf28638ddc43b9122fb54f764bd6feab099b694e6eef24d19bb8c33fbb462"))
	oldJournal := testJournalBytes(oldEvent)
	decision := makeDecisionEvent(2, "atomic-decide-2", syntheticJournalObservation(journalRightRev, "a9fcf28638ddc43b9122fb54f764bd6feab099b694e6eef24d19bb8c33fbb462"), Authority{
		Owner: "fixture-owner", Status: "declared", AuthoritativeProjectID: stringPointer(journalRightID), PermittedDirections: []string{"right-to-left"},
	}, Outcome{State: "unresolved"})
	newJournal := testJournalBytes(oldEvent, decision)
	if err := os.WriteFile(journalPath, oldJournal, 0o600); err != nil {
		t.Fatal(err)
	}
	dir := openTestJournalDirectory(t, base)

	preRenameErr := persistJournal(dir, filepath.Base(journalPath), newJournal, journalWriteHooks{
		beforeRename: func() error { return errors.New("injected rename failure") },
	})
	assertCode(t, preRenameErr, "internal.reconcile-journal-write")
	var rendered *ldserr.Error
	if !errors.As(preRenameErr, &rendered) || !strings.Contains(rendered.Message, "old journal bytes were preserved") {
		t.Fatalf("pre-rename error did not explain unchanged old version: %v", preRenameErr)
	}
	current, err := os.ReadFile(journalPath)
	if err != nil || string(current) != string(oldJournal) {
		t.Fatalf("pre-rename failure changed old journal bytes: err=%v", err)
	}

	postRenameErr := persistJournal(dir, filepath.Base(journalPath), newJournal, journalWriteHooks{
		beforeDirectorySync: func() error { return errors.New("injected directory sync failure") },
	})
	assertCode(t, postRenameErr, "internal.reconcile-journal-write")
	if !errors.As(postRenameErr, &rendered) || !strings.Contains(rendered.Message, "commit outcome is indeterminate") || !strings.Contains(rendered.Message, "requestId") {
		t.Fatalf("post-rename error did not direct revision/requestId recovery: %v", postRenameErr)
	}
	current, err = os.ReadFile(journalPath)
	if err != nil || string(current) != string(newJournal) {
		t.Fatalf("post-rename directory-sync failure did not leave the complete new journal visible: err=%v", err)
	}

	pairPath := filepath.Join(base, "pair.json")
	pairBytes, err := json.Marshal(map[string]any{
		"schemaVersion": 1, "pairId": journalPairID,
		"left":  map[string]any{"projectId": journalLeftID, "revision": journalLeftRev},
		"right": map[string]any{"projectId": journalRightID, "revision": journalRightRev},
		"scope": map[string]any{"leftPaths": []string{}, "rightPaths": []string{}, "candidates": []any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pairPath, pairBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	read, err := ReadJournal(JournalInput{PairPath: pairPath, JournalPath: journalPath})
	if err != nil || read.Revision != 2 || len(read.Events) != 2 {
		t.Fatalf("indeterminate commit cannot be recovered by reading revision/requestId: revision=%d events=%d err=%v", read.Revision, len(read.Events), err)
	}
	var lastEvent journalEvent
	if err := json.Unmarshal(read.Events[1], &lastEvent); err != nil || lastEvent.RequestID != "atomic-decide-2" || lastEvent.Seq != 2 {
		t.Fatalf("recovery read did not expose committed request identity: event=%+v err=%v", lastEvent, err)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".lds-journal-") {
			t.Errorf("atomic replacement leaked temp file %q", entry.Name())
		}
	}
}

func scanRecordJournal(t *testing.T, scan Response, candidate Candidate, scanBytes []byte) []byte {
	t.Helper()
	if candidate.EvidenceDigest == nil {
		t.Fatal("scan candidate has no evidence digest")
	}
	common := map[string]any{
		"schemaVersion": 1, "requestId": "fixture-scan-record", "expectedJournalRevision": 0,
		"candidateId": candidate.CandidateID, "expectedEvidenceDigest": *candidate.EvidenceDigest,
		"actor": "fixture-reviewer", "at": "2026-10-01T00:00:01Z",
		"rationale": "Fixture only: exercise read-only journal comparison.",
	}
	correspondence := Correspondence{Key: "fixture:reviewed-relation", LeftMembers: []string{"SYNTH-L-01"}, RightMembers: []string{"SYNTH-R-01"}, Relation: "one-to-one"}
	provenance := []Provenance{{Kind: "synthetic", ProjectID: "synthetic", Path: "fixture:reviewed-relation", Claim: "Synthetic relation used only for the disposable reader fixture.", Trust: "synthetic"}}
	assessment := Assessment{Code: "fixture-review", Confidence: Confidence{Level: "not-assessed"}, Trust: "synthetic"}
	request := cloneObject(common)
	request["correspondence"], request["provenance"], request["assessment"] = correspondence, provenance, assessment
	observation := observationPayload{
		CandidateKey:   candidate.CandidateKey,
		Snapshots:      Snapshots{Left: scan.Snapshots.Left, Right: scan.Snapshots.Right},
		EvidenceDigest: *candidate.EvidenceDigest,
		Evidence:       candidate.Evidence,
	}
	payload := observationObject(observation)
	payload["correspondence"], payload["provenance"], payload["assessment"] = correspondence, provenance, assessment
	requestBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	identity := scanIdentity{
		ScanSHA256: digest(scanBytes), PairSHA256: scan.PairSHA256,
		CandidateID: candidate.CandidateID, EvidenceDigest: *candidate.EvidenceDigest,
	}
	event := map[string]any{
		"seq": 1, "requestId": common["requestId"], "candidateId": candidate.CandidateID,
		"type": "candidate.recorded", "actor": common["actor"], "at": common["at"], "rationale": common["rationale"],
		"request": json.RawMessage(requestBytes), "scanIdentity": identity, "payload": payload,
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return testJournalBytesFor(scan.PairID, scan.Snapshots.Left.ProjectID, scan.Snapshots.Right.ProjectID, eventBytes)
}

func syntheticJournalObservation(rightRevision, rightArtifactSHA string) observationPayload {
	left := Snapshot{ProjectID: journalLeftID, Revision: journalLeftRev}
	right := Snapshot{ProjectID: journalRightID, Revision: rightRevision}
	evidence := []Evidence{
		{Kind: "artifact", ProjectID: journalLeftID, Revision: journalLeftRev, Key: "engineering/synthetic/route-model.md", SHA256: "abdcbb890f6a49cc5fa2b97e9f97d68855af593e86d0e4bf4049e80547d8302d"},
		{Kind: "artifact", ProjectID: journalRightID, Revision: rightRevision, Key: "engineering/synthetic/destination-verification.md", SHA256: "983d27fa13e6f90a134c5b31fbea6a8f5b8746cf769efdb483fb3a0e53daa8f8"},
		{Kind: "artifact", ProjectID: journalRightID, Revision: rightRevision, Key: "engineering/synthetic/review-map.txt", SHA256: "925847331172c5e5fcd20116090f5f7219162128bb80f5dfaaf75c01000bbf67"},
		{Kind: "artifact", ProjectID: journalRightID, Revision: rightRevision, Key: "engineering/synthetic/route-declaration.md", SHA256: rightArtifactSHA},
	}
	sort.Slice(evidence, func(i, j int) bool { return evidenceLess(evidence[i], evidence[j]) })
	leftPin := &projectPin{projectID: journalLeftID, revision: journalLeftRev}
	rightPin := &projectPin{projectID: journalRightID, revision: rightRevision}
	return observationPayload{
		CandidateKey:   journalKey,
		Snapshots:      Snapshots{Left: left, Right: right},
		EvidenceDigest: evidenceDigest(journalPairID, leftPin, rightPin, journalKey, evidence),
		Evidence:       evidence,
	}
}

func makeRecordEvent(seq int, eventType, requestID string, observation observationPayload) json.RawMessage {
	common := map[string]any{
		"schemaVersion": 1, "requestId": requestID, "expectedJournalRevision": seq - 1,
		"candidateId": journalCandidateID, "expectedEvidenceDigest": observation.EvidenceDigest,
		"actor": "fixture-reviewer", "at": fmt.Sprintf("2026-10-01T00:00:0%dZ", seq),
		"rationale": "Fixture only: record synthetic split evidence for contract exercise.",
	}
	correspondence := Correspondence{Key: "fixture:route-correspondence", LeftMembers: []string{"SYNTH-S-01"}, RightMembers: []string{"SYNTH-X-01", "SYNTH-X-02"}, Relation: "split"}
	provenance := []Provenance{{Kind: "synthetic", ProjectID: "synthetic", Path: "fixture:route-decomposition", Claim: "Synthetic route decomposition only.", Trust: "synthetic"}}
	assessment := Assessment{Code: "synthetic-split", Confidence: Confidence{Level: "not-assessed"}, Trust: "synthetic"}
	request := cloneObject(common)
	request["correspondence"], request["provenance"], request["assessment"] = correspondence, provenance, assessment
	payload := observationObject(observation)
	payload["correspondence"], payload["provenance"], payload["assessment"] = correspondence, provenance, assessment
	return makeEvent(seq, requestID, eventType, journalCandidateID, stringsRepeat("a", 64), common, request, payload, observation)
}

func makeDecisionEvent(seq int, requestID string, observation observationPayload, authority Authority, outcome Outcome) json.RawMessage {
	common := map[string]any{
		"schemaVersion": 1, "requestId": requestID, "expectedJournalRevision": seq - 1,
		"candidateId": journalCandidateID, "expectedEvidenceDigest": observation.EvidenceDigest,
		"actor": "fixture-reviewer", "at": fmt.Sprintf("2026-10-01T00:00:0%dZ", seq),
		"rationale": "Fixture only: exercise a requested decision without source effects.",
	}
	request := cloneObject(common)
	request["authority"], request["outcome"] = authority, outcome
	payload := observationObject(observation)
	payload["authority"], payload["outcome"] = authority, outcome
	return makeEvent(seq, requestID, "decision.recorded", journalCandidateID, stringsRepeat("a", 64), common, request, payload, observation)
}

func makeReconsiderEvent(seq int, requestID string, observation observationPayload, supersedes int) json.RawMessage {
	common := map[string]any{
		"schemaVersion": 1, "requestId": requestID, "expectedJournalRevision": seq - 1,
		"candidateId": journalCandidateID, "expectedEvidenceDigest": observation.EvidenceDigest,
		"actor": "fixture-reviewer", "at": fmt.Sprintf("2026-10-01T00:00:0%dZ", seq),
		"rationale": "Fixture only: exercise explicit review after a trigger.",
	}
	trigger := Trigger{ID: "fixture:revisit", Kind: "condition", Description: "The fixture owner explicitly requests re-review."}
	authority := Authority{Owner: "fixture-owner", Status: "unresolved", PermittedDirections: []string{}}
	outcome := Outcome{State: "unresolved"}
	request := cloneObject(common)
	request["supersedesSeq"], request["trigger"], request["authority"], request["outcome"] = supersedes, trigger, authority, outcome
	payload := observationObject(observation)
	payload["supersedesSeq"], payload["trigger"], payload["authority"], payload["outcome"] = supersedes, trigger, authority, outcome
	return makeEvent(seq, requestID, "decision.reconsidered", journalCandidateID, stringsRepeat("a", 64), common, request, payload, observation)
}

func makeEvent(seq int, requestID, eventType, candidateIDValue, pairSHA string, common, request, payload map[string]any, observation observationPayload) json.RawMessage {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}
	scanHash := digest([]byte("synthetic-scan-" + requestID))
	identity := scanIdentity{ScanSHA256: scanHash, PairSHA256: pairSHA, CandidateID: candidateIDValue, EvidenceDigest: observation.EvidenceDigest}
	event := map[string]any{
		"seq": seq, "requestId": requestID, "candidateId": candidateIDValue, "type": eventType,
		"actor": common["actor"], "at": common["at"], "rationale": common["rationale"],
		"request": json.RawMessage(requestBytes), "scanIdentity": identity, "payload": payload,
	}
	data, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	return data
}

func observationObject(observation observationPayload) map[string]any {
	return map[string]any{
		"candidateKey":   observation.CandidateKey,
		"snapshots":      observation.Snapshots,
		"evidenceDigest": observation.EvidenceDigest,
		"evidence":       observation.Evidence,
	}
}

func testCandidate(observation observationPayload, id string) Candidate {
	digest := observation.EvidenceDigest
	return Candidate{
		CandidateID: id, CandidateKey: observation.CandidateKey, EvidenceDigest: &digest,
		Evidence: observation.Evidence, Left: []PathRecord{}, Right: []PathRecord{},
		Confidence: "not-assessed", EvidenceStatus: "not-compared",
	}
}

func observationWithKey(observation observationPayload, key string) observationPayload {
	observation.CandidateKey = key
	left := &projectPin{projectID: observation.Snapshots.Left.ProjectID, revision: observation.Snapshots.Left.Revision}
	right := &projectPin{projectID: observation.Snapshots.Right.ProjectID, revision: observation.Snapshots.Right.Revision}
	observation.EvidenceDigest = evidenceDigest(journalPairID, left, right, key, observation.Evidence)
	return observation
}

func parseTestJournal(t *testing.T, events ...json.RawMessage) *journalState {
	t.Helper()
	state, err := parseJournal(testJournalBytes(events...), journalPairID, journalLeftID, journalRightID)
	if err != nil {
		t.Fatalf("parse synthetic fixture journal: %v", err)
	}
	return state
}

func testJournalBytes(events ...json.RawMessage) []byte {
	return testJournalBytesFor(journalPairID, journalLeftID, journalRightID, events...)
}

func testJournalBytesFor(pairID, leftProjectID, rightProjectID string, events ...json.RawMessage) []byte {
	data, err := json.Marshal(journalDocument{
		SchemaVersion: 1, PairID: pairID, LeftProjectID: leftProjectID, RightProjectID: rightProjectID,
		Revision: len(events), Events: events,
	})
	if err != nil {
		panic(err)
	}
	return data
}

func mutateEvent(t *testing.T, original json.RawMessage, mutate func(map[string]any)) json.RawMessage {
	t.Helper()
	var event map[string]any
	if err := json.Unmarshal(original, &event); err != nil {
		t.Fatal(err)
	}
	mutate(event)
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func cloneObject(value map[string]any) map[string]any {
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func stringPointer(value string) *string { return &value }

func stringsRepeat(value string, count int) string {
	return strings.Repeat(value, count)
}
