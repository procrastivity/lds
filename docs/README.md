# LDS user guide

LDS inventories and validates a bounded documentation layer, can propose four
create-only adoption files for a project, and scans explicitly pinned pairs of
Git projects for reconciliation evidence. It also collects maintenance evidence
and reviewer-authored proposals in a report without editing the target, and
separately supports reviewed existing-file updates.
The `lds assess` workflow adds model-authored proposals after
explicit consent to disclose the complete scoped source-content packet; the
offline authored-brief path remains available without a provider.

Use an existing local CLI build or, with Nix installed, build from an LDS
checkout:

```sh
nix build .#default
./result/bin/lds --help
export PATH="$PWD/result/bin:$PATH"
```

The final line makes `lds` available for the examples in your current shell.
This is a checkout build, not a published download or release-install
instruction. You do not need an agent harness or provider key for the offline
tutorial or deterministic plumbing.

## Choose a route

- [Inspect a disposable project](tutorials/inspect-a-project.md) — a read-only,
  offline first run.
- [Maintain a disposable documentation layer](tutorials/maintain-documentation.md)
  — one offline Wordcount fixture connecting init, review, authored assessment,
  a separately approved full-file update and final review; no provider or
  real-project authorization.
- [Review documentation offline](how-to/review-documentation.md) — one root
  command for scoped inventory and local-link findings, without a report or
  semantic verdict.
- [Check documentation in CI](how-to/check-documentation-in-ci.md) — a Linux
  checkout-build recipe preserving review JSON, findings, skips and actual
  exit status outside the inspected root; the repository CI runs the same
  wrapper after its Nix build.
- [Draft, resume and review adoption facts offline](reference/facts.md#draft-facts-with-the-offline-helper)
  — Linux terminal questionnaire and selected-field resume emit pending JSON;
  separate full-bundle review records fresh named facts approval, never plan approval.
- [Initialize documentation offline](how-to/initialize-a-project.md) — Linux
  terminal-only `lds init <root> --facts <file> [--json]`: full plan review and
  fresh exact-digest confirmation for four create-only adoption files, without
  a provider, key or LLM. Implemented and verified with disposable fixtures;
  disposable facts are not real-project approval or release/pilot evidence.
- [Adopt documentation in a project](how-to/adopt-a-project.md) — prepare facts,
  inspect a separately saved plumbing plan and make a fresh human decision,
  or use the distinct advisory-provider `lds adopt` route.
- [Assess documentation](how-to/assess-documentation.md) — an offline
  authored-brief report, collect/observed binding, and a consent-gated
  model workflow; all reports stay outside the project and never apply edits.
- [Update existing documentation](how-to/update-documentation.md) — use the
  unchanged offline collect/request route, or explicitly select existing files
  for consent-gated assisted drafting followed by a separate exact-plan review
  and human apply decision. The manual plumbing plan/apply route remains
  available. No real target is authorized by this guide.
- [Reconcile two projects](how-to/reconcile-projects.md) — scan clean pinned
  snapshots, produce an offline or consent-gated assisted review report, and
  keep an explicit review journal separately.
- [Command reference](reference/commands.md) and [fact bundle reference](reference/facts.md).
- [Safety and scope](explanation/safety-and-scope.md) — what the checks and
  write boundaries do *not* guarantee.

## Optional agent-harness projection

`lds plumbing manifest --json` describes the CLI; only eligible plumbing
verbs are projected into generated skills. `lds install` projects into
detected harnesses, or `lds install claude-code` targets Claude Code alone.
`lds plumbing doctor --json` reports target state and findings;
`lds uninstall claude-code` removes a stamped installation. Install refuses
hand-edited, unstamped, or incompatible targets by default; inspect before
considering its destructive `--force` option. Installation is separate from
initializing or adopting a project. Root `facts` and `init` are plumbing-kind
but neither is projected or an alias; they add no plumbing leaf or shared model flow.
The generated `/lds` guidance reads the live flow via
`lds plumbing adopt --json`, but that reader is preparatory, not an adoption
execution. Installing or discovering a skill does not establish native
execution or parity with the root CLI.

The assessment projection likewise uses the live
`flows/assess.md` through `lds plumbing assess --json`; that reader only
returns a preparatory flow, and installing guidance does not prove native
assessment execution or CLI/skill parity.

Assisted update guidance fetches the live `flows/update.md` through
`lds plumbing update --json` rather than copying its sequence. That reader is
preparatory, not an update execution. The skill harness must disclose its own
processing and obtain appropriate transfer and write consent; it must not
automate the root terminal or treat model output as approval. Installation
does not establish native execution or CLI/skill parity.

Reconciliation guidance fetches the live preparatory
`lds plumbing reconcile flow --json` reader. Deterministic `collect` and
baseline-bound `report --observed --draft --receipt` are plumbing steps, not
permission for a skill to automate the root terminal. A skill must disclose
its own harness processing and obtain its own consent. Projection does not
establish native execution, provider reliability or CLI/skill parity.
