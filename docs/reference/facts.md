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

All strings must be nonempty. Source fields must contain a citation. Local
citations must be safe relative paths, not absolute paths, `..` traversal,
backslash paths or `file://` URLs. HTTP(S) source URLs are also accepted but
are not fetched or verified. Unknown/duplicate fields are rejected.
Structural validation cannot authenticate an approver or prove the operation
works. Verify the claims and rendered Markdown yourself; approval in this
file does **not** confirm a later plan digest. See [adoption steps](../how-to/adopt-a-project.md).
