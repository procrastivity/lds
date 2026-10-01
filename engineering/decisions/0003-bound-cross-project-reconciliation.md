# ADR-0003: Bound cross-project reconciliation authority and journal

**Status:** Accepted for BDS-252 planning

## Context

The [roadmap](../product/roadmap.md) separates reconciliation between projects
from maintaining documentation within one project. [ADR-0001](0001-adopt-two-track-documentation.md)
places LDS's design record under `engineering/`; the
[source-repository map](../architecture/source-repositories.md) treats Scind
and Xcind as references with different roles, not interchangeable authorities.
The [BDS-252 issue](https://linear.app/beausimensen/issue/BDS-252/specify-and-prove-lds-cross-project-reconciliation)
calls for deterministic scan and journal evidence without source edits or an
LLM facilitator. Xcind's
[standing procedure](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/maintenance/cross-project-sync.md)
and ledgers demonstrate one project's reconciliation process; neither supplies
LDS's universal authority rule or journal schema.

The [Stage 1 evidence](../product/bds-252-stage-1-evidence.md) pins Scind
`c63cbc686afd2ca6ecc01eab7825d637d49378c7` and Xcind
`c244856e7119547ed6dceb2b7663a01720bdde66`. The
[expected-case cards](../product/bds-252-stage-1-expected-case-cards.md)
distinguish a same-topic targeting conflict, documentary credit for an
external-proxy promotion with a contradictory map classification, a
future-versus-rejected capability, and a synthetic one-to-many relation.
These cards are evidence and test expectations, not decisions to edit either
project. The owner ratified D1A–D5A for this LDS boundary.

## Decision

1. **D1A — Caller-owned, explicitly located journal.** The caller names and
   owns the journal location outside **both** source snapshots. Scans do not
   implicitly create a journal. Tests write only to a disposable journal
   location. Neither source project is silently made the journal owner or
   write target.
2. **D2A — Pair- and topic-specific authority.** An owner explicitly declares
   the authority and permitted port/backport direction for each project pair
   and topic. An unknown or unapproved authority/direction remains unresolved;
   a source project's title, ADR number, or Xcind's Scind-canon procedure does
   not decide it. A declaration of direction is not permission to edit either
   source project.
3. **D3A — Reviewed correspondence.** Topic keys and relationships require
   review. One-to-one, one-to-many, many-to-one, and unmatched concepts are
   representable without duplicating or forcing a match. A deterministic scan
   reports observable path/byte-hash differences and missing mappings within
   its declared scope; it does not infer semantic equivalence, confidence,
   authority, or a port decision from matching names or numbers.
4. **D4A — Preserve decisions and distinguish outcomes.** Pending candidates,
   intended port/backport work, deferrals with explicit revisit triggers,
   retained divergences with rationale, and applied outcomes are distinct.
   An applied outcome requires an externally verifiable artifact of the
   actual transfer, not an intent, a matching hash, or a ledger status alone.
   A repeated scan preserves existing candidate identities and decisions;
   changed evidence is surfaced for review without automatically reopening,
   closing, erasing, or reclassifying a decision. Reconsideration is an
   explicit reviewed action against its trigger and new evidence.
5. **D5A — Deterministic plumbing boundary.** Provide a scan and explicit
   journal operations with versioned machine-readable JSON and explicit
   source and journal paths. Scan is read-only; journal operations may write
   only within the caller-named journal root. No porcelain command or LLM
   facilitator is required for this Matter. This decision does not fix exact
   command spellings, record layout, fields, exit codes, or implementation
   architecture; the subsequent operation contract must specify them before
   implementation.

## Evidence trust and consequences

The Stage 1 trust boundary applies to every candidate. Scind's pinned tree
supports statements about its accepted design text, **not** Scind runtime
behavior. Inspected Xcind command/library paths support only the bounded
as-built observations listed in the evidence note; no full Xcind self-sync,
regenerated map diff, or runtime integration test was performed. BDS-119's
missing-ADR allegation concerns an earlier map: the pinned map now includes
ADRs 0022/0023, but its external-proxy file/JSON rows say `XCIND-ONLY` while
its topic row says `MATCH`. Treat that relationship classification as
contradictory and unresolved rather than selecting the convenient row.

For example, Scind ADR-0011 and Xcind ADR-0023 are a reviewed same-topic
targeting disagreement, not evidence that either side must change. The
external-proxy records credit a documentary promotion to Scind ADR-0017,
but do not prove runtime parity or justify a new pending port. Xcind's
TCP/SNI divergence concerns Scind's **Future** intent rather than an observed
shipped Scind gap; its recorded trigger calls for reconsideration if that
intent becomes active design. The synthetic split card tests cardinality
without asserting any real correspondence. Keep unverified areas explicitly
untrusted instead of manufacturing findings or treating this spot check as a
complete reconciliation round.

This decision authorizes the design and disposable validation of scan/journal
plumbing only. It does **not** name or authorize a live pilot, a real-pair
apply or port, any mutation of Scind/Xcind or another source repository, an
automatic transfer, or the paused BDS-251 paired CLI/skill workflow. Applied
evidence for a real transfer would require separate source-project authority
and verification; the journal must not claim that such a transfer occurred
merely because an intention was recorded.
