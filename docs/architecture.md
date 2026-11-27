# Architecture

`go-mcp-lib` supplies owned Go APIs for MCP servers and clients around the pinned
upstream SDK. Public library packages are organized by capability. The runtime
module and the independent maintainer generator module both require Go 1.27;
the prepared toolchain and dependency versions are pinned in
[`tools/versions.json`](../tools/versions.json).

## Packages and modules

```text
go.mod                         runtime module; no Go files at the repository root
mcp/                           owned server/client runtime, sessions and registrations
  internal/sdkbridge/          private SDK value conversions for the runtime
codec/                         independent strict raw-JSON API
protocol/                      generated wire data, codecs, schema and contract inventories
internal/validation/           shared schema validation and its colocated fixtures/tests
schema/                        immutable schema input, mappings, semantics and generation profile
cmd/mcpgen/                    independent standard-library-only maintainer module
  internal/                    acquisition, model, emitter, audit and publication mechanisms
scripts/                       Python local validation and qualification owners
tools/                         pinned tool metadata, test inventories and retained reviews
docs/                          decisions, execution records, reports and task guides
```

Consumers import `github.com/maccavelli/go-mcp-lib/mcp`, `/codec` or `/protocol`
for the capability they need. The module root has no compatibility facade.
The generator under `cmd/mcpgen/` is executable maintainer tooling, separate
from the library's imports and dependency graph.

## Dependency direction

The `mcp` package owns client/server handles, connection/session state, typed
tool/resource/prompt registration, middleware and transport handles. Package
private interfaces keep those runtime seams cohesive. It imports the generated
protocol, shared validation and its private SDK bridge. SDK runtime objects
remain behind owned APIs; generated data aliases have a separately audited
allowed SDK data-type closure.

`protocol` also imports shared validation, so validation lives at root
`internal/validation/`. Its corpus stays beside its tests. The independent
`codec` and `protocol` capabilities do not import the runtime or its private
bridge. The bridge does not import its owning runtime. Go's internal-package
rule prevents unrelated sibling packages from importing the bridge. Library
packages do not import command tooling.

Unit tests and external consumer tests live with their owning packages. The
generator module retains its own integration tests and fixtures. Root-module
test selection does not traverse that nested module; the local runner selects
both explicitly and reconciles results against the reviewed test inventory.

## Generation and local validation

The generator renders into a complete scratch candidate and compiles source
and tests for Darwin/arm64, Linux/amd64 and Windows/amd64 across both modules.
Ownership manifests and retained reviews identify generated output by hash.
Publication verifies ownership under an exclusive lock, journals replacements
and installs the manifest last. Recovery verifies the journal and current bytes
before completing or restoring. Per-file replacement is not a multi-file atomic
transaction. See the [generation and recovery guide](guides/generate-protocol.md).

The Makefile is a thin interface to Python check owners. `make pre-add-check`
runs readonly formatting, standalone vet, fresh tests with complete result
accounting, and pinned lint across both modules. `make lint` runs the two
complementary analysis owners: golangci-lint and standalone Revive for
documentation/naming, including generated files. Combined goals share each
matching leaf once per invocation. Layout and dependency checks reuse the
runtime's package-inventory query rather than run a second Go source scan.

`make generate-check` (also `build-check`) invokes the actual CLI's readonly
candidate check. `make docs-check` combines record/link and Markdown checks.
`make gate-selftest` exercises the checkers against broken scratch inputs.
`make setup-network` explicitly prepares pinned tools/dependencies; ordinary
checks run offline. Heavy Make targets use the operator's fleet selection and
shared host queues. Native Windows/amd64, Darwin/arm64 and physical Linux/amd64
qualification is supplemented by WSL. Ordinary builds and tests force CGO off;
CGO is enabled only for separately queued race tests on an eligible host.

## Completion status

The [framework PLAN](decisions/0002-PLAN-build-schema-driven-mcp-framework.md)
records qualification by phase. R2/R3 generation and publication are qualified.
R4/R5 codec/runtime repairs and Stage 7 transport guards remain pending.
Complete conformance, preflight receipts and security integration remain later
work; passing the local foundation does not establish those unfinished exits.
Operator host configuration, skills and global Git hooks are maintained outside
the repository. Upstream-forge configuration is outside this local Make scope.
