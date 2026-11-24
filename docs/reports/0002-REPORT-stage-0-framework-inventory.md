---
status: incomplete
date: 2026-11-24
associated-madr: 0002-MADR-build-schema-driven-mcp-framework.md
---

# Stage 0 grounding and framework inventory

Execution of [0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md),
under [0002-MADR-build-schema-driven-mcp-framework.md](../decisions/0002-MADR-build-schema-driven-mcp-framework.md).

Stage 0, pass 1 is **incomplete**. The owner authorized this stage on
2026-11-24. Investigation found a pre-existing conflict between required
feature coverage and the prohibition on deprecated SDK symbols. No resolution
has been selected, no phase is claimed passing, and stages 1–10 remain
unstarted. This report records observations and does not amend the architecture.

## Grounding obtained

| Input | Observation |
| :--- | :--- |
| go-mcp-lib | Started at `bd99482`, clean `main`; local tracking ref agrees. No remote fetch was performed. |
| Prior scaffold baseline | `f0af2c5` is now followed by the documentation commit; the plan's description of its records as uncommitted is historical. |
| mcplib | HEAD remains `b4eac5280d785e2be47e057f3ac3ddca04535335`; three untracked proposal records, listed below, were preserved. |
| acp-go-sdk2 | HEAD remains `a6495a56ddf2c2503b4613a2f91fb9ca400ef61a`; clean working tree. |
| mcplib source | `go.mod`, `server.go`, and `schema/schema.go` still show SDK v1.6.1, the mutable session wrapper, and permissive schema fallback. |
| ACP generator | `cmd/generate/go.mod`, `main.go`, and `internal/output/write.go` still show a separate Go module, phased generation, and sequential publication. |
| Local toolchain | `go version go1.27.4 darwin/arm64`; Python 3.14.10. |
| SDK package resolution | Offline `go list -mod=readonly -json github.com/modelcontextprotocol/go-sdk/mcp` exited 0 and resolved v1.8.0. |
| SDK codec | `internal/json/json.go` still imports Segmentio JSON and disables case-insensitive struct-field matching. |
| Pinned schema | Retained prototype input is 181,976 bytes, has 155 definitions, and matches the recorded hash. |
| Prototype source | The embedded diagnostic block and retained `qualification_test.go` both match the recorded source hash. No prototype tests were rerun. |
| Conformance requirements | Fresh bytes from the pinned commit match the plan's requirement hash. |

Preserved mcplib working files:

- `docs/decisions/0017-MADR-deliver-hfsc-artifacts-through-resource-links.md`
- `docs/decisions/0018-MADR-make-mcplib-the-canonical-mcp-sdk-surface.md`
- `docs/decisions/0019-MADR-modernize-hfsc-backplane-transfers.md`

The starting dirty-content inventory was empty; its SHA-256 is
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
The original [0002-REPORT-qualify-mcp-framework-boundaries.md](0002-REPORT-qualify-mcp-framework-boundaries.md)
is unchanged. Its prototype results are not implementation qualification.

## Verified input identities

| Input | SHA-256 |
| :--- | :--- |
| Retained MCP schema | `907dcebce6aaeb16c21421ffef8c207093d88b9442fee340658a99cacb06b551` |
| Embedded and retained diagnostic Go source | `7e9a8796cd0c1d3acb5ecf81584f76a33287c4651dad8367c10d54fbd187d4ec` |
| Local SDK v1.8.0 `mcp/protocol.go` | `dfb56019c469a25218816509235dac191bddaf2b357e94e198145ec44463d01e` |
| Local SDK v1.8.0 `internal/json/json.go` | `20af74fbf13163943c908b5082ed0e5f4a660452452d682ac5c77fa1a3b50daa` |
| Fresh conformance `requirements/2026-07-28.yaml` | `ae2f4f6210fd729e2e318edd5bbfa31a43cee0bc608e48052fa26dbf1d939b57` |

The SDK package metadata reports:

```text
Path: github.com/modelcontextprotocol/go-sdk
Version: v1.8.0
GoVersion: 1.25.0
Sum: h1:KIvahhYqwtbeniWVPs3TcXEA7b8jEtwfBpOTAI+Urx4=
GoModSum: h1:dL7u98E/zjJTGzEq+j30jQ8K2k1mb6LeAH4inEcSGts=
```

These sums match the prototype report. This invocation resolves metadata;
it is not a fresh module-download verification or a compilation result.
The full redacted package metadata was retained in session scratch as
`go-list.json`; the portable identity fields needed for this finding are above.

## B01 — Deprecated feature coverage conflicts with API policy

This conflict predates Stage 0; neither the dependency pin nor the feature
requirements were changed during this pass.

| Evidence locator | Observed requirement or source fact |
| :--- | :--- |
| `0002-PLAN-build-schema-driven-mcp-framework.md`, Stage 6 step 3, lines 400–405 at this pass | Includes supported sampling and root interactions alongside modern input-required results. |
| Same PLAN, Stage 3 step 3, lines 277–280 | Forbids deprecated option fields reachable through exported signatures. |
| Same PLAN, Completion Criteria, lines 792–793 | Requires Staticcheck without exceptions and no deprecated transitive alias surface. |
| Same PLAN, Stage 9 steps 1–3, lines 539–552 | Requires conformance coverage and forbids expected-failure suppression or absent required checks. |
| SDK v1.8.0 `mcp/protocol.go:662–666` | `CreateMessageParams` is deprecated for protocol 2026-07-28. |
| SDK v1.8.0 `mcp/protocol.go:1276–1304` | `ListRootsParams`, `ListRootsResult`, and the result's `Root` type form a deprecated surface. |
| SDK v1.8.0 `mcp/protocol.go:453–494` | `ClientCapabilities` exposes deprecated roots and sampling members; blanket aliasing fails the stated audit policy. |
| SDK v1.8.0 `mcp/protocol.go:2262–2284` | `ServerCapabilities` exposes a deprecated logging member. |
| SDK v1.8.0 `mcp/protocol.go:49–52` | `InputRequest` has an unexported marker method; a standalone foreign DTO cannot directly implement it. |
| Pinned conformance requirements, lines 63–64 | The required server list includes `input-required-result-basic-sampling` and `input-required-result-basic-list-roots`. |

SDK locators refer to the locally resolved file identified by hash above.
The [versioned SDK source](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/protocol.go)
and [SEP-2577](https://modelcontextprotocol.io/seps/2577-deprecate-roots-sampling-and-logging)
corroborate the deprecation. The
[immutable conformance requirement file](https://github.com/modelcontextprotocol/conformance/blob/96a6f9478b7f2e125cf0360a213383cf57533893/requirements/2026-07-28.yaml)
establishes that the two scenarios remain in its required list.

The exact review outcome is:

```text
FAIL B01: required sampling/root coverage has no established binding that
satisfies the accepted no-deprecated-symbol/no-suppression policy.
```

This is a contract-review finding, not a failed product test. It does not
prove that every possible SDK integration is impossible. It establishes that
the current plan does not resolve this combination and that a compliant
binding has not been demonstrated. Owned capability policy types can avoid
blanket aliases, but alone do not resolve the sampling/root execution path.

No resolution is selected. Suppressing Staticcheck, skipping the required
scenarios, silently omitting these features, or duplicating SDK dispatch would
change approved constraints and are not proposed as workarounds. The owner
must resolve the architectural conflict before execution resumes. A supported,
nondeprecated upstream binding would need source and contract evidence; none
has been established in this pass. Until then, Stage 0 cannot pass its no-blocker
exit criterion, and Stage 1 cannot begin.

## Stage 0 obligation ledger

| Step | Result | Remaining work |
| :--- | :--- | :--- |
| 1: Re-ground sources and diagnostic hash | Partial | Core repository identities, selected reference files, codec boundary, and source hash verified. Remaining source locators and full transport policy review are unfinished. |
| 2: Complete definitions, methods, and semantics | Incomplete | Confirmed the 155-definition input identity. Per-definition dispositions, method/direction/capability rows, and normative rule inventory are not frozen. |
| 3: SDK metadata and alias closure | Blocked by B01 | Package metadata obtained; deprecated capability and input types identified. No complete alias allowlist or transitive method audit is claimed. |
| 4: Every conformance scenario | Partial | Requirement bytes verified and conflicting required scenarios identified. Full runner scenario, transport, role, feature, and local-fixture reconciliation is unfinished. |
| 5: Execution report and verification owners | Partial | This report preserves the finding. No complete D/G verification-owner assignment or successful phase exit is claimed. |

No product gate, assertion, or test was added. Stage 0's exemption from new
plant experiments applies. Existing prototype plants were neither modified
nor rerun. No build pass occurred, and no implementation phase was committed.

## Commands and execution notes

- `git status --short --branch` in go-mcp-lib, mcplib, and acp-go-sdk2: exit 0;
  full status findings appear in the grounding table.
- `git rev-parse HEAD` in the reference repositories: exit 0; hashes appear above.
- `go version`: exit 0; exact output appears above.
- `go list -mod=readonly -json github.com/modelcontextprotocol/go-sdk/mcp`:
  exit 0, run from the retained prototype module with `GOWORK=off`,
  `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, and an isolated scratch
  `GOCACHE`. No module manifest or dependency was changed.
- Python standard-library hashing read the recorded source block, retained
  prototype files, and SDK files; byte counts and identities were checked
  before conclusions were recorded.
- A direct Python HTTPS fetch first failed with DNS error `[Errno 8]` inside
  the sandbox. The same public pinned requirement fetch succeeded with network
  authorization, returning 7,408 bytes with the expected hash.
- One inline directory-listing expression had a syntax error; a corrected
  read-only listing succeeded. A separate overlength inline command was
  rejected by the local hook before execution; subsequent commands used short
  expressions or one-use scratch Python files. Neither was a product build.

Documentation validation results are recorded in the associated PLAN after
checking this report and the approval edits. No new decision or product change
is hidden in this report.

## Stage 0, pass 2: B01 Option A qualification evidence (2026-11-24)

This section records the approved pass-2 investigation (PLAN amendment,
"Stage 0, pass 2: approved B01 Option A"). All work ran in throwaway scratch
modules under the system temporary directory; no repository file beyond this
report changed. Starting revision remains `bd99482`; the four dirty documents
and their inventory hash
`32d15159809e5e5ee3713b96cb4d589a8cd814e97b24a83d60a5f04002a5e18c`
were preserved untouched.

### Composition under test

- SDK pin: `github.com/modelcontextprotocol/go-sdk` v1.8.0, read from the
  local module cache (offline run, `GOPROXY=off`, `GOSUMDB=off`,
  `GOTOOLCHAIN=local`, `GOWORK=off`, isolated scratch `GOCACHE`).
- Scratch module `github.com/maccavelli/mcplib` (throwaway path under the
  system temporary directory), single test file `option_a_test.go`.
- Three scenarios from the frozen conformance requirements:
  `input-required-result-basic-sampling`, `input-required-result-basic-list-roots`,
  and `input-required-result-basic-elicitation` (elicitation as the modern
  control).
- Server role: SDK `Server` + `NewStreamableHTTPHandler` (stateless,
  JSON-response) behind `httptest`, protocol version 2026-07-28; the tool
  returns `CallToolResult{InputRequests, RequestState}` on the first round and
  consumes `Params.InputResponses` / `Params.RequestState` on the second.
- Both roles tested separately. SDK dispatch, retry, and request-state
  handling stayed enabled (no `MultiRoundTripOptions.Disabled`, no manual
  loop in the automatic-fulfillment test).

### Scratch copies and hashes (SHA-256, `option_a_test.go`)

| Copy | Purpose | Hash |
| :--- | :--- | :--- |
| A (modern-only) | Candidate composition, zero deprecated symbols | `f0632b462e7524672b409a52603dcd8154c56c8448c2e8f3a38cfef0cde709b3` |
| B (deprecated control) | Same composition plus the only SDK v1.8.0 client bindings for sampling and roots | `23817ca3b307163ccd7a6cbc337a20019dc869f1393b16f69a15dbf68d47fbef` |
| C (broken fixture control) | Copy B with corrupted sampling and roots fixtures | `c7f63f94d8310b89e26839d09b0a96ebe5ccc079882dfbd34f77e8e37667dcb8` |

All three share `go.mod` hash
`a99c105becb7dc097b0ba4bc83d2f754622eab9515a5446c922e5519d8f7aad5`.

### Server-role result (both copies A and B)

`TestServerPublicCodec` passes for all three scenarios. Two HTTP wire
rounds per scenario complete through the SDK's stateless streamable handler
with protocol 2026-07-28 headers (`MCP-Protocol-Version`, `MCP-Method`,
and `Mcp-Name` — the 2026-07-28 stateless requirement caught in pass-1
drafting is confirmed in source: `streamable_headers.go` rejects
`tools/call` without `Mcp-Name` when the protocol version is
2026-07-28 or later). First round returns `resultType: "input_required"`
with the request map and `requestState`; second round returns
`resultType: "complete"`. The public codecs `InputRequestMap` /
`InputResponseMap` round-trip owned JSON payloads losslessly; the
application handler reads real responses from `Params.InputResponses`
and verifies `Params.RequestState`. No deprecated symbol is required for
the server role of any scenario. Server-role qualification therefore
passes on both the function and policy axes.

### Client-role result — copy A (modern-only)

`TestClientAutomaticFulfillment`:

- Elicitation: **passes.** SDK dispatch fulfills via the non-deprecated
  `ClientOptions.ElicitationHandler`; the owned callback is invoked once;
  two SDK rounds complete; no middleware involvement
  (`receiving_callback_calls=0`).
- Sampling: **fails functionally.** `CallTool` returns
  `multi round-trip: fulfilling input request "q": client does not support
  CreateMessage`. The SDK's automatic fulfillment dispatch
  (`mrtr.go`, `fulfillInputRequest`) requires a client-side sampling
  handler; v1.8.0 offers only `ClientOptions.CreateMessageHandler` /
  `CreateMessageWithToolsHandler`, both Deprecated. Receiving middleware is
  bypassed entirely by the fulfillment dispatcher
  (`receiving_callback_calls=0`), so no owned non-deprecated hook exists.
- Roots: **fails functionally.** The SDK's built-in default root list
  responds `{"roots":[]}`; the server-side application check rejects it.
  `Client.AddRoots` is the only way to configure the list, and it is
  Deprecated (as are `Root`, `ListRootsResult`, `RootCapabilities`, the
  `RootsV2` capability field, and `ClientCapabilities.Sampling`).

### Client-role result — copy B (deprecated control)

`TestClientAutomaticFulfillment` passes for all three scenarios:
sampling returns real output from the configured handler
(`tool_calls=2`, `receiving_callback_calls=0`,
`elicitation_callback_calls=0`), roots returns the actual configured list
(`file:///fixture/project`), elicitation unchanged. SDK dispatch, retry,
and request-state handling are exercised end-to-end. The composition is
functionally complete for both roles — and every added binding is
deprecated surface.

### Policy-axis result (pinned golangci-lint 2.15.0, staticcheck, no exclusions)

- Copy A (modern-only): `0 issues.`
- Copy B (deprecated control): 3 SA1019 findings, no exclusions applied:
  - `ClientOptions.CreateMessageHandler` deprecated (sampling, SEP-2577)
  - `*CreateMessageResult` deprecated (sampling, SEP-2577)
  - `(*Client).AddRoots` deprecated (roots, SEP-2577)
- The same findings are the only route to a functional client role; there
  is no non-deprecated alternative, so the policy failure is not fixable by
  composition.

### Negative controls

- Copy C (broken fixture control, in a separate copy as required): the
  corrupted sampling and roots fixtures both fail loudly —
  `FAIL sampling application result` and `FAIL roots application result`
  with the observed-vs-wanted output printed — confirming the harness
  detects fixture corruption rather than passing vacuously. The
  server-role test (`TestServerPublicCodec`) continues to pass in copy C
  because its fixtures are derived from the pristine copy-A wire data;
  the client-role assertions are the operative negative control.

### Sealed-interface findings (why no composition can succeed)

- `InputRequest` and `InputResponse` are sealed interfaces
  (unexported marker methods `isInputRequest()` / `isInputResponse()`,
  `protocol.go`). The only implementations are the SDK's concrete types:
  `*ElicitParams`, `*CreateMessageParams`, `*CreateMessageWithToolsParams`,
  `*ListRootsParams` (requests) and `*ElicitResult`,
  `*CreateMessageResult`, `*CreateMessageWithToolsResult`,
  `*ListRootsResult` (responses). Application-owned types cannot
  implement them.
- The modern multi-round-trip wire surface (`InputRequestMap`,
  `InputResponseMap`, `NeedsInput`, `CallToolResult.InputRequests`,
  `Params.InputResponses`, `RequestState`) is itself non-deprecated — the
  sealed payload types that fit the sampling and roots slots are not.
- The manual-resumption path (`MultiRoundTripOptions.Disabled` + echoing
  `InputResponses`) does not escape this: the echoed responses must still
  deserialize into the sealed `InputResponse` types, and the client-side
  fulfillment dispatch and root-list configuration are deprecated-only
  in v1.8.0.
- Checked `v1.9.0-pre.1` (present in the local module cache, not the pin):
  identical SEP-2577 deprecation markers on the same surface. Re-pinning
  would not change the verdict.

### Verdict

**Pass 2 fails qualification on the client role, both required scenarios.**
Server role: pass on both axes. Elicitation: pass on both axes. Sampling
and roots client fulfillment: functionally pass only through Deprecated
APIs (`SA1019` with no exclusions), with no non-deprecated alternative in
SDK v1.8.0 (or v1.9.0-pre.1). B01 remains unresolved; the blocker is the
SEP-2577 deprecation lag between the SDK and the pinned conformance
requirements, which a from-scratch SDK would share because the
conformance pin, not the SDK, mandates the scenarios.

This is a substantive (not environmental) failure: the modern-only
composition ran to completion, elicitation passed, and the sampling/roots
failures are properties of the SDK's public surface, not of the harness.

### Limits

- The proof exercises only the two required input-required scenarios plus
  elicitation; it is not a full conformance run.
- The fulfillment middleware-bypass behavior (`receiving_callback_calls=0`)
  was observed for `fulfillInputRequest` dispatch; receiving middleware may
  still apply elsewhere.
- `v1.9.0-pre.1` was inspected read-only in the module cache; it is a
  pre-release and not the pin, so the verdict under Option A stands on
  v1.8.0. The deprecation window is at least twelve months per the SDK's
  own doc comments, so a future pin is unlikely to lift the markers soon.
- Scratch tests ran with an isolated `GOCACHE`; no repository module,
  lockfile, or product source was touched. All scratch paths are under the
  system temporary directory and are not committed.

### Pass-2 commands and outputs

- `go test ./... -count=1 -v` per copy (isolated env above). Copy A:
  `TestServerPublicCodec` PASS (3 subtests); `TestClientAutomaticFulfillment`
  PASS elicitation / FAIL sampling / FAIL roots. Copy B: all PASS. Copy C:
  `TestClientAutomaticFulfillment` FAIL sampling / FAIL roots (fixture
  corruption detected).
- `golangci-lint run --timeout 5m` per copy (pinned 2.15.0, no exclusions,
  default staticcheck). Copy A: `0 issues.` Copy B: `3 issues:
  staticcheck: 3` (list above).
- Identity scan: no local account name, hostname, or real-machine absolute
  path appears in any scratch file or this report; host paths are described,
  not quoted.

## Pass 3: Option D owned-loop qualification (2026-11-24)

Executed under the pass-3 amendment to
[0002-PLAN-build-schema-driven-mcp-framework.md](../decisions/0002-PLAN-build-schema-driven-mcp-framework.md)
after the owner's Option D ruling. Three fresh scratch copies under the
system temporary directory; the passing pass-2 server fixture reused
unchanged as the server side. No product source, dependency, lockfile, or
repository file was touched beyond this report.

### Composition under proof

The client owns the multi-round-trip loop: `MultiRoundTripOptions{Disabled:
true}` opts out of automatic fulfillment; `CallTool` returns the
input-required result; the owned loop reads `NeedsInput`,
`InputRequests.MarshalJSON` into owned DTOs, dispatches to owned handler
callbacks over owned types (sampling, roots, elicitation), marshals the
owned results, converts through `InputResponseMap.UnmarshalJSON`, and
retries with `Params.InputResponses` plus the echoed `Params.RequestState`.
No deprecated identifier is named in source, public API, or lint scope.

### Evidence

- **Manual-resumption proof (copy `go-mcp-option-d-wrapper`, test file
  sha256 `887039abd7c26bdf3141fa099ace5f165e85ddad15392755896c089f718ef660`).**
  `TestOwnedFulfillmentLoop`: PASS on all three scenarios (sampling, roots,
  elicitation). Each: two server tool invocations, one owned handler call,
  final result complete. Representative log line: `sampling: error=<nil>
  tool_calls=2 owned_handler_calls=1`.
- **Broken-handler control (copy `go-mcp-option-d-broken-handler`).** The
  same test with owned handlers returning wrong values (wrong model, wrong
  root URI, inverted elicitation confirmation) FAILS in all three scenarios
  with the fixture rejecting the application result — the wire carried the
  broken handler's values, proving the loop routes through owned code and
  not a silently succeeding default.
- **Lint gate.** Pinned golangci-lint 2.15.0, no exclusions, over the
  passing copy: `0 issues.` One errcheck finding (unchecked
  `session.Close`) surfaced during the run and was fixed in the proof, not
  suppressed; the negative-control copies were not linted (they are
  deliberate failures, and linting them proves nothing).
- **Discriminator pin (copy `go-mcp-option-d-broken-probe`).**
  `TestDiscriminatorPin` PASS: a sampling response lacking the `role` key
  fails `InputResponseMap` decoding loudly both at the codec directly and
  through the owned loop's conversion step, with the error `cannot determine
  InputResponse type: expected "action", "role", or "roots" key`. The copy
  also lints clean (`0 issues.`). G24's structural coupling is therefore
  observable as a loud failure, not silence.

### Commands and environment

- All runs offline (`GOWORK=off GOTOOLCHAIN=local GOPROXY=off
  GOSUMDB=off`), isolated `GOCACHE` per copy, go1.27.4 darwin/arm64.
- `go test -run TestOwnedFulfillmentLoop -v ./...` (passing copy):
  PASS, 3/3 subtests.
- `golangci-lint run` (passing copy, and the probe pin copy): `0 issues.`
- `golangci-lint run` on this host is version 2 and does not accept
  `--timeout`; the pass-2 report's `--timeout 5m` form belongs to the v1
  CLI. The lint run is unchanged in substance.

### Verdict

B01 is resolved by Option D as ruled on 2026-11-24: the two SEP-2577
conformance scenarios are satisfiable through the owned fulfillment loop
with zero deprecated symbols in source, public API, and lint scope, and the
G24 discriminator coupling is pinned by failing scenario tests. Option C
remains recorded as the fallback and is now moot for these scenarios.
Stage 0 qualification evidence is complete for the client role across
sampling, roots, and elicitation on protocol 2026-07-28.

### Limits

- The proof covers `CallTool` input-required results only; the PLAN's
  extension of the same mechanism to `GetPromptResult` /
  `ReadResourceResult` remains spec'd but unproven here.
- One errcheck finding in the first proof run was fixed in the proof; the
  fixed file is the hashed evidence above.
- Identity scan: no local account name, hostname, or real-machine absolute
  path appears in any scratch file or this section; host paths are
  described, not quoted.
