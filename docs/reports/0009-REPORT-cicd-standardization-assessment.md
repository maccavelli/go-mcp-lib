# CI/CD standardization assessment

Date: 2026-11-24

Status: complete assessment; owner decisions recorded; implementation pending

## Owner review outcome — 2026-11-24

The owner selected D1 A (extend the existing Python gates behind a thin Makefile)
and D3 C (one-hour exact-commit receipts, with fresh local vulnerability evidence
for each push). For D2, the owner reports changing the repository to public and
explicitly removes GitHub enforcement from scope. Branch protection, rulesets,
required-check configuration, plan upgrades and visibility management are not
implementation requirements or blockers. GitHub CI on all three targets remains
in scope. Disclosure guards and security scans remain owner-managed locally;
this work does not install/reconfigure them or add GitHub security-scan jobs.

These rulings supersede the original recommendations where noted below. Historical
API findings remain observations at assessment time. Accepted policy and Stage 9
acceptance cases are recorded in the existing framework MADR/PLAN pair. This review
does not itself start Stage 9 implementation.

## Recommendation

Standardize the **gate contract**, with a thin Makefile calling a single
Python 3.12+ standard-library orchestrator. Use that same implementation on
the local fleet and GitHub-hosted runners. ACP supplies the strongest starting
matrix; combine it with TUI's module-inventory checks, LLM provider's isolated
verification copies, and selfupdate's deliberately broken gate controls.
Copying any one repository unchanged would leave important gaps.

For this repository, extend `scripts/gate.py` rather than add a competing gate
runner. Require native Linux/amd64, Windows/amd64 and Darwin/arm64 tests with
`CGO_ENABLED=0`, on both Go versions already specified by the framework plan.
Use separate race-test invocations with CGO enabled for instrumentation. Require
successful local fleet qualification of the exact outgoing commit before push,
then independently repeat the required checks on GitHub.

This report records the assessment and subsequent owner selections. The existing
[0002-MADR-build-schema-driven-mcp-framework.md](../decisions/0002-MADR-build-schema-driven-mcp-framework.md)
and [0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md)
remain authoritative and now record the selected Stage 9 policy. Existing test
cases and performance thresholds remain in force. Implementation remains pending.

## Evidence and limits

The assessment inspected tracked workflows, Makefiles, local gate/remote tooling,
module and tool pins, the installed pre-push entry point, and read-only GitHub
API responses. Reference sources below are repository-relative paths at these
revisions, not links into an assumed local checkout layout.

| Repository | Inspected revision | Relevant source |
| :--- | :--- | :--- |
| go-selfupdate-lib | `43d33c7c780d18f4e2d7c9827a9f635c8fdc5384` | `.github/workflows/ci.yml`, `Makefile`, `scripts/gate.sh` |
| acp-go-sdk2 | `a6495a56ddf2c2503b4613a2f91fb9ca400ef61a` | `.github/workflows/ci.yml`, `Makefile`, `tools/gates.py`, `tools/platforms.py` |
| go-llmprovider-sdk | `1d9dbe794e8b3edc62bdeb7d353a825f03c947b0` | `.github/workflows/ci.yml`, `Makefile`, `tools/gate.py` |
| go-tui-lib | `2928c3d3a83afae4acf050e8e8b74dfb272a6029` | `.github/workflows/ci.yml`, `Makefile`, `tools/verify/gates.py`, `tools/verify/remote.py` |
| go-mcp-lib | `c11100f35f1f36bdf5bcc0a63112b1d5a59402f4` | `go.mod`, `cmd/mcpgen/go.mod`, `tools/versions.json`, `scripts/gate.py`, `scripts/go-precheck.py` |

go-llmprovider-sdk had unrelated uncommitted work; the assessed CI files were
unchanged. go-mcp-lib advanced during the investigation and has concurrent
body-size policy edits. The snapshot above, rather than its earlier scaffold-only
state, is the assessment baseline. Those edits were preserved.

No reference repository's full suite was rerun for this assessment. Successful
GitHub runs establish that the configured checks ran successfully at the cited
revision; they do not establish unconfigured CGO-free coverage, future reliability,
or enforcement before push. There was no controlled timing comparison, so the
efficiency recommendation is structural, not a measured speed ranking.

### Observed GitHub results and enforcement

Read-only API sampling returned these completed successful CI runs:

| Repository | Run ID | Event and revision |
| :--- | :--- | :--- |
| go-selfupdate-lib | `44355091005` | Push; inspected revision |
| acp-go-sdk2 | `45236923503` | Scheduled; inspected revision |
| go-llmprovider-sdk | `45455873223` | Push; `33e6f09a65e7b95c9a44840a60c21f0c5b3e9748`; the newer inspected revision had a successful Docs run |
| go-tui-lib | `44355180107` | Push; inspected revision |
| go-mcp-lib | None returned | No Actions runs returned by the API |

For all four reference repositories, `rules/branches/main` returned an empty
list and `branches/main/protection` returned HTTP 404, "Branch not protected".
Thus their visible main-branch configuration does not require their green CI
checks before integration. The owner subsequently excluded GitHub enforcement from this project's scope.

At assessment time, go-mcp-lib was private and both protection queries returned
HTTP 403 with a requirement to upgrade to GitHub Pro or make the repository public.
The owner subsequently reported making it public. No visibility or protection
change was made by this assessment; enforcement is no longer an acceptance item.
The original response matched GitHub's documented private-repository restrictions.
[GitHub protected branches](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches).

The installed global `pre-push` entry point delegates only to the GitHub disclosure
guard. It does not invoke this repository's tests or gate runner. A Makefile alone
would therefore not ensure checks run before an ordinary `git push`.

## Comparison

| Reference | Strongest reusable practice | Gap or adaptation needed here |
| :--- | :--- | :--- |
| ACP | Three native OS runners times minimum/preferred Go versions; both modules; explicit floor toolchain verification; Windows race testing; read-only workflow permissions and nonpersistent checkout credentials | Main tests do not explicitly force CGO off. Its Go 1.26 floor and `nojsonv2` compatibility lane are not this project's contract. Some local lint selections name only GOOS. |
| TUI | Checked module inventory feeds the matrix; independent-module tests plus workspace checks; Python gate reporting and remote snapshot transport; avoids some duplicate checks | This repo has no workspace contract: retain `GOWORK=off`. No separate minimum-version matrix. Main workflow lacks an explicit top-level permissions block; main checkout retains default credentials. Local diagnostic skip modes cannot issue release/pre-push success. |
| LLM provider | Thin named Make targets for semantic gates; isolated scratch copy for local checks; gate self-tests; nested-module contract checks; release job waits for validation and alone gets write permission | Main tests do not explicitly force CGO off; race only on Linux; no separate floor lane. Several cross-target checks set GOOS without GOARCH. Its load-average polling is not the fleet's exclusive queue. |
| Selfupdate | Deliberate-failure controls for gate scripts and release hand-off; native OS integration requirements; separate build/rehearsal/verification of release artifacts; correct handling of formatter failure | Service-manager/install/signing jobs are product-specific. Local `GATE_SKIP` can yield overall success. Some lint loops omit GOARCH. Its workflow explicitly documents API self-comparison on a tag. Shell/PyYAML tooling does not fit this repo's Python-stdlib orchestration policy. |

Evidence locations: ACP `.github/workflows/ci.yml:57` (matrix), `:117` (both
modules), `:126` (race); TUI `.github/workflows/ci.yml:60` (module/OS matrix),
`:76` (tests), `:79` (race); LLM provider `.github/workflows/ci.yml:33` (matrix),
`:48` (tests), `:51` (race), `:78` (release); selfupdate
`.github/workflows/ci.yml:53` (tests), `:147` (race), `:200` (static checks),
`:209` (API limitation), `:328` (floor), `:390` (release rehearsal).

Several workflows use `test -z "$(gofmt -l .)"`: a formatter that fails without
stdout can be mistaken for a clean tree. Selfupdate already fixed that pattern.
The proposed Python gate must inspect the formatter's exit status and its output
independently. Existing workflow path filters also require care when adding
required checks: skipped workflows can leave checks pending. GitHub documents
this interaction explicitly.
[Required status check troubleshooting](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/troubleshooting-required-status-checks).

### Current go-mcp-lib gaps

These are source findings at the assessed snapshot, not claims that unimplemented
Stage 9 work has already failed a live run.

| Finding | Evidence | Proposed treatment |
| :--- | :--- | :--- |
| No root Makefile or `.github` workflow tree | Filesystem and tracked inventory | Add a thin Makefile and `.github/workflows/verify.yml` under approved Stage 9 work. |
| Gate environment does not explicitly set CGO, target tuple or readonly module flags | `scripts/gate.py:31`; `scripts/go-precheck.py:24` | Construct and validate a shared environment for every subprocess; do not rely on parent defaults. |
| Pre-add modifies source while reporting a cleanliness check | `scripts/go-precheck.py:61` calls `go fmt ./...` | Use non-mutating formatter inspection; keep explicit `make fmt` separate. |
| Pre-add does not invoke golangci-lint | Whole `scripts/go-precheck.py`; `PRE_ADD_CASES` at line 35 | Bring pre-add into the approved lint contract, including generated files and relevant build variants. |
| Test evidence can be cached and does not reconcile required test IDs | `scripts/gate.py:55` and `:154` | Use `-count=1`, record shuffle seed, reconcile executed/pass/skip outcomes with required inventory. A module containing some tests is insufficient. |
| API gate currently checks generated presence, manifest, build and lint | `scripts/gate.py:334` | Add full signature/alias baseline comparison and external consumer compilation; a header is not compatibility evidence. |
| Generate gate remains a partial stage check | `scripts/gate.py:277` | Invoke the prepared generator's real check/determinism contracts and verify bytes, hashes and ownership. |
| Setup, wire and qualification are incomplete | `mode_setup` and main dispatch in `scripts/gate.py` | Preserve nonzero incomplete results until actual implementations land; never turn them into successful placeholders. |
| Cache/environment policy is inconsistent | `scripts/go-precheck.py:31` says GOCACHE is inherited, but the replacement environment does not copy it | Explicitly pass intended cache and platform variables, including Windows process requirements; test the real native subprocess path. |
| Pins are partly duplicated or stale descriptions | `tools/versions.json`; `LINTER_VERSION` in `scripts/gate.py` | One machine-readable source of versions, modules, targets and required cases; validate descriptive status against actual module files. |

## Proposed execution contract

### Targets, Go versions and CGO

The existing plan requires Go **1.27.0** and **1.27.4** on each native target.
Keep those pins until an approved toolchain update changes them. Do not copy ACP's
older floor or silently let `toolchain` download a newer compiler during a floor run.

| Target | GitHub standard runner label | Fleet requirement | Baseline |
| :--- | :--- | :--- | :--- |
| `linux/amd64` | `ubuntu-24.04` | Native Linux amd64 | Both modules, both Go versions, CGO off |
| `windows/amd64` | `windows-2025` | Native Windows amd64; WSL is not Windows evidence | Both modules, both Go versions, CGO off |
| `darwin/arm64` | `macos-15` | Native Apple Silicon macOS | Both modules, both Go versions, CGO off |

GitHub currently maps those labels to the required architectures, including
`macos-15` to arm64. Labels still receive image updates: record the image version
and fail if actual host/runtime architecture differs. Cross-compilation is
additional evidence, never a substitute for native execution.
[GitHub runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).

Each job checks `go env -json` and an executed fixture's `runtime.GOOS/GOARCH`,
compiler version, and module identity. Baseline environment:

```text
CGO_ENABLED=0
GOWORK=off
GOTOOLCHAIN=local
GOFLAGS=-mod=readonly
GOPROXY=off
GOSUMDB=off
GOOS=<explicit target OS>
GOARCH=<explicit target architecture>
```

Offline gates follow explicit preparation that verifies module sums, tool hashes,
schema/conformance pins and fixture hashes. Disabling network resolution after
verified preparation is not permission to skip authenticity checks during setup.
Reject injected build tags, GOEXPERIMENT or incompatible compiler environment;
preserve only intentional cache/temp/PATH and necessary native OS variables.

Run both modules independently: `.` and `cmd/mcpgen`. Verify that the manifest
covers every production module, with explicit classification of fixture modules;
root `go test ./...` does not traverse a nested module. No `go.work` is needed.

Build and execute a small external consumer with CGO disabled on every target.
Check selected dependency closure and build settings, including generator
standard-library-only policy. Keep application integrations, Python, Node and
conformance tooling out of the product dependency graph.

Race tests are additional **CGO_ENABLED=1 instrumentation**, on the preferred Go
version on all three native targets, after C-toolchain qualification. They do not
replace the CGO-free suite or introduce a C dependency into shipped libraries.
The Go race detector requires CGO and supports all three requested tuples;
Windows needs a suitable MinGW-w64 runtime/compiler.
[Go race detector requirements](https://go.dev/doc/articles/race_detector).

### Makefile and gate ownership

The Makefile should contain aliases and variables, with one Python command per
recipe. Avoid shell loops, shell-specific environment assignments and platform
conditionals. Python owns subprocess argument lists, environment, module iteration,
timeouts, logging and failure aggregation. Document a supported native Make
installation on Windows; test that actual entry point there. Python remains the
underlying portable entry point when bootstrapping a host.

| Proposed target | Contract |
| :--- | :--- |
| `make setup` | Explicit network-enabled preparation of pinned inputs; no floating upgrades |
| `make fmt` | Explicit formatting mutation only |
| `make pre-add-check` | Non-mutating format, affected-package vet/test and pinned lint; deletion/generated-input changes included |
| `make check` | Static checks, module closure, tidy/fix diffs, schema/API/generator checks, docs, workflow validation and tooling tests |
| `make test` | Fresh native CGO-free tests in both modules, with case reconciliation |
| `make race` | Qualified preferred-toolchain race lane in both modules |
| `make fuzz` | Queued 60-second campaigns per required fuzz target, with retained seeds |
| `make conformance` | Frozen official client/server scenarios plus independent advertised-feature cases |
| `make benchmarks` | Existing numerical budgets, samples and native resource measurements |
| `make preflight` | Immutable snapshot, full fleet matrix and required gate graph; emits evidence only on complete success |
| `make verify-receipt` | Validates exact outgoing commit, gate policy, inputs, tool versions and complete target results; runs no implicit push |

These names describe proposed deliverables, not commands that exist today.
Extend the current gate modes compatibly. Read tool versions from
`tools/versions.json`; add missing pins/checksums through reviewed setup work.
Use explicit `--config` for lint in both modules. Keep Staticcheck/SA1019 and
generated-code coverage without suppressions. Modernization checks are
non-mutating and must be supported by the pinned toolchain.

Every result includes case ID, command, module, target, compiler, exit status,
duration and log digest. Capture exits before parsing output. Report every failed
case and continue independent checks; stop dependent steps whose prerequisites
failed. Missing tools, malformed results and timeouts get named failures.
Unexpected skips, missing required cases and diagnostic-only selections cannot
produce a successful preflight receipt.

### Local fleet and pre-push enforcement

1. Prepare an immutable snapshot of the candidate commit plus necessary history
   and tags. Development checks may use an isolated working-tree copy; a push
   receipt must bind to the actual outgoing commit. Never test a changing shared
   checkout or infer identity from a diff summary.
2. Select each required OS through `available-dev-hosts` and its `pick-host`
   mechanism. Route heavy runs through `run-on-host`, or `run-heavy` when the
   selected run stays local. Use the race routing option where required. Serialize
   heavy work per physical host, including shared environments; distinct hosts
   may run concurrently. A load-average check is not a lock.
3. Ship the same snapshot to all targets, verify its digest after transfer, run
   both Go versions serially per host, and retain structured results. Bound queue
   waits, subprocess lifetimes and cleanup. Missing hosts or failed transfers
   leave qualification incomplete, not green.
4. Issue a receipt only after all required outcomes pass. Include commit/tree
   identity, API comparison base, module inventory, policy/tool/input hashes,
   native target identity, CGO mode, test inventory, seed, benchmark evidence,
   gate-control results and timestamps. Keep receipts outside tracked source.
5. Integrate a repository verifier into the existing global pre-push hook chain,
   preserving the disclosure guard. Read and validate every ref update from hook
   stdin, including non-HEAD branches and annotated tags; handle deletion updates
   explicitly. Recheck outgoing identity immediately before allowing push.
6. Reject absent, stale, altered, incomplete or mismatched receipts. Suggested
   validity window: less than one hour from completion, with fresh owner-managed
   local vulnerability evidence for each push. At or beyond one hour, repeat
   preflight; reject malformed or future timestamps.
   Configuration, dependency or source changes invalidate earlier evidence.
   Validate the snapshot used for each ref tip; scan all outgoing commits for
   disclosure. Release refs need the stronger release qualification policy.

Standard Git local hooks are bypassable and only affect installations that load
them. Local test-receipt validation is the selected pre-push policy. GitHub runs
functional CI, but branch-protection and required-check configuration are out of
scope. Hook integration must preserve owner-managed disclosure/security tooling
and be reviewed in the owning global-hooks repository. Do not replace its files
or `core.hooksPath` from this repo. Fresh vulnerability evidence comes from the
owner's local process; a test receipt does not reuse an earlier push's scan result.

### GitHub jobs and efficiency

Use one `verify` workflow on pull requests, pushes to integration/release branches
and release tags, manual dispatch, and a weekly schedule. Its final result job has
a stable name such as `verify / summary` and reports overall CI success or failure.
Configuring it as a protected-branch requirement, or administering merge queues,
is outside this work.

Proposed job graph:

```text
policy / module-and-case inventory
  +-- quality: static, generated, API, docs, workflow and gate-control checks
  +-- native: 3 targets x 2 Go versions; both modules in each job
  |     preferred-version jobs also run separate race and budget invocations
  +-- conformance: prepared official runner and required local supplements
  +-- fuzz: required bounded campaigns; replay retained seeds in native jobs
  +-- summary: inspect every prerequisite and expected result; fail closed
```

Six native jobs avoid multiplying runner startup/cache setup by the two modules.
Keep separate result records for ordinary, race and budget lanes, even when they
share a job. Run heavy steps sequentially inside each runner. Run deterministic
platform-independent checks once; lint/build-tag analysis still explicitly covers
all three GOOS/GOARCH pairs. Native filesystem, subprocess, transport, cancellation,
TLS and generator publication/recovery cases run on each required target/version.

Keep full coverage initially, including documentation pushes. A later proven
classifier may skip expensive jobs only for an explicit safe documentation set,
while the workflow and final summary job always run. Treat fixtures, embedded
docs, generated contracts, workflow/gate scripts, build tags and tool/module files
as code-affecting. Avoid workflow-level `paths-ignore` so events get explicit results.
The final `always()` job must inspect failures, cancellations and unexpected skips;
its own successful execution is not sufficient.

Cache verified module downloads, compiler artifacts and pinned tools with keys
covering OS, architecture, Go version and all relevant module/tool hashes. Avoid
reusing Go test result cache as execution evidence: run `-count=1`. Record shuffled
order seeds for reproduction. Cancel superseded PR runs, retaining final failure
logs; do not cancel release qualification merely because a newer branch push
arrives. Set explicit job/step timeouts and artifact retention.

Actions use reviewed full commit SHAs, read-only default permissions and
`persist-credentials: false`. PR jobs get no publishing credentials and run on
GitHub-hosted machines, not privileged developer hosts. A narrowly scoped release
job may gain write access only after all required tag checks pass. A cache hit or
an unavailable required tool never bypasses a check. Disclosure guards and security
scans remain owner-managed locally. This work does not install/reconfigure them or
add security-scan jobs to GitHub. Local pre-push qualification requires fresh
successful vulnerability evidence for each push; absent or failed evidence is not
a successful scan.

### Idiomatic Go, performance and release policy

CGO-free builds alone do not establish performance or idiomatic design. Retain
the approved lint, API, concurrency, cancellation, error and schema contracts.
Measure the actual library plus an external MCP-server-style consumer using both
transports, representative payloads and the approved concurrency levels.

Preserve all numerical budgets and at least five samples from the existing plan.
Measure equivalent validated workloads on the same native target/toolchain/CPU,
including allocations, retained memory, generator peak RSS and tail latency.
Capture every sample and native Windows resource measurements. GitHub must execute
the budget gates too; local controlled-host evidence complements them. A noisy
host or an exceeded bound triggers investigation under the existing deviation
rules, not discarded samples or a newly relaxed threshold.

Use portable CPU baselines, such as `GOAMD64=v1`, for the supported amd64 contract.
Do not impose advanced instruction sets on downstream applications. Let Go's
compiler/runtime select supported optimizations; require measured improvements
before special architecture paths or PGO profiles. Cross-target compile/lint checks
are cheap additions, but cannot prove runtime behavior or performance.

This is a library: default delivery is immutable semantic-version module tags and
release notes, not a binary deployment pipeline. Treat root and nested-generator
module versions explicitly; qualify the appropriate module-prefixed tag if the
generator is released independently. API comparison must select the previous
applicable release, excluding the candidate tag, and inspect incompatible changes
rather than trust apidiff's exit code alone. The first release needs an explicit
baseline policy, not a silent API-check skip.

No automatic tag creation or publication is proposed without the owner's explicit
release instruction. If prebuilt generator binaries later become a deliverable,
adapt selfupdate's build/verify/publish separation, checksums and artifact identity
checks under that separately approved scope.

## Acceptance evidence for implementation

Before declaring the standard installed, require baseline passes and deliberate
scratch failures for all new gates. At minimum, prove rejection of:

- a missing nested module, wrong native architecture, wrong compiler, CGO-enabled
  baseline, and a platform-only compile error;
- malformed Go with no formatter stdout, an unformatted file, deprecated API use,
  stale generated output, a real API removal and candidate-tag self-comparison;
- a selected test that never executes, unexpected skip, missing conformance
  scenario, nonzero subprocess exit, absent tool and unavailable required input;
- an over-budget copied measurement log, missing benchmark samples, and a race
  detector unavailable because its compiler prerequisites are absent;
- an altered/stale receipt, different outgoing commit, changed tool or policy
  hash, multi-ref push with one unqualified tip, and failed disclosure checking;
- a cancelled matrix cell, omitted target, and unsafe docs classification that
  the final CI summary must reject;
- queue contention, timeout and interrupted remote execution that must release
  owned resources and retain failure evidence without touching unrelated work.

These are proposed acceptance experiments. This report adds no gate and claims
none of those new experiments has already passed. Run the existing records/link
checker and Markdown lint on the report and index before handoff.

## Review decisions

The architecture already requires the two-version native matrix, race qualification,
bounded fuzzing, and numerical budgets. The following choices resolve adoption and
enforcement without dropping those requirements. This table preserves the original
options: D1 A and D3 C were selected; D2 was replaced by the scope ruling above.

| Decision | A — recommended | B | C |
| :--- | :--- | :--- | :--- |
| Standardization approach | Extend this repo's Python gates with the combined practices above; moderate work, direct fit to its current plan and policies | Adapt ACP's workflow/Makefile first, then replace shell logic and close the listed gaps; similar validation cost and more migration work | Build a shared, versioned fleet gate package before integration; greatest initial coordination cost, useful if simultaneous adoption across all repos is authorized |
| Private GitHub enforcement | Keep this repository private and enable the required paid-plan protection capability; recurring cost, least repository disruption | Move it to an appropriately licensed organization; administrative and ownership migration cost | Make it public after a separately authorized publication review; lower protection-plan barrier, substantial visibility/disclosure consequence |
| Local evidence reuse | Reuse complete exact-commit receipts up to 24 hours, with fresh vulnerability evidence per push; moderate implementation cost, avoids repeating unchanged expensive work | Run the entire fleet preflight freshly for every push; simpler validity policy, highest latency and fleet consumption | Use a shorter one-hour receipt lifetime with fresh vulnerability evidence; similar implementation cost, more frequent reruns but less elapsed environment drift |

All three review decisions are resolved. D1 A and D3 C are accepted policy. D2's
GitHub enforcement work is out of scope, not an unmet acceptance requirement.
The owner's publication change is recorded as reported; this work does not manage
visibility or repository protection settings.

## Suggested implementation sequence after approval

1. Amend the existing framework pair with the selected policy, complete case
   inventory and owner-managed local security boundary. That amendment is now
   recorded; implementation and acceptance evidence remain pending.
2. Finish the shared Python gate contract and thin Makefile, including portable
   environment handling and non-mutating pre-add checks. Prove failure controls.
3. Add immutable fleet snapshots, queued native qualification and receipt validation.
   Review hook integration in its owning repository without replacing disclosure.
4. Install the GitHub workflow using the same gates. Observe all six native cells,
   preferred-version race lanes, conformance, fuzz, budgets and final aggregation.
5. Demonstrate local rejection of expired/mismatched receipts and absent fresh
   local vulnerability evidence, then record CI/fleet results. Do not configure
   GitHub branch protection, rulesets or required status checks.

This order yields one maintained verification implementation and two independent
execution surfaces, while keeping every currently decided framework requirement.
