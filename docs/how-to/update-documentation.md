# Update an existing documentation file

`lds plumbing docs update-plan` previews explicitly requested full-file
replacements; `update-apply` requires a separately reviewed plan and fresh
exact-digest human confirmation. This offline Linux workflow does not use a
provider, an agent skill, or an assessment report as write authority. It is
separate from create-only adoption. Use it only on a target and paths you are
authorized to change; no real project is selected by this guide.

## Prepare a disposable plan

With a current `lds` binary on `PATH`, run the following in a trusted shell.
The two new private directories are separate, and the review directory is
outside the target. `set -C` protects this example's shell redirections from
accidentally overwriting files; the host is responsible for exclusive,
no-follow persistence of planner stdout in a trusted existing directory.

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
lds plumbing docs update-plan "$project" \
  --observed "$review/collected.json" --request "$review/edits.json" --json \
  > "$review/plan.json"
```

Do not feed an assessment *report* to `--observed`: it requires the original
`docs collect` packet. The request is strict schema-1 UTF-8 JSON with exactly
`schemaVersion` and `edits`; each edit has exactly `path`, `after` (the entire
new UTF-8 file, not a diff), nonempty `rationale`, and nonempty `citations`
referencing collected evidence. Unknown, duplicate, case-aliased or null
fields refuse. Select only existing root `README.md` or `.md` files below
`docs/` or `engineering/`; no creation, move or deletion is requested.

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
