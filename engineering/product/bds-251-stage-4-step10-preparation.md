# BDS-251 Stage 4 Step 10 preparation

**Status:** accepted under owner-reduced scope: completed executor CLI cases
and independent offline evidence review. No further live test wave is required.
Independent live replay and native installed-skill parity are not established.

## Owner-accepted scope supersedes the live-wave instructions below

The owner accepted the completed work and removed the requirement for further
back-and-forth testing in the
[root thread](https://ampcode.com/threads/T-01a0e9cf-8adb-7786-92d4-166fd8540118).
The remaining procedures below describe the historical preparation and test
plan; they are not authorization or a remaining requirement to launch another
CLI wave, native session, probe, retry, or subscription fallback.

Accepted evidence is the original executor CLI results plus the separately
authorized supplemental decline and independent High offline review. The
original wave has nine verified executions and one mistaken applied decline;
that incident remains unchanged and unpassed. The supplemental decline is a
separate verified refusal (exit 3, no apply, unchanged full target snapshot),
closing the executor's missing decline scenario. Combined use is seven logical
application requests, zero native sessions; wire HTTP requests are unmeasured.
The supplemental authorization/native-binding correction also passed focused
independent offline review. No new owner checkpoint or terminal action is needed.

Original evidence archive SHA-256:
`b0ba35d6c43ec08c24be36bfff97bfc303e60fc18aa42479fe809166e33aa389`.
Supplemental actual-evidence archive SHA-256:
`e0102327087a6fe89ab9c73987450fbf8b1d2123ca14be2fc9162bb8be3b79be`.
Both frozen archives remain unchanged; this acceptance update does not repin
their historical guide, helper, checkpoints, or ledgers.

Limits remain explicit: independent live CLI replay and actual native `/lds`
skill parity were not performed. The skill-fixture noninteractive CLI control
is not a native session. The stale refusal code is source-derived rather than
captured stderr. Three supplemental wrapper-directory modes changed from 0755
to 0700 with unproven cause; no target-fixture drift was demonstrated. Owner
review and typing are supported by owner-authored attestations, not terminal
capture. This is reduced-scope acceptance, not a full parity PASS, and it does
not change frozen Stage 7 evidence or authorize a real-project pilot or release.

## Historical preparation baseline

Preparation was pinned to
`20a0e23d65581e9015c6adc0bf048cb867de384c` and uses only disposable copies of
the existing Wordcount fixtures. It does not change the adoption flow,
contracts, manifest/projections, or frozen Stage 7/reconciliation evidence.

## Scope and stop conditions

This document and [`contrib/bds251-s10.py`](../../contrib/bds251-s10.py)
prepare fixture targets, full byte snapshots, sanitized evidence, and the
separate executor-wave request ledger. The helper has no model API client. Its
`run-root` command launches actual `lds adopt` with the caller's real terminal
attached directly as child stdin and stderr. There is no `script`, PTY relay,
synthesized input, or captured terminal transcript. Noninteractive callers
are rejected before a child launch. The helper captures only child JSON stdout
in memory for strict parsing; it does not retain arbitrary typed lines or raw
streams. The complete displayed plan remains visible in the terminal, and a
separate owner-authored checkpoint records what the owner personally reviewed
and typed. On interruption, the child is stopped, the after-snapshot and
sanitized stopped/unknown outcome are retained, the reservation remains
consumed, and retry is disabled. No raw terminal file is created. The
preparation directory is mode 0700. The selected provider credential is
checked for nonempty presence and passed only to the child environment; its
value is never printed, copied to a file, or hashed.

Stop without live inference if `OPENCODE_API_KEY` is absent, if the worktree
or origin baseline differs, if any fixture hash/self-test fails, if the
human's actual terminal/chat cannot display and accept the review, or if
credential-bearing output/debug logging becomes possible. Never read a
credential store. No WIP command or `.wip/generated` content is part of this
procedure. No real project, reference repository, remote branch, Linear issue,
or GitHub state may be written.

## Offline preparation

Use the child thread's Terminal tab for later interactive work. The CLI gate
is human-operated through that terminal; the helper does not provide an
approval driver. Start with Boolean-only key presence:

```sh
if test -n "${OPENCODE_API_KEY:-}"; then
  printf 'OPENCODE_API_KEY available: true\n'
else
  printf 'OPENCODE_API_KEY available: false; stop and request the permitted root refresh\n'
fi
```

Choose a new, empty disposable directory. Build without changing the
repository and prepare the paired roots/snapshots:

```sh
S10=/tmp/lds-bds251-s10
python3 contrib/bds251-s10.py self-test
python3 contrib/bds251-s10.py prepare --out "$S10"
mkdir -p "$S10/bin"
nix develop --command go build -o "$S10/bin/lds" ./cmd/lds
```

Point `CLAUDE_BIN` at a separately installed and verified Claude Code
2.1.286 executable. Copy only that executable (not its HOME, settings,
credentials, or OAuth state) into the disposable prepared binary directory;
capture its version/help with isolated HOME and XDG before packaging:

```sh
: "${CLAUDE_BIN:?point to the separately verified Claude Code 2.1.286 executable}"
mkdir -p "$S10/bin" "$S10/home" "$S10/xdg/skill/fresh"
install -m 0755 "$CLAUDE_BIN" "$S10/bin/claude"
env -i PATH="$S10/bin:/usr/bin:/bin" HOME="$S10/home" \
  XDG_CONFIG_HOME="$S10/xdg/skill/fresh" TERM="${TERM:-xterm-256color}" \
  LANG="${LANG:-C.UTF-8}" "$S10/bin/claude" --version \
  > "$S10/evidence/claude-version.txt"
env -i PATH="$S10/bin:/usr/bin:/bin" HOME="$S10/home" \
  XDG_CONFIG_HOME="$S10/xdg/skill/fresh" TERM="${TERM:-xterm-256color}" \
  LANG="${LANG:-C.UTF-8}" "$S10/bin/claude" --help \
  > "$S10/evidence/claude-help.txt"
```

`env -i` prevents inherited `CLAUDE_CONFIG_DIR`, route/model overrides,
debug/logging flags, proxy settings, and authentication state from affecting
these checks. Record the executable hash as binary provenance; never inspect,
copy, or export any credential/configuration store.

`prepare` refuses a nonempty output directory. It creates distinct but
byte-identical `root-cli` and `skill` target roots for fresh/rerun, partial
engineering, different-output collision, partial legacy, decline, stale,
post-install override, and noninteractive cases. Every base includes the
independently hashed source files and mixed root README. Each initial snapshot
stores every relative path/type/mode and exact regular-file bytes (base64 plus
SHA-256); it does not follow symlinks. Expected output and overlay hashes are
independently pinned in the helper and checked against the existing fixture
files and acceptance projection. Facts remain the fixture's explicitly
test-only approval, not real-project facts.

The RFC 8785 checker rejects duplicate keys and unsafe numbers, sorts object
keys by UTF-16 code units, checks the repository's independent canonical
vector plus asymmetric number/Unicode vectors, and recomputes
`sha256:<hex>` over every plan field except the top-level `planDigest`. It is
an evidence checker only; its digest is never used as a human decision. The
offline self-test includes a scheduling adversary using a fake CLI and mock
transport only. It places a synthetic digest in the caller's terminal queue
while the mock response is pending; the direct-terminal harness connects the
fake CLI to that same terminal and does not relay input. This mock regression
does not establish actual `lds adopt` behavior or provider parity. Actual
product terminal input flushing is separately covered by the Linux Go PTY
tests. No real `lds` apply or provider request occurs in either test. Personal
review and typing are recorded separately as the owner's attestation, never
inferred by the helper.

Install the generated LDS skill into the isolated fixture home as an offline
asset operation (no Claude process or inference). Record the command result,
skill-tree snapshot, and `SKILL.md` hash. Every channel/case has a separate
XDG directory created empty. Only **after** installation, copy the prepared
override into the `override` case directories for both channels; every other
case directory remains empty. The override is the unchanged live flow plus a
disposable comment marker, so its only semantic purpose is to prove live
asset resolution. Immediately before each root launch and in later native
session evidence checks, the helper verifies the selected case-local XDG
contents, expected live-flow source/hash, and unchanged installed skill. The
override flow result must name `source: override` and match its case-local
hash.

Use the explicit skill destination so nothing is installed into the user's
real home:

```sh
mkdir -p "$S10/home" "$S10/xdg/skill"
HOME="$S10/home" XDG_CONFIG_HOME="$S10/xdg/skill/fresh" \
  LDS_CLAUDE_SKILLS_DIR="$S10/home/.claude/skills" \
  "$S10/bin/lds" install claude-code --json > "$S10/evidence/skill-install.json"
HOME="$S10/home" XDG_CONFIG_HOME="$S10/xdg/skill/fresh" \
  LDS_CLAUDE_SKILLS_DIR="$S10/home/.claude/skills" \
  "$S10/bin/lds" doctor --json > "$S10/evidence/skill-doctor.json"
sha256sum "$S10/home/.claude/skills/lds/SKILL.md" \
  "$S10/home/.claude/skills/lds/.lds-manifest-stamp.json"
mkdir -p "$S10/xdg/skill/override/lds/flows" "$S10/xdg/root-cli/override/lds/flows"
cp "$S10/evidence/override-seed/adopt.md" "$S10/xdg/skill/override/lds/flows/adopt.md"
cp "$S10/evidence/override-seed/adopt.md" "$S10/xdg/root-cli/override/lds/flows/adopt.md"
printf '%s\n' '{"mcpServers":{}}' > "$S10/evidence/empty-mcp.json"
```

Take a second install-tree/hash snapshot after adding the case-local XDG
overrides. The skill manifest stamp and generated `SKILL.md` must stay exactly
unchanged; every non-override case directory must remain empty; `lds doctor
--json` against the isolated installation should still report `current`.

Pin the freshly built runtime after preparation and before reserving any
provider call. This binds the helper bytes, binary, all expected before
snapshots, assets, and exact baseline; rerun preparation rather than replacing
a pin if any input changes:

```sh
python3 contrib/bds251-s10.py pin-runtime --prepared "$S10" --binary "$S10/bin/lds"
```

## CLI-first live matrix and human gates

Do not start a skill inference session until every root CLI case and its
owner-authored human checkpoint is complete. Cases are `fresh`, `rerun`,
`partial-engineering`, `decline`, `stale`, and `override`; `collision`,
`partial-legacy`, and `noninteractive` are no-request controls. The rerun is
allowed only after fresh has passed `verify-root`; its before snapshot is then
the successful fresh after snapshot. Each of the six eligible root cases
uses one application-level Chat call at most, not a wire-HTTP request cap.

Run eligible cases only from the child thread's real interactive Terminal
tab. The helper requires a TTY on both stdin and stderr and attaches the
product directly to that terminal; it creates no `script` PTY, input relay,
or terminal log. Reserve exactly one matching ledger row before launching
each eligible case. Example for fresh:

```sh
python3 contrib/bds251-s10.py ledger-reserve-root \
  --ledger "$S10/evidence/ledger.json" --case fresh
python3 contrib/bds251-s10.py run-root \
  --prepared "$S10" --binary "$S10/bin/lds" --case fresh
```

At the product's separate data-transfer prompt, personally read the request
and type the exact `send` decision. After the provider response, personally
read the entire raw plan displayed in the same terminal, then type the fresh
exact `apply <planDigest>` displayed by the product, or the product's
categorical decline for `decline`. Do not paste a prepared digest or use a
script/tool to approve. The helper captures only product JSON stdout in
memory for strict sanitization; it does not capture terminal stderr or typed
input. Its result records presentation as direct-terminal/unobserved by the
harness, not as owner consent.

After each command, the owner must manually author
`$S10/evidence/human-checkpoints/root-cli/<case>.json` (schema version 1)
with `ownerAuthored: true`, exact channel/case, the digest personally read
from the complete displayed plan, and `fullPlanPersonallyReviewed: true`.
Record the actual `providerConsent` (`send` or `not-requested`) and whether
it was personally typed, the actual decision (`apply`, `decline`, or
`veto-no-confirmation`), whether it was personally typed, and the exact typed
decision (`apply <digest>`, `decline`, or `not-applicable`). This is an
owner-authored attestation, not automated proof that the terminal was
watched. Never have the helper, model, or another person generate, relay, or
replay this checkpoint. Then run `verify-root`; for example:

```sh
python3 contrib/bds251-s10.py verify-root --prepared "$S10" --case fresh \
  --result evidence/live/root-cli/fresh/result.json \
  --before evidence/snapshots/run-before/root-cli-fresh.json \
  --after evidence/snapshots/after/root-cli-fresh.json
```

Run collision and partial-legacy before eligible calls. They must display the
complete veto plan, exit 3, and stop before any provider-transfer prompt;
there is no `send` or apply confirmation. Collision's read-only `docs
validate` diagnostic is independently expected to report exit 1 with
`validation.local-target-missing` at `docs/reference/usage.md:5` for
`../howto.md`. That diagnostic alone does not veto adoption; the distinct
plan finding `refusal.output-collision` does. Partial-legacy inventory and
validation both exit 0 while its plan reports `refusal.legacy-docs`. After
personally inspecting the whole plan, manually author its checkpoint with
`providerConsent: not-requested`, `decision: veto-no-confirmation`,
`decisionPersonallyTyped: false`, and `typedDecision: not-applicable`, then
verify with `verify-root`. Run root and skill noninteractive fixture controls
with `run-root --case noninteractive` (add `--channel skill` for the skill
fixture). The child gets stdin disconnected and must exit 3 with
`refusal.interactive-required`, zero stdout, no plan, zero provider calls,
and an unchanged complete snapshot. Verify each with
`verify-control --prepared "$S10" --channel <root-cli|skill>`. These controls
make no provider request and start no Claude session.

After offline preparation is complete, package the correction regressions and
isolated skill evidence for transfer and independent review. While C2 holds,
use `--offline-only`: it emits no current root controls and does not import
the historical collision/legacy controls, whose shared override did not prove
case-local selection. Do not launch even the zero-inference root controls
until root explicitly authorizes the CLI phase. Once authorized, collect the
three current root controls and omit `--offline-only` only when packaging
their verified results:

```sh
python3 contrib/bds251-s10.py pack-preflight --offline-only --prepared "$S10" --root-cli-binary "$S10/bin/lds" --claude-binary "$S10/bin/claude" --claude-version 2.1.286 --out engineering/product/bds-251-stage-4-step10-preflight-evidence.json
```

The packer refuses a changed baseline, altered snapshots, unverified full-plan
digests, unexpected classifications, a stale installed skill, or missing
restricted-tool flags. Its status check is case-specific: collision and
partial-legacy require `refused` with their full veto plans; noninteractive
requires `status: null`, no plan, and a passing `verify-control` result. In
offline-only mode it records zero current root controls and explicitly states
why historical controls were not imported. An existing output may only be
refreshed with `--replace-existing` when it is same-baseline zero-request
preparation evidence. The file contains fixture and sanitized evidence only;
no credentials, raw provider streams, terminal logs, Claude configuration,
or model responses.

After fresh passes `verify-root`, run and verify `rerun`; its before snapshot
is loaded from the fresh after snapshot. The rerun must be `current`, have
four `identical-untouched`, create nothing, and compare byte-for-byte
unchanged. Partial engineering preserves its pre-existing decision and
README/source bytes while adding only the four approved outputs and missing
parents. Decline consumes one eligible provider call, shows the entire plan,
and is personally declined at the apply prompt; it creates nothing.

For stale injection, reserve `stale`, start `run-root --case stale`, and wait
until its full plan and exact approval prompt are displayed. Personally
review the entire plan first. In a second Terminal, run the fixture
controller while the first command waits:

```sh
python3 contrib/bds251-s10.py stale-insert \
  --prepared "$S10" --channel root-cli \
  --target "$S10/targets/root-cli/stale" \
  --plan "$S10/evidence/live/root-cli/stale/prelaunch-plan.json"
```

The stale controller is authorized only inside a trusted, exclusively
controlled, quiescent disposable fixture domain. Before starting the actual
CLI or skill, stop every other target or ancestor writer, watcher, sync tool,
and process that can rename or change permissions on the target or any
ancestor. While the actual command waits for the owner's decision, the
explicit stale controller is the only permitted fixture writer. Do not
concurrently rename the target/root or ancestors, tamper with permissions,
or bypass effective permissions or the fixture boundary. The controller
revalidates the full pinned before-state, opens expected ancestors without
following symlinks, temporarily seals the target root and `docs` directory,
and exclusively creates only the pinned differing
`docs/reference/usage.md` fixture. A reference-directory rename adversary
regression checks the supplementary permission seal. These seals are not a
filesystem security guarantee or unconditional race containment; the
controller relies only on the root-approved trusted, exclusively controlled,
quiescent disposable-fixture boundary. Stop every target/ancestor writer
before launch; during the review wait, only the explicitly invoked controller
may write the fixture. No concurrent target or ancestor rename, permission
tampering, or privilege bypass is permitted. If exclusivity, effective
permissions, insertion success, or exact mode restoration cannot be
established, retain the failure and whole after snapshot and STOP WITHOUT
APPROVAL. Never type the old digest after a failed or uncertain controller
outcome. Only after a successful controller event may the owner return to the
waiting prompt and personally type the old displayed digest if they still
choose to approve. The expected result is `refusal.plan-stale`, exit 3, zero
adoption creates, and byte-exact preservation of the injected collision. The
skill stale controller uses its own channel-scoped event and after-snapshot
files and cannot overwrite root-cli evidence.

After each command, `run-root` emits a sanitized result/full-plan artifact
and an after snapshot. `verify-root` binds the case schema/status, apply
result, output bytes, human checkpoint, ledger reservation, and complete
snapshots. Semantic comparison is reserved for the later paired root/skill
review: it excludes only the exact top-level target-root, target-identity,
and plan-digest fields plus `existingParentDirectories[*].identity`, while
retaining arbitrary nested identity-like fields. Complete snapshots are
compared without normalization. Root and skill digests are independently
valid but differ across target roots; do not compare literal digests.

## Native skill preparation and human workflow

No native inference starts during preparation. Pinned Claude Code is
`2.1.286`, the installed skill is discovered from the isolated HOME, the
project XDG override remains live after installation, and the model remains
`deepseek-v4-flash` through the selected OpenCode Go Messages gateway. Map
`OPENCODE_API_KEY` only in the Claude child environment as
`ANTHROPIC_API_KEY`; do not inspect, copy, or export credential stores or
OAuth tokens. Native execution is blocked while N2 accounting/envelope
authorization is pending. A gateway failure must be retained with its
observed counts and failed outcome; it never switches routes automatically.
The Anthropic-subscription escape hatch is approved in principle only. Root
must first settle access, model/configuration, and remaining or revised
budget after seeing the gateway failure. This procedure does not authorize a
fallback or provider substitution on its own.

The feasible interactive tool boundary is:

```text
--tools Bash,Read,Skill
--disallowedTools Write,Edit
--permission-mode default
--strict-mcp-config --mcp-config <outside-target empty JSON file>
--setting-sources user
```

There is no native launch during this preparation. Only after root settles N2
and explicitly authorizes native execution, reserve and consume exactly one
case before starting a session:

```sh
python3 contrib/bds251-s10.py ledger-reserve-skill \
  --ledger "$S10/evidence/ledger.json" --case fresh
python3 contrib/bds251-s10.py ledger-consume-skill \
  --ledger "$S10/evidence/ledger.json" --case fresh
```

The pinned Claude Code executable is kept at `$S10/bin/claude`; only that
binary is copied into the preparation root. Start with the exact case, then
launch interactively from its matching target using an environment allowlist.
`env -i` omits inherited `CLAUDE_CONFIG_DIR`, route/model overrides,
debug/logging flags, proxy settings, and authentication state. HOME and XDG
are set to this disposable home and case-local directory; only the selected
gateway/model and the environment-only key mapping are added. This does not
inspect, copy, or export credential stores or OAuth tokens. Never print the
key or enable shell tracing:

```sh
REPO="$(git rev-parse --show-toplevel)"
FACTS="$REPO/internal/asset/testdata/adoption/greenfield/facts.json"
(
  CASE=fresh
  cd "$S10/targets/skill/$CASE"
  env -i PATH="$S10/bin:/usr/bin:/bin" HOME="$S10/home" \
    XDG_CONFIG_HOME="$S10/xdg/skill/$CASE" \
    S10="$S10" CASE="$CASE" \
    TARGET="$S10/targets/skill/$CASE" \
    STAGE="$S10/evidence/collection-staging/skill/$CASE" \
    FACTS="$FACTS" \
    TERM="${TERM:-xterm-256color}" LANG="${LANG:-C.UTF-8}" \
    ANTHROPIC_BASE_URL=https://opencode.ai/zen/go \
    ANTHROPIC_MODEL=deepseek-v4-flash \
    ANTHROPIC_API_KEY="${OPENCODE_API_KEY:?}" \
    CLAUDE_CODE_MAX_RETRIES=0 \
    "$S10/bin/claude" --model deepseek-v4-flash \
      --tools Bash,Read,Skill --disallowedTools Write,Edit \
      --permission-mode default --strict-mcp-config \
      --mcp-config "$S10/evidence/empty-mcp.json" --setting-sources user
)
```

At each permission prompt, manually approve only the exact required `lds`
operation or outside-target reviewed-plan `Read`; deny unrelated commands.
Invoke `/lds`, personally read the complete raw plan, and type the fresh
exact `apply <digest>` in chat only if approving. Do not give Claude
preapproval or allow the model to author a decision. Retain exact actual
JSON tool results and an owner-authored session record with `toolExitCodes`
for the live reader (`adopt`: 0), inventory/validate/render, plan, and apply
when actually invoked. Inventory and render exit 0. Validation exits 1 only
for the collision fixture, with the exact `validation.local-target-missing`
diagnostic at `docs/reference/usage.md:5` for `../howto.md`; all other cases
exit 0. Plan exits 3 only for collision and legacy veto; apply exits 3 only
for stale refusal and 0 for successful/current apply. Decline and veto cases
have no apply tool result. That validation diagnostic alone does not veto
adoption. The helper checks this exact per-case operation set, binds
inventory paths/hashes to the full before snapshot, checks render bytes
against independent fixture outputs, retains and compares validation
findings/skips, and binds plan/apply to the full reviewed digest. In the
actual root `adopt` result, inventory and validation are embedded as their
body objects (`files`/`links`/`skipped` and `findings`/`skipped`); unlike
standalone plumbing results, those bodies have no `schemaVersion` or
`operation` envelope. The helper validates each actual shape and compares the
complete bodies against standalone prelaunch observations after removing
only those two wrapper fields. Stop at eight logical turns or on any gateway
trouble; the command's ordinary retry setting is not a wire-request cap.

The empty MCP file is `{"mcpServers":{}}`. A future authorized launch must
set `CLAUDE_CODE_MAX_RETRIES=0` and leave `CLAUDE_CODE_RETRY_WATCHDOG` unset;
this removes the ordinary retry budget but is not a zero-wire-request
guarantee. Do not use `-p`,
`--allowedTools`, `dontAsk`, `bypassPermissions`, auto-accept, broad Bash
allow rules, or `Write`/`Edit`. The operator must manually review and answer
each permission prompt for an exact approved `lds` command or the outside-
target plan `Read`; deny unrelated commands. Start each new Claude session
inside its prepared target, invoke the installed `/lds` skill, and keep
collection inputs in the distinct outside-target staging directory
`$S10/evidence/collection-staging/skill/<case>/`. Finalized evidence remains
under `$S10/evidence/live/skill/<case>/`; never put staging inputs in the
target or final evidence slot. The case staging root must contain exactly
`flow-result.json`, `reviewed-plan.json`, `before-snapshot.json`,
`session-evidence.json`, `human-checkpoint.json`, and `tool-results/`.
`tool-results/` contains exactly `inventory.json`, `validate.json`, and
`render.json`, plus `apply.json` only for cases that actually invoke apply.
`reviewed-plan.json` is the single captured `docs plan` stdout and the single
source of truth for the complete reviewed plan; do not create a duplicate
`tool-results/plan.json`. `tool-results/apply.json` is the sole captured
`docs apply` result; do not supply or retain a standalone duplicate apply
artifact. Capture each actual operation's own stdout once, and never replace
it with a clean copied result. Capture the plan from the actual `/lds` session,
personally read the complete JSON (through the case-staged outside-target
`Read`), and do not regenerate or replace it to retry a session. The skill
reads the live shared flow, runs only the scoped LDS plumbing operations, and
presents the entire `docs plan --json` artifact
through `Read`. The owner personally types a new exact digest decision in
chat only after seeing all fields. The owner must also manually author a
case/channel/digest-bound checkpoint attesting complete-plan review and the
actual personally typed decision; never derive it from a transcript with a
script. Chat text, not a script, is the human decision; the model must not
decide the digest or infer consent. A vetoed plan never requests confirmation
or invokes apply.

Before the session, create the case staging directory and record the exact
fixture before-state with the helper:

```sh
CASE=fresh
TARGET="$S10/targets/skill/$CASE"
STAGE="$S10/evidence/collection-staging/skill/$CASE"
mkdir -m 0700 -p "$STAGE/tool-results"
python3 contrib/bds251-s10.py snapshot --root "$TARGET" \
  --out "$STAGE/before-snapshot.json"
```

During the actual skill session,
capture each existing LDS tool invocation's own JSON stdout into its matching
staged file; do not rerun an operation just to manufacture a missing tool
result. For the complete plan, use the exact plan invocation once with
`set -o pipefail` and `tee` to write the sole reviewed-plan artifact:

```sh
set -o pipefail
"$S10/bin/lds" plumbing docs plan "$TARGET" --facts "$FACTS" --json | \
  tee "$STAGE/reviewed-plan.json"
```

Then use `Read` on `$STAGE/reviewed-plan.json` so the entire raw plan is
directly visible before any owner decision. Save the actual reader result as
`flow-result.json`, inventory/validation/render outputs in their named
`tool-results/` files, and an invoked apply result only in
`tool-results/apply.json`; decline/veto cases have no apply file. The
reviewed-plan file and apply tool result each have one actual captured source;
the collector never replaces an observed artifact with a duplicate. This
writes only outside-target evidence; manually review each Bash permission and
deny anything beyond the exact LDS operation and those named artifacts. Do
not capture a Claude transcript, debug stream, or raw terminal input.

For stale skill, pause after the owner has read the entire plan artifact. In a
separate terminal, use `stale-insert --channel skill --target
<prepared-skill-stale-target> --plan <outside-target-plan.json>`; then the
owner personally sends the old exact digest in chat. The controller changes
only the prepared stale fixture target and writes channel-scoped event and
snapshot artifacts. On decline, do not call apply. Retain actual flow, plan,
tool-result, process-exit, session, turn/retry, human-checkpoint, and snapshot
evidence without raw transcript or stream material. Settle every attempt
once with `ledger-settle-skill`; failures remain failed/stopped and cannot be
retried or reset. Only a fully successful case may proceed through
`sanitize-skill` and `verify-skill`; collection is not verification, and
missing or failed evidence cannot receive PASS. Use `compare-semantic` only
after per-case verification, for each paired scenario.

The collection command consumes only that case's exact staged inputs and
copies sanitized artifacts into the separate final evidence slot. For a
successful fresh case, the invocation is:

```sh
STAGE="$S10/evidence/collection-staging/skill/fresh"
: "${OBSERVED_OPERATION_EXIT:?set from the retained actual tool result}"
: "${OBSERVED_SESSION_EXIT:?set from the actual Claude process exit}"
: "${OBSERVED_TURNS:?set from the observed logical model-turn count}"
: "${OBSERVED_RETRY_EVENTS:?set to the observed retry category/count}"
: "${OBSERVED_OUTCOME:?set from the actual case outcome}"
python3 contrib/bds251-s10.py sanitize-skill --prepared "$S10" --case fresh \
  --flow-result "$STAGE/flow-result.json" \
  --plan "$STAGE/reviewed-plan.json" \
  --before "$STAGE/before-snapshot.json" \
  --tool-results "$STAGE/tool-results" \
  --session-evidence "$STAGE/session-evidence.json" \
  --human-checkpoint "$STAGE/human-checkpoint.json" \
  --operation-exit-code "$OBSERVED_OPERATION_EXIT" \
  --session-exit-code "$OBSERVED_SESSION_EXIT" --turns "$OBSERVED_TURNS" \
  --retry-events "$OBSERVED_RETRY_EVENTS" --outcome "$OBSERVED_OUTCOME"
python3 contrib/bds251-s10.py verify-skill --prepared "$S10" --case fresh
```

For decline or a veto with no apply, `tool-results/apply.json` is absent and
the collector retains no substitute apply artifact. `sanitize-skill` reads
that one actual apply result when required and refuses missing, malformed,
extra, symlinked, cross-case, or already-finalized evidence; it does not
overwrite an earlier result or authorize a retry. An offline validation of
this collection sequence uses isolated disposable copies and synthetic
test-only session/checkpoint fixtures; those fixtures are never human consent
or S10 live evidence.

The ledger reserves no more than six root Chat calls and nine native
sessions, with an eight-logical-turn ceiling reserved per native session:
`6 + 9 × 8 = 78` logical requests for this executor wave. The verifier owns
its separately authorized wave. A failed attempt is recorded, not retried or
reset; no revised prompt or retry-until-green is allowed. A route change is
also blocked until root reviews the retained failure/counts and separately
settles access, model/configuration, and budget. The ledger's wire-request
field explicitly says **not measured/not a proven cap**.

## Separate one-call supplemental decline profile

The original executor preparation and its six consumed rows are immutable.
The owner-approved supplemental exception is prepared in a **new** disposable
root and does not enlarge, reset, or reinterpret that executor ledger. Select
the explicit profile when preparing it:

```sh
SUPPLEMENTAL=/tmp/lds-bds251-s10-supplemental-decline-native-binding
python3 contrib/bds251-s10.py prepare --profile supplemental-decline \
  --out "$SUPPLEMENTAL"
mkdir -p "$SUPPLEMENTAL/bin"
nix develop --command go build -o "$SUPPLEMENTAL/bin/lds" ./cmd/lds
test "$(sha256sum "$SUPPLEMENTAL/bin/lds" | cut -d' ' -f1)" = \
  cecaa451e28e9abb39a5d283e90adff4f5e231b0120a31a5d5e2c4e8d1240e8a
python3 contrib/bds251-s10.py pin-runtime \
  --prepared "$SUPPLEMENTAL" --binary "$SUPPLEMENTAL/bin/lds"
```

That profile is enforced by the helper's authorization table and the pinned
manifest/ledger binding: its only eligible root case is `decline`,
`rootChatCalls` and `logicalRequestsPerWave` are each exactly 1, native
sessions and logical turns are both 0, and native authorization must remain
blocked. The selected case cannot be reserved twice or launched again after
consumption; other executor cases, mutated limits, executor-profile promotion,
and native reservations are rejected. The default `executor` profile retains
its six-call/nine-session/78-logical-request limits and sixth-call boundary.
Every native reserve, consume, and settle command resolves the canonical
prepared manifest and ledger, validates the selected profile plus immutable
helper/binary/fixture/before-snapshot runtime pins before locking or writing,
and refuses zero-native profiles even if ledger fields are promoted. Settlement
also requires exactly one consumed/unknown row before writing a session
observation.
The wire-request count remains unmeasured; this profile caps application
requests, not gateway retries.

Preparation alone creates no reservation, checkpoint, or provider request.
Pin the new helper, exact baseline binary, fixture assets, and before-snapshot
bytes before any later owner instruction. Do not reuse the completed
owner-ready root, edit its ledger, or run `pack-preflight` over its accepted
evidence. The supplemental root is not eligible for a reservation until the
profile correction has independent offline High review and root provides a
fresh owner-Terminal instruction. If that gate is subsequently met, the one
owner-operated sequence is:

```sh
python3 contrib/bds251-s10.py ledger-reserve-root \
  --ledger "$SUPPLEMENTAL/evidence/ledger.json" --case decline
python3 contrib/bds251-s10.py run-root \
  --prepared "$SUPPLEMENTAL" --binary "$SUPPLEMENTAL/bin/lds" --case decline
```

The owner alone must operate the real Terminal, personally type `send`, read
the complete plan, and personally type the product's categorical `decline`;
the helper/model must not relay input or author the checkpoint. Retain the
actual owner checkpoint and run `verify-root` only after that single attempt.
A failed, interrupted, uncertain, or applied outcome consumes the sole
supplemental row and cannot be retried or reclassified. This exception is not
a full CLI matrix or S10 parity PASS, does not authorize further decline
attempts, and does not change deferred native N2. Existing preflight evidence
continues to describe the original executor preparation and remains
unchanged.

## Native retry limit: unresolved before any skill inference

Offline review of pinned Claude Code `2.1.286` and the current official
[error reference](https://code.claude.com/docs/en/errors#automatic-retries),
[environment-variable reference](https://code.claude.com/docs/en/env-vars),
and [gateway compatibility guide](https://code.claude.com/docs/en/llm-gateway-protocol#automatic-retry-and-error-forwarding)
confirms that `CLAUDE_CODE_MAX_RETRIES=0` removes the ordinary retry budget,
but does **not** establish a zero-wire-request guarantee. The error reference
documents a separately bounded reissue for some stalled streams outside the
ordinary budget. The gateway guide documents additional capability-rejection
reissues and does not promise that all such paths are disabled by
`MAX_RETRIES=0`. `CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK=1` can disable
one fallback class, but the documented streaming-404 exception remains. The
pinned executable also contains a `CLAUDE_CODE_NONSTREAMING_TIMEOUT_RETRIES`
setting name; its supported semantics were not established offline, so it
must not be set or relied upon.

The interactive TUI displays retry indicators and attempt counters for
visible retry paths, but neither official docs nor the pinned CLI expose a
guaranteed wire-HTTP request counter. The retry watchdog must remain unset;
setting it can remove retry caps. Interactive `--max-turns` is unavailable
(the documented limit is print-mode-only), so the owner must manually stop
at eight logical model turns. Strict permissions and empty MCP bound tools,
not network reissues.

Therefore native skill inference is **blocked pending a new root decision**.
Do not attempt a probe to see whether a retry happens. Root should choose
either (N1) an explicit bounded elapsed-time/spend envelope that accepts the
documented client reissues under `CLAUDE_CODE_MAX_RETRIES=0` with the watchdog
unset, or (N2) defer native inference until a separately permitted route or
request-accounting method is settled. My lean is N2 because neither a
complete retry-disable switch nor a wire-level count was established. If
there is any gateway/model/auth/config trouble after a later authorization,
stop immediately with the failed outcome and counts; root owns any
Anthropic-subscription access, route change, or remaining/revised budget. Do
not self-configure that fallback.

## Evidence limits

S10 evidence remains disposable fixture evidence. It does not revise or
explain S7's independently failed `exactCommand` attempt; that remains
failed/unexplained. Do not claim a real pilot, first-release inclusion,
Anthropic support, a Claude gateway guarantee, universal provider/tool
reliability, or a transactional filesystem guarantee. Preserve the accepted
fixture facts and frozen Stage 7, S7/reconciliation and current flow/plumbing
contracts unchanged.
