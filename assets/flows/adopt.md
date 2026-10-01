# Prepare a create-only LDS documentation adoption

**Status:** Preparation source only. This flow is not an executable paired
CLI/skill delivery. No real target has been selected or authorized, and this
document does not authorize writes.

The future `lds adopt` workflow and its Claude Code presentation must consume
this one numbered source. Before that can be delivered, the provider,
endpoint, model, credential source, and authorized disposable live access
must be settled. The first step below depends on the future deterministic
`lds plumbing adopt` document verb (Toolsmith C8.4); that verb does not exist
yet. It is distinct from `lds plumbing docs plan` and from the later human
confirmation of a plan digest.

## Shared flow

1. **Read the live adoption instructions.** Obtain the readable document from
   future `lds plumbing adopt` (structured `--json` is also required). Stop if
   that document verb is unavailable; do not substitute this asset or infer
   missing behavior. This is the shared-flow entry point, not a target
   preflight and not approval.

2. **Establish the explicit target and bounded documentation inventory.**
   Ask for an existing target root; never infer it from the current directory.
   Run `lds plumbing docs inventory <root> --json`. Review scoped Markdown
   paths, hashes, supported links, and skips. Inventory covers root
   `README.md` and Markdown under `engineering/` and `docs/`; it is not a
   source-code or whole-repository scan. Symlinks in scope are skipped and a
   symlinked root/ancestor is refused. Do not traverse outside the target.

3. **Check supported local links.** Run
   `lds plumbing docs validate <root> --json`. Explain each missing relative
   file target. External URLs are not fetched, fragments are not checked,
   and other unsupported checks are skips, not passes. Broken-link findings
   alone do not veto adoption; they remain visible for review. Do not claim
   complete Markdown or external-link validation.

4. **Collect verified target facts and approval evidence.** Obtain the
   complete fact bundle required by the
   [adoption contract](../../engineering/product/adoption-contract.md) and
   [operations contract](../../engineering/product/adoption-operations-contract.md):
   owner-approved display name and purpose, current capability and sources,
   and a verified existing user operation with prerequisites, exact
   invocation and input, observed or source-backed output, limitations,
   evidence/source references, approver, and provenance. Facts describe the
   target, never LDS by default. Do not fabricate, infer, or turn plans into
   current capability. A fixture may supply explicitly test-only approval
   and source/run evidence for that disposable fixture; this is not real
   project authorization or real-pilot evidence. Stop without planning if
   any required fact is missing or unverified; ask the owner to supply or
   verify it rather than filling gaps.

5. **Render and inspect the four candidate files.** Run
   `lds plumbing docs render <root> --facts <facts.json> --json`. Review the
   exact content, SHA-256, and resolved asset name/source/hash for the four
   allowlisted outputs. Treat fact values as plain text and inspect their
   rendered Markdown context. A render/validation error stops without
   writes. Rendering is read-only and does not establish that owner claims
   are true.

6. **Build and present the full read-only preflight.** Run
   `lds plumbing docs plan <root> --facts <facts.json> --json`. Present the
   entire result, including all four output classifications and renderable
   content/hashes/provenance, necessary parent directories, preserved
   evidence, findings/skips, report-only root README signpost, veto, and
   `planDigest`. Incomplete facts produce four `refusal` classifications,
   `veto: true`, and `planDigest: null`; there is no approvable plan. Any
   collision, policy refusal, unsafe target, or legacy/ambiguous LDS under
   `docs/` vetoes every create. Preserve all occupied bytes; never migrate
   or overwrite. Broken-link diagnostics and skips alone do not veto. The
   only outputs are `engineering/README.md`,
   `engineering/product/overview.md`, `docs/README.md`, and
   `docs/reference/usage.md`; only necessary parent directories may be
   created. Root `README.md` is report-only. A read-only plan never writes.
   Continue only when `veto` is `false` and `planDigest` is non-null;
   otherwise present the refusal/collision reasons and stop without asking
   for confirmation or invoking apply.

7. **Obtain a fresh human decision for this exact plan.** Show all of the
   reviewed plan again and ask the human to approve or decline the exact
   displayed digest. Do not infer approval from facts, prior consent, a
   printed/extracted digest, or fixture metadata. A decline or noninteractive
   invocation stops with no writes. Any changed facts, assets, diagnostics,
   target, or other displayed plan field requires a new full review and
   fresh decision; never silently substitute a newly computed plan.

8. **Apply only the reviewed, eligible plan.** After explicit fresh
   confirmation, invoke
   `lds plumbing docs apply <root> --facts <facts.json> --plan <reviewed.json> --confirm <digest> --json`
   with the exact reviewed JSON and digest. Apply revalidates the plan and
   refuses stale inputs, vetoes, unsafe paths, or unsupported platform
   invariants before creation. It may exclusively create only missing
   approved files and necessary parents; it never edits, replaces, moves, or
   deletes existing content. A decline, invalid/vetoed plan, or stale
   refusal is a no-write stop. Report the actual `created`,
   `identicalUntouched`, `unattempted`, `failed`, and `preserved` results and
   hashes. If a late I/O/race failure leaves partial effects, stop, report
   exactly what is observed, do not roll back or claim transactionality, and
   require a new preflight/review before another attempt.

## Presentation boundary

The numbered actions and decision gates above are shared. A future CLI may
collect explicit arguments and display JSON; Claude Code may ask for values
and present the same results conversationally. Neither presentation may
omit the full plan, turn diagnostics into hidden decisions, or bypass the
fresh exact-digest human gate. Noninteractive execution stops without
writing. This preparation source does not claim that `lds adopt`,
`lds plumbing adopt`, a provider-backed workflow, or a projected skill is
implemented or tested, and it does not establish CLI/skill parity.

## Accepted boundaries

This flow follows [ADR-0002](../../engineering/decisions/0002-bds-251-adoption-workflow-boundary.md),
the [BDS-251 flow/plumbing map](../../engineering/product/adoption-flow-plumbing-map.md),
the [CLI surface audit](../../engineering/product/cli-surface-audit.md), and
the [runtime handoff](../../engineering/product/adoption-operations-runtime-handoff.md).
The approved behavior is create-only, limited to the four files above, with
all-create veto on collision/refusal, no migration or overwrite, and no
real-pilot claim. See the
[BDS-249 fixture handoff](../../engineering/product/adoption-stage-3-handoff.md)
for the test-only boundary. Fixtures and disposable runtime results do not
authorize or validate a real project.
