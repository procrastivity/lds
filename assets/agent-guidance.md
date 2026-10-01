This installation exposes the Toolsmith scaffold's version, install, and
uninstall root commands, plus canonical `lds plumbing manifest` and
`lds plumbing doctor`. The root `lds manifest` and `lds doctor` spellings
remain compatibility aliases for already-installed skills and existing
callers. Consult `lds plumbing manifest --json` for the current command
surface.

`lds plumbing adopt` reads the live preparation-only flow document; it does
not inspect a target, start a workflow, request approval, or write files.
Its presence in the projected verb list is not permission to execute the
flow. Paired execution remains unavailable until the provider and live-access
gates are separately resolved.

The `lds plumbing docs inventory|validate|render|plan|apply` commands provide
deterministic documentation operations for an explicitly named target; they
are substrate, not a human-facing adoption workflow. No LLM workflow or
`lds adopt` command is implemented. Do not infer owner approval
from facts or a digest: show the whole plan and obtain a fresh decision before
invoking `apply` with its exact reviewed digest. Installation writes generated
agent skills; it does not create or modify project documentation.
