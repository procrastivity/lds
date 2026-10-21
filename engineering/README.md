# LDS engineering documentation

This is the design of record for people and agents building the `lds` tool.
It follows [ADR-0001](decisions/0001-adopt-two-track-documentation.md):
engineering decisions and plans live here; [user documentation](../docs/README.md)
lives in `docs/` and follows Diátaxis. The tool currently has a Toolsmith
scaffold and harness-management commands plus six deterministic
`lds plumbing docs` leaves, as well as deterministic `lds plumbing reconcile`
scan and explicit journal operations. Reconciliation is bounded to pinned
evidence and a caller-owned journal. The interactive root
`lds adopt <root> --facts <facts.json>` form uses the live shared flow and
those deterministic docs operations. Generated Claude Code `/lds` adoption
guidance fetches that flow live through `lds plumbing adopt --json`; the reader
is preparatory and the guidance is not behavioral enforcement. S10 accepts
executor CLI evidence and independent offline review, not native adoption or
CLI/skill parity. No pilot or first-release claim follows; a human-facing
reconciliation facilitator remains deferred.

Start with the [roadmap](product/roadmap.md) for proposed capabilities and
their status. The [source-repository map](architecture/source-repositories.md)
records what other projects contribute and how their local checkouts are found.
The [within-project maintenance design](product/documentation-maintenance-workflow.md)
records the accepted assessment-first scope. `lds plumbing docs assess`
collects bounded evidence and authored proposals in an external report;
it does not infer semantic impacts, call a model or edit target files.
The [adoption runtime handoff](product/adoption-operations-runtime-handoff.md)
records the implemented plumbing, executable disposable evidence, and limits.
The [BDS-251 engineering handoff](product/bds-251-stage-4-engineering-handoff.md)
records the owner-reduced acceptance, offline regressions, and unverified
native parity boundary.
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
authorize source-project edits. The [Stage 4 engineering handoff](product/bds-252-stage-4-engineering-handoff.md)
records the implemented command surface, the bounded Linux runtime audit, and
deferred real-pair/pilot choices; the [runtime round](product/bds-252-stage-4-runtime-round.md)
and [machine-readable evidence](product/bds-252-stage-4-runtime-evidence.json)
record disposable execution results.
The [LDS Linear project](https://linear.app/beausimensen/project/lds-22892af0659b)
tracks actionable work; decisions and command contracts belong in this repo.

The seven LDS layers are decisions, product, architecture, specifications,
reference, behaviors, and implementation. Only layers with actual content
need directories now; add the rest as the tool takes shape. Do not mistake
proposals in the roadmap for implemented CLI commands.
