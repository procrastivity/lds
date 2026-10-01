# BDS-251 provider and Claude Code access gate

**Status:** Stage 4 Step 7 independent High review **PASS for bounded
provider/harness access only**; accepted by root for the Step 7 gate.
The original failed attempt remains failed and unexplained. Step 8 has not
started at this record's acceptance.
This records authorized disposable access, not an implemented `lds adopt`,
paired-form acceptance, or first-release inclusion.

## Owner selection and secure configuration

The owner resumed orchestration and authorized bounded live endpoint and
Claude Code probes with nonsensitive disposable coding fixtures.

| Item | Recorded selection and boundary |
| --- | --- |
| Provider | OpenCode Go; no provider fallback or OpenCode-harness substitution. |
| CLI endpoint | `https://opencode.ai/zen/go/v1/chat/completions`, OpenAI-compatible Chat Completions. |
| Exact model | `deepseek-v4-flash`, without an app-provider prefix, alias, or fallback. |
| Credential source | `OPENCODE_API_KEY`, supplied privately as an LDS Amp project secret. Its private Ubuntu-transfer match is an owner/root assertion, not a credential-file inspection performed here. |
| Authorized data | Disposable Wordcount fixture and bounded adoption/document-reader instructions, not private project content or a real pilot. |
| Harness | Claude Code only, preserving [ADR-0002](../decisions/0002-bds-251-adoption-workflow-boundary.md) 5A. |
| Write decision | Still 6A: display the entire eligible plan and obtain a fresh human decision for its exact digest. Facts, access, a returned digest and fixture consent are not approval. Noninteractive adoption stops without writes. |

The secret was present/nonempty; only a Boolean was checked and reported.
No credential file was inspected, and no credential value, authentication
header or secret-derived hash was printed or retained in evidence. No
refresh was needed after the root-scheduled refresh. A future job must stop
if the secret remains absent after the permitted process refresh; no
substitute credential or mock satisfies this gate.

The connection's secure configuration is environment-only. The CLI source
is the named project secret; harness probes map it in child-process memory
to `ANTHROPIC_API_KEY`. No committed credential setting, token argument,
persistent key file or new credential source is introduced. Missing access
is a stop, not automatic login/provider/model selection. This records a
connection boundary, not a runtime-client library, orchestrator architecture
or new LDS flags/configuration schema.

## Live protocol and harness evidence

Official [OpenCode Go documentation](https://opencode.ai/docs/go), read on
2026-10-01, lists this exact model on the selected Chat Completions endpoint.
Direct probes sent coding-agent fixture traffic, application-specific
User-Agent `lds-adoption-gate/0.1` and stable `x-opencode-session`
`lds-bds251-step7-wordcount-20261001`. Go also lists Claude Code as validated
and recognizes its native session header without a wrapper. Actual Claude
Code used native client/session behavior; request headers were not dumped
to inspect it.

Claude Code's official [gateway protocol](https://code.claude.com/docs/en/llm-gateway-protocol)
requires Anthropic Messages, not Chat Completions. Its
[connection guide](https://code.claude.com/docs/en/llm-gateway-connect)
documents the credential mapping and base URL. The verified direct harness
path is `ANTHROPIC_BASE_URL=https://opencode.ai/zen/go`, allowing Claude Code
to call `/v1/messages` using the **same** provider, model and secret source.
This is a companion harness interface, not replacement of the selected LDS
CLI endpoint or a locally built protocol proxy.

| Probe on 2026-10-01 UTC | Sanitized observed result | Evidence limit |
| --- | --- | --- |
| Chat Completions tool request, `max_tokens: 2048`, nonstreaming | HTTP 200, response model `deepseek-v4-flash`, `finish_reason: tool_calls`; exactly one `docs_inventory` function with the explicitly supplied disposable root. | Actual authenticated model-labelled response and structured tool request, not general model/tool reliability or weight-identity proof. |
| Actual LDS inventory result returned in that conversation | Inventory exit 0; schema 1, operation `docs inventory`, empty `files`, `links`, `skipped`. Continuation: HTTP 200, same response model, `finish_reason: stop`, nonempty text mentioning human/digest decision. | Live tool-result continuation; `reasoning_content` retained in memory for tool history. The text check is lexical, not content accuracy or approval enforcement. |
| Direct Messages diagnostic with bearer authentication | HTTP 401 with JSON error. | This variant failed; do not infer that all credentials or Messages access fail. |
| Same-model Messages diagnostic with `x-api-key` | HTTP 200. | Accepts the selected model through Messages with this credential convention, although its Go model table lists Chat Completions. No service guarantee inferred. |
| Actual Claude Code 2.1.286, isolated home, no tools, selected Messages connection | Exit 0, JSON result `is_error: false`, no API-error code observed. | Authenticated inference, not installation alone or a mocked client. |
| Actual non-bare Claude Code 2.1.286 with existing generated `lds` skill and read-only document operation | Exit 0, `is_error: false`, two turns, zero permission denials. Init listed `lds` and exact model; `modelUsage` had only `deepseek-v4-flash`. Bash requested exactly `lds plumbing adopt --json`; actual tool result contained schema 1, `preparatory`, `unavailable-for-execution` and expected embedded asset hash. | Actual skill discovery plus live reader/tool-result access. Expanded skill-body text was not emitted in the captured stream, so this is not independent proof of expansion contents or paired adoption. |

An earlier bare-mode reader probe succeeded but did not list the skill;
it is not the skill-discovery evidence. Keyword flags over a whole successful
response are not error verdicts. The retained decisive evidence is process
and result status, actual tool-use/result records and observed model keys.

Anthropic's [gateway documentation](https://code.claude.com/docs/en/llm-gateway)
explicitly does **not** support non-Claude models through gateways. This is
bounded OpenCode-provider compatibility observed with Claude Code 2.1.286,
not an Anthropic support/endorsement claim or proof that every Go model
works. No protocol bridge, model translation, alternate harness or broad
infrastructure project was installed. Later paired acceptance must exercise
the actual forms again.

## Baseline, commands and preservation checks

Root reported committed local main
`3c3a5cde69680c31c4daec52cd810aa1deb5c4fd`. The mismatching original planner
checkout was left unchanged. An ordinary disposable clone at
`/tmp/lds-bds251-step7.efDYs4` checked out the
[origin baseline](https://github.com/procrastivity/lds/commit/3c9af08c10d2eb8c004ec5ca1a674f4e2632cb4e)
and applied the supplied full-tree patch after `git apply --check`.

- Input `bds251-stage4-step7-root-baseline.patch.gz`: 1,225,773 bytes;
  SHA-256 `b1e53d15108cd9636911286c32dcb6d4b9aa170c07948b27a381c59570eab578`.
  Reverse `git apply --check` succeeded before this record was added. Two
  supplied-baseline whitespace warnings were left unchanged. No worktree,
  root-store access, commit or push was used.
- Go was absent from PATH. Isolated Go 1.26.8 linux/amd64 came from
  `https://go.dev/dl/go1.26.8.linux-amd64.tar.gz`; the published archive SHA
  `d0f743b33e8d8945e6b1f432edd15785c70507121d6e2a723b21285eddf8b57b`
  matched. `go build -o /tmp/lds-bds251-step7-tools/bin/lds ./cmd/lds`
  succeeded. Binary SHA-256
  `663fffef934de861fc9476a9ffc3fe2f9a399ebd7a0500417d93fbbfda641bc2`
  identifies this build, not reproducible-build equivalence.
- The official [native Claude Code installer](https://code.claude.com/docs/en/setup)
  ran with disposable `HOME=/tmp/lds-bds251-step7-tools/home`.
  `claude --version` returned `2.1.286 (Claude Code)`; installation alone
  was not counted as live access.
- `lds install claude-code --json`, with that home, disposable XDG root and
  explicit disposable `LDS_CLAUDE_SKILLS_DIR`, exited 0, `status: installed`.
  Generated `lds/SKILL.md` SHA-256:
  `b91179e87faa58919c65152d0a45b450a6d97cdc48c4d30a90e1daa817ef884e`.
- `go test ./internal/cli -run '^TestAdoptDocument_' -count=1` returned
  `ok github.com/procrastivity/lds/internal/cli` (0.364s). These tests
  cover the preparatory reader/override/manifest/projection, not the paired
  provider-backed workflow.

Probes used a copy of `internal/asset/testdata/adoption/greenfield/target`
at `/tmp/lds-bds251-step7-tools/target`. Its only files before and after
all probes were:

| Path | Unchanged byte SHA-256 |
| --- | --- |
| `sample.txt` | `0fedab7a6bcdee67c26091670c1aba184dacda6b69a321611f1dd53910e917c9` |
| `word-count.sh` | `fe429d0d1c8969ab6b4aa7df28e32c29a7394d851e0f2b365927241859722559` |

No adoption output, real-target plan/apply or source/reference repository
write occurred. Generated skill/tool installation stayed in the disposable
tooling home. Sanitized summaries contain statuses, structural predicates and
public fixture hashes, not raw authentication headers, credentials, model
reasoning or debug logs.

## Shared-flow compatibility and remaining work

The [single flow asset](../../assets/flows/adopt.md) is still the shared
source required by [Toolsmith C8.1–C8.4](https://github.com/procrastivity/toolsmith/blob/main/CONTRACT.md#c8--shapes).
The [flow/plumbing map](adoption-flow-plumbing-map.md) exposes the same-named
document reader and five deterministic docs operations. Embedded flow
SHA-256, independently checked on disk and in the actual harness tool result:
`00c1cecd7f323b716793ebae2f4b06dd3c2bff76e6b26be78c12060e004a4f33`.
The [reader](../../internal/verbs/adopt/adopt.go) correctly reports
preparatory/unavailable-for-execution; root `lds adopt` is still absent.

The flow's introduction and final disclaimer incorrectly say the plumbing
document reader does not exist yet; the introductory provider-selection gate
is now historical. Step 8 must correct those temporal claims with coherent
availability and skill guidance when it implements the workflow. Do not
duplicate the numbered flow, treat its current preparation disclaimer as
execution permission or claim the installed general `lds` skill is already
the delivered adoption shape. This record changes no asset, reader, manifest,
projection or runtime code.

Step 8 must preserve the [adoption contract](adoption-contract.md) and
[operations contract](adoption-operations-contract.md): complete verified
owner facts including user-operation facts; exactly four fixed files;
all-create preflight veto on collision/refusal; no migration/overwrite or
symlink traversal; full displayed-plan digest commitment and revalidation;
fresh exact-digest human decision; honest late partial-effect reporting.
Broken-link findings/skips alone remain review diagnostics, not vetoes. The
provider does not approve facts or authorize writes. A model's mention of
approval does not enforce that gate.

**Recommendation:** selection, credential availability, actual endpoint access
and actual Claude Code inference/document access have bounded live evidence.
Submit for independent High verification before Step 8. No demonstrated
access blocker currently requires another owner choice. The companion
Messages interface is empirically verified, not promised by the model's Go
endpoint table. Later errors/changes must stop and be reported, not silently
bridged or substituted. Paired execution, full-plan human interaction,
parity, provider-failure behavior and disposable workflow acceptance remain
later evidence; this is not C8 completion. The 1B real-pilot deferral and
first-release exclusion remain unchanged.

## Sanitized reproduction

Use the authorized disposable tree/home/target and private environment
secret only. Build/install as above; copy the two fixture files into `target/`,
keep `xdg/` empty and install the existing skill at
`home/.claude/skills/lds`. The following equivalent fixture prompts reproduce
bounded operations, not byte-for-byte request replay or a workflow suite.
Never dump shell environments, raw HTTP responses or harness/debug streams.
Native Claude Code uses its provider-recognized session header.

```python
import hashlib, json, os, subprocess, urllib.request, urllib.error
from pathlib import Path
base = Path('/tmp/lds-bds251-step7-tools')
root, model = base/'target', 'deepseek-v4-flash'
key = os.environ.get('OPENCODE_API_KEY')
if not key:
    raise SystemExit('credential_available: false; stop')
def snapshot():
    return {p.relative_to(root).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
            for p in sorted(root.rglob('*')) if p.is_file()}
before = snapshot()
class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None
opener = urllib.request.build_opener(NoRedirect)
def post(payload):
    req = urllib.request.Request(
        'https://opencode.ai/zen/go/v1/chat/completions',
        data=json.dumps(payload).encode(), headers={
            'Content-Type': 'application/json', 'Authorization': 'Bearer '+key,
            'User-Agent': 'lds-adoption-gate/0.1',
            'x-opencode-session': 'lds-bds251-step7-wordcount-20261001'})
    try:
        with opener.open(req, timeout=90) as response:
            status, data = response.status, json.loads(response.read(2_000_000))
    except urllib.error.HTTPError as error:
        raise SystemExit('HTTP '+str(error.code)+'; stop; body not logged')
    if status != 200 or data.get('model') != model:
        raise SystemExit('Unexpected provider result; stop')
    print(json.dumps({'http': status, 'responseModel': model,
                      'finishReason': data['choices'][0]['finish_reason']}))
    return data['choices'][0]['message']
messages = [
    {'role': 'system', 'content': 'Coding-agent review of disposable Wordcount. '
     'No writes. First call docs_inventory with the exact root; then explain '
     'that verified facts and a fresh human digest decision are required.'},
    {'role': 'user', 'content': 'Target root: '+str(root)+
     '\nword-count.sh uses wc -w < "$1". sample.txt contains one two three. '
     'This is a test-only fixture. Request inventory now.'}]
tool = {'type': 'function', 'function': {
    'name': 'docs_inventory', 'description': 'Read-only LDS docs inventory.',
    'parameters': {'type': 'object', 'properties': {'root': {'type': 'string'}},
                   'required': ['root'], 'additionalProperties': False}}}
reply = post({'model': model, 'max_tokens': 2048, 'stream': False,
              'messages': messages, 'tools': [tool]})
calls = reply.get('tool_calls', [])
if (len(calls) != 1 or calls[0]['function']['name'] != 'docs_inventory' or
        json.loads(calls[0]['function']['arguments']) != {'root': str(root)}):
    raise SystemExit('Unexpected tool request; stop without executing it')
inventory = subprocess.run([str(base/'bin/lds'), 'plumbing', 'docs', 'inventory',
    str(root), '--json'], env={**os.environ, 'XDG_CONFIG_HOME': str(base/'xdg')},
    capture_output=True, text=True, check=True).stdout
messages.extend([reply, {'role': 'tool', 'tool_call_id': calls[0]['id'],
                        'content': inventory}])  # keep reasoning_content
post({'model': model, 'max_tokens': 2048, 'stream': False, 'messages': messages})
env = {k: v for k, v in os.environ.items() if k in
       ('LANG', 'LC_ALL', 'SSL_CERT_FILE', 'SSL_CERT_DIR', 'HTTPS_PROXY',
        'HTTP_PROXY', 'NO_PROXY')}
env.update({'PATH': str(base/'bin')+':'+os.environ['PATH'], 'HOME': str(base/'home'),
    'CLAUDE_CONFIG_DIR': str(base/'home/.claude'), 'XDG_CONFIG_HOME': str(base/'xdg'),
    'ANTHROPIC_BASE_URL': 'https://opencode.ai/zen/go', 'ANTHROPIC_API_KEY': key,
    'ANTHROPIC_MODEL': model, 'ANTHROPIC_DEFAULT_OPUS_MODEL': model,
    'ANTHROPIC_DEFAULT_SONNET_MODEL': model, 'ANTHROPIC_DEFAULT_HAIKU_MODEL': model,
    'CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC': '1',
    'CLAUDE_CODE_MAX_OUTPUT_TOKENS': '2048'})
run = subprocess.run([str(base/'home/.local/bin/claude'), '--model', model,
    '--tools', 'Bash,Read,Skill', '--allowedTools',
    'Bash(lds plumbing adopt --json)', 'Read', 'Skill(lds)',
    '--permission-mode', 'dontAsk', '--strict-mcp-config',
    '--mcp-config', '{"mcpServers":{}}', '--setting-sources', 'user',
    '--no-session-persistence', '--output-format', 'stream-json', '--verbose',
    '--max-budget-usd', '0.10', '-p', '/lds Run exactly lds plumbing adopt --json '
    'and report status/availability/asset hash only. Do not execute the flow or '
    'write files. Disposable access probe, not plan/apply approval.'],
    cwd=root, env=env, capture_output=True, text=True, timeout=120)
events = [json.loads(line) for line in run.stdout.splitlines() if line.startswith('{')]
init = next(e for e in events if e.get('type') == 'system' and e.get('subtype') == 'init')
result = next(e for e in events if e.get('type') == 'result')
import shlex
commands, documents = [], []
bash_ids, document_ids = [], []  # raw commands/IDs remain in memory only
for event in events:
    blocks = event.get('message', {}).get('content', [])
    for block in blocks if isinstance(blocks, list) else []:
        if block.get('type') == 'tool_use' and block.get('name') == 'Bash':
            commands.append(block.get('input', {}).get('command'))
            bash_ids.append(block.get('id'))
        if block.get('type') == 'tool_result' and isinstance(block.get('content'), str):
            try:
                doc = json.loads(block['content'])
                if doc.get('operation') == 'adopt':
                    documents.append(doc)
                    document_ids.append(block.get('tool_use_id'))
            except ValueError:
                pass
def valid_id(value):
    return isinstance(value, str) and bool(value)
id_commands = {}
normalized_matches, normalization_failures = 0, 0
for tool_id, command in zip(bash_ids, commands):
    if valid_id(tool_id):
        id_commands.setdefault(tool_id, []).append(command)
    if isinstance(command, str):
        try:
            normalized_matches += shlex.split(command, comments=False, posix=True) == [
                'lds', 'plumbing', 'adopt', '--json']
        except ValueError:
            normalization_failures += 1
matching_result_ids = [value for value in document_ids
                       if valid_id(value) and value in id_commands]
diagnostics = {
    'bashToolUseBlockCount': len(commands),
    'exactStringMatchCount': sum(value == 'lds plumbing adopt --json' for value in commands),
    'argvNormalizedMatchCount': normalized_matches,
    'argvNormalizationFailureCount': normalization_failures,
    'nonStringCommandCount': sum(not isinstance(value, str) for value in commands),
    'uniqueBashToolIdCount': len(id_commands),
    'repeatedBashToolIdCount': sum(len(values) > 1 for values in id_commands.values()),
    'repeatedBashToolIdObservationCount': sum(len(values)-1 for values in id_commands.values()),
    'missingBashToolIdCount': sum(not valid_id(value) for value in bash_ids),
    'repeatedIdConflictingCommandCount': sum(
        any(value != values[0] for value in values[1:]) for values in id_commands.values()),
    'documentToolResultBlockCount': len(documents),
    'matchingDocumentToolResultIdCount': len(matching_result_ids),
    'uniqueMatchingDocumentToolResultIdCount': len(set(matching_result_ids)),
    'missingDocumentToolResultIdCount': sum(not valid_id(value) for value in document_ids),
    'unmatchedDocumentToolResultIdCount': sum(
        valid_id(value) and value not in id_commands for value in document_ids),
}
print(json.dumps({'commandDiagnostics': diagnostics}))  # counts only, never strings/IDs
expected = '00c1cecd7f323b716793ebae2f4b06dd3c2bff76e6b26be78c12060e004a4f33'
checks = {'exit0': run.returncode == 0, 'resultOK': result.get('is_error') is False,
    'exactModel': init.get('model') == model and sorted(result.get('modelUsage', {})) == [model],
    'skillListed': 'lds' in init.get('skills', []) or 'lds' in init.get('slash_commands', []),
    'exactCommand': commands == ['lds plumbing adopt --json'],
    'documentOK': len(documents) == 1 and documents[0].get('schemaVersion') == 1
        and documents[0].get('status') == 'preparatory'
        and documents[0].get('availability') == 'unavailable-for-execution'
        and documents[0].get('asset', {}).get('sha256') == expected,
    'noPermissionDenials': not result.get('permission_denials'),
    'targetUnchanged': before == snapshot() and sorted(before) == ['sample.txt', 'word-count.sh']}
print(json.dumps(checks))
if not all(checks.values()):
    raise SystemExit('Unexpected observation; report it, do not infer a pass')
```

Inference is nondeterministic. Unexpected requests/responses must stop and be
reported, not executed or revised to manufacture a pass. The recipe does not
replay the failed bearer diagnostic or bare-mode probes; those observations
remain distinct in the evidence table.

The original recipe, before the diagnostic instrumentation above, was
executed after authoring this record, not merely compiled. Its original
planner-run sanitized stdout was:

```jsonl
{"http": 200, "responseModel": "deepseek-v4-flash", "finishReason": "tool_calls"}
{"http": 200, "responseModel": "deepseek-v4-flash", "finishReason": "stop"}
{"exit0": true, "resultOK": true, "exactModel": true, "skillListed": true, "exactCommand": true, "documentOK": true, "noPermissionDenials": true, "targetUnchanged": true}
```

### Independent blocked attempt and diagnostic boundary

Root reports that independent High verification executed the original
recipe with only the disposable base substituted. Both Chat Completions
turns returned HTTP 200. All eight native predicates were true **except
`exactCommand: false`**. The recipe stopped as specified; the verifier did
not rerun or revise inference to manufacture a pass. No safe command-shape
diagnostics were retained. This is a blocked acceptance attempt, not evidence
of an unsafe command or failed provider/harness access. Its cause cannot be
reconstructed from the predicates alone. This failure remains part of the
gate evidence regardless of a later diagnostic result.

The instrumentation adds diagnostics only: `exactCommand` still compares
the original collected list to exactly `['lds plumbing adopt --json']`.
No prompt, model, invocation, permission, endpoint or acceptance predicate
has changed. Root separately authorized **one** new bounded diagnostic
attempt under the owner's resumed disposable live-access authorization.
Further inference or an acceptance correction requires root judgment;
neither a successful diagnostic attempt nor these additional counts seals
Step 7 or starts Step 8.

Counts refer to observed tool-use/result blocks, not independently measured
process executions. `exactStringMatchCount` counts literal matches;
`argvNormalizedMatchCount` uses POSIX `shlex.split` with comments disabled
and requires exactly the four argv tokens above. This is lexical quote/space
normalization, not shell-safety validation or permission to run a variant.
Non-string commands and parsing failures are counted separately. A valid
tool ID here is a nonempty string, held only in memory; missing/invalid IDs
count as missing. `uniqueBashToolIdCount` counts distinct valid IDs,
`repeatedBashToolIdCount` counts IDs observed more than once, and
`repeatedBashToolIdObservationCount` counts their additional observations.
`repeatedIdConflictingCommandCount` counts IDs whose subsequent commands
differ from their first command. Document result counts cover parsed
`operation: adopt` results, with `tool_use_id` membership compared against
the observed Bash IDs; matching counts include both block occurrences and
distinct matched IDs. The original `documentOK` still checks exact document
shape/hash separately. No raw command, ID, stream, credential or header is
printed or saved by this instrumentation.

### Single authorized diagnostic result

The planner executed the instrumented recipe **once**, on 2026-10-01 UTC,
using the original disposable base and unchanged prompts/invocation/model.
The Boolean secret check was true. Existing isolated Claude Code 2.1.286
and the same LDS binary/skill were used; no new provider, harness, permission
or credential source was introduced. Sanitized stdout was:

```jsonl
{"http": 200, "responseModel": "deepseek-v4-flash", "finishReason": "tool_calls"}
{"http": 200, "responseModel": "deepseek-v4-flash", "finishReason": "stop"}
{"commandDiagnostics": {"bashToolUseBlockCount": 1, "exactStringMatchCount": 1, "argvNormalizedMatchCount": 1, "argvNormalizationFailureCount": 0, "nonStringCommandCount": 0, "uniqueBashToolIdCount": 1, "repeatedBashToolIdCount": 0, "repeatedBashToolIdObservationCount": 0, "missingBashToolIdCount": 0, "repeatedIdConflictingCommandCount": 0, "documentToolResultBlockCount": 1, "matchingDocumentToolResultIdCount": 1, "uniqueMatchingDocumentToolResultIdCount": 1, "missingDocumentToolResultIdCount": 0, "unmatchedDocumentToolResultIdCount": 0}}
{"exit0": true, "resultOK": true, "exactModel": true, "skillListed": true, "exactCommand": true, "documentOK": true, "noPermissionDenials": true, "targetUnchanged": true}
```

This observation has one literal-exact Bash block, one unique valid ID,
no repeated/missing IDs or conflicting commands, and one document result
matching that ID. It identifies **neither stream duplication nor a spelling
or count discrepancy in the earlier independent attempt**, whose command
diagnostics are unavailable. The failed independent acceptance remains
unexplained; this later successful diagnostic does not erase it or turn the
independent review into PASS. No unexpected observation occurred in this
single diagnostic attempt, and it was not repeated.

**Planner recommendation before the independent diagnostic review:** keep the original
`exactCommand` predicate and the BLOCKED/unsealed status. There is no
observed basis for an acceptance correction, ID deduplication, argv-match
substitution or relaxed command/count allowance. Any further independent
instrumented attempt requires separate root authorization; its actual
outcome must be retained. Step 8 remains unstarted.

Offline checks exercised eight asymmetric collector cases: one exact call,
quote/space-only argv equivalence, identical repeated ID, distinct IDs,
conflicting repeated ID, missing IDs, non-string/unparseable commands with
an unmatched result, and repeated document results. Counts and original
exact-list outcomes matched independent expectations. A byte comparison to
the original record confirmed unchanged live prompts/invocation and every
original acceptance predicate; this diagnostic revision changes collection
metadata and reporting only, not those gates.

### Independent diagnostic review and root access-gate judgment

The independent High reviewer verified the exact diagnostic record SHA-256
`ea41a7ce2f1490d001b29320d29e37c7f18fa0200ee44ea584614ee477ded776`
over baseline `b1e53d15108cd9636911286c32dcb6d4b9aa170c07948b27a381c59570eab578`.
All eight asymmetric offline collector cases and all 15 counts passed.
The reviewer consumed exactly one separately authorized live instrumented
attempt with a fresh disposable baseline, fixtures and Claude Code 2.1.286,
substituting only the disposable base path. Both Chat calls returned HTTP
200 with the exact model and expected finish reasons. All eight unchanged
native acceptance predicates were true, including `exactCommand`.
The counts matched the planner's diagnostic: one literal-exact Bash block,
one unique Bash ID, one matching document result and unique result ID,
and zero anomaly counts. The reader returned the expected preparatory
document and asset hash; the generated skill hash matched. Source and
two-file target bytes remained unchanged. Uncached `TestAdoptDocument_`,
baseline reverse-apply, six links, Python, JSONL, LF and whitespace checks
passed. No additional inference attempt or acceptance relaxation occurred.

Root accepts this independent PASS as resolving ADR-0002's bounded
provider/endpoint/model/credential/access prerequisite for proceeding to
Step 8. The original independent failure remains failed and unexplained;
the later result does not diagnose it or establish universal reliability.
Normalization remains diagnostic-only. No further live attempt is
authorized by this status adjustment. Paired delivery, expanded skill-body
proof, fresh full-plan human approval enforcement and CLI/skill parity
still require the remaining steps. No real pilot, first-release inclusion
or Anthropic support for non-Claude gateway models is claimed.
