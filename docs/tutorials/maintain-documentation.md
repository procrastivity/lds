# Maintain one disposable documentation layer offline

This connected Linux walkthrough uses **one disposable Wordcount project**
from initialization to review, assessment, a separately approved update and
final review. It needs a current checkout-built `lds` on `PATH`, Python 3,
POSIX `sh` and `wc`, but no provider, credential or agent harness. Start in the
LDS checkout and keep the same trusted shell and variables throughout.
The two decision points must run directly in a Linux terminal.

The supplied facts retain their **test-only simulated approval**. Neither
those facts, the authored advice below nor this tutorial authorizes a real
project. You decide whether to make each disposable write after reviewing
its complete plan. No approval is carried from one command to another.

## 1. Prepare and initialize the fixture

```sh
umask 077
project=$(mktemp -d)
review=$(mktemp -d)
cp internal/asset/testdata/adoption/greenfield/target/word-count.sh "$project/"
cp internal/asset/testdata/adoption/greenfield/target/sample.txt "$project/"
cp internal/asset/testdata/adoption/greenfield/facts.json "$review/facts.json"
(cd "$project" && sh word-count.sh count sample.txt)
```

Expect exactly `Word count: 4` followed by LF and exit 0. This executes the
disposable fixture command yourself; LDS does not run it. The facts and all
later evidence/report/request artifacts stay in `review`, outside `project`.

In your terminal, run:

```sh
lds init "$project" --facts "$review/facts.json" --json
```

Read the **entire** plan on stderr: root identity, facts and template
provenance, exact output contents, existing/preserved paths, parents,
findings, skips, veto and digest. With stock checkout templates this proposes
only `engineering/README.md`, `engineering/product/overview.md`,
`docs/README.md` and `docs/reference/usage.md`; root `README.md` remains
report-only. Facts approval is not plan approval.

After the complete display, freshly type `apply sha256:<exact displayed digest>`
only if you choose to approve this disposable plan. Substitute the displayed
digest yourself. Or type `decline` and stop the walkthrough without writes.
Never pipe, prequeue or script a digest confirmation. Inspect final stdout,
stderr, exit code and actual files; continue only after `status: "applied"`
and exit 0. See the [init how-to](../how-to/initialize-a-project.md) for the
full contract and partial-write boundaries.

## 2. Review, then deliberately introduce a missing file link

The exact expectations below use the stock adoption templates. Check their
bytes before proceeding; if a user/installed override was selected, stop
and inspect it rather than silently accepting a different fixture.

```sh
for path in engineering/README.md engineering/product/overview.md docs/README.md docs/reference/usage.md; do
  cmp "internal/asset/testdata/adoption/greenfield/expected/$path" "$project/$path" || exit 1
done
lds review "$project" --json > "$review/initial-review.json"
python3 -m json.tool "$review/initial-review.json"
```

Expect exit 0, `status: "no-findings"`, four inventoried Markdown files,
empty findings and no skips. This is a link observation, not verification of
the facts or prose. Now your shell, **not LDS**, deliberately appends two
fixture links to the user navigation page:

```sh
printf '\n[Extra notes](missing-notes.md)\n\n[External reference](https://example.com).\n' >> "$project/docs/README.md"
if lds review "$project" --json > "$review/broken-review.json"; then
  printf '%s\n' 'Expected the deliberate missing-file finding' >&2
  exit 1
else
  status=$?
  [ "$status" -eq 1 ] || exit "$status"
fi
python3 -m json.tool "$review/broken-review.json"
```

Expect one finding: `validation.local-target-missing` at `docs/README.md`
line 5, target `missing-notes.md` (resolved to `docs/missing-notes.md`).
Expect one skip: `inventory.external-skipped` at the same page's line 7.
The external URL is never fetched. The inventory still contains four files.
This intentionally uses a missing **file**, not any new heading-fragment
behavior. No missing-notes page should be created in this tutorial.

## 3. Collect once, author advice, and inspect an external report

```sh
lds plumbing docs collect "$project" \
  --reason 'Remove a deliberately broken fixture navigation link' \
  --audience 'Wordcount fixture readers' \
  --source word-count.sh --source sample.txt --json > "$review/collected.json"
python3 -m json.tool "$review/collected.json"
```

Expect exit 0, `status: "review-required"`, `reportPath: ""`,
`validationExitCode: 1`, the same finding and skip, and six evidence entries:
the four documents plus the two explicitly selected source files, with full
contents and hashes. Collection succeeding does not mean validation passed.
Its `brief.proposals` and `brief.questions` are empty. Keep these private full
bytes and do not modify the target from now until the reviewed update.

After reading that packet, write this **authored example advice**. Python
copies only the declared collection scope/hashes and safely encodes the
human-authored text; it does not infer a proposal or approve one.

```sh
python3 - "$review" <<'PY'
import json
import pathlib
import sys

review = pathlib.Path(sys.argv[1])
observed = json.loads((review / "collected.json").read_text())
brief = {
    "schemaVersion": 1,
    "reason": observed["brief"]["reason"],
    "audience": observed["brief"]["audience"],
    "evidence": observed["brief"]["evidence"],
    "proposals": [{
        "path": "docs/README.md",
        "summary": "Remove the invented notes link and describe the existing usage page",
        "evidence": ["docs/README.md", "docs/reference/usage.md", "word-count.sh", "sample.txt"],
        "proposedText": "The usage page documents the fixture's word-count command."
    }],
    "questions": ["Is the external reference useful? It remains unchecked."]
}
with (review / "brief.json").open("x", encoding="utf-8") as output:
    json.dump(brief, output, indent=2)
    output.write("\n")
PY
lds plumbing docs assess "$project" --brief "$review/brief.json" \
  --observed "$review/collected.json" \
  --output "$review/assessment.md" --json > "$review/assessment.json"
cat "$review/assessment.md"
```

Expect exit 0 and `status: "review-required"`, not applied or approved.
The external Markdown report separates observations, supplied proposal and
question, and includes full evidence JSON. The broken link and skip remain;
the target is unchanged. `proposedText` is just a suggested **excerpt**, not
the replacement for an entire file. Decide whether its advice is useful;
report creation and citations do not verify it or grant write authority.
See the [assessment how-to](../how-to/assess-documentation.md).

## 4. Separately author a complete replacement request

Suppose, for this disposable exercise, you choose to remove the deliberately
invented notes link, preserve the existing usage link and retain the external
reference as an explicitly unchecked link. Independently author the complete
desired file below. This is **not** a conversion of the report or its excerpt
to an edit, and nothing is automatically applied.

```sh
python3 - "$review" <<'PY'
import json
import pathlib
import sys

review = pathlib.Path(sys.argv[1])
after = (
    "# User documentation\n\n"
    "[Usage](reference/usage.md)\n\n"
    "The usage page documents the fixture's word-count command.\n\n"
    "[External reference](https://example.com).\n"
)
request = {
    "schemaVersion": 1,
    "edits": [{
        "path": "docs/README.md",
        "after": after,
        "rationale": "Remove the deliberately broken notes link; keep the existing usage and unchecked external links",
        "citations": ["docs/README.md", "docs/reference/usage.md", "word-count.sh", "sample.txt"]
    }]
}
with (review / "request.json").open("x", encoding="utf-8") as output:
    json.dump(request, output, indent=2)
    output.write("\n")
PY
python3 -m json.tool "$review/request.json"
```

Here `after` is the **whole UTF-8 file**, including headings, navigation,
blank lines and final LF: exactly **154 bytes** in this example. An excerpt
in that field would replace the entire document and lose other content.
Citations reference collected evidence, not an authenticated approver.
Review the replacement on its own merits.

## 5. Make a fresh terminal decision for the exact update plan

Use the **original collection packet**, never `assessment.md` or
`assessment.json`, as `--observed`. The report has authored advice and a report
path; it is not the unchanged collection baseline.

```sh
lds update "$project" \
  --observed "$review/collected.json" --request "$review/request.json" --json
```

Read the entire deterministic plan on stderr: target and baseline, exact
before/after bytes, citations, selected metadata/ancestors, findings, skips,
veto and `planDigest`. The plan's findings and skips retain the **original**
missing-file observation and external-URL skip; they are not a fresh final
review. Inspect `after` to confirm that it removes the broken link and keeps
the unchecked external reference. The init decision and acceptance of advice
authorize **none** of this update. Only after reviewing this display, freshly
type `apply sha256:<exact displayed digest>` if you personally choose to
approve this disposable replacement; otherwise type `decline` and stop.
There is no scripted digest extraction, stdin approval or `--yes` bypass.

Inspect exit status, both streams, final JSON `plan`/`apply` and actual files.
Continue only after exit 0 and `status: "applied"`. A refusal, partial or
indeterminate outcome requires inspection and a new collection/review, not
an automatic retry. Only the existing `docs/README.md` is requested for
replacement; sources and other documents must remain unchanged. The
[update how-to](../how-to/update-documentation.md) describes metadata limits,
stale-baseline refusals and nontransactional per-file replacement.

## 6. Verify the final bytes and interpret review narrowly

```sh
python3 - "$project" "$review" <<'PY'
import json
import pathlib
import sys

project, review = map(pathlib.Path, sys.argv[1:])
request = json.loads((review / "request.json").read_text())
expected = request["edits"][0]["after"].encode("utf-8")
assert (project / "docs/README.md").read_bytes() == expected
observed = json.loads((review / "collected.json").read_text())
for item in observed["evidence"]:
    if item["path"] != "docs/README.md":
        assert (project / item["path"]).read_bytes() == item["content"].encode("utf-8")
assert not (project / "docs/missing-notes.md").exists()
print(f"Replacement matches all {len(expected)} requested bytes; other collected files unchanged.")
PY
lds review "$project" --json > "$review/final-review.json"
python3 -m json.tool "$review/final-review.json"
```

Expect exit 0, `status: "no-findings"`, the same four Markdown paths and an
empty findings array. The external URL at `docs/README.md` line 7 is **still
one skip**, not a successful URL check. The updated file has exactly the
bytes specified by `after`; no notes page was created. Do not reuse the old
collection after this change.

This demonstrates the bounded offline sequence, not arbitrary Markdown
completeness, semantic correctness, a real-project pilot, provider behavior
or native-skill parity. Supported scoped Markdown heading-fragment checks do
not validate every Markdown dialect: complex/Unicode, Setext, HTML and
container heading syntax can remain explicit fragment skips. Supported local
file checks still operate independently. Neither an empty scope nor skips
are correctness evidence. For automation without writes or terminal
decisions, see [the separate CI recipe](../how-to/check-documentation-in-ci.md).

Inspect the paths before deleting the disposable directories. When finished,
remove only these temporary fixtures and external artifacts:

```sh
printf 'Disposable project: %s\nPrivate review artifacts: %s\n' "$project" "$review"
# After checking both values are the temporary directories created above:
rm -rf -- "$project" "$review"
```
