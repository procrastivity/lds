# lds

LDS is a CLI for a Layered Documentation System: engineering documentation
alongside Diátaxis user documentation. It has deterministic `lds plumbing docs`
inventory, local-link validation, four-asset rendering, reviewed planning,
and create-only application. Human-facing documentation workflows are not
implemented yet.

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
documents supported `lds plumbing docs` behavior and its limits. No real project has
been authorized as an adoption pilot; user-facing Diátaxis documentation for
LDS itself is still to be written.
The GitHub CI workflow runs; the generated release workflow has not been
exercised yet.
