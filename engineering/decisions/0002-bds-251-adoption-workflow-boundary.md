# ADR-0002: Bound the BDS-251 adoption workflow

**Status:** Accepted for BDS-251 planning

## Context

[ADR-0001](0001-adopt-two-track-documentation.md) establishes the engineering
and user-facing documentation tracks, but does not implement an adoption
workflow. The [adoption behavior contract](../product/adoption-contract.md)
limits creation to four approved files, requires complete owner-approved
facts, and preserves or refuses rather than migrating existing content. The
[operations contract](../product/adoption-operations-contract.md) defines
the deterministic plumbing, reviewed plan, digest confirmation, and stale
input refusal. Toolsmith's [C8 shape contract](https://github.com/procrastivity/toolsmith/blob/main/CONTRACT.md#c8--shapes)
defines the future CLI/skill boundary and the `plumbing` namespace. These
decisions do not name a real pilot or establish real-project usability.

## Decision

1. **Keep the pilot gate (1B).** No real target is selected or authorized
   for modification. BDS-249 fixtures and BDS-250 disposable runtime results
   remain test evidence only.
2. **Prepare create-only `lds adopt` now (2A), outside first-release scope.**
   BDS-251 is to specify the future adoption entry point and its bounded
   create-only behavior. This does not authorize migration, replacement,
   deletion, or writes to a real repository, and it does not make the command
   part of the first release.
3. **Adopt the C8 surface model (3A), without a blanket move.** Audit every
   existing root command for its intended human versus agent audience and
   its kind (`plumbing`, `llm`, or `control-plane`). Do not move all root
   commands under a namespace by default. Agent-facing projection requires
   both the `plumbing` namespace and the deterministic `plumbing` kind;
   neither audience nor projectability is inferred from a command's name or
   package location.
   The reviewed per-command dispositions are recorded in the [BDS-251 CLI
   surface audit](../product/cli-surface-audit.md). This is a future surface
   decision, not a command-tree change in this ADR.
4. **Defer paired LLM delivery (4B).** Do not ship an LLM-backed CLI workflow
   paired with a skill until LDS has one settled provider, endpoint, model,
   and credential source, and authorized disposable live access for exercising
   that integration. Deterministic plumbing and workflow preparation do not
   imply that this gate has been met.
5. **Limit the initial harness audience (5A) to Claude Code.** Do not claim
   another harness or provider is supported by the initial adoption shape.
6. **Require a fresh human decision for writes (6A).** Any future interactive
   form must present the entire reviewed four-output plan and obtain a fresh
   human decision tied to its exact digest. A changed plan requires a new
   review and decision. A noninteractive invocation stops without writing;
   it must not infer approval, extract a digest as consent, or continue from
   an earlier decision.

## Consequences and limits

- The intended adoption remains create-only: complete verified target facts
  are required, and any collision or refusal vetoes all creates. Existing
  files are never edited, replaced, moved, or deleted; no automatic migration
  is authorized. Fixture-owner approval is not real-project authorization.
- A skill-only implementation is not completion of the paired workflow.
  Conversely, a deterministic plumbing command is not the deferred LLM
  workflow or its skill.
- The future namespace projection must preserve the distinction between
  human-facing commands and deterministic kind. The per-command audit is
  recorded separately; implementation mechanics remain out of scope here.
- Reconsidering the first-release exclusion, selecting provider configuration
  and live integration access, and selecting/authorizing any real pilot require
  separate decisions. This record does not assert that `lds adopt`, a paired
  workflow, or a corresponding skill exists.
