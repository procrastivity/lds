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
**not** create, assess, or reconcile project documentation.
Five bounded `lds plumbing docs` leaves now inventory, check supported local
links, render the four adoption assets, preview a plan, and exclusively create
missing approved files after reviewed digest confirmation. See the
[runtime handoff](adoption-operations-runtime-handoff.md) for evidence and
limitations; no human-facing flow or real pilot is established.

BDS-252 adds `lds plumbing reconcile scan` and the explicit
`lds plumbing reconcile journal read|record|decide|reconsider` operations for
strict, deterministic evidence inspection and caller-owned decision history.
The actual manifest currently contains 18 leaves, including 13 qualified
plumbing-kind projection entries. These commands do not infer semantic
correspondence, choose real-pair authority, edit source repositories, or
provide a human-facing reconciliation workflow. See the
[Stage 4 engineering handoff](bds-252-stage-4-engineering-handoff.md) for the
audited surface and boundaries.

The [frozen adoption contract](adoption-contract.md) retains a pre-runtime
roadmap-status paragraph as historical text. Its accepted adoption behavior
is unchanged; this current-state section and the linked runtime handoffs
supersede that paragraph only as descriptions of implementation status.

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
- **LLM execution:** Will the first release include a CLI-side LLM workflow
  as well as a projected skill? If so, how will the CLI obtain an LLM?
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
   review both tracks. Keep the skill projection derived from the CLI
   manifest and shipped assets.
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

Names and boundaries below are proposals except the implemented docs and
reconciliation plumbing rows. Porcelain is the human-facing workflow;
plumbing is deterministic and suitable for JSON/exit-code use by skills and
scripts. `toolsmith new` bootstraps the *CLI application*; a future `lds init`
would bootstrap documentation *inside another project*.

| Surface | Candidate | Purpose |
| --- | --- | --- |
| Porcelain | `lds init` | Set up the selected documentation tracks and templates. |
| Porcelain | `lds assess` | Inventory an existing project's docs and plan adoption. |
| Porcelain | `lds update` | Bring one project's docs into line with its changed code and decisions. |
| Porcelain | `lds review` | Audit coverage, links, and cross-track consistency. |
| Porcelain | `lds reconcile` | Compare projects and guide explicit port/backport/divergence decisions. |
| Plumbing (implemented, bounded) | `lds plumbing docs inventory`, `validate`, `render`, `plan`, `apply` | Report scoped files/links, check supported local targets, render four assets, preview and create missing approved files without an LLM. |
| Plumbing (implemented, bounded) | `lds plumbing reconcile scan`; `lds plumbing reconcile journal read`, `record`, `decide`, `reconsider` | Inspect explicitly scoped pinned artifacts/maps and persist caller-reviewed correspondence and outcomes; never infer semantic matches or write a source. See the [engineering handoff](bds-252-stage-4-engineering-handoff.md). |

Agent-specific skills worth evaluating: adopting a partial tree, drafting
and reviewing an ADR, documentation impact review after code changes,
Diátaxis placement review, and a reconciliation facilitator. Their prompts
should use the same shipped/overridable assets and plumbing as the CLI,
not become a second independent implementation.
