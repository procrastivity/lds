# lds

LDS is a CLI for a Layered Documentation System: engineering documentation
alongside Diátaxis user documentation. It has deterministic `lds plumbing docs`
inventory, local-link validation, four-asset rendering, reviewed planning,
and create-only application, plus bounded `lds plumbing reconcile` evidence
scanning and explicit caller-owned journal operations. Reconciliation reads
clean pinned Git snapshots and does not infer semantic matches or mutate
source projects. The interactive root `lds adopt <root> --facts <facts.json>`
form uses those deterministic docs operations with an advisory OpenCode Go
request and a fresh exact-plan human gate; it is not a paired skill,
real-pilot authorization, or first-release claim. The paired adoption skill
and a human-facing reconciliation facilitator remain deferred.

Built on the [toolsmith contract](https://github.com/procrastivity/toolsmith)
(`toolsmith/v1` — the manifest declares it): a single static Go binary
that projects itself into agent harnesses as generated, stamped skills.

## Install

```
nix profile install .#default   # from this checkout
lds install                     # project into every detected harness
```

`lds install <harness>` targets one harness; `lds uninstall
<harness>` removes exactly what install wrote; `lds plumbing doctor` reports
stale or drifted projections.

## Develop

```
direnv allow      # or: nix develop
make check        # lint + test
make hooks        # pre-commit, both stages
```

Version stamps come from release tags (`git describe --match 'v[0-9]*'`);
pushing an annotated `vX.Y.Z` tag is the only human release action.
CHANGELOG.md is generated per release, never committed.

## Design of record

The [engineering documentation](engineering/README.md) is the design of
record, starting with the [adoption decision](engineering/decisions/0001-adopt-two-track-documentation.md),
[source-repository map](engineering/architecture/source-repositories.md),
and [roadmap](engineering/product/roadmap.md). The [runtime handoff](engineering/product/adoption-operations-runtime-handoff.md)
documents supported `lds plumbing docs` behavior and its limits; the
[flow/plumbing map](engineering/product/adoption-flow-plumbing-map.md)
records the root form's boundaries. No real project has been authorized as
an adoption pilot; user-facing Diátaxis documentation for
LDS itself is still to be written.
The [BDS-252 engineering handoff](engineering/product/bds-252-stage-4-engineering-handoff.md)
documents the reconciliation plumbing, bounded runtime evidence, and limits;
it does not authorize a real-pair pilot or source transfer.
The GitHub CI workflow runs; the generated release workflow has not been
exercised yet.
