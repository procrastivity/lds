package reconcile

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"golang.org/x/sys/unix"
)

// JournalPreview is the complete, approval-bound v1 read and proposed write.
// PlanDigest hashes its canonical JSON representation with PlanDigest empty.
type JournalPreview struct {
	SchemaVersion int             `json:"schemaVersion"`
	Operation     string          `json:"operation"`
	JournalPath   string          `json:"journalPath"`
	PairSHA256    string          `json:"pairSha256"`
	JournalExists bool            `json:"journalExists"`
	JournalSHA256 string          `json:"journalSha256"`
	Scan          Response        `json:"scan"`
	Journal       JournalResponse `json:"journal"`
	Request       json.RawMessage `json:"request"`
	PlanDigest    string          `json:"planDigest"`
}

// JournalReview holds immutable source and raw journal baseline evidence. Its
// exported projections and previews are copies, not references to that state.
type JournalReview struct {
	input       Input
	pair        loadedPair
	scan        Response
	scanBytes   []byte
	journalPath canonicalJournalPath
	journal     JournalResponse
	journalRaw  []byte
	exists      bool
	mu          sync.Mutex
	previews    map[string]JournalPreview
}

// PrepareJournalReview captures a fresh scan without consulting Input.Journal.
// It does not create a lock, journal, scratch file, or output.
func PrepareJournalReview(input Input, journalPath string) (*JournalReview, error) {
	input.Journal = ""
	input.MapPaths = append([]string(nil), input.MapPaths...)
	scan, err := Scan(input)
	if err != nil {
		return nil, err
	}
	pair, err := loadPair(input.PairPath)
	if err != nil {
		return nil, err
	}
	if scan.PairPath != pair.path || scan.PairSHA256 != pair.hash {
		return nil, staleEvidence("pair bytes changed during review preparation")
	}
	path, err := resolveJournalPath(journalPath, true)
	if err != nil {
		return nil, err
	}
	protected, err := protectedGitDirs(scan)
	if err != nil {
		return nil, staleEvidence("cannot validate source snapshot boundaries: %v", err)
	}
	if err := ensureWriteBoundary(path.target, path.lock, protected); err != nil {
		return nil, err
	}
	dir, err := openJournalDirectory(filepath.Dir(path.target))
	if err != nil {
		return nil, evidenceInvalid("cannot anchor journal directory: %v", err)
	}
	defer func() { _ = dir.Close() }()
	raw, exists, err := readReviewJournal(dir, path.target)
	if err != nil {
		return nil, err
	}
	state := &journalState{Candidates: make(map[string]*journalCandidate), Events: []json.RawMessage{}}
	if exists {
		state, err = parseJournal(raw, *pair.file.PairID, *pair.file.Left.ProjectID, *pair.file.Right.ProjectID)
		if err != nil {
			return nil, err
		}
	}
	scanBytes, err := json.Marshal(scan)
	if err != nil {
		return nil, evidenceInvalid("cannot encode captured scan: %v", err)
	}
	if _, _, err := validateScanForWrite(scanBytes, pair); err != nil {
		return nil, err
	}
	review := &JournalReview{
		input: input, pair: pair, scan: scan, scanBytes: scanBytes,
		journalPath: path, journal: projectJournal("reconcile journal read", path.target, *pair.file.PairID, snapshotsFromPair(pair.file), state),
		journalRaw: raw, exists: exists, previews: make(map[string]JournalPreview),
	}
	if err := review.checkSource(); err != nil {
		return nil, err
	}
	return review, nil
}

// Scan returns an independent copy of the captured source evidence.
func (r *JournalReview) Scan() Response {
	var result Response
	_ = json.Unmarshal(r.scanBytes, &result)
	return result
}

// Journal returns an independent copy of the captured journal projection.
func (r *JournalReview) Journal() JournalResponse {
	data, _ := json.Marshal(r.journal)
	var result JournalResponse
	_ = json.Unmarshal(data, &result)
	return result
}

// Preview validates the request and replays the proposed event, without I/O
// side effects. An existing same-request event uses the writer's replay rule.
func (r *JournalReview) Preview(operation string, request []byte) (JournalPreview, error) {
	if operation != "record" && operation != "decide" && operation != "reconsider" {
		return JournalPreview{}, evidenceInvalid("unknown journal write operation %q", operation)
	}
	parsed, err := parseWriteRequest(operation, request, r.pair)
	if err != nil {
		return JournalPreview{}, err
	}
	if !r.exists && operation != "record" {
		return JournalPreview{}, evidenceInvalid("journal file does not exist")
	}
	state := &journalState{Candidates: make(map[string]*journalCandidate), Events: []json.RawMessage{}}
	if r.exists {
		state, err = parseJournal(r.journalRaw, *r.pair.file.PairID, *r.pair.file.Left.ProjectID, *r.pair.file.Right.ProjectID)
		if err != nil {
			return JournalPreview{}, err
		}
	}
	if _, err := planJournalWrite(operation, r.pair, r.scan, digest(r.scanBytes), parsed, state, r.journalPath.target); err != nil {
		return JournalPreview{}, err
	}
	// RawMessage is compacted by the JSON encoder used for the displayed
	// preview. Store that same representation so a JSON round trip remains
	// exactly approvable, including requests supplied with whitespace.
	canonicalRequest, err := json.Marshal(json.RawMessage(request))
	if err != nil {
		return JournalPreview{}, evidenceInvalid("cannot encode journal request: %v", err)
	}
	preview := JournalPreview{
		SchemaVersion: 1, Operation: operation, JournalPath: r.journalPath.target,
		PairSHA256: r.pair.hash, JournalExists: r.exists,
		Scan: r.Scan(), Journal: r.Journal(), Request: canonicalRequest,
	}
	if r.exists {
		preview.JournalSHA256 = digest(r.journalRaw)
	}
	preview.PlanDigest, err = journalPlanDigest(preview)
	if err != nil {
		return JournalPreview{}, err
	}
	stored := copyPreview(preview)
	r.mu.Lock()
	r.previews[preview.PlanDigest] = stored
	r.mu.Unlock()
	return preview, nil
}

func journalPlanDigest(preview JournalPreview) (string, error) {
	preview.PlanDigest = ""
	data, err := json.Marshal(preview)
	if err != nil {
		return "", evidenceInvalid("cannot encode preview: %v", err)
	}
	return "sha256:" + digest(data), nil
}

func copyPreview(preview JournalPreview) JournalPreview {
	data, _ := json.Marshal(preview)
	var result JournalPreview
	_ = json.Unmarshal(data, &result)
	return result
}

// Apply requires the exact preview and digest confirmation. It rechecks source
// evidence before locking and immediately before persistence; journal bytes and
// existence are checked under the same flock as the file-based writer.
func (r *JournalReview) Apply(preview JournalPreview, confirmation string) (JournalResponse, error) {
	wantDigest, err := journalPlanDigest(preview)
	if err != nil || confirmation != preview.PlanDigest || preview.PlanDigest != wantDigest {
		return JournalResponse{}, staleEvidence("preview digest or confirmation does not match")
	}
	r.mu.Lock()
	approved, ok := r.previews[preview.PlanDigest]
	r.mu.Unlock()
	if !ok || !reflect.DeepEqual(approved, preview) {
		return JournalResponse{}, staleEvidence("preview differs from the captured approval plan")
	}
	if err := r.checkSource(); err != nil {
		return JournalResponse{}, err
	}
	parsed, err := parseWriteRequest(preview.Operation, preview.Request, r.pair)
	if err != nil {
		return JournalResponse{}, err
	}
	path, err := resolveJournalPath(r.journalPath.input, preview.Operation == "record")
	if err != nil || path.target != r.journalPath.target {
		return JournalResponse{}, staleEvidence("journal path changed after preview")
	}
	return writeJournalLocked(preview.Operation, r.pair, r.scan, digest(r.scanBytes), parsed, path, r)
}

func (r *JournalReview) checkSource() error {
	pair, err := loadPair(r.input.PairPath)
	if err != nil || pair.path != r.pair.path || pair.hash != r.pair.hash {
		return staleEvidence("pair path or bytes changed after review preparation")
	}
	scan, err := Scan(r.input)
	if err != nil {
		return staleEvidence("source scan is no longer clean or pinned: %v", err)
	}
	data, err := json.Marshal(scan)
	if err != nil || !bytes.Equal(data, r.scanBytes) {
		return staleEvidence("source roots, maps, or scan evidence changed after review preparation")
	}
	return nil
}

func (r *JournalReview) checkJournalBaseline(dir *os.File) error {
	data, exists, err := readReviewJournal(dir, r.journalPath.target)
	if err != nil {
		return err
	}
	if exists != r.exists || !bytes.Equal(data, r.journalRaw) {
		return staleEvidence("journal existence or raw bytes changed after review preparation")
	}
	return nil
}

func readReviewJournal(dir *os.File, target string) ([]byte, bool, error) {
	file, err := openJournalChild(dir, filepath.Base(target), unix.O_RDONLY, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, evidenceInvalid("cannot open journal baseline: %v", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	pathInfo, pathErr := os.Lstat(target)
	if err != nil || pathErr != nil || !info.Mode().IsRegular() || !pathInfo.Mode().IsRegular() || !os.SameFile(info, pathInfo) {
		return nil, false, staleEvidence("journal baseline path is not a stable regular file")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, false, evidenceInvalid("cannot read journal baseline: %v", err)
	}
	return data, true, nil
}
