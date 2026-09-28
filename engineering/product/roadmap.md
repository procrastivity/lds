# Building LDS as a Toolsmith application

This is a direction and candidate command surface, **not a released CLI
specification**. Track implementation in the
[LDS Linear project](https://linear.app/beausimensen/project/lds-22892af0659b);
record settled behavior in decisions, specs, and reference docs here.

## Current state

`toolsmith new --name lds` provided a Go CLI, embedded assets and override
resolution, a manifest, and harness-management plumbing: `version`,
`manifest`, `install`, `uninstall`, `doctor`. These commands manage the CLI
and projected agent skills. They do **not** create, assess, or reconcile
project documentation yet.

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

Names and boundaries below are **proposals**. Porcelain is the human-facing
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
| Plumbing | inventory, validate, render | Report files/links, check structure, and render assets without an LLM. |
| Plumbing | reconciliation scan and journal operations | Produce candidate differences and persist decisions without assuming a 1:1 port. |

Agent-specific skills worth evaluating: adopting a partial tree, drafting
and reviewing an ADR, documentation impact review after code changes,
Diátaxis placement review, and a reconciliation facilitator. Their prompts
should use the same shipped/overridable assets and plumbing as the CLI,
not become a second independent implementation.
