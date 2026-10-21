# lds

LDS is a CLI for a Layered Documentation System: engineering documentation
alongside Diátaxis user documentation. It has deterministic `lds plumbing docs`
inventory, local-link validation, four-asset rendering, reviewed planning,
create-only application, and maintenance assessment reports with
reviewer-authored proposals, plus bounded `lds plumbing reconcile` evidence
scanning and explicit caller-owned journal operations. Reconciliation reads
clean pinned Git snapshots and does not infer semantic matches or mutate
source projects. The interactive root `lds adopt <root> --facts <facts.json>`
form uses those deterministic docs operations with an advisory OpenCode Go
request and a fresh exact-plan human gate. Generated Claude Code `/lds`
adoption guidance fetches `lds plumbing adopt --json` live and follows the
shared `assets/flows/adopt.md` procedure; the reader remains preparatory and
unavailable for execution, and the guidance does not enforce behavior. S10
was accepted for executor CLI cases and independent offline review, not native
skill execution or CLI/skill parity. Neither form authorizes a real pilot or
establishes first-release inclusion. A human-facing reconciliation facilitator
remains deferred.

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

## Use LDS

Start with the [user documentation](docs/README.md) for an offline inspection
tutorial, adoption and reconciliation how-tos, command reference, and the
current safety boundaries. The [assessment how-to](docs/how-to/assess-documentation.md)
collects evidence and proposed text outside the target without editing it.
These guides describe implemented behavior; they
do not select an adoption pilot or add automatic maintenance of existing
documentation.

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
records the root form's boundaries. The [S10 preparation record](engineering/product/bds-251-stage-4-step10-preparation.md)
sets out the reduced-scope acceptance and evidence limits; the
[BDS-251 engineering handoff](engineering/product/bds-251-stage-4-engineering-handoff.md)
records offline regressions and remaining limits. No real project has been
authorized as an adoption pilot. The
[maintenance-workflow design](engineering/product/documentation-maintenance-workflow.md)
records the accepted assessment/report boundary and implemented deterministic
collection; automatic drafting and existing-file updates remain unimplemented.
The [BDS-252 engineering handoff](engineering/product/bds-252-stage-4-engineering-handoff.md)
documents the reconciliation plumbing, bounded runtime evidence, and limits;
it does not authorize a real-pair pilot or source transfer.
The GitHub CI workflow runs; the generated release workflow has not been
exercised yet.
