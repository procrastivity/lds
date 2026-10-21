# Review scoped documentation offline

`lds review <root> [--json]` is a Linux-only, read-only first pass over a
project's documentation. It needs no terminal, credential or authored JSON
inputs. It prints to stdout and creates no report artifact. Use a current
`lds` binary on `PATH` (see the [user guide](../README.md)).

## Try a disposable broken link

Run this in a trusted shell; the fixture is not a real project or an LDS
pilot. The shell creates and later repairs the example files, **not** LDS.

```sh
project=$(mktemp -d)
mkdir -p "$project/docs" "$project/engineering"
printf '# Demo\n\nSee [guide](docs/guide.md).\n' > "$project/README.md"
printf '# Guide\n\nSee [missing](missing.md), [section](../README.md#unknown), and [outside](https://example.com).\n' > "$project/docs/guide.md"
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
with sizes and hashes. Validation reports the missing `docs/missing.md` target
and skips the unchecked fragment and external URL; its `skipped` array also
includes inventory skips, if any. Exit **1** means a supported local file
target is missing, not that the other links were fully checked. Human output
(omit `--json`) lists the files, findings, skips and limitations instead.

## Repair the file, then interpret exit 0 narrowly

```sh
printf '# Missing\n\nThis fixture could still describe the wrong behavior.\n' > "$project/docs/missing.md"
lds review "$project" --json
```

The second run should exit **0** with `status: "no-findings"` and an empty
findings array. It does not establish whether the fragment exists, the
external URL works, or any sentence is correct or current. Empty or absent
optional documentation tracks are not findings by themselves; empty coverage
and skipped checks are not passes. Delete the disposable directory when done
only after checking its value (`rm -rf "$project"`).

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
