# Inspect a disposable project (offline)

This exercise uses an existing, throwaway directory; it never calls a
provider or writes adoption files. Use a locally available `lds` binary
(see the [index](../README.md)). The commands below use a fresh directory
outside your repositories:

```sh
project=$(mktemp -d)
mkdir -p "$project/docs"
printf '# Demo\n\nSee [guide](docs/guide.md) and [missing](docs/missing.md).\n' > "$project/README.md"
printf '# Guide\n' > "$project/docs/guide.md"
lds plumbing docs inventory "$project" --json
lds plumbing docs validate "$project" --json
```

Inventory exits 0 with `operation: "docs inventory"`: `files` includes
`README.md` and `docs/guide.md` with hashes/sizes, and `links` includes both
relative targets. Validate emits `operation: "docs validate"`, one finding
for the missing `docs/missing.md`, and exits **1**. On this diagnostic exit,
JSON can appear on stdout and an error on stderr; do not interpret stdout
alone as success. Create the missing file and check again:

```sh
printf '# Missing\n' > "$project/docs/missing.md"
lds plumbing docs validate "$project" --json
```

Now `findings` is empty and exit is 0. The validator checks supported local
**file existence**, not whether a fragment exists or the linked text is
correct; it does not fetch external URLs. See [safety and scope](../explanation/safety-and-scope.md)
before applying these tools to a real project. Delete this disposable directory
when finished (`rm -rf "$project"` only after checking its value).
