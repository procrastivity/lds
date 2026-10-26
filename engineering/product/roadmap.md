# Building LDS as a Toolsmith application

This is a direction and candidate command surface, **not a released CLI
specification**. Track implementation in the
[LDS Linear project](https://linear.app/beausimensen/project/lds-22892af0659b);
record settled behavior in decisions, specs, and reference docs here.

## Current state

`toolsmith new --name lds` provided a Go CLI, embedded assets and override
resolution, and harness management. The root porcelain retains `version`,
`install`, and `uninstall`; the canonical plumbing members are
`lds plumbing manifest` and `lds plumbing doctor`, with reviewed root
aliases. These commands manage the CLI and projected agent skills; they do
not assess or reconcile project documentation; BDS-251 Stage 4 adds a
separate explicit adoption form described below.
The original seven bounded `lds plumbing docs` leaves inventory, check
supported local links, render the four adoption assets, preview a plan, and exclusively create
missing approved files after reviewed digest confirmation, plus collect
maintenance evidence, compute source hashes, and bind authored proposals to
observations in an external assessment report.
That deterministic assessment does not infer semantic impacts or edit target
files. The root `lds adopt <root> --facts <facts.json>` CLI form now uses the
shared live flow, OpenCode Go / `deepseek-v4-flash`, and those deterministic
operations. It requires owner-supplied facts and interactive exact-digest human
confirmation; the model is advisory and has no tools or source-file access.
This is not an installed-skill parity result, a named pilot, or first-release
inclusion.
Stage 4 Step 9 added generated Claude Code `/lds` adoption guidance: it reads
`lds plumbing adopt --json` live and delegates sequencing to
`assets/flows/adopt.md`. The reader still reports `preparatory` and
`unavailable-for-execution`; the guidance is not behavioral enforcement.
S10 is accepted under reduced scope for completed executor CLI cases and
independent High offline review, without independent live CLI replay or native
skill execution/parity; neither further test wave is required for this
acceptance. See the [S10 preparation record](bds-251-stage-4-step10-preparation.md)
for evidence limits, the [flow/plumbing map](adoption-flow-plumbing-map.md)
for bounded behavior, and the
[BDS-251 engineering handoff](bds-251-stage-4-engineering-handoff.md) for offline
regressions and observed limits. The
[runtime handoff](adoption-operations-runtime-handoff.md) covers deterministic
documentation operations. No real pilot is established.

BDS-252 adds `lds plumbing reconcile scan` and the explicit
`lds plumbing reconcile journal read|record|decide|reconsider` operations for
strict, deterministic evidence inspection and caller-owned decision history.
The first offline reconciliation report slice adds `lds reconcile report` and
deterministic `lds plumbing reconcile report`: Linux-only, bounded pinned evidence
plus a reviewer-authored brief in a new private external Markdown report with
provenance JSON. It never writes sources or the journal. The root default
remains offline; explicit `--assist` adds a consent-gated OpenCode Go request
for unverified proposals, so root report is LLM-kind and not projected.
`lds plumbing reconcile collect` captures the bounded original baseline,
extended plumbing `report --observed --draft --receipt` validates against it,
and `lds plumbing reconcile flow --json` reads the live preparatory flow.
Only qualified plumbing leaves enter the generated skill table. None is a
native execution or parity finding.
The Linux-only root `lds reconcile journal record|decide|reconsider` adds a
guided offline terminal route to the existing journal plumbing. It freshly
scans pinned evidence, collects explicit human claims and authority/outcome,
displays the complete digest-bound request/evidence/journal preview and gates
each write on a fresh exact confirmation. No authored scan/request files are
needed; the manual plumbing route remains available. These root-only
plumbing-kind leaves are not projected and do not execute the report flow. They do not
verify applied proof or authorize source transfer, a real pair or a pilot.
The human-facing assessment workflow, bounded existing-file update plumbing,
unchanged offline root `lds update`, and explicit assisted update mode are
implemented. The live `lds plumbing update` reader supplies a preparatory
shared flow to generated Claude Code guidance. The offline initialization
host below is implemented and verified with disposable offline fixtures.
The resulting manifest has 36 leaves,
including 22 qualified plumbing-kind projection entries and nine `docs`
leaves; these counts do not establish native execution or parity.
Reconciliation commands do not
infer semantic correspondence, choose real-pair authority or edit source
repositories. Assisted reports and shared preparatory guidance are not a
full facilitator or native skill/parity result. See the
[Stage 4 engineering handoff](bds-252-stage-4-engineering-handoff.md) for the
audited surface and boundaries.

The [frozen adoption contract](adoption-contract.md) retains a pre-runtime
roadmap-status paragraph as historical text. Its accepted adoption behavior
is unchanged; this current-state section and the linked runtime handoffs
supersede that paragraph only as descriptions of implementation status.

## User documentation and assessment proceed without a pilot

The [user documentation](../../docs/README.md) describes current commands and
limits, with disposable offline examples. Writing these guides manually is
not an LDS adoption pilot. Local WIP Matter `lds-user-documentation` tracks
the first guide; `lds-maintenance-workflow-design` tracks the separate
[within-project maintenance design](documentation-maintenance-workflow.md).
Neither deliverable requires another owner-driven live test wave or reopens
S10's reduced acceptance.

The owner accepted assessment and reviewable text outside the target as the
first implementation scope. Local Matter `lds-maintenance-assessment` tracks
`lds plumbing docs assess`: repeated inventory/link checks, explicitly hashed
source evidence, and caller-authored proposals/questions in a new report.
It needs no provider, pilot or target edits. Semantic impact analysis and
drafting remain reviewer responsibilities on this deterministic route.
Local Matter `lds-human-maintenance-assessment` tracks the implemented root
`lds assess` workflow: automatic scoped collection, fresh consent to transfer
the complete packet to OpenCode Go, one bounded tool-free model request, and
unverified proposed text in an external report. Offline
`lds plumbing docs collect` and optional `--observed` on `docs assess` bind an
authored brief to the full baseline. The root CLI and projected skill use the
same live `flows/assess.md`; `lds plumbing assess` only reads that preparatory
flow. Mocked terminal/provider tests and disposable offline examples cover
reporting, consent/refusal, stale evidence, invalid drafts, preservation and
live flow resolution. No live provider/native execution, parity, release
inclusion or owner checkpoint wave is claimed or required for this slice.

Bounded `docs update-plan` and `docs update-apply` are implemented under the separate
[update contract](documentation-update-contract.md), with full exact-byte
review and a fresh digest gate. Local WIP Matter
`lds-human-update-workflow` tracks the offline Linux root `lds update` host:
complete plan display on stderr, fresh terminal-only digest approval and the
same strict in-memory apply. It takes the original collected packet and
explicit edits, without a provider, new baseline or temporary artifact.
This is not approval for any real target. Explicit `lds update --assist`
selects 1–32 unique existing scoped Markdown targets, collects the original
bounded baseline, discloses the complete system+user payload for fresh `send`,
and asks one fixed tool-free OpenCode Go / `deepseek-v4-flash` request for
unverified full replacements and questions. Empty edits end review-required
without apply. Nonempty drafts receive strict original-baseline planning and
full exact-byte preview, then separate fresh `apply sha256:<digest>` or
`decline`. Baseline and live flow rechecks surround transfer and precede apply;
send is not write authority. The Linux replacement and partial/indeterminate
limits remain. The root registration is now LLM-kind even though its default
offline route remains provider-free and behaviorally unchanged. The new live
`lds plumbing update [--json]` flow reader is preparatory only; generated
guidance fetches it live and must disclose harness processing and obtain its
own consent without terminal automation or model-derived approval. No real
provider, target/pilot, native execution or parity is claimed by mock and
disposable verification. Broader semantic review remains deferred; root
`init` is implemented locally under the bounded contract below.
Local WIP Matter `lds-offline-review` tracks the authorized first Linux-only
`lds review <root> [--json]` slice. It composes existing inventory and
local-link validation on stdout without authored inputs, a terminal, provider,
report artifact or target write. Exit 0 only means no supported missing local
file target; it does not certify coverage, semantics, cross-track consistency,
code alignment or runtime behavior. The [disposable how-to](../../docs/how-to/review-documentation.md)
shows its bounded use. No pilot, owner checkpoint wave or live test follows.
Pilot selection and first-release inclusion remain separate decisions.

## Offline initialization: bounded local implementation

The owner authorized `lds init <root> --facts <file> [--json]` as a Linux
terminal-only, provider-free composition of the existing four-output
adoption planner and `adoption.ApplyReviewed`. It requires an explicit
existing root and complete approved facts, displays the full canonical
`docs plan` JSON on stderr and refuses veto/incomplete facts before prompting
or writes. Otherwise it flushes queued input and accepts only fresh
`apply sha256:<exact displayed digest>` or `decline` alone; other text/EOF
refuses, with no confirmation-flag or stdin-pipe bypass. The exact displayed
in-memory plan and same optional global `--project-root` selector go to apply;
no temporary plan, provider/key/flow reader or LLM is involved.

Identical outputs remain untouched; any differing occupied output or legacy
LDS evidence vetoes all creates. No migrations, root README edits, Git
commands, harness installation, extra outputs or empty layers. An identical
rerun requires a new review/confirmation and returns `current`. Existing
facts/template/source/root/parent stale-input checks and per-file exclusive
creation govern: partial effects are possible, without rollback or automatic
retry. Facts approval neither authenticates claims nor approves the plan.
Init is root-only plumbing-kind, not projected or an alias, and adds no
plumbing leaf or shared model flow. See the
[bounded contract](initialization-workflow.md) and
[disposable how-to](../../docs/how-to/initialize-a-project.md).
Status is **implemented and verified offline**, not a
pilot, release or native-parity finding; historical handoffs stay unchanged.

## Separate update implementation and assisted reconciliation design

The owner requested these two design workstreams after the assessment workflow
was delivered. They proceed without selecting a pilot or reopening S10:

- **Reviewed existing-file updates:** local Matter
  `lds-existing-documentation-updates` tracks implementation following the
  `lds-reviewed-documentation-updates` design and
  [update contract](documentation-update-contract.md). The implemented first
  slice replaces only explicitly selected existing Markdown files after full
  before/after review and fresh digest-bound approval. Creates, deletes, moves,
  inferred ownership and automatic assessment acceptance are not authorized.
  No real-target write or pilot is authorized; adoption remains create-only.
- **Human-facing reconciliation:** local Matter
  `lds-human-reconciliation-workflow` tracks the
  [workflow contract](reconciliation-workflow.md). Its offline report produces
  a private external review packet from explicit pinned pair/map, reviewer
  brief and optional journal evidence, without writing source or journal.
  The separate guided journal slice can write only a confirmed caller-owned
  external journal; local Matter `lds-guided-reconciliation-journal` tracks
  its implementation and disposable verification. Assisted report advice and
  the preparatory shared CLI/skill flow are implemented; real-pair authority
  remains deferred.

The implemented update plumbing and root modes do not grant
permission to edit a real project. Any live provider access, real target write, pilot or
release needs its own authorization. Existing
assessment and reconciliation plumbing remain usable.

## Planning constraints and open decisions

Settled: the two-track layout is the intended default, and within-project
documentation maintenance is distinct from cross-project reconciliation.
For any LLM workflow LDS ships, [Toolsmith's shape contract](https://github.com/procrastivity/toolsmith/blob/main/CONTRACT.md#c8--shapes)
requires one flow asset shared by the CLI verb and projected skill; every
deterministic step must be available through plumbing. Do not design the
two forms as independently maintained workflows.

The following product choices remain open; they are not prerequisites for the
implemented deterministic plumbing and must be settled before any real use:

- **First usable release:** Which project is the pilot, and is the bounded
  four-file offline `lds init` sufficient for that release? The create-only
  output/preservation contract is settled; broader adoption and real-target
  usefulness are not established by local implementation.
- **Existing-file write safety for a real target:** Decide the authorized
  project/paths/reviewer and accept inode/timestamp replacement there. The
  bounded update contract requires exact before/after approval, stale-input
  refusal, per-file replacement and partial-result recovery. Adoption's
  create-only policy does not grant update permission.
- **LLM release and pairing:** Step 8 implements the CLI-side form using the
  selected provider and Step 9 projects adoption guidance into Claude Code.
  S10's reduced-scope acceptance is not native adoption or a parity PASS;
  first-release inclusion remains a separate choice. Do not infer provider
  reliability, Anthropic support, or resolution of the original failed S7
  attempt from this evidence.
- **Reconciliation application:** Which real pair, if any, is an authorized
  pilot? Who may declare pair/topic authority and permitted directions, and
  what external evidence is sufficient to verify an applied transfer?
  ADR-0003 and the v1 contract define the journal boundary and state model,
  not these real-world values. Xcind's ledger remains evidence, not LDS's
  universal authority rule.

## Candidate work, in dependency order

1. Specify the initial two-track project layout and select/adapt templates
   from the reference repositories. Ship defaults as embedded assets with
   the existing installed/user override chain. Repository-local overrides
   are now available upstream through
   [Toolsmith BDS-234](https://linear.app/beausimensen/issue/BDS-234/support-repository-local-overrides-for-generated-tools-assets-and),
   merged in [`92fcfcf`](https://github.com/procrastivity/toolsmith/commit/92fcfcf85418d95afed6e8952a74a6421ac99f15)
   ([PR #6](https://github.com/procrastivity/toolsmith/pull/6)). On 2026-10-22,
   the orb's Toolsmith checkout and installed binary were updated/rebuilt
   from that main, with resolver, CLI and generated-tool tests passing.
   Optional C5.5 uses explicit global `--project-root PATH` selection of
   `<canonical-root>/.lds/assets/<shipped-relative-name>` and
   project → user-XDG → installed → embedded precedence for eligible runtime
   assets only. **The LDS port is implemented locally**; updating the
   generator alone does not rewrite its existing generated resolver. The LDS
   integration opts the four adoption templates into docs
   render/plan/apply and root init/adopt (init reads templates only), and the adoption, assessment, update
   and reconciliation flows into their plumbing readers and root adopt,
   assess, assisted update and assisted report forms. No positional target or
   reconciliation pair root implies the selector; no cwd/Git/environment
   discovery is in scope. A missing selected project path falls through per
   file; invalid roots, symlinks below the canonical root, nonregular paths
   and read errors refuse without fallback, as does malformed content at its
   consuming reader. Trust selected directory/writers; there is no hostile
   concurrent-writer sandbox guarantee. Project source/path/hash provenance
   belongs in runtime observations and their freshness/approval checks.
   Generated guidance must carry a human-selected root unchanged into live
   flow readers **and** adoption template plumbing, not bake repository bytes
   into installed skills. Manifest/projection digest/config/install/uninstall/
   doctor and installed skill bytes remain project-independent even if the
   flag is supplied. Consent, exact-plan approval and staleness checks remain;
   this is local implementation, not a pilot, provider/native-parity result
   or contract reconciliation. See the [command reference](../../docs/reference/commands.md#project-root-runtime-asset-selection)
   and [safety boundary](../../docs/explanation/safety-and-scope.md#project-local-asset-boundary).
2. Implement deterministic operations for inventorying files and links,
   validating the chosen layout, and rendering/scaffolding templates.
   Provide machine-readable results and tests before agent-driven flows.
3. Add human-facing flows and agent-specific skills that use those
   operations to adopt LDS, keep documentation aligned with code, and
   review both tracks. The create-only `lds adopt` CLI form is implemented
   under BDS-251 Stage 4 Step 8; Step 9 implements generated Claude Code
   adoption guidance, but native execution and CLI/skill parity remain
   unverified. Keep the skill projection derived from the CLI manifest and
   shipped assets.
4. **Implemented, bounded:** specify and implement deterministic cross-project
   reconciliation plumbing separately from within-project documentation
   maintenance. The v1 scan reports pinned evidence; explicit journal requests
   record reviewed correspondence and decisions without source edits. This is
   not semantic matching, a real-pair decision, or a facilitator. Xcind's
   [cross-project procedure](https://github.com/scinddev/xcind/blob/main/engineering/maintenance/cross-project-sync.md)
   and ledgers remain reference evidence, not a schema copied into LDS.
5. **Offline/assisted report and guided journal slices implemented:** create a bounded
   report separately from journal writes and any authorized pilot; guide
   explicitly reviewed record/decide/reconsider journal entries with terminal
   preview and confirmation. Assistance uses one bounded disclosed request
   and a live preparatory flow; no model claim becomes authority.
   Disposable source-derived and synthetic fixtures do not validate a live
   adoption, port, or parity claim.

## Candidate verbs

Names and boundaries below are proposals except the marked implemented rows.
Porcelain is the human-facing workflow;
plumbing is deterministic and suitable for JSON/exit-code use by skills and
scripts. `toolsmith new` bootstraps the *CLI application*; the bounded `lds init`
bootstraps four documentation files *inside an existing explicit project*.

| Surface | Candidate | Purpose |
| --- | --- | --- |
| Porcelain (implemented offline) | `lds init <root> --facts <file> [--json]` | Linux terminal-only offline four-output adoption host: complete plan, fresh exact-digest approval, same in-memory create-only apply and optional global asset selector. Root-only plumbing-kind, no projection/alias/new plumbing leaf/shared flow. See the [contract](initialization-workflow.md). |
| Porcelain (implemented, outside first-release claim) | `lds assess <root> --reason <text> --audience <text> --output <absolute-new-report.md> [--source <relative-file> ...] [--json]` | Collect scoped documentation and named source bytes, ask fresh transfer consent, and request reviewable model-authored maintenance proposals; no target edits. |
| Porcelain (implemented offline mode) | `lds update <root> --observed <collected.json> --request <edits.json> [--json]` | Unchanged Linux terminal host of the complete deterministic plan and fresh exact-digest human gate; reuses original packet and authored edits, then strict in-memory apply. No provider or temporary plan/report. Whole root registration is LLM-kind due to the explicit assisted mode, but this mode stays offline; root is not projected. See the [update how-to](../../docs/how-to/update-documentation.md). |
| Porcelain (implemented assisted mode) | `lds update <root> --assist --reason <text> --audience <text> --document <relative-existing-markdown> [--document ...] [--source <relative-file> ...] [--json]` | Explicit bounded original collection, full outbound disclosure and fresh send consent, one tool-free request for unverified full drafts/questions, strict original-baseline exact-byte plan and independent fresh apply gate. No inferred edit selection or real-target authority. |
| Porcelain (implemented first offline slice) | `lds review <root> [--json]` | Linux read-only scoped inventory and local-file link findings, skips and limitations on stdout. Root-only plumbing-kind composition, not projected or an alias; no report, completeness or semantic verdict. Broader review remains deferred. |
| Porcelain (implemented, outside first-release claim) | `lds adopt <root> --facts <facts.json>` | Interactively review a create-only four-output documentation plan and apply only after fresh exact-digest human consent. |
| Porcelain (implemented offline slice) | `lds reconcile report --pair <absolute-pair.json> --left-root <absolute-dir> --right-root <absolute-dir> --map <absolute-tracked-file> [--map ...] [--journal <absolute-existing-journal.json>] --review <absolute-review.json> --output <absolute-new-report.md> [--json]` | Linux private external report from clean pinned evidence and reviewer assertions; no model, source/journal writes, or project commands. See the [how-to](../../docs/how-to/reconcile-projects.md). |
| Porcelain (implemented assisted mode) | `lds reconcile report --assist` (same required flags) | Full payload/destination disclosure and fresh send consent, one tool-free OpenCode Go request, strict unverified proposed claims/questions and attributed receipt in private report; original baseline retained, no source/journal writes. Root registration LLM-kind, not projected. |
| Porcelain (implemented guided offline slice) | `lds reconcile journal record\|decide\|reconsider --pair <absolute-pair.json> --left-root <absolute-dir> --right-root <absolute-dir> --map <absolute-tracked-file> [--map ...] --journal <absolute-external-journal.json> [--json]` | Linux terminal evidence review, explicit reviewer claims, complete digest-bound preview and fresh operation-specific confirmation; only the named journal may be written. Root-only plumbing-kind, not projected. See the [how-to](../../docs/how-to/reconcile-projects.md#guide-a-journal-entry-in-the-terminal). |
| Plumbing (implemented, bounded) | `lds plumbing docs inventory`, `validate`, `render`, `plan`, `apply` | Report scoped files/links, check supported local targets, render four assets, preview and create missing approved files without an LLM. |
| Plumbing (implemented, bounded) | `lds plumbing docs assess` | Collect documentation and explicitly hashed source evidence, link findings and authored proposals/questions in one new external report; no provider, semantic inference or target edits. See the [design](documentation-maintenance-workflow.md). |
| Plumbing (implemented, bounded) | `lds plumbing docs collect`; `lds plumbing docs assess --observed <collected.json>` | Offline full baseline packet with automatic source hashes; optional observed binding for a strict authored brief and external report. |
| Plumbing (implemented, bounded) | `lds plumbing docs update-plan <root> --observed <collected.json> --request <edits.json> --json`; `lds plumbing docs update-apply <root> --plan <reviewed.json> --confirm <sha256:digest> --json` | Existing selected Markdown only; full digest-bound review, fresh approval and per-file Linux replacement. No real target or pilot authorized. See the [how-to](../../docs/how-to/update-documentation.md). |
| Plumbing flow reader (implemented) | `lds plumbing assess [--json]` | Read live `flows/assess.md` content/source/hash; preparatory, unavailable for execution. |
| Plumbing flow reader (implemented) | `lds plumbing update [--json]` | Read live `flows/update.md` content/source/hash; preparatory, unavailable for execution. Generated guidance fetches it live. |
| Plumbing (implemented, bounded) | `lds plumbing reconcile scan`; `lds plumbing reconcile journal read`, `record`, `decide`, `reconsider` | Inspect explicitly scoped pinned artifacts/maps and persist caller-reviewed correspondence and outcomes; never infer semantic matches or write a source. See the [engineering handoff](bds-252-stage-4-engineering-handoff.md). |
| Plumbing (implemented, bounded) | `lds plumbing reconcile collect` (same base flags); `lds plumbing reconcile report` (same offline flags, optionally all of `--observed --draft --receipt`) | Collect bounded original evidence without publishing; report validates externally drafted advice against that baseline and publishes privately. Both eligible for skill table; no provider or native/parity claim. |
| Plumbing flow reader (implemented) | `lds plumbing reconcile flow --json` | Read live preparatory reconciliation flow; unavailable for execution. Generated guidance must disclose harness processing and obtain its own consent, not automate root terminal. |

Agent-specific skills worth evaluating: adopting a partial tree, drafting
and reviewing an ADR, documentation impact review after code changes,
Diátaxis placement review, and a reconciliation facilitator. Their prompts
should use the same shipped/overridable assets and plumbing as the CLI,
not become a second independent implementation.
