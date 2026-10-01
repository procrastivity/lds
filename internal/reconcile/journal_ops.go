package reconcile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/procrastivity/lds/internal/ldserr"
)

// JournalResponse is the stable v1 read/write projection.
type JournalResponse struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Operation     string                 `json:"operation"`
	OK            bool                   `json:"ok"`
	JournalPath   string                 `json:"journalPath"`
	PairID        string                 `json:"pairId"`
	Revision      int                    `json:"revision"`
	Snapshots     Snapshots              `json:"snapshots"`
	Candidates    []JournalCandidateView `json:"candidates"`
	Events        []json.RawMessage      `json:"events"`
	Replayed      *bool                  `json:"replayed,omitempty"`
}

// JournalCandidateView is one deterministically ordered materialized candidate.
type JournalCandidateView struct {
	CandidateID    string          `json:"candidateId"`
	CandidateKey   string          `json:"candidateKey"`
	Snapshots      Snapshots       `json:"snapshots"`
	EvidenceDigest string          `json:"evidenceDigest"`
	Evidence       []Evidence      `json:"evidence"`
	Correspondence Correspondence  `json:"correspondence"`
	Provenance     []Provenance    `json:"provenance"`
	Assessment     Assessment      `json:"assessment"`
	Authority      json.RawMessage `json:"authority"`
	Decision       Outcome         `json:"decision"`
	LastDecision   *LastDecision   `json:"lastDecision"`
	LastEventSeq   int             `json:"lastEventSeq"`
}

// JournalInput names the explicit pair, journal, and (for writes) saved scan
// and caller-authored request files.
type JournalInput struct {
	PairPath    string
	JournalPath string
	ScanPath    string
	RequestPath string
	Operation   string
}

type loadedPair struct {
	path string
	file pairFile
	hash string
}

type journalWriteRequest struct {
	common     requestCommon
	data       json.RawMessage
	record     *recordRequest
	decide     *decideRequest
	reconsider *reconsiderRequest
}

type canonicalJournalPath struct {
	input  string
	target string
	lock   string
}

// ReadJournal validates and projects the named journal without initializing it.
func ReadJournal(input JournalInput) (JournalResponse, error) {
	pair, err := loadPair(input.PairPath)
	if err != nil {
		return JournalResponse{}, err
	}
	journalPath, err := resolveJournalPath(input.JournalPath, false)
	if err != nil {
		return JournalResponse{}, err
	}
	data, err := os.ReadFile(journalPath.target)
	if err != nil {
		return JournalResponse{}, evidenceInvalid("cannot read journal: %v", err)
	}
	state, err := parseJournal(data, *pair.file.PairID, *pair.file.Left.ProjectID, *pair.file.Right.ProjectID)
	if err != nil {
		return JournalResponse{}, err
	}
	return projectJournal("reconcile journal read", journalPath.target, *pair.file.PairID, Snapshots{
		Left:  Snapshot{ProjectID: *pair.file.Left.ProjectID, Revision: *pair.file.Left.Revision},
		Right: Snapshot{ProjectID: *pair.file.Right.ProjectID, Revision: *pair.file.Right.Revision},
	}, state), nil
}

func loadPair(path string) (loadedPair, error) {
	canonical, err := canonicalAbsoluteFile(path)
	if err != nil {
		return loadedPair{}, pairInvalid("pair path must name an existing absolute regular file: %v", err)
	}
	data, err := os.ReadFile(canonical)
	if err != nil {
		return loadedPair{}, pairInvalid("cannot read pair file: %v", err)
	}
	pair, err := parsePair(data)
	if err != nil {
		return loadedPair{}, err
	}
	if _, err := validateScope(pair); err != nil {
		return loadedPair{}, err
	}
	return loadedPair{path: canonical, file: pair, hash: digest(data)}, nil
}

func projectJournal(operation, path, pairID string, snapshots Snapshots, state *journalState) JournalResponse {
	views := make([]JournalCandidateView, 0, len(state.Candidates))
	for _, candidate := range state.Candidates {
		authority := marshalRaw(candidate.Authority)
		if candidate.Authority.Status == "not-declared" {
			authority = json.RawMessage(`{"status":"not-declared"}`)
		}
		views = append(views, JournalCandidateView{
			CandidateID:    candidate.CandidateID,
			CandidateKey:   candidate.CandidateKey,
			Snapshots:      candidate.Snapshots,
			EvidenceDigest: candidate.EvidenceDigest,
			Evidence:       candidate.Evidence,
			Correspondence: candidate.Correspondence,
			Provenance:     candidate.Provenance,
			Assessment:     candidate.Assessment,
			Authority:      authority,
			Decision:       candidate.Decision,
			LastDecision:   candidate.LastDecision,
			LastEventSeq:   candidate.LastEventSeq,
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].CandidateID < views[j].CandidateID })
	return JournalResponse{
		SchemaVersion: 1,
		Operation:     operation,
		OK:            true,
		JournalPath:   path,
		PairID:        pairID,
		Revision:      state.Revision,
		Snapshots:     snapshots,
		Candidates:    views,
		Events:        state.Events,
	}
}

func canonicalJournalEvent(event journalEvent) (json.RawMessage, error) {
	var request any
	switch event.Type {
	case "candidate.recorded", "evidence.observed":
		var value recordRequest
		if err := decodeStrictJSON(event.Request, &value); err != nil {
			return nil, err
		}
		request = value
	case "decision.recorded":
		var value decideRequest
		if err := decodeStrictJSON(event.Request, &value); err != nil {
			return nil, err
		}
		request = value
	case "decision.reconsidered":
		var value reconsiderRequest
		if err := decodeStrictJSON(event.Request, &value); err != nil {
			return nil, err
		}
		request = value
	default:
		return nil, fmt.Errorf("unknown event type %q", event.Type)
	}
	var payload any
	switch event.Type {
	case "candidate.recorded", "evidence.observed":
		var value recordPayload
		if err := decodeStrictJSON(event.Payload, &value); err != nil {
			return nil, err
		}
		payload = value
	case "decision.recorded":
		var value decidePayload
		if err := decodeStrictJSON(event.Payload, &value); err != nil {
			return nil, err
		}
		payload = value
	case "decision.reconsidered":
		var value reconsiderPayload
		if err := decodeStrictJSON(event.Payload, &value); err != nil {
			return nil, err
		}
		payload = value
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	event.Request = requestBytes
	event.Payload = payloadBytes
	encoded, err := json.Marshal(event)
	return json.RawMessage(encoded), err
}

func resolveJournalPath(input string, allowMissing bool) (canonicalJournalPath, error) {
	if !filepath.IsAbs(input) || strings.TrimSpace(input) != input || input == "" {
		return canonicalJournalPath{}, evidenceInvalid("journal path must be a non-empty absolute path")
	}
	clean := filepath.Clean(input)
	base := filepath.Base(clean)
	if base == "." || base == string(filepath.Separator) || base == ".." {
		return canonicalJournalPath{}, evidenceInvalid("journal path must name a file")
	}
	parent, err := canonicalAbsoluteDir(filepath.Dir(clean))
	if err != nil {
		return canonicalJournalPath{}, evidenceInvalid("journal parent must be an existing absolute directory: %v", err)
	}
	target := filepath.Join(parent, base)
	info, err := os.Lstat(target)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, resolveErr := filepath.EvalSymlinks(target)
			if resolveErr != nil {
				return canonicalJournalPath{}, evidenceInvalid("journal symlink cannot be resolved: %v", resolveErr)
			}
			target = resolved
		}
		canonical, fileErr := canonicalAbsoluteFile(target)
		if fileErr != nil {
			return canonicalJournalPath{}, evidenceInvalid("journal must resolve to an existing regular file: %v", fileErr)
		}
		target = canonical
	case errors.Is(err, os.ErrNotExist) && allowMissing:
	case errors.Is(err, os.ErrNotExist):
		return canonicalJournalPath{}, evidenceInvalid("journal file does not exist")
	default:
		return canonicalJournalPath{}, evidenceInvalid("cannot inspect journal path: %v", err)
	}
	return canonicalJournalPath{input: input, target: target, lock: target + ".lock"}, nil
}

func validateScanForWrite(data []byte, pair loadedPair) (Response, string, error) {
	var scan Response
	if err := decodeObject(data, &scan,
		"schemaVersion", "operation", "ok", "pairId", "pairPath", "pairSha256", "leftRoot", "rightRoot", "snapshots", "maps", "candidates",
	); err != nil {
		return Response{}, "", evidenceInvalid("invalid saved scan JSON: %v", err)
	}
	if scan.SchemaVersion != 1 || scan.Operation != "reconcile scan" || !scan.OK || scan.PairID != *pair.file.PairID || scan.PairPath != pair.path || scan.PairSHA256 != pair.hash {
		return Response{}, "", staleEvidence("saved scan does not describe the explicitly named pair bytes")
	}
	if scan.Snapshots.Left.ProjectID != *pair.file.Left.ProjectID || scan.Snapshots.Right.ProjectID != *pair.file.Right.ProjectID ||
		scan.Snapshots.Left.Revision != *pair.file.Left.Revision || scan.Snapshots.Right.Revision != *pair.file.Right.Revision {
		return Response{}, "", staleEvidence("saved scan pins differ from the explicitly named pair")
	}
	if !filepath.IsAbs(scan.LeftRoot) || filepath.Clean(scan.LeftRoot) != scan.LeftRoot || !filepath.IsAbs(scan.RightRoot) || filepath.Clean(scan.RightRoot) != scan.RightRoot || scan.LeftRoot == scan.RightRoot {
		return Response{}, "", evidenceInvalid("saved scan roots must be distinct canonical absolute directories")
	}
	buckets, err := validateScope(pair.file)
	if err != nil {
		return Response{}, "", err
	}
	expected := make(map[string]bucket, len(buckets))
	for _, value := range buckets {
		expected[value.key] = value
	}
	if len(scan.Maps) == 0 || scan.Candidates == nil {
		return Response{}, "", evidenceInvalid("saved scan requires at least one raw map and a candidates array")
	}
	fields, _ := rawObject(data)
	_, withJournal := fields["journalPath"]
	if withJournal && (!filepath.IsAbs(scan.JournalPath) || filepath.Clean(scan.JournalPath) != scan.JournalPath) {
		return Response{}, "", evidenceInvalid("saved scan journalPath must be a canonical absolute path")
	}
	var rawCandidates []json.RawMessage
	_ = json.Unmarshal(fields["candidates"], &rawCandidates)
	mapEvidence := make([]Evidence, 0, len(scan.Maps))
	seenMaps := make(map[string]bool, len(scan.Maps))
	for index, observation := range scan.Maps {
		if (observation.ProjectID != *pair.file.Left.ProjectID && observation.ProjectID != *pair.file.Right.ProjectID) ||
			!validRevision(observation.Revision) || validateRepoPath(observation.Path) != nil || !isLowerHex(observation.SHA256, 64) {
			return Response{}, "", evidenceInvalid("saved scan map observation %d is invalid", index)
		}
		wantRevision := *pair.file.Left.Revision
		if observation.ProjectID == *pair.file.Right.ProjectID {
			wantRevision = *pair.file.Right.Revision
		}
		if observation.Revision != wantRevision {
			return Response{}, "", evidenceInvalid("saved scan map observation %d contradicts its project pin", index)
		}
		key := observation.ProjectID + "\x00" + observation.Revision + "\x00" + observation.Path
		if seenMaps[key] {
			return Response{}, "", evidenceInvalid("saved scan repeats map path %q", observation.Path)
		}
		seenMaps[key] = true
		if index > 0 && !mapObservationLess(
			mapFile{root: &projectPin{projectID: scan.Maps[index-1].ProjectID, revision: scan.Maps[index-1].Revision}, relative: scan.Maps[index-1].Path, sha256: scan.Maps[index-1].SHA256},
			mapFile{root: &projectPin{projectID: observation.ProjectID, revision: observation.Revision}, relative: observation.Path, sha256: observation.SHA256},
		) {
			return Response{}, "", evidenceInvalid("saved scan maps are not in canonical tuple order")
		}
		mapEvidence = append(mapEvidence, Evidence{Kind: "artifact", ProjectID: observation.ProjectID, Revision: observation.Revision, Key: observation.Path, SHA256: observation.SHA256})
	}
	lastID := ""
	seenKeys := make(map[string]bool, len(scan.Candidates))
	currentCount := 0
	var journalRevision *int
	for index, candidate := range scan.Candidates {
		if !isCandidateID(candidate.CandidateID) || !validCandidateKey(candidate.CandidateKey) || candidateID(*pair.file.PairID, candidate.CandidateKey) != candidate.CandidateID ||
			candidate.CandidateID <= lastID || seenKeys[candidate.CandidateKey] || candidate.Confidence != "not-assessed" || candidate.Evidence == nil || candidate.Left == nil || candidate.Right == nil {
			return Response{}, "", evidenceInvalid("saved scan candidate %d has invalid or unordered identity/arrays", index)
		}
		if err := validateScanComparison(candidate, rawCandidates[index], withJournal, *pair.file.Left.ProjectID, *pair.file.Right.ProjectID); err != nil {
			return Response{}, "", evidenceInvalid("saved scan candidate %d comparison: %v", index, err)
		}
		if withJournal {
			if journalRevision != nil && *journalRevision != *candidate.JournalRevision {
				return Response{}, "", evidenceInvalid("saved scan comparison revisions disagree")
			}
			journalRevision = candidate.JournalRevision
		}
		lastID = candidate.CandidateID
		seenKeys[candidate.CandidateKey] = true
		bucket, inScope := expected[candidate.CandidateKey]
		if !inScope {
			if candidate.EvidenceStatus != "out-of-scope" || candidate.EvidenceDigest != nil || len(candidate.Evidence) != 0 || len(candidate.Left) != 0 || len(candidate.Right) != 0 {
				return Response{}, "", evidenceInvalid("saved scan candidate %q is not in current scope", candidate.CandidateKey)
			}
			if candidate.JournalRevision == nil || candidate.JournalEvidenceDigest == nil || candidate.LastDecision == nil {
				return Response{}, "", evidenceInvalid("out-of-scope candidate %q lacks journal comparison metadata", candidate.CandidateKey)
			}
			continue
		}
		currentCount++
		if !validEvidenceStatus(candidate.EvidenceStatus) || candidate.EvidenceStatus == "out-of-scope" || candidate.EvidenceDigest == nil || !isEvidenceDigest(*candidate.EvidenceDigest) {
			return Response{}, "", evidenceInvalid("saved scan candidate %q has invalid current evidence status", candidate.CandidateKey)
		}
		leftEvidence, pathErr := validatePathRecords(bucket.leftPaths, candidate.Left, *pair.file.Left.ProjectID, scan.Snapshots.Left.Revision)
		if pathErr != nil {
			return Response{}, "", evidenceInvalid("saved scan candidate %q left path observations: %v", candidate.CandidateKey, pathErr)
		}
		rightEvidence, pathErr := validatePathRecords(bucket.rightPaths, candidate.Right, *pair.file.Right.ProjectID, scan.Snapshots.Right.Revision)
		if pathErr != nil {
			return Response{}, "", evidenceInvalid("saved scan candidate %q right path observations: %v", candidate.CandidateKey, pathErr)
		}
		expectedEvidence := append([]Evidence(nil), mapEvidence...)
		expectedEvidence = append(expectedEvidence, leftEvidence...)
		expectedEvidence = append(expectedEvidence, rightEvidence...)
		expectedEvidence = uniqueEvidence(expectedEvidence)
		sort.Slice(expectedEvidence, func(i, j int) bool { return evidenceLess(expectedEvidence[i], expectedEvidence[j]) })
		if !reflect.DeepEqual(candidate.Evidence, expectedEvidence) {
			return Response{}, "", evidenceInvalid("saved scan candidate %q evidence rows are inconsistent", candidate.CandidateKey)
		}
		leftPin := &projectPin{projectID: *pair.file.Left.ProjectID, revision: scan.Snapshots.Left.Revision}
		rightPin := &projectPin{projectID: *pair.file.Right.ProjectID, revision: scan.Snapshots.Right.Revision}
		wantDigest := evidenceDigest(*pair.file.PairID, leftPin, rightPin, candidate.CandidateKey, expectedEvidence)
		if *candidate.EvidenceDigest != wantDigest {
			return Response{}, "", evidenceInvalid("saved scan candidate %q digest does not match its rows", candidate.CandidateKey)
		}
	}
	if currentCount != len(expected) {
		return Response{}, "", evidenceInvalid("saved scan does not contain every current pair bucket")
	}
	return scan, digest(data), nil
}

func validateScanComparison(candidate Candidate, raw json.RawMessage, withJournal bool, leftID, rightID string) error {
	fields, _ := rawObject(raw)
	for _, field := range []string{"journalRevision", "journalEvidenceDigest", "lastDecision"} {
		_, present := fields[field]
		if present != withJournal {
			return errors.New("comparison metadata must be present exactly when journalPath is supplied")
		}
	}
	if !withJournal {
		if candidate.EvidenceStatus != "not-compared" {
			return errors.New("scan without journalPath must be not-compared")
		}
		return nil
	}
	if candidate.JournalRevision == nil || *candidate.JournalRevision < 0 {
		return errors.New("journalRevision must be a non-negative integer")
	}
	digestIsNull := bytes.Equal(bytes.TrimSpace(candidate.JournalEvidenceDigest), []byte("null"))
	decisionIsNull := bytes.Equal(bytes.TrimSpace(candidate.LastDecision), []byte("null"))
	switch candidate.EvidenceStatus {
	case "new":
		if !digestIsNull || !decisionIsNull {
			return errors.New("new candidate must have null prior digest and decision")
		}
		return nil
	case "unchanged", "changed", "out-of-scope":
		var observed string
		if digestIsNull || json.Unmarshal(candidate.JournalEvidenceDigest, &observed) != nil || !isEvidenceDigest(observed) || *candidate.JournalRevision < 1 {
			return errors.New("observed candidate requires a valid prior digest and positive journal revision")
		}
		if candidate.EvidenceStatus != "out-of-scope" {
			if candidate.EvidenceDigest == nil || (candidate.EvidenceStatus == "unchanged") != (*candidate.EvidenceDigest == observed) {
				return errors.New("comparison status disagrees with observed and current digests")
			}
		}
	default:
		return errors.New("journal comparison has invalid evidenceStatus")
	}
	if decisionIsNull {
		return nil
	}
	var last LastDecision
	if err := decodeObject(candidate.LastDecision, &last, "seq", "evidenceDigest", "outcome"); err != nil {
		return err
	}
	if last.Seq < 2 || last.Seq > *candidate.JournalRevision || !isEvidenceDigest(last.EvidenceDigest) {
		return errors.New("lastDecision sequence or digest is invalid")
	}
	lastFields, _ := rawObject(candidate.LastDecision)
	if _, err := outcomeFieldSet(lastFields["outcome"]); err != nil {
		return err
	}
	// Scan comparison carries no authority declaration. Validate the outcome
	// shape and pair/direction proof, without inventing authority for its history.
	return validateOutcome(last.Outcome, Authority{Status: "declared", PermittedDirections: []string{"left-to-right", "right-to-left"}}, leftID, rightID)
}

func validEvidenceStatus(value string) bool {
	switch value {
	case "not-compared", "new", "unchanged", "changed":
		return true
	default:
		return false
	}
}

func validatePathRecords(paths []string, records []PathRecord, projectID, revision string) ([]Evidence, error) {
	if len(paths) != len(records) {
		return nil, errors.New("path count differs from declared bucket")
	}
	result := make([]Evidence, 0, len(records))
	for index, record := range records {
		if record.Path != paths[index] {
			return nil, fmt.Errorf("path %q is missing, repeated, or out of order", record.Path)
		}
		if record.Present {
			if record.SHA256 == nil || !isLowerHex(*record.SHA256, 64) {
				return nil, fmt.Errorf("present path %q lacks a valid hash", record.Path)
			}
			result = append(result, Evidence{Kind: "artifact", ProjectID: projectID, Revision: revision, Key: record.Path, SHA256: *record.SHA256})
		} else {
			if record.SHA256 != nil {
				return nil, fmt.Errorf("missing path %q must have null hash", record.Path)
			}
			result = append(result, Evidence{Kind: "missing", ProjectID: projectID, Revision: revision, Key: record.Path, SHA256: "-"})
		}
	}
	return result, nil
}

func parseWriteRequest(operation string, data []byte, pair loadedPair) (journalWriteRequest, error) {
	result := journalWriteRequest{data: append(json.RawMessage(nil), data...)}
	leftID, rightID := *pair.file.Left.ProjectID, *pair.file.Right.ProjectID
	switch operation {
	case "record":
		var request recordRequest
		if err := decodeObject(data, &request,
			"schemaVersion", "requestId", "expectedJournalRevision", "candidateId", "expectedEvidenceDigest", "actor", "at", "rationale", "correspondence", "provenance", "assessment",
		); err != nil {
			return journalWriteRequest{}, evidenceInvalid("invalid record request: %v", err)
		}
		result.common, result.record = request.requestCommon, &request
		if err := validateCorrespondence(request.Correspondence); err != nil {
			return journalWriteRequest{}, evidenceInvalid("invalid record correspondence: %v", err)
		}
		if err := validateAssessment(request.Assessment); err != nil {
			return journalWriteRequest{}, evidenceInvalid("invalid record assessment: %v", err)
		}
		if err := validateProvenance(request.Provenance, observationPayload{Evidence: []Evidence{}}, leftID, rightID); err != nil {
			return journalWriteRequest{}, evidenceInvalid("invalid record provenance: %v", err)
		}
	case "decide":
		var request decideRequest
		if err := decodeObject(data, &request,
			"schemaVersion", "requestId", "expectedJournalRevision", "candidateId", "expectedEvidenceDigest", "actor", "at", "rationale", "authority", "outcome",
		); err != nil {
			return journalWriteRequest{}, evidenceInvalid("invalid decision request: %v", err)
		}
		result.common, result.decide = request.requestCommon, &request
		if err := validateLiveAuthorityOutcome(data, request.Authority, request.Outcome, leftID, rightID); err != nil {
			return journalWriteRequest{}, err
		}
	case "reconsider":
		var request reconsiderRequest
		if err := decodeObject(data, &request,
			"schemaVersion", "requestId", "expectedJournalRevision", "candidateId", "expectedEvidenceDigest", "actor", "at", "rationale", "supersedesSeq", "trigger", "authority", "outcome",
		); err != nil {
			return journalWriteRequest{}, evidenceInvalid("invalid reconsider request: %v", err)
		}
		result.common, result.reconsider = request.requestCommon, &request
		if request.SupersedesSeq < 1 {
			return journalWriteRequest{}, evidenceInvalid("supersedesSeq must be positive")
		}
		if err := validateTrigger(request.Trigger); err != nil {
			return journalWriteRequest{}, evidenceInvalid("invalid reconsider trigger: %v", err)
		}
		if err := optionalReplacementGroup(data); err != nil {
			return journalWriteRequest{}, evidenceInvalid("invalid reconsider replacement group: %v", err)
		}
		if err := validateLiveAuthorityOutcome(data, request.Authority, request.Outcome, leftID, rightID); err != nil {
			return journalWriteRequest{}, err
		}
		if request.Correspondence != nil {
			if err := validateCorrespondence(*request.Correspondence); err != nil {
				return journalWriteRequest{}, evidenceInvalid("invalid reconsider correspondence: %v", err)
			}
			if request.Provenance == nil || request.Assessment == nil {
				return journalWriteRequest{}, evidenceInvalid("reconsider replacement fields cannot be null")
			}
			if err := validateAssessment(*request.Assessment); err != nil {
				return journalWriteRequest{}, evidenceInvalid("invalid reconsider assessment: %v", err)
			}
			if err := validateProvenance(*request.Provenance, observationPayload{Evidence: []Evidence{}}, leftID, rightID); err != nil {
				return journalWriteRequest{}, evidenceInvalid("invalid reconsider provenance: %v", err)
			}
		}
	default:
		return journalWriteRequest{}, evidenceInvalid("unknown journal operation %q", operation)
	}
	if err := validateLiveRequestCommon(result.common); err != nil {
		return journalWriteRequest{}, evidenceInvalid("invalid journal request envelope: %v", err)
	}
	return result, nil
}

func validateLiveRequestCommon(request requestCommon) error {
	if request.SchemaVersion != 1 || !validIdentifier(request.RequestID) || !isCandidateID(request.CandidateID) || !isEvidenceDigest(request.ExpectedEvidenceDigest) || !validIdentifier(request.Actor) || strings.TrimSpace(request.Rationale) == "" || !validUTC(request.At) || request.ExpectedJournalRevision < 0 {
		return errors.New("request common fields are invalid")
	}
	return nil
}

func validateLiveAuthorityOutcome(requestData json.RawMessage, authority Authority, outcome Outcome, leftProjectID, rightProjectID string) error {
	fields, err := rawObject(requestData)
	if err != nil {
		return evidenceInvalid("request must be an object")
	}
	outcomeFields, err := rawObject(fields["outcome"])
	if err != nil {
		return evidenceInvalid("outcome must be an object")
	}
	if err := validateAuthority(authority, leftProjectID, rightProjectID); err != nil {
		return evidenceInvalid("invalid authority declaration: %v", err)
	}
	if outcome.State == "applied" {
		if !validDirection(outcome.Direction) {
			return evidenceInvalid("applied outcome direction is invalid")
		}
		proofData, supplied := outcomeFields["appliedProof"]
		if !supplied {
			if len(outcomeFields) != 2 || outcomeFields["state"] == nil || outcomeFields["direction"] == nil {
				return evidenceInvalid("applied outcome without proof contains invalid fields")
			}
			if authority.Status == "unresolved" || !contains(authority.PermittedDirections, outcome.Direction) {
				return ldserr.New("refusal.reconcile-authority-unresolved", "applied direction is not permitted by declared authority")
			}
			return ldserr.New("refusal.reconcile-applied-proof-required", "applied outcome requires appliedProof")
		}
		if _, err := outcomeFieldSet(fields["outcome"]); err != nil {
			return evidenceInvalid("invalid outcome fields: %v", err)
		}
		if bytes.Equal(bytes.TrimSpace(proofData), []byte("null")) || outcome.AppliedProof == nil {
			return evidenceInvalid("supplied appliedProof must be a valid object")
		}
		if err := validateAppliedProof(*outcome.AppliedProof, outcome.Direction, leftProjectID, rightProjectID); err != nil {
			return evidenceInvalid("invalid appliedProof: %v", err)
		}
	} else if _, err := outcomeFieldSet(fields["outcome"]); err != nil {
		return evidenceInvalid("invalid outcome fields: %v", err)
	}
	if outcome.State != "unresolved" && authority.Status == "unresolved" {
		return ldserr.New("refusal.reconcile-authority-unresolved", "non-unresolved outcome requires declared authority")
	}
	if (outcome.State == "intent" || outcome.State == "applied") && validDirection(outcome.Direction) && !contains(authority.PermittedDirections, outcome.Direction) {
		return ldserr.New("refusal.reconcile-authority-unresolved", "requested direction is not permitted by declared authority")
	}
	if err := validateOutcome(outcome, authority, leftProjectID, rightProjectID); err != nil {
		return evidenceInvalid("invalid outcome: %v", err)
	}
	return nil
}

// WriteJournal performs one explicit record, decide, or reconsider request.
// The scan and pair are validated before a journal lock is acquired; all
// journal state and duplicate-request decisions are then read under that lock.
func WriteJournal(input JournalInput) (JournalResponse, error) {
	if input.Operation != "record" && input.Operation != "decide" && input.Operation != "reconsider" {
		return JournalResponse{}, evidenceInvalid("unknown journal write operation %q", input.Operation)
	}
	pair, err := loadPair(input.PairPath)
	if err != nil {
		return JournalResponse{}, err
	}
	scanPath, err := canonicalAbsoluteFile(input.ScanPath)
	if err != nil {
		return JournalResponse{}, evidenceInvalid("scan path must name an existing absolute regular file: %v", err)
	}
	scanBytes, err := os.ReadFile(scanPath)
	if err != nil {
		return JournalResponse{}, evidenceInvalid("cannot read saved scan: %v", err)
	}
	scan, scanHash, err := validateScanForWrite(scanBytes, pair)
	if err != nil {
		return JournalResponse{}, err
	}
	requestPath, err := canonicalAbsoluteFile(input.RequestPath)
	if err != nil {
		return JournalResponse{}, evidenceInvalid("request path must name an existing absolute regular file: %v", err)
	}
	requestBytes, err := os.ReadFile(requestPath)
	if err != nil {
		return JournalResponse{}, evidenceInvalid("cannot read journal request: %v", err)
	}
	request, err := parseWriteRequest(input.Operation, requestBytes, pair)
	if err != nil {
		return JournalResponse{}, err
	}
	journalPath, err := resolveJournalPath(input.JournalPath, input.Operation == "record")
	if err != nil {
		return JournalResponse{}, err
	}
	protected, err := protectedGitDirs(scan)
	if err != nil {
		return JournalResponse{}, staleEvidence("cannot validate source snapshot boundaries: %v", err)
	}
	if err := ensureWriteBoundary(journalPath.target, journalPath.lock, protected); err != nil {
		return JournalResponse{}, err
	}
	dir, err := openJournalDirectory(filepath.Dir(journalPath.target))
	if err != nil {
		return JournalResponse{}, ldserr.New("refusal.reconcile-write-target", fmt.Sprintf("cannot anchor journal parent without following symlinks: %v", err))
	}
	defer func() { _ = dir.Close() }()
	lock, err := openJournalChild(dir, filepath.Base(journalPath.lock), unix.O_RDWR|unix.O_CREAT, 0o600)
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return JournalResponse{}, ldserr.New("refusal.reconcile-write-target", "journal lock became a symlink")
		}
		return JournalResponse{}, ldserr.New("internal.reconcile-journal-write", fmt.Sprintf("cannot acquire journal writer lock: %v", err))
	}
	lockInfo, statErr := lock.Stat()
	pathInfo, pathErr := os.Lstat(journalPath.lock)
	if statErr != nil || pathErr != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !pathInfo.Mode().IsRegular() || !os.SameFile(lockInfo, pathInfo) {
		_ = lock.Close()
		return JournalResponse{}, ldserr.New("refusal.reconcile-write-target", "journal lock path is not a stable sibling regular file")
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lock.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return JournalResponse{}, ldserr.New("refusal.reconcile-journal-conflict", "journal writer lock is already held")
		}
		return JournalResponse{}, ldserr.New("internal.reconcile-journal-write", fmt.Sprintf("cannot acquire journal writer lock: %v", err))
	}
	defer func() {
		_ = unix.Flock(int(lock.Fd()), unix.LOCK_UN)
		_ = lock.Close()
	}()
	currentPath, err := resolveJournalPath(journalPath.input, input.Operation == "record")
	if err != nil || currentPath.target != journalPath.target {
		return JournalResponse{}, ldserr.New("refusal.reconcile-write-target", "journal path changed after writer lock acquisition")
	}
	if err := checkJournalPathStillSafe(currentPath.target, currentPath.lock, dir, lock, protected); err != nil {
		return JournalResponse{}, err
	}
	state, err := readJournalState(dir, filepath.Base(currentPath.target), input.Operation == "record", pair)
	if err != nil {
		return JournalResponse{}, err
	}
	identity := scanIdentity{
		ScanSHA256:     scanHash,
		PairSHA256:     pair.hash,
		CandidateID:    request.common.CandidateID,
		EvidenceDigest: request.common.ExpectedEvidenceDigest,
	}
	if prior := eventForRequestID(state.Events, request.common.RequestID); prior != nil {
		if !eventMatchesOperation(input.Operation, prior.Type) || !jsonValuesEqual(prior.Request, request.data) || prior.ScanIdentity != identity {
			return JournalResponse{}, ldserr.New("refusal.reconcile-journal-conflict", "requestId is already stored with different request fields or scan identity")
		}
		replayed := true
		response := projectJournal("reconcile journal "+input.Operation, currentPath.target, *pair.file.PairID, snapshotsFromPair(pair.file), state)
		response.Replayed = &replayed
		return response, nil
	}
	if request.common.ExpectedJournalRevision != state.Revision {
		return JournalResponse{}, ldserr.New("refusal.reconcile-journal-conflict", fmt.Sprintf("expected journal revision %d, current revision is %d", request.common.ExpectedJournalRevision, state.Revision))
	}
	candidate, ok := candidateByID(scan.Candidates, request.common.CandidateID)
	if !ok || candidate.EvidenceStatus == "out-of-scope" || candidate.EvidenceDigest == nil {
		return JournalResponse{}, staleEvidence("request candidate is absent from current scan scope")
	}
	if *candidate.EvidenceDigest != request.common.ExpectedEvidenceDigest {
		return JournalResponse{}, staleEvidence("request evidence digest differs from selected scan candidate")
	}
	observation := observationPayload{
		CandidateKey:   candidate.CandidateKey,
		Snapshots:      scan.Snapshots,
		EvidenceDigest: *candidate.EvidenceDigest,
		Evidence:       candidate.Evidence,
	}
	if request.common.CandidateID != candidateID(*pair.file.PairID, candidate.CandidateKey) {
		return JournalResponse{}, staleEvidence("request candidate ID differs from selected scan candidate")
	}
	if err := validateWriteTransition(input.Operation, request, state, observation, *pair.file.PairID, *pair.file.Left.ProjectID, *pair.file.Right.ProjectID); err != nil {
		return JournalResponse{}, err
	}
	event, err := makeWriteEvent(input.Operation, request, observation, state.Revision+1, identity, state)
	if err != nil {
		return JournalResponse{}, ldserr.New("internal.reconcile-journal-write", fmt.Sprintf("cannot serialize journal event: %v", err))
	}
	events := append(append([]json.RawMessage(nil), state.Events...), event)
	document := journalDocument{
		SchemaVersion:  1,
		PairID:         *pair.file.PairID,
		LeftProjectID:  *pair.file.Left.ProjectID,
		RightProjectID: *pair.file.Right.ProjectID,
		Revision:       state.Revision + 1,
		Events:         events,
	}
	newBytes, err := json.Marshal(document)
	if err != nil {
		return JournalResponse{}, ldserr.New("internal.reconcile-journal-write", fmt.Sprintf("cannot serialize journal: %v", err))
	}
	newState, err := parseJournal(newBytes, *pair.file.PairID, *pair.file.Left.ProjectID, *pair.file.Right.ProjectID)
	if err != nil {
		return JournalResponse{}, evidenceInvalid("new event does not replay as a valid journal: %v", err)
	}
	if err := checkJournalPathStillSafe(currentPath.target, currentPath.lock, dir, lock, protected); err != nil {
		return JournalResponse{}, err
	}
	if err := persistJournal(dir, filepath.Base(currentPath.target), newBytes, journalWriteHooks{}); err != nil {
		return JournalResponse{}, err
	}
	replayed := false
	response := projectJournal("reconcile journal "+input.Operation, currentPath.target, *pair.file.PairID, snapshotsFromPair(pair.file), newState)
	response.Replayed = &replayed
	return response, nil
}

func snapshotsFromPair(pair pairFile) Snapshots {
	return Snapshots{
		Left:  Snapshot{ProjectID: *pair.Left.ProjectID, Revision: *pair.Left.Revision},
		Right: Snapshot{ProjectID: *pair.Right.ProjectID, Revision: *pair.Right.Revision},
	}
}

func candidateByID(candidates []Candidate, id string) (Candidate, bool) {
	for _, candidate := range candidates {
		if candidate.CandidateID == id {
			return candidate, true
		}
	}
	return Candidate{}, false
}

func eventForRequestID(events []json.RawMessage, requestID string) *journalEvent {
	for _, data := range events {
		var event journalEvent
		if json.Unmarshal(data, &event) == nil && event.RequestID == requestID {
			return &event
		}
	}
	return nil
}

func eventMatchesOperation(operation, eventType string) bool {
	switch operation {
	case "record":
		return eventType == "candidate.recorded" || eventType == "evidence.observed"
	case "decide":
		return eventType == "decision.recorded"
	case "reconsider":
		return eventType == "decision.reconsidered"
	default:
		return false
	}
}

func readJournalState(dir *os.File, name string, create bool, pair loadedPair) (*journalState, error) {
	file, err := openJournalChild(dir, name, unix.O_RDONLY, 0)
	if errors.Is(err, os.ErrNotExist) && create {
		return &journalState{Candidates: make(map[string]*journalCandidate), Events: []json.RawMessage{}}, nil
	}
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return nil, ldserr.New("refusal.reconcile-write-target", "journal target became a symlink")
		}
		return nil, evidenceInvalid("cannot open journal: %v", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, evidenceInvalid("journal target must be a regular file")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, evidenceInvalid("cannot read journal: %v", err)
	}
	return parseJournal(data, *pair.file.PairID, *pair.file.Left.ProjectID, *pair.file.Right.ProjectID)
}

func validateWriteTransition(operation string, request journalWriteRequest, state *journalState, observation observationPayload, pairID, leftProjectID, rightProjectID string) error {
	candidate := state.Candidates[request.common.CandidateID]
	switch operation {
	case "record":
		value := request.record
		if candidate != nil && candidate.CandidateKey != observation.CandidateKey {
			return ldserr.New("refusal.reconcile-journal-conflict", "candidate ID is already associated with another candidate key")
		}
		if candidate != nil && candidate.LastDecision != nil && !reflect.DeepEqual(candidate.Correspondence, value.Correspondence) {
			return ldserr.New("refusal.reconcile-journal-conflict", "a decided correspondence can change only through reconsider")
		}
		if err := validateProvenance(value.Provenance, observation, leftProjectID, rightProjectID); err != nil {
			return evidenceInvalid("record provenance contradicts current evidence: %v", err)
		}
		replacement := candidateFromRecord(candidate, request.common.CandidateID, observation, *value)
		if err := validateReplacementCorrespondences(state, replacement); err != nil {
			return err
		}
	case "decide":
		if candidate == nil || candidate.CandidateKey != observation.CandidateKey {
			return ldserr.New("refusal.reconcile-journal-conflict", "decision requires a previously recorded candidate")
		}
		if candidate.Decision.State != "pending-review" {
			return ldserr.New("refusal.reconcile-journal-conflict", "an existing decision can change only through reconsider")
		}
		if candidate.EvidenceDigest != observation.EvidenceDigest {
			return staleEvidence("decision scan does not match the latest observed candidate evidence")
		}
	case "reconsider":
		value := request.reconsider
		if candidate == nil || candidate.CandidateKey != observation.CandidateKey || candidate.LastDecision == nil {
			return ldserr.New("refusal.reconcile-journal-conflict", "reconsider requires a previously decided candidate")
		}
		if value.SupersedesSeq != candidate.LastDecision.Seq {
			return ldserr.New("refusal.reconcile-journal-conflict", "supersedesSeq must identify this candidate's latest decision")
		}
		if value.Correspondence != nil {
			if err := validateProvenance(*value.Provenance, observation, leftProjectID, rightProjectID); err != nil {
				return evidenceInvalid("reconsider provenance contradicts current evidence: %v", err)
			}
			replacement := *candidate
			replacement.Correspondence = *value.Correspondence
			replacement.Provenance = *value.Provenance
			replacement.Assessment = *value.Assessment
			if err := validateReplacementCorrespondences(state, &replacement); err != nil {
				return err
			}
		}
	}
	_ = pairID
	return nil
}

func candidateFromRecord(previous *journalCandidate, id string, observation observationPayload, request recordRequest) *journalCandidate {
	candidate := &journalCandidate{
		CandidateID:    id,
		CandidateKey:   observation.CandidateKey,
		Snapshots:      observation.Snapshots,
		EvidenceDigest: observation.EvidenceDigest,
		Evidence:       observation.Evidence,
		Correspondence: request.Correspondence,
		Provenance:     request.Provenance,
		Assessment:     request.Assessment,
		Authority:      Authority{Status: "not-declared"},
		Decision:       Outcome{State: "pending-review"},
	}
	if previous != nil {
		updated := *previous
		updated.Snapshots = observation.Snapshots
		updated.EvidenceDigest = observation.EvidenceDigest
		updated.Evidence = observation.Evidence
		updated.Correspondence = request.Correspondence
		updated.Provenance = request.Provenance
		updated.Assessment = request.Assessment
		candidate = &updated
	}
	return candidate
}

func validateReplacementCorrespondences(state *journalState, replacement *journalCandidate) error {
	projected := &journalState{Candidates: make(map[string]*journalCandidate, len(state.Candidates)+1)}
	for id, candidate := range state.Candidates {
		projected.Candidates[id] = candidate
	}
	projected.Candidates[replacement.CandidateID] = replacement
	if err := validateCurrentCorrespondences(projected); err != nil {
		return ldserr.New("refusal.reconcile-journal-conflict", fmt.Sprintf("reviewed correspondence overlaps existing candidate state: %v", err))
	}
	return nil
}

func makeWriteEvent(operation string, request journalWriteRequest, observation observationPayload, seq int, identity scanIdentity, state *journalState) (json.RawMessage, error) {
	event := journalEvent{
		Seq:          seq,
		RequestID:    request.common.RequestID,
		CandidateID:  request.common.CandidateID,
		Actor:        request.common.Actor,
		At:           request.common.At,
		Rationale:    request.common.Rationale,
		Request:      request.data,
		ScanIdentity: identity,
	}
	var payload any
	switch operation {
	case "record":
		if state.Candidates[request.common.CandidateID] == nil {
			event.Type = "candidate.recorded"
		} else {
			event.Type = "evidence.observed"
		}
		payload = recordPayload{
			observationPayload: observation,
			Correspondence:     request.record.Correspondence,
			Provenance:         request.record.Provenance,
			Assessment:         request.record.Assessment,
		}
	case "decide":
		event.Type = "decision.recorded"
		payload = decidePayload{observationPayload: observation, Authority: request.decide.Authority, Outcome: request.decide.Outcome}
	case "reconsider":
		event.Type = "decision.reconsidered"
		payload = reconsiderPayload{
			observationPayload: observation,
			SupersedesSeq:      request.reconsider.SupersedesSeq,
			Trigger:            request.reconsider.Trigger,
			Authority:          request.reconsider.Authority,
			Outcome:            request.reconsider.Outcome,
			Correspondence:     request.reconsider.Correspondence,
			Provenance:         request.reconsider.Provenance,
			Assessment:         request.reconsider.Assessment,
		}
	default:
		return nil, fmt.Errorf("unknown operation %q", operation)
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	event.Payload = payloadBytes
	encoded, err := json.Marshal(event)
	return json.RawMessage(encoded), err
}

func protectedGitDirs(scan Response) ([]string, error) {
	dirs := make([]string, 0, 4)
	for _, root := range []string{scan.LeftRoot, scan.RightRoot} {
		canonical, err := canonicalAbsoluteDir(root)
		if err != nil || canonical != root {
			return nil, fmt.Errorf("snapshot root %q is no longer canonical or accessible", root)
		}
		commonBytes, err := gitRead(root, "rev-parse", "--git-common-dir")
		if err != nil {
			return nil, err
		}
		common := strings.TrimSpace(string(commonBytes))
		if !filepath.IsAbs(common) {
			common = filepath.Join(root, common)
		}
		common, err = canonicalAbsoluteDir(common)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve Git common directory for %q: %w", root, err)
		}
		for _, path := range []string{root, common} {
			found := false
			for _, existing := range dirs {
				if existing == path {
					found = true
					break
				}
			}
			if !found {
				dirs = append(dirs, path)
			}
		}
	}
	return dirs, nil
}

func ensureWriteBoundary(target, lock string, protected []string) error {
	for _, path := range []string{target, lock} {
		parent := filepath.Dir(path)
		resolved, err := canonicalAbsoluteDir(parent)
		if err != nil || resolved != parent {
			return ldserr.New("refusal.reconcile-write-target", "journal or lock parent changed from its canonical directory")
		}
	}
	for _, candidate := range []string{target, filepath.Dir(target), lock, filepath.Dir(lock)} {
		for _, root := range protected {
			if isWithin(root, candidate) {
				return ldserr.New("refusal.reconcile-write-target", "journal, lock, or replacement directory resolves inside a source snapshot or Git common directory")
			}
		}
	}
	for _, alias := range []string{target, lock} {
		info, err := os.Lstat(alias)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		if alias == lock {
			return ldserr.New("refusal.reconcile-write-target", "journal lock must be a regular sibling file, not a symlink")
		}
		resolved, err := filepath.EvalSymlinks(alias)
		if err != nil {
			return ldserr.New("refusal.reconcile-write-target", "journal or lock symlink cannot be safely resolved")
		}
		for _, root := range protected {
			if isWithin(root, resolved) {
				return ldserr.New("refusal.reconcile-write-target", "journal or lock symlink resolves inside a source snapshot or Git common directory")
			}
		}
	}
	return nil
}

func checkJournalPathStillSafe(target, lockPath string, dir, lockFile *os.File, protected []string) error {
	if err := ensureWriteBoundary(target, lockPath, protected); err != nil {
		return err
	}
	dirInfo, err := dir.Stat()
	parentInfo, parentErr := os.Lstat(filepath.Dir(target))
	if err != nil || parentErr != nil || !parentInfo.IsDir() || !os.SameFile(dirInfo, parentInfo) {
		return ldserr.New("refusal.reconcile-write-target", "journal parent path changed from its anchored directory")
	}
	lockInfo, err := lockFile.Stat()
	pathInfo, pathErr := os.Lstat(lockPath)
	if err != nil || pathErr != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !pathInfo.Mode().IsRegular() || !os.SameFile(lockInfo, pathInfo) {
		return ldserr.New("refusal.reconcile-write-target", "journal lock path changed before replacement")
	}
	info, err := os.Lstat(target)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return ldserr.New("internal.reconcile-journal-write", fmt.Sprintf("cannot recheck journal target before replacement: %v", err))
	}
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, resolveErr := filepath.EvalSymlinks(target)
		if resolveErr != nil {
			return ldserr.New("refusal.reconcile-write-target", "journal target became an unresolvable symlink")
		}
		for _, root := range protected {
			if isWithin(root, resolved) {
				return ldserr.New("refusal.reconcile-write-target", "journal target became an alias into a source snapshot or Git common directory")
			}
		}
	}
	return nil
}

type journalWriteHooks struct {
	beforeCreateTemp    func() error
	beforeRename        func() error
	beforeDirectorySync func() error
}

func persistJournal(dir *os.File, name string, data []byte, hooks journalWriteHooks) error {
	committed, err := atomicReplaceJournal(dir, name, data, hooks)
	if err == nil {
		return nil
	}
	if committed {
		return ldserr.New("internal.reconcile-journal-write", fmt.Sprintf("directory sync failed after journal rename; commit outcome is indeterminate; read the journal revision and requestId before retrying: %v", err))
	}
	return ldserr.New("internal.reconcile-journal-write", fmt.Sprintf("journal write failed before rename; old journal bytes were preserved: %v", err))
}

func atomicReplaceJournal(dir *os.File, name string, data []byte, hooks journalWriteHooks) (bool, error) {
	if hooks.beforeCreateTemp != nil {
		if err := hooks.beforeCreateTemp(); err != nil {
			return false, err
		}
	}
	file, err := createJournalTemp(dir)
	if err != nil {
		return false, err
	}
	tempName := file.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = unix.Unlinkat(int(dir.Fd()), tempName, 0)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return false, err
	}
	n, err := file.Write(data)
	if err == nil && n != len(data) {
		err = errors.New("short write")
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return false, err
	}
	if hooks.beforeRename != nil {
		if err := hooks.beforeRename(); err != nil {
			return false, err
		}
	}
	if err := unix.Renameat(int(dir.Fd()), tempName, int(dir.Fd()), name); err != nil {
		return false, err
	}
	removeTemp = false
	if hooks.beforeDirectorySync != nil {
		if err := hooks.beforeDirectorySync(); err != nil {
			return true, err
		}
	}
	if err := dir.Sync(); err != nil {
		return true, err
	}
	return true, nil
}
