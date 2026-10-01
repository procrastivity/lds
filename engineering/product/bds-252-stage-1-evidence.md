# BDS-252 Stage 1 — bounded Scind/Xcind evidence

This is a snapshot-bound evidence table for Stage 1, not a reconciliation
decision, live pilot, or claim that LDS can infer semantic equivalence. It
checks two bounded topics against pinned source material. A matching document
claim does not establish runtime behavior; only the inspected code supports
the as-built observations below.

## Scope and trust boundary

| Snapshot | Pinned revision | Checkout / evidence boundary |
| --- | --- | --- |
| Scind | [`c63cbc686afd2ca6ecc01eab7825d637d49378c7`](https://github.com/scinddev/scind/tree/c63cbc686afd2ca6ecc01eab7825d637d49378c7) | Clean checkout; 89 tracked files. The pinned tree has `docs/` and `engineering/` but no `cmd/`, `internal/`, `src/`, or `bin/` implementation paths. Scind claims below are verified as accepted engineering text only; corresponding runtime behavior is unverified and must not be trusted as as-built evidence. |
| Xcind | [`c244856e7119547ed6dceb2b7663a01720bdde66`](https://github.com/scinddev/xcind/tree/c244856e7119547ed6dceb2b7663a01720bdde66) | Clean checkout. Engineering claims are treated as bounded only where checked against the listed `bin/` or `lib/` bytes. This is not a complete Xcind self-sync or a full-map rebuild. |

The LDS [source-repository map](../architecture/source-repositories.md) treats
these projects as references with different roles, not interchangeable
authorities. The [roadmap](./roadmap.md) says cross-project reconciliation is
separate from within-project maintenance and that Xcind's ledger is evidence,
not LDS's schema. BDS-252 leaves journal ownership/schema, pair direction,
confidence, non-1:1 matching, and unresolved/deferred/divergence semantics
open. Nothing in this table settles those choices or authorizes a source edit.

## Bounded topic evidence

Hashes below are SHA-256 of the exact repository-relative file bytes at the
pinned commit. Code references name the relevant functions/sections; the
content hashes pin the full files, not just those excerpts.

| Topic and evidence paths (SHA-256) | Engineering claim checked | As-built observation and bounded conclusion |
| --- | --- | --- |
| **External proxy mode.** Scind: [`engineering/decisions/0017-external-proxy-mode.md`](https://github.com/scinddev/scind/blob/c63cbc686afd2ca6ecc01eab7825d637d49378c7/engineering/decisions/0017-external-proxy-mode.md) `9557cbfde3718cc23918e20effb76498eb788b1630e93396b49c00fddad1b7cc`; [`engineering/specs/proxy-infrastructure.md`](https://github.com/scinddev/scind/blob/c63cbc686afd2ca6ecc01eab7825d637d49378c7/engineering/specs/proxy-infrastructure.md) `3218e843722dc6d4d10317515ce7c7abe9b413064e5738b137d42ca771cdf8da`. Xcind: [`engineering/decisions/0022-external-proxy-mode.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/decisions/0022-external-proxy-mode.md) `db5d802fb20b687829d46e5438334f00c33b3821ca4cb7cff3c3ef2d51377723`; [`engineering/specs/proxy-infrastructure.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/specs/proxy-infrastructure.md) `1a809b29235b9868a8978c3cd57b59ebd4f0fad815eef1254b4c726a5ab3195a`; [`bin/xcind-proxy`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/bin/xcind-proxy) `37718dcbbf11cffdca3217d858f60570a063d22bb0b008e59a3869628f45c4c4`; [`lib/xcind/xcind-proxy-lib.bash`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/lib/xcind/xcind-proxy-lib.bash) `b4dda08ce0427064d535bbc5d40f05325512cd3b94ddde8839850e24d19a0ee7`. | Both decision records describe a managed/external distinction; the external proxy owns TLS termination while applications use its shared network and configured routing labels. Xcind's spec describes `up` as verifying the shared network, refuses `up --force`, and reports external status. | Xcind code confirms `managed|external` validation and rejects `tls-mode custom` in external mode (`bin/xcind-proxy`, `init` validation). For external `up`, `bin/xcind-proxy:298–306` calls `__xcind-proxy-ensure-network` and then reports that nothing is started; `lib/xcind/xcind-proxy-lib.bash:58–74` creates the Docker network when it is absent. Thus as-built behavior is **ensure network (which may create it), but do not start the external proxy**; this is more specific than the spec's “verifies the network” wording. `up --force` is refused; external status emits `mode`, network, entrypoints and resolver; the library bypasses certificate provisioning and uses configured entrypoint/certresolver values when rendering labels. This supports those specific Xcind behaviors, not every table row or runtime integration. Scind's parallel behavior is **unverified as-built**: its pinned snapshot supplies engineering documentation but no implementation path. Do not treat the pair as proof of implementation parity. |
| **Workspace targeting.** Scind: [`engineering/decisions/0011-options-based-targeting.md`](https://github.com/scinddev/scind/blob/c63cbc686afd2ca6ecc01eab7825d637d49378c7/engineering/decisions/0011-options-based-targeting.md) `92416f10e61b7afb891eac7cc6b1e8a9e8960b635d842b059cc9a86eef48bb73`. Xcind: [`engineering/decisions/0023-location-based-targeting.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/decisions/0023-location-based-targeting.md) `88d2174b069096a01df7d0476302e42fa3ce6511352b3df8b592c27af794448d`; [`engineering/specs/context-detection.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/specs/context-detection.md) `4c4bf1a6007363ac4d5b9710af279b08e946d8d550235ec142e8e54d59d68210`; [`engineering/specs/workspace-lifecycle.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/specs/workspace-lifecycle.md) `8359128872c75e7fdecab054b77dbe3c1eda81b99ec92dcb728d9160818b178a`; [`bin/xcind-workspace`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/bin/xcind-workspace) `a57336131d2b376e379acc4fcd162a8b022d474d6495a1b2335b08c7a6500b26`; [`lib/xcind/xcind-workspace-lib.bash`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/lib/xcind/xcind-workspace-lib.bash) `e5afb3af452eee16881f48ec86f58101f886f863e1e80e239934ecb86ccdd00d`. | Scind ADR-0011 proposes `--workspace` and `--app` options plus current-directory context detection. Xcind ADR-0023 rejects name-based workspace targeting; its clarification says repeatable `-a/--app` only filters apps inside an already located workspace. | Xcind `__xcind-workspace-find-root` accepts a directory and walks parents for a declared workspace (`bin/xcind-workspace`, lines 203–245); `__xcind-workspace-updown` parses `-a/--app`, enumerates app directories under that root, filters by basename, and errors on an unknown app (lines 1090 onward, 1187–1225). This corroborates location-based workspace resolution plus app filtering, not registry-based name-to-location targeting. Scind's proposed behavior is **unverified as-built** for the same no-implementation reason above. The filter's precise effects are bounded to these `up/down/restart` code paths. |

The Xcind observations are source inspection, not executed behavior tests in
this task. Claims outside the named symbols, command paths, and pinned files
remain unverified. Engineering prose, correspondence labels, and candidate
relationships are not proof that one project's decision should be ported to
the other.

## BDS-119 stale-map claim: before and pinned-after

| Evidence state | What the evidence says | Trust treatment |
| --- | --- | --- |
| **Before — issue's reported baseline** | [BDS-119](https://linear.app/beausimensen/issue/BDS-119/correspondence-map-is-stale-missing-adrs-00210023-contradicts-adr) reports an audit scoped from Xcind `9f975e4` and says the then-current map omitted ADR-0022/0023 and contradicted ADR-0023. That is a historical claim about that map revision, not the pinned current snapshot. | Trust as a record of what was reported and the stated baseline; do not reuse it as a description of current rows. |
| **After — pinned Xcind `c244856e7119547ed6dceb2b7663a01720bdde66`** | The specific missing-row allegation is stale: `correspondence-map.md` contains the targeting pair Scind 0011 ↔ Xcind 0023 and external-proxy pair Scind 0017 ↔ Xcind 0022; its Round-2 delta says targeting changed from SCIND-ONLY and external mode was promoted. However, the file-level external-proxy row at `correspondence-map.md:100` still labels ADR-0022 `XCIND-ONLY`; JSON row 38 has `scind: null` and `rel: XCIND-ONLY`. This conflicts with the topic-level table at `correspondence-map.md:227`, which pairs Scind 0017 and Xcind 0022 as `MATCH`, and with the pinned Scind ADR-0017. The map header's decision-count parenthetical also says only 0019/0020 were newly included. | ADR-0022/0023 are no longer absent from the current map, but its external-proxy classification is internally inconsistent across the file-level row, JSON and topic-level table; treat that classification as unresolved/untrusted. The old header parenthetical is stale metadata too. Do **not** conclude from these spot checks that the full correspondence map is complete or that every row is fresh: the standing procedure requires a full Xcind self-sync and regenerated-map diff, neither performed here. |

Pinned map evidence: Xcind
[`engineering/sync/artifacts/correspondence-map.md`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/sync/artifacts/correspondence-map.md)
SHA-256 `3c121e241bd963f8cbe3792a80ac09dd8c500ed52100f228cab4d55ab6932c05`;
[`correspondence-map.json`](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/sync/artifacts/correspondence-map.json)
SHA-256 `175d69f1a499e512c571b7cb83476ca25a9dd814256d6e64447a9d29e1262c17`.
The [standing Xcind procedure](https://github.com/scinddev/xcind/blob/c244856e7119547ed6dceb2b7663a01720bdde66/engineering/maintenance/cross-project-sync.md)
requires a green Xcind self-sync for compared areas, a rebuilt map diff, and
explicitly says unverified areas must be marked untrusted. This spot check
does not satisfy that whole procedure.

## Bounded conclusion

Trusted here: the pinned document/code bytes and the limited Xcind code
observations described above; the existence of the listed map rows at the
current Xcind pin; and BDS-119 as a historical report. Untrusted/unverified:
Scind runtime behavior, all behavior outside the inspected Xcind code, the
freshness/completeness of the full map, and any inferred port/backport,
authority, or semantic-equivalence decision. No source repository was edited,
no reconciliation was run, and no live pilot or runtime execution is claimed.
