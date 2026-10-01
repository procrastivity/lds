# Cross-project reconciliation operation contract — v1

**Status:** Accepted operation contract for ADR-0003 D1A–D5A. The JSON and
journal formats in this document are version 1; this contract selects the
spellings and storage left open by D5A. It specifies behavior, not an
implementation or a live reconciliation decision.

This contract implements the boundary in
[ADR-0003](../decisions/0003-bound-cross-project-reconciliation.md). Its
examples derive from the pinned [Stage 1 evidence](../product/bds-252-stage-1-evidence.md)
and [expected-case cards](../product/bds-252-stage-1-expected-case-cards.md).
Those inputs bound evidence; they do not authorize a pilot, choose a real
pair's authority, or authorize edits to Scind, Xcind, or any other source.

## D5A command surface

Reconciliation is deterministic plumbing only. The canonical command tree is:

```text
lds plumbing reconcile scan --pair <absolute-pair.json> \
  --left-root <absolute-dir> --right-root <absolute-dir> \
  --map <absolute-file> [--map <absolute-file> ...] \
  [--journal <absolute-journal.json>]

lds plumbing reconcile journal read --pair <absolute-pair.json> \
  --journal <absolute-journal.json>

lds plumbing reconcile journal record --pair <absolute-pair.json> \
  --journal <absolute-journal.json> --scan <absolute-scan.json> \
  --request <absolute-record-v1.json>

lds plumbing reconcile journal decide --pair <absolute-pair.json> \
  --journal <absolute-journal.json> --scan <absolute-scan.json> \
  --request <absolute-decision-v1.json>

lds plumbing reconcile journal reconsider --pair <absolute-pair.json> \
  --journal <absolute-journal.json> --scan <absolute-scan.json> \
  --request <absolute-reconsider-v1.json>
```

Every path is explicit and absolute; there are no environment, current-directory,
remote, or implicit journal defaults. `--map` is repeatable and names raw,
tracked map artifacts inside either pinned snapshot. The scanner hashes map
bytes and reports their paths; it does not parse labels into correspondence or
authority. Repeated maps resolving to the same canonical tracked path are
invalid, even if supplied through different path spellings. `--scan` is a
saved v1 scan response. All write operations consume that exact response; they
do not rerun scans. `scan` and `read` never
create or write a journal. A missing journal is created only by an explicit
`record` request at its caller-supplied path.

Journal commands require the pair ID and left/right project IDs to match the
journal. Journal identity is the stable pair ID plus ordered project IDs;
revisions are recorded per evidence observation. A write rejects a scan whose
pair descriptor digest or pinned revisions differ from the explicitly named
pair file. `decide` also requires the scan candidate's evidence digest to match
the latest journal observation. If a new scan changes evidence, `record` may
append that observation without changing the decision; a decided candidate
then requires explicit `reconsider`.

The pair file is a caller-owned JSON descriptor with required `schemaVersion: 1`,
`pairId`, `left` and `right` objects (`projectId`, full Git `revision`), and a
declared `scope`; no other fields are accepted at these object levels.
`scope.leftPaths` and `scope.rightPaths` are the complete repo-relative file
sets to inspect. `scope.candidates` contains unique stable `candidateKey`
values and subsets of those paths for review bucketing. Each bucket has exactly
`candidateKey`, `leftPaths`, and `rightPaths` (all required); no correspondence,
authority, or outcome is encoded in a bucket. Any scoped path not in a
candidate bucket is emitted as a side-qualified unmatched scan candidate.
Each side-qualified path may occur in at most one bucket; duplicate paths or
overlapping buckets are invalid rather than silently assigning one bucket.
Empty path lists on one side are valid; a bucket empty on both sides is not.
Grouping paths under a candidate key is only a scan boundary, not a reviewed
semantic match. Only a journal `record` can assert a reviewed correspondence.

Each snapshot root must be a clean local Git checkout whose `HEAD` exactly
matches the pair file's full object ID. The implementation does not fetch,
checkout, or otherwise contact a remote. It reads only declared, tracked files;
scope paths must be normalized repo-relative paths without `..`, absolute
prefixes, control characters, or symlink escapes. A supplied map file must be
tracked in one of those same pinned snapshots. Matching byte hashes establish
byte equality only, never semantic equivalence.

## Candidate identity, evidence, and reviewed correspondence

The pair file's candidate key is a stable, caller-selected bucket name. For an
unassigned path, the key is `unmapped:<side>:<repo-relative-path>`. Candidate
keys supplied in `scope.candidates` must not begin with the reserved `unmapped:`
prefix; reject them rather than colliding with generated keys. All supplied and
generated keys within a pair must be unique before computing IDs. Candidate
identity is independent of source revisions, file contents, map bytes, and
review outcome:

```text
candidateId = "cand-v1-" + hex(SHA256(
  "lds-reconcile-candidate-v1\n" + pairId + "\n" + candidateKey + "\n"))
```

The full lowercase SHA-256 is retained. Revisions or evidence changes therefore
do not rename a candidate; changing the pair ID or candidate key creates a new
identity and does not delete the old journal history.

`evidenceDigest` is a separate `sha256:<64 lowercase hex>` value over the exact
evidence set for one candidate. To make the digest reproducible across working
directory locations, its UTF-8 input is the following newline-terminated
sequence:

1. `lds-reconcile-evidence-v1`, pair ID, candidate key, left project ID, left
   revision, right project ID, right revision, each on its own line.
2. One row per evidence item, sorted by
   `(kind, projectId, revision, key, sha256)`, with tab-separated fields
   `kind`, `projectId`, `revision`, `key`, `sha256`.
   Artifact `key` is its repo-relative path. Every supplied map artifact is
   included as an `artifact` item in **each** candidate's evidence set, even
   when it is not a scoped candidate member; a changed map therefore changes
   every candidate evidence digest. If a map is also a scoped member, include
   that artifact only once per candidate. A missing scoped artifact uses
   `kind: "missing"` and the literal `-` for `sha256`.

IDs and paths reject tabs, newlines, and NUL so these delimiters are
unambiguous. Actor, time, rationale, confidence, trust assessment, and decision
state are not part of `evidenceDigest`; updating a human assessment must not
masquerade as changed source evidence. A scan reports its exact sorted evidence
items as well as the digest.

A journal record's `correspondence` has a stable, explicitly reviewed `key`,
`leftMembers`, `rightMembers`, and one relation:

| Relation | Left members | Right members |
| --- | ---: | ---: |
| `one-to-one` | 1 | 1 |
| `left-unmatched` | 1 | 0 |
| `right-unmatched` | 0 | 1 |
| `split` | 1 | 2 or more |
| `merged` | 2 or more | 1 |

Members are stable project-local node keys, not filenames inferred by LDS. The
correspondence key is unique within a pair; a member can occur in only one
current correspondence. Many-to-many is outside v1 and is refused, not
flattened. An edit to an already decided relationship requires `reconsider`;
history remains intact.

Each reviewer `provenance` item has exactly `kind`, `projectId`, `revision`,
`path`, `sha256`, `claim`, and `trust`. For `artifact`/`missing`, name a pair
project, full pinned revision, normalized repo-relative path, and lowercase
SHA-256 (`-` for missing). When it references a scanned row at that revision,
`path` corresponds to the row's `key` and the hash must agree. Other reviewer
references are assertions, not additional scanned evidence. Preserved review
metadata can refer to an earlier pin after the observation changes. For
`synthetic`, `projectId` is `synthetic`, `revision` and `sha256` are null, and
`path` is the fixture key, never a claimed source path/hash. The bounded claim
is non-empty and its claim-level trust is one of:
`trusted`, `bounded`, `untrusted`, `contradictory`, or `synthetic`. Trust is
about that claim, not blanket authority for a repository. Scanning never
assigns semantic confidence: its value is `not-assessed`. A reviewer may record
`low`, `medium`, or `high` confidence with an explicit rationale; these are
qualitative human assessments, not scores or thresholds for automation.
The scanner itself emits only `artifact` and `missing` observations from
declared paths and maps. `synthetic` is reviewer-supplied provenance for a
disposable fixture claim, not an extra scanner input or a source-file hash;
it does not enter `evidenceDigest`. To exercise the synthetic split through
the CLI, use separate disposable pinned Git fixture repos with tracked
stand-in files, bucket them under one candidate key, then record synthetic
member keys and provenance. Their scanned file hashes describe only fixture
bytes, never Scind/Xcind facts or the invented nodes' source hashes.

## Journal format and operations — D1A, D2A, D4A

The on-disk journal is one UTF-8 `journal.json` document. It is caller-owned and
must be outside both source roots and both Git common directories. Its v1
top-level shape is:

```json
{
  "schemaVersion": 1,
  "pairId": "<stable-pair-id>",
  "leftProjectId": "<project-id>",
  "rightProjectId": "<project-id>",
  "revision": 0,
  "events": []
}
```

The journal is an append-only event history with a materialized read view; it
does not store a second editable map. Each event has exactly the following
required fields, in this serialization order: `seq`, `requestId`, `candidateId`,
`type`, `actor`, `at`, `rationale`, `request`, `scanIdentity`, `payload`.
`seq` is a consecutive positive integer, `requestId` is unique across the
journal, `at` is UTC RFC 3339, and `rationale` is non-empty. `revision` is the
last event sequence (zero for a newly created journal). The actor is supplied,
never inferred from the host. Journal reads derive current correspondence,
evidence, authority, and decision state by replaying events; arrays are sorted
by `candidateId`, with history in increasing `seq` order.
The v1 event types are `candidate.recorded`, `evidence.observed`,
`decision.recorded`, and `decision.reconsidered`; unknown event types are not
silently ignored.

All write requests are JSON objects with required `schemaVersion: 1`,
`requestId`, `expectedJournalRevision`, `candidateId`, `expectedEvidenceDigest`,
`actor`, `at`, `rationale`, and the operation-specific fields below. `request`
stores that **entire validated original request object**, not a selected subset
or a projection reconstructed later. Equality for retry means JSON value
equality, including every field and array order (object member order and
whitespace are immaterial). Envelope `requestId`, `candidateId`, `actor`, `at`,
and `rationale` must equal their request values. The request's expected revision
must equal `seq - 1` for an appended event, and its expected evidence digest
must equal the payload's digest.

`scanIdentity` has exactly `scanSha256`, `pairSha256`, `candidateId`, and
`evidenceDigest`. The first is the lowercase raw SHA-256 of the **exact consumed
scan file bytes**, including whitespace and final newline; `pairSha256` is the
raw lowercase SHA-256 reported for the consumed pair descriptor. The candidate
and evidence values must match the selected scan candidate, envelope, request,
and payload. Different serialized scan bytes are a different scan identity
even when their parsed observations agree. Request equality **and** complete
`scanIdentity` equality drive idempotence; an evidence digest alone does not.

For a **new** request ID, writes
require an exact journal revision and `expectedEvidenceDigest` for the
candidate. `record` supplies the `candidateId`, reviewed correspondence,
provenance/trust/confidence; its exact observation is taken from the validated
scan, not guessed from those reviewer fields. `decide` supplies
the explicit authority declaration and outcome, and must match the latest
observation in its scan. `reconsider` includes a fresh scan, trigger reference,
replacement outcome, and the decision it supersedes. Under the writer lock,
after validating the request and pair identity, check whether `requestId` is
already in the journal **before** comparing the expected journal revision or
current evidence. If its validated fields and consumed scan identity match the
stored event, return success with the current materialized journal projection,
current revision, and `replayed: true`; append nothing, even if later requests
have advanced the journal. The same ID with different fields or scan identity
is a conflict. Only a new ID proceeds to revision/evidence checks and an append
(success with `replayed: false`).

Every event `payload` begins with the same complete observation, in order:
`candidateKey`, `snapshots`, `evidenceDigest`, `evidence`. `snapshots` is exactly
`left` and `right`, each with `projectId` and full pinned `revision`. Each
`evidence` item has exactly `kind`, `projectId`, `revision`, `key`, `sha256`, in
that order, and the array follows the evidence-digest tuple sort above. This
observation is copied from the selected valid scan candidate and its snapshots;
its digest and candidate ID must recompute under the unchanged v1 formulas.
Only scanned `artifact`/`missing` items enter this array. Reviewer `provenance`
uses `path` (or the synthetic fixture key in that field), `claim`, and `trust`
in addition to its kind/project/revision/hash; it is distinct from evidence's
digest-bearing `key` and does not add evidence rows or alter their digest.

| Event type / request operation | Additional request and payload fields after the common fields | Replay effect |
| --- | --- | --- |
| `candidate.recorded` / `record` | Required `correspondence`, `provenance`, `assessment`. Assessment has `code`, `confidence` (`level`, and a non-empty `rationale` for low/medium/high), and claim-level `trust`. | Introduce the previously absent candidate, exact observation and reviewed fields; authority is not declared, decision is `pending-review`, and `lastDecision` is null. |
| `evidence.observed` / `record` | The same required reviewed fields as the first record. | Update the current observation and review metadata, never an existing authority/outcome or decision digest. On a decided candidate the supplied correspondence must equal the current one; changing it requires reconsideration. On a still-pending candidate an explicit record may replace reviewed correspondence subject to cardinality/member uniqueness. |
| `decision.recorded` / `decide` | Required `authority`, `outcome`. | Select an outcome only for a recorded `pending-review` candidate and the same latest observed evidence; otherwise refuse, not overwrite a decision. |
| `decision.reconsidered` / `reconsider` | Required `supersedesSeq`, `trigger`, `authority`, `outcome`; optional `correspondence`, `provenance`, `assessment` only as an all-or-none replacement group. | Validate that `supersedesSeq` names this candidate's latest decision event, then explicitly replace its outcome/authority and observed evidence. If the replacement group is absent, preserve the reviewed relationship/metadata; if present, replace it subject to all correspondence/provenance checks. Retain every superseded event. |

These request-specific fields are copied unchanged into the corresponding
payload; the common observation is derived from the consumed scan. A
`reconsider` may consume new evidence directly without an intervening `record`;
its expected digest must match its scan. A plain `decide` cannot. No operation
implicitly transitions an existing decision because an observation changed.
`trigger` is exactly `id`, `kind` (`event`, `date`, or `condition`), and
`description`. `outcome` contains required `state` and only the fields appropriate
to that state: `unresolved` and `divergence-retained` have no additional fields;
`intent` requires `direction`; `deferred` requires `trigger`; `applied` requires
`direction` and `appliedProof` subject to the existing proof refusal below.
A `deferred` outcome requires a non-empty trigger ID, trigger kind
(`event`, `date`, or `condition`), and explicit condition/description. The
trigger is never polled or acted on automatically.

Replay validates the whole journal before using it: known event shapes and
types, consecutive sequences/revision, unique request IDs, request/envelope and
request/payload agreement, operation/type agreement, identity/digest formulas,
stable pair/project identity, and lawful lifecycle/supersession. Invalid or unknown fields/events,
malformed payloads, contradictory identities or digests, and inconsistent
request/sequence history are invalid v1 evidence (exit 1), not tolerated guessed
payloads (`validation.reconcile-evidence-invalid`). A live new request with a
stale expected revision/digest still follows the existing exit-3 refusal rules;
malformed persisted history is not a retry conflict. The journal stores the raw
scan hash as provenance, not the original scan bytes; replay cannot independently
reconstruct that hash from the payload.
Writers verify it against the consumed file and replay checks its shape and
agreement with the stored candidate/digest identities. A journal is not an
authenticated proof of its author's claims.

| Operation | Effect and refusal boundary |
| --- | --- |
| `journal read` | Read and project the named journal for the named pair. Missing or incompatible files fail; read never initializes them. |
| `journal record` | Append a reviewed correspondence, provenance/trust/confidence assessment, and the exact scan evidence. A first record enters `pending-review`, not pending port work. A repeat scan may append `evidence.observed` while preserving the candidate's existing decision. |
| `journal decide` | Append one explicit D2A authority declaration and D4A outcome. Outcomes are `unresolved`, `deferred` (requires a revisit trigger), `divergence-retained`, `intent` (requires an allowed `left-to-right` or `right-to-left` direction), or `applied` (requires external proof). |
| `journal reconsider` | Against a fresh scan, append an explicit trigger/evidence review and replacement outcome. It preserves the superseded decision and never runs automatically on changed bytes, changed maps, or a later revision. |

An authority declaration is scoped to the pair and correspondence key. It names
the responsible `owner`, a `status` of `declared` or `unresolved`, an
`authoritativeProjectId` (one of the pair or null while unresolved), and
`permittedDirections` (`left-to-right` and/or `right-to-left`; empty while
unresolved). The event's actor, time, and rationale record who made that
declaration. Unknown authority or an unapproved direction can only remain
`unresolved`; LDS does not choose a canonical side. Any outcome other than
`unresolved` requires an explicit declaration. An `intent` records intent
only; it is not permission to edit a source.

`applied` requires an `appliedProof` reference containing an externally
verifiable artifact of the actual transfer. Its object has exactly these
required fields, in serialization order; reader and writer use the same names,
types, and nested shapes, rejecting unknown fields at every level:

| Field | Required v1 type and meaning |
| --- | --- |
| `uri` | Absolute URI string naming the external artifact; a reference only. |
| `sha256` | String of exactly 64 lowercase hexadecimal characters, asserting the referenced artifact's byte hash. |
| `source` | Object with exactly `projectId` (source pair-project ID string) and `revision` (full lowercase Git object ID string: 40 or 64 hex characters), in that order. |
| `destination` | Object with exactly `projectId` (destination pair-project ID string) and `revision` (full lowercase Git object ID string: 40 or 64 hex characters), in that order. |
| `scope` | Object with exactly `sourcePaths` and `destinationPaths`, in that order. Each is a non-empty array of unique, normalized repo-relative file-path strings, sorted lexicographically, with the existing path/control-character restrictions. |
| `verifier` | Non-empty supplied identity string; not inferred or authenticated. |
| `verifiedAt` | UTC RFC 3339 string recording the supplied verification time. |
| `rationale` | Non-empty string explaining the claimed transfer verification. |

For `left-to-right`, `source.projectId` equals the pair's left project and
`destination.projectId` its right project; `right-to-left` reverses them.
They must be distinct pair projects. The outcome direction must be permitted
by the declared authority, but `authoritativeProjectId` need not equal the
source project when the owner permits a backport. Proof revisions need not
equal the current scan pins: the artifact may describe a later destination
transfer revision. Scope array lengths may differ for a split or merge and do
not assert positional pairing or runtime parity.

An otherwise valid live applied request omitting `appliedProof` retains exit 3,
`refusal.reconcile-applied-proof-required`. A supplied malformed proof (including
wrong types, unknown fields, invalid revisions/paths, or project/direction
mismatch) is exit 1, `validation.reconcile-evidence-invalid`; validate that
shape and project/direction agreement rather than treating it as an authority
refusal. A structurally valid proof for a direction not permitted by the
declaration remains exit 3, `refusal.reconcile-authority-unresolved`. A
persisted applied event with absent or malformed proof, project/direction
mismatch, or an impermissible authority/direction is invalid replay history:
exit 1, `validation.reconcile-evidence-invalid`, not a live write refusal.

URI, hash, verification metadata, and the claim that the artifact describes an
actual transfer are reviewer assertions. LDS neither fetches the URI nor
authenticates the verifier or independently verifies those assertions. A
matching source hash, an intent, a map label, or a journal state is not proof;
no applied outcome or runtime parity is inferred automatically. Recording the
reference and review never performs a transfer or mutates either source.

## Scan and JSON contract — D3A, D5A

These plumbing operations always emit exactly one v1 JSON object to stdout on
success; there is no human-text success mode or verb-specific `--json` switch.
The root's inherited `--json` flag remains available for machine-readable
errors. Outputs have
`schemaVersion: 1`, canonical `operation`, and `ok: true`. A scan contains the
pair ID, `pairPath` and raw-byte `pairSha256`, explicit input paths, pinned
snapshot IDs/revisions, sorted map path/hash observations, and sorted
candidates. Each candidate contains its
stable ID/key, current evidence digest, sorted per-side path/presence/byte-hash
observations, sorted digest-bearing `evidence`, and always `evidenceStatus`.
For a complete scan response, the fixed field order is `schemaVersion`,
`operation`, `ok`, `pairId`, `pairPath`, `pairSha256`, `leftRoot`, `rightRoot`,
`snapshots`, `maps`, optional `journalPath`, `candidates`. Each map item is
`projectId`, `revision`, `path`, `sha256`, sorted by that tuple. Each candidate
is `candidateId`, `candidateKey`, `evidenceDigest`, `evidence`, `left`, `right`,
`confidence` (always `not-assessed`), `evidenceStatus`, then the three journal
comparison fields below when supplied. `left` and `right` are arrays sorted by
`path` of `{path, present, sha256}`; absent files have `present: false` and
`sha256: null` there, but their digest-bearing evidence row uses `missing`/`-`.
Its comparison to a journal is fixed:

| Inputs / candidate | `evidenceStatus` | Journal comparison fields |
| --- | --- | --- |
| No `--journal` | `not-compared` | `journalRevision`, `journalEvidenceDigest`, and `lastDecision` are absent, not null or inferred. |
| Valid supplied journal; candidate absent from its replayed view | `new` | Current journal revision; `journalEvidenceDigest: null`, `lastDecision: null`. |
| Candidate present in scan and journal; digest equals latest observed digest | `unchanged` | Current journal revision, saved observed digest, and saved last decision or null. |
| Candidate present in both; digest differs from latest observed digest | `changed` | The same saved metadata; comparison never clears or replaces it. |
| Journal candidate absent from current scan scope | `out-of-scope` | Preserve the saved observed digest and decision; current `evidenceDigest: null`, `evidence: []`, and both current side-path arrays empty. |

`journalRevision` is the supplied journal's global revision, not a candidate
event sequence. `journalEvidenceDigest` is its latest **observed** candidate
digest. `lastDecision` is null before any decision, otherwise exactly
`{seq, evidenceDigest, outcome}` from the latest decision/reconsider event.
Its digest can differ from `journalEvidenceDigest` after an explicit record
of changed evidence preserves an older decision; this distinction must remain
visible. For example, decide at seq 2 on digest E0, then record at seq 3 on E1:
scanning E1 is `unchanged` against the latest observation, but `lastDecision`
still carries seq 2, E0, and that decision's unchanged outcome. The comparison
does not claim the decision was made on E1. The union of current scan and
journal candidate IDs is sorted by ID;
journal-only keys come from saved observations, not invented current evidence.
A missing or incompatible explicitly supplied journal fails with empty stdout
and no initialization. For no-journal scans only current candidates are emitted.
The scanner does not parse semantic map claims, choose trust,
assign confidence, or create/change a journal state.

Journal read emits the same envelope with `journalPath`, `pairId`, current
`revision`, candidates sorted by ID, and append-only events sorted by sequence.
Its candidate view contains `candidateId`, the common latest observation fields,
`correspondence`, `provenance`, `assessment`, `authority`, `decision`,
`lastDecision`, and `lastEventSeq`. Initially `authority` is
`{"status":"not-declared"}` and `decision` is `{"state":"pending-review"}`;
later they are the saved declaration and outcome. All full events appear in
`events`; a shortened `lastEvent` object is not an event envelope. Write success
uses this same view with its own operation name and required `replayed` boolean.
JSON object keys have fixed v1 field order; arrays use the sorting rules above.
No scan timestamp or host-dependent ordering enters the result. With identical
pair bytes, canonical input paths, pinned source bytes, map bytes, and journal
revision, a scan is byte-for-byte deterministic.

Failures follow LDS's existing CLI posture: stdout is empty. Verb-raised
`ldserr` failures write the stable JSON error envelope to stderr with the
inherited `--json` flag, or a human diagnostic without it. Cobra usage and
argument errors write a human diagnostic even with `--json`. Error codes are
stable tokens; messages may improve without changing the code.

| Exit | Meaning |
| ---: | --- |
| 0 | Operation completed, whether or not scan findings exist. |
| 1 | Invalid v1 JSON, invalid pair/scope/correspondence, or missing required evidence. |
| 2 | CLI usage/argument error. |
| 3 | Safety refusal, stale evidence/revision, journal conflict/lock, or forbidden write location. No journal update. |
| 4 | Unexpected I/O or internal failure. |

The stable operation-specific codes are:

| Error code | Exit | Example |
| --- | ---: | --- |
| `validation.reconcile-pair-invalid` | 1 | Malformed pair descriptor, unsupported relation cardinality, or out-of-scope member. |
| `validation.reconcile-evidence-invalid` | 1 | Invalid scan/request JSON or missing pinned evidence. |
| `refusal.reconcile-write-target` | 3 | Journal, lock, or replacement path resolves inside either source snapshot/Git directory. |
| `refusal.reconcile-stale-evidence` | 3 | Pair pins, scan digest, or expected candidate digest do not match. |
| `refusal.reconcile-journal-conflict` | 3 | Journal revision changed, writer lock is held, or request ID was reused with different content. |
| `refusal.reconcile-authority-unresolved` | 3 | Directional/non-unresolved outcome lacks an explicit allowed authority declaration. |
| `refusal.reconcile-applied-proof-required` | 3 | An applied outcome lacks the required external proof reference. |
| `internal.reconcile-journal-write` | 4 | Journal serialization, flush, or atomic replacement failed. |

## Stage 1 expected lifecycle

These are disposable fixture expectations derived from the four Stage 1 cards,
not accepted owner decisions. The linked [case cards](../product/bds-252-stage-1-expected-case-cards.md)
and [evidence note](../product/bds-252-stage-1-evidence.md) contain the full
source paths, hashes, and bounded claims.

| Fixture | Candidate relation / fixture assessment | Trust and provenance treatment | Initial journal state; prohibited inference |
| --- | --- | --- | --- |
| Case 1 — Scind ADR-0011 / Xcind ADR-0023 | `one-to-one`; same topic, opposite targeting; assessment `divergence-evidence`. | Scind ADR is trusted evidence of accepted design text, but Scind runtime is `untrusted` (no implementation paths in the pinned tree). Xcind code observations remain bounded to cited paths. | `pending-review` until an owner decides. Do not infer which project is authoritative or call it Scind-only. |
| Case 2 — Scind ADR-0017 / Xcind ADR-0022 | `one-to-one`; contradictory external-proxy map assessment is `unresolved`. | Preserve both Xcind Markdown and JSON claims (`XCIND-ONLY`/null) and the conflicting topic-level `MATCH`; cited records credit promotion as already applied, but runtime parity is unverified. | `pending-review` is review state, not pending port work. Do not replay the credited promotion or set `applied` without external proof. |
| Case 3 — TCP/SNI future versus rejected | Topic candidate; the cards do not ratify a cardinality. Assessment `accepted-defer-example`, not a shipped gap. | Scind `Future` is roadmap intent and runtime is `untrusted`; Xcind's rejection/trigger is a source-record claim, with only bounded test evidence. | `pending-review` in LDS fixtures; the example trigger is Scind promoting TCP/SNI from Future to active design. Do not auto-adopt Xcind's divergence status as an LDS decision. |
| Case 4 — synthetic route decomposition | `split`: one left member to two right members; assessment/trust `synthetic`. | Invented nodes have no Scind/Xcind path or hash. Disposable tracked stand-ins have fixture-only paths and hashes, not source-project evidence. | `pending-review` only in disposable fixture data. Do not describe this relation as Scind/Xcind fact. |

## Complete pair, scan, and journal examples

These are fixture-only Case 2 examples, not executed CLI output, a live journal,
or permission to access or edit a source. The pair and scan raw-byte hashes
below use the exact JSON fenced bodies including one final LF. Candidate and
evidence formulas are unchanged; this example's pair ID differs from the
independent Stage 2 fixture pair. The example `--map` inputs are the two pinned
Xcind correspondence files, outside this minimal pair's scoped member paths.

Complete minimal `pair.json`:

```json
{
  "schemaVersion": 1,
  "pairId": "example-stage1-fixture-not-authorized",
  "left": {"projectId": "scind", "revision": "c63cbc686afd2ca6ecc01eab7825d637d49378c7"},
  "right": {"projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66"},
  "scope": {
    "leftPaths": ["engineering/decisions/0017-external-proxy-mode.md"],
    "rightPaths": ["engineering/decisions/0022-external-proxy-mode.md"],
    "candidates": [
      {"candidateKey": "fixture:stage1:case-2", "leftPaths": ["engineering/decisions/0017-external-proxy-mode.md"], "rightPaths": ["engineering/decisions/0022-external-proxy-mode.md"]}
    ]
  }
}
```

Complete saved `scan.json`, without journal comparison:

```json
{
  "schemaVersion": 1,
  "operation": "reconcile scan",
  "ok": true,
  "pairId": "example-stage1-fixture-not-authorized",
  "pairPath": "/tmp/example-only/pair.json",
  "pairSha256": "cce7a44185fe70790d8eb7ff366ad3ee3c57d409289ce70ac79b3e6cd57cf8f0",
  "leftRoot": "/tmp/example-only/scind",
  "rightRoot": "/tmp/example-only/xcind",
  "snapshots": {
    "left": {"projectId": "scind", "revision": "c63cbc686afd2ca6ecc01eab7825d637d49378c7"},
    "right": {"projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66"}
  },
  "maps": [
    {"projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "path": "engineering/sync/artifacts/correspondence-map.json", "sha256": "175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17"},
    {"projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "path": "engineering/sync/artifacts/correspondence-map.md", "sha256": "3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05"}
  ],
  "candidates": [
    {
      "candidateId": "cand-v1-472eb3a0bf207fb4fb79879a0b6f6f8aa48d453dcb77346dd352ff85d556b1b1",
      "candidateKey": "fixture:stage1:case-2",
      "evidenceDigest": "sha256:2f88fd65187e2180dfd6ce2f6eb460dcf3e554e62088440c9a8dea7f484e5ee0",
      "evidence": [
        {"kind": "artifact", "projectId": "scind", "revision": "c63cbc686afd2ca6ecc01eab7825d637d49378c7", "key": "engineering/decisions/0017-external-proxy-mode.md", "sha256": "9557cbfde3718cc23918e20effb76498eb788b1630e93396b49c00fddad1b7cc"},
        {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "key": "engineering/decisions/0022-external-proxy-mode.md", "sha256": "db5d802fb20b687829d46e5438334f00c33b3821ca4cb7cff3c3ef2d51377723"},
        {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "key": "engineering/sync/artifacts/correspondence-map.json", "sha256": "175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17"},
        {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "key": "engineering/sync/artifacts/correspondence-map.md", "sha256": "3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05"}
      ],
      "left": [{"path": "engineering/decisions/0017-external-proxy-mode.md", "present": true, "sha256": "9557cbfde3718cc23918e20effb76498eb788b1630e93396b49c00fddad1b7cc"}],
      "right": [{"path": "engineering/decisions/0022-external-proxy-mode.md", "present": true, "sha256": "db5d802fb20b687829d46e5438334f00c33b3821ca4cb7cff3c3ef2d51377723"}],
      "confidence": "not-assessed",
      "evidenceStatus": "not-compared"
    }
  ]
}
```

Complete stored `journal.json` after one explicit fixture-only record. The
`request` object is also the complete supplied record request; it does not
carry the common observation a second time. No event field is omitted:

```json
{
  "schemaVersion": 1,
  "pairId": "example-stage1-fixture-not-authorized",
  "leftProjectId": "scind",
  "rightProjectId": "xcind",
  "revision": 1,
  "events": [
    {
      "seq": 1,
      "requestId": "fixture-record-case-2",
      "candidateId": "cand-v1-472eb3a0bf207fb4fb79879a0b6f6f8aa48d453dcb77346dd352ff85d556b1b1",
      "type": "candidate.recorded",
      "actor": "fixture-only",
      "at": "2026-10-01T00:00:00Z",
      "rationale": "Fixture only: conflicting map claims; no transfer authorized.",
      "request": {
        "schemaVersion": 1,
        "requestId": "fixture-record-case-2",
        "expectedJournalRevision": 0,
        "candidateId": "cand-v1-472eb3a0bf207fb4fb79879a0b6f6f8aa48d453dcb77346dd352ff85d556b1b1",
        "expectedEvidenceDigest": "sha256:2f88fd65187e2180dfd6ce2f6eb460dcf3e554e62088440c9a8dea7f484e5ee0",
        "actor": "fixture-only",
        "at": "2026-10-01T00:00:00Z",
        "rationale": "Fixture only: conflicting map claims; no transfer authorized.",
        "correspondence": {"key": "topic:external-proxy-mode", "leftMembers": ["scind:adr:0017"], "rightMembers": ["xcind:adr:0022"], "relation": "one-to-one"},
        "provenance": [
          {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "path": "engineering/sync/artifacts/correspondence-map.json", "sha256": "175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17", "claim": "The file-level JSON row says XCIND-ONLY and scind is null.", "trust": "contradictory"},
          {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "path": "engineering/sync/artifacts/correspondence-map.md", "sha256": "3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05", "claim": "The topic table says MATCH while the file row says XCIND-ONLY.", "trust": "contradictory"}
        ],
        "assessment": {"code": "unresolved-map", "confidence": {"level": "not-assessed"}, "trust": "contradictory"}
      },
      "scanIdentity": {
        "scanSha256": "0ceb5c524c3dd066669830598c24a1c69a758d6542b577e02d5ad6de1b465e79",
        "pairSha256": "cce7a44185fe70790d8eb7ff366ad3ee3c57d409289ce70ac79b3e6cd57cf8f0",
        "candidateId": "cand-v1-472eb3a0bf207fb4fb79879a0b6f6f8aa48d453dcb77346dd352ff85d556b1b1",
        "evidenceDigest": "sha256:2f88fd65187e2180dfd6ce2f6eb460dcf3e554e62088440c9a8dea7f484e5ee0"
      },
      "payload": {
        "candidateKey": "fixture:stage1:case-2",
        "snapshots": {
          "left": {"projectId": "scind", "revision": "c63cbc686afd2ca6ecc01eab7825d637d49378c7"},
          "right": {"projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66"}
        },
        "evidenceDigest": "sha256:2f88fd65187e2180dfd6ce2f6eb460dcf3e554e62088440c9a8dea7f484e5ee0",
        "evidence": [
          {"kind": "artifact", "projectId": "scind", "revision": "c63cbc686afd2ca6ecc01eab7825d637d49378c7", "key": "engineering/decisions/0017-external-proxy-mode.md", "sha256": "9557cbfde3718cc23918e20effb76498eb788b1630e93396b49c00fddad1b7cc"},
          {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "key": "engineering/decisions/0022-external-proxy-mode.md", "sha256": "db5d802fb20b687829d46e5438334f00c33b3821ca4cb7cff3c3ef2d51377723"},
          {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "key": "engineering/sync/artifacts/correspondence-map.json", "sha256": "175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17"},
          {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "key": "engineering/sync/artifacts/correspondence-map.md", "sha256": "3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05"}
        ],
        "correspondence": {"key": "topic:external-proxy-mode", "leftMembers": ["scind:adr:0017"], "rightMembers": ["xcind:adr:0022"], "relation": "one-to-one"},
        "provenance": [
          {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "path": "engineering/sync/artifacts/correspondence-map.json", "sha256": "175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17", "claim": "The file-level JSON row says XCIND-ONLY and scind is null.", "trust": "contradictory"},
          {"kind": "artifact", "projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66", "path": "engineering/sync/artifacts/correspondence-map.md", "sha256": "3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05", "claim": "The topic table says MATCH while the file row says XCIND-ONLY.", "trust": "contradictory"}
        ],
        "assessment": {"code": "unresolved-map", "confidence": {"level": "not-assessed"}, "trust": "contradictory"}
      }
    }
  ]
}
```

Replay projects revision 1, this observation/review, `lastEventSeq: 1`,
`authority: {"status":"not-declared"}`, `decision: {"state":"pending-review"}`,
and `lastDecision: null`. Scanning against it yields `unchanged`,
`journalRevision: 1`, the saved digest in `journalEvidenceDigest`, and
`lastDecision: null`; no authority or applied outcome is inferred.

## Step 6/7 constraints

- `scan` and `read` are read-only. Every journal write, lock, and temporary
  replacement file is confined to the named journal location, outside both
  canonical snapshot roots and their Git common directories. Recheck this after
  symlink resolution; refuse aliases into either source.
- Serialize journal writers with an exclusive lock and compare
  `expectedJournalRevision` for new request IDs while holding it, after the
  duplicate-ID check above. Write a complete sibling temp
  file, flush it, atomically rename it over the journal, and flush the parent
  directory. A pre-rename failure, conflict, or refusal leaves the old journal
  bytes unchanged. If the directory flush fails **after** rename, report an
  internal write error with an indeterminate commit outcome: the file may
  already contain the new complete version. The caller must read and validate
  the journal revision and request ID before retrying; do not claim rollback
  or blindly append a duplicate event.
- Changed evidence only changes scan comparison output. A record may append the
  new observation without changing the decision. Only `decide` or
  `reconsider` changes an outcome; neither operation edits or fetches source
  repositories.
- Reject unsupported schema versions, unknown fields, and ambiguous/duplicate
  JSON keys rather than guessing. Do not add a source-mutating verb, root
  porcelain, LLM call, remote synchronization, or automatic status transition
  in this contract.
