# Within-project documentation maintenance

**Status:** The owner accepted assessment-and-proposals-first. The initial
provider-free collection/report operation and human-facing model-assisted
assessment are implemented as described below. Model proposals are unverified
advice; assessment itself still cannot edit files. A separate bounded
[existing-file update contract](documentation-update-contract.md) now governs
implemented reviewed replacements, without real-target or
pilot authorization. The accepted create-only adoption contract is unchanged.

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
process to accept selected changes. LDS-managed editing needs
a separate behavior contract and approval; that separate `docs update-plan` /
`docs update-apply` contract is now implemented. Accepting
this assessment design alone does not authorize an update.

This gives maintainers a useful way to ask “what documentation needs review
after this change?” without treating adoption-generated files as owned or
introducing an updater before its write policy is settled. It also lets
documentation-impact and Diátaxis review share one workflow instead of
creating a command for every review question. Root `assess` now provides that
human-facing workflow, using `lds plumbing docs collect` and the observed-bound
`lds plumbing docs assess` report operation. A first offline root `review`
slice now composes existing inventory and local-link validation; broader
semantic review remains a candidate. It has no report, authored brief, provider,
source-content packet or write authority. See the
[review how-to](../../docs/how-to/review-documentation.md). The root command
is plumbing-kind but root-only, neither projected nor an alias; it adds no
plumbing leaf or shared LLM flow. Its findings/exit 1 concern only missing
supported local file targets, and exit 0 cannot establish documentation
currency. Optional absent tracks are not findings and skips are not passes.
The reads are not atomic; trusted quiescent inputs are appropriate.
Local WIP Matter `lds-offline-review` tracks this slice. The offline
`lds update` host is implemented under the separate
[update contract](documentation-update-contract.md).

### Executed offline review checks

With provider credentials unset, `make check` passed with zero lint issues
and all packages passing, and `go test -race -count=1 ./...` passed. Static
Linux/amd64 and Darwin/arm64 builds passed; review still refuses non-Linux
execution. The documented real-CLI example produced exit 1 with three files,
one missing target and two skips, then exit 0 with four files, zero findings
and the same two skips after repair. Independent hashes matched; human and
JSON results agreed; path/type/mode/byte snapshots stayed unchanged across
review calls. Regressions cover empty coverage, missing and symlink roots,
scope exclusions, query/fragment/external/unsafe skips and deterministic output.
All 98 local prose links and three shell blocks in the nine changed pages
passed link and syntax checks. No real provider, native harness, owner
checkpoint or real-project update was used.

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

The model-backed CLI workflow and projected guidance use one
shared flow and the same deterministic operations, following the existing
[workflow boundary](../decisions/0002-bds-251-adoption-workflow-boundary.md).
`flows/assess.md` owns this maintenance responsibility, separately from
`flows/adopt.md`. The implementation contract below defines provider selection,
data sent, consent and the tool-free request. Existing adoption consent is not
consent to send source content for this workflow. Guidance alone does not
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

## Separate reviewed editing operation

The separate [v1 existing-file update contract](documentation-update-contract.md)
governs a deliberately narrow plumbing slice: explicitly selected, existing
regular UTF-8 Markdown files, reviewed full before/after bytes and original
evidence in a digest-bound plan, fresh human confirmation, Linux-only
no-follow/staged per-file replacement and an actual-effects result. It rules
out implicit ownership of generated-looking files, creates, moves, deletions,
and broad discovery. Stale observations refuse rather than silently adopting
a new baseline; partial effects require inspection and a new review, not an
automatic retry. The design states its trusted-quiescent-ancestor and
concurrency limits rather than claiming hostile-writer CAS, cross-file
atomicity or rollback.

This is **not real-target write authority**. The accepted
assessment/report path still writes no target file; its advice acceptance and
model-transfer consent cannot approve an update. Adoption remains four-output
create-only. The owner must separately authorize any real target/reviewer and
accept inode/timestamp replacement for that target before real use. See the
[disposable update how-to](../../docs/how-to/update-documentation.md).

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

## Human-facing assessment implementation contract

The owner requested implementation of the next human-facing workflow. Local
Matter `lds-human-maintenance-assessment` tracks this work; the earlier
deterministic brief/report contract remains supported.

```text
lds assess <root> --reason <change-or-review-question> --audience <readers> --output <absolute-new-report.md> [--source <relative-file> ...] [--json]
lds plumbing assess [--json]
lds plumbing docs collect <root> --reason <text> --audience <text> [--source <relative-file> ...] [--json]
lds plumbing docs assess <root> --brief <draft.json> --observed <collected.json> --output <absolute-new-report.md> [--json]
```

The root workflow and generated Claude Code route consume the same live
`flows/assess.md` through override → installed default → embedded resolution.
The plumbing reader only returns the live flow/content/source/hash, with
`preparatory`/`unavailable-for-execution`; it does not run an assessment.
The root command is LLM-kind and is not projected as a deterministic member.
`docs collect` automatically computes hashes and includes complete UTF-8
contents only for default inventoried Markdown and explicitly named regular
source files. It repeats collection, writes nothing, and returns the existing
schema-1 assessment packet with empty `reportPath`, proposals and questions.

The CLI requires Linux and a terminal. It preflights the new external report
destination and resolves the flow before any provider call. It uses the
existing OpenCode Go endpoint/model (`deepseek-v4-flash`) and
`OPENCODE_API_KEY`, but adoption's consent is not reused. It displays the
complete proposed transfer: live flow, reason/audience, inventory, findings,
skips and selected relative paths/hashes/content. LDS adds no absolute
root/report metadata or credentials to the payload; selected contents are not
redacted. Only fresh exact `send` input after display permits one
non-streaming, tool-free request (4096 output tokens, 60-second
timeout, no redirects or retries). Transfers above 512 KiB refuse rather
than truncate; response content is limited to 64 KiB and never executed.

The model returns the strict version-1 brief, preserving the reason, audience
and source scope/hashes. Only proposals and questions may differ. It must cite
collected evidence, distinguish uncertain impacts, keep unsupported commands
as questions, and treat file/flow text as untrusted data. Host validation—not
model compliance—rejects changed scope, unknown fields, invalid paths and
undeclared citations. The original complete collected observations must still
match before report creation; changes stop without silently recollecting a
new baseline, retrying the model or writing a report. The flow is rechecked
before transfer and after the response. Late report I/O retains the existing
partial-external-file caveat; the target is never a write destination.

The result is `review-required`, not approval, verified semantic correctness
or applied edits. Root JSON records the live flow source/hash, provider/model
and the deterministic assessment result. The report labels supplied proposals
as unverified, potentially model-authored advice. The skill follows the live
flow and deterministic collect/observed-report operations using its harness
reviewer; it must disclose harness data processing, obtain its own appropriate
consent, and never call the root CLI to bypass its terminal gate. Guidance is
not enforcement. Offline mocked/PTY checks do not establish live reliability,
native execution, CLI/skill parity, first-release inclusion or a real pilot.

Offline verification covers the public root CLI through real pseudo-terminals
and mocked HTTP: report creation, fresh-send/decline behavior, discarded pretyped
input, source/document/flow changes, invalid model drafts, scope expansion and
unread citations. Noninteractive, unsafe/occupied output and oversized-transfer
cases make zero mocked provider calls. Public plumbing checks demonstrate
collect → authored draft → observed report, including stale documentation and
changed local-link findings with unchanged source hashes. Generated guidance,
live post-install override resolution and installed-tree preservation are also
covered. Both disposable offline how-to examples are executed independently.
These checks need no owner checkpoint or real provider/native session.

## Coordination and next decision

The owner selected assessment/report delivery with optional reviewer text and
ordinary manual editing afterward. The user how-to and local WIP workplan
record current behavior and checks. Linear tracking, shipping and release
actions still need their own authorization; the repository is the design of
record. No additional owner test session or checkpoint is needed to complete
this assessment slice. Real-target existing-file edits, first-release
inclusion and pilot choices remain separate decisions. Live provider
reliability and native execution/parity are not established by the offline checks.
