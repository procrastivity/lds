# lds

LDS is a CLI for a Layered Documentation System: engineering documentation
alongside Diátaxis user documentation. It has deterministic `lds plumbing docs`
inventory, local-link validation, four-asset rendering, reviewed planning,
create-only application, and maintenance assessment reports with
reviewer-authored proposals, plus separately reviewed existing-file update
planning/application and bounded `lds plumbing reconcile` evidence
scanning, explicit caller-owned journal operations, and an offline
`lds reconcile report` review packet. Linux root-only interactive
`lds reconcile journal record|decide|reconsider` hosts guide explicit journal
entries directly from clean pinned roots, without authored scan or request
JSON. Reconciliation reads clean pinned Git snapshots and does not infer
semantic matches or mutate source projects. The report is a private new
external file, not a journal decision or applied proof;
the identical deterministic `lds plumbing reconcile report` is the only
reconciliation report surface eligible for generated skill projection. No
assisted reconciliation flow or native skill/parity result is claimed. The
interactive root `lds adopt <root> --facts <facts.json>`
form uses those deterministic docs operations with an advisory OpenCode Go
request and a fresh exact-plan human gate. Generated Claude Code `/lds`
adoption guidance fetches `lds plumbing adopt --json` live and follows the
shared `assets/flows/adopt.md` procedure; the reader remains preparatory and
unavailable for execution, and the guidance does not enforce behavior. S10
was accepted for executor CLI cases and independent offline review, not native
skill execution or CLI/skill parity. Neither form authorizes a real pilot or
establishes first-release inclusion. Optional reconciliation assistance and a
shared human/skill flow remain deferred.

The human maintenance workflow `lds assess <root>` produces
model-assisted, review-required proposals in a new report **outside** the
target, after fresh consent to send the complete scoped documentation and
selected source contents to OpenCode Go. `lds plumbing docs collect` prepares
the same evidence offline; `lds plumbing docs assess` also accepts an observed
packet with an authored brief. Offline checks do not establish live-provider
reliability or installed-skill parity. No assessment edits the target.
See the [assessment how-to](docs/how-to/assess-documentation.md).

The offline human-facing `lds update <root> --observed <collected.json>
--request <edits.json>` displays the complete deterministic plan and requires
fresh terminal confirmation before replacing explicitly selected existing
documentation files. The separate `lds plumbing docs update-plan` /
`update-apply` route remains available for manual artifact review and apply.
Neither route implicitly accepts assessment advice or alters create-only
adoption; no real target or pilot is authorized. See the
[update how-to](docs/how-to/update-documentation.md).

`lds review <root> [--json]` is a Linux-only offline, read-only first pass over
scoped documentation: file sizes/hashes, supported local-link findings, skips
and explicit limitations on stdout. It needs no terminal, credential or
authored JSON input and writes no report. Exit 0 means no supported missing
local file target, **not** complete or current documentation. See the
[review how-to](docs/how-to/review-documentation.md).

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
The [reconciliation how-to](docs/how-to/reconcile-projects.md) includes a
disposable offline report example using an empty review brief and the guided
journal route. Recording a claim does not verify it or execute a transfer.
The [update how-to](docs/how-to/update-documentation.md) shows disposable
collect/request inputs, a human-invoked root update and the separate manual
plumbing plan/apply route. None selects a pilot or authorizes unattended edits.
The [review how-to](docs/how-to/review-documentation.md) demonstrates a broken
local link and explains why repairing it is not semantic validation.

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
authored-brief report. Its human-facing assessment workflow adds scoped
collection and model-authored proposals. The separate
[existing-file update contract](engineering/product/documentation-update-contract.md)
defines reviewed replacement; assessment itself still cannot edit files.

The [BDS-252 engineering handoff](engineering/product/bds-252-stage-4-engineering-handoff.md)
documents the reconciliation plumbing, bounded runtime evidence, and limits;
it does not authorize a real-pair pilot or source transfer.
The [reconciliation workflow](engineering/product/reconciliation-workflow.md)
records the implemented offline report and guided journal slice alongside
future assistance; neither is the full facilitator.
The GitHub CI workflow runs; the generated release workflow has not been
exercised yet.
