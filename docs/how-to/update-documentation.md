# Update an existing documentation file

The existing `lds update --observed ... --request ...` route is an offline,
human-facing Linux workflow for explicitly requested full-file replacements.
It displays the complete deterministic plan and requires fresh exact-digest
terminal confirmation. Explicit `--assist` is a separate, consent-gated
drafting mode described below. The separate
`lds plumbing docs update-plan` / `update-apply` route lets you persist and
review a plan before a distinct apply command. The offline and manual routes
use no provider. Neither a provider response, an agent skill nor an assessment
report is write authority. All routes are separate from create-only adoption.
Use them only on a target and paths you are authorized to change; no real
project is selected by this guide.

## Request assisted drafts for explicit existing files

In a trusted Linux terminal, the assisted command shape is:

```text
lds update <root> --assist --reason <text> --audience <text> \
  --document <relative-existing-markdown> [--document ...] \
  [--source <relative-file> ...] [--json]
```

For example, on the disposable `project` created below, you could select
`--document README.md`; the reason and audience explain the requested review.
Select 1–32 unique **existing** targets: root `README.md` or Markdown below
`docs/` or `engineering/`. Only these documents can be edited, never other
inventoried documents or cited sources. `--source` explicitly adds bounded
regular UTF-8 evidence files, not edit targets. Inspect the selection and
contents for secrets before proceeding. The command collects its own original
bounded observation; do not supply `--observed`/`--request` in this mode.

Before any model request, inspect the **complete** displayed outbound system
and user payload, including the live flow, reason, audience, inventory,
findings, skips, selected paths, hashes and full scoped contents. Only a
freshly typed exact `send` after that disclosure permits one fixed OpenCode Go
`deepseek-v4-flash` tool-free request; decline instead if anything is out of
scope. The mode needs `OPENCODE_API_KEY` for transfer, unlike the offline route.
No automatic redirect, retry or fallback is made. The model's full replacement
drafts and questions are **unverified**. An empty edit set is valid and ends
`review-required` without applying anything.

For nonempty drafts, LDS builds a strict deterministic plan against the
**original**, not a silently refreshed, baseline. Review the full exact-byte
preview, including all before/after content and diagnostics. A send is not
write approval. Only a separate freshly typed
`apply sha256:<exact displayed digest>` authorizes an eligible plan; `decline`
refuses. There is no unattended confirmation shortcut. Baseline and live flow
are rechecked before/after transfer and before apply; drift refuses instead
of silently recollecting or retrying. Inspect the final exit status, stderr,
JSON if requested, and actual files. Partial or indeterminate replacement
requires tree/scratch inspection, new collection and a new review—never an
automatic rerun. The same Linux metadata, no-follow, per-file replacement and
concurrency restrictions below apply. No live provider, real target, native
skill execution or CLI/skill parity is established by disposable/mock checks.

## Prepare disposable inputs for the offline route

With a current `lds` binary on `PATH`, run the following in a trusted shell.
The two new private directories are separate, and the review directory is
outside the target. `set -C` protects this example's shell redirections from
accidentally overwriting files; for the manual route below, the host is
responsible for exclusive, no-follow persistence of planner stdout in a
trusted existing directory.

```sh
umask 077
set -C
project=$(mktemp -d)
review=$(mktemp -d)
printf '# Example\nOld instructions.\n' > "$project/README.md"
lds plumbing docs collect "$project" \
  --reason 'Review example instructions' --audience 'Example readers' --json \
  > "$review/collected.json"
cat > "$review/edits.json" <<'JSON'
{"schemaVersion":1,"edits":[{"path":"README.md","after":"# Example\nNew instructions.\n","rationale":"Replace the obsolete example wording","citations":["README.md"]}]}
JSON
```

Do not feed an assessment *report* to `--observed`: it requires the original
`docs collect` packet. The request is strict schema-1 UTF-8 JSON with exactly
`schemaVersion` and `edits`; each edit has exactly `path`, `after` (the entire
new UTF-8 file, not a diff), nonempty `rationale`, and nonempty `citations`
referencing collected evidence. Unknown, duplicate, case-aliased or null
fields refuse. Select only existing root `README.md` or `.md` files below
`docs/` or `engineering/`; no creation, move or deletion is requested.

## Invoke the offline human-facing update

Run this yourself in a Linux terminal with an unchanged disposable fixture:

```sh
lds update "$project" \
  --observed "$review/collected.json" --request "$review/edits.json" --json
```

The command builds a plan in memory from those original inputs (it does not
collect a new baseline or write a temporary plan/report). It displays the
**complete** deterministic plan on stderr, including original observations,
exact before/after text, metadata, findings, skips, veto and `planDigest`.
Review all of it and the authorized target, not just the digest. A veto refuses
before prompting. Only after you have inspected the whole display, type
`apply sha256:<exact displayed digest>` freshly at the prompt to authorize
that plan, or type `decline`. Queued input from before the display is flushed;
anything else or EOF refuses. There is no `--yes` or `--confirm` bypass.

The command then uses the same strict in-memory apply as the manual route.
Its final schema-1 JSON on stdout has `operation: "update"`, `status`, `plan`,
and `apply` (null on veto or decline); presentation, prompts and errors use
stderr. Inspect exit status, both streams and actual effects: `refused`,
`partial` or `indeterminate` does not mean success. No credential or provider
is involved. This is an *alternative* to the manual route below, not a
prelude to automatically applying its plan. If you applied it, do not reuse
the old observation or request against the changed fixture.

## Alternatively, review a separate plumbing plan

For an unchanged fixture (recreate and recollect if the root route already
applied), run:

```sh
lds plumbing docs update-plan "$project" \
  --observed "$review/collected.json" --request "$review/edits.json" --json \
  > "$review/plan.json"
```

Read **the entire** `"$review/plan.json"`, not only its digest: check target,
original observation, every exact `before` and `after`, citations, selected
metadata, ancestors, findings, skips and veto. The plan includes complete
collected content and may be sensitive. A missing-link finding is evidence,
not by itself an update veto or proof of correct prose. If anything is wrong,
edit the request or address changed evidence, collect again, and review a new
plan. The planner writes only to stdout; the example's shell persists it
outside the target.

## Confirm separately, after human review

Only after personally reviewing the whole artifact and confirming that the
target is authorized, copy its exact `planDigest` (`sha256:...`) into a *fresh*
command. For the disposable fixture above, the command shape is:

```sh
lds plumbing docs update-apply "$project" \
  --plan "$review/plan.json" --confirm 'sha256:PASTE_THE_REVIEWED_DIGEST_HERE' --json
```

Do not script extraction of `planDigest` into `--confirm`, and do not treat a
model's `send`, an accepted assessment, or an adoption confirmation as update
approval. This command is deliberately separate from the runnable plan example.
Inspect its exit status, JSON actual-effects result and stderr before drawing
conclusions; a refused, partial or indeterminate result is not success.

Only single-link regular files without disallowed mode bits, selected
xattrs/extended ACLs or a default ACL on their staging parent are eligible.
The staged same-parent replacement preserves UID/GID and permission bits
without `chown`, but changes inode and timestamps. It replaces one file at a
time, not all files transactionally; trusted, quiescent ancestors and writers
are required. A stale source, selected file, metadata, ancestor, inventory or
link diagnostic stops the reviewed baseline from being silently replaced.
Failures may leave some files updated, others untouched, or a scratch file;
never automatically rerun or roll back. Inspect the actual tree, recollect and
make a **new** plan and review before another attempt. See
[safety and scope](../explanation/safety-and-scope.md) and the
[command reference](../reference/commands.md) for limits.
