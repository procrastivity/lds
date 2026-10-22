//go:build linux

package reconcile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportIncludesOnlySelectedContentAndAttributedReview(t *testing.T) {
	input, brief := reportFixture(t, []byte("selected left content\n```\n"))
	scan, err := Scan(input.Input)
	if err != nil {
		t.Fatal(err)
	}
	candidate := scan.Candidates[0]
	row := candidate.Evidence[0]
	brief.Selections = []ContentSelection{{Side: "left", Path: "docs/left.md"}}
	brief.Claims = []ReviewClaim{{
		CandidateID: candidate.CandidateID, Status: "proposed",
		Correspondence: Correspondence{Key: "review:topic", LeftMembers: []string{"L-1"}, RightMembers: []string{"R-1"}, Relation: "one-to-one"},
		Rationale:      "A proposal, not a reviewed match.",
		Provenance:     []Provenance{{Kind: row.Kind, ProjectID: row.ProjectID, Revision: &row.Revision, Path: row.Key, SHA256: &row.SHA256, Claim: "Scope evidence only.", Trust: "bounded"}},
		Assessment:     Assessment{Code: "needs-review", Confidence: Confidence{Level: "not-assessed"}, Trust: "bounded"},
		Conflicts:      []string{"Map label is not authority."}, Questions: []string{"Who declares authority?"},
	}}
	writeReportBrief(t, input.ReviewPath, brief)
	before := snapshotFixtureFiles(t, input.LeftRoot, input.RightRoot)
	result, err := Report(input)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(input.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	if result.Status != "review-required" || result.Assistance || result.Candidates != 1 || result.ReportSHA256 != hex.EncodeToString(hash[:]) {
		t.Fatalf("result does not describe actual report: %+v", result)
	}
	info, err := os.Stat(input.OutputPath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("private mode: %v %v", info, err)
	}
	packet := reportAppendix(t, data)
	if packet.Scan.Candidates[0].EvidenceStatus != "not-compared" || packet.Journal != nil || len(packet.Content) != 1 || packet.Content[0].Text != "selected left content\n```\n" || packet.Review.Claims[0].Status != "proposed" {
		t.Fatalf("wrong report content or promoted claim: %+v", packet)
	}
	if bytes.Contains(data, []byte("unselected right content")) || bytes.Contains(data, []byte("unselected raw map")) {
		t.Fatal("report disclosed unselected file content")
	}
	if !equalByteMaps(before, snapshotFixtureFiles(t, input.LeftRoot, input.RightRoot)) {
		t.Fatal("report changed source trees")
	}
	if _, err := Report(input); err == nil {
		t.Fatal("occupied report overwritten")
	} else {
		assertCode(t, err, "refusal.reconcile-report-output")
	}
	after, err := os.ReadFile(input.OutputPath)
	if err != nil || !bytes.Equal(data, after) {
		t.Fatal("rerun changed report")
	}
	input.OutputPath += ".invalid.md"
	brief.Claims[0].Provenance[0].Path = "outside-candidate.md"
	writeReportBrief(t, input.ReviewPath, brief)
	_, err = Report(input)
	assertCode(t, err, "validation.reconcile-review")
	if _, err := os.Stat(input.OutputPath); !os.IsNotExist(err) {
		t.Fatal("unbound citation published report")
	}
}

func TestReportPreservesJournalHistoryAndDoesNotInitializeOrWrite(t *testing.T) {
	input, _ := reportFixture(t, []byte("left content\n"))
	scan, err := Scan(input.Input)
	if err != nil {
		t.Fatal(err)
	}
	scanBytes, err := json.Marshal(scan)
	if err != nil {
		t.Fatal(err)
	}
	input.Journal = filepath.Join(filepath.Dir(input.PairPath), "journal.json")
	journal := scanRecordJournal(t, scan, scan.Candidates[0], scanBytes)
	if err := os.WriteFile(input.Journal, journal, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Report(input); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(input.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	packet := reportAppendix(t, data)
	if packet.Scan.Candidates[0].EvidenceStatus != "unchanged" || packet.Journal == nil || packet.Journal.Revision != 1 || packet.Journal.Candidates[0].Decision.State != "pending-review" {
		t.Fatalf("journal claim lost or promoted: %+v", packet)
	}
	// Removing scope must retain journal history without inventing current bytes.
	writePair(t, input.PairPath, scan.PairID, scan.Snapshots.Left.Revision, scan.Snapshots.Right.Revision, []string{}, []string{}, []candidateInput{})
	input.OutputPath += ".history.md"
	if _, err := Report(input); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(input.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	packet = reportAppendix(t, data)
	if packet.Scan.Candidates[0].EvidenceStatus != "out-of-scope" || packet.Scan.Candidates[0].EvidenceDigest != nil || packet.Journal.Candidates[0].Correspondence.Relation != "one-to-one" {
		t.Fatalf("journal-only history became current evidence: %+v", packet)
	}
	after, err := os.ReadFile(input.Journal)
	if err != nil || !bytes.Equal(journal, after) {
		t.Fatal("journal changed")
	}
	if _, err := os.Stat(input.Journal + ".lock"); !os.IsNotExist(err) {
		t.Fatal("report created journal lock")
	}
	input.Journal += ".missing"
	input.OutputPath += ".missing.md"
	if _, err := Report(input); err == nil {
		t.Fatal("missing journal accepted")
	}
	if _, err := os.Stat(input.Journal); !os.IsNotExist(err) {
		t.Fatal("report initialized journal")
	}
}

func TestReportRejectsUnsafeUnboundAndOversizeInputsWithoutEffects(t *testing.T) {
	cases := []struct {
		name   string
		change func(*testing.T, *ReportInput, *ReviewBrief)
		code   string
	}{
		{"unknown field", func(t *testing.T, in *ReportInput, _ *ReviewBrief) {
			writeReportBytes(t, in.ReviewPath, []byte(`{"schemaVersion":1,"extra":true}`))
		}, "validation.reconcile-review"},
		{"duplicate field", func(t *testing.T, in *ReportInput, _ *ReviewBrief) {
			writeReportBytes(t, in.ReviewPath, []byte(`{"schemaVersion":1,"schemaVersion":1}`))
		}, "validation.reconcile-review"},
		{"unscoped selection", func(t *testing.T, in *ReportInput, b *ReviewBrief) {
			b.Selections = []ContentSelection{{Side: "right", Path: "not-scoped.md"}}
			writeReportBrief(t, in.ReviewPath, *b)
		}, "validation.reconcile-review"},
		{"duplicate selection", func(t *testing.T, in *ReportInput, b *ReviewBrief) {
			b.Selections = []ContentSelection{{Side: "left", Path: "docs/left.md"}, {Side: "left", Path: "docs/left.md"}}
			writeReportBrief(t, in.ReviewPath, *b)
		}, "validation.reconcile-review"},
		{"output in source", func(_ *testing.T, in *ReportInput, _ *ReviewBrief) {
			in.OutputPath = filepath.Join(in.LeftRoot, "report.md")
		}, "refusal.reconcile-report-output"},
		{"output in git", func(_ *testing.T, in *ReportInput, _ *ReviewBrief) {
			in.OutputPath = filepath.Join(in.RightRoot, ".git", "report.md")
		}, "refusal.reconcile-report-output"},
		{"output symlink", func(t *testing.T, in *ReportInput, _ *ReviewBrief) {
			if err := os.Symlink(in.PairPath, in.OutputPath); err != nil {
				t.Fatal(err)
			}
		}, "refusal.reconcile-report-output"},
		{"parent symlink", func(t *testing.T, in *ReportInput, _ *ReviewBrief) {
			parent := filepath.Dir(in.OutputPath)
			link := filepath.Join(parent, "link")
			if err := os.Symlink(parent, link); err != nil {
				t.Fatal(err)
			}
			in.OutputPath = filepath.Join(link, "new.md")
		}, "validation.reconcile-report-path"},
		{"input symlink", func(t *testing.T, in *ReportInput, _ *ReviewBrief) {
			link := in.ReviewPath + ".link"
			if err := os.Symlink(in.ReviewPath, link); err != nil {
				t.Fatal(err)
			}
			in.ReviewPath = link
		}, "validation.reconcile-report-path"},
		{"metadata budget", func(t *testing.T, in *ReportInput, _ *ReviewBrief) {
			writeReportBytes(t, in.ReviewPath, bytes.Repeat([]byte(" "), 2*1024*1024))
		}, "validation.reconcile-report-budget"},
		{"dirty source", func(t *testing.T, in *ReportInput, _ *ReviewBrief) {
			writeReportBytes(t, filepath.Join(in.LeftRoot, "docs/left.md"), []byte("changed baseline"))
		}, "refusal.reconcile-stale-evidence"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			input, brief := reportFixture(t, []byte("left content\n"))
			test.change(t, &input, &brief)
			before := snapshotFixtureFiles(t, input.LeftRoot, input.RightRoot)
			_, err := Report(input)
			assertCode(t, err, test.code)
			if !equalByteMaps(before, snapshotFixtureFiles(t, input.LeftRoot, input.RightRoot)) {
				t.Fatal("refusal changed sources")
			}
			// The occupied symlink case intentionally has an existing output.
			if test.name != "output symlink" {
				if _, err := os.Lstat(input.OutputPath); !os.IsNotExist(err) {
					t.Fatalf("refusal published report: %v", err)
				}
			}
		})
	}
	for _, size := range []int{64 * 1024, 64*1024 + 1} {
		input, _ := reportFixture(t, bytes.Repeat([]byte("x"), size))
		_, err := Report(input)
		if size == 64*1024 {
			if err != nil {
				t.Fatalf("exact file limit refused: %v", err)
			}
		} else {
			assertCode(t, err, "validation.reconcile-report-budget")
		}
	}
}

func reportFixture(t *testing.T, leftData []byte) (ReportInput, ReviewBrief) {
	t.Helper()
	base := t.TempDir()
	left, right := filepath.Join(base, "left"), filepath.Join(base, "right")
	lrev := initGitFixture(t, left, map[string][]byte{"docs/left.md": leftData})
	rrev := initGitFixture(t, right, map[string][]byte{"docs/right.md": []byte("unselected right content\n"), "map.md": []byte("unselected raw map\n")})
	pair := filepath.Join(base, "pair.json")
	writePair(t, pair, "fixture:report:v1", lrev, rrev, []string{"docs/left.md"}, []string{"docs/right.md", "docs/missing.md"}, []candidateInput{{"topic:review", []string{"docs/left.md"}, []string{"docs/right.md", "docs/missing.md"}}})
	brief := ReviewBrief{SchemaVersion: 1, Purpose: "Review a disposable pair", Reviewer: "fixture-reviewer", At: "2026-10-21T00:00:00Z", Selections: []ContentSelection{}, Claims: []ReviewClaim{}, Questions: []string{}}
	input := ReportInput{Input: Input{PairPath: pair, LeftRoot: left, RightRoot: right, MapPaths: []string{filepath.Join(right, "map.md")}}, ReviewPath: filepath.Join(base, "review.json"), OutputPath: filepath.Join(base, "report.md")}
	writeReportBrief(t, input.ReviewPath, brief)
	return input, brief
}

func writeReportBrief(t *testing.T, name string, brief ReviewBrief) {
	t.Helper()
	data, err := json.Marshal(brief)
	if err != nil {
		t.Fatal(err)
	}
	writeReportBytes(t, name, data)
}

func writeReportBytes(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(name, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func reportAppendix(t *testing.T, data []byte) ReportEvidence {
	t.Helper()
	_, appendix, ok := strings.Cut(string(data), "## Provenance and selected content\n\n")
	if !ok {
		t.Fatal("report missing provenance appendix")
	}
	lines := strings.Split(strings.TrimSpace(appendix), "\n")
	var packet ReportEvidence
	if len(lines) < 3 {
		t.Fatal("report missing JSON")
	}
	if err := json.Unmarshal([]byte(strings.Join(lines[1:len(lines)-1], "\n")), &packet); err != nil {
		t.Fatal(err)
	}
	return packet
}
