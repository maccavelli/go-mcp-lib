# 0008 — Stage 5 Report: Strict JSON Codec and Canonical Validator

Status: complete
Date: 2026-11-25
Governing pair: `0002-MADR-build-schema-driven-mcp-framework.md` /
`0002-PLAN-build-schema-driven-mcp-framework.md`, Stage 5
Plan stages: reports carry no decision; this records evidence only.

## Scope executed

Stage 5 steps 1–5 and 7: owned codec entry points over
`encoding/json/v2` + `encoding/json/jsontext`; raw-byte checks before
any lossy decode; the private validator adapter over the pinned
`github.com/santhosh-tekuri/jsonschema/v6` v6.0.3; vendored
test-suite corpus and format fixtures with hashes and license; the
application-schema profile's protocol-location rules; and adversarial
mutation plants. Step 6 (differential codec testing against SDK
values) remains open and moves to the stage-5 follow-up list below —
it requires the runtime API surface of stage 6 to construct
reviewed SDK values.

## Codec (`codec/`)

`ValidateRaw` checks raw bytes before decode: duplicate and
escaped-equivalent member names (rejected natively by the jsontext
decoder and surfaced through the classified `codec.Error`),
UTF-8 validity, trailing values, depth, frame size, and numeric
token/exponent bounds. `DuplicateNames` extracts the offending name
from the decoder's native rejection. Absence/null/empty/zero remain
distinct; numbers carry exact tokens.

Bounds (frozen defaults, `codec.DefaultBounds()`):

| Bound | Value | Tested at and one past |
| --- | --- | --- |
| Frame | 4 MiB | yes |
| Depth | 256 | yes |
| Numeric token | 256 bytes | yes |
| Exponent magnitude | 1,024 | yes |

12 tests: accepts, trailing, depth, frame, token, exponent sign and
leading-zero forms, duplicate names (plain, escaped-equivalent,
nested), no false duplicates, invalid UTF-8, and the
absence/null/empty/zero distinction with a `json.Number` exact-token
assertion.

Key implementation fact: the jsontext decoder rejects duplicate and
escaped-equivalent names natively (`jsontext: duplicate object member
name "a"`), including `"\u0061"` against `"a"`. The owned walker
surfaces that rejection with position and name extraction rather than
re-implementing equivalence detection.

## Canonical validator (`internal/validation/`)

The adapter wraps the pinned `jsonschema/v6` v6.0.3 (root `go.mod`
require added at first import, consistent with the module-graph
honesty rule). Draft 2020-12 is forced via `DefaultDraft`; local
references resolve through resources the caller registers — no
network, ever. Compilations cache per immutable identity with a
retention bound that refuses to grow silently; concurrent access is
mutex-guarded. Numbers stay exact: schemas and instances decode
through `exactUnmarshal` with `UseNumber`, and
`TestExactNumbers` asserts the decoded instance is `json.Number`, not
`float64`, before validating the exact-minimum comparison.

Pinned-schema proofs:

- 155/155 definitions of `schema/inputs/2026-07-28/schema.json`
  compile individually through the adapter as Draft 2020-12
  subschemas, each wrapped so internal `$defs` references resolve
  through the registered full document.
- Representative instances validate: a minimal `Tool` (name +
  inputSchema) passes; `Tool` without `name` is rejected.

## Vendored corpus (no network in tests)

Source: `json-schema-org/JSON-Schema-Test-Suite` at commit
`83e866b46c9f9e7082fd51e83a61c5f2145a1ab7`, the commit pinned in
`tools/versions.json`.

- `internal/validation/testdata/draft2020-12/` — 46 suite files plus
  the `remotes/` tree those files reference via `localhost:1234`,
  registered as local resources; per-file SHA-256 in
  `corpus-manifest.json`; suite `LICENSE` vendored.
- `internal/validation/testdata/formats/` — 15 format files
  (date-time, date, duration, email, hostname, json-pointer, regex,
  relative-json-pointer, time, uri, uri-reference, uri-template,
  uuid, ipv4, ipv6), 380 fixture cases; SHA-256 hashes in
  `formats-manifest.json`; suite `LICENSE` vendored.

Re-vendoring is scripted: `scripts/vendor_testsuite.py` and
`scripts/vendor_formats.py` (idiomatic Python, per the tooling
mandate).

## Results

| Check | Result |
| --- | --- |
| Suite corpus (draft2020-12) | 1237 passed, 0 failed, 18 annotation-skips |
| Format fixtures | 380 cases, all passing, per-format tallies in test log |
| 155 definition compilations | 155/155 |
| Lint (`golangci-lint run ./...`) | 0 issues |
| Mutation plants | 3/3 caught |

Annotation-skips (18 cases) are deliberate profile exclusions —
content encodings, legacy `dependencies`, unknown-keyword handling —
listed in the harness rather than silently passing.

## Application-schema profile

Accepted: Draft 2020-12 with the standard vocabulary, plus the
framework's location rules. Regex semantics are Go RE2 (host
language); patterns needing backreferences or lookaround are rejected
at declaration with an actionable pointer
(`RejectUnknownRegexSemantics`) instead of silently applying
incompatible behavior. The framework's own duration checker obeys
this rule — it validates structurally, without lookaround.

Protocol-location rules (`internal/validation/locations.go`):

- `Root.uri` — required, absolute URI, not a relative reference.
- URI locations — absolute.
- URI-template locations — URI-shaped with `{var}`, `{var:NNN}`,
  `{var*}` varspecs, balanced braces; relative templates rejected at
  these locations (distinguished from the `uri-template` *format*,
  which permits relative).
- Base64 content — canonical, padded, round-trip exact.

Format checkers were hardened against the suite's adversarial cases:
offset-aware leap-second validity (RFC 3339 §5.7 — `:60` valid only
when the UTC-adjusted time is 23:59), quoted email local parts with
embedded `@`, address literals (`[127.0.0.1]`, `[IPv6:::1]`),
dot-atom rules, backslash rejection in URI references, and RFC 6570
varspec shape including prefix modifiers.

## Mutation plants (step 7)

Run via `scripts/verify_mutations.py` against scratch copies; the
repository is never mutated:

1. Numeric token bound removed → `TestNumericTokenBound` fails.
2. `UseNumber` dropped (float64 rounding) → `TestExactNumbers` fails
   with `exact decode produced float64, not json.Number`.
3. Duplicate-name classification bypassed →
   `TestValidateRawClassifiesDuplicates` fails.

All three caught; re-verified after the dead-code lint cleanup.

## Self-caught defects

- `TestExactNumbers` originally constructed values with a helper that
  produced `json.Number` regardless of decoder settings — the rounding
  mutation sailed through. The honest fix decodes wire bytes through
  `exactUnmarshal` and asserts the type directly.
- My first duration regex used a lookahead — the exact construct the
  profile forbids in user schemas. Rewritten structurally.
- Fixed-offset slicing in the time/date-time checkers panicked on
  short strings before the pattern guaranteed lengths; replaced with
  structural parsing.

## Follow-ups (stage 5 residue)

- Step 6 differential testing (owned values vs reviewed SDK values
  under v1, v2, and the SDK codec path) — requires stage 6's runtime
  API surface to construct SDK-typed values; fold into stage 6
  verification.

## Identity scan

Scanned the new files for the local account name and host-specific
paths before writing this report; none committed. Verification
scripts derive any runtime-specific strings from `os.environ` at run
time, never embedding them.
