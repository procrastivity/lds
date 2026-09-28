# ADR-0001: Adopt LDS engineering documentation and Diátaxis user documentation

**Status:** Accepted

## Context

LDS is being built as a Toolsmith application. Its design, templates, and
future documentation workflows need a home in the same repository as the
code. Contributors and agents need decisions and specifications; people
using the finished CLI will need task-oriented guidance. Those are different
audiences, and a single undifferentiated `docs/` tree would blur them.

This decision draws on [Hypomnema's adoption ADR](https://github.com/gethmn/hypomnema/blob/main/docs/decisions/0001-adopt-layered-documentation-system.md),
[Xcind's LDS ADR](https://github.com/scinddev/xcind/blob/main/engineering/decisions/0011-layered-documentation-system.md),
and [Xcind's two-track ADR](https://github.com/scinddev/xcind/blob/main/engineering/decisions/0014-two-track-documentation.md).

## Decision

- `engineering/` is the tool's design of record. Use the seven LDS layers
  (decisions, product, architecture, specs, reference, behaviors,
  implementation) as needed. Empty layers do not require placeholder files.
- `docs/` is reserved for user-facing Diátaxis tutorials, how-to guides,
  reference, and explanation. Create it when there is real LDS behavior to
  document. The repository README points readers to both audiences.
- Proposed capabilities and repository roles live in engineering documents;
  Linear tracks tasks and progress, not the canonical design text.
- The intended LDS-generated layout also has two tracks. Exact templates,
  adoption/migration rules, and command contracts remain to be specified;
  this ADR does not claim the scaffold implements them.

## Consequences

Engineering plans can evolve without being presented as user instructions.
Contributors must choose the audience before adding a document and link
across tracks rather than silently duplicating authoritative facts. The
initial tree is intentionally partial, so missing layers and `docs/` are
not validation failures.
