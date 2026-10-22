//go:build linux

package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfflineReconcileReportCLI(t *testing.T) {
	base := t.TempDir()
	left, right := filepath.Join(base, "left"), filepath.Join(base, "right")
	lrev := makeReconcileSnapshot(t, left, map[string][]byte{"left.md": []byte("left\n")})
	rrev := makeReconcileSnapshot(t, right, map[string][]byte{"right.md": []byte("right\n"), "map.md": []byte("unparsed map\n")})
	pair := filepath.Join(base, "pair.json")
	writeReconcilePair(t, pair, map[string]any{
		"schemaVersion": 1, "pairId": "fixture:cli-report:v1",
		"left":  map[string]any{"projectId": "left", "revision": lrev},
		"right": map[string]any{"projectId": "right", "revision": rrev},
		"scope": map[string]any{"leftPaths": []string{"left.md"}, "rightPaths": []string{"right.md"}, "candidates": []any{}},
	})
	review := filepath.Join(base, "review.json")
	if err := os.WriteFile(review, []byte(`{"schemaVersion":1,"purpose":"Offline review","reviewer":"fixture","at":"2026-10-21T00:00:00Z","selections":[],"claims":[],"questions":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(base, "report.md")
	flags := []string{"--pair", pair, "--left-root", left, "--right-root", right, "--map", filepath.Join(right, "map.md"), "--review", review, "--output", output}
	rootArgs := append([]string{"reconcile", "report", "--json"}, flags...)
	first := run(t, []string{"OPENCODE_API_KEY=", "ANTHROPIC_API_KEY=", "OPENAI_API_KEY="}, rootArgs...)
	if first.exitCode != 0 || first.stderr != "" {
		t.Fatalf("offline report: %+v", first)
	}
	var result struct {
		Status       string `json:"status"`
		ReportPath   string `json:"reportPath"`
		ReportSHA256 string `json:"reportSha256"`
		Assistance   bool   `json:"assistance"`
	}
	if err := json.Unmarshal([]byte(first.stdout), &result); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	if result.Status != "review-required" || result.ReportPath != output || result.Assistance || result.ReportSHA256 != hex.EncodeToString(hash[:]) {
		t.Fatalf("wrong summary: %+v", result)
	}
	second := run(t, nil, rootArgs...)
	if second.exitCode != 3 || second.stdout != "" || parseErrorEnvelope(t, second.stderr).Error.Code != "refusal.reconcile-report-output" {
		t.Fatalf("occupied report: %+v", second)
	}
	flags[len(flags)-1] = filepath.Join(base, "plumbing-report.md")
	human := run(t, nil, append([]string{"plumbing", "reconcile", "report"}, flags...)...)
	if human.exitCode != 0 || human.stderr != "" || !strings.Contains(human.stdout, "Review required:") {
		t.Fatalf("plumbing human summary: %+v", human)
	}
	missing := run(t, nil, "reconcile", "report", "--json")
	if missing.exitCode != 2 || missing.stdout != "" {
		t.Fatalf("missing flags: %+v", missing)
	}
}

func TestReconciliationOriginalCaptureArtifactRoute(t *testing.T) {
	setup := newJournalCLISetup(t)
	inputs := t.TempDir()
	write := func(path, value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	review, output := filepath.Join(inputs, "review.json"), filepath.Join(inputs, "report.md")
	write(review, `{"schemaVersion":1,"purpose":"Review","reviewer":"fixture","at":"2026-10-22T00:00:00Z","selections":[],"claims":[],"questions":[]}`)
	flags := []string{"--pair", setup.pairPath, "--left-root", setup.leftRoot, "--right-root", setup.rightRoot, "--map", filepath.Join(setup.rightRoot, "maps/a.md"), "--review", review, "--output", output, "--json"}
	before := assessmentTree(t, setup.base) + assessmentTree(t, inputs)
	collected := run(t, []string{"OPENCODE_API_KEY="}, append([]string{"plumbing", "reconcile", "collect"}, flags...)...)
	if collected.exitCode != 0 || collected.stderr != "" || assessmentTree(t, setup.base)+assessmentTree(t, inputs) != before {
		t.Fatalf("capture has effects: %+v", collected)
	}
	observed, draft, receipt := filepath.Join(inputs, "observed.json"), filepath.Join(inputs, "draft.json"), filepath.Join(inputs, "receipt.json")
	write(observed, collected.stdout)
	write(draft, `{"schemaVersion":1,"claims":[],"questions":["No semantic claim without selected text"]}`)
	write(receipt, `{"provider":"OpenCode Go","model":"deepseek-v4-flash","flowSource":"embedded","flowSha256":"`+strings.Repeat("a", 64)+`","systemSha256":"`+strings.Repeat("b", 64)+`","userSha256":"`+strings.Repeat("c", 64)+`","systemBytes":100,"userBytes":200,"consentAt":"2026-10-22T00:00:00Z"}`)
	before = assessmentTree(t, setup.base) + assessmentTree(t, inputs)
	artifactFlags := []string{"--observed", observed, "--draft", draft, "--receipt", receipt}
	// Changing only the future output filename must not replace the original review baseline.
	changed := append([]string{}, flags...)
	for i, value := range changed {
		if value == output {
			changed[i] = output + ".changed"
		}
	}
	stale := run(t, nil, append(append([]string{"plumbing", "reconcile", "report"}, changed...), artifactFlags...)...)
	if stale.exitCode != 3 || stale.stdout != "" || parseErrorEnvelope(t, stale.stderr).Error.Code != "refusal.reconcile-report-stale" {
		t.Fatalf("output baseline changed: %+v", stale)
	}
	missing := run(t, nil, append(append([]string{"plumbing", "reconcile", "report"}, flags...), "--observed", observed)...)
	if missing.exitCode != 2 || missing.stdout != "" {
		t.Fatalf("incomplete artifact route: %+v", missing)
	}
	// The saved observation itself must be a no-follow external regular file.
	link := observed + ".link"
	if err := os.Symlink(observed, link); err != nil {
		t.Fatal(err)
	}
	badFlags := []string{"--observed", link, "--draft", draft, "--receipt", receipt}
	bad := run(t, nil, append(append([]string{"plumbing", "reconcile", "report"}, flags...), badFlags...)...)
	if bad.exitCode != 1 || bad.stdout != "" || parseErrorEnvelope(t, bad.stderr).Error.Code != "validation.reconcile-report-path" {
		t.Fatalf("symlink observation: %+v", bad)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if assessmentTree(t, setup.base)+assessmentTree(t, inputs) != before {
		t.Fatal("refusals changed inputs/output")
	}
	assisted := run(t, []string{"OPENCODE_API_KEY="}, append(append([]string{"plumbing", "reconcile", "report"}, flags...), artifactFlags...)...)
	if assisted.exitCode != 0 || assisted.stderr != "" || !strings.Contains(assisted.stdout, `"assistance":true`) {
		t.Fatalf("deterministic authored-receipt report: %+v", assisted)
	}
	data, err := os.ReadFile(output)
	if err != nil || !strings.Contains(string(data), "attributed, not authenticated") || !strings.Contains(string(data), "No semantic claim without selected text") {
		t.Fatalf("receipt misrepresented as verified: %v", err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}
	if assessmentTree(t, setup.base)+assessmentTree(t, inputs) != before {
		t.Fatal("artifact report changed inputs")
	}
	// Root assistance cannot be driven through piped input or combined with saved-response mode.
	noninteractive := run(t, nil, append([]string{"reconcile", "report", "--assist"}, flags...)...)
	if noninteractive.exitCode != 3 || noninteractive.stdout != "" || parseErrorEnvelope(t, noninteractive.stderr).Error.Code != "refusal.interactive-required" {
		t.Fatalf("noninteractive assistance: %+v", noninteractive)
	}
	mixed := run(t, nil, append(append([]string{"reconcile", "report", "--assist"}, flags...), artifactFlags...)...)
	if mixed.exitCode != 2 || mixed.stdout != "" {
		t.Fatalf("mixed modes: %+v", mixed)
	}
}
