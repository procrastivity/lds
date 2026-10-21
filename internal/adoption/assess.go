package adoption

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/procrastivity/lds/internal/ldserr"
)

// AssessmentBrief declares a review's purpose, additional read scope and advice.
// Proposals are caller-authored interpretations, never verified product facts.
type AssessmentBrief struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Reason        string               `json:"reason"`
	Audience      string               `json:"audience"`
	Evidence      []AssessmentSource   `json:"evidence"`
	Proposals     []AssessmentProposal `json:"proposals"`
	Questions     []string             `json:"questions"`
}

// AssessmentSource explicitly permits reading one additional target-relative file.
type AssessmentSource struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// AssessmentProposal is suggested text with references to collected evidence.
type AssessmentProposal struct {
	Path         string   `json:"path"`
	Summary      string   `json:"summary"`
	Evidence     []string `json:"evidence"`
	ProposedText string   `json:"proposedText"`
}

// AssessmentEvidence records local bytes or the absence of a proposed document.
type AssessmentEvidence struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	SHA256  string `json:"sha256"`
	Content string `json:"content"`
}

// AssessmentResult is an evidence packet, not approval or an applied edit set.
type AssessmentResult struct {
	SchemaVersion      int                  `json:"schemaVersion"`
	Operation          string               `json:"operation"`
	Status             string               `json:"status"`
	TargetRoot         string               `json:"targetRoot"`
	TargetIdentity     string               `json:"targetIdentity"`
	ReportPath         string               `json:"reportPath"`
	BriefSHA256        string               `json:"briefSha256"`
	Brief              AssessmentBrief      `json:"brief"`
	Inventory          Inventory            `json:"inventory"`
	InventoryExitCode  int                  `json:"inventoryExitCode"`
	Validation         Validation           `json:"validation"`
	ValidationExitCode int                  `json:"validationExitCode"`
	Evidence           []AssessmentEvidence `json:"evidence"`
}

// Assess creates one new report outside the explicit target, never target edits.
// It performs no inference, provider requests, Git operations or project commands.
func Assess(root, briefPath, reportPath string) (AssessmentResult, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return AssessmentResult{}, err
	}
	if !filepath.IsAbs(reportPath) {
		return AssessmentResult{}, ldserr.New("validation.assessment-output", "--output must name an absolute report path outside the target")
	}
	reportPath = filepath.Clean(reportPath)
	relative, err := filepath.Rel(root, reportPath)
	if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return AssessmentResult{}, ldserr.New("refusal.assessment-output", "report must be outside the target; no target writes are allowed")
	}
	data, err := readAssessmentInput(briefPath)
	if err != nil {
		return AssessmentResult{}, err
	}
	brief, err := decodeAssessmentBrief(data)
	if err != nil {
		return AssessmentResult{}, ldserr.New("validation.assessment-brief", err.Error())
	}
	result, err := collectAssessment(root, reportPath, data, brief)
	if err != nil {
		return AssessmentResult{}, err
	}
	current, err := collectAssessment(root, reportPath, data, brief)
	if err != nil {
		return AssessmentResult{}, err
	}
	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return AssessmentResult{}, err
	}
	currentPayload, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return AssessmentResult{}, err
	}
	currentBrief, err := readAssessmentInput(briefPath)
	if err != nil || !bytes.Equal(data, currentBrief) || !bytes.Equal(payload, currentPayload) {
		return AssessmentResult{}, ldserr.New("refusal.assessment-stale", "review inputs changed during collection; start a new assessment")
	}
	if err := writeAssessmentReport(reportPath, result.TargetIdentity, assessmentMarkdown(result, payload)); err != nil {
		return AssessmentResult{}, err
	}
	return result, nil
}

func readAssessmentInput(name string) ([]byte, error) {
	parent, err := openRoot(filepath.Dir(name))
	if err != nil {
		return nil, fmt.Errorf("open brief parent: %w", err)
	}
	defer func() { _ = parent.Close() }()
	file, err := openChild(parent, filepath.Base(name))
	if err != nil {
		return nil, fmt.Errorf("open brief: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ldserr.New("validation.assessment-brief", "brief must be a regular JSON file")
	}
	return io.ReadAll(file)
}

func decodeAssessmentBrief(data []byte) (AssessmentBrief, error) {
	var brief AssessmentBrief
	if !utf8.Valid(data) {
		return brief, errors.New("brief must be UTF-8 JSON")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return brief, err
	}
	if err := assessmentObject(data, "schemaVersion", "reason", "audience", "evidence", "proposals", "questions"); err != nil {
		return brief, err
	}
	var objects struct {
		Evidence  []json.RawMessage `json:"evidence"`
		Proposals []json.RawMessage `json:"proposals"`
	}
	if err := json.Unmarshal(data, &objects); err != nil {
		return brief, err
	}
	for _, source := range objects.Evidence {
		if err := assessmentObject(source, "path", "sha256"); err != nil {
			return brief, err
		}
	}
	for _, proposal := range objects.Proposals {
		if err := assessmentObject(proposal, "path", "summary", "evidence", "proposedText"); err != nil {
			return brief, err
		}
	}
	if err := json.Unmarshal(data, &brief); err != nil {
		return brief, err
	}
	if brief.SchemaVersion != 1 || !assessmentText(brief.Reason) || !assessmentText(brief.Audience) {
		return brief, errors.New("brief requires schemaVersion 1 and nonempty reason/audience")
	}
	seen := map[string]bool{}
	for _, source := range brief.Evidence {
		digest, err := hex.DecodeString(source.SHA256)
		if !assessmentPath(source.Path) || seen[source.Path] || err != nil || len(digest) != 32 || strings.ToLower(source.SHA256) != source.SHA256 {
			return brief, errors.New("evidence requires unique safe relative paths and lowercase SHA-256 hashes")
		}
		seen[source.Path] = true
	}
	for _, proposal := range brief.Proposals {
		if !assessmentDocPath(proposal.Path) || !assessmentText(proposal.Summary) || !assessmentText(proposal.ProposedText) || len(proposal.Evidence) == 0 {
			return brief, errors.New("proposals require a scoped Markdown path, summary, proposedText and evidence references")
		}
		for _, name := range proposal.Evidence {
			if !assessmentPath(name) {
				return brief, errors.New("proposal evidence references must be safe relative paths")
			}
		}
	}
	for _, question := range brief.Questions {
		if !assessmentText(question) {
			return brief, errors.New("questions must be nonempty text")
		}
	}
	return brief, nil
}

func assessmentObject(data []byte, fields ...string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	if len(object) != len(fields) {
		return errors.New("brief objects must contain exactly the documented fields")
	}
	for _, name := range fields {
		value, ok := object[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("brief field %q is required and cannot be null", name)
		}
	}
	return nil
}

func assessmentPath(name string) bool {
	return assessmentText(name) && name != "." && name != ".." && path.Clean(name) == name &&
		!path.IsAbs(name) && !strings.HasPrefix(name, "../") && !strings.ContainsAny(name, "\\:\x00\r\n\t")
}

func assessmentDocPath(name string) bool {
	return assessmentPath(name) && (name == "README.md" ||
		((strings.HasPrefix(name, "docs/") || strings.HasPrefix(name, "engineering/")) && strings.HasSuffix(strings.ToLower(name), ".md")))
}

func assessmentText(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	for _, r := range text {
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			return false
		}
	}
	return true
}

func collectAssessment(root, reportPath string, data []byte, brief AssessmentBrief) (AssessmentResult, error) {
	result := AssessmentResult{
		SchemaVersion: 1, Operation: "docs assess", Status: "review-required",
		TargetRoot: root, ReportPath: reportPath, BriefSHA256: sha256Hex(data), Brief: brief,
		Evidence: []AssessmentEvidence{},
	}
	dir, err := openRoot(root)
	if err != nil {
		return result, err
	}
	defer func() { _ = dir.Close() }()
	result.TargetIdentity, err = rootIdentity(dir)
	if err != nil {
		return result, err
	}
	result.Inventory, err = Inspect(root)
	if err != nil {
		return result, err
	}
	result.Validation, err = Validate(root)
	if err != nil {
		return result, err
	}
	if len(result.Validation.Findings) != 0 {
		result.ValidationExitCode = 1
	}
	paths := map[string]string{}
	for _, file := range result.Inventory.Files {
		paths[file.Path] = file.SHA256
	}
	for _, source := range brief.Evidence {
		if expected, ok := paths[source.Path]; ok && expected != source.SHA256 {
			return result, ldserr.New("refusal.assessment-stale", "declared evidence differs from observed documentation")
		}
		paths[source.Path] = source.SHA256
	}
	for _, proposal := range brief.Proposals {
		for _, source := range proposal.Evidence {
			if _, ok := paths[source]; !ok {
				return result, ldserr.New("validation.assessment-brief", "proposal cites evidence outside the inventory and declared read scope: "+source)
			}
		}
	}
	// Bind missing proposal targets too, without treating absence as a defect.
	for _, proposal := range brief.Proposals {
		if _, ok := paths[proposal.Path]; !ok {
			paths[proposal.Path] = ""
		}
	}
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		expected := paths[name]
		probe, err := probeTargetFile(dir, name)
		if errors.Is(err, syscall.ENOTDIR) {
			return result, fmt.Errorf("%w: evidence must be a regular file with directory ancestors: %s", ErrUnsafeTarget, name)
		}
		if err != nil {
			return result, err
		}
		entry := AssessmentEvidence{Path: name, Exists: probe.exists}
		if probe.exists {
			if !utf8.Valid(probe.data) {
				return result, ldserr.New("validation.assessment-evidence", "evidence must be UTF-8 text: "+name)
			}
			entry.SHA256, entry.Content = sha256Hex(probe.data), string(probe.data)
		}
		if expected != "" && (!probe.exists || entry.SHA256 != expected) {
			return result, ldserr.New("refusal.assessment-stale", "evidence is missing or differs from its expected SHA-256: "+name)
		}
		result.Evidence = append(result.Evidence, entry)
	}
	return result, nil
}

func assessmentMarkdown(result AssessmentResult, payload []byte) []byte {
	var report strings.Builder
	report.WriteString("# Documentation maintenance assessment\n\nReview required. No target files were changed. This is not an approved edit plan.\n\n")
	report.WriteString("## Review brief\n\nReason: " + escapeMarkdownText(result.Brief.Reason) + "\n\nAudience: " + escapeMarkdownText(result.Brief.Audience) + "\n\n")
	report.WriteString("## Observed local-link findings\n\n")
	if len(result.Validation.Findings) == 0 {
		report.WriteString("No missing supported local file targets observed. This does not verify content accuracy or complete link coverage.\n\n")
	}
	for _, finding := range result.Validation.Findings {
		report.WriteString("- " + escapeMarkdownText(fmt.Sprintf("%s:%d: %s", finding.Path, finding.Line, finding.Message)) + "\n")
	}
	report.WriteString("\n## Reviewer proposals — not verified facts or applied edits\n\n")
	if len(result.Brief.Proposals) == 0 {
		report.WriteString("No proposals supplied. LDS does not infer semantic impacts or invent replacement instructions.\n\n")
	}
	for _, proposal := range result.Brief.Proposals {
		report.WriteString("### " + escapeMarkdownText(proposal.Path) + "\n\n" + escapeMarkdownText(proposal.Summary) + "\n\n")
		report.WriteString("Cited evidence: " + escapeMarkdownText(strings.Join(proposal.Evidence, ", ")) + "\n\nProposed text (reviewer supplied):\n\n" + fencedInvocation(proposal.ProposedText) + "\n\n")
	}
	report.WriteString("## Unresolved questions\n\n")
	if len(result.Brief.Questions) == 0 {
		report.WriteString("None supplied; this is not a completeness claim.\n\n")
	}
	for _, question := range result.Brief.Questions {
		report.WriteString("- " + escapeMarkdownText(question) + "\n")
	}
	report.WriteString("\n## Evidence and limits\n\nInputs are observed working-tree bytes, not authenticated claims or clean Git pins. Collection was repeated to detect changes, not taken atomically. Recheck evidence before acting on this report. Trusted, quiescent target and report ancestors are required.\n\nInventory covers root README.md and Markdown under engineering/ and docs/. Additional files were explicitly named by the brief. Skipped checks are not passes; fragments, external URLs and semantic correctness are not verified. No project commands, provider calls, automatic matching or target edits were performed. Accepting advice does not apply it.\n\nComplete inventory, validation, skips, input hashes/content and authored brief:\n\n")
	report.WriteString(fencedInvocation(string(payload)) + "\n")
	return []byte(report.String())
}
