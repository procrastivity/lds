package adoption

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"syscall"
)

// ApplyPath reports the observed bytes at one approved target path.
type ApplyPath struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
}

// ApplyFailure describes the first failed creation, if any.
type ApplyFailure struct {
	Path    string `json:"path"`
	SHA256  string `json:"sha256,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ApplyResult reports observed effects; it makes no transactionality claim.
type ApplyResult struct {
	SchemaVersion      int             `json:"schemaVersion"`
	Operation          string          `json:"operation"`
	PlanDigest         string          `json:"planDigest"`
	Status             string          `json:"status"`
	Created            []ApplyPath     `json:"created"`
	IdenticalUntouched []ApplyPath     `json:"identicalUntouched"`
	Unattempted        []ApplyPath     `json:"unattempted"`
	Failed             *ApplyFailure   `json:"failed"`
	Preserved          []PreservedPath `json:"preserved"`
}

// ApplyError carries the actual-effects result for policy refusal or I/O
// failure. Callers should emit Result before rendering the error envelope.
type ApplyError struct {
	Code     string
	Message  string
	Result   ApplyResult
	Internal bool
}

func (e *ApplyError) Error() string { return e.Message }

var createOutputFile = createExclusive

type createTarget struct {
	root            *os.File
	rootPath        string
	rootIdentity    string
	existingParents map[string]string
	missingParents  map[string]bool
	createdParents  map[string]string
}

// Apply consumes a reviewed plan and creates only its missing approved files.
// It re-plans before any write and refuses when any displayed input changed.
func Apply(root, factsPath, planPath, confirmation string) (ApplyResult, error) {
	data, err := os.ReadFile(planPath)
	if err != nil {
		return refusedApplyResult(PlanResult{}, "refusal.plan-invalid", "reviewed plan cannot be read")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return refusedApplyResult(PlanResult{}, "refusal.plan-invalid", "reviewed plan contains duplicate JSON keys")
	}
	var reviewed PlanResult
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&reviewed); err != nil || requireJSONEOF(dec) != nil || reviewed.SchemaVersion != 1 || reviewed.Operation != "docs plan" || reviewed.PlanDigest == nil {
		return refusedApplyResult(reviewed, "refusal.plan-invalid", "reviewed plan is not a complete schemaVersion 1 plan")
	}
	calculated, err := displayedPlanDigest(data)
	if err != nil || calculated != *reviewed.PlanDigest {
		return refusedApplyResult(reviewed, "refusal.plan-invalid", "reviewed plan digest does not match its displayed fields")
	}
	if confirmation != *reviewed.PlanDigest {
		return refusedApplyResult(reviewed, "refusal.confirmation-declined", "explicit digest confirmation does not match the reviewed plan")
	}
	if reviewed.Veto {
		return refusedApplyResult(reviewed, "refusal.plan-veto", "a vetoed plan cannot be applied")
	}

	current, planErr := Plan(root, factsPath)
	if planErr != nil || current.PlanDigest == nil || *current.PlanDigest != *reviewed.PlanDigest {
		return refusedApplyWithObserved(reviewed, current, "refusal.plan-stale", "target, facts, assets, or diagnostics changed since review")
	}
	rootDir, err := openRoot(root)
	if err != nil {
		return refusedApplyResult(reviewed, "refusal.unsafe-target", "target root cannot be opened safely")
	}
	defer func() { _ = rootDir.Close() }()
	identity, err := rootIdentity(rootDir)
	if err != nil || identity != reviewed.TargetIdentity {
		return refusedApplyWithObserved(reviewed, current, "refusal.plan-stale", "target identity changed since review")
	}

	result := ApplyResult{
		SchemaVersion: 1, Operation: "docs apply", PlanDigest: *reviewed.PlanDigest,
		Status: "applied", Created: []ApplyPath{}, IdenticalUntouched: []ApplyPath{},
		Unattempted: []ApplyPath{}, Preserved: append([]PreservedPath{}, reviewed.Preserved...),
	}
	target := &createTarget{
		root: rootDir, rootPath: root, rootIdentity: reviewed.TargetIdentity,
		existingParents: map[string]string{}, missingParents: map[string]bool{}, createdParents: map[string]string{},
	}
	for _, parent := range reviewed.ExistingParentDirectories {
		target.existingParents[parent.Path] = parent.Identity
	}
	for _, parent := range reviewed.NecessaryParentDirectories {
		target.missingParents[parent] = true
	}
	for _, output := range reviewed.Outputs {
		switch output.Classification {
		case "identical-untouched":
			result.IdenticalUntouched = append(result.IdenticalUntouched, ApplyPath{Path: output.Path, SHA256: output.ExistingSHA256})
		case "create":
			if err := createOutputFile(target, output.Path, []byte(output.Content)); err != nil {
				code := "internal.apply-failed"
				switch {
				case errors.Is(err, syscall.EEXIST):
					code = "refusal.output-collision"
				case errors.Is(err, ErrUnsafeTarget):
					code = "refusal.plan-stale"
				}
				result.Status = "refused"
				if len(result.Created) > 0 {
					result.Status = "partial"
				}
				result.Failed = &ApplyFailure{Path: output.Path, Code: code, Message: "exclusive creation failed; no rollback was attempted"}
				if observed, readErr := probeTargetFile(rootDir, output.Path); readErr == nil && observed.exists {
					result.Failed.Message = "creation failed; observed occupied bytes were preserved"
					result.Failed.SHA256 = sha256Hex(observed.data)
				}
				for _, later := range reviewed.Outputs {
					if later.Path > output.Path && later.Classification == "create" {
						result.Unattempted = append(result.Unattempted, ApplyPath{Path: later.Path})
					}
				}
				sortApply(&result)
				return result, &ApplyError{Code: code, Message: "exclusive creation failed; result reports observed effects", Result: result, Internal: code == "internal.apply-failed"}
			}
			observed, err := probeTargetFile(rootDir, output.Path)
			if err != nil || !observed.exists || !bytes.Equal(observed.data, []byte(output.Content)) {
				result.Status = "partial"
				result.Failed = &ApplyFailure{Path: output.Path, Code: "apply.verify-failed", Message: "created path did not match reviewed bytes"}
				if err == nil && observed.exists {
					result.Failed.SHA256 = sha256Hex(observed.data)
				}
				for _, later := range reviewed.Outputs {
					if later.Path > output.Path && later.Classification == "create" {
						result.Unattempted = append(result.Unattempted, ApplyPath{Path: later.Path})
					}
				}
				sortApply(&result)
				return result, &ApplyError{Code: "internal.apply-failed", Message: "created path failed post-write verification", Result: result, Internal: true}
			}
			result.Created = append(result.Created, ApplyPath{Path: output.Path, SHA256: sha256Hex(observed.data)})
		default:
			return refusedApplyResult(reviewed, "refusal.plan-veto", "reviewed plan contains a non-applicable output classification")
		}
	}
	if len(result.Created) == 0 {
		result.Status = "current"
	}
	sortApply(&result)
	return result, nil
}

// Digest the raw displayed JSON, not the decoded struct: encoding/json accepts
// case-insensitive field aliases and can otherwise silently normalize a
// review artifact before checking what the human confirmed.
func displayedPlanDigest(data []byte) (string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return "", err
	}
	delete(fields, "planDigest")
	projection, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalJSON(projection)
	if err != nil {
		return "", err
	}
	return "sha256:" + sha256Hex(canonical), nil
}

func refusedApplyWithObserved(reviewed, current PlanResult, code, message string) (ApplyResult, error) {
	result, _ := refusedApplyResult(reviewed, code, message)
	// A stale plan's former preserved hashes are not observations. If a fresh
	// preflight completed, report its actual readable preserved file hashes.
	result.Preserved = append([]PreservedPath{}, current.Preserved...)
	result.IdenticalUntouched = []ApplyPath{}
	return result, &ApplyError{Code: code, Message: message, Result: result}
}

func refusedApplyResult(plan PlanResult, code, message string) (ApplyResult, error) {
	result := ApplyResult{SchemaVersion: 1, Operation: "docs apply", PlanDigest: "", Status: "refused", Created: []ApplyPath{}, IdenticalUntouched: []ApplyPath{}, Unattempted: []ApplyPath{}, Preserved: []PreservedPath{}}
	if plan.PlanDigest != nil {
		result.PlanDigest = *plan.PlanDigest
	}
	for _, output := range plan.Outputs {
		switch output.Classification {
		case "create":
			result.Unattempted = append(result.Unattempted, ApplyPath{Path: output.Path})
		}
	}
	sortApply(&result)
	return result, &ApplyError{Code: code, Message: message, Result: result}
}

func sortApply(result *ApplyResult) {
	sort.Slice(result.Created, func(i, j int) bool { return result.Created[i].Path < result.Created[j].Path })
	sort.Slice(result.IdenticalUntouched, func(i, j int) bool { return result.IdenticalUntouched[i].Path < result.IdenticalUntouched[j].Path })
	sort.Slice(result.Unattempted, func(i, j int) bool { return result.Unattempted[i].Path < result.Unattempted[j].Path })
	sort.Slice(result.Preserved, func(i, j int) bool { return result.Preserved[i].Path < result.Preserved[j].Path })
}
