# Review scoped documentation offline

`lds review <root> [--json]` is a Linux-only, read-only first pass over a
project's documentation. It needs no terminal, credential or authored JSON
inputs. It prints to stdout and creates no report artifact. Use a current
`lds` binary on `PATH` (see the [user guide](../README.md)).

## Try disposable missing file and heading links

Run this in a trusted shell; the fixture is not a real project or an LDS
pilot. The shell creates and later repairs the example files, **not** LDS.

```sh
project=$(mktemp -d)
mkdir -p "$project/docs" "$project/engineering"
printf '# Demo\n\nSee [guide](docs/guide.md).\n' > "$project/README.md"
printf '# Guide\n\nSee [missing](missing.md), [section](../README.md#details), and [outside](https://example.com).\n' > "$project/docs/guide.md"
printf '# Design\n\nThis is only a fixture.\n' > "$project/engineering/design.md"
if lds review "$project" --json; then
  printf '%s\n' 'Expected a missing-local-target exit (1)' >&2
  exit 1
else
  result=$?
  [ "$result" -eq 1 ] || exit "$result"
fi
```

Inspect stdout: schema-1 JSON has `operation: "review"`, the absolute
`targetRoot`, `status: "findings"`, `inventory`, `validation`, and a nonempty
`limitations` string array. Inventory lists the three regular Markdown files
with sizes and hashes. Validation reports the missing `docs/missing.md` file
(`validation.local-target-missing`) and missing `#details` heading in root
README (`validation.local-anchor-missing`). It skips the external URL; its
`skipped` array also includes inventory skips, if any. Exit **1** means a
supported local file or heading target is missing, not that every link was
fully checked. Human output
(omit `--json`) lists the files, findings, skips and limitations instead.

## Repair both targets, then interpret exit 0 narrowly

```sh
printf '# Missing\n\nThis fixture could still describe the wrong behavior.\n' > "$project/docs/missing.md"
printf '\n## Details\n\nOnly a fixture heading.\n' >> "$project/README.md"
lds review "$project" --json
```

The second run should exit **0** with `status: "no-findings"` and an empty
findings array. The supported `#details` anchor now exists; the external URL
remains unchecked. This does not establish whether any sentence is correct
or current. Empty or absent
optional documentation tracks are not findings by themselves; empty coverage
and skipped checks are not passes. Delete the disposable directory when done
only after checking its value (`rm -rf "$project"`).

## Understand the bounded heading subset

Local links such as `guide.md#install` and same-document `#install` are
checked only against inventoried UTF-8 Markdown of at most 1 MiB. Supported
headings use one to six leading `#` characters (ATX syntax), up to three
leading spaces, ASCII text and optional closing hashes. Plain text and
simple single-backtick code are supported: ``## API `read_file`!`` produces
`#api-read_file`. ASCII letters become lowercase, spaces become hyphens and
punctuation is removed. Repeated headings get collision-aware `-1`, `-2`, …
suffixes in document order. Links are case-sensitive; percent encoding is
decoded without treating `+` as a space. Fenced code contributes no headings.

Unicode or formatted headings, entities, complex code spans, Setext,
container/indented headings and raw HTML can make the **whole target's**
heading index uncertain, including duplicate numbering. Those fragments
receive `validation.fragment-skipped`, not a missing-anchor finding or a
pass. Source-line references such as `#L8-L14` are not heading anchors and
remain unchecked. Empty/malformed/control-bearing fragments, oversized/non-UTF-8 or
changed target content, and targets outside the inventoried Markdown scope
are also explicit skips. File-existence checks remain independent. This is
a conservative GitHub-style subset, not every Markdown dialect or custom
anchor convention. Create-only adoption plans retain file-only diagnostics;
run `lds review` separately for heading checks.

For a real root, review is only a bounded observation: root `README.md` and
regular Markdown below `docs/` and `engineering/`, without following symlinks.
It does not read arbitrary source code, check layout or completeness, compare
the two tracks, run the project, edit files or contact a provider. There is no
`--output` flag: if you redirect stdout to a file, that write is your shell's
responsibility, not an LDS report. Use trusted, quiescent inputs; the reads are
not an atomic snapshot. See [safety and scope](../explanation/safety-and-scope.md)
and the [command reference](../reference/commands.md) for further limits and
exit codes. For evidence-backed proposed changes rather than link inspection,
use the separate [assessment how-to](assess-documentation.md).
For automation see the [CI recipe](check-documentation-in-ci.md); for one
connected offline sequence see the
[maintenance tutorial](../tutorials/maintain-documentation.md).
