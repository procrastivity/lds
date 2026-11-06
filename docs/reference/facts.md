# Adoption fact bundle

`--facts` is a UTF-8 JSON file with exactly the keys below. The
[complete fixture JSON](../../internal/asset/testdata/adoption/greenfield/facts.json)
is a valid **test-only** example of the shape, not a fact bundle or approval
for your project. Write your own assertions and citations; do not copy its
simulated approver, operation, or output evidence into a real adoption.

| JSON keys | What to supply |
| --- | --- |
| `DisplayName`, `Purpose`, `CurrentCapability` | Approved project identity, purpose, and **current**, not proposed, capability. |
| `DisplayNameSource`, `PurposeSource`, `CurrentCapabilitySource` | Source citations and approval/provenance for those assertions. |
| `Prerequisites`, `OperationName`, `OperationSummary` | A verified existing user operation and what is needed to run it. |
| `Invocation`, `Input`, `OutputEvidence`, `OperationSource`, `Limitations` | Exact invocation and input, observed or source-backed output, source citation, and limitations. |
| `approval` | Object with `approver`, `approvedFacts`, and `status`. The four `approvedFacts` values are `DisplayName`, `Purpose`, `CurrentCapability`, `VerifiedUserOperation`; for a real approved bundle `status` is `approved`. |

For an approved bundle, all strings must be nonempty. Source fields must
contain a citation. Local citations must be safe relative paths, not absolute paths, `..` traversal,
backslash paths or `file://` URLs. HTTP(S) source URLs are also accepted but
are not fetched or verified. Unknown, duplicate, case-aliased, missing and
null fields are rejected. Use empty strings for unknown assertions in drafts.
Structural validation cannot authenticate an approver or prove the operation
works. Verify the claims and rendered Markdown yourself; approval in this
file does **not** confirm a later plan digest. See [adoption steps](../how-to/adopt-a-project.md).

## Draft facts with the offline helper

`lds facts [--json]` asks for the fourteen assertion and citation fields in a
Linux terminal. It takes no project argument and reads no project files,
templates or flows. It never executes the entered invocation, fetches a
citation, calls a provider, invents evidence or records approval. The global
asset selector has no effect on any facts mode. Use `--resume <file>` to edit
a saved bundle or `--review <file>` for the separate approval step below;
these flags are mutually exclusive.

Run directly in a terminal, with input and stderr attached to it. Save stdout
privately **outside** the intended project; for example:

```sh
umask 077
review=$(mktemp -d)
(set -C; lds facts --json > "$review/facts-draft.json")
```

Prompts and the list of unanswered fields go to stderr. Each answer is one
line of at most **2048 UTF-8 bytes**, not characters; control characters and
invalid UTF-8 refuse. Leave unknowns blank rather than fabricating evidence.
Blank or whitespace-only answers become empty strings. Other answers stay
unchanged. Citation syntax and truth are not validated by the questionnaire;
the existing adoption consumer still validates complete approved bundles.
For longer or multiline assertions, edit the private draft separately.

Successful stdout is the **raw fact bundle**, with or without `--json`, not a
workflow envelope. Its approval is always
`{"approver":"","approvedFacts":[],"status":"pending"}`, including when
every question has an answer. Exit 0 means only that a draft was emitted;
existing render/plan/init/adopt operations still refuse this unapproved draft.
Complete unknown fields and independently verify every claim and citation.
An authorized reviewer can use `--review` below to record facts approval
without hand-editing the JSON. The helper reduces JSON authoring; it does not
replace evidence, facts approval or fresh plan approval.

## Resume a saved draft

Run in the same Linux terminal, saving to a **different** private file:

```sh
(set -C; lds facts --resume "$review/facts-draft.json" --json > "$review/facts-revised.json")
```

The complete pending draft is displayed on stderr. Type an exact assertion
or citation key, such as `Purpose` or `OutputEvidence`, then its replacement
answer. Blank clears that field to unknown; fields not selected retain their
values, including existing multiline assertions. New answers have the same
2048-byte single-line limit as the questionnaire. Type `done` to emit the
pending bundle or `:cancel` to refuse. Approval is not an editable field.
**Any loaded approval is discarded**, even when no assertions change.
Resume permits unknown or syntactically invalid citations so you can repair
them; it never infers missing evidence or grants approval.

## Review and approve complete facts

Once every assertion and citation has been independently verified, an
authorized reviewer can run:

```sh
(set -C; lds facts --review "$review/facts-revised.json" --json > "$review/facts-approved.json")
```

LDS first checks completeness and citation syntax using adoption's validator.
Unknowns or unsafe citations fail before requesting approval. Enter the
reviewer's name; LDS displays the **entire candidate bundle**, including the
new name, all four approved groups and `status: "approved"`. Review it in full,
then freshly type exactly `approve facts sha256:<displayed digest>`, or
`decline` to refuse. Queued input is discarded before the decision prompt;
neither `--yes`, a bare digest nor nonterminal input bypasses this gate.
Successful stdout contains the exact displayed JSON bytes, including the
final newline, whose SHA-256 is the displayed digest. A later resume always
resets approval and requires a new review.

This approval is an **unauthenticated reviewer assertion**, not proof of
truth, verified command execution, real-project authorization or approval of
an init/adopt plan. LDS does not inspect the cited files, fetch URLs or run
the invocation. Using the approved bundle with init/adopt still requires the
separate complete-plan review and fresh exact-plan confirmation.

## Saved-file and output limits

Saved bundles must be regular UTF-8 JSON files of at most **256 KiB**; final
symlinks and special files refuse. Saved assertion values may contain line
feeds but no other control characters. The encoded output also has a 256 KiB
limit. Both saved modes recheck original file identity and raw bytes before
emitting; review additionally rechecks before displaying its candidate.
Observed replacement or changes, even whitespace-only changes, refuse with
exit 3 and empty stdout. Inputs and ancestors require trusted, quiescent
writers: these checks are not atomic snapshots or hostile-writer isolation.
Neither mode overwrites its input or invokes init/adopt.

**Never redirect stdout over the input file**, including through another
path to the same file: the shell can truncate it before LDS starts. Use a
different private output path and noclobber as shown above. Inspect failures
before retaining an output file; repeat a refused operation with a new output
path or deliberately remove its empty capture first.

Type `:cancel` or end input to refuse with exit 3 and empty stdout. Nonterminal
input/presentation or a non-Linux platform also refuses with 3. Invalid answers
or saved JSON exit 1, usage errors 2, unexpected I/O errors 4. Nothing is emitted
until the questionnaire, resume or fresh approval finishes; an output-write
failure can leave partial JSON. Shell
redirection itself may create an empty file even on refusal; inspect its exit
status and contents before retaining it. LDS does not manage that file or
guarantee its location, permissions, exclusive creation or atomic publication.
The example's private directory, `umask` and noclobber are caller precautions,
not a new LDS write boundary. Do not put secrets in answers or share the draft
without reviewing its contents.
