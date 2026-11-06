# Check documentation in CI without a provider

Use `lds review <explicit-root> --json` for a bounded, read-only Linux CI
check. It needs no credential, provider, terminal, approval or apply step.
It does not execute the inspected project's commands. This recipe builds
LDS from the checked-out source, not a released installer.

## Keep the result and the actual exit status

The following is a **proposed workflow**, not an installed workflow. It uses
the checkout and Nix action pins already in this repository's
`.github/workflows/ci.yml`. Review it before separately deciding to enable CI;
this guide does not add an active workflow or trigger any shared state.
The explicit inspected root here is the LDS checkout itself. In a different
integration, deliberately choose the root and the source checkout used to
build LDS rather than silently inspecting the current directory.

```yaml
name: Documentation review (example)
on: [pull_request]
permissions:
  contents: read
jobs:
  documentation:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
      - uses: DeterminateSystems/nix-installer-action@e50d5f73bfe71c2dd0aa4218de8f4afa59f8f81d # v16
      - name: Build the checked-out LDS source
        run: nix build .#default
      - name: Review the explicit documentation root
        shell: bash
        run: |
          set -eu
          umask 077
          root=$(realpath "${GITHUB_WORKSPACE:?}")
          scratch=$(realpath "${RUNNER_TEMP:?}")
          case "$scratch/" in
            "$root/"*) printf '%s\n' 'RUNNER_TEMP must be outside the inspected root' >&2; exit 4 ;;
          esac
          capture=$(mktemp -d "$scratch/lds-review.XXXXXX")
          status=0
          ./result/bin/lds review "$root" --json \
            > "$capture/review.json" 2> "$capture/review.stderr" || status=$?
          render_status=0
          python3 - "$capture" <<'PY' || render_status=$?
          import json
          import pathlib
          import sys

          capture = pathlib.Path(sys.argv[1])
          # Prefix and JSON-escape untrusted text, including line breaks and
          # control characters, so it cannot become a workflow log command.
          def show(label, value):
              print(label + " " + json.dumps(value, ensure_ascii=True))

          errors = (capture / "review.stderr").read_text(errors="replace")
          for line in errors.splitlines():
              show("LDS stderr:", line)
          try:
              result = json.loads((capture / "review.json").read_text())
              if (result["schemaVersion"] != 1 or result["operation"] != "review"
                      or result["status"] not in ("findings", "no-findings")
                      or not isinstance(result["targetRoot"], str)):
                  raise ValueError("not a schema-1 review result")
              files = result["inventory"]["files"]
              findings = result["validation"]["findings"]
              skips = result["validation"]["skipped"]
              limits = result["limitations"]
              for entries in (files, findings, skips):
                  if not isinstance(entries, list) or not all(isinstance(e, dict) for e in entries):
                      raise ValueError("expected arrays of result objects")
              if not isinstance(limits, list) or not limits or not all(isinstance(e, str) for e in limits):
                  raise ValueError("expected nonempty limitations")
              show("Review status:", result["status"])
              show("Inspected root:", result["targetRoot"])
              print(f"Scope: {len(files)} Markdown file(s), {len(findings)} finding(s), {len(skips)} skip(s)")
              for label, entries in (("FILE:", files), ("FINDING:", findings), ("SKIP:", skips)):
                  for entry in entries:
                      show(label, entry)
              if not files:
                  print("Empty scope is not evidence of documentation correctness.")
              if not findings:
                  print("No supported findings; unchecked areas are not passes.")
              for limit in limits:
                  show("LIMIT:", limit)
          except (OSError, UnicodeError, ValueError, KeyError, TypeError) as error:
              show("Cannot render review JSON:", str(error))
              sys.exit(1)
          PY
          printf 'LDS exit status: %s; renderer exit status: %s\n' "$status" "$render_status"
          printf 'Runner-local captures: %s\n' "$capture"
          if [ "$render_status" -ne 0 ] && [ "$status" -eq 0 ]; then
            printf '%s\n' 'LDS returned 0 without usable JSON; failing the wrapper with 4' >&2
            exit 4
          fi
          exit "$status"
```

The shell captures stdout and stderr separately **outside** the inspected
root, without a pipeline or `|| true` that loses the command's status. Both
findings and skips remain readable in the job log, with the reported
limitations and observed file count. Validation's `skipped` array includes
inventory skips; it is not necessary to print those a second time.
The small Python standard-library renderer checks the expected result shape;
it is not a replacement for LDS or a complete JSON-schema validator.

An early operational error can leave stdout empty or malformed. The renderer
reports that problem and the captured stderr, but the wrapper still returns
LDS's original **nonzero** status, rather than masking it as a JSON parse
failure. Only an unusable result paired with LDS exit 0 becomes wrapper exit
4; the log records both statuses. Shell setup/build failures are separate
from an LDS review result.

| LDS exit | Meaning for this check |
| --- | --- |
| 0 | No supported link findings in the observed scope; not a correctness verdict. |
| 1 | Validation finding or invalid input, such as a missing root; inspect JSON and stderr. |
| 2 | Usage error. |
| 3 | Safety refusal. |
| 4 | Unexpected operational/I/O failure. |

Do not reduce every nonzero status to “broken documentation,” or turn every
skip into success. Missing optional tracks and even an empty inventory can
return 0. Supported local file existence and bounded heading-fragment checks
are not a full Markdown implementation: external URLs, unsupported syntax
and explicit fragment skips remain unchecked. Neither exit 0 nor CI approval
establishes completeness, prose accuracy, code alignment or runtime behavior.
See [review scope](review-documentation.md) and
[safety and scope](../explanation/safety-and-scope.md).

## Retain JSON deliberately, and try the step locally

`review.json` and `review.stderr` remain in the printed runner-local directory
during the job. This workflow has **no artifact-upload step**: it does not
promise a downloadable artifact or persistence after runner cleanup. If you
need longer retention, separately choose a reviewed storage/upload policy
and access controls, then save those exact captures. Do not move captures
inside the inspected root. Paths, hashes and diagnostics can be sensitive;
assess what may be retained or shared.

For a local Linux rehearsal from the LDS checkout, build with
`nix build .#default`, put the exact contents of the YAML review step's `run`
block in a temporary shell script **outside** the checkout, and run it with
Bash. Set `GITHUB_WORKSPACE` to the absolute checkout root and `RUNNER_TEMP`
to an existing private temporary directory outside it. No GitHub event,
terminal interaction, provider or active workflow is needed. Keep the script's
exit status and inspect both captures, including skips; existing findings in
the chosen checkout are evidence to review, not a reason to weaken the step.
