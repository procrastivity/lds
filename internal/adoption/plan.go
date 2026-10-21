package adoption

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path"
	"sort"
	"strings"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	planIncompleteFacts = "refusal.incomplete-facts"
	planLegacyDocs      = "refusal.legacy-docs"
	planOutputCollision = "refusal.output-collision"
	planUnsafeTarget    = "refusal.unsafe-target"
)

// PlanFinding records a policy refusal or a non-vetoing link diagnostic.
type PlanFinding struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Line    int    `json:"line,omitempty"`
	Target  string `json:"target,omitempty"`
	Message string `json:"message"`
}

// PlanOutput classifies one of the four approved adoption output files.
type PlanOutput struct {
	Path                    string       `json:"path"`
	Classification          string       `json:"classification"`
	Reason                  string       `json:"reason"`
	Content                 string       `json:"content,omitempty"`
	ContentSHA256           string       `json:"contentSha256,omitempty"`
	Asset                   *RenderAsset `json:"asset,omitempty"`
	ExistingSHA256          string       `json:"existingSha256,omitempty"`
	ExistingContent         string       `json:"existingContent,omitempty"`
	ExistingContentBase64   string       `json:"existingContentBase64,omitempty"`
	ExistingContentEncoding string       `json:"existingContentEncoding,omitempty"`
	ExistingProvenance      string       `json:"existingProvenance,omitempty"`
	AffectedLinks           []Link       `json:"affectedLinks,omitempty"`
}

// PreservedPath is byte-identity evidence for an existing target file.
type PreservedPath struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}

// ParentIdentity binds an existing approved-output ancestor to its directory.
type ParentIdentity struct {
	Path     string `json:"path"`
	Identity string `json:"identity"`
}

// ReportOnlyNotice describes information that is not an approved output.
type ReportOnlyNotice struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// PlanProjection contains every review field included in the plan digest.
type PlanProjection struct {
	SchemaVersion              int                `json:"schemaVersion"`
	Operation                  string             `json:"operation"`
	TargetRoot                 string             `json:"targetRoot"`
	TargetIdentity             string             `json:"targetIdentity"`
	FactsDigest                string             `json:"factsDigest"`
	Outputs                    []PlanOutput       `json:"outputs"`
	NecessaryParentDirectories []string           `json:"necessaryParentDirectories"`
	ExistingParentDirectories  []ParentIdentity   `json:"existingParentDirectories"`
	Preserved                  []PreservedPath    `json:"preserved"`
	ReportOnly                 []ReportOnlyNotice `json:"reportOnly"`
	Veto                       bool               `json:"veto"`
	Findings                   []PlanFinding      `json:"findings"`
	Skipped                    []Skip             `json:"skipped"`
}

// PlanResult is a read-only preflight and its digest. A nil PlanDigest is
// reserved for incomplete facts, which have no eligible reviewed result.
type PlanResult struct {
	PlanProjection
	PlanDigest *string `json:"planDigest"`
}

// PlanRefusal carries a complete, reviewable plan that vetoes all creates.
type PlanRefusal struct {
	Code    string
	Message string
}

func (e *PlanRefusal) Error() string { return e.Message }

type planRefusalCause struct {
	code    string
	path    string
	message string
}

// Plan renders and classifies the four approved outputs without writing to
// root. It returns a PlanRefusal only when the result is complete enough to
// review; a missing target or an unexpected read failure is an ordinary error.
func Plan(root, factsPath string) (PlanResult, error) {
	result := PlanResult{PlanProjection: PlanProjection{
		SchemaVersion: 1, Operation: "docs plan", TargetRoot: root,
		Outputs: []PlanOutput{}, NecessaryParentDirectories: []string{},
		ExistingParentDirectories: []ParentIdentity{},
		Preserved:                 []PreservedPath{}, ReportOnly: []ReportOnlyNotice{},
		Findings: []PlanFinding{}, Skipped: []Skip{},
	}}
	data, facts, factsErr := readRenderFacts(factsPath)
	if len(data) != 0 {
		result.FactsDigest = "sha256:" + sha256Hex(data)
	}
	if factsErr != nil {
		result.Veto = true
		result.Findings = append(result.Findings, PlanFinding{
			Code: planIncompleteFacts, Message: factsErr.Error(),
		})
		refuseAllOutputs(&result, "required owner-approved facts are incomplete or invalid")
		return result, &PlanRefusal{Code: planIncompleteFacts, Message: factsErr.Error()}
	}

	rendered, renderErr := renderBundleOutputs(facts)
	if renderErr != nil {
		return PlanResult{}, renderErr
	}

	rootDir, err := openRoot(root)
	if err != nil {
		if errors.Is(err, ErrUnsafeTarget) {
			result.Veto = true
			result.Findings = append(result.Findings, PlanFinding{Code: planUnsafeTarget, Path: ".", Message: "target root is unsafe and will not be traversed"})
			refuseRenderedOutputs(&result, rendered, "target root cannot be traversed safely")
			if err := setPlanDigest(&result); err != nil {
				return PlanResult{}, err
			}
			return result, &PlanRefusal{Code: planUnsafeTarget, Message: err.Error()}
		}
		return result, fmt.Errorf("open plan target: %w", err)
	}
	defer func() { _ = rootDir.Close() }()
	result.TargetIdentity, err = rootIdentity(rootDir)
	if err != nil {
		return result, fmt.Errorf("identify plan target: %w", err)
	}

	inventory, err := Inspect(root)
	if err != nil {
		if errors.Is(err, ErrUnsafeTarget) {
			result.Veto = true
			result.Findings = append(result.Findings, PlanFinding{Code: planUnsafeTarget, Path: ".", Message: "target root changed to an unsafe path during inventory"})
			refuseRenderedOutputs(&result, rendered, "target root cannot be inventoried safely")
			if err := setPlanDigest(&result); err != nil {
				return PlanResult{}, err
			}
			return result, &PlanRefusal{Code: planUnsafeTarget, Message: err.Error()}
		}
		if errors.Is(err, fs.ErrPermission) {
			result.Veto = true
			result.Findings = append(result.Findings, PlanFinding{Code: "refusal.unreadable-target", Path: "docs/", Message: "scoped target content is unreadable; no writes are authorized"})
			refuseRenderedOutputs(&result, rendered, "target evidence is unreadable; no writes are authorized")
			if err := setPlanDigest(&result); err != nil {
				return PlanResult{}, err
			}
			return result, &PlanRefusal{Code: "refusal.unreadable-target", Message: "scoped target content is unreadable; no writes are authorized"}
		}
		return result, fmt.Errorf("inventory plan target: %w", err)
	}
	result.Skipped = append(result.Skipped, inventory.Skipped...)
	contents, err := collectPreserved(rootDir, inventory, &result)
	if err != nil {
		return result, fmt.Errorf("collect preserved target evidence: %w", err)
	}
	if _, ok := contents["README.md"]; ok {
		result.ReportOnly = append(result.ReportOnly, ReportOnlyNotice{
			Path: "README.md", Message: "Root README signpost opportunity only; it is not an adoption output and will not be edited.",
		})
	}

	legacyKind := classifyLegacyDocs(contents)
	if legacyKind != "" {
		result.Veto = true
		result.Findings = append(result.Findings, PlanFinding{
			Code: planLegacyDocs, Path: "docs/", Message: "existing docs/ content is " + legacyKind + " legacy LDS evidence; no migration is performed",
		})
		refuseRenderedOutputs(&result, rendered, "legacy documentation under docs/ requires manual review; no migration is authorized")
	} else {
		for _, output := range rendered {
			planned, parents, cause, err := classifyOutput(rootDir, output)
			if err != nil {
				return result, fmt.Errorf("classify %s: %w", output.Path, err)
			}
			result.Outputs = append(result.Outputs, planned)
			result.NecessaryParentDirectories = append(result.NecessaryParentDirectories, parents...)
			if cause != nil {
				result.Veto = true
				result.Findings = append(result.Findings, PlanFinding{Code: cause.code, Path: cause.path, Message: cause.message})
			}
		}
	}
	if !result.Veto {
		result.ExistingParentDirectories, err = collectParentIdentities(rootDir, rendered)
		if err != nil {
			return result, fmt.Errorf("inspect output parents: %w", err)
		}
	}

	result.Findings = append(result.Findings, planLinkFindings(rootDir, inventory, rendered, &result.Skipped)...)
	sortPlan(&result)
	if result.Veto {
		if err := setPlanDigest(&result); err != nil {
			return PlanResult{}, err
		}
		return result, refusalForPlan(result)
	}
	if err := setPlanDigest(&result); err != nil {
		return PlanResult{}, err
	}
	return result, nil
}

func refuseAllOutputs(result *PlanResult, reason string) {
	for _, spec := range renderAssets {
		result.Outputs = append(result.Outputs, PlanOutput{Path: spec.output, Classification: "refusal", Reason: reason})
	}
	sortPlan(result)
}

func refuseRenderedOutputs(result *PlanResult, outputs []RenderOutput, reason string) {
	for _, output := range outputs {
		result.Outputs = append(result.Outputs, PlanOutput{
			Path: output.Path, Classification: "refusal", Reason: reason,
			Content: output.Content, ContentSHA256: output.SHA256, Asset: &output.Asset,
		})
	}
	sortPlan(result)
}

func classifyOutput(root *os.File, output RenderOutput) (PlanOutput, []string, *planRefusalCause, error) {
	probe, err := probeTargetFile(root, output.Path)
	if err != nil {
		if errors.Is(err, ErrUnsafeTarget) || errors.Is(err, syscall.ENOTDIR) {
			return PlanOutput{Path: output.Path, Classification: "refusal", Reason: "an output or ancestor is not a safe regular-file path", Content: output.Content, ContentSHA256: output.SHA256, Asset: &output.Asset}, nil, &planRefusalCause{
				code: planUnsafeTarget, path: output.Path, message: "output path or ancestor is unsafe; no writes are authorized",
			}, nil
		}
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrInvalid) {
			return PlanOutput{Path: output.Path, Classification: "refusal", Reason: "occupied output or ancestor is unreadable", Content: output.Content, ContentSHA256: output.SHA256, Asset: &output.Asset}, nil, &planRefusalCause{
				code: planUnsafeTarget, path: output.Path, message: "occupied output or ancestor cannot be read safely",
			}, nil
		}
		return PlanOutput{}, nil, nil, err
	}
	if len(probe.missingParents) > 0 {
		return PlanOutput{Path: output.Path, Classification: "create", Reason: "approved output is absent; parents are absent and listed separately", Content: output.Content, ContentSHA256: output.SHA256, Asset: &output.Asset}, probe.missingParents, nil, nil
	}
	if !probe.exists {
		return PlanOutput{Path: output.Path, Classification: "create", Reason: "approved output is absent", Content: output.Content, ContentSHA256: output.SHA256, Asset: &output.Asset}, nil, nil, nil
	}
	existingHash := sha256Hex(probe.data)
	if string(probe.data) == output.Content {
		return PlanOutput{Path: output.Path, Classification: "identical-untouched", Reason: "occupied bytes are identical; no write is proposed", Content: output.Content, ContentSHA256: output.SHA256, Asset: &output.Asset, ExistingSHA256: existingHash}, nil, nil, nil
	}
	links, _ := extractLinks(output.Path, string(probe.data))
	planned := PlanOutput{
		Path: output.Path, Classification: "different-collision", Reason: "occupied bytes differ; report for manual review outside this contract; no writes in this attempt",
		Content: output.Content, ContentSHA256: output.SHA256, Asset: &output.Asset, ExistingSHA256: existingHash,
		ExistingProvenance: "unknown; content and harness metadata do not establish project-document ownership",
		AffectedLinks:      links,
	}
	if utf8.Valid(probe.data) {
		planned.ExistingContent = string(probe.data)
	} else {
		planned.ExistingContentBase64 = base64.StdEncoding.EncodeToString(probe.data)
		planned.ExistingContentEncoding = "base64"
	}
	return planned, nil, &planRefusalCause{
		code: planOutputCollision, path: output.Path, message: "occupied output differs from the approved render; all creates are vetoed",
	}, nil
}

type probedFile struct {
	exists         bool
	data           []byte
	missingParents []string
}

func probeTargetFile(root *os.File, relative string) (probedFile, error) {
	components := strings.Split(relative, "/")
	parent := root
	var opened []*os.File
	defer func() {
		for _, f := range opened {
			_ = f.Close()
		}
	}()
	for i, component := range components[:len(components)-1] {
		next, err := openChild(parent, component)
		if errors.Is(err, os.ErrNotExist) {
			missing := make([]string, 0, len(components)-i-1)
			for last := i; last < len(components)-1; last++ {
				missing = append(missing, strings.Join(components[:last+1], "/")+"/")
			}
			return probedFile{missingParents: missing}, nil
		}
		if err != nil {
			return probedFile{}, err
		}
		info, err := next.Stat()
		if err != nil {
			_ = next.Close()
			return probedFile{}, err
		}
		if !info.IsDir() {
			_ = next.Close()
			return probedFile{}, syscall.ENOTDIR
		}
		opened = append(opened, next)
		parent = next
	}
	file, err := openChild(parent, components[len(components)-1])
	if errors.Is(err, os.ErrNotExist) {
		return probedFile{}, nil
	}
	if err != nil {
		return probedFile{}, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return probedFile{}, err
	}
	if !info.Mode().IsRegular() {
		return probedFile{}, syscall.ENOTDIR
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return probedFile{}, err
	}
	return probedFile{exists: true, data: data}, nil
}

func collectPreserved(root *os.File, inventory Inventory, result *PlanResult) (map[string][]byte, error) {
	contents := make(map[string][]byte, len(inventory.Files)+4)
	for _, file := range inventory.Files {
		data, err := readTargetFile(root, file.Path)
		if err != nil {
			return nil, err
		}
		contents[file.Path] = data
	}
	entries, err := root.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		file, err := openChild(root, entry.Name())
		if err != nil {
			if errors.Is(err, ErrUnsafeTarget) {
				continue
			}
			return nil, err
		}
		info, err := file.Stat()
		if err != nil {
			_ = file.Close()
			return nil, err
		}
		if info.Mode().IsRegular() {
			data, err := io.ReadAll(file)
			if err != nil {
				_ = file.Close()
				return nil, err
			}
			contents[entry.Name()] = data
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
	}
	paths := make([]string, 0, len(contents))
	for name := range contents {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		data := contents[name]
		result.Preserved = append(result.Preserved, PreservedPath{Path: name, SHA256: sha256Hex(data), Size: len(data)})
	}
	return contents, nil
}

func readTargetFile(root *os.File, relative string) ([]byte, error) {
	components := strings.Split(relative, "/")
	parent := root
	var opened []*os.File
	defer func() {
		for _, f := range opened {
			_ = f.Close()
		}
	}()
	for _, component := range components[:len(components)-1] {
		next, err := openChild(parent, component)
		if err != nil {
			return nil, err
		}
		opened = append(opened, next)
		parent = next
	}
	file, err := openChild(parent, components[len(components)-1])
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return io.ReadAll(file)
}

func collectParentIdentities(root *os.File, outputs []RenderOutput) ([]ParentIdentity, error) {
	seen := map[string]string{}
	for _, output := range outputs {
		parent := root
		var opened []*os.File
		for i, component := range strings.Split(output.Path, "/")[:strings.Count(output.Path, "/")] {
			next, err := openChild(parent, component)
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			if err != nil {
				for _, file := range opened {
					_ = file.Close()
				}
				return nil, err
			}
			identity, err := rootIdentity(next)
			if err != nil {
				_ = next.Close()
				for _, file := range opened {
					_ = file.Close()
				}
				return nil, err
			}
			name := strings.Join(strings.Split(output.Path, "/")[:i+1], "/") + "/"
			if previous, ok := seen[name]; ok && previous != identity {
				_ = next.Close()
				for _, file := range opened {
					_ = file.Close()
				}
				return nil, ErrUnsafeTarget
			}
			seen[name] = identity
			opened = append(opened, next)
			parent = next
		}
		for _, file := range opened {
			_ = file.Close()
		}
	}
	result := make([]ParentIdentity, 0, len(seen))
	for name, identity := range seen {
		result = append(result, ParentIdentity{Path: name, Identity: identity})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func classifyLegacyDocs(contents map[string][]byte) string {
	for name, data := range contents {
		if !strings.HasPrefix(name, "docs/") {
			continue
		}
		text := strings.ToLower(strings.Join(strings.Fields(string(data)), " "))
		if strings.Contains(text, "this `docs/` directory is the project's existing layered documentation system") ||
			strings.Contains(text, "adopt the layered documentation system under `docs/`") ||
			strings.Contains(text, "canonical lds root") ||
			strings.Contains(text, "installed layered documentation system") && strings.Contains(text, "docs/") ||
			strings.HasPrefix(name, "docs/decisions/") && strings.Contains(text, "adopt the layered documentation system") && strings.Contains(text, "**status**: accepted") && strings.Contains(text, "## decision") {
			return "recognizable"
		}
	}
	var decisions, product, reference, ambiguousCue bool
	for name, data := range contents {
		if !strings.HasPrefix(name, "docs/") {
			continue
		}
		switch {
		case strings.HasPrefix(name, "docs/decisions/"):
			decisions = true
		case strings.HasPrefix(name, "docs/product/"):
			product = true
		case strings.HasPrefix(name, "docs/reference/"):
			reference = true
		}
		text := strings.ToLower(strings.Join(strings.Fields(string(data)), " "))
		if strings.Contains(text, "does not identify a canonical documentation system") || strings.Contains(text, "ownership") && strings.Contains(text, "unclear") {
			ambiguousCue = true
		}
	}
	if decisions && product && reference && ambiguousCue {
		return "ambiguous"
	}
	return ""
}

func planLinkFindings(root *os.File, inventory Inventory, rendered []RenderOutput, skipped *[]Skip) []PlanFinding {
	overlay := make(map[string]struct{}, len(rendered))
	links := make([]Link, 0, len(inventory.Links))
	for _, output := range rendered {
		overlay[output.Path] = struct{}{}
	}
	for _, link := range inventory.Links {
		if _, replaced := overlay[link.Source]; !replaced {
			links = append(links, link)
		}
	}
	for _, output := range rendered {
		found, ignored := extractLinks(output.Path, output.Content)
		links = append(links, found...)
		*skipped = append(*skipped, ignored...)
	}
	findings := []PlanFinding{}
	for _, link := range links {
		target := link.Target
		if strings.Contains(target, "?") {
			*skipped = append(*skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.query-skipped", Reason: "query-bearing link is not checked"})
			continue
		}
		hasFragment := false
		if i := strings.IndexByte(target, '#'); i >= 0 {
			target, hasFragment = target[:i], true
		}
		if target == "" {
			*skipped = append(*skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.fragment-skipped", Reason: "fragment anchor is not checked"})
			continue
		}
		if path.IsAbs(target) || strings.HasPrefix(target, "\\") {
			*skipped = append(*skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.unsafe-link-skipped", Reason: "absolute local target is not followed"})
			continue
		}
		candidate := path.Clean(path.Join(path.Dir(link.Source), target))
		if candidate == ".." || strings.HasPrefix(candidate, "../") {
			*skipped = append(*skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.unsafe-link-skipped", Reason: "target escapes the explicit root"})
			continue
		}
		if _, planned := overlay[candidate]; planned {
			if hasFragment {
				*skipped = append(*skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.fragment-skipped", Reason: "fragment anchor is not checked"})
			}
			continue
		}
		regular, err := regularTarget(root, candidate)
		switch {
		case errors.Is(err, ErrUnsafeTarget):
			*skipped = append(*skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.unsafe-link-skipped", Reason: "symlink target is not followed"})
		case errors.Is(err, os.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
			findings = append(findings, PlanFinding{Code: "validation.local-target-missing", Path: link.Source, Line: link.Line, Target: link.Target, Message: "local file target is missing: " + link.Target})
		case err != nil:
			*skipped = append(*skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.target-unreadable-skipped", Reason: "target could not be inspected"})
		case !regular:
			*skipped = append(*skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.not-file-skipped", Reason: "local target is not a regular file"})
		case hasFragment:
			*skipped = append(*skipped, Skip{Source: link.Source, Line: link.Line, Code: "validation.fragment-skipped", Reason: "fragment anchor is not checked"})
		}
	}
	return findings
}

func sortPlan(result *PlanResult) {
	sort.Slice(result.Outputs, func(i, j int) bool { return result.Outputs[i].Path < result.Outputs[j].Path })
	sort.Strings(result.NecessaryParentDirectories)
	parents := result.NecessaryParentDirectories[:0]
	for _, parent := range result.NecessaryParentDirectories {
		if len(parents) == 0 || parents[len(parents)-1] != parent {
			parents = append(parents, parent)
		}
	}
	result.NecessaryParentDirectories = parents
	sort.Slice(result.Preserved, func(i, j int) bool { return result.Preserved[i].Path < result.Preserved[j].Path })
	sort.Slice(result.ReportOnly, func(i, j int) bool { return result.ReportOnly[i].Path < result.ReportOnly[j].Path })
	sort.SliceStable(result.Findings, func(i, j int) bool {
		a, b := result.Findings[i], result.Findings[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Code < b.Code
	})
	sort.SliceStable(result.Skipped, func(i, j int) bool {
		a, b := result.Skipped[i], result.Skipped[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Code < b.Code
	})
}

func refusalForPlan(result PlanResult) *PlanRefusal {
	for _, finding := range result.Findings {
		if strings.HasPrefix(finding.Code, "refusal.") {
			return &PlanRefusal{Code: finding.Code, Message: finding.Message}
		}
	}
	return &PlanRefusal{Code: planOutputCollision, Message: "plan vetoes all creates"}
}

func setPlanDigest(result *PlanResult) error {
	data, err := json.Marshal(result.PlanProjection)
	if err != nil {
		return err
	}
	canonical, err := canonicalJSON(data)
	if err != nil {
		return err
	}
	digest := "sha256:" + sha256Hex(canonical)
	result.PlanDigest = &digest
	return nil
}

// CanonicalPlanDigest returns the RFC 8785 digest of every displayed plan
// field except planDigest. It is exported to allow independent CLI checks.
func CanonicalPlanDigest(result PlanResult) (string, error) {
	data, err := json.Marshal(result.PlanProjection)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalJSON(data)
	if err != nil {
		return "", err
	}
	return "sha256:" + sha256Hex(canonical), nil
}

func canonicalJSON(data []byte) ([]byte, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, err
	}
	var out strings.Builder
	if err := appendCanonical(&out, value); err != nil {
		return nil, err
	}
	return []byte(out.String()), nil
}

func appendCanonical(out *strings.Builder, value any) error {
	switch item := value.(type) {
	case nil:
		out.WriteString("null")
	case bool:
		if item {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case string:
		appendCanonicalString(out, item)
	case json.Number:
		integer, ok := new(big.Int).SetString(string(item), 10)
		if !ok {
			return fmt.Errorf("plan contains a non-integer number %q", item)
		}
		out.WriteString(integer.String())
	case []any:
		out.WriteByte('[')
		for i, element := range item {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := appendCanonical(out, element); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(item))
		for key := range item {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return utf16Less(keys[i], keys[j]) })
		out.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := appendCanonical(out, key); err != nil {
				return err
			}
			out.WriteByte(':')
			if err := appendCanonical(out, item[key]); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON value %T", value)
	}
	return nil
}

func appendCanonicalString(out *strings.Builder, value string) {
	const hexDigits = "0123456789abcdef"
	out.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"', '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		case '\b':
			out.WriteString(`\b`)
		case '\t':
			out.WriteString(`\t`)
		case '\n':
			out.WriteString(`\n`)
		case '\f':
			out.WriteString(`\f`)
		case '\r':
			out.WriteString(`\r`)
		default:
			if r < 0x20 {
				out.WriteString(`\u00`)
				out.WriteByte(hexDigits[byte(r)>>4])
				out.WriteByte(hexDigits[byte(r)&0x0f])
			} else {
				out.WriteRune(r)
			}
		}
	}
	out.WriteByte('"')
}

func utf16Less(left, right string) bool {
	a, b := utf16.Encode([]rune(left)), utf16.Encode([]rune(right))
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}
