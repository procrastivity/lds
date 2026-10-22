package adoption

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestUpdateDraftSelectionAndAsymmetricEvidence(t *testing.T) {
	observed := AssessmentResult{Evidence: []AssessmentEvidence{
		{Path: "README.md", Exists: true},
		{Path: "docs/guide.md", Exists: true},
		{Path: "docs/other.md", Exists: true},
		{Path: "docs/new.md", Exists: false},
		{Path: "config.txt", Exists: true},
		{Path: "missing.txt", Exists: false},
	}}
	selected := []string{"README.md", "docs/guide.md"}
	if err := ValidateUpdateSelection(observed, selected); err != nil {
		t.Fatal(err)
	}
	draft := UpdateDraft{SchemaVersion: 1, Edits: []UpdateProposal{
		{Path: "docs/guide.md", After: "# Updated\n", Rationale: "Reflect config", Citations: []string{"config.txt"}},
	}, Questions: []string{"Is this wording correct?"}}
	data, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeUpdateDraft(observed, selected, data)
	if err != nil || !reflect.DeepEqual(got, draft) {
		t.Fatalf("asymmetric edit and citation: %+v, %v", got, err)
	}
	for name, scope := range map[string][]string{
		"empty": {}, "duplicate": {"README.md", "README.md"},
		"uncaptured": {"docs/unknown.md"}, "missing": {"docs/new.md"},
		"non-Markdown": {"config.txt"}, "escape": {"docs/../README.md"},
		"control": {"docs/a\n.md"},
	} {
		t.Run("selection "+name, func(t *testing.T) {
			if err := ValidateUpdateSelection(observed, scope); err == nil {
				t.Fatal("invalid explicit scope accepted")
			}
			if _, err := DecodeUpdateDraft(observed, scope, data); err == nil {
				t.Fatal("invalid scope accepted with otherwise valid draft")
			}
		})
	}
	tooMany := make([]string, 33)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("docs/%d.md", i)
	}
	if err := ValidateUpdateSelection(observed, tooMany); err == nil {
		t.Fatal("more than 32 selections accepted")
	}
	for name, edit := range map[string]UpdateProposal{
		"unselected captured": {Path: "docs/other.md", After: "new", Rationale: "Reason", Citations: []string{"config.txt"}},
		"uncaptured target":   {Path: "docs/unknown.md", After: "new", Rationale: "Reason", Citations: []string{"config.txt"}},
		"missing target":      {Path: "docs/new.md", After: "new", Rationale: "Reason", Citations: []string{"config.txt"}},
		"uncaptured citation": {Path: "README.md", After: "new", Rationale: "Reason", Citations: []string{"unknown.txt"}},
		"missing citation":    {Path: "README.md", After: "new", Rationale: "Reason", Citations: []string{"missing.txt"}},
		"duplicate citation":  {Path: "README.md", After: "new", Rationale: "Reason", Citations: []string{"config.txt", "config.txt"}},
		"empty content":       {Path: "README.md", After: "", Rationale: "Reason", Citations: []string{"config.txt"}},
		"blank rationale":     {Path: "README.md", After: "new", Rationale: "  ", Citations: []string{"config.txt"}},
		"unsafe content":      {Path: "README.md", After: "new\x00", Rationale: "Reason", Citations: []string{"config.txt"}},
		"unsafe rationale":    {Path: "README.md", After: "new", Rationale: "bad\nreason", Citations: []string{"config.txt"}},
		"bidi content":        {Path: "README.md", After: "new\u202e", Rationale: "Reason", Citations: []string{"config.txt"}},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := UpdateDraft{SchemaVersion: 1, Edits: []UpdateProposal{edit}, Questions: []string{}}
			payload, _ := json.Marshal(candidate)
			if _, err := DecodeUpdateDraft(observed, selected, payload); err == nil {
				t.Fatal("invalid model advice accepted")
			}
		})
	}
	draft.Edits = append(draft.Edits, draft.Edits[0])
	data, _ = json.Marshal(draft)
	if _, err := DecodeUpdateDraft(observed, selected, data); err == nil {
		t.Fatal("duplicate edit accepted")
	}
}

func TestUpdateDraftStrictJSONAndNoChangeAdvice(t *testing.T) {
	observed := AssessmentResult{Evidence: []AssessmentEvidence{{Path: "README.md", Exists: true}, {Path: "source.txt", Exists: true}}}
	selected := []string{"README.md"}
	noChange := []byte(`{"schemaVersion":1,"edits":[],"questions":["No edit needed; verify the source?"]}`)
	got, err := DecodeUpdateDraft(observed, selected, noChange)
	if err != nil || len(got.Edits) != 0 || len(got.Questions) != 1 {
		t.Fatalf("no-change advice rejected: %+v, %v", got, err)
	}
	valid := `{"schemaVersion":1,"edits":[{"path":"README.md","after":"# New\n","rationale":"Source changed","citations":["source.txt"]}],"questions":[]}`
	if _, err := DecodeUpdateDraft(observed, selected, []byte(valid)); err != nil {
		t.Fatalf("valid full replacement rejected: %v", err)
	}
	for name, raw := range map[string][]byte{
		"unknown":          []byte(strings.Replace(valid, `"questions":[]`, `"extra":true,"questions":[]`, 1)),
		"duplicate":        []byte(strings.Replace(valid, `"questions":[]`, `"questions":[],"questions":[]`, 1)),
		"alias":            []byte(strings.Replace(valid, `"schemaVersion"`, `"SchemaVersion"`, 1)),
		"nested alias":     []byte(strings.Replace(valid, `"path"`, `"Path"`, 1)),
		"missing field":    []byte(`{"schemaVersion":1,"edits":[]}`),
		"nested missing":   []byte(strings.Replace(valid, `,"citations":["source.txt"]`, ``, 1)),
		"null":             []byte(`{"schemaVersion":1,"edits":[],"questions":null}`),
		"null edits":       []byte(`{"schemaVersion":1,"edits":null,"questions":[]}`),
		"nested null":      []byte(strings.Replace(valid, `"citations":["source.txt"]`, `"citations":null`, 1)),
		"wrong type":       []byte(`{"schemaVersion":"1","edits":[],"questions":[]}`),
		"wrong array type": []byte(`{"schemaVersion":1,"edits":{},"questions":[]}`),
		"trailing":         []byte(valid + `{}`),
		"surrogate":        []byte(strings.Replace(valid, "Source changed", `\ud800`, 1)),
		"invalid UTF-8":    append([]byte(valid), 0xff),
		"wrong version":    []byte(strings.Replace(valid, `"schemaVersion":1`, `"schemaVersion":2`, 1)),
		"blank question":   []byte(`{"schemaVersion":1,"edits":[],"questions":[" "]}`),
		"control question": []byte(`{"schemaVersion":1,"edits":[],"questions":["bad\u0000"]}`),
		"oversize":         bytes.Repeat([]byte(" "), 64<<10+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeUpdateDraft(observed, selected, raw); err == nil {
				t.Fatal("invalid draft accepted")
			}
		})
	}
}
