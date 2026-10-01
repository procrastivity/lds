# BDS-252 Stage 4 Step 9 — engineering audit and handoff

**Status:** The deterministic reconciliation v1 contract is implemented and
bounded Linux runtime behavior is recorded for disposable fixtures. The
Step 8 implementation is independently verified and sealed on root local
`main` `2bfd96a71a20540f4c0fe0c470d3f3a8bb4589ae`, which remains unpushed
relative to `origin/main` `3c9af08c10d2eb8c004ec5ca1a674f4e2632cb4e`. This
handoff summarizes that boundary; it does not extend the frozen contract,
choose real-pair authority, or authorize source writes.

The accepted [ADR-0003](../decisions/0003-bound-cross-project-reconciliation.md)
and [v1 operation contract](../specs/cross-project-reconciliation-v1.md)
define the behavior. Their pinned hashes remain respectively
`fee5a9862f2bf896fee233d949fd082b3ccb669afe373ac069ae2f386eb5cd92` and
`f721bc58a7c6fd8eb3ff0ac87f3708c1bb752a4f6a2b6bb6264fafb5d4f3f587`. The
[Stage 1 evidence](bds-252-stage-1-evidence.md),
[four expected-case cards](bds-252-stage-1-expected-case-cards.md), and
[Stage 2 independent fixture expectations](bds-252-stage-2-fixture-expectations.md)
remain unchanged. This document is the current implementation/status summary;
the frozen BDS-250 adoption contract retains its earlier roadmap-status
paragraph as historical text. Its accepted adoption behavior is unchanged;
this handoff and the current roadmap supersede that paragraph only for
implementation status.

## Implemented deterministic surface

The actual manifest has 18 command leaves, of which 13 are qualified
`plumbing ...` entries with plumbing-kind projection. Bare `lds --help` keeps
the human/root entries `version`, `install`, and `uninstall`, the existing
root aliases `doctor` and `manifest`, and the `plumbing` group; it does not
flatten reconciliation leaves into root commands. `lds plumbing reconcile
--help` lists `scan` and `journal`; `lds plumbing reconcile journal --help`
lists exactly `read`, `record`, `decide`, and `reconsider`. These observations
were checked against the built CLI and `lds plumbing manifest --json`.
Projection eligibility is a manifest property, not a claim that an
LLM-assisted or BDS-251 adoption workflow was delivered.

All paths below are caller-supplied absolute paths; there are no implicit
project roots, network fetches, or journal defaults:

```text
lds plumbing reconcile scan --pair <pair.json> --left-root <dir> \
  --right-root <dir> --map <tracked-file> [--map <tracked-file> ...] \
  [--journal <existing-journal.json>]

lds plumbing reconcile journal read --pair <pair.json> --journal <journal.json>
lds plumbing reconcile journal record --pair <pair.json> --journal <journal.json> \
  --scan <exact-saved-scan.json> --request <record-request.json>
lds plumbing reconcile journal decide --pair <pair.json> --journal <journal.json> \
  --scan <exact-saved-scan.json> --request <decision-request.json>
lds plumbing reconcile journal reconsider --pair <pair.json> --journal <journal.json> \
  --scan <fresh-scan.json> --request <reconsider-request.json>
```

`scan` accepts a strict v1 pair descriptor: `schemaVersion: 1`, stable pair
ID, ordered project IDs and full revision pins, complete scoped path sets, and
non-overlapping candidate buckets. Both roots must be clean Git snapshots at
the exact pins. Only declared tracked files and tracked raw `--map` files are
read. Unsafe/duplicate paths, maps, JSON keys, unknown fields, or scope
overlaps fail closed. The scanner does not fetch, checkout, parse map claims,
or infer semantic matches, confidence, trust, authority, or direction.

Candidate IDs are stable functions of pair ID and candidate key, separate
from evidence digests. Digests include the exact pinned artifact/missing
rows and every supplied map hash for every candidate; changing bytes or pins
changes evidence while preserving candidate identity. Candidate and evidence
arrays have specified stable ordering. Without `--journal`, scan emits
`not-compared` and omits comparison metadata; with one, each candidate is
`new`, `unchanged`, `changed`, or `out-of-scope` as specified by the contract.
Success always emits one v1 JSON object. Verb-raised failures use the inherited
root `--json` error behavior; Cobra usage errors remain human diagnostics.
Exit 0 is success, 1 invalid input/evidence, 2 usage, 3 safety/staleness/
conflict refusal, and 4 unexpected I/O/internal failure.

`scan` and `journal read` never initialize or mutate a journal. The first
explicit `record` may create it. Journal requests carry actor, UTC time,
rationale, expected revision and evidence digest, reviewed correspondence
and provenance, or an explicit authority/outcome. Events preserve the full
validated request and exact consumed scan-byte identity. A retry succeeds as
replayed only when request fields and scan identity match; duplicate request
IDs are checked before current revision/evidence comparisons. New requests
refuse stale pair, revision, or digest. `record` can observe changed evidence
without erasing an earlier decision. Only `decide` records an initial outcome;
`reconsider` explicitly names the latest decision it supersedes and a trigger,
retaining event history. Changed bytes never trigger a decision transition by
themselves; scan comparison status may change.

Journal, lock, and replacement writes are restricted to the caller-named
journal location, outside both snapshot roots and Git common directories.
Writes serialize under an exclusive lock and replace the complete journal
atomically. Pre-rename failures preserve old journal bytes; a post-rename
directory-sync failure has an indeterminate outcome, so recovery reads the
journal revision/request ID before retrying. No command writes either source
repository, derives an outcome automatically, or executes a port/backport.
`appliedProof` is a caller-supplied external reference whose shape is checked;
its URI, verifier, transfer, and claims are not fetched or authenticated by
LDS.

## Evidence and interpretation limits

The [Stage 4 runtime round](bds-252-stage-4-runtime-round.md) and
[machine-readable evidence](bds-252-stage-4-runtime-evidence.json) record
34 CLI invocations. The evidence reports all expected exits and state checks
passing, all 34 source/index snapshots preserved, and byte-identical repeated
baseline scans. It records the four independently expected source-pair
candidate IDs and evidence digests, the synthetic split ID, scan/map changes,
journal retries, stale/conflict/proof refusals, and decision preservation.
The Step 8 high-mode verification additionally rebuilt a byte-identical
binary and independently replayed all 34 calls, recomputing candidate IDs,
evidence digests, raw input and journal hashes. The recorded binary SHA-256 is
`43871ca3d5ba59f827efea520ec0f0af8e40a9d6208fda989fec4d11a417696c`; the
runtime-evidence JSON itself is
`0cd83021014c6ea608779f952064e97a2fcc5307fcdd84a93cd858b6bde9929d`.

The four Stage 1 cases remain bounded evidence examples, not LDS decisions:

| Case | What the evidence supports | What it does not establish |
| --- | --- | --- |
| Scind ADR-0011 / Xcind ADR-0023 | Same topic; opposite targeting choices; a 1:1 review candidate. | No universal authority or chosen port direction. Scind runtime remains unverified. |
| Scind ADR-0017 / Xcind ADR-0022 | Cited documents credit the promotion as already applied. The Xcind topic row says `MATCH`, while its file/JSON rows say `XCIND-ONLY`/null; retain the relationship as unresolved. | Not a new pending port; no independently verified runtime parity or BDS-119 repair claim. |
| TCP/SNI | Scind marks the capability Future; Xcind records rejection and a revisit trigger if Scind promotes it to active design. | Not a shipped Scind gap or automatic divergence decision. |
| Route decomposition | A disposable fixture exercises one-to-many `split` correspondence. | It is synthetic, not a Scind/Xcind fact or real map entry. |

This run is a bounded deterministic evidence scan, not a complete semantic
reconciliation of either project. It exercised fixture authority declarations
and refusal of an applied request with no proof, not a successful applied-proof
operation or authenticated real-world approval. Linux/amd64 is the only
runtime-verified platform; Darwin/arm64 is build-only and Windows is
unsupported/not claimed. There is no live pilot, completed port, source parity
claim, LLM facilitator, real-world adoption, BDS-119 repair, or BDS-251 workflow
delivery. The preserved upstream vendor whitespace noted in the runtime-round
report remains unchanged.

## Deferred owner/product choices

ADR-0003 D1A–D5A and the v1 contract settle the deterministic interface and
journal boundary, not the values for any real pair. Before any live use, an
authorized owner must select a pilot/pair, identify who may declare
pair/topic-specific authority and allowed directions, and decide what external
verification is sufficient for an applied claim. A human-facing reconciliation
facilitator or additional skill/porcelain surface is future product work, not
part of this implementation. Adoption pilot, approval, partial-adoption, and
LLM choices in the roadmap remain separately open.
