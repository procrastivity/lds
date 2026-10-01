# BDS-252 Stage 1 Step 2 — expected-case cards

**Purpose:** These are independently authored, asymmetric owner-review cases,
not scanner output, a reconciliation decision, or a proposed journal schema.
They expose evidence and uncertainty that a future decision model must retain.

The source snapshot boundary and Stage 1 findings are recorded in the
[bounded evidence note](bds-252-stage-1-evidence.md) (SHA-256
`0884630e811209b1c46b2bb1af7e747e8a5d91527cba885a50163838765a74bd`). The
checked Scind tree was clean at `c63cbc686afd2ca6ecc01eab7825d637d49378c7`;
the checked Xcind tree was clean at
`c244856e7119547ed6dceb2b7663a01720bdde66`. Every source-file hash below is
SHA-256 of the complete repository-relative bytes at that pin, not a hash of
an excerpt. No source repository was modified or synchronized for these cases.

The [live BDS-252 issue](https://linear.app/beausimensen/issue/BDS-252/specify-and-prove-lds-cross-project-reconciliation)
leaves authority, candidate identity/cardinality, state meanings, and
reconsideration open. [BDS-119](https://linear.app/beausimensen/issue/BDS-119/correspondence-map-is-stale-missing-adrs-00210023-contradicts-adr)
is treated as a historical report about its stated earlier baseline, not as
current map evidence. Xcind's
[ADR-0021](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/decisions/0021-cross-repo-adr-cross-referencing.md)
joins ADRs by topic, but that Xcind policy is evidence about Xcind's process,
not an LDS authority rule (SHA-256
`619d180ff6eeba7d8c6924f11850d250d019f19116bfe05aba9cc485c61b482a`). Its standing
[cross-project procedure](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/maintenance/cross-project-sync.md)
also requires a green Xcind self-sync and a regenerated map diff (SHA-256
`33cb188fc9a91d4cfe9ab8cbfa003c3f7712cc82c5898e311e5e561a719c6a87`); neither
was run for this bounded exercise.

## Case 1 — same topic, opposite targeting decisions

**Shape:** source-derived 1:1 candidate. **Expected treatment:** `DIVERGED`
by topic, not `SCIND-ONLY`; no direction to port is implied.

| Evidence at pinned revision | SHA-256 |
| --- | --- |
| Scind [`engineering/decisions/0011-options-based-targeting.md`](https://github.com/scinddev/scind/blob/c63cbc686afd2ca6ecc01eab7825d637d49378c7/engineering/decisions/0011-options-based-targeting.md) (`c63cbc686afd2ca6ecc01eab7825d637d49378c7`) | `92416f10e61b7afb891eac7cc6b1e8a9e8960b635d842b059cc9a86eef48bb73` |
| Xcind [`engineering/decisions/0023-location-based-targeting.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/decisions/0023-location-based-targeting.md) (`c244856e7119547ed6dceb2b7663a01720bdde66`) | `88d2174b069096a01df7d0476302e42fa3ce6511352b3df8b592c27af794448d` |
| Xcind [`engineering/sync/artifacts/correspondence-map.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/sync/artifacts/correspondence-map.md) | `3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05` |
| Xcind [`bin/xcind-workspace`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/bin/xcind-workspace) and [`lib/xcind/xcind-workspace-lib.bash`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/lib/xcind/xcind-workspace-lib.bash) | `a57336131d2b376e379acc4fcd162a8b022d474d6495a1b2335b08c7a6500b26`; `e5afb3af452eee16881f48ec86f58101f886f863e1e80e239934ecb86ccdd00d` |

Scind ADR-0011 accepts `--workspace` and `--app` options with current-directory
context detection. Xcind ADR-0023 rejects name-to-location flags and resolves
targets by location; its `-a/--app` clarification filters applications only
after the workspace has been found. The pinned map's file row and topic table
pair Scind 0011 with Xcind 0023 as `DIVERGED` (map lines 88 and 215). This is
the same topic with opposite targeting decisions, not a missing Xcind record.
It also means BDS-119's specific earlier “ADR-0023 absent” allegation is stale
at this pin.

**Uncertainty and owner prompt:** The Xcind code paths corroborate location
resolution and app filtering for the named commands. Scind's pinned tree has
no implementation paths, so its accepted ADR is not as-built runtime proof.
The pair does not decide which model should govern either project or whether
any change is wanted. What authority/direction should an LDS owner record for
an explicit 1:1 decision conflict?

## Case 2 — credited promotion, contradictory map classification

**Shape:** source-derived 1:1 candidate. **Expected treatment:** do not create
a new pending port for an outcome the cited documents credit as promoted;
classify the map's relationship evidence as `UNRESOLVED` pending human
adjudication.

| Evidence at pinned revision | SHA-256 |
| --- | --- |
| Scind [`engineering/decisions/0017-external-proxy-mode.md`](https://github.com/scinddev/scind/blob/c63cbc686afd2ca6ecc01eab7825d637d49378c7/engineering/decisions/0017-external-proxy-mode.md) (`c63cbc686afd2ca6ecc01eab7825d637d49378c7`) | `9557cbfde3718cc23918e20effb76498eb788b1630e93396b49c00fddad1b7cc` |
| Xcind [`engineering/decisions/0022-external-proxy-mode.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/decisions/0022-external-proxy-mode.md) (`c244856e7119547ed6dceb2b7663a01720bdde66`) | `db5d802fb20b687829d46e5438334f00c33b3821ca4cb7cff3c3ef2d51377723` |
| Xcind [`engineering/decisions/0021-cross-repo-adr-cross-referencing.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/decisions/0021-cross-repo-adr-cross-referencing.md) | `619d180ff6eeba7d8c6924f11850d250d019f19116bfe05aba9cc485c61b482a` |
| Xcind [`engineering/sync/artifacts/correspondence-map.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/sync/artifacts/correspondence-map.md) and [`correspondence-map.json`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/sync/artifacts/correspondence-map.json) | `3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05`; `175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17` |

Scind ADR-0017 credits the Xcind proof-of-concept; the Xcind map's Round-2
delta says ADR-0022 was promoted to Scind ADR-0017 under RL-116, and its
topic-level table says `MATCH` (line 227). That is documentary credit for an
already-applied promotion, not a new pending port. But the same pinned map's
file-level row 100 says `XCIND-ONLY`, and the JSON record has `scind: null`
and `rel: XCIND-ONLY`, even while its note says “Promoted to Scind as
ADR-0017 (RL-116).” The mapping evidence is therefore internally inconsistent
and must remain unresolved; neither the `MATCH` row nor the stale one-sided
row alone settles it.

**Uncertainty and owner prompt:** The cited records credit an applied
documentary promotion, but this exercise did not rebuild the map, inspect the
promotion's full history, or establish runtime parity. Keep “credited as
applied” distinct from “independently verified as built,” and do not replay it
as pending work. What evidence should resolve conflicting map projections,
and what threshold distinguishes a recorded promotion from a proven transfer?

## Case 3 — accepted deferment, not a shipped gap

**Shape:** source-derived scope divergence. **Expected treatment:** retain as
an accepted/deferred divergence example, not `SCIND-ONLY` pending work or a
claim of a shipped capability gap.

| Evidence at pinned revision | SHA-256 |
| --- | --- |
| Scind [`engineering/product/roadmap.md`](https://github.com/scinddev/scind/blob/c63cbc686afd2ca6ecc01eab7825d637d49378c7/engineering/product/roadmap.md) | `989aa9177981653fcd263dc29d7f63120b22462132e7bbd555443dfed2ac4fd0` |
| Scind [`engineering/specs/port-types.md`](https://github.com/scinddev/scind/blob/c63cbc686afd2ca6ecc01eab7825d637d49378c7/engineering/specs/port-types.md) | `9f002cb97b8fcb530a488b34a258e9994b41c60ae62b455e135f35d60622ded3` |
| Xcind [`engineering/sync/divergence/0030-port-type-plugins-tcp-sni.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/sync/divergence/0030-port-type-plugins-tcp-sni.md) | `0fac817abd1cb0a18f81746da10f7039cb2fa6438c5684b711d3924e36aefa8c` |
| Xcind [`engineering/product/roadmap.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/product/roadmap.md), [`lib/xcind/xcind-proxy-lib.bash`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/lib/xcind/xcind-proxy-lib.bash), and [`test/test-xcind-proxy.sh`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/test/test-xcind-proxy.sh) | `7e75229a97a909f5c7329b51e6c1c329764ccb0b326ebfef434ed3a147cdb7f4`; `b4dda08ce0427064d535bbc5d40f05325512cd3b94ddde8839850e24d19a0ee7`; `93078ede2a5abb5e0ba39999854c5e18f18ff4ee19ca71243923613dd8f2adf2` |
| Xcind [`engineering/maintenance/cross-project-sync.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/maintenance/cross-project-sync.md) | `33cb188fc9a91d4cfe9ab8cbfa003c3f7712cc82c5898e311e5e561a719c6a87` |

Scind's roadmap keeps port-type plugins and TCP/SNI routing marked **Future**.
Xcind divergence 0030 is `Active`, calls Xcind's rejection a scope boundary,
and records `SURVIVES-AS-DIVERGENCE`; it says Scind should keep its future
intent. Its explicit revisit trigger is: if Scind promotes TCP/SNI from
Future to active design, re-evaluate whether the difference remains a
divergence or becomes a genuine gap, and re-audit each round.

**Uncertainty and owner prompt:** The Xcind divergence record describes the
rejection as test-enforced. The pinned parser accepts only `proxied` or
`assigned` types, and its test rejects `type=bogus` (test lines 732–735); that
is bounded evidence for the type vocabulary, not a direct TCP/SNI-specific
negative test or a complete runtime audit. Scind's roadmap is intent, not a
shipped feature. Treat the source record and trigger as a candidate policy
example, not an LDS decision. What evidence should convert an accepted defer
to a pending gap, and should the revisit trigger be event-based, round-based,
or both?

## Case 4 — [SYNTHETIC] one-to-many decomposition

**Shape:** deliberately invented 1:n relationship; it is not a Scind/Xcind
source fact. Snapshot context is Scind `c63cbc686afd2ca6ecc01eab7825d637d49378c7`
and Xcind `c244856e7119547ed6dceb2b7663a01720bdde66`, but the synthetic IDs
below have no repository path or source-file hash.

- `SYNTH-S-01`: one hypothetical source concept requiring an operator-visible
  route declaration and a verifiable destination.
- `SYNTH-X-01`: one hypothetical target concept describing route declaration.
- `SYNTH-X-02`: a separate hypothetical target concept describing destination
  verification.

**Expected treatment:** keep one source node and two target nodes with an
explicit 1:n candidate relation. Do not duplicate the source node, collapse
the targets into a false 1:1 `MATCH`, or infer that either target is pending
port work. Ask the owner whether this is a genuine concept split, a partial
mapping plus an unmatched target, or no correspondence. This card exists to
exercise cardinality and stable candidate identity only; the names and content
are fabricated and must never enter source history or a real correspondence
map.

## Owner-choice hooks (all open)

These D1–D5 labels follow the accepted BDS-252 planner. The cards supply
examples for those gate questions; they do not settle the choices.

| Hook | Question the cards expose | Cases |
| --- | --- | --- |
| **D1 — journal owner and location** | Which LDS owner is responsible for the review journal, and where is it maintained? The cards leave both ownership and location open. | 1–4 |
| **D2 — pair/topic, direction, and authority** | What defines a candidate pair/topic, and who may decide authority and port/backport direction when source decisions conflict? Do not import Xcind's “Scind is canon” procedure as a universal LDS rule. | 1–3 |
| **D3 — reviewed correspondence and cardinality** | Which candidate relations require owner review, and how should 1:1, 1:n, and unmatched items be represented without treating a candidate as a decision? | 1, 2, 4 |
| **D4 — state, revisit, and applied proof** | How should divergence, unresolved map evidence, accepted deferment, pending intent, and credited/applied work be distinguished; what evidence or trigger changes those states? | 1–3 |
| **D5 — deterministic scan and journal interface/storage** | What stable scan output does the journal consume and store, and how are reviewed decisions preserved or reconciled across repeated scans? | 1–4 |

No card authorizes a source edit, selects a real pair's authority, or settles
journal ownership/location, scan-journal interface/storage, schema, or status
vocabulary.
