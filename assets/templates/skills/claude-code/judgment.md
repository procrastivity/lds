Use this generated Claude Code skill to inspect LDS or, when explicitly
asked, prepare a create-only documentation adoption for one named target.
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

## Guardrails while following the live flow

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
