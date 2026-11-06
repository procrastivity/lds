// Package facts collects and reviews adoption assertions without inspecting a project.
package facts

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
	"github.com/procrastivity/lds/internal/manifest"
	"github.com/procrastivity/lds/internal/surface"
)

// Stay below Linux's canonical terminal line limit, including multibyte text.
const maxAnswerBytes = 2048

type question struct {
	name, prompt string
	answer       *string
}

// Command drafts, resumes or explicitly reviews caller-authored adoption facts.
// The caller owns output redirection; facts approval never approves a write plan.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "facts", Short: "draft, resume or review adoption facts offline in a terminal",
		Long: "Collect caller-authored assertions and source citations in a Linux terminal. " +
			"Leave unknown answers blank, or --resume a saved bundle to edit selected fields and emit a pending draft. " +
			"Use --review separately to validate a saved complete bundle, identify its reviewer and display the full candidate for fresh digest-bound facts approval. " +
			"Always emits a raw adoption JSON bundle on stdout, with or without --json; prompts go to stderr. " +
			"No project argument, file writes, command execution, provider, inferred evidence or plan approval. " +
			"Save stdout privately outside your project; never redirect over the input file. Facts approval is an unauthenticated reviewer assertion, not verified truth. " +
			"Type :cancel or end input to stop with empty stdout.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			streams := &iostreams.Streams{In: cmd.InOrStdin(), Out: streams.Out, Err: streams.Err}
			if !streams.IsInteractive() {
				return ldserr.New("refusal.interactive-required", "lds facts requires a Linux terminal for input and stderr; no draft was emitted")
			}
			resume, _ := cmd.Flags().GetString("resume")
			review, _ := cmd.Flags().GetString("review")
			if cmd.Flags().Changed("resume") || cmd.Flags().Changed("review") {
				path := resume
				if cmd.Flags().Changed("review") {
					path = review
				}
				if path == "" {
					return ldserr.New("validation.facts-file", "saved facts path must not be empty")
				}
				return savedWorkflow(path, cmd.Flags().Changed("review"), streams)
			}
			return questionnaire(streams)
		},
	}
	cmd.Flags().String("resume", "", "saved facts JSON to edit; emits a pending draft and discards prior approval")
	cmd.Flags().String("review", "", "saved complete facts JSON to review for separate fresh facts approval")
	cmd.MarkFlagsMutuallyExclusive("resume", "review")
	surface.Annotate(cmd, surface.Plumbing)
	manifest.SetOutputSchema(cmd, json.RawMessage(`{
		"type":"object",
		"required":["DisplayName","Purpose","CurrentCapability","DisplayNameSource","PurposeSource","CurrentCapabilitySource","Prerequisites","OperationName","OperationSummary","Invocation","Input","OutputEvidence","OperationSource","Limitations","approval"],
		"additionalProperties":false,
		"properties":{
			"DisplayName":{"type":"string"},"Purpose":{"type":"string"},"CurrentCapability":{"type":"string"},
			"DisplayNameSource":{"type":"string"},"PurposeSource":{"type":"string"},"CurrentCapabilitySource":{"type":"string"},
			"Prerequisites":{"type":"string"},"OperationName":{"type":"string"},"OperationSummary":{"type":"string"},
			"Invocation":{"type":"string"},"Input":{"type":"string"},"OutputEvidence":{"type":"string"},"OperationSource":{"type":"string"},"Limitations":{"type":"string"},
			"approval":{"type":"object","required":["approver","approvedFacts","status"],"additionalProperties":false,"properties":{"approver":{"type":"string"},"approvedFacts":{"type":"array","maxItems":4,"uniqueItems":true,"items":{"enum":["DisplayName","Purpose","CurrentCapability","VerifiedUserOperation"]}},"status":{"enum":["pending","approved"]}}}
		}
	}`))
	return cmd
}

func questionnaire(streams *iostreams.Streams) error {
	var bundle adoption.FactBundle
	bundle.Approval.Status = "pending"
	bundle.Approval.ApprovedFacts = []string{}
	if _, err := fmt.Fprintln(streams.Err, "Unapproved adoption facts draft. No project will be read and no commands will run.\nAnswers are your assertions, not verified evidence. Do not enter secrets.\nOne line per answer (at most 2048 UTF-8 bytes); leave unknowns blank.\nType :cancel or end input to stop without a draft. Completion never approves facts."); err != nil {
		return err
	}
	reader := bufio.NewReaderSize(streams.In, maxAnswerBytes+2)
	unknown := []string{}
	for _, question := range questions(&bundle) {
		if _, err := fmt.Fprintf(streams.Err, "%s — %s:\n> ", question.name, question.prompt); err != nil {
			return err
		}
		answer, err := readAnswer(reader)
		if err != nil {
			return err
		}
		if answer == "" {
			unknown = append(unknown, question.name)
		}
		*question.answer = answer
	}
	if _, err := fmt.Fprintf(streams.Err, "Unanswered fields: %s\nDraft only: approval remains pending even if every answer is filled.\nVerify every assertion and citation and obtain independent facts approval before init/adopt. Facts approval is not plan approval.\n", strings.Join(unknown, ", ")); err != nil {
		return err
	}
	encoder := json.NewEncoder(streams.Out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(bundle)
}

func questions(bundle *adoption.FactBundle) []question {
	return []question{
		{"DisplayName", "Project display name", &bundle.DisplayName},
		{"DisplayNameSource", "Citation for that name (safe relative path or HTTP(S) URL)", &bundle.DisplayNameSource},
		{"Purpose", "What is this project for?", &bundle.Purpose},
		{"PurposeSource", "Citation for that purpose", &bundle.PurposeSource},
		{"CurrentCapability", "What does it actually do today (not planned capability)?", &bundle.CurrentCapability},
		{"CurrentCapabilitySource", "Citation for that current capability", &bundle.CurrentCapabilitySource},
		{"Prerequisites", "Prerequisites for one independently verified existing user operation", &bundle.Prerequisites},
		{"OperationName", "Name of that operation", &bundle.OperationName},
		{"OperationSummary", "What does that operation do?", &bundle.OperationSummary},
		{"Invocation", "Exact invocation (recorded as text, never executed)", &bundle.Invocation},
		{"Input", "Exact input used for that operation", &bundle.Input},
		{"OutputEvidence", "Observed or source-backed output evidence; leave blank if unverified", &bundle.OutputEvidence},
		{"OperationSource", "Citation for the operation and its evidence", &bundle.OperationSource},
		{"Limitations", "Known limitations, uncertainty and unchecked behavior", &bundle.Limitations},
	}
}

func readAnswer(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		return "", ldserr.New("validation.facts-answer", "answer exceeds 2048 bytes; no draft was emitted")
	}
	if errors.Is(err, io.EOF) {
		return "", ldserr.New("refusal.facts-cancelled", "questionnaire did not finish; no draft was emitted")
	}
	if err != nil {
		return "", ldserr.New("internal.facts-input", "could not read an answer; no draft was emitted")
	}
	answer := strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r")
	if answer == ":cancel" {
		return "", ldserr.New("refusal.facts-cancelled", "questionnaire cancelled; no draft was emitted")
	}
	if len(answer) > maxAnswerBytes || !utf8.ValidString(answer) || strings.ContainsFunc(answer, unicode.IsControl) {
		return "", ldserr.New("validation.facts-answer", "answer must be one line of at most 2048 UTF-8 bytes without control characters; no draft was emitted")
	}
	if strings.TrimSpace(answer) == "" {
		answer = ""
	}
	return answer, nil
}
