package adoption

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/procrastivity/lds/internal/ldserr"
)

// UpdatePath reports observed metadata, not a promise of preservation.
type UpdatePath struct {
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	Identity string `json:"identity"`
	Mode     uint32 `json:"mode"`
	UID      uint32 `json:"uid"`
	GID      uint32 `json:"gid"`
}

// UpdateFailure identifies the stopped phase and any readable actual state.
type UpdateFailure struct {
	Path     string      `json:"path"`
	Phase    string      `json:"phase"`
	Code     string      `json:"code"`
	Message  string      `json:"message"`
	Observed *UpdatePath `json:"observed"`
}

// UpdateResult reports actual effects without implying rollback or safe retry.
type UpdateResult struct {
	SchemaVersion int            `json:"schemaVersion"`
	Operation     string         `json:"operation"`
	PlanDigest    string         `json:"planDigest"`
	Status        string         `json:"status"`
	Replaced      []UpdatePath   `json:"replaced"`
	Untouched     []UpdatePath   `json:"untouched"`
	Unattempted   []string       `json:"unattempted"`
	Failed        *UpdateFailure `json:"failed"`
	Scratch       string         `json:"scratch"`
	CleanupError  string         `json:"cleanupError"`
}

// ApplyUpdates consumes the exact displayed plan, not an assessment report or
// an adoption plan. Confirmation compares strings; it cannot authenticate a human.
func ApplyUpdates(root, planPath, confirmation string) (UpdateResult, error) {
	var plan UpdatePlan
	if err := readUpdateJSON(planPath, &plan); err != nil {
		return updateFailure(root, UpdatePlan{}, 0, "", UpdateReplacement{Phase: "plan"}, "refusal.update-plan-invalid", err.Error(), nil)
	}
	return applyUpdatesReviewed(root, plan, confirmation)
}

func applyUpdatesReviewed(root string, plan UpdatePlan, confirmation string) (UpdateResult, error) {
	result := newUpdateResult(plan)
	if plan.SchemaVersion != 1 || plan.Operation != "docs update-plan" || len(plan.Edits) == 0 || len(plan.Edits) > 32 || !strings.HasPrefix(plan.PlanDigest, "sha256:") {
		return updateFailure(root, UpdatePlan{}, 0, "", UpdateReplacement{Phase: "plan"}, "refusal.update-plan-invalid", "expected a complete schemaVersion 1 docs update-plan", nil)
	}
	// Re-encoding is safe only after strictUpdateJSON has required exact raw
	// field names, types, Unicode and equality with the typed artifact.
	data, err := json.Marshal(plan)
	if err != nil {
		return result, err
	}
	digest, err := displayedPlanDigest(data)
	if err != nil || digest != plan.PlanDigest {
		return updateFailure(root, plan, 0, "", UpdateReplacement{Phase: "plan"}, "refusal.update-plan-invalid", "plan digest does not match every displayed field", nil)
	}
	if confirmation != plan.PlanDigest {
		return updateFailure(root, plan, 0, "", UpdateReplacement{Phase: "confirmation"}, "refusal.update-confirmation", "fresh exact digest confirmation is required", nil)
	}
	if plan.Veto {
		return updateFailure(root, plan, 0, "", UpdateReplacement{Phase: "plan"}, "refusal.update-veto", "a vetoed plan cannot be applied", nil)
	}
	request := UpdateRequest{SchemaVersion: 1, Edits: []UpdateProposal{}}
	for _, edit := range plan.Edits {
		request.Edits = append(request.Edits, edit.UpdateProposal)
	}
	current, err := PlanUpdatesReviewed(root, plan.Observation, request)
	if err != nil || current.PlanDigest != plan.PlanDigest {
		return updateFailure(root, plan, 0, "", UpdateReplacement{Phase: "preflight"}, "refusal.update-stale", "original observation, selected metadata or displayed plan changed; collect and review a new plan", nil)
	}
	completed := map[string]UpdateFileState{}
	for i, edit := range plan.Edits {
		recheck := func() error { return recheckUpdate(root, plan, completed) }
		if err := recheck(); err != nil {
			return updateFailure(root, plan, i, edit.Path, UpdateReplacement{Phase: "recheck"}, "refusal.update-stale", err.Error(), result.Replaced)
		}
		actual, err := replaceUpdateFile(root, edit.Path, edit.Before, edit.After, edit.Ancestors, recheck)
		if err != nil {
			code := "internal.update-apply"
			var structured *ldserr.Error
			if errors.Is(err, ErrUnsafeTarget) || errors.As(err, &structured) && strings.HasPrefix(structured.Code, "refusal.") {
				code = "refusal.update-stale"
			}
			return updateFailure(root, plan, i+1, edit.Path, actual, code, err.Error(), result.Replaced)
		}
		if !actual.Renamed || actual.Observed == nil {
			return updateFailure(root, plan, i+1, edit.Path, actual, "internal.update-apply", "replacement did not return a verified observation", result.Replaced)
		}
		completed[edit.Path] = *actual.Observed
		result.Replaced = append(result.Replaced, updatePath(edit.Path, *actual.Observed))
		// Includes the last replacement: drift in earlier edited files, sources
		// or diagnostics after a rename is not a successful applied result.
		if err := recheck(); err != nil {
			actual.Phase = "post-recheck"
			return updateFailure(root, plan, i+1, edit.Path, actual, "refusal.update-stale", err.Error(), result.Replaced)
		}
	}
	result.Status = "applied"
	return result, nil
}

var replaceUpdateFile = updateReplace

func recheckUpdate(root string, plan UpdatePlan, completed map[string]UpdateFileState) error {
	root, err := filepath.Abs(root)
	if err != nil || root != plan.TargetRoot {
		return ldserr.New("refusal.update-stale", "target root changed")
	}
	overlay := map[string]string{}
	expectedHash := updateObservationHash(plan.Observation)
	for _, edit := range plan.Edits {
		expected := edit.Before
		if state, done := completed[edit.Path]; done {
			expected = state
			overlay[edit.Path] = edit.After
			expectedHash = edit.AfterObservationSHA256
		}
		actual, parents, err := updateRead(root, edit.Path)
		if err != nil || actual != expected || !sameUpdateJSON(append([]ParentIdentity{}, parents...), edit.Ancestors) {
			return ldserr.New("refusal.update-stale", "selected file, metadata or ancestors changed: "+edit.Path)
		}
	}
	actual, err := updateObservation(root, plan.Observation, overlay, false)
	if err != nil || updateObservationHash(actual) != expectedHash {
		return ldserr.New("refusal.update-stale", "complete original observations differ beyond previously completed edits")
	}
	return nil
}

func newUpdateResult(plan UpdatePlan) UpdateResult {
	return UpdateResult{SchemaVersion: 1, Operation: "docs update-apply", PlanDigest: plan.PlanDigest, Status: "refused", Replaced: []UpdatePath{}, Untouched: []UpdatePath{}, Unattempted: []string{}}
}

func updatePath(path string, state UpdateFileState) UpdatePath {
	return UpdatePath{Path: path, SHA256: state.SHA256, Identity: state.Identity, Mode: state.Mode, UID: state.UID, GID: state.GID}
}

func updateFailure(root string, plan UpdatePlan, next int, failedPath string, actual UpdateReplacement, code, message string, replaced []UpdatePath) (UpdateResult, error) {
	result := newUpdateResult(plan)
	result.Replaced = append(result.Replaced, replaced...)
	result.Failed = &UpdateFailure{Path: failedPath, Phase: actual.Phase, Code: code, Message: message}
	result.Scratch, result.CleanupError = actual.Scratch, actual.CleanupError
	uncertain := actual.Renamed || actual.Phase == "rename-uncertain"
	if actual.Observed != nil {
		observed := updatePath(failedPath, *actual.Observed)
		result.Failed.Observed = &observed
	}
	if len(result.Replaced) > 0 || result.Scratch != "" {
		result.Status = "partial"
	}
	if uncertain {
		result.Status = "indeterminate"
	}
	for i, edit := range plan.Edits {
		if i >= next {
			result.Unattempted = append(result.Unattempted, edit.Path)
		}
		alreadyReplaced := false
		for _, done := range result.Replaced {
			alreadyReplaced = alreadyReplaced || done.Path == edit.Path
		}
		if alreadyReplaced || uncertain && edit.Path == failedPath || !updateDocPath(edit.Path) {
			continue
		}
		// Read now, not from the former plan. Unreadable or changed entries
		// are never reported as untouched, even on a preflight refusal.
		state, _, err := updateRead(root, edit.Path)
		if state.Identity != "" && result.Failed.Observed == nil && (edit.Path == failedPath || failedPath == "" && state != edit.Before) {
			observed := updatePath(edit.Path, state)
			result.Failed.Path, result.Failed.Observed = edit.Path, &observed
		}
		if err == nil && state == edit.Before {
			result.Untouched = append(result.Untouched, updatePath(edit.Path, state))
		}
	}
	return result, ldserr.New(code, message+"; inspect actual effects before making a new plan; no rollback or retry was attempted")
}
