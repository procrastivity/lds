# Human-facing cross-project reconciliation — offline and assisted report slices

**Status:** The Linux-only offline and explicit consent-gated assisted report,
deterministic collect/extended report, live preparatory flow reader and root-only
guided journal record/decide/reconsider slices are implemented. This is not
authorization for a real pair,
pilot, provider call, native integration, CLI/skill parity, release, source edit
or real-pair journal authority. This is separate from the
[within-project existing-file maintenance workflow](documentation-maintenance-workflow.md).
The owner authorized offline implementation separately from these later slices.

The accepted [ADR-0003](../decisions/0003-bound-cross-project-reconciliation.md),
frozen [v1 operation contract](../specs/cross-project-reconciliation-v1.md),
and [BDS-252 engineering handoff](bds-252-stage-4-engineering-handoff.md)
remain authoritative for the implemented deterministic surface. The
[current how-to](../../docs/how-to/reconcile-projects.md) describes both report
modes and external-artifact plumbing, not a full facilitator. The Stage 4 fixtures are bounded
evidence, not a real-pair authority choice or a completed reconciliation.

## Offline report and non-goals

The **report-only first slice** delivers one private, caller-owned external
Markdown report combining the current scan comparison, any explicitly reviewed
correspondence, contradictory claims, unresolved authority and questions.
It is useful offline, with no model: a reviewer can supply annotations
and receive a traceable review packet without recording a decision. Optional
model assistance drafts *advice* only after a separate, scoped content
disclosure and fresh consent. This report causes no source or journal writes,
including initialization, locks, or request staging in either root. A report is
not a journal event, approval, port plan, applied proof, or runtime-parity finding.

Implemented offline shape (also available as `lds plumbing reconcile report`):

```text
lds reconcile report --pair <absolute-pair.json> --left-root <absolute-dir> \
  --right-root <absolute-dir> --map <absolute-tracked-file> [--map ...] \
  [--journal <absolute-existing-journal.json>] \
  --review <absolute-review.json> --output <absolute-new-private-report.md> \
  [--json]
```

Without `--assist`, this slice makes no provider call. The same required flags
with explicit `--assist` use a Linux terminal and `OPENCODE_API_KEY`, display
the complete system/user payload and provider destination, flush queued input
and require fresh exact `send`. The root report registration is LLM-kind, not
an alias or projected member. One fixed OpenCode Go / `deepseek-v4-flash`
request has no tools, redirects, retries or fallback; only proposed advice
may be added to the private report. `lds plumbing reconcile collect` emits a
bounded original evidence JSON without publishing; extended deterministic
`plumbing reconcile report --observed --draft --receipt` requires all three
external artifacts and binds them to that original baseline. The live
`lds plumbing reconcile flow --json` reader is preparatory/unavailable for
execution. Generated guidance fetches it live, discloses its own harness
processing and obtains its own consent; it must not automate the root terminal.
Today's `lds plumbing reconcile scan` and `journal read` are read-only;
`journal record`, `decide`, and `reconsider` alone write the explicitly named
external journal. `reconcile.Report` owns bounded collection, strict review,
recheck and publication. Qualified plumbing entries project into the skill
table; neither native skill execution nor CLI/skill parity is established.
A flow or skill cannot enforce boundaries by prompt alone. Report uses existing
scan identity and journal replay. No implicit cwd, remote, journal, pair, map, or
output defaults; no fetch, checkout, project command, source transfer, or
auto-generated source patch.

The implemented review brief has required `schemaVersion: 1`, `purpose`,
asserted `reviewer` and RFC3339 `at`, and non-null `selections`, `claims`,
`questions` arrays. Selections are `{side, path}` with side `left` or `right`.
Claims contain `candidateId`, `status` (`reviewed`/`proposed`), v1
`correspondence`, `rationale`, nonempty v1 `provenance`, v1 `assessment`, and
non-null `conflicts`/`questions` arrays. Each citation must match a captured
current candidate evidence row exactly; synthetic or historical citations
cannot assert a new current relationship. Saved journal claims retain their
own historical provenance unchanged. An empty brief is valid apart from the
required purpose, reviewer and timestamp. See the [runnable example](../../docs/how-to/reconcile-projects.md#create-an-offline-review-report).

## Inputs, budget, and trust

- The caller names two ordered **clean pinned Git worktree roots** and a strict
  v1 pair descriptor containing pair ID, project IDs, full revisions, complete
  scoped paths and non-overlapping candidate buckets. At least one explicit
  tracked map file belongs to exactly one pinned root. An optional journal
  must already exist, match pair/project identity, and be outside both roots
  and Git common directories. Missing explicit journal is a refusal, not a
  request to create one. Pair and review files and output must be external to
  both roots/Git directories too. The host checks canonical paths and no-follow
  output creation; report mode 0600, no overwrite or parent creation. Require
  trusted, quiescent ancestors; do not claim arbitrary hostile rename-race
  protection.
- Recommend a strict versioned review brief with question/purpose, reviewer
  identity and time **supplied, not authenticated**, bounded per-candidate
  assertions, and optional content selection. Asserted correspondence uses
  explicit stable project-local member keys and v1 relations (1:1, unmatched,
  split, merged), not filenames or candidate buckets as semantic matches.
  Each assertion says `reviewed` or `proposed`, identifies reviewer/rationale,
  cited side/path/revision/hash and claim-level trust, qualitative confidence
  with rationale if assessed, conflicts and open questions. The host validates
  shape, cardinality, scope, unique member use, citations against captured
  evidence when applicable, and rejects unknown/duplicate fields; it cannot
  authenticate the reviewer or truth of the assertion. An empty review is
  valid: report all relationships as unreviewed. A saved journal's reviewed
  correspondence and decisions are displayed as *journal claims*, with their
  event and observation provenance, not silently replaced by brief assertions.
  Conflicting brief and journal assertions remain side by side for review.
- Set an explicit read/disclosure budget before collection: recommended maximum
  64 declared scoped files plus at most 16 maps, 64 KiB raw bytes per file
  (including unselected files), 512 KiB total selected
  UTF-8 content considered for review, and a separately declared 2 MiB combined
  ceiling for pair/review/journal inputs. These are report limits, not changes
  to v1 scan limits. Preflight known sizes and enforce while reading;
  refuse oversize/binary content needed for interpretation rather than
  truncating and claiming full review. Deterministic scan still hashes each
  declared raw artifact; content inclusion is an additional opt-in list by
  side and path, not an instruction to crawl. If content is not selected, the
  report can compare hashes/presence and label semantic claims unsupported by
  content. Display exactly which files, byte counts, hashes, maps and journal
  fields enter the report and, separately, which selected full file bytes
  would leave the host. The host adds no environment variables, credentials,
  absolute local paths or unselected repository bytes to the model packet.
  Selected content may itself contain secrets or absolute paths; it is not
  automatically redacted. The caller must inspect it and remove sensitive
  selections before consenting. No network call in offline mode.
- Pair, map, source text, journal, review brief and model response are
  **evidence/data**, never executable instructions or universal authority.
  Map labels such as `MATCH` and `XCIND-ONLY` may contradict; neither decides
  correspondence, direction, or applied status. Byte equality proves only
  equality. A journal replay validates structure and lifecycle, not actor
  identity, provenance truth, external proof URI, or the transfer it alleges.

The host establishes scope and pins, invokes scan with the optional journal,
reads the journal projection when supplied, validates the brief and creates
the private report. It must enforce paths, budget, no-write boundary,
revalidation and consent independent of any reviewer. The human reviewer
selects the pair/scope and disclosure, examines source and contradictory
claims, explicitly reviews correspondence and trust, names the pair/topic
owner for unresolved authority, and decides what follow-up evidence is needed.
An assisted reviewer may propose text, but cannot approve for that human.

## Guided offline journal slice

The separate root-only `lds reconcile journal record|decide|reconsider` Linux
hosts take absolute `--pair`, `--left-root`, `--right-root`, repeatable tracked
raw `--map` and external `--journal` paths, plus optional `--json`. The clean
Git roots must match the pair's full pins. No authored scan or request JSON is
needed: the host freshly scans and presents candidate evidence and prior
decisions, then constructs the same strict v1 requests for the existing
record/decide/reconsider plumbing. The manual saved-scan/request-file route
remains available, unchanged. The three journal leaves are root-only
plumbing-kind and not projected; they do not run the report flow. The manifest
now has 35 leaves, 22 qualified projected plumbing entries and nine docs
leaves.

The operator explicitly supplies plain-text correspondence with stable
member keys and relation, provenance claims with trust, assessment, actor and
rationale for a record; reconsider may replace the reviewed triple only as a
complete optional group. For decide/reconsider the operator declares scoped
owner authority and permitted directions and selects an outcome; reconsider
also names a trigger and the latest decision being superseded. `unresolved`,
`deferred` (with trigger), `divergence-retained`, `intent` (permitted direction)
and `applied` (permitted direction and externally authored proof reference with
URI, hash, source/destination revisions and paths, verifier, time and rationale)
are not inferred from map labels, matching names or bytes. An applied proof
reference is recorded as an unverified assertion, not authenticated or acted
on. No host edits either source, executes a transfer, calls a model/network or
runs project commands beyond the read-only Git scan.

The terminal presents the **full** digest-bound request/evidence/journal
preview and prompts on stderr even with `--json`; JSON success goes to stdout.
After flushing queued input, only a freshly typed `record sha256:<digest>`,
`decide sha256:<digest>` or `reconsider sha256:<digest>` for the exact preview
permits its write; `decline` refuses. Preview or decline creates no journal,
lock or output file. Pair/source/map/journal are checked afresh before the
write; drift refuses without automatic retry. These checks are not an atomic
snapshot. After an indeterminate write, read the journal revision and request
ID before another attempt. A real pair, authority, proof-verification policy
or pilot remains a separate owner decision. See the
[guided how-to](../../docs/how-to/reconcile-projects.md#guide-a-journal-entry-in-the-terminal)
and [safety notes](../../docs/explanation/safety-and-scope.md).

## Report contract and baseline

Recommend one report with a machine-readable provenance appendix, not a
second editable mapping database. Its first page states **review required**,
scope/exclusions, pair/project pins, report creation time and output location;
then a table by stable candidate ID/key with current left/right paths,
presence/hashes, map hashes, `not-compared`/`new`/`unchanged`/`changed`/
`out-of-scope`, observed digest, journal observed digest, journal revision,
and last decision's sequence/digest/outcome where present. Display current
observation and last decision separately even if the scan is `unchanged` to
the latest observation but the last decision was made against an older digest.
An out-of-scope journal-only candidate is **history**, with no current digest
or current scoped content; preserve its decision without asserting current
absence or deleting it. No-journal results have no journal comparison.

For each candidate distinguish (1) raw observations, (2) saved reviewed
correspondence, authority and outcome with exact journal event citations,
(3) newly reviewed or merely proposed brief/model relationships and their
claim-level provenance, (4) conflicts, missing evidence and unreviewed
members, and (5) unresolved owner/authority/direction and questions. Do not
copy a journal `pending-review` into a port intent. An `intent` remains intent,
even with matching bytes; `applied` is only an attributed *journal assertion*
with its external proof reference and verification limits, never an inference
from bytes, map claims or report advice. Preserve `deferred` triggers and
`divergence-retained` rationale; do not reopen/reclassify them on scan changes.
Report whether citations refer to earlier pins rather than silently treating
them as current. Call model claims proposed, not reviewed, unless an explicit
human review subsequently supplies that status in a new report run.

Bind the report to canonical input paths and raw SHA-256 of pair, review,
selected content, maps, optional journal and exact serialized scan/read
responses; include ordered pins, candidate IDs, per-candidate evidence digests,
journal revision and relevant last-event/last-decision sequences, collector
version, coverage/exclusions and disclosure receipt (none offline). The scan
digest binds evidence rows and pins, **not** correspondence, direction or
applied proof. A candidate ID binds pair ID/key, not changing evidence.
Hashes/paths bind observed bytes, not authorship. Saved scans used later are
checked by v1 for pair identity, pins, scope and digests, but are not a fresh
authenticated observation of current roots or of semantic claims.

Before publishing, recollect/revalidate pair bytes, clean root HEAD/index,
declared source/map hashes and optional journal bytes/revision against the
original scan; re-read review bytes and consent/disclosure selection, and
reject any drift without silently substituting a new baseline. If a model was
called, revalidate both before transfer and after its response. A later edit,
new pin, changed map or journal revision makes the report potentially stale;
the user must rerun and review, not treat this check as an atomic snapshot or
perpetual approval. Keep exact evidence-status meanings from v1, including
preservation of a decided candidate's older decision digest after a later
`record` observes changed evidence.

## Results, refusal and assistance

Offline report construction needs no provider. Implemented `--assist` shows
the exact bounded outbound packet—question, scoped evidence content, cited
claims and flow text, with hashes and byte count—plus destination/provider
and retention caveat, **before** any request. Obtain fresh affirmative consent
for this run and this packet after display; previous adoption/maintenance
consent does not apply. Decline, noninteractive input, changed packet or
unavailable provider yields no provider call. Use a tool-free, one-shot,
size/time-bounded request with no project execution, fetch, redirects or
automatic retries. Validate response schema, citations and budget; strip no
unsupported claim into a verified field. Refuse an invalid or scope-expanding
response in assisted mode; do not silently fall back to a different report
after a disclosed transfer. Never execute response text. Keep provider details
and disclosure receipt in the private report; consent is not a guarantee that
selected content contains no secrets.

Recommend result `review-required` on a complete report (findings and open
questions do not make a command failure), with report path/hash, pair/pins,
input/scan/journal identities, covered candidates, exclusions and whether
assistance was used. Failure produces no success result: invalid pair, brief,
scope or citation; missing/unsafe roots, map or journal; size or content
budget overrun; stale pin/bytes/revision; unsafe/occupied output; consent
decline or invalid model output in assisted mode; and I/O errors must have
distinct stable reasons. No partial report should be mistaken for success:
the Linux report writer uses exclusive sibling staging, sync/close, then
no-overwrite `linkat` publication and scratch removal/directory sync. This is
not a guarantee of the separate assessment writer. On late I/O failure the
report may already be visible; inspect the named report and any identified
scratch rather than assuming nothing happened. Cleanup removes only verified
owned scratch. Do not claim rollback or retry automatically.
No journal or source mutation is allowed on either success or refusal.

## Shared preparatory assistance flow

Assistance uses **one live shared reconciliation flow**
for CLI and generated skill, resolved from the installed/overridden/embedded
flow in the same precedence and with a source hash. Both routes call
the same deterministic scan, journal-read and bounded report
collector, rather than duplicate match/authority rules in prompts. A skill's
host may have different consent UI and data processing; it must disclose its
own actual outbound packet and enforce the same host-side boundaries. Flow
hashes and offline contract tests can check guidance consistency, not native
execution or CLI/skill behavioral parity. The offline primitive supplies
bounded capture, strict review validation, baseline recheck, journal joining,
private publication and typed result/refusal without a provider. Assistance
retains these boundaries and adds disclosure and response checks: strict
schema-1 `claims`/`questions`, only proposed needs-review claims citing
explicitly selected current captured file/map evidence, and empty claims valid.
The total prompt budget is 512 KiB; response is limited to 64 KiB. Model input
includes selected full text, sanitized scan metadata, seed claims/questions
and live flow, not host-added absolute paths, credentials or raw journal
contents. Selected text/claims are not redacted. Flow and input drift refuse
before send, after response and before private publication. Seed reviewer
claims and journal history remain separately attributed. The report marks
assistance true and includes an attributed, not authenticated, provider/model,
flow source/hash, payload hash/byte-count and consent-time receipt.
The v1 scan/journal JSON remains unchanged.

The implemented guided journal route remains separate from report assistance:
it consumes the actual v1 `record`/`decide`/`reconsider` operations with fresh
scan identity, expected evidence digest/revision and unique request ID. A new
observation may preserve a prior decision; changing a decided relationship or
outcome needs explicit reconsideration and trigger. Applied requires external
proof reference and separate verification/authority; a report cannot mint
proof or grant a source edit. Skill guidance must not promote report
proposals to journal decisions without that independent review.

## Report boundary scenarios

| Scenario | Expected report/refusal |
| --- | --- |
| Same bytes or matching title/ADR number, no reviewed relationship | Show byte equality or bucket only; correspondence and authority remain unreviewed. |
| Reviewed split/unmatched in brief; reviewer cites map rows saying `MATCH` and `XCIND-ONLY` | Preserve cardinality and cited conflicting claims as reviewer assertions, not scanner-parsed map facts; no automatic direction. |
| No journal; existing journal supplied; supplied path missing | Respectively `not-compared`; current/history comparison with saved decision; refuse missing journal without creating it. |
| Decision on E0; later journal `record` observes E1; scan E1 | Show `unchanged` to observed E1 and decision sequence/digest E0; no auto-reconsideration or applied promotion. |
| Candidate disappears from pair scope | Show journal-only `out-of-scope` history, no invented current observation or erased decision. |
| Intent, matching bytes, or journal `applied` with a syntactically valid URI | Never claim actual transfer/parity; attribute proof claim and require independent review. |
| Pair/root pin, map bytes, selected source, review, or journal changes mid-run | Refuse stale baseline before report; after publication label report time-bound and require a new run. |
| Oversize/binary selection, undeclared citation, symlink/occupied output | Refuse without truncated semantic assessment, scope expansion, overwrite, or source/journal write. |
| Offline; consent declined; model returns extra paths or instructions | Offline creates review-required report; decline sends nothing; invalid assisted output refuses and is never executed. |

## Genuine owner decisions and integration

Before a **real-pair** review the authorized owner must choose the pair,
scope/pins/map(s), data classification and permitted disclosure, responsible
pair/topic authority declarers, and later what independent external transfer
verification would suffice for an applied claim. Any real provider use needs
separate authorization and data-handling approval; an implemented consent
gate is not that approval. These choices are not supplied by Scind/Xcind
examples or by an existing journal. The bounded command spellings, budgets
and collector split here describe the implemented report slice, not a
real-pair integration result. No frozen v1 contract, source project or live
journal was changed for this slice.
