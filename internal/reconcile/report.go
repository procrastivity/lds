package reconcile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/procrastivity/lds/internal/ldserr"
)

const (
	reportInputLimit = 2 * 1024 * 1024
	reportFileLimit  = 64 * 1024
	reportTextLimit  = 512 * 1024
)

// ReportInput names explicit read-only inputs and a new external report.
type ReportInput struct {
	Input
	ReviewPath string
	OutputPath string
}

// ReviewBrief contains attributed assertions, never write authority.
type ReviewBrief struct {
	SchemaVersion int                `json:"schemaVersion"`
	Purpose       string             `json:"purpose"`
	Reviewer      string             `json:"reviewer"`
	At            string             `json:"at"`
	Selections    []ContentSelection `json:"selections"`
	Claims        []ReviewClaim      `json:"claims"`
	Questions     []string           `json:"questions"`
}

// ContentSelection opts into full bytes of one scoped artifact or map.
type ContentSelection struct {
	Side string `json:"side"`
	Path string `json:"path"`
}

// ReviewClaim keeps new assertions separate from saved journal claims.
type ReviewClaim struct {
	CandidateID    string         `json:"candidateId"`
	Status         string         `json:"status"`
	Correspondence Correspondence `json:"correspondence"`
	Rationale      string         `json:"rationale"`
	Provenance     []Provenance   `json:"provenance"`
	Assessment     Assessment     `json:"assessment"`
	Conflicts      []string       `json:"conflicts"`
	Questions      []string       `json:"questions"`
}

// ReportResult describes the published report, not a reconciliation decision.
type ReportResult struct {
	SchemaVersion int       `json:"schemaVersion"`
	Operation     string    `json:"operation"`
	Status        string    `json:"status"`
	ReportPath    string    `json:"reportPath"`
	ReportSHA256  string    `json:"reportSha256"`
	PairID        string    `json:"pairId"`
	Snapshots     Snapshots `json:"snapshots"`
	Candidates    int       `json:"candidates"`
	Assistance    bool      `json:"assistance"`
}

type reportObservation struct {
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	Bytes    int    `json:"bytes"`
	Identity string `json:"identity"`
}

type reportContent struct {
	ContentSelection
	SHA256 string `json:"sha256"`
	Text   string `json:"text"`
}

// ReportAssistance is an attributed transfer receipt, not proof of provider identity or consent.
type ReportAssistance struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	FlowSource   string `json:"flowSource"`
	FlowSHA256   string `json:"flowSha256"`
	SystemSHA256 string `json:"systemSha256"`
	UserSHA256   string `json:"userSha256"`
	SystemBytes  int    `json:"systemBytes"`
	UserBytes    int    `json:"userBytes"`
	ConsentAt    string `json:"consentAt"`
}

// ReportDraft contains unverified provider advice, separate from human review.
type ReportDraft struct {
	SchemaVersion int           `json:"schemaVersion"`
	Claims        []ReviewClaim `json:"claims"`
	Questions     []string      `json:"questions"`
}

// ReportEvidence is the captured, bounded baseline for one report. Treat it as
// immutable after collection; RecheckReport compares it with a fresh capture.
type ReportEvidence struct {
	SchemaVersion        int                 `json:"schemaVersion"`
	Collector            string              `json:"collector"`
	Inputs               []reportObservation `json:"inputs"`
	Review               ReviewBrief         `json:"review"`
	Content              []reportContent     `json:"content"`
	Scan                 Response            `json:"scan"`
	ScanSHA256           string              `json:"scanSha256"`
	Journal              *JournalResponse    `json:"journal,omitempty"`
	JournalSHA256        string              `json:"journalProjectionSha256,omitempty"`
	RootIdentities       []string            `json:"rootIdentities"`
	OutputPath           string              `json:"outputPath"`
	OutputParentIdentity string              `json:"outputParentIdentity"`
	Excluded             []string            `json:"excluded"`
	Assistance           *ReportAssistance   `json:"assistance,omitempty"`
	Advice               *ReportDraft        `json:"advice,omitempty"`
}

// Report constructs and rechecks a bounded offline review packet, then
// publishes one private external Markdown file without replacing any path.
// No source or journal write operation is called, even on refusal.
func Report(input ReportInput) (result ReportResult, resultErr error) {
	defer func() { resultErr = reportFailure(resultErr) }()
	packet, err := collectReport(input)
	if err != nil {
		return ReportResult{}, err
	}
	data := renderReport(packet, input.OutputPath)
	current, err := collectReport(input)
	if err != nil {
		return ReportResult{}, err
	}
	if !reflect.DeepEqual(packet, current) {
		return ReportResult{}, reportError("refusal.reconcile-report-stale", "report inputs changed; collect and review a new baseline")
	}
	if err := publishReport(input.OutputPath, packet.OutputParentIdentity, data); err != nil {
		return ReportResult{}, err
	}
	return ReportResult{
		SchemaVersion: 1, Operation: "reconcile report", Status: "review-required",
		ReportPath: input.OutputPath, ReportSHA256: digest(data), PairID: packet.Scan.PairID,
		Snapshots: packet.Scan.Snapshots, Candidates: len(packet.Scan.Candidates), Assistance: false,
	}, nil
}

// CollectReport reads the explicit inputs without publishing or writing to a source or journal.
func CollectReport(input ReportInput) (ReportEvidence, error) {
	packet, err := collectReport(input)
	return packet, reportFailure(err)
}

// RecheckReport refuses a changed baseline, including changed output availability.
func RecheckReport(input ReportInput, observed ReportEvidence) error {
	if observed.Assistance != nil || observed.Advice != nil {
		return reportError("refusal.reconcile-report-stale", "expected an unassisted collected baseline")
	}
	current, err := CollectReport(input)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(observed, current) {
		return reportError("refusal.reconcile-report-stale", "report inputs changed; collect and review a new baseline")
	}
	return nil
}

func collectReport(input ReportInput) (ReportEvidence, error) {
	packet := ReportEvidence{
		SchemaVersion: 1, Collector: "lds/reconcile-report/v1",
		OutputPath: input.OutputPath,
		Inputs:     []reportObservation{}, Content: []reportContent{},
		Excluded: []string{"Unselected file content; all files outside declared scope/maps.", "Semantic correspondence, authority, transfer and parity verification.", "Provider assistance and source/journal writes."},
	}
	remaining := reportInputLimit
	readInput := func(name string) ([]byte, error) {
		data, observation, err := readReportFile(name, remaining)
		if err == nil {
			remaining -= len(data)
			packet.Inputs = append(packet.Inputs, observation)
		}
		return data, err
	}
	pairBytes, err := readInput(input.PairPath)
	if err != nil {
		return packet, err
	}
	pair, err := parsePair(pairBytes)
	if err != nil {
		return packet, err
	}
	if _, err := validateScope(pair); err != nil {
		return packet, err
	}
	if len(*pair.Scope.LeftPaths)+len(*pair.Scope.RightPaths) > 64 || len(input.MapPaths) == 0 || len(input.MapPaths) > 16 {
		return packet, reportError("validation.reconcile-report-budget", "report requires at most 64 scoped paths and 1–16 maps")
	}
	// Report collection is deliberately stricter than v1 scan: no symlink
	// components, including external input parents, are admitted.
	left, err := inspectReportRoot(input.LeftRoot, pair.Left)
	if err != nil {
		return packet, err
	}
	right, err := inspectReportRoot(input.RightRoot, pair.Right)
	if err != nil {
		return packet, err
	}
	for _, root := range []*projectPin{left, right} {
		identity, err := reportDirectoryIdentity(root.root)
		if err != nil {
			return packet, err
		}
		packet.RootIdentities = append(packet.RootIdentities, identity)
	}
	external := []string{input.PairPath, input.ReviewPath, input.OutputPath}
	if input.Journal != "" {
		external = append(external, input.Journal)
	}
	for _, name := range external {
		if err := reportExternalPath(name, left, right); err != nil {
			return packet, err
		}
	}
	if err := checkReportOutput(input.OutputPath); err != nil {
		return packet, err
	}
	packet.OutputParentIdentity, err = reportDirectoryIdentity(filepath.Dir(input.OutputPath))
	if err != nil {
		return packet, err
	}
	reviewBytes, err := readInput(input.ReviewPath)
	if err != nil {
		return packet, err
	}
	if err := decodeStrictJSON(reviewBytes, &packet.Review); err != nil {
		return packet, reportError("validation.reconcile-review", err.Error())
	}
	var journalBytes []byte
	if input.Journal != "" {
		journalBytes, err = readInput(input.Journal)
		if err != nil {
			return packet, err
		}
	}

	// Preflight every raw scan artifact before invoking the unchanged v1
	// scanner. Only selected content is retained in the review report.
	artifacts := map[ContentSelection][]byte{}
	for _, side := range []struct {
		name  string
		root  *projectPin
		paths []string
	}{{"left", left, *pair.Scope.LeftPaths}, {"right", right, *pair.Scope.RightPaths}} {
		for _, name := range side.paths {
			full := filepath.Join(side.root.root, filepath.FromSlash(name))
			data, obs, err := readReportFile(full, reportFileLimit)
			if os.IsNotExist(err) {
				continue // Scan checks that missing artifacts are untracked.
			}
			if err != nil {
				return packet, err
			}
			packet.Inputs = append(packet.Inputs, obs)
			artifacts[ContentSelection{Side: side.name, Path: name}] = data
		}
	}
	for _, name := range input.MapPaths {
		data, obs, err := readReportFile(name, reportFileLimit)
		if err != nil {
			return packet, err
		}
		packet.Inputs = append(packet.Inputs, obs)
		for _, side := range []struct {
			name string
			root *projectPin
		}{{"left", left}, {"right", right}} {
			if rel, inside := relativeWithin(side.root.root, name); inside {
				artifacts[ContentSelection{Side: side.name, Path: filepath.ToSlash(rel)}] = data
			}
		}
	}
	scanInput := input.Input
	scanInput.Journal = ""
	packet.Scan, err = Scan(scanInput)
	if err != nil {
		return packet, err
	}
	if packet.Scan.PairSHA256 != digest(pairBytes) {
		return packet, reportError("refusal.reconcile-report-stale", "pair changed during report collection")
	}
	// Bind preflight-captured bytes to the pinned scanner's actual rows.
	for selection, data := range artifacts {
		projectID := left.projectID
		if selection.Side == "right" {
			projectID = right.projectID
		}
		found := false
		for _, m := range packet.Scan.Maps {
			if m.ProjectID == projectID && m.Path == selection.Path && m.SHA256 == digest(data) {
				found = true
			}
		}
		for _, c := range packet.Scan.Candidates {
			for _, e := range c.Evidence {
				if e.ProjectID == projectID && e.Key == selection.Path && e.Kind == "artifact" && e.SHA256 == digest(data) {
					found = true
				}
			}
		}
		if !found {
			return packet, reportError("refusal.reconcile-report-stale", "captured artifact differs from scan evidence")
		}
	}
	if input.Journal != "" {
		state, err := parseJournal(journalBytes, packet.Scan.PairID, left.projectID, right.projectID)
		if err != nil {
			return packet, err
		}
		journal := projectJournal("reconcile journal read", input.Journal, packet.Scan.PairID, packet.Scan.Snapshots, state)
		packet.Journal = &journal
		packet.Scan.JournalPath = input.Journal
		packet.Scan.Candidates = compareJournal(packet.Scan.Candidates, state)
		data, err := json.Marshal(journal)
		if err != nil {
			return packet, err
		}
		packet.JournalSHA256 = digest(data)
	}
	if err := validateReview(packet.Review, packet.Scan); err != nil {
		return packet, reportError("validation.reconcile-review", err.Error())
	}
	total := 0
	seen := map[ContentSelection]bool{}
	for _, selection := range packet.Review.Selections {
		data, exists := artifacts[selection]
		if !exists || seen[selection] || !utf8.Valid(data) || bytes.ContainsRune(data, 0) {
			return packet, reportError("validation.reconcile-review", "content selection must name a unique present scoped/map UTF-8 text file")
		}
		seen[selection] = true
		total += len(data)
		if total > reportTextLimit {
			return packet, reportError("validation.reconcile-report-budget", "selected content exceeds 512 KiB")
		}
		packet.Content = append(packet.Content, reportContent{ContentSelection: selection, SHA256: digest(data), Text: string(data)})
	}
	sort.Slice(packet.Inputs, func(i, j int) bool { return packet.Inputs[i].Path < packet.Inputs[j].Path })
	data, err := json.Marshal(packet.Scan)
	if err != nil {
		return packet, err
	}
	packet.ScanSHA256 = digest(data)
	return packet, nil
}

func inspectReportRoot(name string, pin *projectFile) (*projectPin, error) {
	if _, err := reportDirectoryIdentity(name); err != nil {
		return nil, err
	}
	return inspectPin(name, pin)
}

func reportExternalPath(name string, roots ...*projectPin) error {
	if !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return reportError("validation.reconcile-report-path", "report inputs/output must be clean absolute paths")
	}
	for _, root := range roots {
		for _, dir := range []string{root.root, root.commonDir} {
			if name == dir || isWithin(dir, name) {
				return reportError("refusal.reconcile-report-output", "pair, review, journal and report must be outside both roots and Git common directories")
			}
		}
	}
	return nil
}

func validateReview(brief ReviewBrief, scan Response) error {
	if brief.SchemaVersion != 1 || strings.TrimSpace(brief.Purpose) == "" || strings.TrimSpace(brief.Reviewer) == "" || brief.Claims == nil || brief.Selections == nil || brief.Questions == nil {
		return fmt.Errorf("review requires schemaVersion 1, purpose, reviewer, at and non-null selections/claims/questions arrays")
	}
	if _, err := time.Parse(time.RFC3339, brief.At); err != nil {
		return fmt.Errorf("review at must be RFC3339")
	}
	candidates := map[string]Candidate{}
	for _, candidate := range scan.Candidates {
		candidates[candidate.CandidateID] = candidate
	}
	keys, members := map[string]bool{}, map[string]bool{}
	for _, claim := range brief.Claims {
		candidate, exists := candidates[claim.CandidateID]
		if !exists || candidate.EvidenceDigest == nil || (claim.Status != "reviewed" && claim.Status != "proposed") || strings.TrimSpace(claim.Rationale) == "" || claim.Conflicts == nil || claim.Questions == nil || len(claim.Provenance) == 0 {
			return fmt.Errorf("claim needs a current candidate, status, rationale, citations and conflicts/questions arrays")
		}
		if err := validateCorrespondence(claim.Correspondence); err != nil {
			return err
		}
		if err := validateAssessment(claim.Assessment); err != nil {
			return err
		}
		if keys[claim.Correspondence.Key] {
			return fmt.Errorf("duplicate correspondence key")
		}
		keys[claim.Correspondence.Key] = true
		for side, names := range [][]string{claim.Correspondence.LeftMembers, claim.Correspondence.RightMembers} {
			for _, name := range names {
				key := fmt.Sprintf("%d:%s", side, name)
				if members[key] {
					return fmt.Errorf("member reused in review correspondence")
				}
				members[key] = true
			}
		}
		if err := validateProvenance(claim.Provenance, observationPayload{Evidence: candidate.Evidence}, scan.Snapshots.Left.ProjectID, scan.Snapshots.Right.ProjectID); err != nil {
			return err
		}
		for _, citation := range claim.Provenance {
			matched := false
			for _, row := range candidate.Evidence {
				if citation.Revision != nil && citation.SHA256 != nil && row.ProjectID == citation.ProjectID && row.Revision == *citation.Revision && row.Key == citation.Path && row.Kind == citation.Kind && row.SHA256 == *citation.SHA256 {
					matched = true
				}
			}
			if !matched {
				return fmt.Errorf("claim citation is not captured candidate evidence")
			}
		}
	}
	return nil
}

func renderReport(packet ReportEvidence, output string) []byte {
	var text strings.Builder
	fmt.Fprintf(&text, "# Reconciliation review — review required\n\nCreated: %s\n\nReport: %s\n\n", time.Now().UTC().Format(time.RFC3339), reportCell(output))
	if packet.Assistance == nil {
		text.WriteString("Offline; no provider assistance, source edits or journal writes. Reviewer identity/time and all correspondence, authority and outcomes are attributed assertions, not authenticated facts. Byte equality is not semantic correspondence or transfer proof. Missing review and unresolved authority remain open.\n\n")
	} else {
		text.WriteString("Assisted; provider advice is unverified and cannot approve correspondence, authority, transfer or parity. The transfer receipt is attributed, not authenticated consent or provider identity. No source edits or journal writes. Human review and saved history remain separate; byte equality is not semantic correspondence or transfer proof.\n\n")
	}
	fmt.Fprintf(&text, "Pair: %s\n\nLeft: %s at %s\n\nRight: %s at %s\n\n", reportCell(packet.Scan.PairID), reportCell(packet.Scan.Snapshots.Left.ProjectID), packet.Scan.Snapshots.Left.Revision, reportCell(packet.Scan.Snapshots.Right.ProjectID), packet.Scan.Snapshots.Right.Revision)
	text.WriteString("## Scope and current observations\n\n| Candidate | Comparison | Current digest | Journal observed digest | Last decision (sequence / digest / outcome) |\n| --- | --- | --- | --- | --- |\n")
	for _, c := range packet.Scan.Candidates {
		current := "none (history only)"
		if c.EvidenceDigest != nil {
			current = *c.EvidenceDigest
		}
		fmt.Fprintf(&text, "| %s | %s | %s | %s | %s |\n", reportCell(c.CandidateKey), c.EvidenceStatus, current, reportCell(string(c.JournalEvidenceDigest)), reportCell(string(c.LastDecision)))
	}
	text.WriteString("\nAll relationships without an explicit brief or journal assertion remain unreviewed.\n\n### Scoped files and raw maps\n\n| Candidate / map | Side / project | Path | Presence | SHA-256 |\n| --- | --- | --- | --- | --- |\n")
	for _, c := range packet.Scan.Candidates {
		for side, paths := range [][]PathRecord{c.Left, c.Right} {
			for _, p := range paths {
				hash := "none"
				if p.SHA256 != nil {
					hash = *p.SHA256
				}
				fmt.Fprintf(&text, "| %s | %s | %s | %t | %s |\n", reportCell(c.CandidateKey), []string{"left", "right"}[side], reportCell(p.Path), p.Present, hash)
			}
		}
	}
	for _, m := range packet.Scan.Maps {
		fmt.Fprintf(&text, "| raw map | %s | %s | true | %s |\n", reportCell(m.ProjectID), reportCell(m.Path), m.SHA256)
	}
	text.WriteString("\n## New review assertions and conflicts\n\n")
	fmt.Fprintf(&text, "Purpose: %s\n\nReviewer (asserted): %s at %s\n\n", reportCell(packet.Review.Purpose), reportCell(packet.Review.Reviewer), reportCell(packet.Review.At))
	if len(packet.Review.Claims) == 0 {
		text.WriteString("No new correspondence assertions supplied.\n\n")
	}
	for _, claim := range packet.Review.Claims {
		fmt.Fprintf(&text, "### %s — %s assertion\n\nCandidate: %s\n\nRelationship: %s; left [%s], right [%s].\n\nRationale: %s\n\nTrust: %s; confidence: %s. These are reviewer assessments, not verified semantics.\n\n", reportCell(claim.Correspondence.Key), claim.Status, reportCell(claim.CandidateID), claim.Correspondence.Relation, reportCell(strings.Join(claim.Correspondence.LeftMembers, ", ")), reportCell(strings.Join(claim.Correspondence.RightMembers, ", ")), reportCell(claim.Rationale), claim.Assessment.Trust, claim.Assessment.Confidence.Level)
		for _, conflict := range claim.Conflicts {
			fmt.Fprintf(&text, "- Conflict: %s\n", reportCell(conflict))
		}
		for _, question := range claim.Questions {
			fmt.Fprintf(&text, "- Question: %s\n", reportCell(question))
		}
		text.WriteString("\nExact citations and confidence rationale are retained in the provenance appendix.\n\n")
	}
	for _, question := range packet.Review.Questions {
		fmt.Fprintf(&text, "- Open question: %s\n", reportCell(question))
	}
	if packet.Advice != nil {
		text.WriteString("\n## Assisted advice — unverified proposals, not human assertions\n\n")
		fmt.Fprintf(&text, "Attributed transfer: %s / %s; flow %s (%s), system %s (%d bytes), user %s (%d bytes); consent asserted at %s. Full receipt and advice are in the appendix.\n\n", reportCell(packet.Assistance.Provider), reportCell(packet.Assistance.Model), reportCell(packet.Assistance.FlowSource), packet.Assistance.FlowSHA256, packet.Assistance.SystemSHA256, packet.Assistance.SystemBytes, packet.Assistance.UserSHA256, packet.Assistance.UserBytes, reportCell(packet.Assistance.ConsentAt))
		if len(packet.Advice.Claims) == 0 {
			text.WriteString("No model claims supplied.\n\n")
		}
		for _, claim := range packet.Advice.Claims {
			fmt.Fprintf(&text, "### %s — proposed model advice (needs review)\n\nCandidate: %s\n\nRelationship: %s; left [%s], right [%s].\n\nRationale: %s\n\nTrust: %s; confidence: %s (attributed, not verified).\n\n", reportCell(claim.Correspondence.Key), reportCell(claim.CandidateID), claim.Correspondence.Relation, reportCell(strings.Join(claim.Correspondence.LeftMembers, ", ")), reportCell(strings.Join(claim.Correspondence.RightMembers, ", ")), reportCell(claim.Rationale), claim.Assessment.Trust, claim.Assessment.Confidence.Level)
			for _, conflict := range claim.Conflicts {
				fmt.Fprintf(&text, "- Conflict: %s\n", reportCell(conflict))
			}
			for _, question := range claim.Questions {
				fmt.Fprintf(&text, "- Question: %s\n", reportCell(question))
			}
		}
		for _, question := range packet.Advice.Questions {
			fmt.Fprintf(&text, "- Model question: %s\n", reportCell(question))
		}
	}
	text.WriteString("\n## Saved journal claims\n\n")
	if packet.Journal == nil {
		text.WriteString("No journal supplied; no comparison or decision is inferred.\n")
	} else {
		text.WriteString("These are journal claims, not independently verified authority or applied proof. Older decision digests remain distinct from latest observations. Conflicting new assertions above do not replace saved claims.\n\n")
		for _, c := range packet.Journal.Candidates {
			fmt.Fprintf(&text, "- %s: %s correspondence, %s outcome; last event %d. Authority claim: %s. Full event/proof citations are in the appendix.\n", reportCell(c.CandidateKey), c.Correspondence.Relation, c.Decision.State, c.LastEventSeq, reportCell(string(c.Authority)))
		}
	}
	text.WriteString("\n## Coverage and limits\n\n")
	for _, exclusion := range packet.Excluded {
		fmt.Fprintf(&text, "- %s\n", exclusion)
	}
	text.WriteString("\nOnly explicitly selected content is included in full below. Unselected semantic claims are not independently supported by content review. Inputs were rechecked before publication; this is not an atomic snapshot, perpetual approval, or permission to record a decision. Recollect if pins, source/map/review bytes or journal revision change.\n\n## Provenance and selected content\n\n")
	writeReportJSON(&text, packet)
	return []byte(text.String())
}

func writeReportJSON(text *strings.Builder, value any) {
	data, _ := json.MarshalIndent(value, "", "  ") // Packet types contain only JSON-safe values.
	fence := "```"
	for strings.Contains(string(data), fence) {
		fence += "`"
	}
	fmt.Fprintf(text, "%sjson\n%s\n%s\n", fence, data, fence)
}

func reportCell(value string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ", "\r", " ", "`", "\\`", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_").Replace(html.EscapeString(value))
}

func reportError(code, message string) error { return ldserr.New(code, message) }

func reportFailure(err error) error {
	var structured *ldserr.Error
	if err == nil || errors.As(err, &structured) {
		return err
	}
	code := "internal.reconcile-report-io"
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ELOOP) || errors.Is(err, syscall.ENOTDIR) {
		code = "validation.reconcile-report-path"
	}
	return reportError(code, err.Error())
}
