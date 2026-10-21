# LDS user guide

LDS inventories and validates a bounded documentation layer, can propose four
create-only adoption files for a project, and scans explicitly pinned pairs of
Git projects for reconciliation evidence. It also collects maintenance evidence
and reviewer-authored proposals in a report without editing the target.
Use an existing local CLI build or,
with Nix installed, build from an LDS checkout:

```sh
nix build .#default
./result/bin/lds --help
export PATH="$PWD/result/bin:$PATH"
```

The final line makes `lds` available for the examples in your current shell.
This is a checkout build, not a published download or release-install
instruction. You do not need an agent harness or provider key for the offline
tutorial or deterministic plumbing.

## Choose a route

- [Inspect a disposable project](tutorials/inspect-a-project.md) — a read-only,
  offline first run.
- [Adopt documentation in a project](how-to/adopt-a-project.md) — prepare facts,
  inspect the complete plan, and make a separate human decision.
- [Assess documentation](how-to/assess-documentation.md) — collect evidence,
  proposed text and questions outside the project; no automatic edits.
- [Reconcile two projects](how-to/reconcile-projects.md) — scan clean pinned
  snapshots and keep an explicit review journal.
- [Command reference](reference/commands.md) and [fact bundle reference](reference/facts.md).
- [Safety and scope](explanation/safety-and-scope.md) — what the checks and
  write boundaries do *not* guarantee.

## Optional agent-harness projection

`lds plumbing manifest --json` describes the CLI; only eligible plumbing
verbs are projected into generated skills. `lds install` projects into
detected harnesses, or `lds install claude-code` targets Claude Code alone.
`lds plumbing doctor --json` reports target state and findings;
`lds uninstall claude-code` removes a stamped installation. Install refuses
hand-edited, unstamped, or incompatible targets by default; inspect before
considering its destructive `--force` option. Installation is separate from
adopting a project. The generated `/lds` guidance reads the live flow via
`lds plumbing adopt --json`, but that reader is preparatory, not an adoption
execution. Installing or discovering a skill does not establish native
execution or parity with the root CLI.
