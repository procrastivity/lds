# BDS-251 CLI surface audit

**Status:** Stage 2 Step 3 audit approved; Stage 2 Step 4 placement is
implemented. The original rows record the pre-Step-4 root leaves; Stage 4
Step 8 adds the separately classified root LLM form shown as the final row.
Step 6 also added the preparation-only `lds plumbing adopt` reader.

This records the audience decision for every runnable application leaf in the
pre-Step-4 root command. Its `docs` node was a grouping command with five
leaves; Cobra's built-in `help` and `completion` commands are framework
infrastructure, not application verbs. Placement follows
[ADR-0002](../decisions/0002-bds-251-adoption-workflow-boundary.md) and
[Toolsmith C8.3–C8.5](https://github.com/procrastivity/toolsmith/blob/main/CONTRACT.md#c8--shapes):
audience is expressed by command-tree placement, while kind describes the
deterministic/LLM/control-plane contract. Every audited leaf has kind
`plumbing`; kind alone does not make a verb skill-projectable. Projection now
requires both a canonical `lds plumbing` path and kind `plumbing`.

| Audited or added spelling | Intended audience · kind | Current registration: root porcelain or canonical plumbing member | Projectable under namespace **and** kind? | Real existing consumer justifies root alias? |
| --- | --- | --- | --- | --- |
| `lds version` | Human/operator checking build identity · `plumbing` | Retain `lds version` as root porcelain. The manifest carries the same version, commit, and date for machine consumers. | No: root-only, despite its kind. | No skill/script call site found. |
| `lds manifest` | Skill/scripts reading the machine self-description; human-readable summary is also available · `plumbing` | Canonical `lds plumbing manifest`; retain `lds manifest` as a root compatibility alias. | Yes, canonical member only; the alias is not projected. | **Yes.** Previously generated skills and shared guidance called `lds manifest --json`; retaining the alias lets already-installed older skill versions continue working. Current guidance prefers the canonical path. |
| `lds install <harness>` | Human/operator installing or refreshing the generated harness skill (`<harness>` is optional for install-all mode) · `plumbing` | Retain `lds install <harness>` as root porcelain. | No: root-only, despite its kind. | No. The generated skill tells the user how to reinstall after an upgrade; installation is not a skill operation. |
| `lds uninstall <harness>` | Human/operator removing the generated harness skill · `plumbing` | Retain `lds uninstall <harness>` as root porcelain. | No: root-only, despite its kind. | No skill operation calls it. |
| `lds doctor` | Human/operator diagnosis and the current Claude skill's installation-diagnosis task · `plumbing` | Canonical `lds plumbing doctor`; retain `lds doctor` as a root compatibility alias. | Yes, canonical member only; the alias is not projected. | **Yes.** Previously generated skills expose root `doctor` for installation diagnosis; preserving that path keeps existing generated skill versions working. |
| `lds docs apply <root>` | Future adoption skill/script applying an explicitly reviewed plan · `plumbing` | Canonical `lds plumbing docs apply`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified. |
| `lds docs inventory <root>` | Future adoption skill/script inspecting the bounded documentation surface · `plumbing` | Canonical `lds plumbing docs inventory`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified. |
| `lds docs plan <root>` | Future adoption skill/script building the read-only review artifact · `plumbing` | Canonical `lds plumbing docs plan`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified. |
| `lds docs render <root>` | Future adoption skill/script rendering candidate documentation · `plumbing` | Canonical `lds plumbing docs render`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified. |
| `lds docs validate <root>` | Future adoption skill/script checking supported local links · `plumbing` | Canonical `lds plumbing docs validate`; no root alias. | Yes, through the canonical member. | No current shipped consumer identified. |
| `lds adopt <root> --facts <facts.json>` | Human/operator running the bounded CLI-side LLM adoption form · `llm` | Retain as root porcelain; it is separate from the read-only `lds plumbing adopt` reader. | No: root-only and kind `llm`; both gates exclude it. | No projected skill or shipped consumer calls it; no compatibility alias. |

The `manifest` and `doctor` aliases use the same constructors as their
canonical members. The manifest records `alias-of` only on these two root
rows. No blanket move or speculative alias was added. An adoption skill
needs the five docs operations as plumbing and the same-named deterministic
document `lds plumbing adopt`, a read-only reader backed by the one live flow
asset. Stage 4 Step 8 now adds a distinct root `lds adopt` LLM form; it is
not a plumbing member and is not included in the generated skill. This does
not establish CLI/skill parity or revise ADR-0002's first-release and pilot
gates.

## Observed projections

- `lds --help` lists human root commands including LLM-kind `adopt`, the two compatibility aliases, and
  the `plumbing` group; it does not list docs leaves. `lds plumbing --help`
  lists the preparation-only `adopt` reader, `manifest`, `doctor`, and the
  `docs` group; `lds plumbing docs --help` lists the five docs leaves.
  Cobra's built-ins remain framework help only.
- `lds plumbing manifest --json` reports nineteen application leaves, of
  which thirteen qualified plumbing-kind members are projectable, and records
  `alias-of` only on root `manifest` and `doctor`. Root `adopt` is kind `llm`
  and is not projectable. The [manifest walker](../../internal/manifest/verbs.go) excludes
  framework built-ins and records canonical qualified paths for plumbing
  members.
- `lds install claude-code` in a disposable skills directory produces thirteen
  table rows: canonical `plumbing adopt`, `plumbing manifest`, `plumbing doctor`,
  the five `plumbing docs …` members, and the five `plumbing reconcile …`
  members. The [projection filter](../../internal/harness/harness.go)
  checks namespace and kind, excluding root aliases and root-only
  `version`, `install`, `uninstall`, and LLM-kind root `adopt`. The generated skill is stamped and
  `lds plumbing doctor --json` reports the target as `current`.
- Alias tests pin byte-identical stdout/stderr and exit codes in text and JSON
  modes, help differing only by command path, and plumbing-canonical failure
  diagnostics. Docs payload values such as `"operation":"docs plan"` remain
  unchanged; deterministic adoption behavior is exercised through
  `lds plumbing docs plan/apply`.

The root registrations, leaf annotations, and projection are visible in
[`internal/cli/root.go`](../../internal/cli/root.go), the constructors under
[`internal/verbs`](../../internal/verbs), and the
[CLI end-to-end tests](../../internal/cli/e2e_test.go). Stage 4 Step 8 adds
the host-gated root form in `internal/verbs/adoptflow`; the existing plumbing
reader remains read-only. The installed skill still projects only deterministic
plumbing and does not execute the paired adoption flow. No skill parity,
real pilot, or first-release inclusion is claimed.
