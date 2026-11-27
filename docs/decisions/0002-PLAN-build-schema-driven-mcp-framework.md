---
status: in-progress
date: 2026-11-27
associated-madr: 0002-MADR-build-schema-driven-mcp-framework.md
---

# Implement the schema-driven MCP framework around the upstream Go SDK

Associated MADR: [0002-MADR-build-schema-driven-mcp-framework.md](0002-MADR-build-schema-driven-mcp-framework.md).
Evidence: [0002-REPORT-qualify-mcp-framework-boundaries.md](../reports/0002-REPORT-qualify-mcp-framework-boundaries.md).

The owner approved this implementation plan on 2026-11-23 and reaffirmed
acceptance of the completed MADR, including its numerical acceptance contract.
The initial instruction was to wait for further authorization. On 2026-11-24,
the owner reaffirmed MADR acceptance and PLAN approval and authorized Stage 0.
The historical execution entries below record the B01 investigation. Product
source subsequently landed through the claimed Stage 6 baseline. The
[completion audit](../reports/0011-REPORT-pre-stage-7-completion-audit.md) found
unmet exits in every Stage 0–6, so those completion claims are not the current
acceptance status. The owner selected post-audit option A, “do A repairs”.
R1–R5 below specify that option's repair amendment. The owner initially
authorized R1 only, then approved execution of R1–R5 and commits to main between
qualified coherent phases. R1 passed qualification on 2026-11-25. R2's input,
model/API and complete candidate exit passed on 2026-11-26; R3 publication and
installed-tree qualification passed on 2026-11-27. The coherent R2/R3 commit is
ready after the Make/hook repair and pass-7 installed checks described below.
R4/R5 and Stage 7 remain pending.
The owner committed and pushed the R2 work in progress as `5c00e36` on
2026-11-26. That historical checkpoint was not a qualified R2/R3 exit;
the reviewed contract has now been safely published and requalified locally.
The overall PLAN remains in-progress. Stage 7 is not ready to start.

The 2026-11-25 research refinement below evaluates three implementation options
per repair and specifies the recommended work. It is a reviewable planning
update; the later owner approval and dated execution entries record which work
was authorized and what actually passed. Shared-runner changes were separately
approved through D1 A. Other product repositories remain outside this work.

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
| SDK HTTP policy | SDK v1.8.0; `mcp/streamable.go:174–224` | Localhost protection defaults on; the cross-origin option is deprecated; request cancellation propagation is opt-in; the SDK body default is 4 MiB, which the amended policy adopts as the undeclared default (owner amendment 2026-11-25). |
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

This table preserves the original deliverables. Some paths now exist; existence
does not establish acceptance. The repair ledger below gives current status.
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

### Proposed repair amendment — R1–R5 before Stage 7

This is the concrete execution amendment for the owner's post-audit option A.
It restores the existing Stage 0–6 requirements and the MADR's RA01–RA20 states.
The initial selection authorized drafting for review. The subsequent instruction
“resume r1” authorizes R1 implementation and verification; R2–R5 remain pending
execution approval. Historical stage instructions and
measurements remain below. Nothing in this section retrospectively marks a
failed or untested obligation passing.

#### Current acceptance ledger

| Original stage | Current state | Repair ownership and completion condition |
| :--- | :--- | :--- |
| 0 | Incomplete; B01 feasibility resolved, full inventory unfinished | R1 inventory; R2 transitive mapping evidence; R5 actual feature reachability. All Stage 0 obligations must be reconciled. |
| 1 | Incomplete; modules exist, preparation/accounting/lint gaps remain | R1 restores foundational gates and preparation; full-product checks remain failed wherever later defects persist. |
| 2 | Incomplete; loader/model/reference/mapping failures reproduced | R2; original Stage 2 exit plus RA05–RA09 pass. |
| 3 | Incomplete; emission/alias/candidate gaps remain | R2 establishes complete candidate contract; R3 safely promotes it; R4/R5 supply the original codec/runtime integration evidence. |
| 4 | Incomplete; unsafe publication and nonrecoverable journals | R3; original Stage 4 exit plus RA12–RA14 pass natively on all three targets. |
| 5 | Incomplete; strict codec and production validation gaps | R4; original Stage 5 exit plus RA15–RA17 pass. |
| 6 | Incomplete; enforcement and required API behavior absent | R5; original Stage 6 exit plus RA18–RA20 and G23–G25 production cases pass. |
| 7–10 | Not certified or started by this repair amendment | Original scope retained, including Stage 9 CI01–CI08; separate subsequent execution instruction. |

Repair phases are work packages, not substitute definitions of the original
stage exits. An early phase may prove a gate correctly rejects the still-broken
product. Record that product failure separately; never label the complete
product green or use expected failures to close a required case. Requalify
affected cross-phase cases after their dependencies change.

#### R1 — Restore evidence, preparation and trustworthy local gates

Scope: `scripts/gate.py`, `scripts/go-precheck.py`, their tests, preparation
helpers under `scripts/`, `tools/versions.json`, `.golangci.yml`, required
fixture inventory and the documentation index. Existing generator test setup
may change only to resolve prepared SDK locations and replace opt-in mutation
helpers with equivalent always-executed scratch tests. Product generation and
runtime behavior remain R2–R5 work.

1. Re-ground every audit locator at execution HEAD, record the starting dirty
   inventory and hashes, and preserve the existing report/index changes.
   Inventory all D/G/CI obligations, all 155 definitions, standard methods and
   pinned conformance scenarios, with role, direction, transport, normative
   reference, named acceptance owner and expected outcome. A missing required
   binding is a blocker, not an inapplicable row. Preserve previous failed
   observations and distinguish prototype, helper and production evidence.
2. Implement test-result accounting by module/package/test identity. Require
   fresh execution (`-count=1`) and matching terminal outcomes. Reject malformed,
   duplicate/conflicting, empty, cached-only, missing, failed and unexpectedly
   skipped results, including a zero child exit with an incomplete stream.
   Python checks aggregate every independent failure into a named FAIL line.
   Support strict `--module runtime|generator|all` selection and reject invalid
   arguments. Keep incomplete gate modes nonzero until their actual contracts
   are implemented; a helper's success cannot stand in for a product gate.
3. Make pre-add non-mutating: use `gofmt -l`, vet, appropriate package tests and
   the pinned linter with the repository configuration in both modules. Include
   generated/test code in lint. Prove actual lint catches generated and
   handwritten deprecated-symbol plants. Fix missing-tool/version diagnostics
   so independent checks continue and report their own outcomes.
4. Implement explicit pinned setup, including local tool/dependency identities,
   conformance commit/lock/requirements verification, unchanged-lock `npm ci`,
   build and prepared-runner digest. Normal gates are offline and readonly.
   Resolve Go locations through module-aware Go metadata rather than a presumed
   home-directory layout. Fail missing prerequisites without network fallback.
   Test interrupted/partial setup and hash mismatch without promoting a partial
   preparation. Do not reconfigure owner security/disclosure tooling.
5. Run the existing suites with honest outcome accounting. Required emission
   tests must execute with verified prepared SDK inputs. Replace the opt-in
   repository-writing test mechanism with equivalent scratch generation/CLI
   coverage; preserve its assertions and demonstrate ordinary tests leave the
   input tree unchanged. Report any newly exposed product failure under its
   existing repair ID rather than weakening checks.

Exit: no R1 blocker; RA01–RA04 all tested; each gate's broken scratch control
caught; prepared identities and full inventory recorded. Overall product
qualification remains incomplete while R2–R5 cases fail. No Stage 9 workflow,
Makefile/receipt rollout or GitHub enforcement is introduced here.

#### R2 — Repair the input/model/API contract and candidate qualification

Scope: generator `load`, `resolve`, `model`, `sdkmap`, `emit`, `qualify` and
diagnostics, `generate.go`, `check.go`, reviewed schema/SDK/semantic manifests,
generated contract artifacts and independent fixtures. Include root signature
and `internal/sdkbridge/` adaptations needed to compile and exercise the
corrected owned types with the existing runtime. Such adaptations must preserve
existing working behavior; no runtime stub or deprecated compatibility alias
is permitted. Complete additional canonical operations in R5.

1. Replace incomplete handwritten JSON parsing with strict standard-library
   token parsing and exact numeric representation. Bound file reads before
   allocation beyond the input allowance. Exercise every A05 input and valid
   Unicode-pair control; enforce dialect, duplicates, trailing content,
   definition limits and deterministic bounded source-pointer diagnostics.
2. Resolve local registered canonical URIs, fragments and JSON Pointers without
   lossy suffix rewriting. Carry the correct base across references, decode
   pointer escapes correctly, validate array indices without overflow, retain
   recursive graph edges and enforce acyclic-depth/reference budgets. Reject
   missing targets, duplicate IDs, encoded traversal and symlink escape; never
   fetch references. Invoke resolution in the actual generation pipeline.
3. Model and classify the pinned vocabulary and explicitly supported inputs.
   Preserve assertions, annotations, const/enum, required/nullable/presence,
   unions/compositions and object openness. `minLength` must either be qualified
   and represented or explicitly rejected, never relabeled an annotation.
   Require all reviewed semantic and SDK mapping inputs with their independent
   version/hash identities before emission, including first generation.
4. Audit alias closure using module-aware source/type information, including
   instantiated generics, fields, embedded/inherited methods, request/session
   reachability and deprecation comments. Resolve SDK paths with Go metadata.
   Generate compiled identity/signature probes; replace unsafe aliases with
   owned public types and private conversions. Consumers use owned request
   contexts/handles; no public `Session` field or accessor exposes the SDK.
5. Produce Go ASTs, protocol method/version registries, declaration/schema
   bindings, owned DTOs/hooks, API/coverage tables and hashed ownership manifest.
   Every one of the 155 definitions has an explicit reviewed disposition:
   alias, owned, validation-only, or release-blocking unsupported. Validation-only
   is justified in the independent inventory, never inferred from emission
   failure. Required Option D union types and open maps must have real bindings.
   Detect keywords, exported names/methods, handwritten and platform declarations,
   file and case-insensitive collisions. Test serialization against handwritten
   fixtures; retain R4 differential obligations explicitly.
6. Wire `check` and `generate` through complete scratch-candidate qualification:
   all proposed files plus real handwritten source, internal/external tests,
   module boundaries and tags; stale owned files removed only in scratch after
   verified ownership. Set `CGO_ENABLED=0`, `GOWORK=off`, `GOTOOLCHAIN=local`,
   `GOPROXY=off`, `GOSUMDB=off` and readonly modules. Compile package and test
   binaries for Linux/amd64, Windows/amd64 and Darwin/arm64. Never execute foreign
   binaries. Check every failure leaves destinations byte-identical.
7. Treat the existing hashless ownership list as unverified migration input.
   Render and qualify the new pinned candidate in scratch; record exact old and
   proposed file inventories and hashes for review. Do not run the old publisher
   to install it. R3 owns the verified publication/migration. Handwritten bridge
   adaptations must compile against the committed baseline and the proposed
   candidate, or remain an explicit coordinated R2/R3 change until both can land
   coherently; do not commit a broken intermediate tree.

Exit: no R2 blocker; RA05–RA11 tested; malformed/ref/missing-map/transitive-alias/
collision/platform compile plants caught; the actual proposed product candidate
compiles on all three targets. The candidate is not declared published until
R3. No Stage 3 completion claim precedes its codec/runtime integration evidence.

#### R3 — Make publication, recovery and CLI contracts dependable

Scope: generator CLI and `publish`, `recover`, lock/acquisition support,
versioned manifests/journals, R2 candidate promotion, `scripts/verify_*` and
committed generator fault fixtures. Refactor packages where useful without
changing the pure-Go product boundary. Include required Python ignore rules
when adding reusable orchestration. No reusable helper remains scratch-only.

1. Validate paths before all mutation: reject absolute/traversing/encoded or
   aliased paths, symlink escapes, unowned collisions and unsafe platform names.
   Verify containment through existing ancestors and destination identity;
   revalidate under the exclusive lock before touching files. Test a raced path
   replacement, case-insensitive collision and intervening edit. Apply the same
   checks to recovery and stale deletion, including files originally new.
2. Implement bounded context-aware lock ownership and explicit conflict handling.
   Do not steal a lock based on age. Stage original and candidate bytes plus
   hashes on the destination filesystem; durably journal intent before each
   destructive boundary, use per-file atomic replacement where supported, and
   promote the manifest last. Preserve unchanged mtimes and inventory order.
   Report platform sync limitations accurately without claiming transactionality.
3. Implement read-only inspect and locked complete/restore. Validate journal
   version, paths, ownership, original/candidate bytes and manifest state before
   mutation. Detect manual changes to new as well as replaced files. Completion
   supplies missing staged bytes and finishes the manifest; restore reinstates
   original bytes, including removed files and the original manifest. Preserve
   recoverable state on failure; an unreadable/corrupt journal is an error, not
   “no journal”. Repeated completed recovery is safe and deterministic.
4. Migrate the old hashless manifest only through explicit verified bootstrap:
   compare the frozen baseline's Git object bytes and independent R2 digest
   inventory with all currently owned paths, reject dirty/unowned/conflicting
   paths, and journal original bytes before installing the qualified candidate.
   Never infer permission from filenames or bless current bytes by rehashing
   them. Test clean bootstrap, absent output, modified output and unknown files.
   Thereafter ordinary generation always verifies the previous hashed manifest.
5. Complete all six CLI contracts: strict common arguments/format/versioned
   results and exit codes; inspect complete provenance/coverage; semantic/API/
   capability/vocabulary diff even when filenames do not change; bounded update
   with immutable commit/hash provenance, cancellation and disjoint candidate
   storage. No live input/output promotion or ordinary network use. Diagnostics
   and structured output retain their separate channels.
6. Replace synthetic interruption claims with actual child-process kill tests
   at deterministic publication checkpoints. Inject write, sync, rename,
   deletion, manifest, cleanup, permission-denied and disk-full failures. Test
   cancellation, competing generators/recovery, stale locks and tampered journal
   state. `check` must reject mixed output; complete/restore must preserve every
   unrelated sentinel. Run native filesystem/recovery cases on all three targets.
7. Regenerate twice under changed scratch names/source-map ordering; require
   identical bytes, inventory and retained mtimes. Run actual CLI check and API
   gates after promotion. Update documentation to identify the verified migration
   and recovery commands; no direct scratch-binary path is a prerequisite.

Exit: no R3 blocker; RA12–RA14 and original Stage 4 contracts tested; all fault
and manual-edit plants caught; verified R2 candidate safely promoted and normal
regeneration clean. Commit coordinated R2/R3 files together if necessary for a
coherent build, recording both phase results rather than inventing a partial
Stage 3/4 success.

#### R4 — Complete strict owned codecs and canonical validation

Scope: `codec/`, `internal/rawjson/`, `internal/validation/`, schema declaration
bindings, generated serialization hooks and their fixtures, location helpers,
runtime callers of those helpers and `scripts/verify_mutations.py`.

1. Require exactly one complete valid raw JSON value. Distinguish actual end of
   input from truncated strings/numbers/literals/containers; do not match error
   text for success. Preserve escaped-equivalent duplicate and invalid UTF-8/
   surrogate rejection. Test empty/whitespace, trailing value, every audit input,
   numeric signs/leading zeros and exact/one-past limits. Restore depth 64 default
   with explicit valid bounds up to 256; retain the frozen frame/token/exponent
   limits and reject invalid bound configuration.
2. Implement the owned JSON v2 API and generated presence/union/open-object
   behavior. Cross-check v1, v2 and the real SDK path using independent frames,
   custom marshalers, raw values, nil/empty collections, unknown fields, case,
   resource links and exact numeric neighbors. Compare required byte identities
   separately from independently validated semantic equivalence.
3. Use one production adapter/configuration path in qualification and runtime.
   Force the accepted draft/profile, local resource registry and lossless numeric
   projection; prohibit external loading. Compile/cache by immutable content and
   profile identity with bounded retention and concurrent-safe immutable inputs.
   The same caller key with changed bytes must not reuse an old schema.
4. Preserve arbitrary application `format` as annotation unless the accepted
   policy mandates an assertion. Exercise generic format syntax through the same
   production semantic checkers used at mandated protocol locations. Do not fix
   A16 by globally enabling `AssertFormat`. Enforce `Root.uri` file semantics,
   strict URI/base64 syntax and the pinned RFC 6570 parser in actual declarations
   and validation; remove divergent runtime checks only once equivalent stronger
   production coverage passes. Qualify and document any supported compatible
   regex subset; reject unqualified pattern/vocabulary inputs with pointers.
5. Reconcile the original 421 assertions, 155 definitions, 36 format fixtures
   and precision cases with the larger vendored corpus by fixture identity and
   expected outcome. Report annotation/nonapplicable cases explicitly; corpus
   size alone is no evidence of equivalence. Convert diagnostic probes into
   maintained regression cases and preserve independent oracles.
6. Run genuine scratch mutations: remove a required assertion, round a large
   number, and actually permit duplicate input past the raw guard. Each must
   fail its intended production-path test. An error-message change or unrelated
   compile failure does not count as catching the bypass.

Exit: no R4 blocker; RA15–RA17 and original Stage 5 requirements pass, including
actual SDK differential cases; genuine bypass plants caught. This does not
claim guarded production stdio/HTTP/SSE, which remains Stage 7.

#### R5 — Restore the live canonical runtime and requalify prerequisites

Scope: owned root handles/options/registration/middleware/errors/session APIs,
`internal/sdkbridge/`, generated API bindings, `internal/contracttest/`, external
consumer fixtures and the transport handle seam. Stage 7 guards, Stage 8 quotas
and Stage 9 CI remain in their original stages.

1. Expose owned server connect/serve and connection/session operations that
   external consumers can use. Complete the independent method/capability
   inventory: tools, resources/templates/links, prompts/completion, discovery,
   cursor/request metadata, subscriptions, progress/cancellation and supported
   input-required operations. Advertise only configured usable capabilities;
   required features cannot disappear under an “optional” label.
2. Resolve the private-method package conflict with an opaque owned transport
   handle shared through an internal package. Public `transport/` constructors
   return that handle; root accepts it and the internal bridge unwraps it.
   Keep fields and SDK extraction private, with no exported SDK-returning method
   on the handle and no root import from `transport/`. Test external consumer
   compilation, zero/invalid handles, import direction and inability to obtain
   SDK runtime objects. Keep the original SDK connection inside the bridge.
   This specifies the package seam, not completion of Stage 7 adapters.
3. Implement Option D fulfillment with owned callbacks/DTOs and the SDK's public
   map codecs, disabling SDK automatic fulfillment. Carry request state and
   metadata across tool/prompt/resource input-required paths where the pinned
   protocol requires them. Exercise sampling with actual callback output, actual
   configured roots and elicitation. Pin missing/wrong `role`/`roots`/`action`
   discriminators, absent callbacks, cancellation and no blind reissue of side
   effects. Keep deprecated runtime traversal entirely inside the SDK package.
4. Connect the real dispatch path to mandatory validation/capability policy,
   observation, authorization and admission before user side effects. Install
   each user link once in declared order. Validate final output after all user
   transformations, including middleware short-circuit results and missing
   required structured content. Denial must invoke zero tool handlers; retain
   counters for each layer so a detached helper cannot pass the test. Validate
   raw canonical input before any lossy typed conversion; SDK validation remains
   separate. Later Stage 7 wire policy cannot be claimed by these dispatch tests.
5. Return expected declaration errors without fallback. Check schemas before SDK
   registration and convert only identified declaration panics; unrelated
   programmer faults propagate. Rejected registrations leave no stale registry
   or compiled-output entry, and corrected retry works. Declare/validate output
   schemas consistently with advertised tool metadata. Deep-copy owned mutable
   configuration on entry and on retrieval. Use classified, bounded sanitized
   public errors with private cause access; test Is/As, Unicode bounds and secret
   values. Keep application response caching disabled.
6. Replace private-SDK-only round-trip evidence with external consumer modules
   importing go-mcp-lib public packages only, no direct SDK imports or workspace
   assistance. Exercise allowed/denied calls, invalid input/output, callbacks,
   all required canonical operations and two servers/two connections with one
   blocked handler. Run relevant native tests on all three targets with CGO off;
   run race cases separately on qualified native hosts with CGO enabled.
7. Re-run every affected R1–R5 acceptance group and reconcile the original
   Stage 0–6 exits against the final source/input/tool identities. Include the
   minimum and preferred Go versions on all three native targets for prerequisite
   functional/publication cases; cross-compilation is additional evidence.
   Record failures as failures. Update original report status annotations and
   the index only with links to new evidence; preserve original observations.

Exit: no R5 blocker; RA18–RA20, G23–G25 and all applicable original Stage 0–6
groups pass; all corresponding plants caught; final generated drift/API and
pre-add checks pass. Stage 7 becomes ready only after this evidence is reviewed
and every prerequisite row is closed. R1–R5 completion does not authorize it.

#### Research refinement — 2026-11-25

Assessment baseline: HEAD `5bfdd9f5e4a6e9286cf593fdc1f3ccec60890532` plus
the unfinished R1 working tree. The audit's product defects remain relevant;
the draft R1 improves orchestration but has not demonstrated its exit. These
are blockers to declaring Stages 0–6 complete and starting Stage 7, not reasons
to stop investigating or repairing the prerequisite stages. R1 establishes
trustworthy evidence; R2 establishes the generated contract; R3 protects its
publication; R4 enforces wire/schema semantics; R5 applies them on live SDK
operations. A passing helper or cross-build cannot substitute for those exits.

The recommendations below refine the existing RA01–RA20 outcomes. Alternatives
that require a dependency or architecture change are identified explicitly and
are not selected. Keep the accepted SDK/protocol/tool pins, numerical bounds,
Option D, and standard-library-only generator. Do not upgrade a dependency merely
to avoid investigating the pinned behavior. Relative costs describe engineering
and qualification work, not measured time estimates.

##### R1 options and remaining work

| Option | Evaluation |
| :--- | :--- |
| A — Finish the shared Python runner and explicit check graph **(recommended)** | Moderate orchestration work; preserves the repository's tooling policy and existing draft. Go tools remain the authorities for Go compilation, analysis and execution. |
| B — Replace the draft with one audited reference repository's runner | Moderate-to-high adaptation cost; useful only if it demonstrably covers both modules, strict result accounting and all native targets. Copying a runner does not transfer its qualification. Assess and adapt only in this repository. |
| C — Reimplement orchestration as a Go command | High migration and bootstrap cost; introduces a tooling-policy exception and another maintained program. No demonstrated advantage justifies changing the Python orchestration rule here. |

The draft's specific gaps are in `scripts/gate_common.py` (`environment`,
`run`, `account_tests`, `precheck`), `scripts/prepare.py` (`prepare`), and
`scripts/acceptance_inventory.py` (`build_inventory`). Complete R1 as follows:

1. Replace generic inventory labels with reviewed per-obligation bindings:
   method/definition/scenario, role, direction, applicable transport, normative
   reference, required acceptance group and expected named cases. Account for
   server-to-client cancellation where the pinned protocol permits it; do not
   derive direction from a method-name prefix. Keep expected obligations
   independent of observed passing test names. Missing RA/D/G bindings cannot
   be hidden by one unrelated passing test or by regenerating the inventory.
2. Parse the documented `go test -json` event stream, including interleaved
   packages, parallel subtests, build failure and package-without-tests cases.
   Reconcile expected groups and required cases as well as package terminals.
   Use `-count=1`, attributable command/target identity and completed execution
   for freshness. Remove the draft's arbitrary `"(cached)"` log-substring
   rejection; a test may legitimately print that text. Qualify parsing against
   real events from both pinned Go versions, not invented event fields alone.
   Keep process exit and stderr/build diagnostics alongside the event stream.
   See the [Go event contract](https://pkg.go.dev/cmd/test2json).
3. Specify and verify the execution environment rather than inheriting a foreign
   `GOOS`, `GOARCH`, experiment, workspace or compiler selection. Record actual
   compiler and target identities. Ordinary execution uses CGO off; only the
   explicitly selected race leaf enables it. Resolve executables using verified
   configuration/PATH and module metadata, including native Windows executable
   naming. A user-home layout is not a tool identity.
4. Stream full logs to bounded/rotated scratch artifacts with bounded summaries;
   retain partial diagnostics on timeout. Distinguish missing executable,
   timeout, signal termination and child failure. Qualify cancellation and
   descendant-process cleanup on native targets before releasing preparation
   locks or reporting completion. Continue independent checks after a failure.
5. Separate offline preparation verification from explicit acquisition. The
   draft calls `go mod download` in live modules and only checks hashes afterward;
   move acquisition to verified temporary module metadata or an equivalent
   isolated location. A readonly build flag alone does not prove `go.sum` and
   caches cannot change under module-management commands. Verify exact runtime
   dependency pins as well as archive integrity; include conformance source,
   lock, requirements, runtime dependency closure and runnable entry point in
   prepared identities. Interrupted/tampered preparation must not be promoted.
   See the [Go module reference](https://go.dev/ref/mod) and
   [npm ci contract](https://docs.npmjs.com/cli/v11/commands/npm-ci/).
6. Finish the local analyzer-ownership table before deduplicating. The current
   path runs standalone vet, default test vet, and a linter configuration with
   govet. Recommended owner: standalone full `go vet` for the agreed analyzer
   set; tests execute with `-vet=off` only in graph paths requiring that vet
   result. Remove overlapping linter analyzers only after proving equivalent
   options, package/test/generated coverage, tags and target context with real
   plants. Retain any non-equivalent analysis. This is ownership relocation,
   not waiver of an assertion. Keep `gofmt -l`, staticcheck/SA1019 and necessary
   independent build/API checks. Fail unexplained nested modules or uncovered
   source rather than silently omitting them. Distinguish invalid Go fixture
   inputs from product source using an explicit reviewed inventory.
7. Execute matching leaves once within an invocation and feed their identified
   results to all consumers; do not add cross-invocation success caching in R1.
   Record execution counts and timings, including preparation and queue time.
   Prove zero unexplained duplicate leaves and unchanged obligations. Leave the
   Makefile, exact-commit receipts and full Stage 9 rollout in their own stage.

RA01–RA04 add controls for a missing expected group despite other passing tests,
legitimate cache-text output, malformed/unterminated events, inherited foreign
target, module mutation, orphaned child, runner tampering and duplicated vet.
Keep the actual generated/handwritten SA1019 plants and scratch emission tests.
R1 must expose later product failures without redefining them as R1 success or
weakening the checks needed to commit a coherent change.

##### R2 options and remaining work

| Option | Evaluation |
| :--- | :--- |
| A — Standard-library parsing/types with explicit Go module metadata **(recommended within the accepted dependency constraint)** | Moderate-to-high implementation cost. Preserves the standalone generator; requires qualification of the export-data loader and transitive type audit. |
| B — Put a `go/packages` auditor in a separate developer-tools module | Uses the Go project's recommended general package loader while retaining the standalone generator boundary. Adds a third module, orchestration and a decision about which generator commands require the auditor; requires an approved scope amendment. |
| C — Add pinned `golang.org/x/tools/go/packages` to the generator | Least custom loading code and strong module-aware analysis; changes the accepted standard-library-only generator constraint and dependency footprint. Requires an architectural decision before implementation. |

Go's [importer documentation](https://pkg.go.dev/go/importer) recommends
[`go/packages`](https://pkg.go.dev/golang.org/x/tools/go/packages) for general
module-aware loading. Option A is a constrained tradeoff, not a claim that
`importer.Default()` is the modern module loader. Implement it with
`go list -deps -export -json` from the verified candidate module, explicit
`importer.ForCompiler` export-file lookup, `go/types`, and `go/ast` for source
comments. Never fall back to GOPATH lookup. Reject incomplete/error metadata,
unexpected replacements and compiler/export-data identity mismatches. Qualify
target-dependent declarations, generic instantiations and pointer/value method
sets before accepting the approach. If it cannot satisfy RA09, present B/C for
decision rather than approximating a type checker with names or regular expressions.

1. Replace `internal/load`'s handwritten grammar with bounded streaming
   `encoding/json/jsontext` parsing. Build the lossless node representation from
   standard-library tokens/raw spans; preserve number lexemes without float64
   conversion. Bound the reader before allocating the document and retain
   deterministic source pointers. Strict duplicate/Unicode handling is the
   default; do not enable permissive options. Test parser error classification
   without unchecked type assertions.
2. Replace suffix stripping and recursive reference expansion in
   `internal/resolve` with interned graph nodes keyed by canonical resource URI
   and fragment. Track `$id` bases and supported anchors; decode URI fragments
   and RFC 6901 tokens in their specified order. Empty pointer and `/` refer to
   different locations. Reject invalid escapes, invalid/overflowing indices,
   duplicate IDs and unresolved targets. Cycles are edges with visited/active
   states, not automatic depth failures. Enforce separate graph/depth budgets.
   A registered HTTPS identifier is resolvable without network access; absence
   from the local registry is what fails. Apply filesystem containment only
   when acquiring approved local resources.
3. Keep an explicit dialect/vocabulary capability registry in `internal/model`.
   Distinguish qualified assertions, annotations and unsupported required
   semantics. Unknown keywords are not automatically assertions under JSON
   Schema; the reviewed generation profile must additionally detect an
   unreviewed input change instead of silently omitting its intended semantics.
   Resolve required vocabularies/anchors/dynamic features completely when
   supported, otherwise report the unsupported feature before emission. See
   [JSON Schema 2020-12 Core](https://json-schema.org/draft/2020-12/json-schema-core)
   and [RFC 6901](https://www.rfc-editor.org/rfc/rfc6901.html).
4. Walk every exported reachable field, embedded type, method, parameter,
   result, type argument and constraint in `internal/sdkmap`; join deprecation
   comments to resolved symbols. Include unexported reachable SDK runtime/sealed
   types where they prevent safe ownership or construction. Emit compiled
   identity/signature probes and independent wire fixtures for approved aliases.
   SDK sessions, transport objects and handler signatures stay inside the bridge.
5. Emit `go/ast` nodes and format with `go/format`; use typed errors and stable
   sorted inventories. Derive presence/union/open-object code from the reviewed
   model, not field-name heuristics. Check all 155 dispositions and handwritten,
   test and platform collisions. Candidate qualification includes both modules
   and external consumers. Cross-compile tests on all three targets, then use
   native execution in the applicable repair exit; these are different proofs.

RA05–RA11 add empty-member pointer, registered HTTPS identifier, cyclic graph,
generic/pointer-method leakage, target-specific alias and deprecated embedded
member cases. R2 must not depend on the runtime module to share parser code:
keep module boundaries intact and share independent fixture specifications.
The R2 candidate remains staged until R3 supplies safe migration/publication;
R4/R5 subsequently requalify generated semantics and public runtime bindings.

##### R3 options and remaining work

| Option | Evaluation |
| :--- | :--- |
| A — Rooted file operations plus a durable recovery journal **(recommended)** | Moderate-to-high work; standard library and CGO-free. Provides verified recovery and mixed-generation detection without assuming a multi-file atomic replacement primitive. |
| B — Platform-specific replacement/locking adapters plus the same journal | Higher maintenance and native fault-testing cost. May improve individual platform guarantees, but does not remove the journal or make multiple file replacements transactional. Requires concrete evidence that A cannot meet an accepted case. |
| C — Immutable generation directories with a switched reference | High layout and migration cost; changes compiler/import/output ownership assumptions. The reference switch still needs platform qualification and recovery. Requires a separately reviewed publication/layout amendment. |

1. Use Go 1.27 `os.OpenRoot`/`os.Root` operations throughout publish, deletion
   and recovery, including temporary creation and rename. Lexical path cleaning
   or `EvalSymlinks` followed by unrooted mutation does not close a path race.
   Root containment is separate from ownership: reject unsafe aliases, symlinks
   or other file kinds under the accepted policy, verify existing identities,
   and preserve unrelated/manual edits. Check Windows reserved names and
   case collisions. See Go's [rooted filesystem guidance](https://go.dev/blog/osroot)
   and [`os` API contracts](https://pkg.go.dev/os).
2. Implement an explicit publication/recovery state machine: exclusive owned
   lock, verified original/candidate bytes on the destination filesystem,
   durable versioned intent, bounded replacements/deletions, manifest last,
   then cleanup. Do not infer ownership from age, an untrusted journal path or
   the old hashless manifest. Detect conflicts again at destructive boundaries.
3. Document exactly what each native filesystem supports. `os.Rename` does not
   promise atomicity on non-Unix targets, even within one directory. RA13 therefore
   requires verified process-interruption recovery and rejection of mixed
   output; neither rename nor a journal alone proves power-loss durability.
   Check write/sync/close/rename errors, qualify file and directory synchronization
   where supported, and fail/report an unsupported required durability guarantee.
   Do not suppress a sync error or advertise a universal atomic transaction.
4. Test real killed subprocesses and injected filesystem failures at each
   journal boundary. Include conflicting manual edits, originally new files,
   stale/live lock owners, repeated recovery, absent/corrupt staged data and
   manifest-last failure. A platform fault seam supplements actual native
   process/filesystem tests. Preserve recoverable state after cleanup failure.
5. Route all six CLI modes through the same validated inputs/ownership rules.
   Finish semantic/API diff and bounded immutable candidate-only update. Migrate
   the hashless manifest using reviewed baseline bytes; install R2 output only
   after candidate qualification and reversible journal preparation.

RA12–RA14 retain all existing fault cases and add native rename/sync limitations
to the evidence record. No fallback may overwrite unverified bytes. Commit R2/R3
together when necessary to avoid a broken intermediate generated/runtime tree.

##### R4 options and remaining work

| Option | Evaluation |
| :--- | :--- |
| A — Standard `jsontext` syntax guard, owned JSON v2 codecs and one canonical validator **(recommended)** | Moderate work; removes custom JSON grammar and keeps the accepted exact-number/profile contract. Share validation results within a boundary when their input identity is unchanged. |
| B — An independent full raw pre-scan followed by v2 decoding at every boundary | Straightforward separation of concerns, but adds repeated parsing and allocation. Valid if measurements and isolation justify it; deduplicate only where the same predicate and bytes are proven. |
| C — Fully repair and maintain the handwritten parser | Highest grammar, fuzzing and maintenance burden; no demonstrated benefit over the pinned standard library. Must meet exactly the same Unicode, duplicate, numeric and limit cases. |

Go 1.27 makes JSON v2 available by default; no experimental enablement is
required. Verify this at the minimum and preferred compiler pins and reject
an environment that disables it. Use the stable API, including current struct
tags, rather than copying pre-release examples. v1 compatibility behavior and
the SDK's actual conversions still need differential tests. See the
[Go 1.27 release notes](https://go.dev/doc/go1.27),
[migration guide](https://go.dev/doc/jsonv2-migration) and
[`jsontext` reference](https://pkg.go.dev/encoding/json/jsontext).

1. Replace both EOF-text checks in `codec/codec.go`. Accept only one complete
   root value followed by actual `io.EOF`; use `errors.Is`/`errors.As` and typed
   syntax errors. Truncation is never successful EOF. Enforce depth, byte,
   numeric-token and exponent limits while parsing; preserve numeric lexemes.
   Duplicate-name diagnostics use structured parser data, not quoted substrings
   extracted from error prose. Include valid surrogate pairs and invalid pairs.
2. Generate owned v2 hooks for presence, unions and open members using the
   stable API. Preserve unknown data only where the schema permits it. Keep
   lexical identity tests separate from semantic equivalence tests and invoke
   real SDK marshaling/dispatch in the differential suite.
3. Repair `internal/validation` around one immutable compiled-schema profile.
   In the pinned validator, `DefaultDraft` supplies a fallback, not rejection
   of an explicitly different dialect. Validate `$schema` and required
   vocabularies; install a registry-only loader that cannot fetch files/URLs.
   Snapshot caller-owned schemas/resources before compilation. Cache keys bind
   schema content, referenced resource closure and profile/version identities;
   enforce retention bounds and concurrent access without mutable shared input.
   See the [pinned validator API](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6@v6.0.3).
4. Keep generic `format` annotations distinct from mandated protocol checks.
   Route location assertions through the same production semantic functions;
   use the [pinned RFC 6570 parser](https://pkg.go.dev/github.com/yosida95/uritemplate/v3@v3.0.2).
   Do not treat Go's regex engine as complete ECMAScript regex support. Define
   and qualify the accepted subset; reject unsupported patterns with source
   pointers rather than silently changing their meaning.
5. Reconcile the 421 assertion, 155 definition and 36 format identities with
   maintained production-path cases. Add changed-resource/same-key cache tests,
   mutable caller input, explicit other dialect, network-loader denial and
   concurrent compilation. Prove actual duplicate/precision/assertion bypasses
   fail the intended tests. Retain RA15–RA17 as the acceptance owners.

##### R5 options and remaining work

| Option | Evaluation |
| :--- | :--- |
| A — Public low-level SDK registration with owned typed adapters **(recommended)** | Moderate-to-high parity work, but gives strict v2 decoding and validation an explicit boundary. SDK keeps transport/session/dispatch ownership; wrapper owns conversion, policy and result construction. |
| B — Keep generic SDK `AddTool` behind owned callbacks and mandatory guards | Less registration adapter code; must prove SDK decoding/defaults cannot violate owned v2 semantics. Generic registration still panics on declaration failure, so error conversion and exact raw/output enforcement remain difficult. |
| C — Qualify a different SDK pin with suitable public registration/codec hooks | Potentially less wrapper maintenance, but requires evidence of actual APIs, full requalification and an approved pin change. No newer release is assumed to solve these defects. |

The pinned public `(*mcp.Server).AddTool` supplies raw arguments and explicitly
leaves conversion, validation and result construction to the caller. This is
a supported composition point, not a second dispatcher. Generic `mcp.AddTool`
performs additional inference/default/error work; A must preserve required SDK
behavior deliberately. Inspect the [v1.8.0 registration source](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/server.go).

1. Define owned generic callback/request/result types; remove the public
   `sdk.ToolHandlerFor` and SDK middleware signatures. Keep SDK conversion in
   `internal/sdkbridge/`, a planned package rather than an existing qualified
   boundary. Implement the opaque transport seam and owned connect/session APIs
   before external-consumer parity tests; do not import transport subpackages
   from the root or export SDK extraction methods.
2. For low-level tool registration, qualify inference, explicit schemas,
   defaults, pointer/nil/`any` handling, title/annotations, non-object output,
   structured/text content and tool-error versus protocol-error behavior.
   Reuse the public pinned
   [schema inference API](https://pkg.go.dev/github.com/google/jsonschema-go@v0.4.3/jsonschema)
   where SDK parity requires it; do not copy SDK-private functions or replace
   canonical instance validation with another validator. Any direct import
   promotes the existing transitive dependency to direct with the same reviewed
   version and an explicit dependency-manifest update. Test generated v2 types
   against inferred tags/defaults; inference success alone is not parity proof.
3. Prevalidate declarations and normalize immutable SDK inputs before calling
   registration. The pinned SDK does not expose a typed declaration-panic
   contract: matching panic text or recovering every panic is not a safe
   classifier. The earlier instruction to convert identified declaration panics
   is satisfied by returning errors at owned prevalidation boundaries where
   possible; a genuinely unavoidable SDK panic needs a proven narrow boundary
   and review before conversion. Unexpected programmer faults propagate.
   Stage registry/cache updates transactionally, serialize name conflicts and
   roll back rejected registrations so corrected retries work. Do not hold a
   registry lock while invoking user callbacks.
4. Install one receiving bridge, then use coherent per-operation snapshots of
   owned configuration. Avoid re-adding the accumulated SDK middleware chain.
   Mandatory input/capability checks and authorization/admission must precede
   user middleware as well as handlers; test denial invokes neither. Trusted
   observation records the final outcome, including rejection. The result guard
   surrounds user transformations and checks the final returned value, including
   short-circuits and missing required structured content. Repeated registration,
   configuration updates and concurrent calls must retain declared ordering.
5. Distinguish validation layers: SDK parsing occurs before receiving middleware.
   Validate raw tool arguments before typed conversion, and validate all owned
   ingress/egress at the earliest available boundary. A parsed SDK request cannot
   prove original wire duplicate/number behavior. Preserve original connections;
   the complete stdio/HTTP/SSE wire guards and lifecycle remain Stage 7. Record
   that limitation in R5 evidence instead of claiming middleware solves it.
6. Keep Option D's SDK automatic fulfillment disabled. Exercise real owned
   sampling/roots/elicitation callbacks and SDK public map codecs through tools,
   prompts and resources. Preserve the pinned bounded-round behavior, request
   state, metadata, cancellation and error classification; do not blindly retry
   side effects. Test repeated input-required rounds, absent callbacks and wrong
   discriminators against the [pinned fulfillment implementation](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/mrtr.go).
7. Deep-copy mutable configuration and returned snapshots, use context-aware
   operations and ordinary wrapped/classified errors with working `errors.Is`
   and `errors.As`; bound public messages without exposing private causes on the
   wire. Keep response caching disabled. Exercise SDK-free external consumers,
   two servers/connections, handler errors, concurrent registration and policy
   changes. Run relevant native functional cases with CGO off and race cases
   separately with CGO on. RA18–RA20 and G23–G25 remain the exit owners.

##### Integration sequence and evidence required for the revised work

1. Finish and qualify R1's graph/accounting/preparation fixes first. Keep all
   unverified draft changes visible. A setup success is not a successful suite,
   lint plant, native qualification or repaired product gate.
2. Build the R2 candidate and type/alias contract, then R3's safe publisher and
   migration. Coordinate those changes when neither can land alone coherently.
3. Complete R4's owned codecs/validation profile and regenerate affected hooks;
   rerun R2 API, alias, candidate and serialization cases and R3 drift checks.
4. Complete R5's owned SDK adapters/live enforcement. Rerun affected RA groups
   and all applicable Stage 0–6 D/G obligations against the final tree. Preserve
   dependency identities so only genuinely equivalent leaves are reused within
   the run; changed source or policy invalidates affected results.
5. Qualify native Windows/amd64, Linux/amd64 and Darwin/arm64 on the required
   compiler floor/preferred versions; retain WSL as the additional local runner
   environment. WSL cannot stand in for Windows filesystem or process behavior.
   Use the existing SSH host-selection and heavy-run queue policy. Cross-builds
   remain CGO-free and are supplementary to native execution. Race is the sole
   CGO-enabled execution class.

Every repair exit still requires no blocker, all decided cases tested, and
deliberately broken scratch inputs caught by their intended checks. Record
command, source/input/tool/target identities, full bounded artifacts, named
failure and exit for each plant. Do not introduce new numerical/performance
claims without measurement. This refinement adds implementation detail to the
existing acceptance groups below; it neither removes cases nor declares them
passing. Stage 7 remains pending prerequisite evidence and execution approval.

#### Repair acceptance mapping

These are planned maintained groups, not existing tests. Python groups are
`unittest.TestCase` classes with named test methods; Go groups are top-level Go
tests with named subcases. Gate reconciliation uses the exact group IDs.
The full expected-state matrix is in MADR RA01–RA20 and the steps above.

| State / finding | Acceptance group and location | Phase |
| :--- | :--- | :--- |
| RA01 / A01 | `TestRepairA01Evidence`, Python inventory tests in `scripts/` | R1 |
| RA02 / A02 | `TestRepairA02GateAccounting`, `scripts/test_gate.py` | R1 |
| RA03 / A03 | `TestRepairA03LintAndPreAdd`, Python gate/pre-add integration tests | R1 |
| RA04 / A04 | `TestRepairA04Preparation`, Python setup tests | R1 |
| RA05 / A05 | `TestRepairA05StrictLoader`, generator `internal/load` | R2 |
| RA06 / A06 | `TestRepairA06References`, generator `internal/resolve` | R2 |
| RA07 / A07 | `TestRepairA07Vocabulary`, generator `internal/model` | R2 |
| RA08 / A08 | `TestRepairA08ProvenanceAndMapping`, generator CLI integration | R2 |
| RA09 / A09 | `TestRepairA09AliasClosure`, generator `internal/sdkmap` | R2 |
| RA10 / A10 | `TestRepairA10GeneratedContract`, generator `internal/emit` | R2; rerun after R4/R5 |
| RA11 / A11 | `TestRepairA11CandidateQualification`, generator CLI integration | R2; rerun after R3–R5 |
| RA12 / A12 | `TestRepairA12PathContainment`, generator publication integration | R3 |
| RA13 / A13 | `TestRepairA13Recovery`, generator publication integration | R3 |
| RA14 / A14 | `TestRepairA14CLI`, generator CLI integration | R3 |
| RA15 / A15 | `TestRepairA15RawCodec`, `codec/` | R4 |
| RA16 / A16 | `TestRepairA16ValidationProfile`, `internal/contracttest/` | R4 |
| RA17 / A17 | `TestRepairA17Locations`, `internal/validation/` with runtime callers exercised | R4 |
| RA18 / A18 | `TestRepairA18LiveEnforcement`, external runtime integration fixtures | R5 |
| RA19 / A19 | `TestRepairA19CanonicalRuntime`, `internal/contracttest/` external consumers | R5 |
| RA20 / A20 | `TestRepairA20RuntimeContracts`, root runtime tests and external fixtures | R5 |

Each group must have a passing valid control and an intended failing scratch
plant. Record the plant's exact command, mutation/input hash, named failure and
exit. A01's broken control is a missing/false completion inventory; A03 uses the
actual linter; A13 uses real interruption as well as fault seams. Missing groups
and empty selections are not success. Preserve original D/G group obligations;
RA groups supplement them and may share fixtures, not replace their coverage.

#### Verification, commits and rollback during repair

Use the original commands and host policy below, with R1's explicit selectors
and non-mutating pre-add implementation. Run documentation gates for records;
Python harness tests for orchestration; affected Go package checks after edits;
and queued, uncached full suites at each relevant phase exit. For cross-target
candidate builds use prepared compilers/caches and readonly offline modules.
For native OS behavior use the fleet's selected target and serialized heavy-run
queue; no Windows result may be replaced by WSL execution.

Number every repair build pass as `R<n> — pass <m>`, retaining failures. The
third-build deviation rule applies to these work packages; splitting a name or
changing host does not reset attempts. Do not add ad hoc retries or suppressions.
Commit each approved, coherent phase after required pre-add checks. If repairing
a gate exposes a defect assigned to a later repair, report the case and preserve
the failure; do not bypass a required pre-add check to commit. Prepare dependent
corrections as one coherent change where this amendment explicitly permits it,
otherwise use the deviation process. No push, tag or release is included.

Treat generator manifest migration as a recoverable local change, with exact
original bytes and verified candidate bytes. Rollback uses the verified journal
or a reviewed phase revert; never clean/reset the user's tree. Preserve the
audit report and pre-existing work. A rollback or change to source/input/policy
identities invalidates affected evidence and requires requalification.

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
   event size including framing (4 MiB default, 8 MiB declared ceiling,
   above-ceiling configurable to a 16 MiB hard limit — owner amendment
   2026-11-25; test all three tiers and the above-16 rejection), invalid UTF-8
   and duplicates after a valid event
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
   middleware, set the body policy explicitly — undeclared default 4 MiB
   (SDK parity), declared ceiling 16 MiB, above-ceiling declarations
   configurable with a 64 MiB hard cap, nothing above 64 MiB constructible
   (owner amendment 2026-11-25) — and enable modern request cancellation
   propagation. Test origin/Host rejection before handler invocation, and test
   the body-size boundaries: undeclared default 4 MiB, declared 16 MiB,
   configurable above-ceiling to 64 MiB, rejected declaration above 64 MiB,
   and oversize-input 413 mapping.
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

### Stage 9 amendment — selected CI/CD policy, 2026-11-24

This amendment records D1 A, D3 C and the D2 scope instruction. It adds the
Makefile, receipt verification and cases below to Stage 9; it does not start
Stage 9 or mark any acceptance case implemented.

1. Re-ground the source locations in
   [0009-REPORT-cicd-standardization-assessment.md](../reports/0009-REPORT-cicd-standardization-assessment.md)
   at execution HEAD. Preserve concurrent work and existing cases/budgets. Extend
   `scripts/gate.py`, `scripts/go-precheck.py`, their tests, `tools/versions.json`,
   the proposed Makefile and `verify.yml` as one gate implementation.
2. Implement shared module/case selection and non-mutating pre-add checks. Run
   all six native tuple/toolchain combinations on queued fleet hosts and GitHub,
   both modules with CGO off, readonly modules and offline gates after explicit
   setup. Run preferred-version race instrumentation separately with CGO enabled.
   Retain the existing conformance, fuzz and numerical budget requirements.
3. Implement immutable snapshots and one-hour exact-commit receipts. Bind
   policy/tool/input identities and every outgoing ref tip; reject age at least
   3,600 seconds, future timestamps and incomplete results. Consume fresh local
   vulnerability evidence supplied by the owner for each push. Do not install
   or reconfigure security scans or disclosure guards. Hook integration belongs
   in its owning repository and must preserve its disclosure behavior.
4. Add an honest GitHub summary that rejects missing/cancelled/failed cases.
   GitHub protection, rulesets, required-check configuration, visibility/billing
   changes and GitHub security-scan jobs are excluded.
5. Run each decided CI state case below, with baseline pass and broken scratch
   controls. Retain failure evidence and numbered Stage 9 passes.

| State | Planned acceptance case | Required evidence |
| :--- | :--- | :--- |
| CI01 | `TestCI01SharedGateContract` | Make/CI select identical modules/cases; planted command failure propagates |
| CI02 | `TestCI02NativeMatrix` | Both modules in six CGO-free native combinations and separate race evidence; wrong architecture/compiler/CGO and missing module rejected |
| CI03 | `TestCI03FreshReceipt` | Complete exact-commit receipt below one hour reused with fresh successful local vulnerability evidence |
| CI04 | `TestCI04ReceiptRejection` | Missing/malformed/altered/future/expired or mismatched evidence rejected, including one bad tip in multi-ref push; ages immediately below, at and above one hour |
| CI05 | `TestCI05LocalSecurityBoundary` | Missing/failed/reused scan evidence rejects qualification; disclosure failure remains effective; no scanner/guard mutation or GitHub scan job |
| CI06 | `TestCI06CISummary` | Failed/cancelled/missing/unexpectedly skipped result fails summary; complete success passes |
| CI07 | `TestCI07FleetLifecycle` | Queue serializes work; transfer failure, timeout and cancellation retain failure evidence and clean up owned resources |
| CI08 | Out-of-scope inventory | No GitHub enforcement, visibility or billing configuration is added or required for acceptance |

Additional Stage 9 exit: CI01–CI07 pass, all new plants are caught, CI08 remains
excluded, and no blocker remains. Receipt/security-boundary fixtures do not replace
actual native matrix execution. Existing execution authorization is unchanged.

### Stage 9 amendment — local-cicd-scaffolding contract, 2026-11-24

The dedicated skill is named `local-cicd-scaffolding`. Its reusable contract ends
at local pre-push readiness. This amendment specifies local adoption; it does not
start Stage 9, install a fleet skill, or approve pending R1–R5 execution. The
existing upstream implementation steps are separate repository work and must not
enter the skill's instructions, assets, configuration or acceptance dependencies.

1. Re-ground the local source evidence at execution HEAD: the four reference
   repositories' Makefiles, runners, pre-add checks, module inventories, tooling
   tests and accepted local policies. Inspect the canonical workspace skill and
   host-routing interfaces. Treat documented defects as defects, not precedent.
   Re-ground the call paths in
   [0012-REPORT-local-gate-duplication-assessment.md](../reports/0012-REPORT-local-gate-duplication-assessment.md).
   Capture the transitive command graph, including nested validators, scripts,
   self-test baselines and hook adapters; identify duplicate requests separately
   from intentionally distinct build/test contexts. Preserve concurrent work.
2. Record the skill implementation/deployment in its owning library's MADR/PLAN.
   Keep the name and scope selected here. Validate the skill with the fleet's
   skill-authoring gate and deploy through its established registration/linking
   mechanism; verify discovery before claiming installation. This consumer plan
   does not authorize edits in that separate repository.
3. Build a local requirements ledger covering every area in the associated
   MADR's complete local standards inventory. For each entry, record its source,
   applicability, command, module/target coverage, required data, acceptance test
   and broken-input control. Preserve all decided product cases and budgets.
4. Adapt the versioned local scaffold to `scripts/gate.py`, pre-add checks,
   local pins and the proposed thin Makefile. Preserve workspace-owned templates
   and global hook/security ownership. Keep pinned policy independent of upstream
   configuration and agent-skill installation. Validate the Python entry points
   and native Make invocation on each supported host.
   Define canonical leaves and dependency edges; legacy modes and Make targets
   select them. Replace nested aggregate execution with shared prerequisites.
   Attach every existing required checker to an owner, including documentation
   checks omitted by legacy aggregates. Fix the current gate correctness defects
   before using their outputs as reusable evidence. Keep standalone target use
   complete by resolving its prerequisites through the same graph.
5. Qualify inspection, first application, repeat application, managed/unmanaged
   conflicts, explicit overrides, upgrade/rollback, unsafe paths, concurrent
   changes, permission failure and interrupted publication in isolated fixtures.
   Compare content, mode, link target and modification time; ignore no generated
   scaffold file when asserting the second application is a no-op. Readiness
   checks may write untracked evidence/cache outside the source snapshot; that
   does not authorize scaffold refreshes to rewrite unchanged files.
6. Qualify the local gate graph on immutable candidate snapshots using the
   approved fleet queues, all modules and native target/toolchain combinations.
   Require the four runner roles recorded in the MADR: native Windows over SSH,
   WSL through the same SSH connection, session-local Darwin, and standalone
   Ubuntu over SSH. Probe actual environments and run both compiler versions in
   each: eight baseline cells, with both modules in every cell. Retain the six
   native cells independently of WSL's two additional cells. Serialize Windows
   and WSL heavy work on their shared physical queue; preserve the existing WSL
   race exclusion and run preferred-version race cases on the three native hosts.
   Set and verify `CGO_ENABLED=0` for all non-race Go operations and child builds;
   isolate `CGO_ENABLED=1` to race-test processes and their instrumented builds.
   Check that race settings cannot leak into subsequent commands or ordinary
   fixture builds, alter persistent Go settings, or certify deliverable artifacts.
   Run the existing product cases, tooling tests and scratch failure controls.
   Verify exact-commit receipts and owner-supplied security evidence under
   CI03–CI05. Hook-chain changes, if needed, remain in the hook owner's plan.
7. Record the chosen scaffold version, repository extensions, full per-case
   evidence, numbered execution passes and remaining limitations. Do not claim
   idempotence from a clean text diff alone or declare gates effective from only
   successful inputs. Every new gate needs an observed nonzero broken-input run.
8. Qualify deduplication with LC13–LC16. Trace actual subprocess requests, not
   only declared target names. Prove two aggregate paths and concurrent requests
   execute a matching leaf once, while changed tags/compiler/CGO/workspace mode,
   runner environment, fixture, selection/count and relevant history remain
   distinct. A shared failure must fail every dependent readiness result.
   Inventory analyzer versions/settings and deliberate failure behavior before
   relocating checks; prove identical coverage rather than deleting overlap by
   name. Required rules, tests and diagnostic classes must remain covered.
9. Separate reusable producers from validators, starting with tests/coverage,
   module/file inventories and clean gate-self-test baselines. Preserve each
   mutated fixture's distinct execution and history-sensitive baseline. Record
   fresh security evidence once per required scope and push attempt through its
   existing owner; do not make repeated scans permissive to reduce cost.
10. Measure baseline and candidate on comparable snapshots and queued hosts,
    recording cache state, wall time, queue wait, setup/transfer, critical path,
    invocation counts and reuse reasons. Keep all eight baseline environment/
    compiler cells, both modules, native race runs and accepted product cases.
    Record remaining necessary overlap explicitly. Reference-repository changes
    require their own implementation records; this plan does not silently alter
    their accepted checks, pre-add policies, hooks or execution schedules.

| State | Planned acceptance case | Required evidence |
| :--- | :--- | :--- |
| LC01 | `TestLC01LocalBoundary` | Local qualification works without a configured remote or upstream files; unrelated upstream content is unchanged and cannot affect local pins or case selection; no upstream service call or operation |
| LC02 | `TestLC02Ownership` | Existing workspace files, hooks, owner-managed guards and product targets survive adoption; duplicate runner/target fixtures are rejected |
| LC03 | `TestLC03StandardsCoverage` | All inventory areas have applicable checks or recorded reasons; plant an omitted required case/module/gate and observe rejection |
| LC04 | `TestLC04Idempotence` | Read-only inspection changes nothing; applying twice preserves all file bytes, modes, links and modification times, including managed metadata |
| LC05 | `TestLC05Drift` | Named overrides survive refresh; conflicting edits and a concurrent base change reject publication without losing user content |
| LC06 | `TestLC06FailureRecovery` | Malformed/unsupported profile, path escape, external symlink, denied write, concurrent application and killed process fail clearly; recovery preserves unrelated files and emits no false readiness |
| LC07 | `TestLC07Migration` | Versioned upgrade followed by rollback restores the prior managed state; concurrent unrelated edits remain; unsupported migration fails |
| LC08 | `TestLC08HonestReadiness` | Required modules and native targets execute; wrong architecture/compiler/CGO, failed formatter with empty output, empty selection, unfinished test, skip, malformed result, missing tool, timeout and cancelled host prevent readiness |
| LC09 | `TestLC09EvidenceValidity` | Exact candidate/ref-tip matching and CI03–CI05 age/security rules; changed policy/tools/inputs, tampered evidence and one unqualified outgoing tip fail |
| LC10 | `TestLC10RepositoryProfile` | Product fixtures and budget values remain repository-owned; preserve explicit applicability; a missing required extension or planted contract/budget failure prevents readiness |
| LC11 | `TestLC11FleetRunners` | Eight baseline environment/compiler cells cover both modules; SSH and WSL routing reach the correct runtime; shared Windows/WSL contention serializes; wrong environment, unreachable host, omitted WSL result or Linux substitution fails readiness; private identifiers stay out of committed artifacts |
| LC12 | `TestLC12CGOIsolation` | Inject inherited CGO on, child-process leakage, a non-race build with CGO on, and a race artifact presented as baseline evidence; each is overridden to the required mode or rejected. A race run uses CGO on only with qualified C tools; its ordinary child build and the following build prove CGO off; no persistent environment change |
| LC13 | `TestLC13SharedExecution` | Two aliases, nested selections and concurrent requests launch an equivalent leaf once; every consumer cites its result; planted cycle, duplicate execution and missing obligation owner fail |
| LC14 | `TestLC14CoverageEquivalence` | Before/after coverage ledger matches all required rules, modules, files and contexts; a planted analyzer gap, stricter formatter failure lost during consolidation, missing nested module or incorrect coverage scope fails |
| LC15 | `TestLC15ReuseInvalidation` | Changed source/config/tool/target/compiler/tag/CGO/workspace/fixture/history invalidates matching dependencies; mutate a supposedly safe document input; partial, failed, expired or clean-for-mutant results cannot certify readiness |
| LC16 | `TestLC16CostAccounting` | Real recorder counts match executed and reused leaves; each required repeat has a distinct justified context; queue and service time are separated; missing timing/count evidence or a duplicate concealed as reuse fails |

Additional local adoption exit: LC01–LC16 pass with deliberate scratch failures,
all applicable inventory areas are covered, no blocker remains, and the repeated
application is proven unchanged. The gate graph has no unexplained duplicate
executions and retains the complete required coverage, with measured cost evidence.
These are planned tests, not existing commands or completed evidence. Report
skill deployment, consumer adoption and product
qualification separately so one cannot masquerade as another's completion.

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
| G23 | `internal/contracttest: TestG23OwnedFulfillment` | R5, 6, 9 | Real owned sampling/roots/elicitation callbacks, disabled SDK automatic fulfillment, tool/prompt/resource request-state paths where required |
| G24 | `internal/contracttest: TestG24InputDiscriminators` | R5, 6, 9 | Missing/wrong roots/role/action keys fail loudly at actual SDK codec conversion; valid controls complete |
| G25 | `internal/contracttest: TestG25BridgeRemovalBoundary` | R5, 6, 9 | Public consumer signature fixture contains no deprecated/bridge-specific SDK types; scratch bridge-removal design check preserves consumer compile contract, without dropping currently required production features |

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
| JSON frame / HTTP JSON body | 4 MiB undeclared default (SDK parity); declared ceiling 16 MiB; above-ceiling declarations configurable to a 64 MiB hard cap; both receive and send; nothing above 64 MiB constructible; HTTP overflow 413 (owner amendment 2026-11-25) |
| SSE event | 4 MiB default including framing; 8 MiB declared ceiling; configurable above-ceiling to a 16 MiB hard limit, nothing above constructible; per event, not stream lifetime (owner amendment 2026-11-25) |
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

The commands below are required deliverables in stage 1 and subsequent stages.
Some now exist but have audit gaps; R1–R5 restore their contracts. Invoke one command at
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

### 2026-11-27 — R3 pass 7 exit: installed Make/lint and regeneration passed

After all four native qualification lanes passed, ordinary verified CLI
generation published exactly `aliases.go`, `codecs.go`, `constants.go`,
`owned.go` and `generated-manifest.json`. The installed manifest is
`2d2d88d86a78b2806043c9b7a2580001ea83b952040ad57a1fc7f223500f0112`.
All nine artifacts match the retained proposed SHA-256 inventory; the
handwritten protocol file is unchanged. Product tool, module, input, schema
and profile pins were preserved; the previous review remains archived.

Queued native Darwin `make pre-add-check lint` passed with one dispatcher
invocation: 70 runtime tests across five packages, 346 generator tests across
twelve packages, readonly formatting, standalone vet and both pinned lint
owners, including generated documentation. Ordinary operations used CGO off.
Repeat CLI generation returned `unchanged`; the final CLI check returned `ok`,
with both modules' three-target source/test matrices passing and no drift or
unowned output. All ten protocol files retained SHA-256 and nanosecond mtimes
through repeat generation/check. No new source revision followed qualification.

The literal hook target, its seven Make regressions and all required broken
controls are verified. The R2/R3 source/output prerequisite is qualified for a
coherent commit on main after record/link/Markdown and staged identifier checks.
Those checks passed: all record links, all 36 Markdown files and the staged
97-file identifier scan. Git's whitespace diagnostic still reports the raw
schema's extra final blank line at `protocol/schema.json:3963`; its bytes are
identical to both the pinned input and that input at HEAD, with SHA-256
`907dcebce6aaeb16c21421ffef8c207093d88b9442fee340658a99cacb06b551`.
Preserve that reviewed raw artifact rather than change its input identity for
an advisory. Formatting and both lint owners passed; no gate was disabled.
The full PLAN remains in progress: R4/R5 and Stage 7 are not completed by this
exit. No push or tag is included. Layout L0 re-grounds at the resulting commit
before any runtime or bridge relocation.

### 2026-11-27 — R3 pass 7: all native qualification lanes passed

All four `scripts/qualify_r3.py --allow-network` jobs completed with zero
failures against source identity
`1e841afbadc7eb5abf86d9d048712989bb33003398d863e086a7bf95b1c931cf`.
The native classes were Darwin/arm64, Windows/amd64 and physical Linux/amd64,
plus WSL Linux/amd64. Each accounted for 346 passing generator tests across
twelve packages, including actual candidate runtime pre-add and both modules'
three-target source/test compilation. Pinned analyzer checks passed with
generated files included. The seven Make cases passed, including literal hook
discovery, standalone/combined pre-add dispatch and failed-command propagation.
All eight documentation plants and the retained loader, reference, alias,
profile, emitter, candidate and fifteen publication controls were caught.
Every lane's source-preservation check passed; normal execution forced CGO off.
No new race result is claimed for this comment/Make-only revision.

Safe publication into the session checkout and the grouped installed
Make/pre-add, regeneration and drift checks are queued in that order under
the host's shared heavy-run queue. No generated file is patched by hand.
The prerequisite is not yet committed, so layout L0 remains waiting.

### 2026-11-27 — R3 pass 7 native qualification started

The Makefile now declares `pre-add-check:` literally and shares the existing
single dispatcher prerequisite with the other goals. All seven Make foundation
tests passed in preparation. Actual scratch Make runs covered standalone
pre-add, combined pre-add/lint/test and lint/test, each with one dispatch and
nonzero failure propagation. The deliberately shared-only scratch rule failed
discovery with `FAIL pre-add hook cannot discover literal target`.

Freeze this source and run `scripts/qualify_r3.py --allow-network` through the
native host queues for Darwin/arm64, Windows/amd64, physical Linux/amd64 and
WSL. Each driver records its source identity and checks preservation. Retain
the previously reviewed documentation-only candidate; no publisher, schema,
profile, API or SDK behavior changed in this revision. The same separately
attributed physical-Linux race evidence remains applicable to unchanged R3
behavior. Qualification, safe live promotion and a coherent commit are still
required; no pass 8 is inferred.

### 2026-11-27 — R3 pass 7 approved: literal hook target and qualification

The owner selected A from the hook-discovery deviation prompt. Authorize one
additional source revision and qualification cycle, pass 7, for a literal
`pre-add-check:` Make entry and a scratch regression that rejects the previous
shared-only rule. Retain combined-goal coalescing and verify actual standalone
and combined pre-add dispatch, including nonzero propagation. The installed
hook and all acceptance gates remain unchanged.

Run the complete frozen-source qualification on native Windows/amd64,
Darwin/arm64, physical Linux/amd64 and WSL through the existing queues. Preserve
the reviewed generated candidate and input/profile/schema identities. After
qualification, publish through the verified CLI, check repeat generation and
installed pre-add/lint, and commit the coherent prerequisite on main. Then
resume the already approved layout PLAN. No push, tag, pass 8, broader tooling
scope or bypass is authorized. Stop if another source revision is needed.

### 2026-11-27 — R3 pass 6: hook-discovery defect; allowance decision pending

The frozen source identity was
`1a7b6666ccaf522db21252e28f1189a92133c7bf31f576d04ba86f9421ca5672`.
Windows/amd64 and WSL completed `scripts/qualify_r3.py --allow-network` with
zero failures. Each accounted for 346 generator tests across twelve packages,
passed the six Make foundation cases, caught all eight handwritten/generated
documentation plants through actual Make, and passed retained publication plants.
Their source-preservation checks passed. Physical Linux passed preparation and
the generator pre-add checks before interruption; macOS remained queued and
executed nothing. Neither incomplete lane is a passing native qualification.

Read-only inspection of the installed hook found a remaining adapter defect:
it selects Make only when a line matches `^pre-add-check:`. The new Makefile's
shared rule at lines 10–12 includes that goal without such a literal entry.
The existing combined-goal regression passes, but does not exercise discovery.
This defect was introduced by the new Make foundation; the earlier fallback
selection was the pre-existing problem. No global hook change is appropriate.

Correcting the literal target and adding a scratch discovery/dispatch control
requires another source revision. The approved allowance ends at pass 6, so
implementation and publication stop before pass 7. Cancel only this repository's
queued macOS and unfinished physical-Linux pass-6 jobs rather than spend another
qualification cycle on this known incomplete adapter. Both wrappers exited 130;
the macOS ticket was removed without running, and the remote Linux process group
was stopped, with its interrupted scratch retained temporarily by the runner.
No source correction, new output publication, prerequisite commit or runtime
relocation is included in this stop.

The owner was offered three resolutions, with unchanged acceptance gates:

| Option | Work and cost | Consequence |
| :--- | :--- | :--- |
| A — Authorize pass 7 **(recommended)** | Small literal-target correction and scratch regression, followed by one additional native qualification cycle. | Commit the qualified prerequisite, then resume the approved layout. |
| B — Authorize passes 7–8 | Same repair and verification scope, with a second revision available if a further defect is found. | Same exit, with a larger iteration allowance. |
| C — Review before another allowance | Preserve the current candidate and inspect the proposed repair before resuming. | The prerequisite commit and layout remain waiting. |

Doing nothing leaves the hook on its legacy fallback and the prerequisite
uncommittable. No option bypasses that hook or removes a validation. Record the
owner's selected resolution before changing source or starting another pass.

### 2026-11-27 — R3 pass 6 native qualification started

Freeze the Make foundation, canonical module selectors, standalone documentation
analyzer, emitter comment/header correction and preserved candidate reviews.
The Make regression suite has six passing cases, including real combined-goal
dispatch, deliberately failed Make propagation, worker admission and wrong
version/builder/module metadata controls. Focused Go parser tests passed and
reported rejection of the old header placement and missing method documentation.
Generator `make lint MODULE=generator` passed both pinned analysis owners.

Run the complete `scripts/qualify_r3.py --allow-network` driver on native
Darwin/arm64, Windows/amd64, physical Linux/amd64 and WSL. New Make/lint controls
join the retained R2/R3 controls; normal checks force CGO off. This is the last
approved R3 pass. Retain separately attributed prior race evidence for the
unchanged publication/ownership/CLI behavior; no ordinary operation enables CGO.
No publication of the corrected output or layout move precedes this exit.

### 2026-11-27 — Make foundation A approved; R3 pass 6 preparation

The owner selected A from the prerequisite deviation prompt. Execute the new
ML01–ML05 ruling in the paired MADR before the coherent R2/R3 commit, then
resume the already approved layout PLAN. This is the remaining approved R3
source revision/pass 6; no pass 7 is authorized.

The documentation-only candidate compiles both module/source/test matrices
with manifest
`2d2d88d86a78b2806043c9b7a2580001ea83b952040ad57a1fc7f223500f0112`.
`api.json`, `bindings.json`, `coverage.json`, raw schema and profile identities
are unchanged. The previous migration review is retained at
`tools/r2-candidate-migration-pre-lint.json`; the current review captures the
five changed Go/manifest artifacts. The initial capture correctly refused an
addition to the product-hashed tool manifest. Keep that manifest byte-identical
and place the new maintainer-only Revive pin in `tools/documentation-lint.json`.
No product input guard was changed. The final native qualification still
precedes verified publication.

The standalone Revive controls caught all four rule violations in both
handwritten and generated scratch files through actual `make lint` on native
Windows. Positive controls passed. Additional handwritten preparation found
one undocumented loader constant block and a URI-template function comment
attached to preceding variables; document the block and attach the existing
function comment to its function, preserving behavior.

Preparation probes: the Make selector tests passed and the parser controls
rejected the old misplaced header and an undocumented generated method.
The first fleet lint probe correctly refused an unprepared older global
linter. After explicit tool preparation, handwritten rules fired, but the
generated-file plants exposed the pinned golangci Revive adapter's internal
generated-file skip. Inspecting that adapter and its pinned Revive source
confirmed the cause. Within ML02, pin standalone Revive at the same dependency
version, 1.17.1, and give those four rules one owner across all files. Require
its verified build/module metadata and preserve generated analysis through the
configuration and repeated broken-input controls. The blank-import control now
uses a side-effect import; Revive deliberately accepts a valid embed directive,
so an embed directive is not a broken input for its modern rule.

1. Re-ground the four-reference assessment and shared-hook dispatch at HEAD
   `ff6d700`. Preserve the staged R2/R3 work and the installed reviewed protocol.
2. Add a thin Makefile and Python dispatcher for existing preparation,
   formatting, lint, vet, test, generation/build, inventory, documentation and
   checker-self-test owners. Combined goals share equivalent leaves. Expose
   `pre-add-check` through the existing canonical precheck, with queued heavy
   execution and no shell shim or global-hook edit. Do not claim complete
   preflight, API/wire readiness or Stage 9 receipts.
3. Add equivalent modern exported/package/naming/blank-import documentation
   checks to the pinned lint configuration, retaining generated analysis.
   Fix handwritten export comments and the emitter's marker/method comments.
   Strengthen header verification to inspect parser-recognized placement and
   exported method docs, with old malformed/omitted cases in scratch.
4. Preserve the pass-5 migration review separately, render the new candidate,
   compare its public declarations and input/profile/schema identities, and
   retain the new qualified output review. No owned output is hand-edited.
5. Run the focused new controls and the complete existing R3 qualification
   matrix for this frozen revision. Use a queued fleet host for heavy work;
   normal CGO remains off, with race attributed separately. Run actual Make
   targets and prove nonzero propagation on broken scratch input.
6. Publish through the verified R3 CLI only after qualification; repeat normal
   generation/check and require unchanged bytes/mtimes. Record pre-add,
   documentation and identifier results and commit the coherent R2/R3 phase.
   Then re-ground L0 at that commit before starting the layout migration.

Exit: no blocker, ML01–ML05 and retained RA cases verified, required plants
caught, `make lint` passes, the installed-hook route uses the same checked
implementation, and the prerequisite phase is committed. A new scope decision
or a need for R3 pass 7 stops for owner review.

### 2026-11-27 — R3 deviation: commit lint gate and missing Make interface

The attempted coherent R2/R3 `git commit --no-edit` was blocked before commit by
the agent harness's shared precommit quality gate. The pinned repository gate
had passed, but the shared hook did not discover its Python-only entry point:
there is no executable shell adapter and no root Makefile `pre-add-check:` target.
It therefore selected its generic legacy `golint` fallback. The commit was not
created; the qualified implementation/output and records remain staged. No
runtime relocation has started.

Representative exact findings and provenance:

| Evidence | Finding | Provenance |
| :--- | :--- | :--- |
| `cmd/mcpgen/internal/acquire/acquisition.go:27:6` | `exported type Options should have comment or be unexported` | New R3 export; analogous findings name acquisition Result, publication Entry and journal/ownership constants. |
| `cmd/mcpgen/internal/emit/emit.go:7:2` | `a blank import should be only in a main or test package, or have a comment justifying it` | The blank embed import was already present in HEAD. |
| `cmd/mcpgen/internal/model/model.go:27:2` | `exported const ClassAssertion should have comment (or a comment on this block) or be unexported` | The constant block was already present in HEAD. |
| `protocol/owned.go:17:1` | `exported method AudioContent.MarshalJSONTo should have comment or be unexported` | Newly installed R2 generated method; other JSON methods and union accessors have the same producer defect. |
| `protocol/owned.go:1` | Generated marker appears after the `package` keyword, not as a leading standard comment. | Emitter AST formatting defect; current `TestGeneratedHeader` checks substring presence and misses placement. |

The owner directed that `make lint` must pass and requested comparison with all
four reference Makefiles. The refreshed evidence, target contract and proposed
order are in
[0012-REPORT-local-gate-duplication-assessment.md](../reports/0012-REPORT-local-gate-duplication-assessment.md#makefile-reassessment--2026-11-27).
Their shared hook contract supports `make pre-add-check`, so a thin Makefile
can delegate to the existing Python runner without a shell shim or global-hook
edit. Documentation/naming findings must remain checked by the modern pinned
linter; the new entry point must not weaken checks or hide generated findings.

Proposed repair scope, pending owner selection under the deviation rule:

| Option | Work and cost | Effect on the approved layout |
| :--- | :--- | :--- |
| A — Canonical Make foundation and focused lint repair **(recommended)** | Moderate: expose existing check owners through thin Make/Python selectors; add real `lint` and `pre-add-check`, modern equivalent documentation rules and broken-input controls; repair handwritten comments and emitter header/method docs; review new generated hashes and requalify under the remaining R3 pass 6. Keep input/profile/schema pins fixed. | Complete the coherent R2/R3 prerequisite commit, then resume approved L0/L1. Broader preflight/security/stage-specific graph work remains explicitly scheduled rather than claimed complete. |
| B — Complete local validation graph before relocation | Larger: A plus graph/receipt deduplication, all local target families, fleet dispatch and owner-managed security evidence integration. Existing incomplete API/wire/product acceptance still fails until its planned implementation is complete. Phase this separately; no additional R3 build passes are silently granted. | Layout remains approved but waits for the expanded tooling work and its qualification. |
| C — Retain assessment and hold execution | No source/tooling changes; keep the staged generation and records intact. | The commit gate remains blocked, `make lint` remains unavailable and L1 cannot begin. |

No source, linter, Makefile or global hook was changed for this assessment.
Do not hand-edit generated output, bypass a hook, add an exclusion, discard an
assertion or reset staged work. Record the chosen scope before implementation.
If no repair is selected, the coherent R2/R3 commit and runtime move remain
incomplete despite the valid functional qualification evidence.

### 2026-11-27 — R3 pass 5 exit: publication and installed checks passed

The four native jobs completed against source identity
`9a284508e220fa21f43a106efa2d39f3ebacd128bb3fccd1c7a2381826f3d902`.
Each reported `343 tests passed; 12 packages accounted`, `0 issues`, all shared
R2 controls passed, all fifteen publication/CLI bypass plants caught,
`pass source-preservation` and `R3 qualification failures 0`. The native
Darwin/arm64, Windows/amd64, physical Linux/amd64 and WSL Linux/amd64 driver
elapsed times were 212, 222, 507 and 145 seconds respectively, excluding queue
waits. The physical-Linux actual-CLI race run from pass 4 also passed (323.349
seconds); that earlier result is explicitly attributed to pass 4, whose two
fixture/control errors were corrected and caught by pass 5. All ordinary
operations used CGO off; only the separately attributed race command enabled it.

After the owner accepted the independent layout pair, its documentation-only
commit `ff6d700` preserved the required boundary before runtime relocation.
The same R3 implementation then performed the live explicit legacy bootstrap
under the session host queue, offline and with CGO off. `generate` exited 0
with `status: complete`, publishing the nine reviewed generated files with
manifest SHA256
`57de122dd1234b6b99ff922c126413dd32c92800f3b8c610962616e5648531c6`.
The schema/profile pins and retained migration review were unchanged.

Installed-generation validation, also queued on native Darwin/arm64:

* `python3 -B scripts/go-precheck.py --module generator`: exit 0; formatting,
  standalone vet, all 343 generator tests across twelve packages and lint
  passed. Its actual-product test reused the complete runtime pre-add owner
  inside the installed candidate; no duplicate standalone runtime suite ran.
* Ordinary `generate --repo ../.. --format json`: exit 0, `unchanged`, empty
  changed-file list. All ten protocol files, including handwritten `doc.go`,
  retained their exact SHA256 and nanosecond mtimes after regeneration/check.
* `check --repo ../.. --format json`: exit 0, `ok`, no drift, unowned output or
  interruption; bootstrap no longer required. Runtime and generator source
  and tests compiled for Darwin/arm64, Linux/amd64 and Windows/amd64.

The guide [generate-protocol.md](../guides/generate-protocol.md) documents
ordinary generation/check and explicit verified bootstrap/recovery. The R2
candidate and R3 publication functional checks passed; their coordinated phase
commit remains blocked by the subsequent harness lint finding recorded above.
The staged change contains their implementation, generated output and evidence.
This does not
declare R4/R5, historical whole-stage exits or Stage 7 complete. Runtime layout
relocation remains the subsequent approved
[0014-PLAN-organize-mcp-runtime-package.md](0014-PLAN-organize-mcp-runtime-package.md).

An additional staged Git whitespace diagnostic reports
`protocol/schema.json:3963: new blank line at EOF`. The generated schema is
byte-for-byte identical to both the pinned input and its existing HEAD object,
including that upstream trailing blank line. This is outside the R3 acceptance
cases and is retained as an advisory follow-up; changing it would violate the
reviewed raw-schema identity. No formatter, lint rule or assertion was relaxed.
Required pre-add, record/link, Markdown and identifier checks all passed.

### 2026-11-27 — R3, pass 4 results and pass 5 preparation

All four native environments passed the complete 343-test / 12-package
generator inventory against frozen source identity
`f02cc3caff9c4e008d04ed10df1cb17b3f8d73cc2abda377dec2f9bc20765de7`.
This includes actual CLI Git bootstrap, staged-change refusal, deterministic
repeat generation and retained mtimes, inspection/check, full restore and the
three killed-CLI checkpoints. Formatting, vet, preparation and the existing
loader/model/reference/alias/profile/emitter controls passed. The two corrected
candidate digest plants now caught their intended test assertions.

The aggregate failed for two new fixture/control findings: `errcheck` at
`cmd/mcpgen/cli_publication_test.go:93` requires handling the checkpoint write
error; the CLI scope mutant lacked the reviewed root files needed by its
`profileFixture`, so it failed setup instead of the intended nonce assertion.
Return the checkpoint write error and copy the same independently pinned root
inputs already used by the selected-profile plant into that scratch layout.
Retain all helper containment and nonce checks. Neither failure grants negative
evidence for the missing scope control. The additional physical-Linux actual-CLI
race result is still being collected when this entry is prepared.

The record/link check also exposed four historical audit links to replaced
`publish.go`/`recover.go`. Pin those links to the audit's existing immutable
baseline, whose objects were verified locally, preserving its original findings
and line references. Do not relabel the repaired implementation as the source
of the historical defects.

Pass 5 requalifies the corrected fixtures/controls with the same complete native
driver and explicit race-only CGO policy. No live protocol publication or phase
commit has occurred. The separately selected named `mcp/` layout is recorded in
[0014-MADR-organize-mcp-runtime-package.md](0014-MADR-organize-mcp-runtime-package.md);
its proposed PLAN does not relocate source during R3 or extend this allowance.

### 2026-11-27 — R3, pass 4: full pre-publication qualification started

Apply the two pass-3 plant corrections and lint predicate fix. The actual CLI
fixture owns its temporary Git HEAD/index and borrows only immutable source
objects read-only. It verifies frozen Git evidence, wrong frozen bytes, dirty
staged owned output, explicit bootstrap, repeat generation/mtimes, complete
inspection, ordinary check and full restoration. Actual CLI children are killed
at staging, first replacement and final cleanup. A reaped fixture owner is
explicitly resolved by its parent; production never steals a stale lock. Check
must fail mixed output and may accept a fully installed terminal generation
after a cleanup interruption. Capability-scoped helpers cannot target live roots.

Run `scripts/qualify_r3.py --allow-network` on the native target matrix and WSL
against one frozen source revision. The driver shares the R2 control list,
invokes each pre-add/test/control owner once, and adds publication/CLI controls;
it does not nest the R2 driver or duplicate its test suite. A new Git-output
bound control also proves `io.Copy` cannot bypass the checked writer. No live
publication occurs in this driver. Its results must precede safe live promotion,
post-installation checks and the coherent R2/R3 commit.
The physical-Linux job adds `--race` for the new actual-CLI fixtures; all normal
checks still force CGO off and only that `go test -race` command enables CGO.

### 2026-11-27 — R3, pass 3: Windows cases and Linux race passed; lint/plant fixes

Frozen source identity:
`1d2f63195d7a55ce8b2142f1dd30b528657af4b5e7abe748a026667de3476c55`.
Windows completed in 78 seconds with no queue wait: all 336 tests / 12 packages,
formatting, vet, preparation, source preservation and 13 publication/CLI bypass
controls passed. The early directory-swap case and late native-refusal case both
passed. All eight candidate controls ran; six caught their intended bypass.
Two plants failed compilation after the ownership refactor because disabling
the digest predicate left `body`/`sum` unused. These are new plant defects,
not successful negative evidence. Keep the digest expressions evaluated and
disable only their Boolean result in scratch, then require the original test
diagnostics. Lint reported QF1001 on the late-swap refusal predicate; name the
positive native-denial predicate before negation, retaining all checks.

The physical-Linux race job completed in nine seconds with no queue wait:
publication, ownership, profiles and acquisition all passed under `go test
-race`, with CGO enabled only for that command. All processes exited and its
scratch tree was removed. Ordinary output remains unchanged. Continue with
actual CLI bootstrap/publication and killed-CLI recovery fixtures, remaining
negative controls and native qualification inside approved passes 4–6. No R3
exit, live promotion or joint commit is claimed from this partial pass.

### 2026-11-27 — R3 deviation approved: passes 3–6

The owner selected A from the pass-2 handoff: authorize up to four further R3
source revisions, numbered passes 3–6, for the recorded findings and existing
CLI/bootstrap/recovery/diff/update scope, native qualification, a physical-Linux
race check, safe candidate publication and the coherent R2/R3 commit only after
the complete exit passes. This resolves the previous third-build stop and the
automatic review rejection; it does not authorize a push or tag. Stop at an
unresolved new decision or the exhausted allowance. Preserve every accepted
case, fault fixture and deliberate-failure control.

Pass 3 starts by removing the unused default mapping wrapper and distinguishing
a successful late directory swap from native refusal. Require the checkpoint
to run, retain the exact swap error and verify the original public inventory
when Windows prevents the rename. Add a real swap before the destination handle
opens on every target, with a same-inventory replacement and independent byte
and sentinel checks. A scratch identity-guard bypass must make that test fail.
Then share hashed ownership across qualification/publication, exclude only
reserved operational metadata from compiler snapshots, and wire the exact
qualified bytes plus source revalidation into publication. Preserve frozen Git
object bootstrap verification, complete restoration and strict CLI contracts.

Pass-3 verification covers the new early/late directory-race fixtures, shared
ownership and bounded scratch stale removal, operational-metadata exclusion,
installed reviewed-generation regression, CLI help/usage/format/JSON channels,
hashed semantic sidecars and bounded immutable candidate acquisition. Run the
core driver and candidate/publication bypass controls natively on Windows;
run the publication/ownership/profile/acquisition packages with the race detector
on physical Linux, enabling CGO only for that race command. These are two target
jobs against one frozen pass-3 source revision. Actual CLI bootstrap/promotion,
killed CLI integration, complete native exit and the joint commit remain later
work within the approved allowance; core results cannot substitute for them.

### 2026-11-26 — R3, pass 1: owner selected profile flag A; execution started

The owner selected A for the pending CLI clarification. `--manifest` on
`inspect`, `check` and `generate` selects the complete verified generation
profile, default `schema/generation-profile.json`. Other commands keep their
command-specific inputs. Record the selection in the MADR's state table; preserve
all independent pins and ownership rules. R1–R5 execution and the coherent
R2/R3 commit boundary remain authorized; no push or tag is authorized.

Start with a standard-library rooted publication/recovery engine and native
scratch fixtures for containment, ownership, staging, durable intent, per-file
replacement, manifest-last ordering, unchanged mtimes, lock conflicts, manual
edits and actual killed subprocesses. Wire the CLI to that engine and the same
qualified candidate before promotion. Initial verification is partial R3
evidence, not permission to publish or claim RA12–RA14 complete. The ordinary
third-build-pass rule applies; no third R3 build is preauthorized.

### 2026-11-26 — R3, pass 1 result and pass 2 preparation

The first Linux core run failed against source identity
`f03b347ebf9171fa6677e6f686b0223cb2ce0685dcc374fc0b5095523b00e631`.
Preparation, formatting, vet, R2 candidate regression and all four publication
mutation controls passed. Publication, bootstrap, containment, filesystem-fault
and manual-edit/corrupt-recovery cases passed. All six killed-child cases failed
before their checkpoint: the fixture required `os.TempDir`, while Go's
`testing.makeTempDir` uses `GOTMPDIR` when configured. The runner has distinct Go
and ordinary temporary roots. This was a new fixture failure, not a product
publication failure. Lint also reported two QF1001 predicates and ST1012 on the
injected-error variable.

For pass 2, use the effective Go temporary root while retaining canonical-path
containment, a random fixture capability and independently fixed fixture bytes.
Add explicit subprocess refusal cases for missing capability and changed fixture
bytes; prove the capability assertion catches a scratch bypass. Simplify the two
predicates and use `errInjected`. Exercise the selected profile flag, including
path escape, aliased selection and independent input pins; prove profile-alias
and ignored-flag bypasses fail in scratch copies with the repository layout.

Run `scripts/qualify_r3_core.py` through the fleet queue against a frozen tree.
This remains partial R3 evidence: the CLI still requires publication/recovery
integration and the full native exit before live promotion or the joint R2/R3
commit. No third R3 build pass is authorized.

### 2026-11-26 — R3, pass 2: native results; third-build approval pending

The three completed runs used frozen source identity
`9b9dff5fde4a550b7b88bb64dffc2afbd6bb7c646bea7077cd27f81b821e88f1`.

| Environment | Execution | Queue wait | Result |
| :--- | :--- | :--- | :--- |
| Windows/amd64 | 64 seconds | 92 seconds | Failed raced-directory fixture postconditions and unused wrapper lint. |
| WSL Linux/amd64 | 39 seconds | 158 seconds | All 278 tests / 12 packages passed; failed unused wrapper lint. |
| Darwin/arm64 | 42 seconds | 343 seconds | All 278 tests / 12 packages passed; failed unused wrapper lint. |

Each run passed preparation, formatting, vet, source preservation and all seven
scratch mutation controls: selected-profile alias, ignored profile flag, child
fixture capability, portable output names, manifest-last order, manual-edit
recovery conflict and exclusive lock. The corrected killed-process cases no
longer failed. All processes exited; the duplicate automatically selected WSL
job was canceled before execution and its staging removed. The explicit WSL run
completed and stopped the distro. There is no physical-Linux pass-2 result.

New findings, neither pre-existing at the pushed baseline:

* `cmd/mcpgen/check.go:145`: `func computeSDKMapping is unused (unused)`.
  The named-profile implementation left a default wrapper with no callers.
  Remove that dead wrapper; retain the named-profile implementation and tests.
* `cmd/mcpgen/internal/publish/engine_test.go:277` and `:281`:
  `FAIL raced directory sentinel touched` and `FAIL raced destination mutated`.
  The fixture attempts to rename the open destination at lines 245–246, returns
  any rename error, then assumes the swap succeeded in both postconditions.
  It does not retain the hook error, so this output alone does not establish
  whether production containment failed. Go 1.27.4's local
  `os/root_test.go`, `TestRootRenameAfterOpen` and `TestRootRaceRenameDir`,
  explicitly account for Windows refusing an open-directory rename. Capture
  the exact native error and assert unchanged original bytes when the OS denies
  the swap. Also exercise an actual swap before opening the destination handle
  on every target, with independent sentinel/owned-byte assertions and a scratch
  identity-guard bypass. Retain the late-swap safety case; do not skip it, loosen
  the preservation assertions or change publication policy.

Automatic approval review rejected an additional physical-Linux race command
because it counted as another build beyond the authorized second pass. It did
not run. No alternate route was used. Normal qualification remains CGO-disabled;
only an approved race command may enable CGO. `git diff --check` passed. Live
`protocol/` publication and the joint R2/R3 commit remain pending.

The global agent rule says, “A third build pass on one phase is a deviation:
stop and prompt.” Pending owner choices, with all existing R3 cases retained:

| Option | Work and cost |
| :--- | :--- |
| A — Authorize R3 passes 3–6 **(recommended)** | Up to four further source revisions: resolve these findings, integrate CLI/bootstrap/recovery/diff/update, run the native matrix and physical-Linux race check, then safely promote and commit only after the complete exit passes. Stop sooner on a new decision or exhausted allowance. |
| B — Authorize R3 passes 3–10 | Same scope and gates; up to eight further revisions and a larger verification budget before another allowance decision. |
| C — Stop with the working tree preserved | No further builds or implementation; R3, safe candidate publication and the coherent R2/R3 commit remain incomplete. |

No option authorizes push or tags. This entry records evidence and proposed next
verification allowances; it does not approve a third build.

### 2026-11-26 — R2, pass 24: complete candidate exit qualified in four environments

All four queued native runs passed against one frozen source identity:
`b1134516af45ea5d2f904e087731189d4778f4c4c8dca62ef15f84ac2c9ad24e`.

| Environment | Execution | Queue wait | Result |
| :--- | ---: | ---: | :--- |
| Darwin/arm64 | 89 seconds | 0 seconds | Passed |
| Windows/amd64 | 103 seconds | 0 seconds | Passed |
| Primary Linux/amd64 | 175 seconds | 0 seconds | Passed, including SDK/profile race tests |
| WSL Linux/amd64 | 65 seconds | 88 seconds | Passed; original stopped state restored |

Each run verified Go 1.27.4, pinned preparation, 22 gate-runner regressions,
fragmented/invalid evidence cases, formatting, standalone vet, 216 generator
tests/subtests across 10 packages, zero-issue lint and every retained negative
control. The actual product candidate compiled both modules' packages and test
binaries for all three targets, then passed its native runtime pre-add and
owned decoder/public API/middleware boundary plants. Foreign binaries were not
executed. Ordinary operations used `CGO_ENABLED=0`; only the additional primary
Linux `sdkmap`/`profile` race tests used `CGO_ENABLED=1`. Both race packages passed.

The eight emitted files now include versioned API and coverage artifacts. They
cover all 155 definition dispositions, public generated declarations and alias
compiler surfaces on all three targets, all 840 schema locations and keyword
sets, the 21 reviewed method rules and capability pointers. The coverage oracle
is independently derived from the pinned source. It explicitly leaves full
codec/runtime qualification to R4/R5. Same-count location substitution failed
with `FAIL coverage source location or keyword inventory`; declaration omission
failed with `missing API binding JSONValue`. Missing pointer methods, changed
retained inventories and wrong proposed hash maps were also caught. These were
deliberately broken scratch copies, never live source changes.

The repaired collector successfully reassembled Go test output fragments and
retained one inventory from the already passing candidate. Every environment
independently reproduced the exact contents of
`tools/r2-candidate-migration.json`, including all four historical and nine
proposed file identities and both complete target matrices. The actual product
test also matched this retained review record. The proposed hashed manifest is
`57de122dd1234b6b99ff922c126413dd32c92800f3b8c610962616e5648531c6`;
the full file inventory remains in that record rather than being abbreviated
in prose. No additional compiler audit or repeated product build was introduced
solely to capture evidence.

RA05–RA11 and the complete proposed-candidate R2 exit are now qualified. This
does not declare publication, Stage 3 runtime integration, R3–R5 or Stage 7
complete. Preserve the old generation until R3 supplies the rooted publisher,
durable original/candidate staging, locked recovery and verified bootstrap.
The coordinated coherent-tree commit remains after R3 qualification; the owner
already published the earlier partial baseline as `5c00e36`. No agent stage,
commit, push or tag was performed. Published history was not rewritten.

All runner sessions exited, remote success staging was removed and source
preservation passed. A one-use comparison read all four complete logs and
verified identical review records. A scan of all 29 modified/untracked files
found no local account, host alias or real-machine path. No polling loop remains.
The initial Linux launch was rejected for an oversized inline command before
execution; a one-use scratch launcher then ran the same approved command.
Passes 21–24 are consumed; no pass 25 is authorized.

### 2026-11-26 — R3 preimplementation CLI clarification for owner review

The original Stage 4 contract requires common `--manifest` arguments but does
not identify their input type. `commands.go` still has no such flag, and
`profile.go` now selects `schema/generation-profile.json` internally. There are
two distinct input records: the generation profile binds every reviewed input;
the schema provenance manifest binds the schema source. The output ownership
manifest remains a third, separate record and cannot authorize input changes.
Resolve the public flag meaning before implementing that R3 CLI contract:

| Option | Public meaning | Cost and consequence |
| :--- | :--- | :--- |
| A — Generation profile **(recommended)** | `inspect`, `check` and `generate` use `--manifest` for the complete verified input profile; default `schema/generation-profile.json`. Other commands retain their command-specific inputs. | Smallest interface change; the selected profile and every member retain independent pin/containment validation. |
| B — Schema provenance | `--manifest` selects the revision's schema provenance record; the generation profile remains fixed and must cross-check that selected record. | More path and consistency wiring; preserves the earlier schema-focused reading of the flag. |
| C — Explicit names | Replace the ambiguous flag with `--profile` and `--source-manifest` where applicable. | Clearest distinction, with a deliberate amendment to the original public flag contract and corresponding help/tests. |

All choices preserve offline ordinary commands, strict formats/exit codes,
verified ownership and the existing output layout. No choice weakens bootstrap
or permits the hashless inventory to authorize replacement. The owner has been
prompted with these three options and subsequently selected A. The dated R3
execution entry and MADR clarification supersede this pending question; retain
the alternatives as review history. Implement the selected profile meaning
before claiming the R3 CLI acceptance cases complete.

### 2026-11-26 — R2, pass 23: all product/control checks passed; evidence fragments need joining

The Darwin-native run finished after 91 seconds using identity
`c743286e46c87c978a88c7fc238e62327365f20846bfff268dbf4a206eecbf43`.
Preparation, 22 gate-runner regressions, the evidence regression, formatting,
vet, 216 generator tests/subtests across 10 packages, zero-issue lint and every
loader/reference/alias/profile/emitter/candidate control passed. The independent
840-location fixture matched every model-derived pointer and keyword set.
The real candidate compiled both modules for all targets and passed the native
runtime pre-add and decoder/public API/middleware plants. The API/coverage,
compiler pointer-method, migration-map and retained-record omission plants were
caught. Source preservation passed.

The aggregate driver failed only its evidence extraction leaf:
`Expecting ',' delimiter: line 1 column 978 (char 977)`. Go test JSON splits a
long output line across events. Reassemble output fragments for the exact test
before parsing its logical inventory line, and require that test's passing
terminal result. Add a deliberately fragmented positive fixture and retain
missing/duplicate/malformed/unfinished evidence rejection.

Recover the inventory from this run's retained passing Go test log only after
confirming that every other driver leaf and source preservation passed. The
explicit extraction-recovery command rejects a failed product/control leaf.
It produced `tools/r2-candidate-migration.json`: four independently verified
historical identities and nine proposed identities including the hashed
manifest, both modules' complete compile-only matrix, and immutable source/
profile pins. This does not relabel the aggregate pass-23 driver as successful.
No new generation or build was needed to recover the already tested evidence.

Use approved pass 24 to qualify the corrected collector and the retained record
natively in all four environments. Add a same-count coverage-location
substitution plant to demonstrate the independent pointer oracle catches more
than a count change. Keep the source snapshot frozen until every runner exits.
R3 publication remains pending; no pass 25 or agent push/tag is authorized.

### 2026-11-26 — R2, pass 22: candidate passed; incorrect coverage test count rejected

The Windows-native run finished after 104 seconds, using identity
`99c15c71786140ea86d9023df7107c0bea1a38cb77b90f8d9777c9be5e5f6ab1`.
The corrected multiple-definition method binding passed. The actual candidate
compiled both modules' packages/test binaries on all three targets, and its
native runtime pre-add and all decoder/API/middleware plants passed. Formatting,
vet, zero-issue lint, Python regressions and every input/control command passed.
The generator's one failing test was `TestRepairA10Artifacts`:
`FAIL coverage node/definition inventory: aliases=14 definitions=155 nodes=840`.
Its newly written expectation of 841 locations was incorrect; no accepted
contract specified that count. The overall driver remained nonzero and did not
retain a successful migration record. The failed scratch tree was retained by
the runner's diagnostic policy; source preservation passed.

Before pass 23, derive an independent location/key inventory directly from the
immutable schema, with the source SHA-256 verified before maintenance output.
`scripts/schema_node_inventory.py` found exactly 840 locations including the
root. Retain that fixture in `internal/emit/testdata/schema-nodes.json` and
compare every pointer and keyword set, as well as the total, against coverage.
This replaces the erroneous estimate with a stronger exact source-location
oracle, preserving the full-coverage requirement. The maintenance derivation
does not run in ordinary gates and does not use the production model walker.
The coverage-omission plant remains mandatory against the corrected baseline.

The direct retained-record test and bypass plant also passed: changing the
reviewed hash fails; bypassing that comparison triggers `FAIL retained
inventory guard`. Use approved pass 23 for the independent coverage inventory
and complete evidence capture. Pass 24 remains for the retained-record/native
matrix. No pass 25, publication or agent push/tag is authorized.

### 2026-11-26 — R2, pass 21: API/coverage and migration artifact implementation

Extend the existing protocol ownership inventory with versioned `api.json` and
`coverage.json`. The API records public generated declarations, source pointers,
import qualifiers and alias representations/value/pointer method sets on all
three targets. SDK metadata comes from the existing compiler audit, with no
second export-data load. Coverage records all model nodes (including the
document root), keyword classifications, 155 definition bindings, reviewed
method rules and capability source pointers. Codec/runtime qualification is
explicitly pending R4/R5; generated binding coverage is not live conformance.

Capture exact previous/proposed hashes, including both manifests, from the
already compiled actual candidate. The canonical gate exposes its retained
test-log location; R2 extracts the fixture's versioned inventory and emits it
only after its checks and source preservation succeed. Retain the result as
`tools/r2-candidate-migration.json` after a successful frozen run, then require
the actual product fixture to match that reviewed record in subsequent runs.
This file is review evidence, not publisher authority. `check` also includes
the qualified migration inventory in its structured report.

New cases cover representative exact API signatures, the complete definition/
node/method inventories, shuffled method order, incomplete method/alias target
inputs, generic/promoted pointer method sets and malformed/absent/duplicate
evidence. Scratch plants omit an API declaration, a coverage node, a pointer
method and proposed migration hashes. Retain every earlier control. The
ordinary checked-in generation remains unchanged; R3 still owns publication.
The Linux-native run finished after 121 seconds, using identity
`6c65a04b47e1da87e2279064ebe161aa4fdea0222648cbe1267e3dec61a17707`.
Preparation, Python regressions, formatting, vet and zero-issue lint passed.
Generator tests and four emitter controls failed before candidate compilation:
`emit: missing or duplicate method coverage notifications/cancelled`. The new
coverage code collapsed multiple schema definitions for one wire method into
one pointer. Both `CancelledNotification` and `ClientNotification` bind this
method; the reviewed semantic row selects the former. The existing profile
correctly retains a set of permitted definitions per method. Repair coverage
to retain that same set, add a concrete cancellation binding assertion, and
retain every coverage/inventory assertion. This is an implementation bug in
the new artifact builder, not a changed protocol decision.

The new API omission, compiler pointer-method omission and migration-map
substitution plants were caught. The coverage omission and three older emitter
controls could not reach their intended failure because of the coverage bug;
they remain required. All other input/reference/alias/profile/candidate
controls and remote source preservation passed. Two aggregate driver leaves
failed; no successful migration evidence was retained or publication performed.
The runner retained its failed scratch snapshot under its diagnostic policy.
Use approved pass 22 for the correction and a direct retained-record mismatch
fixture/plant. Passes 23–24 remain available for final retention and the native
matrix. No additional allowance is inferred from this failure.

### 2026-11-26 — R2 deviation: passes 21–24 approved

The owner selected A after pass 20. Execute bounded R2 passes 21–24 to finish
the planned generated API/coverage artifacts and retained before/after migration
hash inventory, then qualify the complete candidate and omission plants. Reuse
the existing schema model, semantic input and three-target SDK compiler audit;
do not introduce a redundant scanner or imply R4/R5 runtime completion from
binding coverage. Preserve the published baseline `5c00e36`. R3 still owns
verified publication and recovery. No pass 25, agent push or tag is authorized.

### 2026-11-26 — R2, pass 20: four-environment matrix passed; artifact exit remains open

The full R2 input/control command completed successfully in all four requested
environments, through their shared heavy-run queues:

| Environment | Execution | Queue wait | Result |
| :--- | ---: | ---: | :--- |
| Windows/amd64 | 93 seconds | 0 seconds | Passed |
| WSL Linux/amd64 | 59 seconds | 92 seconds | Passed; originally stopped distro returned to stopped |
| Primary Linux/amd64 | 157 seconds | 320 seconds | Passed, including additional SDK/profile race tests |
| Darwin/arm64 | 84 seconds | 998 seconds | Passed |

Each completed run verified Go 1.27.4, pinned preparation, 22 Python regressions,
208 generator tests/subtests across 10 packages, formatting, standalone vet,
pinned lint with zero issues, and every retained loader/reference/model/alias/
profile/emitter/candidate negative control. `TestRepairA11ActualProduct` compiled
the actual proposed product's two modules, including internal and external test
binaries, for Darwin/arm64, Linux/amd64 and Windows/amd64. Its native runtime
pre-add gate and the owned decoder, public SDK accessor, middleware parameter/
result type and SDK complete-result projection plants passed in each environment.
The represented-field alias corpus and its field-dropping serializer plant
remain passing evidence; R4's broader differential obligation is unchanged.

All ordinary operations used `CGO_ENABLED=0`. Only the primary Linux run's
additional `sdkmap` and `profile` tests used `-race` and `CGO_ENABLED=1`; both
passed. No race test ran in WSL. Source preservation passed in every completed
run. Remote staging was removed for the completed Windows, WSL and primary
Linux runs. A redundant second Linux dispatch also selected WSL; it was
cancelled after starting, exited 130 after 39 seconds, stopped its process group
and returned the distro to its original stopped state. Its interrupted work
root was retained by the runner's interrupted-run policy; transport staging
was removed. It is not counted as another completed qualification.

The three remote snapshots used identity
`905546d25f99289a8137c74a71d6531a42644c496e59b80e21adada4bc724bea`.
While Darwin was queued, the owner committed and pushed all existing work as
`5c00e3680393e960f74a8907b1d6d8eaaf5f7336`; local `main` and `origin/main`
matched and the working tree was clean. No file contents changed. The deleted
`cmd/mcpgen/schema_util.go` ceased to be a tracked missing path, changing the
index-based identity to
`94a002376bffef21acda1b642fa636376379533be37447d5719d0e4fe1ddc34d`.
Reconstructing that one historical missing inventory entry reproduced the
remote identity exactly. Darwin tested the committed inventory and preserved
it. All pass-20 runner sessions have exited; no qualification loop remains.

**Remaining R2 exit work:** step 5 and the original Stage 3 require generated
API/coverage artifacts. `internal/emit/emit.go` currently emits six files:
`aliases.go`, `bindings.json`, `codecs.go`, `constants.go`, `owned.go` and
`schema.json`. The bindings enumerate definition/name/disposition/source
correspondence; the external API probe checks public SDK reachability. These
do not yet provide the planned reviewable API-signature and vocabulary/method/
capability coverage artifacts. Generate them from the already verified model,
compiler information and semantic rules, with explicit separation of binding
coverage from R4/R5 runtime qualification. Test determinism, completeness and
deliberate omissions without adding a second redundant audit or scan.

Step 7 also requires the exact old/proposed file inventories and hashes to be
retained for review. `qualify.Candidate.Previous` and `Proposed` hold those maps
during qualification, but the actual product fixture does not retain a
reviewable migration inventory. Capture that evidence with the final candidate,
including its manifest, without publishing or trusting the hashless list.

The owner-pushed commit contains bridge references to the proposed owned types
while `protocol/` still contains the historical generation, including SDK
aliases. This remains the explicitly coordinated R2/R3 integration work;
publication has not happened and an ordinary coherent-tree qualification is
not claimed. Preserve the published commit and complete remaining changes in
follow-up commits after their exits pass. Do not rewrite published history.
R3's rooted publisher, journal/recovery and verified migration remain unstarted;
the old publisher must not be used to install the candidate. R4/R5 and Stage 7
remain pending. The agent performed no commit, push or tag in this pass.

The approved passes 17–20 are consumed. Before another R2 build, present the
owner these options: A (recommended), authorize bounded passes 21–24 for the
remaining API/coverage and retained migration artifacts and their qualification;
B, authorize passes 21–28 for the same scope with a larger iteration allowance;
C, stop with the pushed work preserved and the R2 exit/R3 promotion pending.
A costs one further bounded implementation/verification cycle; B reserves up
to two such cycles and does not broaden scope. C leaves the committed
generated/runtime mismatch unresolved. No pass 21 or agent push/tag is
authorized by the successful matrix or the owner's completed push.

### 2026-11-26 — R2, pass 19: owned middleware and retained alias fixtures passed on Windows

The Windows-native full R2 command passed in 96 seconds with source identity
`1915c53f3818248f250eaf4a62b4463030b5c9817cedd17aaa429d86b82d6e54`.
Pinned preparation, 22 Python regressions, formatting, standalone vet, 208
generator tests/subtests across 10 packages, pinned lint and every earlier
input/reference/alias/profile/emitter/candidate control passed. Both real
modules' packages and test binaries compiled for all three targets. The actual
candidate's native runtime pre-add gate passed, including the formerly failing
live middleware connection/delegation test and external public API probe.

The private SDK complete-result projection preserves the original handler
result and matches finalization in the pinned SDK. Its omission plant failed
with `FAIL SDK complete-result projection`. The SDK-runtime accessor,
parameter-type, result-type and owned-decoder bypass plants were also caught.
All fourteen retained aliases passed independent represented-field examples
through SDK and native serialization. A compilable SDK serializer that drops
the fields failed with `FAIL SDK alias wire`. Source preservation passed; the
remote scratch tree was removed after success. No assertion was weakened.

Use approved pass 20 to qualify the same implementation across native Darwin,
Linux, Windows and WSL, keeping one source snapshot frozen. R4 still owns the
broader codec/validation differentials and R5 still owns complete live runtime
enforcement. The R2 contract is not yet promoted; R3 owns verified publication
and the coherent R2/R3 commit. No pass 21 or push/tag is authorized.

### 2026-11-26 — R2, pass 18: semantic/API guards passed; live discovery failed

The Linux-native full R2 command ran 160 seconds using source identity
`e76b0ec435eb1397d4ae9e23dbd28ec63310bba1c8171a06133f3cbe9bab0403`.
Both modules' package/test binaries compiled for all three targets. Pinned
preparation, 22 Python regressions, generator/runtime formatting, standalone
vet and pinned lint passed. The complete semantic rows, SDK-symbol availability
checks and their malformed-policy/context plants passed. The external public
API probe passed, and scratch SDK-runtime accessor, parameter-type bypass,
result-type bypass and owned-decoder bypass were each caught by the unchanged
fixtures. Earlier loader/reference/alias/profile/emitter/candidate controls and
source preservation passed.

Native runtime accounting rejected `TestOwnedMiddlewareLive` at
`middleware_bridge_test.go:47`: client connection failed with
`calling "initialize": duplicate "initialize" received`. Returned diagnostics
preserved that nested failure. No overall qualification or R2 exit is claimed.

Re-grounding the pinned SDK shows that its `Server.discover` stores the modern
session state and returns a result before `ServerSession.handleRequest` invokes
`setCompleteResultType`, after receiving middleware returns. The private
`completeResultWithType` initially omits `resultType`. The new owned receiving
bridge therefore attempted canonical validation before this SDK finalization;
the SDK client fell back to legacy initialization after the discovery failure,
which then met the already-initialized session. The main SDK source locators are
`mcp/server.go:926`, `mcp/server.go:2038` and `mcp/protocol.go:31–47` at v1.8.0.

Before approved pass 19, project that exact SDK-defined complete discriminator
onto copies of its seven complete-result types at the private in-process
middleware boundary. Do not mutate the SDK handler's original result or apply
this operation to received wire input. Keep multi-round-trip result handling
separate, preserve strict owned validation, and add a projection fixture plus
a compilable omission plant. This preserves the SDK's actual dispatch/wire
behavior rather than weakening schema requirements or disabling middleware.

Also add independently handwritten wire examples for every retained alias,
checking represented fields through both public SDK codecs and native codecs,
with a scratch field-dropping codec plant. R4 retains the full absent/null/zero/
unknown/numeric differential and strict-validator profile; these examples alone
cannot qualify that larger contract. Passes 19–20 remain approved; pass 21 does
not. Generated files remain unpublished and R2 remains uncommitted.

### 2026-11-26 — R2, pass 17: actual decoder bypass and canonical candidate gate passed

The Linux-native full R2 command passed after 170 seconds with source identity
`0b426ddc5ecb3a87c6784cd49662b072fc67b85ad46d186887e586f946e806b6`.
Pinned preparation, formatting, standalone vet, 193 generator tests/subtests
across 10 packages, pinned lint and the prior loader/reference/alias/profile/
emitter/candidate plants passed. All 22 Python regressions passed. A diagnostic
omission control deliberately returned no failed-test details and the unchanged
regression failed with `fixture owned validation failed`; the driver caught it.

The actual candidate compiles both modules' packages and test binaries for all
three targets, then reuses the canonical runtime pre-add gate for native
formatting, inventory-accounted fresh tests, standalone vet and pinned lint.
That gate passed. The replacement scratch mutation actually discards the owned
decoder's validation error for nonempty input. `TestOwnedResultBoundary` failed
with `FAIL invalid owned result accepted or unclassified`, and the fixture
required that failure. No live source was broken or generated output published.
Source preservation passed. Native Windows/Darwin/WSL requalification of this
new runner path remains required; the R2 exit is not claimed.

### 2026-11-26 — R2 semantic and public middleware audit before pass 18

Completed all 21 reviewed method rows with normative references, exact SDK
parameter symbols, capability paths bound to the pinned schema, completion
mode, cancellation scope, caller-controlled retry policy, conditions and R5
fixture ownership. Research checked the revision-specific primary sections for
[tools](https://modelcontextprotocol.io/specification/2026-07-28/server/tools),
[prompts](https://modelcontextprotocol.io/specification/2026-07-28/server/prompts),
[resources](https://modelcontextprotocol.io/specification/2026-07-28/server/resources),
[completion](https://modelcontextprotocol.io/specification/2026-07-28/server/utilities/completion),
[subscriptions](https://modelcontextprotocol.io/specification/2026-07-28/basic/patterns/subscriptions),
[cancellation](https://modelcontextprotocol.io/specification/2026-07-28/basic/patterns/cancellation),
[progress](https://modelcontextprotocol.io/specification/2026-07-28/basic/patterns/progress),
[logging](https://modelcontextprotocol.io/specification/2026-07-28/server/utilities/logging),
[elicitation](https://modelcontextprotocol.io/specification/2026-07-28/client/elicitation),
[sampling](https://modelcontextprotocol.io/specification/2026-07-28/client/sampling),
[roots](https://modelcontextprotocol.io/specification/2026-07-28/client/roots) and
[versioning](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning).
The pinned local schema and SDK declarations remain immutable mechanical
inputs; normal gates do not fetch these pages. The semantic member is now
29,106 bytes, SHA-256
`d30bc3e2bc9991e33086fb90a8e0059db219f6e54564660091b44e0ff41f5a82`.
After reviewing that member change, updated its profile member identity and
independent generator-source profile pin to
`e57dcb9eb38e6d79b7f3e4daa937fe085bdbaa89697fb8600a3411d916086ee4`.
No SDK/schema version or other profile member changed. Added structural policy
cases and SDK-symbol availability verification using the existing three-target
compiler export audit. This is contract evidence, not R5 behavior qualification.

The public middleware signature still used SDK request/result/session types.
Replaced it with owned `Request` and `MethodHandler` views. Canonical parameters
and results have their real owned DTO types; auxiliary/extension payloads use
the unconstrained owned JSON value. The private bridge clones parameters,
retains the original SDK connection, snapshots headers and rejects wrong method,
parameter or result types. Sealed SDK construction stays private. Added an
external SDK-free consumer registry reconciled with selected source declarations
and a transitive public-signature walk; its SDK data allowance is the already
compiler-audited alias closure. Planned scratch plants add a raw SDK runtime
accessor and bypass the parameter/result type guards. Live and direct fixtures
cover delegation, snapshots and short-circuiting. These changes do not claim
R5's single-install middleware chain, final ordering, capability enforcement or
transport-wire qualification. Remaining R2 alias wire evidence and R3 promotion
are still explicit outstanding work.

### 2026-11-26 — R2 deviation: passes 17–20 approved

The owner selected A after reviewing pass 16. Execute bounded R2 passes 17–20:
repair the ineffective owned-decoder experiment without weakening its rejection
fixture, reuse the canonical runtime pre-add gate for candidate verification,
retain failed-test diagnostics in the returned stream across WSL shutdown, and
continue the existing R2 API/semantic/alias audit. This is the same accepted
scope and coordinated R2/R3 commit boundary. No pass 21, publication, push or
tag is authorized by this allowance. Keep each qualification snapshot frozen
across its native matrix and record failures as well as passing controls.

### 2026-11-26 — R2, pass 16: actual product works; new mutation control is ineffective

Ran the complete R2 input/control command against one frozen source identity:
`ae29fba08ca6d1219a83488b8ed0e7a31ada807996d0a602c1b87867b921e334`.
Windows SSH recovered before this pass; verified its Go 1.27.4 toolchain.
Windows and WSL ran serially on their shared queue. Darwin waited 190 seconds
behind unrelated work; the entire source stayed frozen until it finished.

| Check | Windows/amd64 | Linux/amd64 | WSL Linux/amd64 | Darwin/arm64 |
| :--- | :--- | :--- | :--- | :--- |
| Pinned preparation; 21 Python regressions | passed | passed | passed | passed |
| Generator formatting, vet and pinned lint | passed | passed | passed | passed |
| Earlier loader/reference/alias/profile/emitter/candidate controls | passed | passed | passed | passed |
| Actual product package/test-binary compilation, both modules, all three targets | passed | passed | detailed evidence unavailable | passed |
| Actual product native runtime tests, vet and pinned lint | passed | passed | detailed evidence unavailable | passed |
| New owned-decoder mutation control | failed | failed | actual-product test failed; detail unavailable | failed |
| Source preservation | passed | passed | passed | passed |
| Execution seconds, excluding transfer/queue | 92 | 134 | 56 | 76 |

The actual product test reports `FAIL owned decoder bypass was not detected:
exit 0` at `candidate_runtime_test.go:94`. Its scratch mutation at lines 85–89
replaces native decoding with `encoding/json.Unmarshal`. Inspection of the
pinned Go 1.27.4 documentation and selected standard-library implementation
confirms that this decoder also calls `UnmarshalJSONFrom`. It therefore did not
disable owned validation. The test correctly refuses to count an ineffective
mutation as detection; this is an unqualified experiment, not evidence that
the production decoder accepted malformed results. Native fixtures including
`TestOwnedResultBoundary`, SDK custom-codec round trips and connection isolation
passed on the three physical platforms. No assertion was loosened.

The unit gate now prints its retained complete-log location on failure. Reading
those logs recovered the detailed results on all three physical platforms. WSL's
temporary log was unavailable after the runner restored the distro's original
stopped state; an artifact-only queued read confirmed the file was gone. No
build or test was rerun for that read. Preserve this evidence gap: the WSL
top-level actual-product failure does not establish which nested checks passed.
Future diagnostics must retain the failed-test details in the returned stream
or in an artifact that survives stopping the distro.

The runtime module's complete Go file inventory also passed a read-only gofmt
check. Ordinary builds and tests kept CGO off; no race run or publication took
place. R2 remains uncommitted because its exit and coordinated R2/R3 migration
are incomplete. The live generated files remain the historical baseline; the
handwritten changes and corrected generated contract are still a coordinated
candidate, not a working-tree qualification claim.

Required next work, pending another pass allowance: replace the ineffective
mutation with a compilable scratch implementation that actually discards owned
validation failure and require the unchanged rejection fixture to catch it;
route native candidate verification through the canonical runtime pre-add gate
so formatting, test inventory accounting, vet and lint each have one owner;
retain failed-test diagnostics across WSL shutdown. Then finish the public
middleware/SDK reachability audit, complete normative semantic rows and remaining
alias wire evidence before claiming the R2 exit. R3 still owns publication.

The approved passes 13–16 are exhausted. Pending owner choice: A, recommended,
approve bounded passes 17–20 for these repairs and continued R2 qualification;
B, approve passes 17–24 for the same scope with a larger iteration allowance;
C, review the existing diff before further execution. Doing nothing leaves the
new mutation experiment and the R2 exit unqualified. No pass 17, push or tag is
authorized by this entry.

### 2026-11-26 — R2, pass 15: package builds passed; two stale test imports rejected

The Linux-native product run used source identity
`d433c2af6586fc0b6953e2d38b38ac25133c809eb65fc2b2df2321958c0fe4a9`.
It exited 1 after 29 seconds. Both modules' package builds passed on
Darwin/arm64, Linux/amd64 and Windows/amd64. Generator test binaries passed;
runtime test binaries failed on all three targets. The corrected diagnostics
identified unused SDK imports at `isolation_test.go:9:2` and
`roundtrip_test.go:8:2`, left after migrating their callbacks. Removed those
imports and normalized the affected import groups. This is a test-source
repair; no assertion or runtime capability was removed. Source preservation
passed. Native runtime tests were not reached by this failed compile fixture.

Use approved pass 16 for complete generator/input/control qualification and
actual-candidate native runtime checks on available Linux and Darwin runners.
Windows/WSL native execution still requires restoring SSH connectivity to their
shared machine. The public middleware reachability audit, complete semantic
rows, remaining alias serialization evidence and R2/R3 integration remain
pending; a pass here cannot alone establish the R2 exit. No pass 17 is approved.

### 2026-11-26 — R2, pass 14: candidate compilation failed; diagnostic fixed

The focused Linux-native product run used source identity
`81fceda1b8068fa8df2b2d26d9c56f6067ed6aaf90553b3d2de07fe3ccb735e4`.
It exited 1 after 29 seconds: actual-product compilation failed before native
runtime checks. The fixture printed report pointer addresses instead of the
per-module/per-target diagnostics. Corrected the fixture to print each failed
target's package/test outcomes and retained compiler detail. No compiler cause
is claimed from the pointer-only message. Source preservation passed.

The owned tool/prompt/resource callback signatures and request/session snapshots
are in this candidate. Existing callback fixtures were migrated without
removing their behavior assertions. The resource fixture explicitly constructs
its typed contents, complete discriminator and cache hints. Use approved pass
15 to expose and resolve the actual compilation blocker; pass 16 remains the
last currently approved pass. No phase exit or publication occurred.

### 2026-11-26 — R2, pass 13: complete runtime fixture added; product test failed

Added private bidirectional SDK wire conversions and replaced the four direct
client result returns. Updated the existing round-trip checks to use the owned
content unions and presence fields. Added a complete actual-product fixture:
compile both modules and test binaries for all three targets, execute runtime
tests natively, run vet and the pinned linter, then deliberately bypass the
owned decoder in scratch. This fixture never publishes the candidate.

The Linux-native qualification ran 134 seconds with source identity
`958ae2c93316b9671d29f13a1fd0400dd5d2c278bedac41d964501a8f2cd8731`.
Pinned preparation, 21 Python regressions, generator vet/lint, existing loader,
reference, alias, profile, emitter and candidate plants passed. The newly added
`TestRepairA11ActualProduct` failed. The existing unit gate retained its terminal
failure and the stream tail, which did not contain the nested failure detail;
no runtime success or three-target compilation success is inferred from that
incomplete diagnostic. The source-preservation check passed. Windows/WSL
native execution was unavailable because the shared machine's SSH probe timed
out. Source is uncommitted and unpublished.

Continue with the already approved pass 14: owned callback/context adaptations
and the focused product driver, which retains the complete test diagnostic
stream. Tool, prompt and resource callbacks use owned parameter snapshots and
owned session handles while preserving SDK typed inference/default fulfillment.
The remaining public middleware reachability audit and R2 exit work are still
pending. Passes 15–16 remain available for corrections and qualification.

### 2026-11-26 — R2 deviation: passes 13–16 approved

The owner said proceed after the three continuation options. Execute recommended
option A: bounded R2 passes 13–16 for the SDK/owned boundary, corresponding
runtime/API tests and remaining R2 qualification. Preserve actual SDK dispatch
and serialization behavior, use private conversions and owned request handles,
and retain every earlier control. The R2 scope and coherent R2/R3 commit
boundary remain unchanged. This resolves the pending allowance after pass 12;
it does not authorize pass 17 or claim a phase exit.

### 2026-11-25 — R2, pass 12: actual product candidate rejected on all targets

Ran the actual `mcpgen check --repo ../.. --format json` command through the
Linux-native shared queue. The environment explicitly set CGO off, workspace
and Go environment configuration off, the local toolchain, readonly modules
and offline dependency resolution. The input snapshot was unchanged from
pass 11. The command ran 35 seconds and exited 1 with status `unqualified`.

| Actual module | Darwin/arm64 package/tests | Linux/amd64 package/tests | Windows/amd64 package/tests |
| :--- | :--- | :--- | :--- |
| Generator | passed/passed | passed/passed | passed/passed |
| Runtime | failed/failed | failed/failed | failed/failed |

The runtime rejects direct SDK result returns against the corrected owned
protocol definitions. Retained diagnostics include `client.go:140:9` for
`*github.com/modelcontextprotocol/go-sdk/mcp.ReadResourceResult` returned as
`*protocol.ReadResourceResult`, and `client.go:193:9` for the analogous
`CallToolResult` mismatch. Inspection also identifies direct returns at
`client.go:113` and `client.go:125` for list results. The committed baseline
compiled these through old aliases; the proposed owned contract requires real
private conversions and the corresponding public/API test adaptations.

This is the first complete proposed product candidate attempted by the repaired
check path. Both real modules and their test binaries were submitted for
compilation; no foreign binary executed. Historical output removal occurred in
scratch only after independent byte verification. The post-compilation source fingerprint passed;
no publication occurred. A successful isolated generated-package fixture is
not substituted for this failed product result.

Remaining R2 work includes `internal/sdkbridge/` and root result/request
boundaries, the public SDK-runtime reachability audit, complete normative
semantic rows, remaining alias wire evidence and coherent R2/R3 integration.
The existing publication implementation remains unqualified and must not
install the candidate. R3 retains the verified publication/migration work.

The approved passes 9–12 are exhausted. Pending owner choice: A, recommended,
approve bounded R2 passes 13–16 for the proper SDK/owned boundary and remaining
exit work; B, approve passes 13–20 for the same scope with a larger iteration
allowance; C, review the current diff before more execution. Doing nothing
leaves the product candidate blocked. No R2 commit or Stage 7 advancement is
claimed; no push or tag is authorized.

### 2026-11-25 — R2, pass 11: expanded candidate controls passed on four environments

Corrected camel-case Go initialisms and package/method comments. Emission tests
now consume the reviewed profile, including intended Go names and ownership.
Generated-source analyzers include the validator's real test context. The
historical-identity bypass remains compilable, and the generated deprecation
control uses the exported deprecated constructor already proven by R1.

Ran the R2 input/candidate-control qualification command on the same snapshot
across all four environments, identity:
`61aa9ee85d0c423c9a42eb4990030266a9af24cfd6d922719bf1145635eb97a7`.

| Check | Windows/amd64 | Linux/amd64 | WSL Linux/amd64 | Darwin/arm64 |
| :--- | :--- | :--- | :--- | :--- |
| Pinned preparation; formatting; generator vet/lint | passed | passed | passed | passed |
| Python runner regressions | 21 passed | 21 passed | 21 passed | 21 passed |
| Generator tests/subtests; packages accounted | 192; 10 | 192; 10 | 192; 10 | 192; 10 |
| Generated-package fixtures; vet/lint | passed | passed | passed | passed |
| Prior input/reference/alias/profile controls | passed | passed | passed | passed |
| Ten emitter mutation controls | caught | caught | caught | caught |
| Six candidate mutation controls | caught | caught | caught | caught |
| Source preservation | passed | passed | passed | passed |
| Execution seconds, excluding queue/transfer | 88 | 105 | 70 | 71 |

The macOS run waited 654 seconds in the shared host queue; source remained
frozen throughout. Windows and WSL ran serially on their shared physical host.
All ordinary compilation and test commands kept CGO off. No race run was
requested for this batch.

The new controls observed real Windows-only compilation failure while the
other target/module controls compiled, rejected incomplete/duplicate target
reports, caught forged or tampered ownership and intervening edits, and proved
timeout cancellation by deliberately leaving a child alive in a scratch
implementation. Generated alias identity failure and generated SA1019 were
observed, with no linter suppression. These results qualify the generator and
candidate-control foundation; the actual product candidate has the separate
pass-12 result above. The phase exit remains incomplete.

### 2026-11-25 — R2, pass 10: candidate controls passed; generated-source lint blocked

Implemented complete scratch assembly with rooted source access, byte/link and
inventory fingerprints, independently verified historical output hashes and
case-insensitive unowned-file collision rejection. Historical fixture bytes are
frozen from R1 commit `28a9284`; the hashless manifest is never accepted solely
for its filenames. Existing handwritten source, tests, test data and both Go
modules survive assembly. First-generation `check` now takes the same complete
candidate path; incomplete compilation reports cannot authorize publication.

The Go qualifier now bounds compiler commands and output, retains bounded
diagnostics, sanitizes target environment overrides and cancels child processes.
Generated aliases carry two-way identity assignments. Generated-package vet and
lint run on their own source context; the canonical Python tool selector passes
the already selected pinned linter to child tests. A test rejects attempts to
use that transport to override CGO policy.

Ran the R2 qualification command through the Linux-native queue using identity
`4b8885ed8c25e701ce7228ada71657fd81a5797312171dc01f791c1814492f00`.
The command exited 1 after 108 seconds. Pinned preparation, 21 Python runner
regressions, generator formatting and standalone vet passed. New snapshot,
ownership/containment, real Windows-only compilation plant, complete target
accounting and bounded-runner tests passed. Five candidate mutation controls
were caught, including a deliberately orphaned child. The historical-identity
mutation was inconclusive because its patch made a local variable unused; the
control is being corrected to retain a compilable bypass.

Generated-package lint found ST1000 on the codec header and ST1003 on field
names such as `TtlMs`, `RequestId`, subscription IDs and `ToolUseId`. Generator
lint also found ST1020 on the new `Compile` method comment. The isolated fixture
copied validator implementation but omitted its real tests, creating additional
unused diagnostics absent from the complete package context. Correct it by
including those test sources, retaining all analyzer checks. The generated
deprecation plant did not produce its required SA1019 diagnostic and remains
unqualified; reuse the exported deprecated-constructor control already proven
by R1. No assertion or linter exclusion is being removed.

Corrections prepared for pass 11: tokenize camel-case field words before
normalizing Go initialisms; separate the generated header from the package
comment; fix the method comment; use the complete validator test context and
the unchanged reviewed input profile in emission fixtures. Retain all pending
mutation experiments. Full product integration, normative rules and final alias
wire evidence remain separate R2 obligations. Passes 11–12 remain approved.

### 2026-11-25 — R2, pass 9: typed emission and wire controls passed on four environments

Inventoried all 840 schema locations in the pinned document: 277 references,
24 `anyOf` sites and three `allOf` sites. Implemented typed field projection for
the three object intersections while retaining the original schema pointers
and complete composed validation. Base required members and refined typed data
remain distinct. Shared reference DAGs expand once; recursive structural
projection and unqualified shape intersections fail explicitly. Corrected the
two ST1023 findings without changing linter configuration.

Ran `python3 -B scripts/qualify_r2_inputs.py --allow-network` through the shared
queues with the same source identity in all four environments:
`54fdb395db22c65aec4f8f71b64b831ba1951c468041acf3379c1c0a277d93c1`.

| Check | Linux/amd64 | Windows/amd64 | Darwin/arm64 | WSL Linux/amd64 |
| :--- | :--- | :--- | :--- | :--- |
| Pinned preparation, formatting and standalone vet | passed | passed | passed | passed |
| Generator tests/subtests; packages accounted | 171; 10 | 171; 10 | 171; 10 | 171; 10 |
| Generator lint | 0 issues | 0 issues | 0 issues | 0 issues |
| Prior loader/model/reference/alias/profile controls and plants | passed | passed | passed | passed |
| Seven emitter mutation controls | caught | caught | caught | caught |
| Source preservation | passed | passed | passed | passed |
| Execution seconds, excluding queue/transfer | 89 | 67 | 49 | 36 |

The generated package executed independent content-union, presence,
required-array, exact-ID, recursive-map, composed-error and registry fixtures.
The in-package validation bypass produced both `FAIL invalid wire accepted`
and `FAIL allOf branch assertion bypassed`. Separate scratch plants caught a
name collision, optional-empty omission, changed output hash, omitted binding,
shared mutable document bytes, dropped inherited required member and repeated
shared-DAG expansion, each with its intended diagnostic.

This qualifies the emission foundation and its standalone generated-package
fixtures. Full product-candidate assembly, handwritten/platform qualification,
complete alias serialization/signature evidence and normative semantic rules
remain R2 obligations. No R2 phase exit or product publication is claimed.
Passes 10–12 remain approved for those repairs and qualification.

### 2026-11-25 — R2 deviation: passes 9–12 approved

The owner selected A and approved bounded R2 passes 9–12. Audit the complete
nested emission surface before the next build; implement typed object
composition without dropping original assertions, correct the two ST1023
findings, and retain every pending wire/ownership/coverage plant. Qualify the
unchanged snapshot natively across the supported target environments once
the initial gate succeeds. The R2 scope, complete-candidate exit and coordinated
R2/R3 commit boundary remain unchanged. This resolves the pending allowance
after pass 8; it does not authorize pass 13 or claim completion.

### 2026-11-25 — R2, pass 8: nested composition support remains blocking

Ran `python3 -B scripts/qualify_r2_inputs.py --allow-network` through the
Windows-native shared queue, using frozen source identity
`c7591a9cb8d1b2ea3cc1266f796ffb6b043e080f302c0b8c4ec9152b4de6e3fc`.
The command waited 36 seconds, ran 65 seconds and exited 1. Pinned Go 1.27.4,
golangci-lint 2.15.0, dependencies, schema and conformance preparation passed.
Formatting and standalone vet passed. Source preservation passed.

The emitter stopped at `internal/emit/compiler.go:179` with the exact diagnostic:
`emit: #/$defs/HeaderMismatchError/properties/error: allOf requires an explicit reviewed binding`.
This is a previously incomplete generation contract, now rejected explicitly by
the replacement emitter rather than silently lowered to an untyped field.
Complete emission, total-binding checks and generated-wire execution remain
unqualified. No further native copies were dispatched after this shared
deterministic failure.

A read-only traversal of the complete pinned JSON document found three `allOf`
sites, all under `properties/error`: `HeaderMismatchError`,
`MissingRequiredClientCapabilityError`, and `UnsupportedProtocolVersionError`.
Each combines `$ref: #/$defs/Error` with an object refinement. The refinements
fix error codes and, for the latter two, require typed error data. A repair must
retain the base `message` requirement, refined fields, requiredness and all
original branch assertions. It must not replace these compositions with `any`,
discard constraints or relax the expected fixtures.

Lint also rejected new redundant annotations at `internal/emit/compiler.go:222`
and `:236`, both `ST1023: should omit type ast.Expr from declaration; it will be inferred from the right-hand side`.
These are new implementation findings; the linter policy remains unchanged.

Prior loader/model/reference/alias/profile controls and plants passed. The new
name-collision bypass and shared-document-storage plant were caught with their
intended diagnostics. The optional-empty, ownership-hash and omitted-binding
plants were inconclusive: the composition rejection occurred first. They remain
required and must fail for their intended reason after emission succeeds.
The generated-validation bypass inside the wire test has not executed yet.

The approved passes 7–8 are exhausted. Pending owner selection: recommended
bounded passes 9–12 for a complete nested-emission audit, typed composition
repair, lint corrections and qualification; alternatively passes 9–10 followed
by reassessment, or a diff review before additional execution. The scope and
phase exit remain unchanged. No R2 commit, product publication or Stage 7
completion is claimed. Doing nothing leaves R2 blocked at emission.

### 2026-11-25 — R2, pass 7: AST emission failed before generated-wire execution

Replaced string-based object-only emission with Go AST declarations, references,
primitive/array/union bindings, typed open maps, presence fields, native JSON
hooks, canonical method inventory and API/schema correspondence. Candidate
outputs now include four Go sources, the exact schema and `bindings.json`, with
version-2 SHA-256 ownership metadata. This remains scratch emission; the old
publisher has not installed it in the product package.

Ran the R2 input qualification command through the shared Windows-native queue
using source identity
`d3c6af15dc8203c5f433dd1986fb800ecaa38d0d05568220ab98f684a43e8d36`.
The command exited 1 after 27 seconds. Formatting and standalone vet passed;
emission failed at
`#/$defs/ElicitRequestFormParams/properties/requestedSchema/properties/$schema`
with `invalid field name "$schema"`. Therefore the new generated-wire fixture
did not execute. The name-collision bypass plant was caught; the optional-empty
plant was inconclusive because emission failed first. Prior loader/model,
reference, alias and profile controls/plants passed. Source preservation passed.

The fresh scratch workspace also lacked the prepared linter: its installed
2.14.0 did not match required 2.15.0, so lint did not run. This invocation omitted
the explicit `--allow-network` preparation used in pass 6. Pass 8 will use that
flag for pinned acquisition; ordinary validation remains offline.

Corrections prepared for pass 8: normalize wire-property punctuation into valid
Go identifiers while preserving exact JSON tags; place inline field declarations
under an explicit `Fields` namespace so distinct `Request.params` and
`RequestParams` schemas retain separate bindings; retain collision rejection.
Separate schema-valued `additionalProperties` from the property map, preventing
a literal property from overwriting synthetic model metadata. The historical
vocabulary plant gains only an inert IR-field compatibility shim, so its failure
still comes from malformed acceptance rather than a compilation mismatch.
Add scratch plants for output hashes, omitted bindings and document ownership.

Pending R2 obligations remain: full normative semantic rows, complete SDK wire
and signature evidence, handwritten/platform declaration checks, complete
product-candidate integration and three-target qualification. R3 owns verified
publication; R4 retains validator/codec differential and resource-limit work.
No R2 phase exit or commit is claimed. Pass 9 is not authorized.

### 2026-11-25 — R2 deviation: passes 7–8 approved

The owner selected A and approved R2 passes 7–8. Continue the remaining
semantic-rule and generated-contract work within the existing R2 scope.
Preserve the independently pinned inputs, prior failure plants, native target
matrix and coherent R2/R3 commit boundary. This resolves the pending allowance
after pass 6; it does not authorize pass 9 or declare a phase exit.

### 2026-11-25 — R2, pass 6: pinned input/profile guards passed in four environments

Implemented the reviewed version-2 SDK map for all 155 definitions, with an
explicit ownership reason and intended Go name for each. The policy has 14
structural alias candidates, 122 owned and 19 owned-codec entries. Implemented
21 schema-bound canonical method/direction rows and separate modern-only legacy
rejection metadata. The pinned schema represents the cancellation notification
under both `CancelledNotification` and `ClientNotification`; validation retains
both definition bindings while requiring one reviewed canonical method row.

Added `schema/generation-profile.json` and the independent generator-source pin
`3bc9d41c36e5c4877c47fe8ff997c9a509fa5b1e7de06780b781f88d226b0f20`.
The profile binds exact bytes, sizes and versions for seven required inputs.
Bounded `os.Root` acquisition rejects missing/unreviewed/escaping inputs before
emission; inspection and first-generation checks require the same profile.
Rendering reuses the verified snapshot rather than loading it again. Removed
the superseded unbounded schema reader and stale cached-disposition inspection.
Fresh SDK source/module identity and transitive alias safety must match before
reviewed protocol ownership is applied. Generated Go-name consumption and full
product-candidate assembly remain separate pending work.

Ran `python3 -u scripts/qualify_r2_inputs.py --allow-network --mapping-report`
through the shared queues in all four environments. Every run used source
identity:

`e6e36509c984d6e46f0662226be41bbcc68665843d621e359d08c031b4ae88df`

| Check | Windows/amd64 | Darwin/arm64 | Linux/amd64 | WSL Linux/amd64 |
| :--- | :--- | :--- | :--- | :--- |
| Pinned preparation; formatting; standalone vet | passed | passed | passed | passed |
| Generator tests/subtests; packages accounted | 149; 10 | 149; 10 | 149; 10 | 149; 10 |
| Generator lint | 0 issues | 0 issues | 0 issues | 0 issues |
| Prior loader/model/reference/alias controls and plants | passed | passed | passed | passed |
| Independent profile source-pin control and bypass plant | passed | passed | passed | passed |
| Complete mapping-policy control and bypass plant | passed | passed | passed | passed |
| SDK-context control and bypass plant | passed | passed | passed | passed |
| Fresh mapping report; nine malformed-report controls | passed | passed | passed | passed |
| Source preservation; final exit | passed; 0 | passed; 0 | passed; 0 | passed; 0 |
| Execution seconds, excluding queue/transfer | 63 | 37 | 72 | 32 |

Native Windows also executed the real symlink-escape fixture successfully;
there was no fixture skip or host-policy change. All four exercised absent
profile/SDK-map/semantic inputs, altered schema bytes, a forged self-consistent
profile, symlink escape, destination preservation, invalid/missing/colliding
names, unbound/duplicate/missing methods and unsafe/mismatched SDK mappings.
The new source-pin bypass failed with
`FAIL profile bypass reached SDK or output`; the mapping-policy bypass failed
with `FAIL invalid reviewed input policy accepted`; the SDK-context bypass
failed with `FAIL SDK context or unsafe mapping accepted`. Broken code and
inputs existed only in temporary copies. Ordinary operations remained offline
with CGO disabled; only explicit preparation could acquire pinned inputs.

The Darwin turn waited 907 seconds behind two existing heavy jobs before its
37-second run; the Windows/WSL turns serialized through their physical queue.
No process belonging to another task was changed. Verification completed and
all R2 runner sessions exited; no polling loop or queued R2 run was left behind.

**Remaining R2 work:** complete the normative semantic-rule fields and their
independent fixtures, beyond the now-pinned total method/direction inventory;
finish RA09's generated compiled identity/signature and wire probes; implement
RA10's AST emission, actual Go-name bindings, presence/union/open-object codecs,
registries and API/coverage artifacts; finish RA11's complete handwritten,
internal/external and platform candidate compilation. Do not infer those exits
from the passing input/profile checks. R3 still owns verified publication and
the baseline migration; R4/R5 remain pending. No generated product outputs were
installed, no R2 files were staged or committed, and no push/tag occurred.

The approved allowance for passes 5–6 is consumed. Obtain the owner's next R2
build allowance before pass 7, preserving every acceptance exit and the coherent
R2/R3 phase commit rule.

### 2026-11-25 — R2, pass 5: fresh SDK mapping evidence captured

Changed `inspect` to compute its audit from prepared module/compiler metadata;
it no longer trusts a cached disposition artifact or claims complete candidate
qualification. Removed the redundant source-AST export-name scan: compiler type
scopes own availability; selected source ASTs still own deprecation comments.
Added a capture helper with independent schema/checksum/envelope checks and
nine deliberately malformed report controls.

Ran `python3 -u scripts/qualify_r2_inputs.py --allow-network --mapping-report`
through the native Windows shared runner/queue. Source identity:
`059835dd704a10975947affac4aee24db096832044417882b3ab89960df6dcba`.
Pinned preparation, formatting, standalone vet, all 124 generator tests/subtests
across nine packages, lint (zero issues), previous plants and source preservation
passed. All nine mapping-report plants were rejected. Final exit 0; execution
54 seconds excluding transfer/queue time. The captured audit covers all 155
definitions: 14 structural alias candidates, 129 owned and 12 owned-codec.
The source/module SDK context is
`b6f8c43eed4ca771695ddf38ed62597a67e4033d21eb0a2f9848381f9bc386b9`.
This is ownership/audit evidence; compiled generated API and wire identity
probes remain required before accepting the aliases at the R2 exit.

Before approved pass 6, replace the representative SDK input with the complete
155-entry reviewed table, reasons and Go-name bindings. Preserve the existing
protocol-level codec policy independently of same-name SDK availability:
`CreateMessageRequest`, `CreateMessageRequestParams`, `ListRootsRequest`,
`InputRequests`, `InputResponses`, `InputResponseRequestParams` and
`SamplingMessageContentBlock` remain private-codec DTOs. This yields 14 alias
candidates, 122 owned and 19 owned-codec entries. The input records
`RequestId`'s intended Go name as `RequestID`; RA10 must consume those reviewed
names rather than infer inconsistent spellings.

Complete the semantic input's 21 schema-bound canonical method rows using the
independent direction inventory. Move the stale `initialize` entry to explicit
legacy rejection policy, consistent with MADR G12; do not add it to canonical
registries. Version the repaired SDK and semantic inputs as version 2.

Add a generation profile pinned independently in generator source, with exact
member hashes, sizes and versions for the schema, its provenance manifest,
SDK/semantic maps, tool pins and runtime module metadata. Require that profile
before emission, inspection and even the first-generation check path; reuse
the verified snapshot through rendering instead of rescanning it. Use bounded
reads through `os.Root` and reject SDK-context drift/unsafe aliases while
applying reviewed ownership. Remove the superseded unbounded schema reader.
Exercise absent inputs, changed bytes, a forged self-consistent manifest,
symlink escape, structural mapping/name/method errors and SDK context errors;
prove the source-pin, mapping-policy and SDK-context guards with scratch bypass
plants. None of this substitutes for RA10/RA11 or R3 safe publication.

### 2026-11-25 — R2 deviation: passes 5–6 approved

The owner explicitly approved R2 passes 5–6. Continue the complete reviewed
mapping/provenance inputs, generated API contract and scratch candidate work
under the existing R2 scope. Pass 5 first captures fresh compiler-backed SDK
mapping evidence for review; use that evidence to replace the representative
mapping input with the complete reviewed table. Preserve the acceptance exits,
the source/target failure plants and the coherent R2/R3 publication boundary.
This resolves the pending allowance after pass 4; it does not authorize pass 7,
declare R2 complete or authorize a push/tag.

### 2026-11-25 — R2, pass 4: input/audit checks passed in four environments

Completed approved inventory repair A and corrected the unsafe `TextContent`
alias expectation to require ownership with its audit reason. Safe `Tool` and
`Implementation` aliases still have explicit passing expectations. The source
comment parser recognizes deprecation paragraphs rather than incidental text,
joins deprecated embedded fields and generic receivers, and keeps package
identities distinct. The same production export-data loader now has a
handwritten platform fixture: missing target types, a Windows variable replacing
a type, a Windows runtime method leak and a Windows-only deprecation all have
independent assertions. An omitted-Windows-audit scratch plant fails those
assertions. Production availability intersects type declarations across the
three targets; a same-named variable is not an available alias target.

Ran `python3 -u scripts/qualify_r2_inputs.py --allow-network` through the shared
host queues on all four environments, with the same source identity:

`7cfa38eaa333a2aeb46e15693bcdf9b9e9442393828cde58a7f22f87807ea9e5`

| Check | Windows/amd64 | Darwin/arm64 | Linux/amd64 | WSL Linux/amd64 |
| :--- | :--- | :--- | :--- | :--- |
| Pinned preparation | passed | passed | passed | passed |
| Generator formatting and standalone vet | passed | passed | passed | passed |
| Generator tests/subtests; packages accounted | 124; 9 | 124; 9 | 124; 9 | 124; 9 |
| Generator lint | 0 issues | 0 issues | 0 issues | 0 issues |
| Loader/model valid and broken-baseline controls | passed | passed | passed | passed |
| Reference depth valid and bypass controls | passed | passed | passed | passed |
| Alias closure valid and bypass controls | passed | passed | passed | passed |
| Source inventory valid and false-deprecation controls | passed | passed | passed | passed |
| Target exports valid and omitted-Windows controls | passed | passed | passed | passed |
| Source preservation; final exit | passed; 0 | passed; 0 | passed; 0 | passed; 0 |
| Execution seconds, excluding queue wait | 64 | 46 | 61 | 31 |

All ordinary operations used CGO disabled and offline, readonly module settings;
network access was confined to explicit pinned preparation. The target fixture
loads compiler export data for all three targets without executing foreign
binaries. It does not substitute for compiling the complete proposed product
candidate. The Windows and WSL turns ran sequentially through their physical
host's shared queue; the Darwin turn waited for another heavy run to finish.

The deliberately broken copies failed on the intended diagnostics:
`FAIL strict loader accepted deliberately broken`,
`FAIL unqualified or malformed keyword accepted`,
`FAIL over-depth graph accepted`, `FAIL alias closure`,
`Meta has no source deprecation marker`, and the two
`FAIL target-dependent ... accepted` assertions. The source tree was never
broken to exercise them. Full SDK package testing runs once in the generator
pre-add gate; the alias helper runs only its separate control/plant pairs.

This completes the authorized A inventory correction and passes 3–4, not R2.
RA08's independent input identities and complete reviewed mapping, RA10's AST
emission/presence/union/open-object/API contract, and RA11's complete scratch
product candidate are still pending. RA09 still needs the generated compiled
identity/signature and wire probes at that candidate boundary; the passing
export audit is foundation evidence, not a substitute for them. R3 publication
and all R4/R5 acceptance work remain pending. No generated outputs were
installed, no R2 files were staged or committed, and no push/tag occurred.

Further compiler-backed mapping/candidate work requires another R2 build pass.
The approved allowance ends at pass 4; obtain the owner's next allowance before
pass 5. Preserve the coherent R2/R3 phase commit rule and all remaining exits.

### 2026-11-25 — R2, pass 3: inventory and plants passed; alias expectation failed

Applied approved repair A: `Meta` is no longer a frozen SDK deprecation and
has a separate explicit owned policy. The remaining frozen source markers are
checked on every supported target; missing exports and missing markers now have
distinct diagnostics. The existing sampling/roots owned codec policy remains.
Production reference validation now computes bounded acyclic graph depth while
retaining cycles, normalizes nested resource identities and rejects annotation
values as schema targets. Added independent broken-baseline loader/model
controls, a reference-validation bypass plant and a false-`Meta`-deprecation
plant. No generated output was published.

Ran `python3 -u scripts/qualify_r2_inputs.py --allow-network` through the shared
Linux runner/queue with source identity
`907aff2a4a93b86be869830fd29072c73a2dcb163a6835f3a9d1062a633f78c4`.
Pinned preparation passed, including Go, linter and dependency identities.
Formatting, standalone vet and generator lint passed with zero lint issues.
All loader/model/reference/alias controls and their failure plants passed;
source preservation passed. The overall pass exited 1 because
`TestKnownDispositions` still expected `TextContent` to alias:

```text
FAIL TextContent: got "owned" want "alias"
```

The new audit follows its private `fromWire(*wireContent)` method into an
SDK-owned wire type; the former name-only expectation did not check this
closure. Correct this expectation to require ownership plus its audit reason,
consistent with RA09, while preserving the safe `Tool`/`Implementation` alias
assertions. The source deprecation correction and codec ownership control both
passed against the real pinned SDK. This pass does not qualify all RA05–RA11.

Before approved pass 4, add a handwritten target-specific module fixture that
uses the same explicit export loader as production. Pin rejection of a type
missing on other targets, a declaration that becomes a variable on Windows,
a Windows-only runtime method leak and a Windows-only deprecation. Include an
omitted-Windows-audit plant. Keep full SDK package tests owned by the generator
pre-add gate; the alias plant helper runs only its independent controls, avoiding
a redundant second full SDK package test run.

### 2026-11-25 — R2 deviation: inventory repair A and passes 3–4 approved

The owner selected A after pass 2: correct the SDK deprecation inventory while
preserving the required owned surfaces and authorize R2 verification passes
3–4. Reconcile the complete frozen list against the pinned source comments;
record actual deprecations independently from ownership/codec policy. Preserve
the source-comment guard and the transitive forbidden-exposure assertions.
This authorizes the next two passes, not a phase exit or publication. Continue
the approved R2 acceptance scope and retain the coordinated R2/R3 commit rule.
The pending authorization in the preceding pass entry is resolved by this
dated entry. No additional dependency or SDK pin change was requested.

### 2026-11-25 — R2, pass 2: alias controls passed; pinned inventory failed

Implemented the approved standard-library export-data auditor using isolated
copies of the runtime module metadata, offline Go commands and explicit
compiler export-file lookup for Darwin/arm64, Linux/amd64 and Windows/amd64.
The candidate mapping now records its SDK source/module context and examines
transitive types, fields, pointer/value methods, instantiated arguments and
source deprecation comments. This is uncommitted implementation work, not an
RA09 exit claim; target-specific adversarial qualification, generated identity
probes and the full candidate contract remain unfinished.

Ran `python3 -u scripts/verify_alias_guards.py` through the shared Linux host
runner/queue. The safe/forbidden fixture control passed, including pointer and
promoted method leakage, generic arguments, sealed interfaces, recursive safe
types, deprecated fields/embedded members/generic methods and package-qualified
comment association. A deliberately broken scratch auditor that accepts every
alias failed with `FAIL alias closure`; the harness reported the plant caught.
No live source was broken to exercise that plant.

The same pass's pinned SDK package tests exited 1:

```text
FAIL build: sdkmap: frozen deprecation Meta no longer appears as an exported SDK type; the pin or the audit drifted
```

The diagnostic combines export availability and deprecation-marker checks.
The pinned SDK's `mcp/shared.go` declares `Meta` as an exported map type without
a `Deprecated:` comment. The existing `internal/sdkmap/sdkmap.go` frozen list
already classifies it as deprecated at baseline `28a9284`; the new comment
verification exposed that pre-existing factual discrepancy. Do not remove the
guard or silently change its assertion. Within the approved R2 audit, reconcile
actual source deprecations separately from explicit framework-owned policy,
preserve required owned codec surfaces, and check the entire frozen inventory
before updating reviewed mappings and expectations. An SDK source correction
is not an SDK version change or permission to expose forbidden signatures.

No generated destination was published, and no R2 files were staged or
committed. RA05–RA11 are not collectively qualified; R3–R5 and Stage 7 remain
unfinished. The local AGENTS rule makes a third build pass a deviation, so
execution pauses before any third R2 build. Owner choices for the verification
allowance are A — authorize passes 3–4 (recommended, bounded review point),
B — authorize passes 3–6 (more uninterrupted verification), or C — review the
current diff before authorizing another pass. Record the selection before
resuming; the R2 implementation and acceptance scope stays unchanged.

### 2026-11-25 — R2, pass 1: implementation started

R1 phase committed on main as `28a9284`; the working tree was clean afterward.
Shared runner dependency is `c49c694` in its owning repository. Execution of R2
is already approved. Re-grounding confirms that the current loader still uses
an incomplete handwritten JSON grammar, references are not integrated into the
generation pipeline, and SDK mapping inspects names without transitive types.
Implement the approved standard-library parser/graph/type approach and preserve
all pending RA05–RA11 exits. No R2 qualification or publication is claimed by
this start entry. Retain the R2/R3 coherent publication boundary.

Implemented strict `encoding/json/jsontext` loading with exact numeric tokens
and bounded file reads; registered-resource reference resolution integrated
into generation; deterministic vocabulary checks and exact integral item
bounds; and offline, readonly, CGO-disabled candidate compilation settings.
Added the initial RA05, RA06, RA07 and RA11 regression cases. These are partial
implementations; the complete candidate assembly and RA08/RA10 remain pending.

Ran `python3 -u scripts/gate.py unit --module generator` through the shared
Linux host runner/queue. Standalone vet passed and 101 tests/subtests passed
across nine packages. The overall gate correctly exited 1 with two failures:
the fresh copied tree lacked the prepared pinned linter, and the system linter
was 2.14.0 instead of required 2.15.0. Consequently generator lint did not run.
Use explicit pinned-tool preparation before subsequent lint qualification;
do not accept or substitute the system version. The later SDK auditor changes
were not present in this pass and are not covered by its passing test count.

### 2026-11-25 — R1, pass 5: qualified across all four local environments

Implemented approved corpus binding A: the existing bounds fixture names
`Bounded` and `Empty`; the harness rejects ambiguous/missing named definitions
and represents expected item bounds with pointers so zero is checked. It also
requires the corresponding assertion to be present. Single-definition fixtures
retain every original check. No generator production behavior was changed.

Ran `python3 -u scripts/qualify_r1.py --allow-network` through the shared host
runner/queue on Windows/amd64, Darwin/arm64, Linux/amd64 and the additional WSL
environment. The Windows/WSL runs were serialized by their shared physical queue.
All four used input identity
`2f8a43fc8c04f82736888f20dbe517c3b5c285d6a339d56b52603a408c2dc44d`.

| Check | Windows | Darwin | Linux | WSL |
| :--- | :--- | :--- | :--- | :--- |
| R1 harness | 20 passed | 20 passed | 20 passed | 20 passed |
| Ownership inventory | Passed | Passed | Passed | Passed |
| Pinned preparation and actual runner entry | Passed | Passed | Passed | Passed |
| Runtime formatter/vet/tests/lint | Passed; 56 tests | Passed; 56 tests | Passed; 56 tests | Passed; 56 tests |
| Generator formatter/vet/tests/lint | Passed; 66 tests | Passed; 66 tests | Passed; 66 tests | Passed; 66 tests |
| Actual SA1019/printf valid and broken controls | Passed/caught | Passed/caught | Passed/caught | Passed/caught |
| Corpus-binding valid and broken controls | Passed/caught | Passed/caught | Passed/caught | Passed/caught |
| Source preservation | Passed | Passed | Passed | Passed |
| Overall exit | 0 | 0 | 0 | 0 |
| Execution wall time, excluding queue/transfer | 62 s | 32 s | 75 s | 43 s |

Go test counts include subtests. Every run used Go 1.27.4, CGO off, no workspace
and readonly modules for checks. Explicit setup alone could acquire inputs.
Pinned Node/npm/linter identities and conformance commit/hash checks passed.
The prepared runner digest remained
`52212bd32f2222e7513b634f6b9e14d379c9886e67362cf1ae58696eccb6e1f3`.
No speedup claim is made from these timings. The compiler-floor/product/race
matrix remains assigned to its later qualification steps.

The maintained `scripts/verify_fixture_bindings.py` ran one valid control and
ten deliberately broken scratch copies on every environment. Missing numeric
binding, missing item binding and unknown binding failed with
`FAIL fixture binding`; wrong binding/minimum/maximum failed with their numeric
token diagnostics; changing either zero item bound failed its value assertion;
removing either zero assertion failed `FAIL minItems: assertion absent` or
`FAIL maxItems: assertion absent`. All mutation commands returned nonzero for
the intended diagnostic; the positive control returned zero. Source fixtures
were never dirtied by a plant. Actual generated/handwritten SA1019 plants and
the standalone-vet/former-govet printf controls likewise passed valid controls
and caught their deliberately broken inputs.

R1's scoped RA01–RA04 exit is satisfied. Product API/generation/wire/full
qualification modes still report their documented incomplete contracts; R1
does not certify R2–R5, Stage 7 or Stage 9. Preserve the prior failed passes.
No unnecessary sixth R1 pass was run. Shared runner repair was committed on
its owning repository's main as `c49c694`; the 26 shared regression tests and
native Windows archive controls underpin that dependency. The R1 phase commit
includes these records and the earlier preserved audit/duplication reports.

### 2026-11-25 — Corpus binding deviation: Option A approved

The owner selected A: add explicit definition bindings to existing fixture
expectations, retain all numerical assertions, and explicitly check zero item
bounds. Extend R1 to the generator's `internal/model/fixtures_test.go` and
`testdata/fixtures/bounds.json`, with maintained scratch controls under
`scripts/`. Ambiguous multi-definition expectations must fail instead of using
map order. Single-definition fixtures retain their existing assertions. Verify
wrong numeric tokens, wrong zero bounds and missing bindings on scratch copies.
R1 passes 5 and 6 are authorized; preserve previous failures. Commit to main
only after the phase's required checks pass. R2–R5 execution approval continues.

### 2026-11-25 — R1, pass 4: lint and preparation fixed; flaky corpus binding found

Added the approved handwritten `protocol/doc.go` package documentation.
Executed `python3 -u scripts/qualify_r1.py --allow-network` on native
Windows/amd64, Darwin/arm64 and Linux/amd64 through the shared runner and
per-host queue. All used this identical input identity:
`ce14fc01551475f6c14f2a4ccac174f2422a540cba40bed0eb800234a14f35c3`.
No source change occurred during any run. Linux selection used the primary
Linux machine; this pass did not qualify the additional WSL environment.

| Check | Windows | Darwin | Linux |
| :--- | :--- | :--- | :--- |
| R1 harness, including scratch lock-mutation and explicit-network controls | 20 passed | 20 passed | 20 passed |
| Ownership inventory | Passed | Passed | Passed |
| Exact tools, isolated dependency metadata, schema and conformance preparation | Passed | Passed | Passed |
| Runtime formatter/vet/tests/lint | Passed; 56 tests | Passed; 56 tests | Passed; 56 tests |
| Generator formatter/vet/lint | Passed | Passed | Passed |
| Generator tests | Failed: `TestFixtureCorpus/bounds.json` | Same failure | Passed; 66 tests including subtests |
| Generated/handwritten SA1019 valid and broken controls | Passed/caught | Passed/caught | Passed/caught |
| Standalone vet and former duplicate govet printf controls | Passed/caught | Passed/caught | Passed/caught |
| Source preservation | Passed | Passed | Passed |
| Overall | Exit 1 | Exit 1 | Exit 0 |

The actual lint plants now have valid clean baselines. All intended broken
analyzer controls were caught on every target. No suppression was added.
Native preferred compiler was Go 1.27.4 with CGO off throughout. Root and
generator module metadata remained unchanged. Complete logs remain in session
scratch artifacts; private host paths are not copied into these records.

Read-only diagnosis of the new failure:

* `cmd/mcpgen/internal/model/fixtures_test.go:87–96` compares minimum and maximum
  against `firstNode(m)`. That helper at the end of the file returns the first
  value encountered in a Go map, with no definition binding.
* `cmd/mcpgen/testdata/fixtures/bounds.json` contains both `Bounded` (numeric
  minimum/maximum) and `Empty` (array item bounds). When `Empty` is selected,
  diagnostics are `FAIL minimum token: got "" want "0.30000000000000004"` and
  `FAIL maximum token: got "" want "1e308"`.
* Both the corpus harness and fixture are unchanged from HEAD (empty diff).
  Pass 3 and Linux pass 4 passing do not invalidate the Windows/Darwin failures.
  The harness also conditions item-bound checks on nonzero expected values,
  leaving the fixture's explicit zero bounds unchecked; that must be corrected
  while preserving the numerical assertions.

R1 remains incomplete; no source is staged or committed. The two additional
passes approved by D2 B have been used. Before another build, request approval
to bind expectations to explicit definitions (recommended A), split bounds into
separate numeric/array fixtures (B), or convert all fixture expectations to a
per-definition map (C). Each option retains every existing predicate and tests
zero bounds explicitly; none skips the failing test or retries until it passes.
R2–R5 remain authorized but unstarted behind the required R1 phase boundary.

### 2026-11-25 — Package documentation deviation: Option A approved

The owner selected A, approving a handwritten `protocol/doc.go` package comment
within R1. Add package documentation without changing generated declarations,
the generated ownership manifest, assertions or lint policy. This resolves the
pending ST1000 source-scope decision recorded in pass 3. Pass 4 includes this
file and the prepared R1 fixture/module-preparation corrections. Existing
approval to execute R1–R5 and commit each qualified coherent phase to main
continues; no push or Stage 7 execution is authorized.

### 2026-11-25 — D1 A / D2 B: approved deviation and broader R1 review

The owner selected D1 A (repair and qualify the shared Windows transfer) and
D2 B (broader read-only R1 review, then formatter/tool corrections and two
further qualification passes). Execute the existing 0002 pair with this scope
amendment: add the canonical `available-dev-hosts/scripts/run-on-host` and its
maintained transfer tests in the shared skill source, and formatting-only
changes to `output_test.go`. The shared runner source lives in its owning
dotfiles repository; do not change other product repositories or skill policy.
Preserve its queue, source-copy checks and failure behavior. Record shared
changes separately from this product's phase commit. No push is authorized.

Read-only R1 review covered the shared runner's archive/copy/verify paths, gate
dispatch and pre-add call graph, event accounting, preparation and acceptance
inventory. Remaining gaps: expected test identities are not wired into live
accounting; default test vet duplicates standalone vet and linter govet;
inventory role/direction labels need independent method bindings; module
preparation failures need per-module aggregation; conformance verification
needs runtime dependency integrity and usable-entry evidence. Timeout/process
behavior and actual generated/handwritten lint plants remain unqualified.

Native Windows scratch probe confirms Python 3.14 can create and read the
required relative symlink with target `CLAUDE.md`; no host symlink setting
change is necessary. The current transfer delegates extraction to the MINGW
tar command. Repair it with native Python archive extraction and explicit link
verification, preserving link targets and rejecting unsafe archive entries.
Use maintained valid/broken archive fixtures, then exercise actual native
Windows transfer. This replaces the failed transfer mechanism without dropping
the required symlink or relaxing copy verification.

R1 passes 3 and 4 are authorized after this broader review. Preserve every
failure; if those passes do not establish the exit, report the remaining cases
and request another decision under the third-pass rule.

### 2026-11-25 — R1, pass 3: Windows transfer qualified; product gate failed

Completed the broader review before the implementation updates. The shared
runner now extracts Windows archives using native Python, with files before
links, path/type/link-ancestor validation and explicit symlink target checks.
The existing source-copy count checks, queue and failure fallback remain.
Native Windows `test_windows_archive.py`: exit 0, two tests passed. Valid
link-before-target control preserved `AGENTS.md -> CLAUDE.md`. Seven broken
archive controls were rejected: traversal, absolute path, drive path, link
escape, link ancestor, duplicate and device entry. The shared runner's full
`python3 -B -m unittest -v test_run_on_host test_windows_archive` regression
suite on queued Darwin: exit 0, 26 tests passed in 80.239 seconds. This includes
queue, WSL isolation/cancellation, copy verification and exit-status contracts.
The shared source/test changes are uncommitted in their owning repository.

R1 source updates: corrected only whitespace in `output_test.go`; added
`tools/test-inventory.json` with 56 runtime and 58 generator top-level test
identities, wired as required passing results; retained D/G obligations as
independent, unqualified product requirements. Added reviewed method directions
and definition-to-method reference tracing to the acceptance inventory.
Preserved 39 obligations, 155 definitions, 21 methods and 89 scenarios.
Standalone full vet owns its analysis once per selected module; test execution
uses `-vet=off`, and golangci's duplicate govet producer is disabled. No
staticcheck suppression or generated-file exclusion was introduced.

Pinned preparation can build a missing/wrong linter into an isolated native
tool cache and verifies its version/build compiler before promotion. Module
acquisition uses scratch metadata. Conformance preparation now verifies runtime
dependency bytes as well as the built runner and checks its actual CLI entry
point. Explicit preparation retains invalid cache material separately before
rebuilding; ordinary offline checks cannot rebuild it. Native npm invokes its
Node entry point without a shell. Added `scripts/qualify_r1.py` to run named
independent checks and verify the source tree is byte-identical afterward.

Executed the frozen pass-3 candidate through the repaired shared runner on
native Windows and queued Darwin:

```text
python3 -u scripts/qualify_r1.py --allow-network
input identity: caf609c99422a4a00f28bf32d9a05fb967dfaf784143e478ceb293cbb490c3eb
```

| Check | Windows | Darwin |
| :--- | :--- | :--- |
| Harness and deliberately broken event/cache controls | 18 tests passed | 18 tests passed |
| Acceptance ownership inventory | Passed; product remains unqualified | Passed; product remains unqualified |
| Pinned tools, schema and built conformance runner | Passed | Passed |
| Runtime dependency metadata preparation | Failed: `module lookup disabled by GOPROXY=off` | Passed |
| Formatter, standalone vet and runtime tests | Passed; 56 tests | Passed; 56 tests |
| Runtime lint | Failed: three ST1000 reports in `protocol/` | Same failure |
| Generator formatter/vet/tests/lint | Passed; 66 tests including subtests | Passed; 66 tests including subtests |
| Actual SA1019 plants, generated and handwritten | Both caught | Both caught |
| Standalone vet / former govet valid and broken printf controls | All passed/caught as intended | All passed/caught as intended |
| Lint plant clean baseline | Failed: fixture lacked valid package documentation | Same failure |
| Source preservation | Passed | Passed |
| Overall | Exit 1, three failed check groups | Exit 1, two failed check groups |

Full logs remain session scratch artifacts, with private host paths excluded
from these records. No R1 exit, phase commit or later repair completion is
claimed. Native Linux/WSL and compiler-floor qualification remain outstanding.

Pass-4 corrections prepared within R1: give the lint fixtures proper package
comments, allow module metadata acquisition only during explicit network setup,
aggregate each module's preparation failures independently, and add scratch
lock-mutation/network-permission controls. These corrections have passed static
Python AST parsing but have not executed in a qualification pass.

Pending deviation decision: actual lint revealed missing package documentation
in existing generated `protocol/aliases.go`, `protocol/constants.go` and
`protocol/owned.go` (`ST1000`, package declaration lines). This is pre-existing
product source outside the listed R1 gate/test-setup files. Offered A (a small
handwritten `protocol/doc.go` in R1), B (generator emission plus R2/R3 qualified
publication), or C (one coherent R1–R3 change and commit). Recommended A.
No protocol source was changed while that decision is pending. The next
qualification pass remains pass 4; no checks are bypassed to commit R1.

### 2026-11-25 — R1–R5 execution authorization

The owner approved execution of R1–R5 and explicitly requested staging and
committing to main between phases. This supersedes the earlier R2–R5 pending
authorization statements; the recommended implementation options are approved.
Stage 7 and push/tag remain outside this instruction. Preserve unfinished R1
work and commit each coherent phase only after its acceptance and pre-add
checks pass. Starting revision remains `5bfdd9f`.

### 2026-11-25 — R1, pass 2: partial repairs and failed pre-add

Applied initial R1 corrections to `scripts/gate_common.py`,
`scripts/prepare.py` and `scripts/test_gate.py`: isolate inherited Go environment
settings; resolve the linter on PATH; retain bounded disk command logs and
partial timeout diagnostics; terminate timed-out process trees; distinguish
timeout/missing-command/log-limit statuses; support required named test results;
remove arbitrary cache-text rejection; acquire/verify Go dependencies using
scratch module metadata and verify pinned runtime versions. Added controls for
missing required cases, literal cache text, environment pollution and timeout
diagnostics. These changes remain unqualified; required-case accounting still
needs production inventory wiring, and the remaining R1 steps are incomplete.

The existing shared runner selected native Windows for
`python3 -m unittest discover -s scripts -p test_gate.py -v`. Its transfer failed:
`tar: ./AGENTS.md: Cannot create symlink to 'CLAUDE.md': No such file or directory`.
The symlink is required and unchanged in the source tree. The runner released
the remote claim and queued a local fallback behind another repository's job.
This is a dispatch failure, not evidence that the harness ran on Windows.
An owner decision was requested before repairing the shared runner or changing
host symlink administration; neither is implicitly part of this repository's
R1 source changes.

Independently dispatched `python3 -u scripts/go-precheck.py` through
`run-on-host --need linux`; selection chose the WSL environment. Exact relevant
output, with the machine-specific scratch prefix removed:

```text
FAIL pre-add.linter: required golangci-lint 2.15.0 built with go1.27.4; golangci-lint has version 2.14.0 built with go1.27.4
FAIL pre-add.runtime.gofmt: gofmt would rewrite: <scratch>/output_test.go
pass pre-add.runtime.vet
pass unit.runtime: 56 tests passed; 4 packages accounted
FAIL lint.runtime: pinned linter unavailable
pass pre-add.generator.gofmt
pass pre-add.generator.vet
pass unit.generator: 66 tests passed; 9 packages accounted
FAIL lint.generator: pinned linter unavailable
gate pre-add: 4 FAIL case(s)
```

Exit: 1. Full dispatch output is retained in the session scratch log, not a
committed artifact containing private machine paths. The runner stopped the
WSL environment it had started. The linter mismatch is pre-existing host drift;
the formatter finding is pre-existing at HEAD (`output_test.go`, tool
registration literals), confirmed by the empty starting diff for that file.
No test assertion was weakened and no phase exit or commit is claimed.

Deviation pending review: align the runner's pinned tools and qualify Windows
symlink transfer; include the existing `output_test.go` formatter correction in
R1 if approved. A further build pass must respect the repository's third-pass
review rule. Keep all outstanding R1 inventory, analyzer ownership, preparation,
actual lint plants and native qualification obligations visible.

After 241 seconds waiting behind another repository's heavy job, cancelled only
this task's queued harness ticket while the required deviation decisions were
pending. Runner output: `interrupted while waiting; ticket removed; nothing run`;
exit 130. No harness test or its new negative controls executed in this pass.
No test process or polling loop remains running for this task. Static Python
AST parsing of all three changed helpers passed; records validation,
markdownlint-cli2 v0.23.2 (32 files, zero issues) and `git diff --check` passed.

### 2026-11-25 — R1–R5 research and planning pass 1

The owner requested assessment, options and a PLAN update. Reviewed product
source and unfinished R1 changes at HEAD `5bfdd9f`, the pinned SDK v1.8.0
module source, and the primary documentation linked in the research refinement.
Updated this PLAN, its MADR clarification and the documentation index only.
No repair implementation, product build, suite, negative-control plant, native
host qualification, commit or push was performed in this planning pass. It is
not a numbered repair build attempt and establishes no RA exit result.

The sorted modified/untracked non-documentation path-NUL/content-NUL SHA-256
was identical before and after editing:
`e1080c42481ace4a254aa426fe01f289e0323882459f3feb15ee63e3e8b837ff`.
Existing R1 source/configuration work is preserved and remains unqualified.

Documentation verification:

* `python3 scripts/check_records.py --check-all`: exit 0; no diagnostics.
* Cached `markdownlint-cli2` v0.23.2, invoked through Node with this MADR, PLAN
  and `docs/README.md` as arguments: exit 0. Repository configuration expanded
  the selection to 32 files; output: `Summary: 0 issues in 0 files`.
* `git diff --check`: exit 0; no diagnostics.

### 2026-11-24 — R1, pass 1: execution authorized

The owner instructed “resume r1” after reviewing the repair sequence. Execute
only R1 in this repository; no fleet skill adoption, other-repository changes,
Stage 9 rollout, push or tag is authorized. Starting HEAD is `5bfdd9f`.
The five existing dirty documentation files are preserved. Their sorted
path-NUL/content-NUL inventory SHA-256 before this amendment is
`663933681242d4380d5090b05dbedb8862c0286db4d3c76d959f620f8849b42f`.
Re-grounding confirms A02–A04 in the current gate, pre-add and lint configuration.
Implementation and verification results follow when established; no R1 exit is
claimed by this authorization entry.

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

### 2026-11-24 — Stage 0, pass 1: authorized investigation

The owner instructed: “set the madr to accepted, approve the plan, and proceed
to stage 0”. The MADR was already accepted and the PLAN already approved; this
entry records renewed approval and supersedes the earlier wait instruction for
Stage 0 only. Starting revision: `bd99482`; the working tree was clean.
The empty dirty-content inventory has SHA-256
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

Investigation is incomplete. See
[0002-REPORT-stage-0-framework-inventory.md](../reports/0002-REPORT-stage-0-framework-inventory.md)
for verified identities, commands, B01 evidence, and the unfinished obligation
ledger. The original prototype report remains unchanged. No product source,
dependency, or configuration changed.

**Deviation requiring owner ruling:** the pinned SDK deprecates sampling and
roots, while Stage 6 includes those interactions and the pinned conformance
set requires their basic scenarios. The accepted contract also forbids
deprecated SDK symbols, transitive exposure, and Staticcheck exceptions. A
compliant binding has not been established. This is a pre-existing contract
conflict, not a product test failure. No workaround or changed scope has been
selected. Stop under the repository deviation rule; amend the pair with the
owner's resolution before resuming. Stage 0 has not met its exit criterion.

Affected files: this PLAN, its MADR (approval chronology and stale approval
wording only), `docs/README.md`, and the execution report. No architecture
amendment has been accepted for B01. No new gate or plant experiment is due;
no build pass or phase commit occurred.

**Documentation verification, pass 1:** `run-heavy --max-wait 30 -- python3
scripts/check_records.py --check-all` returned 3 because the queue directory
was not writable; it ran nothing. The available-dev-hosts skill explicitly
permits a direct fallback for this exit. The direct record validator exited 0
with no output; `git diff --check` also exited 0 with no output.

`npx --offline --yes markdownlint-cli2@0.23.2` could not resolve its cached
registry metadata (`ENOTCACHED`, exit 1), so lint had not run. Inspection found
the exact 0.23.2 package already installed in the npm execution cache. Running
its declared CLI entry point with Node exited 0, using the unchanged repository
configuration and all configured Markdown globs:

```text
markdownlint-cli2 v0.23.2 (markdownlint v0.41.1)
Linting: 23 files
Summary: 0 issues in 0 files
```

The four changed documents were reviewed and checked for local account, host,
and real-machine path disclosures. The original qualification report remains
byte-identical to HEAD. These documentation checks do not resolve B01 or mark
Stage 0 complete. The incomplete phase remains uncommitted.

### 2026-11-24 — Stage 0, pass 2: approved B01 Option A

The owner instructed “move forward with option a” after reviewing three
choices. This authorizes qualifying the existing pinned SDK integration;
it does not waive feature coverage, permit deprecated API exposure or lint
suppression, or replace SDK dispatch. The pass-1 stop instruction is superseded
for this investigation. B01 remains unresolved until evidence satisfies it.

Starting revision remains `bd99482`. The four existing dirty documents are
preserved. Their ordered path-NUL/content-NUL inventory has SHA-256
`32d15159809e5e5ee3713b96cb4d589a8cd814e97b24a83d60a5f04002a5e18c`.

Approved investigation steps:

1. Inspect public SDK multi-round codecs, callback registration, middleware,
   custom-method APIs, and capability handling against actual source.
2. Build a scratch proof using owned application values and public,
   nondeprecated SDK entry points. Test server requests and client fulfillment
   separately. SDK dispatch, retries, and request-state handling must remain
   enabled; success for one role cannot stand in for the other.
3. Require sampling output from the configured application handler and the
   actual configured root list. Empty roots, ignored callbacks, or an error
   from an advertised sampling capability fail qualification.
4. Run the pinned Staticcheck through golangci-lint, including scratch tests,
   with no exclusions. In separate copies, demonstrate that deprecated direct
   bindings are caught and that broken functional fixtures fail. Keep every
   original expectation; report failures instead of deleting features or tests.
5. Preserve source, commands, outputs, hashes, negative controls, and limits
   of the proof in the execution report. Resume the remaining Stage 0 inventory
   only if B01 is resolved. Otherwise return the concrete failure for review.

Exit: a demonstrated compliant integration for both roles, or an explicit
failed qualification with the remaining blocker retained. The latter completes
this investigation attempt but does not complete Stage 0. A third build attempt
requires the repository deviation process. Added scope is scratch diagnostic
code and evidence in the existing report; no product source or dependency
change is authorized. New diagnostic assertions require broken-copy controls;
the earlier exemption applied only to pass 1, which introduced none.

### 2026-11-24 — Stage 0, pass 3: Option D wrapper qualification

The owner selected Option D (codec-mediated fulfillment wrapper) after
reviewing the pass-2 failure and the option ledger, and instructed scoping
and specification. The ruling, consumer scope, and state rows G23–G25 are
recorded in the MADR amendment above; this section is the executable
specification of the pass-3 proof that qualifies them. Pass 1 failed
environmentally and pass 2 substantively; this is the third qualification
attempt, authorized by the recorded 2026-11-24 owner instruction that the
deviation process requires.

Composition under proof, per G23: the framework owns the client fulfillment
loop with `MultiRoundTripOptions.Disabled`; owned DTOs for
`InputRequest`/`InputResponse` are converted at the SDK seam only through
the public `InputRequestMap`/`InputResponseMap` codecs; no deprecated symbol
is named in source, public API, or lint scope; runtime traversal of the
SDK's deprecated concrete types inside its own package is the accepted,
documented boundary of this composition. G24's structural-discriminator
coupling is accepted without suppression and pinned by scenario tests.

Approved investigation steps:

1. **Manual-resumption proof (client role).** In a fresh scratch module,
   build the owned fulfillment loop: `MultiRoundTripOptions{Disabled: true}`
   on the client session; `CallTool` returns an input-required result;
   `NeedsInput()`, `InputRequests`, and `RequestState` are read through the
   owned DTOs via `InputRequestMap.MarshalJSON`; owned handlers produce
   owned response DTOs; retry with `Params.InputResponses` populated via
   `InputResponseMap.UnmarshalJSON` and the echoed `RequestState`. The
   server fixture is the passing pass-2 server role, reused unchanged.
   Success requires real sampling output from an owned handler and the
   actual configured root list — empty, ignored, or erroring responses fail,
   per the pass-2 criteria.
2. **Proxy-handler control.** A separate copy demonstrates that a wired but
   deliberately broken owned handler (wrong model, wrong root) fails the
   scenario, proving the loop routes through owned code and not a silently
   succeeding default.
3. **Lint gate.** Pinned golangci-lint 2.15.0, staticcheck, no exclusions,
   over the scratch proof: zero findings is the pass condition. Any SA1019
   finding names a composition defect, not a policy allowance.
4. **Discriminator pin.** A separate copy demonstrates that a malformed
   probe payload (a sampling response lacking the `role` key) fails
   decoding loudly, pinning G24's coupling so an SDK probe change surfaces
   as a scenario failure rather than silence.
5. **Evidence and exit.** Record source, commands, outputs, hashes,
   negative controls, and limits in
   [0002-REPORT-stage-0-framework-inventory.md](../reports/0002-REPORT-stage-0-framework-inventory.md),
   with host paths redacted. Exit criterion: both scenarios pass through the
   owned loop with zero lint findings, or an explicit failed qualification
   returns to the owner with Option C recorded as the pre-approved fallback
   that a fresh ruling must still select.

Scope and authority: scratch diagnostic code and report evidence only. No
product source, dependency, lockfile, or implementation stage is authorized.
The four dirty documents and their inventory hash are preserved untouched.
Option C remains recorded but unexecuted; selecting it requires a fresh
owner ruling under G23 because it amends the frozen conformance pin.

### 2026-11-24 — CI/CD review decisions and Stage 9 scope amendment

The owner selected D1 A (existing Python gates plus thin Makefile) and D3 C
(one-hour exact-commit receipts with fresh local vulnerability evidence per push).
For D2, the owner reports making the repository public, manages disclosure and
security scans locally, and excludes GitHub enforcement. This supersedes the
assessment's private-repository protection dependency and 24-hour recommendation.
GitHub functional CI on all three targets remains in scope.

Affected documentation: this MADR/PLAN pair,
[0009-REPORT-cicd-standardization-assessment.md](../reports/0009-REPORT-cicd-standardization-assessment.md),
and its index. Stage 9 gains the Makefile, receipts and CI01–CI07 acceptance cases;
GitHub enforcement and security-scan administration are excluded. This is a
policy/specification amendment, not a build pass or completed phase. No workflow,
hook, dependency or product code was changed by this amendment.

### 2026-11-24 — Audit deviation and proposed R1–R5 repair amendment

Evidence: [0011-REPORT-pre-stage-7-completion-audit.md](../reports/0011-REPORT-pre-stage-7-completion-audit.md),
findings A01–A20 at `5bfdd9f5e4a6e9286cf593fdc1f3ccec60890532`. The failures are
pre-existing. The audit records exact source locators and executed negative
inputs; eight runtime diagnostic groups and eight generator diagnostic groups
fail despite successful existing Linux suites. The generator suite also skips
two tests. This is an implementation/completion deviation under the accepted
contract, not permission to loosen it.

Owner selection: “do A repairs”, selecting the offered option A to draft the
existing pair's repair amendment and acceptance ledger for review before
execution. R1–R5 and RA01–RA20 above make the selected repair concrete. The
architecture remains accepted; review/approval of this detailed execution
amendment is pending. No repair build pass, product edit or phase commit has
occurred in preparing it.

Starting dirty files: `docs/README.md` and the audit report. Their sorted
path-NUL/content-NUL inventory SHA-256 is
`6d9bead6eef4152cec9c441fa1a4552f1e8b793263bcfe88ab1faa273f38f5c3`.
The report is preserved unchanged; the index gains current audit/repair status
annotations. This amendment edits only the MADR, PLAN and documentation index.
Affected product scope for subsequent approval is enumerated per repair phase,
including original-manifest migration, production-path fixtures, private SDK
bridge adaptations and external consumer tests. Stage 7–10 scope and CI08's
explicit exclusion remain unchanged.

Documentation verification: `python3 scripts/check_records.py --check-all`
and `git diff --check` exited 0. The installed pinned markdownlint-cli2 0.23.2
entry point, invoked with Node and the repository configuration, checked 31
Markdown files with zero issues. A one-use read-only comparison verified that
all 20 RA state IDs map one-to-one to the same named groups in this PLAN.
These are documentation checks, not repair implementation or product tests.

### 2026-11-24 — Dedicated local scaffolding name and scope

The owner selected `local-cicd-scaffolding` and explicitly limited it to local
acceptance, validation, tests and gates through pre-push readiness, with idempotent
application and no upstream-forge-specific content or operations. The MADR now
records ownership, the complete local capability inventory and LC01–LC10; this
PLAN maps those states to future adoption evidence. The conversational working
name is superseded; no existing skill directory required a rename.

This is a specification amendment to the current pair, not a skill installation,
repair build pass, hook change or Stage 9 execution. Existing upstream work stays
outside the skill, and pending repair authorization is unchanged. The skill's
canonical implementation belongs to the fleet skill library and needs that
repository's implementation record. Only this pair and its index were changed.

### 2026-11-24 — Explicit local runners and race-only CGO

The owner clarified the required runner set: native Windows 11, its WSL Linux
environment, the session Mac and standalone Ubuntu 26 Linux, with SSH for remote
execution. Host aliases remain in private fleet configuration. This amendment
adds explicit four-environment coverage and LC11–LC12 to the local skill contract;
the two WSL baseline cells supplement the six native cells. Existing upstream
matrix requirements are unchanged. The current fleet policy keeps WSL out of
race runs and serializes its heavy work with native Windows on the same machine.

The owner also requires CGO off for all non-race operations and builds. The
contract now names explicit subprocess environments, child-build handling and
artifact separation; CGO on is limited to qualified race-test execution and its
instrumented compilation. This is documentation of required behavior, not an
executed host probe, implemented gate, or completed qualification run.

### 2026-11-24 — Local gate duplication assessment and streamlined execution

The owner requested analysis of overlapping Make targets, tests, linters,
validators and checkers across the reference repositories, with necessary checks
retained and redundant scans removed. Findings and conservative command-count
reductions are recorded in
[0012-REPORT-local-gate-duplication-assessment.md](../reports/0012-REPORT-local-gate-duplication-assessment.md).
The MADR now requires canonical execution ownership, evidence reuse with complete
context, coverage equivalence and measured costs. LC13–LC16 specify future proof.

This amendment changes the skill/adoption specification, not existing gate source
or configuration. No suite was rerun, no check was disabled and no timing saving
is claimed. The reference repositories' uncommitted work was preserved. Pending
repair and Stage 9 execution status are unchanged; consumer and fleet skill
implementations remain separate recorded work.
