# Generate and check the protocol contract

Use the pinned Go toolchain and dependencies prepared by the repository's setup
gate. Generation and checking work offline; they do not acquire new inputs.
These commands run from `cmd/mcpgen/` and use the reviewed root generation
profile. Run them through the development host's heavy-run queue.

```text
env CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local GOFLAGS=-mod=readonly GOPROXY=off GOSUMDB=off go run . check --repo ../.. --format json
env CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local GOFLAGS=-mod=readonly GOPROXY=off GOSUMDB=off go run . generate --repo ../.. --format json
```

`check` renders a complete candidate in scratch, compiles source and tests for
both modules on the three supported targets, and compares the owned output.
It does not publish files. A successful check reports `ok` and no drift.
Cross-compilation complements the separately required native test gates.

`generate` qualifies the exact candidate before publishing it. It checks
ownership and content hashes under an exclusive lock, journals original and
candidate bytes, replaces files in deterministic order, and publishes the
manifest last. An unchanged generation preserves bytes and modification times.
The receipt describes filesystem durability limitations; replacements do not
form a multi-file atomic transaction.

The `--manifest` option selects a complete verified generation profile, relative
to the repository root. It does not name the generated ownership manifest.

## Migrate a frozen legacy generation

For a checkout that still has the old hashless ownership manifest, use explicit
bootstrap after reviewing the candidate migration inventory:

```text
env CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local GOFLAGS=-mod=readonly GOPROXY=off GOSUMDB=off go run . generate --repo ../.. --bootstrap-legacy --format json
```

Bootstrap verifies frozen Git object bytes, independently pinned hashes, current
files and the index before replacing anything. Dirty, unknown or conflicting
output is refused. The required historical Git objects must exist locally.
After successful migration, use ordinary generation; current bytes are never
blessed merely by hashing them.

## Inspect or recover an interrupted publication

```text
env CGO_ENABLED=0 GOWORK=off GOPROXY=off GOSUMDB=off go run . recover --repo ../.. --action inspect --format json
env CGO_ENABLED=0 GOWORK=off GOPROXY=off GOSUMDB=off go run . recover --repo ../.. --action complete --format json
env CGO_ENABLED=0 GOWORK=off GOPROXY=off GOSUMDB=off go run . recover --repo ../.. --action restore --format json
```

Inspect is read-only. Choose complete to install the verified candidate or
restore to reinstate the journaled originals. Recovery validates ownership,
paths and hashes, preserves unrelated files, and refuses manual changes or a
corrupt journal. An active or unverifiable lock owner is a conflict; age alone
never permits stealing a lock. Resolve ownership explicitly before recovery.

Help exits 0, invalid arguments exit 2, and validation/drift/recovery failures
exit 1. Structured results are versioned JSON on stdout; diagnostics go to
stderr. `update` acquires a bounded candidate with explicit immutable commit/hash
provenance outside the repository and never promotes live inputs or output.
