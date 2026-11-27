---
status: complete
date: 2026-11-24
associated-madr: 0002-MADR-build-schema-driven-mcp-framework.md
---

# Audit of Stage 0–6 completion before Stage 7

**Stage 7 is not ready to begin on the strength of the existing completion
reports.** The implementation contains useful foundations, but earlier stages
have reproducible correctness failures and substantial unfulfilled exit criteria.
The appropriate next work is prerequisite repair and requalification under the
existing decision pair.

This report implements the owner's selection of option B: audit the preceding
stages more fully before proceeding. It observes and recommends; it does not
approve repairs, change the accepted contract, or execute Stage 7.

## Scope and baseline

- Audited revision: `5bfdd9f5e4a6e9286cf593fdc1f3ccec60890532`.
- The working tree was clean before the audit and remained clean throughout
  product investigation. Only this report and its documentation index are changed.
- Authority: [0002-MADR-build-schema-driven-mcp-framework.md](../decisions/0002-MADR-build-schema-driven-mcp-framework.md)
  and [0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md),
  including the Option D and CI/CD amendments. Both were read in full.
- Evidence reviewed: the Stage 0 inventory and Stage 1–6 reports, generator and
  runtime source, tests, gate scripts, generated artifacts, and relevant locally
  pinned SDK/linter source. Source locators below refer to this revision.
- Existing suites ran in isolated fleet copies, sequentially through the host
  queue. Root tests ran on Linux amd64 under WSL; generator tests ran on the
  Linux amd64 host. The latter reports Go 1.27.4. Both commands set
  `CGO_ENABLED=0`, `GOWORK=off`, and `GOTOOLCHAIN=local`.
- New diagnostic probes ran in a separate scratch copy on Darwin arm64 with
  Go 1.27.4, CGO disabled, network module access disabled, and scratch build
  cache. Destructive-path probes touched only scratch sentinel files.
- All findings below pre-exist this audit. None is introduced by an audit edit.

GitHub enforcement remains out of scope. Disclosure guards and security scans
remain owner-managed locally. The selected Makefile, exact-commit receipts and
three-target CI policy remain Stage 9 obligations; their absence is not counted
as a Stage 0–6 implementation defect.

## Stage-by-stage verdict

“Not complete” means at least one explicit exit condition is unmet. It does not
mean the stage produced no useful work. Step references are to the PLAN.

| Stage | Evidence that holds | Unmet obligations | Audit verdict |
| :--- | :--- | :--- | :--- |
| 0: inventory | Schema identity and the Option D feasibility proof are recorded. The committed schema still matches its frozen SHA-256. | Steps 2–5: complete semantic/method inventory, transitive alias allowlist, scenario-to-fixture reconciliation and verification owners are not established by the completion evidence. | Not complete; resolving B01 did not close the whole inventory. |
| 1: foundations | Both modules exist; generator remains standard-library-only; tools are versioned; basic orchestration exists. | Steps 2, 4–6: setup never prepares conformance, generated lint is excluded, and execution accounting accepts missing outcomes/skips. | Not complete; the gates cannot substantiate later completion claims. |
| 2: model and mapping | Pinned input and licenses exist; exact number tokens and fixture packages exist. | Steps 2–6: malformed JSON acceptance/panic, broken recursive references, assertion misclassification, unused reviewed maps, incomplete SDK audit. | Not complete; input-to-model correctness is not established. |
| 3: generated API | Deterministic Go files and a basic ownership list exist; the qualification helper can compile small scratch modules. | Steps 1–6: missing definitions/artifacts, lost wire semantics, transitive SDK leakage, no integrated complete-candidate qualification. | Not complete; successful compilation is substantially narrower than the promised API qualification. |
| 4: CLI/publication | Commands, byte comparison, exclusive generation lock and repeatable output exist. | Steps 1–7: incomplete command contracts, missing provenance checks, unsafe paths, nonrecoverable publication and absent fault coverage. | Not complete; publication/recovery must be repaired before being trusted. |
| 5: codec/validation | Independent corpus exercises the pinned validator; exact-number checks and 155 definition compilations exist. | Steps 1–3, 5–7: invalid raw JSON accepted, wrong default depth, missing owned v2 API/differentials, location/profile divergence, insufficient duplicate-bypass plant. | Not complete; this is a direct Stage 7 prerequisite. |
| 6: runtime | Owned handles, registrations and some client operations exist; existing routing tests pass. | Steps 1–6, 8: policies detached from live dispatch, output validation bypassed, missing fulfillment/API operations, declaration panics, mutable options and incomplete consumer qualification. | Not complete; this is a direct Stage 7 prerequisite. |

## Findings

### A01 — Completion records overstate the established baseline

The [Stage 0 inventory](0002-REPORT-stage-0-framework-inventory.md) still has
`status: incomplete` and an unfinished obligation ledger. Its final section
qualifies three client-role scenarios, not all Stage 0 requirements. The
[documentation index](../README.md) calls the record complete. The PLAN's opening
status also describes Stage 0 as incomplete while the index and subsequent
reports label Stages 1–6 complete.

Later reports explicitly defer work that their stage exits require: reviewed
mapping completion, semantic diff, crash recovery, codec differential tests,
and external consumers. These deferrals are not equivalent to an accepted
architectural amendment. Reconcile each obligation and its evidence; retain the
historical reports rather than rewriting their measurements as successes.

Consequence: downstream execution has no trustworthy prerequisite ledger.

### A02 — Test accounting can pass without a single passing test

[`scripts/gate.py`](../../scripts/gate.py), `mode_unit` at line 154, only requires
some `run` events after a zero process exit. It does not require matching final
outcomes or reject unexpected skips. Controlled subprocess fixtures returned
success for both of these event sequences in both modules:

```text
run Required; skip Required -> pass, 0 passed, 1 skipped
run Required               -> pass, 0 passed, 0 skipped
```

The reader discards malformed JSON lines, tests run without `-count=1`, and
`generate-check` uses similarly weak accounting. It checks a minimum seed-map
size rather than required definition/method coverage. It does not invoke CLI
`check`. `api` checks three headers, a filename list and a build, rather than
the promised API/coverage contract. `--module` selection is not implemented.

The Linux generator baseline illustrates the practical issue: it exits zero
with a required scratch-emission test skipped for the assumed SDK cache path.
Its second skip is an intentionally opt-in repository-emission helper; that
helper should be classified separately, not mistaken for required coverage.

### A03 — Generated-code lint exclusion contradicts policy

[`.golangci.yml`](../../.golangci.yml), line 19, sets `generated: strict`.
In the pinned golangci-lint 2.15.0 source, this mode recognizes the standard
generated-file marker and filters those files' findings. It does not mean
“strictly lint generated code.” All emitted Go files carry that marker.

Source basis: the pinned linter's
`pkg/result/processors/exclusion_generated_file_filter.go`, `Process` and
`shouldPassIssue`, and `exclusion_generated_file_matcher.go`.

[`scripts/go-precheck.py`](../../scripts/go-precheck.py) also omits lint and
runs `go fmt`, which rewrites files before reporting that formatting would
change them. Stage 1 requires lint in each module and a real generated-code
negative control. This audit established the configuration defect from source;
it did not rerun the full lint matrix.

### A04 — Setup preparation remains an explicit stub

[`scripts/gate.py`](../../scripts/gate.py), `mode_setup` at line 77, reports
conformance preparation incomplete even with `--allow-network`. It neither
prepares nor hashes the runner. This fails honestly, which is preferable to a
false pass, but does not meet Stage 1 step 2. The Stage 1 completion claim and
tool-manifest description of a prepared setup path therefore need correction.

### A05 — Generator “strict” JSON loading accepts invalid input and can panic

[`internal/load/load.go`](../../cmd/mcpgen/internal/load/load.go),
`parseStrict`, `parseNumber`, `parseString` and `parseObject`, has an incomplete
handwritten JSON parser. Scratch probes observed:

| Input | Observed result |
| :--- | :--- |
| `{} {}` | Accepted; trailing document ignored. |
| `{"x":+1}`, `{"x":01}`, `{"x":1e}` | Invalid numeric tokens accepted. |
| `{"x":"\ud800"}` | Lone surrogate accepted and normalized. |
| `{"x":"\ud83d\ude00"}` | Valid surrogate pair corrupted instead of decoding to its Unicode scalar. |
| `{"x":1,` | Index-out-of-range panic instead of a bounded diagnostic. |

`LoadFile` also reads the entire file before applying its size bound. Dialect
validation and the advertised diagnostic-cap behavior are not implemented by
this loader. These defects precede emission; downstream formatting cannot
recover the lost semantics.

### A06 — Resolution does not preserve the required reference graph

[`internal/resolve/resolve.go`](../../cmd/mcpgen/internal/resolve/resolve.go),
`ResolveChain` at line 190, treats a self-reference as repeated hops. The valid
recursive fixture fails with `reference chain exceeds 256 hops`; the PLAN
explicitly requires revisited nodes to become graph edges without exhausting
the acyclic depth budget.

Source inspection also finds URI suffix stripping, incomplete fragment/pointer
handling and no reference-base update across a chain. More fundamentally,
[`SchemaToProtocol`](../../cmd/mcpgen/generate.go) never invokes the resolver.
Thus passing resolver unit tests would not qualify reference handling during
actual generation. Traversal/symlink protections and diagnostics must be tested
at that integrated boundary as well as within the resolver.

### A07 — Unsupported assertions can become annotations

[`internal/model/model.go`](../../cmd/mcpgen/internal/model/model.go),
`buildNode` at line 165, classifies unknown keys as annotations after consulting
an incomplete assertion list. A scratch schema with `minLength: 3` is accepted
and records `minLength` as an annotation. The model neither represents nor
rejects that assertion. This violates Stage 2 step 4 and its fail-before-emission
exit; a partial keyword blacklist cannot establish total classification.

### A08 — Reviewed semantics, maps and input provenance are not enforced

[`schema/semantics.json`](../../schema/semantics.json) contains five method rows
and three definition rows. [`schema/sdk-map.json`](../../schema/sdk-map.json)
is still a representative seed. Generation computes dispositions mechanically
without consuming either reviewed input.

Scratch `checkTree` succeeds with both maps and the input manifest absent.
It also succeeds after appending whitespace to the frozen schema, changing its
hash while leaving generated bytes unchanged. The hash is calculated and
reported, not compared with the accepted source pin. These are executed probes,
not merely missing-test inferences.

### A09 — The transitive alias audit is absent and forbidden types are reachable

[`internal/sdkmap/sdkmap.go`](../../cmd/mcpgen/internal/sdkmap/sdkmap.go),
`inspectSDK` at line 126, collects exported type names. It does not inspect
deprecation comments, field graphs, method sets or instantiated signatures.
The “deprecations verified” result only verifies that hardcoded names exist.

[`protocol/aliases.go`](../../protocol/aliases.go) exports
`ClientCapabilities`, `ServerCapabilities`, `CreateMessageRequest`,
`ListRootsRequest` and other SDK request aliases. In SDK v1.8.0,
`ClientCapabilities` exposes deprecated sampling/roots fields;
`CreateMessageRequest` reaches deprecated sampling parameters. SDK request
types also expose SDK sessions through `Session` fields and `GetSession`.
The accepted reviewed-data-alias boundary is therefore not satisfied.

Stage 2/3 repair needs a transitive source/type audit and actual consumer compile
probes. A direct deprecated-symbol lint plant alone cannot prove this property.

### A10 — Disposition counts are not a complete generated contract

The pinned schema has 155 definitions; the generated Go files declare 132
distinct type names. Twenty-three definition names lack corresponding types,
including `InputRequest`, `InputResponse`, `ContentBlock`, `JSONValue`,
`ProgressToken`, `RequestId` and `ResultType`. This count alone is not a demand
for one Go type per schema node: the defect is that the emitter silently skips
non-object owned definitions without another verified binding/artifact.

[`internal/emit/emit.go`](../../cmd/mcpgen/internal/emit/emit.go),
`renderOwned` and `emitsStruct`, implement that skip. Reference-valued fields
become `any`; arrays/unions remain raw placeholders. Open-object definitions
such as `InputRequests`, `InputResponses` and `JSONObject` become empty structs
in [`protocol/owned.go`](../../protocol/owned.go), losing their members on normal
struct decoding. Optional scalar fields use value fields plus `omitempty`,
without the promised presence semantics or compensating codecs.

Only a protocol-revision constant is emitted; the planned method registries,
bindings, façade signatures, API/coverage artifacts and collision audit are
not implemented. Stage 5 did not complete the deferred serialization contract.

### A11 — CLI checks do not qualify the complete candidate

[`cmd/mcpgen/check.go`](../../cmd/mcpgen/check.go), `checkTree` at line 17,
renders three files and compares their bytes. It never calls `CompileTree`.
A scratch handwritten file containing `var X = missingIdentifier` leaves
`checkTree` successful. The positive unmodified-copy control also succeeds.

[`internal/qualify/qualify.go`](../../cmd/mcpgen/internal/qualify/qualify.go)
has a useful compile helper, but its callers are tests. Those tests qualify
small synthetic modules, not the full proposed product tree. Its environment
uses `GOFLAGS=-mod=mod`, contradicting readonly mode, and does not explicitly
enforce `CGO_ENABLED=0`.

The three-target helper is not evidence that CLI publication qualifies the
actual library for all three targets.

### A12 — Publication and recovery paths can escape the output directory

[`publish.go`](https://github.com/maccavelli/go-mcp-lib/blob/5bfdd9f5e4a6e9286cf593fdc1f3ccec60890532/cmd/mcpgen/publish.go), `staleOwned` at line 157, trusts
manifest filenames. [`recover.go`](https://github.com/maccavelli/go-mcp-lib/blob/5bfdd9f5e4a6e9286cf593fdc1f3ccec60890532/cmd/mcpgen/recover.go),
`recoverRestore` at line 118, trusts journal filenames and removes entries
recorded as new without verifying their current candidate hash.

Executed, isolated probes:

1. Add `../SENTINEL` to a scratch generated manifest. `checkTree` accepts it;
   `publish` returns success and deletes that sentinel outside `protocol/`.
2. Put `../SENTINEL` in a scratch journal as a new file. Give the sentinel
   manually changed bytes that do not match the candidate hash. Restore still
   deletes it and reports one new file removed.

No non-scratch file was affected. This is a direct violation of Stage 4's
ownership, containment and intervening-edit protection requirements. Path and
symlink verification must precede all writes/deletions, including recovery.

### A13 — Journal recovery cannot perform its promised rollback or completion

[`publish.go`](https://github.com/maccavelli/go-mcp-lib/blob/5bfdd9f5e4a6e9286cf593fdc1f3ccec60890532/cmd/mcpgen/publish.go), `journalFile` at line 30, records
hashes but no recoverable original/candidate bytes. `writeSynced` at line 144
truncates and writes destination files directly; it does not stage and rename
them. Staging uses a generic temporary directory, not guaranteed to be on the
destination filesystem. Stale removal is recorded after removal, and recovery
does not coordinate through the generation lock.

`recoverRestore` explicitly refuses replaced originals because their bytes were
not retained. `recoverComplete` only verifies that every file already matches
its candidate hash; it cannot supply a missing/replaced file or finish manifest
promotion. These are source-confirmed defects, independent of A12's executed
path probes.

[`scripts/verify_recovery.py`](../../scripts/verify_recovery.py) claims in its
docstring to kill a publishing child but actually deletes a generated file and
regenerates. It does not test a killed writer, partial write, disk-full, sync,
rename or recovery rollback. Its absolute scratch CLI dependency also limits
repeatability. The Stage 4 report's recovery qualification is insufficient.

### A14 — Remaining CLI contracts are narrower than the approved commands

[`check.go`](../../cmd/mcpgen/check.go), `diffManifests`, compares only revision
and filename sets. It cannot report changed assertions, capabilities, methods,
aliases or API signatures. A changed definition with the same output filenames
is invisible to this diff.

[`cli.go`](../../cmd/mcpgen/cli.go) silently normalizes an unknown `--format`
to text; common `--manifest` is absent. [`recover.go`](https://github.com/maccavelli/go-mcp-lib/blob/5bfdd9f5e4a6e9286cf593fdc1f3ccec60890532/cmd/mcpgen/recover.go),
`runUpdate`/`noTraversal`, cleans an absolute path but does not enforce separation
from live schema inputs or reject symlink escapes. Revision is not validated
as immutable provenance. These Stage 4 obligations should be restored with
positive and negative command-level tests.

### A15 — Runtime raw validation accepts incomplete JSON

[`codec/codec.go`](../../codec/codec.go), `ValidateRaw` at line 51, treats any
error containing `EOF` as successful termination. Empty/whitespace frames,
an unfinished quoted string and the truncated literal `tru` all return nil
error. The duplicate-name rejection and valid-object controls behave correctly.

`DefaultBounds` sets depth 256; the accepted runtime default is 64 with a
maximum configurable depth of 256. A depth-65 input is accepted by the default.
The generator's separate depth limit must not be substituted for runtime policy.

Stage 7 must not build its first-error latch around this validator until these
failures and boundary cases are corrected.

### A16 — Owned codec and production validator qualification are incomplete

There is no owned `encoding/json/v2` encode/decode API or v1/v2/SDK differential
test path in the implementation. `codec` supplies token validation helpers;
other code uses v1 JSON and SDK codecs. Existing absence/null tests do not
establish the promised owned v2 behavior.

[`internal/validation/validation.go`](../../internal/validation/validation.go),
`Compile` at line 59, does not register the format checkers or assert formats.
[`formats_test.go`](../../internal/validation/formats_test.go) constructs a
different compiler and explicitly does both. The production adapter accepts
`"not a URI"` against `{ "type": "string", "format": "uri" }`, while the
format test describes assertions as the accepted profile. A schema's ordinary
format annotation may correctly be nonasserting; the failure is the unqualified
difference between claimed profile, location assertions and production behavior.

The regex-rejection helper is invoked by its test, not by schema declaration.
Pattern/profile qualification must be wired into the real declaration path
rather than inferred from a helper that always returns an error.

The duplicate-bypass mutation in
[`scripts/verify_mutations.py`](../../scripts/verify_mutations.py) disables a
branch that classifies a native duplicate error. It does not disable native
duplicate rejection; the following return still rejects the input. Its failing
test proves error-classification sensitivity, not detection of validation bypass.

### A17 — Protocol-location helpers are incorrect and disconnected

[`internal/validation/locations.go`](../../internal/validation/locations.go)
accepts an HTTPS `Root.uri`, despite the accepted file-URI requirement. Its URI
checker accepts `file: bad URI`. The URI-template helper rejects the supported
reserved expansion `{+path}`. All three outcomes were reproduced.

These helpers are used by their tests, not by the runtime adapter. Separately,
[`resources.go`](../../resources.go) implements different, weaker checks rather
than using the qualified validator/parser. Consequently helper coverage does
not establish enforcement during registration or protocol processing.

### A18 — Live runtime dispatch bypasses configured policies and output checks

[`middleware.go`](../../middleware.go), `chainSnapshot` at line 207, constructs
the owned chain but is only called by tests. `rebuildReceiving` at line 140
installs only user middleware into the SDK. `Authorize`, `Observation` and
`Admission` store callbacks that are not connected to dispatch.

A real in-memory tool round trip with rejecting authorization/admission and an
observation counter produced:

```text
auth=0 observe=0 admit=0 handler=1 err=<nil>
```

The declared-output validator is also absent from live dispatch. A handler
returning `reply: "wrong"` under a declared `const: "required"` output property
returns a successful result. [`tools.go`](../../tools.go) does not pass the
declared `OutputSchema` to the SDK either.

Even wiring the current chain unchanged would be insufficient: its result
validator is inside user middleware, so post-handler transformations occur
after validation. `UseMiddleware` also reinstalls the accumulated chain on every
call; a live ping after installing A then B invokes A twice and B once.

These are direct Stage 6 step 4/6 failures, separate from the raw transport
checks correctly scheduled for Stage 7 and quota counts scheduled for Stage 8.

### A19 — Accepted fulfillment and the canonical API are unfinished

The MADR's Option D ruling, lines 799–810, requires owned fulfillment using
`MultiRoundTripOptions.Disabled` and public request/response map codecs.
Dropping sampling and roots requires a fresh ruling; it was not the selected
resolution. The [Stage 6 report](0010-REPORT-stage-6-runtime-api-feature-matrix.md)
instead describes their exclusion. [`client.go`](../../client.go) configures
no owned handlers or fulfillment loop. The required owned union types are
also missing under A10.

The canonical client lacks prompt listing/getting, completion, discovery,
resource-template listing and cursor/request-metadata parameters. Subscription
methods alone do not establish configured callbacks/capability coverage.
There is no exported server serve/connect operation; round-trip tests reach
`server.sdk.Connect` privately. External consumers therefore cannot reproduce
those tests through the advertised owned surface.

[`transport.go`](../../transport.go) also seals `Transport` with a root-package
method, while the PLAN places adapters in a `transport/` subpackage. That
subpackage cannot directly implement the private method. Resolve the package
boundary in the approved pair before Stage 7 implementation; do not expose an
unguarded SDK escape hatch to work around it.

### A20 — Registration errors, immutable options and public errors fail their contracts

[`tools.go`](../../tools.go), `AddTool` at line 46, catches and re-panics every
SDK declaration panic. The scratch malformed input-schema probe panics with
`input schema must have type "object" (got bad-type)` instead of returning the
promised registration error. Input validation remains SDK-internal rather
than the canonical adapter checked before lossy conversion.

[`server.go`](../../server.go), `Options` at line 117, returns a shallow copy
whose `Implementation` pointer is shared. Changing the returned implementation
changes future `Options()` results; reproduced. `client.go` has the same
pointer-copy pattern. SDK construction copies some fields, so the defect is
not a claim that every such edit changes the live wire identity.

[`errors.go`](../../errors.go), `TypedError.Error`, formats the entire cause
and message without bounds or sanitization. Several API error paths return
unclassified errors; no internal-error sentinel is provided. Wire disclosure
through a future transport was not tested here, but the current public error
contract is already weaker than Stage 6 step 5 requires.

## Executed evidence and its limits

| Run | Result | Meaning |
| :--- | :--- | :--- |
| Root `go test -count=1 -json ./...`, Linux amd64/WSL, CGO=0 | Exit 0; 56 test events passed, no Go-test skips; `protocol` has no tests. | Existing assertions pass. Does not prove omitted behavior. |
| Generator `go -C cmd/mcpgen test -count=1 -json ./...`, Linux amd64 host, CGO=0 | Exit 0; 64 passed, two skipped. | Scratch emission skipped for assumed SDK path; repository-emission helper intentionally opt-in. |
| Vendored validator corpus inside root suite | 1,237 instance cases passed, zero failed, 18 reported annotation skips. | Distinct from Go test-event counts; these skips are internally counted, not `testing.T.Skip` events. |
| Scratch `TestAuditRuntime` | Exit 1; eight top-level diagnostic subtests fail. | Confirms A15–A18 and A20 with live calls or direct production entry points. |
| Scratch `TestAuditGenerator` | Exit 1; eight top-level diagnostics fail; positive check control passes. | Confirms A05–A08 and A11–A12. Six malformed-input subtests also report their failures individually. |
| Controlled `mode_unit` fixtures | Gate returns zero failures for all-skipped and unterminated test streams. | Confirms A02 without executing mocked child commands. |
| Pinned schema digest | Matches `907dcebce6aaeb16c21421ffef8c207093d88b9442fee340658a99cacb06b551`. | The current committed input is intact despite missing enforcement. |

The first local scratch runtime invocation could not access the configured
build cache and failed during setup. The recorded diagnostic run used a writable
scratch cache and reached all tests. No product repair/build iteration occurred.

The new diagnostic assertions were observed failing on the actual defects;
malformed bytes, a broken candidate and tampered scratch ownership records
were deliberate negative inputs. Valid-object, duplicate-rejection and
unchanged-generator controls distinguish those failures from a wholly broken
harness. No new product gate or committed test is introduced by this report.

This is a completion audit, not full release certification. It did not rerun
the pinned MCP conformance runner, race/fuzz suites, Go 1.27.0 matrix, native
Windows tests, full macOS suites, TLS/SSE integration or performance budgets.
Source-only findings are identified above. Linux suite success and the tiny
qualification-helper builds do not establish the required three-platform
product matrix or target optimization.

## Recommended repair sequence and Stage 7 entry gate

Amend the existing `0002` MADR/PLAN; do not create a competing architecture.
Preserve the owner's accepted semantics and target policy. Assign each finding
an acceptance case and state in the plan before product changes.

| Order | Repair scope | Evidence required before closing |
| :--- | :--- | :--- |
| R1 | A01–A04: completion ledger, prepared inputs, honest execution accounting and generated lint. | Missing/invalid/skipped outcomes fail; generated-code plant is caught; complete scenario and definition inventory has owners. |
| R2 | A05–A11: strict loading, graph resolution, semantics, transitive aliases, generated API and integrated qualification. | Independent malformed/recursive/Unicode/presence fixtures; full reviewed bindings; actual product candidate compiles on three targets; bad candidates cannot publish. |
| R3 | A12–A14: publication, recovery and full CLI contracts. | Every path stays contained; original/candidate bytes recover; manual edits survive; actual killed-child and filesystem-fault experiments pass in scratch. |
| R4 | A15–A17: owned codec, canonical validation and location/profile enforcement. | Frozen bounds; real v1/v2/SDK differential coverage; original qualification fixtures through production adapter; genuine raw-validation bypass plant caught. |
| R5 | A18–A20: live policies, canonical API/fulfillment, error contracts and immutable configuration. | Public-consumer fixtures; denying policies prevent side effects; transformed outputs validate last; Option D scenarios complete; registration errors return; isolation tests pass. |

Stage 7 may start after those prerequisite exits are requalified, the transport
package seam is decided, and the records accurately identify remaining
Stage 7–10 work. Stage 7 still owns guarded stdio/HTTP/SSE, original SDK
connection-hook preservation, first-error propagation, modern lifecycle and
real transport tests. Stage 8 quotas and Stage 9 CI must not be silently pulled
forward or used to defer already-required Stage 0–6 behavior.

## Review options

| Option | Next action | Cost and recommendation |
| :--- | :--- | :--- |
| A | Draft the `0002` repair amendment and acceptance ledger from R1–R5, for review before execution. | **Recommended.** Moderate planning effort, substantial implementation work; directly addresses demonstrated defects without changing requirements. |
| B | Obtain an independent review/reproduction of this audit, then draft the repair amendment. | Higher review cost and later start; useful if a second assessment is wanted before accepting the repair scope. |
| C | Hold implementation while the owner reviews the report and selects the repair sequence. | Lowest immediate work; all demonstrated defects and the Stage 7 block remain. |

Doing nothing leaves misleading completion claims, unreliable gates, unsafe
generator recovery and unenforced runtime policies in place. Starting Stage 7
against that baseline would make transport results harder to interpret and
would not satisfy the accepted plan.
