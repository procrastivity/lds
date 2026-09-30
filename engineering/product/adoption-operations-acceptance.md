# BDS-250 deterministic adoption acceptance matrix

**Status:** pre-implementation acceptance specification; no BDS-250 runtime
behavior is established by this document.

This matrix independently translates the accepted
[operations contract](adoption-operations-contract.md) (SHA-256
`adf0e770a7d5c3df9c56fce939cd50892cd6e5a0323043e64a836620c815f85d`) into
machine-checkable result projections in
[`expected-results.json`](../../internal/cli/testdata/adoption-operations/expected-results.json).
The [BDS-249 handoff](adoption-stage-3-handoff.md) and
[greenfield](adoption-fixtures.md)/[preservation](adoption-preservation-fixtures.md)
fixtures provide inputs and product expectations, not executed adoption
results. All expected values are specifications until the later runtime
suite records actual commands, stdout, stderr, exit codes, and before/after
target bytes.

## Reading the expected-results file

The JSON distinguishes `fixture-projection` cases (deterministic expectations
derived from Step 7/8 facts, overlays, templates, and accepted policy) from
`runtime-observation` cases (outcomes that must be exercised on disposable
targets). `expectedStdout` is a projection, not always a literal stdout
capture: ordinary values are exact, and the reserved string `empty` means a
zero-byte stream; an object with an `assert` member is a predicate over a field
in the eventual CLI JSON. For example, plan digests
must be recomputed from the actual displayed plan, and target-root identity
must equal the explicitly supplied target, never a checked-in absolute path.
The CLI must still emit the complete operation schema.
For the Wordcount target, result paths are relative to `greenfield/target`:
the preserved files are `sample.txt` and `word-count.sh`; fixture source
references may retain the `target/` prefix.

## Acceptance matrix

| Case group | Expected result and exit | Evidence status |
| --- | --- | --- |
| Greenfield `inventory` / `validate` | Empty scoped documentation arrays; exit 0. Source code and sample input are not documentation inventory. | Fixture specification; runtime observation remains required. |
| Greenfield `render` / `plan` | Four outputs byte-match Step 7 independent expected files; four `create` classifications, necessary parents only, no veto; plan exit 0. Digest is a relation, not a fixture constant. | Fixture specification; runtime observation remains required. |
| Greenfield `apply` / identical rerun | After explicit approval, first apply reports four actual creates and exit 0; rerun reports `current`, four `identicalUntouched`, zero creates, exit 0. | Runtime observation required; Step 7 counts are plans only. |
| Eight Step 8 preservation cases | Mixed README and partial engineering: four creates, no veto, exit 0. Recognizable/partial/ambiguous legacy under `docs/`: all four refusals, veto, exit 3. Differing/edited occupied output: one collision, three other candidates `create`, veto all, exit 3. Four identical outputs: all `identical-untouched`, no veto, exit 0. | Fixture specifications; no adoption command has run. |
| Broken local link; external URL and fragment-only link | Missing relative target is `validation.local-target-missing` in `findings`, exit 1. External and fragment-only links appear in `skipped` with reasons; they are not successful checks. A plan with link diagnostics alone remains eligible, not vetoed, exit 0. | Input/result projection; runtime observation required. |
| Missing or unverified facts | `render` rejects invalid/missing render input, exit 1. `plan` refuses all candidates, has `planDigest: null`, vetoes writes, exit 3. Neither track proceeds alone. | Deterministic policy projection; diagnostic wording/code except stable class awaits implementation. |
| Markdown metacharacters | Valid plain-text values render inertly in Markdown context: no injected heading, link, HTML, template action, or extra block. Invocation delimiters exceed any delimiter run in the value. Unsafe controls/local source references and malformed template actions fail render, exit 1. | Semantic acceptance only; exact escaping bytes require runtime observation. |
| Stale facts, effective asset, target, or ancestor | Change after review refuses apply with no creates, exit 3; unsafe symlink ancestors are not followed. Recomputed digest changes when a displayed input/evidence changes. | Runtime observation required. |
| Declined confirmation | Apply reports `refused`, no creates, eligible paths unattempted, preserved bytes unchanged; exit 3. | Runtime observation required. |
| Late injected I/O failure | After at least one exclusive create succeeds, report `partial`, actual created/failed/unattempted paths and readable hashes, preserve existing files, do not roll back; injected I/O exit 4. | Runtime observation required; exact affected path set is determined by the observed failure point. |

The greenfield and eight Step 8 rows are **fixture specifications**, not
runtime observations. Every executable operation—including those rows—still
requires a later runtime run on disposable copies. Link, invalid-fact,
metacharacter, stale-plan, decline, and late-failure rows likewise state
acceptance expectations; only the Step 7 shell operation has been executed so
far, not an adoption operation.

## Result and digest constraints

The JSON projections pin schema version, operation names, stable
classifications, slash-separated relative paths, empty arrays, the stable
diagnostic code explicitly named by the contract, output/asset hashes, and
fixture-byte references. They deliberately do **not** assign human messages,
invent unspecified diagnostic tokens, freeze an absolute target root, or store
a plan digest. For each eligible plan, check:

`planDigest == "sha256:" + hex(SHA256(RFC8785(plan object with planDigest omitted)))`.

Recompute over the entire displayed plan. Changes to facts/provenance,
effective asset bytes/source, target state, parents, preserved paths, findings,
skips, veto, or root README notice must change the reviewed digest. Incomplete
facts produce `planDigest: null`; apply must refuse stale inputs rather than
silently replanning.

The JSON includes a fixed RFC 8785 canonicalization vector. Its SHA-256 is a
canonicalizer test value, **not** a plan digest. Rendered output hashes point
to Step 7's independent expected files; asset hashes point to selected asset
bytes. Result arrays must be sorted by slash-separated path even where fixture
input files were authored in another order.
The independent expected-output files and projection hashes now include the
Step 6 escaped-Markdown text and variable-length invocation fence; those
expected files remain separate from renderer output.

Existing CLI sources establish the conventions to reuse:
[execution/error routing](../../internal/cli/execute.go),
[exit codes](../../internal/exitcode/exitcode.go), and
[JSON error envelope](../../internal/ldserr/ldserr.go). This is an acceptance
oracle only: it adds no command, real-target fixture, or runtime claim.
