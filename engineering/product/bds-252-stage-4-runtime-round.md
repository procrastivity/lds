# BDS-252 Stage 4 — bounded Linux runtime round

This records actual CLI behavior for Step 8; it does not change or extend the
accepted scanner/journal contract. The run was built from a disposable ordinary
clone at `origin/main` `3c9af08c10d2eb8c004ec5ca1a674f4e2632cb4e`, with the
supplied root-delta archive (`bds252-step8-root-baseline.patch.gz`, SHA-256
`6460a80ce540e1f7ad276ebc2c721b094298c20e10821a2fe22cc6afe498a0e3`)
applied. Its uncompressed patch SHA-256 is
`264c54eae2725978f8c8c3e30180ad50d3a085a429b893f17973a9cd5db16325`.
The accepted contract SHA-256 is
`f721bc58a7c6fd8eb3ff0ac87f3708c1bb752a4f6a2b6bb6264fafb5d4f3f587`; the
Step 5 fixture-expectations SHA-256 is
`1514b633f8925e2bbe1f51392c761ab3c402cb8abee3581d9d53b8cfc6fe7fa1`.

## Executed results

The [machine-readable execution evidence](bds-252-stage-4-runtime-evidence.json)
contains 34 invocations, complete stdout/stderr/exit results, invocation
arguments, pair/request/saved-scan inputs and hashes, and before/after source
snapshots. Every scanner/journal invocation preserved tracked source-file bytes
and Git index bytes (the evidence records aggregate tracked-content and exact
index SHA-256, HEAD, and status before and after). Intentional fixture setup
mutations are separately recorded with before/after commit, tree and index
snapshots. No source/reference checkout was written; the only changed content
was committed in disposable Xcind clones and synthetic fixture repositories.

At Scind `c63cbc686afd2ca6ecc01eab7825d637d49378c7` and Xcind
`c244856e7119547ed6dceb2b7663a01720bdde66`, all nine pinned source/map
artifact hashes matched the independent Step 5 expectations. The four runtime
candidate IDs and all four baseline `evidenceDigest` values matched exactly.
Candidate and evidence rows sorted as specified, and a repeated baseline scan
was byte-identical even when the two `--map` arguments were reversed. The
synthetic files and raw map matched all four expected hashes; its
`fixture:route-decomposition` ID matched
`cand-v1-aed9fd6fc71e2ccff638d2e91a338b80ba6761a929d4b44a67dd26d242d81af6`.

The runtime journal observations were:

- `pending-review` was recorded and read; the isolated TCP/SNI journal advanced
  to `deferred`. An isolated external-proxy candidate was explicitly recorded
  as `divergence-retained`.
- With fixture authority permitting only `right-to-left`, a `left-to-right`
  intent was refused with exit 3 / `refusal.reconcile-authority-unresolved`
  and unchanged journal bytes; the permitted reverse direction then succeeded.
- Exact request retry returned success with `replayed: true` and no journal
  byte change. Reusing its ID with changed request content refused with exit 3 /
  `refusal.reconcile-journal-conflict`, leaving bytes unchanged.
- An otherwise-valid applied request with authority, permitted direction,
  current revision/digest and all non-proof fields, but no proof, refused with
  exit 3 / `refusal.reconcile-applied-proof-required`; journal bytes were
  unchanged. A stale-digest decision likewise refused with exit 3 /
  `refusal.reconcile-stale-evidence` without a journal write.
- After the baseline targeting observation and a fixture-only deferred
  decision, the disposable Xcind clone received only the expected appended
  comment to ADR-0023 (content SHA-256
  `69ceadc9e13d55f1de1d2bc8cfa090f1f137ea4aa1931944472b6fbec22abb4f`) and
  a new pin. The rescan retained its candidate ID, changed its digest/status,
  and displayed the older deferred decision basis. An explicit observation
  preserved that decision; explicit reconsideration then superseded it as
  unresolved. In a separate clone, appending the expected comment to the raw
  JSON map (SHA-256
  `9fc37f1cfd3804809b6a0bdded002560be3f2c19621dc76bf32219ce37f8bc34`) also
  changed digests while the prior decision and its older evidence basis
  remained visible.
- The synthetic 1:2 correspondence was explicitly recorded/read as `split`.
  The source-derived right-only ADR-0021 was separately recorded/read as
  `right-unmatched`; no map label was interpreted as a relationship.

## Commands, scope and limits

Built the binary using `nix develop --command go build -trimpath -o lds
./cmd/lds` (Go 1.24.10, Linux/amd64). Checks passed:

- `nix develop --command go test ./internal/reconcile ./internal/cli`
- `nix develop --command go test ./...`
- The 34-command runtime round; every recorded expected exit, state, baseline
  constant, and source/index-preservation assertion passed.

`git apply --check` reported the repository-policy-preserved upstream vendor
whitespace warning at patch line 22006 (trailing whitespace / new blank line
at EOF); the patch was applied without editing or normalizing that upstream
content. No code, contract, source repository, reference checkout, or
configuration was changed for this evidence round.

This is Linux runtime evidence only; Darwin is build-only, and Windows is
unsupported/not claimed. Authority and outcomes are fixture-only. No real
owner authority, proof authentication, port behavior, pilot, runtime parity,
or BDS-251 workflow delivery is established.
