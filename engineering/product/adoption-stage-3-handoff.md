# BDS-249 Stage 3 Step 9: Review and BDS-250 handoff

## Review result

The transferred archive matched SHA-256
`5805a6e82ea9ff2b42db76ff7a219e514d0df9a5c4902df0eab61d063fefe581` and
contained 32 paths; those files were extracted over the `origin/main` checkout.
The accepted behavior is recorded in [the adoption contract](adoption-contract.md).
No material inconsistency was found between that contract, the four template
assets, and the greenfield/preservation fixture expectations.

Verified in this checkout:

- `nix develop --command go test ./internal/asset ./internal/manifest ./internal/cli` passed all three packages.
- The greenfield test executes `sh word-count.sh count sample.txt`, checks exact stdout, empty stderr, and exit success, then compares renders of all four assets with independently authored expected files.
- Asset tests cover override, installed-default, and embedded resolution for each adoption asset; the CLI test checks each asset appears once in the JSON manifest with its byte-derived SHA-256.
- The preservation test checks eight declared scenarios, fixture source hashes, classifications, proposed-create/veto metadata, affected links, and the bytes/hashes named as preserved.

The Step 8 high-mode PASS is fixture-level evidence only. In particular,
`adoption_preservation_test.go` never invokes an adoption workflow or mutates a
target: preserved bytes are fixture inputs checked against their declared
hashes, not before/after observations from a runtime attempt. The greenfield
first-run and rerun write counts are also fixture plans. Neither set proves
executable adoption, confirmation behavior, atomicity, or a transactional
filesystem guarantee.

## Deferred real-pilot gate

The accepted choice is **1B: defer naming a real pilot**. No real repository
was selected or authorized for modification. Fixture facts and the Wordcount
operation are test-only evidence; they do not validate real-project facts,
usability, or adoption. BDS-250 must not treat another repository as an
implicit pilot; naming and authorization remain a separate gate.

## BDS-250 must prove at runtime

1. Require a complete, target-specific, owner-approved fact bundle and
   provenance for the name, purpose, current capability, and verified user
   operation. Missing or unverified facts must refuse before writes.
2. Resolve and render the four approved assets for those facts, with an
   explicit safe escaping policy for Markdown/template values. Verify asset
   override/default/embedded behavior in the actual runtime path; the current
   fixtures use ordinary values and do not test escaping.
3. Present a reviewable preflight for every candidate (`create`,
   `identical-untouched`, `different-collision`, or `refusal`), including
   affected existing content. Require human confirmation; declining or any
   collision/refusal must produce no creates.
4. Create only the four approved files and only necessary parent directories.
   Exercise partially occupied output sets, missing-output partial trees,
   root README and unrelated-file preservation, and the all-create veto.
   Legacy, ambiguous, and partial LDS under `docs/` must refuse without
   migration; `docs/reference/` alone is not conclusive. Unsafe symlinks and
   unreadable occupied paths must refuse.
5. Run an identical adoption twice against a disposable target and establish
   that the second run performs zero writes and leaves bytes unchanged.
6. For preservation, snapshot real target bytes before each relevant runtime
   case and compare them afterward (byte comparison plus hashes), including
   unrelated files, root README, collisions, and edited generated-looking
   content. Keep this distinct from any claim of crash-safe atomicity, which
   the accepted contract does not guarantee.

The repository currently provides the accepted contract, assets, and fixture
evidence—not an adoption command or workflow. The runtime claims above remain
open for BDS-250.
