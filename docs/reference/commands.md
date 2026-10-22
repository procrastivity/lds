# CLI commands

`lds --help` and each command's `--help` describe flags for your binary.
`--json` is a root persistent flag. Canonical plumbing paths (rather than
historical `lds docs ...` spellings):

| Command | Effect |
| --- | --- |
| `lds version --json` | Report compiled version, commit and build date. |
| `lds plumbing docs inventory <root> --json` | Scoped root README and `engineering/`, `docs/` Markdown inventory, links, skips. |
| `lds plumbing docs validate <root> --json` | Supported local file-target existence; findings exit 1. |
| `lds review <root> [--json]` | Linux-only offline read-only root command composing canonical inventory and validate; scoped files, local-link findings, combined skips and limitations on stdout, no report artifact. See the [disposable how-to](../how-to/review-documentation.md). |
| `lds plumbing docs render <root> --facts <file> --json` | Validate facts and render four candidate files without writes. |
| `lds plumbing docs plan <root> --facts <file> --json` | Full read-only create/identical/collision/refusal plan and digest. |
| `lds plumbing docs apply <root> --facts <file> --plan <reviewed.json> --confirm <digest> --json` | Linux create-only application after separate review and exact confirmation. |
| `lds plumbing docs assess <root> --brief <file> --output <absolute-new-report.md> --json` | Linux evidence/report collection with authored proposals; exit 0 means report created, not semantic approval. No target edits. See the [brief contract and example](../how-to/assess-documentation.md). |
| `lds plumbing docs collect <root> --reason <text> --audience <text> [--source <relative-file> ...] [--json]` | Offline scoped collection, automatic hashes and complete UTF-8 contents in a schema-1 packet; no report, provider or target writes. |
| `lds plumbing docs assess <root> --brief <draft.json> --observed <collected.json> --output <absolute-new-report.md> [--json]` | Bind a strict authored brief to the complete collected baseline before creating a new external review report. The existing no-`--observed` form remains supported. |
| `lds plumbing docs update-plan <root> --observed <collected.json> --request <edits.json> --json` | Linux read-only plan for explicitly selected existing UTF-8 Markdown; complete exact before/after, evidence, metadata, diagnostics and digest on stdout only. Caller persists it privately outside the target. |
| `lds plumbing docs update-apply <root> --plan <reviewed.json> --confirm <sha256:digest> --json` | Linux staged same-parent, per-file replacement after full artifact review and fresh exact digest confirmation; structured actual effects, not a transaction or automatic retry. See the [update how-to](../how-to/update-documentation.md). |
| `lds update <root> --observed <collected.json> --request <edits.json> [--json]` | Offline Linux human-facing update: display the complete deterministic plan on stderr, refuse veto before prompting, flush queued input and accept only freshly typed `apply sha256:<exact displayed digest>` or `decline`. No confirmation flag, provider, new collection or temporary plan/report artifact. Uses the same strict in-memory apply. |
| `lds update <root> --assist --reason <text> --audience <text> --document <relative-existing-markdown> [--document ...] [--source <relative-file> ...] [--json]` | Explicit Linux assisted mode: collect bounded original evidence, show complete system+user payload for fresh `send` consent, request unverified full replacements/questions from OpenCode Go, then show the complete original-baseline deterministic plan for a separate fresh exact-digest apply decision. Empty edits end review-required without apply. See the [update how-to](../how-to/update-documentation.md). |
| `lds plumbing update [--json]` | Read live `flows/update.md` content/source/hash as preparatory/unavailable-for-execution; does not run an update. Generated Claude Code guidance fetches this flow live. |
| `lds assess <root> --reason <text> --audience <text> --output <absolute-new-report.md> [--source <relative-file> ...] [--json]` | Linux interactive model-assisted proposals; displays complete transfer for fresh `send` consent, then creates only an external review report. Requires `OPENCODE_API_KEY`; sends scoped source content. |
| `lds plumbing assess [--json]` | Return live `flows/assess.md` content/source/hash as `preparatory`/`unavailable-for-execution`; does not run assessment. |
| `lds adopt <root> --facts <file>` | Interactive root flow with advisory provider and fresh human gate. |
| `lds plumbing adopt --json` | Read live preparatory flow; **does not execute adoption**. |
| `lds plumbing manifest --json` | Describe command surface and projection eligibility. |
| `lds plumbing doctor --json` | Check harness projection state (root `doctor` is an alias). |
| `lds install [harness]`, `lds uninstall <harness>` | Project into detected/named harnesses; remove named stamped projection. |
| `lds plumbing reconcile scan --pair <absolute-file> --left-root <absolute-dir> --right-root <absolute-dir> --map <absolute-tracked-file> [--map ...] [--journal <absolute-existing-file>]` | Scan pinned clean roots; journal comparison optional. |
| `lds reconcile report --pair <absolute-pair.json> --left-root <absolute-dir> --right-root <absolute-dir> --map <absolute-tracked-file> [--map ...] [--journal <absolute-existing-journal.json>] --review <absolute-review.json> --output <absolute-new-report.md> [--json]` | Unchanged Linux offline deterministic review-required private external Markdown report (0600), no model, overwrite, source or journal write. |
| `lds reconcile report --assist` (same required pair/roots/repeatable maps/review/output and optional journal) | Explicit Linux terminal-assisted report; disclose full outbound system/user content and provider destination, flush queued input, require fresh exact `send`, then request unverified proposed claims/questions. No source or journal write. Requires `OPENCODE_API_KEY` only in this mode. |
| `lds plumbing reconcile collect` (same base flags including review/output and optional journal) | Offline bounded original evidence JSON on stdout, without publication at the named future report output or provider. Save stdout privately; preserve it as the baseline with that output path. |
| `lds plumbing reconcile report` (same offline base flags) | Deterministic offline report operation; also accepts `--observed <original-collect.json> --draft <model-draft.json> --receipt <receipt.json>` **together** for baseline-bound validation and reporting. Only qualified plumbing members are skill-table eligible, not proof of native execution or parity. |
| `lds plumbing reconcile flow --json` | Live reconciliation flow content/source/hash; preparatory and unavailable for execution. Generated skill guidance must fetch it live and disclose its own harness processing, not automate root terminal consent. |
| `lds reconcile journal record\|decide\|reconsider --pair <absolute-pair.json> --left-root <absolute-clean-pinned-Git-root> --right-root <absolute-clean-pinned-Git-root> --map <absolute-tracked-raw-map> [--map ...] --journal <absolute-external-journal-path> [--json]` | Linux root-only interactive hosts for explicit reviewed journal entries. Fresh scan and prior journal state, full preview and fresh terminal digest confirmation; no authored `--scan`/`--request` files. First confirmed record may create the journal. See the [guided how-to](../how-to/reconcile-projects.md#guide-a-journal-entry-in-the-terminal). |
| `lds plumbing reconcile journal read --pair <absolute-file> --journal <absolute-file>` | Read existing journal without writing. |
| `lds plumbing reconcile journal record --pair <absolute-file> --journal <absolute-file> --scan <absolute-saved-file> --request <absolute-file>` | Record reviewed correspondence and evidence. |
| `lds plumbing reconcile journal decide --pair <absolute-file> --journal <absolute-file> --scan <absolute-saved-file> --request <absolute-file>` | Record an explicit initial outcome. |
| `lds plumbing reconcile journal reconsider --pair <absolute-file> --journal <absolute-file> --scan <absolute-saved-file> --request <absolute-file>` | Explicitly supersede a decision; none edit sources. |

The guided hosts require terminal input and presentation even with `--json`:
prompts, candidate evidence, prior decisions and the **full** digest-bound
request/evidence/journal preview go to stderr; successful JSON goes to stdout.
After queued input is flushed, type exactly `record sha256:<digest>`,
`decide sha256:<digest>`, or `reconsider sha256:<digest>` for the displayed
operation, or `decline`. No `--confirm` shortcut. The host collects explicit
plain-text correspondence/provenance/assessment for record (optional complete
replacement during reconsider), actor/rationale, owner authority declaration,
permitted directions and outcome for decisions. Reconsider names its trigger
and explicitly supersedes the last decision. An `applied` outcome requires an
externally authored proof reference and explicit direction and authority;
the host does not verify proof or perform transfer. No source edits, model,
network, fetch, checkout or project commands beyond the existing read-only
Git scan. Fresh pair/source/map/journal checks refuse drift before writing;
there are no automatic retries. Preview/decline creates no journal, lock or
output files. For an indeterminate write, read the journal and inspect its
revision/request ID before any retry. The original saved-scan/request-file
plumbing remains available. These three root-only plumbing-kind leaves are
not generated/projection entries or a shared reconciliation flow.

The report brief requires `schemaVersion: 1`, nonempty `purpose` and
`reviewer`, an asserted RFC3339 `at` string, and `selections`, `claims`, and
`questions` arrays (empty is valid). Each selection is
`{"side":"left"|"right","path":"<scoped file or map>"}` and opts in
full UTF-8 content. Authored claims require `candidateId`, `status`
(`reviewed` or `proposed`), v1 `correspondence`, `rationale`, v1
`provenance` array, v1 `assessment`, `conflicts` array and `questions` array;
these are assertions, not verified matches. See the
[disposable example](../how-to/reconcile-projects.md).
Assisted drafts are strict `{"schemaVersion":1,"claims":[],"questions":[]}`
JSON (empty claims valid); nonempty claims use the existing `ReviewClaim`
shape but only `status: "proposed"` and a needs-review assessment, with
citations to explicitly selected current captured file/map evidence. Seed
claims/questions and journal history remain separate. The attributed,
**unauthenticated** receipt carries `provider`, `model`, `flowSource`,
`flowSha256`, `systemSha256`, `userSha256`, `systemBytes`, `userBytes`,
`consentAt`. All three external artifacts are required together by the
extended plumbing report; it does not refresh the observed baseline or call
a provider.
Report scope is at most 64 declared scoped files plus 16 maps. Every existing
scoped/map raw file is limited to 64 KiB, including unselected files; selected
UTF-8 content totals at most 512 KiB, and pair/review/optional journal total
at most 2 MiB. Paths for pair, review, optional existing journal and new
output must be absolute and outside both roots and their Git common
directories; the output path must have no symlink components. Roots must be
clean and match full pins. Original inputs are revalidated before publication.
Assistance allows only selected full scoped/map UTF-8 contents with sanitized
scan metadata, seed claims/questions and the live flow in a 512 KiB **total
prompt** budget; response is at most 64 KiB. It sends no host-added absolute
paths, credentials or raw journal contents, and does not redact selected text
or claims. One fixed OpenCode Go / `deepseek-v4-flash` request uses no tools,
redirects, retries or fallback. Flow/input drift is refused before send,
after response and before private publication, not repaired by recollection.

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
In offline mode, root `lds update` takes the same original collected packet
and explicit edits, not an assessment report or inferred proposal. With
`--json`, final stdout is
schema-1 JSON containing `operation: "update"`, `status`, the complete `plan`, and `apply`
(null if vetoed or declined); plan display, prompt and errors go to stderr.
Without `--json`, stdout is a human status summary.
Wrong input or EOF refuses. The root registration is now `llm` kind because
explicit `--assist` can invoke a model, although this default mode stays
offline. It is root-only and not projected; it is not a same-deliverable alias
for either canonical deterministic step.

Assisted mode requires 1–32 unique explicitly selected existing Markdown
targets under the same update scope; it never infers edits from inventory,
citations or assessment. It automatically collects bounded original evidence
and discloses the complete outbound system and user payload before fresh
exact `send` consent. One fixed `deepseek-v4-flash` OpenCode Go request has no
tools, redirects, retries or fallbacks. The full drafted replacements and
questions are unverified; no edits is valid and ends `review-required` without
apply. Nonempty drafts enter strict original-baseline update planning, full
exact-byte preview and separate freshly typed `apply sha256:<digest>` or
`decline`. The baseline and live flow are rechecked before/after transfer and
before apply; send alone never authorizes writes. Existing Linux replacement
and partial/indeterminate recovery rules apply. The new `plumbing update`
reader is preparatory only; projected guidance must fetch it live, disclose
harness processing and independently obtain consent rather than automate the
root terminal or derive approval from the model.

`lds review` also is root-only plumbing-kind, not projected or an alias, and
adds no plumbing leaf or shared LLM flow. It requires no terminal, credential,
authored JSON or `--output`; stdout contains human output or schema-1 JSON
(`operation: "review"`, absolute `targetRoot`, `status` of `findings` or
`no-findings`, `inventory`, `validation`, nonempty string-array `limitations`).
Human output lists scoped regular Markdown files with sizes and hashes,
local-link findings, skips and limitations. Validation includes inventory
skips. Only metadata and links, not full source text, are reported. Exit 1
means a supported local file target is missing, 0 means none was found; 2 is
usage, 3 unsafe target or unsupported platform, and 4 unexpected I/O. A
missing root exits 1 without a review result. Empty or absent optional
documentation tracks are not automatically findings, and empty coverage or
skipped checks are not passes.
Neither status establishes complete, correct or current documentation. Shell
redirection, if used, is the caller's write responsibility.

`lds manifest` is also an alias. Plumbing scan and journal successes emit JSON
even without `--json`; report prints a human summary by default and JSON with
`--json`. Exit 0 is success; 1 is validation/evidence finding,
2 usage, 3 refusal/staleness, 4 unexpected internal/I/O failure. Some docs
diagnostics emit a JSON result to stdout **and** an error to stderr: inspect
exit code and both streams. Cobra usage errors are human diagnostics. Root
`init` remains a candidate; broader review is deferred. Root `reconcile` has
offline/assisted report and guided journal slices, not an authority-granting
facilitator. The manifest has 35 application leaves, 22 qualified projected plumbing-kind
entries and nine docs leaves;
projection eligibility is not evidence of native execution or parity. Root
report is LLM-kind (although default remains offline), not a plumbing alias
or projected member; root journal remains deterministic root-only. See the
[assessment how-to](../how-to/assess-documentation.md) for the separate
provider-free and consent-gated routes.
