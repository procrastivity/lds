This installation exposes the Toolsmith scaffold's version, install, and
uninstall root commands, plus canonical `lds plumbing manifest` and
`lds plumbing doctor`. The root `lds manifest` and `lds doctor` spellings
remain compatibility aliases for already-installed skills and existing
callers. Consult `lds plumbing manifest --json` for the current command
surface.

`lds plumbing adopt` reads the live shared flow document; it does not inspect
a target, start a workflow, request approval, or write files. The separate
root `lds adopt <root> --facts <facts.json>` form is an interactive CLI LLM
workflow and is not projected into this deterministic skill. It sends the
owner-supplied facts and bounded inventory/link metadata to the selected
provider only after explicit transfer consent, displays the complete host
plan, and requires a fresh exact-digest human decision. The model has no
tools, does not verify owner claims, and cannot authorize writes.

The `lds plumbing docs inventory|validate|render|plan|apply` commands provide
deterministic documentation operations for an explicitly named target; they
are substrate, not the root LLM workflow. Do not infer owner approval from
facts, model output, or a digest: show the whole plan and obtain a fresh
decision before invoking `apply` with its exact reviewed digest. The root LLM
form is outside the first-release and real-pilot gates; this guidance is not
evidence of CLI/skill parity. Installation writes generated agent skills; it
does not create or modify project documentation.
