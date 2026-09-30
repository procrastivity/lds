# Adoption preservation and collision fixtures

Every case shares the Step 7 greenfield Wordcount target and its complete,
simulated-owner-approved facts as a common baseline. `overlayFiles` lists only
that case's extra pre-existing target content; overlay paths and bytes remain
independent and asymmetric. The mixed root README describes the baseline's
verified Wordcount operation. `internal/asset/testdata/adoption/preservation/cases.json`
records the baseline, each overlay's exact target path and fixture source,
expected SHA-256, all four approved-output classifications, proposed creates,
veto behavior, and overlay paths whose bytes must remain unchanged. Baseline
code and input are present and byte-preserved for every case; the facts JSON
is external test evidence, not a target file. The
identical-generated case reuses the independently authored Step 7 expected
files as its concrete pre-existing overlay bytes; the fixture test compares
those bytes with a fresh render of the current templates. Step 7 continues to
check its independent golden outputs. The edited-generated-looking case has
distinct bytes.

The cases specify expected preflight behavior only. They do not run an
adoption workflow, apply the proposed creates, or provide a transactional
filesystem guarantee. `create` means eligible for a reviewed first-run plan;
human confirmation remains required. Collision and refusal cases predict no
creates in that attempt. No fixture is a real project or pilot.

| Case | Input and expected preflight |
| --- | --- |
| `mixed-root-readme` | A mixed user/contributor root `README.md` describing the Wordcount baseline operation is preserved byte-for-byte; it yields only a report-only signpost, while all four approved files are eligible creates. |
| `recognizable-legacy-under-docs` | Existing guide, decisions, and product content identify legacy LDS under `docs/`; refuse all four candidates, preserve all inputs, and do not migrate. |
| `ambiguous-legacy-under-docs` | Decisions, product, and reference content suggest a legacy layout, but ownership is unclear; refuse all candidates. The reference directory alone is not treated as conclusive. |
| `partial-legacy-under-docs` | A partial LDS ADR explicitly roots the tree at `docs/`; refuse all candidates despite missing layers. Hypomnema demonstrates a real partial installation: its ADR adopts six of seven layers and says missing layers are not errors ([ADR-0001, lines 21–41](https://github.com/gethmn/hypomnema/blob/main/docs/decisions/0001-adopt-layered-documentation-system.md#L21-L41)); its guide marks Behaviors absent while listing the other layers ([guide, lines 57–67](https://github.com/gethmn/hypomnema/blob/main/docs/DOCUMENTATION-GUIDE.md#L57-L67)). |
| `partial-engineering-missing-layers` | A lone existing `engineering/decisions/` file, with `docs/` absent, does not make missing layers an error; all four outputs remain eligible creates. Only necessary missing parents are listed. |
| `one-different-approved-output` | A distinct occupied `docs/reference/usage.md` collides; other absent files remain classified `create` but all creates are vetoed. Its existing link is reported, and bytes are preserved. |
| `already-generated-identical` | Four byte-identical occupied outputs classify `identical-untouched`; preflight predicts a no-op and zero writes. |
| `edited-generated-looking-output` | A generated-looking but edited occupied output is a different-byte collision, not owned by a harness stamp; the three absent creates are vetoed, and the edited bytes/link are preserved. |

The contract's `docs/reference/`-alone caveat remains: it is not conclusive
legacy LDS evidence. These fixtures do not define a classifier for arbitrary
trees or settle ambiguous cases beyond the supplied examples.
