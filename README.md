# lds

LDS is a CLI for a Layered Documentation System: engineering documentation
alongside Diátaxis user documentation. It has deterministic `lds plumbing docs`
inventory, local-link validation, four-asset rendering, reviewed planning,
create-only application, and maintenance assessment reports with
reviewer-authored proposals, plus separately reviewed existing-file update
planning/application and bounded `lds plumbing reconcile` evidence
scanning, explicit caller-owned journal operations, and an offline
`lds reconcile report` review packet with optional consent-gated assistance.
Linux root-only interactive
`lds reconcile journal record|decide|reconsider` hosts guide explicit journal
entries directly from clean pinned roots, without authored scan or request
JSON. Reconciliation reads clean pinned Git snapshots and does not infer
semantic matches or mutate source projects. The report is a private new
external file, not a journal decision or applied proof;
the deterministic `lds plumbing reconcile report` is eligible for generated
skill projection. `lds plumbing reconcile collect` captures an original bounded
baseline, and `lds plumbing reconcile flow --json` reads the live preparatory
flow. Assisted reports retain reviewer assertions separately from unverified
model proposals; no native skill/parity result is claimed. The
interactive root `lds adopt <root> --facts <facts.json>`
form uses those deterministic docs operations with an advisory OpenCode Go
request and a fresh exact-plan human gate. Generated Claude Code `/lds`
adoption guidance fetches `lds plumbing adopt --json` live and follows the
shared `assets/flows/adopt.md` procedure; the reader remains preparatory and
unavailable for execution, and the guidance does not enforce behavior. S10
was accepted for executor CLI cases and independent offline review, not native
skill execution or CLI/skill parity. Neither form authorizes a real pilot or
establishes first-release inclusion.

The offline `lds init <root> --facts <file> [--json]` is a Linux terminal-only
host of the same four-file create-only adoption plan and apply, without a
provider, key, flow reader or LLM. It requires an existing explicit root and
complete approved facts, displays the full plan on stderr and accepts only a
freshly typed `apply sha256:<exact displayed digest>` or `decline` after review.
Collisions or incomplete facts refuse before prompting; no confirmation flag
or stdin-pipe bypass exists. See the
[disposable initialization how-to](docs/how-to/initialize-a-project.md).
This is implemented and verified offline, not release,
pilot or native-skill/parity evidence; facts approval is not plan approval.

The human maintenance workflow `lds assess <root>` produces
model-assisted, review-required proposals in a new report **outside** the
target, after fresh consent to send the complete scoped documentation and
selected source contents to OpenCode Go. `lds plumbing docs collect` prepares
the same evidence offline; `lds plumbing docs assess` also accepts an observed
packet with an authored brief. Offline checks do not establish live-provider
reliability or installed-skill parity. No assessment edits the target.
See the [assessment how-to](docs/how-to/assess-documentation.md).

The offline `lds update <root> --observed <collected.json>
--request <edits.json>` remains provider-free and unchanged. Explicit
`lds update <root> --assist --reason <text> --audience <text>
--document <relative-existing-markdown> [--document ...] [--source <relative-file> ...]`
instead collects bounded evidence, discloses the complete model payload for
fresh `send` consent, and returns unverified full-file drafts and questions.
No edits is a valid review-required result. Nonempty drafts receive a complete
deterministic plan and require a *separate* freshly typed exact-digest apply
decision. The separate `lds plumbing docs update-plan` / `update-apply` route
remains available. None implicitly accepts assessment advice or alters
create-only adoption; no real target or pilot is authorized. See the
[update how-to](docs/how-to/update-documentation.md).

`lds review <root> [--json]` is a Linux-only offline, read-only first pass over
scoped documentation: file sizes/hashes, supported local-link findings, skips
and explicit limitations on stdout. It needs no terminal, credential or
authored JSON input and writes no report. Exit 0 means no supported missing
local file or supported heading target, **not** complete or current documentation.
Heading checks use a bounded ASCII ATX subset; unsupported syntax remains
explicitly skipped. See the [review how-to](docs/how-to/review-documentation.md),
[CI recipe](docs/how-to/check-documentation-in-ci.md) and
[connected offline maintenance tutorial](docs/tutorials/maintain-documentation.md).

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

## Project-local runtime assets (BDS-234 port)

LDS accepts an explicit global `--project-root PATH` for
runtime asset reads. For a known shipped asset name, it tries
`<canonical-root>/.lds/assets/<shipped-relative-name>` before the existing
user-XDG → installed → embedded chain. For example, selecting `/work/acme`
lets adoption render use `/work/acme/.lds/assets/templates/adoption/docs/README.md`;
the flag is not inferred from the current directory, Git, environment,
an init/adoption/assessment/update target, or reconciliation pair roots. A missing
project file falls through for that asset only; unsafe or unreadable
selected paths refuse rather than fall back. See the
[command reference](docs/reference/commands.md#project-root-runtime-asset-selection)
and [safety boundaries](docs/explanation/safety-and-scope.md#project-local-asset-boundary).

The selection applies to adoption templates (including root `init`) and the
live adoption, assessment, update and reconciliation flows where their runtime
readers opt in. Init reads templates only, not a flow. It does not change
manifest/projection-digest identity, configuration, install,
uninstall, doctor or generated installed skill bytes. A generated skill must
carry an explicitly human-selected root to its live flow and template
plumbing calls, not embed repository content at installation time. Existing
consent, plan approval and freshness gates still apply; this is a local
integration, not a pilot, provider test, native skill/parity finding or
contract-reconciliation claim.

## Use LDS

Start with the [user documentation](docs/README.md) for an offline inspection
tutorial, adoption and reconciliation how-tos, command reference, and the
current safety boundaries. The [initialization how-to](docs/how-to/initialize-a-project.md)
uses disposable test-only facts for offline terminal review and create-only
application; it authorizes no real target. The [assessment how-to](docs/how-to/assess-documentation.md)
collects evidence and proposed text outside the target without editing it.
The [reconciliation how-to](docs/how-to/reconcile-projects.md) includes a
disposable offline report example using an empty review brief and the guided
journal route, plus bounded assisted and external-artifact examples without a
provider call. Recording a claim does not verify it or execute a transfer.
The [update how-to](docs/how-to/update-documentation.md) distinguishes the
provider-free collect/request route, consent-gated assisted drafting and the
separate manual plumbing plan/apply route. None selects a pilot or authorizes
unattended edits.
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
[initialization contract](engineering/product/initialization-workflow.md)
bounds the offline root host, exact in-memory plan review and existing
create-only apply, with disposable verification and explicit safety limits.
The
[maintenance-workflow design](engineering/product/documentation-maintenance-workflow.md)
records the accepted assessment/report boundary and implemented deterministic
authored-brief report. Its human-facing assessment workflow adds scoped
collection and model-authored proposals. The separate
[existing-file update contract](engineering/product/documentation-update-contract.md)
defines reviewed replacement and the assisted update boundary; assessment
itself still cannot edit files. Generated Claude Code guidance reads the live
preparatory `lds plumbing update --json` flow rather than owning another
sequence. This does not establish native execution or CLI/skill parity.

The [BDS-252 engineering handoff](engineering/product/bds-252-stage-4-engineering-handoff.md)
documents the reconciliation plumbing, bounded runtime evidence, and limits;
it does not authorize a real-pair pilot or source transfer.
The [reconciliation workflow](engineering/product/reconciliation-workflow.md)
records the implemented offline and assisted report and guided journal slices;
none is a full facilitator or real-pair authority decision.
The GitHub CI workflow runs; the generated release workflow has not been
exercised yet.
