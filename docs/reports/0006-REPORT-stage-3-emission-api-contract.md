# 0006-REPORT — Stage 3 execution: emission and API contract qualification

Execution evidence for stage 3 of
[0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md).

| Item | Value |
| :--- | :--- |
| Stage | 3 of 10 — emit and qualify the reviewed Go/API contract |
| Source revision | `5a42db2` on `main` plus uncommitted stage-3 tree |
| Go | go1.27.4 darwin/arm64, `GOWORK=off`, `GOTOOLCHAIN=local` |
| Lint | golangci-lint 2.15.0, Staticcheck all-checks, zero exclusions |
| Dispositions | 155 total: 47 alias, 96 owned, 12 owned-codec (generic `ClientRequest`/`ServerRequest` reclassified owned) |

## Deliverables

| Path | Content |
| :--- | :--- |
| `cmd/mcpgen/internal/sdkmap/` | Mechanical disposition completion from SDK ASTs: 155 definitions classified; frozen deprecation list verified against the SDK source at run time; generic SDK types excluded from aliasing |
| `cmd/mcpgen/internal/emit/` | AST-checked emission: aliases with source pointers, owned DTOs for deprecated/sealed surfaces (G23/G25-safe public API), constants, generated headers, manifest ownership, deterministic bytes |
| `cmd/mcpgen/internal/qualify/` | Three-platform compile tree: darwin/arm64, linux/amd64, windows/amd64; package and external-test compilation; compile-only, foreign binaries never executed; stale-output exclusion by manifest |
| `cmd/mcpgen/generate.go` | `SchemaToProtocol`: the in-process load→model→sdkmap→emit pipeline |
| `protocol/` | The generated contract, committed: `aliases.go`, `owned.go`, `constants.go`, `generated-manifest.json` |
| `go.mod` | The SDK require is now real: `github.com/modelcontextprotocol/go-sdk v1.8.0` — the module graph grew organically exactly when the first real import landed, as the stage-1 report promised |

## Verification

| Command | Exit | Named cases |
| :--- | :--- | :--- |
| `go test ./...` (generator) | 0 | 8 packages: acquire, load, resolve, model, emit, sdkmap, qualify, root |
| `golangci-lint run` (both modules) | 0 | `0 issues.` each after fixing 3 QF findings |
| `scripts/gate.py api` | 0 | manifest ownership, three generated files with headers, runtime builds, lint |
| `scripts/gate.py generate-check` | 0 | schema, fixtures, 22+ dispositions, tests, lint |
| `python3 -m unittest scripts.test_gate` | 0 | 14 orchestration tests, including api-fails-without-protocol |
| Emit determinism | pass | two consecutive scratch emissions byte-identical |

## Plants caught

- **Transitive deprecated alias** (scratch `/tmp` module):
  `type CreateMessageResult = mcp.CreateMessageResult` fails the real lint
  gate with SA1019 — the emitted alias policy is enforced by the
  toolchain, not only by the generator's table.
- **Platform compile plant** (qualify tests): a planted windows compile failure produces a failing report with detail; a broken scratch module fails real three-platform qualification.
- **Generic aliasing**: `ClientRequest`/`ServerRequest` are exported
  generic SDK types; a blanket alias does not compile. Caught by the first
  real `go build` of the generated package; sdkmap now classifies generics
  as owned.

## In-flight corrections

- The qualify `Result` zero-value initially meant failure (empty struct with `Package: false`); construction was fixed so success is explicit.
- `emit` emitted a struct field for the model's synthetic `<additional>` member, producing unparseable Go; synthetic and non-identifier members are now skipped, with openness deferred to the stage-5 codec.
- The first committed emission was regenerated after the generics fix; the committed `protocol/` bytes are the post-fix deterministic emission.

## Limits

- Union-typed and array-shaped properties emit as `json.RawMessage` placeholders; stage 5's strict codec completes their typing, per the plan's stage-5 assignment of G06 serialization.
- `qualify` compiles the scratch candidate tree for three platforms but
  has not yet run against the full repository tree with external-package
  tests; stage 4's CLI drives it over the real candidate before
  publication.
- The `diagnostic` stage remains a nil entry wired for stage 4.
- Identity scan: no local account name, hostname, or real-machine absolute path in any committed file or this report; the scratch plant module lived under the system temporary directory.
