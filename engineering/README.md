# LDS engineering documentation

This is the design of record for people and agents building the `lds` tool.
It follows [ADR-0001](decisions/0001-adopt-two-track-documentation.md):
engineering decisions and plans live here; [user documentation](../docs/README.md)
lives in `docs/` and follows Diátaxis. The tool has a Toolsmith scaffold and
harness-management commands plus nine deterministic `lds plumbing docs`
leaves, including separate existing-file update planning/application,
as well as deterministic `lds plumbing reconcile`
scan, explicit journal operations, bounded original report collection, a
deterministic report primitive and a preparatory live flow reader. Root
`lds reconcile report` is LLM-kind because explicit `--assist` adds
consent-gated advice; its default offline behavior remains provider-free.
Only qualified plumbing forms are generated-skill-table eligible.
Reconciliation is bounded to pinned
evidence and a caller-owned journal. Linux root-only guided
`lds reconcile journal record|decide|reconsider` hosts now collect explicit
reviewer assertions from a terminal, show full digest-bound previews and
require fresh confirmation before journal writes; they do not require authored
scan/request JSON, edit sources or validate external applied proof. The
interactive root
`lds adopt <root> --facts <facts.json>` form uses the live shared flow and
those deterministic docs operations. Generated Claude Code `/lds` adoption
guidance fetches that flow live through `lds plumbing adopt --json`; the reader
is preparatory and the guidance is not behavioral enforcement. S10 accepts
executor CLI evidence and independent offline review, not native adoption or
CLI/skill parity. No pilot or first-release claim follows.

The separate Linux-only offline `lds review <root> [--json]` is a root-only
plumbing-kind composition of existing documentation inventory and local-link
validation, now including bounded ASCII ATX heading targets with explicit
unsupported-fragment skips. It writes no report and does not infer coverage or semantics; it
adds no plumbing leaf, projection or shared LLM flow. See the
[disposable review how-to](../docs/how-to/review-documentation.md) and
[safety boundaries](../docs/explanation/safety-and-scope.md).
The [CI recipe](../docs/how-to/check-documentation-in-ci.md) is opt-in, not an
installed workflow; the [connected tutorial](../docs/tutorials/maintain-documentation.md)
uses only one disposable offline project. Neither requires a pilot.

The offline Linux terminal-only `lds init <root> --facts <file> [--json]`
hosts the existing four-output adoption plan and create-only apply. It needs
an existing explicit root and approved facts, no provider/key/flow reader/LLM.
Full canonical plan JSON goes to stderr; veto/incomplete facts refuse before
prompting, otherwise queued input is flushed and a fresh exact-digest human
decision gates `adoption.ApplyReviewed` of the displayed in-memory plan and
same optional `--project-root` selector. It adds no temporary plan, plumbing
leaf, shared model flow, alias or projected entry; root is plumbing-kind.
See the [bounded initialization contract](product/initialization-workflow.md)
and [disposable how-to](../docs/how-to/initialize-a-project.md).
This is implemented and verified offline, not pilot,
release or native-parity evidence. The resulting manifest is 36 total leaves,
22 qualified projected entries and nine docs leaves; historical counts in
older handoffs remain historical.

Start with the [roadmap](product/roadmap.md) for proposed capabilities and
their status. The [source-repository map](architecture/source-repositories.md)
records what other projects contribute and how their local checkouts are found.
The [within-project maintenance design](product/documentation-maintenance-workflow.md)
records the accepted assessment-first scope. `lds plumbing docs assess`
collects bounded evidence and authored proposals in an external report;
it does not infer semantic impacts, call a model or edit target files. Its
human-facing implementation adds the `lds assess` flow,
offline `docs collect`, and an optional observed packet for `docs assess`.
The root CLI and generated skill use the same live `flows/assess.md`; the
plumbing flow reader does not execute it. Model-authored proposals remain
unverified advice, not target edits or native skill/CLI parity evidence.
The [existing-file update contract](product/documentation-update-contract.md)
describes implemented `docs update-plan` / `docs update-apply` plumbing and
both the unchanged offline and explicit assisted `lds update` modes, with
exact-byte review, fresh digest confirmation and bounded per-file replacement.
The root registration is LLM-kind because assistance can request a model, but
offline mode remains provider-free. The live preparatory `lds plumbing update`
reader supplies `flows/update.md` to generated Claude Code guidance, not an
executable update or native parity result; the
[update how-to](../docs/how-to/update-documentation.md) uses a disposable
target. The [human-facing reconciliation workflow](product/reconciliation-workflow.md)
describes the offline/assisted report and guided journal slices. The assisted
root host discloses selected full text and the complete payload before fresh
send consent; draft claims remain unverified and separate from seed and journal.
`lds plumbing reconcile collect` and extended baseline-bound `report` supply
deterministic steps; `flow --json` reads the live preparatory shared flow. The
three journal hosts remain plumbing-kind but root-only and not projected. No real-target update,
source transfer, real-pair journal authority, pilot or native skill/parity
result is authorized or established by these documents. The roadmap also
records the local LDS port of the source-checked Toolsmith BDS-234 asset
selector, including initialization's template reads.
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
