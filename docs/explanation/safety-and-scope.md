# Safety and scope

LDS's bounded checks are not an audit of a project. Inventory reads root
`README.md` and Markdown below `engineering/` and `docs/`, not all source
files. The link validator checks supported relative local **file existence**;
it does not check fragments, fetch external URLs, validate content semantics,
or parse every Markdown/HTML link form. Unsupported checks are skips, not
passes. A missing-link finding does not itself veto adoption.

The Linux-only `lds review <root> [--json]` composes that same inventory and
local-link validation as a root-only, offline, read-only first pass. It follows
no symlinks, needs no terminal, credential or authored JSON, and prints only
file metadata (sizes/hashes), links, findings, combined skips and limitations
to stdout—not full source text or a report artifact. Exit 1 means a supported
local file target is missing; exit 0 means no such finding, **not** that docs
are complete, correct or current. Absent optional tracks are allowed, and
empty coverage or skipped checks are not passes. It does not validate layout,
completeness, semantics, cross-track consistency, code alignment or runtime
behavior, fetch URLs, call a provider, run project commands, edit sources or
write files. A caller redirecting stdout owns that write. Reads are not an
atomic snapshot: use trusted, quiescent inputs. See the
[disposable review how-to](../how-to/review-documentation.md).

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

Adoption's `docs apply` is Linux-only supported creation: four fixed paths and
required parent directories, never a root README rewrite. Different occupied
files veto every create, identical ones remain untouched; it has no update,
delete, or migration mode. By default assets resolve user-XDG → installed-default →
embedded; the target repo cannot override them without the explicit
selector below. Apply rechecks the plan and uses exclusive/no-follow creates,
but a late failure may leave some files created.
It does not roll back or provide a transaction. Inspect state and make a new
plan and decision before any rerun.

## Project-local asset boundary

The BDS-234 port allows an explicitly human-selected global
`--project-root PATH` to prepend
`<canonical-root>/.lds/assets/<shipped-relative-name>` to the existing
user-XDG → installed → embedded asset chain for eligible **runtime reads**.
Only known shipped names may be selected. The root must exist as a directory;
relative selector paths are relative to invocation cwd and canonicalized.
No cwd, Git or environment discovery selects a project. Positional docs
targets, the root adoption/assessment/update targets, and reconciliation pair
roots do not implicitly select one. Absent a flag, the existing chain remains.
A missing project path falls through per file; invalid roots, symlinks at any
component below the canonical root, nonregular components and filesystem/read
errors refuse rather than falling back. Selected bytes are never silently
replaced with a default after a flow/template parse or validation failure.

The opt-in readers are adoption render/plan/apply and root adopt for the
four adoption templates, plus the adoption, assessment, update and
reconciliation live flow readers and their root adopt/assess/assisted update/
assisted report forms. Project provenance (`source: "project"`, canonical
path where exposed and content hash) makes the selected bytes visible to
review and freshness checks. A changed template or flow requires the
existing fresh review/consent and exact-plan approval where applicable; a
project file is not owner approval. Explicit source selection never broadens
the allowed output paths, source collection scope or provider authority.
Manifest and projection digest/stamps, configuration, install, uninstall,
doctor and installed generated skill bytes remain project-independent
even with a selector. Generated guidance must fetch live assets with the
human's selection rather than copying project text into an installed skill.
Trust the selected directory and its writers; the port does not guarantee
isolation from hostile concurrent writers or make reads atomic. This is a
local integration boundary, not pilot authorization, provider validation,
native skill execution/parity or reconciliation of the Toolsmith contract.

Separately, offline `lds update` and the canonical `docs update-plan` /
`docs update-apply` steps handle only explicitly selected **existing** root
`README.md` or Markdown below `docs/` and `engineering/`. They do not infer
ownership from generated-looking content, create missing files, move, delete,
run project commands, or apply assessment advice. The offline form does not
use a provider. The original
`docs collect` packet—not an authored report—binds the complete observation.
The planner emits exact before/after content and evidence, metadata and
diagnostics to stdout only. The manual route must persist that artifact
exclusively without following symlinks, outside the target in a trusted
private directory. Root `lds update` instead
builds the same plan in memory and displays it **completely** on stderr without
a temporary plan/report artifact or new collection. It refuses a veto before
prompting, flushes queued terminal input after display, and accepts only a
freshly typed `apply sha256:<exact displayed digest>` or `decline`; all other
input or EOF refuses. No `--confirm`/`--yes` bypass or credential is needed.
Personally review the whole plan and freshly confirm its digest; a digest
copied by script is not authorization. The default offline interface remains
unchanged, even though the root registration is now LLM-kind because explicit
`--assist` can call a model. Root update is not projected into a skill.

Assisted update requires explicit selection of 1–32 unique existing scoped
Markdown targets and a reason/audience; cited or inventoried files are not
implicitly selected for edits. It automatically collects bounded original
evidence (including explicitly selected source files), then displays the
**complete outbound system and user payload** for fresh exact `send` consent.
The OpenCode Go `deepseek-v4-flash` request is tool-free and fixed, with no
redirects, retries or fallback. No payload content is independently verified
as true or redacted; do not select secret material. Drafted full replacements
and questions are unverified, not evidence or approval. Zero edits validly end
`review-required` without apply. For nonempty drafts, a strict original-baseline
deterministic update plan and complete exact-byte preview precede a **separate**
freshly typed `apply sha256:<digest>` or `decline`. Transfer consent never
authorizes writes. Rechecks of the baseline and live flow before/after transfer
and before apply refuse drift rather than silently adopting a new observation.
The Linux write and recovery limits below apply unchanged.

The new `lds plumbing update [--json]` is only a preparatory live
`flows/update.md` reader, unavailable for execution. Generated Claude Code
guidance fetches it live rather than maintaining its own sequence. The skill
harness must disclose its own processing and get its own consent; it cannot
automate the root terminal or infer approval from model output. Guidance is
not enforcement. Mock/disposable checks do not establish a real provider,
target/pilot, native skill execution or CLI/skill parity.

Linux update apply refuses symlinks, multi-linked or nonregular selected
files, special mode bits, selected xattrs/extended ACLs, default ACLs on
staging parents and unsupported or unreadable metadata. It preserves UID/GID
and permission bits without `chown`; inode, timestamps and open-handle
behavior are **not** preserved. Selected content/metadata, ancestors and
original inventory, source evidence, findings and skips are rechecked for
staleness. Each file is staged in its verified parent and replaced separately:
there is no multi-file transaction, rollback, hostile-writer compare-and-swap,
or safe automatic retry. Trusted, quiescent root/parents and other writers are
required. `refused`, `partial` and `indeterminate` results demand inspection
of actual target/scratch state followed by a new observation, plan and human
review before another attempt. No real target or pilot is authorized merely
by this implemented command. See the [update how-to](../how-to/update-documentation.md).

Reconciliation checks named clean roots against full pins, reads only scoped
tracked evidence and explicit raw maps, and optionally compares a named
journal. It never infers correspondence, confidence, authority, or direction;
never edits source files or transfers changes. Journal writes stay at the
caller-named location outside both roots/Git directories. A decision or
`appliedProof` is caller-supplied, not an authenticated external fact.

The Linux root-only `lds reconcile journal record|decide|reconsider` hosts
guide those explicit journal operations without caller-authored scan or request
JSON. They freshly scan clean pinned roots and tracked raw maps, display
candidate evidence and prior decisions, and ask for plain-text reviewed
correspondence, provenance and assessment (or an optional complete replacement
on reconsider), actor/rationale, authority declaration, permitted directions,
outcome and reconsider trigger as appropriate. The host never derives those
claims from matching bytes, names or map labels. `applied` additionally needs
explicit direction, authority and an external proof reference; neither a
recorded reference nor a successful command verifies a transfer or edits a
source. The full digest-bound request/evidence/journal preview and prompts go
to stderr even under `--json` (successful JSON goes to stdout). Queued input is
flushed before fresh exact `record sha256:<digest>`, `decide sha256:<digest>` or
`reconsider sha256:<digest>` confirmation; `decline` refuses. A preview or
decline creates no journal, lock or output file. Pair, pinned sources, maps and
journal are rechecked before write, but this is not an atomic snapshot or
protection from hostile concurrent writers. No automatic retry follows drift
or an indeterminate write: read the journal and inspect revision/request ID
before any new attempt. These hosts do no model/network calls or project
commands beyond the existing read-only Git scan. Manual journal plumbing
remains supported. The hosts are root-only plumbing-kind, not generated skill
entries and not a shared reconciliation flow; no real pair, owner authority,
pilot, provider or native parity is established.

The Linux-only offline `lds reconcile report` and deterministic
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
truncate selected content. The unchanged offline report has no provider or
terminal requirement. `lds plumbing reconcile collect` instead emits the
original bounded evidence JSON on stdout without report publication; saving
it externally is the caller's private-write responsibility. The
extended plumbing report requires `--observed`, `--draft` and `--receipt`
together, validates against that exact baseline and never silently refreshes
it. These private artifacts are caller-owned and should be protected.

Explicit `lds reconcile report --assist` requires a Linux terminal and
`OPENCODE_API_KEY`. Before any transfer it displays the complete system/user
content and OpenCode Go destination, flushes queued input and requires a
fresh exact `send`; `decline` sends nothing. One fixed tool-free
`deepseek-v4-flash` request has no redirects, retries or fallback. Only
explicitly selected full scoped/map text, sanitized scan metadata, seed
claims/questions and live flow leave the host; no host-added absolute paths,
credentials or raw journal contents are included. Selected text and claims
are **not redacted**: review for secrets and prompt-injection text before
consenting. The total prompt is bounded to 512 KiB and response to 64 KiB.
Flow and original inputs are rechecked before send, after response and before
private publication; drift refuses, but these checks are not an atomic
snapshot or a guarantee about provider retention. The report records
assistance and an attributed provider/flow/disclosure receipt, **not** an
authenticated consent or trustworthy model. Strict schema-1 draft claims
may only propose needs-review assessments citing explicitly selected current
captured evidence; empty claims are valid. Reviewer seed assertions, model
advice and journal history remain distinct. None determines semantic match,
authority, direction or applied proof. The live
`lds plumbing reconcile flow --json` reader is preparatory; a generated
skill must disclose its own harness processing and obtain its own consent,
not automate the root terminal. Neither projection nor disposable tests prove
native skill/parity or a real provider call. The separate guided journal
hosts do not turn the report into a decision or a full facilitator. A real
pair or pilot requires separate authorization.
