# Assess documentation without editing it

The existing, offline `lds plumbing docs assess` route on Linux collects
documentation evidence, local-link findings, reviewer-authored proposals and
unresolved questions in one external report. A human or agent supplies its
brief and proposed wording; this route does **not** infer semantic impacts or
call a model. The `lds assess` route instead asks a model for
proposals after fresh consent to disclose scoped source content. Neither route
edits the target. Use the authored-brief route when you need a provider-free
assessment; use the interactive route for model-authored draft proposals.

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

## Request draft proposals interactively

Use a terminal on Linux, an `OPENCODE_API_KEY` supplied through your secure
credential setup, and a new report path outside the project. For example,
after reviewing what the default documentation scope and selected source file
will disclose:

```sh
project=/absolute/path/to/project
review=/absolute/path/to/existing/private-review-directory
lds assess "$project" --reason 'Review CLI guidance after the mode change' \
  --audience 'CLI users' --source 'cmd/example.go' \
  --output "$review/assessment.md" --json
```

Replace both example paths and the source with your own. `--source` is
repeatable and accepts only explicit target-relative regular UTF-8 files; it
does not authorize a repository crawl. The default collection covers root
`README.md` and Markdown under `docs/` and `engineering/`. LDS computes their
observed hashes and includes complete content, inventory, link findings and
skips. It displays the **complete proposed transfer**, including the live
flow, reason, audience and selected paths/hashes/content. LDS adds no absolute
root/report metadata or credentials to that payload. Selected contents are
not redacted: do not include secrets. Type the fresh exact `send`
only if that entire disclosure is acceptable; declining stops before provider
access. Adoption's consent does not carry over. The provider is OpenCode Go
(`deepseek-v4-flash`); one bounded, non-streaming request uses no tools,
redirects or retries. The transfer limit is 512 KiB; response content is limited
to 64 KiB and 4096 output tokens, with a 60-second request timeout.
Treat the response as unverified advice: the report is
`review-required`, not a patch, approved edit or semantic proof. Model/flow
text is not executed. The flow and collected baseline are rechecked; a change
refuses rather than silently substituting a new baseline or retrying.

## Bind an authored brief to an offline collection

`lds plumbing docs collect` makes the same complete evidence packet without
a provider or report write. Its `reportPath` is empty and its proposals and
questions are empty. With `--observed`, `docs assess` binds a strict authored
brief to that full packet rather than just its declared source hashes. For a
disposable example using `jq`:

```sh
project=$(mktemp -d)
review=$(mktemp -d)
mkdir "$project/docs"
printf '# Usage\nSee [details](details.md).\n' > "$project/docs/usage.md"
printf 'example --mode current\n' > "$project/command.txt"
lds plumbing docs collect "$project" \
  --reason 'Review instructions after a mode change' \
  --audience 'CLI users' --source command.txt --json > "$review/collected.json"
source_hash=$(jq -r '.evidence[] | select(.path == "command.txt") | .sha256' \
  "$review/collected.json")
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
  --observed "$review/collected.json" \
  --output "$review/observed-assessment.md" --json
```

Inspect the collected JSON before writing the brief. Both packet and report
contain full source bytes: keep them private, and review the report's baseline
and uncertain proposal before making any separately authorized edits. This
fixture does not establish a real product's correct command.

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
parent and an unoccupied filename. For the existing authored-brief route,
brief and output ancestors cannot be symlinks. LDS exclusively creates one
Markdown file with mode 0600 (subject to umask); it never overwrites an old
report or creates parent directories. Reports and JSON stdout include source
content: store or share it only with the intended audience. Offline plumbing
sends nothing to a provider. The interactive route sends content
only after consent.

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
appropriate link/command checks. The offline route has no apply, confirmation,
retry or automatic update mode; adoption's plan/digest cannot apply its advice.
The interactive route's `send` confirms disclosure, **not** an edit or semantic
approval.
