# Local gate duplication and execution-cost assessment

Date: 2026-11-24; Makefile reassessment: 2026-11-27

Status: source assessment complete; consolidation proposed; implementation and
before/after timing pending

The [2026-11-27 Makefile reassessment](#makefile-reassessment--2026-11-27)
below records the current entry-point gap and the proposed implementation order.
The original observations retain their historical baseline.

## Finding

The local tooling contains repeated work at three levels: aggregate commands
invoke other aggregates, semantic validators launch tests already requested by
the outer runner, and related analysis tools inspect overlapping packages.
Standardize one explicit execution graph per repository. Make targets and hook
adapters select obligations from that graph; they do not independently orchestrate
another full set of checks. Execute each equivalent leaf once, retain its result,
and report every obligation that it satisfies.

This is part of the `local-cicd-scaffolding` contract in
[0002-MADR-build-schema-driven-mcp-framework.md](../decisions/0002-MADR-build-schema-driven-mcp-framework.md)
and [0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md).
This report assesses local tooling only. It neither changes upstream processes
nor implements or removes a gate in any assessed repository.

## Evidence and limits

The main Make targets, local runners, pre-add scripts, lint configurations and
relevant nested validators were read. The following revisions identify the
source baseline; source locators below are relative to the named repository.
No private machine identifiers are required to reproduce the findings.

| Repository | Baseline revision | Main entry points |
| :--- | :--- | :--- |
| go-selfupdate-lib | `43d33c7c780d18f4e2d7c9827a9f635c8fdc5384` | `Makefile`, `scripts/gate.sh`, `scripts/go-precheck.sh` |
| acp-go-sdk2 | `a6495a56ddf2c2503b4613a2f91fb9ca400ef61a` | `Makefile`, `tools/gates.py`, `tools/platforms.py`, `scripts/go-precheck.sh` |
| go-llmprovider-sdk | `cfc911d2c6ebf53f451ee05de2a982bc1d0c8f94` | `Makefile`, `tools/gate.py`, `scripts/go-precheck.sh`, `scripts/check_coverage.py`, `scripts/check_contract.py`, `scripts/test_gates.py` |
| go-tui-lib | `2928c3d3a83afae4acf050e8e8b74dfb272a6029` | `Makefile`, `tools/verify/gates.py`, `scripts/go-precheck.py`, `scripts/go-examples.py` |
| go-mcp-lib | `5bfdd9f5e4a6e9286cf593fdc1f3ccec60890532` | `scripts/gate.py`, `scripts/go-precheck.py`, `.golangci.yml` |

The LLM provider checkout has concurrent unrelated work, including edits to
`scripts/test_gates.py` and `tools/test_tools.py` during this investigation. Its
clean-copy/self-test finding was re-grounded against the immutable revision above.
Its main Makefile, runner, precheck and coverage/contract validators were unchanged
when inspected. Other repositories were read only; their work was preserved.

Counts below are static command requests on the successful default paths, not
observed subprocess timings or proof that every request does fresh compiler work.
They assume unchanged inputs, available tools, default selections and a normal
non-Windows session unless stated otherwise. Go/tool caches can reduce work inside
a repeated command; they do not eliminate orchestration or establish equivalent
coverage. No full suite, remote qualification or timing campaign was run for this
assessment. Numerical speedup and fleet-wide completeness remain unmeasured.

## Repository findings

### acp-go-sdk2

| ID | Observed call path and source | Assessment and proposed treatment |
| :--- | :--- | :--- |
| AC1 | `tools/gates.py:ORDER` includes `precheck` and `lint`; `scripts/go-precheck.sh:lint_module` checks root and generator for host and Windows; `Makefile:lint` repeats those four combinations. | Eight lint requests cover four distinct module/target combinations on a non-Windows host. Resolve aliases into four canonical leaves. On native Windows, host and explicit Windows collapse further to two distinct module/target combinations; do not schedule the same tuple twice. |
| AC2 | Precheck calls `tidy_module` for both modules; `tools/gates.py:Gates.tidy` calls `go mod tidy -diff` for both again. | Four tidy requests can become two, once the toolchain, environment and module inputs match. |
| AC3 | Precheck's no-argument file list includes tracked and untracked Go files; `Gates.newfiles` checks formatting in the untracked subset again. | Share one complete file inventory and formatter result. First fix the precheck's missing formatter-exit handling: `newfiles` currently has stricter failure handling through `fails_on_output`, so deleting it immediately would lose protection. |
| AC4 | `Gates.winvet` vets both modules; `tools/platforms.py:checks` vets selected process packages across eleven explicit tuples and lints Linux/FreeBSD subsets; precheck also vets the host. | Reconcile overlapping package coverage per exact tuple. Keep the additional platform coverage and build variants. A GOOS-only command is insufficient evidence for a declared GOOS/GOARCH obligation. |
| AC5 | `Gates.nojsonv2`, `Gates.race3` and `tools/floor.py` change experiment, repetition count or compiler. | These are distinct requirements, not removable duplicates. A shared race invocation may satisfy a smaller selection only after preserving required repetition and test inventory; the older-compiler lane must remain independent. |

The lint config also enables `modernize`, while `Gates.fix` invokes `go fix -diff`.
Treat this as possible analyzer overlap requiring pinned-version comparison, not
proof that either tool wholly replaces the other. Preserve each distinct finding
class until equivalence is demonstrated.

### go-llmprovider-sdk

| ID | Observed call path and source | Assessment and proposed treatment |
| :--- | :--- | :--- |
| LL1 | `tools/gate.py:CHECKS` invokes `make pre-add-check` then `make lint`. Precheck requests root host/Windows lint and nested-module host lint; Make requests root and nested host/Windows lint. | Seven lint requests cover four distinct combinations on a non-Windows host. Consolidate three repeated requests, preserving `live_gateways` on root lint and the nested module's different tag selection. |
| LL2 | Precheck runs root `go mod tidy -diff`; `CHECKS` independently runs `tidy-diff`. | Use one root tidy result. Keep the nested module's separate tidy requirement. |
| LL3 | Precheck vets and race-tests `authcontract`; `check_contract.py:vet` and `tests_and_coverage` repeat vet and add a fresh race/coverage run. | Split contract validation from test production. One qualifying nested-module vet result and one qualifying race/coverage result can feed all consumers; an old cached precheck test is not sufficient evidence. Keep dependency, API and schema checks. |
| LL4 | Root precheck race tests, outer `test-race-cover`, `test-shuffle`, `test-repeat`, and `check_coverage.py:main` all request root suite execution. The coverage checker launches another importer test run for the support package declared in `coverage-floors.txt`. | Collect reusable JSON results and coverage in the required test runs, then evaluate floors without launching a hidden suite. Retain race, repetition and shuffle semantics. Support-package importer coverage is a different measurement and must be preserved explicitly. |
| LL5 | `scripts/test_gates.py:GATES` lists 13 validators; `GateSelfTest.test_clean_copy_passes_every_gate` runs all of them, and the outer runner also invokes those gates separately. Broken-input tests invoke validators on their own clones. | Reuse a matching clean baseline or define one shared baseline fixture per validator. Do not deduplicate mutant inputs. `copy_tree` deletes nested-module tags, so its history-sensitive baseline is not automatically equivalent to the outer runner's baseline. |
| LL6 | `tools/gate.py:markdown_checks` includes the entire `gate-selftest`; that suite contains code, coverage, API, contract and nested-precheck tests. | A Markdown-only entry point currently reaches substantial Go work through its self-tests. Separate validators from their self-test groups and use a proven input-dependency map. Preserve tests affected by Markdown inputs and all required failure controls; a filename suffix alone cannot classify an embedded document as safe. |

The outer runner's six default/live-tag vet selections and the three-repeat
`TestWireGoldens` selection have distinct declared purposes. Do not drop them
merely because another command mentions `./...`. The support-package coverage
run measures `llmprovider/internal/wirecase` through provider tests with a specific
`-coverpkg`; ordinary per-package coverage is not equivalent.

### go-tui-lib

| ID | Observed call path and source | Assessment and proposed treatment |
| :--- | :--- | :--- |
| TU1 | `tools/verify/gates.py:default_gates` already marks `pre-add-check`, `lint`, `workspace-mode`, `tidy`, `modules`, `apicheck` and `examples` non-default because `release-check` reaches them. | Retain this explicit ownership improvement; turn it into a graph so selecting multiple aliases cannot reintroduce repeats. The explicit `modernize` entry is necessary because release-check does not run it. |
| TU2 | Default `vuln` runs `make vuln`; `release-check` calls `scripts/go-precheck.py:check_module`, which calls `vuln` again. The latter permits an unreachable database to be skipped. | Two scans per checked module on successful paths. Replace both with one strict owner-managed scan/evidence result per required context. Preserve the strict failure contract; removing the outer scan alone would weaken it. |
| TU3 | `Makefile:lint` depends on `modernize`; separate invocations of `make modernize` and `make lint` repeat modernization. `release-check` followed by `make docs-check` repeats guide-excerpt checks through `go-examples.py`. | Canonicalize shared prerequisites across entry points. A single Make invocation can deduplicate its prerequisites, but separate wrapper processes do not share that graph. Reuse guide validation and a matching `TestDefaultsInventory` result when actually executed. |
| TU4 | Release-check tests modules with `GOWORK=off` and workspace mode; outer gates add race, shuffled repeated tests and `LC_ALL=C`. | Keep distinct environments. At this revision `go.work` contains only `use .`, but do not remove workspace acceptance merely because the present graph is small; prove equivalence or retain its separate obligation. |
| TU5 | `Makefile:test` enumerates modules; `test-sum` directly runs root `./...`. | These are alternative presentations, not two required suites. Make both select the same complete test graph and format one event stream. Do not use root-only reporting as a replacement for future nested-module coverage. |

The default runner has many checks but does not directly select every Make
documentation target. Build the obligation inventory from accepted requirements
as well as the current runner; deleting duplicates alone cannot prove completeness.
OS-specific script tests remain separate portability obligations when their
behavior depends on filesystem, subprocess or shell semantics.

### go-selfupdate-lib

| ID | Observed call path and source | Assessment and proposed treatment |
| :--- | :--- | :--- |
| SU1 | `scripts/gate.sh:164` onward runs formatting, `make lint`, vet, race, shuffle/repeat, tidy, API, fuzz and vulnerability checks. `make pre-add-check` separately runs formatting, three-target lint, vet/test and vulnerability checks. | The full gate does not itself call pre-add. Duplication occurs when a caller runs both unchanged paths, or a later hook repeats the precheck. Share leaf results across those entry points with exact input identity; do not claim every gate invocation currently duplicates them. |
| SU2 | The gate has `links` and `arch`; `Makefile:docs-check` runs the same link and architecture checks and its script tests. `script_tests` already runs the documentation script's test harness. | A session invoking both aggregates repeats validators and the harness. Use shared leaves while retaining the script harness as a distinct check of the validator. |
| SU3 | The gate's `ids` scans changed/untracked files with a private deny list; Make's `disclosure-check` scans tracked files with optional private rules. | These are overlapping scans with different scopes and failure contracts. One shared file read can serve multiple predicates, but the changed/untracked set, tracked set and mandatory private policy must all remain accounted for. |
| SU4 | `scripts/gate.sh` does not directly invoke Make's records or Markdown lint targets. The fuzz wrapper already uses `-run '^$'` for each campaign. | Add missing required obligations to the graph rather than equating the existing aggregate with complete acceptance. Preserve the fuzz wrapper's avoidance of ordinary-test replay; do not reduce campaign targets or durations. |

Extra FreeBSD/OpenBSD/Linux-386 vet checks are existing local obligations and are
not replaced by this project's three native platform requirements. Service-manager
and other product-specific checks stay in the reference repository's profile.

### go-mcp-lib

There is no root Makefile yet. The existing modes in `scripts/gate.py` expose
overlap that a naive Make wrapper would multiply:

| ID | Observed call path and source | Assessment and proposed treatment |
| :--- | :--- | :--- |
| MC1 | `mode_preadd` runs tests in both modules; `mode_unit` runs both again; `mode_generate_check` runs the generator suite again. | The proposed sequence `pre-add`, `unit`, `generate-check` requests root tests twice and generator tests three times. Use one complete qualifying producer per module/context, with several semantic result consumers. |
| MC2 | `mode_unit` lints both modules; `mode_generate_check` lints the generator again; `mode_api` lints the root again. | Four lint requests can become two per unchanged build context. API and generation validators retain their own distinct checks. |
| MC3 | Pre-add builds both modules; `mode_api` builds the root again. | Reuse the root build result. Keep standalone build/consumer obligations: compilation performed by a test is not automatically proof of all non-test or distributable artifacts. |
| MC4 | Pre-add currently calls mutating `go fmt`; test evidence and generated-code lint have audit defects. | R1 and the accepted repair contract must fix these first. Caching or consolidating an unsound gate would only make a false pass faster. |

The MC counts describe a possible composition of current modes, not an existing
successful preflight. Full qualification remains incomplete as recorded in
[0011-REPORT-pre-stage-7-completion-audit.md](0011-REPORT-pre-stage-7-completion-audit.md).

## Analyzer overlap is not tool equivalence

All four reference lint configs enable `govet` and Staticcheck. A standalone
`go vet` adds another analysis entry point, and ordinary `go test` invokes a
curated vet subset while building tests. The Go command documents this behavior
and the `-vet` selector. [Go command reference](https://pkg.go.dev/cmd/go).

Keep an explicit analyzer ownership ledger: analyzer/version, package universe,
test/generated-file inclusion, tags, tuple, compiler and exclusions. A canonical
vet leaf may satisfy repeated requests only after proving those inputs and
diagnostics match. Do not simply delete standalone vet because the lint config
says `govet`: the latter is described as approximately equivalent, and its
analyzers/settings can differ. Any relocation of an analyzer out of a redundant
entry point must preserve its accepted coverage and failure control.
[golangci-lint quick start](https://golangci-lint.run/docs/welcome/quick-start/).

The documented `golangci-lint run` command also executes configured formatters
as checks, without rewriting source. All four reference configurations enable
`gofmt` and `goimports`, creating another overlap with standalone `gofmt -l`.
Resolve that ownership using actual file coverage: formatter exclusions and
platform-selected files can differ from an explicit all-files scan. Keep required
generated/untracked-file and import-order checks; qualify the pinned binary and
missing-file/parser failure behavior before consolidation. The separate `fmt`
command and `run --fix` mutate code and are not readiness checks.
[golangci-lint command reference](https://golangci-lint.run/docs/configuration/cli/#run).

Do not equate lint with module tidiness, public API compatibility, schema behavior,
dependency closure, fixture privacy or runtime tests. These predicates differ
even if they read some of the same files. Similarly, records, links, Markdown
style, documentation/code parity and evidence-claim checks need separate results;
they can share a file inventory and parsing where their semantics permit it.

## Recommended execution contract

### One graph, several entry points

Keep a thin Makefile, a portable Python runner and a versioned repository profile.
Declare the graph before execution and reject cycles, unknown obligations,
duplicate definitions and required checks with no owner. Composite targets expand
to shared leaves; leaf validators do not invoke other aggregates or launch hidden
suites. A standalone target requests the same prerequisites as full preflight.
Legacy target aliases remain useful interfaces, not independent implementations.

| Check family | Producer and consumers | Multiplicity |
| :--- | :--- | :--- |
| Inventory and document parsing | One immutable file/module inventory; records, links, style, parity and disclosure consume the appropriate scoped views. | Once per relevant snapshot/input context; retain platform-sensitive checker tests. |
| Formatting | One non-mutating file-set check; mutation is a separate explicit action. | Once per formatter version/config/file set. |
| Static analysis | One owner per required analyzer and build context; lint, vet, modernization and dependency predicates retain separate findings. | Once per qualified module/package/tuple/toolchain/tag context. |
| Tests and coverage | Canonical test producers emit complete JSON events and coverage artifacts; unit summaries, coverage floors and product validators consume them. | Once per required execution variant; repetition count is part of the variant. |
| Build, generation, API and examples | Explicit artifact or comparison producer; independent semantic checks consume identified artifacts and baselines. | Once per exact inputs, output purpose and required build context. |
| Gate self-tests | Shared preparation and matching clean baseline; isolated broken fixtures with their own results. | Once per required fixture/context; never reuse a clean result for a mutation. |
| Local security evidence | Existing owner-managed producer; all local readiness consumers refer to its current successful result. | Preserve fresh-per-push requirements and each required scan scope. |
| Native qualification | Fleet queues run the immutable snapshot; validators inspect recorded outcomes. | Preserve all four environments, both compilers and modules required by this profile, plus separate native race runs. |

For platform-independent checks, choose one execution owner rather than copying
the entire gate list onto every native runner. Classify this deliberately: document
path checks, Python tooling and filesystem behavior can depend on the host and
may need native portability tests. Cross-target analysis does not replace native
execution. WSL and standalone Linux remain separate required environments.

### Identity and reuse

Use a semantic check ID and complete input fingerprint, not the command's display
name or a success stamp. Include the source snapshot, module/package set, inputs,
policy/config, tool binaries/versions, resolved dependencies, compiler, tuple,
tags/experiments, CGO/workspace mode, relevant environment, test selection/count/
seed, coverage mode and scope, fixtures, required history/API baseline and output
purpose. Runtime evidence also binds the required runner environment.

Coalesce identical pending requests under one lock/future. Many aliases may consume
one successful result; none may turn a failure, timeout, cancellation, missing
case or skip into success. Explain in the report which producer satisfied each
obligation and why a result was reused. Partial selections do not satisfy broader
ones without explicit, verified coverage reconciliation.

Within a qualification run, reuse the one freshly produced result. Across runs,
reuse only under the repository's approved receipt policy, with matching inputs
and freshness; do not substitute the implicit Go test cache for execution evidence.
For this repository, exact outgoing-commit receipts still expire at one hour and
local security evidence remains fresh per push. A pre-commit working-tree result
does not automatically certify a later commit. Prove identity and satisfy any
history/commit-sensitive inputs before reuse; otherwise rerun the affected leaf.

Share verified dependency/tool downloads and compilation caches. Resolve pinned
tools once during preparation, not through repeated installation/discovery in
each validator. Reuse module metadata per exact context where output semantics
match. Do not trade away independent-module, native OS, race or CGO-free coverage.

### Change selection and costs

Separate quick edit feedback, required pre-add validation and complete pre-push
qualification through explicit selectors over the same graph. Do not weaken an
existing pre-add policy or silently move its checks to a later phase. Changes to
tools, configs, generated inputs, fixtures, build tags, module files and embedded
documents invalidate their dependent checks. Unknown dependencies select the
complete applicable graph. Selective execution needs demonstrated dependency
coverage, especially for subprocess builds and non-Go inputs.

Record leaf wall time, queue wait, setup/transfer time, invocations, reuse hits,
reason for every repeated context, log/artifact identities and overall critical
path. Compare baseline and candidate on the same source and qualified hosts,
with explicit cold/warm cache conditions. Native runtime checks and benchmarks
retain their existing resource and correctness requirements. No target time or
percentage improvement is asserted without measurements.

## Adoption priority and required proof

1. First expose and deduplicate identical aggregate leaves: AC1/AC2, LL1/LL2,
   TU2 with strict failure semantics, and MC1–MC3 after the existing correctness
   repairs. These have the clearest command-count savings.
2. Separate expensive producers from validators: coverage artifacts, nested-module
   contract inputs, module/file inventories and test-event consumers. Preserve
   every required measurement and fail if a required artifact is missing.
3. Isolate self-test groups and share matching preparation/baselines. Keep all
   deliberate-failure controls; qualify any selective execution with conservative
   invalidation and actual planted regressions.
4. Compare overlapping analyzers on the pinned versions before changing their
   execution owners. Until equivalence is established, record the reason for the
   remaining overlap rather than silently relaxing coverage.
5. Measure the resulting graph. Acceptance needs unchanged required outcomes,
   caught plants, zero unexplained duplicate leaf executions, complete native
   evidence and a measured cost comparison. Existing repository-specific records
   govern implementation changes in each reference repository.

Required tests include two aliases reaching one leaf; concurrent duplicate requests;
a changed build tag/compiler/CGO mode/fixture forcing a distinct run; a mutated
input never borrowing clean evidence; one shared failure failing every dependent
aggregate; a missing nested module; stale receipts; and a falsely classified
documentation change that must invalidate a Go check. Every new deduplication or
selection gate must reject a deliberately broken scratch case before acceptance.

The assessment completes the source-level analysis. It does not establish that
the proposed consolidation has been implemented, that current reference gates
are complete, or that a particular wall-clock saving has been achieved.

## Makefile reassessment — 2026-11-27

The owner requested `make lint` to pass and asked what the other repositories'
Makefiles establish as the standard to implement here. Their Makefiles, precheck
wrappers and relevant linter configuration were inspected read-only. No reference
Makefile or source was changed, and no reference suite was run. The source
baselines for self-update, TUI and ACP remain those above; the provider baseline
is now `223bb8dc597b37a7a3f0fcea70f85742ed03580b`, with concurrent unrelated
working-tree changes. MCP HEAD is
`ff6d700` with the approved R2/R3 implementation staged but uncommitted.

### What the four Makefiles establish

| Repository | Useful precedent | Differences and costs to account for |
| :--- | :--- | :--- |
| go-selfupdate-lib | Named formatting, vet, lint, test, fuzz, API, docs, disclosure and pre-add entry points; a full local gate; three-GOOS lint. | Much orchestration is shell; full gate and precheck are distinct overlapping entry points. GOOS-only lint leaves GOARCH inherited. Library release/signing/service-manager tooling is product-specific. |
| go-llmprovider-sdk | Root and `authcontract` tested separately; explicit dependency, coverage, API, schema, generated-output, fixture, documentation and self-test targets. | Root host/Windows and nested-module selection differ by target. Some normal commands inherit CGO; precheck race is mixed into the default path. The vulnerability-tool pin is parsed from a forge configuration file. These details do not meet MCP's local-only, explicit-environment contract. |
| go-tui-lib | Thin public targets backed by Python; explicit per-module work; read-only modernization and tidy checks; examples and API checks. | Its Makefile still contains shell loops; lint has a modernization prerequisite, while other callers also request modernization. Precheck adds workspace-mode tests and repeated security scans. These are not all interchangeable with MCP's two independent modules. |
| acp-go-sdk2 | Explicit root and generator-module coverage; uncached/shuffled tests; generator/tool self-tests; generation, API, fuzz, docs and disclosure commands. | Shell/platform bootstrap and product release/download recipes are specific to ACP. Host-plus-Windows lint repeats Windows on a Windows host and does not set an architecture. Its vulnerability pin also comes from forge configuration. |

The shared standard is the stable command interface and the obligations behind
it, not an identical file copied from one repository. MCP should use Make only
for dispatch, with Python 3 standard-library orchestration, an explicit two-module
inventory and tool pins in `tools/versions.json`. Documentation-only targets
must not invoke Go merely while Make expands variables.

### Why the current commit reached legacy golint

MCP has `scripts/go-precheck.py`, which delegates to the pinned canonical gate,
but has no root Makefile and no executable `scripts/go-precheck.sh` adapter.
The installed shared agent precommit hook selects, in order:

1. An executable `scripts/go-precheck.sh`.
2. A literal `pre-add-check:` Makefile target.
3. Its generic per-file `gofmt` / legacy `golint` fallback.

The failed R2/R3 commit therefore used the fallback. The reference repositories
provide the first adapter; MCP can provide the second without a new shell script
or a global-hook edit. `make pre-add-check` must reach the same Python precheck
and preserve all its required checks. Supplying an empty or weaker target would
be a bypass, not a repair.

The fallback also exposed real documentation defects. Examples are missing
comments on `Options` at `cmd/mcpgen/internal/acquire/acquisition.go:27`, on
publication/ownership exports, and on generated methods such as
`AudioContent.MarshalJSONTo` at `protocol/owned.go:17`. The emitter's blank
`embed` import and keyword-class constant block already existed in HEAD; the
new R3 exports and generated method surface add other findings. In addition,
`protocol/owned.go:1` currently starts `package // Code generated ...`, placing
the marker after the package keyword rather than in the standard leading
comment. The current header test only searches for a substring and misses that
placement defect.

Correct those producers and add explicit modern documentation/naming checks to
the pinned linter, with broken-input controls. Keep generated source analyzed;
fixing the marker must not become a way to suppress missing documentation.
Do not hand-edit manifest-owned output or change a global hook to evade findings.
Emitter/header/comment repairs require a freshly rendered, reviewed and qualified
output inventory before replacing the current reviewed generated bytes.

### Recommended target contract for MCP

| Target family | What to implement | Current backing and required work |
| :--- | :--- | :--- |
| `help`, `setup` | Stable help and explicit pinned preparation; network acquisition stays opt-in. | Reuse `gate.py setup`; Make performs no eager Go/shell discovery. |
| `fmt-check`, `lint`, `vet`, `test` | Independent selectors over one runner; both modules, generated and untracked applicable sources; no writes by validation. | Existing precheck owns these leaves but lacks separate clean selectors. Keep standalone vet as owner and `go test -vet=off -count=1 -json`; configured lint does not repeat govet. Preserve event/test-inventory accounting. |
| `pre-add-check` | The canonical local precheck entry point used by Make and the installed agent hook. | Delegate to `scripts/go-precheck.py`; do not recreate checks in the Makefile or add a shell implementation. |
| `check-records`, `markdownlint`, `docs-check` | Record/link checks, Markdown style, and any actual code/documentation parity requirements. | Reuse `check_records.py --check-all` for the links it already owns and the pinned Markdown tool. Do not add a second generic link scan over the same scope. Separate absent parity checks from existing record/link coverage. |
| `generate-check`, `build-check` | Read-only generated drift/ownership check and source/test compilation for both modules on all three target tuples. | R3's CLI `check` now performs the complete candidate compile matrix. Let one result satisfy its compile obligations instead of launching the same matrix again under another alias. |
| `gate-selftest`, `inventory-check` | Checker regression tests, deliberate failure controls and acceptance/test ownership reconciliation. | Existing Python tests, R2/R3 plants and inventories provide owners. Add Make dispatch and missing Make/lint/header controls; do not hide another clean full suite inside every validator. |
| `race` | Separate, explicitly CGO-enabled test variant on eligible queued native hosts. | Add a selector and complete test accounting; retain accepted native obligations. WSL remains outside race admission. Race never replaces CGO-disabled ordinary tests/builds. |
| `modernize`, `tidy-check` | Read-only Go modernization and module tidiness, per independent module. | Add explicit leaves if adopted; prove analyzer overlap before removing an existing owner. No `go fix` rewrite or `go mod tidy` mutation in readiness commands. |
| `api-check`, `coverage-check`, `fuzz` | Owned public API/consumer compatibility, recorded coverage obligations and bounded declared fuzz campaigns. | Bind accepted R4/R5 and later-stage requirements. Current `api` readiness remains incomplete; no foreign repository's tag baseline, numeric coverage floor or fuzz count is imported without its own rationale. |
| `vuln`, `disclosure-check` | Strict local security evidence through the owner's existing tooling and freshness policy. | Keep pins and policy local and forge-independent. A required missing, unavailable or stale scan is not a successful scan. Do not copy private host/account rules into the repository. |
| `preflight` | Complete accepted local pre-push graph, native evidence and current security evidence, with every failure propagated. | Finish graph/evidence integration before claiming readiness. Existing incomplete wire/API/product qualification remains a failure until its planned work is complete. This target performs no push, tag, release or forge operation. |

`fmt` and `tidy`, if exposed, are explicitly mutating maintenance commands and
do not stand in for their read-only checks. `test-sum` is an alternative renderer
of the same complete event stream, not a second mandatory suite. A coverage
validator consumes the required test artifact rather than starting a hidden run.

### Coverage and duplication rules

- Model targets as exact `darwin/arm64`, `linux/amd64`, `windows/amd64` tuples;
  assigning GOOS alone does not set the required architecture. Lint each
  required module/tuple once, either on its native runner or through one
  explicitly selected cross-target analysis owner. Do not repeat the whole
  cross-target lint matrix on every native runner.
- Keep CGO off and independent-module readonly/offline environments for normal
  analysis, tests and builds. Only an explicit race command enables CGO.
- Heavy Make targets dispatch through the fleet selection/queue mechanism.
  Named machines and credentials remain operator-owned, outside committed files.
  Physical Linux, native Windows, Darwin and WSL evidence retain their distinct
  accepted roles; cross-compilation does not execute a native test obligation.
- Share exact successful leaves within a qualification invocation. Source,
  configuration, toolchain, tuple, tags, CGO mode, fixtures, history or required
  test selection changes invalidate their dependent results. An old success
  stamp or ordinary Go test cache does not certify a new input.
- Keep the existing vet/govet split; use modern revive documentation/naming
  rules to preserve the legacy fallback's relevant findings. Confirm each rule
  with its own broken scratch input. Do not enable an entire foreign lint
  profile, with its product-specific exclusions, by copying the file.
- `preflight` selects leaves once; it must not run `pre-add-check`, then `test`,
  then `lint` as three independent aggregates. Generator candidate runtime
  checks also need explicit ownership so a live runtime suite is not replayed
  unnecessarily against equivalent bytes.
- Tests of checker failure propagation and missing input are distinct from
  ordinary validators. Retain all negative controls while avoiding repeated
  identical clean baselines.

### Proposed implementation order and review boundary

1. Establish the common Make entry points for already implemented validation
   leaves, including a real `lint` and the `pre-add-check` hook adapter. Add the
   Make selector/failure-propagation controls without changing global hooks.
2. Repair handwritten export documentation and generator header/method comments,
   preserve the checks with modern analyzer rules, and render/qualify the new
   generated identity before the blocked R2/R3 commit. Preserve the pinned raw
   schema bytes and all input/profile/SDK identities.
3. Complete that coherent prerequisite commit, then execute the already approved
   runtime relocation under
   [0014-PLAN-organize-mcp-runtime-package.md](../decisions/0014-PLAN-organize-mcp-runtime-package.md).
4. Complete the broader local preflight graph, artifact consumption, security
   integration and remaining stage-specific gates as their accepted product work
   becomes available. Measure repeated-leaf elimination and wall time before
   claiming a speedup or complete acceptance.

The linked framework PLAN records the blocked commit and proposed scope choices.
This reassessment does not install a Makefile, change a linter, repair the
generator or authorize edits in the reference repositories. It establishes what
the immediate repair and the complete local standard each require.

### Execution refinement after option A selection

The owner selected A and implementation is now governed by the framework pair.
The Make interface and shared Python selectors have been written; final native
qualification and the phase commit remain pending. Broken-input probes exposed
a generated-file skip inside the pinned golangci Revive adapter. Standalone
Revive 1.17.1, the same dependency version, now owns the four documentation and
naming rules with generated headers explicitly inspected. Golangci retains the
other analyzers, avoiding duplicate rule execution. All four generated and
handwritten violations were caught through actual Make on a native Windows
scratch snapshot; valid controls passed.

The new tool pin is isolated in `tools/documentation-lint.json`; the existing
generation-profile tool manifest remains byte-identical. Parser-based checks
also rejected the old malformed generated marker and a missing method comment.
The new candidate's API/bindings/coverage/schema/profile identities are unchanged;
only generated Go headers/comments and their ownership hashes change. These
preparation results do not claim the pending native phase exit or full preflight.
