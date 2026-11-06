# Check documentation in CI without a provider

Use `lds review <explicit-root> --json` for a bounded, read-only Linux CI
check. It needs no credential, provider, terminal, approval or apply step.
It does not execute the inspected project's commands. This recipe builds
LDS from the checked-out source, not a released installer.

## Keep the result and the actual exit status

This repository's [CI workflow](../../.github/workflows/ci.yml) runs
[the review wrapper](../../contrib/review-documentation-ci) immediately after
its existing `nix build` step. Pushes and pull requests to `main` therefore
review the LDS checkout with that checkout's binary, without a second build.
The wrapper requires Bash, Python 3 and `realpath`, available on the Ubuntu
runner. It does not require a provider or upload artifacts.

The following is a standalone equivalent for an LDS checkout. In a different
integration, deliberately choose the inspected root and the source checkout
used to build LDS rather than silently inspecting the current directory;
copying this snippet alone does not supply LDS's wrapper or source.

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
        run: bash contrib/review-documentation-ci
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

For a local Linux rehearsal from the LDS checkout:

```sh
nix build .#default
umask 077
scratch=$(mktemp -d)
GITHUB_WORKSPACE="$PWD" RUNNER_TEMP="$scratch" bash contrib/review-documentation-ci
```

The scratch directory must be outside the checkout. No GitHub event,
terminal interaction or provider is needed. Keep the wrapper's exit status
and inspect both captures, including skips; existing findings in the chosen
checkout are evidence to review, not a reason to weaken the step. The wrapper
never modifies the inspected documentation; the build produces its normal
`result` link separately.
