# Source repositories and local checkouts

The [LDS Amp project](https://ampcode.com/@beausimensen/LDS) now clones
[`procrastivity/lds`](https://github.com/procrastivity/lds) as its primary
GitHub repository. The former Amp-hosted Git remote remains an `amp` remote
in the current checkout for history; it is not the source for new project
orbs. This repository is the authority for the new tool's decisions and code.

These repositories are **references, not interchangeable sources of truth**:

| Repository | Role for LDS work |
| --- | --- |
| [`procrastivity/toolsmith`](https://github.com/procrastivity/toolsmith) (`main`) | Application contract, generated chassis, asset resolution, and build conventions. LDS consumes the contract rather than copying Toolsmith's internal design. |
| [`scinddev/xcind`](https://github.com/scinddev/xcind) (`main`) | Working two-track adoption and cross-project reconciliation records, including intentional divergence. |
| [`scinddev/scind`](https://github.com/scinddev/scind) (`main`) | Earlier engineering specification and maintenance/sync guidance; compare it with Xcind rather than assuming one-to-one ports. |
| [`simensen/lds-old`](https://github.com/simensen/lds-old) (`main`) | Historical LDS templates and workflows to evaluate, not automatically import. |
| [`gethmn/hypomnema`](https://github.com/gethmn/hypomnema) (`main`) | Example of an incremental LDS adoption ADR and a partially filled tree. |
| [`simensen/weftlo`](https://github.com/simensen/weftlo) (`main`) | Historical skills-distribution context, not the distribution mechanism for this CLI. |
| [`simensen/bizapps-symfony-bot`](https://github.com/simensen/bizapps-symfony-bot) (`main`) | Another adoption case for testing assumptions against real projects. |
| [`procrastivity/wip`](https://github.com/procrastivity/wip) (**`go`**, not `main`) | Toolsmith-style Go CLI and porcelain/plumbing reference. Its `main` branch is the older shell implementation. |

In Amp orbs, additional checkouts are expected under `../repos/<name>`
relative to this repo. The Amp project lists their repository URLs; its
`.agents/setup` switches Wip to `go` before building it. The setup and
update scripts currently assume this sibling layout, so **the checkout
root is not configurable yet**. For development outside orbs, use these
Git URLs to place checkouts in the same relative layout for now. A later
change should allow a local repo-root override without making these
reference checkouts a runtime dependency of `lds`.
