# Adopt LDS documentation in a project

Choose an **existing, explicit** project root and a responsible human who can
verify its current capability and approve the [complete fact bundle](../reference/facts.md).
Facts are caller assertions, not independently authenticated truth. Do not
use the fixture's simulated approval as authorization for a real project.
No project is implicitly selected or approved by LDS.

## Read-only preparation

With `root` set to the intended existing root and `facts` to the reviewed JSON:

```sh
review_dir=$(mktemp -d)
plan="$review_dir/reviewed.json"
lds plumbing docs inventory "$root" --json
lds plumbing docs validate "$root" --json
lds plumbing docs render "$root" --facts "$facts" --json
lds plumbing docs plan "$root" --facts "$facts" --json > "$plan"
```

Save the plan outside the target tree. Inspect each command's exit status;
validation exit 1 supplies link findings to review, not permission to ignore
other failures. Stop on a failed render or vetoed/refused plan.
Review the entire inventory (including skips), link findings, exact rendered
content and asset provenance. The plan is read-only: open **all** of
the file named by `$plan`, including every output's content/classification,
existing bytes/hashes, necessary parents, preserved paths, findings, skips,
report-only README notice, `veto`, and `planDigest`. Keep the exact plan bytes for apply.
Only a non-vetoed plan with a non-null digest is eligible. A differing occupied
file, unsafe path, or recognizable/ambiguous legacy LDS tree vetoes **every**
create; do not move files out of the way to make an unreviewed plan pass.
Broken-link findings alone are diagnostics, not a veto.

The four possible output files are `engineering/README.md`,
`engineering/product/overview.md`, `docs/README.md`, and
`docs/reference/usage.md`. Missing parents may be created. The root
`README.md` is only reported, never edited. Existing identical outputs are
untouched; different occupied outputs cannot be updated, deleted, or migrated.
Template resolution is user-XDG override → installed default → embedded,
with no target-repository override. Review source and hashes in the plan.

## Decide, then apply (Linux)

Obtain a **fresh human confirmation** of the exact digest after the person
has seen the full plan. This is a separate decision from the facts-file
`approval`, not a requirement for a second person. Do not pipe a parsed digest
straight to `--confirm`, auto-confirm in a script, or invoke apply on a decline.
Once a human has independently entered the reviewed digest, the explicit
plumbing form is:

```sh
lds plumbing docs apply "$root" --facts "$facts" \
  --plan "$plan" --confirm '<human-entered sha256:... digest>' --json
```

Replace the quoted placeholder with the **exact** confirmed `planDigest`,
not its label or brackets. Apply re-plans and refuses changed facts, assets,
diagnostics, root/parents, stale confirmation, or veto. It creates only
missing approved files using Linux no-follow/exclusive operations. Inspect
`created`, `identicalUntouched`, `unattempted`, `failed`, `preserved`, and
`status`, plus stderr and exit status. An identical rerun is a no-op only
after a newly reviewed plan and decision. Late failure can be **partial**:
there is no rollback or four-file transaction. Inspect actual state, generate
a new plan, and obtain a new review rather than blindly retrying.

Alternatively, `lds adopt <root> --facts <file>` is the **interactive** route.
It uses the live flow and local deterministic operations, asks for exact
`send` before transferring the flow, supplied facts, and bounded inventory
metadata to OpenCode Go (`deepseek-v4-flash`), then presents an advisory
response and the complete deterministic plan. It does not send target source
contents or give the model tools or authority. Only a fresh host-side
`apply <exact planDigest>` response
authorizes creation. To decline, type `decline` alone, with no digest; declining
either prompt stops with exit 3 and no target writes. It requires a terminal
and an `OPENCODE_API_KEY` available in its environment; use your secure
credential setup rather than putting a key in a command, document or log.
Noninteractive invocation refuses before provider access or target writes.
The advisory cannot verify facts, grant approval, or choose a different output
route. For provider-free review, use the deterministic plumbing commands above.
