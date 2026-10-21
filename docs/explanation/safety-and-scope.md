# Safety and scope

LDS's bounded checks are not an audit of a project. Inventory reads root
`README.md` and Markdown below `engineering/` and `docs/`, not all source
files. The link validator checks supported relative local **file existence**;
it does not check fragments, fetch external URLs, validate content semantics,
or parse every Markdown/HTML link form. Unsupported checks are skips, not
passes. A missing-link finding does not itself veto adoption.

Assessment creates a new external report, not any adoption output or target
edit. The existing authored-brief report and JSON contain default documentation
bytes and additional UTF-8 source files permitted by the brief's paths/hashes;
protect them accordingly. Proposals and questions are reviewer input, not
inferred or verified facts. Repeated collection detects changed evidence but
is not an atomic snapshot. Exit 0 means report creation, even with missing-link
findings.
Linux no-follow/exclusive output creation requires trusted, quiescent ancestors;
a late I/O error may leave a partial external report. The existing authored-brief
assessment has no provider, automatic drafting, apply or overwrite mode. See the
[assessment how-to](../how-to/assess-documentation.md) for its brief and limits.

The human route collects the same bounded default documents and
only explicitly selected regular UTF-8 source files, with automatic hashes.
Offline `docs collect` sends nothing; optional `docs assess --observed` checks
the complete prior packet against current observations, not just named source
hashes. Collection and rechecks are not semantic proof, a clean Git pin or an
atomic filesystem snapshot. The interactive `lds assess` preflights a new
external report and live flow before provider access; it displays the **full**
proposed transfer (flow, reason/audience, inventory, findings, skips, relative
paths, hashes and content). Only fresh exact `send` allows that transfer to
OpenCode Go (`deepseek-v4-flash`) using `OPENCODE_API_KEY`. LDS adds no absolute
root/report metadata or credentials to the model payload. Selected contents
are not redacted: do not include secrets. Transfers above 512 KiB refuse rather
than truncate; response content is limited to 64 KiB. One non-streaming request
has a 4096-token output bound, 60-second timeout, no tools, redirects or retries;
model output is never executed.
Adoption consent does not authorize this disclosure. A decline or changed
baseline/flow stops instead of recollecting or silently retrying. The report
labels model-authored proposals as unverified, `review-required` advice. It
never edits the target. The CLI and generated skill share the live
`flows/assess.md`, but `lds plumbing assess` is only a preparatory reader; a
skill must obtain its own consent and disclose harness data processing.
Guidance is not enforcement, and offline checks establish neither live
provider reliability nor native skill execution/parity, release inclusion or
a pilot.

Adoption facts are caller assertions: structural validation cannot verify
approval, evidence, or current capability. The interactive provider sees
the live flow, supplied facts, and bounded counts after explicit `send`;
its answer is advisory, with no tools or source access. The human must review
the **full** deterministic plan and independently confirm the fresh exact
digest. Neither provider output, a facts-file approval, nor a script that
extracts the digest is authorization. Generated Claude Code `/lds` guidance
reads the live flow, while `lds plumbing adopt --json` itself is
`preparatory`/`unavailable-for-execution`. Guidance is not enforcement;
installation does not prove native adoption or parity.

Docs apply is Linux-only supported creation: four fixed paths and required
parent directories, never a root README rewrite. Different occupied files
veto every create, identical ones remain untouched, and there is no update,
delete, or migration. Assets resolve user-XDG → installed-default → embedded;
the target repo cannot override them. Apply rechecks the plan and uses
exclusive/no-follow creates, but a late failure may leave some files created.
It does not roll back or provide a transaction. Inspect state and make a new
plan and decision before any rerun.

Reconciliation checks named clean roots against full pins, reads only scoped
tracked evidence and explicit raw maps, and optionally compares a named
journal. It never infers correspondence, confidence, authority, or direction;
never edits source files or transfers changes. Journal writes stay at the
caller-named location outside both roots/Git directories. A decision or
`appliedProof` is caller-supplied, not an authenticated external fact.

The Linux-only offline `lds reconcile report` and identical
`lds plumbing reconcile report` add a bounded, private Markdown review packet
with a provenance JSON appendix, not a journal event or authorization. Pair,
review, optional existing journal, and new output must be external to **both**
roots and their Git common directories. No input or output path component for
the report may be a symlink; publication creates a new mode-0600 file without
overwrite. It does not initialize or lock a journal, write source or journal,
or run project commands. Original inputs are revalidated before publication,
not frozen forever or made into an atomic filesystem snapshot. Use trusted,
quiescent ancestors and keep the report private: it may contain selected full
UTF-8 bytes and unverified reviewer claims. The `reviewer` and `at` fields are
assertions, not authentication. Raw map labels, matching hashes, candidate
buckets, journal `appliedProof`, and report claims are not semantic matching,
authority, or verified transfer. Even unselected scoped/map files face a
64-KiB raw-file ceiling; report collection does not silently expand scope or
truncate selected content. There is no assistance, model disclosure, shared
flow, journal-decision UX, or native-skill/parity claim for this slice. A real
pair or pilot requires separate authorization.
