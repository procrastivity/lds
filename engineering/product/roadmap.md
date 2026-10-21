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
Seven bounded `lds plumbing docs` leaves now inventory, check supported local
links, render the four adoption assets, preview a plan, and exclusively create
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
The human-facing assessment workflow is implemented. The manifest has 23
leaves, including 16 qualified plumbing-kind projection entries and seven
`docs` leaves; these counts do not establish native execution or parity.
Reconciliation commands do not infer semantic correspondence, choose real-pair
authority, edit source repositories,
or provide a human-facing reconciliation workflow. See the
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

LDS-managed existing-file edits would require a separate write contract and
approval. Root `review` and `update` remain candidate workflows.
Pilot selection and first-release inclusion remain separate decisions.

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
- **Write safety:** When may LDS create, update, or replace existing files?
  Which actions need a reviewable plan or human approval?
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
   do not claim they work in LDS yet.
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
5. Future: after explicit owner choices, consider a human-facing reconciliation
   workflow and an authorized pilot. Disposable source-derived and synthetic
   fixtures do not validate a live adoption, port, or parity claim.

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
| Porcelain | `lds update` | Bring one project's docs into line with its changed code and decisions. |
| Porcelain | `lds review` | Audit coverage, links, and cross-track consistency. |
| Porcelain (implemented, outside first-release claim) | `lds adopt <root> --facts <facts.json>` | Interactively review a create-only four-output documentation plan and apply only after fresh exact-digest human consent. |
| Porcelain | `lds reconcile` | Compare projects and guide explicit port/backport/divergence decisions. |
| Plumbing (implemented, bounded) | `lds plumbing docs inventory`, `validate`, `render`, `plan`, `apply` | Report scoped files/links, check supported local targets, render four assets, preview and create missing approved files without an LLM. |
| Plumbing (implemented, bounded) | `lds plumbing docs assess` | Collect documentation and explicitly hashed source evidence, link findings and authored proposals/questions in one new external report; no provider, semantic inference or target edits. See the [design](documentation-maintenance-workflow.md). |
| Plumbing (implemented, bounded) | `lds plumbing docs collect`; `lds plumbing docs assess --observed <collected.json>` | Offline full baseline packet with automatic source hashes; optional observed binding for a strict authored brief and external report. |
| Plumbing flow reader (implemented) | `lds plumbing assess [--json]` | Read live `flows/assess.md` content/source/hash; preparatory, unavailable for execution. |
| Plumbing (implemented, bounded) | `lds plumbing reconcile scan`; `lds plumbing reconcile journal read`, `record`, `decide`, `reconsider` | Inspect explicitly scoped pinned artifacts/maps and persist caller-reviewed correspondence and outcomes; never infer semantic matches or write a source. See the [engineering handoff](bds-252-stage-4-engineering-handoff.md). |

Agent-specific skills worth evaluating: adopting a partial tree, drafting
and reviewing an ADR, documentation impact review after code changes,
Diátaxis placement review, and a reconciliation facilitator. Their prompts
should use the same shipped/overridable assets and plumbing as the CLI,
not become a second independent implementation.
