# Reconcile evidence from two projects

Reconciliation is deterministic plumbing, not a semantic matchmaker. Before
scanning, obtain two **clean** Git worktree roots at explicit full revision
pins. Author a strict v1 pair JSON outside both roots with stable `pairId`,
ordered `left`/`right` project IDs and revisions, complete scoped path arrays,
and non-overlapping candidate buckets. The tool neither fetches nor checks
out revisions. A map is a caller-selected, tracked **raw file** at one of the
pins; its bytes are evidence, not parsed claims. All paths supplied as flags
are absolute.

For an exact, disposable offline scan, run this in a shell with `lds` and Git
available (it creates only under a new temporary directory):

```sh
work=$(mktemp -d)
left="$work/left"; right="$work/right"; pair="$work/pair.json"
for root in "$left" "$right"; do
  mkdir -p "$root/docs"
  git -C "$root" init -q
  printf '# %s topic\n' "${root##*/}" > "$root/docs/topic.md"
  printf 'raw map\n' > "$root/docs/map.md"
  git -C "$root" add docs
  git -C "$root" -c user.name=Fixture -c user.email=fixture@example.invalid commit -qm fixture
done
left_pin=$(git -C "$left" rev-parse HEAD)
right_pin=$(git -C "$right" rev-parse HEAD)
cat > "$pair" <<EOF
{"schemaVersion":1,"pairId":"example:docs:v1",
 "left":{"projectId":"left","revision":"$left_pin"},
 "right":{"projectId":"right","revision":"$right_pin"},
 "scope":{"leftPaths":["docs/topic.md"],"rightPaths":["docs/topic.md"],
 "candidates":[{"candidateKey":"topic:example","leftPaths":["docs/topic.md"],
 "rightPaths":["docs/topic.md"]}]}}
EOF
lds plumbing reconcile scan --pair "$pair" --left-root "$left" \
  --right-root "$right" --map "$right/docs/map.md" > "$work/scan.json"
```

A successful scan emits one v1 JSON object with `ok: true`, snapshots, map
hashes, and candidates. Without `--journal`, each `evidenceStatus` is
`not-compared`; `confidence` is `not-assessed`. Candidate IDs identify keys;
evidence digests change with pinned evidence. Review correspondence yourself.
For a real pair, supply your own authorized pinned roots, tracked paths, map,
and pair identity; the fixture is not an authority or correspondence claim.
With an **existing** journal, add `--journal "$journal"` to compare evidence;
status becomes `new`, `unchanged`, `changed`, or `out-of-scope` as applicable.
No status selects authority or transfer direction.

## Create an offline review report

Using the disposable roots and pair from the scan above, author a minimal
review brief outside both roots. The timestamp and reviewer are **assertions**,
not authenticated identities or approvals. Empty selections and claims are
valid: this report does not invent a correspondence from matching names or
bytes. The output path must not exist yet.

```sh
review="$work/review.json"; report="$work/report.md"
cat > "$review" <<EOF
{"schemaVersion":1,"purpose":"Review the disposable pair's evidence",
 "reviewer":"fixture-reviewer","at":"$(date -u +%Y-%m-%dT%H:%M:%SZ)",
 "selections":[],"claims":[],"questions":[]}
EOF
lds reconcile report --pair "$pair" --left-root "$left" \
  --right-root "$right" --map "$right/docs/map.md" \
  --review "$review" --output "$report" --json
ls -l "$report"
```

`lds plumbing reconcile report` takes the same flags and produces the same
deterministic report. Inspect the Markdown and provenance JSON appendix as
private review material; the file is created externally with mode 0600 and
cannot be overwritten by a rerun. To rerun, name a different **new** external
output. With a pre-existing journal, add `--journal "$journal"`; neither form
initializes, locks, or writes it. For opt-in full UTF-8 content, add entries
such as `{"side":"left","path":"docs/topic.md"}` to `selections` (a
declared scoped file or map). Unselected raw files are still checked against
the per-file size bound. The offline report calls no model and does not
authenticate claims, select authority, or prove an applied transfer. It
revalidates inputs before publication, but a later change requires a fresh
report and review. See the [command reference](../reference/commands.md) for
limits and the [safety notes](../explanation/safety-and-scope.md) for trust
boundaries.

For deliberate review history, name a caller-owned journal path **outside
both worktrees and their Git common directories**. `journal read` and scan
never initialize it. The first explicit `record` may create it. Write a
request JSON with actor, UTC time, rationale, request ID, expected journal
revision and evidence digest, plus reviewed correspondence/provenance and
assessment; use the [v1 contract](../../engineering/specs/cross-project-reconciliation-v1.md)
for exact request fields and allowed values. Set `request` to the absolute path
of that authored JSON; keep the exact scan JSON bytes:

```sh
lds plumbing reconcile journal record --pair "$pair" --journal "$journal" \
  --scan "$work/scan.json" --request "$request"
lds plumbing reconcile journal read --pair "$pair" --journal "$journal"
```

Only an explicit `decide` request records initial authority and outcome;
`reconsider` names the decision superseded and a trigger, retaining history.
Both require `--pair`, `--journal`, `--scan` (fresh exact saved scan), and
`--request` with operation-specific fields. Stale revision/evidence refuses;
on an uncertain write outcome read the journal revision and request ID before
retrying. Journal writes are confined to the named location. Nothing edits
source files, executes a transfer, authenticates an `appliedProof`, or chooses
which project is authoritative. Consult the [command reference](../reference/commands.md)
for spellings and [scope notes](../explanation/safety-and-scope.md) for limits.
