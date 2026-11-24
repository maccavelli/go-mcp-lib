# 0004-REPORT — Stage 1 execution: modules, tool pins, and honest gates

Execution evidence for stage 1 of
[0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md),
recorded per its execution-discipline rule. Stage 0 and its B01 resolution
are recorded in [0002-REPORT-stage-0-framework-inventory.md](0002-REPORT-stage-0-framework-inventory.md).

| Item | Value |
| :--- | :--- |
| Stage | 1 of 10 — modules, tool pins, honest gates |
| Source revision | `e717d85` on `main`, tree clean before writes |
| Go | go1.27.4 darwin/arm64, `GOWORK=off`, `GOTOOLCHAIN=local` |
| Lint | golangci-lint 2.15.0, Staticcheck all-checks, zero exclusions |
| Orchestrator | Python 3.15.2 standard library, subprocess argument arrays only |

## Deliverables

| Path | Content |
| :--- | :--- |
| `go.mod` | Runtime module `github.com/maccavelli/go-mcp-lib`, `go 1.27.0`, `toolchain go1.27.4` |
| `doc.go`, `doc_test.go` | Root package `mcp` documentation and pin-assertion test |
| `cmd/mcpgen/go.mod`, `cmd/mcpgen/main.go`, `cmd/mcpgen/main_test.go` | Generator module, standard-library-only CLI with explicit incomplete-stage exits (exit 2), stage-vocabulary test |
| `tools/versions.json` | Frozen pins: Go floor/toolchain, runtime dependencies, lint, schema, conformance, test-suite commits |
| `scripts/go-precheck.py` | Pre-add checks: explicit enumeration of both modules, gofmt/vet/build/test, named FAIL lines |
| `scripts/gate.py` | Modes `setup --allow-network`, `pre-add`, `unit`, `generate-check`, `api`, `wire`, `qualification`, `plants`; honest-incomplete on unimplemented stages |
| `scripts/test_gate.py` | 11 orchestration tests over controlled subprocess fixtures |
| `.golangci.yml` | Pinned linter policy: all Staticcheck checks, SA1019 on, empty exclusions, `generated: strict` |
| `.gitignore` | Extended, not replaced: original rules retained, Go transients added |

## Honest-incomplete contract, observed

- `gate.py setup` offline: FAIL `setup.conformance` with the `--allow-network` instruction — never a silent pass.
- `gate.py unit` on a module with no tests: FAIL `unit.<name>.empty`; execution is verified through `go test -json` events, not the textual PASS summary.
- `generate-check`, `api`, `wire`, `qualification`: FAIL-incomplete with the stage that lands them named.
- `gate.py plants`: synthesizes the deprecated-SDK plant
  (`CreateMessageHandler`, SA1019 under SEP-2577) in a temporary module,
  tidies offline against the pinned `go-sdk v1.8.0` from the local module
  cache, and requires the pinned linter to catch it: caught, 3 SA1019
  findings. A clean plant would FAIL the gate — the gate is thereby proven
  real, not decorative.

## Module-graph honesty

The stage-1 module graph is deliberately empty: no source imports the SDK
yet, so `go mod tidy` pruned the require entries when they were added
without an import. The three runtime pins are recorded in
`tools/versions.json` with their landing stages; the `go.mod` require list
grows organically as stages 2, 5, and 7 first import each dependency. No
placeholder import or fake require survives in committed source.

## Gate results

| Command | Exit | Named cases |
| :--- | :--- | :--- |
| `scripts/gate.py pre-add` | 0 | 8 pass lines (gofmt/vet/build/test × runtime, generator) |
| `scripts/gate.py unit` | 0 | 4 pass lines (tests + lint per module) |
| `scripts/gate.py plants` | 0 | `plants.lint-deprecated` caught 3 SA1019 |
| `python3 -m unittest scripts.test_gate` | 0 | 11 tests OK |
| `golangci-lint run` (root, generator) | 0 | `0 issues.` each |
| `gate.py setup` (offline) | 1 | `setup.go-toolchain/node/npm` pass; `setup.conformance` FAIL as designed |

Stage 1 exit criteria: both modules build; foundational tests pass; no
missing or empty gate reports success; tools pinned and prepared
explicitly; the deprecated-SDK plant is caught by the actual lint gate; no
blocker. Coverage obligations G01/G18/G22 infrastructure and D03
foundation are satisfied at the infrastructure level defined for this
stage.

## Limits and deviations

- `gate.py setup --allow-network` does not yet clone or build the
  conformance runner; it checks pins and reports the conformance
  preparation as incomplete. Network preparation lands with stage 9
  qualification wiring; the requirements hash is already verified in the
  stage-0 report.
- `setup.conformance` currently reports incomplete even with
  `--allow-network`; the message names stage 9 rather than performing a
  partial preparation, keeping the honest-incomplete contract.
- One in-flight correction: the `.gitignore` overwrite was caught by the
  working-tree diff and restored as a union of the original rules and the
  new Go transients before any commit.
- No license files are vendored yet: `tools/versions.json` records pin
  provenance; upstream licenses land with stage 2's committed schema
  inputs.
- Identity scan: no local account name, hostname, or real-machine absolute
  path appears in any committed file or this report; scratch plants lived
  in the system temporary directory and are not committed.
