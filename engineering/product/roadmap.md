# Building LDS as a Toolsmith application

This is a direction and candidate command surface, **not a released CLI
specification**. Track implementation in the
[LDS Linear project](https://linear.app/beausimensen/project/lds-22892af0659b);
record settled behavior in decisions, specs, and reference docs here.

## Current state

`toolsmith new --name lds` provided a Go CLI, embedded assets and override
resolution, and harness management. The root porcelain retains `version`,
`install`, and `uninstall`; the canonical plumbing members are
`lds plumbing manifest` and `lds plumbing doctor`, with reviewed root
aliases. These commands manage the CLI and projected agent skills; they do
not assess or reconcile project documentation; BDS-251 Stage 4 adds a
separate explicit adoption form described below.
The original seven bounded `lds plumbing docs` leaves inventory, check
supported local links, render the four adoption assets, preview a plan, and exclusively create
missing approved files after reviewed digest confirmation, plus collect
maintenance evidence, compute source hashes, and bind authored proposals to
observations in an external assessment report.
That deterministic assessment does not infer semantic impacts or edit target
files. The root `lds adopt <root> --facts <facts.json>` CLI form now uses the
shared live flow, OpenCode Go / `deepseek-v4-flash`, and those deterministic
operations. It requires owner-supplied facts and interactive exact-digest human
confirmation; the model is advisory and has no tools or source-file access.
This is not an installed-skill parity result, a named pilot, or first-release
inclusion.
Stage 4 Step 9 added generated Claude Code `/lds` adoption guidance: it reads
`lds plumbing adopt --json` live and delegates sequencing to
`assets/flows/adopt.md`. The reader still reports `preparatory` and
`unavailable-for-execution`; the guidance is not behavioral enforcement.
S10 is accepted under reduced scope for completed executor CLI cases and
independent High offline review, without independent live CLI replay or native
skill execution/parity; neither further test wave is required for this
acceptance. See the [S10 preparation record](bds-251-stage-4-step10-preparation.md)
for evidence limits, the [flow/plumbing map](adoption-flow-plumbing-map.md)
for bounded behavior, and the
[BDS-251 engineering handoff](bds-251-stage-4-engineering-handoff.md) for offline
regressions and observed limits. The
[runtime handoff](adoption-operations-runtime-handoff.md) covers deterministic
documentation operations. No real pilot is established.

BDS-252 adds `lds plumbing reconcile scan` and the explicit
`lds plumbing reconcile journal read|record|decide|reconsider` operations for
strict, deterministic evidence inspection and caller-owned decision history.
The first offline reconciliation report slice adds `lds reconcile report` and
identical `lds plumbing reconcile report`: Linux-only, bounded pinned evidence
plus a reviewer-authored brief in a new private external Markdown report with
provenance JSON. It never writes sources or the journal. The root form is
deterministic; only the qualified plumbing form enters the generated skill
table. Neither is a native execution or parity finding.
The human-facing assessment workflow and bounded existing-file update plumbing
are implemented. The manifest has 27 leaves, including 19 qualified
plumbing-kind projection entries and nine `docs` leaves; these counts do not
establish native execution or parity.
Reconciliation commands do not
infer semantic correspondence, choose real-pair authority, edit source
repositories, or provide an assisted/shared human-skill workflow. See the
[Stage 4 engineering handoff](bds-252-stage-4-engineering-handoff.md) for the
audited surface and boundaries.

The [frozen adoption contract](adoption-contract.md) retains a pre-runtime
roadmap-status paragraph as historical text. Its accepted adoption behavior
is unchanged; this current-state section and the linked runtime handoffs
supersede that paragraph only as descriptions of implementation status.

## User documentation and assessment proceed without a pilot

The [user documentation](../../docs/README.md) describes current commands and
limits, with disposable offline examples. Writing these guides manually is
not an LDS adoption pilot. Local WIP Matter `lds-user-documentation` tracks
the first guide; `lds-maintenance-workflow-design` tracks the separate
[within-project maintenance design](documentation-maintenance-workflow.md).
Neither deliverable requires another owner-driven live test wave or reopens
S10's reduced acceptance.

The owner accepted assessment and reviewable text outside the target as the
first implementation scope. Local Matter `lds-maintenance-assessment` tracks
`lds plumbing docs assess`: repeated inventory/link checks, explicitly hashed
source evidence, and caller-authored proposals/questions in a new report.
It needs no provider, pilot or target edits. Semantic impact analysis and
drafting remain reviewer responsibilities on this deterministic route.
Local Matter `lds-human-maintenance-assessment` tracks the implemented root
`lds assess` workflow: automatic scoped collection, fresh consent to transfer
the complete packet to OpenCode Go, one bounded tool-free model request, and
unverified proposed text in an external report. Offline
`lds plumbing docs collect` and optional `--observed` on `docs assess` bind an
authored brief to the full baseline. The root CLI and projected skill use the
same live `flows/assess.md`; `lds plumbing assess` only reads that preparatory
flow. Mocked terminal/provider tests and disposable offline examples cover
reporting, consent/refusal, stale evidence, invalid drafts, preservation and
live flow resolution. No live provider/native execution, parity, release
inclusion or owner checkpoint wave is claimed or required for this slice.

Bounded `docs update-plan` and `docs update-apply` are implemented under the separate
[update contract](documentation-update-contract.md), with full exact-byte
review and a fresh digest gate. This is not approval for any real target.
Root `review` and `update` remain candidate workflows.
Pilot selection and first-release inclusion remain separate decisions.

## Separate update implementation and assisted reconciliation design

The owner requested these two design workstreams after the assessment workflow
was delivered. They proceed without selecting a pilot or reopening S10:

- **Reviewed existing-file updates:** local Matter
  `lds-existing-documentation-updates` tracks implementation following the
  `lds-reviewed-documentation-updates` design and
  [update contract](documentation-update-contract.md). The implemented first
  slice replaces only explicitly selected existing Markdown files after full
  before/after review and fresh digest-bound approval. Creates, deletes, moves,
  inferred ownership and automatic assessment acceptance are not authorized.
  No real-target write or pilot is authorized; adoption remains create-only.
- **Human-facing reconciliation:** local Matter
  `lds-human-reconciliation-workflow` tracks the proposed
  [workflow contract](reconciliation-workflow.md). Its first offline slice now
  produces a private external review report from explicit pinned pair/map,
  reviewer brief, and optional existing-journal evidence. It does not write
  either source or the journal. Optional model assistance, a shared CLI/skill
  flow, journal decision UX, and real-pair authority remain deferred.

The implemented update plumbing does not grant
permission to edit a real project. The remainder of the reconciliation
workflow is proposed. Any live provider access, real target write, pilot or
release needs its own authorization. Existing
assessment and reconciliation plumbing remain usable.

## Planning constraints and open decisions

Settled: the two-track layout is the intended default, and within-project
documentation maintenance is distinct from cross-project reconciliation.
For any LLM workflow LDS ships, [Toolsmith's shape contract](https://github.com/procrastivity/toolsmith/blob/main/CONTRACT.md#c8--shapes)
requires one flow asset shared by the CLI verb and projected skill; every
deterministic step must be available through plumbing. Do not design the
two forms as independently maintained workflows.

The following product choices remain open; they are not prerequisites for the
implemented deterministic plumbing and must be settled before any real use:

- **First usable release:** Which project is the pilot? What must `lds init`
  produce or preserve to count as useful, and how much partial adoption is
  supported?
- **Existing-file write safety for a real target:** Decide the authorized
  project/paths/reviewer and accept inode/timestamp replacement there. The
  bounded update contract requires exact before/after approval, stale-input
  refusal, per-file replacement and partial-result recovery. Adoption's
  create-only policy does not grant update permission.
- **LLM release and pairing:** Step 8 implements the CLI-side form using the
  selected provider and Step 9 projects adoption guidance into Claude Code.
  S10's reduced-scope acceptance is not native adoption or a parity PASS;
  first-release inclusion remains a separate choice. Do not infer provider
  reliability, Anthropic support, or resolution of the original failed S7
  attempt from this evidence.
- **Reconciliation application:** Which real pair, if any, is an authorized
  pilot? Who may declare pair/topic authority and permitted directions, and
  what external evidence is sufficient to verify an applied transfer?
  ADR-0003 and the v1 contract define the journal boundary and state model,
  not these real-world values. Xcind's ledger remains evidence, not LDS's
  universal authority rule.

## Candidate work, in dependency order

1. Specify the initial two-track project layout and select/adapt templates
   from the reference repositories. Ship defaults as embedded assets with
   the existing installed/user override chain. Repository-local overrides
   depend on [Toolsmith BDS-234](https://linear.app/beausimensen/issue/BDS-234/support-repository-local-overrides-for-generated-tools-assets-and);
   do not claim they work in LDS yet. Read-only follow-up on 2026-10-21 found
   the issue still Backlog, with no comments or linked implementation, no open
   Toolsmith PRs, and no PR matching BDS-234. Remote Toolsmith main
   [`20e9e44`](https://github.com/procrastivity/toolsmith/commit/20e9e44b2a70e37216b28a58142b5e3e17cf2259)
   still specifies only user-XDG → installed → embedded in
   [C5](https://github.com/procrastivity/toolsmith/blob/20e9e44b2a70e37216b28a58142b5e3e17cf2259/CONTRACT.md#L242-L257)
   and the [generated resolver](https://github.com/procrastivity/toolsmith/blob/20e9e44b2a70e37216b28a58142b5e3e17cf2259/assets/_skeleton/internal/asset/asset.go#L67-L143).
   The feature is not implemented on that main; existing generated LDS source
   would also need an explicit port/rebuild after upstream support lands.
2. Implement deterministic operations for inventorying files and links,
   validating the chosen layout, and rendering/scaffolding templates.
   Provide machine-readable results and tests before agent-driven flows.
3. Add human-facing flows and agent-specific skills that use those
   operations to adopt LDS, keep documentation aligned with code, and
   review both tracks. The create-only `lds adopt` CLI form is implemented
   under BDS-251 Stage 4 Step 8; Step 9 implements generated Claude Code
   adoption guidance, but native execution and CLI/skill parity remain
   unverified. Keep the skill projection derived from the CLI manifest and
   shipped assets.
4. **Implemented, bounded:** specify and implement deterministic cross-project
   reconciliation plumbing separately from within-project documentation
   maintenance. The v1 scan reports pinned evidence; explicit journal requests
   record reviewed correspondence and decisions without source edits. This is
   not semantic matching, a real-pair decision, or a facilitator. Xcind's
   [cross-project procedure](https://github.com/scinddev/xcind/blob/main/engineering/maintenance/cross-project-sync.md)
   and ledgers remain reference evidence, not a schema copied into LDS.
5. **First offline report slice implemented:** create a bounded report
   separately from journal writes and any authorized pilot. Assistance,
   shared flow and decision UX remain design work. Disposable source-derived
   and synthetic fixtures do not validate a live adoption, port, or parity
   claim.

## Candidate verbs

Names and boundaries below are proposals except the marked implemented rows.
Porcelain is the human-facing workflow;
plumbing is deterministic and suitable for JSON/exit-code use by skills and
scripts. `toolsmith new` bootstraps the *CLI application*; a future `lds init`
would bootstrap documentation *inside another project*.

| Surface | Candidate | Purpose |
| --- | --- | --- |
| Porcelain | `lds init` | Set up the selected documentation tracks and templates. |
| Porcelain (implemented, outside first-release claim) | `lds assess <root> --reason <text> --audience <text> --output <absolute-new-report.md> [--source <relative-file> ...] [--json]` | Collect scoped documentation and named source bytes, ask fresh transfer consent, and request reviewable model-authored maintenance proposals; no target edits. |
| Porcelain (candidate; not specified or implemented) | `lds update` | Future human-facing update orchestration. The [bounded implementation contract](documentation-update-contract.md) specifies deterministic `docs update-plan` / `docs update-apply` plumbing only, with exact-byte review and separate approval. |
| Porcelain | `lds review` | Audit coverage, links, and cross-track consistency. |
| Porcelain (implemented, outside first-release claim) | `lds adopt <root> --facts <facts.json>` | Interactively review a create-only four-output documentation plan and apply only after fresh exact-digest human consent. |
| Porcelain (implemented offline slice) | `lds reconcile report --pair <absolute-pair.json> --left-root <absolute-dir> --right-root <absolute-dir> --map <absolute-tracked-file> [--map ...] [--journal <absolute-existing-journal.json>] --review <absolute-review.json> --output <absolute-new-report.md> [--json]` | Linux private external report from clean pinned evidence and reviewer assertions; no model, source/journal writes, or project commands. See the [how-to](../../docs/how-to/reconcile-projects.md). |
| Plumbing (implemented, bounded) | `lds plumbing docs inventory`, `validate`, `render`, `plan`, `apply` | Report scoped files/links, check supported local targets, render four assets, preview and create missing approved files without an LLM. |
| Plumbing (implemented, bounded) | `lds plumbing docs assess` | Collect documentation and explicitly hashed source evidence, link findings and authored proposals/questions in one new external report; no provider, semantic inference or target edits. See the [design](documentation-maintenance-workflow.md). |
| Plumbing (implemented, bounded) | `lds plumbing docs collect`; `lds plumbing docs assess --observed <collected.json>` | Offline full baseline packet with automatic source hashes; optional observed binding for a strict authored brief and external report. |
| Plumbing (implemented, bounded) | `lds plumbing docs update-plan <root> --observed <collected.json> --request <edits.json> --json`; `lds plumbing docs update-apply <root> --plan <reviewed.json> --confirm <sha256:digest> --json` | Existing selected Markdown only; full digest-bound review, fresh approval and per-file Linux replacement. No real target or pilot authorized. See the [how-to](../../docs/how-to/update-documentation.md). |
| Plumbing flow reader (implemented) | `lds plumbing assess [--json]` | Read live `flows/assess.md` content/source/hash; preparatory, unavailable for execution. |
| Plumbing (implemented, bounded) | `lds plumbing reconcile scan`; `lds plumbing reconcile journal read`, `record`, `decide`, `reconsider` | Inspect explicitly scoped pinned artifacts/maps and persist caller-reviewed correspondence and outcomes; never infer semantic matches or write a source. See the [engineering handoff](bds-252-stage-4-engineering-handoff.md). |
| Plumbing (implemented, bounded) | `lds plumbing reconcile report` (same flags as root report) | Identical deterministic report operation and only report form eligible for generated skill table; no native/parity claim. |

Agent-specific skills worth evaluating: adopting a partial tree, drafting
and reviewing an ADR, documentation impact review after code changes,
Diátaxis placement review, and a reconciliation facilitator. Their prompts
should use the same shipped/overridable assets and plumbing as the CLI,
not become a second independent implementation.
