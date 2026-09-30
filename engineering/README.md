# LDS engineering documentation

This is the design of record for people and agents building the `lds` tool.
It follows [ADR-0001](decisions/0001-adopt-two-track-documentation.md):
engineering decisions and plans live here; future user-facing documentation
belongs in `docs/` and follows Diátaxis. The tool currently has a Toolsmith
scaffold and harness-management commands plus five deterministic `lds docs`
plumbing leaves; human-facing documentation workflows are not implemented.

Start with the [roadmap](product/roadmap.md) for proposed capabilities and
their status. The [source-repository map](architecture/source-repositories.md)
records what other projects contribute and how their local checkouts are found.
The [adoption runtime handoff](product/adoption-operations-runtime-handoff.md)
records the implemented plumbing, executable disposable evidence, and limits.
The [BDS-251 workflow decision](decisions/0002-bds-251-adoption-workflow-boundary.md)
records the accepted preparation scope and gates for paired delivery.
The [flow-to-plumbing map](product/adoption-flow-plumbing-map.md) connects
those decisions to existing documentation operations and disposable cases.
The [LDS Linear project](https://linear.app/beausimensen/project/lds-22892af0659b)
tracks actionable work; decisions and command contracts belong in this repo.

The seven LDS layers are decisions, product, architecture, specifications,
reference, behaviors, and implementation. Only layers with actual content
need directories now; add the rest as the tool takes shape. Do not mistake
proposals in the roadmap for implemented CLI commands.
