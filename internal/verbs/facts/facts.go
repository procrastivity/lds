// Package facts collects unapproved adoption assertions without inspecting a project.
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

// Command emits a raw, pending fact bundle on stdout only after all questions.
// The caller owns any redirection; neither answers nor completion grant approval.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "facts", Short: "draft adoption facts offline with a terminal questionnaire; never approve them",
		Long: "Collect caller-authored assertions and source citations in a Linux terminal. " +
			"Leave unknown answers blank. Always emits a raw unapproved adoption JSON bundle, with or without --json; prompts go to stderr. " +
			"No project argument, file reads/writes, command execution, provider, inferred evidence or approval. " +
			"Save stdout privately outside your project, verify and complete the assertions, and obtain independent facts approval before init/adopt. " +
			"Type :cancel or end input to stop with empty stdout.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			streams := &iostreams.Streams{In: cmd.InOrStdin(), Out: streams.Out, Err: streams.Err}
			if !streams.IsInteractive() {
				return ldserr.New("refusal.interactive-required", "lds facts requires a Linux terminal for input and stderr; no draft was emitted")
			}
			return questionnaire(streams)
		},
	}
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
			"approval":{"type":"object","required":["approver","approvedFacts","status"],"additionalProperties":false,"properties":{"approver":{"const":""},"approvedFacts":{"type":"array","maxItems":0},"status":{"const":"pending"}}}
		}
	}`))
	return cmd
}

func questionnaire(streams *iostreams.Streams) error {
	var bundle adoption.FactBundle
	bundle.Approval.Status = "pending"
	bundle.Approval.ApprovedFacts = []string{}
	questions := []struct {
		name, prompt string
		answer       *string
	}{
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
	if _, err := fmt.Fprintln(streams.Err, "Unapproved adoption facts draft. No project will be read and no commands will run.\nAnswers are your assertions, not verified evidence. Do not enter secrets.\nOne line per answer (at most 2048 UTF-8 bytes); leave unknowns blank.\nType :cancel or end input to stop without a draft. Completion never approves facts."); err != nil {
		return err
	}
	reader := bufio.NewReaderSize(streams.In, maxAnswerBytes+2)
	unknown := []string{}
	for _, question := range questions {
		if _, err := fmt.Fprintf(streams.Err, "%s — %s:\n> ", question.name, question.prompt); err != nil {
			return err
		}
		line, err := reader.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			return ldserr.New("validation.facts-answer", "answer exceeds 2048 bytes; no draft was emitted")
		}
		if errors.Is(err, io.EOF) {
			return ldserr.New("refusal.facts-cancelled", "questionnaire did not finish; no draft was emitted")
		}
		if err != nil {
			return ldserr.New("internal.facts-input", "could not read an answer; no draft was emitted")
		}
		answer := strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r")
		if answer == ":cancel" {
			return ldserr.New("refusal.facts-cancelled", "questionnaire cancelled; no draft was emitted")
		}
		if len(answer) > maxAnswerBytes || !utf8.ValidString(answer) || strings.ContainsFunc(answer, unicode.IsControl) {
			return ldserr.New("validation.facts-answer", "answer must be one line of at most 2048 UTF-8 bytes without control characters; no draft was emitted")
		}
		if strings.TrimSpace(answer) == "" {
			answer = ""
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
