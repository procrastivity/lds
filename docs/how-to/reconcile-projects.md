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

## Prepare a bounded assisted review

The original report command above remains offline and provider-free. Make a
**new** review brief with explicit selections; its purpose, reviewer, time,
claims and questions are your assertions. This disposable pair selects only
the right scoped file and tracked map for possible disclosure:

```sh
assisted_review="$work/assisted-review.json"
cat > "$assisted_review" <<EOF
{"schemaVersion":1,"purpose":"Ask for review questions about this fixture",
 "reviewer":"fixture-reviewer","at":"$(date -u +%Y-%m-%dT%H:%M:%SZ)",
 "selections":[{"side":"right","path":"docs/topic.md"},
               {"side":"right","path":"docs/map.md"}],
 "claims":[],"questions":[]}
EOF
lds plumbing reconcile flow --json
lds plumbing reconcile collect --pair "$pair" --left-root "$left" \
  --right-root "$right" --map "$right/docs/map.md" \
  --review "$assisted_review" --output "$work/from-draft.md" \
  > "$work/observed.json"
```

`collect` emits bounded original evidence JSON to stdout without publishing
the named report or contacting a provider. The disposable `mktemp -d` work
directory is private; for another location, use a trusted private directory
and restrictive `umask` before redirecting sensitive output. The captured
baseline binds the **future** report output path too. Keep it and the brief
private and unchanged. The flow reader merely returns live preparatory
guidance. The root assisted command shape for an **authorized** pair is:

```sh
# Do not run without approving the selected bytes for disclosure.
# A Linux terminal and OPENCODE_API_KEY are required for this explicit mode.
lds reconcile report --assist --pair "$pair" --left-root "$left" \
  --right-root "$right" --map "$right/docs/map.md" \
  --review "$assisted_review" --output "$work/assisted-report.md" --json
```

The host shows complete system and user content and the OpenCode Go
destination, flushes queued input, then accepts a freshly typed exact `send`
or `decline`. Only `send` transfers selected full UTF-8 scoped/map contents,
sanitized scan metadata, seed claims/questions and live flow. Selected text
is not redacted; inspect it for secrets and embedded instructions first. The
single tool-free `deepseek-v4-flash` response is unverified proposed
claims/questions; an empty claim list is valid advice. The private report
attributes provider, flow and disclosure receipt but keeps model advice
separate from reviewer claims and journal decisions. Drift refuses rather
than silently recollecting. No report call edits a root or journal.

For a separate harness or manual drafting workflow, supply **all three**
external original-baseline artifacts together to deterministic plumbing.
The draft must be strict schema-1 `claims` and `questions`, and the receipt
must attribute provider/model, flow source/hash, payload hashes/byte counts
and consent time. These are caller-supplied, not authenticated; retain the
exact collected JSON rather than recollecting after drafting:

```sh
# Illustrative: draft.json and receipt.json must already be valid private files.
# The output must match the path captured by collect and must still be new.
lds plumbing reconcile report --pair "$pair" --left-root "$left" \
  --right-root "$right" --map "$right/docs/map.md" \
  --review "$assisted_review" --output "$work/from-draft.md" \
  --observed "$work/observed.json" --draft "$work/draft.json" \
  --receipt "$work/receipt.json" --json
```

That route makes no provider call and validates the draft against exact
captured selected current evidence. It cannot authenticate consent, claims or
receipt. Do not present a handcrafted receipt as proof of a real transfer.
See the [reference](../reference/commands.md) for bounds.

For deliberate review history, name a caller-owned journal path **outside
both worktrees and their Git common directories**. `journal read` and scan
never initialize it. The first confirmed `record` may create it.

## Guide a journal entry in the terminal

Using the disposable pair and roots above, choose a new external path. Start
with a Linux terminal, even when requesting JSON output:

```sh
journal="$work/journal.json"
lds reconcile journal record --pair "$pair" --left-root "$left" \
  --right-root "$right" --map "$right/docs/map.md" \
  --journal "$journal" --json
```

The host freshly scans the declared candidate and map bytes, shows the
evidence and any prior journal decisions, and asks for **your** plain-text
correspondence (stable member keys and relation), cited provenance claims,
assessment, actor and rationale. The bucket and matching filenames are not a
reviewed match. Follow each prompt rather than piping answers or extracting a
digest automatically. Select candidates and citations by their displayed
1-based numbers. Lists use comma-separated values; a blank list means none.
For this disposable example, a reviewed `one-to-one` correspondence could use
member identifiers `left:topic` and `right:topic`; those identifiers are your
claims, not discovered matches. Answers must be single-line UTF-8, at most
4095 bytes excluding the newline. Invalid answers or EOF end the operation
without writing; the manual plumbing route supports separately authored JSON.
Before a write, inspect the **full** digest-bound
request/evidence/journal preview on stderr. Input already queued before the
preview is flushed; type `record sha256:<exact displayed digest>` freshly, or
`decline`. Decline or a failed preview makes no journal, lock or output file.
`--json` sends only a successful result to stdout; prompts and preview still
require a terminal on stderr.

After a confirmed record, a separate run can decide the pending candidate:

```sh
lds reconcile journal decide --pair "$pair" --left-root "$left" \
  --right-root "$right" --map "$right/docs/map.md" \
  --journal "$journal" --json
lds plumbing reconcile journal read --pair "$pair" --journal "$journal"
```

Declare pair/topic owner authority and permitted directions explicitly, then
select an allowed outcome: `unresolved`, `deferred` (revisit trigger),
`divergence-retained`, `intent` (permitted direction), or `applied` (permitted
direction and externally authored proof reference with explicit URI, hash,
source/destination projects and revisions, scoped paths, verifier, time and
rationale). A proof reference is an **unverified claim**, not a transfer
performed or verified by LDS. Inspect the full preview and freshly type
`decide sha256:<exact displayed digest>` or `decline`. To replace a decision
later, run `lds reconcile journal reconsider`
with the same flags. Review the latest decision to be superseded, supply an
explicit trigger and replacement authority/outcome, and optionally replace
correspondence, provenance and assessment together. Confirm only with fresh
`reconsider sha256:<exact displayed digest>` or `decline`; history remains.
No option infers authority, direction, correspondence, or semantic confidence.
All three hosts recheck pair, clean pinned roots, map and journal before
writing, refuse drift rather than silently retrying, and never edit a source,
run project commands or use a model/network. If a write result is indeterminate,
**read the journal before any retry** to check its revision and request ID.
This disposable example is not authorization for a real pair or applied claim.

## Retain the manual plumbing route

The existing `lds plumbing reconcile journal` operations remain available
when you need to author and review exact artifacts yourself. For this separate
route (not a replay of the guided record), write a request JSON with actor,
UTC time, rationale, request ID, expected journal
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
