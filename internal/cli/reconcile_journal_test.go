package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

type journalCLISetup struct {
	base      string
	leftRoot  string
	rightRoot string
	pairPath  string
	journal   string
	scanA     string
	scanB     string
	request   string
	candidate struct {
		ID     string `json:"candidateId"`
		Digest string `json:"evidenceDigest"`
	}
	leftRev  string
	rightRev string
}

func TestReconcileJournalCLILifecycleRetriesAndPreservesDecisionEvidence(t *testing.T) {
	setup := newJournalCLISetup(t)
	before := snapshotCLIRepositories(t, setup.leftRoot, setup.rightRoot)
	scanA := writeCLIScan(t, setup, filepath.Join(setup.rightRoot, "maps", "a.md"), setup.scanA)
	setup.candidate = scanCandidateIdentity(t, scanA)
	requestA := recordCLIRequest(t, setup.request, "request-record-1", 0, setup.candidate, "corr:split-review")

	recorded := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", setup.request)
	assertJournalSuccess(t, recorded, 1, false)
	first := decodeJournalCLI(t, recorded.stdout)
	if len(first.Events) != 1 || first.Events[0].Type != "candidate.recorded" || first.Candidates[0].Decision.State != "pending-review" || first.Candidates[0].Authority.Status != "not-declared" {
		t.Fatalf("record response lost replayed review state: %+v", first)
	}
	var storedEvent struct {
		Request json.RawMessage `json:"request"`
		Scan    struct {
			ScanSHA string `json:"scanSha256"`
		} `json:"scanIdentity"`
	}
	if err := json.Unmarshal(first.Events[0].Raw, &storedEvent); err != nil || !jsonValuesEqualCLI(storedEvent.Request, requestA) || storedEvent.Scan.ScanSHA != sha256HexCLI(t, scanA) {
		t.Fatalf("stored event did not retain request/raw scan identity: event=%+v err=%v", storedEvent, err)
	}

	read := run(t, nil, "plumbing", "reconcile", "journal", "read", "--pair", setup.pairPath, "--journal", setup.journal)
	assertJournalSuccess(t, read, 1, nil)
	journalBeforeRead, err := os.ReadFile(setup.journal)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte(read.stdout), []byte(recorded.stdout)) {
		// Operation names differ: read is expected to project the same state, not
		// reuse the write response envelope byte-for-byte.
		readView := decodeJournalCLI(t, read.stdout)
		if readView.Revision != first.Revision || len(readView.Events) != len(first.Events) || readView.Candidates[0].Decision.State != first.Candidates[0].Decision.State {
			t.Fatalf("journal read did not reproduce materialized record state: read=%+v record=%+v", readView, first)
		}
	}
	journalAfterRead, err := os.ReadFile(setup.journal)
	if err != nil || !bytes.Equal(journalBeforeRead, journalAfterRead) {
		t.Fatalf("journal read changed on-disk bytes: %v", err)
	}

	authority := map[string]any{
		"owner": "fixture-owner", "status": "declared", "authoritativeProjectId": "right", "permittedDirections": []string{"right-to-left"},
	}
	decisionRequest := map[string]any{
		"schemaVersion": 1, "requestId": "request-decide-2", "expectedJournalRevision": 1,
		"candidateId": setup.candidate.ID, "expectedEvidenceDigest": setup.candidate.Digest,
		"actor": "fixture-owner", "at": "2026-10-01T00:00:02Z", "rationale": "Preserve a bounded fixture divergence pending its trigger.",
		"authority": authority,
		"outcome":   map[string]any{"state": "deferred", "trigger": map[string]any{"id": "fixture:review", "kind": "condition", "description": "A fixture condition requests reconsideration."}},
	}
	decisionPath := filepath.Join(setup.base, "decision.json")
	writeCLIJSON(t, decisionPath, decisionRequest)
	decided := run(t, nil, "plumbing", "reconcile", "journal", "decide", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", decisionPath)
	assertJournalSuccess(t, decided, 2, false)

	compared := run(t, nil, "plumbing", "reconcile", "scan", "--pair", setup.pairPath, "--left-root", setup.leftRoot, "--right-root", setup.rightRoot, "--map", filepath.Join(setup.rightRoot, "maps", "b.md"), "--journal", setup.journal)
	if compared.exitCode != 0 || compared.stderr != "" {
		t.Fatalf("journal-compared scan failed: %+v", compared)
	}
	scanB := []byte(compared.stdout)
	if err := os.WriteFile(setup.scanB, scanB, 0o600); err != nil {
		t.Fatal(err)
	}
	changedCandidate := scanCandidateIdentity(t, scanB)
	if changedCandidate.ID != setup.candidate.ID || changedCandidate.Digest == setup.candidate.Digest {
		t.Fatalf("changed map must preserve candidate identity and change evidence digest: before=%+v after=%+v", setup.candidate, changedCandidate)
	}
	requestBPath := filepath.Join(setup.base, "record-observation.json")
	recordCLIRequest(t, requestBPath, "request-record-3", 2, changedCandidate, "corr:split-review")
	observed := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanB, "--request", requestBPath)
	assertJournalSuccess(t, observed, 3, false)
	observedView := decodeJournalCLI(t, observed.stdout)
	latest := observedView.Candidates[0]
	if observedView.Events[2].Type != "evidence.observed" || latest.EvidenceDigest != changedCandidate.Digest || latest.Decision.State != "deferred" || latest.LastDecision == nil || latest.LastDecision.Seq != 2 || latest.LastDecision.EvidenceDigest != setup.candidate.Digest {
		t.Fatalf("new observation failed to preserve older decision basis: %+v", latest)
	}

	replayed := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", setup.request)
	assertJournalSuccess(t, replayed, 3, true)
	if len(decodeJournalCLI(t, replayed.stdout).Events) != 3 {
		t.Fatal("exact retry appended a second event")
	}

	journalBeforeRefusals, err := os.ReadFile(setup.journal)
	if err != nil {
		t.Fatal(err)
	}
	pairBeforeRefusals, err := os.ReadFile(setup.pairPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(setup.pairPath, append(append([]byte(nil), pairBeforeRefusals...), ' ', '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	stalePairPath := filepath.Join(setup.base, "record-stale-pair.json")
	recordCLIRequest(t, stalePairPath, "request-stale-pair", 3, changedCandidate, "corr:split-review")
	stalePair := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanB, "--request", stalePairPath, "--json")
	assertJournalError(t, stalePair, 3, "refusal.reconcile-stale-evidence")
	if err := os.WriteFile(setup.pairPath, pairBeforeRefusals, 0o600); err != nil {
		t.Fatal(err)
	}
	changedScanPath := filepath.Join(setup.base, "scan-a-whitespace.json")
	if err := os.WriteFile(changedScanPath, append(append([]byte(nil), scanA...), ' ', '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	changedIdentity := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", changedScanPath, "--request", setup.request, "--json")
	assertJournalError(t, changedIdentity, 3, "refusal.reconcile-journal-conflict")

	changedRequest := map[string]any{}
	if err := json.Unmarshal(requestA, &changedRequest); err != nil {
		t.Fatal(err)
	}
	changedRequest["rationale"] = "Changed content with a reused request ID."
	changedRequestPath := filepath.Join(setup.base, "record-changed.json")
	writeCLIJSON(t, changedRequestPath, changedRequest)
	changedRequestResult := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", changedRequestPath, "--json")
	assertJournalError(t, changedRequestResult, 3, "refusal.reconcile-journal-conflict")

	staleRevisionPath := filepath.Join(setup.base, "record-stale-revision.json")
	recordCLIRequest(t, staleRevisionPath, "request-stale-revision", 0, changedCandidate, "corr:split-review")
	staleRevision := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanB, "--request", staleRevisionPath, "--json")
	assertJournalError(t, staleRevision, 3, "refusal.reconcile-journal-conflict")
	staleDigestPath := filepath.Join(setup.base, "record-stale-digest.json")
	staleDigestCandidate := changedCandidate
	staleDigestCandidate.Digest = setup.candidate.Digest
	recordCLIRequest(t, staleDigestPath, "request-stale-digest", 3, staleDigestCandidate, "corr:split-review")
	staleDigest := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanB, "--request", staleDigestPath, "--json")
	assertJournalError(t, staleDigest, 3, "refusal.reconcile-stale-evidence")
	journalAfterRefusals, err := os.ReadFile(setup.journal)
	if err != nil || !bytes.Equal(journalBeforeRefusals, journalAfterRefusals) {
		t.Fatalf("refused retries changed journal bytes: %v", err)
	}

	reconsiderPath := filepath.Join(setup.base, "reconsider.json")
	reconsiderRequest := map[string]any{
		"schemaVersion": 1, "requestId": "request-reconsider-4", "expectedJournalRevision": 3,
		"candidateId": setup.candidate.ID, "expectedEvidenceDigest": changedCandidate.Digest,
		"actor": "fixture-owner", "at": "2026-10-01T00:00:04Z", "rationale": "Revisit the changed fixture observation explicitly.",
		"supersedesSeq": 2,
		"trigger":       map[string]any{"id": "fixture:review", "kind": "condition", "description": "The external fixture condition changed."},
		"authority":     map[string]any{"owner": "fixture-owner", "status": "declared", "authoritativeProjectId": "right", "permittedDirections": []string{"right-to-left"}},
		"outcome":       map[string]any{"state": "unresolved"},
	}
	writeCLIJSON(t, reconsiderPath, reconsiderRequest)
	reconsidered := run(t, nil, "plumbing", "reconcile", "journal", "reconsider", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanB, "--request", reconsiderPath)
	assertJournalSuccess(t, reconsidered, 4, false)
	final := decodeJournalCLI(t, reconsidered.stdout)
	if final.Events[3].Type != "decision.reconsidered" || final.Candidates[0].Decision.State != "unresolved" || final.Candidates[0].LastDecision.Seq != 4 || final.Candidates[0].LastDecision.EvidenceDigest != changedCandidate.Digest || final.Candidates[0].Correspondence.Key != "corr:split-review" {
		t.Fatalf("reconsideration failed to supersede while preserving absent replacement group: %+v", final.Candidates[0])
	}
	lateRetry := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", setup.request)
	assertJournalSuccess(t, lateRetry, 4, true)
	if !equalCLIFiles(before, snapshotCLIRepositories(t, setup.leftRoot, setup.rightRoot)) {
		t.Fatal("journal lifecycle modified pinned source snapshot or Git metadata bytes")
	}
	finalRead := run(t, nil, "plumbing", "reconcile", "journal", "read", "--pair", setup.pairPath, "--journal", setup.journal)
	assertJournalSuccess(t, finalRead, 4, nil)
	if len(decodeJournalCLI(t, finalRead.stdout).Events) != 4 {
		t.Fatal("journal read omitted lifecycle history")
	}

	usage := run(t, nil, "plumbing", "reconcile", "journal", "read", "--pair", setup.pairPath, "--json")
	if usage.exitCode != 2 || usage.stdout != "" || strings.HasPrefix(strings.TrimSpace(usage.stderr), "{") {
		t.Fatalf("journal usage errors stay human under --json: %+v", usage)
	}
}

func TestReconcileJournalCLIRetryPreservesOptionalEmptyFields(t *testing.T) {
	setup := newJournalCLISetup(t)
	scan := writeCLIScan(t, setup, filepath.Join(setup.rightRoot, "maps", "a.md"), setup.scanA)
	setup.candidate = scanCandidateIdentity(t, scan)
	requestBytes := recordCLIRequest(t, setup.request, "empty-rationale-record", 0, setup.candidate, "corr:empty-rationale")
	var request map[string]any
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		t.Fatal(err)
	}
	confidence := request["assessment"].(map[string]any)["confidence"].(map[string]any)
	confidence["rationale"] = ""
	requestBytes = writeCLIJSON(t, setup.request, request)
	args := []string{"plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", setup.request, "--json"}
	recorded := run(t, nil, args...)
	assertJournalSuccess(t, recorded, 1, false)
	var event struct {
		Request json.RawMessage `json:"request"`
	}
	if err := json.Unmarshal(decodeJournalCLI(t, recorded.stdout).Events[0].Raw, &event); err != nil || !jsonValuesEqualCLI(event.Request, requestBytes) {
		t.Fatalf("full validated request was not preserved: request=%s event=%s err=%v", requestBytes, event.Request, err)
	}
	before, err := os.ReadFile(setup.journal)
	if err != nil {
		t.Fatal(err)
	}
	assertJournalSuccess(t, run(t, nil, args...), 1, true)
	delete(confidence, "rationale")
	writeCLIJSON(t, setup.request, request)
	assertJournalError(t, run(t, nil, args...), 3, "refusal.reconcile-journal-conflict")
	request["requestId"], request["expectedJournalRevision"] = "null-rationale-record", 1
	confidence["rationale"] = nil
	writeCLIJSON(t, setup.request, request)
	assertJournalError(t, run(t, nil, args...), 1, "validation.reconcile-evidence-invalid")
	after, err := os.ReadFile(setup.journal)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("retry/refusal changed journal bytes: %v", err)
	}
}

func TestReconcileJournalCLIRejectsInvalidSavedScanComparisonAndEmptyMaps(t *testing.T) {
	setup := newJournalCLISetup(t)
	baseline := writeCLIScan(t, setup, filepath.Join(setup.rightRoot, "maps", "a.md"), setup.scanA)
	setup.candidate = scanCandidateIdentity(t, baseline)
	recordCLIRequest(t, setup.request, "scan-validation-record", 0, setup.candidate, "corr:scan-validation")
	before := snapshotCLIRepositories(t, setup.leftRoot, setup.rightRoot)
	comparison := func(scan, candidate map[string]any, status string, observed any) {
		scan["journalPath"] = setup.journal
		candidate["evidenceStatus"], candidate["journalRevision"] = status, 2
		candidate["journalEvidenceDigest"], candidate["lastDecision"] = observed, nil
	}
	for _, test := range []struct {
		name   string
		mutate func(map[string]any, map[string]any)
	}{
		{"opaque decision without journal", func(_, c map[string]any) { c["lastDecision"] = "not a decision object" }},
		{"null revision without journal", func(_, c map[string]any) { c["journalRevision"] = nil }},
		{"journal but not-compared", func(s, _ map[string]any) { s["journalPath"] = setup.journal }},
		{"new missing comparison fields", func(s, c map[string]any) { s["journalPath"], c["evidenceStatus"] = setup.journal, "new" }},
		{"new with prior digest", func(s, c map[string]any) { comparison(s, c, "new", setup.candidate.Digest) }},
		{"nonstring observed digest", func(s, c map[string]any) {
			comparison(s, c, "unchanged", map[string]any{"digest": setup.candidate.Digest})
		}},
		{"changed with equal digest", func(s, c map[string]any) { comparison(s, c, "changed", setup.candidate.Digest) }},
		{"unchanged with different digest", func(s, c map[string]any) { comparison(s, c, "unchanged", "sha256:"+strings.Repeat("0", 64)) }},
		{"opaque decision in comparison", func(s, c map[string]any) {
			comparison(s, c, "unchanged", setup.candidate.Digest)
			c["lastDecision"] = "not a decision object"
		}},
		{"decision beyond journal revision", func(s, c map[string]any) {
			comparison(s, c, "unchanged", setup.candidate.Digest)
			c["lastDecision"] = map[string]any{"seq": 3, "evidenceDigest": setup.candidate.Digest, "outcome": map[string]any{"state": "unresolved"}}
		}},
		{"malformed decision outcome", func(s, c map[string]any) {
			comparison(s, c, "unchanged", setup.candidate.Digest)
			c["lastDecision"] = map[string]any{"seq": 2, "evidenceDigest": setup.candidate.Digest, "outcome": "not an outcome object"}
		}},
		{"malformed deferred outcome", func(s, c map[string]any) {
			comparison(s, c, "unchanged", setup.candidate.Digest)
			c["lastDecision"] = map[string]any{"seq": 2, "evidenceDigest": setup.candidate.Digest, "outcome": map[string]any{"state": "deferred", "trigger": map[string]any{"id": "fixture:review"}}}
		}},
		{"empty maps with self-consistent digest", func(s, c map[string]any) {
			s["maps"] = []any{}
			rows := []any{}
			for _, row := range c["evidence"].([]any) {
				if !strings.HasPrefix(row.(map[string]any)["key"].(string), "maps/") {
					rows = append(rows, row)
				}
			}
			c["evidence"] = rows
			// Independent frozen-contract transcript, not the production digest helper.
			transcript := strings.Join([]string{"lds-reconcile-evidence-v1", s["pairId"].(string), c["candidateKey"].(string), "left", setup.leftRev, "right", setup.rightRev}, "\n") + "\n"
			for _, row := range rows {
				item := row.(map[string]any)
				transcript += strings.Join([]string{item["kind"].(string), item["projectId"].(string), item["revision"].(string), item["key"].(string), item["sha256"].(string)}, "\t") + "\n"
			}
			hash := sha256.Sum256([]byte(transcript))
			c["evidenceDigest"] = "sha256:" + hex.EncodeToString(hash[:])
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var scan map[string]any
			if err := json.Unmarshal(baseline, &scan); err != nil {
				t.Fatal(err)
			}
			candidate := scan["candidates"].([]any)[0].(map[string]any)
			test.mutate(scan, candidate)
			request := recordCLIRequest(t, setup.request, "scan-validation-record", 0, setup.candidate, "corr:scan-validation")
			var fields map[string]any
			if err := json.Unmarshal(request, &fields); err != nil {
				t.Fatal(err)
			}
			fields["expectedEvidenceDigest"] = candidate["evidenceDigest"]
			writeCLIJSON(t, setup.request, fields)
			writeCLIJSON(t, setup.scanA, scan)
			result := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", setup.request, "--json")
			assertJournalError(t, result, 1, "validation.reconcile-evidence-invalid")
			for _, path := range []string{setup.journal, setup.journal + ".lock"} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("invalid scan created journal or lock: %s: %v", path, err)
				}
			}
			if !equalCLIFiles(before, snapshotCLIRepositories(t, setup.leftRoot, setup.rightRoot)) {
				t.Fatal("invalid scan changed source or Git bytes")
			}
		})
	}
	if err := os.WriteFile(setup.scanA, baseline, 0o600); err != nil {
		t.Fatal(err)
	}
	recordCLIRequest(t, setup.request, "scan-validation-record", 0, setup.candidate, "corr:scan-validation")
	assertJournalSuccess(t, run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", setup.request), 1, false)
}

func TestReconcileJournalCLIRejectsMalformedAndUnauthorizedAppliedProofs(t *testing.T) {
	setup := newJournalCLISetup(t)
	scan := writeCLIScan(t, setup, filepath.Join(setup.rightRoot, "maps", "a.md"), setup.scanA)
	setup.candidate = scanCandidateIdentity(t, scan)
	recordCLIRequest(t, setup.request, "proof-record", 0, setup.candidate, "corr:proof-test")
	recorded := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", setup.request)
	assertJournalSuccess(t, recorded, 1, false)
	before, err := os.ReadFile(setup.journal)
	if err != nil {
		t.Fatal(err)
	}
	baseRequest := func(id string, proof any, includeProof bool, permitted []string) map[string]any {
		outcome := map[string]any{"state": "applied", "direction": "left-to-right"}
		if includeProof {
			outcome["appliedProof"] = proof
		}
		return map[string]any{
			"schemaVersion": 1, "requestId": id, "expectedJournalRevision": 1,
			"candidateId": setup.candidate.ID, "expectedEvidenceDigest": setup.candidate.Digest,
			"actor": "fixture-owner", "at": "2026-10-01T00:00:02Z", "rationale": "Test applied proof validation.",
			"authority": map[string]any{"owner": "fixture-owner", "status": "declared", "authoritativeProjectId": "left", "permittedDirections": permitted},
			"outcome":   outcome,
		}
	}
	validProof := map[string]any{
		"uri": "https://example.invalid/proof.json", "sha256": strings.Repeat("a", 64),
		"source":      map[string]any{"projectId": "left", "revision": setup.leftRev},
		"destination": map[string]any{"projectId": "right", "revision": setup.rightRev},
		"scope":       map[string]any{"sourcePaths": []string{"left.md"}, "destinationPaths": []string{"right.md"}},
		"verifier":    "fixture-verifier", "verifiedAt": "2026-10-01T00:00:02Z", "rationale": "Disposable schema validation only.",
	}
	cases := []struct {
		name     string
		request  map[string]any
		wantCode string
		wantExit int
	}{
		{"proof absent", baseRequest("proof-absent", nil, false, []string{"left-to-right"}), "refusal.reconcile-applied-proof-required", 3},
		{"proof malformed", baseRequest("proof-malformed", map[string]any{"uri": "https://example.invalid/proof.json"}, true, []string{"right-to-left"}), "validation.reconcile-evidence-invalid", 1},
		{"proof direction malformed before authority refusal", baseRequest("proof-direction-malformed", map[string]any{
			"uri": "https://example.invalid/proof.json", "sha256": strings.Repeat("a", 64),
			"source":      map[string]any{"projectId": "right", "revision": setup.rightRev},
			"destination": map[string]any{"projectId": "left", "revision": setup.leftRev},
			"scope":       map[string]any{"sourcePaths": []string{"right.md"}, "destinationPaths": []string{"left.md"}},
			"verifier":    "fixture-verifier", "verifiedAt": "2026-10-01T00:00:02Z", "rationale": "Wrong direction fixture.",
		}, true, []string{"right-to-left"}), "validation.reconcile-evidence-invalid", 1},
		{"valid proof direction not permitted", baseRequest("proof-not-permitted", validProof, true, []string{"right-to-left"}), "refusal.reconcile-authority-unresolved", 3},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			requestPath := filepath.Join(setup.base, strings.ReplaceAll(test.name, " ", "-")+".json")
			writeCLIJSON(t, requestPath, test.request)
			result := run(t, nil, "plumbing", "reconcile", "journal", "decide", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", requestPath, "--json")
			assertJournalError(t, result, test.wantExit, test.wantCode)
		})
	}
	duplicateRequestPath := filepath.Join(setup.base, "duplicate-key.json")
	duplicate := `{"schemaVersion":1,"requestId":"bad","requestId":"bad2"}`
	if err := os.WriteFile(duplicateRequestPath, []byte(duplicate), 0o600); err != nil {
		t.Fatal(err)
	}
	duplicateResult := run(t, nil, "plumbing", "reconcile", "journal", "decide", "--pair", setup.pairPath, "--journal", setup.journal, "--scan", setup.scanA, "--request", duplicateRequestPath, "--json")
	assertJournalError(t, duplicateResult, 1, "validation.reconcile-evidence-invalid")
	after, err := os.ReadFile(setup.journal)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("invalid proof/request cases modified journal: %v", err)
	}
}

func TestReconcileJournalCLIBoundariesLocksAliasesAndMissingRead(t *testing.T) {
	setup := newJournalCLISetup(t)
	scan := writeCLIScan(t, setup, filepath.Join(setup.rightRoot, "maps", "a.md"), setup.scanA)
	setup.candidate = scanCandidateIdentity(t, scan)
	recordCLIRequest(t, setup.request, "boundary-record", 0, setup.candidate, "corr:boundary")
	sourceBefore := snapshotCLIRepositories(t, setup.leftRoot, setup.rightRoot)
	for _, target := range []string{
		filepath.Join(setup.leftRoot, "journal-inside.json"),
		filepath.Join(setup.leftRoot, ".git", "HEAD"),
	} {
		result := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", target, "--scan", setup.scanA, "--request", setup.request, "--json")
		assertJournalError(t, result, 3, "refusal.reconcile-write-target")
	}
	alias := filepath.Join(setup.base, "journal-alias.json")
	if err := os.Symlink(filepath.Join(setup.rightRoot, "right.md"), alias); err != nil {
		t.Fatal(err)
	}
	aliasResult := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", alias, "--scan", setup.scanA, "--request", setup.request, "--json")
	assertJournalError(t, aliasResult, 3, "refusal.reconcile-write-target")

	lockedJournal := filepath.Join(setup.base, "locked.json")
	lockFile, err := os.OpenFile(lockedJournal+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lockFile.Close()
		t.Fatal(err)
	}
	lockedResult := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", lockedJournal, "--scan", setup.scanA, "--request", setup.request, "--json")
	_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	_ = lockFile.Close()
	assertJournalError(t, lockedResult, 3, "refusal.reconcile-journal-conflict")
	if _, err := os.Stat(lockedJournal); !os.IsNotExist(err) {
		t.Fatalf("locked first record initialized journal: %v", err)
	}

	lockAliasJournal := filepath.Join(setup.base, "lock-alias.json")
	if err := os.Symlink(filepath.Join(setup.leftRoot, "left.md"), lockAliasJournal+".lock"); err != nil {
		t.Fatal(err)
	}
	lockAliasResult := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", setup.pairPath, "--journal", lockAliasJournal, "--scan", setup.scanA, "--request", setup.request, "--json")
	assertJournalError(t, lockAliasResult, 3, "refusal.reconcile-write-target")

	missingRead := run(t, nil, "plumbing", "reconcile", "journal", "read", "--pair", setup.pairPath, "--journal", filepath.Join(setup.base, "missing.json"), "--json")
	assertJournalError(t, missingRead, 1, "validation.reconcile-evidence-invalid")
	if _, err := os.Stat(filepath.Join(setup.base, "missing.json")); !os.IsNotExist(err) {
		t.Fatalf("read initialized a missing journal: %v", err)
	}
	if !equalCLIFiles(sourceBefore, snapshotCLIRepositories(t, setup.leftRoot, setup.rightRoot)) {
		t.Fatal("refused journal boundary cases changed a source or Git metadata byte")
	}
}

func TestReconcileJournalCLISplitUnmatchedAndCrossCandidateOverlap(t *testing.T) {
	base := t.TempDir()
	leftRoot, rightRoot := filepath.Join(base, "left"), filepath.Join(base, "right")
	leftRev := makeReconcileSnapshot(t, leftRoot, map[string][]byte{"left/a.md": []byte("a\n"), "left/b.md": []byte("b\n")})
	rightRev := makeReconcileSnapshot(t, rightRoot, map[string][]byte{
		"right/x.md": []byte("x\n"), "right/y.md": []byte("y\n"), "right/unmatched.md": []byte("u\n"),
	})
	pairPath := filepath.Join(base, "pair.json")
	writeReconcilePair(t, pairPath, map[string]any{
		"schemaVersion": 1, "pairId": "fixture:split-unmatched:v1",
		"left": map[string]any{"projectId": "left", "revision": leftRev}, "right": map[string]any{"projectId": "right", "revision": rightRev},
		"scope": map[string]any{
			"leftPaths": []string{"left/a.md"}, "rightPaths": []string{"right/unmatched.md", "right/x.md", "right/y.md"},
			"candidates": []any{map[string]any{"candidateKey": "topic:split", "leftPaths": []string{"left/a.md"}, "rightPaths": []string{"right/x.md", "right/y.md"}}},
		},
	})
	journalPath, scanPath := filepath.Join(base, "journal.json"), filepath.Join(base, "scan.json")
	scan := run(t, nil, "plumbing", "reconcile", "scan", "--pair", pairPath, "--left-root", leftRoot, "--right-root", rightRoot, "--map", filepath.Join(rightRoot, "right", "unmatched.md"))
	if scan.exitCode != 0 || scan.stderr != "" {
		t.Fatalf("split/unmatched scan: %+v", scan)
	}
	if err := os.WriteFile(scanPath, []byte(scan.stdout), 0o600); err != nil {
		t.Fatal(err)
	}
	var scanned struct {
		Candidates []struct {
			ID     string `json:"candidateId"`
			Key    string `json:"candidateKey"`
			Digest string `json:"evidenceDigest"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(scan.stdout), &scanned); err != nil || len(scanned.Candidates) != 2 {
		t.Fatalf("scan candidates = %+v err=%v", scanned, err)
	}
	byKey := map[string]struct{ ID, Digest string }{}
	for _, candidate := range scanned.Candidates {
		byKey[candidate.Key] = struct{ ID, Digest string }{candidate.ID, candidate.Digest}
	}
	requests := []struct {
		key            string
		correspondence map[string]any
	}{
		{"topic:split", map[string]any{"key": "corr:split", "leftMembers": []string{"node:a"}, "rightMembers": []string{"node:x", "node:y"}, "relation": "split"}},
		{"unmapped:right:right/unmatched.md", map[string]any{"key": "corr:unmatched", "leftMembers": []string{}, "rightMembers": []string{"node:unmatched"}, "relation": "right-unmatched"}},
	}
	for index, item := range requests {
		identity := byKey[item.key]
		requestPath := filepath.Join(base, fmt.Sprintf("record-%d.json", index))
		record := map[string]any{
			"schemaVersion": 1, "requestId": fmt.Sprintf("split-record-%d", index), "expectedJournalRevision": index,
			"candidateId": identity.ID, "expectedEvidenceDigest": identity.Digest,
			"actor": "fixture-owner", "at": fmt.Sprintf("2026-10-01T00:00:0%dZ", index+1), "rationale": "Record the fixture-only reviewed relation.",
			"correspondence": item.correspondence, "provenance": []any{},
			"assessment": map[string]any{"code": "synthetic-fixture", "confidence": map[string]any{"level": "not-assessed"}, "trust": "synthetic"},
		}
		writeCLIJSON(t, requestPath, record)
		result := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", pairPath, "--journal", journalPath, "--scan", scanPath, "--request", requestPath)
		assertJournalSuccess(t, result, index+1, false)
	}
	read := run(t, nil, "plumbing", "reconcile", "journal", "read", "--pair", pairPath, "--journal", journalPath)
	assertJournalSuccess(t, read, 2, nil)
	before, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	overlapPath := filepath.Join(base, "overlap.json")
	third := byKey["unmapped:right:right/unmatched.md"]
	overlap := map[string]any{
		"schemaVersion": 1, "requestId": "overlap-attempt", "expectedJournalRevision": 2,
		"candidateId": third.ID, "expectedEvidenceDigest": third.Digest,
		"actor": "fixture-owner", "at": "2026-10-01T00:00:03Z", "rationale": "Attempt a conflicting correspondence.",
		"correspondence": map[string]any{"key": "corr:overlap", "leftMembers": []string{"node:a"}, "rightMembers": []string{"node:other"}, "relation": "one-to-one"},
		"provenance":     []any{}, "assessment": map[string]any{"code": "synthetic-fixture", "confidence": map[string]any{"level": "not-assessed"}, "trust": "synthetic"},
	}
	writeCLIJSON(t, overlapPath, overlap)
	result := run(t, nil, "plumbing", "reconcile", "journal", "record", "--pair", pairPath, "--journal", journalPath, "--scan", scanPath, "--request", overlapPath, "--json")
	assertJournalError(t, result, 3, "refusal.reconcile-journal-conflict")
	after, err := os.ReadFile(journalPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("cross-candidate overlap changed journal: %v", err)
	}
}

func newJournalCLISetup(t *testing.T) journalCLISetup {
	t.Helper()
	base := t.TempDir()
	leftRoot, rightRoot := filepath.Join(base, "left"), filepath.Join(base, "right")
	leftRev := makeReconcileSnapshot(t, leftRoot, map[string][]byte{"left.md": []byte("left fixture evidence\n")})
	rightRev := makeReconcileSnapshot(t, rightRoot, map[string][]byte{
		"right.md": []byte("right fixture evidence\n"), "maps/a.md": []byte("map a\n"), "maps/b.md": []byte("map b\n"),
	})
	pairPath := filepath.Join(base, "pair.json")
	writeReconcilePair(t, pairPath, map[string]any{
		"schemaVersion": 1, "pairId": "fixture:journal-cli:v1",
		"left": map[string]any{"projectId": "left", "revision": leftRev}, "right": map[string]any{"projectId": "right", "revision": rightRev},
		"scope": map[string]any{
			"leftPaths": []string{"left.md"}, "rightPaths": []string{"right.md"},
			"candidates": []any{map[string]any{"candidateKey": "topic:journal-cli", "leftPaths": []string{"left.md"}, "rightPaths": []string{"right.md"}}},
		},
	})
	return journalCLISetup{
		base: base, leftRoot: leftRoot, rightRoot: rightRoot, pairPath: pairPath,
		journal: filepath.Join(base, "journal.json"), scanA: filepath.Join(base, "scan-a.json"),
		scanB: filepath.Join(base, "scan-b.json"), request: filepath.Join(base, "record.json"),
		leftRev: leftRev, rightRev: rightRev,
	}
}

func writeCLIScan(t *testing.T, setup journalCLISetup, mapPath, outputPath string) []byte {
	t.Helper()
	result := run(t, nil, "plumbing", "reconcile", "scan", "--pair", setup.pairPath, "--left-root", setup.leftRoot, "--right-root", setup.rightRoot, "--map", mapPath)
	if result.exitCode != 0 || result.stderr != "" {
		t.Fatalf("scan: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
	}
	data := []byte(result.stdout)
	if err := os.WriteFile(outputPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return data
}

func scanCandidateIdentity(t *testing.T, scan []byte) struct {
	ID     string `json:"candidateId"`
	Digest string `json:"evidenceDigest"`
} {
	t.Helper()
	var response struct {
		Candidates []struct {
			ID     string `json:"candidateId"`
			Digest string `json:"evidenceDigest"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(scan, &response); err != nil || len(response.Candidates) == 0 {
		t.Fatalf("decode scan candidate: err=%v candidates=%+v", err, response.Candidates)
	}
	return struct {
		ID     string `json:"candidateId"`
		Digest string `json:"evidenceDigest"`
	}{response.Candidates[0].ID, response.Candidates[0].Digest}
}

func recordCLIRequest(t *testing.T, path, requestID string, revision int, candidate struct {
	ID     string `json:"candidateId"`
	Digest string `json:"evidenceDigest"`
}, correspondenceKey string,
) []byte {
	t.Helper()
	request := map[string]any{
		"schemaVersion": 1, "requestId": requestID, "expectedJournalRevision": revision,
		"candidateId": candidate.ID, "expectedEvidenceDigest": candidate.Digest,
		"actor": "fixture-owner", "at": "2026-10-01T00:00:01Z", "rationale": "Record the reviewed fixture correspondence.",
		"correspondence": map[string]any{"key": correspondenceKey, "leftMembers": []string{"node:left"}, "rightMembers": []string{"node:right"}, "relation": "one-to-one"},
		"provenance":     []any{},
		"assessment":     map[string]any{"code": "bounded-fixture", "confidence": map[string]any{"level": "not-assessed"}, "trust": "bounded"},
	}
	return writeCLIJSON(t, path, request)
}

func writeCLIJSON(t *testing.T, path string, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return data
}

type journalCLICandidate struct {
	EvidenceDigest string `json:"evidenceDigest"`
	Decision       struct {
		State string `json:"state"`
	} `json:"decision"`
	Correspondence struct {
		Key string `json:"key"`
	} `json:"correspondence"`
	LastDecision *struct {
		Seq            int    `json:"seq"`
		EvidenceDigest string `json:"evidenceDigest"`
	} `json:"lastDecision"`
	Authority struct {
		Status string `json:"status"`
	} `json:"authority"`
}

type journalCLIEvent struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"-"`
}

type journalCLIResponse struct {
	Revision   int                   `json:"revision"`
	Replayed   *bool                 `json:"replayed"`
	Candidates []journalCLICandidate `json:"candidates"`
	Events     []journalCLIEvent     `json:"events"`
}

func decodeJournalCLI(t *testing.T, output string) journalCLIResponse {
	t.Helper()
	var response journalCLIResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("journal output is not JSON: %v: %s", err, output)
	}
	var events []json.RawMessage
	var raw struct {
		Events []json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		t.Fatal(err)
	}
	events = raw.Events
	for index := range response.Events {
		response.Events[index].Raw = events[index]
	}
	return response
}

func assertJournalSuccess(t *testing.T, result result, revision int, replayed any) {
	t.Helper()
	if result.exitCode != 0 || result.stderr != "" || result.stdout == "" {
		t.Fatalf("journal success: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
	}
	var value struct {
		SchemaVersion int    `json:"schemaVersion"`
		Operation     string `json:"operation"`
		OK            bool   `json:"ok"`
		Revision      int    `json:"revision"`
		Replayed      *bool  `json:"replayed"`
	}
	if err := json.Unmarshal([]byte(result.stdout), &value); err != nil {
		t.Fatalf("journal success is not JSON: %v: %s", err, result.stdout)
	}
	if value.SchemaVersion != 1 || !value.OK || value.Revision != revision || !strings.HasPrefix(value.Operation, "reconcile journal ") {
		t.Fatalf("unexpected journal envelope: %+v", value)
	}
	if expected, ok := replayed.(bool); ok && (value.Replayed == nil || *value.Replayed != expected) {
		t.Fatalf("replayed = %v, want %t", value.Replayed, expected)
	} else if replayed == nil && value.Replayed != nil {
		t.Fatalf("read response has replayed field: %+v", value)
	}
}

func assertJournalError(t *testing.T, result result, exit int, code string) {
	t.Helper()
	if result.exitCode != exit || result.stdout != "" {
		t.Fatalf("journal error %s: exit=%d stdout=%q stderr=%q", code, result.exitCode, result.stdout, result.stderr)
	}
	if strings.HasPrefix(strings.TrimSpace(result.stderr), "{") {
		if got := parseErrorEnvelope(t, result.stderr).Error.Code; got != code {
			t.Fatalf("error code=%q, want %q; stderr=%s", got, code, result.stderr)
		}
	} else if !strings.Contains(result.stderr, code) && exit != 2 {
		t.Fatalf("human error does not identify %s: %s", code, result.stderr)
	}
}

func snapshotCLIRepositories(t *testing.T, roots ...string) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	for index, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			result[filepath.Join(fmt.Sprint(index), relative)] = data
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func equalCLIFiles(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for name, content := range left {
		if !bytes.Equal(content, right[name]) {
			return false
		}
	}
	return true
}

func jsonValuesEqualCLI(left, right []byte) bool {
	var a, b any
	return json.Unmarshal(left, &a) == nil && json.Unmarshal(right, &b) == nil && reflect.DeepEqual(a, b)
}

func sha256HexCLI(t *testing.T, data []byte) string {
	t.Helper()
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
