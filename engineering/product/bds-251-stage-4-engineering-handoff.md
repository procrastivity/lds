# BDS-251 Stage 4 engineering handoff

**Status:** S10 is owner-accepted under reduced scope; S11 offline regressions
passed locally and in independent High review. Independent live CLI
replay and native installed-skill adoption/parity were not performed. Neither
is required by the revised acceptance, and neither is classified as PASS.

## Implemented forms and their boundaries

The interactive `lds adopt <root> --facts <facts.json>` form uses the live
shared flow, owner-supplied facts, deterministic documentation plumbing, and an
advisory OpenCode Go / `deepseek-v4-flash` request. The model has no tools and
cannot authorize writes. The product presents the complete deterministic plan
and requires a fresh exact-digest human decision before create-only application.
Noninteractive use refuses before facts or provider access.

Generated Claude Code `/lds` guidance is implemented, not a second workflow.
It fetches `lds plumbing adopt --json` at use time and delegates sequencing to
the returned `assets/flows/adopt.md` document. That reader still reports
`preparatory` / `unavailable-for-execution`: it only returns the document, not
an executed adoption. The skill's instructions are guidance, not behavioral
enforcement by LDS. Installation, discovery, and a current doctor result do
not establish native workflow execution or parity.

The [flow/plumbing map](adoption-flow-plumbing-map.md),
[operations handoff](adoption-operations-runtime-handoff.md), and
[S10 preparation/acceptance record](bds-251-stage-4-step10-preparation.md)
describe the deterministic boundaries and retained evidence. The procedures
in the historical S10 preparation record do not authorize another live wave.

## Accepted executor evidence and preserved incidents

The original executor wave attempted ten CLI executions, including root and
skill-fixture noninteractive controls. Nine have verified PASS artifacts. The
decline-labeled attempt actually applied after the owner typed the exact apply
decision; its actual result, checkpoint, consumed allocation, and missing PASS
remain unchanged. It is not a product approval bypass or a successful decline.

A separately authorized, isolated supplemental decline refused with exit 3,
`apply=null`, and unchanged full path/type/mode/byte target snapshots. The
owner-authored checkpoint and separate verification passed; independent High
offline review corroborated that evidence and closed the missing executor
decline scenario. It did not relabel the original incident or conduct another
live replay. Original and supplemental archives and ledgers remain immutable.

Combined executor use was seven logical application requests, zero native
adoption sessions, and no automatic retries or consumed-attempt relaunches.
The supplemental request was a separate explicit owner exception, not a reset
of the original budget. Wire HTTP requests are unmeasured. The skill-fixture
noninteractive CLI control is not a native Claude session. The supplemental
authorization and native prewrite-binding correction also passed independent
offline review without granting native launch authority.

Retained qualifications:

- Stale-plan refusal has a non-null refused apply, no adoption creates, and a
  preserved injected collision. The literal `refusal.plan-stale` stderr code
  was not captured; that classification is derived from the retained observed
  result and pinned product source. The verifier checks result and snapshots,
  not a captured stderr code.
- Three supplemental wrapper-directory modes changed from 0755 to 0700 with
  unproven cause. High found no demonstrated target-fixture drift; this is not
  a claim that the entire prepared root was unchanged.
- Personal review and typed decisions are supported by owner-authored
  checkpoints, not automated proof or retained terminal capture.
- The mock-only Ctrl-Z/fg experiment was unresolved and is excluded from the
  actual stale-case PASS. No job-control workflow is validated by it.
- The original failed Stage 7 `exactCommand` attempt remains failed and
  unexplained. Later bounded access evidence does not resolve it.

## S11 offline regressions

Validation ran on Linux amd64 against product source at the
[accepted baseline](https://github.com/procrastivity/lds/commit/20a0e23d65581e9015c6adc0bf048cb867de384c),
with documentation-only S11 changes and the accepted S10 preparation files.
The Nix development shell provided Go 1.24.10 and golangci-lint 2.1.6. No
product code, embedded flow, templates, or frozen evidence changed in S11.

Provider keys were unset for the regression commands; the standalone CLI
checks used a minimal environment with no provider credentials and isolated
temporary HOME, XDG, skill, and target directories. No live endpoint or native
inference was launched, and no owner terminal or checkpoint was used.

| Check | Observed result |
| --- | --- |
| `nix develop --command make check` | ShellCheck passed; golangci-lint reported `0 issues.`; `go test ./...` passed across all packages. |
| `nix develop --command go test -race ./...` | All packages passed; no race report. |
| `python3 contrib/bds251-s10.py check-fixtures` | Canonicalizer and three independent numeric/UTF-16 vectors passed; pinned source, overlay, facts, and four-output hashes matched. |
| `python3 contrib/bds251-s10.py self-test` in a clean disposable baseline clone | `selfTest=pass`, `mockOnly=true`, `providerRequests=0`, `nativeSessions=0`; includes supplemental one-slot/no-retry and native binding/no-write refusal regressions. |
| Built CLI manifest | `toolsmith/v1`; 19 leaves; 13 canonical plumbing-kind projection entries; byte-identical repeated JSON and independently recomputed manifest digest matched. Root `adopt` remains kind `llm`, outside the plumbing projection. |
| Isolated `install claude-code --json`, repeated | `installed`, then `current`; installed path/type/mode/bytes unchanged on the current result. |
| Isolated `plumbing doctor --json`, before and after live flow override | Zero findings; installed Claude Code target `current` both times. |
| Isolated post-install XDG flow override | Reader returned exact override bytes/hash/source without reinstall; generated skill and stamp tree remained path/type/mode/byte-identical. |
| Isolated noninteractive `adopt … --facts <absent-file> --json` | Exit 3, `refusal.interactive-required`, empty stdout, target unchanged. Refused before reading facts or using a provider. |
| Isolated `uninstall claude-code --json` | Removed the stamped skill; unrelated target remained unchanged. |

The S10 self-test initially refused the documentation-modified root checkout
with `tracked repository files differ from the pinned baseline`. This is the
intended provenance guard, not a product regression. It passed after copying
the unchanged helper to a clean disposable clone of the exact baseline; the
guard and retained live-evidence pins were not weakened or repinned.

The unstamped S11 CLI build matched the accepted binary SHA-256
`cecaa451e28e9abb39a5d283e90adff4f5e231b0120a31a5d5e2c4e8d1240e8a`.
Its manifest digest was
`sha256:c922372e6be06c00d1896db855fb06978f0c272dad49035265b21fb3329fedc8`;
version-stamped builds may have a different manifest digest.

Existing tests cover manifest contracts, projection/install/doctor lifecycle,
live override behavior, adoption preservation/refusals, provider-fault mocks,
fresh approval input, and reconciliation. They were not converted into live
tests or used as evidence of native parity. No new permanent test was needed
for these documentation-only changes.

Independent [High review](https://ampcode.com/threads/T-01a0f90c-5238-7798-9e27-aa13652422cc)
returned PASS with no corrections. It verified the exact documentation overlay,
repeated `make check` and `go test -race -count=1 ./...`, ran fixture checks and
the full mock-only self-test, and rebuilt the CLI. Its isolated CLI assertions
produced a byte-identical summary with the binary and manifest digests above;
all 34 local documentation links, whitespace, and helper syntax checks passed.
The original and supplemental evidence archive hashes remained unchanged.

## Acceptance does not authorize deployment or a pilot

The owner removed further independent CLI and native test waves from required
acceptance. No more owner-driven testing, JSON checkpoints, provider probes,
native sessions, retries, or subscription fallback are required or authorized
by S11. Native retry/reissue accounting remains unresolved, not tested away.

This handoff does not establish a real-project adoption pilot, first-release
inclusion, Anthropic support, general provider reliability, transactional
filesystem safety, or a human-facing reconciliation facilitator. No source
reference repository or shared infrastructure was modified. Commit, push,
release, and deployment are separate delivery actions, not outcomes implied
by passing local regressions or owner acceptance.
