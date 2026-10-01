In Claude Code, use this skill to inspect the LDS CLI manifest or diagnose
its generated skill installation. `lds plumbing adopt` only displays a
preparation document; it does not start adoption or authorize target actions.
The listed documentation commands are deterministic operations for an
explicitly named target, not an adoption workflow; use them only for an
explicitly requested operation and never infer owner approval from supplied
facts or a plan digest. Do not execute an adoption flow from the document.
