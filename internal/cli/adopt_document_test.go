package cli_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	rootassets "github.com/procrastivity/lds/assets"
)

func TestAdoptDocument_ReadOnlyTextJSONManifestAndUsage(t *testing.T) {
	embedded, err := rootassets.FS.ReadFile("flows/adopt.md")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	marker := filepath.Join(root, "keep.txt")
	if err := os.WriteFile(marker, []byte("leave this byte-exact\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	env := []string{"XDG_CONFIG_HOME=" + t.TempDir()}

	text := runInDir(t, root, env, "plumbing", "adopt")
	if text.exitCode != 0 || text.stderr != "" || text.stdout != string(embedded) {
		t.Fatalf("text result = %+v; want exact embedded Markdown", text)
	}

	jsonRun := runInDir(t, root, env, "plumbing", "adopt", "--json")
	if jsonRun.exitCode != 0 || jsonRun.stderr != "" {
		t.Fatalf("JSON result = %+v", jsonRun)
	}
	var payload struct {
		SchemaVersion int    `json:"schemaVersion"`
		Operation     string `json:"operation"`
		Status        string `json:"status"`
		Availability  string `json:"availability"`
		Asset         struct {
			RelativeName string `json:"relativeName"`
			Source       string `json:"source"`
			SHA256       string `json:"sha256"`
		} `json:"asset"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(jsonRun.stdout), &payload); err != nil {
		t.Fatalf("decode command JSON: %v; stdout=%q", err, jsonRun.stdout)
	}
	sum := sha256.Sum256(embedded)
	if payload.SchemaVersion != 1 || payload.Operation != "adopt" || payload.Status != "preparatory" || payload.Availability != "unavailable-for-execution" || payload.Asset.RelativeName != "flows/adopt.md" || payload.Asset.Source != "embedded" || payload.Asset.SHA256 != hex.EncodeToString(sum[:]) || payload.Content != string(embedded) {
		t.Fatalf("unexpected command JSON: %+v", payload)
	}

	manifestRun := runInDir(t, root, env, "manifest", "--json")
	if manifestRun.exitCode != 0 || manifestRun.stderr != "" {
		t.Fatalf("manifest result = %+v", manifestRun)
	}
	var manifest struct {
		Verbs []struct {
			Name         string          `json:"name"`
			Kind         string          `json:"kind"`
			OutputSchema json.RawMessage `json:"outputSchema"`
		} `json:"verbs"`
		Assets []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"assets"`
	}
	if err := json.Unmarshal([]byte(manifestRun.stdout), &manifest); err != nil {
		t.Fatalf("decode manifest JSON: %v", err)
	}
	var verbMatches, assetMatches int
	for _, verb := range manifest.Verbs {
		if verb.Name == "plumbing adopt" {
			verbMatches++
			if verb.Kind != "plumbing" || !json.Valid(verb.OutputSchema) || !strings.Contains(string(verb.OutputSchema), `"availability"`) {
				t.Errorf("manifest adoption verb = %+v", verb)
			}
		}
	}
	for _, asset := range manifest.Assets {
		if asset.Path == "flows/adopt.md" {
			assetMatches++
			if asset.SHA256 != hex.EncodeToString(sum[:]) {
				t.Errorf("manifest flow asset hash = %q, want %q", asset.SHA256, hex.EncodeToString(sum[:]))
			}
		}
	}
	if verbMatches != 1 || assetMatches != 1 {
		t.Fatalf("manifest entries for plumbing adopt/flow = %d/%d, want 1/1", verbMatches, assetMatches)
	}

	help := runInDir(t, root, env, "plumbing", "adopt", "--help")
	if help.exitCode != 0 || !strings.Contains(help.stdout, "preparation-only") || !strings.Contains(help.stdout, "does not run adoption") {
		t.Fatalf("help result = %+v", help)
	}
	usage := runInDir(t, root, env, "plumbing", "adopt", "extra")
	if usage.exitCode != 2 || usage.stdout != "" {
		t.Fatalf("usage result = %+v; want exit 2 without stdout", usage)
	}
	after, err := os.ReadFile(marker)
	if err != nil || string(after) != string(before) {
		t.Fatalf("document command changed target marker: bytes=%q err=%v", after, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "keep.txt" {
		t.Fatalf("document command changed target tree: entries=%v err=%v", entries, err)
	}
}

func TestAdoptDocument_OverrideIsLiveAfterInstallAndMalformedOverrideRefuses(t *testing.T) {
	root := t.TempDir()
	skills := filepath.Join(t.TempDir(), "skills")
	xdg := t.TempDir()
	env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills, "XDG_CONFIG_HOME=" + xdg}
	installed := runInDir(t, root, env, "install", "claude-code")
	if installed.exitCode != 0 || installed.stderr != "" {
		t.Fatalf("install result = %+v", installed)
	}
	skillDir := filepath.Join(skills, "lds")
	stampPath := filepath.Join(skillDir, ".lds-manifest-stamp.json")
	stampBefore, err := os.ReadFile(stampPath)
	if err != nil {
		t.Fatalf("read install stamp: %v", err)
	}
	skill, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skill), "plumbing adopt") || !strings.Contains(string(skill), "does not start adoption") || strings.Contains(string(skill), "# Prepare a create-only LDS documentation adoption") {
		t.Fatalf("generated skill does not project cautiously or copied the flow: %s", skill)
	}

	override := []byte("# Changed live adoption document\n\nPreparation only.\n")
	overridePath := filepath.Join(xdg, "lds", "flows", "adopt.md")
	if err := os.MkdirAll(filepath.Dir(overridePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overridePath, override, 0o600); err != nil {
		t.Fatal(err)
	}
	read := runInDir(t, root, env, "plumbing", "adopt", "--json")
	if read.exitCode != 0 || read.stderr != "" {
		t.Fatalf("live override result = %+v", read)
	}
	var payload struct {
		Status       string `json:"status"`
		Availability string `json:"availability"`
		Content      string `json:"content"`
		Asset        struct {
			Source string `json:"source"`
			SHA256 string `json:"sha256"`
		} `json:"asset"`
	}
	if err := json.Unmarshal([]byte(read.stdout), &payload); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(override)
	if payload.Status != "preparatory" || payload.Availability != "unavailable-for-execution" || payload.Asset.Source != "override" || payload.Asset.SHA256 != hex.EncodeToString(sum[:]) || payload.Content != string(override) {
		t.Fatalf("live override was not returned: %+v", payload)
	}

	doctor := runInDir(t, root, env, "doctor", "--json")
	if doctor.exitCode != 0 || !strings.Contains(doctor.stdout, `"state":"current"`) {
		t.Fatalf("doctor after post-install override = %+v; want installed tree current", doctor)
	}
	stampAfter, err := os.ReadFile(stampPath)
	if err != nil || string(stampAfter) != string(stampBefore) {
		t.Fatalf("post-install override changed install stamp: err=%v", err)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("document operations changed target project: entries=%v err=%v", entries, err)
	}

	if err := os.WriteFile(overridePath, []byte{0xff, 0xfe}, 0o600); err != nil {
		t.Fatal(err)
	}
	malformed := runInDir(t, root, env, "plumbing", "adopt", "--json")
	if malformed.exitCode != 1 || malformed.stdout != "" || parseErrorEnvelope(t, malformed.stderr).Error.Code != "validation.adopt-document-invalid" {
		t.Fatalf("malformed override result = %+v", malformed)
	}
}

func runInDir(t *testing.T, dir string, env []string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	cmd.Env = hermeticEnv(t, env)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running %v: %v", args, err)
		}
		exitCode = exitErr.ExitCode()
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), exitCode: exitCode}
}
