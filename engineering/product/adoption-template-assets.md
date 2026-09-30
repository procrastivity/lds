# Adoption template assets

These four assets are the template sources for the output files accepted by
the [adoption contract](adoption-contract.md#accepted-output-paths). Their
mapping and data fields are:

| Asset source | Rendered project file | Fields |
| --- | --- | --- |
| `assets/templates/adoption/engineering/README.md` | `engineering/README.md` | None; navigation link only to `product/overview.md`. |
| `assets/templates/adoption/engineering/product/overview.md` | `engineering/product/overview.md` | `DisplayName`, `Purpose`, `CurrentCapability`, `DisplayNameSource`, `PurposeSource`, `CurrentCapabilitySource`. |
| `assets/templates/adoption/docs/README.md` | `docs/README.md` | None; hub link only to `reference/usage.md`. |
| `assets/templates/adoption/docs/reference/usage.md` | `docs/reference/usage.md` | `Prerequisites`, `OperationName`, `OperationSummary`, `Invocation`, `Input`, `OutputEvidence`, `OperationSource`, `Limitations`. |

The `{{.FieldName}}` placeholders use Go `text/template` action syntax as a
small, reversible convention; existing LDS assets do not establish a project
documentation template renderer. This note specifies template provenance and
placeholder names, not a runtime implementation or escaping policy. Values
must be supplied from the target-specific, owner-approved and verified fact
bundle described in the adoption contract. Fixture values may only describe a
disposable fixture target and do not validate a real pilot.
