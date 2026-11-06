package facts

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/procrastivity/lds/internal/adoption"
	"github.com/procrastivity/lds/internal/iostreams"
	"github.com/procrastivity/lds/internal/ldserr"
)

const maxSavedBytes = 256 << 10

func savedWorkflow(path string, review bool, streams *iostreams.Streams) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return ldserr.New("validation.facts-file", "could not resolve the saved facts path")
	}
	baseline, identity, err := readSaved(path)
	if err != nil {
		return err
	}
	bundle, err := adoption.DecodeFactBundle(baseline)
	if err != nil {
		return ldserr.New("validation.facts-file", err.Error())
	}
	// Loading any saved approval never carries it forward, even without edits.
	bundle.Approval.Approver = ""
	bundle.Approval.ApprovedFacts = []string{}
	bundle.Approval.Status = "pending"
	for _, question := range questions(&bundle) {
		if strings.ContainsFunc(*question.answer, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) {
			return ldserr.New("validation.facts-file", "saved assertions contain a control character")
		}
	}
	recheck := func() error {
		current, info, err := readSaved(path)
		if err != nil || !os.SameFile(identity, info) || !bytes.Equal(baseline, current) {
			return ldserr.New("refusal.facts-stale", "saved facts changed or became unavailable; no bundle was emitted; reload and review again")
		}
		return nil
	}
	if review {
		return reviewBundle(bundle, streams, recheck)
	}
	if err := editBundle(&bundle, streams); err != nil {
		return err
	}
	if err := recheck(); err != nil {
		return err
	}
	data, err := bundleJSON(bundle)
	if err != nil {
		return err
	}
	_, err = streams.Out.Write(data)
	return err
}

// Saved inputs and their ancestors must have trusted, quiescent writers. These
// reads/rechecks are not atomic snapshots or hostile-writer isolation.
func readSaved(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, ldserr.New("validation.facts-file", "could not inspect the saved facts file")
	}
	if !info.Mode().IsRegular() {
		return nil, nil, ldserr.New("refusal.facts-file", "saved facts must be a regular file, not a symlink or special file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, ldserr.New("internal.facts-file", "could not open the saved facts file")
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, nil, ldserr.New("refusal.facts-file", "saved facts file identity changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxSavedBytes+1))
	if err != nil {
		return nil, nil, ldserr.New("internal.facts-file", "could not read the saved facts file")
	}
	if len(data) > maxSavedBytes {
		return nil, nil, ldserr.New("validation.facts-size", "saved facts exceed 256 KiB")
	}
	return data, opened, nil
}

func editBundle(bundle *adoption.FactBundle, streams *iostreams.Streams) error {
	reader := bufio.NewReaderSize(streams.In, maxAnswerBytes+2)
	for {
		data, err := bundleJSON(*bundle)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(streams.Err, "Complete pending draft (prior approval discarded):\n%sType a field name to replace its answer, done to emit this pending draft, or :cancel.\nField> ", data); err != nil {
			return err
		}
		name, err := readAnswer(reader)
		if err != nil {
			return err
		}
		if name == "done" {
			return nil
		}
		var selected *question
		for _, question := range questions(bundle) {
			if question.name == name {
				selected = &question
				break
			}
		}
		if selected == nil {
			if _, err := fmt.Fprintln(streams.Err, "Choose an exact assertion/citation field name from the displayed JSON; approval is not editable here."); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(streams.Err, "%s — %s (blank clears to unknown):\n> ", selected.name, selected.prompt); err != nil {
			return err
		}
		answer, err := readAnswer(reader)
		if err != nil {
			return err
		}
		*selected.answer = answer
	}
}

func reviewBundle(bundle adoption.FactBundle, streams *iostreams.Streams, recheck func() error) error {
	// Use the same completeness/citation validation as adoption, without
	// accepting any loaded approval or substituting unknown assertion values.
	if err := adoption.ValidateFactAssertions(bundle); err != nil {
		return ldserr.New("validation.facts-incomplete", err.Error())
	}
	if _, err := fmt.Fprint(streams.Err, "Facts approval is your assertion of verification, not authenticated identity or plan approval. No project will be inspected or command executed.\nReviewer name (required):\n> "); err != nil {
		return err
	}
	reader := bufio.NewReaderSize(streams.In, maxAnswerBytes+2)
	approver, err := readAnswer(reader)
	if err != nil {
		return err
	}
	bundle.Approval.Approver = approver
	bundle.Approval.Status = "approved"
	bundle.Approval.ApprovedFacts = []string{"DisplayName", "Purpose", "CurrentCapability", "VerifiedUserOperation"}
	if err := adoption.ValidateFactBundle(bundle); err != nil {
		return ldserr.New("validation.facts-incomplete", err.Error())
	}
	data, err := bundleJSON(bundle)
	if err != nil {
		return err
	}
	if err := recheck(); err != nil {
		return err
	}
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	if _, err := fmt.Fprintf(streams.Err, "Complete candidate facts bundle (review every assertion, citation and approval field):\n%sApproving asserts you verified all four fact groups. This does not approve any init/adopt plan or target write.\n", data); err != nil {
		return err
	}
	fresh, err := streams.FreshInput()
	if err != nil {
		return ldserr.New("refusal.approval-input-unavailable", "fresh terminal input unavailable; no approved bundle was emitted")
	}
	if _, err := fmt.Fprintf(streams.Err, "To approve only these facts, type exactly: approve facts %s\nType decline or anything else to refuse.\n", digest); err != nil {
		return err
	}
	decision, err := readAnswer(fresh)
	if err != nil {
		return err
	}
	if decision != "approve facts "+digest {
		return ldserr.New("refusal.facts-declined", "fresh exact-digest facts approval was not received; no approved bundle was emitted")
	}
	if err := recheck(); err != nil {
		return err
	}
	_, err = streams.Out.Write(data)
	return err
}

func bundleJSON(bundle adoption.FactBundle) ([]byte, error) {
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) > maxSavedBytes {
		return nil, ldserr.New("validation.facts-size", "encoded facts exceed 256 KiB; no bundle was emitted")
	}
	return data, nil
}
