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
	if !strings.Contains(string(skill), "name: lds") ||
		!strings.Contains(string(skill), "Use when inspecting or maintaining the LDS CLI") ||
		!strings.Contains(string(skill), "lds plumbing adopt --json") ||
		!strings.Contains(string(skill), "unavailable-for-execution` availability") ||
		!strings.Contains(string(skill), "sole source of workflow sequencing") ||
		!strings.Contains(string(skill), "display the entire exact plan") ||
		!strings.Contains(string(skill), "veto` is false and `planDigest` is") ||
		!strings.Contains(string(skill), "fresh human decision for that completely displayed") ||
		!strings.Contains(string(skill), "findings/skips alone do not veto") ||
		!strings.Contains(string(skill), "Keep any plan artifact") ||
		!strings.Contains(string(skill), "not behavioral enforcement") ||
		strings.Contains(string(skill), "# Prepare a create-only LDS documentation adoption") {
		t.Fatalf("generated skill does not project cautiously or copied the flow: %s", skill)
	}
	if strings.Contains(string(skill), "| `lds adopt <root>") || strings.Contains(string(skill), "root adoption form has no paired skill workflow yet") {
		t.Fatalf("generated skill must not project the root LLM form or stale description: %s", skill)
	}
	orderedItems := 0
	for _, line := range strings.Split(string(skill), "\n") {
		line = strings.TrimSpace(line)
		if len(line) >= 3 && line[0] >= '1' && line[0] <= '9' && line[1] == '.' && line[2] == ' ' {
			orderedItems++
		}
	}
	if orderedItems != 1 {
		t.Fatalf("generated skill defines %d numbered steps; only the live-flow fetch may be procedural, with all workflow steps delegated to its current content", orderedItems)
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

func TestMaintenanceFlow_LiveOverrideAndInstalledGuidance(t *testing.T) {
	for _, operation := range []string{"assess", "update", "reconcile"} {
		t.Run(operation, func(t *testing.T) {
			args := []string{"plumbing", operation}
			reader := "lds plumbing " + operation + " --json"
			if operation == "reconcile" {
				args = append(args, "flow")
				reader = "lds plumbing reconcile flow --json"
			}
			root, skills, xdg := t.TempDir(), t.TempDir(), t.TempDir()
			env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills, "XDG_CONFIG_HOME=" + xdg}
			installed := runInDir(t, root, env, "install", "claude-code")
			if installed.exitCode != 0 || installed.stderr != "" {
				t.Fatalf("install: %+v", installed)
			}
			skillDir := filepath.Join(skills, "lds")
			before := renderTargetSnapshot(t, skillDir)
			skill, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(skill), reader) || !strings.Contains(string(skill), "--observed") || strings.Contains(string(skill), "| `lds "+operation+" <root>") || strings.Contains(string(skill), "| `lds reconcile report") {
				t.Fatalf("generated guidance lost the live reader/observed route or projected the root workflow: %s", skill)
			}
			embedded, err := rootassets.FS.ReadFile("flows/" + operation + ".md")
			if err != nil {
				t.Fatal(err)
			}
			text := runInDir(t, root, env, args...)
			if text.exitCode != 0 || text.stderr != "" || text.stdout != string(embedded) {
				t.Fatalf("embedded flow: %+v", text)
			}
			override := []byte("# Locally reviewed assessment flow\n\nReport only, no edits.\n")
			path := filepath.Join(xdg, "lds", "flows", operation+".md")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, override, 0o600); err != nil {
				t.Fatal(err)
			}
			read := runInDir(t, root, env, append(args, "--json")...)
			var payload struct {
				Operation, Status, Availability, Content string
				Asset                                    struct{ Source, SHA256 string }
			}
			if err := json.Unmarshal([]byte(read.stdout), &payload); err != nil {
				t.Fatalf("live flow: %+v; %v", read, err)
			}
			sum := sha256.Sum256(override)
			if read.exitCode != 0 || read.stderr != "" || payload.Operation != operation || payload.Status != "preparatory" || payload.Availability != "unavailable-for-execution" || payload.Asset.Source != "override" || payload.Asset.SHA256 != hex.EncodeToString(sum[:]) || payload.Content != string(override) {
				t.Fatalf("post-install flow is not live: %+v", payload)
			}
			doctor := runInDir(t, root, env, "doctor", "--json")
			if doctor.exitCode != 0 || !strings.Contains(doctor.stdout, `"state":"current"`) || !equalRenderSnapshots(before, renderTargetSnapshot(t, skillDir)) {
				t.Fatalf("live override changed the installed tree: %+v", doctor)
			}
			if err := os.WriteFile(path, []byte{0xff}, 0o600); err != nil {
				t.Fatal(err)
			}
			bad := runInDir(t, root, env, append(args, "--json")...)
			if bad.exitCode != 1 || bad.stdout != "" || parseErrorEnvelope(t, bad.stderr).Error.Code != "validation."+operation+"-document-invalid" {
				t.Fatalf("malformed live flow accepted: %+v", bad)
			}
			if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
				t.Fatalf("reader/installation changed project: %v %v", entries, err)
			}
		})
	}
}

func TestProjectAssets_RuntimeSelectionAndAdministrativeIsolation(t *testing.T) {
	a, b, xdg, skills := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	nested := filepath.Join(a, "nested")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	env := []string{"XDG_CONFIG_HOME=" + xdg, "LDS_CLAUDE_SKILLS_DIR=" + skills, "LDS_PROJECT_ROOT=" + b}
	flows := [][]string{{"adopt"}, {"assess"}, {"update"}, {"reconcile", "flow"}}
	for i, name := range []string{"adopt", "assess", "update", "reconcile"} {
		for _, item := range []struct{ dir, marker string }{
			{filepath.Join(a, ".lds/assets"), "Project A"},
			{filepath.Join(b, ".lds/assets"), "Project B"},
			{filepath.Join(xdg, "lds"), "User"},
		} {
			writeRenderAsset(t, item.dir, "flows/"+name+".md", []byte("# "+item.marker+" "+name+"\n"))
		}
		flows[i] = append([]string{"plumbing"}, flows[i]...)
	}
	// Administrative generators must not consume even valid project skill assets.
	writeRenderAsset(t, filepath.Join(a, ".lds/assets"), "templates/skills/claude-code/judgment.md", []byte("DO NOT INSTALL PROJECT GUIDANCE\n"))
	writeRenderAsset(t, filepath.Join(a, ".lds/assets"), "config.default.yaml", []byte("invalid: [project config\n"))
	manifest := runInDir(t, nested, env, "manifest", "--json")
	installed := runInDir(t, nested, env, "install", "claude-code", "--json")
	if manifest.exitCode != 0 || installed.exitCode != 0 {
		t.Fatalf("baseline manifest/install: %+v / %+v", manifest, installed)
	}
	skillDir := filepath.Join(skills, "lds")
	before := renderTargetSnapshot(t, skillDir)
	checkFlow := func(args []string, selected, marker, source string) {
		t.Helper()
		command := append([]string{}, args...)
		if selected != "" {
			command = append(command, "--project-root", selected)
		}
		r := runInDir(t, nested, env, append(command, "--json")...)
		var payload struct {
			Content string
			Asset   struct{ Source, SHA256 string }
		}
		if err := json.Unmarshal([]byte(r.stdout), &payload); err != nil || r.exitCode != 0 || r.stderr != "" {
			t.Fatalf("flow %v: %+v, %v", command, r, err)
		}
		if !strings.HasPrefix(payload.Content, "# "+marker+" ") || payload.Asset.Source != source {
			t.Fatalf("flow %v = %+v, want %s/%s", command, payload, marker, source)
		}
		sum := sha256.Sum256([]byte(payload.Content))
		if payload.Asset.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatal("flow provenance does not match displayed bytes")
		}
	}
	for _, flow := range flows {
		checkFlow(flow, a, "Project A", "project")
		checkFlow(flow, b, "Project B", "project")
		checkFlow(flow, "..", "Project A", "project")
		checkFlow(flow, ".", "User", "override") // No ancestor discovery.
		checkFlow(flow, "", "User", "override")  // No cwd/env selection.
	}
	writeRenderAsset(t, filepath.Join(a, ".lds/assets"), "flows/adopt.md", []byte("# Edited A adopt\n"))
	checkFlow(flows[0], a, "Edited A", "project")
	if err := os.Remove(filepath.Join(a, ".lds/assets/flows/adopt.md")); err != nil {
		t.Fatal(err)
	}
	checkFlow(flows[0], a, "User", "override")
	for _, selected := range []string{a, b, filepath.Join(a, "missing")} {
		got := runInDir(t, nested, env, "--project-root", selected, "manifest", "--json")
		reinstall := runInDir(t, nested, env, "--project-root", selected, "install", "claude-code", "--json")
		doctor := runInDir(t, nested, env, "--project-root", selected, "doctor", "--json")
		if got.exitCode != 0 || got.stdout != manifest.stdout {
			t.Fatalf("project selection changed manifest for %s: exit=%d stderr=%s", selected, got.exitCode, got.stderr)
		}
		if reinstall.exitCode != 0 || !strings.Contains(reinstall.stdout, `"status":"current"`) || doctor.exitCode != 0 || !strings.Contains(doctor.stdout, `"findings":[]`) || !equalRenderSnapshots(before, renderTargetSnapshot(t, skillDir)) {
			t.Fatalf("project selection changed install/doctor state: %+v / %+v", reinstall, doctor)
		}
	}
	for _, flow := range flows {
		bad := runInDir(t, nested, env, append(append([]string{}, flow...), "--project-root", filepath.Join(a, "missing"), "--json")...)
		if bad.exitCode != 3 || bad.stdout != "" || parseErrorEnvelope(t, bad.stderr).Error.Code != "refusal.project-asset" {
			t.Fatalf("missing project root did not refuse: %+v", bad)
		}
	}
	unsafe := filepath.Join(a, ".lds/assets/flows/adopt.md")
	if err := os.Symlink(filepath.Join(b, ".lds/assets/flows/adopt.md"), unsafe); err != nil {
		t.Fatal(err)
	}
	bad := runInDir(t, nested, env, "--project-root", a, "plumbing", "adopt", "--json")
	if bad.exitCode != 3 || bad.stdout != "" || parseErrorEnvelope(t, bad.stderr).Error.Code != "refusal.project-asset" {
		t.Fatalf("unsafe project flow fell through: %+v", bad)
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
