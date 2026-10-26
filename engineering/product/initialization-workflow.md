# Bounded contract: offline project initialization

**Status: implemented and verified offline with disposable fixtures.**
The owner authorized this bounded offline `lds init` host. This is not a
published release, real-target approval, pilot, native skill execution or
CLI/skill parity result. It does not reopen historical adoption acceptance
or change the [four-output adoption contract](adoption-contract.md) and
[reviewed plan/application boundary](adoption-operations-contract.md#reviewed-plan-and-application).

## Interface and composition

```text
lds init <root> --facts <file> [--json]
lds --project-root PATH init <root> --facts <file> [--json]
```

Require Linux, terminal input and terminal presentation, an explicit existing
target directory and the existing complete approved fact schema. No implicit
cwd/Git/environment root, project-root creation, provider/key requirement,
network, flow reader or LLM is involved. Init is root-only **plumbing-kind**
porcelain: a composition, not an alias of `docs plan` or `docs apply`, not a
qualified projected entry. Add no `plumbing init` leaf, shared model flow or
installed-skill sequence. The resulting manifest has **36 application leaves,
22 qualified projected plumbing-kind entries and nine docs leaves**; older
handoff counts remain historical, not updated acceptance evidence.

Reuse `adoption.Plan` and `adoption.ApplyReviewed`, with the same explicitly
selected global `--project-root PATH` passed through planning and application.
Only the four existing adoption templates are consumed; no `flows/adopt.md`
read or advisory request is needed. Without the flag, retain user-XDG →
installed → embedded precedence; with it, prepend the canonical selected
root's `.lds/assets/<shipped-relative-name>` per existing asset rules.
The positional target never implies the selector. Missing project assets fall
through per file; unsafe/unreadable selections refuse. Selected sources and
hashes remain reviewable and freshness-bound. Project selection does not
alter manifest identity, projection digest/stamps, configuration or installed
guidance. Registering the new init leaf does change the manifest normally;
existing harness installations remain subject to normal doctor/install drift
checks even though the set of projected commands is unchanged.

Facts use the [existing schema](../../docs/reference/facts.md), not newly
inferred, model-authored or placeholder claims. All required identity,
purpose, current-capability and verified-operation groups need their existing
citations and approval fields. Validation cannot authenticate an approver or
prove claims; facts approval is **not** approval of a plan digest. Test-only
simulated fixture approval is valid only for disposable fixture exercises,
never a real-project authorization.

## Display, refusal and fresh human decision

1. Preflight platform/terminal/arguments without target writes. Build the
   canonical adoption plan in memory; no temporary plan/report artifact.
2. Display the **complete** `adoption.Plan` result as JSON on stderr, including
   every field present in the canonical serialization, not a hand-picked
   projection or summary. Keep `operation: "docs plan"`, `schemaVersion`,
   target/root identity, facts digest, all outputs and their content/hashes,
   asset provenance, occupied content/encoding/provenance and affected links,
   necessary/existing parents, preserved paths, report-only notice, findings,
   skips, veto and nullable `planDigest`. Omitted optional fields follow the
   existing canonical schema; init adds no field filtering.
3. On a veto or incomplete facts, return refused without a prompt or writes.
   Incomplete facts have no eligible digest. Supported missing-local-link
   findings alone are diagnostics, not a veto or semantic approval.
4. For an eligible plan, flush queued terminal input **after** display. Accept
   only freshly entered `apply sha256:<exact displayed digest>` or `decline`
   alone. Other text (including a bare digest), mismatch or EOF refuses.
   No `--yes`/`--confirm` bypass or apply via a stdin pipe; JSON mode retains
   terminal requirements. Do not automate digest extraction/confirmation.
5. On affirmation, call `adoption.ApplyReviewed` with the **exact in-memory
   displayed plan bytes**, confirmed digest, same root/facts and same selector.
   Do not persist a plan or silently replan into a replacement approval.
   The existing strict digest, stale-input and exclusive-create checks govern
   application; internal replanning is a comparison against the reviewed
   snapshot, never authority for changed inputs.

## Write boundary and recovery

Eligible creates are only `engineering/README.md`,
`engineering/product/overview.md`, `docs/README.md` and
`docs/reference/usage.md`, plus only their necessary missing parents
(`engineering/`, `engineering/product/`, `docs/`, `docs/reference/`). No extra
empty layers, speculative content or other output paths. The root README is
report-only and never edited. No migration, update, move, deletion, Git/project
commands or harness installation belongs to init.

Byte-identical occupied outputs remain untouched. Any different occupied
output, unsafe path or recognizable/ambiguous legacy LDS evidence vetoes
**every** create; do not skip conflicts and proceed elsewhere. An unambiguous
partial tree may fill only missing approved paths after full review. Changed
facts targeting occupied outputs remain collisions, not updates.

Apply rechecks facts, selected template sources/bytes, original evidence and
diagnostics, root and existing parent identities through the existing
stale-input boundary. Changed inputs refuse, not silently refresh or retry.
Linux descriptor-relative no-follow/exclusive per-file writes do not provide
an atomic snapshot, hostile-writer isolation, a four-file transaction or
rollback. Late failure may leave created parents, completed files or partial
failed output. Inspect actual state and the structured effects before a new
plan, review and fresh decision; never automatically retry or undo writes.
Require trusted, quiescent ancestors and writers.

An identical rerun still displays a new complete plan, flushes queued input
and requires a new review/confirmation. Only confirmed application returns
`current` with zero file creates; do not short-circuit the human gate when
all four outputs are identical.

## Results and streams

With `--json`, final stdout is exactly the schema-1 init envelope:

| Field | Contract |
| --- | --- |
| `schemaVersion` | `1`. |
| `operation` | `"init"`. |
| `status` | `"applied"`, `"current"`, `"refused"` or `"partial"`. |
| `plan` | Complete canonical plan object, retaining `operation: "docs plan"`. |
| `apply` | Canonical actual-effects object when apply was attempted, otherwise null (including veto/decline); nested operation stays `"docs apply"`. |

Without JSON, stdout is the human status summary. Full plan presentation,
prompts and errors stay on stderr in either mode. An early failure before a
plan exists, including noninteractive refusal, emits only the existing stderr
error envelope and leaves stdout empty; do not invent an empty init envelope.
Inspect exit code and both streams. Usage remains exit 2, policy/refusal or
staleness exit 3 and unexpected internal/I/O exit 4; successful applied/current
results exit 0. Partial effects are not success, regardless of error class.

Nested apply retains `planDigest`, `status`, `created`, `identicalUntouched`,
`unattempted`, `failed` and `preserved` under the existing schema. A result is
observed effects, not authenticated approval or semantic proof. If output
itself fails, inspect the tree rather than assuming no writes happened.

## Verification and limits

The [disposable how-to](../../docs/how-to/initialize-a-project.md) supplies
the fixture setup used for real-binary checks: its source command produced
`Word count: 4`; initialization created exactly the four expected fixture
outputs; a freshly confirmed rerun returned `current` without changes; and
decline left an empty target unchanged. Manifest checks observed 36 leaves
and 22 eligible projected entries. Linux amd64 and Darwin arm64 CGO-disabled
builds passed; execution remains Linux-only.

Terminal regression tests cover full plan display and result equality,
fresh queued-input flush, decline/bare or wrong digest/EOF refusal, incomplete
facts, all-create collision and unsafe-parent vetoes, selected project
templates, stale facts/templates/target evidence, confirmed identical outputs
and filling an unambiguous partial tree. Real-binary noninteractive and
approval-flag refusals leave stdout empty and write nothing. The unchanged
adoption tests also cover legacy refusals, late parent/root replacement and
partial effects without rollback. `make check` and
`go test -race -count=1 ./...` passed with the provider key unset.

These checks do not authenticate facts or authorize a real target. No
provider, owner-driven real-target test, pilot or release is required or
authorized by this slice; historical acceptance records remain unchanged.
