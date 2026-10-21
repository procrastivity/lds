# CLI commands

`lds --help` and each command's `--help` describe flags for your binary.
`--json` is a root persistent flag. Canonical plumbing paths (rather than
historical `lds docs ...` spellings):

| Command | Effect |
| --- | --- |
| `lds version --json` | Report compiled version, commit and build date. |
| `lds plumbing docs inventory <root> --json` | Scoped root README and `engineering/`, `docs/` Markdown inventory, links, skips. |
| `lds plumbing docs validate <root> --json` | Supported local file-target existence; findings exit 1. |
| `lds plumbing docs render <root> --facts <file> --json` | Validate facts and render four candidate files without writes. |
| `lds plumbing docs plan <root> --facts <file> --json` | Full read-only create/identical/collision/refusal plan and digest. |
| `lds plumbing docs apply <root> --facts <file> --plan <reviewed.json> --confirm <digest> --json` | Linux create-only application after separate review and exact confirmation. |
| `lds plumbing docs assess <root> --brief <file> --output <absolute-new-report.md> --json` | Linux evidence/report collection with authored proposals; exit 0 means report created, not semantic approval. No target edits. See the [brief contract and example](../how-to/assess-documentation.md). |
| `lds adopt <root> --facts <file>` | Interactive root flow with advisory provider and fresh human gate. |
| `lds plumbing adopt --json` | Read live preparatory flow; **does not execute adoption**. |
| `lds plumbing manifest --json` | Describe command surface and projection eligibility. |
| `lds plumbing doctor --json` | Check harness projection state (root `doctor` is an alias). |
| `lds install [harness]`, `lds uninstall <harness>` | Project into detected/named harnesses; remove named stamped projection. |
| `lds plumbing reconcile scan --pair <absolute-file> --left-root <absolute-dir> --right-root <absolute-dir> --map <absolute-tracked-file> [--map ...] [--journal <absolute-existing-file>]` | Scan pinned clean roots; journal comparison optional. |
| `lds plumbing reconcile journal read --pair <absolute-file> --journal <absolute-file>` | Read existing journal without writing. |
| `lds plumbing reconcile journal record --pair <absolute-file> --journal <absolute-file> --scan <absolute-saved-file> --request <absolute-file>` | Record reviewed correspondence and evidence. |
| `lds plumbing reconcile journal decide --pair <absolute-file> --journal <absolute-file> --scan <absolute-saved-file> --request <absolute-file>` | Record an explicit initial outcome. |
| `lds plumbing reconcile journal reconsider --pair <absolute-file> --journal <absolute-file> --scan <absolute-saved-file> --request <absolute-file>` | Explicitly supersede a decision; none edit sources. |

`lds manifest` is also an alias. Reconciliation successes emit JSON even
without `--json`. Exit 0 is success; 1 is validation/evidence finding,
2 usage, 3 refusal/staleness, 4 unexpected internal/I/O failure. Some docs
diagnostics emit a JSON result to stdout **and** an error to stderr: inspect
exit code and both streams. Cobra usage errors are human diagnostics. There
are no root `init`, `assess`, `update`, `review`, or `reconcile` commands;
assessment is the qualified plumbing operation above.
