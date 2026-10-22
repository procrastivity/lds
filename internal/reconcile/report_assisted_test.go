//go:build linux

package reconcile

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func assistedReceipt() ReportAssistance {
	return ReportAssistance{
		Provider: "OpenCode Go", Model: "deepseek-v4-flash", FlowSource: "embedded",
		FlowSHA256: strings.Repeat("a", 64), SystemSHA256: strings.Repeat("b", 64), UserSHA256: strings.Repeat("c", 64),
		SystemBytes: 100, UserBytes: 200, ConsentAt: "2026-10-22T12:00:00Z",
	}
}

func assistedClaim(packet ReportEvidence) ReviewClaim {
	row := packet.Scan.Candidates[0].Evidence[0]
	return ReviewClaim{
		CandidateID: packet.Scan.Candidates[0].CandidateID, Status: "proposed",
		Correspondence: Correspondence{Key: "model:topic", LeftMembers: []string{"L-1"}, RightMembers: []string{"R-1"}, Relation: "one-to-one"},
		Rationale:      "Candidate for human review.",
		Provenance: []Provenance{{
			Kind: row.Kind, ProjectID: row.ProjectID, Revision: &row.Revision,
			Path: row.Key, SHA256: &row.SHA256, Claim: "Selected file", Trust: "bounded",
		}},
		Assessment: Assessment{Code: "needs-review", Confidence: Confidence{Level: "not-assessed"}, Trust: "bounded"},
		Conflicts:  []string{}, Questions: []string{},
	}
}

func assistedJSON(t *testing.T, claims ...ReviewClaim) []byte {
	t.Helper()
	data, err := json.Marshal(ReportDraft{SchemaVersion: 1, Claims: append([]ReviewClaim{}, claims...), Questions: []string{"Who owns this?"}})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAssistedReportPreservesOriginalReviewAndJournal(t *testing.T) {
	input, brief := reportFixture(t, []byte("selected content\n"))
	brief.Selections = []ContentSelection{{Side: "left", Path: "docs/left.md"}}
	brief.Questions = []string{"Human question"}
	scan, err := Scan(input.Input)
	if err != nil {
		t.Fatal(err)
	}
	human := assistedClaim(ReportEvidence{Scan: scan})
	human.Correspondence.Key = "human:topic"
	human.Status = "reviewed"
	human.Rationale = "Human assertion, not model advice."
	brief.Claims = []ReviewClaim{human}
	writeReportBrief(t, input.ReviewPath, brief)
	scanBytes, _ := json.Marshal(scan)
	input.Journal = filepath.Join(filepath.Dir(input.PairPath), "journal.json")
	journal := scanRecordJournal(t, scan, scan.Candidates[0], scanBytes)
	writeReportBytes(t, input.Journal, journal)
	observed, err := CollectReport(input)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Assistance != nil || observed.Advice != nil {
		t.Fatal("baseline was assisted")
	}
	claim := assistedClaim(observed)
	original, _ := json.Marshal(observed)
	before := snapshotFixtureFiles(t, input.LeftRoot, input.RightRoot)
	if err := RecheckReport(input, observed); err != nil {
		t.Fatal(err)
	}
	result, err := ReportReviewed(input, observed, assistedJSON(t, claim), assistedReceipt())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(input.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	packet := reportAppendix(t, data)
	afterObserved, _ := json.Marshal(observed)
	if !bytes.Equal(original, afterObserved) || !reflect.DeepEqual(packet.Review, brief) || packet.Journal == nil || packet.Journal.Revision != 1 || packet.Assistance == nil || packet.Advice == nil || len(packet.Advice.Claims) != 1 || packet.Advice.Claims[0].Status != "proposed" || !result.Assistance || result.ReportSHA256 != digest(data) {
		t.Fatalf("assisted report lost or promoted history: %+v", packet)
	}
	if bytes.Contains(data, []byte("Offline; no provider assistance")) || !bytes.Contains(data, []byte("Assisted advice")) || !bytes.Contains(data, []byte("consent asserted at")) || bytes.Contains(data, []byte("unselected right content")) {
		t.Fatal("report disclosure or attribution incorrect")
	}
	info, err := os.Stat(input.OutputPath)
	if err != nil || info.Mode().Perm() != 0o600 || !equalByteMaps(before, snapshotFixtureFiles(t, input.LeftRoot, input.RightRoot)) {
		t.Fatal("publication changed sources or permissions")
	}
	currentJournal, _ := os.ReadFile(input.Journal)
	if !bytes.Equal(journal, currentJournal) {
		t.Fatal("journal was changed")
	}
	if _, err := os.Stat(input.Journal + ".lock"); !os.IsNotExist(err) {
		t.Fatal("journal lock created")
	}
	assertCode(t, RecheckReport(input, observed), "refusal.reconcile-report-output")
}

func TestAssistedReportRejectsChangedBaselineAndOccupiedOutput(t *testing.T) {
	for _, change := range []struct {
		name string
		edit func(*testing.T, ReportInput)
	}{
		{"review", func(t *testing.T, in ReportInput) {
			writeReportBytes(t, in.ReviewPath, append([]byte(" "), mustRead(t, in.ReviewPath)...))
		}},
		{"map", func(t *testing.T, in ReportInput) { writeReportBytes(t, in.MapPaths[0], []byte("new map")) }},
		{"output", func(t *testing.T, in ReportInput) { writeReportBytes(t, in.OutputPath, []byte("owned")) }},
	} {
		t.Run(change.name, func(t *testing.T) {
			input, _ := reportFixture(t, []byte("left content\n"))
			observed, err := CollectReport(input)
			if err != nil {
				t.Fatal(err)
			}
			change.edit(t, input)
			if err := RecheckReport(input, observed); err == nil {
				t.Fatal("changed baseline accepted")
			}
			_, err = ReportReviewed(input, observed, assistedJSON(t), assistedReceipt())
			if err == nil {
				t.Fatal("changed baseline published")
			}
			if change.name == "output" && !bytes.Equal(mustRead(t, input.OutputPath), []byte("owned")) {
				t.Fatal("occupied output overwritten")
			}
		})
	}
}

func TestAssistedReportRejectsInvalidDraftAndReceiptWithoutEffects(t *testing.T) {
	input, brief := reportFixture(t, []byte("left content\n"))
	brief.Selections = []ContentSelection{{Side: "left", Path: "docs/left.md"}}
	writeReportBrief(t, input.ReviewPath, brief)
	observed, err := CollectReport(input)
	if err != nil {
		t.Fatal(err)
	}
	claim := assistedClaim(observed)
	var other, mapRow Evidence
	for _, row := range observed.Scan.Candidates[0].Evidence {
		switch row.Key {
		case "docs/right.md":
			other = row
		case "map.md":
			mapRow = row
		}
	}
	if other.Key == "" || mapRow.Key == "" {
		t.Fatal("fixture lacks unselected candidate and map rows")
	}
	cases := []struct {
		name string
		data []byte
	}{
		{"empty", assistedJSON(t)},
		{"unknown", []byte(`{"schemaVersion":1,"claims":[],"questions":[],"extra":1}`)},
		{"duplicate", []byte(`{"schemaVersion":1,"claims":[],"questions":[],"claims":[]}`)},
		{"null", []byte(`null`)},
		{"missing arrays", []byte(`{"schemaVersion":1}`)},
		{"oversize", bytes.Repeat([]byte("x"), 64*1024+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReportReviewed(input, observed, tc.data, assistedReceipt())
			if tc.name == "empty" {
				if err != nil {
					t.Fatal(err)
				}
				if packet := reportAppendix(t, mustRead(t, input.OutputPath)); packet.Advice == nil || len(packet.Advice.Claims) != 0 {
					t.Fatal("empty advice not preserved")
				}
				if err := os.Remove(input.OutputPath); err != nil {
					t.Fatal(err)
				}
			} else {
				assertCode(t, err, "validation.reconcile-report-draft")
			}
		})
	}
	for _, tc := range []struct {
		name string
		edit func(*ReviewClaim)
	}{
		{"reviewed", func(c *ReviewClaim) { c.Status = "reviewed" }},
		{"assessment", func(c *ReviewClaim) { c.Assessment.Code = "verified" }},
		{"unselected", func(c *ReviewClaim) {
			c.Provenance[0] = Provenance{Kind: other.Kind, ProjectID: other.ProjectID, Revision: &other.Revision, Path: other.Key, SHA256: &other.SHA256, Claim: "unselected", Trust: "bounded"}
		}},
		{"metadata-only map", func(c *ReviewClaim) {
			c.Provenance[0] = Provenance{Kind: mapRow.Kind, ProjectID: mapRow.ProjectID, Revision: &mapRow.Revision, Path: mapRow.Key, SHA256: &mapRow.SHA256, Claim: "map metadata", Trust: "bounded"}
		}},
		{"fake hash", func(c *ReviewClaim) { hash := strings.Repeat("0", 64); c.Provenance[0].SHA256 = &hash }},
		{"missing", func(c *ReviewClaim) { c.Provenance[0].Kind = "missing" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := claim
			c.Provenance = append([]Provenance(nil), claim.Provenance...)
			tc.edit(&c)
			_, err := ReportReviewed(input, observed, assistedJSON(t, c), assistedReceipt())
			assertCode(t, err, "validation.reconcile-report-draft")
		})
	}
	bad := assistedReceipt()
	bad.FlowSource = "remote"
	_, err = ReportReviewed(input, observed, assistedJSON(t), bad)
	assertCode(t, err, "validation.reconcile-report-assistance")
	altered := observed
	altered.Review.Purpose = "changed in caller"
	assertCode(t, RecheckReport(input, altered), "refusal.reconcile-report-stale")
	_, err = ReportReviewed(input, altered, assistedJSON(t), assistedReceipt())
	assertCode(t, err, "refusal.reconcile-report-stale")
	if _, err := os.Stat(input.OutputPath); !os.IsNotExist(err) {
		t.Fatal("refusal published output")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
