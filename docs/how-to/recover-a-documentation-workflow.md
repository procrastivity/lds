# Recover from a refused or interrupted documentation write

Use this guide for Linux `lds init`, offline or assisted `lds update`, and
their separate adoption/update plumbing. It does not authorize a real
project, broaden a previous approval or make the writes transactional.
Reconciliation reports and journals have separate
[recovery boundaries](reconcile-projects.md).

## Stop writes and preserve the evidence first

Do **not** rerun apply, roll back automatically, delete occupied outputs or
clean up `.lds-*` names to make the next attempt pass. Stop overlapping
writers and arrange trusted, quiescent ancestors before investigating.
If you cannot establish that boundary, stop here.

Keep the original stdout, stderr, exit status, complete plan, facts,
collection packet and request privately **outside** the target. Keep any
provider advice separate from approval and actual-effects evidence. Do not
overwrite an old artifact with a new observation or publish full source
packets in an issue. If the process is still running, establish whether it
has finished before starting any other writer.

If still available in the original shell, save `$?` immediately after the
command, before another command replaces it. Otherwise record the exit as
unknown. Do not rerun a write just to reconstruct its output. Empty,
truncated or missing stdout after an interrupted apply is **not** evidence
that nothing changed.

## Read the result, not just the exit code

Root `init --json` and `update --json` return an envelope with `status`, the complete
`plan` and nullable `apply`. Their nested operations are `docs apply` and
`docs update-apply`. The corresponding plumbing apply commands put the
actual-effects result directly at the top level.

| Outcome | Meaning and next action |
| --- | --- |
| `refused` before apply, with `apply: null` | No apply was invoked on this route. Inspect the refusal and current tree; resolve its cause before a new review. This does not prove another writer left the tree unchanged. |
| `refused` with an apply result | Read its failure and observed-effects fields. A stale check can refuse after inputs changed; do not assume the current tree matches the old plan. |
| `partial` | Some creations/replacements or residual staging effects may exist. Inspect every reported and pending path; there is no rollback. |
| `indeterminate` (update) | Replacement may have occurred but its outcome could not be fully verified. Inspect the failed path even if it is absent from `replaced` or `untouched`. |
| Missing/unusable result | Treat actual effects as unknown. Preserve what remains and inspect before any further write. |
| `applied`, or confirmed init `current` | Check reported effects and actual files. `current` only concerns identical adoption outputs after a fresh decision; neither status verifies the prose or grants future approval. |

Exit **3** means a policy refusal, but a late stale-input refusal can accompany
partial or indeterminate effects. Exit **4** means an internal/I/O failure,
not a promise of zero writes. Exit **2** is usage; **1** covers validation
failures. On `lds review`, exit 1 can instead report supported link findings;
it says nothing about apply effects. Inspect both JSON and stderr instead of
treating all nonzero exits alike.

For adoption, inspect `created`, `identicalUntouched`, `failed`, `unattempted`
and `preserved`. A failed creation can leave a partial output or new parent
directory. For updates, inspect `replaced`, `untouched`, `unattempted`,
`failed.path`, `failed.phase`, `failed.code`, `failed.observed`, `scratch` and
`cleanupError`. `untouched` records readable files still matching the before
state; it is not a fallback label for an unreadable or uncertain file.

## Resolve common refusals without bypassing the guard

| Evidence | Safe response |
| --- | --- |
| `refusal.interactive-required` | Real `facts`, `init` and root update decisions need a Linux terminal for input and stderr. Use read-only review in automation; do not use pipes, prequeued text or the test suite's simulated PTY decisions as real approval. |
| `refusal.confirmation-declined` or `refusal.update-confirmation` | A decline, wrong digest or unavailable fresh decision did not approve that plan. If another attempt is wanted, present the entire new plan and obtain a fresh decision. Never extract a digest directly into apply. |
| `refusal.incomplete-facts` | Verify all assertions, source citations and current command/output evidence. Use [facts resume/review](../reference/facts.md#draft-facts-with-the-offline-helper) on a separate output file if needed; resume resets approval to pending. Facts approval remains distinct from plan approval. |
| `refusal.output-collision` or `refusal.legacy-docs` | Init/adoption cannot overwrite or migrate existing different content. Keep it in place and inspect the full classifications. Any legitimate existing-file maintenance belongs to a separately authorized update or manual change, not a forced init. |
| `refusal.plan-stale` or `refusal.update-stale` | The old facts/assets, observations, bytes, metadata, identity or diagnostics no longer match. Do not restore obsolete bytes merely to satisfy a digest. Inspect actual effects, then rebuild the baseline as described below. |
| `refusal.update-veto`, unsafe paths or ineligible metadata | Inspect the plan's findings and each selected path. Updates require existing scoped regular Markdown, single links and supported metadata. Do not strip ACLs/xattrs, change ownership/modes or follow a symlink just to bypass a refusal. Choose an authorized workflow that can preserve the required metadata. |

Template selection is also part of review. If using `--project-root`, retain
the explicit human-selected root and inspect project → user-XDG → installed
→ embedded provenance. Edits, deletion or a switch of selected source/path
can invalidate an old plan even when some bytes match. See the
[asset selector reference](../reference/commands.md#project-root-runtime-asset-selection).

## Inspect the current tree without writing to it

Set `root` to the explicit target and `incident` to a **new, private, existing
directory outside that root**. Verify both locations first. The following
creates only new external captures; `set -C` refuses occupied capture files.
It does not invoke apply, follow target symlinks or fetch external links.

```sh
umask 077
set -C
review_status=0
lds review "$root" --json \
  > "$incident/current-review.json" 2> "$incident/current-review.stderr" \
  || review_status=$?
printf '%s\n' "$review_status" > "$incident/current-review.exit"
printf 'Current review exit: %s\n' "$review_status"
```

Inspect both captures and the recorded status. Shell/redirection setup
failures are not LDS results. A failed read needs investigation, not an
assumption that the old bytes remain. A successful review inventories only
root `README.md` and regular Markdown below `docs/` and `engineering/`, with
current sizes/hashes and explicit skips. It is not a recovery journal, a full
filesystem snapshot or a semantic verdict. Sources outside that inventory,
metadata and any reported scratch path need separate authorized inspection.

Compare each affected path with the **old** complete plan: adoption output
content/hashes, or update `edits[].before` and `afterSha256`. Establish
whether the current file has before bytes, after bytes, different bytes or
cannot be read safely. Matching content alone does not establish identity,
ownership, permission or durability. Do not assume `unattempted` means
unchanged by other writers, or that a clean link check resolves uncertainty
after a failed rename/sync.

Inspect a reported scratch path without treating its name as proof of
ownership. LDS only attempts cleanup of an owned, identity-checked staging
file; a foreign or uncertain path is left alone. Do not glob-delete scratch
files or rename one over the target. If ownership/identity cannot be
established, preserve the path and ask a maintainer to assess it before any
cleanup. Preserve the recorded `cleanupError` even if the target appears right.

## Start a new plan only after the actual state is understood

- **Init/adoption:** reverify changed facts and selected templates, then make
  a fresh read-only plan against the actual tree. Already identical outputs
  may remain; any different occupied output still vetoes all creates. A
  truncated output is not permission to delete it. Resolve its disposition
  separately, and obtain a fresh full-plan decision before any creation.
- **Existing-file updates:** collect a **new** packet from the inspected
  current state, with the intended reason, audience and explicit sources.
  Re-author the request for only the still-needed existing-file replacements.
  A new plan against the **old** collection does not repair a stale baseline.
  An assessment report is not a collection packet. Review exact before/after
  bytes, citations, metadata, diagnostics and the new digest before a separate
  fresh apply decision. See the [update how-to](update-documentation.md).
- **Assisted modes:** old advice is not authority to change a new baseline.
  Review its relevance again. Any new provider request needs its own complete
  disclosure and fresh send consent; send still does not approve writes.

After an authorized repair, verify actual bytes and unaffected files and run
a new scoped review, retaining skips as unchecked. Do not reuse the old
collection, plan or digest. No step in this guide performs automatic retry,
rollback, migration or scratch cleanup. The
[safety limits](../explanation/safety-and-scope.md) still require trusted,
quiescent writers and per-file, not cross-file-atomic, operation.
