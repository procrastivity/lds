# Initialize documentation offline

`lds init <root> --facts <file> [--json]` is the Linux terminal-only,
provider-free route to the existing four-file, create-only adoption contract.
It uses no provider, key, live flow reader or LLM. Disposable verification
does not establish release, pilot or native-skill/parity
evidence. For separate saved-plan review, use the
[adoption plumbing route](adopt-a-project.md); `lds adopt` remains a distinct
advisory-provider workflow.

Choose an explicit root that **already exists** and a complete
[approved fact bundle](../reference/facts.md). LDS does not create a project
root, infer facts, authenticate an approver or verify capability claims.
Facts-file approval is not approval of the eventual plan. This guide selects
no real target and grants no real-project write authority.

## Prepare a disposable example

With a current local `lds` binary on `PATH`, run this from the LDS checkout
in a trusted shell. These copies are **test-only**: retain the fixture's
simulated-approval label and never reuse its facts or output claims for a
real target. The facts stay outside the disposable target.

```sh
umask 077
project=$(mktemp -d)
review=$(mktemp -d)
cp internal/asset/testdata/adoption/greenfield/target/word-count.sh "$project/"
cp internal/asset/testdata/adoption/greenfield/target/sample.txt "$project/"
cp internal/asset/testdata/adoption/greenfield/facts.json "$review/facts.json"
(cd "$project" && sh word-count.sh count sample.txt)
```

The disposable operation should print `Word count: 4`. Its source and sample
are the fixture evidence, not proof about another project. For a real use,
prepare independently verified, owner-approved facts about that target
instead; do not replace the fixture's simulated approval with a real claim.

## Review and decide in the terminal

Run directly in a Linux terminal, not through a stdin pipe:

```sh
lds init "$project" --facts "$review/facts.json" --json
```

Read the **entire** deterministic plan JSON on stderr, not just its digest.
The nested operation remains `docs plan`. Check target/root identity, facts
digest, every output's classification and exact proposed content, existing
content/hashes/provenance, template sources/hashes, affected links, necessary
and existing parents, preserved paths, report-only root README notice,
findings, skips, `veto`, and `planDigest`. No fields are replaced by a summary.
Incomplete facts, unsafe paths, collisions or legacy LDS evidence refuse
before a prompt or writes; broken-link findings alone are not a veto.

For an eligible plan, LDS flushes queued input **after** display. Only then,
after personally reviewing the whole plan, freshly type either:

- `apply sha256:<exact displayed digest>` to approve that plan; or
- `decline` alone to refuse without writes.

Replace the digest placeholder with the exact displayed value. Any other
text, a bare digest or EOF refuses. There is no `--yes` or `--confirm` bypass;
do not script digest extraction or pipe approval into stdin. `--json` does
not make this noninteractive. No temporary plan or report is created: apply
uses the exact displayed plan held in memory and the same asset selector.

Only these four files and their necessary parents may be created:

| File | Purpose |
| --- | --- |
| `engineering/README.md` | Engineering navigation. |
| `engineering/product/overview.md` | Supplied identity, purpose, current capability and sources. |
| `docs/README.md` | User documentation navigation. |
| `docs/reference/usage.md` | Supplied verified operation, prerequisites, invocation, evidence and limits. |

The root `README.md` is report-only. Identical occupied outputs stay untouched;
**any** different occupied output or recognizable/ambiguous legacy LDS
evidence vetoes **all** creates, not just the differing file. No migrations,
replacement, deletion, Git commands, harness installation, other output files
or extra empty layers are part of init.

## Inspect the result and any rerun

With `--json`, final stdout contains `schemaVersion: 1`, `operation: "init"`,
`status` (`applied`, `current`, `refused` or `partial`), the full `plan` object,
and `apply` (an object when attempted, otherwise null). The nested apply keeps
`operation: "docs apply"`; inspect its `created`, `identicalUntouched`,
`unattempted`, `failed` and `preserved` fields as well as status, stderr, exit
code and actual files. Without `--json`, stdout is a human status summary.
Early failures such as noninteractive invocation emit only the existing
stderr error envelope and leave stdout empty, not a fabricated init result.

After a successful apply, run the same command again:

```sh
lds init "$project" --facts "$review/facts.json" --json
```

An identical rerun still needs a **new full review and fresh confirmation**.
After confirmation it returns `current` with no file creates. It is not an
unattended idempotence shortcut. Declining either run returns `refused`.

Changed facts, selected template sources/bytes, root/parents, preserved
evidence or diagnostics refuse through existing stale-input checks; approval
never carries to a silently refreshed plan. Writes use Linux exclusive,
no-follow per-file creation, not a transaction. A late failure may leave
parents or some files created, including a partial failed output; there is no
rollback or automatic retry. Inspect actual state, make a new plan and obtain
a new review before any further attempt. Use trusted, quiescent writers and
ancestors; checks are not an atomic snapshot or hostile-writer isolation.

## Optional explicit project templates

To select project-local runtime templates, supply the global flag explicitly:

```sh
lds --project-root "$project" init "$project" --facts "$review/facts.json" --json
```

The target never implicitly selects templates. Only the existing four
adoption templates are read: project → user-XDG → installed → embedded,
falling through per missing file and refusing unsafe/unreadable selections.
No adoption flow is read. Inspect the selected provenance in the full plan;
the same selector is retained through apply. See the
[asset selector reference](../reference/commands.md#project-root-runtime-asset-selection)
and [safety boundaries](../explanation/safety-and-scope.md).
