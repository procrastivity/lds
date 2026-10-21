package adoption

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/procrastivity/lds/internal/ldserr"
)

// CollectAssessment captures the explicit review scope without a report or
// authored advice. Hashes are calculated locally; source paths are never inferred.
func CollectAssessment(root, reason, audience string, sources []string) (AssessmentResult, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return AssessmentResult{}, err
	}
	brief := AssessmentBrief{
		SchemaVersion: 1, Reason: reason, Audience: audience,
		Evidence: []AssessmentSource{}, Proposals: []AssessmentProposal{}, Questions: []string{},
	}
	for _, name := range sources {
		brief.Evidence = append(brief.Evidence, AssessmentSource{Path: name, SHA256: strings.Repeat("0", 64)})
	}
	data, err := json.Marshal(brief)
	if err != nil {
		return AssessmentResult{}, err
	}
	if _, err := decodeAssessmentBrief(data); err != nil {
		return AssessmentResult{}, ldserr.New("validation.assessment-brief", err.Error())
	}
	dir, err := openRoot(root)
	if err != nil {
		return AssessmentResult{}, err
	}
	defer func() { _ = dir.Close() }()
	for i, source := range brief.Evidence {
		probe, err := probeTargetFile(dir, source.Path)
		if errors.Is(err, syscall.ENOTDIR) {
			return AssessmentResult{}, fmt.Errorf("%w: source must be a regular file: %s", ErrUnsafeTarget, source.Path)
		}
		if err != nil {
			return AssessmentResult{}, err
		}
		if !probe.exists {
			return AssessmentResult{}, ldserr.New("validation.assessment-source-missing", "explicit source is missing: "+source.Path)
		}
		brief.Evidence[i].SHA256 = sha256Hex(probe.data)
	}
	data, err = json.Marshal(brief)
	if err != nil {
		return AssessmentResult{}, err
	}
	result, err := collectAssessment(root, "", data, brief)
	if err != nil {
		return AssessmentResult{}, err
	}
	if err := RecheckAssessment(root, result); err != nil {
		return AssessmentResult{}, err
	}
	return result, nil
}

// RecheckAssessment refuses any change to the complete collected observations.
// A supplied observation packet is evidence, not authenticated authority.
func RecheckAssessment(root string, observed AssessmentResult) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	data, err := json.Marshal(observed.Brief)
	if err != nil {
		return err
	}
	brief, err := decodeAssessmentBrief(data)
	if err != nil || observed.SchemaVersion != 1 || observed.Operation != "docs assess" || observed.Status != "review-required" || observed.ReportPath != "" || len(brief.Proposals) != 0 || len(brief.Questions) != 0 || observed.BriefSHA256 != sha256Hex(data) {
		return ldserr.New("validation.assessment-observed", "expected an unchanged version-1 docs collect packet without authored advice or a report")
	}
	current, err := collectAssessment(root, "", data, brief)
	if err != nil {
		return err
	}
	before, err := json.Marshal(observed)
	if err != nil {
		return err
	}
	after, err := json.Marshal(current)
	if err != nil {
		return err
	}
	if !bytes.Equal(before, after) {
		return ldserr.New("refusal.assessment-stale", "collected observations changed; start a new assessment without reusing the draft")
	}
	return nil
}

// AssessReviewed validates an untrusted draft against the original read scope,
// rechecks the full baseline, and exclusively creates one external report.
func AssessReviewed(root string, observed AssessmentResult, draft []byte, reportPath string) (AssessmentResult, error) {
	if err := CheckAssessmentOutput(root, reportPath); err != nil {
		return AssessmentResult{}, err
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return AssessmentResult{}, err
	}
	brief, err := decodeAssessmentBrief(draft)
	if err != nil {
		return AssessmentResult{}, ldserr.New("validation.assessment-brief", err.Error())
	}
	scope := brief
	scope.Proposals, scope.Questions = []AssessmentProposal{}, []string{}
	actualScope, err := json.Marshal(scope)
	if err != nil {
		return AssessmentResult{}, err
	}
	expectedScope, err := json.Marshal(observed.Brief)
	if err != nil {
		return AssessmentResult{}, err
	}
	if !bytes.Equal(actualScope, expectedScope) {
		return AssessmentResult{}, ldserr.New("refusal.assessment-scope-changed", "draft changed the reason, audience or explicit source scope; no report was created")
	}
	if err := RecheckAssessment(root, observed); err != nil {
		return AssessmentResult{}, err
	}
	result, err := collectAssessment(root, filepath.Clean(reportPath), draft, brief)
	if err != nil {
		return AssessmentResult{}, err
	}
	if err := RecheckAssessment(root, observed); err != nil {
		return AssessmentResult{}, err
	}
	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return AssessmentResult{}, err
	}
	if err := writeAssessmentReport(result.ReportPath, result.TargetIdentity, assessmentMarkdown(result, payload)); err != nil {
		return AssessmentResult{}, err
	}
	return result, nil
}

// AssessWithObservation is the file-based plumbing equivalent of AssessReviewed.
func AssessWithObservation(root, briefPath, observedPath, reportPath string) (AssessmentResult, error) {
	draft, err := readAssessmentInput(briefPath)
	if err != nil {
		return AssessmentResult{}, err
	}
	data, err := readAssessmentInput(observedPath)
	if err != nil {
		return AssessmentResult{}, err
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return AssessmentResult{}, ldserr.New("validation.assessment-observed", err.Error())
	}
	var observed AssessmentResult
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&observed); err != nil {
		return AssessmentResult{}, ldserr.New("validation.assessment-observed", err.Error())
	}
	return AssessReviewed(root, observed, draft, reportPath)
}

// CheckAssessmentOutput is a no-write preflight, not a reservation. The final
// exclusive open must still refuse a path occupied after this check.
func CheckAssessmentOutput(root, reportPath string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(reportPath) {
		return ldserr.New("validation.assessment-output", "--output must name an absolute report path outside the target")
	}
	reportPath = filepath.Clean(reportPath)
	relative, err := filepath.Rel(root, reportPath)
	if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return ldserr.New("refusal.assessment-output", "report must be outside the target; no target writes are allowed")
	}
	parent, err := openRoot(filepath.Dir(reportPath))
	if err != nil {
		return err
	}
	defer func() { _ = parent.Close() }()
	file, err := openChild(parent, filepath.Base(reportPath))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_ = file.Close()
	return ldserr.New("refusal.assessment-output", "report path is occupied; choose a new path, existing bytes were not replaced")
}
