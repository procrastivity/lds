package adoption

import (
	"unicode"
	"unicode/utf8"

	"github.com/procrastivity/lds/internal/ldserr"
)

// UpdateDraft is untrusted model advice, not a reviewed replacement plan.
// Only explicitly selected, captured Markdown files may appear in Edits.
type UpdateDraft struct {
	SchemaVersion int              `json:"schemaVersion"`
	Edits         []UpdateProposal `json:"edits"`
	Questions     []string         `json:"questions"`
}

// ValidateUpdateSelection checks the caller's explicit scope against the
// original evidence packet. It does not read or infer paths from the target.
func ValidateUpdateSelection(observed AssessmentResult, selected []string) error {
	if len(selected) < 1 || len(selected) > 32 {
		return ldserr.New("validation.update-selection", "select 1..32 explicit Markdown paths")
	}
	evidence := make(map[string]bool, len(observed.Evidence))
	for _, entry := range observed.Evidence {
		if _, exists := evidence[entry.Path]; exists {
			return ldserr.New("validation.update-selection", "ambiguous captured evidence path")
		}
		evidence[entry.Path] = entry.Exists
	}
	seen := make(map[string]bool, len(selected))
	for _, name := range selected {
		if !updateDocPath(name) || seen[name] || !evidence[name] {
			return ldserr.New("validation.update-selection", "selection requires unique eligible paths in the original captured evidence")
		}
		seen[name] = true
	}
	return nil
}

// DecodeUpdateDraft validates bounded untrusted advice without authorizing
// any reads or writes. Planning still checks metadata, freshness and no-ops.
func DecodeUpdateDraft(observed AssessmentResult, selected []string, data []byte) (UpdateDraft, error) {
	var draft UpdateDraft
	if err := ValidateUpdateSelection(observed, selected); err != nil {
		return draft, err
	}
	if len(data) > 64<<10 {
		return draft, ldserr.New("validation.update-draft", "draft exceeds 64 KiB")
	}
	if err := strictUpdateJSON(data, &draft); err != nil {
		return UpdateDraft{}, ldserr.New("validation.update-draft", err.Error())
	}
	if draft.SchemaVersion != 1 || draft.Edits == nil || len(draft.Edits) > 32 || draft.Questions == nil {
		return UpdateDraft{}, ldserr.New("validation.update-draft", "expected schemaVersion 1, 0..32 edits and a questions array")
	}
	scope := make(map[string]bool, len(selected))
	for _, name := range selected {
		scope[name] = true
	}
	evidence := make(map[string]bool, len(observed.Evidence))
	for _, entry := range observed.Evidence {
		evidence[entry.Path] = entry.Exists
	}
	seen := make(map[string]bool, len(draft.Edits))
	for _, edit := range draft.Edits {
		if !scope[edit.Path] || seen[edit.Path] || !safeUpdateDraftText(edit.Rationale) ||
			edit.After == "" || !utf8.ValidString(edit.After) || !safeUpdateDraftContent(edit.After) || len(edit.After) > 8<<20 || len(edit.Citations) == 0 {
			return UpdateDraft{}, ldserr.New("validation.update-draft", "edits require unique selected paths, full safe content, rationale and citations")
		}
		seen[edit.Path] = true
		citations := map[string]bool{}
		for _, name := range edit.Citations {
			if !evidence[name] || citations[name] {
				return UpdateDraft{}, ldserr.New("validation.update-draft", "citations must be unique, present original captured evidence")
			}
			citations[name] = true
		}
	}
	for _, question := range draft.Questions {
		if !safeUpdateDraftText(question) {
			return UpdateDraft{}, ldserr.New("validation.update-draft", "questions require nonblank safe text")
		}
	}
	return draft, nil
}

func safeUpdateDraftText(s string) bool {
	if !assessmentText(s) || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}

func safeUpdateDraftContent(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cf, r) || unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}
