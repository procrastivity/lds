package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReconcileScanCLIJSONSuccessAndErrorModes(t *testing.T) {
	base := t.TempDir()
	leftRoot := filepath.Join(base, "left")
	rightRoot := filepath.Join(base, "right")
	leftRevision := makeReconcileSnapshot(t, leftRoot, map[string][]byte{
		"docs/left.md": []byte("left artifact\n"),
	})
	rightRevision := makeReconcileSnapshot(t, rightRoot, map[string][]byte{
		"docs/right.md":  []byte("right artifact\n"),
		"maps/review.md": []byte("uninterpreted map\n"),
	})
	pairPath := filepath.Join(base, "pair.json")
	pair := map[string]any{
		"schemaVersion": 1,
		"pairId":        "fixture:cli-scan:v1",
		"left":          map[string]any{"projectId": "left", "revision": leftRevision},
		"right":         map[string]any{"projectId": "right", "revision": rightRevision},
		"scope": map[string]any{
			"leftPaths":  []string{"docs/left-missing.md", "docs/left.md"},
			"rightPaths": []string{"docs/right.md"},
			"candidates": []any{map[string]any{
				"candidateKey": "topic:cli-scan",
				"leftPaths":    []string{"docs/left-missing.md", "docs/left.md"},
				"rightPaths":   []string{"docs/right.md"},
			}},
		},
	}
	writeReconcilePair(t, pairPath, pair)
	args := []string{
		"plumbing", "reconcile", "scan",
		"--pair", pairPath,
		"--left-root", leftRoot,
		"--right-root", rightRoot,
		"--map", filepath.Join(rightRoot, "maps", "review.md"),
	}

	first := run(t, nil, args...)
	if first.exitCode != 0 || first.stderr != "" || first.stdout == "" {
		t.Fatalf("scan success: exit=%d stdout=%q stderr=%q", first.exitCode, first.stdout, first.stderr)
	}
	var response struct {
		SchemaVersion int    `json:"schemaVersion"`
		Operation     string `json:"operation"`
		OK            bool   `json:"ok"`
		Candidates    []struct {
			CandidateKey   string `json:"candidateKey"`
			CandidateID    string `json:"candidateId"`
			EvidenceDigest string `json:"evidenceDigest"`
			EvidenceStatus string `json:"evidenceStatus"`
			Confidence     string `json:"confidence"`
			Evidence       []struct {
				Kind string `json:"kind"`
				Key  string `json:"key"`
			} `json:"evidence"`
			Left []struct {
				Path    string          `json:"path"`
				Present bool            `json:"present"`
				SHA256  json.RawMessage `json:"sha256"`
			} `json:"left"`
			JournalRevision       *int            `json:"journalRevision"`
			JournalEvidenceDigest json.RawMessage `json:"journalEvidenceDigest"`
			LastDecision          json.RawMessage `json:"lastDecision"`
		} `json:"candidates"`
		JournalPath string `json:"journalPath"`
		Maps        []struct {
			Path string `json:"path"`
		} `json:"maps"`
	}
	if err := json.Unmarshal([]byte(first.stdout), &response); err != nil {
		t.Fatalf("scan success is not JSON: %v; stdout=%q", err, first.stdout)
	}
	if response.SchemaVersion != 1 || response.Operation != "reconcile scan" || !response.OK || len(response.Candidates) != 1 {
		t.Fatalf("unexpected scan envelope: %+v", response)
	}
	candidate := response.Candidates[0]
	if candidate.CandidateKey != "topic:cli-scan" || !strings.HasPrefix(candidate.CandidateID, "cand-v1-") || !strings.HasPrefix(candidate.EvidenceDigest, "sha256:") || candidate.EvidenceStatus != "not-compared" || candidate.Confidence != "not-assessed" {
		t.Fatalf("unexpected candidate projection: %+v", candidate)
	}
	if len(candidate.Evidence) != 4 || candidate.Evidence[2].Key != "maps/review.md" || candidate.Evidence[2].Kind != "artifact" {
		t.Fatalf("scan omitted raw map evidence: %+v", candidate.Evidence)
	}
	if len(candidate.Left) != 2 || candidate.Left[0].Path != "docs/left-missing.md" || candidate.Left[0].Present || string(candidate.Left[0].SHA256) != "null" {
		t.Fatalf("missing path projection must use present:false and sha256:null: %+v", candidate.Left)
	}
	if len(response.Maps) != 1 || response.Maps[0].Path != "maps/review.md" {
		t.Fatalf("scan map observation = %+v", response.Maps)
	}
	second := run(t, nil, args...)
	if second.exitCode != 0 || second.stdout != first.stdout || second.stderr != "" {
		t.Fatalf("repeated scan changed success output: first=%+v second=%+v", first, second)
	}
	var noJournalObject map[string]json.RawMessage
	if err := json.Unmarshal([]byte(first.stdout), &noJournalObject); err != nil {
		t.Fatal(err)
	}
	if _, present := noJournalObject["journalPath"]; present || candidate.JournalRevision != nil || candidate.JournalEvidenceDigest != nil || candidate.LastDecision != nil {
		t.Fatalf("no-journal scan emitted comparison metadata: %+v", noJournalObject)
	}

	journalPath := filepath.Join(base, "journal.json")
	emptyJournal, err := json.Marshal(map[string]any{
		"schemaVersion": 1, "pairId": "fixture:cli-scan:v1", "leftProjectId": "left", "rightProjectId": "right", "revision": 0, "events": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journalPath, emptyJournal, 0o600); err != nil {
		t.Fatal(err)
	}
	journalBefore, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	compared := run(t, nil, append(append([]string(nil), args...), "--journal", journalPath)...)
	if compared.exitCode != 0 || compared.stderr != "" {
		t.Fatalf("empty-journal scan: exit=%d stdout=%q stderr=%q", compared.exitCode, compared.stdout, compared.stderr)
	}
	var comparedResponse struct {
		JournalPath string `json:"journalPath"`
		Candidates  []struct {
			EvidenceStatus        string          `json:"evidenceStatus"`
			JournalRevision       *int            `json:"journalRevision"`
			JournalEvidenceDigest json.RawMessage `json:"journalEvidenceDigest"`
			LastDecision          json.RawMessage `json:"lastDecision"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(compared.stdout), &comparedResponse); err != nil {
		t.Fatalf("compared scan is not JSON: %v; stdout=%q", err, compared.stdout)
	}
	if comparedResponse.JournalPath != journalPath || len(comparedResponse.Candidates) != 1 || comparedResponse.Candidates[0].EvidenceStatus != "new" || comparedResponse.Candidates[0].JournalRevision == nil || *comparedResponse.Candidates[0].JournalRevision != 0 || string(comparedResponse.Candidates[0].JournalEvidenceDigest) != "null" || string(comparedResponse.Candidates[0].LastDecision) != "null" {
		t.Fatalf("empty-journal comparison fields = %+v", comparedResponse)
	}
	journalAfter, err := os.ReadFile(journalPath)
	if err != nil || !bytes.Equal(journalBefore, journalAfter) {
		t.Fatalf("read-only scan changed the journal bytes: err=%v before=%q after=%q", err, journalBefore, journalAfter)
	}

	malformed := []byte(`{"schemaVersion":1,"schemaVersion":1}`)
	if err := os.WriteFile(pairPath, malformed, 0o600); err != nil {
		t.Fatal(err)
	}
	jsonArgs := append(append([]string(nil), args...), "--json")
	jsonFailure := run(t, nil, jsonArgs...)
	if jsonFailure.exitCode != 1 || jsonFailure.stdout != "" {
		t.Fatalf("JSON pair failure: exit=%d stdout=%q stderr=%q", jsonFailure.exitCode, jsonFailure.stdout, jsonFailure.stderr)
	}
	if code := parseErrorEnvelope(t, jsonFailure.stderr).Error.Code; code != "validation.reconcile-pair-invalid" {
		t.Fatalf("JSON pair error code=%q", code)
	}
	textFailure := run(t, nil, args...)
	if textFailure.exitCode != jsonFailure.exitCode || textFailure.stdout != "" || !strings.HasPrefix(textFailure.stderr, "lds: plumbing reconcile scan: ") {
		t.Fatalf("text pair failure: exit=%d stdout=%q stderr=%q", textFailure.exitCode, textFailure.stdout, textFailure.stderr)
	}

	usageArgs := []string{
		"plumbing", "reconcile", "scan", "--pair", pairPath,
		"--left-root", leftRoot, "--right-root", rightRoot, "--json",
	}
	usage := run(t, nil, usageArgs...)
	if usage.exitCode != 2 || usage.stdout != "" || strings.HasPrefix(strings.TrimSpace(usage.stderr), "{") {
		t.Fatalf("usage error must remain human-mode under --json: exit=%d stdout=%q stderr=%q", usage.exitCode, usage.stdout, usage.stderr)
	}
}

func makeReconcileSnapshot(t *testing.T, root string, files map[string][]byte) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runReconcileGit(t, root, "init", "-q")
	runReconcileGit(t, root, "add", "-A")
	runReconcileGit(t, root, "commit", "-q", "-m", "fixture")
	return strings.TrimSpace(string(runReconcileGit(t, root, "rev-parse", "HEAD")))
}

func writeReconcilePair(t *testing.T, path string, pair any) {
	t.Helper()
	data, err := json.Marshal(pair)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func runReconcileGit(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = reconcileFixtureGitEnv()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return bytes.TrimSpace(output)
}

func reconcileFixtureGitEnv() []string {
	var env []string
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if strings.HasPrefix(key, "GIT_") {
			continue
		}
		env = append(env, item)
	}
	return append(env,
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_AUTHOR_NAME=Fixture",
		"GIT_AUTHOR_EMAIL=fixture@example.invalid",
		"GIT_COMMITTER_NAME=Fixture",
		"GIT_COMMITTER_EMAIL=fixture@example.invalid",
		"GIT_AUTHOR_DATE=2000-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2000-01-01T00:00:00Z",
	)
}
