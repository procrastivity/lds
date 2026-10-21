package adoption

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/procrastivity/lds/internal/ldserr"
)

const updateInputLimit = 32 << 20

// UpdateRequest explicitly selects existing files and their complete new bytes.
// Citations explain the request; they do not authorize additional reads/writes.
type UpdateRequest struct {
	SchemaVersion int              `json:"schemaVersion"`
	Edits         []UpdateProposal `json:"edits"`
}

// UpdateProposal is a caller-authored replacement, not verified prose.
type UpdateProposal struct {
	Path      string   `json:"path"`
	After     string   `json:"after"`
	Rationale string   `json:"rationale"`
	Citations []string `json:"citations"`
}

// UpdateEdit binds content, metadata and the complete expected observation
// after this entry and all earlier entries in stable path order.
type UpdateEdit struct {
	UpdateProposal
	Before                 UpdateFileState  `json:"before"`
	AfterSHA256            string           `json:"afterSha256"`
	AfterBytes             int              `json:"afterBytes"`
	Ancestors              []ParentIdentity `json:"ancestors"`
	AfterObservationSHA256 string           `json:"afterObservationSha256"`
}

// UpdatePlan is the complete digest-bound artifact requiring human review.
type UpdatePlan struct {
	SchemaVersion  int              `json:"schemaVersion"`
	Operation      string           `json:"operation"`
	TargetRoot     string           `json:"targetRoot"`
	TargetIdentity string           `json:"targetIdentity"`
	Observation    AssessmentResult `json:"observation"`
	Edits          []UpdateEdit     `json:"edits"`
	Veto           bool             `json:"veto"`
	Findings       []Finding        `json:"findings"`
	Skipped        []Skip           `json:"skipped"`
	PlanDigest     string           `json:"planDigest"`
}

// PlanUpdates reads JSON inputs and returns a read-only exact replacement plan.
func PlanUpdates(root, observedPath, requestPath string) (UpdatePlan, error) {
	var observed AssessmentResult
	if err := readUpdateJSON(observedPath, &observed); err != nil {
		return UpdatePlan{}, ldserr.New("validation.update-observed", err.Error())
	}
	var request UpdateRequest
	if err := readUpdateJSON(requestPath, &request); err != nil {
		return UpdatePlan{}, ldserr.New("validation.update-request", err.Error())
	}
	return PlanUpdatesReviewed(root, observed, request)
}

// PlanUpdatesReviewed is shared with apply's original-baseline recheck. It
// never substitutes a newly collected baseline for the supplied observation.
func PlanUpdatesReviewed(root string, observed AssessmentResult, request UpdateRequest) (UpdatePlan, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return UpdatePlan{}, err
	}
	if request.SchemaVersion != 1 || len(request.Edits) == 0 || len(request.Edits) > 32 {
		return UpdatePlan{}, ldserr.New("validation.update-request", "expected schemaVersion 1 and 1..32 explicit edits")
	}
	if err := RecheckAssessment(root, observed); err != nil {
		return UpdatePlan{}, ldserr.New("refusal.update-stale", "original collected observation is invalid or stale: "+err.Error())
	}
	plan := UpdatePlan{
		SchemaVersion: 1, Operation: "docs update-plan", TargetRoot: root, TargetIdentity: observed.TargetIdentity,
		Observation: observed, Edits: []UpdateEdit{},
		Findings: append([]Finding{}, observed.Validation.Findings...), Skipped: append([]Skip{}, observed.Validation.Skipped...),
	}
	evidence := map[string]AssessmentEvidence{}
	for _, entry := range observed.Evidence {
		evidence[entry.Path] = entry
	}
	selected := map[string]bool{}
	proposals := append([]UpdateProposal{}, request.Edits...)
	sort.Slice(proposals, func(i, j int) bool { return proposals[i].Path < proposals[j].Path })
	for _, proposal := range proposals {
		if selected[proposal.Path] || !assessmentText(proposal.Rationale) || len(proposal.Citations) == 0 || !utf8.ValidString(proposal.After) || len(proposal.After) > 8<<20 {
			return UpdatePlan{}, ldserr.New("validation.update-request", "edits require unique paths, rationale, citations and UTF-8 after bytes (maximum 8 MiB)")
		}
		selected[proposal.Path] = true
		proposal.Citations = append([]string{}, proposal.Citations...)
		sort.Strings(proposal.Citations)
		edit := UpdateEdit{UpdateProposal: proposal, AfterSHA256: sha256Hex([]byte(proposal.After)), AfterBytes: len(proposal.After), Ancestors: []ParentIdentity{}}
		plan.Edits = append(plan.Edits, edit)
		index := len(plan.Edits) - 1
		if !updateDocPath(proposal.Path) {
			plan.updateVeto(proposal.Path, "selected path is outside the clean relative Markdown scope")
			continue
		}
		validCitations := true
		for i, citation := range proposal.Citations {
			entry, exists := evidence[citation]
			if !exists || !entry.Exists || (i > 0 && citation == proposal.Citations[i-1]) {
				validCitations = false
			}
		}
		entry, exists := evidence[proposal.Path]
		if !exists || !entry.Exists || !validCitations {
			plan.updateVeto(proposal.Path, "selected file and unique citations must exist in the original captured evidence")
			continue
		}
		state, parents, err := updateRead(root, proposal.Path)
		if err != nil {
			plan.updateVeto(proposal.Path, "selected file is not eligible for replacement: "+err.Error())
			continue
		}
		plan.Edits[index].Before, plan.Edits[index].Ancestors = state, append([]ParentIdentity{}, parents...)
		if state.Content != entry.Content || state.SHA256 != entry.SHA256 {
			plan.updateVeto(proposal.Path, "selected bytes differ from the original captured evidence")
		} else if state.Content == proposal.After {
			plan.updateVeto(proposal.Path, "before and after bytes are identical; no replacement is needed")
		}
	}
	if !plan.Veto {
		overlay := map[string]string{}
		for i, edit := range plan.Edits {
			overlay[edit.Path] = edit.After
			preview, err := updateObservation(root, observed, overlay, true)
			if err != nil {
				return UpdatePlan{}, ldserr.New("refusal.update-stale", "cannot preview the original observation: "+err.Error())
			}
			plan.Edits[i].AfterObservationSHA256 = updateObservationHash(preview)
		}
		for _, edit := range plan.Edits {
			state, parents, err := updateRead(root, edit.Path)
			if err != nil || state != edit.Before || !sameUpdateJSON(append([]ParentIdentity{}, parents...), edit.Ancestors) {
				return UpdatePlan{}, ldserr.New("refusal.update-stale", "selected metadata or ancestors changed during planning")
			}
		}
	}
	if err := RecheckAssessment(root, observed); err != nil {
		return UpdatePlan{}, ldserr.New("refusal.update-stale", "original observations changed during planning")
	}
	data, err := json.Marshal(plan)
	if err != nil {
		return UpdatePlan{}, err
	}
	plan.PlanDigest, err = displayedPlanDigest(data)
	if err != nil {
		return UpdatePlan{}, err
	}
	data, err = json.Marshal(plan)
	if err != nil || len(data) > updateInputLimit {
		return UpdatePlan{}, ldserr.New("validation.update-limit", "complete displayed plan exceeds 32 MiB")
	}
	if plan.Veto {
		return plan, ldserr.New("refusal.update-veto", "plan is vetoed; no replacements are authorized")
	}
	return plan, nil
}

func (p *UpdatePlan) updateVeto(path, message string) {
	p.Veto = true
	p.Findings = append(p.Findings, Finding{Code: "refusal.update-ineligible", Path: path, Message: message})
}

func updateDocPath(name string) bool {
	return assessmentDocPath(name) && !strings.ContainsFunc(name, unicode.IsControl)
}

// Preview uses only planned completed bytes. Real rechecks pass no content
// overlay: they read every actual file. Only declared source hashes advance;
// the original brief and every other observation remain digest-bound.
func updateObservation(root string, original AssessmentResult, completed map[string]string, preview bool) (AssessmentResult, error) {
	brief := original.Brief
	brief.Evidence = append([]AssessmentSource{}, brief.Evidence...)
	for i, source := range brief.Evidence {
		if content, ok := completed[source.Path]; ok {
			brief.Evidence[i].SHA256 = sha256Hex([]byte(content))
		}
	}
	data, err := json.Marshal(brief)
	if err != nil {
		return AssessmentResult{}, err
	}
	var overlay map[string]string
	if preview {
		overlay = completed
	}
	result, err := collectAssessmentWithOverlay(root, "", data, brief, overlay)
	result.Brief, result.BriefSHA256 = original.Brief, original.BriefSHA256
	return result, err
}

func updateObservationHash(observed AssessmentResult) string {
	data, _ := json.Marshal(observed)
	return sha256Hex(data)
}

func sameUpdateJSON(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return bytes.Equal(left, right)
}

func readUpdateJSON(path string, value any) error {
	file, err := openRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	input, err := openChild(file, filepath.Base(path))
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > updateInputLimit {
		return errors.New("update input must be a regular JSON file of at most 32 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(input, updateInputLimit+1))
	if err != nil {
		return err
	}
	return strictUpdateJSON(data, value)
}

// Compare raw and typed canonical JSON too: encoding/json's case aliases,
// omitted zero fields and null coercions cannot silently change the artifact.
func strictUpdateJSON(data []byte, value any) error {
	if len(data) > updateInputLimit || !utf8.Valid(data) || !json.Valid(data) {
		return errors.New("expected one UTF-8 JSON value of at most 32 MiB")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	if err := updateJSONUnicode(data); err != nil {
		return err
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if updateContainsNull(raw) {
		return errors.New("update JSON fields cannot be null")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return err
	}
	typed, err := json.Marshal(value)
	if err != nil {
		return err
	}
	rawCanonical, err := canonicalJSON(data)
	if err != nil {
		return err
	}
	typedCanonical, err := canonicalJSON(typed)
	if err != nil || !bytes.Equal(rawCanonical, typedCanonical) {
		return errors.New("JSON must contain exactly the documented fields and types, without null or case aliases")
	}
	return nil
}

func updateContainsNull(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case []any:
		for _, item := range v {
			if updateContainsNull(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range v {
			if updateContainsNull(item) {
				return true
			}
		}
	}
	return false
}

// encoding/json replaces isolated escaped surrogates with U+FFFD. That is
// unsafe for an exact-byte reviewed artifact, even when raw UTF-8 is valid.
func updateJSONUnicode(data []byte) error {
	for i := 0; i < len(data); i++ {
		if data[i] != '\\' {
			continue
		}
		i++
		if i >= len(data) || data[i] != 'u' {
			continue
		}
		n, err := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		if err != nil {
			return err
		}
		i += 4
		if n >= 0xdc00 && n <= 0xdfff {
			return errors.New("isolated low Unicode surrogate")
		}
		if n < 0xd800 || n > 0xdbff {
			continue
		}
		if i+6 >= len(data) || string(data[i+1:i+3]) != `\u` {
			return errors.New("isolated high Unicode surrogate")
		}
		low, err := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return errors.New("invalid Unicode surrogate pair")
		}
		i += 6
	}
	return nil
}
