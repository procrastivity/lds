# V1 contract: reviewed existing-file documentation updates

**Status: deterministic steps and offline human-facing root host implemented
and verified in disposable offline fixtures.** The
owner requested `lds plumbing docs update-plan` followed by
`lds plumbing docs update-apply`, and subsequently authorized `lds update`
as an interactive host of the same steps. This contract grants no authority
to run either route on a real target. No real target, reviewer or pilot is
authorized. It is separate from the
[accepted assessment workflow](documentation-maintenance-workflow.md) and
[create-only adoption](adoption-contract.md). Adoption's four
outputs, collision veto, facts approval and `docs plan`/`docs apply` remain
unchanged. A generated-looking file has no special ownership status.

## Recommendation and scope

Offer a separate deterministic, Linux-only **plan → fresh human approval →
apply → actual-effects result** path for exact replacements of explicitly
selected, *existing* regular UTF-8 Markdown files. Candidate paths are only
root `README.md` and `.md` files below `docs/` or `engineering/`, in the
explicit target root. Selection is per file, never inferred from inventory,
assessment proposals, a template signature, or an adoption plan. No creates,
missing parents, moves, renames, deletions, broad crawl, project commands,
provider calls, or implicit expansion to cited source files. A maintainer can
always continue editing manually instead. Model transfer consent permits a
request, not a write; acceptance of assessment advice is not edit approval.

The canonical plumbing names are `docs update-plan` and `docs update-apply`
(not aliases for adoption apply). The root offline host below adds no model
or shared skill flow; model-assisted update orchestration remains deferred.
The caller supplies the original **collected**
`docs collect` observation packet and an explicit edit request. The packet's
empty proposals/questions and `reportPath` distinguish it from a later authored
assessment report. A report may motivate an edit, but is not an executable
plan. The request names a nonempty set of unique selected paths, the exact
proposed after bytes, a nonempty per-file rationale and cited paths within the
collected evidence. Evidence claims remain reviewer assertions, not verified
semantic facts. The caller must inspect any sensitive content before sharing
the plan; no extra repository files are read because a citation suggests them.

The command forms are:

```text
lds plumbing docs update-plan <root> --observed <collected.json> --request <edits.json> --json
lds plumbing docs update-apply <root> --plan <reviewed.json> --confirm <sha256:digest> --json
```

The request is strict UTF-8 schema-1 JSON with exactly `schemaVersion` and
`edits`; each edit has exactly `path`, `after`, `rationale`, and `citations`.
Unknown, duplicate, case-aliased, missing or null fields refuse. At most 32
edits are allowed; each selected file and after content is at most 8 MiB, and
each input JSON at most 32 MiB. The original packet is produced by
`docs collect`, not an authored assessment report. See the
[disposable how-to](../../docs/how-to/update-documentation.md).

## Offline human-facing host

The first root slice is `lds update <root> --observed <collected.json>
--request <edits.json> [--json]` on Linux with a terminal. It reuses the
original collected packet and explicit schema-1 authored request, not an
assessment report or inferred proposal, and does not collect a new baseline.
It builds the identical deterministic plan in memory and displays the
**complete** plan on stderr. It refuses a veto before prompting, flushes
queued terminal input after display, and accepts only freshly typed
`apply sha256:<exact displayed digest>` or categorical `decline`; every other
input or EOF refuses. There is no `--confirm`/`--yes` bypass. On affirmation,
it invokes the same strict in-memory apply; no temporary plan/report artifact,
provider/native execution, credential, or project command is involved.
With `--json`, final stdout is schema-1 JSON with `operation: "update"`,
`status`, complete `plan` and `apply` (null on veto/decline); otherwise it is
a human status summary. Display, prompt and errors are on stderr.
The original staleness, per-file, partial and indeterminate contracts
below still govern actual effects and recovery. The manual separately reviewed
plumbing route remains canonical; root is not a same-deliverable alias for
either step. It is root-only plumbing-kind, not projected, since only qualified
`plumbing ...` entries project. There is no new LLM/shared flow asset or
preparatory reader for this offline host.

## Review artifact and read-only plan

The read-only planner returns its artifact on stdout only; if a host persists it,
the destination must be an explicitly named new file **outside** the target,
created exclusively with no-follow existing trusted private parents, never a
target-side scratch or an occupied path. An external artifact write failure may leave a
partial external file and conveys no target-write permission.

Return one version-1 JSON artifact with `operation: "docs update-plan"`,
`targetRoot`, `targetIdentity`, the **complete original collected observation
packet**, and sorted edit entries. Each entry includes relative path, rationale,
citations, exact full `before` and `after` UTF-8 content (not a context diff),
byte lengths and SHA-256 hashes of both, and observed file identity (`dev:ino`),
link count, UID/GID, permission mode and metadata eligibility checks. Include
selected path and ancestor identities, veto, findings and skips (including
original inventory/validation diagnostics).
`before` contains `content`, `sha256`, `bytes`, `identity`, `mode` (numeric
permission bits), `uid`, `gid`, `links`, and `metadata`. After content is the
`after` string, with `afterSha256` and `afterBytes`. Each edit also binds
`afterObservationSha256`: SHA-256 of the compact JSON observation with only
that entry and earlier sorted entries overlaid, retaining the original brief.
It is a bounded inventory/link/evidence checkpoint, not semantic verification.
Apply reads actual bytes without an overlay and compares them to that checkpoint.
The packet retains reason/audience, inventory, validation, their skips and
captured source content; its original brief hash and observation fields are
not reduced to a list of selected hashes. All displayed review metadata is
part of the artifact. Sort arrays deterministically; reject duplicate, null,
unknown or case-aliased fields and invalid hashes. `planDigest` is `sha256:`
plus SHA-256 of RFC 8785 canonical JSON of the entire *displayed* artifact
except `planDigest`, as in [adoption's full-plan digest](adoption-operations-contract.md#reviewed-plan-and-application).
Check the raw JSON on apply as well as its typed form so decoding cannot
silently normalize a differently displayed field. A vetoed or incomplete plan
is reviewable but ineligible for apply. A plan is a snapshot, not a reservation
or durable authority over any file.

The planner validates the collected packet through `RecheckAssessment` against
the original root and its **complete** observations, rather than accepting a
freshly recollected packet as the old one. Selected files must occur as existing
captured evidence in that packet; require their exact before bytes to match
the held no-follow file descriptors. Refuse equal before/after bytes, invalid
UTF-8, a missing/unreadable/nonregular target, changed or absent cited evidence,
or an unsafe path. Include the original local-link findings and skips as
diagnostics, not a promise of semantic correctness or a veto solely for a
broken link. The plan must not recalculate evidence and quietly bless a new
baseline. It may report an after-edit link preview, but that must be labeled
as bounded/unchecked and digest-bound if displayed.

## Path, platform and replacement boundary

Require an explicit existing root on Linux. Accept clean slash-relative paths
with no `.`/`..`, absolute/drive/backslash/control components or escapes; apply
the Markdown scope above, not a filename similarity test. Open root from `/`
component by component and open each ancestor and selected file descriptor-
relative with `O_NOFOLLOW`; verify directories and regular files with `fstat`.
Reject symlinked root ancestors, in-tree ancestors and selected leaves; do not
follow or replace a symlink as an eligible target. Require all selected files
to have link count **one**, both at plan and at apply: replacement of a
multi-linked inode would silently diverge its aliases. Refuse special mode
bits and nonregular files. Capture `dev:ino`, mode and link count for each leaf
and `dev:ino` for the root and each existing selected-path ancestor; require
them unchanged along with exact bytes at recheck. Never create a target parent.

Keep the existing file's permission bits (`0o777`) exactly by setting the
new staged inode's mode explicitly, not relying on umask. Require the original
file's UID/GID to remain unchanged and the staged inode to match both before
rename; do not change ownership to force eligibility. Refuse extended ACLs or
other xattrs on selected files, and default ACLs on staging parents. Require
the staged inode to have no extended ACLs/xattrs too. If these checks are
unsupported or unreadable, refuse rather than assume absence. Capture the
original UID/GID and these eligibility observations in the plan and recheck
them before replacement. This first slice preserves ownership and ordinary
permission bits; it does not migrate richer metadata.

Replacing an inode does **not** preserve inode identity, timestamps or
open-handle behavior; use manual/project-specific editing where these are
required. Do not claim the assessment collector's file hash covers metadata:
this plan must observe it separately. Linux support assumes an appropriate
local filesystem with descriptor-relative no-follow open, same-directory staging and atomic
single-directory-entry rename semantics. Other platforms refuse update apply;
a successful cross-build is not platform support.

After exact plan-digest confirmation, revalidate the raw artifact, original
packet and **all original relevant observations** with `RecheckAssessment`
before any target write: root identity, full inventory/validation results and
skips, original brief/evidence/source bytes, selected contents and metadata,
and selected ancestor identities. Refuse if any differ, even if the requested
after bytes are unchanged. Never silently replan or reuse approval for a new
baseline. Require a fresh affirmative human confirmation of this particular
displayed digest at the apply boundary; a CLI can compare strings but cannot
authenticate the person. A declined, mismatched or vetoed approval writes
nothing. Confirming an adoption plan, consenting to model transfer or accepting
an assessment cannot substitute for this confirmation.

Apply in stable path order. Before each replacement, verify the still-pending
original files/evidence and earlier completed files against their expected
after bytes, as well as the visible root/ancestors and the current selected
leaf against its original identity, link count, mode and bytes. The complete
original assessment cannot simply be rerun after the first successful update:
that update itself changes its inventory. Subsequent checks must compare the
remaining original evidence to the original packet with an explicit overlay
for *only* previously completed entries, including expected inventory/link
diagnostics recomputed against that overlay. Changes outside those entries
are stale, not a new baseline. Stage full after bytes at a unique unpredictable
unused name in the same verified parent directory using descriptor-relative `O_CREAT|O_EXCL|
O_NOFOLLOW` (0600 initially), write/verify length and hash, set the reviewed
permission bits, verify UID/GID and metadata eligibility, sync and close. Recheck visible root/ancestors and leaf just
before `renameat` replaces **that one** selected leaf; never stage in an
external directory and never pass a path to adoption's create API. Verify
the visible post-rename bytes, mode, UID/GID and metadata eligibility and sync
the parent. This is a best-effort
stale check and per-file replacement, **not** an atomic multi-file transaction
or a compare-and-swap against hostile writers. A writer can race the last
check and rename; descriptors prevent following symlinks but not all path
substitution attacks. Require trusted, quiescent root ancestors, selected
parents and other writers during plan/review/apply; stop if that cannot be
arranged. Concurrent LDS applies to overlapping paths are unsupported.

## Result and recovery

Return a structured version-1 `docs update-apply` actual-effects result with
`planDigest`, `status` (`applied`, `refused`, `partial` or `indeterminate`),
`replaced` (path and observed after hash/mode), `untouched` (path and observed
before hash/mode), `unattempted`, and `failed` (path, phase, stable code,
observed hash/mode/identity when readable). Distinguish a pre-write refusal
with zero target effects from a staged-but-not-renamed failure, verified
replacement, and post-rename verification/sync uncertainty. Do not call a
possibly replaced file preserved merely because its post-write read failed.
Refuse pending paths after the first failure. No cross-file atomicity, rollback,
or safe automatic retry: an error may leave some selected files replaced and
others original; a crash can leave an exclusive scratch file. Cleanup may
unlink **only** an owned scratch name whose identity is still verified, never
an occupied/foreign path or the target; record cleanup failure and residual
scratch path. Do not automatically delete a completed replacement. On
uncertain post-rename or crash outcome, inspect actual target and scratch
state and start a *new* plan/review against the observed baseline before any
further attempt. If a result itself cannot be emitted, the caller must inspect
the tree rather than assume nothing happened. No durable journal is implied.
`unattempted` is an array of path strings. `failed.observed` is an observation
object or null; `scratch` and `cleanupError` are empty strings when absent.

## Previously executed deterministic plumbing checks

`make check` passed with zero lint issues and all Go packages passing;
`go test -race -count=1 ./...` passed. Linux/amd64 and Darwin/arm64 static
builds passed; Darwin update support is not claimed. The documented disposable
recipe ran through the real CLI: decline exited 3 without target changes;
confirmed apply exited 0 with exact bytes, ownership and permission mode.
An independent JSON/SHA-256 calculation matched the displayed plan digest.
Real filesystem controls refused xattrs, parent default ACLs, hardlinks,
special modes and symlink leaves without writes. Unique edit-count boundaries
accepted 32 and refused 33. Regression tests also cover changed source,
inventory/link diagnostics, selected bytes/identity/mode/ancestors, strict raw
JSON, completed-edit overlays, partial/uncertain effects and foreign scratch
preservation. Fault-injected failure tests are not evidence of hostile-writer
immunity. No provider, native harness or real-project update ran. These checks
predate the root host.

## Executed offline root-host checks

The documented preparation and root command ran against the built CLI in
disposable Linux terminals: fresh approval replaced exactly `README.md` with
the requested bytes; categorical decline exited 3 without writes. The full
display matched the plumbing plan, and an independent JSON/SHA-256 calculation
matched its digest. No provider, native harness or real-project update ran.

Terminal regressions cover queued approval being discarded, EOF, wrong and
bare digests, a veto without a prompt, stale evidence and changed request
files after display (only the displayed bytes apply). Noninteractive calls
refuse; neither `--yes` nor `--confirm` exists. The strict raw-artifact refusal
cases also cover the in-memory entry point. The manifest has 28 leaves,
19 qualified projected plumbing entries and nine docs leaves.

`make check` passed with zero lint issues; the full race suite passed on rerun.
The first race run failed while snapshotting a disappearing Git
`objects/maintenance.lock` in a reconciliation fixture, not with a race report.
Linux/amd64 and Darwin/arm64 static builds passed; this does not add Darwin
update support. All 80 local links and five shell blocks in the nine updated
documents passed link and shell-syntax checks.

## Implementation separation and adversarial acceptance

Reuse `CollectAssessment`/`RecheckAssessment` for bounded original evidence,
path validation/no-follow root and descriptor opens where suitable, SHA-256
and strict JSON/canonical-digest machinery from adoption. `AssessReviewed`
still creates only an external report and is not approval. Adoption
`Plan`/`ApplyReviewed` and Linux `createExclusive` are create-only; replacement,
metadata checks, same-parent staging/rename, overlay rechecks and actual-effects
reporting are genuinely new operations. Avoid retrofitting replacement into
the four-output adoption contract.

| Independent scenario | Required distinguishable outcome |
| --- | --- |
| One explicitly selected existing UTF-8 doc, complete evidence, fresh matching digest | Exactly that leaf is replaced with reviewed bytes and permission bits; unrelated files remain unchanged. |
| Edited adoption-generated-looking file not selected, or a selected generated-looking file | First is untouched; second is eligible **only** on explicit selection and fresh approval, never due to its header. |
| Missing target, new output, move/delete request, or path outside allowed Markdown scope | Plan veto; no target write, not a create or migration. |
| Selected symlink, symlink ancestor/root, nonregular leaf or hardlink count above one | Unsafe-target refusal; no target write; aliases and external symlink destination unchanged. |
| Extended ACL/xattr, default parent ACL, unreadable metadata or staged UID/GID mismatch | Refuse without replacing the target; never silently discard metadata or change ownership. |
| Declined digest, assessment acceptance alone, model send consent alone, tampered raw JSON | Distinct approval/invalid-plan refusal before staging; no target write. |
| Selected bytes, metadata, ancestor identity, cited source, inventory, local-link finding or skip changes after review | Stale refusal before first write; retain old artifact, require new review. |
| Late occupied scratch name versus an unrelated occupied doc | Exclusive staging collision only affects that attempt; never overwrite occupied scratch or unrelated doc. |
| Failure writing scratch before rename versus failure after first rename in a two-file set | Former: no replacement, report owned scratch/cleanup; latter: report verified first replacement, failed/uncertain second and unattempted remainder, no rollback. |
| Hostile writer swaps leaf immediately after last check | Explicit concurrency limitation, not a claimed successful CAS guarantee; do not assert tests can prove immunity. |

The implemented contract settles the bounded engineering choices: Linux-only,
existing selected Markdown, no hardlinks, exact-content artifact, same-parent
staging, ownership/mode preservation with richer metadata refused, stable
ordering and explicit partial results. **Owner decisions still needed before
real-target use:** which real project/paths and reviewer may be authorized,
and whether inode/timestamp replacement is acceptable for that target.
Release/pilot claims need separate authorization. None is supplied by this
implementation request.
