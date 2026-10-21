# Within-project documentation maintenance

**Status:** The owner accepted assessment-and-proposals-first. The initial
provider-free collection/report operation is implemented as described below;
automatic semantic assessment, model drafting and existing-file editing are
not implemented or authorized. No adoption pilot is required. The accepted
create-only adoption contract is unchanged.

The owner requested user documentation and maintenance-workflow design as
the next independent work after the completed adoption and reconciliation
work. Local WIP Matters `lds-user-documentation` and
`lds-maintenance-workflow-design` track those deliverables. The
[roadmap](roadmap.md) records their relationship to the remaining pilot and
release choices. [ADR-0001](../decisions/0001-adopt-two-track-documentation.md)
keeps user guidance in `docs/` and product design here. Local Matter
`lds-maintenance-assessment` tracks implementation of the accepted first slice.

## Start with assessment and reviewable proposals

The accepted first maintenance slice produces an evidence-backed review
report and proposed text outside the target tree. It **does not apply edits**.
A maintainer can use an ordinary editor and the project's existing review
process to accept selected changes. A later LDS editing operation would need
a separate behavior contract and approval; accepting this assessment design
would not authorize that operation.

This gives maintainers a useful way to ask “what documentation needs review
after this change?” without treating adoption-generated files as owned or
introducing an updater before its write policy is settled. It also lets
documentation-impact and Diátaxis review share one workflow instead of
creating a command for every review question. Root `assess`, `review`, and
`update` remain candidate names; the implemented operation is qualified
`lds plumbing docs assess`, not a second root workflow.

## Initial deterministic operation

```text
lds plumbing docs assess <root> --brief <brief.json> --output <absolute-new-report.md> [--json]
```

The root is explicit and existing. The caller chooses a new Markdown report
outside it, with existing no-follow parents. The operation creates no target
files or report-parent directories, never replaces an occupied report, and
requires Linux's descriptor-relative no-follow/exclusive creation. It uses
mode 0600, subject to umask, because the packet includes source content.
Trusted, quiescent ancestors are required; hostile mount/rename races are
not covered. A late I/O error may leave a partial external report, not a
transaction or an implicit retry authorization.

The strict, case-sensitive version-1 brief contains exactly `schemaVersion`,
`reason`, `audience`, `evidence`, `proposals`, and `questions`. All fields are
required/non-null; the three top-level arrays may be empty. Each additional
evidence entry has exactly `path` and lowercase 64-hex `sha256`, authorizing
that one regular UTF-8 file inside the target. Paths are clean slash-relative
paths without traversal, drive prefixes or symlinks. Each proposal has exactly
`path`, `summary`, a nonempty `evidence` path-reference array, and `proposedText`.
Proposal targets are root `README.md` or Markdown under `docs/`/`engineering/`;
citations must name default inventoried documents or declared source files.
Nonempty text is required for the reason, audience, proposals and questions.
Unknown, duplicate, missing and null fields refuse rather than broadening scope.
The [user how-to](../../docs/how-to/assess-documentation.md) contains a complete
brief and runnable disposable example.

The operation calls existing inventory and validation, captures all inventoried
Markdown plus explicitly named source bytes/hashes, and binds existing or absent
proposal targets. It repeats collection and re-reads the brief before reporting;
changed observations, root identity, brief bytes or expected evidence hashes
stop without a report. Collection is not atomic and reports can become stale
afterward. Evidence is labeled as observed working-tree bytes, not clean Git
pins, authenticated project facts or semantic proof.

The report separates observed link findings, caller-authored proposals and
unresolved questions, and includes the complete JSON evidence packet. It
does not discover semantic impacts or draft text on its own. Reviewers—human
or agent—supply those interpretations; the operation collects and preserves
them without promoting them to verified facts. No provider, network fetch,
Git command or target command is invoked. This deterministic plumbing member
enters the normal generated command projection; it adds no LLM flow or claim
of a maintenance facilitator/native workflow.

JSON stdout has `schemaVersion: 1`, `operation: "docs assess"`,
`status: "review-required"`, root path/identity, report path, exact brief hash
and parsed brief, complete inventory/validation, their equivalent exit statuses,
and sorted captured evidence (`path`, `exists`, `sha256`, `content`). Exit 0
means report creation, even with link findings (`validationExitCode: 1`);
it never means documentation is current. Invalid brief/path input exits 1,
usage exits 2, unsafe/stale/occupied-output refusal exits 3, and unexpected I/O
failure exits 4, using the usual error envelope under `--json`. Failures emit
no success result; late report writes may be partial as noted above.

## Reuse existing operations without extending their meaning

| Existing capability | Use in maintenance | Boundary |
| --- | --- | --- |
| `lds plumbing docs inventory <root> --json` | Capture scoped Markdown paths, hashes, supported links and skips. | Root `README.md`, `engineering/`, and `docs/` only; not a source-code inventory, ownership map or coverage assessment. |
| `lds plumbing docs validate <root> --json` | Include supported missing-local-file findings in the review. | Does not check fragments, external URLs, semantic accuracy or all Markdown syntax. A clean result is not “documentation is current.” |
| `lds plumbing docs render`, `plan`, `apply` | Continue to support separate, create-only adoption when appropriate. | Four fixed outputs, no existing-file edits. A differing output is a collision, not an update instruction or ownership claim. |
| `lds plumbing reconcile scan` and journal operations | Remain a separate cross-project workflow. | A within-project report does not need a pair descriptor or reconciliation journal. Neither workflow transfers content between repositories. |

The current [adoption runtime handoff](adoption-operations-runtime-handoff.md)
and [reconciliation handoff](bds-252-stage-4-engineering-handoff.md) describe
these implemented boundaries. Frozen contracts and historical evidence stay
unchanged. No new plumbing is justified merely to rename existing inventory
or link checking.

Inventory and validation still work independently. Assessment now combines
their evidence with an authored brief; semantic impact analysis and drafting
in the review sequence below remain the reviewer's responsibility.

## Inputs must identify the change and the review boundary

An assessment starts with an explicit target root, a reason for review, the
permitted read scope, and a caller-selected output location outside that
target. Reasons include a code/API change, a changed decision, a reported
documentation problem, or a periodic review. None implies approval to edit.

The review brief names the changed paths or decision, the intended audience
and task, and the source evidence for the claimed current behavior. Use
pinned commits when available; local uncommitted evidence needs exact path
and byte-hash identities and must be labeled as such. Do not describe a
working-tree observation as a clean committed snapshot. A comparison revision
is useful for impact review but is not required to review a reported defect.

Documentation scope is the default inventory; the brief bounds additional
source files, whose complete UTF-8 content enters the packet. Inventory alone
cannot establish how code works. Material outside its scope must be explicitly
declared; do not crawl the rest of a repository implicitly. Do not execute project commands,
install dependencies, fetch URLs or inspect secrets as part of assessment.
An operation needing those actions requires separate authorization.

Facts and citations are assertions to examine, not authenticated truth. A
generated-looking header, an adoption template match or a previously accepted
fact bundle does not establish continuing accuracy or authority over a file.

## Review sequence and division of responsibility

1. **Establish the review brief.** Confirm the explicit target, read scope,
   intended audience/task, change evidence and external report location. If
   evidence is missing, report the missing evidence rather than inventing a
   project fact or expanding the read scope.
2. **Collect bounded evidence.** Run existing inventory and validation and
   retain their complete results, skips and exit statuses. A validation exit
   1 supplies findings to review; an unsafe target or failed read stops the
   affected assessment. Capture the identities of any additional authorized
   inputs. These captures are not an atomic whole-repository snapshot.
3. **Identify candidate documentation impacts.** A reviewer may connect a
   changed operation or decision to particular documents. Every connection
   names its evidence and distinguishes an observed mismatch from a possible
   impact or a placement suggestion. Similar filenames alone are not
   evidence of relevance, and missing evidence is not proof of missing docs.
4. **Draft a review report.** Explain the user task, proposed wording or
   bounded change, current bytes being discussed, sources, link findings,
   exclusions and remaining questions. Separate user instructions from design
   intent; do not present a roadmap capability as a working command.
5. **Review the report, without applying it.** The maintainer can accept a
   recommendation, defer it with a reason, reject it, or request evidence.
   These dispositions describe a review decision, not an applied change.
   Current project tooling can track them; no LDS journal or approval database
   is required for this slice.
6. **Check resulting edits separately.** If a maintainer later changes files
   using their normal authorized process, rerun appropriate link and command
   checks and inspect the actual diff. A report is not proof that those edits
   happened or that the resulting instructions work.

Deterministic collection should reuse existing plumbing. If further stable
collection primitives prove necessary, specify their inputs, results and
failure behavior before implementing them. Semantic relevance, placement
advice and prose drafting are reviewer tasks, not claims the link checker can
prove. A model could assist those tasks, but it cannot approve edits or turn
an assertion into evidence.

A future model-backed CLI workflow and projected guidance should use one
shared flow and the same deterministic operations, following the existing
[workflow boundary](../decisions/0002-bds-251-adoption-workflow-boundary.md).
Do not reuse `flows/adopt.md` for a different maintenance responsibility or
maintain separate CLI and skill procedures. Provider selection, data sent,
consent and any permitted tools need their own reviewed scope before a
model-backed maintenance implementation; existing adoption consent is not
consent to send source excerpts for this workflow. Guidance alone does not
enforce read scope or decisions, and no native parity is claimed here.

## A report must expose evidence and uncertainty

The initial operation writes a human-readable report containing the complete
versioned JSON packet. It preserves inventory and validation outputs rather
than paraphrasing away their skips.

| Report content | Required distinction |
| --- | --- |
| Target, read scope, change/review brief, input identities and exclusions | A bound review, not a complete-repository audit. |
| Observations with path/location and source evidence | An observed broken local target or contradicted instruction, not an inferred ownership rule. |
| Candidate impacts and proposed text or placement | Reviewer interpretation; never silently labeled as verified current behavior. |
| Proposed additions to either documentation track | Real content for an audience/task, not empty layers or placeholders. An optional absent LDS layer is valid. |
| Findings, skips, unresolved questions and unreviewed scope | An unsupported check is not a pass; a model's confidence is not independent verification. |
| Input changes detected during or after collection | Mark affected conclusions stale; recollect and review, do not silently substitute new evidence. |
| Review disposition and verification recommendations | Acceptance of advice is not an applied outcome or permission to mutate files. |

The collector compares input identities again before finalizing a report.
If evidence changes it refuses, rather than claiming a consistent snapshot.
Before using a report to guide later edits, the maintainer must check that
its cited baseline is still relevant. This is
stale-evidence detection, not a filesystem race or transaction guarantee.

## Any future editing operation needs a separate decision

Read-only assessment deliberately leaves these choices outside its scope:

- **Eligible edits:** which explicitly selected files may be modified, and
  whether creates, moves, renames or deletions are allowed. No generated-file
  heuristic or standing adoption approval supplies that permission.
- **Review artifact:** exact before/after bytes or a complete diff, affected
  paths, provenance, baseline identities and a digest binding the entire
  proposed edit set. Approval must be fresh and tied to that artifact;
  approving a report or an adoption digest is not interchangeable with it.
- **Revalidation and safety:** refuse changed inputs, occupied new paths,
  unsafe roots/ancestors and stale approval rather than overwriting or
  silently replanning. Define supported platforms and identity/race limits;
  do not borrow a stronger safety claim from create-only apply.
- **Failure and recovery:** define reporting for actual partial effects,
  preservation and unattempted edits. State any atomicity limits explicitly;
  do not promise rollback or retry without knowing what reached disk.
- **Authority and interaction:** decide how a human approves existing-file
  edits and which deterministic surface performs them. A proposed workflow
  name or model response cannot supply that authority.

These need not be settled for the accepted assessment-only implementation.
Existing files remain outside its write authority by construction.

## Review scenarios and implementation boundary

Disposable evidence is sufficient for the initial operation's checks. The CLI
regression covers report contents, authored proposals, scoped/hashed evidence,
missing targets and skipped checks, stale hashes, invalid inputs, no-target-write
behavior and output/symlink refusals. The scenarios below also describe the
human review process; the command does not infer semantic contradictions.
No provider session, owner terminal exercise, native replay or pilot is required.

| Scenario | Expected boundary |
| --- | --- |
| A supplied code diff removes an option still documented in a scoped how-to. | Report the exact conflicting instruction and source evidence; draft corrected text outside the target; no target writes. |
| Source evidence is missing for a supposed replacement command. | Mark the question unresolved; do not draft the command as supported or invent a fixture fact as a real capability. |
| An external URL and an unknown fragment accompany a valid local filename. | Preserve skipped/unchecked coverage; do not conclude the whole link or page is valid. |
| A user edits an adoption-generated file. | It remains user content; neither maintenance nor adoption may infer permission to overwrite it. |
| Engineering-only documentation has no tutorials or optional layer directory. | No failure merely for absence; propose a user page only for an evidenced task and audience. |
| An in-scope file or cited source changes during report collection. | Mark the affected assessment stale or stop it; no target edit and no silent baseline replacement. |
| A maintainer accepts a proposed paragraph. | Record advice as accepted if desired; no LDS edit occurs and no applied outcome is claimed. |
| A source or proposed output path escapes scope or passes through a symlink. | Refuse the affected access/write; do not follow it or broaden scope automatically. |

## Coordination and next decision

The owner selected assessment/report delivery with optional reviewer text and
ordinary manual editing afterward. The user how-to and local WIP workplan
record current behavior and checks. Linear tracking, shipping and release
actions still need their own authorization; the repository is the design of
record. No additional owner test session or checkpoint is needed to complete
this first slice. LDS-managed existing-file edits, automatic/model drafting,
first-release inclusion and pilot choices remain separate future scope.
