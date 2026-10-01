// Package reconcile implements the deterministic, read-only reconciliation scan.
package reconcile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/procrastivity/lds/internal/ldserr"
)

const reservedCandidatePrefix = "unmapped:"

// Input names the explicit caller-owned files and snapshot roots for one scan.
type Input struct {
	PairPath  string
	LeftRoot  string
	RightRoot string
	MapPaths  []string
	Journal   string
}

// Response is the stable v1 scan success object.
type Response struct {
	SchemaVersion int              `json:"schemaVersion"`
	Operation     string           `json:"operation"`
	OK            bool             `json:"ok"`
	PairID        string           `json:"pairId"`
	PairPath      string           `json:"pairPath"`
	PairSHA256    string           `json:"pairSha256"`
	LeftRoot      string           `json:"leftRoot"`
	RightRoot     string           `json:"rightRoot"`
	Snapshots     Snapshots        `json:"snapshots"`
	Maps          []MapObservation `json:"maps"`
	JournalPath   string           `json:"journalPath,omitempty"`
	Candidates    []Candidate      `json:"candidates"`
}

// Snapshots records the ordered project pins from the pair file.
type Snapshots struct {
	Left  Snapshot `json:"left"`
	Right Snapshot `json:"right"`
}

// Snapshot identifies one pinned project tree.
type Snapshot struct {
	ProjectID string `json:"projectId"`
	Revision  string `json:"revision"`
}

// MapObservation reports raw map bytes without interpreting map labels.
type MapObservation struct {
	ProjectID string `json:"projectId"`
	Revision  string `json:"revision"`
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
}

// Candidate contains a deterministic bucket identity and byte-level evidence.
type Candidate struct {
	CandidateID           string          `json:"candidateId"`
	CandidateKey          string          `json:"candidateKey"`
	EvidenceDigest        *string         `json:"evidenceDigest"`
	Evidence              []Evidence      `json:"evidence"`
	Left                  []PathRecord    `json:"left"`
	Right                 []PathRecord    `json:"right"`
	Confidence            string          `json:"confidence"`
	EvidenceStatus        string          `json:"evidenceStatus"`
	JournalRevision       *int            `json:"journalRevision,omitempty"`
	JournalEvidenceDigest json.RawMessage `json:"journalEvidenceDigest,omitempty"`
	LastDecision          json.RawMessage `json:"lastDecision,omitempty"`
}

// Evidence is one row in the contract's candidate digest.
type Evidence struct {
	Kind      string `json:"kind"`
	ProjectID string `json:"projectId"`
	Revision  string `json:"revision"`
	Key       string `json:"key"`
	SHA256    string `json:"sha256"`
}

// PathRecord reports whether one declared path exists as a tracked file at
// the pinned revision and the SHA-256 of its checked-out bytes.
type PathRecord struct {
	Path    string  `json:"path"`
	Present bool    `json:"present"`
	SHA256  *string `json:"sha256"`
}

type pairFile struct {
	SchemaVersion *int         `json:"schemaVersion"`
	PairID        *string      `json:"pairId"`
	Left          *projectFile `json:"left"`
	Right         *projectFile `json:"right"`
	Scope         *scopeFile   `json:"scope"`
}

type projectFile struct {
	ProjectID *string `json:"projectId"`
	Revision  *string `json:"revision"`
}

type scopeFile struct {
	LeftPaths  *[]string        `json:"leftPaths"`
	RightPaths *[]string        `json:"rightPaths"`
	Candidates *[]candidateFile `json:"candidates"`
}

type candidateFile struct {
	CandidateKey *string   `json:"candidateKey"`
	LeftPaths    *[]string `json:"leftPaths"`
	RightPaths   *[]string `json:"rightPaths"`
}

type projectPin struct {
	projectID string
	revision  string
	root      string
	commonDir string
}

type bucket struct {
	key        string
	leftPaths  []string
	rightPaths []string
}

type mapFile struct {
	root     *projectPin
	relative string
	sha256   string
}

// Scan validates the v1 pair/snapshot inputs and emits one deterministic
// projection. It never writes to either snapshot or the optional journal.
func Scan(input Input) (Response, error) {
	pairPath, err := canonicalAbsoluteFile(input.PairPath)
	if err != nil {
		return Response{}, pairInvalid("pair path must name an existing absolute regular file: %v", err)
	}
	pairBytes, err := os.ReadFile(pairPath)
	if err != nil {
		return Response{}, pairInvalid("cannot read pair file: %v", err)
	}
	pair, err := parsePair(pairBytes)
	if err != nil {
		return Response{}, err
	}

	leftRoot, err := canonicalAbsoluteDir(input.LeftRoot)
	if err != nil {
		return Response{}, pairInvalid("left root must name an existing absolute directory: %v", err)
	}
	rightRoot, err := canonicalAbsoluteDir(input.RightRoot)
	if err != nil {
		return Response{}, pairInvalid("right root must name an existing absolute directory: %v", err)
	}
	if leftRoot == rightRoot {
		return Response{}, pairInvalid("left and right roots resolve to the same directory")
	}

	left, err := inspectPin(leftRoot, pair.Left)
	if err != nil {
		return Response{}, err
	}
	right, err := inspectPin(rightRoot, pair.Right)
	if err != nil {
		return Response{}, err
	}

	buckets, err := validateScope(pair)
	if err != nil {
		return Response{}, err
	}
	mapFiles, err := readMaps(input.MapPaths, left, right)
	if err != nil {
		return Response{}, err
	}

	mapObservations := make([]MapObservation, len(mapFiles))
	for i, file := range mapFiles {
		mapObservations[i] = MapObservation{
			ProjectID: file.root.projectID,
			Revision:  file.root.revision,
			Path:      file.relative,
			SHA256:    file.sha256,
		}
	}

	response := Response{
		SchemaVersion: 1,
		Operation:     "reconcile scan",
		OK:            true,
		PairID:        *pair.PairID,
		PairPath:      pairPath,
		PairSHA256:    digest(pairBytes),
		LeftRoot:      leftRoot,
		RightRoot:     rightRoot,
		Snapshots: Snapshots{
			Left:  Snapshot{ProjectID: left.projectID, Revision: left.revision},
			Right: Snapshot{ProjectID: right.projectID, Revision: right.revision},
		},
		Maps:       mapObservations,
		Candidates: make([]Candidate, 0, len(buckets)),
	}
	for _, b := range buckets {
		candidate, err := scanBucket(pair.PairID, b, left, right, mapFiles)
		if err != nil {
			return Response{}, err
		}
		response.Candidates = append(response.Candidates, candidate)
	}
	sort.Slice(response.Candidates, func(i, j int) bool {
		return response.Candidates[i].CandidateID < response.Candidates[j].CandidateID
	})
	if input.Journal != "" {
		journalPath, err := canonicalAbsoluteFile(input.Journal)
		if err != nil {
			return Response{}, evidenceInvalid("journal path must name an existing absolute regular file: %v", err)
		}
		if isWithin(left.root, journalPath) || isWithin(right.root, journalPath) || isWithin(left.commonDir, journalPath) || isWithin(right.commonDir, journalPath) {
			return Response{}, ldserr.New("refusal.reconcile-write-target", "journal path resolves inside a source snapshot or Git common directory")
		}
		journalBytes, err := os.ReadFile(journalPath)
		if err != nil {
			return Response{}, evidenceInvalid("cannot read journal: %v", err)
		}
		journal, err := parseJournal(journalBytes, *pair.PairID, left.projectID, right.projectID)
		if err != nil {
			return Response{}, err
		}
		response.JournalPath = journalPath
		response.Candidates = compareJournal(response.Candidates, journal)
	}
	return response, nil
}

func parsePair(data []byte) (pairFile, error) {
	var pair pairFile
	if err := decodeStrictJSON(data, &pair); err != nil {
		return pairFile{}, pairInvalid("invalid v1 pair JSON: %v", err)
	}
	if pair.SchemaVersion == nil || *pair.SchemaVersion != 1 || pair.PairID == nil || pair.Left == nil || pair.Right == nil || pair.Scope == nil {
		return pairFile{}, pairInvalid("pair file requires schemaVersion 1, pairId, left, right, and scope")
	}
	if !validIdentifier(*pair.PairID) {
		return pairFile{}, pairInvalid("pairId is empty or contains a digest delimiter")
	}
	if err := validateProject("left", pair.Left); err != nil {
		return pairFile{}, err
	}
	if err := validateProject("right", pair.Right); err != nil {
		return pairFile{}, err
	}
	if *pair.Left.ProjectID == *pair.Right.ProjectID {
		return pairFile{}, pairInvalid("left and right projectId values must differ")
	}
	if pair.Scope.LeftPaths == nil || pair.Scope.RightPaths == nil || pair.Scope.Candidates == nil {
		return pairFile{}, pairInvalid("scope requires leftPaths, rightPaths, and candidates arrays")
	}
	return pair, nil
}

func validateProject(side string, project *projectFile) error {
	if project.ProjectID == nil || !validIdentifier(*project.ProjectID) {
		return pairInvalid("%s projectId is empty or contains a digest delimiter", side)
	}
	if project.Revision == nil || !validRevision(*project.Revision) {
		return pairInvalid("%s revision must be a full lowercase Git object ID", side)
	}
	return nil
}

func validateScope(pair pairFile) ([]bucket, error) {
	leftSet, err := validatePathSet(*pair.Scope.LeftPaths, "scope.leftPaths")
	if err != nil {
		return nil, err
	}
	rightSet, err := validatePathSet(*pair.Scope.RightPaths, "scope.rightPaths")
	if err != nil {
		return nil, err
	}
	assignedLeft, assignedRight := map[string]bool{}, map[string]bool{}
	keys := map[string]bool{}
	buckets := make([]bucket, 0, len(*pair.Scope.Candidates)+len(leftSet)+len(rightSet))
	for index, candidate := range *pair.Scope.Candidates {
		if candidate.CandidateKey == nil || !validIdentifier(*candidate.CandidateKey) || strings.HasPrefix(*candidate.CandidateKey, reservedCandidatePrefix) {
			return nil, pairInvalid("scope.candidates[%d].candidateKey is invalid or uses reserved prefix %q", index, reservedCandidatePrefix)
		}
		key := *candidate.CandidateKey
		if keys[key] {
			return nil, pairInvalid("candidateKey %q is duplicated", key)
		}
		keys[key] = true
		if candidate.LeftPaths == nil || candidate.RightPaths == nil {
			return nil, pairInvalid("candidate %q requires leftPaths and rightPaths arrays", key)
		}
		leftPaths, err := validateBucketPaths(*candidate.LeftPaths, leftSet, assignedLeft, key, "left")
		if err != nil {
			return nil, err
		}
		rightPaths, err := validateBucketPaths(*candidate.RightPaths, rightSet, assignedRight, key, "right")
		if err != nil {
			return nil, err
		}
		if len(leftPaths) == 0 && len(rightPaths) == 0 {
			return nil, pairInvalid("candidate %q has no paths on either side", key)
		}
		buckets = append(buckets, bucket{key: key, leftPaths: leftPaths, rightPaths: rightPaths})
	}

	for _, name := range sortedSetDifference(leftSet, assignedLeft) {
		key := "unmapped:left:" + name
		if keys[key] {
			return nil, pairInvalid("candidateKey %q conflicts with a generated unmatched key", key)
		}
		keys[key] = true
		buckets = append(buckets, bucket{key: key, leftPaths: []string{name}})
	}
	for _, name := range sortedSetDifference(rightSet, assignedRight) {
		key := "unmapped:right:" + name
		if keys[key] {
			return nil, pairInvalid("candidateKey %q conflicts with a generated unmatched key", key)
		}
		keys[key] = true
		buckets = append(buckets, bucket{key: key, rightPaths: []string{name}})
	}
	return buckets, nil
}

func validatePathSet(paths []string, name string) (map[string]bool, error) {
	set := make(map[string]bool, len(paths))
	for _, item := range paths {
		if err := validateRepoPath(item); err != nil {
			return nil, pairInvalid("%s contains invalid path %q: %v", name, item, err)
		}
		if set[item] {
			return nil, pairInvalid("%s contains duplicate path %q", name, item)
		}
		set[item] = true
	}
	return set, nil
}

func validateBucketPaths(paths []string, scoped, assigned map[string]bool, key, side string) ([]string, error) {
	seen := map[string]bool{}
	result := make([]string, 0, len(paths))
	for _, item := range paths {
		if err := validateRepoPath(item); err != nil {
			return nil, pairInvalid("candidate %q has invalid %s path %q: %v", key, side, item, err)
		}
		if seen[item] {
			return nil, pairInvalid("candidate %q repeats %s path %q", key, side, item)
		}
		seen[item] = true
		if !scoped[item] {
			return nil, pairInvalid("candidate %q has out-of-scope %s path %q", key, side, item)
		}
		if assigned[item] {
			return nil, pairInvalid("%s path %q belongs to more than one candidate bucket", side, item)
		}
		assigned[item] = true
		result = append(result, item)
	}
	sort.Strings(result)
	return result, nil
}

func scanBucket(pairID *string, b bucket, left, right *projectPin, maps []mapFile) (Candidate, error) {
	evidenceHash := ""
	candidate := Candidate{
		CandidateID:    candidateID(*pairID, b.key),
		CandidateKey:   b.key,
		EvidenceStatus: "not-compared",
		Confidence:     "not-assessed",
		Evidence:       make([]Evidence, 0, len(b.leftPaths)+len(b.rightPaths)+len(maps)),
		Left:           make([]PathRecord, 0, len(b.leftPaths)),
		Right:          make([]PathRecord, 0, len(b.rightPaths)),
	}
	evidence := make([]Evidence, 0, len(b.leftPaths)+len(b.rightPaths)+len(maps))
	for _, item := range b.leftPaths {
		record, row, err := observePath(left, item)
		if err != nil {
			return Candidate{}, err
		}
		candidate.Left = append(candidate.Left, record)
		evidence = append(evidence, row)
	}
	for _, item := range b.rightPaths {
		record, row, err := observePath(right, item)
		if err != nil {
			return Candidate{}, err
		}
		candidate.Right = append(candidate.Right, record)
		evidence = append(evidence, row)
	}
	for _, file := range maps {
		evidence = append(evidence, Evidence{
			Kind:      "artifact",
			ProjectID: file.root.projectID,
			Revision:  file.root.revision,
			Key:       file.relative,
			SHA256:    file.sha256,
		})
	}
	evidence = uniqueEvidence(evidence)
	sort.Slice(evidence, func(i, j int) bool { return evidenceLess(evidence[i], evidence[j]) })
	candidate.Evidence = evidence
	evidenceHash = evidenceDigest(*pairID, left, right, b.key, evidence)
	candidate.EvidenceDigest = &evidenceHash
	return candidate, nil
}

func observePath(project *projectPin, name string) (PathRecord, Evidence, error) {
	tracked, mode, err := trackedPath(project, name)
	if err != nil {
		return PathRecord{}, Evidence{}, err
	}
	if !tracked {
		_, exists, err := safeRepoPath(project.root, name)
		if err != nil {
			return PathRecord{}, Evidence{}, pairInvalid("scope path %q is unsafe: %v", name, err)
		}
		if exists {
			return PathRecord{}, Evidence{}, evidenceInvalid("scope path %q exists but is not tracked at the pinned revision", name)
		}
		row := Evidence{Kind: "missing", ProjectID: project.projectID, Revision: project.revision, Key: name, SHA256: "-"}
		return PathRecord{Path: name, Present: false}, row, nil
	}
	if mode == "160000" {
		return PathRecord{}, Evidence{}, evidenceInvalid("scope path %q is a submodule, not a file", name)
	}
	resolved, exists, err := safeRepoPath(project.root, name)
	if err != nil {
		return PathRecord{}, Evidence{}, pairInvalid("scope path %q is unsafe: %v", name, err)
	}
	if !exists {
		return PathRecord{}, Evidence{}, staleEvidence("tracked scope path %q is missing from the clean checkout", name)
	}
	if resolved != filepath.Join(project.root, filepath.FromSlash(name)) {
		relative, _ := filepath.Rel(project.root, resolved)
		tracked, mode, err := trackedPath(project, filepath.ToSlash(relative))
		if err != nil {
			return PathRecord{}, Evidence{}, err
		}
		if !tracked || (mode != "100644" && mode != "100755") {
			return PathRecord{}, Evidence{}, evidenceInvalid("scope path %q resolves to a target that is not a tracked regular file at the pinned revision", name)
		}
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return PathRecord{}, Evidence{}, internalFailure("cannot read scope path %q: %v", name, err)
	}
	if err := verifyPinnedBytes(project, name, data); err != nil {
		return PathRecord{}, Evidence{}, err
	}
	hash := digest(data)
	row := Evidence{Kind: "artifact", ProjectID: project.projectID, Revision: project.revision, Key: name, SHA256: hash}
	return PathRecord{Path: name, Present: true, SHA256: &hash}, row, nil
}

func readMaps(paths []string, left, right *projectPin) ([]mapFile, error) {
	if len(paths) == 0 {
		return nil, evidenceInvalid("at least one --map file is required")
	}
	seen := map[string]bool{}
	maps := make([]mapFile, 0, len(paths))
	for _, raw := range paths {
		canonical, err := canonicalAbsoluteFile(raw)
		if err != nil {
			return nil, evidenceInvalid("map path must name an existing absolute regular file: %v", err)
		}
		if seen[canonical] {
			return nil, pairInvalid("same canonical map path supplied more than once: %q", canonical)
		}
		seen[canonical] = true
		var project *projectPin
		var relative string
		leftRel, inLeft := relativeWithin(left.root, canonical)
		rightRel, inRight := relativeWithin(right.root, canonical)
		if inLeft == inRight {
			return nil, evidenceInvalid("map %q must belong to exactly one pinned snapshot", canonical)
		}
		if inLeft {
			project, relative = left, filepath.ToSlash(leftRel)
		} else {
			project, relative = right, filepath.ToSlash(rightRel)
		}
		if err := validateRepoPath(relative); err != nil {
			return nil, evidenceInvalid("map %q resolves to invalid repository path: %v", canonical, err)
		}
		tracked, mode, err := trackedPath(project, relative)
		if err != nil {
			return nil, err
		}
		if !tracked || mode == "160000" {
			return nil, evidenceInvalid("map %q is not a tracked regular file in the pinned snapshot", canonical)
		}
		resolved, exists, err := safeRepoPath(project.root, relative)
		if err != nil || !exists {
			return nil, evidenceInvalid("map %q is missing or escapes its snapshot", canonical)
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return nil, internalFailure("cannot read map %q: %v", canonical, err)
		}
		if err := verifyPinnedBytes(project, relative, data); err != nil {
			return nil, err
		}
		maps = append(maps, mapFile{root: project, relative: relative, sha256: digest(data)})
	}
	sort.Slice(maps, func(i, j int) bool {
		return mapObservationLess(maps[i], maps[j])
	})
	return maps, nil
}

func inspectPin(root string, pin *projectFile) (*projectPin, error) {
	gitRoot, err := gitRead(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, pairInvalid("snapshot root %q is not a Git worktree", root)
	}
	canonicalGitRoot, err := filepath.EvalSymlinks(strings.TrimSpace(string(gitRoot)))
	if err != nil || filepath.Clean(canonicalGitRoot) != root {
		return nil, pairInvalid("snapshot root %q must be the worktree root", root)
	}
	commonBytes, err := gitRead(root, "rev-parse", "--git-common-dir")
	if err != nil {
		return nil, pairInvalid("cannot resolve Git common directory for snapshot root %q", root)
	}
	commonDir := strings.TrimSpace(string(commonBytes))
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(root, commonDir)
	}
	commonDir, err = filepath.EvalSymlinks(commonDir)
	if err != nil {
		return nil, pairInvalid("cannot resolve Git common directory for snapshot root %q", root)
	}
	headBytes, err := gitRead(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return nil, staleEvidence("cannot resolve snapshot HEAD: %v", err)
	}
	head := strings.TrimSpace(string(headBytes))
	if head != *pin.Revision {
		return nil, staleEvidence("snapshot %q HEAD does not match pinned revision", *pin.ProjectID)
	}
	config, err := gitRead(root, "config", "--null", "--list")
	if err != nil {
		return nil, staleEvidence("cannot inspect snapshot %q filter configuration", *pin.ProjectID)
	}
	for _, item := range bytes.Split(config, []byte{0}) {
		key, value, _ := bytes.Cut(item, []byte{'\n'})
		name := string(key)
		if strings.HasPrefix(name, "filter.") && (strings.HasSuffix(name, ".clean") || strings.HasSuffix(name, ".process")) && len(value) != 0 {
			return nil, staleEvidence("cannot safely check snapshot %q with configured clean/process filters", *pin.ProjectID)
		}
	}
	index, err := gitRead(root, "ls-files", "-v", "--stage", "-z")
	if err != nil {
		return nil, staleEvidence("cannot inspect snapshot %q index flags", *pin.ProjectID)
	}
	for _, item := range bytes.Split(index, []byte{0}) {
		if len(item) == 0 {
			continue
		}
		if item[0] == 'S' || (item[0] >= 'a' && item[0] <= 'z') {
			return nil, staleEvidence("cannot verify clean snapshot %q with assume-unchanged or skip-worktree index entries", *pin.ProjectID)
		}
		metadata, name, ok := bytes.Cut(item, []byte{'\t'})
		fields := strings.Fields(string(metadata))
		if !ok || len(fields) != 4 {
			return nil, staleEvidence("cannot parse snapshot %q index entries", *pin.ProjectID)
		}
		if fields[1] != "160000" {
			continue
		}
		if validateRepoPath(string(name)) != nil {
			return nil, staleEvidence("snapshot %q has an unsafe submodule path", *pin.ProjectID)
		}
		subRoot := filepath.Join(root, filepath.FromSlash(string(name)))
		if _, err := os.Lstat(filepath.Join(subRoot, ".git")); errors.Is(err, os.ErrNotExist) {
			continue // An uninitialized submodule cannot run a status/filter subprocess.
		} else if err != nil {
			return nil, staleEvidence("cannot inspect snapshot %q submodule", *pin.ProjectID)
		}
		subRoot, err := canonicalAbsoluteDir(subRoot)
		if err != nil || !isWithin(root, subRoot) {
			return nil, staleEvidence("snapshot %q submodule escapes its root", *pin.ProjectID)
		}
		// Parent status recurses into initialized submodules, so establish their
		// pin and filter/index safety before allowing that command to run.
		if _, err := inspectPin(subRoot, &projectFile{ProjectID: pin.ProjectID, Revision: &fields[2]}); err != nil {
			return nil, err
		}
	}
	status, err := gitRead(root, "status", "--porcelain=v1", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return nil, staleEvidence("cannot verify clean snapshot %q: %v", *pin.ProjectID, err)
	}
	if len(status) != 0 {
		return nil, staleEvidence("snapshot %q is not clean", *pin.ProjectID)
	}
	return &projectPin{projectID: *pin.ProjectID, revision: head, root: root, commonDir: filepath.Clean(commonDir)}, nil
}

func trackedPath(project *projectPin, relative string) (bool, string, error) {
	spec := ":(literal)" + relative
	data, err := gitRead(project.root, "ls-tree", "-r", "-z", "--full-tree", project.revision, "--", spec)
	if err != nil {
		return false, "", staleEvidence("cannot inspect tracked path %q: %v", relative, err)
	}
	for _, record := range bytes.Split(data, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		parts := bytes.SplitN(record, []byte{'\t'}, 2)
		if len(parts) != 2 || string(parts[1]) != relative {
			continue
		}
		meta := strings.Fields(string(parts[0]))
		if len(meta) < 2 {
			return false, "", staleEvidence("Git returned malformed tree metadata for %q", relative)
		}
		return true, meta[0], nil
	}
	return false, "", nil
}

// Compare the exact captured bytes against local immutable objects, not Git's
// worktree stat cache. Symlink declarations and every target must also be pinned.
func verifyPinnedBytes(project *projectPin, name string, observed []byte) error {
	seen := map[string]bool{}
	for {
		if seen[name] {
			return evidenceInvalid("scope path %q has a cyclic pinned symlink chain", name)
		}
		seen[name] = true
		tracked, mode, err := trackedPath(project, name)
		if err != nil {
			return err
		}
		if !tracked || (mode != "100644" && mode != "100755" && mode != "120000") {
			return evidenceInvalid("path %q is not a tracked file in the pinned snapshot", name)
		}
		pinned, err := gitRead(project.root, "cat-file", "blob", project.revision+":"+name)
		if err != nil {
			return staleEvidence("cannot read local pinned bytes for %q: %v", name, err)
		}
		if mode != "120000" {
			if !bytes.Equal(observed, pinned) {
				return staleEvidence("observed bytes for %q differ from the pinned snapshot", name)
			}
			return nil
		}
		link, err := os.Readlink(filepath.Join(project.root, filepath.FromSlash(name)))
		if err != nil || link != string(pinned) {
			return staleEvidence("symlink %q differs from its pinned declaration", name)
		}
		target := link
		if !filepath.IsAbs(target) {
			target = filepath.Join(project.root, filepath.FromSlash(path.Dir(name)), target)
		}
		relative, inside := relativeWithin(project.root, filepath.Clean(target))
		if !inside || validateRepoPath(filepath.ToSlash(relative)) != nil {
			return evidenceInvalid("pinned symlink %q has an unsafe target", name)
		}
		name = filepath.ToSlash(relative)
	}
}

func gitRead(root string, args ...string) ([]byte, error) {
	all := []string{"-c", "core.fsmonitor=false", "-c", "core.untrackedCache=false", "-C", root}
	all = append(all, args...)
	cmd := exec.Command("git", all...)
	cmd.Env = gitReadEnvironment()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func gitReadEnvironment() []string {
	var env []string
	for _, item := range os.Environ() {
		if key, _, ok := strings.Cut(item, "="); ok && strings.HasPrefix(key, "GIT_") {
			continue
		}
		env = append(env, item)
	}
	return append(env, "GIT_OPTIONAL_LOCKS=0", "GIT_NO_REPLACE_OBJECTS=1", "GIT_NO_LAZY_FETCH=1")
}

func safeRepoPath(root, relative string) (string, bool, error) {
	current := root
	for _, component := range strings.Split(relative, "/") {
		next := filepath.Join(current, component)
		info, err := os.Lstat(next)
		if errors.Is(err, os.ErrNotExist) {
			return next, false, nil
		}
		if err != nil {
			return "", false, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(next)
			if err != nil {
				return "", false, errors.New("broken or unresolved symlink")
			}
			if !isWithin(root, resolved) {
				return "", false, errors.New("path resolves outside snapshot root")
			}
			current = resolved
			continue
		}
		current = next
		if !isWithin(root, current) {
			return "", false, errors.New("path resolves outside snapshot root")
		}
	}
	info, err := os.Stat(current)
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, errors.New("path is not a regular file")
	}
	return current, true, nil
}

func canonicalAbsoluteFile(value string) (string, error) {
	if !filepath.IsAbs(value) {
		return "", errors.New("path is not absolute")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(value))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("path is not a regular file")
	}
	return filepath.Clean(resolved), nil
}

func canonicalAbsoluteDir(value string) (string, error) {
	if !filepath.IsAbs(value) {
		return "", errors.New("path is not absolute")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(value))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return filepath.Clean(resolved), nil
}

func relativeWithin(root, target string) (string, bool) {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return relative, true
}

func isWithin(root, target string) bool {
	_, ok := relativeWithin(root, target)
	return ok
}

func validateRepoPath(value string) error {
	if value == "" || value == "." || strings.ContainsAny(value, "\\\t\n\r\x00") || strings.HasPrefix(value, "/") || filepath.IsAbs(value) {
		return errors.New("path must be normalized, repository-relative, and contain no control delimiters")
	}
	if len(value) >= 2 && value[1] == ':' && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) {
		return errors.New("path must not be drive-qualified")
	}
	if path.Clean(value) != value || strings.HasPrefix(value, "../") || value == ".." {
		return errors.New("path is not normalized or escapes the repository")
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return errors.New("path contains a control character")
		}
	}
	return nil
}

func validIdentifier(value string) bool {
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\t\n\r\x00") {
		return false
	}
	return true
}

func validRevision(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func candidateID(pairID, key string) string {
	return "cand-v1-" + digest([]byte("lds-reconcile-candidate-v1\n"+pairID+"\n"+key+"\n"))
}

func evidenceDigest(pairID string, left, right *projectPin, key string, evidence []Evidence) string {
	var b strings.Builder
	for _, field := range []string{"lds-reconcile-evidence-v1", pairID, key, left.projectID, left.revision, right.projectID, right.revision} {
		b.WriteString(field)
		b.WriteByte('\n')
	}
	for _, item := range evidence {
		b.WriteString(strings.Join([]string{item.Kind, item.ProjectID, item.Revision, item.Key, item.SHA256}, "\t"))
		b.WriteByte('\n')
	}
	return "sha256:" + digest([]byte(b.String()))
}

func uniqueEvidence(items []Evidence) []Evidence {
	seen := map[Evidence]bool{}
	result := make([]Evidence, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func evidenceLess(a, b Evidence) bool {
	left := []string{a.Kind, a.ProjectID, a.Revision, a.Key, a.SHA256}
	right := []string{b.Kind, b.ProjectID, b.Revision, b.Key, b.SHA256}
	for i := range left {
		if left[i] != right[i] {
			return left[i] < right[i]
		}
	}
	return false
}

func mapObservationLess(a, b mapFile) bool {
	return evidenceLess(
		Evidence{Kind: "artifact", ProjectID: a.root.projectID, Revision: a.root.revision, Key: a.relative, SHA256: a.sha256},
		Evidence{Kind: "artifact", ProjectID: b.root.projectID, Revision: b.root.revision, Key: b.relative, SHA256: b.sha256},
	)
}

func sortedSetDifference(set, assigned map[string]bool) []string {
	var result []string
	for value := range set {
		if !assigned[value] {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func digest(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func decodeStrictJSON(data []byte, target any) error {
	if !utf8.Valid(data) {
		return errors.New("JSON input is not valid UTF-8")
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if err := rejectUnknownJSONFields(value, reflect.TypeOf(target)); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func rejectUnknownJSONFields(value any, targetType reflect.Type) error {
	if value == nil {
		switch targetType.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
			return nil
		default:
			return fmt.Errorf("null is not valid for JSON %s", targetType.Kind())
		}
	}
	for targetType.Kind() == reflect.Pointer {
		targetType = targetType.Elem()
	}
	switch targetType.Kind() {
	case reflect.Struct:
		object, ok := value.(map[string]any)
		if !ok {
			return nil // The typed decoder below reports shape mismatches.
		}
		fields, required := jsonStructFields(targetType)
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fieldType, ok := fields[key]
			if !ok {
				return fmt.Errorf("unknown JSON field %q", key)
			}
			if err := rejectUnknownJSONFields(object[key], fieldType); err != nil {
				return err
			}
		}
		for name := range required {
			if _, ok := object[name]; !ok {
				return fmt.Errorf("missing required JSON field %q", name)
			}
		}
	case reflect.Slice, reflect.Array:
		items, ok := value.([]any)
		if !ok {
			return nil // The typed decoder below reports shape mismatches.
		}
		for _, item := range items {
			if err := rejectUnknownJSONFields(item, targetType.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

func jsonStructFields(targetType reflect.Type) (map[string]reflect.Type, map[string]bool) {
	fields := make(map[string]reflect.Type, targetType.NumField())
	required := make(map[string]bool, targetType.NumField())
	var add func(reflect.Type)
	add = func(current reflect.Type) {
		for index := 0; index < current.NumField(); index++ {
			field := current.Field(index)
			parts := strings.Split(field.Tag.Get("json"), ",")
			name := parts[0]
			if field.Anonymous && name == "" {
				nested := field.Type
				for nested.Kind() == reflect.Pointer {
					nested = nested.Elem()
				}
				if nested.Kind() == reflect.Struct {
					add(nested)
					continue
				}
			}
			if name == "" {
				name = field.Name
			}
			if name == "-" {
				continue
			}
			fields[name] = field.Type
			optional := false
			for _, option := range parts[1:] {
				optional = optional || option == "omitempty"
			}
			if !optional {
				required[name] = true
			}
		}
	}
	add(targetType)
	return fields, required
}

func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
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
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if seen[key] {
				return fmt.Errorf("duplicate object key %q", key)
			}
			seen[key] = true
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

func pairInvalid(format string, args ...any) error {
	return ldserr.New("validation.reconcile-pair-invalid", fmt.Sprintf(format, args...))
}

func evidenceInvalid(format string, args ...any) error {
	return ldserr.New("validation.reconcile-evidence-invalid", fmt.Sprintf(format, args...))
}

func staleEvidence(format string, args ...any) error {
	return ldserr.New("refusal.reconcile-stale-evidence", fmt.Sprintf(format, args...))
}

func internalFailure(format string, args ...any) error {
	return ldserr.New("internal.reconcile-scan-failed", fmt.Sprintf(format, args...))
}
