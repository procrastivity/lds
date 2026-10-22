package reconcile

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/cliflags"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	core "github.com/procrastivity/lds/internal/reconcile"
)

// guidedJournalCommand composes the existing journal operations; it is not a
// plumbing alias and cannot supply interactive authority to a generated skill.
func guidedJournalCommand(streams *iostreams.Streams) *cobra.Command {
	group := &cobra.Command{Use: "journal", Short: "review evidence and explicitly record journal claims offline"}
	for _, operation := range []string{"record", "decide", "reconsider"} {
		leaf := &cobra.Command{
			Use: operation, Short: journalDescription(operation), Args: cobra.NoArgs,
			Long: "Linux terminal workflow with a fresh scan, plain-text questions, complete request preview and fresh exact-digest confirmation. Writes only the explicit external journal. No model, inferred authority, project edits, automatic retries or verification of external applied-proof claims. Existing plumbing accepts authored request files separately.",
			RunE: func(cmd *cobra.Command, _ []string) error {
				terminal := &iostreams.Streams{In: cmd.InOrStdin(), Out: streams.Out, Err: streams.Err}
				if !terminal.IsInteractive() {
					return ldserr.New("refusal.interactive-required", "guided journal operations require a Linux terminal; no journal write was made")
				}
				pair, _ := cmd.Flags().GetString("pair")
				left, _ := cmd.Flags().GetString("left-root")
				right, _ := cmd.Flags().GetString("right-root")
				maps, _ := cmd.Flags().GetStringArray("map")
				journal, _ := cmd.Flags().GetString("journal")
				return runGuidedJournal(terminal, core.Input{PairPath: pair, LeftRoot: left, RightRoot: right, MapPaths: maps}, journal, operation, cliflags.FromContext(cmd.Context()).JSON)
			},
		}
		addPairJournalFlags(leaf)
		leaf.Flags().String("left-root", "", "absolute clean pinned left Git worktree")
		leaf.Flags().String("right-root", "", "absolute clean pinned right Git worktree")
		leaf.Flags().StringArray("map", nil, "absolute tracked raw map path (repeatable)")
		markJournalFlagsRequired(leaf, "left-root", "right-root", "map")
		annotateJournalLeaf(leaf)
		group.AddCommand(leaf)
	}
	return group
}

func runGuidedJournal(streams *iostreams.Streams, input core.Input, journal, operation string, jsonOut bool) error {
	review, err := core.PrepareJournalReview(input, journal)
	if err != nil {
		return err
	}
	scan, current := review.Scan(), review.Journal()
	q := journalQuestions{reader: bufio.NewReader(streams.In), out: streams.Err}
	q.show("Current evidence (byte observations, not semantic correspondence)", scan)
	q.show("Current journal (attributed claims, not authenticated authority or verified transfers)", current)
	q.print("No files have been created. Answers are claims for review, not authorization to edit either project. EOF cancels.\n")
	for index, candidate := range scan.Candidates {
		q.print("Candidate %d: %q (%s)\n", index+1, candidate.CandidateKey, candidate.CandidateID)
	}
	index := q.number("Candidate number (1-based in the displayed scan)", len(scan.Candidates))
	if q.err != nil {
		return q.err
	}
	candidate := scan.Candidates[index-1]
	if candidate.EvidenceDigest == nil || candidate.EvidenceStatus == "out-of-scope" {
		return ldserr.New("refusal.reconcile-stale-evidence", "select a candidate with current in-scope evidence")
	}
	var previous *core.JournalCandidateView
	for _, view := range current.Candidates {
		if view.CandidateID == candidate.CandidateID {
			previous = &view
			break
		}
	}
	if operation == "decide" && (previous == nil || previous.Decision.State != "pending-review") {
		return ldserr.New("refusal.reconcile-journal-conflict", "decide requires a previously recorded pending-review candidate; use reconsider to change a decision")
	}
	if operation == "reconsider" && (previous == nil || previous.LastDecision == nil) {
		return ldserr.New("refusal.reconcile-journal-conflict", "reconsider requires a previously decided candidate")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	request := map[string]any{
		"schemaVersion": 1, "requestId": "review-" + hex.EncodeToString(nonce[:]),
		"expectedJournalRevision": current.Revision, "candidateId": candidate.CandidateID,
		"expectedEvidenceDigest": *candidate.EvidenceDigest,
		"actor":                  q.text("Actor identifier"), "at": time.Now().UTC().Format(time.RFC3339Nano),
		"rationale": q.text("Rationale for this journal operation"),
	}
	if operation == "record" {
		q.correspondence(request, candidate)
	}
	if operation == "reconsider" {
		request["supersedesSeq"] = previous.LastDecision.Seq
		q.show("Decision explicitly superseded by this reconsideration", previous.LastDecision)
		request["trigger"] = q.trigger()
		if q.choice("Replace correspondence, provenance and assessment together", "yes", "no") == "yes" {
			q.correspondence(request, candidate)
		}
	}
	if operation != "record" {
		authority := q.authority(scan.Snapshots)
		request["authority"] = authority
		request["outcome"] = q.outcome(authority, scan.Snapshots)
	}
	if q.err != nil {
		return q.err
	}
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	preview, err := review.Preview(operation, data)
	if err != nil {
		return err
	}
	q.show("Complete journal preview (review every field; still no writes)", preview)
	if q.err != nil {
		return q.err
	}
	reader, err := streams.FreshInput()
	if err != nil {
		return ldserr.New("refusal.approval-input-unavailable", "could not establish fresh terminal input after the complete preview; no journal write was made")
	}
	q.reader = reader
	q.print("To authorize only this journal request, type exactly: %s %s\nType decline (without a digest), or anything else, to refuse.\n", operation, preview.PlanDigest)
	confirmation := q.line("Confirmation", false)
	if q.err != nil || confirmation != operation+" "+preview.PlanDigest {
		return ldserr.New("refusal.confirmation-declined", "fresh exact-digest confirmation was not received; no journal write was made")
	}
	result, err := review.Apply(preview, preview.PlanDigest)
	if err != nil {
		return err
	}
	if jsonOut {
		err = json.NewEncoder(streams.Out).Encode(result)
	} else {
		_, err = fmt.Fprintf(streams.Out, "Journal %s recorded at revision %d; no project edits or transfer verification.\n", operation, result.Revision)
	}
	if err != nil {
		return ldserr.New("internal.reconcile-journal-output", "could not write final response; read the journal and requestId before considering any retry")
	}
	return nil
}

// Questions retain the first input/output error; no subsequent prompt or write
// can turn a truncated answer or EOF into an affirmative choice.
type journalQuestions struct {
	reader *bufio.Reader
	out    io.Writer
	err    error
}

func (q *journalQuestions) print(format string, args ...any) {
	if q.err == nil {
		_, q.err = fmt.Fprintf(q.out, format, args...)
	}
}

func (q *journalQuestions) show(label string, value any) {
	if q.err != nil {
		return
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		q.err = err
		return
	}
	q.print("%s:\n%s\n", label, data)
}

func (q *journalQuestions) line(label string, required bool) string {
	q.print("%s: ", label)
	if q.err != nil {
		return ""
	}
	data, err := q.reader.ReadSlice('\n')
	if err != nil {
		q.err = ldserr.New("refusal.reconcile-input", "an answer was incomplete or exceeded 4095 bytes; no journal write was made")
		return ""
	}
	value := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	if !utf8.ValidString(value) {
		q.err = ldserr.New("validation.reconcile-answer", "answers must be valid UTF-8; no journal write was made")
	}
	if required && strings.TrimSpace(value) == "" {
		q.err = ldserr.New("validation.reconcile-answer", "an explicit non-empty answer is required; no journal write was made")
	}
	return value
}

func (q *journalQuestions) text(label string) string { return q.line(label, true) }

func (q *journalQuestions) choice(label string, choices ...string) string {
	value := q.text(label + " [" + strings.Join(choices, " / ") + "]")
	if q.err == nil {
		for _, choice := range choices {
			if value == choice {
				return value
			}
		}
		q.err = ldserr.New("validation.reconcile-answer", "answer must exactly match one of the displayed choices; no journal write was made")
	}
	return ""
}

func (q *journalQuestions) list(label string) []string {
	value := q.line(label+" (comma-separated; blank for none)", false)
	result := []string{}
	if value != "" {
		for _, item := range strings.Split(value, ",") {
			result = append(result, strings.TrimSpace(item))
		}
	}
	return result
}

func (q *journalQuestions) number(label string, limit int) int {
	value, err := strconv.Atoi(q.text(label))
	if q.err == nil && (err != nil || value < 1 || value > limit) {
		q.err = ldserr.New("validation.reconcile-answer", "selection must be a displayed 1-based number; no journal write was made")
	}
	return value
}

func (q *journalQuestions) trust(label string) string {
	return q.choice(label, "trusted", "bounded", "untrusted", "contradictory")
}

func (q *journalQuestions) correspondence(request map[string]any, candidate core.Candidate) {
	request["correspondence"] = core.Correspondence{
		Key:         q.text("Correspondence key (your reviewed identity, not inferred from paths)"),
		Relation:    q.choice("Correspondence relation", "one-to-one", "left-unmatched", "right-unmatched", "split", "merged"),
		LeftMembers: q.list("Left member identifiers"), RightMembers: q.list("Right member identifiers"),
	}
	q.show("Selected candidate evidence rows (1-based)", candidate.Evidence)
	for index, row := range candidate.Evidence {
		q.print("Evidence %d: project %q, path %q, SHA-256 %q\n", index+1, row.ProjectID, row.Key, row.SHA256)
	}
	provenance := []core.Provenance{}
	seen := map[int]bool{}
	for _, selection := range q.list("Evidence row numbers to cite") {
		index, err := strconv.Atoi(selection)
		if err != nil || index < 1 || index > len(candidate.Evidence) || seen[index] {
			if q.err == nil {
				q.err = ldserr.New("validation.reconcile-answer", "citations must name unique displayed evidence row numbers")
			}
			break
		}
		seen[index] = true
		row := candidate.Evidence[index-1]
		kind := "artifact"
		if row.SHA256 == "-" {
			kind = "missing"
		}
		provenance = append(provenance, core.Provenance{
			Kind: kind, ProjectID: row.ProjectID, Revision: &row.Revision, Path: row.Key, SHA256: &row.SHA256,
			Claim: q.text("Claim supported by the selected evidence row"), Trust: q.trust("Trust in this claim"),
		})
	}
	request["provenance"] = provenance
	assessment := core.Assessment{Code: q.text("Assessment code (your label)"), Trust: q.trust("Assessment trust")}
	assessment.Confidence.Level = q.choice("Confidence level", "not-assessed", "low", "medium", "high")
	if assessment.Confidence.Level != "not-assessed" && q.err == nil {
		assessment.Confidence.Rationale, _ = json.Marshal(q.text("Confidence rationale"))
	}
	request["assessment"] = assessment
}

func (q *journalQuestions) trigger() core.Trigger {
	return core.Trigger{ID: q.text("Trigger identifier"), Kind: q.choice("Trigger kind", "event", "date", "condition"), Description: q.text("Trigger description")}
}

func (q *journalQuestions) authority(snapshots core.Snapshots) core.Authority {
	q.print("Authority is your explicit declaration; evidence does not establish who may authorize a transfer.\n")
	q.print("Left project: %q; right project: %q.\n", snapshots.Left.ProjectID, snapshots.Right.ProjectID)
	result := core.Authority{Owner: q.text("Authority owner/declarer"), Status: q.choice("Authority status", "unresolved", "declared"), PermittedDirections: []string{}}
	if result.Status == "declared" {
		side := q.choice("Authoritative project side", "left", "right")
		project := snapshots.Left.ProjectID
		if side == "right" {
			project = snapshots.Right.ProjectID
		}
		result.AuthoritativeProjectID = &project
		result.PermittedDirections = q.list("Permitted directions: left-to-right and/or right-to-left")
	}
	return result
}

func (q *journalQuestions) outcome(authority core.Authority, snapshots core.Snapshots) core.Outcome {
	states := []string{"unresolved"}
	if authority.Status == "declared" {
		states = append(states, "divergence-retained", "intent", "deferred", "applied")
	}
	result := core.Outcome{State: q.choice("Outcome (journal claim only; does not perform a transfer)", states...)}
	switch result.State {
	case "intent", "applied":
		result.Direction = q.choice("Outcome direction", authority.PermittedDirections...)
	case "deferred":
		trigger := q.trigger()
		result.Trigger = &trigger
	}
	if result.State == "applied" {
		q.print("Applied is an attributed external-proof claim, NOT LDS verification. Supply actual external verifier evidence; never substitute these scanned pins for verification. No URI is fetched.\n")
		source, destination := snapshots.Left.ProjectID, snapshots.Right.ProjectID
		if result.Direction == "right-to-left" {
			source, destination = destination, source
		}
		proof := &core.AppliedProof{URI: q.text("External proof URI"), SHA256: q.text("External proof SHA-256 (64 lowercase hex digits)")}
		proof.Source = core.ProofProject{ProjectID: source, Revision: q.text("Externally verified source Git revision")}
		proof.Destination = core.ProofProject{ProjectID: destination, Revision: q.text("Externally verified destination Git revision")}
		proof.Scope = core.ProofScope{SourcePaths: q.list("Externally verified source paths (sorted)"), DestinationPaths: q.list("Externally verified destination paths (sorted)")}
		proof.Verifier = q.text("External verifier")
		proof.VerifiedAt = q.text("External verification time (UTC RFC3339)")
		proof.Rationale = q.text("External proof rationale")
		result.AppliedProof = proof
	}
	return result
}
