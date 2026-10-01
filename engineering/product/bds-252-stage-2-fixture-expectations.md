# BDS-252 Stage 2 — independent fixture expectations

These are expected results authored from the accepted operation contract and
Stage 1 evidence/case cards **before scanner or journal implementation**.
They are not command output, an LDS reconciliation decision, or permission to
change either source repository. The [v1 operation contract](../specs/cross-project-reconciliation-v1.md)
and [ADR-0003](../decisions/0003-bound-cross-project-reconciliation.md)
define behavior; the [Stage 1 evidence](bds-252-stage-1-evidence.md) and
[case cards](bds-252-stage-1-expected-case-cards.md) supply bounded inputs.

Pinned contract SHA-256: `f721bc58a7c6fd8eb3ff0ac87f3708c1bb752a4f6a2b6bb6264fafb5d4f3f587`.
Pinned ADR-0003 SHA-256: `fee5a9862f2bf896fee233d949fd082b3ccb669afe373ac069ae2f386eb5cd92`.
Pinned Stage 1 evidence SHA-256: `0884630e811209b1c46b2bb1af7e747e8a5d91527cba885a50163838765a74bd`.
Pinned case-card SHA-256: `7d04c5dec48a6ebe1db2e4d9c3ed206c6ec8b0fb5b6ecb8d45daecce0204d430`.

## Fixture pair and independently computed identities

The source-derived fixture uses ordered projects `scind` (left) and `xcind`
(right), pair ID `fixture:lds-bds252:scind-xcind:v1`, and the pinned full
revisions Scind `c63cbc686afd2ca6ecc01eab7825d637d49378c7`, Xcind
`c244856e7119547ed6dceb2b7663a01720bdde66`. It scopes the three topic
buckets below, leaves Xcind ADR-0021 unbucketed on the right, and supplies
both tracked Xcind raw correspondence maps:

| Input path | Pinned SHA-256 / scope role |
| --- | --- |
| Scind `engineering/decisions/0011-options-based-targeting.md` | `92416f10e61b7afb891eac7cc6b1e8a9e8960b635d842b059cc9a86eef48bb73`; `topic:targeting` left member |
| Xcind `engineering/decisions/0023-location-based-targeting.md` | `88d2174b069096a01df7d0476302e42fa3ce6511352b3df8b592c27af794448d`; `topic:targeting` right member |
| Scind `engineering/decisions/0017-external-proxy-mode.md` | `9557cbfde3718cc23918e20effb76498eb788b1630e93396b49c00fddad1b7cc`; `topic:external-proxy` left member |
| Xcind `engineering/decisions/0022-external-proxy-mode.md` | `db5d802fb20b687829d46e5438334f00c33b3821ca4cb7cff3c3ef2d51377723`; `topic:external-proxy` right member |
| Scind `engineering/specs/port-types.md` | `9f002cb97b8fcb530a488b34a258e9994b41c60ae62b455e135f35d60622ded3`; `topic:tcp-sni` left evidence |
| Xcind `engineering/sync/divergence/0030-port-type-plugins-tcp-sni.md` | `0fac817abd1cb0a18f81746da10f7039cb2fa6438c5684b711d3924e36aefa8c`; `topic:tcp-sni` right evidence |
| Xcind `engineering/decisions/0021-cross-repo-adr-cross-referencing.md` | `619d180ff6eeba7d8c6924f11850d250d019f19116bfe05aba9cc485c61b482a`; right-only, deliberately not assigned to a topic bucket |
| Xcind `engineering/sync/artifacts/correspondence-map.json` | `175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17`; raw `--map` evidence included in every candidate |
| Xcind `engineering/sync/artifacts/correspondence-map.md` | `3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05`; raw `--map` evidence included in every candidate |

The map is evidence bytes only: do not parse its `MATCH`, `XCIND-ONLY`, or
other labels into candidate relationships or authority. In particular, the
external-proxy file/JSON classification is internally contradictory with its
topic row, so its evidence is bounded/contradictory, not a decision. Scind
runtime behavior remains unverified; Xcind observations remain bounded to
the cited code. TCP/SNI is Scind **Future** intent contrasted with Xcind's
recorded rejection, not a verified shipped Scind gap. Synthetic nodes below
are test-only, never Scind/Xcind facts.

The following IDs are independently computed from the contract formula
`cand-v1-` + SHA-256 of
`lds-reconcile-candidate-v1\n<pairId>\n<candidateKey>\n`; neither revision,
source hash, map hash, nor review state contributes:

| Pair ID | Candidate key | Expected stable candidate ID |
| --- | --- | --- |
| `fixture:lds-bds252:scind-xcind:v1` | `topic:targeting` | `cand-v1-4d5c8ebec9a4811936cda396f8f91f3f9dca37bd1ce96f85ce398d878d433bae` |
| same | `topic:external-proxy` | `cand-v1-29cfd6874814f7b3001b8babffc4705eb7d2fec4d4fcb789ba10a22baa1d894b` |
| same | `topic:tcp-sni` | `cand-v1-17b084d3d73c93a47b07d60b1934563196e12150d410b0b8ff7d13c5de9c486c` |
| same | `unmapped:right:engineering/decisions/0021-cross-repo-adr-cross-referencing.md` | `cand-v1-6de68402c8df9a0f3f211586b6ec7debef5ee63a121e640ae20a053d8a1d9304` |
| `fixture:lds-bds252:synthetic-split:v1` | `fixture:route-decomposition` | `cand-v1-aed9fd6fc71e2ccff638d2e91a338b80ba6761a929d4b44a67dd26d242d81af6` |

For each successful scan, candidate arrays sort by full candidate ID. Evidence
arrays sort by `(kind, projectId, revision, key, sha256)`; within Xcind map
artifacts, `correspondence-map.json` precedes `correspondence-map.md`. All
supplied map artifacts appear once in **every** candidate's evidence, even if
outside that candidate's scoped members. Thus both map hashes are present and
sorted in each candidate. The four source-pair evidence digests are also fixed
because this baseline pins the pair ID, candidate keys, source revisions,
artifact paths, and file hashes. They are independently computed using the
contract's exact newline-terminated preamble plus sorted tab-separated
evidence rows:

| Candidate key | Expected baseline evidenceDigest |
| --- | --- |
| `topic:targeting` | `sha256:2765ed93a8a5078ecc60cb983fde4d26b67ac95d04ba0fbd4bacaa635997f349` |
| `topic:external-proxy` | `sha256:5775c620170e2a16c56cd306b1d21f742cf39369d028fed23d59e8075d3b2236` |
| `topic:tcp-sni` | `sha256:b299310a771287426152781b60c52648002022080442696c0f03241ef390a168` |
| `unmapped:right:engineering/decisions/0021-cross-repo-adr-cross-referencing.md` | `sha256:21bd6eefd00a2033ac324832cef6ab5893c9fd6afd01c3cbde2d3fd8abd03f96` |

Absolute input paths are reported separately and do not enter
`evidenceDigest`. The later changed-source/map fixture commits alter the
revision preamble, so their resulting digest vectors are correctly
run-context values rather than these baseline constants.

Specifically, for targeting and external-proxy, evidence rows sort as Scind
source, Xcind ADR, Xcind correspondence-map JSON, then Xcind correspondence-
map Markdown. For TCP/SNI, the map keys under `engineering/sync/artifacts/`
sort before the divergence record under `engineering/sync/divergence/`. The
unmatched ADR-0021 evidence precedes the two Xcind map rows. This is the
expected evidence-item order regardless of input `--map` ordering.

## Scan and evidence-change vectors

In escaped byte strings in this table, `\n` denotes one LF byte (`0A`).
The appended comments are ASCII.

| Input / operation | Expected projection and exit | Boundary this catches |
| --- | --- | --- |
| Scan the exact pair, candidate buckets, two raw maps, and pinned trees twice; supply no journal or the same unchanged journal revision. | Both calls exit **0** and produce byte-identical v1 JSON for identical canonical input paths and bytes. Candidate IDs above remain identical. Candidate and evidence arrays have the required stable ordering; map artifact rows have exactly the two pinned hashes above. Scan creates/writes no journal or source file. | No clock/host-order leakage; stable IDs are not re-derived from source or outcome; sorting is over evidence tuples and IDs, not incidental input order. |
| Append `\n<!-- fixture-source-change: one -->\n` to the *disposable tracked copy* of Xcind ADR-0023, commit that copy, and pin the new full Xcind revision in a copied pair descriptor. Rescan. | Exit **0**. `topic:targeting` keeps its candidate ID, exposes the changed ADR byte hash, and has a different evidence digest/status `changed`. Since both pinned revisions are in each evidence-digest prefix, every candidate digest changes/status is `changed`, even where its scoped source-item hashes are otherwise unchanged. An explicit `record` may append the new observation, but preserves any prior decision. | Changed evidence is visible without renaming candidates, inferring changed semantics, or automatically clearing/altering a decision. Fixture-only revision, never a changed reference checkout. |
| Append `\n<!-- fixture-map-change: one -->\n` to a disposable tracked copy of `correspondence-map.json`, commit it, update the pair's Xcind revision and map path to that tracked file, then rescan. | Exit **0**. Every candidate retains its ID but has changed evidence digest/status; each includes the changed raw JSON map hash plus the unchanged Markdown map hash. Other scoped source-file hashes remain byte-identical. The scanner reports raw map bytes; it does not reinterpret the map's relationship claim. | A map is first-class evidence even when not a member of a candidate. Since maps must be tracked at the exact pinned revision, changing map bytes also changes the Xcind pin and therefore every candidate digest prefix. |
| Repeat the baseline pair but reverse the CLI left/right roots without changing the pair descriptor's ordered projects. | Exit **3**, `refusal.reconcile-stale-evidence`; no scan success object and no writes because each root's `HEAD` no longer matches its ordered pair revision. | CLI roots must agree with the explicit ordered pair; swapping paths is not an implicit authority/direction decision. |

For both changed-evidence rows, first record the baseline scan observations in
a disposable journal, then pass that journal to the changed scan. This makes
the expected `evidenceStatus: "changed"` comparable; without `--journal`, the
contract omits journal comparison fields. Recording the new scan is a separate
explicit request and must preserve existing decisions.

The changed-source row appends the exact bytes `\n<!-- fixture-source-change:
one -->\n`; against the pinned ADR-0023 bytes this yields SHA-256
`69ceadc9e13d55f1de1d2bc8cfa090f1f137ea4aa1931944472b6fbec22abb4f`.
The changed-map row appends `\n<!-- fixture-map-change: one -->\n` to the
pinned JSON map and yields SHA-256
`9fc37f1cfd3804809b6a0bdded002560be3f2c19621dc76bf32219ce37f8bc34`. Those
are file-content expectations only; the disposable commit IDs and evidence
digests depend on their fixture execution context.

The two changed-evidence vectors use a **disposable ordinary clone** with a
new tracked commit, not edits to the pinned Scind/Xcind checkouts. The
swapped-root refusal vector does not create a commit. Because the changed
vectors update the revision, the expected digest relationship (`new != old`)
is deterministic; their exact resulting digests and commit IDs are
fixture-run values, not fabricated constants in this specification.

## Journal lifecycle and authority vectors

All values and records described as owner choices below are fixture-only
inputs. They do not assign authority for the real Scind/Xcind pair. Each
successful write is to a disposable journal outside both repositories.
Every named journal is independent and starts empty except for its listed
events. `J-TCP` alone carries TCP/SNI through seq 1 record, seq 2 defer, and
seq 3 reconsider; targeting cases use their own journals, so they cannot
insert events into or renumber the TCP history.
Every write request also carries the required actor (`fixture-reviewer`),
fixed UTC `at` value for its own sequence (seq 1 =
`2026-10-01T00:00:01Z`, seq 2 = `2026-10-01T00:00:02Z`, seq 3 =
`2026-10-01T00:00:03Z`), non-empty rationale,
`expectedJournalRevision`, and the exact current `expectedEvidenceDigest`.
Record rationale is “Fixture only: record bounded evidence for owner review;
no semantic match or direction is asserted.” Decision rationale is “Fixture
only: exercise this requested state; it is not a real owner decision or
transfer.” Reconsider rationale is “Fixture only: exercise explicit review
after the declared trigger.” These are supplied inputs, not journal inferences.

| Sequence / input | Expected state, persistence, and exit | Counterexample / refusal |
| --- | --- | --- |
| In empty `J-TCP`, `record` `topic:tcp-sni` against baseline digest `sha256:b299310a771287426152781b60c52648002022080442696c0f03241ef390a168`, expected journal revision 0, request ID `r-tcp-record-001`, actor `fixture-reviewer`, fixed seq-1 time, and the record rationale above. | Exit **0**; event seq 1, journal revision 1; decision `pending-review`. A repeat identical scan says `unchanged`; `read` returns this event/state. `pending-review` is review work, not pending port work. | A read or scan with no journal has no write; do not auto-create an empty journal. |
| In `J-TCP`, fixture-only `decide` request `r-tcp-defer-001`, expected revision 1 and the same digest, changes the recorded candidate to `deferred`, with authority `{owner:"fixture-owner", status:"declared", authoritativeProjectId:"scind", permittedDirections:["left-to-right"]}`, trigger `{id:"fixture:scind-tcp-sni-active", kind:"condition", description:"Scind promotes TCP/SNI from Future to active design"}`, fixed seq-2 time, and decision rationale above. | Exit **0**; event seq 2; `deferred` and its trigger persist with the test-only authority. No trigger is polled. Repeated scans leave it deferred. | Missing trigger ID, kind, or description: exit **1**, `validation.reconcile-evidence-invalid`; no append. Changed bytes alone do not auto-reconsider. |
| In separate empty `J-RETAIN`, `record` the targeting candidate against baseline digest `sha256:2765ed93a8a5078ecc60cb983fde4d26b67ac95d04ba0fbd4bacaa635997f349` as `r-target-retain-record-001` at seq 1, then `decide` request `r-target-retain-001` at expected revision 1 as `divergence-retained`, with authority `{owner:"fixture-owner", status:"declared", authoritativeProjectId:"xcind", permittedDirections:["right-to-left"]}`, fixed seq-2 time, and decision rationale above. | Both operations exit **0**; the decide event is seq 2 and the projection is `divergence-retained`, distinct from `pending-review`, `deferred`, and `intent`. No source direction is executed. | Repeated scans do not convert it to intent/pending work. This targeting event is not part of `J-TCP`. |
| In `J-TCP`, after seq 2 and a fresh scan following the declared trigger fixture, request `r-tcp-reconsider-001` at expected revision 2 names `supersedesSeq: 2`, the trigger reference, fresh evidence, authority `{owner:"fixture-owner", status:"unresolved", authoritativeProjectId:null, permittedDirections:[]}`, replacement outcome `unresolved`, fixed seq-3 time, and reconsider rationale above. | Exit **0**; event seq 3; seq 2 remains in history and `unresolved` materializes. | Without explicit reconsider, the state remains deferred. A non-fresh scan or wrong superseded sequence refuses with exit **3** and leaves `J-TCP` unchanged. |
| In separate empty `J-DIRECTION`, `record` the targeting candidate and baseline digest at seq 1 as `r-target-direction-record-001`. With declared Xcind authority permitting only `right-to-left`, first submit otherwise-valid `intent` request `r-target-intent-reverse-001` for `left-to-right`, expected revision 1 and current baseline digest. | Exit **3**, `refusal.reconcile-authority-unresolved`; the refusal appends nothing and `J-DIRECTION` stays at revision 1. Then submit `r-target-intent-right-001` for `right-to-left` at the same expected revision/digest, with declared authority, fixed seq-2 time, actor and rationale above: exit **0**, event seq 2, projection `intent`/`right-to-left`. Intent does not edit either source. | The refused request cannot consume a sequence or make the following valid request stale. |
| In separate empty `J-UNRESOLVED`, first `record` the targeting candidate and baseline digest as `r-target-unresolved-record-001` at seq 1. Submit `r-target-intent-no-authority-001` at expected revision 1 and current digest with structurally valid authority `{owner:"fixture-owner", status:"unresolved", authoritativeProjectId:null, permittedDirections:[]}` and otherwise valid intent fields. | Exit **3**, `refusal.reconcile-authority-unresolved`; the journal remains revision 1. | Missing required authority fields or an out-of-pair `authoritativeProjectId` are separate invalid-request cases: exit **1**, `validation.reconcile-evidence-invalid`, no append. Do not classify malformed authority as an unresolved-authority refusal. |

## Cardinality and malformed-pair vectors

The synthetic pair has pair ID `fixture:lds-bds252:synthetic-split:v1`,
ordered project IDs `fixture-left`/`fixture-right`, disposable tracked source
files, and candidate key `fixture:route-decomposition`. Its independently
computed ID is listed above. The source-side fixture has one invented node
`SYNTH-S-01`; the right has two distinct invented nodes `SYNTH-X-01` and
`SYNTH-X-02`. The fixture `record` supplies `synthetic` provenance and
reviewed `split` correspondence (one left member, two right members); no
synthetic value is represented as Scind/Xcind data or a source-file hash.
The pair descriptor pins the actual commit IDs generated for these two
disposable fixture repositories; those commit IDs are setup outputs, not
source revisions or fixed expected constants.

The synthetic scan also requires a raw `--map` artifact tracked in the
synthetic right repository. This tiny fixture map is outside
`scope.rightPaths`, so it enters candidate evidence exactly once as a map
artifact; its contents are synthetic evidence, not a relationship fact:

| Map path | Exact UTF-8 bytes; `LF` denotes one line-feed byte | SHA-256 |
| --- | --- | --- |
| right `engineering/synthetic/review-map.txt` | `Fixture relationship hint (synthetic only)`, LF, `SYNTH-S-01 => SYNTH-X-01 + SYNTH-X-02`, LF | `925847331172c5e5fcd20116090f5f7219162128bb80f5dfaaf75c01000bbf67` |

The split vector's scan exits 0 with this `--map`; only the subsequent
explicit journal record asserts the reviewed one-to-many correspondence.

The disposable stand-in files use these exact UTF-8 bytes (including final
LF) and expected SHA-256 values:

| Fixture path | Exact bytes; `LF` denotes one line-feed byte | SHA-256 |
| --- | --- | --- |
| left `engineering/synthetic/route-model.md` | `# SYNTH-S-01\nRequires declared route and verified destination.\n` | `abdcbb890f6a49cc5fa2b97e9f97d68855af593e86d0e4bf4049e80547d8302d` |
| right `engineering/synthetic/route-declaration.md` | `# SYNTH-X-01\nDeclares operator-visible route.\n` | `a9fcf28638ddc43b9122fb54f764bd6feab099b694e6eef24d19bb8c33fbb462` |
| right `engineering/synthetic/destination-verification.md` | `# SYNTH-X-02\nVerifies destination reachability.\n` | `983d27fa13e6f90a134c5b31fbea6a8f5b8746cf769efdb483fb3a0e53daa8f8` |

| Pair/request shape | Expected result |
| --- | --- |
| Split with one left/two right; explicit reviewed record | Scan exit 0; record exit 0; one stable candidate ID and one correspondence preserving all three distinct member keys. |
| One tracked right-only path `engineering/decisions/0021-cross-repo-adr-cross-referencing.md`, deliberately outside every bucket | Scan reports the right-side path as unmatched evidence under the generated key `unmapped:right:engineering/decisions/0021-cross-repo-adr-cross-referencing.md` and the computed ID above. It does **not** assert a reviewed `right-unmatched` relationship; an explicit journal record with zero left members/one right member is needed. |
| Valid explicit `left-unmatched` (1:0) or `right-unmatched` (0:1) record | Exit 0; keep the present side member and empty opposite side exactly; do not invent a match. |
| Candidate key repeats, begins with reserved `unmapped:`, or a repo-relative path is listed in two candidate buckets | Exit **1**, `validation.reconcile-pair-invalid`; stdout has no success object; no journal change. |
| Candidate bucket has 0:0 members, unsupported many-to-many relation (2:2), or member outside declared scope | Exit **1**, `validation.reconcile-pair-invalid`; no reassignment, flattening, or write. |
| Same canonical tracked map supplied twice under two path spellings | Exit **1**, `validation.reconcile-pair-invalid`; no duplicate evidence rows. Distinct map files are valid and remain separately sorted. |

## Journal idempotency, staleness, and applied-proof boundaries

| Request and journal state | Expected result | Opposite boundary |
| --- | --- | --- |
| Event `r-tcp-record-001` is stored at seq 1. Retry that byte-equivalent validated request with the same request ID and consumed scan identity after later events advanced the journal and evidence changed. | Exit **0**, `replayed: true`, current materialized projection and current revision; append nothing. Duplicate-ID lookup occurs before current revision/evidence checks. | Reuse `r-tcp-record-001` with a changed rationale, payload, or consumed scan identity: exit **3**, `refusal.reconcile-journal-conflict`; bytes/revision unchanged. |
| New request ID expects journal revision 1 but current revision is 2. | Exit **3**, `refusal.reconcile-journal-conflict`; no event append. | Exact already-recorded request ID retry is the idempotent success above, not stale-revision failure. |
| In separate `J-STALE-EVIDENCE`, first record the targeting candidate at seq 1 with its baseline digest. A later scan has digest `E1 != E0`; a new `decide` or `reconsider` request at expected revision 1 presents stale `E0`. | Exit **3**, `refusal.reconcile-stale-evidence`; `J-STALE-EVIDENCE` stays at revision 1 and its bytes are unchanged. | Matching pair descriptor, fresh scan and current expected digest can proceed subject to authority and revision checks. |
| In separate `J-APPLIED-PROOF`, first record the targeting candidate at seq 1 with its baseline digest. Then submit an otherwise schema-valid `applied` outcome request with request ID, actor, fixed time, non-empty rationale, expected revision 1, current baseline evidence digest, and declared authority `{owner:"fixture-owner", status:"declared", authoritativeProjectId:"xcind", permittedDirections:["right-to-left"]}` permitting the requested direction, but omit `appliedProof`. | Exit **3**, `refusal.reconcile-applied-proof-required`; no applied event; journal remains revision 1. The sole invalid/missing condition is proof. | Supplying only a map label, matching source hash, intent, or journal assertion is still not proof. A valid proof requires a real externally verifiable transfer artifact with URI, SHA-256, source/destination projects and revisions, transferred scope, verifier, verification time and rationale. This fixture fabricates no such artifact and claims no applied result. |

For all refusals above, preserve existing journal bytes exactly. Under the
contract, verb-raised failures have empty stdout and a stable JSON error
envelope on stderr when inherited `--json` is active. Exit 1 is invalid input,
2 is CLI usage/argument error, 3 is safety/staleness/conflict refusal, and 4
is unexpected I/O/internal failure. A failure to replace the journal before
rename leaves old bytes unchanged; a directory-flush failure after rename is
indeterminate and must be resolved by reading revision/request ID, not blind
retry. These are expected behaviors for later executable checks, not results
observed here.

## Explicit exclusions

The source-derived rows are bounded by the cited Stage 1 pins: Scind runtime
remains unverified; Xcind's external-proxy map classification remains
internally contradictory; TCP/SNI remains a Future-versus-rejected example,
not an observed Scind implementation gap. The split is wholly synthetic.
No scanner or journal was implemented or executed, no live pair authority was
selected, no source repository was written, and no pilot or transfer is
claimed.
