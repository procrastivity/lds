package adoption

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"text/template"
	"text/template/parse"
	"unicode"
	"unicode/utf8"

	"github.com/procrastivity/lds/internal/asset"
)

// ErrInvalidRenderInput marks a facts or template validation failure.
var ErrInvalidRenderInput = errors.New("invalid render input")

// ErrProjectAsset marks a failed explicitly selected project asset lookup.
var ErrProjectAsset = errors.New("project asset unavailable")

// RenderAsset identifies the selected template asset and its source hash.
type RenderAsset struct {
	RelativeName string `json:"relativeName"`
	Source       string `json:"source"`
	SHA256       string `json:"sha256"`
}

// RenderOutput is one rendered adoption file and its provenance.
type RenderOutput struct {
	Path    string      `json:"path"`
	Content string      `json:"content"`
	SHA256  string      `json:"sha256"`
	Asset   RenderAsset `json:"asset"`
}

// FactBundle contains caller-authored adoption assertions and approval metadata.
// Its shape alone does not establish completeness, approval or verified facts.
type FactBundle struct {
	DisplayName             string `json:"DisplayName"`
	Purpose                 string `json:"Purpose"`
	CurrentCapability       string `json:"CurrentCapability"`
	DisplayNameSource       string `json:"DisplayNameSource"`
	PurposeSource           string `json:"PurposeSource"`
	CurrentCapabilitySource string `json:"CurrentCapabilitySource"`
	Prerequisites           string `json:"Prerequisites"`
	OperationName           string `json:"OperationName"`
	OperationSummary        string `json:"OperationSummary"`
	Invocation              string `json:"Invocation"`
	Input                   string `json:"Input"`
	OutputEvidence          string `json:"OutputEvidence"`
	OperationSource         string `json:"OperationSource"`
	Limitations             string `json:"Limitations"`
	Approval                struct {
		Approver      string   `json:"approver"`
		ApprovedFacts []string `json:"approvedFacts"`
		Status        string   `json:"status"`
	} `json:"approval"`
}

type renderAssetSpec struct {
	assetPath string
	output    string
}

var renderAssets = []renderAssetSpec{
	{assetPath: "templates/adoption/docs/README.md", output: "docs/README.md"},
	{assetPath: "templates/adoption/docs/reference/usage.md", output: "docs/reference/usage.md"},
	{assetPath: "templates/adoption/engineering/README.md", output: "engineering/README.md"},
	{assetPath: "templates/adoption/engineering/product/overview.md", output: "engineering/product/overview.md"},
}

var allowedRenderFields = map[string]struct{}{
	"DisplayName": {}, "Purpose": {}, "CurrentCapability": {},
	"DisplayNameSource": {}, "PurposeSource": {}, "CurrentCapabilitySource": {},
	"Prerequisites": {}, "OperationName": {}, "OperationSummary": {},
	"Invocation": {}, "Input": {}, "OutputEvidence": {},
	"OperationSource": {}, "Limitations": {},
}

var (
	localSourceDrive  = regexp.MustCompile(`^[A-Za-z]:`)
	templateFenceLine = regexp.MustCompile("(?m)^ {0,3}(`{3,}|~{3,})")
)

// Render validates the structure of an owner-approved fact bundle and renders
// only the four approved adoption assets. Validation does not authenticate the
// approver or independently verify project claims. It reads the explicit
// target root and facts file; it never writes to either.
func Render(root, factsPath, projectRoot string) ([]RenderOutput, error) {
	rootDir, err := openRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open render target: %w", err)
	}
	if err := rootDir.Close(); err != nil {
		return nil, fmt.Errorf("close render target: %w", err)
	}

	_, facts, err := readRenderFacts(factsPath)
	if err != nil {
		return nil, err
	}
	return renderBundleOutputs(facts, projectRoot)
}

func readRenderFacts(factsPath string) ([]byte, FactBundle, error) {
	data, err := os.ReadFile(factsPath)
	if err != nil {
		return nil, FactBundle{}, invalidRender("cannot read facts file")
	}
	facts, err := DecodeFactBundle(data)
	if err != nil {
		return data, FactBundle{}, err
	}
	if err := ValidateFactBundle(facts); err != nil {
		return data, FactBundle{}, invalidRender("validate facts: %v", err)
	}
	return data, facts, nil
}

// DecodeFactBundle strictly decodes caller assertions without requiring approval
// or nonempty values. ValidateFactBundle is still required before adoption.
func DecodeFactBundle(data []byte) (FactBundle, error) {
	if !utf8.Valid(data) {
		return FactBundle{}, invalidRender("facts file is not valid UTF-8")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return FactBundle{}, invalidRender("decode facts file: %v", err)
	}
	if err := validateExactFactKeys(data); err != nil {
		return FactBundle{}, invalidRender("decode facts file: %v", err)
	}

	var facts FactBundle
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&facts); err != nil {
		return FactBundle{}, invalidRender("decode facts file: %v", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return FactBundle{}, invalidRender("decode facts file: %v", err)
	}
	return facts, nil
}

func renderBundleOutputs(facts FactBundle, projectRoot string) ([]RenderOutput, error) {
	values := map[string]string{
		"DisplayName":             escapeMarkdownText(facts.DisplayName),
		"Purpose":                 escapeMarkdownText(facts.Purpose),
		"CurrentCapability":       escapeMarkdownText(facts.CurrentCapability),
		"DisplayNameSource":       escapeMarkdownText(facts.DisplayNameSource),
		"PurposeSource":           escapeMarkdownText(facts.PurposeSource),
		"CurrentCapabilitySource": escapeMarkdownText(facts.CurrentCapabilitySource),
		"Prerequisites":           escapeMarkdownText(facts.Prerequisites),
		"OperationName":           escapeMarkdownText(facts.OperationName),
		"OperationSummary":        escapeMarkdownText(facts.OperationSummary),
		"Invocation":              fencedInvocation(facts.Invocation),
		"Input":                   escapeMarkdownText(facts.Input),
		"OutputEvidence":          escapeMarkdownText(facts.OutputEvidence),
		"OperationSource":         escapeMarkdownText(facts.OperationSource),
		"Limitations":             escapeMarkdownText(facts.Limitations),
	}

	outputs := make([]RenderOutput, 0, len(renderAssets))
	for _, spec := range renderAssets {
		var resolved asset.Resolved
		var err error
		if projectRoot != "" {
			resolved, err = asset.ResolveIn(projectRoot, spec.assetPath)
			if err != nil {
				return nil, fmt.Errorf("%w: resolve asset %q: %v", ErrProjectAsset, spec.assetPath, err)
			}
		} else {
			// Preserve the legacy override validation classification.
			var present bool
			resolved, present, err = asset.ReadOverride(spec.assetPath)
			if err != nil {
				return nil, invalidRender("asset %q override: %v", spec.assetPath, err)
			}
			if !present {
				resolved, err = asset.Resolve(spec.assetPath)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("resolve asset %q: %w", spec.assetPath, err)
		}
		parsed, err := parseStrictAsset(spec.assetPath, resolved.Bytes())
		if err != nil {
			return nil, invalidRender("asset %q: %v", spec.assetPath, err)
		}
		var content bytes.Buffer
		if err := parsed.Execute(&content, values); err != nil {
			return nil, invalidRender("render asset %q: %v", spec.assetPath, err)
		}
		contentBytes := content.Bytes()
		outputs = append(outputs, RenderOutput{
			Path:    spec.output,
			Content: content.String(),
			SHA256:  sha256Hex(contentBytes),
			Asset: RenderAsset{
				RelativeName: spec.assetPath,
				Source:       resolved.Source.String(),
				SHA256:       sha256Hex(resolved.Bytes()),
			},
		})
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	return outputs, nil
}

// ValidateFactAssertions checks complete assertions and safe citation syntax,
// without requiring approval or verifying the truth of any assertion.
func ValidateFactAssertions(facts FactBundle) error {
	stringsToValidate := []struct {
		name  string
		value string
	}{
		{"DisplayName", facts.DisplayName},
		{"Purpose", facts.Purpose},
		{"CurrentCapability", facts.CurrentCapability},
		{"DisplayNameSource", facts.DisplayNameSource},
		{"PurposeSource", facts.PurposeSource},
		{"CurrentCapabilitySource", facts.CurrentCapabilitySource},
		{"Prerequisites", facts.Prerequisites},
		{"OperationName", facts.OperationName},
		{"OperationSummary", facts.OperationSummary},
		{"Invocation", facts.Invocation},
		{"Input", facts.Input},
		{"OutputEvidence", facts.OutputEvidence},
		{"OperationSource", facts.OperationSource},
		{"Limitations", facts.Limitations},
	}
	for _, field := range stringsToValidate {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
		for _, r := range field.value {
			if unicode.IsControl(r) && r != '\n' {
				return fmt.Errorf("%s contains a control character", field.name)
			}
		}
	}
	for _, field := range []struct {
		name  string
		value string
	}{{"DisplayNameSource", facts.DisplayNameSource}, {"PurposeSource", facts.PurposeSource}, {"CurrentCapabilitySource", facts.CurrentCapabilitySource}, {"OperationSource", facts.OperationSource}} {
		if strings.ContainsAny(field.value, "\r\n\t\\") || hasUnsafeLocalSource(field.value) || !hasSourceCitation(field.value) {
			return fmt.Errorf("%s must contain a safe relative source citation", field.name)
		}
	}
	return nil
}

// ValidateFactBundle checks completeness, safe citation syntax and explicit
// approval structure, not the truth of assertions or the approver's identity.
func ValidateFactBundle(facts FactBundle) error {
	if err := ValidateFactAssertions(facts); err != nil {
		return err
	}
	for _, field := range []struct{ name, value string }{
		{"approval.approver", facts.Approval.Approver},
		{"approval.status", facts.Approval.Status},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
		if strings.ContainsFunc(field.value, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) {
			return fmt.Errorf("%s contains a control character", field.name)
		}
	}
	// Free-form status text cannot reliably distinguish approval from its
	// negation (for example, "disapproved" or "approval denied").
	if facts.Approval.Status != "approved" && facts.Approval.Status != "test-only simulated approval; not a real project authorization" {
		return errors.New("approval must identify an approver and assert approval status")
	}
	if len(facts.Approval.ApprovedFacts) != 4 {
		return errors.New("approval.approvedFacts must cover all four fact groups")
	}
	requiredGroups := []string{"DisplayName", "Purpose", "CurrentCapability", "VerifiedUserOperation"}
	wantApproved := make(map[string]bool, len(requiredGroups))
	for _, name := range requiredGroups {
		wantApproved[name] = false
	}
	for _, name := range facts.Approval.ApprovedFacts {
		if _, ok := wantApproved[name]; !ok || wantApproved[name] {
			return fmt.Errorf("approval.approvedFacts contains unexpected or duplicate group %q", name)
		}
		wantApproved[name] = true
	}
	for _, name := range requiredGroups {
		if !wantApproved[name] {
			return fmt.Errorf("approval.approvedFacts omits %s", name)
		}
	}
	return nil
}

func hasSourceCitation(source string) bool {
	for _, token := range strings.FieldsFunc(source, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(";,()[]", r)
	}) {
		token = strings.Trim(token, ".:!?'\"`")
		if strings.HasPrefix(token, "https://") || strings.HasPrefix(token, "http://") || strings.Contains(token, ".") || strings.Contains(token, "/") {
			return true
		}
	}
	return false
}

func hasUnsafeLocalSource(source string) bool {
	for _, token := range strings.FieldsFunc(source, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(";,()[]", r)
	}) {
		token = strings.Trim(token, ".:!?'\"`")
		if strings.HasPrefix(token, "https://") || strings.HasPrefix(token, "http://") {
			continue
		}
		local := strings.SplitN(token, "#", 2)[0]
		if strings.Contains(local, `\`) || path.IsAbs(local) || localSourceDrive.MatchString(local) {
			return true
		}
		for _, component := range strings.Split(local, "/") {
			if component == ".." {
				return true
			}
		}
	}
	return strings.Contains(strings.ToLower(source), "file://")
}

func parseStrictAsset(name string, data []byte) (*template.Template, error) {
	if !utf8.Valid(data) {
		return nil, errors.New("asset is not valid UTF-8")
	}
	for _, r := range string(data) {
		if unicode.IsControl(r) && r != '\n' {
			return nil, errors.New("asset contains a control character")
		}
	}
	parsed, err := template.New(name).Option("missingkey=error").Parse(string(data))
	if err != nil {
		return nil, err
	}
	for _, defined := range parsed.Templates() {
		if err := validateTemplateNodes(defined.Root.Nodes); err != nil {
			return nil, err
		}
		if err := validateInvocationPlacement(name, defined.Root.Nodes); err != nil {
			return nil, err
		}
	}
	return parsed, nil
}

// Invocation expands into a multiline fenced block, so it is only valid in
// the standalone invocation section of the usage asset. A field-name-only
// check would allow an override to insert that block into a heading or link.
func validateInvocationPlacement(name string, nodes []parse.Node) error {
	for i, node := range nodes {
		action, ok := node.(*parse.ActionNode)
		if !ok || action.Pipe == nil || len(action.Pipe.Cmds) != 1 || len(action.Pipe.Cmds[0].Args) != 1 {
			continue
		}
		field, ok := action.Pipe.Cmds[0].Args[0].(*parse.FieldNode)
		if !ok || len(field.Ident) != 1 || field.Ident[0] != "Invocation" {
			continue
		}
		if name != "templates/adoption/docs/reference/usage.md" || i == 0 || i+1 == len(nodes) {
			return errors.New("invocation requires a standalone usage section")
		}
		before, beforeOK := nodes[i-1].(*parse.TextNode)
		after, afterOK := nodes[i+1].(*parse.TextNode)
		if !beforeOK || !afterOK || !bytes.HasSuffix(before.Text, []byte("### Invocation and input\n\n")) || !bytes.HasPrefix(after.Text, []byte("\n\nInput: ")) {
			return errors.New("invocation requires a standalone usage section")
		}
		// A fence opened earlier in an override would turn this apparent
		// paragraph into code and let the generated fence terminate it.
		for _, prior := range nodes[:i] {
			if text, ok := prior.(*parse.TextNode); ok && templateFenceLine.Match(text.Text) {
				return errors.New("invocation cannot follow a template code fence")
			}
		}
	}
	return nil
}

func validateTemplateNodes(nodes []parse.Node) error {
	for _, node := range nodes {
		switch n := node.(type) {
		case *parse.TextNode:
		case *parse.ActionNode:
			if n.Pipe == nil || len(n.Pipe.Decl) != 0 || len(n.Pipe.Cmds) != 1 || len(n.Pipe.Cmds[0].Args) != 1 {
				return errors.New("only direct declared-field actions are allowed")
			}
			field, ok := n.Pipe.Cmds[0].Args[0].(*parse.FieldNode)
			if !ok || len(field.Ident) != 1 {
				return errors.New("only direct declared-field actions are allowed")
			}
			if _, ok := allowedRenderFields[field.Ident[0]]; !ok {
				return fmt.Errorf("unknown template field %q", field.Ident[0])
			}
		default:
			return fmt.Errorf("template action %T is not allowed", node)
		}
	}
	return nil
}

func escapeMarkdownText(value string) string {
	const punctuation = `\` + "!\"#$%&'()*+,-./:;<=>?@[]^_`{|}~"
	lines := strings.Split(value, "\n")
	var out strings.Builder
	for i, line := range lines {
		for _, r := range line {
			if strings.ContainsRune(punctuation, r) {
				out.WriteByte('\\')
			}
			out.WriteRune(r)
		}
		if i < len(lines)-1 {
			out.WriteString("\\\n")
		}
	}
	return out.String()
}

func fencedInvocation(value string) string {
	maxRun, current := 0, 0
	for _, r := range value {
		if r == '`' {
			current++
			if current > maxRun {
				maxRun = current
			}
		} else {
			current = 0
		}
	}
	length := maxRun + 1
	if length < 3 {
		length = 3
	}
	fence := strings.Repeat("`", length)
	body := value
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return fence + "\n" + body + fence
}

func validateExactFactKeys(data []byte) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return err
	}
	wantTop := map[string]struct{}{
		"DisplayName": {}, "Purpose": {}, "CurrentCapability": {},
		"DisplayNameSource": {}, "PurposeSource": {}, "CurrentCapabilitySource": {},
		"Prerequisites": {}, "OperationName": {}, "OperationSummary": {},
		"Invocation": {}, "Input": {}, "OutputEvidence": {},
		"OperationSource": {}, "Limitations": {}, "approval": {},
	}
	for key := range top {
		if _, ok := wantTop[key]; !ok {
			return fmt.Errorf("unknown fact field %q", key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(wantTop)) {
		if value, ok := top[key]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("fact field %q must be present and non-null; use an empty string for an unknown assertion", key)
		}
	}
	var approval map[string]json.RawMessage
	if err := json.Unmarshal(top["approval"], &approval); err != nil {
		return fmt.Errorf("approval must be an object: %w", err)
	}
	wantApproval := map[string]struct{}{"approver": {}, "approvedFacts": {}, "status": {}}
	for key := range approval {
		if _, ok := wantApproval[key]; !ok {
			return fmt.Errorf("unknown approval field %q", key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(wantApproval)) {
		if value, ok := approval[key]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("approval field %q must be present and non-null", key)
		}
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return errors.New("top-level value must be an object")
	}
	if err := walkJSONObject(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func walkJSONObject(decoder *json.Decoder) error {
	seen := map[string]struct{}{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return errors.New("object key is not a string")
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate object key %q", key)
		}
		seen[key] = struct{}{}
		if err := walkJSONValue(decoder); err != nil {
			return err
		}
	}
	_, err := decoder.Token() // closing object delimiter
	return err
}

func walkJSONArray(decoder *json.Decoder) error {
	for decoder.More() {
		if err := walkJSONValue(decoder); err != nil {
			return err
		}
	}
	_, err := decoder.Token() // closing array delimiter
	return err
}

func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		return walkJSONObject(decoder)
	case '[':
		return walkJSONArray(decoder)
	default:
		return fmt.Errorf("unexpected delimiter %q", delimiter)
	}
}

func requireJSONEOF(decoder *json.Decoder) error {
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func invalidRender(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRenderInput, fmt.Sprintf(format, args...))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
