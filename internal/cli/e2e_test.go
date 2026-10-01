// End-to-end tests for the chassis: exit codes, error shape, and the
// manifest/install/uninstall/doctor loop, exercised through the actual
// built binary rather than in-process, so Cobra's own argument-parsing
// path is covered along with the exitcode/ldserr machinery (C2.7).
package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	rootassets "github.com/procrastivity/lds/assets"
	"github.com/procrastivity/lds/internal/adoption"
)

var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "lds-e2e")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}
	binPath = filepath.Join(binDir, "lds")

	build := exec.Command("go", "build", "-o", binPath, "github.com/procrastivity/lds/cmd/lds")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building lds for e2e tests: %v\n%s", err, out)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

type result struct {
	stdout   string
	stderr   string
	exitCode int
}

// hermeticEnv is the process environment plus the caller's overrides, with
// every harness's skills-dir seam defaulted to its own per-test temp dir
// when the caller does not set it — the suite must never read this host's
// real skill installs (C2.7; wip found this live: a stamped tree from an
// older build failed doctor inside tests that never mentioned skills).
// Add each new harness's seam var to this list when adding a target.
func hermeticEnv(t *testing.T, env []string) []string {
	t.Helper()
	out := append(os.Environ(), env...)
	for _, name := range []string{"LDS_CLAUDE_SKILLS_DIR"} {
		set := false
		for _, e := range env {
			if strings.HasPrefix(e, name+"=") {
				set = true
				break
			}
		}
		if !set {
			out = append(out, name+"="+t.TempDir())
		}
	}
	return out
}

func run(t *testing.T, env []string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = hermeticEnv(t, env)
	var stdout, stderr bytes.Buffer
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

func TestUsageError_BadFlag(t *testing.T) {
	r := run(t, nil, "--bogus")
	if r.exitCode != 2 {
		t.Fatalf("exit code = %d, want 2 (usage error); stderr=%q", r.exitCode, r.stderr)
	}
	if r.stdout != "" {
		t.Fatalf("stdout = %q, want empty on failure", r.stdout)
	}
	if !strings.HasPrefix(r.stderr, "lds: ") {
		t.Fatalf("stderr = %q, want it to start with %q", r.stderr, "lds: ")
	}
}

func TestUsageError_UnknownCommand(t *testing.T) {
	r := run(t, nil, "no-such-command")
	if r.exitCode != 2 || r.stdout != "" || !strings.HasPrefix(r.stderr, "lds: ") {
		t.Fatalf("unknown command: exit=%d stdout=%q stderr=%q; want usage 2, empty stdout, text diagnostic", r.exitCode, r.stdout, r.stderr)
	}
}

func TestHelpSeparatesPorcelainAndPlumbing(t *testing.T) {
	root := run(t, nil, "--help")
	if root.exitCode != 0 || root.stderr != "" {
		t.Fatalf("root help: exit=%d stderr=%q", root.exitCode, root.stderr)
	}
	for _, name := range []string{"doctor", "install", "manifest", "plumbing", "uninstall", "version"} {
		if !strings.Contains(root.stdout, "\n  "+name) {
			t.Errorf("root help omits root command %q: %s", name, root.stdout)
		}
	}
	for _, leaf := range []string{"inventory", "validate", "render", "plan", "apply"} {
		if strings.Contains(root.stdout, "\n  "+leaf+" ") {
			t.Errorf("root help leaks plumbing docs leaf %q: %s", leaf, root.stdout)
		}
	}

	plumbing := run(t, nil, "plumbing", "--help")
	if plumbing.exitCode != 0 || plumbing.stderr != "" || !strings.Contains(plumbing.stdout, "\n  lds plumbing ") {
		t.Fatalf("plumbing help: exit=%d stdout=%q stderr=%q", plumbing.exitCode, plumbing.stdout, plumbing.stderr)
	}
	for _, member := range []string{"doctor", "manifest", "docs", "reconcile"} {
		if !strings.Contains(plumbing.stdout, "\n  "+member) {
			t.Errorf("plumbing help omits canonical member %q: %s", member, plumbing.stdout)
		}
	}
	if strings.Contains(plumbing.stdout, "inventory") || strings.Contains(plumbing.stdout, "apply") {
		t.Fatalf("plumbing group help leaks nested docs leaves: %s", plumbing.stdout)
	}
	docs := run(t, nil, "plumbing", "docs", "--help")
	if docs.exitCode != 0 || docs.stderr != "" || !strings.Contains(docs.stdout, "\n  lds plumbing docs ") {
		t.Fatalf("plumbing docs help: exit=%d stdout=%q stderr=%q", docs.exitCode, docs.stdout, docs.stderr)
	}
	for _, leaf := range []string{"apply", "inventory", "plan", "render", "validate"} {
		if !strings.Contains(docs.stdout, "\n  "+leaf) {
			t.Errorf("plumbing docs help omits %q: %s", leaf, docs.stdout)
		}
	}
	for _, path := range []string{
		"plumbing adopt", "plumbing doctor", "plumbing manifest", "plumbing reconcile scan",
		"plumbing docs apply", "plumbing docs inventory", "plumbing docs plan",
		"plumbing docs render", "plumbing docs validate",
	} {
		args := strings.Fields(path)
		helpArgs := append(append([]string(nil), args...), "--help")
		help := run(t, nil, helpArgs...)
		if help.exitCode != 0 || help.stderr != "" || !strings.Contains(help.stdout, "\n  lds "+path+" ") {
			t.Errorf("canonical help for %q is not qualified: %+v", path, help)
		}
	}
	for _, leaf := range []string{"apply", "inventory", "plan", "render", "validate"} {
		legacy := run(t, nil, "docs", leaf, "--help")
		if legacy.exitCode != 2 || legacy.stdout != "" {
			t.Errorf("unaliased root docs spelling %q was accepted: %+v", leaf, legacy)
		}
	}

	for _, pair := range []struct{ alias, canonical string }{
		{"manifest", "plumbing manifest"},
		{"doctor", "plumbing doctor"},
	} {
		aliasHelp := run(t, nil, pair.alias, "--help")
		canonicalArgs := strings.Fields(pair.canonical)
		canonicalArgs = append(canonicalArgs, "--help")
		canonicalHelp := run(t, nil, canonicalArgs...)
		oldPath := "  lds " + pair.alias
		newPath := "  lds " + pair.canonical
		if aliasHelp.exitCode != 0 || canonicalHelp.exitCode != 0 || aliasHelp.stderr != "" || canonicalHelp.stderr != "" || strings.Count(aliasHelp.stdout, oldPath) != 1 {
			t.Fatalf("help pair %q/%q: alias=%+v canonical=%+v", pair.alias, pair.canonical, aliasHelp, canonicalHelp)
		}
		if strings.Replace(aliasHelp.stdout, oldPath, newPath, 1) != canonicalHelp.stdout {
			t.Errorf("help pair %q/%q differs beyond the command path:\nalias:\n%s\ncanonical:\n%s", pair.alias, pair.canonical, aliasHelp.stdout, canonicalHelp.stdout)
		}
	}
}

func TestRootAliasesMatchCanonicalPlumbingMembers(t *testing.T) {
	skills := t.TempDir()
	env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills}
	for _, pair := range []struct {
		alias     []string
		canonical []string
	}{
		{[]string{"manifest"}, []string{"plumbing", "manifest"}},
		{[]string{"manifest", "--json"}, []string{"plumbing", "manifest", "--json"}},
		{[]string{"doctor"}, []string{"plumbing", "doctor"}},
		{[]string{"doctor", "--json"}, []string{"plumbing", "doctor", "--json"}},
	} {
		alias := run(t, env, pair.alias...)
		canonical := run(t, env, pair.canonical...)
		if alias.exitCode != canonical.exitCode || alias.stdout != canonical.stdout || alias.stderr != canonical.stderr {
			t.Errorf("alias %v differs from %v:\nalias=%+v\ncanonical=%+v", pair.alias, pair.canonical, alias, canonical)
		}
	}

	installClean(t, skills, env)
	stampPath := filepath.Join(skills, "lds", ".lds-manifest-stamp.json")
	if err := os.WriteFile(stampPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	alias := run(t, env, "doctor")
	canonical := run(t, env, "plumbing", "doctor")
	if alias.exitCode != 1 || alias.exitCode != canonical.exitCode || alias.stdout != canonical.stdout || alias.stderr != canonical.stderr {
		t.Fatalf("failing doctor alias differs from canonical: alias=%+v canonical=%+v", alias, canonical)
	}
	if !strings.HasPrefix(alias.stderr, "lds: plumbing doctor: ") {
		t.Fatalf("alias failure diagnostic is not canonicalized: %q", alias.stderr)
	}
	aliasJSON := run(t, env, "doctor", "--json")
	canonicalJSON := run(t, env, "plumbing", "doctor", "--json")
	if aliasJSON.exitCode != canonicalJSON.exitCode || aliasJSON.stdout != canonicalJSON.stdout || aliasJSON.stderr != canonicalJSON.stderr {
		t.Fatalf("failing JSON doctor alias differs from canonical: alias=%+v canonical=%+v", aliasJSON, canonicalJSON)
	}
}

func TestInstall_RunEFailureIsInternalAndArgsFailureRemainsUsage(t *testing.T) {
	// install's own Args validator and RunE share the same Cobra command:
	// excess args must remain usage, while a real filesystem failure from
	// the verb must be rendered as internal in both output modes.
	if r := run(t, nil, "install", "claude-code", "extra"); r.exitCode != 2 || r.stdout != "" {
		t.Fatalf("install Args failure: exit=%d stdout=%q stderr=%q; want usage 2 and empty stdout", r.exitCode, r.stdout, r.stderr)
	}

	skillsFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(skillsFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skillsFile}
	text := run(t, env, "install", "claude-code", "--force")
	if text.exitCode != 4 || text.stdout != "" || !strings.HasPrefix(text.stderr, "lds: install: claudecode: creating ") || !strings.Contains(text.stderr, "not a directory") {
		t.Fatalf("text RunE failure: exit=%d stdout=%q stderr=%q; want internal 4 with one verb-prefixed diagnostic", text.exitCode, text.stdout, text.stderr)
	}
	jsonResult := run(t, env, "install", "claude-code", "--force", "--json")
	if jsonResult.exitCode != text.exitCode || jsonResult.stdout != "" {
		t.Fatalf("JSON RunE failure: exit=%d stdout=%q stderr=%q; want exit parity and empty stdout", jsonResult.exitCode, jsonResult.stdout, jsonResult.stderr)
	}
	envelope := parseErrorEnvelope(t, jsonResult.stderr)
	if envelope.Error.Code != "internal.command-failed" || !strings.Contains(envelope.Error.Message, "not a directory") {
		t.Fatalf("JSON error = %+v; want internal.command-failed with the filesystem failure", envelope.Error)
	}
}

func TestVersion_JSON(t *testing.T) {
	r := run(t, nil, "version", "--json")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	var payload struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &payload); err != nil {
		t.Fatalf("version --json stdout is not one JSON value: %v; stdout=%q", err, r.stdout)
	}
	if payload.Version == "" {
		t.Fatalf("version --json reported an empty version")
	}
}

func TestManifest_JSON_DeclaresContractAndDigest(t *testing.T) {
	r := run(t, nil, "plumbing", "manifest", "--json")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	var m struct {
		SchemaVersion           int    `json:"schemaVersion"`
		Contract                string `json:"contract"`
		ContractReconciledMinor int    `json:"contractReconciledMinor"`
		ManifestDigest          string `json:"manifest_digest"`
		Verbs                   []struct {
			Name    string `json:"name"`
			Kind    string `json:"kind"`
			AliasOf string `json:"alias-of"`
		} `json:"verbs"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &m); err != nil {
		t.Fatalf("manifest --json stdout is not one JSON value: %v; stdout=%q", err, r.stdout)
	}
	if m.Contract == "" {
		t.Fatalf("manifest declares no contract version (C3.6)")
	}
	if m.ContractReconciledMinor != 2 {
		t.Fatalf("contractReconciledMinor = %d, want 2 (T25)", m.ContractReconciledMinor)
	}
	if !strings.HasPrefix(m.ManifestDigest, "sha256:") {
		t.Fatalf("manifest_digest = %q, want a sha256: prefix (C3.4)", m.ManifestDigest)
	}
	if len(m.Verbs) == 0 {
		t.Fatalf("manifest lists no verbs")
	}
	for _, v := range m.Verbs {
		if v.Kind == "" {
			t.Fatalf("verb %q carries no surface kind (C3.2)", v.Name)
		}
	}
	want := map[string]string{
		"version":                 "",
		"manifest":                "plumbing manifest",
		"install":                 "",
		"uninstall":               "",
		"doctor":                  "plumbing doctor",
		"plumbing adopt":          "",
		"plumbing manifest":       "",
		"plumbing doctor":         "",
		"plumbing docs apply":     "",
		"plumbing docs inventory": "",
		"plumbing docs plan":      "",
		"plumbing docs render":    "",
		"plumbing docs validate":  "",
		"plumbing reconcile scan": "",
	}
	seen := map[string]bool{}
	projectable := 0
	for _, verb := range m.Verbs {
		if strings.HasPrefix(verb.Name, "docs ") {
			t.Errorf("manifest retained an unqualified docs leaf %q", verb.Name)
		}
		if canonical, required := want[verb.Name]; required {
			seen[verb.Name] = true
			if verb.AliasOf != canonical {
				t.Errorf("verb %q alias-of = %q, want %q", verb.Name, verb.AliasOf, canonical)
			}
		} else if verb.AliasOf != "" {
			t.Errorf("unreviewed verb %q has alias-of %q", verb.Name, verb.AliasOf)
		}
		if verb.Kind == "plumbing" && strings.HasPrefix(verb.Name, "plumbing ") {
			projectable++
		}
	}
	if len(m.Verbs) != 14 || len(seen) != len(want) || projectable != 9 {
		t.Fatalf("manifest leaves=%d, expected names seen=%d/%d, projectable=%d; want fourteen leaves and nine projected members", len(m.Verbs), len(seen), len(want), projectable)
	}
}

func TestManifest_JSON_ListsAdoptionTemplatesWithChecksums(t *testing.T) {
	r := run(t, nil, "plumbing", "manifest", "--json")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	var m struct {
		Assets []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"assets"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &m); err != nil {
		t.Fatalf("manifest --json stdout is not one JSON value: %v; stdout=%q", err, r.stdout)
	}

	wantPaths := []string{
		"flows/adopt.md",
		"templates/adoption/docs/README.md",
		"templates/adoption/docs/reference/usage.md",
		"templates/adoption/engineering/README.md",
		"templates/adoption/engineering/product/overview.md",
	}
	for _, path := range wantPaths {
		data, err := rootassets.FS.ReadFile(path)
		if err != nil {
			t.Fatalf("read embedded adoption template %q: %v", path, err)
		}
		sum := sha256.Sum256(data)
		wantChecksum := hex.EncodeToString(sum[:])
		matches := 0
		for _, asset := range m.Assets {
			if asset.Path != path {
				continue
			}
			matches++
			if asset.SHA256 != wantChecksum {
				t.Errorf("manifest checksum for %q = %q, want %q", path, asset.SHA256, wantChecksum)
			}
		}
		if matches != 1 {
			t.Errorf("manifest lists %q %d times, want exactly once", path, matches)
		}
	}
}

func TestDocsRender_GreenfieldJSONAndReadOnly(t *testing.T) {
	fixture := filepath.Join("..", "asset", "testdata", "adoption", "greenfield")
	root := filepath.Join(fixture, "target")
	facts := filepath.Join(fixture, "facts.json")
	xdg := t.TempDir()
	before := renderTargetSnapshot(t, root)
	r := run(t, []string{"XDG_CONFIG_HOME=" + xdg}, "plumbing", "docs", "render", root, "--facts", facts, "--json")
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("render exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	var result struct {
		SchemaVersion int    `json:"schemaVersion"`
		Operation     string `json:"operation"`
		Outputs       []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			SHA256  string `json:"sha256"`
			Asset   struct {
				RelativeName string `json:"relativeName"`
				Source       string `json:"source"`
				SHA256       string `json:"sha256"`
			} `json:"asset"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &result); err != nil {
		t.Fatalf("render stdout is not JSON: %v; stdout=%q", err, r.stdout)
	}
	if result.SchemaVersion != 1 || result.Operation != "docs render" || len(result.Outputs) != 4 {
		t.Fatalf("render result header/outputs = %+v", result)
	}
	wantPaths := []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"}
	for i, output := range result.Outputs {
		if output.Path != wantPaths[i] || output.Asset.Source != "embedded" {
			t.Errorf("output[%d] path/source = %q/%q", i, output.Path, output.Asset.Source)
		}
		want, err := os.ReadFile(filepath.Join(fixture, "expected", filepath.FromSlash(output.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if output.Content != string(want) {
			t.Errorf("rendered %s differs from independent expected bytes", output.Path)
		}
		sum := sha256.Sum256([]byte(output.Content))
		if output.SHA256 != hex.EncodeToString(sum[:]) {
			t.Errorf("content hash for %s = %s", output.Path, output.SHA256)
		}
		assetBytes, err := rootassets.FS.ReadFile(output.Asset.RelativeName)
		if err != nil {
			t.Fatal(err)
		}
		assetSum := sha256.Sum256(assetBytes)
		if output.Asset.SHA256 != hex.EncodeToString(assetSum[:]) {
			t.Errorf("asset hash for %s = %s", output.Path, output.Asset.SHA256)
		}
	}
	if after := renderTargetSnapshot(t, root); !equalRenderSnapshots(before, after) {
		t.Fatalf("render modified target: before=%v after=%v", before, after)
	}
}

func TestDocsRender_ResolverPrecedenceAndShadowing(t *testing.T) {
	fixture := filepath.Join("..", "asset", "testdata", "adoption", "greenfield")
	root, facts := filepath.Join(fixture, "target"), filepath.Join(fixture, "facts.json")
	xdg := t.TempDir()
	defaultDir := filepath.Join(filepath.Dir(filepath.Dir(binPath)), "share", "lds", "assets")
	if err := os.RemoveAll(defaultDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(defaultDir) })
	templates := []struct {
		asset  string
		output string
	}{
		{"templates/adoption/docs/README.md", "docs/README.md"},
		{"templates/adoption/docs/reference/usage.md", "docs/reference/usage.md"},
		{"templates/adoption/engineering/README.md", "engineering/README.md"},
		{"templates/adoption/engineering/product/overview.md", "engineering/product/overview.md"},
	}
	defaultAssets := map[string][]byte{}
	overrideAssets := map[string][]byte{}
	for _, item := range templates {
		base, err := rootassets.FS.ReadFile(item.asset)
		if err != nil {
			t.Fatal(err)
		}
		defaultAssets[item.asset] = append([]byte("<!-- installed-default-only:"+item.output+" -->\n"), base...)
		overrideAssets[item.asset] = append([]byte("<!-- user-override-only:"+item.output+" -->\n"), base...)
		writeRenderAsset(t, defaultDir, item.asset, defaultAssets[item.asset])
	}
	checkSources := func(r result, source, expectedMarker, rejectedMarker string) {
		t.Helper()
		for _, item := range templates {
			output := mustRenderOutput(t, r, item.output)
			if output.Asset.Source != source || (expectedMarker != "" && !strings.Contains(output.Content, expectedMarker+item.output)) || (rejectedMarker != "" && strings.Contains(output.Content, rejectedMarker)) {
				t.Errorf("%s resolution for %s = %+v", source, item.output, output)
			}
			var assetBytes []byte
			switch source {
			case "default":
				assetBytes = defaultAssets[item.asset]
			case "override":
				assetBytes = overrideAssets[item.asset]
			default:
				assetBytes, _ = rootassets.FS.ReadFile(item.asset)
			}
			sum := sha256.Sum256(assetBytes)
			if output.Asset.SHA256 != hex.EncodeToString(sum[:]) {
				t.Errorf("asset hash for %s = %s", item.output, output.Asset.SHA256)
			}
		}
	}

	r := run(t, []string{"XDG_CONFIG_HOME=" + xdg}, "plumbing", "docs", "render", root, "--facts", facts, "--json")
	checkSources(r, "default", "installed-default-only:", "user-override-only:")

	overrideDir := filepath.Join(xdg, "lds")
	for _, item := range templates {
		writeRenderAsset(t, overrideDir, item.asset, overrideAssets[item.asset])
	}
	r = run(t, []string{"XDG_CONFIG_HOME=" + xdg}, "plumbing", "docs", "render", root, "--facts", facts, "--json")
	checkSources(r, "override", "user-override-only:", "installed-default-only:")

	for _, item := range templates {
		if err := os.Remove(filepath.Join(overrideDir, filepath.FromSlash(item.asset))); err != nil {
			t.Fatal(err)
		}
	}
	r = run(t, []string{"XDG_CONFIG_HOME=" + xdg}, "plumbing", "docs", "render", root, "--facts", facts, "--json")
	checkSources(r, "default", "installed-default-only:", "user-override-only:")

	for _, item := range templates {
		if err := os.Remove(filepath.Join(defaultDir, filepath.FromSlash(item.asset))); err != nil {
			t.Fatal(err)
		}
	}
	r = run(t, []string{"XDG_CONFIG_HOME=" + xdg}, "plumbing", "docs", "render", root, "--facts", facts, "--json")
	checkSources(r, "embedded", "", "installed-default-only:")
	for _, item := range templates {
		output := mustRenderOutput(t, r, item.output)
		wantContent, err := os.ReadFile(filepath.Join(fixture, "expected", filepath.FromSlash(item.output)))
		if err != nil {
			t.Fatal(err)
		}
		if output.Content != string(wantContent) {
			t.Errorf("embedded fallback for %s differs from independent expected bytes", item.output)
		}
	}
}

func TestDocsRender_InvalidFactsAndTemplateExitOne(t *testing.T) {
	fixture := filepath.Join("..", "asset", "testdata", "adoption", "greenfield")
	root := filepath.Join(fixture, "target")
	valid, err := os.ReadFile(filepath.Join(fixture, "facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "missing required evidence"},
		{name: "malformed facts", data: []byte(`{"DisplayName":`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := filepath.Join(t.TempDir(), "facts.json")
			data := tc.data
			if tc.name == "missing required evidence" {
				var raw map[string]any
				if err := json.Unmarshal(valid, &raw); err != nil {
					t.Fatal(err)
				}
				raw["OutputEvidence"] = ""
				data, err = json.Marshal(raw)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(facts, data, 0o600); err != nil {
				t.Fatal(err)
			}
			r := run(t, []string{"XDG_CONFIG_HOME=" + t.TempDir()}, "plumbing", "docs", "render", root, "--facts", facts, "--json")
			if r.exitCode != 1 || r.stdout != "" || parseErrorEnvelope(t, r.stderr).Error.Code != "validation.render-invalid" {
				t.Fatalf("invalid render result = %+v", r)
			}
		})
	}

	badOverrideHome := t.TempDir()
	writeRenderAsset(t, filepath.Join(badOverrideHome, "lds"), "templates/adoption/docs/README.md", []byte("{{.Undeclared}}"))
	badOverride := run(t, []string{"XDG_CONFIG_HOME=" + badOverrideHome}, "plumbing", "docs", "render", root, "--facts", filepath.Join(fixture, "facts.json"), "--json")
	if badOverride.exitCode != 1 || badOverride.stdout != "" || parseErrorEnvelope(t, badOverride.stderr).Error.Code != "validation.render-invalid" {
		t.Fatalf("invalid template override result = %+v", badOverride)
	}

	r := run(t, []string{"XDG_CONFIG_HOME=" + t.TempDir()}, "plumbing", "docs", "render", root, "--facts", filepath.Join(fixture, "facts.json"), "--json", "--bad-template-option")
	if r.exitCode != 2 || r.stdout != "" {
		t.Fatalf("render usage exit=%d stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
}

func TestDocsReadOnly_ManifestAndFreshTarget(t *testing.T) {
	m := run(t, nil, "plumbing", "manifest", "--json")
	if m.exitCode != 0 {
		t.Fatalf("manifest exit=%d stderr=%q", m.exitCode, m.stderr)
	}
	var declaration struct {
		Verbs []struct {
			Name         string          `json:"name"`
			Kind         string          `json:"kind"`
			Usage        string          `json:"usage"`
			OutputSchema json.RawMessage `json:"outputSchema"`
		} `json:"verbs"`
	}
	if err := json.Unmarshal([]byte(m.stdout), &declaration); err != nil {
		t.Fatal(err)
	}
	matches := map[string]int{"plumbing docs inventory": 0, "plumbing docs validate": 0, "plumbing docs render": 0, "plumbing docs plan": 0, "plumbing docs apply": 0}
	for _, verb := range declaration.Verbs {
		if _, ok := matches[verb.Name]; !ok {
			continue
		}
		matches[verb.Name]++
		var wantField string
		switch verb.Name {
		case "plumbing docs inventory":
			wantField = `"files"`
		case "plumbing docs validate":
			wantField = `"findings"`
		case "plumbing docs render":
			wantField = `"outputs"`
		case "plumbing docs plan":
			wantField = `"planDigest"`
		case "plumbing docs apply":
			wantField = `"created"`
		}
		if verb.Kind != "plumbing" || !json.Valid(verb.OutputSchema) || !bytes.Contains(verb.OutputSchema, []byte(wantField)) {
			t.Fatalf("docs declaration = %+v", verb)
		}
	}
	for name, count := range matches {
		if count != 1 {
			t.Fatalf("manifest has %d %s declarations, want one", count, name)
		}
	}
	r := run(t, nil, "plumbing", "docs", "inventory", t.TempDir(), "--json")
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("inventory exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	if r.stdout != "{\"schemaVersion\":1,\"operation\":\"docs inventory\",\"files\":[],\"links\":[],\"skipped\":[]}\n" {
		t.Fatalf("fresh inventory stdout = %q", r.stdout)
	}
}

func TestDocsPlan_GreenfieldJSONAndReadOnly(t *testing.T) {
	fixture := filepath.Join("..", "asset", "testdata", "adoption", "greenfield")
	root := filepath.Join(fixture, "target")
	facts := filepath.Join(fixture, "facts.json")
	before := renderTargetSnapshot(t, root)
	r := run(t, []string{"XDG_CONFIG_HOME=" + t.TempDir()}, "plumbing", "docs", "plan", root, "--facts", facts, "--json")
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("plan exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	var plan adoption.PlanResult
	if err := json.Unmarshal([]byte(r.stdout), &plan); err != nil {
		t.Fatalf("plan stdout is not JSON: %v; stdout=%q", err, r.stdout)
	}
	if plan.SchemaVersion != 1 || plan.Operation != "docs plan" || plan.TargetRoot != root || plan.Veto || len(plan.Outputs) != 4 || plan.PlanDigest == nil {
		t.Fatalf("plan header = %+v", plan.PlanProjection)
	}
	digest, err := adoption.CanonicalPlanDigest(plan)
	if err != nil || digest != *plan.PlanDigest {
		t.Fatalf("plan digest=%q err=%v, want %q", digest, err, *plan.PlanDigest)
	}
	wantPaths := []string{"docs/README.md", "docs/reference/usage.md", "engineering/README.md", "engineering/product/overview.md"}
	for i, output := range plan.Outputs {
		if output.Path != wantPaths[i] || output.Classification != "create" || output.Content == "" || output.Asset == nil || output.Asset.Source != "embedded" {
			t.Errorf("output[%d] = %+v", i, output)
		}
		want, err := os.ReadFile(filepath.Join(fixture, "expected", filepath.FromSlash(output.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if output.Content != string(want) {
			t.Errorf("plan content for %s differs from independent expected bytes", output.Path)
		}
		sum := sha256.Sum256(want)
		if output.ContentSHA256 != hex.EncodeToString(sum[:]) {
			t.Errorf("plan content hash for %s = %s", output.Path, output.ContentSHA256)
		}
	}
	for _, baseline := range []struct{ path, hash string }{
		{"sample.txt", "0fedab7a6bcdee67c26091670c1aba184dacda6b69a321611f1dd53910e917c9"},
		{"word-count.sh", "fe429d0d1c8969ab6b4aa7df28e32c29a7394d851e0f2b365927241859722559"},
	} {
		found := false
		for _, preserved := range plan.Preserved {
			if preserved.Path == baseline.path && preserved.SHA256 == baseline.hash {
				found = true
			}
		}
		if !found {
			t.Errorf("plan does not preserve baseline %s with hash %s", baseline.path, baseline.hash)
		}
	}
	if after := renderTargetSnapshot(t, root); !equalRenderSnapshots(before, after) {
		t.Fatalf("plan modified target: before=%v after=%v", before, after)
	}
}

func TestDocsApply_ReviewedDigestCreatesThenNoops(t *testing.T) {
	fixture := filepath.Join("..", "asset", "testdata", "adoption", "greenfield")
	root := t.TempDir()
	facts := filepath.Join(fixture, "facts.json")
	planRun := run(t, []string{"XDG_CONFIG_HOME=" + t.TempDir()}, "plumbing", "docs", "plan", root, "--facts", facts, "--json")
	if planRun.exitCode != 0 || planRun.stderr != "" {
		t.Fatalf("plan exit=%d stderr=%q", planRun.exitCode, planRun.stderr)
	}
	var plan adoption.PlanResult
	if err := json.Unmarshal([]byte(planRun.stdout), &plan); err != nil || plan.PlanDigest == nil {
		t.Fatalf("decode reviewed plan: %v", err)
	}
	planFile := filepath.Join(t.TempDir(), "reviewed-plan.json")
	if err := os.WriteFile(planFile, []byte(planRun.stdout), 0o600); err != nil {
		t.Fatal(err)
	}
	env := []string{"XDG_CONFIG_HOME=" + t.TempDir()}
	declined := run(t, env, "plumbing", "docs", "apply", root, "--facts", facts, "--plan", planFile, "--confirm", "sha256:declined", "--json")
	var decline adoption.ApplyResult
	if declined.exitCode != 3 || parseErrorEnvelope(t, declined.stderr).Error.Code != "refusal.confirmation-declined" || json.Unmarshal([]byte(declined.stdout), &decline) != nil || decline.Status != "refused" || len(decline.Unattempted) != 4 {
		t.Fatalf("declined apply=%+v result=%+v", declined, decline)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("declined apply changed target: entries=%v err=%v", entries, err)
	}
	first := run(t, env, "plumbing", "docs", "apply", root, "--facts", facts, "--plan", planFile, "--confirm", *plan.PlanDigest, "--json")
	if first.exitCode != 0 || first.stderr != "" {
		t.Fatalf("apply exit=%d stderr=%q stdout=%q", first.exitCode, first.stderr, first.stdout)
	}
	var applied adoption.ApplyResult
	if err := json.Unmarshal([]byte(first.stdout), &applied); err != nil || applied.Status != "applied" || len(applied.Created) != 4 {
		t.Fatalf("apply result=%+v err=%v", applied, err)
	}
	for _, output := range plan.Outputs {
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(output.Path)))
		if err != nil || string(actual) != output.Content {
			t.Fatalf("applied bytes %s differ from reviewed plan: err=%v", output.Path, err)
		}
	}
	secondPlanRun := run(t, env, "plumbing", "docs", "plan", root, "--facts", facts, "--json")
	var secondPlan adoption.PlanResult
	if secondPlanRun.exitCode != 0 || json.Unmarshal([]byte(secondPlanRun.stdout), &secondPlan) != nil || secondPlan.PlanDigest == nil {
		t.Fatalf("second plan: %+v", secondPlanRun)
	}
	if err := os.WriteFile(planFile, []byte(secondPlanRun.stdout), 0o600); err != nil {
		t.Fatal(err)
	}
	second := run(t, env, "plumbing", "docs", "apply", root, "--facts", facts, "--plan", planFile, "--confirm", *secondPlan.PlanDigest, "--json")
	var current adoption.ApplyResult
	if second.exitCode != 0 || second.stderr != "" || json.Unmarshal([]byte(second.stdout), &current) != nil || current.Status != "current" || len(current.Created) != 0 || len(current.IdenticalUntouched) != 4 {
		t.Fatalf("identical rerun=%+v result=%+v", second, current)
	}
}

func TestDocsPlan_IncompleteFactsCollisionAndUsage(t *testing.T) {
	fixture := filepath.Join("..", "asset", "testdata", "adoption", "greenfield")
	root := filepath.Join(fixture, "target")
	facts := filepath.Join(fixture, "facts.json")

	incomplete := t.TempDir()
	data, err := os.ReadFile(facts)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	fields["OutputEvidence"] = ""
	data, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	incompleteFacts := filepath.Join(incomplete, "facts.json")
	if err := os.WriteFile(incompleteFacts, data, 0o600); err != nil {
		t.Fatal(err)
	}
	refused := run(t, []string{"XDG_CONFIG_HOME=" + t.TempDir()}, "plumbing", "docs", "plan", root, "--facts", incompleteFacts, "--json")
	if refused.exitCode != 3 || parseErrorEnvelope(t, refused.stderr).Error.Code != "refusal.incomplete-facts" {
		t.Fatalf("incomplete facts result=%+v", refused)
	}
	var incompletePlan adoption.PlanResult
	if err := json.Unmarshal([]byte(refused.stdout), &incompletePlan); err != nil || incompletePlan.PlanDigest != nil || len(incompletePlan.Outputs) != 4 {
		t.Fatalf("incomplete facts stdout=%q err=%v", refused.stdout, err)
	}

	collisionRoot := t.TempDir()
	collisionPath := filepath.Join(collisionRoot, "docs", "reference", "usage.md")
	if err := os.MkdirAll(filepath.Dir(collisionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	oldBytes := []byte("# Owner page\n\n[old guide](../howto.md)\n")
	if err := os.WriteFile(collisionPath, oldBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	before := renderTargetSnapshot(t, collisionRoot)
	collision := run(t, []string{"XDG_CONFIG_HOME=" + t.TempDir()}, "plumbing", "docs", "plan", collisionRoot, "--facts", facts, "--json")
	if collision.exitCode != 3 || parseErrorEnvelope(t, collision.stderr).Error.Code != "refusal.output-collision" {
		t.Fatalf("collision result=%+v", collision)
	}
	var collisionPlan adoption.PlanResult
	if err := json.Unmarshal([]byte(collision.stdout), &collisionPlan); err != nil {
		t.Fatalf("collision stdout: %v", err)
	}
	if !collisionPlan.Veto || findCLITestOutput(collisionPlan, "docs/reference/usage.md").ExistingContent != string(oldBytes) {
		t.Fatalf("collision plan omitted old bytes or veto: %+v", collisionPlan)
	}
	if after := renderTargetSnapshot(t, collisionRoot); !equalRenderSnapshots(before, after) {
		t.Fatalf("collision plan changed target: before=%v after=%v", before, after)
	}

	usage := run(t, []string{"XDG_CONFIG_HOME=" + t.TempDir()}, "plumbing", "docs", "plan", root, "--json")
	if usage.exitCode != 2 || usage.stdout != "" || !strings.HasPrefix(usage.stderr, "lds: ") {
		t.Fatalf("missing required facts flag usage=%+v", usage)
	}
}

func TestDocsPlan_BrokenLinkIsDiagnosticNotVeto(t *testing.T) {
	fixture := filepath.Join("..", "asset", "testdata", "adoption", "greenfield")
	root := t.TempDir()
	for _, name := range []string{"sample.txt", "word-count.sh"} {
		data, err := os.ReadFile(filepath.Join(fixture, "target", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	links := "[missing](reference/missing.md#install)\n[external](https://example.invalid/docs)\n[anchor](#overview)\n"
	if err := os.WriteFile(filepath.Join(root, "docs", "legacy-links.md"), []byte(links), 0o644); err != nil {
		t.Fatal(err)
	}
	before := renderTargetSnapshot(t, root)
	r := run(t, []string{"XDG_CONFIG_HOME=" + t.TempDir()}, "plumbing", "docs", "plan", root, "--facts", filepath.Join(fixture, "facts.json"), "--json")
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("diagnostic plan exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	var result adoption.PlanResult
	if err := json.Unmarshal([]byte(r.stdout), &result); err != nil {
		t.Fatalf("plan stdout: %v", err)
	}
	if result.Veto || len(result.Findings) != 1 || result.Findings[0].Code != "validation.local-target-missing" || result.Findings[0].Path != "docs/legacy-links.md" || result.Findings[0].Line != 1 {
		t.Fatalf("link finding improperly vetoed or missing: %+v", result)
	}
	var external, anchor bool
	for _, skipped := range result.Skipped {
		external = external || skipped.Code == "inventory.external-skipped"
		anchor = anchor || skipped.Code == "inventory.fragment-only-skipped"
	}
	if !external || !anchor || result.PlanDigest == nil {
		t.Fatalf("plan diagnostics/digest = %+v", result)
	}
	if after := renderTargetSnapshot(t, root); !equalRenderSnapshots(before, after) {
		t.Fatalf("diagnostic plan changed target: before=%v after=%v", before, after)
	}
}

func findCLITestOutput(plan adoption.PlanResult, target string) adoption.PlanOutput {
	for _, output := range plan.Outputs {
		if output.Path == target {
			return output
		}
	}
	return adoption.PlanOutput{}
}

func TestDocsInventory_UnsafeTargetRefuses(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	r := run(t, nil, "plumbing", "docs", "inventory", alias, "--json")
	if r.exitCode != 3 || r.stdout != "" {
		t.Fatalf("unsafe target exit=%d stdout=%q stderr=%q; want refusal 3 and no partial inventory", r.exitCode, r.stdout, r.stderr)
	}
	if e := parseErrorEnvelope(t, r.stderr); e.Error.Code != "refusal.unsafe-target" {
		t.Fatalf("unsafe target error = %+v", e.Error)
	}
}

func TestDocsReadOnly_ExactJSONAndExits(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"README.md":            "[guide](engineering/guide.md)\n",
		"engineering/guide.md": "[doc][ref]\n\n[ref]: ../docs/target.md\n",
		"docs/target.md":       "# Target\n",
	}
	for path, data := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := run(t, nil, "plumbing", "docs", "inventory", root, "--json")
	want := fmt.Sprintf(`{"schemaVersion":1,"operation":"docs inventory","files":[{"path":"README.md","sha256":"%x","size":%d},{"path":"docs/target.md","sha256":"%x","size":%d},{"path":"engineering/guide.md","sha256":"%x","size":%d}],"links":[{"source":"README.md","line":1,"target":"engineering/guide.md","kind":"inline"},{"source":"engineering/guide.md","line":1,"target":"../docs/target.md","kind":"reference"}],"skipped":[]}`+"\n",
		sha256.Sum256([]byte(files["README.md"])), len(files["README.md"]),
		sha256.Sum256([]byte(files["docs/target.md"])), len(files["docs/target.md"]),
		sha256.Sum256([]byte(files["engineering/guide.md"])), len(files["engineering/guide.md"]))
	if r.exitCode != 0 || r.stdout != want || r.stderr != "" {
		t.Fatalf("inventory: exit=%d stdout=%q stderr=%q; want stdout=%q", r.exitCode, r.stdout, r.stderr, want)
	}

	r = run(t, nil, "plumbing", "docs", "validate", root, "--json")
	if r.exitCode != 0 || r.stdout != `{"schemaVersion":1,"operation":"docs validate","findings":[],"skipped":[]}`+"\n" || r.stderr != "" {
		t.Fatalf("clean validate: exit=%d stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}

	links := "# Links\n[missing](reference/missing.md#install)\n[external](https://example.invalid/docs)\n[fragment](#overview)\n"
	if err := os.WriteFile(filepath.Join(root, "docs", "links.md"), []byte(links), 0o644); err != nil {
		t.Fatal(err)
	}
	r = run(t, nil, "plumbing", "docs", "validate", root, "--json")
	want = `{"schemaVersion":1,"operation":"docs validate","findings":[{"code":"validation.local-target-missing","path":"docs/links.md","line":2,"target":"reference/missing.md#install","message":"local file target is missing: reference/missing.md#install"}],"skipped":[{"source":"docs/links.md","line":3,"code":"inventory.external-skipped","reason":"external URL is not fetched"},{"source":"docs/links.md","line":4,"code":"inventory.fragment-only-skipped","reason":"fragment-only target is not checked"}]}` + "\n"
	if r.exitCode != 1 || r.stdout != want || parseErrorEnvelope(t, r.stderr).Error.Code != "validation.local-target-missing" {
		t.Fatalf("finding validate: exit=%d stdout=%q stderr=%q; want stdout=%q", r.exitCode, r.stdout, r.stderr, want)
	}

	if err := os.WriteFile(filepath.Join(root, "docs", "links.md"), []byte("[external](https://example.invalid)\n[fragment](#only)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r = run(t, nil, "plumbing", "docs", "validate", root, "--json")
	want = `{"schemaVersion":1,"operation":"docs validate","findings":[],"skipped":[{"source":"docs/links.md","line":1,"code":"inventory.external-skipped","reason":"external URL is not fetched"},{"source":"docs/links.md","line":2,"code":"inventory.fragment-only-skipped","reason":"fragment-only target is not checked"}]}` + "\n"
	if r.exitCode != 0 || r.stderr != "" || r.stdout != want {
		t.Fatalf("skips-only validate: exit=%d stdout=%q stderr=%q; want stdout=%q", r.exitCode, r.stdout, r.stderr, want)
	}
}

func TestDocsReadOnly_UsageMissingAndIO(t *testing.T) {
	for _, leaf := range []string{"inventory", "validate"} {
		t.Run(leaf, func(t *testing.T) {
			usage := run(t, nil, "plumbing", "docs", leaf, "--json")
			if usage.exitCode != 2 || usage.stdout != "" || !strings.HasPrefix(usage.stderr, "lds: ") {
				t.Fatalf("usage: %+v", usage)
			}
			missing := run(t, nil, "plumbing", "docs", leaf, filepath.Join(t.TempDir(), "absent"), "--json")
			if missing.exitCode != 1 || missing.stdout != "" || parseErrorEnvelope(t, missing.stderr).Error.Code != "validation.target-root-missing" {
				t.Fatalf("missing root: %+v", missing)
			}
			tooLong := filepath.Join(t.TempDir(), strings.Repeat("x", 256))
			failure := run(t, nil, "plumbing", "docs", leaf, tooLong, "--json")
			if failure.exitCode != 4 || failure.stdout != "" || !strings.HasPrefix(parseErrorEnvelope(t, failure.stderr).Error.Code, "internal.") {
				t.Fatalf("I/O path failure: %+v", failure)
			}
		})
	}
}

func renderTargetSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root {
				relative, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				files["dir:"+filepath.ToSlash(relative)] = ""
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files["file:"+filepath.ToSlash(relative)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func equalRenderSnapshots(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for path, data := range a {
		if b[path] != data {
			return false
		}
	}
	return true
}

func mustRenderOutput(t *testing.T, r result, path string) struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Asset   struct {
		RelativeName string `json:"relativeName"`
		Source       string `json:"source"`
		SHA256       string `json:"sha256"`
	} `json:"asset"`
} {
	t.Helper()
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("render exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	var payload struct {
		Outputs []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Asset   struct {
				RelativeName string `json:"relativeName"`
				Source       string `json:"source"`
				SHA256       string `json:"sha256"`
			} `json:"asset"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &payload); err != nil {
		t.Fatalf("decode render result: %v", err)
	}
	for _, output := range payload.Outputs {
		if output.Path == path {
			return output
		}
	}
	t.Fatalf("render result has no output %q", path)
	return struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Asset   struct {
			RelativeName string `json:"relativeName"`
			Source       string `json:"source"`
			SHA256       string `json:"sha256"`
		} `json:"asset"`
	}{}
}

func writeRenderAsset(t *testing.T, base, relative string, data []byte) {
	t.Helper()
	path := filepath.Join(base, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestManifest_DigestCommitsToTheDocument checks C3.4's substance, not just
// its prefix: the emitted manifest_digest must be a sha256 over the exact
// bytes the caller received, with the digest field held empty. It does the
// blanking on the raw stdout rather than by re-encoding a decoded document,
// so it reproduces what an external consumer can do and stays independent
// of the producing code's own marshalling.
//
// The property is what makes the asset list load-bearing: without it, an
// edited shipped asset could leave the digest where it was.
func TestManifest_DigestCommitsToTheDocument(t *testing.T) {
	r := run(t, nil, "plumbing", "manifest", "--json")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	var m struct {
		ManifestDigest string `json:"manifest_digest"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &m); err != nil {
		t.Fatalf("manifest --json stdout is not one JSON value: %v", err)
	}

	emitted := strings.TrimRight(r.stdout, "\n")
	blanked := strings.Replace(emitted, `"manifest_digest":"`+m.ManifestDigest+`"`, `"manifest_digest":""`, 1)
	if blanked == emitted {
		t.Fatalf("manifest_digest field not found verbatim in stdout; stdout=%q", emitted)
	}
	sum := sha256.Sum256([]byte(blanked))
	want := "sha256:" + hex.EncodeToString(sum[:])
	if m.ManifestDigest != want {
		t.Fatalf("manifest_digest = %q, want %q — the digest does not commit to the document it rides in (C3.4)", m.ManifestDigest, want)
	}
}

// TestManifest_IsDeterministic pins the premise manifest_digest's usefulness
// as a comparable identity rests on: the same binary must emit the same
// document every time, so neither the verb walk nor the asset walk may vary
// with map or filesystem iteration order.
func TestManifest_IsDeterministic(t *testing.T) {
	first := run(t, nil, "plumbing", "manifest", "--json")
	if first.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", first.exitCode, first.stderr)
	}
	for i := range 2 {
		again := run(t, nil, "plumbing", "manifest", "--json")
		if again.stdout != first.stdout {
			t.Fatalf("run %d differs from the first; the manifest is not deterministic", i+2)
		}
	}
}

// TestInstallLoop drives the full install lifecycle against a hermetic
// skills dir: install writes a stamped tree, a re-install reports current,
// a hand-edit flips install to a refusal (exit 3) and doctor keeps
// advising, --force recovers, and uninstall removes exactly the tree.
func TestInstallLoop(t *testing.T) {
	skills := t.TempDir()
	env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills}

	if r := run(t, env, "install", "claude-code"); r.exitCode != 0 {
		t.Fatalf("install: exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	skillDir := filepath.Join(skills, "lds")
	for _, f := range []string{"SKILL.md", ".claude-plugin/plugin.json", ".lds-manifest-stamp.json"} {
		if _, err := os.Stat(filepath.Join(skillDir, f)); err != nil {
			t.Fatalf("after install, %s: %v", f, err)
		}
	}
	skillBytes, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	wantRows := map[string]bool{
		"lds plumbing adopt":                 false,
		"lds plumbing doctor":                false,
		"lds plumbing manifest":              false,
		"lds plumbing docs apply <root>":     false,
		"lds plumbing docs inventory <root>": false,
		"lds plumbing docs plan <root>":      false,
		"lds plumbing docs render <root>":    false,
		"lds plumbing docs validate <root>":  false,
		"lds plumbing reconcile scan":        false,
	}
	var rows []string
	for _, line := range strings.Split(string(skillBytes), "\n") {
		if !strings.HasPrefix(line, "| `lds ") {
			continue
		}
		end := strings.Index(line, "` |")
		if end < 0 {
			t.Fatalf("malformed generated verb table row: %q", line)
		}
		invocation := line[3:end]
		rows = append(rows, invocation)
		if _, ok := wantRows[invocation]; ok {
			wantRows[invocation] = true
		}
	}
	if len(rows) != len(wantRows) {
		t.Fatalf("generated skill has %d verb rows %v, want exactly %d canonical plumbing rows", len(rows), rows, len(wantRows))
	}
	for invocation, found := range wantRows {
		if !found {
			t.Errorf("generated skill omits %q; rows=%v", invocation, rows)
		}
	}
	stampBytes, err := os.ReadFile(filepath.Join(skillDir, ".lds-manifest-stamp.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stamp struct {
		Files map[string]string `json:"files"`
	}
	if err := json.Unmarshal(stampBytes, &stamp); err != nil {
		t.Fatalf("decode generated stamp: %v", err)
	}
	skillHash := sha256.Sum256(skillBytes)
	if stamp.Files["SKILL.md"] != hex.EncodeToString(skillHash[:]) || len(stamp.Files) != 2 {
		t.Fatalf("generated stamp files = %v; want exact generated files with SKILL.md hash %x", stamp.Files, skillHash)
	}

	if r := run(t, env, "install", "claude-code", "--json"); r.exitCode != 0 || !strings.Contains(r.stdout, `"status":"current"`) {
		t.Fatalf("re-install: exit=%d stdout=%q, want status current", r.exitCode, r.stdout)
	}

	if r := run(t, env, "plumbing", "doctor"); r.exitCode != 0 {
		t.Fatalf("doctor on a current install: exit=%d stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}

	skillMD := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillMD, []byte("hand-edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := run(t, env, "install", "claude-code"); r.exitCode != 3 {
		t.Fatalf("install over a hand-edited tree: exit=%d, want 3 (refusal); stderr=%q", r.exitCode, r.stderr)
	}
	if r := run(t, env, "install", "claude-code", "--force"); r.exitCode != 0 {
		t.Fatalf("install --force: exit=%d stderr=%q", r.exitCode, r.stderr)
	}

	if r := run(t, env, "uninstall", "claude-code"); r.exitCode != 0 {
		t.Fatalf("uninstall: exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("after uninstall, %s still exists", skillDir)
	}
}

func TestUnknownHarness_JSONEnvelope(t *testing.T) {
	r := run(t, nil, "install", "no-such-harness", "--json")
	if r.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1 (validation); stderr=%q", r.exitCode, r.stderr)
	}
	if r.stdout != "" {
		t.Fatalf("stdout = %q, want empty on failure", r.stdout)
	}
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(r.stderr), &envelope); err != nil {
		t.Fatalf("stderr is not the {\"error\":...} envelope: %v; stderr=%q", err, r.stderr)
	}
	if envelope.Error.Code != "validation.unknown-harness" {
		t.Fatalf("error code = %q, want validation.unknown-harness", envelope.Error.Code)
	}
}

// errorEnvelope is the {"error":{"code","message"}} shape a structured
// error renders on stderr under --json (C2.5).
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// parseErrorEnvelope decodes stderr as the --json error envelope, failing
// the test if it is not one JSON value of that shape.
func parseErrorEnvelope(t *testing.T, stderr string) errorEnvelope {
	t.Helper()
	var envelope errorEnvelope
	if err := json.Unmarshal([]byte(stderr), &envelope); err != nil {
		t.Fatalf("stderr is not the {\"error\":...} envelope: %v; stderr=%q", err, stderr)
	}
	return envelope
}

// installClean installs claude-code cleanly into skills and returns its
// install directory, the fixture every refusing-state setup below starts
// from.
func installClean(t *testing.T, skills string, env []string) string {
	t.Helper()
	if r := run(t, env, "install", "claude-code"); r.exitCode != 0 {
		t.Fatalf("install: exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	return filepath.Join(skills, "lds")
}

// makeUnownedConflict puts a file at claude-code's install path with no
// stamp beside it — Status's UnownedConflict: content the tool never
// wrote (C4.5).
func makeUnownedConflict(t *testing.T, skills string, _ []string) string {
	t.Helper()
	skillDir := filepath.Join(skills, "lds")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "foreign.txt"), []byte("not ours\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

// makeModified installs cleanly, then hand-edits a generated file —
// Status's Modified: the disk no longer matches the stamp (C4.5).
func makeModified(t *testing.T, skills string, env []string) string {
	t.Helper()
	skillDir := installClean(t, skills, env)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("hand-edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

// makeIncompatibleUnparseable installs cleanly, then corrupts the stamp's
// JSON — Status's Incompatible via manifest.ErrStampUnparseable (C4.5,
// §1.2).
func makeIncompatibleUnparseable(t *testing.T, skills string, env []string) string {
	t.Helper()
	skillDir := installClean(t, skills, env)
	stampPath := filepath.Join(skillDir, ".lds-manifest-stamp.json")
	if err := os.WriteFile(stampPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

// makeIncompatibleSchemaVersion installs cleanly, then rewrites the
// stamp's schemaVersion to a value this binary does not recognize —
// Status's Incompatible via the schemaVersion mismatch (C4.5, §1.2).
func makeIncompatibleSchemaVersion(t *testing.T, skills string, env []string) string {
	t.Helper()
	skillDir := installClean(t, skills, env)
	stampPath := filepath.Join(skillDir, ".lds-manifest-stamp.json")
	raw, err := os.ReadFile(stampPath)
	if err != nil {
		t.Fatal(err)
	}
	var stamp map[string]any
	if err := json.Unmarshal(raw, &stamp); err != nil {
		t.Fatal(err)
	}
	stamp["schemaVersion"] = 999
	out, err := json.Marshal(stamp)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stampPath, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

// refusalCases is the fixture table both TestInstall_RefusalCodes and
// TestUninstall_RefusalCodes drive: one row per refusing state, each
// naming the code Status maps it to (C4.5 §1.4).
var refusalCases = []struct {
	name  string
	setup func(t *testing.T, skills string, env []string) string
	code  string
}{
	{"unowned_conflict", makeUnownedConflict, "refusal.unowned-harness-target"},
	{"modified", makeModified, "refusal.modified-harness-target"},
	{"incompatible unparseable stamp", makeIncompatibleUnparseable, "refusal.incompatible-harness-target"},
	{"incompatible schemaVersion", makeIncompatibleSchemaVersion, "refusal.incompatible-harness-target"},
}

// TestInstall_RefusalCodes asserts that install refuses each of the three
// unsafe states with its own code and exit 3, names --force in the
// message (naming what it would do to that state's content), and that
// --force still overwrites regardless of which state refused it.
func TestInstall_RefusalCodes(t *testing.T) {
	for _, c := range refusalCases {
		t.Run(c.name, func(t *testing.T) {
			skills := t.TempDir()
			env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills}
			c.setup(t, skills, env)

			r := run(t, env, "install", "claude-code", "--json")
			if r.exitCode != 3 {
				t.Fatalf("install: exit=%d, want 3 (refusal); stderr=%q", r.exitCode, r.stderr)
			}
			if r.stdout != "" {
				t.Fatalf("stdout = %q, want empty on failure", r.stdout)
			}
			envelope := parseErrorEnvelope(t, r.stderr)
			if envelope.Error.Code != c.code {
				t.Fatalf("error code = %q, want %q", envelope.Error.Code, c.code)
			}
			if !strings.Contains(envelope.Error.Message, "--force") {
				t.Fatalf("install refusal message = %q, want it to name --force", envelope.Error.Message)
			}

			if r := run(t, env, "install", "claude-code", "--force"); r.exitCode != 0 {
				t.Fatalf("install --force over %s: exit=%d stderr=%q", c.name, r.exitCode, r.stderr)
			}
		})
	}
}

// TestUninstall_RefusalCodes asserts that uninstall refuses each of the
// same three unsafe states with its own code and exit 3, and that — since
// uninstall has no --force — its message never names it.
func TestUninstall_RefusalCodes(t *testing.T) {
	for _, c := range refusalCases {
		t.Run(c.name, func(t *testing.T) {
			skills := t.TempDir()
			env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills}
			c.setup(t, skills, env)

			r := run(t, env, "uninstall", "claude-code", "--json")
			if r.exitCode != 3 {
				t.Fatalf("uninstall: exit=%d, want 3 (refusal); stderr=%q", r.exitCode, r.stderr)
			}
			if r.stdout != "" {
				t.Fatalf("stdout = %q, want empty on failure", r.stdout)
			}
			envelope := parseErrorEnvelope(t, r.stderr)
			if envelope.Error.Code != c.code {
				t.Fatalf("error code = %q, want %q", envelope.Error.Code, c.code)
			}
			if strings.Contains(envelope.Error.Message, "--force") {
				t.Fatalf("uninstall refusal message = %q, want it not to name --force (uninstall has none)", envelope.Error.Message)
			}
		})
	}
}

// TestUninstall_EmptyDirNoStamp_NotFound asserts that an existing but
// empty, unstamped install directory reads as Missing (C4.5 §1.4: it used
// to be a refusal) and so uninstall reports not-found, exit 1, not a
// refusal.
func TestUninstall_EmptyDirNoStamp_NotFound(t *testing.T) {
	skills := t.TempDir()
	env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills}
	skillDir := filepath.Join(skills, "lds")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	r := run(t, env, "uninstall", "claude-code", "--json")
	if r.exitCode != 1 {
		t.Fatalf("uninstall on an empty, unstamped directory: exit=%d, want 1 (not-found); stderr=%q", r.exitCode, r.stderr)
	}
	envelope := parseErrorEnvelope(t, r.stderr)
	if envelope.Error.Code != "not-found.harness-not-installed" {
		t.Fatalf("error code = %q, want not-found.harness-not-installed", envelope.Error.Code)
	}
}

// TestInstall_MalformedDescriptionOverride pins the error for a user
// override of the skill description that is empty or spans lines: a
// structured validation code with exit 1, not a plain error that the
// fallback path reports as usage (C2.4, C2.5).
func TestInstall_MalformedDescriptionOverride(t *testing.T) {
	for name, content := range map[string]string{
		"two lines": "first line\nsecond line\n",
		"empty":     "\n",
	} {
		t.Run(name, func(t *testing.T) {
			xdg := t.TempDir()
			override := filepath.Join(xdg, "lds", "templates", "skills", "claude-code", "description.txt")
			if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(override, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			env := []string{"LDS_CLAUDE_SKILLS_DIR=" + t.TempDir(), "XDG_CONFIG_HOME=" + xdg}

			r := run(t, env, "install", "claude-code", "--json")
			if r.exitCode != 1 {
				t.Fatalf("exit=%d, want 1; stderr=%q", r.exitCode, r.stderr)
			}
			if code := parseErrorEnvelope(t, r.stderr).Error.Code; code != "validation.skill-description" {
				t.Fatalf("error code = %q, want validation.skill-description", code)
			}
		})
	}
}

// TestInstallAll_OneRefused drives the bare `install` (no harness name)
// path against a refused target. The skeleton's registry lists exactly
// one harness, so refusing it also refuses the whole run: each per-target
// result carries its own refusal code in the results JSON on stdout, and
// the closing summary error on stderr carries the shared
// refusal.harness-targets-refused code (C4.5, C4.6).
func TestInstallAll_OneRefused(t *testing.T) {
	skills := t.TempDir()
	env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills}
	makeUnownedConflict(t, skills, env)

	r := run(t, env, "install", "--json")
	if r.exitCode != 3 {
		t.Fatalf("bare install: exit=%d, want 3 (refusal); stderr=%q", r.exitCode, r.stderr)
	}

	var payload struct {
		Results []struct {
			Harness string `json:"harness"`
			Status  string `json:"status"`
			Error   *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &payload); err != nil {
		t.Fatalf("stdout is not one JSON value: %v; stdout=%q", err, r.stdout)
	}
	found := false
	for _, res := range payload.Results {
		if res.Harness != "claude-code" {
			continue
		}
		found = true
		if res.Status != "refused" {
			t.Fatalf("claude-code result status = %q, want refused", res.Status)
		}
		if res.Error == nil || res.Error.Code != "refusal.unowned-harness-target" {
			t.Fatalf("claude-code result error = %+v, want code refusal.unowned-harness-target", res.Error)
		}
	}
	if !found {
		t.Fatalf("no claude-code result in %+v", payload.Results)
	}

	envelope := parseErrorEnvelope(t, r.stderr)
	if envelope.Error.Code != "refusal.harness-targets-refused" {
		t.Fatalf("error code = %q, want refusal.harness-targets-refused", envelope.Error.Code)
	}
}

// makeMissing is the doctor state fixture for Missing: nothing installed.
func makeMissing(t *testing.T, skills string, _ []string) string {
	t.Helper()
	return filepath.Join(skills, "lds")
}

// makeStale installs cleanly, then tampers with one generated file's disk
// content and its stamp entry together, to the same wrong value — disk
// still matches the stamp (no Modified), but the current binary's own
// output for that file (untouched) no longer matches the stamp: Status's
// Stale, the binary-vs-stamp question alone (C4.6). A plain hand-edit
// would also break disk-vs-stamp and read as Modified instead, which is
// why this rewrites the stamp entry too.
func makeStale(t *testing.T, skills string, env []string) string {
	t.Helper()
	skillDir := installClean(t, skills, env)

	staleContent := []byte("stale content\n")
	sum := sha256.Sum256(staleContent)
	staleHash := hex.EncodeToString(sum[:])
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), staleContent, 0o644); err != nil {
		t.Fatal(err)
	}

	stampPath := filepath.Join(skillDir, ".lds-manifest-stamp.json")
	raw, err := os.ReadFile(stampPath)
	if err != nil {
		t.Fatal(err)
	}
	var stamp map[string]any
	if err := json.Unmarshal(raw, &stamp); err != nil {
		t.Fatal(err)
	}
	files, ok := stamp["files"].(map[string]any)
	if !ok {
		t.Fatalf("stamp %q has no files object: %v", stampPath, stamp)
	}
	files["SKILL.md"] = staleHash
	out, err := json.Marshal(stamp)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stampPath, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

// TestDoctor_JSON_States drives doctor --json across all six drift states
// and asserts the reported target state, the finding codes, and the exit
// code: 1 only for Incompatible (the one state whose finding keeps the
// "refusal." prefix and so fails the run, C4.7), 0 for every other state.
func TestDoctor_JSON_States(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(t *testing.T, skills string, env []string) string
		wantState string
		wantExit  int
		wantCodes []string
	}{
		{"missing", makeMissing, "missing", 0, nil},
		{"current", installClean, "current", 0, nil},
		{"stale", makeStale, "stale", 0, []string{"advisory.stale-harness-artifact"}},
		{"modified", makeModified, "modified", 0, []string{"advisory.modified-harness-target"}},
		{"unowned_conflict", makeUnownedConflict, "unowned_conflict", 0, []string{"advisory.unowned-harness-target"}},
		{"incompatible unparseable stamp", makeIncompatibleUnparseable, "incompatible", 1, []string{"refusal.incompatible-harness-target"}},
		{"incompatible schemaVersion", makeIncompatibleSchemaVersion, "incompatible", 1, []string{"refusal.incompatible-harness-target"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			skills := t.TempDir()
			env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills}
			c.setup(t, skills, env)

			r := run(t, env, "plumbing", "doctor", "--json")
			if r.exitCode != c.wantExit {
				t.Fatalf("doctor --json: exit=%d, want %d; stdout=%q stderr=%q", r.exitCode, c.wantExit, r.stdout, r.stderr)
			}

			var payload struct {
				Findings []struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"findings"`
				Targets []struct {
					Harness string `json:"harness"`
					Dir     string `json:"dir"`
					State   string `json:"state"`
				} `json:"targets"`
			}
			if err := json.Unmarshal([]byte(r.stdout), &payload); err != nil {
				t.Fatalf("doctor --json stdout is not one JSON value: %v; stdout=%q", err, r.stdout)
			}
			if len(payload.Targets) != 1 || payload.Targets[0].Harness != "claude-code" {
				t.Fatalf("targets = %+v, want exactly one claude-code target", payload.Targets)
			}
			if payload.Targets[0].State != c.wantState {
				t.Fatalf("targets[0].state = %q, want %q", payload.Targets[0].State, c.wantState)
			}

			gotCodes := map[string]bool{}
			for _, f := range payload.Findings {
				gotCodes[f.Code] = true
			}
			for _, code := range c.wantCodes {
				if !gotCodes[code] {
					t.Errorf("findings = %+v, want a finding with code %q", payload.Findings, code)
				}
			}
			if len(c.wantCodes) == 0 && len(payload.Findings) != 0 {
				t.Errorf("findings = %+v, want none", payload.Findings)
			}

			if c.wantExit == 1 {
				envelope := parseErrorEnvelope(t, r.stderr)
				if envelope.Error.Code != "doctor.findings-present" {
					t.Errorf("error code = %q, want doctor.findings-present", envelope.Error.Code)
				}
			} else if r.stderr != "" {
				t.Errorf("stderr = %q, want empty on a passing doctor run", r.stderr)
			}
		})
	}
}

// TestDoctor_TextMode_StateLine asserts the text-mode "<harness>: <state>
// at <dir>" line appears ahead of the "no issues found" line.
func TestDoctor_TextMode_StateLine(t *testing.T) {
	skills := t.TempDir()
	env := []string{"LDS_CLAUDE_SKILLS_DIR=" + skills}
	skillDir := filepath.Join(skills, "lds")

	r := run(t, env, "plumbing", "doctor")
	if r.exitCode != 0 {
		t.Fatalf("doctor: exit=%d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	wantLine := fmt.Sprintf("claude-code: missing at %s", skillDir)
	stateIdx := strings.Index(r.stdout, wantLine)
	if stateIdx < 0 {
		t.Fatalf("doctor stdout = %q, want it to contain %q", r.stdout, wantLine)
	}
	issuesIdx := strings.Index(r.stdout, "no issues found")
	if issuesIdx < 0 {
		t.Fatalf("doctor stdout = %q, want it to still say no issues found", r.stdout)
	}
	if stateIdx > issuesIdx {
		t.Fatalf("doctor stdout = %q, want the state line ahead of the findings/no-issues line", r.stdout)
	}
}
