package reconcile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ReportReviewed validates unverified advice against the original collected
// baseline and publishes a private report. It does not contact a provider.
func ReportReviewed(input ReportInput, observed ReportEvidence, draft []byte, assistance ReportAssistance) (ReportResult, error) {
	if err := validateReportAssistance(assistance); err != nil {
		return ReportResult{}, reportError("validation.reconcile-report-assistance", err.Error())
	}
	// Detach all slices, maps, pointers and RawMessages before enriching the
	// report. The caller's observed baseline is never used as an output buffer.
	encoded, err := json.Marshal(observed)
	if err != nil {
		return ReportResult{}, reportError("validation.reconcile-report-baseline", err.Error())
	}
	var baseline ReportEvidence
	if err := json.Unmarshal(encoded, &baseline); err != nil {
		return ReportResult{}, reportError("validation.reconcile-report-baseline", err.Error())
	}
	if err := RecheckReport(input, baseline); err != nil {
		return ReportResult{}, err
	}
	advice, err := decodeReportDraft(draft, baseline)
	if err != nil {
		return ReportResult{}, err
	}
	output := baseline // share only with our detached local copy
	output.Assistance = &assistance
	output.Advice = &advice
	output.Excluded = append([]string{}, baseline.Excluded...)
	for i, exclusion := range output.Excluded {
		if exclusion == "Provider assistance and source/journal writes." {
			output.Excluded[i] = "Provider verification and source/journal writes."
		}
	}
	data := renderReport(output, input.OutputPath)
	// Recheck the untouched original, never the augmented appendix or a
	// freshly substituted baseline, immediately before private publication.
	if err := RecheckReport(input, baseline); err != nil {
		return ReportResult{}, err
	}
	if err := publishReport(input.OutputPath, baseline.OutputParentIdentity, data); err != nil {
		return ReportResult{}, reportFailure(err)
	}
	return ReportResult{
		SchemaVersion: 1, Operation: "reconcile report", Status: "review-required",
		ReportPath: input.OutputPath, ReportSHA256: digest(data), PairID: baseline.Scan.PairID,
		Snapshots: baseline.Scan.Snapshots, Candidates: len(baseline.Scan.Candidates), Assistance: true,
	}, nil
}

func validateReportAssistance(value ReportAssistance) error {
	if value.Provider != "OpenCode Go" || value.Model != "deepseek-v4-flash" {
		return fmt.Errorf("unsupported provider or model")
	}
	switch value.FlowSource {
	case "project", "override", "default", "embedded":
	default:
		return fmt.Errorf("invalid flow source")
	}
	if !isLowerHex(value.FlowSHA256, 64) || !isLowerHex(value.SystemSHA256, 64) || !isLowerHex(value.UserSHA256, 64) || value.SystemBytes <= 0 || value.UserBytes <= 0 || value.SystemBytes > reportTextLimit || value.UserBytes > reportTextLimit-value.SystemBytes {
		return fmt.Errorf("invalid transfer digest or size")
	}
	if _, err := time.Parse(time.RFC3339, value.ConsentAt); err != nil {
		return fmt.Errorf("consent time must be RFC3339")
	}
	return nil
}

func decodeReportDraft(data []byte, observed ReportEvidence) (ReportDraft, error) {
	var draft ReportDraft
	if len(data) > reportFileLimit || len(data) == 0 {
		return draft, reportError("validation.reconcile-report-draft", "draft must be at most 64 KiB")
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return draft, reportError("validation.reconcile-report-draft", "draft must be an object")
	}
	if err := decodeStrictJSON(data, &draft); err != nil {
		return draft, reportError("validation.reconcile-report-draft", err.Error())
	}
	if draft.SchemaVersion != 1 || draft.Claims == nil || draft.Questions == nil {
		return draft, reportError("validation.reconcile-report-draft", "draft requires schemaVersion 1 and non-null claims/questions arrays")
	}
	for _, claim := range draft.Claims {
		if claim.Status != "proposed" || claim.Assessment.Code != "needs-review" {
			return draft, reportError("validation.reconcile-report-draft", "model claims must be proposed and needs-review")
		}
		for _, citation := range claim.Provenance {
			selected := false
			for _, content := range observed.Content {
				snapshot := observed.Scan.Snapshots.Left
				if content.Side == "right" {
					snapshot = observed.Scan.Snapshots.Right
				}
				if citation.Kind == "artifact" && citation.ProjectID == snapshot.ProjectID && citation.Revision != nil && *citation.Revision == snapshot.Revision && citation.Path == content.Path && citation.SHA256 != nil && *citation.SHA256 == content.SHA256 && digest([]byte(content.Text)) == content.SHA256 {
					selected = true
					break
				}
			}
			if !selected {
				return draft, reportError("validation.reconcile-report-draft", "model citation must name explicitly selected captured content")
			}
		}
	}
	brief := observed.Review
	brief.Claims = draft.Claims // validate separately so human assertions remain independent
	if err := validateReview(brief, observed.Scan); err != nil {
		return draft, reportError("validation.reconcile-report-draft", err.Error())
	}
	for _, question := range draft.Questions {
		if strings.TrimSpace(question) == "" {
			return draft, reportError("validation.reconcile-report-draft", "model questions cannot be blank")
		}
	}
	return draft, nil
}
