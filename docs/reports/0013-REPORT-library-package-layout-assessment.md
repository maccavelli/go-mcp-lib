# Library package layout assessment

Date: 2026-11-27

Status: four-repository reassessment complete; revised MADR accepted

## Finding

The common standard across the four repositories is functional package ownership,
private implementation boundaries, explicit import direction and colocated tests.
Three keep Go source out of the root; `acp-go-sdk2` deliberately retains a root
`acp` package. An empty root is therefore a valid owner preference for MCP, not
a universal Go requirement or a property shared by all four references.

For MCP, retain the independent `codec/` and `protocol/` packages, put the
cohesive runtime in `mcp/`, and scope its private bridge to
`mcp/internal/sdkbridge/`. Retain shared schema validation at root `internal/`
and the separate generator module under `cmd/mcpgen/`. That command is tooling;
library consumers import functional packages, not a command package.

This report records observations and recommendations. It does not authorize file
relocation, alter any reference repository or change the accepted execution
scope of [0002-MADR-build-schema-driven-mcp-framework.md](../decisions/0002-MADR-build-schema-driven-mcp-framework.md)
and [0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md).
The owner previously selected a simpler A relocation. The expanded assessment
and revised private-package scope were subsequently accepted in
[0014-MADR-organize-mcp-runtime-package.md](../decisions/0014-MADR-organize-mcp-runtime-package.md).
The owner approved the linked PLAN for execution on 2026-11-27; its L0
prerequisite precedes source relocation.
The owner's request names `go-llmprovider-lib`; the local checkout assessed is
`go-llmprovider-sdk`. No distinct checkout with the former name was found.

## Evidence and limits

The working trees, module boundaries, architecture records and relevant MCP
runtime declarations were inspected read-only. Counts are immediate `*.go`
children of each named directory, separating `*_test.go`; they are not recursive
repository totals. Reference-repository tests and builds were not run for this
layout assessment. The provider checkout is being edited concurrently: a later
baseline probe found source and documentation changes as well as new tests.
Those changes remain untouched; observations describe its inspected working
tree, not a claim that every cited file equals HEAD. The other three reference
working trees were clean at the baseline probe.

| Repository | Root production | Root tests | Observed public organization |
| :--- | ---: | ---: | :--- |
| go-selfupdate-lib | 0 | 0 | `selfupdate/`, `buildinfo/`; functional self-update subpackages |
| go-llmprovider-sdk | 0 | 0 | `llmprovider/`, `wizard/`, independent `authcontract/`; auth/catalog/provider subpackages |
| go-tui-lib | 0 | 0 | Top-level capabilities including `command/`, `workspace/`, `stream/`, `termcap/`; scoped adapters |
| acp-go-sdk2 | 19 | 97 | Root `acp` core; `agentproc/`, `clienthost/`, `remote/`, `acptest/`, `unstable/` |
| go-mcp-lib | 12 | 12 | Root `mcp` runtime; existing `codec/`, `protocol/` and generator module |

Reference HEADs at the baseline probe:

- `go-selfupdate-lib`: `43d33c7c780d18f4e2d7c9827a9f635c8fdc5384`.
- `go-llmprovider-sdk`: `223bb8dc597b37a7a3f0fcea70f85742ed03580b`.
- `go-tui-lib`: `2928c3d3a83afae4acf050e8e8b74dfb272a6029`.
- `acp-go-sdk2`: `a6495a56ddf2c2503b4613a2f91fb9ca400ef61a`.

The current MCP checkout includes approved, uncommitted R2/R3 work. Its runtime
root layout is unchanged by that work. File count alone does not establish a
package defect: the reference cores themselves retain substantial, cohesive
packages and colocated tests. Their useful precedent is separation by concern
and a clear repository entry, rather than an arbitrary per-directory file limit.
The earlier measurement found 48 production files and 91 tests in `selfupdate/`
itself, and 27 production files and 156 tests in `llmprovider/`. Those cohesive
cores are evidence against splitting a package solely to reduce its immediate
file count; the provider's counts are not a fixed current gate.

### Provider SDK precedent

In `go-llmprovider-sdk`,
`docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md`, decision D2,
defines packages by concern: core contracts in `llmprovider/`, authentication in
`llmprovider/auth/`, catalogs in `llmprovider/catalog/`, implementations in
`llmprovider/providers/<id>/`, conformance helpers in `llmprovider/llmtest/`,
shared wire/HTTP mechanisms in private packages, and the wizard in `wizard/`.
Its current `docs/architecture.md` records the later independently released
`authcontract/` module. That additional module has its own dependency/versioning
reason; it is not a requirement for organizing MCP source.

### Self-update precedent

In `go-selfupdate-lib`, `docs/architecture.md` explicitly specifies no root
package. `buildinfo/` owns stamps; `selfupdate/` owns the update API; subpackages
own archives, signing, verification, release specifications and service-manager
adapters. Private process, redaction, error and fixture support live under
`internal/`. The library separates functionality without moving all unit tests
into a generic test directory.

Maintainer tools live at `internal/cmd/selfupdate-release` and
`internal/cmd/selfupdate-note`. Consumers use imports such as
`github.com/maccavelli/go-selfupdate-lib/selfupdate` and
`github.com/maccavelli/go-selfupdate-lib/selfupdate/service/systemd`, not those
command paths. The fixture module under the release tool's `testdata/` is a
test fixture, not another distributed library boundary.

### TUI precedent

`go-tui-lib`'s `AGENTS.md` explicitly assigns each capability a top-level
directory, with shared private helpers under `internal/` and no root package.
Its `docs/architecture.md` distinguishes capability cores from framework
adapters such as `command/commandtea` and `stream/streamtea`.
`command` may consume the foundational `when` package; `workspace` composes
`command` and `keymap`. Adapter dependencies point toward the adapted core,
while the core does not import the adapter. Narrow dependency closures have
their own conformance checks.

Consumer imports include `github.com/maccavelli/go-tui-lib/v2/command` and
`github.com/maccavelli/go-tui-lib/v2/workspace`. Public test helpers such as
`tuitest/` and scoped `stream/streamtest/` serve consumers; they do not replace
the tests beside each implementation. Its current `go.work` uses the root
module; documentation about a future independent adapter module is not
evidence of an implemented nested module.

### ACP precedent and exception

`acp-go-sdk2`'s `docs/architecture.md` explicitly describes a root `acp`
protocol/connection core and functional public helpers. The root still has
19 production Go files and 97 tests. A consumer may import the root core, or
`github.com/maccavelli/acp-go-sdk2/agentproc`, `/clienthost` or `/remote` for
those capabilities. `acptest/` owns supported consumer test helpers; private
process-tree and tail-buffer mechanisms live under root `internal/`.

The separate `cmd/generate/go.mod` names
`github.com/maccavelli/acp-go-sdk2/cmd/generate`. Its command implementation has
private load/model/emit/output packages. This directly distinguishes generator
tooling from consumer library imports and supports retaining MCP's existing
generator boundary. Calling every reference an empty-root layout, or claiming
none has a `cmd` module, would be inaccurate.

### Unified standard supported by the evidence

| Concern | Common principle | Application to MCP |
| :--- | :--- | :--- |
| Public API | Name packages after coherent capabilities; keep coupled core contracts together. | `mcp`, `codec`, `protocol`; no repository-name wrapper or arbitrary server/client split. |
| Private mechanisms | Use the smallest common owning `internal/` scope. | Runtime-only bridge under `mcp/internal/`; shared runtime/generated-protocol validation under root `internal/`. |
| Imports | Keep the graph acyclic; adapters depend on their core. | Protocol/codec do not import runtime; runtime owns its SDK adaptation and sealed interfaces. |
| Tests and fixtures | Colocate tests with their owner and fixtures under its `testdata/`. | Move runtime/bridge tests with code; retain generator integration tests and validator fixtures with their owners. |
| Public test helpers | Give an actual reusable consumer testing capability its own package. | Do not invent `mcptest/` or a generic test module during this relocation. |
| Commands | Put executable tooling under command directories, independently of public API placement. | Retain `cmd/mcpgen`; no application library API is imported from it. |
| Modules | Create independent modules for dependency/release isolation, not folder grouping. | Preserve the existing root and generator module identities. |

### MCP dependencies that constrain a move

The twelve production files are `client.go`, `doc.go`, `errors.go`,
`middleware.go`, `middleware_bridge.go`, `output.go`, `request.go`,
`resources.go`, `server.go`, `session.go`, `tools.go` and `transport.go`.
All belong to `package mcp`. Public handles, request contexts, middleware and
registration methods share private state and helper functions.

`Transport` in `transport.go` has a package-private `transport` method; `Request`
in `request.go` has a package-private `ownedRequest` method. These boundaries
keep SDK runtime objects private. Independently moving implementations into
public sibling packages requires an API/dependency design; a directory rename
alone cannot preserve access to package-private declarations.

The bridge's production importers are `client.go`, `tools.go`, `resources.go`
and `middleware_bridge.go`; its only additional Go importer is a runtime test.
That supports narrowing `internal/sdkbridge/` to `mcp/internal/sdkbridge/`.
`output.go` delegates to the existing canonical `internal/validation` adapter.
The pending generator's `cmd/mcpgen/internal/emit/support.go.txt` emits protocol
support importing that same adapter, so validation belongs at the shared root
scope. The currently published R1 protocol is not evidence that the unpromoted
R2/R3 candidate has already landed. `codec/` currently implements strict raw-JSON
checks; it does not itself import the schema validator.

The existing external API tests import `github.com/maccavelli/go-mcp-lib`.
`cmd/mcpgen/candidate_runtime_test.go` locates `server.go` and
`middleware_bridge.go` at the candidate root and plants failures there.
`tools/test-inventory.json` identifies runtime tests by the root import path.
A relocation must update all three, retaining every negative control, and move
the owned-decoder plant's `internal/sdkbridge/codec.go` locator with its owner.
No local Git tags were observed; downstream consumer usage still needs an explicit
inventory before claiming that an import-path change affects nobody.

## Options for owner review

| Option | Result | Work and compatibility cost |
| :--- | :--- | :--- |
| A — Functional packages with scoped internals **(recommended)** | Zero root Go files. Keep the cohesive runtime in `mcp/`, its bridge in `mcp/internal/sdkbridge/`, independent codec/protocol packages and shared validation at root scope. Retain the generator module. | Moderate migration plus qualification. Public runtime imports change to `github.com/maccavelli/go-mcp-lib/mcp`; private bridge imports also change. Update inventories, source locators, examples and guidance together. |
| B — Thin root facade | Existing root import path remains; implementation moves into private packages. | Higher boundary/forwarding work. Some Go files remain at the root. Avoid leaking SDK types or introducing unnecessary public aliases merely to hide files. |
| C — Separate public client/server/transport packages | Zero root Go files and additional public package boundaries. | Highest API/design cost. Requires shared contract ownership, acyclic imports, a replacement for package-private cross-package seams and a larger migration/conformance campaign. |

Recommended shape for A:

```text
go.mod
mcp/                    public server/client/runtime API and its tests
  internal/sdkbridge/   private runtime-owned SDK translation and tests
codec/                  strict raw-JSON API and its tests
protocol/               generated protocol DTOs and contract artifacts
internal/validation/    shared canonical schema validator, tests and fixtures
cmd/mcpgen/             separately bounded generator command, not a library API
  internal/             generator-owned implementation, tests and fixtures
schema/                 reviewed generation inputs
scripts/                local validations and their controls
tools/                  inventories and maintainer tooling
docs/                   records, architecture and guides
```

## Required implementation work after the decision

1. Record the selected layout and import-path treatment in a paired MADR/PLAN;
   obtain execution approval before relocation. Keep the R2/R3 commit boundary
   explicit, and do not fold an unreviewed public API move into publication.
2. Move the runtime package, private bridge and their tests coherently;
   retain names, behavior, SDK isolation, build constraints and pure-Go policy.
3. Update imports, package identities in inventories, generator qualification
   locators and deliberate-failure controls. Keep both module boundaries intact.
4. Update architecture, examples and migration guidance. Add a layout check that
   rejects a planted root Go file in scratch; prove forbidden package edges,
   import migration and SDK-boundary checks catch their own broken inputs.
   Reuse existing lint/API/compiler checks rather than adding duplicate scans.
5. Run the existing local gates on the changed package graph. Compile all three
   targets with CGO disabled, execute native tests on the supported fleet, and
   enable CGO only for race tests. Preserve source/output pins and the complete
   candidate-before-publication requirement.

Go's [module layout guidance](https://go.dev/doc/modules/layout) supports both
root packages and supporting/internal packages. Its
[code organization documentation](https://go.dev/doc/code#Code_organization)
defines packages by directory and import paths by module plus subdirectory.
Its [package-naming guidance](https://go.dev/blog/package-names) supports clear,
cohesive package names. The empty-root layout is the owner's preference and
the precedent in three references; the functional ownership standard also
accounts for ACP's valid root core. This assessment has changed documentation
only; the owner subsequently accepted the revised decision and approved its
implementation PLAN on 2026-11-27.
