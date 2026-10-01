In Claude Code, use this skill to inspect the LDS CLI manifest or diagnose
its generated skill installation. `lds plumbing adopt` only displays a
live adoption flow; it does not start adoption or authorize target actions.
The listed documentation commands are deterministic operations for an
explicitly named target. Root `lds adopt` is an LLM-kind interactive CLI
form and is not projected into this plumbing-only skill; do not invoke it as
a substitute for a fresh human review. Never infer owner approval from
supplied facts, model output, or a plan digest. Do not execute an adoption
flow from this skill; S10 must independently establish any later CLI/skill
parity claim.
