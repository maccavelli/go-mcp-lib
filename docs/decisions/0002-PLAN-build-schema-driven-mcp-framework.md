---
status: accepted
date: 2026-11-23
associated-madr: 0002-MADR-build-schema-driven-mcp-framework.md
---

# Implement the schema-driven MCP framework around the upstream Go SDK

Associated MADR: [0002-MADR-build-schema-driven-mcp-framework.md](0002-MADR-build-schema-driven-mcp-framework.md).
Evidence: [0002-REPORT-qualify-mcp-framework-boundaries.md](../reports/0002-REPORT-qualify-mcp-framework-boundaries.md).

The owner approved this implementation plan on 2026-11-23 and reaffirmed
acceptance of the completed MADR, including its numerical acceptance contract.
No phase has executed. The owner explicitly instructed waiting for further
instructions; execution requires an instruction to proceed or execute named stages.

## Goal

Deliver a pre-v1, reproducible Go library and pure Go generator that provide the
modern MCP SDK through owned runtime handles, reviewed standard data aliases,
pinned schema enforcement, native JSON v2 for owned values, and bounded stdio
and Streamable HTTP assembly. Qualify the actual implementation against every
applicable MADR decision and state row. Produce a reviewable release candidate;
creating a tag, publishing a release, or migrating a consumer is not a stage here.

## Grounding Reference

The following sources were rechecked on 2026-11-23. Paths in other repositories
are locators at the stated revisions, not dependencies to copy wholesale.
The first execution step repeats this grounding against actual HEAD and dirty
files; line numbers locate the assessed snapshot rather than immutable APIs.

| Repository or input | Revision and locator | Fact used by this plan |
| :--- | :--- | :--- |
| go-mcp-lib | `f0af2c50ce21b471b335cc9d2767f00b05a58fc5`; `README.md`, `docs/architecture.md`, tracked file inventory | Scaffold only: no Go module, library, generator, Makefile, or CI. The MADR/report and index edits are uncommitted review work. |
| Canonical module | Configured origin `https://github.com/maccavelli/go-mcp-lib.git` | Root module path is `github.com/maccavelli/go-mcp-lib`; generator module adds `/cmd/mcpgen`. |
| Existing integration | mcplib `b4eac5280d785e2be47e057f3ac3ddca04535335`; `go.mod:9`, `server.go:14`, `server.go:37` | SDK is v1.6.1. A mutable session is stored on one server wrapper. Replacement tests must demonstrate independent connection routing. |
| Existing admission and schema handling | Same mcplib revision; `hardened.go:71`, `getToolMutex`, `schema/schema.go:27` | Tool-name locks are package-global; derivation can substitute an open object on error. Neither behavior is copied. |
| Deprecated HFSC interface | Same mcplib revision; `hfsc/dispatch.go:44`, `LogSession` | Exported logging signature uses the deprecated SDK type; no adapter or suppression enters this project. |
| Reference generator | acp-go-sdk2 `a6495a56ddf2c2503b4613a2f91fb9ca400ef61a`; `cmd/generate/go.mod`, `main.go:72`, `internal/output/write.go:23` | Separate standard-library-only Go generator; load/model/render/check/write phases; staged sequential replacements are not an atomic multi-file transaction. |
| SDK transport hooks | SDK v1.8.0; `mcp/transport.go:149`, `mcp/streamable.go:128`, `mcp/server.go:185` | `IOTransport`, outer HTTP handlers, and supplied HTTP round trippers are available. Restricting supported versions alone does not eliminate legacy initialization. |
| SDK HTTP policy | SDK v1.8.0; `mcp/streamable.go:174–224` | Localhost protection defaults on; the cross-origin option is deprecated; request cancellation propagation is opt-in; the SDK body default is 4 MiB, different from this design's 1 MiB. |
| Qualification evidence | Companion report, exact embedded Go source and dependency checksums | Public interception feasibility, lossless-validator failure/replacement, 421 assertion fixtures, 155 definitions, 36 format fixtures, and 120 benchmark samples. These are prototype observations. |
| Repository gates | `CLAUDE.md`, `.markdownlint-cli2.jsonc`, `scripts/check_records.py` | Documentation gates exist. Go gates and CI must be added. Records use the existing 0002 pair; no new sequence number is allocated. |

The upstream sources used for the additional tooling details are the
[golangci-lint v2.15.0 release](https://github.com/golangci/golangci-lint/releases/tag/v2.15.0)
and the immutable
[conformance package manifest](https://github.com/modelcontextprotocol/conformance/blob/96a6f9478b7f2e125cf0360a213383cf57533893/package.json),
[runner documentation](https://github.com/modelcontextprotocol/conformance/blob/96a6f9478b7f2e125cf0360a213383cf57533893/README.md),
and [2026-07-28 requirement set](https://github.com/modelcontextprotocol/conformance/blob/96a6f9478b7f2e125cf0360a213383cf57533893/requirements/2026-07-28.yaml).
The runner distinguishes required, unscored, and skipped checks. Its exit status
alone is insufficient evidence for this framework's complete advertised surface.

## Scope

### Modules, packages, and files to create

All paths in this table are planned deliverables, not existing implementation.
Use root package name `mcp`; callers can import the module with that conventional
local name. The generator's package is `main` and its engine remains internal.

| Path | Responsibility and dependency direction |
| :--- | :--- |
| `go.mod`, `go.sum`, `doc.go` | Runtime module, Go floor, patched toolchain, exact direct pins, package contract |
| `protocol/` | Generated reviewed data aliases, method/version constants, API and schema coverage identifiers; imports upstream data types, never root |
| `codec/`, `internal/rawjson/` | Owned JSON v2 API and bounded raw token checks shared by runtime adapters |
| `schema/`, `schema/inputs/`, `schema/manifest.json`, `internal/validation/` | Embedded immutable sources, declaration API, private replacement-validator adapter and semantic checks |
| `server.go`, `client.go`, `connection.go`, `registration.go`, `errors.go`, `options.go` | Owned runtime handles and typed top-level registration functions; no embedded raw SDK runtime object |
| `internal/sdkbridge/` | Explicit SDK conversions, method dispatch adaptation, version policy, first-failure propagation |
| `transport/` | Guarded stdio and HTTP assembly; accepts owned transport configuration; imports lower layers, never root |
| `extension/` | Namespaced extension descriptors and schemas; root consumes descriptors to avoid an import cycle |
| `internal/admission/`, `internal/observe/` | Instance-scoped quotas, lifecycle accounting, observation and redaction |
| `mcptest/`, `internal/contracttest/`, `testdata/` | Public test utilities, independent protocol fixtures, malformed frames and integration harnesses; production packages never import these |
| `cmd/mcpgen/go.mod`, `cmd/mcpgen/main.go` | Separate pure Go CLI module; no dependency on runtime packages or Python |
| `cmd/mcpgen/internal/{acquire,load,resolve,model,sdkmap,emit,qualify,publish,diagnostic}/` | Generator stages; standard library only initially; compilation uses the Go executable against scratch candidates |
| `cmd/mcpgen/testdata/`, `schema/semantics.json`, `schema/sdk-map.json` | Independent engine fixtures, normative protocol rules, explicit 155-definition dispositions |
| `generated/manifest.json`, `generated/api.json`, `generated/coverage.json` | Deterministic ownership, public API, vocabulary/method/capability coverage artifacts |
| `tools/versions.json`, `tools/conformance/`, `scripts/go-precheck.py`, `scripts/gate.py` | Pinned development tools and Python 3.12+ standard-library orchestration; product generation remains Go |
| `.golangci.yml`, `.github/workflows/verify.yml`, `.gitignore` | No-exception lint policy, both-module CI, ignored transient outputs |
| `examples/`, root README, architecture and guides | Compile-tested canonical API examples, installation, generation/recovery, transport and migration guidance |

Do not add a committed `go.work` to hide missing module requirements. Test each
module independently with `GOWORK=off`. Runtime consumers must not need the
nested generator module, development tooling, Node, or Python to build.

### Deliberate boundaries

The work includes modern resources, resource links, bounded payload primitives,
and generic extensions. G19 excludes a file store, artifact-part service, HFSC
wire implementation, MagicTools routing, and consumer migrations. Future HFSC's
50 MB/concurrency-3 contract is retained as context; this plan neither defines
its unresolved byte unit nor changes its scope by inference.

Do not implement an authorization server, CA, certificate issuance/rotation
service, process supervisor, telemetry vendor integration, or legacy MCP mode.
Expose application-supplied HTTP authentication and TLS configuration and test
the boundary. Keep response caching disabled unless its complete isolation,
revocation, expiry, and retention behavior is implemented and qualified; schema
compilation reuse is a separate bounded, immutable-input optimization.

## Prerequisites

### Frozen implementation inputs

| Input | Pin or requirement |
| :--- | :--- |
| Runtime and generator language | `go 1.27.0`; `toolchain go1.27.4`; test both exact versions |
| SDK | `github.com/modelcontextprotocol/go-sdk v1.8.0` |
| Canonical runtime validator | `github.com/santhosh-tekuri/jsonschema/v6 v6.0.3` |
| URI-template parser | `github.com/yosida95/uritemplate/v3 v3.0.2` |
| MCP schema | Revision `2026-07-28`, upstream commit `ac00313ff7f5e9f47b0e0d1ebdf9c0ad6411380b`; 181,976 bytes, 155 definitions |
| Schema SHA-256 | `907dcebce6aaeb16c21421ffef8c207093d88b9442fee340658a99cacb06b551` |
| Independent schema corpus | JSON Schema Test Suite commit `83e866b46c9f9e7082fd51e83a61c5f2145a1ab7`; preserve the report's selected fixture hashes and local references |
| Go lint | `golangci-lint v2.15.0`, built with Go 1.27.4; Staticcheck enabled, including SA1019 |
| Documentation lint | `markdownlint-cli2 0.23.2` |
| Official conformance source | Commit `96a6f9478b7f2e125cf0360a213383cf57533893`, package version `0.2.0-alpha.12`; use source and its lockfile together |
| Conformance requirement hash | `requirements/2026-07-28.yaml`: SHA-256 `ae2f4f6210fd729e2e318edd5bbfa31a43cee0bc608e48052fa26dbf1d939b57` |
| Conformance lock hash | `package-lock.json`: SHA-256 `e9d0405b0fbc905c962af951382780743a28dffa6ce9999eb9ec5cc86b28b348` |
| Conformance build tools | Node `26.13.0`, npm `11.20.0`; development-only pins observed locally; verify on each execution host |
| Supported targets | Darwin/arm64, Linux/amd64, Windows/amd64; native execution distinct from cross-compilation |

Pin development tool download checksums and CI action commit SHAs during the
explicit setup step. Record the resolved bytes in `tools/versions.json`; do
not guess those hashes in this document or use floating tags in normal gates.
The alpha conformance package is an external referee, not a production library
stability claim. The plan pins it by commit because its revision-aware coverage
is materially different from the older stable release listing.

### Execution discipline

* Re-ground before writes. Preserve existing dirty work. Verify the applicable
  Git branch/commit skill and global commit hook before the first phase commit.
* Run heavy checks through the available-dev-hosts selection and queue policy.
  Load that skill before suites, race, fuzz, or gate runs. Record host platform
  and CPU, never private host/account identifiers in committed evidence.
* Record stage/pass number, source revision and dirty-content digest, command,
  tool versions, exit code, complete redacted result artifact, and plants. A
  third build pass on a stage invokes the repository deviation gate.
* Each stage ends with no blocker, all cases assigned to that stage passing,
  and its deliberately broken controls caught in scratch. Commit only that
  approved phase after pre-add checks; no push or tag is implied.
* A changed architectural premise or failed requirement is a deviation: record
  evidence, present resolutions, amend the pair, and obtain the required ruling.
  Do not waive a check, broaden a budget, add an expected failure, or drop a
  transport to make a phase pass.

## Implementation Steps

Stages execute in order. Stages 2–4 establish the generator before broad runtime
features. Stage 5 supplies codecs and validation; stages 6–8 integrate them with
owned APIs and both transports. Stage 9 qualifies the combined system; stage 10
prepares review artifacts. A stage can exercise lower-level fixtures before the
final runtime exists, but cannot report a later integration obligation as passed.

## Stage 0 — Re-ground the contract and freeze the execution inventory

1. Recheck every grounding locator against HEAD and working-tree contents in
   go-mcp-lib, mcplib, and acp-go-sdk2. Record changed facts before implementation.
   Read the MADR, companion report, SDK v1.8.0 public signatures and private codec
   boundaries; confirm the evidence source hash in the report still matches.
2. Inventory all 155 definitions, standard methods, directions, capabilities,
   and normative behaviors from the pinned schema, TypeScript authority, and
   dated prose. Record a disposition for each definition and method; a missing
   binding or rule must have an explicit release-blocking outcome.
3. Inspect `go list -json` output and public source/method sets at the selected
   SDK version. Prepare an alias allowlist that includes field types, embedded
   members, inherited methods, transitive signatures, and deprecated comments.
   SDK server/client/session objects and HTTP option structs are owned wrappers,
   not blanket aliases. Include modern discovery, per-request metadata, multi-
   round input-required results, subscriptions, progress, and cancellation.
4. Read the exact conformance pin and classify every scenario by version, role,
   transport, feature, normative reference, and expected execution. Unsupported
   harness coverage gets a named local fixture, not a silent omission. Optional
   disabled capabilities must be tested for absence. A required advertised case
   cannot be marked inapplicable because it currently fails.
5. Record the inventory and hashes in the execution evidence report associated
   with this pair. Reuse this number with a distinct REPORT slug for execution
   evidence; retain the original prototype report unchanged. Resolve any
   architectural conflict through the deviation process before stage 1.

Exit: grounding matches or approved amendments explain differences; every D/G
row and advertised feature has a named verification owner; no blocker. This
investigation/report stage introduces no product gate, so no new plant is due yet.

## Stage 1 — Establish modules, tool pins, and honest gates

1. Create both module manifests with the exact paths and Go versions above.
   Add only required runtime dependencies and their verified sums. Keep the
   generator standard-library-only. Record upstream input and fixture licenses;
   no ACP code is copied without file-level provenance and required notices.
   Add the root package documentation and minimal CLI entry point so each module
   builds; unavailable commands return an explicit incomplete error until stage 4.
2. Add `tools/versions.json` and explicit setup orchestration for toolchain,
   dependency, fixture, and conformance preparation through
   `scripts/gate.py setup --allow-network`. Separate allowed network
   setup from offline execution. Prepare the conformance source at its commit
   with `npm ci` against the unchanged upstream lock, then build it. Hash the
   prepared runner and record its provenance; do not install it into runtime.
3. Implement `scripts/go-precheck.py` and `scripts/gate.py` in Python standard
   library, subprocess argument arrays only. Enumerate the root and nested Go
   modules explicitly; root `go test ./...` does not cover `cmd/mcpgen`.
   Capture command exits before parsing output, emit a named FAIL for every
   failed case, continue independent checks, and exit nonzero if any fail.
4. Configure gofmt cleanliness, vet, package tests, and pinned golangci-lint in
   each module. Enable Staticcheck with no symbol, file, test, or generated-code
   exclusion. Inspect generated files with the same policy; test fixtures that
   deliberately violate policy reside in scratch or noncompiled testdata.
5. Provide explicit gate modes `pre-add`, `unit`, `generate-check`, `api`, `wire`,
   `qualification`, and `plants`, with `--module runtime|generator|all` where
   applicable. An unimplemented stage or absent test inventory returns an
   incomplete/nonzero result; it must not be reported as a successful gate.
   Consume `go test -json` events to verify actual test execution and skips, not
   just the textual PASS summary. Setup does not claim test success.
6. Add tests for orchestration using controlled subprocess fixtures: nonzero
   child exit, missing module, missing required tool, wrong version, empty test
   selection, malformed result, and aggregation of multiple failures. Plant a
   deprecated SDK use in scratch and prove the actual lint gate catches it.

Exit: both modules build; foundational tests pass; no missing/empty gate reports
success; tools are pinned and prepared explicitly; foundational plants fail as
expected; no blocker. Coverage: G01/G18/G22 infrastructure, D03 foundation.

## Stage 2 — Build lossless schema loading, reference resolution, and mappings

1. Commit the verified schema and licenses under `schema/inputs/2026-07-28/`.
   Define versioned input manifest, semantics map, and SDK map formats with
   deterministic JSON encodings. Separate source manifests from generated-output
   ownership manifests; changing a source hash cannot legitimize manual output.
2. Implement `load` and `diagnostic`: strict JSON parsing, dialect validation,
   exact number tokens, duplicate/UTF-8 rejection, and stable diagnostics with
   phase, source URI, JSON Pointer, code, and bounded message. Limit documents
   to 2 MiB, definitions to 4,096, and reference edges to 65,536. Add a 256-step
   acyclic resolution-depth limit and a 1,000-diagnostic cap with a final
   truncation diagnostic; these are generator safeguards, not schema rules.
3. Implement local registry resolution of canonical URI plus fragment and JSON
   Pointer. Detect missing targets, duplicate canonical IDs, invalid pointers,
   encoded path traversal, and symlink escapes. Cycles become graph edges and
   must not exhaust depth merely by revisiting a node. Refuse network reference
   resolution even when a document contains an HTTP URI.
4. Build the intermediate model without collapsing required/optional/null/zero,
   number precision, enum openness, unions, compositions, or object openness.
   Inventory the 17 pinned keywords. Classify every keyword; annotations remain
   annotations unless an explicit protocol rule asserts them. Unsupported
   assertions and unknown required vocabularies fail with a source pointer.
5. Encode semantic rules with normative section, schema pointer, SDK symbol,
   direction, capability, completion, cancellation, retry safety, and fixture ID.
   Validate total mapping and distinguish standard methods from extension names.
   Describe current behavior, not ACP-specific `x-*` conventions or old MCP
   initialization semantics. Preserve unknown members only where schemas allow.
6. Build `sdkmap` from module-aware Go metadata and source ASTs. Generate scratch
   probes compiled by Go to verify type identity and method signatures; use
   source/type information for the deprecated/transitive API audit. Do not add
   a hand-rolled module importer or a second MCP dispatcher.
7. Add independent small schema fixtures for recursive references, nullability,
   conflicting compositions, numeric bounds, empty collections, reserved names,
   unsupported assertions, annotations, and extension vocabularies. Keep expected
   outcomes handwritten and provenance-linked rather than generated by the model.

Exit: all pinned definitions and methods have dispositions; supported fixtures
retain semantics; malformed/unsupported inputs fail before emission; dropped
required fields and missing capability/mapping plants are caught; no blocker.
Primary cases: G03/G04/G05, model portions of G06/G11/G16/G21.

## Stage 3 — Emit and qualify the reviewed Go/API contract

1. Implement `emit` with Go ASTs and `go/format`. Produce reviewed data aliases,
   constants, registries, schema bindings, owned DTOs/hooks where necessary,
   documentation tables, fixture scaffolding, and coverage/API artifacts. Every
   generated declaration identifies its source pointer and mapping revision.
2. Generate owned runtime façade signatures from the reviewed API contract;
   runtime-dependent emission is qualified against scratch handwritten fixtures
   until stage 6 supplies real implementations. Do not ship placeholder runtime
   implementations or treat fixture-only compilation as finished API behavior.
3. Check names against keywords, handwritten exports and methods, file names,
   case-insensitive filesystem collisions, and platform-specific declarations.
   Audit aliases transitively. No raw session alias or deprecated option field
   may become reachable through an exported framework signature.
4. Implement `qualify` using a complete scratch candidate tree including proposed
   generated files, handwritten source, internal tests, and external-package
   tests. Exclude stale owned outputs from the candidate by manifest ownership.
   Run Go with `GOWORK=off`, `GOTOOLCHAIN=local`, `GOPROXY=off`, and readonly module
   mode; missing prepared dependencies fail before publication.
5. Compile candidates for all three target platforms. Compile external tests
   as well as package sources; never attempt to execute a foreign test binary.
   Preserve build tags and module boundaries. Keep compile-only evidence
   separate from the native test matrix in stage 9.
6. Compare generated type identity and serialized representative values against
   independent fixtures and the schema adapter when available. Stage 3 qualifies
   emission structure; stage 5 completes runtime schema/codec assertions and
   stage 6 replaces all runtime fixture stand-ins with the actual assembly.

Exit: valid candidate outputs compile; incomplete mappings, transitive deprecated
symbols, wrong discriminators, export collisions, and a platform-specific compile
plant fail without publishing; no blocker. Primary cases: G09/G16/G18/G22;
G06 serialization completion remains assigned to stage 5.

## Stage 4 — Complete the CLI, deterministic publication, and recovery

1. Expose `inspect`, `check`, `generate`, `diff`, `update`, and `recover`.
   Common explicit arguments are `--repo`, `--manifest`, and `--format text|json`;
   commands reject unknown flags and positional arguments. Help exits 0, usage
   errors 2, failed validation/drift/incomplete recovery 1. Diagnostics go to
   stderr; machine-readable command results go to stdout with a format version.
2. `inspect` reports provenance, keyword dispositions, mappings, and unsupported
   features. `check` renders and qualifies in scratch, then compares bytes and
   owned inventory without changing destinations. `generate` executes the same
   checks before publication. All three require local prepared inputs.
3. `diff --from <manifest> --to <manifest>` reports semantic, vocabulary,
   capability, method, alias/API, and generated-inventory changes, including
   removals and stricter validation. `update --source <URL> --revision <commit>
   --sha256 <digest> --candidate <directory>` is the only network acquisition
   command; it produces a bounded candidate, never promotes live input or output.
   Require explicit immutable provenance, bounded reads/timeouts, local-path
   containment, and candidate directory conflict checks.
4. Acquire an exclusive per-repository generation lock. Stage candidate bytes,
   original owned bytes/hashes, and a journal on the same destination filesystem.
   Check ownership and existing hashes before replacements. Reject manual edits,
   unowned collisions, traversal, symlinks, and another active generator.
5. Publish changed files in deterministic order, remove only verified stale
   manifest-owned outputs, and publish the manifest last. Preserve unchanged
   modification times. Sync recoverable state before irreversible replacement;
   report filesystem sync limitations honestly. Sequential rename is not an
   atomic transaction, and readers/gates must detect mixed generations.
6. `recover --action inspect|complete|restore` validates journal paths, original
   and candidate hashes, and lock ownership. Inspection is read-only; complete
   and restore only operate on verified journal-owned files. Never steal a lock
   based solely on age; an unverifiable live owner or intervening manual edit
   produces a conflict diagnostic requiring operator resolution.
7. Fault-inject every publication boundary: journal write/sync, staging, rename,
   stale deletion, manifest promotion, cleanup, permission denied, disk-full,
   cancellation, simultaneous invocation, and stale lock. Kill a child generator
   mid-publication in scratch; prove `check` rejects the mixed tree and recovery
   restores or completes without changing unrelated sentinel files.
8. Regenerate twice from identical inputs, including shuffled map insertion and
   source-order fixtures, under different temporary directory names. Assert
   identical bytes/inventory and unchanged mtimes on the second run. Verify no
   timestamps, machine paths, account names, or network requests enter outputs.

Exit: every CLI command has positive and negative contract tests; repeat generation
is deterministic; interrupted publication is detectable and recoverable; manual
edit and mixed-tree plants are caught; no blocker. Primary cases: G01/G02/G17,
D03; completes publication obligations for G03/G05/G16.

## Stage 5 — Implement strict JSON v2 and the canonical validator

1. Implement owned codec entry points with stable `encoding/json/v2` and
   `encoding/json/jsontext`. Preserve absence/null/empty/zero as modeled; do
   not use a blanket float64 conversion or unconditional schema-default filling.
2. Check raw bytes before any lossy decode: duplicate and escaped-equivalent
   names, UTF-8, Unicode escapes, trailing values, depth, frame size, and numeric
   token/exponent bounds. Reject a numeric token over 256 bytes or exponent
   magnitude over 1,024 before arbitrary-precision validation. Test exact bounds,
   one past each bound, exponent signs/leading zeros, strings containing digits,
   and deeply nested adversarial input without stack exhaustion.
3. Implement the private v6.0.3 validator adapter with explicit Draft 2020-12 and
   a local reference registry. Compile declarations once per immutable identity;
   enforce retention bounds and concurrent access. Preserve exact numeric schema
   and instance values. Keep SDK-internal validation distinct from canonical
   validation; no decode/convert/decode path may round before enforcement.
4. Port and review the qualification report's checks into maintainable tests,
   retaining original inputs/expectations. Run all 421 selected assertion fixtures,
   all 155 definition compilations, 36 format fixtures, and exact/adjacent integer
   const/enum/bounds cases. Vendor required local reference fixtures with hashes
   and licenses so ordinary tests need no network or external module-cache layout.
5. Implement protocol-location URI, URI-template, base64 and property-specific
   rules such as `Root.uri`; distinguish format annotations from mandated
   assertions. Reject unqualified pattern/vocabulary support during declaration
   rather than silently applying incompatible regex semantics. Document the
   accepted application-schema profile and return an actionable schema pointer
   for unsupported inputs; do not claim universal JSON Schema support.
6. Differentially test owned values and reviewed SDK values under v1, v2, and
   the actual SDK codec path. Cover content unions, resource links, custom
   marshalers, raw values, nil arrays/maps, case, unknown members, number limits,
   and malformed result shapes. Exact equality is required only where the
   contract requires identical bytes; otherwise compare independently validated
   meaning while retaining raw bytes for duplicate/presence checks.
7. Remove a required assertion, round a large number, and bypass raw duplicate
   validation in separate scratch mutations; the real tests must fail. Confirm
   malformed declarations cannot become permissive schemas or successful
   registrations, and user handlers are never invoked for invalid input.

Exit: original qualification cases and expanded boundary matrix pass; canonical
numbers remain exact; adversarial plants fail; no blocker. Primary cases:
G06/G07/G09/G10/G21, D02 owned-codec and D04 validation portions.

## Stage 6 — Build owned runtime APIs and complete standard feature coverage

1. Implement owned Server, Client, and Connection handles with context-first
   blocking operations and immutable validated options. Keep the original SDK
   connection inside the bridge where needed; expose stable IDs and scoped
   request context, never a mutable global last-session routing authority.
2. Add top-level generic typed registration functions, since Go methods cannot
   introduce their own type parameters. Require explicit validated input/output
   schemas or a documented supported derivation path that returns errors. Check
   duplicate registrations and invalid handler signatures before serving. Text
   presentation is explicit and independent from structured output.
3. Complete the generated availability matrix with canonical operations for
   tools, resources/templates/links, prompts/completion, discovery/pagination,
   progress/cancellation, subscriptions, and supported elicitation, sampling,
   and root interactions. Bind modern input-required/multi-round result handling
   and request metadata where required by the pinned revision. Preserve SDK
   execution; do not invent a second dispatcher or legacy initialization flow.
4. Enforce schema and capability checks before invoking user code and validate
   owned outputs before emission. Optional features are advertised only when
   configured and usable. Validate pagination cursors and incomplete/invalid
   results against their selected contracts; no implicit success fallback.
5. Provide typed protocol, registration, tool, transport, cancellation, overload,
   resource-limit, and internal errors with meaningful `errors.Is`/`errors.As`.
   Use bounded sanitized public messages and private diagnostics; do not expose
   arbitrary panic values. Convert only proven expected SDK declaration panics
   at the narrow registration boundary; propagate programmer faults appropriately.
6. Specify and test middleware ordering: mandatory raw/schema/version enforcement
   cannot be disabled by user middleware; observation surrounds the operation;
   configured application authorization precedes side effects and optional cache
   reuse; admission precedes handler execution. Validate results after handler
   and transformation middleware. No bypass through convenience functions.
7. Prefer no raw SDK accessor in the initial API. If an integration accessor is
   required within the MADR's allowance, name its enforcement boundary explicitly
   and add bypass/guard tests before exporting it. Do not let an accessor's raw
   operations be counted as canonically enforced operations.
8. Replace stage 3 scratch façade fixtures with the real runtime, regenerate,
   and compile external consumer examples importing only go-mcp-lib. Test two
   servers with identical tool names and two simultaneous connections; block one
   and prove unrelated work completes and responses route to the correct caller.

Exit: every advertised standard method is reachable through the canonical API,
validated and capability-aware; no fixture stub ships; routing/isolation and
error tests pass and corresponding plants fail; no blocker. Primary cases:
G10/G11/G13/G18/G22, D01 and runtime G09 obligations.

## Stage 7 — Assemble both guarded transports and modern-only lifecycle

1. Promote the report's interception design into bounded production adapters.
   Stdio validates complete input frames in the `IOTransport.Reader` path;
   HTTP requests are validated in an outer handler before SDK parsing; supplied
   HTTP round trippers guard JSON bodies and complete SSE events. Apply checks
   symmetrically wherever the framework receives messages.
2. Validate outbound JSON frames and SSE data before releasing bytes. Preserve
   SDK flush behavior and documented HTTP interfaces in the response writer;
   never buffer an entire subscription lifetime. Catch malformed custom-marshaler
   output, oversized responses, and invalid result shapes before partial delivery.
3. Keep a sticky first validation failure per connection/operation and close the
   source on corruption. Install the failure callback that closes the original
   SDK connection; return that original object to preserve private SDK hooks.
   Handle failure before attachment, concurrent Close, and repeated errors without
   races or deadlock. Public operations return the canonical cause, not accidental
   EOF/retry wording. Terminal corruption must not trigger resumption.
4. Cover stdio split reads/writes, newline boundaries, multiple frames, EOF mid-
   frame, backpressure, broken pipes, cancellation, and ownership of reader/writer
   closure. Connection APIs do not launch or reap user processes. Process-owned
   example helpers must declare that ownership rather than hiding it in Serve.
5. Cover HTTP Content-Length and chunked/HTTP2 bodies, MIME handling, status/error
   mapping, body closure, failed round trips, JSON and SSE responses, multiple
   streams, POST-scoped subscriptions, and resumption after nonterminal network
   interruption. Distinguish replay of a stream from reissuing a tool side effect.
6. For SSE, test LF/CRLF/CR, fragmented delimiters, multiline data, comments, IDs,
   retry fields, empty events, event-name whitespace matching SDK classification,
   event size including framing, invalid UTF-8 and duplicates after a valid event
   ID. One-byte reads must expose zero invalid bytes. With retries enabled,
   terminal corruption must cause exactly one HTTP request.
7. Enforce modern-only policy at raw boundaries on both client and server. Test
   legacy initialize, missing/wrong revision, downgrade responses and SDK fallback,
   incompatible headers/metadata, legacy GET session streams and DELETE behavior,
   plus valid modern discovery/request metadata. Do not synthesize an initialize
   handshake for a stateless revision. Set supported versions but do not rely on
   that SDK option alone. Verify behavior with relevant SDK compatibility
   environment switches enabled; framework policy must not change implicitly.
8. Use owned HTTP options; do not alias the SDK's deprecated cross-origin field.
   Keep SDK localhost/DNS-rebinding protection, apply current outer cross-origin
   middleware, set the 1 MiB body policy explicitly, and enable modern request
   cancellation propagation. Test origin/Host rejection before handler invocation.
9. Accept application-owned `http.Client`, trust roots, server TLS and auth
   middleware inputs. Test verified TLS, private test CA, wrong host, untrusted
   issuer, expired fixture certificate, cancelled handshake, and redirect/header
   handling. Never set `InsecureSkipVerify` or distribute trust implicitly.
   Report certificate/trust failures as transport errors. Stdio resource examples
   must work with no listener, certificate, or CA provisioning.
10. Use both controlled readers/round trippers and real loopback sockets plus a
    subprocess stdio harness. The former test exact failure boundaries; the
    latter catch flush, closure, cancellation, and OS behavior that the scratch
    prototype did not qualify. Run external canonical clients and servers.

Exit: strict checks precede SDK decode in every receiving path; outbound guards
and lifecycle behavior pass; modern valid controls work and legacy/corruption/
TLS-trust plants fail; no blocker. Primary cases: G08/G12, transport G07/G09/G14,
D02 wire guarantee and D06. Both transports are mandatory for completion.

## Stage 8 — Complete quotas, extensions, observations, and test utilities

1. Implement the limits in the Verification section with validated construction
   options, instance ownership, cancellable acquisition, and exact boundary
   tests. Account allocated wire-buffer capacity in both directions. Separate
   control/cancellation paths from application handler admission; no saturated
   queue may prevent cancellation or subscription cleanup. Test saturation with
   partial frames in both directions: credit acquisition must not deadlock by
   holding all capacity while waiting for the remainder of a frame. Reject
   impossible frame/budget configurations at construction.
2. Implement FIFO admission: 32 active handlers and 32 queued calls by default;
   wait at most 1 second or the earlier caller deadline. Overflow returns a typed
   overload error. Explicitly configured per-tool serialization is per owning
   registration, not a package-global name. Test cancellation removes waiters
   without dropping or reordering other accepted calls.
3. Bound shutdown to 5 seconds and return incomplete-shutdown errors for
   noncooperative handlers/resources. Test contexts, slow peers, cancelled buffer
   waits, repeated Close, concurrent shutdown, and idle subscriptions. Join
   framework-owned goroutines; prove surviving user code is identified honestly.
4. Keep application response caches off. Test requests from different principals,
   changed authorization, and expired request context reach current policy and
   do not reuse results. Bound any compiled-schema cache by immutable digest and
   configuration profile; it cannot cache authorization or user results.
5. Implement explicit extension descriptors with namespace, schema/mapping
   versions, direction, capabilities, owned DTOs and modern SDK dispatch hooks.
   Reject duplicate names, standard-method shadowing, unknown required features,
   unsupported schema packs, and calls before negotiation. No dynamic Go plugins,
   schema-executed code, logging-based transport adapter, or HFSC implementation.
6. Add observations for operation, duration, byte counts, outcome and correlation,
   without default bodies, secrets, raw credential-bearing URLs, panic values or
   unbounded labels. Define callback concurrency and ensure user observer failure
   cannot corrupt protocol success/error handling or block cancellation forever.
   Use bounded delivery rather than a goroutine per event; report dropped
   observations. Treat a noncooperative observer as an external resource in the
   incomplete-shutdown result, never as proof that user code was forcibly stopped.
7. Complete `mcptest` recording/replay, raw assertions, in-memory connections,
   failure injection and canonical fixture helpers. Tests use it for mechanics
   but retain independent expected outcomes. Add resource-link and bounded-
   payload examples using synthetic resources; they demonstrate primitives,
   not a production artifact storage or transfer service.

Exit: quota/cancellation/isolation/redaction/extension cases pass under concurrency
and race checks; deliberate overload, cross-instance lock, capability bypass and
secret-leak plants fail; no blocker. Primary cases: G11/G13/G14/G15/G20; G19 remains
an explicit workload exclusion, verified by dependency/API scope inventory.

## Stage 9 — Qualify the integrated product and install CI gates

1. Run the traceability matrix below against both modules and the complete API.
   Check the generated availability inventory has an independent fixture for
   every required or advertised method/capability, not just representative tools.
2. Run official client and server conformance from the immutable prepared runner
   using the 2026-07-28 requirement set. Add the necessary fixture client/server
   adapters under `internal/contracttest/`; application-supplied auth/TLS services
   remain fixtures, not new core products. Record every required, unscored,
   skipped, and locally supplemented case. A harness limitation is explicit
   evidence and triggers review if a required behavior cannot be established.
3. Do not use expected-failure suppression. An unscored official check for a
   feature this framework advertises still needs a passing independent local
   case. Empty selections, zero exchanged frames, absent required checks, and
   scenario setup failure cannot count as success. Keep modern-only rejection
   tests separate from intentionally inapplicable legacy interoperability cases.
4. Execute native functional tests on all three supported targets and both Go
   versions. Run race tests on supported native race hosts with required C tools;
   use the fleet policy for host choice. Cross-compilation never substitutes for
   native lifecycle, filesystem, or child-process behavior.
5. Run bounded fuzz campaigns, initially 60 seconds per target, for raw JSON,
   stdio framing, SSE framing, reference resolution, and journal parsing. Queue
   each run and retain reproducible seeds. Combine explicit joins/close tracking
   with repeated lifecycle tests for leaks; goroutine counts alone are not proof.
6. Benchmark equivalent workloads with all validation enabled: raw frames at the
   four report sizes, small tool calls, large structured outputs, bounded resource
   payloads, and concurrent extensions at concurrency 1 and 3. Measure both
   transports and both HTTP modes. Capture at least five samples, CPU/toolchain,
   fixture hash, allocs, bytes, throughput, time, retained memory, and latency
   distribution. Apply the frozen numerical gates below without discarding slow
   samples or changing correctness semantics.
7. Measure generator load/resolve/model/map/emit/qualify/publish separately, cold
   and warm. For total generation, start from prepared local dependencies but a
   fresh candidate tree. Measure the generator process peak RSS; report compiler
   child memory/time separately, without subtracting child wall time from total
   generation. Use native OS measurements, including Windows process metrics.
8. Run every plant in scratch against the real gates, retain the baseline pass,
   mutated input/diff, expected failure and actual nonzero exit. Restore nothing
   in the working tree because no plant is written there. Verify the performance
   parser with an over-budget copied log, not only a formula unit test.
9. Add CI that enumerates both modules, uses pinned actions/tool artifacts,
   prepares dependencies explicitly, then runs offline checks with readonly
   modules. Include unit, vet/lint, generator drift, API, schema, raw-wire,
   conformance, native platform, race, bounded fuzz and budget jobs. Do not install
   Python or Node as product dependencies. Use serial heavy work per local fleet
   host; CI job scheduling must not bypass shared-host queues if self-hosted.
10. Publish redacted execution evidence to a paired REPORT with source/input/tool
    hashes and per-row results. Failed tests remain recorded until fixed and
    rerun; a single overall green summary cannot replace the case inventory.

Exit: all D01–D06 and decided G rows pass the complete implementation matrix;
all plants are caught; budgets pass; CI definitions run the same documented
checks; no blocker. No stability or release claim is made from prototype results.

## Stage 10 — Prepare the reviewable pre-v1 release candidate

1. Update architecture documentation to describe actual packages and dependency
   boundaries, replacing the current scaffold-only description after source
   exists. Write guides for canonical server/client registration, both transports,
   JSON/schema policy, generator commands/recovery, extensions, and troubleshooting.
2. Compile examples in an external consumer module with no direct SDK import and
   no local workspace assistance. Demonstrate a typed tool with explicit text
   presentation, a resource link and bounded resource result over subprocess
   stdio, explicit HTTP trust/auth configuration, and cancellation/shutdown.
3. Document all default quotas, configuration ranges, error outcomes, ownership,
   unsupported schema features, extension negotiation, observation privacy and
   the distinction between owned v2, SDK data encoding, and guarded wire bytes.
4. Assemble a release manifest containing library candidate version, SDK/schema/
   dialect/generator-format/semantics/toolchain versions, source and output hashes,
   API inventory, licenses, platform results and reproducible generation command.
   Keep schema manifest version and library semver independent.
5. Produce migration guidance mapping mcplib construction/registration/session
   use to canonical operations, including intentional breakage of deprecated
   signatures and removal of permissive schema behavior. Do not edit mcplib or
   MagicTools, add a compatibility adapter, or promise HFSC migration completion.
6. Review the package API, documentation, dependency closure and evidence against
   this PLAN. Mark execution complete only when Completion Criteria hold; record
   the candidate as ready for owner release review. Leave tags, pushes, publishing,
   consumer migration and subsequent stable release authorization to explicit asks.

Exit: complete reproducible candidate and documentation, all rows backed by
passing evidence, no blocker, no hidden required follow-up. Packaging and docs
checks pass; prior stage plants remain attributable to the exact candidate or
are rerun where final changes invalidate their evidence.

## Verification

### One canonical acceptance group per decision and state row

These are planned test names and locations. Each group contains the detailed
positive and negative subcases from its stage; no group may be empty or skip all
cases. The gate reconciles the manifest of IDs with executed results. The G19
exclusion has a scope review, not a fictional passing implementation test.

| ID | Canonical acceptance group | Primary stage | Required evidence |
| :--- | :--- | :--- | :--- |
| D01 | `internal/contracttest: TestD01CanonicalAPI` | 6, 9 | External consumers, owned handle identity, reviewed alias method/signature closure, enforcement boundary |
| D02 | `internal/contracttest: TestD02JSONBoundaries` | 5, 7, 9 | Owned v2, SDK differential fixtures, predecode raw wire guards on every supported direction/mode |
| D03 | `cmd/mcpgen: TestD03GeneratorContract` | 4 | All six commands, pure Go execution, internal import rejection, explicit offline/network distinction |
| D04 | `internal/validation: TestD04ValidatorAdapter` | 5 | Lossless precision, semantic locations, immutable compilation, no permissive fallback |
| D05 | `internal/contracttest: TestD05BudgetEvidence` | 9 | Complete sample inventory, equivalent controls, numerical gate and over-budget log rejection |
| D06 | `internal/contracttest: TestD06TransportQualification` | 7, 9 | Both transports, both HTTP response modes, native lifecycle and official/local coverage |
| G01 | `cmd/mcpgen: TestG01PureGo` | 1, 4 | Build/run without Python/Node in PATH; Go executable available for qualification |
| G02 | `cmd/mcpgen: TestG02DeterministicGeneration` | 4 | Same bytes/inventory, untouched mtimes, no path/time/network leakage |
| G03 | `internal/load: TestG03PinnedInputs` in generator | 2 | Absent/tampered/wrong-dialect source, missing ref, traversal and offline failure |
| G04 | `internal/resolve: TestG04ReferenceGraph` in generator | 2 | Valid cycles, depth/node/edge limits, deterministic diagnostics |
| G05 | `internal/model: TestG05Coverage` in generator | 2 | Complete definitions/methods/keywords; unsupported assertion and missing-rule plants |
| G06 | `codec: TestG06PresenceAndUnions` | 5 | Raw independent required/optional/null/zero/empty/discriminator/open-object fixtures |
| G07 | `codec: TestG07StrictRawJSON` | 5 | Duplicate/escaped duplicate, UTF-8, surrogate, trailing value, depth/size/numeric bounds |
| G08 | `transport: TestG08PredecodeGuards` | 7 | Zero invalid-byte exposure; corrupt resumable SSE closes original connection; no extra request |
| G09 | `internal/contracttest: TestG09CodecInterop` | 5, 9 | v1/v2/SDK custom methods and actual captured frames, including resource links |
| G10 | Root: `TestG10RegistrationFailure` | 6 | Invalid schemas, duplicates, expected declaration panics, no handler invocation/fallback |
| G11 | `internal/contracttest: TestG11Capabilities` | 6, 8 | Disabled capabilities absent and rejected; extension shadow/unknown/negotiation checks |
| G12 | `transport: TestG12ModernOnly` | 7 | Valid modern lifecycle; legacy/fallback/header/metadata/environment bypass rejection |
| G13 | Root: `TestG13InstanceIsolation` | 6, 8 | Same-name tool independence, multiple connections, registration-scoped serialization, race |
| G14 | `internal/contracttest: TestG14LifecycleBounds` | 8 | Admission, slow peers, saturated controls, first failure, cancellation, shutdown and leaks |
| G15 | Root: `TestG15NoCrossPrincipalReuse` | 8 | Cache disabled; current authorization on repeat/revoked/expired calls; bounded schema reuse |
| G16 | `internal/qualify: TestG16CandidateCompilation` in generator | 3 | Handwritten/internal/external/platform candidates; collision and foreign-target compile plants |
| G17 | `internal/publish: TestG17Recovery` in generator | 4 | Fault matrix, killed child, stale lock, edited output, recovery conflict, unrelated sentinels |
| G18 | `internal/contracttest: TestG18ModernSurface` | 3, 9 | Direct/transitive API audit plus actual Staticcheck gate, no suppression/excluded generated files |
| G19 | Scope/dependency/API inventory review | 8, 10 | No HFSC protocol/store/routing or consumer changes; primitives and examples only |
| G20 | `internal/contracttest: TestG20LimitsAndBudgets` | 8, 9 | Every numeric limit boundary and actual benchmark/RSS/cancellation results; planted excess rejected |
| G21 | `internal/validation: TestG21SchemaVocabulary` | 5 | All 155 definitions, 421 assertions, 36 formats, precision and unsupported-profile tests |
| G22 | `cmd/mcpgen: TestG22APIBoundaries` | 3, 10 | Pre-v1 manifest, generator internal import failure, runtime dependency closure and external examples |

Package locations for generator groups are relative to `cmd/mcpgen`. Test names
are acceptance interfaces for gate selection; helper/unit tests can be more
specific. Stage 9 refuses a missing ID, unexpected skip, or absent negative control.

### Resource and performance acceptance

MiB and KiB here mean binary units; they do not settle the later HFSC file limit.
Configuration validation must reject zero/negative explicit limits, overflow,
and inconsistent combinations rather than silently disabling protections.
An omitted option selects the default; distinguish omission from explicit zero.

| Resource | Contract to test |
| :--- | :--- |
| JSON frame / HTTP JSON body | 1 MiB default, positive configurable limit no greater than 16 MiB; receive and send; HTTP overflow 413 |
| SSE event | 1 MiB including framing, maximum configurable 16 MiB; per event, not stream lifetime |
| JSON depth | 64 default, maximum configurable 256; check before recursive validation |
| Numeric representation | At most 256 token bytes, exponent magnitude at most 1,024; reject before arbitrary precision |
| Active and queued application calls | 32 active and 32 FIFO queued by default; explicit validated positive configuration; at most 1 second queue wait or earlier deadline |
| Wire buffers | 64 MiB runtime-owned live capacity across directions; cancellable admission, excludes SDK/user-owned allocations |
| Buffer retention | At most 8 MiB pooled total, at most 256 KiB per retained buffer |
| Schema model | 2 MiB per document, 4,096 definitions, 65,536 reference edges; generator traversal/diagnostic caps from stage 2 |
| Strict check overhead | Median checked minus median SDK decode no greater than `5,000 ns + 12 ns × frame bytes`; additional median no greater than 8,192 B/op and 16 allocs/op |
| Canonical no-op RPC | Median no greater than `1.25 × identically validated reference + 50 µs`; extra median no greater than 8,192 B/op and 16 allocs/op; both transports, concurrency 1 and 3 |
| Cold schema compile | All 155 definitions within 5 seconds, local registry, fresh compiler |
| Full generation | At most 30 seconds wall time and 256 MiB generator peak RSS on the pinned input; compiler children reported separately |
| Cancellation | Return within 250 ms in controlled stalled-reader, admission and terminal-SSE fixtures |
| Shutdown | Bounded call returns within 5 seconds, with explicit incomplete-shutdown failure when necessary |

For time/allocation comparisons retain at least five samples for each path on
the same platform, CPU, Go version, workload and concurrency. Use the report's
frame byte sizes 1,069 / 65,581 / 262,189 / 1,048,493. Generator, cancellation and
shutdown bounds apply to each controlled measured run, not a median that hides
a timeout. Record all samples and environment variation; failure cannot be
fixed by silently changing the budget or excluding the slow sample.

### Commands and gate semantics

The commands below become executable deliverables in stage 1 and subsequent
stages. Today only the two documentation commands exist. Invoke one command at
a time. Heavy commands use fleet wrappers as required by repository policy;
wrappers are omitted from the command column only to show the underlying check.

| Working directory | Command | Expected result |
| :--- | :--- | :--- |
| Root | `python3 scripts/check_records.py --check-all` | Valid pairs, state metadata, relative links |
| Root | `npx --yes markdownlint-cli2@0.23.2` | No Markdown issues |
| Each Go module | `go vet ./...` | Exit 0 with local selected toolchain and readonly module policy |
| Each Go module | `go test -mod=readonly -count=1 ./...` | All required functional groups execute and pass |
| Each Go module | `golangci-lint run --config <repo>/.golangci.yml ./...` | Pinned 2.15.0, no exclusions/suppressions for Staticcheck |
| Root | `python3 scripts/gate.py setup --allow-network` | Explicit preparation only; pins/checksums verified, no floating upgrades |
| Root | `python3 scripts/go-precheck.py --module all` | gofmt cleanliness, vet, affected-package tests and lint; generated files included |
| Root | `python3 scripts/gate.py generate-check` | Build prepared generator, `check`, deterministic output inventory |
| Root | `python3 scripts/gate.py api` | Full export/signature/alias inventory and external consumer compilation |
| Root | `python3 scripts/gate.py wire` | Raw-frame, modern-only and both-transport contract groups |
| Root | `python3 scripts/gate.py qualification` | Both modules; all required groups, conformance reconciliation, budgets and platform evidence |
| Root | `python3 scripts/gate.py plants` | Baseline passes and each scratch mutation produces the intended real gate failure |
| Prepared conformance source | `node dist/index.js list --requirements 2026-07-28` | Scenario inventory matches pinned requirement file |
| Prepared conformance source | `node dist/index.js server --url <fixture-url> --requirements 2026-07-28` | Server results reconciled with applicable/advertised coverage |
| Prepared conformance source | `node dist/index.js client --command <fixture-client> --requirements 2026-07-28` | Client results reconciled; adapter implements required scenario contract |

Python orchestration resolves `<repo>` and fixture paths, launches and cleans up
owned fixture processes, captures exits, and reads structured results; those
placeholders are not shell syntax to copy literally. Keep any upstream command-
string interface isolated to a fixed fixture executable with controlled arguments;
repository subprocess orchestration never uses `shell=True`.

Set `GOWORK=off`, `GOTOOLCHAIN=local`, `GOPROXY=off`, and readonly module mode for
normal generation and gates after preparation. Use a prepared binary for mcpgen;
normal generator commands must not cause `go run` to download tools/modules.
Explicit setup is the only dependency-acquisition path. Test that guard with an
empty isolated dependency/tool cache and a network-denying fixture.

### Platform and negative-control matrix

Native functional and generator publication/recovery tests run on Darwin/arm64,
Linux/amd64 and Windows/amd64, on Go 1.27.0 and 1.27.4. Race runs use Go 1.27.4
on supported native hosts after C-toolchain verification. Fuzz campaigns run on
a selected supported host; replay retained seeds on all target platforms.
Minimum-version testing does not silently use automatic toolchain selection.

Every new gate has a baseline pass and a deliberately broken scratch input.
Required plants include missing/changed hashes, network resolution, dropped
required field, wrong discriminator, numeric rounding, duplicate acceptance,
unsupported assertion, missing capability rule, deprecated transitive signature,
generated manual edit, platform-only compile failure, interrupted publication,
shared tool lock, missing cancellation, secret-bearing observation, missing test
ID, skipped conformance case, and excessive benchmark output. A failure must name
the intended case, not merely fail earlier due to setup or compilation syntax.

## Rollout and Rollback

Land the approved stages as reviewable phase commits. Early commits may be
internally usable but are not release candidates until stage 9 passes. Normal
regeneration cannot auto-promote upstream inputs. Review schema/API/semantic
changes and manifests together. A pre-v1 label does not waive D02/D06 or allow
advertised features to ship without their contract checks.

The first rollout is an unpublished candidate exercised by external fixture
consumers. No running service, existing mcplib integration, persistent artifact
store, or consumer dependency is changed. A later migration needs its own approved
scope and retains the previous integration until replacement behavior passes.

## Rollback

For implementation regressions, prepare a reviewed revert of the affected phase
commits while preserving unrelated work. Do not reset or delete another session's
uncommitted changes. Re-run the affected contract/generation gates after a revert.
No production data rollback is needed because this plan adds no persistent
application store and changes no deployed consumer.

For interrupted generation, use verified `recover --action inspect`, then the
appropriate complete/restore action authorized by the caller. Recovery restores
only manifest/journal-owned bytes; conflicts with manual changes stop explicitly.
Never use Git clean/reset as generator recovery. For bad schema/SDK candidates,
restore the previous reviewed manifest and matching dependencies/output together;
re-run qualification, rather than pairing old generated code with new inputs.

## Completion Criteria

* MADR is accepted and execution was explicitly authorized; this PLAN tracks
  execution honestly and has no unresolved deviation or missing phase evidence.
* Both modules build independently; runtime consumers need no generator tooling.
  Generator remains pure Go and its engine internal; no second MCP runtime exists.
* All D01–D06 groups and every decided G row have passing implementation results,
  with every advertised method/capability accounted for and G19 explicitly excluded.
* All original qualification expectations remain intact; both transports and
  directions, modern-only behavior, lossless validation and native v2 owned APIs
  pass independent fixtures and actual wire tests.
* Every numerical resource/performance rule passes measured implementation gates;
  no benchmark-only claim substitutes for missing runtime or generator evidence.
* Staticcheck is clean with no exceptions, including generated/test code, and
  public aliases expose no deprecated transitive surface.
* Deterministic publication, offline preparation policy, three-platform compilation
  and native execution, race/lifecycle tests, and recovery fault injection pass.
* Every new gate has caught its intended broken scratch input; no hidden skip,
  expected-failure baseline, missing case, or unscored advertised feature remains.
* Documentation, examples, licenses, source/output/API/tool manifests, execution
  results and migration guidance match the exact candidate. Record/link/Markdown
  checks and identifier scans pass. No consumer change, push, tag or publication
  occurred without its separate explicit authorization.

## Execution Record

2026-11-23 — Draft only. Repository/source grounding and tool/conformance pin
research completed. Product implementation stages 0–10 have not executed; none
is marked passing. The earlier scratch qualification belongs to the companion
REPORT and is not retrospectively counted as this PLAN's execution.

2026-11-23 — Owner approved this PLAN and reaffirmed MADR acceptance. Execution
has not started; wait for further owner instructions. No stage is authorized
by this approval alone.

For each executed stage, append a dated `Stage N — pass M` entry containing the
starting revision and dirty-content digest, affected files, exact commands and
exit statuses, complete redacted output artifacts, per-ID results, plants, and
phase commit. Number failed attempts too. Record deviations and the owner's
ruling before continuing; retain the original instruction with an annotation
when a later approved change supersedes it.
