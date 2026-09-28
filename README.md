# lds

LDS is a planned CLI for creating and maintaining a Layered Documentation
System: engineering documentation alongside Diátaxis user documentation.
This repository currently contains the generated Toolsmith application
scaffold, not the LDS documentation workflows yet.

Built on the [toolsmith contract](https://github.com/procrastivity/toolsmith)
(`toolsmith/v1` — the manifest declares it): a single static Go binary
that projects itself into agent harnesses as generated, stamped skills.

## Install

```
nix profile install .#default   # from this checkout
lds install                     # project into every detected harness
```

`lds install <harness>` targets one harness; `lds uninstall
<harness>` removes exactly what install wrote; `lds doctor` reports
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

The LDS design of record is not in this scaffold yet. The intended two-track
layout is `engineering/` for implementation decisions and `docs/` for
Diátaxis user-facing documentation. The generated CI and release workflows
target GitHub and need review before use with this Amp-hosted repository.
