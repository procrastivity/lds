# LDS engineering documentation

This is the design of record for people and agents building the `lds` tool.
It follows [ADR-0001](decisions/0001-adopt-two-track-documentation.md):
engineering decisions and plans live here; future user-facing documentation
belongs in `docs/` and follows Diátaxis. The tool currently has a Toolsmith
scaffold and harness-management commands plus five deterministic
`lds plumbing docs` leaves; human-facing documentation workflows are not
implemented.

Start with the [roadmap](product/roadmap.md) for proposed capabilities and
their status. The [source-repository map](architecture/source-repositories.md)
records what other projects contribute and how their local checkouts are found.
The [adoption runtime handoff](product/adoption-operations-runtime-handoff.md)
records the implemented plumbing, executable disposable evidence, and limits.
The [BDS-251 workflow decision](decisions/0002-bds-251-adoption-workflow-boundary.md)
records the accepted preparation scope and gates for paired delivery.
The [BDS-251 CLI surface audit](product/cli-surface-audit.md) records the
reviewed root-command audience, kind, namespace placement, and compatibility
alias decisions.
The [flow-to-plumbing map](product/adoption-flow-plumbing-map.md) connects
those decisions to existing documentation operations and disposable cases.
The [BDS-252 reconciliation decision](decisions/0003-bound-cross-project-reconciliation.md)
sets pair-specific authority, evidence trust, and caller-owned journal
boundaries; its [pinned evidence](product/bds-252-stage-1-evidence.md) and
[expected-case cards](product/bds-252-stage-1-expected-case-cards.md) do not
authorize source-project edits.
The [LDS Linear project](https://linear.app/beausimensen/project/lds-22892af0659b)
tracks actionable work; decisions and command contracts belong in this repo.

The seven LDS layers are decisions, product, architecture, specifications,
reference, behaviors, and implementation. Only layers with actual content
need directories now; add the rest as the tool takes shape. Do not mistake
proposals in the roadmap for implemented CLI commands.
