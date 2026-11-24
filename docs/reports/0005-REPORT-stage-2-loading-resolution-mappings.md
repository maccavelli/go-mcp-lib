# 0005-REPORT — Stage 2 execution: lossless loading, resolution, and mappings

Execution evidence for stage 2 of
[0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md).

| Item | Value |
| :--- | :--- |
| Stage | 2 of 10 — schema loading, reference resolution, mappings |
| Source revision | `c1b90c6` on `main` plus uncommitted stage-2 tree |
| Go | go1.27.4 darwin/arm64, `GOWORK=off`, `GOTOOLCHAIN=local` |
| Lint | golangci-lint 2.15.0, Staticcheck all-checks, zero exclusions |
| Schema | revision `2026-07-28`, upstream commit `ac00313f…`, SHA-256 `907dcebc…`, 181,976 bytes, 155 definitions |

## Deliverables

| Path | Content |
| :--- | :--- |
| `schema/inputs/2026-07-28/schema.json` | Byte-verified pinned schema; committed bytes match the frozen hash, and an upstream re-fetch at the pinned commit produced identical bytes before the first commit |
| `schema/inputs/2026-07-28/manifest.json` | Versioned input manifest: source repository, commit, URL, hash, license provenance, verification method |
| `schema/inputs/2026-07-28/LICENSE.notice` | Upstream license notice from the schema repository at the pinned commit |
| `cmd/mcpgen/internal/acquire/` | Pin-verified schema acquisition: existing files verify without network; a mismatched fetch is discarded, never written; temp residue impossible |
| `cmd/mcpgen/internal/load/` | Strict lossless loader: duplicate-member rejection, UTF-8 validation, exact number tokens (`load.Number`), document/definition limits, stable diagnostics with phase/source/pointer/code/bounded message |
| `cmd/mcpgen/internal/resolve/` | Local-only reference resolution: JSON Pointer with escapes, canonical URIs with fragments, cycle-tolerant chains under a depth budget, edge budget, network refusal with path-redacted errors |
| `cmd/mcpgen/internal/model/` | Lossless intermediate model: required/optional/nullable/zero distinct, compositions structural, object openness declared, total keyword classification over the pinned 17 keywords, unsupported assertions and unknown required vocabularies fail with source pointers |
| `schema/semantics.json` | Normative protocol rules: methods with direction, capability, completion, cancellation, retry-safety, fixture IDs; definitions with normative section, SDK symbol, deprecation notes |
| `schema/sdk-map.json` | Disposition table seeded with 22 representative entries: `alias` and `owned-codec` classes; stage 3 completes it mechanically from Go metadata |
| `cmd/mcpgen/testdata/fixtures/` | 8 handwritten fixtures: recursive, nullable, conflicting, bounds, reserved, annotations, extension-vocab (expected failure), unsupported (expected failure) |

## Verification

| Command | Exit | Named cases |
| :--- | :--- | :--- |
| `go test ./...` (generator) | 0 | 45 tests across 5 packages, including the real pinned-schema build and the fixture corpus |
| `golangci-lint run` (generator) | 0 | `0 issues.` after fixing 10 real findings the new gate case surfaced |
| `scripts/gate.py generate-check` | 0 | schema present, 8 fixtures, 22 dispositions, 45 tests, lint clean |
| `scripts/gate.py unit` | 0 | runtime and generator tests plus lint |
| `python3 -m unittest scripts.test_gate` | 0 | 12 orchestration tests |
| `scripts/gate.py pre-add` | 0 | gofmt/vet/build/test both modules |

The model loads and builds the actual pinned schema: 155 definitions
modeled with no collapse; `Tool` retains its required `name` and typed
properties; deprecated-feature definitions (`CreateMessageRequest`) survive
losslessly per G23–G25.

## In-flight corrections

- The strict loader first decoded numbers to plain strings, which let
  `{"type": 42}` slip past the model's type check as the string `"42"`.
  Fixed with a distinct `load.Number` type so numbers and strings can never
  be confused again; the model now rejects the malformed fixture.
- One lint-fix edit mangled the acquire error path (dropped brace and lost
  error return); the build caught it immediately and the block was restored
  correctly before any commit.
- The `generate-check` gate contract changed from FAIL-incomplete to a real
  stage-2 check; its orchestration test was updated in the same change so
  the suite states the new truth, not the old one.

## Limits

- `sdk-map.json` dispositions are seeded (22 of 155); stage 3's `sdkmap`
  builder completes the table mechanically from Go metadata and ASTs, and
  the generator will fail on any definition without a disposition at that
  point.
- `semantics.json` records representative methods and definitions; stage 3
  extends it to total mapping over all standard methods.
- Reference resolution is local-only by design; encoded path traversal and
  symlink escape tests land with the registry's file-backed variant in
  stage 3, where documents are loaded from disk rather than bytes.
- Identity scan: no local account name, hostname, or real-machine absolute
  path appears in any committed file or this report; the upstream re-fetch
  URL carries only the public repository.
