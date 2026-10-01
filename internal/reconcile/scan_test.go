package reconcile

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/procrastivity/lds/internal/ldserr"
)

func TestCandidateIDIndependentVectors(t *testing.T) {
	cases := []struct {
		pairID string
		key    string
		want   string
	}{
		{"fixture:lds-bds252:scind-xcind:v1", "topic:targeting", "cand-v1-4d5c8ebec9a4811936cda396f8f91f3f9dca37bd1ce96f85ce398d878d433bae"},
		{"fixture:lds-bds252:scind-xcind:v1", "topic:external-proxy", "cand-v1-29cfd6874814f7b3001b8babffc4705eb7d2fec4d4fcb789ba10a22baa1d894b"},
		{"fixture:lds-bds252:scind-xcind:v1", "topic:tcp-sni", "cand-v1-17b084d3d73c93a47b07d60b1934563196e12150d410b0b8ff7d13c5de9c486c"},
		{"fixture:lds-bds252:scind-xcind:v1", "unmapped:right:engineering/decisions/0021-cross-repo-adr-cross-referencing.md", "cand-v1-6de68402c8df9a0f3f211586b6ec7debef5ee63a121e640ae20a053d8a1d9304"},
		{"fixture:lds-bds252:synthetic-split:v1", "fixture:route-decomposition", "cand-v1-aed9fd6fc71e2ccff638d2e91a338b80ba6761a929d4b44a67dd26d242d81af6"},
	}
	for _, test := range cases {
		if got := candidateID(test.pairID, test.key); got != test.want {
			t.Errorf("candidateID(%q, %q) = %q, want %q", test.pairID, test.key, got, test.want)
		}
	}
}

func TestEvidenceDigestIndependentStage2Vectors(t *testing.T) {
	const (
		pairID       = "fixture:lds-bds252:scind-xcind:v1"
		leftProject  = "scind"
		leftRevision = "c63cbc686afd2ca6ecc01eab7825d637d49378c7"
		rightProject = "xcind"
		rightRev     = "c244856e7119547ed6dceb2b7663a01720bdde66"
		mapJSON      = "engineering/sync/artifacts/correspondence-map.json"
		mapMD        = "engineering/sync/artifacts/correspondence-map.md"
	)
	left := &projectPin{projectID: leftProject, revision: leftRevision}
	right := &projectPin{projectID: rightProject, revision: rightRev}
	maps := []Evidence{
		{Kind: "artifact", ProjectID: rightProject, Revision: rightRev, Key: mapJSON, SHA256: "175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17"},
		{Kind: "artifact", ProjectID: rightProject, Revision: rightRev, Key: mapMD, SHA256: "3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05"},
	}
	rows := func(extra ...Evidence) []Evidence {
		result := append(append([]Evidence(nil), extra...), maps...)
		sort.Slice(result, func(i, j int) bool { return evidenceLess(result[i], result[j]) })
		return result
	}
	cases := []struct {
		key  string
		rows []Evidence
		want string
	}{
		{
			key: "topic:targeting",
			rows: rows(
				Evidence{Kind: "artifact", ProjectID: leftProject, Revision: leftRevision, Key: "engineering/decisions/0011-options-based-targeting.md", SHA256: "92416f10e61b7afb891eac7cc6b1e8a9e8960b635d842b059cc9a86eef48bb73"},
				Evidence{Kind: "artifact", ProjectID: rightProject, Revision: rightRev, Key: "engineering/decisions/0023-location-based-targeting.md", SHA256: "88d2174b069096a01df7d0476302e42fa3ce6511352b3df8b592c27af794448d"},
			),
			want: "sha256:2765ed93a8a5078ecc60cb983fde4d26b67ac95d04ba0fbd4bacaa635997f349",
		},
		{
			key: "topic:external-proxy",
			rows: rows(
				Evidence{Kind: "artifact", ProjectID: leftProject, Revision: leftRevision, Key: "engineering/decisions/0017-external-proxy-mode.md", SHA256: "9557cbfde3718cc23918e20effb76498eb788b1630e93396b49c00fddad1b7cc"},
				Evidence{Kind: "artifact", ProjectID: rightProject, Revision: rightRev, Key: "engineering/decisions/0022-external-proxy-mode.md", SHA256: "db5d802fb20b687829d46e5438334f00c33b3821ca4cb7cff3c3ef2d51377723"},
			),
			want: "sha256:5775c620170e2a16c56cd306b1d21f742cf39369d028fed23d59e8075d3b2236",
		},
		{
			key: "topic:tcp-sni",
			rows: rows(
				Evidence{Kind: "artifact", ProjectID: leftProject, Revision: leftRevision, Key: "engineering/specs/port-types.md", SHA256: "9f002cb97b8fcb530a488b34a258e9994b41c60ae62b455e135f35d60622ded3"},
				Evidence{Kind: "artifact", ProjectID: rightProject, Revision: rightRev, Key: "engineering/sync/divergence/0030-port-type-plugins-tcp-sni.md", SHA256: "0fac817abd1cb0a18f81746da10f7039cb2fa6438c5684b711d3924e36aefa8c"},
			),
			want: "sha256:b299310a771287426152781b60c52648002022080442696c0f03241ef390a168",
		},
		{
			key:  "unmapped:right:engineering/decisions/0021-cross-repo-adr-cross-referencing.md",
			rows: rows(Evidence{Kind: "artifact", ProjectID: rightProject, Revision: rightRev, Key: "engineering/decisions/0021-cross-repo-adr-cross-referencing.md", SHA256: "619d180ff6eeba7d8c6924f11850d250d019f19116bfe05aba9cc485c61b482a"}),
			want: "sha256:21bd6eefd00a2033ac324832cef6ab5893c9fd6afd01c3cbde2d3fd8abd03f96",
		},
	}
	for _, test := range cases {
		if got := evidenceDigest(pairID, left, right, test.key, test.rows); got != test.want {
			t.Errorf("evidenceDigest(%q) = %s, want %s", test.key, got, test.want)
		}
	}
}

func TestStrictPairJSONAndScopeValidation(t *testing.T) {
	valid := `{"schemaVersion":1,"pairId":"fixture:test:v1","left":{"projectId":"left","revision":"0123456789012345678901234567890123456789"},"right":{"projectId":"right","revision":"abcdefabcdefabcdefabcdefabcdefabcdefabcd"},"scope":{"leftPaths":["a.md"],"rightPaths":["b.md"],"candidates":[{"candidateKey":"topic:x","leftPaths":["a.md"],"rightPaths":["b.md"]}]}}`
	if _, err := parsePair([]byte(valid)); err != nil {
		t.Fatalf("valid pair rejected: %v", err)
	}
	cases := []struct {
		name string
		json string
	}{
		{"missing schema version", strings.Replace(valid, `"schemaVersion":1,`, "", 1)},
		{"wrong schema version", strings.Replace(valid, `"schemaVersion":1`, `"schemaVersion":2`, 1)},
		{"unknown top-level field", strings.Replace(valid, `"schemaVersion":1,`, `"schemaVersion":1,"unexpected":true,`, 1)},
		{"case-mismatched field", strings.Replace(valid, `"schemaVersion":1,`, `"schemaVersion":1,"SchemaVersion":1,`, 1)},
		{"duplicate nested key", strings.Replace(valid, `"projectId":"left"`, `"projectId":"left","projectId":"other"`, 1)},
		{"unknown bucket field", strings.Replace(valid, `"candidateKey":"topic:x"`, `"candidateKey":"topic:x","member":true`, 1)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parsePair([]byte(test.json)); err == nil {
				t.Fatal("invalid pair was accepted")
			}
		})
	}

	base, err := parsePair([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name string
		edit func(*pairFile)
	}{
		{"reserved key", func(pair *pairFile) { *(*pair.Scope.Candidates)[0].CandidateKey = "unmapped:user" }},
		{"overlap", func(pair *pairFile) {
			*pair.Scope.Candidates = append(*pair.Scope.Candidates, candidateFile{
				CandidateKey: stringPtr("topic:y"),
				LeftPaths:    stringSlicePtr([]string{"a.md"}),
				RightPaths:   stringSlicePtr([]string{"b.md"}),
			})
		}},
		{"duplicate scoped path", func(pair *pairFile) { *pair.Scope.LeftPaths = append(*pair.Scope.LeftPaths, "a.md") }},
		{"normalized path", func(pair *pairFile) { (*pair.Scope.LeftPaths)[0] = "folder/../a.md" }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			pairCopy := clonePair(t, base)
			mutation.edit(&pairCopy)
			if _, err := validateScope(pairCopy); err == nil {
				t.Fatal("invalid scope was accepted")
			}
		})
	}
}

func TestScanPinnedGitTreesIsDeterministicReadOnlyAndTracksChanges(t *testing.T) {
	base := t.TempDir()
	leftRoot := filepath.Join(base, "left")
	rightRoot := filepath.Join(base, "right")
	leftRevision := initGitFixture(t, leftRoot, map[string][]byte{
		"src/left.md": []byte("left source\n"),
	})
	rightRevision := initGitFixture(t, rightRoot, map[string][]byte{
		"src/right.md": []byte("right source\n"),
		"maps/a.json":  []byte("{\"a\":1}\n"),
		"maps/b.md":    []byte("raw map b\n"),
	})
	pairPath := filepath.Join(base, "pair.json")
	writePair(t, pairPath, "fixture:test:scan:v1", leftRevision, rightRevision,
		[]string{"src/left.md", "src/missing.md"},
		[]string{"src/right.md", "src/unmatched.md", "maps/a.json"},
		[]candidateInput{{"topic:scan", []string{"src/left.md", "src/missing.md"}, []string{"src/right.md", "maps/a.json"}}},
	)
	maps := []string{filepath.Join(rightRoot, "maps", "a.json"), filepath.Join(rightRoot, "maps", "b.md")}
	input := Input{PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot, MapPaths: maps}
	before := snapshotFixtureFiles(t, leftRoot, rightRoot)
	first, err := Scan(input)
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	reversed := input
	reversed.MapPaths = []string{maps[1], maps[0]}
	second, err := Scan(reversed)
	if err != nil {
		t.Fatalf("reversed map scan: %v", err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("map argument order changed scan output:\n%s\n%s", firstJSON, secondJSON)
	}
	if got, want := len(first.Candidates), 2; got != want {
		t.Fatalf("candidate count=%d, want %d", got, want)
	}
	byKey := candidateMap(first.Candidates)
	main := byKey["topic:scan"]
	if main.CandidateID != candidateID("fixture:test:scan:v1", "topic:scan") || main.EvidenceStatus != "not-compared" {
		t.Fatalf("main candidate identity/status = %q/%q", main.CandidateID, main.EvidenceStatus)
	}
	if len(main.Evidence) != 5 {
		t.Fatalf("main evidence rows=%d, want 5 (including maps once and missing path)", len(main.Evidence))
	}
	var hasMissing, mapAOnce bool
	for _, item := range main.Evidence {
		if item.Kind == "missing" && item.Key == "src/missing.md" && item.SHA256 == "-" {
			hasMissing = true
		}
		if item.Key == "maps/a.json" {
			mapAOnce = true
		}
	}
	if !hasMissing || !mapAOnce {
		t.Fatalf("missing or scoped map observation absent: %+v", main.Evidence)
	}
	unmatched := byKey["unmapped:right:src/unmatched.md"]
	if unmatched.CandidateID != candidateID("fixture:test:scan:v1", "unmapped:right:src/unmatched.md") || len(unmatched.Right) != 1 || unmatched.Right[0].Present {
		t.Fatalf("unmatched missing path projection = %+v", unmatched)
	}
	if len(first.Maps) != 2 || first.Maps[0].Path != "maps/a.json" || first.Maps[1].Path != "maps/b.md" {
		t.Fatalf("maps not stably sorted: %+v", first.Maps)
	}
	mapSelection := input
	mapSelection.MapPaths = []string{maps[0]}
	selectedMap, err := Scan(mapSelection)
	if err != nil {
		t.Fatalf("single-map scan: %v", err)
	}
	for _, candidate := range selectedMap.Candidates {
		old := byKey[candidate.CandidateKey]
		if candidate.CandidateID != old.CandidateID || candidate.EvidenceDigest == old.EvidenceDigest {
			t.Errorf("map selection did not change evidence independently of pinned revisions: before=%+v after=%+v", old, candidate)
		}
	}
	after := snapshotFixtureFiles(t, leftRoot, rightRoot)
	if !equalByteMaps(before, after) {
		t.Fatal("scan changed snapshot source bytes or Git index metadata")
	}

	appendFixtureAndCommit(t, leftRoot, "src/left.md", []byte("left source changed\n"))
	newLeftRevision := gitFixtureOutput(t, leftRoot, "rev-parse", "HEAD")
	writePair(t, pairPath, "fixture:test:scan:v1", newLeftRevision, rightRevision,
		[]string{"src/left.md", "src/missing.md"},
		[]string{"src/right.md", "src/unmatched.md", "maps/a.json"},
		[]candidateInput{{"topic:scan", []string{"src/left.md", "src/missing.md"}, []string{"src/right.md", "maps/a.json"}}},
	)
	changedSource, err := Scan(input)
	if err != nil {
		t.Fatalf("changed-source scan: %v", err)
	}
	if got := candidateMap(changedSource.Candidates)["topic:scan"]; got.CandidateID != main.CandidateID || digestValue(got.EvidenceDigest) == digestValue(main.EvidenceDigest) {
		t.Fatalf("source change renamed candidate or kept evidence digest: before=%+v after=%+v", main, got)
	}

	appendFixtureAndCommit(t, rightRoot, "maps/a.json", []byte("{\"a\":2}\n"))
	newRightRevision := gitFixtureOutput(t, rightRoot, "rev-parse", "HEAD")
	writePair(t, pairPath, "fixture:test:scan:v1", newLeftRevision, newRightRevision,
		[]string{"src/left.md", "src/missing.md"},
		[]string{"src/right.md", "src/unmatched.md", "maps/a.json"},
		[]candidateInput{{"topic:scan", []string{"src/left.md", "src/missing.md"}, []string{"src/right.md", "maps/a.json"}}},
	)
	changedMap, err := Scan(input)
	if err != nil {
		t.Fatalf("changed-map scan: %v", err)
	}
	for _, candidate := range changedMap.Candidates {
		old := byKey[candidate.CandidateKey]
		if candidate.CandidateID != old.CandidateID || digestValue(candidate.EvidenceDigest) == digestValue(old.EvidenceDigest) {
			t.Errorf("map/revision change altered identity or failed to alter digest: before=%+v after=%+v", old, candidate)
		}
	}
	if changedMap.Maps[0].SHA256 == first.Maps[0].SHA256 {
		t.Fatal("changed map retained old byte hash")
	}
}

func TestScanRejectsDuplicateMapAliasesAndReversedRoots(t *testing.T) {
	base := t.TempDir()
	leftRoot := filepath.Join(base, "left")
	rightRoot := filepath.Join(base, "right")
	leftRevision := initGitFixture(t, leftRoot, map[string][]byte{"left.txt": []byte("left\n")})
	if err := os.MkdirAll(rightRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rightRoot, "right.txt"), []byte("right\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rightRoot, "map.txt"), []byte("map\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("map.txt", filepath.Join(rightRoot, "map-link.txt")); err != nil {
		t.Fatal(err)
	}
	rightRevision := initExistingGitFixture(t, rightRoot)
	pairPath := filepath.Join(base, "pair.json")
	writePair(t, pairPath, "fixture:test:alias:v1", leftRevision, rightRevision,
		[]string{"left.txt"}, []string{"right.txt"},
		[]candidateInput{{"topic:alias", []string{"left.txt"}, []string{"right.txt"}}},
	)
	mapPath := filepath.Join(rightRoot, "map.txt")
	_, err := Scan(Input{PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot, MapPaths: []string{mapPath, filepath.Join(rightRoot, "map-link.txt")}})
	assertCode(t, err, "validation.reconcile-pair-invalid")
	_, err = Scan(Input{PairPath: pairPath, LeftRoot: rightRoot, RightRoot: leftRoot, MapPaths: []string{mapPath}})
	assertCode(t, err, "refusal.reconcile-stale-evidence")
}

func TestScanRejectsSymlinkEscapeAndDirtySnapshot(t *testing.T) {
	base := t.TempDir()
	leftRoot := filepath.Join(base, "left")
	rightRoot := filepath.Join(base, "right")
	outside := filepath.Join(base, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(leftRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(leftRoot, "escape.txt")); err != nil {
		t.Fatal(err)
	}
	leftRevision := initExistingGitFixture(t, leftRoot)
	rightRevision := initGitFixture(t, rightRoot, map[string][]byte{"right.txt": []byte("right\n"), "map.txt": []byte("map\n")})
	pairPath := filepath.Join(base, "pair.json")
	writePair(t, pairPath, "fixture:test:safety:v1", leftRevision, rightRevision,
		[]string{"escape.txt"}, []string{"right.txt"},
		[]candidateInput{{"topic:safety", []string{"escape.txt"}, []string{"right.txt"}}},
	)
	_, err := Scan(Input{PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot, MapPaths: []string{filepath.Join(rightRoot, "map.txt")}})
	assertCode(t, err, "validation.reconcile-pair-invalid")

	if err := os.WriteFile(filepath.Join(rightRoot, "right.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Scan(Input{PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot, MapPaths: []string{filepath.Join(rightRoot, "map.txt")}})
	assertCode(t, err, "refusal.reconcile-stale-evidence")
}

func TestScanRequiresExplicitAbsolutePaths(t *testing.T) {
	_, err := Scan(Input{PairPath: "relative.json", LeftRoot: "/left", RightRoot: "/right", MapPaths: []string{"/map"}})
	assertCode(t, err, "validation.reconcile-pair-invalid")
}

func TestPinnedBytesIgnoreReplacementsAndValidateLinkDeclarations(t *testing.T) {
	root := filepath.Join(t.TempDir(), "snapshot")
	initGitFixture(t, root, map[string][]byte{
		"target.md": []byte("pinned artifact\n"), "other.md": []byte("pinned artifact\n"), "replacement.md": []byte("spoof! artifact\n"),
	})
	if err := os.Symlink("middle.md", filepath.Join(root, "link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.md", filepath.Join(root, "middle.md")); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "link.md", "middle.md")
	runFixtureGit(t, root, "commit", "-q", "-m", "fixture link chain")
	revision := gitFixtureOutput(t, root, "rev-parse", "HEAD")
	project := &projectPin{root: root, projectID: "fixture", revision: revision}
	original := gitFixtureOutput(t, root, "rev-parse", revision+":target.md")
	replacement := gitFixtureOutput(t, root, "rev-parse", revision+":replacement.md")
	runFixtureGit(t, root, "replace", original, replacement)
	if err := verifyPinnedBytes(project, "link.md", []byte("pinned artifact\n")); err != nil {
		t.Fatalf("raw pinned bytes must ignore object replacements: %v", err)
	}
	assertCode(t, verifyPinnedBytes(project, "link.md", []byte("spoof! artifact\n")), "refusal.reconcile-stale-evidence")
	if err := os.Remove(filepath.Join(root, "middle.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("other.md", filepath.Join(root, "middle.md")); err != nil {
		t.Fatal(err)
	}
	assertCode(t, verifyPinnedBytes(project, "link.md", []byte("pinned artifact\n")), "refusal.reconcile-stale-evidence")
}

func TestScanRejectsStatCacheSpoofedPinnedBytes(t *testing.T) {
	for _, name := range []string{"artifact.md", "map.md", "target.md"} {
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			leftRoot, rightRoot := filepath.Join(base, "left"), filepath.Join(base, "right")
			leftRevision := initGitFixture(t, leftRoot, map[string][]byte{"artifact.md": []byte("left artifact\n")})
			initGitFixture(t, rightRoot, map[string][]byte{
				"artifact.md": []byte("pinned artifact\n"), "map.md": []byte("pinned artifact\n"), "target.md": []byte("pinned artifact\n"),
			})
			if err := os.Symlink("target.md", filepath.Join(rightRoot, "link.md")); err != nil {
				t.Fatal(err)
			}
			runFixtureGit(t, rightRoot, "add", "link.md")
			runFixtureGit(t, rightRoot, "commit", "-q", "-m", "fixture link")
			rightRevision := gitFixtureOutput(t, rightRoot, "rev-parse", "HEAD")
			pairPath := filepath.Join(base, "pair.json")
			writePair(t, pairPath, "fixture:stat-cache:v1", leftRevision, rightRevision,
				[]string{"artifact.md"}, []string{"artifact.md", "link.md"},
				[]candidateInput{{"topic:stat-cache", []string{"artifact.md"}, []string{"artifact.md", "link.md"}}},
			)
			artifact := filepath.Join(rightRoot, name)
			modified := time.Unix(946684800, 0)
			if err := os.Chtimes(artifact, modified, modified); err != nil {
				t.Fatal(err)
			}
			runFixtureGit(t, rightRoot, "update-index", "--refresh")
			runFixtureGit(t, rightRoot, "config", "core.trustctime", "false")
			input := Input{PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot, MapPaths: []string{filepath.Join(rightRoot, "map.md")}}
			initial, err := Scan(input)
			if err != nil {
				t.Fatalf("unchanged pinned bytes with trustctime=false: %v", err)
			}
			if got := initial.Maps[0].SHA256; got != "3bb3644b0e440d678b7911a69240bb50659505e6173d0ffa3cfe612a7fe2c2da" {
				t.Fatalf("pinned map byte hash: %s", got)
			}
			if err := os.WriteFile(artifact, []byte("spoof! artifact\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(artifact, modified, modified); err != nil {
				t.Fatal(err)
			}
			if status := gitFixtureOutput(t, rightRoot, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
				t.Fatalf("stat-cache spoof must evade status: %q", status)
			}
			if flags := gitFixtureOutput(t, rightRoot, "ls-files", "-v", "--", name); flags != "H "+name {
				t.Fatalf("no assume-unchanged/skip-worktree flag allowed: %q", flags)
			}
			before := snapshotFixtureFiles(t, leftRoot, rightRoot)
			_, err = Scan(input)
			assertCode(t, err, "refusal.reconcile-stale-evidence")
			if !equalByteMaps(before, snapshotFixtureFiles(t, leftRoot, rightRoot)) {
				t.Fatal("refused scan changed snapshot bytes")
			}
		})
	}
}

func TestScanRejectsIndexFlagsThatHideDirtyEvidence(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		for _, name := range []string{"artifact.md", "map.md"} {
			t.Run(flag+"/"+name, func(t *testing.T) {
				base := t.TempDir()
				leftRoot, rightRoot := filepath.Join(base, "left"), filepath.Join(base, "right")
				leftRevision := initGitFixture(t, leftRoot, map[string][]byte{"artifact.md": []byte("left pinned\n")})
				rightRevision := initGitFixture(t, rightRoot, map[string][]byte{
					"artifact.md": []byte("right pinned\n"), "map.md": []byte("map pinned\n"),
				})
				pairPath := filepath.Join(base, "pair.json")
				writePair(t, pairPath, "fixture:hidden-dirty:v1", leftRevision, rightRevision,
					[]string{"artifact.md"}, []string{"artifact.md"},
					[]candidateInput{{"topic:hidden-dirty", []string{"artifact.md"}, []string{"artifact.md"}}},
				)
				input := Input{PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot, MapPaths: []string{filepath.Join(rightRoot, "map.md")}}
				if _, err := Scan(input); err != nil {
					t.Fatalf("ordinary clean fixture: %v", err)
				}
				runFixtureGit(t, rightRoot, "update-index", flag, name)
				if err := os.WriteFile(filepath.Join(rightRoot, name), []byte("replacement unpinned bytes\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if status := gitFixtureOutput(t, rightRoot, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
					t.Fatalf("hidden modification must evade status: %q", status)
				}
				before := snapshotFixtureFiles(t, leftRoot, rightRoot)
				_, err := Scan(input)
				assertCode(t, err, "refusal.reconcile-stale-evidence")
				if !equalByteMaps(before, snapshotFixtureFiles(t, leftRoot, rightRoot)) {
					t.Fatal("refused scan changed snapshot bytes")
				}
			})
		}
	}
}

func TestScanRefusesConfiguredFiltersWithoutExecutingThem(t *testing.T) {
	for _, filter := range []string{"clean", "process"} {
		t.Run(filter, func(t *testing.T) {
			base := t.TempDir()
			leftRoot, rightRoot := filepath.Join(base, "left"), filepath.Join(base, "right")
			leftRevision := initGitFixture(t, leftRoot, map[string][]byte{
				"artifact.md":    []byte("left pinned\n"),
				".gitattributes": []byte("artifact.md filter=review-marker\n"),
				".gitignore":     []byte("filter.marker\n"),
			})
			rightRevision := initGitFixture(t, rightRoot, map[string][]byte{"map.md": []byte("map pinned\n")})
			pairPath := filepath.Join(base, "pair.json")
			writePair(t, pairPath, "fixture:filter:v1", leftRevision, rightRevision,
				[]string{"artifact.md"}, []string{},
				[]candidateInput{{"topic:filter", []string{"artifact.md"}, []string{}}},
			)
			command := "printf 'scanner invoked filter\\n' > filter.marker; cat"
			if filter == "process" {
				command = "printf 'scanner invoked filter\\n' > filter.marker; exit 1"
			}
			runFixtureGit(t, leftRoot, "config", "filter.review-marker."+filter, command)
			artifact := filepath.Join(leftRoot, "artifact.md")
			if err := os.WriteFile(artifact, []byte("left pinned\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			modified := time.Unix(946684800, 0)
			if err := os.Chtimes(artifact, modified, modified); err != nil {
				t.Fatal(err)
			}
			before := snapshotFixtureFiles(t, leftRoot, rightRoot)
			_, err := Scan(Input{PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot, MapPaths: []string{filepath.Join(rightRoot, "map.md")}})
			if _, markerErr := os.Stat(filepath.Join(leftRoot, "filter.marker")); !os.IsNotExist(markerErr) {
				t.Fatalf("scan executed configured %s filter: %v", filter, markerErr)
			}
			assertCode(t, err, "refusal.reconcile-stale-evidence")
			if !equalByteMaps(before, snapshotFixtureFiles(t, leftRoot, rightRoot)) {
				t.Fatal("refused scan changed snapshot bytes")
			}
		})
	}
}

func TestScanChecksSubmoduleFiltersBeforeParentStatus(t *testing.T) {
	base := t.TempDir()
	leftRoot, rightRoot := filepath.Join(base, "left"), filepath.Join(base, "right")
	childRoot := filepath.Join(base, "child")
	initGitFixture(t, childRoot, map[string][]byte{
		"artifact.md": []byte("child pinned\n"), ".gitattributes": []byte("artifact.md filter=review-marker\n"), ".gitignore": []byte("filter.marker\n"),
	})
	initGitFixture(t, leftRoot, map[string][]byte{"artifact.md": []byte("left pinned\n")})
	runFixtureGit(t, leftRoot, "-c", "protocol.file.allow=always", "submodule", "add", "-q", childRoot, "nested")
	runFixtureGit(t, leftRoot, "commit", "-q", "-m", "fixture submodule")
	leftRevision := gitFixtureOutput(t, leftRoot, "rev-parse", "HEAD")
	rightRevision := initGitFixture(t, rightRoot, map[string][]byte{"map.md": []byte("map pinned\n")})
	pairPath := filepath.Join(base, "pair.json")
	writePair(t, pairPath, "fixture:submodule-filter:v1", leftRevision, rightRevision,
		[]string{"artifact.md"}, []string{}, []candidateInput{{"topic:filter", []string{"artifact.md"}, []string{}}},
	)
	input := Input{PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot, MapPaths: []string{filepath.Join(rightRoot, "map.md")}}
	if _, err := Scan(input); err != nil {
		t.Fatalf("clean submodule must remain supported: %v", err)
	}
	nested := filepath.Join(leftRoot, "nested")
	runFixtureGit(t, nested, "config", "filter.review-marker.clean", "printf 'scanner invoked filter\\n' > filter.marker; cat")
	modified := time.Unix(946684800, 0)
	if err := os.Chtimes(filepath.Join(nested, "artifact.md"), modified, modified); err != nil {
		t.Fatal(err)
	}
	before := snapshotFixtureFiles(t, leftRoot, rightRoot)
	_, err := Scan(input)
	if _, markerErr := os.Stat(filepath.Join(nested, "filter.marker")); !os.IsNotExist(markerErr) {
		t.Fatalf("parent status executed submodule filter: %v", markerErr)
	}
	assertCode(t, err, "refusal.reconcile-stale-evidence")
	if !equalByteMaps(before, snapshotFixtureFiles(t, leftRoot, rightRoot)) {
		t.Fatal("refused scan changed snapshot bytes")
	}
}

func TestScanSymlinkTargetsMustBePinnedFiles(t *testing.T) {
	for _, target := range []string{"tracked.txt", "ignored.txt", ".git/HEAD"} {
		t.Run(target, func(t *testing.T) {
			base := t.TempDir()
			leftRoot := filepath.Join(base, "left")
			rightRoot := filepath.Join(base, "right")
			initGitFixture(t, leftRoot, map[string][]byte{
				"tracked.txt": []byte("pinned target\n"),
				".gitignore":  []byte("ignored.txt\n"),
			})
			if err := os.Symlink(target, filepath.Join(leftRoot, "link.txt")); err != nil {
				t.Fatal(err)
			}
			runFixtureGit(t, leftRoot, "add", "link.txt")
			runFixtureGit(t, leftRoot, "commit", "-q", "-m", "tracked link")
			leftRevision := gitFixtureOutput(t, leftRoot, "rev-parse", "HEAD")
			if err := os.WriteFile(filepath.Join(leftRoot, "ignored.txt"), []byte("unpinned target\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if status := gitFixtureOutput(t, leftRoot, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
				t.Fatalf("fixture must be clean: %q", status)
			}
			rightRevision := initGitFixture(t, rightRoot, map[string][]byte{"map.txt": []byte("map\n")})
			pairPath := filepath.Join(base, "pair.json")
			writePair(t, pairPath, "fixture:test:symlink:v1", leftRevision, rightRevision,
				[]string{"link.txt"}, []string{},
				[]candidateInput{{"topic:symlink", []string{"link.txt"}, []string{}}},
			)
			result, err := Scan(Input{PairPath: pairPath, LeftRoot: leftRoot, RightRoot: rightRoot, MapPaths: []string{filepath.Join(rightRoot, "map.txt")}})
			if target != "tracked.txt" {
				assertCode(t, err, "validation.reconcile-evidence-invalid")
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := result.Candidates[0].Left[0]; got.Path != "link.txt" || !got.Present || digestValue(got.SHA256) != "47bc66de662c6ab9b94d9a0e182370a0c4add81ad58036104b058ba52d65a93f" {
				t.Fatalf("safe tracked target projection: %+v", got)
			}
		})
	}
}

type candidateInput struct {
	key         string
	left, right []string
}

func initGitFixture(t *testing.T, root string, files map[string][]byte) string {
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
	return initExistingGitFixture(t, root)
}

func initExistingGitFixture(t *testing.T, root string) string {
	t.Helper()
	runFixtureGit(t, root, "init", "-q")
	runFixtureGit(t, root, "add", "-A")
	runFixtureGit(t, root, "commit", "-q", "-m", "fixture baseline")
	return gitFixtureOutput(t, root, "rev-parse", "HEAD")
}

func appendFixtureAndCommit(t *testing.T, root, name string, content []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.WriteFile(full, content, 0o644); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "--", name)
	runFixtureGit(t, root, "commit", "-q", "-m", "fixture change")
}

func runFixtureGit(t *testing.T, root string, args ...string) {
	t.Helper()
	if output, err := runFixtureGitOutput(root, args...); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func gitFixtureOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	output, err := runFixtureGitOutput(root, args...)
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func runFixtureGitOutput(root string, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = fixtureGitEnvironment()
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return stderr.Bytes(), err
	}
	return output, nil
}

func fixtureGitEnvironment() []string {
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

func writePair(t *testing.T, path, pairID, leftRevision, rightRevision string, leftPaths, rightPaths []string, candidates []candidateInput) {
	t.Helper()
	values := make([]map[string]any, len(candidates))
	for i, candidate := range candidates {
		values[i] = map[string]any{"candidateKey": candidate.key, "leftPaths": candidate.left, "rightPaths": candidate.right}
	}
	pair := map[string]any{
		"schemaVersion": 1,
		"pairId":        pairID,
		"left":          map[string]any{"projectId": "left-fixture", "revision": leftRevision},
		"right":         map[string]any{"projectId": "right-fixture", "revision": rightRevision},
		"scope": map[string]any{
			"leftPaths":  leftPaths,
			"rightPaths": rightPaths,
			"candidates": values,
		},
	}
	data, err := json.Marshal(pair)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func snapshotFixtureFiles(t *testing.T, roots ...string) map[string][]byte {
	t.Helper()
	result := map[string][]byte{}
	for _, root := range roots {
		if err := filepath.WalkDir(root, func(name string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if name == root {
				return nil
			}
			relative, _ := filepath.Rel(root, name)
			key := filepath.Base(root) + "/" + relative
			if entry.IsDir() {
				result[key] = []byte("directory")
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				link, err := os.Readlink(name)
				if err == nil {
					result[key] = []byte("symlink:" + link)
				}
				return err
			}
			data, err := os.ReadFile(name)
			if err == nil {
				result[key] = data
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func equalByteMaps(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if !bytes.Equal(value, right[key]) {
			return false
		}
	}
	return true
}

func candidateMap(candidates []Candidate) map[string]Candidate {
	result := make(map[string]Candidate, len(candidates))
	for _, candidate := range candidates {
		result[candidate.CandidateKey] = candidate
	}
	return result
}

func digestValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("got no error, want %s", code)
	}
	var structured *ldserr.Error
	if !errors.As(err, &structured) || structured.Code != code {
		t.Fatalf("error=%v, want code %s", err, code)
	}
}

func stringPtr(value string) *string { return &value }

func stringSlicePtr(value []string) *[]string { return &value }

func clonePair(t *testing.T, pair pairFile) pairFile {
	t.Helper()
	data, err := json.Marshal(pair)
	if err != nil {
		t.Fatal(err)
	}
	var result pairFile
	if err := decodeStrictJSON(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
