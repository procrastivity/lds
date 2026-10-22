Use this generated Claude Code skill to inspect LDS or, when explicitly
asked, prepare a create-only documentation adoption or a report-only
maintenance assessment, or separately review existing-file updates for one
named target.
The steps below are guidance, not a security boundary: this installation
cannot authenticate the owner, verify their claims, enforce approval, or
prove that commands are being used safely. No real target or pilot is
authorized by this skill.

## Adoption route

1. Before inspecting or acting on a target, run exactly
   `lds plumbing adopt --json`. Require schema version 1, the expected
   `preparatory` status and `unavailable-for-execution` availability, and
   nonempty live `content`. The unavailable-for-execution value is expected:
   it says this verb only reads the document, not that the document is
   unavailable. Stop on a reader error, invalid/unexpected result, or missing
   content. The reader provides neither approval nor target preflight.
   Follow the current returned flow's complete numbered procedure as the
   sole source of workflow sequencing; do not rely on prior conversation,
   copied text, or a flow embedded in this skill.

## Assessment route

When asked to assess documentation impacts or propose maintenance, first run
`lds plumbing assess --json`. Require schema version 1, operation `assess`,
`preparatory` status, `unavailable-for-execution` availability and nonempty
live content. Stop on reader errors or unexpected content; follow that live
flow as the sole source of assessment sequencing, not a cached copy here.

- Disclose this harness's processing of selected documentation/source content
  and obtain appropriate consent before reading it. Do not infer source read
  scope, reuse adoption consent, or equate skill installation with permission.
- Use the deterministic collector and preserve its original JSON outside the
  target. Draft only evidence-linked proposals/questions; preserve the reason,
  audience and declared source hashes. Report generation must use that original
  packet with `--observed`, never a silently refreshed baseline.
- An assessment produces only a new external review report. Never use adoption
  apply, a model-supplied command or this skill to edit the target. Root
  `lds assess` is a separate terminal/provider form, not a subprocess to drive
  with relayed input. No native execution or CLI/skill parity is claimed.

## Existing-file update route

When separately authorized to draft existing-file replacements, first run
`lds plumbing update --json`. Require schema version 1, operation `update`,
`preparatory` status, `unavailable-for-execution` availability and nonempty
live content. Follow that returned shared flow, not a cached sequence here.
Disclose this harness's processing and obtain appropriate permission before
reading target content. Preserve the original collected packet, explicit
selected document scope and evidence-linked unverified draft outside the
target. Only the deterministic update planner may produce the reviewed plan;
only separate fresh human approval of the complete exact plan/digest permits
the update apply plumbing. Sending content, model output, assessment advice
and prior approval never authorize replacement. Do not drive or relay input
to root `lds update --assist`. No native execution or parity is established.

## Reconciliation report route

When explicitly asked to review a named project pair, first run
`lds plumbing reconcile flow --json`. Require schema version 1, operation
`reconcile`, `preparatory` status, `unavailable-for-execution` availability and
nonempty live content. Follow that flow as the sole source of sequencing.
Disclose this harness's actual content processing and obtain appropriate
permission before reading. Preserve the original deterministic collect packet
privately outside both roots/Git directories; never silently refresh a stale
baseline. Model advice is proposed, not reviewed correspondence, authority,
direction, applied proof or approval. Do not mislabel this harness/model as
OpenCode Go or fabricate a transfer receipt. Stop if the actual provider and
consent mechanism cannot satisfy the live procedure; offline reporting remains
available. Reports never edit sources or initialize, lock or write a journal.
Keep later human journal decisions separate. Do not automate or relay input to
root `lds reconcile report --assist`. No native execution or parity is claimed.

## Adoption guardrails while following the live flow

These constraints do not define another sequence; use the live document for
the order and mechanics of each operation.

- Require an explicit existing target and complete, verified, owner-supplied
  facts with approval, evidence, and provenance, including the verified
  existing user operation. Do not infer, invent, complete, or rewrite facts.
  Structural fields do not authenticate the owner or prove truth. Stop when
  required facts/evidence are missing or uncertain.
- Stay within the live flow's bounded inventory/link checks and explicit
  skips. Findings and skipped checks are review diagnostics, not passes;
  findings/skips alone do not veto. Do not traverse symlinks or leave the
  target. Only the flow's four fixed create-only outputs are in scope.
  Preserve occupied bytes: any collision, legacy/ambiguous layout, unsafe
  path, or policy refusal vetoes every create. Never migrate, replace, or
  overwrite.
- For an eligible preview, display the entire exact plan: all four output
  classifications and prospective content, content hashes and asset
  provenance, necessary parents, preserved evidence, findings, skips,
  report-only notice, veto, and digest. Proceed to a human decision only
  when `veto` is false and `planDigest` is non-null; otherwise explain the
  refusal and stop without asking for confirmation or invoking apply.
- Any approval must be a fresh human decision for that completely displayed
  digest. Never infer it from facts, model output, fixture metadata, old
  messages, copied/extracted digests, or prior decisions. Decline, ambiguity,
  or noninteractive use means no writes. Changed target, facts, assets,
  diagnostics, or any displayed field requires a new complete review and
  decision; do not silently substitute a new plan. Keep any plan artifact
  outside the target and apply only the exact reviewed plan JSON and digest
  through the live flow's prescribed plumbing interface.
- Report actual created, identical, refused, partial, failed, preserved, and
  unattempted effects. On stale refusal stop; on partial effects report what
  is observed without rollback or a transactionality claim, and require a
  new preflight/review before another attempt.

Root `lds adopt` is a separate provider-backed LLM CLI form and is not
projected here. These guardrails are guidance, not behavioral enforcement;
they do not authenticate owners or verify facts. Installed-skill execution,
endpoint behavior, and CLI/skill parity remain for the separate S10 runtime
matrix. This is preparation only, not a first-release or real-pilot
authorization.
