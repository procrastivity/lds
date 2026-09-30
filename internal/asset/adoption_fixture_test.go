package asset

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"

	rootassets "github.com/procrastivity/lds/assets"
)

const greenfieldFixture = "testdata/adoption/greenfield"

type greenfieldFacts struct {
	DisplayName             string
	Purpose                 string
	CurrentCapability       string
	DisplayNameSource       string
	PurposeSource           string
	CurrentCapabilitySource string
	Prerequisites           string
	OperationName           string
	OperationSummary        string
	Invocation              string
	Input                   string
	OutputEvidence          string
	OperationSource         string
	Limitations             string
	Approval                struct {
		Approver      string   `json:"approver"`
		ApprovedFacts []string `json:"approvedFacts"`
		Status        string   `json:"status"`
	} `json:"approval"`
}

type greenfieldCase struct {
	InitialTargetFiles []string `json:"initialTargetFiles"`
	FirstRun           struct {
		Creates                    []string `json:"creates"`
		NecessaryParentDirectories []string `json:"necessaryParentDirectories"`
	} `json:"firstRun"`
	SecondRun struct {
		Classifications []struct {
			Path  string `json:"path"`
			State string `json:"state"`
		} `json:"classifications"`
		Writes       int `json:"writes"`
		ChangedBytes int `json:"changedBytes"`
	} `json:"secondRun"`
}

func TestGreenfieldFixture_OperationAndApprovalEvidence(t *testing.T) {
	root := filepath.Join(greenfieldFixture)
	var facts greenfieldFacts
	readJSONFixture(t, filepath.Join(root, "facts.json"), &facts)

	if facts.Approval.Approver != "Simulated fixture owner" || !strings.Contains(facts.Approval.Status, "test-only") || !strings.Contains(facts.Approval.Status, "not a real project authorization") {
		t.Fatalf("fixture approval must be explicitly simulated and test-only: %+v", facts.Approval)
	}
	if !equalStrings(facts.Approval.ApprovedFacts, []string{"DisplayName", "Purpose", "CurrentCapability", "VerifiedUserOperation"}) {
		t.Fatalf("approved fixture facts = %v, want explicit approval for name, purpose, capability, and operation", facts.Approval.ApprovedFacts)
	}
	for _, source := range []string{facts.DisplayNameSource, facts.PurposeSource} {
		if !strings.Contains(source, "facts.json#/") || !strings.Contains(source, "Simulated fixture owner") || !strings.Contains(source, "test-only") {
			t.Errorf("fact source %q must cite facts.json and mark approval test-only", source)
		}
	}
	if !strings.Contains(facts.CurrentCapabilitySource, "word-count.sh") || !strings.Contains(facts.OperationSource, "word-count.sh") || !strings.Contains(facts.OperationSource, "sample.txt") {
		t.Fatalf("capability and operation sources must cite fixture code and input")
	}

	target := filepath.Join(root, "target")
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatalf("read greenfield target: %v", err)
	}
	if len(entries) != 2 || entries[0].Name() != "sample.txt" || entries[1].Name() != "word-count.sh" {
		t.Fatalf("starting target files = %v, want only sample.txt and word-count.sh", entries)
	}
	for _, source := range []string{"word-count.sh", "sample.txt"} {
		if _, err := os.Stat(filepath.Join(target, source)); err != nil {
			t.Fatalf("source reference %q is absent from target: %v", source, err)
		}
	}
	input, err := os.ReadFile(filepath.Join(target, "sample.txt"))
	if err != nil {
		t.Fatalf("read fixture input: %v", err)
	}
	if string(input) != "alpha beta gamma delta\n" || !strings.Contains(facts.Input, "alpha beta gamma delta") {
		t.Fatalf("fixture input bytes %q do not match documented input %q", input, facts.Input)
	}
	for _, path := range []string{"engineering", "docs"} {
		if _, err := os.Stat(filepath.Join(target, path)); !os.IsNotExist(err) {
			t.Fatalf("starting target unexpectedly has %s (stat err = %v)", path, err)
		}
	}

	command := strings.Fields(facts.Invocation)
	if len(command) == 0 {
		t.Fatal("fixture invocation must not be empty")
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = target
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run fixture operation: %v; stderr=%q", err, stderr.String())
	}
	if got, want := stdout.String(), "Word count: 4\n"; got != want {
		t.Fatalf("operation stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("operation stderr = %q, want empty", stderr.String())
	}
	wantEvidence := "Executed from target/; observed stdout bytes were `Word count: 4` followed by LF (0x0a), and exit status was 0."
	if facts.OutputEvidence != wantEvidence {
		t.Errorf("OutputEvidence = %q, want %q", facts.OutputEvidence, wantEvidence)
	}
}

func TestGreenfieldFixture_ExpectedFilesAndRunPlans(t *testing.T) {
	root := filepath.Join(greenfieldFixture)
	var facts greenfieldFacts
	readJSONFixture(t, filepath.Join(root, "facts.json"), &facts)
	var scenario greenfieldCase
	readJSONFixture(t, filepath.Join(root, "case.json"), &scenario)

	wantTargetFiles := []string{"target/sample.txt", "target/word-count.sh"}
	if !equalStrings(scenario.InitialTargetFiles, wantTargetFiles) {
		t.Fatalf("initialTargetFiles = %v, want %v", scenario.InitialTargetFiles, wantTargetFiles)
	}
	wantOutputs := []string{
		"engineering/README.md",
		"engineering/product/overview.md",
		"docs/README.md",
		"docs/reference/usage.md",
	}
	if !equalStrings(scenario.FirstRun.Creates, wantOutputs) {
		t.Fatalf("first-run creates = %v, want %v", scenario.FirstRun.Creates, wantOutputs)
	}
	wantDirectories := []string{"engineering/", "engineering/product/", "docs/", "docs/reference/"}
	if !equalStrings(scenario.FirstRun.NecessaryParentDirectories, wantDirectories) {
		t.Fatalf("first-run parent directories = %v, want %v", scenario.FirstRun.NecessaryParentDirectories, wantDirectories)
	}
	if len(scenario.SecondRun.Classifications) != len(wantOutputs) || scenario.SecondRun.Writes != 0 || scenario.SecondRun.ChangedBytes != 0 {
		t.Fatalf("second-run plan = %+v, want four identical-untouched classifications and zero writes/changed bytes", scenario.SecondRun)
	}
	for i, classification := range scenario.SecondRun.Classifications {
		if classification.Path != wantOutputs[i] || classification.State != "identical-untouched" {
			t.Errorf("second-run classification[%d] = %+v", i, classification)
		}
	}

	cases := []struct {
		assetPath string
		output    string
	}{
		{"templates/adoption/engineering/README.md", "engineering/README.md"},
		{"templates/adoption/engineering/product/overview.md", "engineering/product/overview.md"},
		{"templates/adoption/docs/README.md", "docs/README.md"},
		{"templates/adoption/docs/reference/usage.md", "docs/reference/usage.md"},
	}
	expectedRoot := filepath.Join(root, "expected")
	for _, tc := range cases {
		t.Run(tc.output, func(t *testing.T) {
			assetBytes, err := rootassets.FS.ReadFile(tc.assetPath)
			if err != nil {
				t.Fatalf("read template %q: %v", tc.assetPath, err)
			}
			tmpl, err := template.New(tc.assetPath).Parse(string(assetBytes))
			if err != nil {
				t.Fatalf("parse template %q: %v", tc.assetPath, err)
			}
			var rendered bytes.Buffer
			if err := tmpl.Execute(&rendered, facts); err != nil {
				t.Fatalf("render template %q: %v", tc.assetPath, err)
			}
			want, err := os.ReadFile(filepath.Join(expectedRoot, filepath.FromSlash(tc.output)))
			if err != nil {
				t.Fatalf("read independent expected output %q: %v", tc.output, err)
			}
			if !bytes.Equal(rendered.Bytes(), want) {
				t.Fatalf("rendered %q differs from independently-authored expected file", tc.output)
			}
		})
	}

	assertOnlyRelativeLink(t, filepath.Join(expectedRoot, "engineering", "README.md"), "product/overview.md")
	assertOnlyRelativeLink(t, filepath.Join(expectedRoot, "docs", "README.md"), "reference/usage.md")
}

func readJSONFixture(t *testing.T, path string, dest any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", path, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		t.Fatalf("decode fixture %q: %v", path, err)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func assertOnlyRelativeLink(t *testing.T, filePath, wantLink string) {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read expected navigation %q: %v", filePath, err)
	}
	links := regexp.MustCompile(`\]\(([^)]+)\)`).FindAllSubmatch(data, -1)
	if len(links) != 1 || string(links[0][1]) != wantLink {
		t.Fatalf("links in %q = %q, want only %q", filePath, links, wantLink)
	}
	target := filepath.Clean(filepath.Join(filepath.Dir(filePath), filepath.FromSlash(wantLink)))
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("navigation target %q from %q: %v", wantLink, filePath, err)
	}
}
