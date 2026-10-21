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
