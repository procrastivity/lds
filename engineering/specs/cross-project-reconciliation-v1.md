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

The pair file is a caller-owned v1 JSON descriptor with `pairId`, `left` and
`right` objects (`projectId`, full Git `revision`), and a declared `scope`.
`scope.leftPaths` and `scope.rightPaths` are the complete repo-relative file
sets to inspect. `scope.candidates` contains unique stable `candidateKey`
values and subsets of those paths for review bucketing. Any scoped path not in a
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

Each evidence/provenance item records `kind` (`artifact`, `missing`, or
`synthetic`), project (or `synthetic`), pinned revision, repository-relative
path or fixture key, SHA-256 where there is a source artifact, the bounded
claim, and a claim-level trust value:
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
does not store a second editable map. Each event has a consecutive `seq`, a
unique caller-supplied `requestId`, `candidateId`, `type`, `actor`, UTC RFC 3339
`at`, non-empty `rationale`, and the operation payload. `revision` is the last
event sequence (zero for a newly created journal). The actor is supplied, never
inferred from the host. Journal reads derive current correspondence, evidence,
authority, and decision state by replaying events; arrays are sorted by
`candidateId`, with history in increasing `seq` order.
The v1 event types are `candidate.recorded`, `evidence.observed`,
`decision.recorded`, and `decision.reconsidered`; unknown event types are not
silently ignored.

All write requests are v1 JSON files with `requestId`,
`expectedJournalRevision`, `actor`, `at`, `rationale`, and operation-specific
fields. Each event retains the validated request fields (including the expected
revision and evidence digest) and the consumed scan identity, so a retry can
be compared with the original operation. For a **new** request ID, writes
require an exact journal revision and `expectedEvidenceDigest` for the
candidate. `record` supplies the `candidateId`, reviewed correspondence,
provenance/trust/confidence, and exact evidence from its scan. `decide` supplies
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

The operation payloads are fixed: `record` requires `correspondence`,
`provenance[]`, and `assessment`; `decide` requires `authority` and `outcome`;
`reconsider` requires `supersedesSeq`, `trigger`, `authority`, and replacement
`outcome`. A `deferred` outcome requires a non-empty trigger ID, trigger kind
(`event`, `date`, or `condition`), and explicit condition/description. The
trigger is never polled or acted on automatically.

| Operation | Effect and refusal boundary |
| --- | --- |
| `journal read` | Read and project the named journal for the named pair. Missing or incompatible files fail; read never initializes them. |
| `journal record` | Append a reviewed correspondence, provenance/trust/confidence assessment, and the exact scan evidence. A first record enters `pending-review`, not pending port work. A repeat scan may append `evidence-observed` while preserving the candidate's existing decision. |
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
verifiable URI, SHA-256, source and destination project/revision, the transferred
scope, verifier identity, verification time, and rationale. The referenced
artifact must describe the actual transfer. A matching source hash, an intent,
a map label, or a journal state is not proof. LDS records the reference and
review; it does not perform or verify the transfer by mutating either source.

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
observations, and—when `--journal` is supplied—journal revision, last decision,
and `evidenceStatus` (`new`, `unchanged`, `changed`, `out-of-scope`, or
`not-compared`). The scanner does not parse semantic map claims, choose trust,
assign confidence, or create/change a journal state.

Journal read emits the same envelope with `journalPath`, `pairId`, current
`revision`, candidates sorted by ID, and append-only events sorted by sequence.
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

## Representative journal JSON

This is a materialized fixture projection for Case 2, not a live pair journal,
authority declaration, or claim of applied runtime behavior. The candidate ID
and evidence digest follow the v1 formulas above using the exact listed Stage 1
pins and hashes.

```json
{
  "schemaVersion": 1,
  "operation": "reconcile journal read",
  "ok": true,
  "journalPath": "/tmp/example-only/journal.json",
  "pairId": "example-stage1-fixture-not-authorized",
  "revision": 1,
  "snapshots": {
    "left": {"projectId": "scind", "revision": "c63cbc686afd2ca6ecc01eab7825d637d49378c7"},
    "right": {"projectId": "xcind", "revision": "c244856e7119547ed6dceb2b7663a01720bdde66"}
  },
  "candidates": [
    {
      "candidateId": "cand-v1-472eb3a0bf207fb4fb79879a0b6f6f8aa48d453dcb77346dd352ff85d556b1b1",
      "candidateKey": "fixture:stage1:case-2",
      "evidenceDigest": "sha256:2f88fd65187e2180dfd6ce2f6eb460dcf3e554e62088440c9a8dea7f484e5ee0",
      "correspondence": {
        "key": "topic:external-proxy-mode",
        "relation": "one-to-one",
        "leftMembers": ["scind:adr:0017"],
        "rightMembers": ["xcind:adr:0022"]
      },
      "provenance": [
        {
          "kind": "artifact",
          "projectId": "scind",
          "revision": "c63cbc686afd2ca6ecc01eab7825d637d49378c7",
          "path": "engineering/decisions/0017-external-proxy-mode.md",
          "sha256": "9557cbfde3718cc23918e20effb76498eb788b1630e93396b49c00fddad1b7cc",
          "claim": "Scind's ADR describes external-proxy mode and cites the Xcind proof-of-concept.",
          "trust": "trusted"
        },
        {
          "kind": "artifact",
          "projectId": "xcind",
          "revision": "c244856e7119547ed6dceb2b7663a01720bdde66",
          "path": "engineering/decisions/0022-external-proxy-mode.md",
          "sha256": "db5d802fb20b687829d46e5438334f00c33b3821ca4cb7cff3c3ef2d51377723",
          "claim": "Xcind's ADR documents its external-proxy decision.",
          "trust": "trusted"
        },
        {
          "kind": "artifact",
          "projectId": "xcind",
          "revision": "c244856e7119547ed6dceb2b7663a01720bdde66",
          "path": "engineering/sync/artifacts/correspondence-map.md",
          "sha256": "3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05",
          "claim": "The file row says XCIND-ONLY while the topic table says MATCH.",
          "trust": "contradictory"
        },
        {
          "kind": "artifact",
          "projectId": "xcind",
          "revision": "c244856e7119547ed6dceb2b7663a01720bdde66",
          "path": "engineering/sync/artifacts/correspondence-map.json",
          "sha256": "175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17",
          "claim": "The file-level JSON row says scind is null and rel is XCIND-ONLY.",
          "trust": "contradictory"
        }
      ],
      "assessment": {
        "code": "unresolved-map",
        "confidence": {"level": "not-assessed"},
        "trust": "contradictory"
      },
      "authority": {"status": "not-declared"},
      "decision": {"state": "pending-review"},
      "lastEvent": {
        "seq": 1,
        "requestId": "fixture-record-case-2",
        "actor": "fixture-only",
        "at": "2026-10-01T00:00:00Z",
        "rationale": "Fixture only: the map claims conflict; no direction or applied outcome is authorized."
      }
    }
  ]
}
```

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
