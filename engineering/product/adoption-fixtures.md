# Adoption fixture cases

`internal/asset/testdata/adoption/greenfield/` is a disposable greenfield
target: `target/` contains only a word-count shell program and its input, with
no starting documentation. `facts.json` records target-specific facts and a
simulated fixture-owner approval explicitly marked test-only; it is not
authorization to modify a real project. The operation is
`sh word-count.sh count sample.txt`, requiring POSIX-compatible `sh` and
`wc`; the fixture test runs it and checks exact stdout and exit status.

`expected/` is independently authored expected content for the four files in
the adoption contract, not generated from the templates. `case.json` records
the first-run four-file create plan and only necessary parent directories,
then expects all four files to be `identical-untouched` on the second run with
zero writes and zero changed bytes. The test compares template rendering to
the expectations; it does not implement adoption or write to a project tree.

This is fixture evidence only, not a named pilot or real-project usability
validation. The separate testdata layout is intended to allow later
preservation/collision cases without modifying this greenfield baseline.
