package reconcile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
)

type journalDocument struct {
	SchemaVersion  int               `json:"schemaVersion"`
	PairID         string            `json:"pairId"`
	LeftProjectID  string            `json:"leftProjectId"`
	RightProjectID string            `json:"rightProjectId"`
	Revision       int               `json:"revision"`
	Events         []json.RawMessage `json:"events"`
}

type journalEvent struct {
	Seq          int             `json:"seq"`
	RequestID    string          `json:"requestId"`
	CandidateID  string          `json:"candidateId"`
	Type         string          `json:"type"`
	Actor        string          `json:"actor"`
	At           string          `json:"at"`
	Rationale    string          `json:"rationale"`
	Request      json.RawMessage `json:"request"`
	ScanIdentity scanIdentity    `json:"scanIdentity"`
	Payload      json.RawMessage `json:"payload"`
}

type scanIdentity struct {
	ScanSHA256     string `json:"scanSha256"`
	PairSHA256     string `json:"pairSha256"`
	CandidateID    string `json:"candidateId"`
	EvidenceDigest string `json:"evidenceDigest"`
}

type observationPayload struct {
	CandidateKey   string     `json:"candidateKey"`
	Snapshots      Snapshots  `json:"snapshots"`
	EvidenceDigest string     `json:"evidenceDigest"`
	Evidence       []Evidence `json:"evidence"`
}

// Correspondence is a reviewer-supplied relationship between project-local members.
type Correspondence struct {
	Key          string   `json:"key"`
	LeftMembers  []string `json:"leftMembers"`
	RightMembers []string `json:"rightMembers"`
	Relation     string   `json:"relation"`
}

// Provenance records the source or fixture basis for a reviewer claim.
type Provenance struct {
	Kind      string  `json:"kind"`
	ProjectID string  `json:"projectId"`
	Revision  *string `json:"revision"`
	Path      string  `json:"path"`
	SHA256    *string `json:"sha256"`
	Claim     string  `json:"claim"`
	Trust     string  `json:"trust"`
}

// Assessment records reviewer confidence and trust without automating judgment.
type Assessment struct {
	Code       string     `json:"code"`
	Confidence Confidence `json:"confidence"`
	Trust      string     `json:"trust"`
}

// Confidence is a qualitative reviewer assessment, not a score.
type Confidence struct {
	Level     string `json:"level"`
	Rationale string `json:"rationale,omitempty"`
}

// Authority is the explicit owner declaration governing permitted directions.
type Authority struct {
	Owner                  string   `json:"owner"`
	Status                 string   `json:"status"`
	AuthoritativeProjectID *string  `json:"authoritativeProjectId"`
	PermittedDirections    []string `json:"permittedDirections"`
}

// Trigger identifies the explicit condition or event for later reconsideration.
type Trigger struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

// Outcome is the current recorded disposition for a reviewed candidate.
type Outcome struct {
	State        string        `json:"state"`
	Direction    string        `json:"direction,omitempty"`
	Trigger      *Trigger      `json:"trigger,omitempty"`
	AppliedProof *AppliedProof `json:"appliedProof,omitempty"`
}

// AppliedProof references external evidence for a claimed completed transfer.
type AppliedProof struct {
	URI         string       `json:"uri"`
	SHA256      string       `json:"sha256"`
	Source      ProofProject `json:"source"`
	Destination ProofProject `json:"destination"`
	Scope       ProofScope   `json:"scope"`
	Verifier    string       `json:"verifier"`
	VerifiedAt  string       `json:"verifiedAt"`
	Rationale   string       `json:"rationale"`
}

// ProofProject identifies one side of an externally verified transfer.
type ProofProject struct {
	ProjectID string `json:"projectId"`
	Revision  string `json:"revision"`
}

// ProofScope lists the normalized files in the externally verified transfer.
type ProofScope struct {
	SourcePaths      []string `json:"sourcePaths"`
	DestinationPaths []string `json:"destinationPaths"`
}

// LastDecision preserves the last explicit decision and its evidence basis.
type LastDecision struct {
	Seq            int     `json:"seq"`
	EvidenceDigest string  `json:"evidenceDigest"`
	Outcome        Outcome `json:"outcome"`
}

type journalCandidate struct {
	CandidateID    string
	CandidateKey   string
	Snapshots      Snapshots
	EvidenceDigest string
	Evidence       []Evidence
	Correspondence Correspondence
	Provenance     []Provenance
	Assessment     Assessment
	Authority      Authority
	Decision       Outcome
	LastDecision   *LastDecision
	LastEventSeq   int
}

type journalState struct {
	Revision   int
	Candidates map[string]*journalCandidate
}

type requestCommon struct {
	SchemaVersion           int    `json:"schemaVersion"`
	RequestID               string `json:"requestId"`
	ExpectedJournalRevision int    `json:"expectedJournalRevision"`
	CandidateID             string `json:"candidateId"`
	ExpectedEvidenceDigest  string `json:"expectedEvidenceDigest"`
	Actor                   string `json:"actor"`
	At                      string `json:"at"`
	Rationale               string `json:"rationale"`
}

type recordRequest struct {
	requestCommon
	Correspondence Correspondence `json:"correspondence"`
	Provenance     []Provenance   `json:"provenance"`
	Assessment     Assessment     `json:"assessment"`
}

type decideRequest struct {
	requestCommon
	Authority Authority `json:"authority"`
	Outcome   Outcome   `json:"outcome"`
}

type reconsiderRequest struct {
	requestCommon
	SupersedesSeq  int             `json:"supersedesSeq"`
	Trigger        Trigger         `json:"trigger"`
	Authority      Authority       `json:"authority"`
	Outcome        Outcome         `json:"outcome"`
	Correspondence *Correspondence `json:"correspondence,omitempty"`
	Provenance     *[]Provenance   `json:"provenance,omitempty"`
	Assessment     *Assessment     `json:"assessment,omitempty"`
}

type recordPayload struct {
	observationPayload
	Correspondence Correspondence `json:"correspondence"`
	Provenance     []Provenance   `json:"provenance"`
	Assessment     Assessment     `json:"assessment"`
}

type decidePayload struct {
	observationPayload
	Authority Authority `json:"authority"`
	Outcome   Outcome   `json:"outcome"`
}

type reconsiderPayload struct {
	observationPayload
	SupersedesSeq  int             `json:"supersedesSeq"`
	Trigger        Trigger         `json:"trigger"`
	Authority      Authority       `json:"authority"`
	Outcome        Outcome         `json:"outcome"`
	Correspondence *Correspondence `json:"correspondence,omitempty"`
	Provenance     *[]Provenance   `json:"provenance,omitempty"`
	Assessment     *Assessment     `json:"assessment,omitempty"`
}

type replayEvent struct {
	envelope       journalEvent
	observation    observationPayload
	correspondence *Correspondence
	provenance     []Provenance
	assessment     *Assessment
	authority      *Authority
	outcome        *Outcome
	supersedesSeq  int
}

func parseJournal(data []byte, pairID, leftProjectID, rightProjectID string) (*journalState, error) {
	var document journalDocument
	if err := decodeObject(data, &document,
		"schemaVersion", "pairId", "leftProjectId", "rightProjectId", "revision", "events",
	); err != nil {
		return nil, evidenceInvalid("invalid journal JSON: %v", err)
	}
	if document.SchemaVersion != 1 || document.PairID != pairID || document.LeftProjectID != leftProjectID || document.RightProjectID != rightProjectID || document.Revision < 0 || document.Events == nil || document.Revision != len(document.Events) {
		return nil, evidenceInvalid("journal schema, pair identity, revision, or events are incompatible")
	}
	state := &journalState{Revision: document.Revision, Candidates: make(map[string]*journalCandidate)}
	requestIDs := make(map[string]bool, len(document.Events))
	for index, data := range document.Events {
		event, err := parseJournalEvent(data, index+1, pairID, leftProjectID, rightProjectID)
		if err != nil {
			return nil, evidenceInvalid("invalid journal event %d: %v", index+1, err)
		}
		if requestIDs[event.envelope.RequestID] {
			return nil, evidenceInvalid("journal requestId %q is duplicated", event.envelope.RequestID)
		}
		requestIDs[event.envelope.RequestID] = true
		if err := replayJournalEvent(state, event, pairID); err != nil {
			return nil, evidenceInvalid("invalid journal event %d lifecycle: %v", index+1, err)
		}
		if err := validateCurrentCorrespondences(state); err != nil {
			return nil, evidenceInvalid("invalid journal event %d correspondence state: %v", index+1, err)
		}
	}
	return state, nil
}

func parseJournalEvent(data []byte, seq int, pairID, leftProjectID, rightProjectID string) (replayEvent, error) {
	var envelope journalEvent
	if err := decodeObject(data, &envelope,
		"seq", "requestId", "candidateId", "type", "actor", "at", "rationale", "request", "scanIdentity", "payload",
	); err != nil {
		return replayEvent{}, err
	}
	if envelope.Seq != seq || !validIdentifier(envelope.RequestID) || !validIdentifier(envelope.CandidateID) || !validIdentifier(envelope.Actor) || strings.TrimSpace(envelope.Rationale) == "" || !validUTC(envelope.At) {
		return replayEvent{}, errors.New("event sequence or envelope fields are invalid")
	}
	identity := envelope.ScanIdentity
	if !isLowerHex(identity.ScanSHA256, 64) || !isLowerHex(identity.PairSHA256, 64) || identity.CandidateID != envelope.CandidateID || !isEvidenceDigest(identity.EvidenceDigest) {
		return replayEvent{}, errors.New("scanIdentity fields are invalid or contradict the event")
	}
	var result replayEvent
	result.envelope = envelope
	var request requestCommon
	switch envelope.Type {
	case "candidate.recorded", "evidence.observed":
		var typed recordRequest
		if err := decodeObject(envelope.Request, &typed,
			"schemaVersion", "requestId", "expectedJournalRevision", "candidateId", "expectedEvidenceDigest", "actor", "at", "rationale", "correspondence", "provenance", "assessment",
		); err != nil {
			return replayEvent{}, fmt.Errorf("record request: %w", err)
		}
		request = typed.requestCommon
		var payload recordPayload
		if err := decodeObject(envelope.Payload, &payload,
			"candidateKey", "snapshots", "evidenceDigest", "evidence", "correspondence", "provenance", "assessment",
		); err != nil {
			return replayEvent{}, fmt.Errorf("record payload: %w", err)
		}
		if err := payloadFieldsAgree(envelope.Request, envelope.Payload, "correspondence", "provenance", "assessment"); err != nil {
			return replayEvent{}, err
		}
		if err := validateCorrespondence(payload.Correspondence); err != nil {
			return replayEvent{}, err
		}
		if err := validateAssessment(payload.Assessment); err != nil {
			return replayEvent{}, err
		}
		if err := validateProvenance(payload.Provenance, payload.observationPayload, leftProjectID, rightProjectID); err != nil {
			return replayEvent{}, err
		}
		result.observation = payload.observationPayload
		result.correspondence = &payload.Correspondence
		result.provenance = payload.Provenance
		result.assessment = &payload.Assessment
	case "decision.recorded":
		var typed decideRequest
		if err := decodeObject(envelope.Request, &typed,
			"schemaVersion", "requestId", "expectedJournalRevision", "candidateId", "expectedEvidenceDigest", "actor", "at", "rationale", "authority", "outcome",
		); err != nil {
			return replayEvent{}, fmt.Errorf("decision request: %w", err)
		}
		request = typed.requestCommon
		var payload decidePayload
		if err := decodeObject(envelope.Payload, &payload,
			"candidateKey", "snapshots", "evidenceDigest", "evidence", "authority", "outcome",
		); err != nil {
			return replayEvent{}, fmt.Errorf("decision payload: %w", err)
		}
		if err := payloadFieldsAgree(envelope.Request, envelope.Payload, "authority", "outcome"); err != nil {
			return replayEvent{}, err
		}
		if err := validateOutcomeFields(envelope.Request, envelope.Payload); err != nil {
			return replayEvent{}, err
		}
		if err := validateAuthority(payload.Authority, leftProjectID, rightProjectID); err != nil {
			return replayEvent{}, err
		}
		if err := validateOutcome(payload.Outcome, payload.Authority, leftProjectID, rightProjectID); err != nil {
			return replayEvent{}, err
		}
		result.observation = payload.observationPayload
		result.authority = &payload.Authority
		result.outcome = &payload.Outcome
	case "decision.reconsidered":
		var typed reconsiderRequest
		if err := decodeObject(envelope.Request, &typed,
			"schemaVersion", "requestId", "expectedJournalRevision", "candidateId", "expectedEvidenceDigest", "actor", "at", "rationale", "supersedesSeq", "trigger", "authority", "outcome",
		); err != nil {
			return replayEvent{}, fmt.Errorf("reconsider request: %w", err)
		}
		request = typed.requestCommon
		var payload reconsiderPayload
		if err := decodeObject(envelope.Payload, &payload,
			"candidateKey", "snapshots", "evidenceDigest", "evidence", "supersedesSeq", "trigger", "authority", "outcome",
		); err != nil {
			return replayEvent{}, fmt.Errorf("reconsider payload: %w", err)
		}
		if err := optionalReplacementGroup(envelope.Request); err != nil {
			return replayEvent{}, err
		}
		if err := optionalReplacementGroup(envelope.Payload); err != nil {
			return replayEvent{}, err
		}
		if err := payloadFieldsAgree(envelope.Request, envelope.Payload,
			"supersedesSeq", "trigger", "authority", "outcome", "correspondence", "provenance", "assessment",
		); err != nil {
			return replayEvent{}, err
		}
		if err := validateOutcomeFields(envelope.Request, envelope.Payload); err != nil {
			return replayEvent{}, err
		}
		if err := validateTrigger(payload.Trigger); err != nil {
			return replayEvent{}, err
		}
		if err := validateAuthority(payload.Authority, leftProjectID, rightProjectID); err != nil {
			return replayEvent{}, err
		}
		if err := validateOutcome(payload.Outcome, payload.Authority, leftProjectID, rightProjectID); err != nil {
			return replayEvent{}, err
		}
		if payload.Correspondence != nil {
			if payload.Provenance == nil || payload.Assessment == nil {
				return replayEvent{}, errors.New("reconsider replacement group contains null fields")
			}
			if err := validateCorrespondence(*payload.Correspondence); err != nil {
				return replayEvent{}, err
			}
			result.correspondence = payload.Correspondence
			result.provenance = *payload.Provenance
			result.assessment = payload.Assessment
			if err := validateAssessment(*payload.Assessment); err != nil {
				return replayEvent{}, err
			}
			if err := validateProvenance(result.provenance, payload.observationPayload, leftProjectID, rightProjectID); err != nil {
				return replayEvent{}, err
			}
		}
		result.observation = payload.observationPayload
		result.authority = &payload.Authority
		result.outcome = &payload.Outcome
		result.supersedesSeq = payload.SupersedesSeq
	default:
		return replayEvent{}, fmt.Errorf("unknown event type %q", envelope.Type)
	}
	if err := validateRequestEnvelope(envelope, request, seq); err != nil {
		return replayEvent{}, err
	}
	if err := validateObservation(result.observation, pairID, leftProjectID, rightProjectID); err != nil {
		return replayEvent{}, err
	}
	if candidateID(pairID, result.observation.CandidateKey) != envelope.CandidateID {
		return replayEvent{}, errors.New("candidateId does not match pairId and candidateKey")
	}
	if envelope.ScanIdentity.EvidenceDigest != result.observation.EvidenceDigest || request.ExpectedEvidenceDigest != result.observation.EvidenceDigest {
		return replayEvent{}, errors.New("request, scanIdentity, and payload evidence digests disagree")
	}
	return result, nil
}

func decodeObject(data []byte, target any, required ...string) error {
	if err := decodeStrictJSON(data, target); err != nil {
		return err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return errors.New("expected a JSON object")
	}
	for _, key := range required {
		if _, ok := object[key]; !ok {
			return fmt.Errorf("missing required field %q", key)
		}
	}
	return nil
}

func validateRequestEnvelope(event journalEvent, request requestCommon, seq int) error {
	if request.SchemaVersion != 1 || request.RequestID != event.RequestID || request.CandidateID != event.CandidateID || request.Actor != event.Actor || request.At != event.At || request.Rationale != event.Rationale || request.ExpectedJournalRevision != seq-1 || request.ExpectedJournalRevision < 0 || !isEvidenceDigest(request.ExpectedEvidenceDigest) {
		return errors.New("request fields disagree with event envelope or sequence")
	}
	if !validIdentifier(request.RequestID) || !validIdentifier(request.CandidateID) || !validIdentifier(request.Actor) || strings.TrimSpace(request.Rationale) == "" || !validUTC(request.At) {
		return errors.New("request common fields are invalid")
	}
	return nil
}

func validateObservation(observation observationPayload, pairID, leftProjectID, rightProjectID string) error {
	if !validCandidateKey(observation.CandidateKey) || observation.Snapshots.Left.ProjectID != leftProjectID || observation.Snapshots.Right.ProjectID != rightProjectID || !validRevision(observation.Snapshots.Left.Revision) || !validRevision(observation.Snapshots.Right.Revision) || !isEvidenceDigest(observation.EvidenceDigest) || observation.Evidence == nil || len(observation.Evidence) == 0 {
		return errors.New("observation identity, snapshots, or evidence are invalid")
	}
	seen := make(map[Evidence]bool, len(observation.Evidence))
	for _, item := range observation.Evidence {
		if !validateEvidenceItem(item, observation.Snapshots, leftProjectID, rightProjectID) {
			return fmt.Errorf("evidence item %q is invalid", item.Key)
		}
		if seen[item] {
			return fmt.Errorf("evidence item %q is duplicated", item.Key)
		}
		seen[item] = true
	}
	if !sort.SliceIsSorted(observation.Evidence, func(i, j int) bool { return evidenceLess(observation.Evidence[i], observation.Evidence[j]) }) {
		return errors.New("evidence items are not in canonical tuple order")
	}
	left := &projectPin{projectID: leftProjectID, revision: observation.Snapshots.Left.Revision}
	right := &projectPin{projectID: rightProjectID, revision: observation.Snapshots.Right.Revision}
	expected := evidenceDigest(pairID, left, right, observation.CandidateKey, observation.Evidence)
	if observation.EvidenceDigest != expected {
		return errors.New("observation evidenceDigest does not match its exact evidence rows")
	}
	return nil
}

func validateEvidenceItem(item Evidence, snapshots Snapshots, leftProjectID, rightProjectID string) bool {
	if err := validateRepoPath(item.Key); err != nil {
		return false
	}
	switch item.ProjectID {
	case leftProjectID:
		if item.Revision != snapshots.Left.Revision {
			return false
		}
	case rightProjectID:
		if item.Revision != snapshots.Right.Revision {
			return false
		}
	default:
		return false
	}
	if item.Kind == "artifact" {
		return isLowerHex(item.SHA256, 64)
	}
	return item.Kind == "missing" && item.SHA256 == "-"
}

func validateCorrespondence(value Correspondence) error {
	if !validIdentifier(value.Key) || value.LeftMembers == nil || value.RightMembers == nil {
		return errors.New("correspondence key or member arrays are invalid")
	}
	for _, members := range [][]string{value.LeftMembers, value.RightMembers} {
		seen := make(map[string]bool, len(members))
		for _, member := range members {
			if !validIdentifier(member) || seen[member] {
				return errors.New("correspondence members must be non-empty and unique per side")
			}
			seen[member] = true
		}
	}
	left, right := len(value.LeftMembers), len(value.RightMembers)
	valid := false
	switch value.Relation {
	case "one-to-one":
		valid = left == 1 && right == 1
	case "left-unmatched":
		valid = left == 1 && right == 0
	case "right-unmatched":
		valid = left == 0 && right == 1
	case "split":
		valid = left == 1 && right >= 2
	case "merged":
		valid = left >= 2 && right == 1
	}
	if !valid {
		return fmt.Errorf("correspondence relation %q does not match member cardinality", value.Relation)
	}
	return nil
}

func validateProvenance(values []Provenance, observation observationPayload, leftProjectID, rightProjectID string) error {
	if values == nil {
		return errors.New("provenance must be an array")
	}
	for _, item := range values {
		if strings.TrimSpace(item.Claim) == "" || !validTrust(item.Trust) {
			return errors.New("provenance claim or trust is invalid")
		}
		switch item.Kind {
		case "artifact", "missing":
			if (item.ProjectID != leftProjectID && item.ProjectID != rightProjectID) || item.Revision == nil || !validRevision(*item.Revision) || validateRepoPath(item.Path) != nil || item.SHA256 == nil {
				return errors.New("artifact provenance identity is invalid")
			}
			if item.Kind == "artifact" && !isLowerHex(*item.SHA256, 64) || item.Kind == "missing" && *item.SHA256 != "-" {
				return errors.New("artifact provenance hash is invalid")
			}
			for _, evidence := range observation.Evidence {
				if evidence.ProjectID == item.ProjectID && evidence.Revision == *item.Revision && evidence.Key == item.Path && evidence.SHA256 != *item.SHA256 {
					return errors.New("provenance contradicts a digest-bearing evidence row")
				}
			}
		case "synthetic":
			if item.ProjectID != "synthetic" || item.Revision != nil || item.SHA256 != nil || !validIdentifier(item.Path) || item.Trust != "synthetic" {
				return errors.New("synthetic provenance must name only a fixture key and synthetic trust")
			}
		default:
			return fmt.Errorf("unknown provenance kind %q", item.Kind)
		}
	}
	return nil
}

func validateAssessment(value Assessment) error {
	if strings.TrimSpace(value.Code) == "" || !validTrust(value.Trust) {
		return errors.New("assessment code or trust is invalid")
	}
	switch value.Confidence.Level {
	case "not-assessed":
	case "low", "medium", "high":
		if strings.TrimSpace(value.Confidence.Rationale) == "" {
			return errors.New("assessed confidence requires a rationale")
		}
	default:
		return fmt.Errorf("unknown confidence level %q", value.Confidence.Level)
	}
	return nil
}

func validateAuthority(value Authority, leftProjectID, rightProjectID string) error {
	if strings.TrimSpace(value.Owner) == "" || value.PermittedDirections == nil {
		return errors.New("authority owner or permittedDirections is invalid")
	}
	seen := map[string]bool{}
	for _, direction := range value.PermittedDirections {
		if !validDirection(direction) || seen[direction] {
			return errors.New("authority has an invalid or duplicate permitted direction")
		}
		seen[direction] = true
	}
	switch value.Status {
	case "unresolved":
		if value.AuthoritativeProjectID != nil || len(value.PermittedDirections) != 0 {
			return errors.New("unresolved authority must have null project and no permitted directions")
		}
	case "declared":
		if value.AuthoritativeProjectID == nil || (*value.AuthoritativeProjectID != leftProjectID && *value.AuthoritativeProjectID != rightProjectID) || len(value.PermittedDirections) == 0 {
			return errors.New("declared authority must name a pair project and permitted direction")
		}
	default:
		return fmt.Errorf("unknown authority status %q", value.Status)
	}
	return nil
}

func validateOutcome(outcome Outcome, authority Authority, leftProjectID, rightProjectID string) error {
	switch outcome.State {
	case "unresolved":
		if outcome.Direction != "" || outcome.Trigger != nil || outcome.AppliedProof != nil {
			return errors.New("unresolved outcome has unrelated fields")
		}
	case "divergence-retained":
		if authority.Status != "declared" || outcome.Direction != "" || outcome.Trigger != nil || outcome.AppliedProof != nil {
			return errors.New("divergence-retained outcome has invalid authority or fields")
		}
	case "intent":
		if authority.Status != "declared" || !contains(authority.PermittedDirections, outcome.Direction) || outcome.Trigger != nil || outcome.AppliedProof != nil {
			return errors.New("intent direction is not permitted by declared authority")
		}
	case "deferred":
		if authority.Status != "declared" || outcome.Direction != "" || outcome.Trigger == nil || outcome.AppliedProof != nil {
			return errors.New("deferred outcome requires declared authority and a trigger")
		}
		if err := validateTrigger(*outcome.Trigger); err != nil {
			return err
		}
	case "applied":
		if authority.Status != "declared" || !contains(authority.PermittedDirections, outcome.Direction) || outcome.Trigger != nil || outcome.AppliedProof == nil {
			return errors.New("applied outcome lacks permitted direction or proof")
		}
		if err := validateAppliedProof(*outcome.AppliedProof, outcome.Direction, leftProjectID, rightProjectID); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown outcome state %q", outcome.State)
	}
	return nil
}

func validateAppliedProof(proof AppliedProof, direction, leftProjectID, rightProjectID string) error {
	uri, err := url.Parse(proof.URI)
	if err != nil || !uri.IsAbs() || uri.Scheme == "" || !isLowerHex(proof.SHA256, 64) || strings.TrimSpace(proof.Verifier) == "" || !validUTC(proof.VerifiedAt) || strings.TrimSpace(proof.Rationale) == "" {
		return errors.New("appliedProof URI or verification fields are invalid")
	}
	if !validRevision(proof.Source.Revision) || !validRevision(proof.Destination.Revision) || proof.Source.ProjectID == proof.Destination.ProjectID {
		return errors.New("appliedProof project revisions are invalid")
	}
	if direction == "left-to-right" && (proof.Source.ProjectID != leftProjectID || proof.Destination.ProjectID != rightProjectID) || direction == "right-to-left" && (proof.Source.ProjectID != rightProjectID || proof.Destination.ProjectID != leftProjectID) {
		return errors.New("appliedProof projects do not match outcome direction")
	}
	for _, paths := range [][]string{proof.Scope.SourcePaths, proof.Scope.DestinationPaths} {
		if len(paths) == 0 || !sort.StringsAreSorted(paths) {
			return errors.New("appliedProof scope paths must be non-empty and sorted")
		}
		for index, name := range paths {
			if validateRepoPath(name) != nil || index > 0 && paths[index-1] == name {
				return errors.New("appliedProof scope paths must be normalized and unique")
			}
		}
	}
	return nil
}

func validateTrigger(value Trigger) error {
	if !validIdentifier(value.ID) || strings.TrimSpace(value.Description) == "" || (value.Kind != "event" && value.Kind != "date" && value.Kind != "condition") {
		return errors.New("trigger fields are invalid")
	}
	return nil
}

func replayJournalEvent(state *journalState, event replayEvent, pairID string) error {
	envelope := event.envelope
	candidateIDValue := envelope.CandidateID
	if !isCandidateID(candidateIDValue) || candidateID(pairID, event.observation.CandidateKey) != candidateIDValue {
		return errors.New("candidate ID does not match the recorded candidate key")
	}
	candidate := state.Candidates[candidateIDValue]
	seq := envelope.Seq
	switch envelope.Type {
	case "candidate.recorded":
		if candidate != nil {
			return errors.New("candidate.recorded repeats an existing candidate")
		}
		candidate = &journalCandidate{
			CandidateID:    candidateIDValue,
			CandidateKey:   event.observation.CandidateKey,
			Snapshots:      event.observation.Snapshots,
			EvidenceDigest: event.observation.EvidenceDigest,
			Evidence:       event.observation.Evidence,
			Correspondence: *event.correspondence,
			Provenance:     event.provenance,
			Assessment:     *event.assessment,
			Authority:      Authority{Status: "not-declared"},
			Decision:       Outcome{State: "pending-review"},
			LastEventSeq:   seq,
		}
		state.Candidates[candidateIDValue] = candidate
	case "evidence.observed":
		if candidate == nil || candidate.CandidateKey != event.observation.CandidateKey {
			return errors.New("evidence.observed has no matching recorded candidate")
		}
		if candidate.LastDecision != nil && !reflect.DeepEqual(candidate.Correspondence, *event.correspondence) {
			return errors.New("evidence.observed changes a decided correspondence")
		}
		candidate.Snapshots = event.observation.Snapshots
		candidate.EvidenceDigest = event.observation.EvidenceDigest
		candidate.Evidence = event.observation.Evidence
		candidate.Correspondence = *event.correspondence
		candidate.Provenance = event.provenance
		candidate.Assessment = *event.assessment
		candidate.LastEventSeq = seq
	case "decision.recorded":
		if candidate == nil || candidate.CandidateKey != event.observation.CandidateKey || candidate.Decision.State != "pending-review" || candidate.EvidenceDigest != event.observation.EvidenceDigest {
			return errors.New("decision.recorded does not select the latest pending observation")
		}
		candidate.Authority = *event.authority
		candidate.Decision = *event.outcome
		candidate.LastDecision = &LastDecision{Seq: seq, EvidenceDigest: event.observation.EvidenceDigest, Outcome: *event.outcome}
		candidate.LastEventSeq = seq
	case "decision.reconsidered":
		if candidate == nil || candidate.CandidateKey != event.observation.CandidateKey || candidate.LastDecision == nil || event.supersedesSeq != candidate.LastDecision.Seq {
			return errors.New("decision.reconsidered does not supersede the latest decision")
		}
		candidate.Snapshots = event.observation.Snapshots
		candidate.EvidenceDigest = event.observation.EvidenceDigest
		candidate.Evidence = event.observation.Evidence
		if event.correspondence != nil {
			candidate.Correspondence = *event.correspondence
			candidate.Provenance = event.provenance
			candidate.Assessment = *event.assessment
		}
		candidate.Authority = *event.authority
		candidate.Decision = *event.outcome
		candidate.LastDecision = &LastDecision{Seq: seq, EvidenceDigest: event.observation.EvidenceDigest, Outcome: *event.outcome}
		candidate.LastEventSeq = seq
	default:
		return errors.New("unrecognized replay operation")
	}
	return nil
}

func validateCurrentCorrespondences(state *journalState) error {
	keys := map[string]string{}
	members := map[string]string{}
	for id, candidate := range state.Candidates {
		if existing, ok := keys[candidate.Correspondence.Key]; ok && existing != id {
			return fmt.Errorf("correspondence key %q is used by multiple candidates", candidate.Correspondence.Key)
		}
		keys[candidate.Correspondence.Key] = id
		for _, member := range candidate.Correspondence.LeftMembers {
			key := "left\x00" + member
			if existing, ok := members[key]; ok && existing != id {
				return fmt.Errorf("left correspondence member %q is used more than once", member)
			}
			members[key] = id
		}
		for _, member := range candidate.Correspondence.RightMembers {
			key := "right\x00" + member
			if existing, ok := members[key]; ok && existing != id {
				return fmt.Errorf("right correspondence member %q is used more than once", member)
			}
			members[key] = id
		}
	}
	return nil
}

func compareJournal(current []Candidate, journal *journalState) []Candidate {
	byID := make(map[string]Candidate, len(current)+len(journal.Candidates))
	for _, candidate := range current {
		saved := journal.Candidates[candidate.CandidateID]
		candidate.JournalRevision = intPointer(journal.Revision)
		if saved == nil {
			candidate.EvidenceStatus = "new"
			candidate.JournalEvidenceDigest = json.RawMessage("null")
			candidate.LastDecision = json.RawMessage("null")
		} else {
			if *candidate.EvidenceDigest == saved.EvidenceDigest {
				candidate.EvidenceStatus = "unchanged"
			} else {
				candidate.EvidenceStatus = "changed"
			}
			candidate.JournalEvidenceDigest = marshalRaw(saved.EvidenceDigest)
			candidate.LastDecision = marshalRaw(saved.LastDecision)
		}
		byID[candidate.CandidateID] = candidate
	}
	for id, saved := range journal.Candidates {
		if _, present := byID[id]; present {
			continue
		}
		candidate := Candidate{
			CandidateID:           id,
			CandidateKey:          saved.CandidateKey,
			Evidence:              []Evidence{},
			Left:                  []PathRecord{},
			Right:                 []PathRecord{},
			Confidence:            "not-assessed",
			EvidenceStatus:        "out-of-scope",
			JournalRevision:       intPointer(journal.Revision),
			JournalEvidenceDigest: marshalRaw(saved.EvidenceDigest),
			LastDecision:          marshalRaw(saved.LastDecision),
		}
		byID[id] = candidate
	}
	result := make([]Candidate, 0, len(byID))
	for _, candidate := range byID {
		result = append(result, candidate)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CandidateID < result[j].CandidateID })
	return result
}

func payloadFieldsAgree(request, payload json.RawMessage, fields ...string) error {
	requestFields, err := rawObject(request)
	if err != nil {
		return err
	}
	payloadFields, err := rawObject(payload)
	if err != nil {
		return err
	}
	for _, name := range fields {
		a, requestHas := requestFields[name]
		b, payloadHas := payloadFields[name]
		if requestHas != payloadHas {
			return fmt.Errorf("request/payload field %q is omitted or added", name)
		}
		if requestHas && !jsonValuesEqual(a, b) {
			return fmt.Errorf("request and payload field %q disagree", name)
		}
	}
	return nil
}

func validateOutcomeFields(request, payload json.RawMessage) error {
	requestFields, err := rawObject(request)
	if err != nil {
		return err
	}
	payloadFields, err := rawObject(payload)
	if err != nil {
		return err
	}
	requestOutcome, err := outcomeFieldSet(requestFields["outcome"])
	if err != nil {
		return err
	}
	payloadOutcome, err := outcomeFieldSet(payloadFields["outcome"])
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(requestOutcome, payloadOutcome) {
		return errors.New("request and payload outcome fields disagree")
	}
	return nil
}

func outcomeFieldSet(data json.RawMessage) (map[string]bool, error) {
	fields, err := rawObject(data)
	if err != nil {
		return nil, err
	}
	stateRaw, ok := fields["state"]
	if !ok {
		return nil, errors.New("outcome state is required")
	}
	var state string
	if err := json.Unmarshal(stateRaw, &state); err != nil {
		return nil, errors.New("outcome state must be a string")
	}
	allowed := map[string]bool{"state": true}
	switch state {
	case "unresolved", "divergence-retained":
	case "intent":
		allowed["direction"] = true
	case "deferred":
		allowed["trigger"] = true
	case "applied":
		allowed["direction"] = true
		allowed["appliedProof"] = true
	default:
		return nil, fmt.Errorf("unknown outcome state %q", state)
	}
	if len(fields) != len(allowed) {
		return nil, errors.New("outcome contains fields not allowed for its state")
	}
	for name := range allowed {
		if _, ok := fields[name]; !ok {
			return nil, fmt.Errorf("outcome state %q requires field %q", state, name)
		}
	}
	return allowed, nil
}

func optionalReplacementGroup(data json.RawMessage) error {
	fields, err := rawObject(data)
	if err != nil {
		return err
	}
	present := 0
	for _, name := range []string{"correspondence", "provenance", "assessment"} {
		if value, ok := fields[name]; ok {
			present++
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return errors.New("reconsider replacement fields cannot be null")
			}
		}
	}
	if present != 0 && present != 3 {
		return errors.New("reconsider replacement fields are all-or-none")
	}
	return nil
}

func rawObject(data json.RawMessage) (map[string]json.RawMessage, error) {
	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil || result == nil {
		return nil, errors.New("expected JSON object")
	}
	return result, nil
}

func jsonValuesEqual(left, right json.RawMessage) bool {
	var a, b any
	if json.Unmarshal(left, &a) != nil || json.Unmarshal(right, &b) != nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

func validCandidateKey(value string) bool {
	if !strings.HasPrefix(value, reservedCandidatePrefix) {
		return validIdentifier(value)
	}
	for _, side := range []string{"left", "right"} {
		prefix := reservedCandidatePrefix + side + ":"
		if strings.HasPrefix(value, prefix) {
			return validateRepoPath(strings.TrimPrefix(value, prefix)) == nil
		}
	}
	return false
}

func validTrust(value string) bool {
	switch value {
	case "trusted", "bounded", "untrusted", "contradictory", "synthetic":
		return true
	default:
		return false
	}
}

func validDirection(value string) bool {
	return value == "left-to-right" || value == "right-to-left"
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func validUTC(value string) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return false
	}
	_, offset := parsed.Zone()
	return offset == 0
}

func isEvidenceDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && isLowerHex(strings.TrimPrefix(value, "sha256:"), 64)
}

func isCandidateID(value string) bool {
	return strings.HasPrefix(value, "cand-v1-") && isLowerHex(strings.TrimPrefix(value, "cand-v1-"), 64)
}

func isLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func marshalRaw(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("null")
	}
	return data
}

func intPointer(value int) *int { return &value }
