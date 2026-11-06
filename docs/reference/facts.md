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
are not fetched or verified. Unknown/duplicate fields are rejected.
Structural validation cannot authenticate an approver or prove the operation
works. Verify the claims and rendered Markdown yourself; approval in this
file does **not** confirm a later plan digest. See [adoption steps](../how-to/adopt-a-project.md).

## Draft facts with the offline helper

`lds facts [--json]` asks for the fourteen assertion and citation fields in a
Linux terminal. It takes no project argument and reads no project files,
templates or flows. It never executes the entered invocation, fetches a
citation, calls a provider, invents evidence or records approval. The global
asset selector has no effect on this questionnaire.

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
Complete unknown fields, verify every claim and citation, and have an
authorized reviewer independently record their name, the four approved fact
groups above and `status: "approved"` before using it. The helper reduces JSON
authoring; it does not replace evidence, facts approval or fresh plan approval.

Type `:cancel` or end input to refuse with exit 3 and empty stdout. Nonterminal
input/presentation or a non-Linux platform also refuses with 3. Invalid answers
exit 1, usage errors 2, unexpected I/O errors 4. Nothing is emitted until all
questions finish; an output-write failure can leave partial JSON. Shell
redirection itself may create an empty file even on refusal; inspect its exit
status and contents before retaining it. LDS does not manage that file or
guarantee its location, permissions, exclusive creation or atomic publication.
The example's private directory, `umask` and noclobber are caller precautions,
not a new LDS write boundary. Do not put secrets in answers or share the draft
without reviewing its contents.
