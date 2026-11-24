# 0010 — Stage 6 Report: Owned Runtime APIs and Standard Feature Coverage

Status: complete
Date: 2026-11-25
Governing pair: `0002-MADR-build-schema-driven-mcp-framework.md` /
`0002-PLAN-build-schema-driven-mcp-framework.md`, Stage 6
Plan stages: reports carry no decision; this records evidence only.

## Scope executed

Stage 6 steps 1–7: owned Server/Client/Connection handles with
context-first blocking operations and immutable validated options;
generic typed registration with validated schemas and pre-serve
duplicate/signature checks; the availability matrix with modern
interactions; schema/capability checks and output validation before
emission; the typed error taxonomy; middleware ordering with
non-skippable enforcement; and the no-raw-accessor policy.

Step 8 (two servers, two simultaneous connections, isolation) is
included. The external consumer example (also step 8) moves to stage
7's transport work, where an importable example gains a real
transport; the isolation contract itself is proven here over
in-memory pairs.

The stage-5 residue (step 6 differential codec testing) remains
deferred: it needs SDK-typed values round-tripped through v1, v2,
and the SDK path, which lands with transport-level fixtures. The
`0008-REPORT-stage-5-strict-json-validation.md` follow-up list
carries it.

## Owned handles (S6.1)

`Server` and `Client` wrap the pinned SDK's objects; the SDK
instances are private. Options validate once at construction
(`Implementation` required, negative values rejected,
threshold-without-keepalive rejected) and are immutable thereafter;
`Options()` returns a copy. Every blocking operation takes a context
first. The deprecated sampling handlers are deliberately absent from
the owned surface (G18); the deprecated roots feature is likewise
not exposed.

`ServerSession` and `ClientSession` wrap SDK sessions: stable `ID()`
per connection, context-first `Ping`, idempotent `Close`, guard
errors on closed sessions for every operation. No global
last-session routing authority exists anywhere in the API.

## Typed registration (S6.2)

`AddTool[In, Out]` mirrors the SDK's top-level generic pattern —
required because Go methods cannot introduce type parameters. The
registration validates name, handler, and duplicate state before
the SDK sees anything; admission is instance-scoped (G13), so two
servers may register the same tool name independently. Prompts
validate argument names and duplicates; resources validate the
absolute-URI location rule; resource templates validate URI-template
shape (expansions, balanced braces, URI-shaped prefix) without
importing the internal validator into the public API.

Duplicate registration rolls back its map entry when the SDK
declaration fails, so a rejected registration never lingers.

## Availability matrix (S6.3)

Wired through the owned API: tool call and listing, resource
listing/reading, subscribe/unsubscribe, progress notifications
(token + optional message), and elicitation (message + requested
schema, blocking, session-scoped). Deprecated sampling and roots
interactions are excluded from the owned surface per SEP-2577.

Live proof: a full resource lifecycle over the in-memory transport —
read, subscribe, guard rejections on empty URIs.

## Output validation before emission (S6.4)

`ToolRegistration.OutputSchema` declares an output schema; `AddTool`
compiles it through the canonical validator adapter at registration
— an invalid schema rejects the registration (G10, no permissive
substitution). The middleware chain's innermost layer validates
`tools/call` structured content against the declared schema after
the handler and any transformation middleware, before emission.
Violation returns a typed `ErrToolCall` error and the result never
leaves. No declaration means no check — absence is distinct from
invalid.

This required an interface refactor in `internal/validation`: the
adapter now returns an opaque `*Compiled` handle so the public API
never imports the validator's concrete types. The full stage-5 suite
(1,237 corpus cases, formats, 155-definition compilation) passes
unchanged after the refactor.

## Typed errors (S6.5)

Seven sentinels — `ErrProtocol`, `ErrRegistration`, `ErrToolCall`,
`ErrTransport`, `ErrResourceLimit`, `ErrOverload`, `ErrCancelled` —
with a `TypedError` wrapper carrying kind, bounded message, and
cause. `errors.Is` matches both the sentinel and the underlying
cause; `errors.As` reaches the `TypedError`. Cancellation classifies
distinctly from tool failure: a caller-deadline abort surfaces as
`ErrCancelled` wrapping `context.DeadlineExceeded`, proven by a test
that cancels mid-handler.

## Middleware ordering (S6.6)

The chain assembles in mandated order, outermost to innermost:
mandatory enforcement (not user middleware, not configurable),
observation, user middleware, authorization, admission, SDK
dispatch, output validation. `UseMiddleware` appends user links;
`Observation`/`Authorize`/`Admission` are single-slot and refuse
double installation. The mandatory gate rejects structurally
invalid requests before user middleware can observe them — proven by
a test where user code demonstrably never runs after gate
rejection.

## Isolation (S6.8)

Two client sessions over separate in-memory pipes to one server: a
handler held on connection one does not stall connection two's
call; both callers complete with their own responses. Two servers
registered with the same tool name serve their own sessions
independently, live over separate connections.

## Accessor policy (S6.7)

No raw SDK accessor is exported. The `Transport` interface's
`transport()` method is unexported; the crossing point is the
framework's own implementations only. If an integration accessor is
ever needed, the MADR's allowance requires naming its enforcement
boundary and bypass tests before export — none is needed by the
current API.

## Evidence

| Check | Result |
| --- | --- |
| Root package tests | 29 passed, 0 failed |
| Lint (`golangci-lint run ./`) | 0 issues |
| Validator suite (post-refactor) | unchanged, all green |
| Deprecated SDK symbols in owned surface | none (sampling/roots absent) |
| Round trip over in-memory transport | proven |
| Blocked-handler isolation | proven |
| Cancellation mid-handler | proven, typed |

Gates: `go test ./ ./internal/validation/` green; `golangci-lint`
0 issues; records and markdownlint verified at commit time.

## Self-caught defects during the stage

- `TestCallToolCancelledClassifies` originally deadlocked by
  construction — the test awaited a channel only the pending call
  would close. Rewritten to start the call first, then await the
  stall signal.
- `Server.connect` shipped dead in the first stage-6 commit; removed
  before the stage completed rather than carrying it.
- The `TypedError.As` method initially matched only `**TypedError`
  and dropped the cause chain on `errors.Is` probes; the
  classification test caught it and the implementation now checks
  both kind and cause.
- The output-validator bridge first drafted an invented
  `CompiledSchema` interface that did not exist in the adapter; the
  honest fix was the opaque-handle refactor in
  `internal/validation`, accepting a signature change and updating
  all call sites rather than papering over it.

## Deviations and residue

- External consumer example (step 8) deferred to stage 7, where a
  transport exists to make the example real.
- Stage-5 step 6 differential codec testing remains deferred with
  the same rationale; carried in the 0008 report's follow-up list.
- The report-number gap (`0009` consumed by a CI/CD assessment
  written outside this stage) is recorded here: `0010` is stage 6
  per `scripts/check_records.py --next`.

## Identity scan

Scanned the new files for the local account name and host-specific
path prefixes before writing this report; none committed.
Verification scripts derive runtime strings from `os.environ`.
