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

## Planning constraints and open decisions

Settled: the two-track layout is the intended default, and within-project
documentation maintenance is distinct from cross-project reconciliation.
For any LLM workflow LDS ships, [Toolsmith's shape contract](https://github.com/procrastivity/toolsmith/blob/main/CONTRACT.md#c8--shapes)
requires one flow asset shared by the CLI verb and projected skill; every
deterministic step must be available through plumbing. Do not design the
two forms as independently maintained workflows.

Resolve these questions before specifying the corresponding verbs:

- **First usable release:** Which project is the pilot? What must `lds init`
  produce or preserve to count as useful, and how much partial adoption is
  supported?
- **Write safety:** When may LDS create, update, or replace existing files?
  Which actions need a reviewable plan or human approval?
- **LLM execution:** Will the first release include a CLI-side LLM workflow
  as well as a projected skill? If so, how will the CLI obtain an LLM?
- **Reconciliation authority:** Who owns the journal, how are unresolved
  ports/backports distinguished from accepted divergences, and when must a
  divergence be reconsidered? Xcind's ledger is evidence, not LDS's schema.

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
4. Specify cross-project **reconciliation** separately from within-project
   documentation maintenance. Compare corresponding concepts in two
   projects, record each candidate change in a durable journal, and require
   an intentional decision to port, backport, defer, or retain a divergence.
   Xcind's [cross-project procedure](https://github.com/scinddev/xcind/blob/main/engineering/maintenance/cross-project-sync.md)
   and ledgers are reference material, not a schema to copy blindly.
5. Exercise the workflows on a new project and on existing adoptions such
   as Xcind and Hypomnema, checking partial adoption and non-1:1 ports.

## Candidate verbs

Names and boundaries below are **proposals except the implemented docs row**. Porcelain is the human-facing
workflow; plumbing is deterministic and suitable for JSON/exit-code use by
skills and scripts. `toolsmith new` bootstraps the *CLI application*; a
future `lds init` would bootstrap documentation *inside another project*.

| Surface | Candidate | Purpose |
| --- | --- | --- |
| Porcelain | `lds init` | Set up the selected documentation tracks and templates. |
| Porcelain | `lds assess` | Inventory an existing project's docs and plan adoption. |
| Porcelain | `lds update` | Bring one project's docs into line with its changed code and decisions. |
| Porcelain | `lds review` | Audit coverage, links, and cross-track consistency. |
| Porcelain | `lds reconcile` | Compare projects and guide explicit port/backport/divergence decisions. |
| Plumbing (implemented, bounded) | `lds plumbing docs inventory`, `validate`, `render`, `plan`, `apply` | Report scoped files/links, check supported local targets, render four assets, preview and create missing approved files without an LLM. |
| Plumbing | reconciliation scan and journal operations | Produce candidate differences and persist decisions without assuming a 1:1 port. |

Agent-specific skills worth evaluating: adopting a partial tree, drafting
and reviewing an ADR, documentation impact review after code changes,
Diátaxis placement review, and a reconciliation facilitator. Their prompts
should use the same shipped/overridable assets and plumbing as the CLI,
not become a second independent implementation.
