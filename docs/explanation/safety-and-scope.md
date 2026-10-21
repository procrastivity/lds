# Safety and scope

LDS's bounded checks are not an audit of a project. Inventory reads root
`README.md` and Markdown below `engineering/` and `docs/`, not all source
files. The link validator checks supported relative local **file existence**;
it does not check fragments, fetch external URLs, validate content semantics,
or parse every Markdown/HTML link form. Unsupported checks are skips, not
passes. A missing-link finding does not itself veto adoption.

Assessment creates a new external report, not any adoption output or target
edit. The report and JSON contain default documentation bytes and additional
UTF-8 source files explicitly permitted by the brief's paths/hashes; protect
them accordingly. Proposals and questions are reviewer input, not inferred or
verified facts. Repeated collection detects changed evidence but is not an
atomic snapshot. Exit 0 means report creation, even with missing-link findings.
Linux no-follow/exclusive output creation requires trusted, quiescent ancestors;
a late I/O error may leave a partial external report. Assessment has no provider,
automatic drafting, apply or overwrite mode. See the
[assessment how-to](../how-to/assess-documentation.md) for its brief and limits.

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
