---
status: accepted
date: 2026-11-27
---

# Build a schema-driven MCP framework around the upstream Go SDK

## Context and Problem Statement

The owner requests a new, scoped shared Go library in `go-mcp-lib` that will
ultimately replace the MCP SDK integration currently assembled in `mcplib`.
It must provide an idiomatic, extensible Go API, use the latest stable MCP Go
SDK and MCP schema, support JSON v2, and make schema-driven generation a
primary architectural concern. The owner explicitly selected a **pure Go
generator**, as in `acp-go-sdk2`, on 2026-11-22. Python remains appropriate for
repository maintenance scripts, not the product generator.

This record establishes the accepted architecture and qualification requirements. It does
not authorize implementation, migrate consumers, or claim that the new
framework already passes runtime qualification. The original assessment
covered a scaffold without a Go module. Product source now exists; the
2026-11-24 completion audit below supersedes scaffold-only status statements,
while preserving the historical evidence and accepted requirements. The paired
[0002-PLAN-build-schema-driven-mcp-framework.md](0002-PLAN-build-schema-driven-mcp-framework.md)
was approved in its original form. The owner subsequently authorized R1 with
“resume r1”; R2–R5 await execution approval. Recheck evidence at execution.

### Evidence baseline and terminology

The following are observations as of 2026-11-22, not floating upgrade rules.

| Subject | Observed baseline | Evidence |
| :--- | :--- | :--- |
| New repository | Scaffold at `f0af2c50ce21b471b335cc9d2767f00b05a58fc5` | Local Git revision and file inventory |
| Existing integration | `mcplib` at `b4eac5280d785e2be47e057f3ac3ddca04535335` | `go.mod`, `server.go`, `hardened.go`, `schema/schema.go`, `hfsc/dispatch.go` |
| Reference framework | `acp-go-sdk2` at `a6495a56ddf2c2503b4613a2f91fb9ca400ef61a`, schema `1.31.0` | `schema/version`, generator source, JSON contract tests |
| Latest stable MCP Go SDK | `github.com/modelcontextprotocol/go-sdk v1.8.0` | [Official release](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.8.0), marked Latest |
| Latest stable MCP protocol/schema revision | `2026-07-28` | [Stable specification](https://modelcontextprotocol.io/specification/2026-07-28), the destination of `/specification/latest` |
| JSON Schema dialect of selected MCP schema | Draft 2020-12 | `$schema` in the [immutable schema input](https://raw.githubusercontent.com/modelcontextprotocol/modelcontextprotocol/ac00313ff7f5e9f47b0e0d1ebdf9c0ad6411380b/schema/2026-07-28/schema.json) |
| Stable standard-library JSON v2 | Go 1.27 | [Go release notes](https://go.dev/doc/go1.27) and [migration guide](https://go.dev/doc/jsonv2-migration) |

There is no single independent “MCP Go schema version.” SDK module version,
MCP dated protocol revision, JSON Schema dialect, generator version, and this
library's own version are different identifiers. They must all appear in the
release manifest. ACP's schema version does not identify an MCP schema. The SDK v1.8.0 release
notes identify `2026-07-28` as its newest negotiated protocol revision, so the
proposed SDK and schema baselines agree at the version level. Behavioral
coverage still needs the qualification described below.

The selected immutable MCP schema is from upstream commit
`ac00313ff7f5e9f47b0e0d1ebdf9c0ad6411380b`, at
`schema/2026-07-28/schema.json`. A fresh download during this assessment was
181,976 bytes and contained 155 `$defs`. Its SHA-256 is:

```text
907dcebce6aaeb16c21421ffef8c207093d88b9442fee340658a99cacb06b551
```

These values identify the assessed bytes. They do not establish that every
constraint or method is implemented by the SDK. The specification's normative
requirements are based on its TypeScript schema and accompanying prose. Both
must be reviewed alongside the JSON Schema when creating behavioral mappings.

### Findings from mcplib

References in this subsection are repository-relative paths at the mcplib
revision above. They describe inspected code, not a complete security audit.

| Finding | Code evidence | Design consequence |
| :--- | :--- | :--- |
| The existing SDK dependency is `v1.6.1` | `go.mod` | Moving to `v1.8.0` requires a real integration qualification, not just a module edit |
| `MCPServer` holds a single mutable session and exposes the raw SDK server | `server.go`, `MCPServer`, `Serve`, `Session` | Use explicit connection handles and scoped request/session context; do not make a global last session the routing authority |
| Hardening mixes generic MCP concerns with application policy | `hardened.go`, `InternalWrapHandler`, orchestrator signal injection | Separate reusable middleware from orchestration-specific metadata and policy |
| Tool serialization is keyed by name in a package-global map | `hardened.go`, `toolMutexes`, `getToolMutex` | Scope admission to a server/registration; unrelated servers with the same tool name must not share a lock |
| Panic text and simple-handler errors can be returned to the client | `hardened.go`, panic recovery and `HardenedAddSimpleTool` | Use bounded, sanitized public errors and private diagnostic events |
| Schema derivation can silently substitute an open object | `schema/schema.go`, `Derive` recovery and serialization error branches | Registration must report schema errors; an invalid declaration must not widen accepted input |
| HFSC's exported logging contract names the old SDK logging type | `hfsc/dispatch.go:44`, `LogSession`, and chunk/finalizer calls | Do not export or regenerate deprecated logging symbols in the new framework |
| The simple tool helper marshals output into text as well as returning typed output | `hardened.go`, `HardenedAddSimpleTool` | Make text presentation explicit; measure duplicate serialization before choosing a convenience default |

The existing library also serves application concerns outside a generic MCP
framework. LLM providers, configuration wizards, application-specific signals,
and MagicTools backplane routing are not dependencies of the new core.

The earlier mcplib proposals remain separate review artifacts:
`mcplib/docs/decisions/0018-MADR-make-mcplib-the-canonical-mcp-sdk-surface.md`
and
`mcplib/docs/decisions/0019-MADR-modernize-hfsc-backplane-transfers.md`.
The new owner direction places the future shared MCP provider in `go-mcp-lib`.
This draft does not silently accept or rewrite those proposals.

### What ACP demonstrates and what must change

The reference is `acp-go-sdk2`, the repository identified above. Its generator
is a separate Go module under `cmd/generate`. The following comparison is
based on implementation, with documentation used for orientation.

| ACP mechanism and evidence | Assessment for go-mcp-lib |
| :--- | :--- |
| `cmd/generate/main.go:72` loads committed inputs, builds a model, renders, checks, then writes | Adopt the phase separation and refusal to publish invalid output |
| `internal/load/load.go`, `Definition`, parses a bounded schema vocabulary and rejects unmodeled object fields | Adopt explicit support accounting; build an MCP-specific vocabulary inventory instead of assuming this parser accepts MCP |
| `internal/model/protocol.go`, `Protocol`, holds behavior the schema does not express | Adopt a reviewed MCP semantics table with normative source references and tests |
| `internal/emit/file.go`, `goFile`, constructs Go ASTs and formats output | Adopt Go AST emission and `go/format`; keep language generation separate from protocol modeling |
| `internal/output/check.go`, `Check`, checks generated code with handwritten code and tests, including selected cross-platform compilation | Adopt integrated compile checks; the exact supported platform matrix must be stated by this project |
| `internal/output/write.go:23`, staging followed by sequential rename | Preserve prepublication validation, but do not call this a multi-file atomic transaction; design interruption detection and recovery |
| `main_test.go`, `TestGeneratedFilesUpToDate`, compares rendered bytes and generated file inventory | Adopt deterministic regeneration and stale-output detection |
| `json_v1_pin_test.go` pins HTML escaping and sorted map keys using `encoding/json` | ACP is a v1 JSON compatibility reference, not proof of JSON v2 parity |
| `json_parity_test.go` exercises encodings and round trips | Adopt fixtures, supplemented by raw-byte validation that detects duplicate members and omitted/null differences |
| Handwritten `connection.go`, `middleware.go`, `observer.go`, and `acptest/` complement generated bindings | Adopt separation of runtime policy, observations, and test utilities; continue delegating MCP execution to its upstream SDK |

The ACP loader has ACP-specific fields such as `x-side`, `x-method`, and
`x-deserialize-*`. Its inspected `Definition` does not model `maxItems`, which
appears in the MCP schema. Reusing its loader unchanged would therefore fail
or require semantic changes. ACP's schema version and extension fallback
rules must not be transferred into MCP as if they were protocol requirements.

A targeted existing test was executed in ACP's generator module:

```text
go test . -run '^TestGeneratedFilesUpToDate$' -count=1
ok  github.com/maccavelli/acp-go-sdk2/cmd/generate  0.705s
```

This proves current regeneration matches ACP's committed output for that
case. It does not prove MCP schema compatibility, full ACP conformance, or
JSON v2 behavior. No full ACP suite was run for this draft.

ACP's repository license is Apache-2.0. Architectural reuse is the initial
approach. Any later source reuse must retain applicable notices and record
provenance; the PLAN must inspect the exact files being transferred.

## Decision Drivers

* Make the generator a maintainable product subsystem with explicit inputs,
  semantics, outputs, diagnostics, compatibility, and failure behavior.
* Offer a canonical public Go API without creating a competing MCP protocol
  runtime that must independently track every upstream transport fix.
* Enforce schema contracts at actual public and wire boundaries, with evidence
  that cannot pass merely because generator and tests share the same mistake.
* Support stable standard-library JSON v2 with explicit wire semantics.
* Keep application-specific transport extensions, storage, identity policy,
  and process supervision out of the minimal dependency closure.
* Preserve cancellation, bounded resource use, concurrency isolation, and
  observable failures without logging sensitive payloads by default.
* Permit future extensions through documented interfaces and schema inputs,
  without Staticcheck exceptions or deprecated public SDK symbols.
* Optimize using reproducible measurements and workload limits, not assumed
  benefits from code generation or a different codec.

## Considered Options

* A generated, schema-governed framework composed with the upstream MCP Go SDK.
* A thin package of SDK aliases and handwritten convenience wrappers.
* A fully generated independent MCP SDK and transport implementation.
* A maintained fork of the upstream SDK with generator and JSON v2 changes.

## Decision Outcome

Chosen option: “A generated, schema-governed framework composed with the
upstream MCP Go SDK”, because it gives the generator responsibility for
contracts and API completeness while retaining upstream ownership of protocol
execution. The owner selected all six recommended design decisions on
2026-11-22, following the earlier pure Go generator decision. Those selections
are recorded below. The later qualification amendment resolves G08, G20, and
G21 with executable probe results and a concrete acceptance contract. The
owner accepted the completed design on 2026-11-23. The original PLAN was
approved and Stage 0 investigation authorized on 2026-11-24. Implementation
subsequently landed, but the completion audit found unmet Stage 0–6 exits.
The owner selected repair option A; the detailed repair PLAN amendment is
presented for review before execution. This does not reopen accepted D/G/CI
policies or certify the existing implementation.

| Date | Owner ruling | Scope |
| :--- | :--- | :--- |
| 2026-11-23 | Accepted — “mark the madr as accepted” | Architecture, selected dependencies, qualification resolutions, and numerical acceptance contract; no PLAN execution authorization |
| 2026-11-23 | Acceptance reaffirmed; PLAN approved; wait for instructions | No implementation execution authorized |
| 2026-11-24 | Acceptance reaffirmed; PLAN approved; proceed to Stage 0 | Stage 0 investigation and execution evidence authorized; stages 1–10 not authorized |
| 2026-11-24 | B01 Option A — qualify the existing SDK integration | Approved source investigation and scratch proof under existing pins; preserve feature coverage, no deprecated API exposure or lint suppression, and SDK-owned dispatch |
| 2026-11-24 | Post-audit option A — “do A repairs” | Prepare the existing pair's repair amendment and acceptance ledger for review before execution; preserve the accepted contract and keep Stage 7 behind prerequisite requalification |

### Owner decisions recorded on 2026-11-22

The owner instructed “for decisions, follow recommendations” in response to
the six decision prompts. Each prompt presented three alternatives; the selected
recommendations and their acceptance evidence are recorded here.

| ID | Decision | Owner ruling and acceptance evidence |
| :--- | :--- | :--- |
| D01 | Public API | decided — owned server/client/connection handles with reviewed upstream data-type aliases; API inventory checks inherited methods and transitive signatures, with compile and serialization contract tests |
| D02 | JSON v2 release guarantee | decided — native v2 for owned types plus strict raw-JSON checks across every supported transport before lossy decoding; captured-frame adversarial tests must pass before the first stable release; G08 is resolved by the qualification amendment and companion report |
| D03 | Generator exposure | decided — supported CLI with a modular internal pure Go engine; CLI contract and import-boundary tests; public embedding is deferred |
| D04 | Runtime validation | decided — qualify the SDK-aligned validator behind an internal adapter, implement required semantic checks explicitly, and replace the validator if qualification establishes unsupported requirements; vocabulary and independent validation fixtures; G21 qualification selects the replacement documented below |
| D05 | Performance acceptance | decided — measure representative workloads before proposing numerical latency, allocation, memory, and cancellation budgets for approval; reproducible equivalent-workload benchmarks; G20 now has measured baselines and numerical implementation budgets below |
| D06 | Transport stability | decided — qualify both stdio and Streamable HTTP before declaring the framework stable; transport contract, lifecycle, limit, and strict-JSON tests on both paths; HTTPS remains unnecessary for subprocess artifact retrieval |

These decisions authorize recording the selected architecture. They do not
constitute implementation PLAN approval or an instruction to execute it.

### Scope and ownership

Start with one runtime Go module and a separate pure Go generator module,
following ACP's dependency separation. The generator operates on checked-in
inputs and produces committed Go artifacts. Consumers building the runtime
library require neither the generator nor Python. Use the configured canonical
remote to establish the module import path when implementation is authorized.

| Proposed surface | Responsibility | Boundary |
| :--- | :--- | :--- |
| Root package | Server/client construction, typed registration, lifecycle, options, common errors | Delegates execution to upstream SDK; no application globals |
| `protocol` | Reviewed aliases for modern upstream types, method/version identifiers, generated contract inventory | No handwritten duplicate of the entire SDK type system |
| `schema` | Embedded pinned schemas, compiled validation, declaration diagnostics | Never silently weaken a schema |
| `codec` | JSON v2 handling for library-owned values, raw JSON checks, documented boundary options | Does not claim to replace upstream private codecs |
| `extension` | Typed namespaced method registration, negotiation requirements, owned DTOs | Requires explicit support and cannot shadow standard methods |
| `transport` | Bounded upstream stdio/HTTP assembly and lifecycle configuration | TLS and authentication configuration remain explicit application inputs |
| `mcptest` | In-memory connections, raw recording, replay, schema assertions, failure injection | Test-only imports do not enter normal runtime packages |
| `cmd/mcpgen` | Pure Go generator CLI and internal engine packages | Separate module; not imported by runtime |

Names are proposed API boundaries, not promises to create every package before
there is a cohesive implementation. Keep generator IR and SDK adapter details
internal until a real external use case requires a public contract.

Expose standard SDK types through generated, reviewed aliases where identity
matters. Go aliases retain upstream type identity and inherit its methods;
they cannot gain local methods. Do not copy an upstream struct solely to add
codec hooks. Use separately owned types for extensions and framework policy,
and explicit conversions where required. This accepts some upstream API
coupling. An SDK upgrade must pass API comparison and compile tests before it
is promoted into a library release.

Review the full method set before generating an alias. Aliasing a session or
client can expose deprecated methods even when the alias declaration itself
uses a modern symbol. Prefer owned server/client/connection handles and narrow
modern interfaces for these runtime objects. Alias suitable data types only
after checking their fields and inherited methods. API inventory checks must
inspect transitive signatures as well as direct imports.

The canonical API must cover advertised features. An explicitly named upstream
accessor may support integration needs, but operations performed through that
accessor are outside framework enforcement unless separately intercepted and
qualified. Document this boundary and test it. A façade cannot guarantee its
policies for arbitrary direct upstream calls.

### Dependency and release policy

Proposed initial baseline:

* MCP SDK `v1.8.0`, exact module version with normal `go.sum` verification.
* MCP `2026-07-28`, immutable schema commit and digest above.
* Go `1.27.0` minimum for stable `encoding/json/v2` and `encoding/json/jsontext`.
  The qualification matrix includes the minimum supported release and the
  selected patched 1.27 toolchain. Older Go compatibility is out of scope.
* Standard-library Go for the generator engine initially. Use the Go command
  for module-aware compilation rather than building a replacement importer
  for every module-resolution case.
* D04 qualification found that `github.com/google/jsonschema-go v0.4.3`
  cannot enforce the selected lossless-number contract. Use
  `github.com/santhosh-tekuri/jsonschema/v6 v6.0.3` behind the internal
  canonical-validation adapter, with local reference resolution and explicit
  semantic format checks. The SDK retains its own transitive validator;
  validate raw inputs before delegation and do not reuse rounded projections.
  `github.com/yosida95/uritemplate/v3 v3.0.2` supplies the qualified template
  parser. The evidence report retains the exact diagnostic dependency graph.

“Latest” means checked at a release decision. Normal generation and application
startup never resolve floating tags, fetch schema URLs, or auto-upgrade SDKs.
Keep a manifest containing source URLs, revisions, SHA-256 digests, schema
dialect, SDK version, generator format version, toolchain, semantic mapping
revision, and generated file hashes. Extension schemas have independent
identifiers and digests. Include licenses with vendored inputs.

The rejected SDK-aligned candidate documents differences from ECMA-262 regular
expressions and does not validate `format` or content keywords. These are material limits,
not proof that every such annotation requires assertion. Required semantic
checks belong in the protocol mapping or a qualified validator implementation.
Do not silently turn a documented limitation into a standards-compliance claim.
See the [versioned validator documentation](https://pkg.go.dev/github.com/google/jsonschema-go@v0.4.3/jsonschema).

### Generator engine architecture

The pure Go engine uses independent, testable packages for acquisition,
loading, reference resolution, modeling, SDK mapping, emission, qualification,
and publication. The CLI wires them together. Expected errors carry source
URI, JSON Pointer, phase, and actionable reason. Stable diagnostic codes allow
tests and editor integration without matching prose.

#### Inputs and acquisition

An explicit update command fetches candidate schemas and records immutable
provenance. Loading consumes only a manifest and local files. Resolve references
against a bounded, allowlisted local registry. Refuse missing references,
network resolution, path traversal, duplicate JSON members, invalid UTF-8,
and malformed schema syntax before modeling.

Represent recursive references as graph edges. Do not recursively expand them
into an infinite tree. Bound source size, graph nodes, resolution depth, and
diagnostics, with configurable development limits and clear failure messages.
An acquisition result is a candidate until schema diffs, semantic mappings,
SDK capability checks, generated API diffs, and contract tests pass review.

#### Lossless schema model and protocol semantics

Preserve required/optional membership separately from nullability and Go zero
values. Retain constants, enum membership, unions, compositions, references,
object openness, numeric bounds, collection bounds, documentation, annotations,
and provenance. Do not flatten `oneOf` into `anyOf` or intersect `allOf` by
simply overwriting duplicate field names.

Generate a keyword inventory from inputs. Every encountered keyword is marked
as enforced, delegated to a qualified validator, annotation-only under the
selected dialect, or unsupported. Unsupported assertion behavior blocks output.
The engine need not claim to implement every possible JSON Schema vocabulary,
but it must cover all assertions in its pinned inputs and explicitly supported
extension schemas. Distinguish a closed enum from an explicitly extensible
value space; preserve unknown data only where the contract permits it.

A versioned semantic mapping supplies facts that JSON Schema does not fully
encode: direction, method name, capability requirements, request metadata,
result completion, subscription lifecycle, retry safety, cancellation, and
error mapping. Each rule names a normative specification section, relevant
schema pointer, SDK symbol, and an independent contract fixture. A missing or
unmatched rule is a generator error, not a warning that silently drops behavior.

#### SDK mapping and generated output

Map every stable schema definition to one of these explicit dispositions:

* An upstream public type with generated compile-time and serialization checks.
* A library-owned extension or convenience type with explicit conversion rules.
* A validation-only schema definition with no useful exported Go type.
* An unsupported capability that blocks release if it is advertised or required.

Generate reviewed aliases, method/capability registries, declaration checks,
validation adapters, owned DTOs and JSON hooks, documentation tables, fixture
scaffolding, and an API/coverage manifest. Do not generate a second standard
JSON-RPC dispatcher or duplicate the SDK's transport state machine.

A successful Go compile is insufficient evidence of a correct mapping. A Go
field can exist while omitting a required discriminator or encoding null where
the schema requires an array. Validate serialized representative values against
pinned schemas and inspect actual request/result frames.

#### Go emission and compile validation

Use `go/ast`, `go/token`, and `go/format`, following ACP's separation between
model and emitter. Stable name resolution detects collisions with keywords,
handwritten exports, methods, file names, and platform-specific sources.
Generated comments retain source pointers and deprecation annotations, but
deprecated SDK APIs are excluded from the modern public surface.

Render all outputs before modifying destinations. Compile the candidate tree
with its handwritten implementation, internal and external tests, and selected
build configurations using module-aware Go commands and a scratch tree or
overlay. Qualify Darwin/arm64, Linux/amd64, and Windows/amd64 initially; compilation
is distinct from execution on those hosts. Add other targets only with evidence.

#### Deterministic publication and recovery

Normal generation has no network access. Preflight requires the selected Go
toolchain and verified module dependencies to be available locally. Compilation
must not trigger automatic toolchain or module downloads; missing prerequisites
produce a diagnostic and belong to an explicit setup/update operation.
Output ordering, formatting, source identifiers, and file names are deterministic. Exclude timestamps and local
absolute paths. A second generation must produce identical bytes and inventory.

Acquire an exclusive generation lock. Stage and validate the complete candidate
set, compare existing owned files against their manifest hashes, and refuse to
overwrite unexpected manual edits. Rename changed files, remove only manifest-
owned stale outputs, and publish the new manifest last. Record a journal before
replacement so an interrupted run is detectable and can restore or complete
from verified staged data. Never describe sequential per-file renames as an
atomic multi-file commit. `check` and release gates must reject a mixed tree.

Unchanged files retain their bytes and modification times. Permission failures,
disk exhaustion, cancellation, stale locks, symlink escapes, and failed renames
must produce explicit diagnostics and preserve recoverable state. Recovery must
not erase handwritten files or unrelated uncommitted changes.

#### Proposed command surface and extension model

| Command under the pure Go `mcpgen` CLI | Contract |
| :--- | :--- |
| `inspect` | Print provenance, vocabulary coverage, SDK mappings, and unsupported features |
| `check` | Render/qualify in scratch and compare bytes and inventory without changing tracked output |
| `generate` | Publish verified output from pinned local inputs |
| `diff` | Compare schema semantics and public API impact between two manifests |
| `update` | Fetch an explicitly identified candidate and produce a reviewable change |
| `recover` | Diagnose and complete or restore an interrupted publication using its journal |

Owner clarification, 2026-11-26: `--manifest` on `inspect`, `check` and
`generate` selects the complete independently pinned generation profile,
defaulting to `schema/generation-profile.json`. It does not select the output
ownership manifest. Other commands retain their command-specific inputs.

| Profile selection state | Outcome and acceptance owner |
| :--- | :--- |
| Default or explicitly selected reviewed generation profile | decided — verify the profile and every member before emission; `TestRepairA14CLI` |
| Missing, malformed, unreviewed, escaping or aliased selected profile | decided — fail without output mutation; `TestRepairA14CLI` |

The engine has internal typed interfaces for source loading, reference lookup,
semantic rules, SDK mapping, and emission. Add extension packs as versioned
schema-plus-mapping inputs. Do not use dynamically loaded Go plugins or execute
code supplied by schema documents. A public embeddable generator API is deferred
until internal contracts are stable; pure Go implementation does not require
prematurely exporting every engine interface.

### JSON v2 contract and limits

Go 1.27 provides stable `encoding/json/v2`. Its documented defaults differ from
v1, including rejection of duplicate names and invalid UTF-8 and different
nil collection encoding. The new framework should use native v2 APIs for its
owned serialization paths and generated types, with explicit field semantics.
See the [standard package documentation](https://pkg.go.dev/encoding/json/v2).

Use streaming marshal/unmarshal interfaces where useful, but qualify all calls
that actually dispatch to those interfaces. Required arrays must be emitted
as arrays, and absence, null, empty, and zero must remain distinct where the
schema distinguishes them. Use a presence representation only for types that
need those distinctions. Preserve numbers without a blanket float64 conversion.
Unknown member handling follows object openness and explicit extension rules.

The upstream SDK v1.8.0 source `internal/json/json.go` imports
`github.com/segmentio/encoding/json` and configures case-sensitive matching.
Changing this library's imports cannot replace that private decoder. Standard
Go v1/v2 interoperability also does not prove interoperability with a third-party
codec. Aliases of upstream types cannot add new local JSON methods.

Consequently, JSON v2 support has three separately measured contracts:

| Boundary | Proposed ownership | Required evidence |
| :--- | :--- | :--- |
| Owned extension DTOs and library data | Native v2 serialization, generated hooks where needed | Direct v2 encode/decode, presence and discriminator tests |
| Upstream SDK values passed through framework API | Preserve upstream type identity and custom serialization | Cross-codec fixtures for content unions, raw values, nils, numbers, and custom methods |
| Actual stdio and HTTP MCP messages | SDK execution plus framework validation at accessible boundaries | Captured wire bytes and adversarial transport tests; do not infer coverage from standalone codec tests |

Under D02, strict raw JSON checks must run before lossy decoding across every
supported transport before the first stable release, including duplicate-name
and invalid-UTF-8 rejection. Application-owned tool
arguments and extension payloads can be validated where raw bytes are available.
For complete transport frames, implementation must demonstrate a supported
upstream interception point or a bounded framing wrapper that preserves the
transport contract. If such interception cannot be qualified, the stable
release is blocked. Record the failure and return to architectural review;
do not silently weaken D02 or omit a D06 transport. Replacing the upstream
runtime solely to claim v2 everywhere is not authorized by this decision.

Defaults deserve their own contract. JSON Schema's `default` annotation is not
an instruction to mutate every input. Preserve upstream tool defaulting behavior
only where the framework explicitly adopts and tests it; do not have generator,
validator, and codec each apply potentially different defaults.

### Runtime API, safety, and extensibility

Offer typed tool registration with schema diagnostics before serving, explicit
structured output and text presentation, resources and resource templates,
resource links, prompts, discovery, pagination helpers, and supported progress,
cancellation, subscriptions, and elicitation surfaces. Generate the exact
availability matrix for the selected revision. Optional protocol extensions
require explicit configuration and capability negotiation; their mere presence
in generated code must not advertise support.

Use `context.Context` as the first parameter for operations that can block.
Return errors for invalid user configuration and registration. Separate
protocol errors, tool execution failures, transport failures, cancellation,
and internal faults, with `errors.Is`/`errors.As` support where meaningful.
Do not convert arbitrary programmer panics into successful registration. A
narrow adapter around an upstream panic-based registration API requires tests
and must only convert expected declaration failures.

Keep configuration immutable after construction unless a documented concurrent
registration API is used. Scope middleware, quotas, mutexes, caches, and lifecycle
state to the owning server/client/connection. Cancellation must reach admission
waits, handlers, stream operations, and shutdown. A bounded shutdown cannot
forcibly stop an arbitrary handler that ignores its context; report that failure
and test it. Own and document who closes readers, writers, and child processes.

Provide composable receiving/sending middleware and an observation interface
with operation, duration, byte counts, outcome, and correlation information.
Default observations exclude payload bodies, credentials, raw URLs with secrets,
and panic values. Avoid a mandatory telemetry vendor dependency. Do not impose
an implicit global environment switch for semantic behavior.

Protocol policy is modern-only for this project. Server version restriction
and client connection setup must be tested against unwanted negotiation and
fallback. SDK `ServerOptions.SupportedProtocolVersions` narrows support, but
its source explicitly retains behavior for legacy `initialize` requests.
Setting that option alone does not prove that legacy traffic is rejected.
Generate and qualify the policy at actual sending/receiving boundaries.

Bound frames, nesting, in-flight work, queues, and cache memory. Expose
configuration with validated ranges. Admission overflow must have a documented
error or backpressure behavior. Do not automatically retry non-idempotent tool
calls. Caches must isolate identity, authorization changes, and expiry and have
bounded retention. Disable optional caching until that behavior is qualified.

### Artifacts and large transfers

The core provides standard MCP resource links and resource operations. The
preferred artifact integration remains a resource link followed by bounded
parts over an existing subprocess stdio connection. It must not introduce an
HTTPS listener or certificate distribution as a prerequisite for local artifact
retrieval. Resource identity, authorization, expiration, cache behavior, byte
ranges, and part sizing need integration tests before deployment.

The library's extension surface should support later modernized HFSC use cases,
including server-to-server backplane transfers that avoid chat context. The
previous owner requirements of at most 50 MB per file and concurrency 3 remain
workload requirements for that future integration. The exact byte definition of
50 MB and concurrency scope must be resolved in its transfer design. A new HFSC wire protocol, storage subsystem, and MagicTools routing service
require a separate workload decision and remain outside this core proposal.

No deprecated logging adapter and no Staticcheck exception is proposed. The
new extension system uses modern SDK symbols and explicit capability contracts.
Compatibility with legacy MCP servers is out of scope, as already directed by
the owner. Existing consumers are not changed in this work.

### Performance and dependency discipline

Measure generation load/resolve/model/emit/check time separately, including
cold and warm runs. Measure runtime allocations, throughput, latency, retained
memory, cancellation latency, and slow-peer behavior for small tool calls,
large structured output, resource parts, and concurrent extensions.

Compare equivalent validated workloads using upstream SDK directly and the
framework. Compare codecs using equivalent semantics; a faster path that skips
required validation is not an improvement. Register and compile schemas once
where possible, cache by immutable identity with bounded lifetime, and avoid
repeated generic JSON round trips. Buffer pools require maximum retained sizes
and ownership tests. Start without unsafe conversions or unbounded pooling.

The qualification amendment below supplies measured boundary costs and numeric
acceptance budgets. The extra checks cost time; no speedup is claimed. The
implementation must meet the frozen budgets without dropping correctness
checks. JSON v2 and generation are capabilities, not evidence of an automatic
performance improvement.

### Migration and release sequence

The first usable release must include the generator, pinned inputs, standard
API coverage, JSON v2 boundary tests, and runtime contract harness. Begin with
a pre-v1 library release until the API and workload contracts are qualified.
Under D06, both stdio and Streamable HTTP must pass qualification before the
framework is declared stable; qualifying stdio alone is insufficient. D02's
strict raw-JSON guarantee applies to both. Freeze generator manifest/IR format
versions separately from library semver.

A later approved migration can move mcplib's generic MCP assembly to imports
of this library, then integrate MagicTools as a real workload. Remove duplicated
wrappers only after replacement behavior is tested. This draft does not require
moving unrelated mcplib packages, retaining its deprecated public signatures,
or updating all consumers together. Rollback is a dependency-level return to
the prior integration; no persistent data migration is proposed here.

Before Go source is staged, require `gofmt`, `go vet`, appropriate package tests,
and pinned `golangci-lint` with Staticcheck enabled for every new module. No
SA1019 suppression is accepted. CI must run generator drift, API compatibility,
raw-wire contracts, schema qualification, race/leak checks, and the documented
platform matrix. Heavy runs use the repository's fleet host and queue policy.
Exact lint/tool versions and executable gate commands belong in the PLAN.

### CI/CD owner ruling — 2026-11-24

The owner reviewed
[0009-REPORT-cicd-standardization-assessment.md](../reports/0009-REPORT-cicd-standardization-assessment.md)
and selected D1 A and D3 C. Extend the existing Python 3.12+ standard-library
gates behind a thin Makefile, sharing the functional gate contract across queued
native fleet hosts and GitHub. Retain both modules, both pinned Go versions and
Linux/amd64, Windows/amd64 and Darwin/arm64, with CGO-free baseline builds/tests
and separate qualified race instrumentation.

Local test receipts are reusable for less than one hour, only for the exact
outgoing commit and unchanged policy/tool/input identities. Each push needs fresh
successful local vulnerability evidence. Disclosure guards and security scans
remain owner-managed locally; test-receipt integration preserves them without
installing or reconfiguring scanners or guards.

For D2, the owner reports changing the repository to public and excludes GitHub
enforcement. Branch protection, rulesets, required-check configuration, visibility
changes and plan upgrades are out of scope. GitHub functional CI remains required;
its summary reports failures without administering merge/push restrictions. This
supersedes the report's original enforcement proposal and 24-hour recommendation.

These are decided policy states, not claims of implemented tests. Named cases map
to Stage 9 in the paired PLAN.

| ID | Input or state | Ruling and acceptance case |
| :--- | :--- | :--- |
| CI01 | Local Make or GitHub invocation | decided — shared Python gate contract; `TestCI01SharedGateContract` verifies selection and failure propagation |
| CI02 | Native tuple, compiler or CGO mode differs from policy | decided — reject wrong/missing coverage; `TestCI02NativeMatrix` verifies six baseline combinations, both modules and separate preferred-version race results |
| CI03 | Complete unchanged exact-commit receipt below 3,600 seconds old | decided — permit reuse only with fresh local vulnerability evidence for this push; `TestCI03FreshReceipt` verifies positive reuse |
| CI04 | Missing, malformed, altered, future, expired or mismatched receipt | decided — reject; age at least 3,600 seconds requires new preflight; `TestCI04ReceiptRejection` covers boundary times, identity changes and every outgoing ref tip |
| CI05 | Fresh owner-managed local vulnerability evidence absent or unsuccessful | decided — reject local push qualification; no scanner/guard installation or GitHub scan jobs; `TestCI05LocalSecurityBoundary` verifies evidence consumption and preservation of disclosure behavior |
| CI06 | GitHub matrix failure, cancellation, missing result or unexpected skip | decided — final CI summary fails; `TestCI06CISummary` proves incomplete results cannot pass |
| CI07 | Concurrent fleet runs, queue/transfer failure or cancellation | decided — serialize heavy work per host, bound waits, retain failures and clean up owned resources; `TestCI07FleetLifecycle` verifies contention and interruption |
| CI08 | GitHub protection, required checks, visibility or billing administration | out of scope — explicit owner ruling; not an acceptance blocker |

### Local scaffolding ownership and scope — 2026-11-24

The owner names the dedicated skill `local-cicd-scaffolding`. This replaces the
previous conversational working name; no existing skill installation has been
renamed. The skill owns reusable local acceptance, validation, tests and gates
through pre-push readiness. It is forge-neutral: its instructions, references,
templates, configuration, defaults and tests contain no forge-specific policy or
integration. It neither configures nor participates in upstream CI/CD, pushes,
publishes, deploys, creates tags, or administers remote services.

`workspace-scaffolding` retains the agent instruction files, harness pointers,
documentation layout, records tooling, Markdown configuration and line-ending
baseline. `local-cicd-scaffolding` composes those existing checks into the local
gate graph without taking over their templates. The fleet host-selection skill
retains host inventory and exclusive heavy-run scheduling. Existing global hooks
and owner-managed disclosure/security tooling retain their ownership; the new
skill provides local verification entry points and evidence adapters, not a
replacement hook installation or scanner policy.

Maintain the dedicated skill in the canonical fleet skill library. Repositories
receive versioned, reviewable local tooling and an explicit repository profile;
running their gates must not depend on an installed agent skill. Extend a sound
existing runner rather than introducing a second implementation. Use a thin
Makefile over Python 3.12+ standard-library orchestration, with subprocess argument
lists, explicit environments and native Windows support. Store pins and policy in
local machine-readable files; never derive them from upstream configuration or
require an upstream forge account, service, credential or API for local
verification. SSH access to development hosts is a separate local fleet concern.

#### Local runner environments and CGO isolation

Use the existing fleet host-selection and execution interfaces for remote runners
over SSH. Resolve host aliases and the selected WSL distribution from the private
fleet inventory, not committed hostnames or addresses. Use non-interactive key
authentication with bounded connection timeouts. Run session-local work directly
through the local heavy-run queue, without an SSH connection back to this Mac.

| Runner role | Platform | Execution and required coverage |
| :--- | :--- | :--- |
| Native Windows laptop | Windows 11, `windows/amd64` | SSH to the laptop; native Windows Go processes for baseline builds/tests and qualified race tests. |
| WSL on that laptop | Linux, `linux/amd64` | Enter the configured distribution through the same laptop SSH connection; additional Linux baseline builds/tests, recorded separately from native Windows and the standalone Linux host. |
| Session Mac | macOS, `darwin/arm64` | Local execution through the queue; baseline builds/tests and qualified race tests. |
| Standalone Linux host | Owner-specified Ubuntu 26, `linux/amd64` | SSH to the host; baseline builds/tests and qualified race tests. |

These platform descriptions are the owner's requested profile, not fresh host
probe results. Verify actual OS/distribution, architecture, toolchain and required
tools before qualification. Require both modules and both pinned compiler versions
on all four environments: six native baseline cells plus two additional WSL
baseline cells. WSL never substitutes for Windows or the standalone Linux host.
Use preferred-version race runs on the three native targets. The current fleet
policy excludes WSL from race execution pending separate requalification; it
remains a required CGO-free baseline environment here.

Use `pick-host` and `run-on-host` with platform/environment selection for remote
work, and `run-heavy` for session-local heavy work. Pin required environment roles
rather than accepting whichever Linux runner happens to be least loaded. Windows
and all its WSL distributions share one physical-machine queue; their heavy runs
must not overlap. Independent physical hosts may run concurrently. If routing
cannot express a required role, report incomplete qualification rather than
substitute an environment or bypass the queue. Record snapshot identity and
per-environment results; an unreachable or failed required runner is not a pass.

Set `CGO_ENABLED=0` explicitly for every non-race Go operation: preparation that
builds Go tools, generation, module checks, vet/lint, ordinary tests, fuzzing,
benchmarks, conformance, example/consumer builds and all deliverable builds.
Propagate that environment to child Go processes instead of inheriting host
defaults. Only a separately identified `go test -race` invocation and its
race-instrumented compilation may set `CGO_ENABLED=1`, after checking the native
C toolchain. Ordinary Go builds launched by a race test's fixtures must explicitly
restore CGO off. Do not toggle a global environment or persist `go env -w` state.
Keep race artifacts distinct from CGO-free output and prevent their promotion as
ordinary build evidence. Validate effective CGO mode per command, retain build
settings as artifact evidence, and reject a non-race command with CGO enabled.

#### Complete local standards inventory

Inventory every existing gate and accepted repository requirement before applying
the scaffold. Preserve a coverage ledger mapping each requirement to its command,
module, target, required inputs, acceptance case and failure control. A capability
may be inapplicable only with a recorded reason; a missing implementation remains
incomplete. Repository-specific thresholds, fixtures and product contracts stay
in the repository profile rather than becoming universal fleet defaults.

| Area | Required local scaffold capability |
| :--- | :--- |
| Acceptance and traceability | Reconcile accepted MADR/PLAN cases, required test IDs, module/target inventory and gate dependencies; reject missing obligations and false completion claims. |
| Preparation and reproducibility | Explicit preparation of pinned tools, dependencies and fixtures with integrity verification; offline checks afterward; readonly module resolution; independent root/nested-module checks and separately declared workspace checks. |
| Pre-add and static checks | Non-mutating formatting checks, vet, pinned lint including generated and platform-specific code, deprecation checks and supported modernization checks; cover added, changed, renamed and deleted files and changed generator inputs. Keep formatting/fix commands explicit. |
| Module and dependency contracts | Validate module inventory, tidy diffs, approved dependency closure, compiler floors, build tags, experiments, tool versions and product CGO policy. |
| Functional and integration tests | Fresh unit, integration, lifecycle, filesystem, subprocess, transport, cancellation, concurrency and error-contract tests where applicable; reconcile terminal results and intentional exclusions. |
| Native platform qualification | Queue native Linux/amd64, Windows/amd64 and Darwin/arm64 runs for this Go profile, across minimum/preferred compiler pins and every module; CGO-free baselines, independent external consumers and separate qualified race instrumentation. |
| Generated and public contracts | Check deterministic regeneration, byte/hash/ownership drift, schema and fixture provenance, API signatures/aliases and external-consumer compatibility against an explicit applicable baseline. |
| Conformance and examples | Invoke repository-owned protocol/behavior qualification, executable examples, documentation/code parity and fixture validation where present. |
| Fuzz, leaks and performance | Inventory fuzz targets, run bounded campaigns with retained seeds, exercise leak/lifecycle checks and collect complete benchmark samples; enforce repository-approved coverage and performance budgets without inventing fleet-wide thresholds. |
| Documentation and local disclosure | Invoke records/index/link checks, Markdown lint and existing local disclosure/security checks or evidence consumers; preserve owner-managed tools and private configuration. |
| Gate verification | Test orchestration and parsers; observe every new gate fail on deliberately broken scratch inputs; reject missing tools, empty selections, malformed results, timeouts and unexpected skips. |
| Fleet lifecycle and efficiency | Use immutable snapshots and verified transfers, exclusive per-host heavy-run queues, bounded waits, cancellation, owned-resource cleanup and retained logs. Reuse verified build/download caches; never substitute cached test results for required execution. |
| Pre-push readiness | Bind complete evidence to candidate commits, every applicable outgoing ref tip, policy/tools/inputs, native results and timestamps; reject stale, changed, incomplete or unsuccessful evidence. Expose verification without performing a push. |

These capabilities consolidate the local practices assessed in
[0009-REPORT-cicd-standardization-assessment.md](../reports/0009-REPORT-cicd-standardization-assessment.md):
ACP's module/toolchain coverage, TUI's module inventory and local tooling, LLM
provider's isolated checks and semantic gates, and selfupdate's deliberate-failure
controls. Their current implementations are evidence to assess, not templates to
copy unchanged. The completion-audit findings in
[0011-REPORT-pre-stage-7-completion-audit.md](../reports/0011-REPORT-pre-stage-7-completion-audit.md)
remain failures to repair, not accepted scaffold behavior.

#### Streamlined checks with complete coverage — 2026-11-24

The owner requires necessary local checks without duplicate tooling runs or
redundant scans. The source assessment is
[0012-REPORT-local-gate-duplication-assessment.md](../reports/0012-REPORT-local-gate-duplication-assessment.md).
Use one explicit execution graph per repository. Make targets, pre-add adapters
and preflight select shared obligations; an aggregate must not recursively run
another aggregate's already-requested checks. Assign each obligation a canonical
producer and allow several validators to consume its identified result.

Maintain both an obligation-to-check ledger and an analyzer ownership ledger.
Reconcile package/module coverage, generated/test-file inclusion, compiler,
GOOS/GOARCH, tags, experiments, CGO/workspace mode, fixtures, dependency inputs,
baseline/history, repeat counts, coverage scope and relevant host environment
before treating executions as equivalent. Similar tool names or overlapping
diagnostics are insufficient. Keep distinct predicates and required execution
variants; the four environments, both compiler versions and separate race runs
remain required. Do not remove a stronger check in favor of a weaker duplicate.

Execute equivalent leaves once per qualification run, including concurrent or
aliased requests, and propagate their result to every dependent obligation.
Separate expensive producers from consumers: test events and coverage artifacts
feed summaries and budgets without hidden reruns; a shared file/module inventory
feeds independent validators. Share matching clean self-test baselines and
prepared inputs, but give every deliberately broken fixture its own identity and
required execution. Preserve explicit non-mutating formatter, standalone build,
external-consumer and product-specific contracts where no equivalent exists.

Result reuse requires complete semantic identity and the approved freshness
policy. Reuse across separate invocations must not silently rely on successful
timestamps or the Go test result cache. Keep exact-commit receipt and fresh local
security requirements; pre-commit results cannot automatically certify a later
commit. Changes invalidate dependent results; an unknown dependency selects the
full applicable graph. Source-only checks may have a single execution owner,
while platform-sensitive tooling tests retain their native requirements.

Record each requested obligation, its producer/context, execution or reuse reason,
wall time, queue wait, preparation/transfer cost and overall critical path. Require
zero unexplained duplicate leaves and an unchanged acceptance inventory. Claim a
speedup only from before/after measurements on comparable inputs, hosts and cache
conditions. The skill must assess each repository's real call graph rather than
install every known checker and compound the existing duplication. Any analyzer
relocation or selective execution needs equivalence and invalidation proof before
adoption; existing checks remain effective until then.

#### Idempotence and acceptance states

An unchanged second application must leave content, permissions, symlinks and
modification times unchanged, with no duplicate Make targets, pointers or ignore
entries. A read-only inspection reports the proposed delta before application.
Track template version, managed paths/sections and base digests. Preserve product
targets, repository overrides and unrelated work; conflicting managed edits need
an explicit reconciliation, not an overwrite. Validate the complete candidate
before publication, recheck bases before writing, and make interrupted changes
detectable and recoverable. An upgrade is a reviewed migration with a rollback,
not an implicit template replacement.

| ID | Input or state | Ruling and acceptance case |
| :--- | :--- | :--- |
| LC01 | Fresh repository, absent remote or unrelated upstream configuration | decided — scaffold only the local contract with no upstream-specific dependencies or changes; `TestLC01LocalBoundary` |
| LC02 | Existing workspace tooling, runner, hooks, security tools and product targets | decided — preserve ownership and compose existing entry points; reject duplicate/conflicting implementations; `TestLC02Ownership` |
| LC03 | Existing or accepted local requirement omitted from the profile | decided — coverage ledger reports an incomplete obligation; explicit inapplicability needs a reason; `TestLC03StandardsCoverage` |
| LC04 | Repeated inspection/application with unchanged inputs | decided — inspection makes no writes and repeated application is a complete filesystem no-op; `TestLC04Idempotence` |
| LC05 | Repository overrides, dirty managed content or changed bases during application | decided — preserve overrides/unrelated work and stop conflicting publication with a concrete diff; `TestLC05Drift` |
| LC06 | Invalid/unsupported profile, unsafe path/symlink, denied write, interruption or concurrent application | decided — fail with named evidence; no unreported partial success, destructive cleanup or readiness receipt; retain recoverable owned state; `TestLC06FailureRecovery` |
| LC07 | Template version upgrade or requested rollback | decided — explicit versioned migration restores the recorded prior managed state while preserving unrelated changes; `TestLC07Migration` |
| LC08 | Missing module/target/tool/test result, failed command, malformed evidence or unexpected skip | decided — readiness fails; partial/diagnostic runs cannot certify acceptance; `TestLC08HonestReadiness` |
| LC09 | Candidate identity, policy, inputs, tools, evidence age or local security evidence changes | decided — apply the repository's declared validity rules and reject invalid reuse; this repository retains CI03–CI05; `TestLC09EvidenceValidity` |
| LC10 | Local product-specific requirement or established budget | decided — use a repository extension with explicit applicability and independent failure control, without embedding product semantics in the shared core; `TestLC10RepositoryProfile` |
| LC11 | Four runner roles, shared Windows/WSL hardware, wrong environment or unreachable host | decided — require distinct environment evidence, SSH routing and shared physical-host serialization; no substitution, unqueued fallback or silent omission; `TestLC11FleetRunners` |
| LC12 | Inherited CGO setting, race invocation, child build or artifact reuse | decided — force CGO off for every non-race Go operation and permit CGO on only for qualified isolated race execution; reject mode leakage and race artifacts used as baseline evidence; `TestLC12CGOIsolation` |
| LC13 | Several aliases, aggregates or concurrent requests require the same leaf/context | decided — execute once and share the identified result; reject graph cycles, missing owners and unexplained duplicates; `TestLC13SharedExecution` |
| LC14 | Related tools or scopes appear redundant but have different coverage or failure semantics | decided — preserve every required predicate and variant; consolidate only after demonstrated equivalence, including failure controls; `TestLC14CoverageEquivalence` |
| LC15 | Changed input/context, partial selection, mutated fixture or stale/failed shared result | decided — invalidate affected evidence, never promote partial or failed results, and keep changed fixtures/variants distinct; `TestLC15ReuseInvalidation` |
| LC16 | Gate-cost report, shared preparation or claimed performance improvement | decided — report actual invocations/reuse, queue/service costs and comparison conditions; no speedup or completeness claim without evidence; `TestLC16CostAccounting` |

The name, local-only scope and idempotence requirement are owner decisions.
Implementation and deployment of the fleet skill require the owning library's
MADR/PLAN; this amendment records the consumer contract and Stage 9 integration
criteria, not completed skill creation or authorization to modify another tree.
Existing upstream work in this repository's Stage 9 remains separate from the
skill's payload and responsibilities. R1–R5 repair approval is unchanged.

## Consequences

* Good, because one maintained schema and semantics pipeline can produce the
  API inventory, validation bindings, and traceable contract fixtures.
* Good, because the upstream SDK continues to supply protocol and transport
  execution while the library owns consistent registration and policy.
* Good, because pure Go generation keeps AST construction and output type
  checking aligned with the language being generated.
* Good, because JSON v2 is adopted on stable Go APIs with explicit boundaries
  and observable wire tests.
* Bad, because schema completeness, semantic mappings, and SDK behavior remain
  separate obligations that require continuing maintenance.
* Bad, because preserving upstream type identity exposes some upstream API
  changes and limits where local codec methods can be added.
* Bad, because a separate generator module adds release and test orchestration,
  and Go 1.27 excludes older toolchains.
* Bad, because crash recovery, independent validation, and adversarial transport
  qualification require more engineering than a basic struct generator.

## Confirmation

### Qualification resolution on 2026-11-22

The owner requested zero remaining gaps after selecting D01–D06. The resulting
[0002-REPORT-qualify-mcp-framework-boundaries.md](../reports/0002-REPORT-qualify-mcp-framework-boundaries.md)
retains the complete diagnostic Go source, module checksums, fixture provenance,
positive and negative controls, and all benchmark samples. This amendment
replaces the earlier unspecified outcomes; it does not change the selected
API, pure Go generator, or two-transport scope.

#### G08 resolved with public transport hooks

Use `IOTransport.Reader` for complete stdio frames; an outer HTTP handler for
server POST bodies; and the supplied `http.Client.Transport` for JSON response
bodies and bounded SSE events. Validate before releasing bytes to SDK decoding.
`mcp.Connection.Read` middleware alone is too late because it sees parsed data.

The SSE guard assembles a complete event, checks the JSON payload, and releases
only validated data. Its classification must cover the SDK's event-name
normalization as well as ordinary `message` events, so trailing whitespace
cannot bypass validation. Preserve a sticky first error and close the source
on corruption. A connector-installed failure callback closes the original SDK
connection and blocks resumption; return that original connection to the SDK
so private session/version update hooks remain intact. Canonical operation
errors use the stored validation cause rather than relying on EOF wording.

The probes verified successful stdio/JSON/SSE controls, duplicate and invalid
UTF-8 rejection, zero-byte exposure on invalid frames, event framing variants,
and corrupt SSE data following a resumable event. The latter made exactly one
HTTP request with retries otherwise enabled. The HTTP server control returned
200 without the guard and 400 with zero SDK handler invocations with the guard.
Deliberately disabling the guard made the original assertions fail.

This resolves the interception feasibility question without a fork or deprecated
API. Full transport lifecycle/conformance tests remain specified implementation
acceptance tests, as for every feature in this unimplemented library.

#### G21 resolved by qualifying and replacing the validator

The pinned protocol schema uses 17 distinct keywords. Its 14 assertion,
applicator, reference, and definition categories were exercised with 421
independent fixtures; dialect and annotations are accounted for separately.
All 155 definitions compile under the selected replacement. The report lists
every keyword and its count, so this claim is bounded to the actual inputs.

The SDK-aligned candidate classifies `json.Number` as string during type checks
and rounds a large schema constant through float64. It produced 57 mismatches
with the lossless fixture projection and accepted the adjacent wrong integer
for a large constant. Those failures trigger the replacement branch already
approved in D04. Keep the adapter boundary and use the qualified
`santhosh-tekuri/jsonschema/v6 v6.0.3` implementation. All original fixture
expectations and precision assertions pass with it.

Compile schemas with Draft 2020-12 explicitly selected and external loading
restricted to the manifest's local registry. Retain exact numeric values in
schema and instance projections. Add the qualified URI, URI-template, and base64
semantic handlers at the pinned protocol locations. Do not turn format-annotation
into an assertion vocabulary for arbitrary application schemas without an
explicit policy. Apply property-specific normative rules, such as the `file://`
restriction on `Root.uri`, separately from generic URI syntax.

The generator must reject new unsupported assertion behavior before publication.
This covers the pinned schema and explicitly supported extension inputs; it
makes no claim that a finite test suite proves every possible JSON Schema.
The replacement uses Go regular expressions by default. New schemas containing
patterns require an explicit compatible expression profile or a qualified
ECMA-262 engine; silently accepting divergent regular-expression semantics is
forbidden. No pattern assertion occurs in the qualified protocol input.

#### G20 resolved with measured costs and numerical gates

The report measures the same upstream JSON-RPC decoder with and without strict
raw checks at 1,069, 65,581, 262,189, and 1,048,493 bytes. Five samples per case
were collected on Darwin/arm64 and Windows/amd64. A second Windows run used
GOMAXPROCS=1 after substantial timing variation. All samples are retained.
The checks add cost; the larger payloads are not proof of a codec speedup.

The baseline figures qualify the boundary mechanism. The following values are
the concrete design's implementation acceptance contract, authored in response
to the owner's gap-closure request. They are distinct from an assertion that
the complete library or generator already meets them.

| Gate | Numeric acceptance rule | Measurement |
| :--- | :--- | :--- |
| Strict raw check plus SDK decode | Added median time at most `5,000 ns + 12 ns × frame bytes`; added median allocation at most 8 KiB and 16 allocations per operation | Same host/toolchain/fixture/settings; all twelve measured run/size combinations pass |
| Canonical no-op RPC assembly | Median time at most `1.25 × reference + 50 µs`, added allocation at most 8 KiB and 16 allocations per operation | Reference uses the same schema checks, raw checks, SDK, and transport; run stdio and HTTP at concurrency 1 and 3 |
| Pinned-schema cold compilation | All 155 definitions in at most 5 seconds, without network access | Fresh compiler; the complete functional probe containing this work took 0.588 seconds on the assessed Darwin host |
| Full pinned-input generation | At most 30 seconds and 256 MiB peak resident memory for the generator process; Go compiler child costs reported separately | Current 155-definition input, fixed toolchain and output inventory; future input expansion requires a reviewed budget change |
| Cancellation response | At most 250 ms from cancellation or terminal validation failure until the canonical operation returns in controlled no-op fixtures | Stalled-reader, admission-wait, and resumable-SSE cases; handlers that ignore cancellation are reported separately |
| Shutdown | At most 5 seconds for the framework's bounded shutdown call | Return an explicit incomplete-shutdown error if user code or an external resource does not cooperate |

Budgets are evaluated per supported platform using at least five retained
samples and the same settings for both paths. The gate records CPU, Go version,
concurrency, payload hash, and every sample; it does not silently discard slow
samples or waive failed correctness. The reference for the RPC budget includes
all required enforcement so cost differences cannot be hidden by skipping it.
The budget calculation accepted twelve observed run/size combinations and
rejected a deliberately excessive benchmark result.

The initial limits are explicit application policies, not values required by MCP:

| Limit | Default | Configuration and scope |
| :--- | :--- | :--- |
| One JSON frame or JSON HTTP body | 4 MiB (SDK default) when undeclared; 16 MiB declared ceiling; above-ceiling declarations configurable to a 64 MiB hard cap | Declarations are explicit and positive; both receive and send boundaries; oversize HTTP input maps to 413; nothing above 64 MiB is constructible |
| One SSE event | 4 MiB including framing fields; 8 MiB declared ceiling; configurable to a 16 MiB hard limit | Declarations are explicit and positive; independently bounded from stream lifetime; nothing above 16 MiB is constructible |
| JSON nesting depth | 64 | Positive values up to 256; checked before recursive validation |
| Numeric token length and exponent magnitude | 256 bytes and 1,024 respectively | Explicit parser resource limits before arbitrary-precision validation; report a resource-limit error, not schema success |
| Active application handlers | 32 per server instance | Configurable; independent of future HFSC's concurrency-3 workload |
| Queued application calls | 32 per server instance | FIFO admission; at most 1 second waiting or the earlier caller deadline; overflow returns an explicit overload error |
| Framework-owned live wire buffers | 64 MiB per runtime instance | Account allocated capacity across directions; acquisition is cancellable; this does not claim to cap user-handler or SDK-internal memory |
| Retained buffer pool | 8 MiB total, 256 KiB per buffer | Larger buffers are released; bounded lifetime and ownership tests |
| Schema input | 2 MiB per document, 4,096 definitions, 65,536 reference edges | Explicit acquisition/model limits; reject overflow before emission |

> **Amendment 2026-11-25 (body-size policy).** The original policy set a
> 1 MiB default with a 16 MiB configurable ceiling for HTTP JSON bodies. The
> owner ruled the 1 MiB default too small for real payloads: the undeclared
> default now follows the pinned SDK's own 4 MiB, and an explicit declaration
> by the server transport or the consumer may raise the limit to at most
> 8 MiB. The 16 MiB ceiling is superseded for HTTP JSON bodies; declarations
> above 8 MiB are rejected at construction. SSE events remain independently
> bounded per the row above.
>
> **Amendment 2026-11-25 (second ruling, body-size policy).** Supersedes the
> first ruling above in the same sitting: undeclared default stays 4 MiB (SDK
> parity); a declared limit may go to 16 MiB; declarations above 16 MiB are
> configurable up to a hard cap of 64 MiB. Nothing above 64 MiB is
> constructible. The 8 MiB intermediate ceiling is withdrawn.
>
> **Amendment 2026-11-25 (SSE event bounds).** The original SSE policy set a
> 1 MiB per-event default with a 16 MiB configurable ceiling. The owner ruled:
> default 4 MiB, declared ceiling 8 MiB, configurable above-ceiling to a
> 16 MiB hard limit, nothing above 16 MiB constructible. Supersedes the SSE
> row of the original limits table.

Control messages and cancellation do not wait behind the application handler
queue. Long-lived subscription streams are bounded per event, not buffered in
full. Caller contexts govern legitimate idle streams. Future HFSC transfers
remain bounded by their separate file and concurrency contract, using parts
that fit these wire limits. None of these limits changes the MCP schema itself.

There are now zero unspecified outcomes for G08, G20, or G21. Their remaining
work is implementation and testing against stated rules, not further design
choices hidden behind “qualification pending”.

### Qualification diagnostic deviation on 2026-11-22

The scratch qualification harness needed a third build attempt. Attempt 1
failed on a benchmark syntax error; attempt 2 failed because its copied module
still declared Go 1.26 while the probe imports stable Go 1.27 JSON APIs.
Both were diagnostic setup errors. The owner instructed “resume” after the
third-build gate prompt. The selected correction updates only the scratch
module's Go floor to 1.27 and retains every assertion. This diagnostic work
is not product implementation or an implementation PLAN execution.

### Evidence already obtained

The original source inspection established the version/dependency baseline, mcplib's current
integration behavior, ACP's generator phases, its sequential publication model,
and its v1 JSON tests. Fresh upstream schema bytes matched the recorded digest.
ACP's existing generator drift test passed. At that assessment the framework
was unimplemented. The completion audit below records the later implementation
and its evidence limits; prototype results remain distinct from product results.

### Required qualification

Use independent schema validation and normative fixtures alongside generated
tests. A generator must not be its own sole oracle. Validate actual JSON frames
before normalization, including required fields, discriminators, absent/null
handling, case, duplicate names, unknown members, number boundaries, and result
shape. Exercise both stdio and HTTP for every advertised transport guarantee.

Pin a version of the official MCP conformance suite and inventory which cases
apply to the selected protocol and transports. Passing that suite supplements
local policy and integration tests; it does not prove application authorization
or bound every resource. Keep excluded cases explicit with reasons. Record
versions, commands, exits, fixture hashes, and failing raw frames with redaction.

Every new gate must be shown to fail on deliberately broken input in a scratch
copy. Required plants include a dropped required field, incorrect discriminator,
accepted duplicate JSON member, unsupported schema assertion, missing capability
rule, manually altered generated output, and cross-platform compile failure.
Publication fault injection must demonstrate mixed-output detection and recovery.
Do not mutate the working tree to plant these failures.

### B01 pass-2 outcome and Option D ruling on 2026-11-24

Pass 2 executed the approved Option A qualification in throwaway scratch
modules against the pinned SDK v1.8.0; evidence is recorded in
[0002-REPORT-stage-0-framework-inventory.md](../reports/0002-REPORT-stage-0-framework-inventory.md).
The server role passed both required scenarios with zero deprecated symbols
and zero lint findings, and elicitation passed on both axes. The client role
failed the modern-only composition: sampling fulfillment errors without a
deprecated handler, roots return the built-in default empty list without
`AddRoots`, receiving middleware is bypassed by the SDK fulfillment
dispatcher, and the sealed `InputRequest`/`InputResponse` interfaces admit no
application-owned implementations. A deprecated control copy passed
functionally for all scenarios and was flagged by the pinned golangci-lint
with three unsuppressed SA1019 findings. Inspection of v1.9.0-pre.1 in the
module cache shows identical SEP-2577 markers; a re-pin cannot change the
outcome, and a from-scratch SDK would share it because the conformance pin,
not the SDK, mandates the scenarios.

The owner selected **Option D: codec-mediated fulfillment wrapper**. The
framework owns the client fulfillment loop
(`MultiRoundTripOptions.Disabled`), exposes its own public
`InputRequest`/`InputResponse` DTOs emitted by the schema generator, and
converts at the SDK seam exclusively through the non-deprecated public codecs
`InputRequestMap`/`InputResponseMap`. No deprecated symbol is named in
source, public API, or lint scope; the SDK's deprecated concrete types are
constructed only inside the SDK package by its own codecs, during the
SEP-2577 deprecation window. The owner explicitly accepts that runtime
traversal. Dropping the two scenarios (Option C) is recorded as the fallback
only; executing it requires a fresh ruling because it amends the frozen
conformance requirements.

Consumer scope recorded with this ruling: go-mcp-lib deprecates the SDK
functionality of the mcplib project. The first consumer is gobble, which
will consume MCP servers as extensions (client role) and expose itself as an
MCP server to IDE, CLI, and other agent harnesses (server role). This scope
makes Option C a functional regression for the client role during the
deprecation window, and it makes the wrapper's public DTOs product API
rather than test scaffolding.

Pass 1 failed environmentally and pass 2 failed substantively, so the pass-3
proof is the third qualification attempt. The owner's 2026-11-24 instruction
to scope and spec Option D is the recorded authorization required by the
deviation process. Pass 3 remains scratch diagnostic work; it authorizes no
product source, dependency change, or implementation stage.

### State table

The D01–D06 table above records the owner's selected policies. The G rows
below record detailed behavior and evidence requirements. `decided` does not
mean implemented or tested, and does not authorize execution. The qualification amendment resolves the former open evidence rows with
measured results and concrete design limits. Production tests remain
implementation acceptance obligations, not unnamed architectural decisions.

| ID | Input or state | Ruling and acceptance evidence |
| :--- | :--- | :--- |
| G01 | Generator implementation language | decided — owner requires pure Go; test generator module builds and generation needs no Python |
| G02 | Valid pinned inputs, unchanged output | decided — identical bytes and file inventory; deterministic regeneration test |
| G03 | Missing, tampered, or wrong-dialect source | decided — fail before publication; manifest/hash/dialect negative fixtures |
| G04 | Recursive schema reference | decided — resolve as graph with bounds; recursive and over-limit fixtures |
| G05 | Unsupported assertion, missing mapping, or unknown vocabulary | decided — fail with source pointer; vocabulary and method coverage plants |
| G06 | Required, optional, null, zero, empty, union, or open-object values | decided — preserve schema distinctions; independent raw serialization fixture matrix |
| G07 | Duplicate JSON names, invalid UTF-8, or oversized owned payload | decided — reject before owned-value decoding; boundary and limit fixtures |
| G08 | Full SDK wire-frame interception for strict JSON policy | decided — reader/body interception precedes SDK decoding on stdio and both HTTP response modes; fatal SSE latch preserves the original connection; valid, malformed, resumable-event, and disabled-guard controls passed as documented in the qualification report |
| G09 | Standard SDK type or owned DTO uses a custom marshaler | decided — preserve intended wire contract; v1/v2/SDK differential and captured-frame fixtures |
| G10 | Invalid tool schema or registration | decided — fail registration without permissive substitution; declaration and no-handler-invocation tests |
| G11 | Optional feature or extension not enabled | decided — do not advertise or invoke it; capability-negative fixtures |
| G12 | Legacy initialize or downgrade attempt | decided — reject under modern-only policy; raw client/server fallback tests |
| G13 | Concurrent servers register identical tool names | decided — admission remains instance-scoped; race and independence tests |
| G14 | Cancellation, slow peer, queue exhaustion, handler ignoring cancellation | decided — bounded waits and explicit outcomes; leak/admission/shutdown tests |
| G15 | Different principals, revoked permissions, or expired cache entry | decided — no cross-scope reuse; isolation/expiry/retention tests before enabling cache |
| G16 | Generator compile failure, name collision, or platform mismatch | decided — no publication; internal/external test and platform compile plants |
| G17 | Mid-publication failure, stale lock, or manual output edit | decided — reject mixed tree and recover without unrelated loss; fault-injection and ownership tests |
| G18 | Deprecated SDK symbol requested by old HFSC signature | decided — no compatibility adapter or Staticcheck suppression; API inventory and lint tests |
| G19 | Artifact parts and HFSC routing implementation | out of scope — separate workload design and migration; framework exposes resource/extension primitives |
| G20 | Quantitative performance budgets and default quota values | decided — use the measured baselines, numerical gates, and explicit defaults in the qualification amendment; twelve run/size budget cases pass and an over-budget plant fails; full-framework results must be measured during implementation |
| G21 | Runtime validator's complete required vocabulary | decided — replace the failing SDK-aligned candidate under D04; the qualified v6.0.3 adapter compiles all 155 definitions and passes 421 assertion fixtures, 36 format fixtures, and precision cases; vocabulary changes require renewed qualification |
| G22 | Public stable API and embedding of generator internals | decided — pre-v1 framework initially, engine internal; API manifest and import-boundary tests |
| G23 | Client-role input fulfillment for sampling and roots | decided — owned fulfillment loop behind `MultiRoundTripOptions.Disabled`, converting only through the public `InputRequestMap`/`InputResponseMap` codecs; deprecated SDK symbols never named in source or API; runtime traversal inside the SDK package accepted for the SEP-2577 window; the pass-3 scratch proof is the acceptance evidence |
| G24 | SDK internal structural discriminator (the response codec probing `roots`/`role`/`action` keys) | decided — accept the coupling without suppression and pin it with the conformance-scenario tests required by the qualification amendment, so a probe change fails loudly rather than silently |
| G25 | SDK removes, or the conformance suite drops, the deprecated input features | decided — delete the codec bridge and the generator's sampling/roots DTOs; the wrapper's public API is shaped so this removal breaks no consumer signature |

### Completion-audit repair amendment — 2026-11-24

The owner selected option A after reviewing
[0011-REPORT-pre-stage-7-completion-audit.md](../reports/0011-REPORT-pre-stage-7-completion-audit.md).
At revision `5bfdd9f5e4a6e9286cf593fdc1f3ccec60890532`, the audit identifies
20 findings across Stages 0–6, including executed policy bypasses, malformed-JSON
acceptance and out-of-destination deletion in scratch publication/recovery.
Existing Linux tests passing does not override those failures. Historical
completion reports remain evidence of what ran, not proof of unmet exits.

Chosen resolution: repair and requalify the existing design before Stage 7.
This restores D01–D06 and G01–G25; it does not select a fork, second dispatcher,
reduced feature set, relaxed lint policy or new protocol baseline. Architecture
status remains accepted. Following review of the repair sequence, the owner
authorized R1 only with “resume r1”; R2–R5 remain pending execution approval.
No repaired behavior or stage completion is claimed by this record.

* Good, because each demonstrated failure becomes an explicit production-path
  acceptance case, with independent controls and retained failure evidence.
* Good, because the existing pins, Option D fulfillment, modern-only lifecycle
  and platform requirements remain consistent for consumers.
* Bad, because completing the earlier exits requires substantial generator and
  runtime work before transport implementation can resume.
* Neutral, because the unpublished pre-v1 API may change to remove forbidden
  SDK exposure; consumer migrations and compatibility bypasses remain excluded.

The following state rows refine existing requirements; they are not new
permissions or claims of passing tests. Each maps to exactly one repair
acceptance group in the PLAN. A group may contain multiple named subcases.

| State | Inputs or failure states | Required outcome and acceptance group |
| :--- | :--- | :--- |
| RA01 | Historical completion claim, missing obligation or missing scenario owner | decided — retain history, mark unmet exits incomplete, reconcile every applicable requirement; `TestRepairA01Evidence` |
| RA02 | Complete result stream; absent, malformed, duplicate, cached-only, skipped, failed or unterminated required result | decided — accept only attributable fresh complete execution; unexpected skip/incomplete evidence fails; `TestRepairA02GateAccounting` |
| RA03 | Generated or handwritten policy violation; unformatted staged source | decided — actual lint detects both, pre-add reports without rewriting, no suppression; `TestRepairA03LintAndPreAdd` |
| RA04 | Prepared/unprepared or tampered tools/runner; network permission absent/present; setup interrupted | decided — verified explicit setup, bounded acquisition, ordinary offline checks fail missing preparation without downloading; `TestRepairA04Preparation` |
| RA05 | Valid JSON, malformed number/escape, duplicate, trailing document, oversized input or diagnostic overflow | decided — strict lossless load with bounded diagnostics, no panic/normalization of invalid input; `TestRepairA05StrictLoader` |
| RA06 | Recursive graph, cross-document base, invalid pointer, missing target, traversal, symlink escape or exceeded budget | decided — valid cycles retained as edges; invalid/unregistered/escaping references fail before emission; `TestRepairA06References` |
| RA07 | Supported assertion, annotation, unqualified assertion or required vocabulary | decided — preserve/delegate qualified assertions, retain annotations, reject unsupported required semantics; `TestRepairA07Vocabulary` |
| RA08 | Valid/missing/changed source pin, semantics rule or SDK disposition | decided — require independent verified input identities and total reviewed mapping; changed source cannot bless changed output; `TestRepairA08ProvenanceAndMapping` |
| RA09 | Exported alias or owned signature reaches deprecated member, raw SDK runtime object or sealed SDK implementation | decided — audit transitive fields/methods/signatures; reject forbidden exposure and use owned conversions; `TestRepairA09AliasClosure` |
| RA10 | Object/non-object definition, open object, presence distinction, union or naming collision | decided — every definition has an explicit verified disposition; preserve wire semantics and reject collisions; `TestRepairA10GeneratedContract` |
| RA11 | Valid candidate, broken handwritten/internal/external/platform source, stale owned output or missing local dependency | decided — compile complete candidate offline with CGO off and readonly modules before publication, failure changes no destination; `TestRepairA11CandidateQualification` |
| RA12 | Owned/new/stale path, traversal, symlink, unowned collision or intervening edit | decided — verify containment, identity and ownership before mutation, preserve unrelated bytes, reject conflict; `TestRepairA12PathContainment` |
| RA13 | Publication interruption, disk/permission/sync failure, cancellation, live/stale lock or corrupt journal | decided — preserve verified original/candidate bytes, detect mixed generations, serialize recovery, refuse unverifiable ownership; `TestRepairA13Recovery` |
| RA14 | Valid/invalid CLI arguments, semantic change, mutable provenance or candidate/live-input collision | decided — exact exit/format contracts, full semantic/API diff, immutable bounded candidate-only update; `TestRepairA14CLI` |
| RA15 | Valid raw value, empty/truncated/multiple values, invalid Unicode or limit boundary | decided — require one complete valid value; depth 64 default/256 maximum, frozen frame/numeric bounds; `TestRepairA15RawCodec` |
| RA16 | Owned/SDK codec variation, annotation-only format, mandated location, unqualified regex or validator bypass | decided — native v2 owned behavior, actual SDK differentials, one production validator policy, no arbitrary format assertion, reject unqualified patterns; `TestRepairA16ValidationProfile` |
| RA17 | Valid/malformed URI, non-file root, supported template or noncanonical base64 at a mandated location | decided — enforce the qualified location-specific rules through production validation, using the pinned template parser; `TestRepairA17Locations` |
| RA18 | Allowed/denied call, repeated middleware installation, short-circuit, transformed or absent required output | decided — authorization/admission before user side effects, observation surrounds operation, validate final results, install each link once; `TestRepairA18LiveEnforcement` |
| RA19 | Standard method, disabled capability, input-required result, bad discriminator or external consumer connection | decided — canonical owned API reaches every required/advertised method, Option D handles real callbacks, no raw SDK escape; `TestRepairA19CanonicalRuntime` |
| RA20 | Invalid declaration, expected SDK declaration panic, programmer fault, mutable option input or sensitive error cause | decided — return classified declaration errors, propagate programmer faults, deep-copy owned configuration, bound/sanitize public messages; `TestRepairA20RuntimeContracts` |

Review of missing cases also identified migration and integration obligations:
old hashless manifests must never be trusted as authority to overwrite output;
recovery must preserve manual edits even for originally new files; format
annotations must not become universal assertions; a transport package boundary
must not expose an SDK handle; and corrected middleware must validate results
after user transformations. The PLAN includes each explicitly. A new behavioral
choice discovered during repair still invokes the deviation process.

CI01–CI08 and all numeric budgets remain unchanged. Repairing local gates does
not advance Stage 9 or authorize push/tag/release, GitHub enforcement, scanner
installation, guard changes or consumer migration.

### Repair research clarification — 2026-11-25

Execution update: the owner subsequently approved execution of R1–R5 and
commits to main between coherent phases. This supersedes the earlier pending
execution statements; it does not establish that any repair exit has passed.

Subsequent deviation rulings:

| Ruling | State and required outcome | Acceptance owner |
| :--- | :--- | :--- |
| D1 A — shared Windows transfer | decided — repair native archive extraction in the canonical shared runner; preserve actual relative symlinks and reject unsafe archives, retain queue/copy verification | RA04 preparation; maintained `TestWindowsArchive` and existing shared dispatch regressions |
| D2 B — broader R1 review | decided — review the whole R1 gate/preparation/inventory path first, then include formatter-only correction and pinned tool preparation; authorize R1 qualification passes 3 and 4 | RA01–RA04; PLAN execution evidence |
| Package documentation A | decided — add handwritten `protocol/doc.go` within R1 to satisfy package documentation lint while preserving generated ownership and behavior | RA03 actual lint and pre-add |
| Corpus binding A | decided — bind fixture expectations to named definitions; reject ambiguous bindings and check explicit zero bounds without reducing existing assertions; authorize R1 passes 5 and 6 | RA01 evidence and RA02 trustworthy execution; corpus and scratch controls |

The owner requested reassessment of R1–R5, implementation options and an updated
PLAN. Read-only review of the current product source, unfinished R1 changes,
pinned SDK and primary Go documentation supports retaining the existing repair
outcomes. This clarification records evidence and recommendations, not a new
owner ruling or successful repair execution. The accepted architecture and
RA01–RA20 state outcomes remain unchanged. R1 is authorized but unqualified;
R2–R5 execution is still pending approval.

The paired PLAN's
[research refinement](0002-PLAN-build-schema-driven-mcp-framework.md#research-refinement--2026-11-25)
contains three evaluated options per repair, recommended steps, affected
components, integration order and additional cases under the existing groups.
The recommended direction is:

* R1: finish the shared Python orchestration with Go-native check producers,
  independently expected acceptance cases and one owner per equivalent analyzer.
  Good, because result consumers can share one identified execution without
  reducing coverage. The current draft still needs accounting, environment,
  process-cleanup and preparation fixes; successful setup is not qualification.
* R2: use standard JSON syntax parsing and Go AST/type information with explicit
  module/export metadata. Good, because it preserves the accepted standalone,
  standard-library-only generator. Bad, because Go recommends `go/packages`
  for general module-aware loading, and the constrained export-data approach
  needs additional qualification. A new developer-tools module or generator
  dependency remains an alternative requiring approval, not an implicit change.
* R3: use rooted filesystem operations with verified original/candidate bytes
  and a recovery journal. Good, because containment and recoverability become
  explicit. Neutral, because rooted access does not establish file ownership,
  and portable file replacement does not guarantee multi-file atomicity or
  power-loss durability.
* R4: use standard `jsontext` parsing, owned stable JSON v2 codecs and the single
  canonical validator profile. Good, because it avoids a second handwritten
  JSON grammar while retaining exact-number and protocol-specific assertions.
  Arbitrary application formats remain annotations under the accepted policy.
* R5: compose owned typed adapters with the SDK's public low-level registration
  and existing dispatch/session machinery. Good, because strict owned decoding
  and final validation have explicit integration points. Bad, because the
  wrapper must qualify inference, defaults, content/error construction and
  multi-round-trip behavior previously supplied by generic SDK helpers.

The evidence refines four implementation assumptions:

1. JSON v2 is enabled by default in Go 1.27. The existing floor/preferred pins
   therefore support standard-library `encoding/json/v2` and
   `encoding/json/jsontext`; no experimental-enable flag or third-party JSON
   package is needed. Stable API differences from earlier experimental examples
   must be respected. See the [Go 1.27 notes](https://go.dev/doc/go1.27) and
   [migration guide](https://go.dev/doc/jsonv2-migration).
2. Go's `os.Root` closes classes of traversal races when operations remain
   rooted, but ownership, symlink policy and journal validation are separate
   obligations. `os.Rename` does not guarantee atomic replacement on non-Unix
   platforms. RA12/RA13 require tested containment, mixed-generation detection
   and verified recovery; an unqualified universal atomicity/durability claim
   cannot satisfy them. See [rooted operations](https://go.dev/blog/osroot) and
   the [`os` reference](https://pkg.go.dev/os).
3. In SDK v1.8.0, `(*Server).AddTool` is a public composition point that leaves
   raw argument conversion, validation and result construction to the caller.
   Generic `AddTool` additionally handles inference/defaults, but reports some
   declaration failures through untyped panics. No exported typed panic category
   was found in the inspected path. RA20's expected declaration errors should
   be returned by owned prevalidation; it does not authorize panic-text matching
   or swallowing arbitrary programmer faults. Any unavoidable conversion needs
   a demonstrated narrow boundary and review. See the
   [pinned registration implementation](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/server.go).
4. SDK receiving middleware runs after SDK request parsing. It can enforce
   owned dispatch policy and raw tool-argument validation, but cannot prove
   strict handling of information already lost in outer wire decoding. R5's
   live-path cases and Stage 7's transport/raw-frame cases remain separate.
   Option D continues to use SDK public codecs with automatic fulfillment
   disabled; no fork or replacement dispatcher is recommended. See the
   [pinned fulfillment implementation](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/mrtr.go).

No source, dependency, hook, skill or other-repository mutation is part of this
research amendment. Performance improvements remain hypotheses until equivalent
before/after executions demonstrate them. The PLAN retains all native targets,
the additional WSL runner, CGO-free ordinary operation and race-only CGO.

## Pros and Cons of the Options

### Generated framework composed with upstream SDK

* Good, because it combines reproducible schema governance with upstream
  protocol maintenance and gives extensions an explicit contract.
* Bad, because the SDK/validator/codec boundaries must be tested rather than
  assumed to match.

### Thin aliases and handwritten wrappers

* Good, because initial implementation and dependency maintenance are small.
* Bad, because it leaves schema/API drift and inconsistent policy enforcement
  largely manual and does not meet the generator priority.

### Independent generated SDK

* Good, because it would give complete control of serializers and runtime APIs.
* Bad, because generation cannot supply protocol semantics by itself, and a
  second transport/runtime implementation would materially expand maintenance
  and conformance scope beyond the requested upstream SDK provider.

### Maintained upstream fork

* Good, because internal codec changes could be made directly.
* Bad, because the project would own merge conflicts, security update intake,
  and divergence of an entire SDK merely to supply enhancements.

## More Information

### Amendment 2026-11-27 — Functional runtime placement accepted

The accepted [0014-MADR-organize-mcp-runtime-package.md](0014-MADR-organize-mcp-runtime-package.md)
and approved PLAN supersede this record's original root-runtime placement.
The canonical runtime is `mcp/`, with its runtime-only private bridge at
`mcp/internal/sdkbridge/`; codec, generated protocol and shared root validation
retain their responsibilities. Consumer runtime imports add `/mcp`. The module
identities, runtime semantics and generation pins remain unchanged. Original
path references retain their historical context; current implementation
locators and the R4/R5 continuation use the functional paths.

### Amendment 2026-11-27 — Hook-discovery repair allowance accepted

The owner selected A after the pass-6 hook-discovery defect. The canonical
pre-add entry already required by ML01 must include a literal target declaration
recognized by the installed hook. Keep the shared dispatcher and prove that
standalone and combined Make goals still execute it once. The earlier pass-6
limit below records the historical approval; this later ruling extends it once.

| Case | Input or state | Owner ruling and verification |
| :--- | :--- | :--- |
| ML01 hook integration | Shared-only target is invisible to the installed hook | decided — authorize R3 pass 7 for the literal target, a broken scratch discovery control, actual standalone/combined dispatch and native qualification; no hook bypass or change, no pass 8. |

### Amendment 2026-11-27 — Canonical Make foundation and lint repair accepted

The owner selected A from the Makefile/prerequisite repair prompt after requiring
`make lint` to pass. A thin Makefile now belongs to the R3 prerequisite scope:
it dispatches to the existing Python check owners, including a real
`pre-add-check` target discoverable by the installed agent hook. This does not
advance the full Stage 9 receipts, security integration or readiness graph.
No shell orchestration, global-hook change or upstream-forge configuration is
included. The existing accepted architecture and historical rulings remain.

Preserve the legacy lint findings through explicit modern documentation/naming
rules in the pinned lint stack and fix their producers. Generated code remains
analyzed, with no exclusions or suppressed assertions. Correct generated marker
placement and method documentation in the emitter, never by editing owned
output. Review and qualify the resulting new output hashes before publication;
preserve the prior reviewed inventory as evidence. Input/profile/SDK/schema pins,
public signatures and runtime semantics remain fixed. This precedes the accepted
[0014-MADR-organize-mcp-runtime-package.md](0014-MADR-organize-mcp-runtime-package.md)
relocation; its unchanged-generated-bytes requirement uses this qualified
post-repair baseline.

The prescribed broken-input probes found that golangci-lint 2.15.0's Revive
adapter sets `ignore-generated-header` false internally, which makes Revive
1.17.1 omit generated files even with golangci exclusions disabled. Use
standalone Revive at that same version as the sole owner of these four rules,
with `ignore-generated-header = true`. Keep the other analyzers in the pinned
golangci runner. This is a correction to the execution mechanism inside ML02,
not permission to drop a finding class or add an exclusion. Tool versions and
the builder are verified during explicit preparation and normal lint.

| Case | Input or state | Owner ruling and verification |
| :--- | :--- | :--- |
| ML01 | Make entry points, independent modules, or combined targets | decided — one Python implementation and one matching leaf per invocation; both modules accounted; Make selector/dispatch tests and a deliberately failing scratch target. |
| ML02 | Missing handwritten/generated export docs, package docs, naming or blank-import explanation | decided — modern lint fails without changing public signatures; fix source/emitter and prove each rule catches a broken scratch input, including generated files. |
| ML03 | Marker after package clause, or undocumented generated methods | decided — emitter produces a standard leading marker and method docs; parser-based header/doc checks reject the previous malformed placement and an omitted method comment in scratch. |
| ML04 | Changed generated byte identity after the documentation repair | decided — retain the previous review, capture and qualify the new inventory, then use verified publication; no handwritten output patch, pin substitution or hash-only blessing of current files. |
| ML05 | Heavy Make target, missing fleet wrapper/tool or unsupported mode | decided — use the existing host selection/queue contract, preserve diagnostics and nonzero failures, keep normal CGO off, and leave unimplemented product readiness nonzero. Make/runner failure controls and native qualification. |

The owner approved the focused A scope, not the larger validation-graph option.
The existing R3 allowance ends at pass 6; this ruling does not grant pass 7.

### Source map for review

Local code references are reproducible using repository, revision, path, and
symbol. They avoid machine-specific paths and links outside this documentation
tree. Line references are anchors for this snapshot and must be re-grounded.

| Evidence | Locator |
| :--- | :--- |
| mcplib dependency and registration behavior | Revision above, `go.mod`, `hardened.go`, `schema/schema.go` |
| mcplib lifecycle and deprecated HFSC surface | Revision above, `server.go`, `hfsc/dispatch.go`, `LogSession` |
| ACP schema loader and semantics | Revision above, `cmd/generate/internal/load/load.go`, `cmd/generate/internal/model/protocol.go` |
| ACP AST generation and publication | Revision above, `cmd/generate/internal/emit/file.go`, `cmd/generate/internal/output/check.go`, `cmd/generate/internal/output/write.go` |
| ACP reproducibility and JSON behavior | Revision above, `cmd/generate/main_test.go`, `json_v1_pin_test.go`, `json_parity_test.go` |
| SDK private decoder | [v1.8.0 internal JSON source](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/internal/json/json.go) |
| SDK registration and version policy | [v1.8.0 server source](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/server.go), `AddTool`, `Server.AddTool`, `ServerOptions.SupportedProtocolVersions` |
| Protocol source authority | [MCP stable specification](https://modelcontextprotocol.io/specification/2026-07-28) and immutable schema input linked above |
| Stable v2 API and migration semantics | [Go 1.27 release notes](https://go.dev/doc/go1.27), [JSON migration guide](https://go.dev/doc/jsonv2-migration), and [versioned JSON v2 documentation](https://pkg.go.dev/encoding/json/v2@go1.27.4) |

All six owner decisions, qualification resolutions, Option D and CI/CD rulings
remain recorded. G23–G25 have scratch feasibility evidence; production acceptance
is still required. The owner selected the audit's repair option A on 2026-11-24.
The paired PLAN now contains R1–R5 and one acceptance group per RA row for review
before repair execution. Stages 0–6 are not recertified by this documentation
amendment, and Stage 7 remains behind their prerequisite exits. The original
companion report proves prototype results; the completion audit records actual
implementation gaps. Neither is a release qualification.
