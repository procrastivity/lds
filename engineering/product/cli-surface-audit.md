# BDS-251 Stage 2 Step 3: CLI surface audit

**Status:** Reviewed against the supplied BDS-251 Step 1/2 baseline. Audit
only: no `lds plumbing` namespace exists yet, and no command is moved here.

This records the audience decision for every current runnable application
leaf beneath the root command. The `docs` node is a grouping command with five
leaves; Cobra's built-in `help` and `completion` commands are framework
infrastructure, not application verbs. The recommendations follow
[ADR-0002](../decisions/0002-bds-251-adoption-workflow-boundary.md) and
[Toolsmith C8.3–C8.5](https://github.com/procrastivity/toolsmith/blob/main/CONTRACT.md#c8--shapes):
audience is expressed by command-tree placement, while kind describes the
deterministic/LLM/control-plane contract. Every current application leaf has
kind `plumbing`; that fact alone does not make it skill-projectable. Under
the future boundary, a skill receives only canonical members beneath
`lds plumbing` whose kind is also `plumbing`.

| Current spelling | Intended audience · current kind | Disposition when the namespace is implemented | Projectable under namespace **and** kind? | Existing consumer justifies compatibility alias? |
| --- | --- | --- | --- | --- |
| `lds version` | Human/operator checking build identity · `plumbing` | Retain as root porcelain. The manifest already carries the same version, commit, and date for machine consumers. | No: root-only, despite its kind. | No skill/script call site found. |
| `lds manifest` | Skill/scripts reading the machine self-description; human-readable summary is also available · `plumbing` | Canonical `lds plumbing manifest`; retain `lds manifest` as a compatibility alias. | Yes, through the canonical member only; do not project the root alias twice. | **Yes.** Shared [agent guidance](../../assets/agent-guidance.md) tells the installed skill to run `lds manifest --json`. |
| `lds install <harness>` | Human/operator installing or refreshing the generated harness skill (`<harness>` is optional for install-all mode) · `plumbing` | Retain as root porcelain. | No: root-only, despite its kind. | No. The generated skill tells the user how to reinstall after an upgrade; installation is not a skill operation. |
| `lds uninstall <harness>` | Human/operator removing the generated harness skill · `plumbing` | Retain as root porcelain. | No: root-only, despite its kind. | No skill operation calls it. |
| `lds doctor` | Human/operator diagnosis and the current Claude skill's installation-diagnosis task · `plumbing` | Retain root porcelain and provide canonical `lds plumbing doctor` as an alias pair with the same deliverable. | Yes, through the canonical member only. | **Yes.** The [Claude skill judgment](../../assets/templates/skills/claude-code/judgment.md) explicitly covers diagnosing the generated installation, and `doctor` is the corresponding projected verb. A pair must meet C8.5's output/help parity and canonical failure-diagnostic requirements. |
| `lds docs apply <root>` | Future adoption skill/script applying an explicitly reviewed plan · `plumbing` | Canonical `lds plumbing docs apply`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified; the implementation and its tests are in the unpushed baseline. |
| `lds docs inventory <root>` | Future adoption skill/script inspecting the bounded documentation surface · `plumbing` | Canonical `lds plumbing docs inventory`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified; the implementation and its tests are in the unpushed baseline. |
| `lds docs plan <root>` | Future adoption skill/script building the read-only review artifact · `plumbing` | Canonical `lds plumbing docs plan`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified; the implementation and its tests are in the unpushed baseline. |
| `lds docs render <root>` | Future adoption skill/script rendering candidate documentation · `plumbing` | Canonical `lds plumbing docs render`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified; the implementation and its tests are in the unpushed baseline. |
| `lds docs validate <root>` | Future adoption skill/script checking supported local links · `plumbing` | Canonical `lds plumbing docs validate`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified; the implementation and its tests are in the unpushed baseline. |

The `manifest` and `doctor` aliases are compatibility judgments, not
implemented registrations. No blanket move or additional alias is justified
by this audit. The future adoption skill genuinely needs the five `docs`
operations as plumbing. Under C8.4 it also needs the same-named deterministic
adoption document at `lds plumbing adopt`, paired with the future LLM shape
`lds adopt`; neither command exists today, and the paired workflow remains
deferred by ADR-0002.

## Observed projections

- `lds --help` lists the five current root leaves plus the `docs` grouping
  node (and Cobra's built-ins); `lds docs --help` lists the five docs leaves.
  The [manifest walker](../../internal/manifest/verbs.go) excludes framework
  built-ins and records runnable leaves by their current qualified spelling.
- `lds manifest --json` reports ten application leaves, all with kind
  `plumbing`; it has no audience field. `lds install claude-code` in a
  disposable skills directory produced a ten-row skill table because
  [the current projection filter](../../internal/harness/harness.go) checks
  kind alone. This is the current behavior, not the future C8.3 rule.
- After that disposable install, `lds doctor --json` reported an empty
  findings list and the Claude Code target as `current`. Doctor derives the
  expected generated tree and compares it with the host's installed tree,
  which is why the existing skill's diagnosis task has a real consumer.
- The shared guidance asset still says documentation generation/reconciliation
  commands are not implemented, although the five `docs` leaves now exist in
  this baseline. Treat that paragraph as stale scope prose, not as evidence
  that those leaves should stay out of the future adoption skill.

The root registrations, leaf annotations, and current projection are visible
in [`internal/cli/root.go`](../../internal/cli/root.go), the individual
constructors under [`internal/verbs`](../../internal/verbs), and the
[CLI end-to-end tests](../../internal/cli/e2e_test.go). This audit records
placement only; it does not implement Step 4 or establish a namespace.
