# Assess documentation without editing it

Use `lds plumbing docs assess` on Linux to collect documentation evidence,
local-link findings, reviewer-authored proposals and unresolved questions in
one report. It never changes the target. A human or agent supplies the review
brief and proposed wording; LDS does **not** automatically infer semantic
impacts, invent replacement instructions or call a model.

## Run a disposable assessment

With a current `lds` binary on `PATH`, run:

```sh
project=$(mktemp -d)
review=$(mktemp -d)
mkdir "$project/docs"
printf '# Usage\nSee [details](details.md).\n' > "$project/docs/usage.md"
printf 'example --mode current\n' > "$project/command.txt"
source_hash=$(sha256sum "$project/command.txt" | cut -d ' ' -f1)
cat > "$review/brief.json" <<EOF
{
  "schemaVersion": 1,
  "reason": "Review instructions after a mode change",
  "audience": "CLI users",
  "evidence": [{"path": "command.txt", "sha256": "$source_hash"}],
  "proposals": [{
    "path": "docs/usage.md",
    "summary": "Consider explaining the current mode",
    "evidence": ["command.txt"],
    "proposedText": "Use example --mode current."
  }],
  "questions": ["Should the missing details page be restored or the link removed?"]
}
EOF
lds plumbing docs assess "$project" --brief "$review/brief.json" \
  --output "$review/assessment.md" --json
```

Exit 0 means the report was created, **not** that the documentation is correct
or current. The JSON has `status: "review-required"`, `inventoryExitCode: 0`,
`validationExitCode: 1`, one missing-local-target finding and the supplied
proposal. Open the report: observations, proposed text and questions are
separate, and complete JSON evidence includes observed paths, hashes and
content. The target's files and directories remain unchanged. This fixture's
wording is test-only, not verified instructions for a real product.

## Write a review brief for your project

Every object uses exactly the case-sensitive fields shown above; unknown,
duplicate, missing or null fields are rejected. Top-level arrays may be empty;
each proposal must cite at least one evidence path. Reason,
audience, proposal summaries/text and individual questions must be nonempty.

- Default documentation scope is root `README.md` and Markdown under
  `engineering/` and `docs/`. Their content is included in the report.
- `evidence` explicitly permits additional **regular UTF-8 files** inside the
  named target. Each entry needs a unique slash-relative path and expected
  lowercase 64-character SHA-256. This is the only extra source-read scope;
  LDS does not crawl code, follow symlinks or run project commands.
- A proposal's `path` must be root `README.md` or a Markdown path below either
  documentation track. An absent page can be proposed, but is not created.
  `evidence` references must name inventoried documents or explicitly declared
  source files. Citations bind bytes, not the truth of the interpretation.
- `proposedText` is suggested wording, not an executable patch or approved
  edit. Keep uncertain commands/claims in `questions` until verified.

For a real assessment, the responsible reviewer chooses the target, change
scope, sources and wording. A machine-generated brief is still reviewer
input, not independent verification or permission to edit.

## Keep reports outside the project

`--output` must be an absolute path outside the target, with an existing
parent and an unoccupied filename. Brief and output ancestors cannot be
symlinks. LDS exclusively creates one Markdown file with mode 0600 (subject
to a restrictive umask); it never overwrites an old report or creates parent
directories. Reports and JSON stdout include source content: store/share
them only with the intended audience. No source is sent to a provider.

LDS repeats inventory, validation and evidence collection and checks the brief
again before writing. A source hash mismatch or changed observations refuses
with exit 3 and no report. Invalid brief/path input exits 1, missing required
flags exit 2, safety/occupied-output refusals exit 3, and unexpected I/O errors
exit 4. A late write failure can leave a partial **external report**; inspect
it before choosing a new output. The target is never an edit destination.

Collection is not atomic; trust and keep target/report ancestors quiescent.
There is no guarantee against hostile mount or directory-renaming races.
Review a report's baseline again before acting on it. Link skips, fragments,
external URLs, semantic accuracy and completeness retain the limits in
[safety and scope](../explanation/safety-and-scope.md).

Accept, reject or defer recommendations through your normal project review
process. Use an ordinary editor for separately authorized changes, then run
appropriate link/command checks. This operation has no apply, confirmation,
retry or automatic update mode; adoption's plan/digest cannot apply its advice.
