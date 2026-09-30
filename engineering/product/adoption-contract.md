# LDS adoption contract

**Status:** Accepted

This document records the accepted behavior for adopting the LDS two-track
layout in a target project. It is a behavior contract, not a CLI implementation
or a template specification. The layout follows [ADR-0001](../decisions/0001-adopt-two-track-documentation.md):
`engineering/` is the engineering documentation track; `docs/` is the
user-facing Diátaxis track. This contract does not change the repository's
root `README.md`. No real target project has been selected or authorized for
modification.

## Product facts required before creation

The target project must have a complete, owner-approved fact bundle before an
adoption attempt creates any files. If any fact is missing or unverified, the
attempt refuses all writes. Facts must describe the adoption target; they are
never facts about LDS by default. For a contract fixture, the disposable
fixture project is the target, so owner-approved fixture facts and an operation
verified against fixture source and/or a fixture run are valid test inputs.
They do not establish facts or usability for a real pilot. Do not fabricate
claims, placeholders, aspirations, or inferences.

| Required fact | Provenance and validation |
| --- | --- |
| Display name | Approved by the target project's owner; record who approved it and its source. No default name is supplied by this contract. |
| One-sentence purpose | Approved by the target project's owner; record the approved wording and source. For a fixture, use owner-approved fixture wording, not an LDS default. |
| Current capability | Describe what the target project can do now, with citations to authoritative project documentation and/or implementation. Separate shipped behavior from plans. |
| Verified user operation | Identify an operation supported by the target as it exists, its prerequisites, exact invocation and input, observed or source-backed output, and limitations. Verify against target documentation/implementation and an actual run where available. For a fixture target, fixture source/run evidence is valid input; it is not real-pilot validation. Never present a proposed operation as current. |

## Accepted output paths

With a complete fact bundle and a successful preflight, adoption creates or
preserves these four output files, and no other output files:

| Path | Content contract |
| --- | --- |
| `engineering/README.md` | Engineering navigation linking only to `engineering/product/overview.md`. |
| `engineering/product/overview.md` | Display name, one-sentence purpose, current capability, and source references for those facts. |
| `docs/README.md` | User documentation hub linking only to `docs/reference/usage.md`. |
| `docs/reference/usage.md` | The verified user operation, exact invocation and input, observed/source-backed output, prerequisites, and limitations. |

Creation fills only missing approved paths in an unambiguous partial tree.
It may create only necessary parent directories for the approved files
(`engineering/`, `engineering/product/`, `docs/`, and `docs/reference/`); it
creates no unused directories.
The root README may be reported as a signpost opportunity, but is report-only
and is never an output. The contract does not authorize modification of any
real repository. No empty LDS layers, placeholders, speculative goals, or
other output files or unused directories are authorized.

## Preflight and write behavior

Before creating any file, the adoption workflow presents a reviewable
preflight that classifies each candidate path as `create`,
`identical-untouched`, `different-collision`, or `refusal`, and identifies
affected existing paths that will be preserved.
A human must confirm before any creates. Any collision or refusal vetoes all
creates in that attempt. This contract never authorizes editing, replacing,
deleting, or moving existing content. An identical rerun is a no-op; changed
facts targeting an occupied output are a collision, not an update. This is an
application-level rule, not a transactional filesystem guarantee.

| Observed condition | Preflight result | Effect of the attempt |
| --- | --- | --- |
| Approved output path is absent; tree is unambiguous and inputs are complete | `create` | Eligible for creation after human confirmation, if every candidate is eligible. |
| Occupied path is byte-identical to the output for the current facts | `identical-untouched` | Preserve it unchanged; no write to that path. |
| Occupied path has different content | `different-collision` | Record occupied target, distinct bytes/content, provenance, and affected links for manual review outside this contract. No writes occur in this attempt. |
| Any required fact is missing or unverified | `refusal` | Refuse all writes. |
| `docs/` is recognizable as legacy LDS or its status as legacy LDS is ambiguous | `refusal` | Refuse all writes; no automatic migration. `docs/reference/` alone is not conclusive evidence of legacy LDS. |
| An output path or ancestor is an unsafe symlink, or an occupied path is unreadable | `refusal` | Refuse all writes. |

All existing content—including paths classified as identical or conflicting—
remains untouched. A refusal or collision anywhere in the candidate set means
no new paths are created in that attempt. A later attempt may be considered
after the condition changes and a new preflight is reviewed.

On a first run against an eligible tree with complete facts and all four
approved output files absent, preflight reports four file creates;
confirmation creates only those four files and any necessary parent
directories. On an identical rerun, the four files classify as
`identical-untouched` and the expected result is zero writes. In a partial tree,
only missing approved paths are candidates for creation. Any collision or
refusal vetoes all creates for that attempt.

## Scope and validation limits

The accepted choice **1B** defers naming a real pilot. Fixtures can exercise
the contract's input validation, preflight classifications, and
preservation/refusal behavior, but fixture facts and outputs are disposable
and non-normative. They do not establish successful adoption or validate facts
for a real target. Do not present fixture results as real-pilot validation.

This document specifies accepted product behavior; it does not claim that a
runtime CLI implements it. **BDS-250 owns the runtime boundary**:
implementation-specific runtime mechanics are outside this contract and must
not be inferred from it.

## Relationship to the roadmap

This contract settles the adoption behavior above; it does not promote the
roadmap's other candidate verbs to implemented or specified capabilities.
The roadmap describes `init`, `assess`, `update`, `review`, `reconcile`, and
inventory/validate/render as proposals, and states that the current scaffold
does not yet manage project documentation ([`roadmap.md`](roadmap.md#L8-L14),
[`roadmap.md`](roadmap.md#L61-L76)).
