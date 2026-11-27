---
status: implemented
date: 2026-11-27
associated-madr: 0014-MADR-organize-mcp-runtime-package.md
---

# Implement functional MCP packages and scoped internals

Associated MADR:
[0014-MADR-organize-mcp-runtime-package.md](0014-MADR-organize-mcp-runtime-package.md).

## Goal

Place the cohesive public runtime and its tests under `mcp/`, its private bridge
under `mcp/internal/sdkbridge/`, and keep independent public capabilities and
shared validation at their reviewed scopes. Leave zero Go files at the root.
Preserve runtime behavior, SDK isolation, generated
contract bytes and the complete existing test/gate inventory. Document the
intentional migration to `github.com/maccavelli/go-mcp-lib/mcp`.

The owner accepted the revised architecture and approved execution on
2026-11-27. L0 and L1 are complete. The coherent migration commit is
`c28f603b35abb1dcd9fe368ba856663ad79eb46d` on main.

## Grounding Reference

The accepted amended decision is
[0014-MADR-organize-mcp-runtime-package.md](0014-MADR-organize-mcp-runtime-package.md).
The measured reference layouts and source locators are in
[0013-REPORT-library-package-layout-assessment.md](../reports/0013-REPORT-library-package-layout-assessment.md).
The current published baseline is `5c00e3680393e960f74a8907b1d6d8eaaf5f7336`,
with approved R2/R3 repairs then uncommitted. L0 subsequently re-grounded all
move locators at `d1276de4761aa77d6d3389370e6781c049753131`, the qualified
coherent R2/R3 and Make/lint repair commit.

## Scope

* The root runtime package's twelve production files and twelve test files,
  re-inventoried at implementation time, move together into `mcp/`.
* Move the runtime-only bridge and its tests from `internal/sdkbridge/` to
  `mcp/internal/sdkbridge/`; update imports, inventory identities and plants.
* Update imports, examples, package identities in `tools/test-inventory.json`,
  and runtime source/test locators used by generator qualification and plants.
* Add the root-layout and dependency-boundary assertions with meaningful scratch
  controls; integrate them with the canonical local runner. Reuse compiler,
  import lint and inventory checks instead of duplicating scans or suites.
* Update architecture, current documentation links and migration guidance.
* Retain the root module, generator module, schema/profile pins, generated
  `protocol/` inventory and existing public codec/protocol import paths.
* Keep schema validation and fixtures under root `internal/validation/` because
  runtime policy and generated protocol support share that implementation.

The reference repositories remain read-only. Do not add a compatibility facade,
create new public client/server/transport packages, change runtime semantics,
alter generated ownership, modify upstream-forge processes, push, tag or rewrite
history under this PLAN.

## Prerequisites

* Owner acceptance of the revised MADR, resolution of L10/L11, and explicit
  execution approval: satisfied on 2026-11-27.
* The coherent R2/R3 qualification, publication and phase commit completed.
* A reviewed current inventory with no destination or ownership conflict.
* Prepared pinned tools/dependencies and the existing queued native-host workflow.

## Implementation Steps

### L0 — Re-ground the accepted decision and prerequisites

1. Read the current tree, index, HEAD, accepted MADR and the R2/R3 execution
   record in [0002-PLAN-build-schema-driven-mcp-framework.md](0002-PLAN-build-schema-driven-mcp-framework.md).
   Confirm its safe publication, normal regeneration and coherent R2/R3 phase
   commit are complete before moving runtime source. Preserve inherited work.
2. Record the exact root and bridge Go inventories and source/import/test
   locators at that HEAD. Re-ground the MADR's cited files, including every
   bridge importer and generated protocol validator dependency, rather than
   relying on line numbers or counts. Stop if a destination contains conflicting
   work, the bridge has a new sibling consumer, or the expected ownership has
   changed without a reviewed explanation.
3. Inventory in-repo consumers and known external usages read-only. Prepare the
   import migration note; no finding authorizes editing another repository.
4. Record the actual package import graph and module/test inventories. Map every
   current production/test file to the MADR's owner table. Reuse existing
   dependency gates for forbidden edges and identify only missing assertions.

Exit: the prerequisite is qualified and committed, the move inventory is
reviewable, and no destination or ownership conflict remains. L0 is read-only
apart from its execution record; it requires no artificial source commit.

### L1 — Move, update and qualify one coherent package migration

1. Move the inventoried production and test files together into `mcp/`.
   Preserve `package mcp` / `package mcp_test`, public names, test names,
   private interfaces and SDK isolation. Replace current in-repo runtime
   imports with `github.com/maccavelli/go-mcp-lib/mcp`.
   Move the bridge's implementation and tests to `mcp/internal/sdkbridge/` and
   replace its runtime imports. Leave shared validation and its fixtures at
   root scope; do not move generated output or introduce another validator.
2. Update runtime test package identities and generator candidate plant paths.
   In particular, the injected accessor and middleware bypasses must target
   `mcp/server.go` and `mcp/middleware_bridge.go`, and their selected runtime
   tests must run in `./mcp`, rather than the now-empty package root.
   The owned-decoder bypass must target `mcp/internal/sdkbridge/codec.go`.
   Update the bridge's inventory package identity alongside the runtime's.
   Keep `.` as the runtime **module** identity in candidate reports; package
   relocation does not create a different module.
3. Review relative filesystem accesses in moved tests. Preserve complete
   runtime/test discovery and external consumer coverage. Update every active
   locator rather than skipping a test that can no longer find its fixture.
4. Implement `TestLibraryLayout` / `TestLibraryLayoutConflict` in the existing
   Python local-validation/test layer, with explicit FAIL case diagnostics.
   Check zero root Go files and the expected package ownership; reject a
   root-file plant and a conflicting destination in scratch. Reuse Go compiler,
   API checks and configured import lint for semantic/dependency coverage.
   Prove a sibling's import of `mcp/internal/sdkbridge` is refused and each
   enforced forbidden edge is caught using scratch consumers/fixtures. Check
   supported public consumer imports, separate module identities, colocated
   test discovery and absence of command imports from library packages.
   No arbitrary file limits, duplicate scan targets or empty package stubs.
5. Update `docs/architecture.md`, current examples and import references, and
   add a migration guide. Preserve historical evidence, repairing links with
   an explicit old-to-current path mapping where necessary. State that callers
   must migrate the old import and that no root facade is provided.
6. Qualify the complete candidate with both module boundaries. Require identical
   reviewed protocol/profile/output hashes, complete native and compile matrices,
   every existing runtime/API/bridge case, and all old and new planted failures.
   Run each matching gate leaf once per source/environment tuple.
7. Run pre-add formatting/lint through the canonical wrapper, record/link and
   Markdown checks, and the local identifier scan. Commit the coherent L1 phase
   with `git commit --no-edit` after all required checks pass, under the owner's
   applicable branch/commit authorization. No push or tag is included.

Exit: no blocker, every L01–L11 state approved and verified, deliberate failures
caught, zero root Go files, no unreviewed runtime API change, unchanged generated
contract, and the package migration is committed coherently. Number build passes
in the execution record; the global third-pass deviation rule applies to L1.
Its passes do not consume or extend an R3 allowance.

## Verification

| MADR case | Evidence required before the L1 commit |
| :--- | :--- |
| L01 | Root inventory is empty; `mcp/` contains the complete reviewed move; existing same-package and external tests are discovered and pass. |
| L02 | A scratch consumer of `/mcp` builds; a legacy-root consumer is rejected; current examples and migration guidance use the canonical import. |
| L03 | A deliberately planted root Go file makes `TestLibraryLayout` fail with its own diagnostic. |
| L04 | Scratch destination-conflict cases fail without changing either inventory; the actual move preserves inherited work and is reviewed against L0. |
| L05 | Actual complete candidate compilation and native runtime pre-add pass; accessor, parameter/result-type and validation-bypass plants are caught at their new paths. |
| L06 | `TestOwnedPublicAPI`, middleware boundary checks, connection isolation and existing server/client/resource/tool cases all pass; raw SDK runtime exposure remains rejected. |
| L07 | Both modules retain complete target results; the independently pinned inputs and full generated-file/manifest SHA inventory are unchanged; repeat CLI generation/check pass. |
| L08 | Native Windows/amd64, Darwin/arm64, physical Linux/amd64 and requested WSL coverage; queued heavy runs; normal CGO-disabled builds/tests and separately attributed eligible-host race results. |
| L09 | The preceding R2/R3 coherent commit and its publication/regeneration evidence are named in L0 before relocation begins. |
| L10 | The bridge has only runtime owners at L0 and is moved with all tests/plants; a sibling bridge import and forbidden runtime/protocol/codec edges fail in scratch. Shared validation stays at root, its corpus remains discovered, and no cycle or SDK exposure is introduced. |
| L11 | Canonical consumers import functional library packages; library packages do not import command tooling. Both module identities and complete package/test inventories survive; no generic test module or unnecessary helper package is created. |

Reuse the canonical gate/qualification entry points after updating their
package locators. Do not substitute a foreign compile for native execution or
an old package result for the moved package. Record commands, source identity,
host class, compiler, CGO mode, terminal results and negative-control diagnostics.

## Rollout and Rollback

The order is: review and accept the amended MADR and approve this PLAN's
execution; complete the already approved R2/R3 boundary; perform and qualify
L1; publish migration guidance with its commit.
Update downstream applications only under authorization in their own repositories.

## Rollback

A failed migration remains uncommitted while the decided cases are repaired.
Preserve unrelated work and collect concrete conflicts for owner review. Do not
run destructive resets, delete an existing destination or rewrite published
history to make the move appear clean. Any reversal of an already committed
API move requires a reviewed follow-up; it is not an implicit facade exception.

## Completion Criteria

The L1 exit and every L01–L11 verification row are satisfied, every required
plant has failed as intended, and the execution record names the canonical
import and coherent migration commit. No root Go files, stale active package
locators or unexplained generated-byte changes remain. Unrelated work is
preserved; no consumer-repository mutation, push or tag is inferred.

## Execution Record

### 2026-11-27 — L1 committed; PLAN complete

The coherent migration landed on main as
`c28f603b35abb1dcd9fe368ba856663ad79eb46d`
(`refactor(layout): relocate mcp runtime and sdkbridge to mcp/`). Git records
all 26 Go files as renames. The staged 42-file local-identifier scan and
whitespace check passed before commit; the working tree was clean afterward.
The global prepare-commit-msg hook supplied the message through
`git commit --no-edit`.

All L01–L11 cases and completion criteria are satisfied by the pass-2 native,
race, actual-consumer, Make, artifact and deliberate-failure evidence below.
The canonical runtime import is `github.com/maccavelli/go-mcp-lib/mcp`;
`mcp/internal/sdkbridge/` owns the runtime-private conversions and shared
validation remains at root scope. No root Go source/test files, lost test
cases, unexplained generated-byte changes or stale active candidate locators
remain. No third source revision, consumer-repository mutation, push or tag
was performed. This completes the layout PLAN; the framework's R4/R5 and
Stage 7 remain separate unfinished work.

### 2026-11-27 — L1 pass 2 exit checks passed; coherent commit ready

Queued native Darwin ordinary generation returned `unchanged`. The combined
`make pre-add-check lint generate-check build-check docs-check` then passed
with one dispatcher invocation and one generation/check leaf: 70 runtime
tests in five packages, 346 generator tests in twelve packages, readonly
formatting, standalone vet, both lint owners, CLI status `ok`, complete
two-module/three-target compilation, record/link checks and all 37 Markdown
files. Normal Go operations forced CGO off.

The final L0-to-L1 byte audit passed. All 26 moved files retain their exact
contents apart from the five bridge imports, one external consumer import and
reviewed package-comment corrections. No root Go file remains. Every test
inventory row is retained; only 34 runtime and one bridge package identities
change. All 146 captured protected-path identities, including expected absence
of the generator sum file, are unchanged. All ten protocol files preserve
their SHA-256 and nanosecond mtimes through generation/check; the reviewed
manifest remains
`2d2d88d86a78b2806043c9b7a2580001ea83b952040ad57a1fc7f223500f0112`.

| Case | Observed exit evidence |
| :--- | :--- |
| L01 | Complete 26-file move, zero root Go files, five runtime packages and complete passing test accounting. |
| L02 | Actual external consumer builds with canonical capability imports; the legacy-root control is refused by the compiler under readonly policy. |
| L03 | A scratch root Go file produces its named layout failure. |
| L04 | Reviewed L0 preflight passed; destination, parent, source, inventory and alias plants fail while preserving all scratch bytes/link identities. |
| L05 | The real complete candidate passes every native lane and both module/target matrices; decoder, accessor and middleware bypasses remain caught at the moved paths. |
| L06 | Public API, middleware, connection-isolation and runtime feature tests remain required and pass; no new exported runtime identifier or SDK escape hatch. |
| L07 | Protected inputs/reviews and generated-byte inventories are unchanged; ordinary regeneration is unchanged and the actual CLI check is clean. |
| L08 | Four ordinary native lanes pass with CGO off; separately queued physical-Linux race tests pass in all four selected packages with CGO on. |
| L09 | Coherent prerequisite commit `d1276de4761aa77d6d3389370e6781c049753131` precedes the move. |
| L10 | Bridge ownership is runtime-only; forbidden edges and sibling private import fail; shared validation and corpus retain their identities. |
| L11 | Both modules and test inventories survive, library-command imports fail, and actual consumer imports use functional packages without new modules or stubs. |

All L01–L11 verification cases are satisfied. Stage the coherent migration
after the passing pre-add checks and identifier review, then commit on main
with `git commit --no-edit`. Record the resulting migration commit in the
completion entry before marking this PLAN complete. No push or tag is included.
R4/R5 and Stage 7 remain governed by the separate framework PLAN.

### 2026-11-27 — L1 pass 2: native, race and actual consumer checks passed

All four `scripts/qualify_layout.py` lanes completed with zero failures at
source identity
`8d5e56e27db148ad81e5fd770d1b20a503896c32f25b0797f8ab6ffcf62fffbf`:
Darwin/arm64, Windows/amd64, physical Linux/amd64 and WSL Linux/amd64.
Each accounted for 346 generator tests across twelve packages, including the
complete moved-runtime candidate and both modules' three-target source/test
compilation. All nine layout/control tests, seven Make cases, eight lint
plants and retained R2/R3 failure controls passed. Protected-baseline/final
and source-preservation checks passed on every lane. Ordinary Go execution
used CGO off. The source identity remained unchanged at continuation.

The separately admitted physical-Linux race command enabled CGO only for
`go test -race -vet=off -count=1 ./mcp ./mcp/internal/sdkbridge ./internal/validation ./codec`.
All four packages passed; the queue wrapper exited zero after 28 seconds.
WSL was explicitly excluded from race placement.

An additional scratch external module used the actual relocated checkout,
the pinned dependency graph and readonly/offline Go settings. Imports of
`/mcp`, `/codec` and `/protocol`, including `mcp.NewClient`, `mcp.NewServer`
and `mcp.Pins`, built successfully. Switching that actual consumer to the
legacy root import failed with the required readonly missing-package diagnostic.

The remaining exit work is unchanged CLI generation/check, final canonical
Make pre-add/lint/build/docs checks, protected-byte and staged identifier review,
and the coherent migration commit. No third source revision was needed.

### 2026-11-27 — L1 pass 2 native qualification started

All nine scratch layout/control tests now pass on native Windows. The exact
readonly legacy-import diagnostic is pinned, while canonical capability imports
build successfully. Actual Go metadata catches forbidden codec/protocol-to-runtime,
bridge-to-owner and library-to-command edges. The compiler accepts the owning
runtime's private import and rejects a sibling's import. Root files, missing
production/tests, owner changes, extra/malformed modules, the legacy bridge,
invalid metadata and changed protected contracts all fail their own controls.
Relocation conflicts and source/destination/parent aliases fail without changing
scratch contents or link identities. No control is skipped.

Freeze the moved source, updated candidate plants, canonical inventory reuse,
checker integration, current architecture and migration guidance. Run
`scripts/qualify_layout.py --allow-network --baseline d1276de4761aa77d6d3389370e6781c049753131`
on native Darwin/arm64, Windows/amd64, physical Linux/amd64 and WSL through their
queues. The driver retains all R2/R3 plants and complete candidate qualification,
and checks protected inputs against that prerequisite commit. Normal Go work
forces CGO off. After ordinary qualification, run the moved runtime/bridge,
shared validation and codec race tests on eligible physical Linux with CGO on,
then verify unchanged generation/check and complete pre-add before committing.
No third L1 source revision is preauthorized.

### 2026-11-27 — L1 pass 1 controls: strict diagnostic correction for pass 2

The 26-file move and reviewed import/locator/inventory updates landed locally.
Canonical `make lint` passed both modules and both analyzers. The native Windows
scratch layout suite passed seven of eight cases; the legacy consumer built
with the canonical imports and was correctly refused after switching to the
absent root package. The new test expected generic module-resolution wording,
but the enforced readonly policy produced:

```text
cannot find module providing package github.com/maccavelli/go-mcp-lib: import lookup disabled by -mod=readonly
```

This is a new fixture-expectation defect, not a pre-existing runtime failure.
Pass 2 requires that exact readonly diagnostic, the named package and nonzero
exit, while retaining the preceding canonical-consumer success. No assertion
or compiler failure is discarded. Finish explicit alias/conflict and missing
production/module controls before freezing the complete native matrix. The
existing third-build deviation rule still stops before pass 3. No L1 commit
or complete qualification is claimed for pass 1.

### 2026-11-27 — L0 passed; L1 pass 1 preparation started

The coherent prerequisite landed on main as
`d1276de4761aa77d6d3389370e6781c049753131`; the working tree was clean.
R3 pass 7 and installed Make/pre-add/lint, repeat generation, drift and
three-target compilation all passed before this inventory. No push occurred.

Re-grounded native Go metadata shows five runtime-module packages. The runtime
imports its bridge, protocol and shared validation; protocol also imports
shared validation. Bridge production importers remain `client.go`, `tools.go`,
`resources.go` and `middleware_bridge.go`; its additional test importer is
`client_boundary_test.go`. No sibling bridge owner was found. The runtime's
twelve test files have no relative fixture reads; validation's five test files
retain their existing corpus and schema paths. The standard-library-only
generator remains a separate module and has no sum file to relocate or create.

The reviewed root move is exactly:

```text
api_external_test.go      client.go                 client_boundary_test.go
doc.go                    doc_test.go               errors.go
errors_test.go            isolation_test.go         middleware.go
middleware_bridge.go      middleware_bridge_test.go middleware_test.go
output.go                 output_test.go            request.go
resources.go              resources_test.go         roundtrip_test.go
server.go                 server_test.go            session.go
subscribe_test.go         tools.go                  transport.go
```

Move those files under `mcp/`, and the bridge's `codec.go` and `codec_test.go`
from `internal/sdkbridge/` under `mcp/internal/sdkbridge/`. The destination does
not exist and no conflict is present. Capture protected protocol, schema,
validation, module/tool-pin and retained-review identities before moving.
Read-only Go-source searches of the four named reference repositories found
no legacy-root MCP import; this is not an assertion about unknown consumers.

L1 pass 1 will integrate layout/import checks into the canonical runtime
package inventory, reusing its single Go metadata query. Add scratch controls
for root files, dependency edges, consumer/internal-import rules, module/test
ownership and conflict-preserving preflight. Keep generated bytes and pins
unchanged. Existing runtime API and decoder/middleware plants must execute at
their moved locations. The approved third-build rule applies independently
to L1; no additional R3 pass is implied.

### 2026-11-27 — L0 prerequisite native qualification passed

The owner approved R3 pass 7, and all native qualification lanes passed the
literal hook-target repair. Its exact source identity and complete results
are recorded in the prerequisite PLAN. Verified publication and installed
checks are queued; the coherent prerequisite commit still precedes L0's
inventory re-grounding and any L1 source move.

### 2026-11-27 — L0 remains waiting: prerequisite hook discovery

R3 pass 6 completed on native Windows and WSL, but read-only inspection found
that the installed hook cannot discover the new shared Make target rule.
The prerequisite PLAN records the exact defect, partial qualification results
and requested next-pass allowance. Its unfinished Linux and queued macOS jobs
were stopped; neither is reported as qualified. The approved runtime-layout
decision is unchanged, but L0 still requires the coherent prerequisite commit.
No runtime or bridge file has moved, and no additional R3 source revision is
implied by the layout approval.

### 2026-11-27 — Prerequisite Make/lint repair A approved

The owner selected the focused Make foundation and lint repair. Its implementation
and remaining R3 pass-6 qualification are governed by the accepted amendment to
[0002-MADR-build-schema-driven-mcp-framework.md](0002-MADR-build-schema-driven-mcp-framework.md)
and its PLAN. The runtime move still follows the coherent prerequisite commit.
No additional layout decision or execution approval is needed after that exit.

### 2026-11-27 — L0 pass 1: prerequisite publication passed; commit blocked

R3 pass 5 and installed-generation checks passed, and normal regeneration kept
all protocol bytes and mtimes unchanged. The attempted prerequisite commit was
then blocked by the agent harness's fallback legacy lint gate. The repository
has no Makefile adapter, so its existing Python-only canonical precheck was not
selected. Missing export/method comments and an incorrectly placed generated
header need a producer repair and a new qualified generated identity.

The owner requested a passing `make lint` and a four-reference Makefile
assessment. The findings and proposed scope choices are recorded in
[0012-REPORT-local-gate-duplication-assessment.md](../reports/0012-REPORT-local-gate-duplication-assessment.md#makefile-reassessment--2026-11-27)
and the execution record of
[0002-PLAN-build-schema-driven-mcp-framework.md](0002-PLAN-build-schema-driven-mcp-framework.md).
The runtime-layout decision remains accepted and this PLAN remains approved.
L0's coherent-commit exit is unmet; L1 has not started. Preserve all staged work
and resolve the prerequisite deviation before moving any runtime or bridge file.

### 2026-11-27 — Approval recorded; L0 pass 1 started

The owner accepted the amended MADR and approved this PLAN's execution,
including L10/L11. Re-grounding found the root and generator layouts unchanged
and R2/R3 source still uncommitted at `5c00e3680393e960f74a8907b1d6d8eaaf5f7336`.
The already started R3 pass-5 native jobs have all completed successfully;
publication, post-installation checks and the coherent prerequisite commit
remain before L1. No runtime source or bridge has moved in this approval step.

### 2026-11-27 — Architecture A selected; execution approval pending

The owner selected A from the three-option prompt. At that point the MADR was
accepted and the PLAN remained proposed. No runtime source/test files moved. No new
exported identifier, consumer-repository change, push or tag is authorized.
The already approved R3 work remains governed by its own pair and pass allowance.

### 2026-11-27 — Four-reference reassessment; amended decision pending review

The owner requested renewed assessment of self-update, provider, TUI and ACP
and asked to update the MADR and wait. The revised proposal makes functional
ownership explicit, narrows the runtime-only bridge to `mcp/internal/`, retains
shared validation and command/module boundaries, and records ACP's root-package
exception. L10/L11 await review; the MADR and PLAN are proposed. This is a
documentation amendment, not an execution pass. No runtime/bridge files moved,
no new Go validation was implemented or run, and no source commit is included.
