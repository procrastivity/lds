package adoption

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/procrastivity/lds/internal/asset"
)

const renderFixtureDir = "../asset/testdata/adoption/greenfield"

func FuzzFactBundle(f *testing.F) {
	seed, err := os.ReadFile(filepath.Join(renderFixtureDir, "facts.json"))
	if err != nil {
		f.Fatal(err)
	}
	var baseline FactBundle
	if err := json.Unmarshal(seed, &baseline); err != nil {
		f.Fatal(err)
	}
	for i, data := range [][]byte{seed, {}, []byte(`{"Purpose":"a","Purpose":"b"}`), []byte("\xff"), []byte("\"\\\n\t"), []byte("café"), []byte("\u202e"), bytes.Repeat([]byte("x"), 4096)} {
		f.Add(data, uint8(i))
	}
	f.Fuzz(func(t *testing.T, data []byte, mutation uint8) {
		if len(data) > 4096 {
			t.Skip()
		}
		if got, err := DecodeFactBundle(data); err == nil {
			assertJSONPreserved(t, data, got)
		}
		// Every input also exercises successful parsing, not just random rejection.
		// Quoting produces safe, nonempty ASCII while retaining arbitrary escapes.
		want := baseline
		want.Purpose = strconv.QuoteToASCII(string(data))
		wire, err := json.Marshal(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeFactBundle(wire)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("facts changed during decoding: %+v, %v", got, err)
		}
		if err := ValidateFactBundle(got); err != nil {
			t.Fatalf("complete fixture assertions rejected: %v", err)
		}
		purpose, err := json.Marshal(want.Purpose)
		if err != nil {
			t.Fatal(err)
		}
		var invalid []byte
		switch mutation % 8 {
		case 0:
			invalid = append([]byte(`{"Purpose":"different",`), wire[1:]...)
		case 1:
			invalid = bytes.Replace(wire, []byte(`"Purpose":`), []byte(`"purpose":`), 1)
		case 2:
			invalid = bytes.Replace(wire, append([]byte(`"Purpose":`), purpose...), []byte(`"Purpose":null`), 1)
		case 3:
			invalid = append(wire, []byte(`{}`)...)
		case 4:
			invalid = bytes.Replace(wire, []byte(`"status":`), []byte(`"status":"approved","status":`), 1)
		case 5:
			invalid = append([]byte{0xff}, wire...)
		case 6:
			invalid = append([]byte(`{"unexpected":true,`), wire[1:]...)
		case 7:
			invalid = bytes.Replace(wire, []byte(`"Purpose":`+string(purpose)+`,`), nil, 1)
		}
		if _, err := DecodeFactBundle(invalid); err == nil {
			t.Fatalf("invalid facts mutation %d accepted", mutation%8)
		}
	})
}

// Compare JSON values independently of the strict decoder: accepted input must
// not silently drop unknown keys, change key case, or normalize null fields.
func assertJSONPreserved(t *testing.T, input []byte, decoded any) {
	t.Helper()
	var before, after any
	if !utf8.Valid(input) || json.Unmarshal(input, &before) != nil {
		t.Fatal("accepted invalid JSON or raw UTF-8")
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &after); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("accepted JSON changed: before=%#v after=%#v", before, after)
	}
}

func TestRender_GreenfieldMatchesIndependentOutputs(t *testing.T) {
	useIsolatedAssetDirectories(t)
	root := t.TempDir()
	factsPath := filepath.Join(renderFixtureDir, "facts.json")
	before := snapshotRenderTarget(t, root)
	outputs, err := Render(root, factsPath, "")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(outputs) != 4 {
		t.Fatalf("Render returned %d outputs, want four", len(outputs))
	}
	wantPaths := []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"}
	for i, output := range outputs {
		if output.Path != wantPaths[i] {
			t.Errorf("output[%d].Path = %q, want %q", i, output.Path, wantPaths[i])
		}
		want, err := os.ReadFile(filepath.Join(renderFixtureDir, "expected", filepath.FromSlash(output.Path)))
		if err != nil {
			t.Fatalf("read independent expected %s: %v", output.Path, err)
		}
		if !bytes.Equal([]byte(output.Content), want) {
			t.Errorf("rendered %s differs from independent expected bytes\ngot:\n%s\nwant:\n%s", output.Path, output.Content, want)
		}
		if output.SHA256 != sha256Hex([]byte(output.Content)) {
			t.Errorf("content hash for %s does not match rendered bytes", output.Path)
		}
		if output.Asset.Source != asset.SourceEmbedded.String() || output.Asset.RelativeName == "" || output.Asset.SHA256 == "" {
			t.Errorf("asset metadata for %s = %+v", output.Path, output.Asset)
		}
	}
	if after := snapshotRenderTarget(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("Render changed target: before=%v after=%v", before, after)
	}
}

func TestRender_RejectsIncompleteOrUnsafeFacts(t *testing.T) {
	useIsolatedAssetDirectories(t)
	valid := readRenderFixture(t)
	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{name: "missing approval", mutate: func(data []byte) []byte { return replaceJSONField(t, data, "__deleteApproval", "") }},
		{name: "missing output evidence", mutate: func(data []byte) []byte { return replaceJSONField(t, data, "OutputEvidence", "") }},
		{name: "missing operation evidence", mutate: func(data []byte) []byte { return replaceJSONField(t, data, "OperationSource", "") }},
		{name: "unapproved operation", mutate: func(data []byte) []byte { return replaceJSONField(t, data, "approval", "") }},
		{name: "approval explicitly declined", mutate: func(data []byte) []byte { return replaceApprovalStatus(t, data, "not approved") }},
		{name: "disapproved", mutate: func(data []byte) []byte { return replaceApprovalStatus(t, data, "disapproved") }},
		{name: "approval still pending", mutate: func(data []byte) []byte { return replaceApprovalStatus(t, data, "awaiting approval") }},
		{name: "approver control character", mutate: func(data []byte) []byte { return replaceApprovalApprover(t, data, "owner"+string(rune(0))) }},
		{name: "escaping source", mutate: func(data []byte) []byte { return replaceJSONField(t, data, "OperationSource", "../../outside.md") }},
		{name: "control character", mutate: func(data []byte) []byte { return replaceJSONField(t, data, "Purpose", "safe"+string(rune(0))+"unsafe") }},
		{name: "unknown field", mutate: func(data []byte) []byte {
			return bytes.Replace(data, []byte("{\n"), []byte("{\n  \"Unexpected\": \"not accepted\",\n"), 1)
		}},
		{name: "incorrect field case", mutate: func(data []byte) []byte {
			return bytes.Replace(data, []byte(`"DisplayName":`), []byte(`"displayname":`), 1)
		}},
		{name: "duplicate field", mutate: func(data []byte) []byte {
			return bytes.Replace(data, []byte("{\n"), []byte("{\n  \"DisplayName\": \"second\",\n"), 1)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.mutate(append([]byte(nil), valid...))
			if tc.name == "incorrect field case" {
				data = []byte(strings.Replace(string(data), "DisplayName", "displayname", 1))
			}
			factsPath := writeRenderFacts(t, data)
			if outputs, err := Render(t.TempDir(), factsPath, ""); !errors.Is(err, ErrInvalidRenderInput) || outputs != nil {
				t.Fatalf("Render = (%v, %v), want nil outputs and ErrInvalidRenderInput", outputs, err)
			}
		})
	}
}

func TestRender_EscapesMarkdownAndFencesInvocation(t *testing.T) {
	useIsolatedAssetDirectories(t)
	data := readRenderFixture(t)
	var facts map[string]any
	if err := json.Unmarshal(data, &facts); err != nil {
		t.Fatal(err)
	}
	facts["DisplayName"] = "Wordcount # heading [click](https://evil.invalid) <b> \"x\" {{.Unexpected}}"
	facts["Purpose"] = "first line\n# injected heading\n\nsecond line"
	facts["Invocation"] = "printf '" + strings.Repeat("`", 3) + "\\nsecond line'\nsh word-count.sh count sample.txt"
	mutated, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Render(t.TempDir(), writeRenderFacts(t, mutated), "")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	var overview, usage string
	for _, output := range outputs {
		switch output.Path {
		case "engineering/product/overview.md":
			overview = output.Content
		case "docs/reference/usage.md":
			usage = output.Content
		}
	}
	if strings.Contains(overview, "\n# injected heading") || strings.Contains(overview, "[click](") || strings.Contains(overview, "<b>") || strings.Contains(overview, "{{.Unexpected}}") {
		t.Fatalf("plain-text facts escaped Markdown context incompletely:\n%s", overview)
	}
	if !strings.Contains(overview, `\# heading`) || !strings.Contains(overview, `\[click\]`) || !strings.Contains(overview, `\<b\>`) || !strings.Contains(overview, "first line\\\n\\# injected heading\\\n\\\nsecond line") {
		t.Fatalf("expected Markdown metacharacters to be escaped:\n%s", overview)
	}
	invocation := "printf '" + strings.Repeat("`", 3) + "\\nsecond line'\nsh word-count.sh count sample.txt"
	fenced := strings.Repeat("`", 4) + "\n" + invocation + "\n" + strings.Repeat("`", 4)
	if !strings.Contains(usage, fenced) {
		t.Fatalf("invocation was not fenced with a delimiter longer than its backtick run:\n%s", usage)
	}
}

func TestRender_RejectsBadTemplateOverrides(t *testing.T) {
	root := t.TempDir()
	factsPath := filepath.Join(renderFixtureDir, "facts.json")
	for _, override := range []struct {
		name string
		body string
	}{
		{name: "unknown field", body: "{{.NotDeclared}}"},
		{name: "control action", body: "{{if .DisplayName}}unexpected{{end}}"},
		{name: "unknown action in named template", body: `{{define "extra"}}{{.Undeclared}}{{end}}`},
		{name: "malformed action", body: "{{.DisplayName"},
		{name: "control character", body: string(rune(0))},
	} {
		t.Run(override.name, func(t *testing.T) {
			useIsolatedAssetDirectories(t)
			writeAssetOverride(t, os.Getenv("XDG_CONFIG_HOME"), "templates/adoption/docs/README.md", override.body)
			if outputs, err := Render(root, factsPath, ""); !errors.Is(err, ErrInvalidRenderInput) || outputs != nil {
				t.Fatalf("Render = (%v, %v), want no partial outputs and ErrInvalidRenderInput", outputs, err)
			}
		})
	}
}

func TestRender_RejectsInvocationInHeadingOverride(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"heading", "### Invocation: {{.Invocation}}\n"},
		{"code fence", "```\n### Invocation and input\n\n{{.Invocation}}\n\nInput: text\n```\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			useIsolatedAssetDirectories(t)
			writeAssetOverride(t, os.Getenv("XDG_CONFIG_HOME"), "templates/adoption/docs/reference/usage.md", tc.body)
			outputs, err := Render(t.TempDir(), filepath.Join(renderFixtureDir, "facts.json"), "")
			if !errors.Is(err, ErrInvalidRenderInput) || outputs != nil {
				t.Fatalf("Render = (%v, %v), want invalid invocation context", outputs, err)
			}
		})
	}
}

func TestRender_RejectsUnreadableOverrideInsteadOfFallingBack(t *testing.T) {
	for _, kind := range []string{"directory", "dangling symlink"} {
		t.Run(kind, func(t *testing.T) {
			useIsolatedAssetDirectories(t)
			path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "lds", "templates", "adoption", "docs", "README.md")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			var err error
			if kind == "directory" {
				err = os.Mkdir(path, 0o755)
			} else {
				err = os.Symlink(filepath.Join(t.TempDir(), "missing.md"), path)
			}
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Render(t.TempDir(), filepath.Join(renderFixtureDir, "facts.json"), "")
			if !errors.Is(err, ErrInvalidRenderInput) || outputs != nil {
				t.Fatalf("Render = (%v, %v), want refusal of broken override", outputs, err)
			}
		})
	}
}

func readRenderFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(renderFixtureDir, "facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeRenderFacts(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "facts.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func replaceJSONField(t *testing.T, data []byte, key, value string) []byte {
	t.Helper()
	var facts map[string]any
	if err := json.Unmarshal(data, &facts); err != nil {
		t.Fatal(err)
	}
	switch key {
	case "__deleteApproval":
		delete(facts, "approval")
	case "approval":
		facts[key] = map[string]any{"approver": "owner", "approvedFacts": []string{"DisplayName", "Purpose", "CurrentCapability"}, "status": "approved"}
	default:
		facts[key] = value
	}
	mutated, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	return mutated
}

func replaceApprovalStatus(t *testing.T, data []byte, status string) []byte {
	t.Helper()
	var facts map[string]any
	if err := json.Unmarshal(data, &facts); err != nil {
		t.Fatal(err)
	}
	facts["approval"].(map[string]any)["status"] = status
	mutated, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	return mutated
}

func replaceApprovalApprover(t *testing.T, data []byte, approver string) []byte {
	t.Helper()
	var facts map[string]any
	if err := json.Unmarshal(data, &facts); err != nil {
		t.Fatal(err)
	}
	facts["approval"].(map[string]any)["approver"] = approver
	mutated, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	return mutated
}

func useIsolatedAssetDirectories(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func writeAssetOverride(t *testing.T, xdg, relative, content string) {
	t.Helper()
	path := filepath.Join(xdg, "lds", filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func snapshotRenderTarget(t *testing.T, root string) map[string]string {
	t.Helper()
	entries := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(relative)
		if entry.IsDir() {
			entries["dir:"+key] = ""
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries["file:"+key] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
