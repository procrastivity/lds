# BDS-251 adoption flow and plumbing map

**Status:** Acceptance map for a future, create-only `lds adopt`; not a flow
asset, implementation, or claim of paired CLI/skill execution.

This map connects accepted [ADR-0002](../decisions/0002-bds-251-adoption-workflow-boundary.md)
to the [adoption behavior contract](adoption-contract.md), the
[deterministic operations contract](adoption-operations-contract.md), and
the [BDS-250 runtime handoff](adoption-operations-runtime-handoff.md). The
contract and runtime handoff define different things: the former is the
frozen behavior boundary; the latter records tested local CLI behavior and
its limits. BDS-249 fixtures and BDS-250 disposable tests are evidence only,
not a pilot. No real project is selected or authorized.

## Deterministic step map

The five existing leaves are currently `lds docs …` plumbing verbs. If the
C8 namespace is implemented later, their anticipated qualified spellings are
`lds plumbing docs …`; none of those qualified spellings currently exists.
This names a projection target, not a blanket move of root commands. The
human-versus-agent audience and deterministic kind of every root leaf are
recorded in the [C8 surface audit](cli-surface-audit.md). The applicable
Toolsmith rules are
[C8.1–C8.5](https://github.com/procrastivity/toolsmith/blob/main/CONTRACT.md#c8--shapes),
including namespace **and** kind for projection and paired-flow rules.

| Future flow step | Currently tested leaf → anticipated qualified spelling | Inputs, facts, and approval boundary | Output, refusal, or escalation |
| --- | --- | --- | --- |
| Read the deterministic adoption document (C8.4 first step) | **No current leaf.** Future same-named `lds plumbing adopt` document verb; not implemented. | No target facts or approval are implied by reading the document. It is the distinct deterministic document prerequisite for a later shared adoption flow. | Readable Markdown by default and structured `--json`; the future CLI and Claude Code skill consume the same flow live, and each flow starts with this document. It is not `lds docs plan`, does not produce the target-specific four-output preflight, and does not collect human digest confirmation. |
| Inspect only the bounded documentation surface | `lds docs inventory <root> --json` → `lds plumbing docs inventory <root> --json`; constructor: [`internal/verbs/docs/inventory.go`](../../internal/verbs/docs/inventory.go) | Explicit existing target root. It reads root `README.md` and Markdown under `engineering/` and `docs/`, not arbitrary source files as docs. No approval is inferred. | Sorted scoped Markdown paths/hashes, supported links, and skips. Symlinks in scope are skipped; a symlinked root/ancestor is refused, never traversed. Inventory is not a full repository scan or a legacy-migration decision. |
| Check supported local links | `lds docs validate <root> --json` → `lds plumbing docs validate <root> --json`; constructor: [`internal/verbs/docs/validate.go`](../../internal/verbs/docs/validate.go) | Same explicit root and bounded supported Markdown syntax. External URLs are not fetched; fragments are not checked. | Missing relative file targets are findings (exit 1). External, fragment-only, image, and other recognized unsupported checks are skips, not passes. A link finding alone does not veto a plan; a separately established unsafe/legacy policy refusal still does. |
| Obtain target facts and render candidate docs | `lds docs render <root> --facts <facts.json> --json` → `lds plumbing docs render <root> --facts <facts.json> --json`; constructor: [`internal/verbs/docs/render.go`](../../internal/verbs/docs/render.go) | A complete bundle for this target: owner-approved name and purpose; current capability and sources; verified user operation, prerequisites, exact invocation/input, evidence and limitations; approval/provenance. Fixtures may supply test-only approval/evidence for the disposable target. The leaf checks structure and rendering, not truth or approver identity. | Exact content and SHA-256 for only the four approved files, plus resolved asset names/source/hash. Missing/unverified facts or invalid templates refuse rendering; no project writes occur. No default target facts or automatic owner approval. |
| Build the review artifact | `lds docs plan <root> --facts <facts.json> --json` → `lds plumbing docs plan <root> --facts <facts.json> --json`; constructor: [`internal/verbs/docs/plan.go`](../../internal/verbs/docs/plan.go) | Same facts and target as render; reuses current assets. Inspects relevant existing paths, scoped content, bounded link diagnostics, parent requirements, and report-only root README notice. | For renderable facts, includes the four-output classifications and content/hash/asset provenance, necessary parents, preserved evidence, findings/skips, veto, and whole-plan digest. With incomplete facts it returns four `refusal` classifications, `veto: true`, and `planDigest: null`: there is no approvable plan and no render content/provenance. Any other collision/refusal vetoes every create; link skips/findings alone do not veto. The leaf is read-only. |
| Present and decide on the complete plan | **No current plumbing leaf.** Future `lds adopt` must show the entire reviewed plan and obtain a fresh human decision tied to its exact digest. | Human sees all four outputs and all diagnostics/preservation evidence. Approval cannot be inferred from fixture metadata, a previous decision, or automatically extracting the digest. | A decline or noninteractive invocation stops without writes. Changed plan means new full review and a new decision. This is the owner-facing gate, not a machine leaf or a current prompt. |
| Apply only the reviewed eligible plan | `lds docs apply <root> --facts <facts.json> --plan <reviewed.json> --confirm <digest> --json` → `lds plumbing docs apply <root> --facts <facts.json> --plan <reviewed.json> --confirm <digest> --json`; constructor: [`internal/verbs/docs/apply.go`](../../internal/verbs/docs/apply.go) | Exact reviewed plan JSON and a confirmation string for its digest. The orchestrator must obtain a fresh human decision; the leaf checks the supplied digest but cannot authenticate the person or infer consent. Facts/assets/root and target state must still match. Apply re-plans before writes; stale state, veto, decline, or unsafe target refuses before creation. | Linux exclusive no-follow creation of only missing approved files/necessary parents. Reports actual created, identical, unattempted, failed and preserved paths/hashes. A late error may leave partial new effects; report them, do not roll back or imply cross-file atomicity. Other platforms refuse where the invariant is unavailable. |

The qualified spellings above are anticipated only. The current leaf
constructors are annotated `plumbing` in the root `docs` tree; no namespace,
`lds adopt`, flow asset, LLM provider, or skill integration is implemented by
this map. C8 requires the future human-facing workflow and projected
Claude Code skill to share one flow, but ADR-0002 defers shipping that pair
until one settled provider/endpoint/model/credential source and authorized
disposable live access exist. Actual paired execution and parity remain
blocked; do not report them as tested.

## Asymmetric disposable acceptance scenarios

These rows state how the future flow must sequence the existing deterministic
operations. The current CLI/runtime observations are documented in the
[runtime handoff](adoption-operations-runtime-handoff.md); the
[acceptance matrix](adoption-operations-acceptance.md) and
[preservation cases](../../internal/asset/testdata/adoption/preservation/cases.json)
pin disposable inputs/expected classifications. None establishes CLI/skill
parity or real-project usability.

| Disposable scenario | Expected flow result and boundary |
| --- | --- |
| Greenfield Wordcount target | Complete test-only owner facts and source/run evidence allow a four-`create` plan with only necessary parents. A confirmed first application creates those four files; a newly reviewed identical rerun is `current`, with four `identical-untouched` and zero writes. The target's `sample.txt` and `word-count.sh` stay byte-identical. |
| Partial engineering | Preserve the existing `engineering/decisions/` file. Missing approved outputs remain eligible: four creates, with only `docs/`, `docs/reference/`, and `engineering/product/` as necessary parents. Missing optional layers alone are not a refusal. |
| One differing approved output | Three other absent outputs may still classify `create`, but a differing `docs/reference/usage.md` is `different-collision`; veto all creates. The plan reports occupied bytes for manual review outside this contract; applying a vetoed plan refuses with no writes. Before/after fixture bytes prove preservation; the refused apply's empty `preserved` array is not preservation evidence. No migration or overwrite. |
| Recognizable, ambiguous, or partial legacy LDS under `docs/` | Refuse all four candidates and perform no migration or writes. An ADR explicitly rooting a partial legacy tree at `docs/` remains a refusal despite absent layers; `docs/reference/` alone is not conclusive legacy evidence. The vetoed apply's empty `preserved` array is not preservation evidence; fixture snapshots establish no bytes changed. |
| Already-generated identical versus edited-looking | Four byte-identical outputs are `identical-untouched`, zero creates. A changed/generated-looking occupied output is a collision, not proof of ownership; veto the other creates and preserve the edited bytes. |
| Declined digest | Show the complete plan, then stop: no creates and eligible creates remain unattempted. A refusal may have an empty `preserved` result; disposable before/after bytes, not that field, establish unchanged existing content. Do not treat a supplied/printed digest as approval unless the human freshly confirms that exact plan. |
| Stale facts, assets, diagnostics, target, or unsafe ancestor | Refuse before creates (exit 3 under the operations contract) and require a new plan/review. Retain the previously reviewed plan as the caller's artifact, not as current evidence. A completed fresh preflight supplies current preserved hashes in the refusal result; if it cannot be completed, do not present old hashes as observed. Never silently re-plan and apply changed content. Symlink targets are not followed. |
| Late creation/I/O failure | If failure occurs after some exclusive creates, report `partial`, observed created hashes, the failed path and hash when readable, and remaining unattempted paths (internal/I/O exit 4 for injected I/O). Preserve pre-existing files. Do not delete prior creates or claim a transaction; require a new preflight. |

The map does not add candidate outputs, infer missing facts, choose a real
pilot, relax C1's all-write veto, or promote the planned workflow into the
first release. It records orchestration expectations only; the accepted
contracts remain authoritative.
