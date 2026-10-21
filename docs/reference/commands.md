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
| `lds plumbing docs collect <root> --reason <text> --audience <text> [--source <relative-file> ...] [--json]` | Offline scoped collection, automatic hashes and complete UTF-8 contents in a schema-1 packet; no report, provider or target writes. |
| `lds plumbing docs assess <root> --brief <draft.json> --observed <collected.json> --output <absolute-new-report.md> [--json]` | Bind a strict authored brief to the complete collected baseline before creating a new external review report. The existing no-`--observed` form remains supported. |
| `lds plumbing docs update-plan <root> --observed <collected.json> --request <edits.json> --json` | Linux read-only plan for explicitly selected existing UTF-8 Markdown; complete exact before/after, evidence, metadata, diagnostics and digest on stdout only. Caller persists it privately outside the target. |
| `lds plumbing docs update-apply <root> --plan <reviewed.json> --confirm <sha256:digest> --json` | Linux staged same-parent, per-file replacement after full artifact review and fresh exact digest confirmation; structured actual effects, not a transaction or automatic retry. See the [update how-to](../how-to/update-documentation.md). |
| `lds update <root> --observed <collected.json> --request <edits.json> [--json]` | Offline Linux human-facing update: display the complete deterministic plan on stderr, refuse veto before prompting, flush queued input and accept only freshly typed `apply sha256:<exact displayed digest>` or `decline`. No confirmation flag, provider, new collection or temporary plan/report artifact. Uses the same strict in-memory apply. |
| `lds assess <root> --reason <text> --audience <text> --output <absolute-new-report.md> [--source <relative-file> ...] [--json]` | Linux interactive model-assisted proposals; displays complete transfer for fresh `send` consent, then creates only an external review report. Requires `OPENCODE_API_KEY`; sends scoped source content. |
| `lds plumbing assess [--json]` | Return live `flows/assess.md` content/source/hash as `preparatory`/`unavailable-for-execution`; does not run assessment. |
| `lds adopt <root> --facts <file>` | Interactive root flow with advisory provider and fresh human gate. |
| `lds plumbing adopt --json` | Read live preparatory flow; **does not execute adoption**. |
| `lds plumbing manifest --json` | Describe command surface and projection eligibility. |
| `lds plumbing doctor --json` | Check harness projection state (root `doctor` is an alias). |
| `lds install [harness]`, `lds uninstall <harness>` | Project into detected/named harnesses; remove named stamped projection. |
| `lds plumbing reconcile scan --pair <absolute-file> --left-root <absolute-dir> --right-root <absolute-dir> --map <absolute-tracked-file> [--map ...] [--journal <absolute-existing-file>]` | Scan pinned clean roots; journal comparison optional. |
| `lds reconcile report --pair <absolute-pair.json> --left-root <absolute-dir> --right-root <absolute-dir> --map <absolute-tracked-file> [--map ...] [--journal <absolute-existing-journal.json>] --review <absolute-review.json> --output <absolute-new-report.md> [--json]` | Linux offline deterministic review-required Markdown report with provenance JSON appendix; private new external file (0600), no overwrite, model, project command or source/journal write. |
| `lds plumbing reconcile report` (same flags) | Identical deterministic report operation; only qualified plumbing form is generated-skill-table eligible, not proof of native execution or parity. |
| `lds plumbing reconcile journal read --pair <absolute-file> --journal <absolute-file>` | Read existing journal without writing. |
| `lds plumbing reconcile journal record --pair <absolute-file> --journal <absolute-file> --scan <absolute-saved-file> --request <absolute-file>` | Record reviewed correspondence and evidence. |
| `lds plumbing reconcile journal decide --pair <absolute-file> --journal <absolute-file> --scan <absolute-saved-file> --request <absolute-file>` | Record an explicit initial outcome. |
| `lds plumbing reconcile journal reconsider --pair <absolute-file> --journal <absolute-file> --scan <absolute-saved-file> --request <absolute-file>` | Explicitly supersede a decision; none edit sources. |

The report brief requires `schemaVersion: 1`, nonempty `purpose` and
`reviewer`, an asserted RFC3339 `at` string, and `selections`, `claims`, and
`questions` arrays (empty is valid). Each selection is
`{"side":"left"|"right","path":"<scoped file or map>"}` and opts in
full UTF-8 content. Authored claims require `candidateId`, `status`
(`reviewed` or `proposed`), v1 `correspondence`, `rationale`, v1
`provenance` array, v1 `assessment`, `conflicts` array and `questions` array;
these are assertions, not verified matches. See the
[disposable example](../how-to/reconcile-projects.md).
Report scope is at most 64 declared scoped files plus 16 maps. Every existing
scoped/map raw file is limited to 64 KiB, including unselected files; selected
UTF-8 content totals at most 512 KiB, and pair/review/optional journal total
at most 2 MiB. Paths for pair, review, optional existing journal and new
output must be absolute and outside both roots and their Git common
directories; the output path must have no symlink components. Roots must be
clean and match full pins. Original inputs are revalidated before publication.

The update request is strict schema-1 UTF-8 JSON:
`{"schemaVersion":1,"edits":[{"path":"README.md","after":"exact full new UTF-8 content\n","rationale":"nonempty reason","citations":["README.md"]}]}`.
Only existing root `README.md` or Markdown below `docs/` and `engineering/`
can be selected. Unknown, duplicate, case-aliased or null fields refuse; max
32 edits, 8 MiB per selected file/after and 32 MiB per input JSON. The observed
input must be an original `docs collect` packet, not an authored report. The
planner does not create the reviewed file: the host must use an external,
trusted private directory, exclusive no-follow creation and existing parents.
The entire artifact, including the original observation and diagnostics, is
digest-bound; do not automate confirmation. Apply may return `applied`,
`refused`, `partial` or `indeterminate`; inspect actual effects and recover
with a new observation/plan after uncertainty.
The result separates `replaced`, observed `untouched`, path-string
`unattempted`, and `failed` (phase/code and optional actual observation), plus
any residual `scratch` and `cleanupError`. It never promises rollback.
Root `lds update` takes the same original collected packet and explicit edits,
not an assessment report or inferred proposal. With `--json`, final stdout is
schema-1 JSON containing `operation: "update"`, `status`, the complete `plan`, and `apply`
(null if vetoed or declined); plan display, prompt and errors go to stderr.
Without `--json`, stdout is a human status summary.
Wrong input or EOF refuses. It is root-only plumbing-kind, not projected:
only qualified `plumbing ...` members are eligible. It is not a same-deliverable
alias for either canonical deterministic step. There is no update flow asset
or preparatory reader for this offline host.

`lds manifest` is also an alias. Scan and journal successes emit JSON even
without `--json`; report prints a human summary by default and JSON with
`--json`. Exit 0 is success; 1 is validation/evidence finding,
2 usage, 3 refusal/staleness, 4 unexpected internal/I/O failure. Some docs
diagnostics emit a JSON result to stdout **and** an error to stderr: inspect
exit code and both streams. Cobra usage errors are human diagnostics. Root
`init` and `review` remain candidates; model-assisted update orchestration is
deferred. Root `reconcile` has only the offline `report` slice, not an assisted
facilitator. The manifest has 28 application leaves, 19 qualified
projected plumbing-kind entries and nine docs leaves; projection eligibility
is not evidence of native execution or parity. The root
LLM-kind command is not a deterministic projected member. See the
[assessment how-to](../how-to/assess-documentation.md) for the separate
provider-free and consent-gated routes.
