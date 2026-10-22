# Review cross-project reconciliation evidence

Shared report-only procedure for explicit root `lds reconcile report --assist`
and generated reconciliation guidance. Read live through `lds plumbing reconcile
flow --json`. The reader is preparatory and executes nothing. Offline report,
scan and journal plumbing remain available separately. Guidance does not
authenticate people, enforce consent or establish native/CLI parity.

1. **Establish an authorized scope.** Require explicit clean pinned Git roots,
   an external v1 pair descriptor, tracked raw maps, an external seed review
   brief and a new external report path. An optional external journal must
   already exist. The brief declares purpose, asserted reviewer/time, explicit
   full-content selections by side/path, and any human claims/questions.
   Matching filenames, byte equality and map labels are not correspondence,
   authority or applied proof. Never fetch, checkout, run project commands,
   edit either project or write/initialize/lock a journal in this flow. A
   harness must disclose its own processing and obtain permission before
   reading project content; installation and past consent do not authorize it.

2. **Preserve the original bounded capture.** Use `lds plumbing reconcile
   collect --pair <pair.json> --left-root <left> --right-root <right> --map
   <map> [--map ...] --review <seed.json> --output <new-report.md> [--journal
   <existing-journal>] --json`. Keep the exact result privately outside both
   roots/Git directories, not a refreshed substitute. Collection publishes
   nothing. It bounds 64 scoped files, 16 maps, 64 KiB per raw file, 512 KiB
   selected text and 2 MiB pair/review/journal metadata. Only explicitly
   selected present scoped/map UTF-8 content is included in full. Refuse unsafe
   paths, occupied output or stale/unclean pins; do not truncate scope.

3. **Disclose the actual transfer and obtain fresh consent.** Root CLI
   displays complete system instructions and user payload: live flow, seed
   brief, scan metadata and selected full contents/hashes. Host absolute
   paths/identities and raw journal history are excluded from transfer, not
   from the private local report. Selected text and authored claims are not
   redacted and may themselves contain sensitive data. Show destination,
   byte counts and retention caveat. Total prompt budget is 512 KiB. Require
   Linux terminal and OPENCODE_API_KEY for root assistance; flush queued input
   after display and require fresh exact `send`. Recheck original evidence and
   live flow before sending. Use one OpenCode Go/deepseek-v4-flash tool-free
   bounded request, no redirects, automatic retries or fallback. A harness
   cannot reuse root consent or automate its terminal, and must disclose its
   own actual processing. Stop if the available provider/consent mechanism
   cannot meet this procedure; never fabricate a receipt for a different one.

4. **Draft proposed claims and questions, never decisions.** Strict reply
   shape is `{schemaVersion:1, claims:[], questions:[]}` within 64 KiB, no
   fences, null required arrays, unknown/duplicate keys or approval fields.
   Each claim uses the existing review claim schema, status `proposed`,
   assessment code `needs-review`, valid correspondence cardinality and
   nonempty exact current candidate provenance. Every cited artifact/map must
   have explicitly selected captured text with matching project, revision,
   path and hash. Use bounded trust and not-assessed confidence for advice.
   No authority, direction, outcome or applied proof fields. Preserve original
   human assertions/questions and saved journal history separately; do not
   replace, duplicate or promote them. Empty claims is valid; missing evidence
   and unresolved authority belong in questions, not an invented match.

5. **Recheck and publish one private review-required report.** Root invokes
   the same deterministic core in memory against its original capture. The
   artifact route uses `lds plumbing reconcile report` with the same original
   pair/root/map/review/output flags plus `--observed <original-collect.json>
   --draft <draft.json> --receipt <receipt.json>`. All three artifacts must be
   explicit bounded external regular files. Receipt records actual provider,
   model, flow source/SHA-256, exact system/user SHA-256 and byte counts and
   consentAt RFC3339 time; it is an attributed receipt, not authenticated proof
   of disclosure. Current assistance accepts OpenCode Go/deepseek-v4-flash;
   do not mislabel another harness/model as this provider. Refuse input/flow
   drift after drafting and before publication, invalid scope/citations or
   output collisions; no silent recollection, repair or retry. Publish a new
   mode-0600 external report with proposed advice and disclosure provenance.
   On late uncertain I/O inspect the named output/scratch before another run.

6. **Leave decisions with a separate human review.** Report success means
   review-required advice, not verified semantics, transfer, authority,
   approval, pilot or parity. No source/journal writes occurred. Retain human
   and saved journal assertions as attributed history. Any later journal
   record/decide/reconsider needs its own fresh scan, authority/proof claims
   and explicit independent approval through its separate workflow. Do not
   promote model proposals to journal decisions or source changes.
