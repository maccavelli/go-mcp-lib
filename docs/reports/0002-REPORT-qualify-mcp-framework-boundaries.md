---
status: complete
date: 2026-11-22
associated-madr: 0002-MADR-build-schema-driven-mcp-framework.md
---

# Qualify MCP framework transport, schema, and performance boundaries

Associated decision: [0002-MADR-build-schema-driven-mcp-framework.md](../decisions/0002-MADR-build-schema-driven-mcp-framework.md).

This report records diagnostic evidence for G08, G20, and G21. It measures a
scratch prototype against pinned dependencies. It does not claim that the
unimplemented go-mcp-lib runtime or generator passed a release qualification.
The evidence resolves the architecture's feasibility and supplies numerical
acceptance criteria for its implementation.

## Inputs and execution

The assessed MCP schema is revision `2026-07-28`, upstream commit
`ac00313ff7f5e9f47b0e0d1ebdf9c0ad6411380b`, SHA-256
`907dcebce6aaeb16c21421ffef8c207093d88b9442fee340658a99cacb06b551`.
It contains 155 definitions and 181,976 bytes. The SDK is v1.8.0; the toolchain
is Go 1.27.4 and the scratch module declares Go 1.27.0.

Functional probes ran on Darwin/arm64. Benchmarks ran through the host queue
on Darwin/arm64 and through fleet selection and its queue on Windows/amd64.
The fleet dispatcher verified the remote Go toolchain. The scratch module name
remained `github.com/maccavelli/mcplib` from its copied dependency manifest;
no mcplib production package was copied, imported, or tested by these probes.

The diagnostic build sequence exposed two setup errors before functional
qualification: benchmark syntax, followed by the copied Go 1.26 module floor.
The third-build gate was prompted and the owner instructed “resume”. Subsequent
functional failures were retained as findings: a wrong request-ID type in the
HTTP harness, SDK validator numeric behavior, and SSE failure propagation.
The request ID was corrected to the public API's accepted numeric type; no
payload assertion was weakened. The validator and SSE designs were then corrected.

The final functional command exited 0:

```text
go test -mod=readonly -run Test -count=1 -v .
ASSERTION FIXTURES=421 DEFINITIONS=155
OFFICIAL FORMAT FIXTURES=36
ok  github.com/maccavelli/mcplib  0.588s
```

`SCHEMA_SUITE` selected the pinned SDK-aligned validator's local test corpus.
`GOTOOLCHAIN=local` and `GOPROXY=off` were set for functional tests after explicit
dependency preparation. One missing transitive dependency was fetched during
that preparation, not during the offline qualification run.

## G08 transport evidence

All receiving paths have a public interception point before JSON-RPC decoding:

| Receiving path | Interception | Evidence |
| :--- | :--- | :--- |
| Either end of subprocess stdio | Guard the reader passed to `mcp.IOTransport` | Valid response delivered; duplicate names and invalid UTF-8 rejected; unguarded controls accepted both malformed inputs |
| HTTP server requests | Outer `http.Handler` validates and restores the request body before invoking the SDK handler | Unguarded duplicate-method request reached the SDK and returned 200; guarded request returned 400 with zero SDK handler invocations |
| HTTP client JSON responses | Supplied `http.Client.Transport` checks the response body before returning it to the SDK | Valid response delivered; duplicate names and invalid UTF-8 rejected |
| HTTP client SSE responses | Supplied HTTP transport wraps the body with a bounded event reader | Valid response delivered; corrupt event following a valid event with an ID rejected; the HTTP request count remained one |

The raw validator uses stable `jsontext.Value.IsValid`, whose default contract
rejects duplicate names, invalid UTF-8, and invalid Unicode escapes, followed by
a depth bound. The probes cover escaped duplicate names, trailing JSON values,
oversized payloads, and excessive depth. A one-byte read proves that an invalid
frame releases zero bytes. SSE cases cover LF, CRLF, CR, multiple data lines,
comments, event IDs, and retry fields. Additional cases cover the SDK's
whitespace normalization of event names, preventing a classification bypass.
The diagnostic SSE envelope normalizes
line endings; it validates the assembled data payload before any SDK decoding.

An SSE reader error alone is insufficient for fail-stop behavior. SDK v1.8.0
may interpret it as an interrupted stream and attempt resumption. The qualified
reader records a terminal error, closes its source, and invokes a connector-
installed callback that closes the original SDK connection. The original
connection is returned unchanged to the SDK so its private update hooks survive.
The test retains five allowed SDK retries and introduces a preceding event ID;
no second HTTP request occurs. The SDK may expose EOF or a reconnect-cancelled
error, so the canonical API must retain and return its own first validation
failure rather than depending on the SDK's error wording.

Relevant SDK source is `mcp/transport.go`, `newIOConnLimited` and `ioConn.Read`,
and `mcp/streamable.go`, `ephemeralConnectOpts`, `servePOST`, `handleJSON`,
`processStream`, `handleSSE`, and `Close`, all at v1.8.0. Wrapping an already
parsed `mcp.Connection.Read` message is too late for duplicate detection.

The current protocol uses POST-scoped SSE, including `subscriptions/listen`;
legacy GET-stream support is unnecessary. The
[stable HTTP binding](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http)
and [stdio binding](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio)
identify the applicable framing and directions. This qualification does not
add an HTTP listener to subprocess artifact delivery.

## G21 validator evidence

The pinned schema's complete vocabulary inventory is:

| Keyword | Occurrences | Responsibility |
| :--- | ---: | :--- |
| `$schema` | 1 | Dialect selection |
| `$defs` | 1 | Local definition registry |
| `$ref` | 277 | Reference resolution |
| `type` | 517 | Assertion |
| `const` | 88 | Assertion |
| `enum` | 15 | Assertion |
| `properties` | 149 | Object applicator |
| `required` | 128 | Assertion |
| `additionalProperties` | 20 | Object applicator/assertion |
| `items` | 41 | Array applicator |
| `maxItems` | 1 | Assertion |
| `minimum` | 11 | Assertion |
| `maximum` | 4 | Assertion |
| `allOf` | 3 | Composition |
| `anyOf` | 24 | Composition |
| `format` | 17 | Annotation plus explicitly selected MCP semantic checks |
| `description` | 391 | Annotation |

Inventory traversal distinguishes schema keywords from property names. For
example, an object property named `enum` is not an `enum` assertion. No `pattern`
assertion occurs in this pinned schema. This inventory is about protocol schema
nodes; application-provided tool schemas require their own inventory and
qualification and cannot silently inherit a claim of universal support.

The SDK-aligned `github.com/google/jsonschema-go v0.4.3` failed the required
lossless-number contract. Passing `json.Number("1")` to an integer schema
reports string type. Loading `{"const":9007199254740993}` rounds the schema
constant and accepts `9007199254740992` while rejecting the intended value.
Source corroboration is `jsonschema/util.go`, `jsonType`, and
`jsonschema/schema.go`, `UnmarshalJSON`'s `unmarshalAnyPtr`.
The lossless projection produced 57 mismatches in the 421 selected fixtures.
This is not a claim that its normal float64-based test suite fails.

Under the owner's D04 fallback, the qualified replacement behind the internal
adapter is `github.com/santhosh-tekuri/jsonschema/v6 v6.0.3`. Its lossless JSON
loader and compiler passed all 421 selected fixtures, all 155 MCP definition
compilations, and exact/adjacent-integer cases for const, enum, and numeric bounds.
The SDK dependency remains unchanged and may retain its own validator internally.
The framework validates raw application payloads before delegation; its adapter
must not feed rounded values into its own canonical checks.

The assertion fixtures come from the fourteen corresponding files in
`github.com/google/jsonschema-go@v0.4.3/jsonschema/testdata/draft2020-12`.
That corpus's README identifies upstream JSON Schema Test Suite commit
`83e866b46c9f9e7082fd51e83a61c5f2145a1ab7`. The file list is `defs`, `ref`,
`type`, `const`, `enum`, `properties`, `required`, `additionalProperties`,
`items`, `maxItems`, `minimum`, `maximum`, `allOf`, and `anyOf`, each `.json`.
Reference fixtures also use the pinned corpus's local `remotes` and meta-schemas.
The ordered `filename SHA256` manifest has SHA-256
`506e858547ead334368287eb18940b3d72801030d8cb058bf6020ea0e9ff6585`.
No reference was fetched over the network during these tests.

Formats comprise three `byte`, twelve `uri`, and two `uri-template` locations.
The explicit semantic layer uses an absolute URI syntax check, the SDK's
`uritemplate/v3 v3.0.2` parser, and strict standard base64 decoding. It passed
36 official URI/URI-template fixtures plus positive and negative base64 cases.
Tests also compile the format handlers into the replacement validator and check
valid acceptance and invalid rejection. Format syntax is separate from access
policy: `Root.uri` still requires the protocol's `file://` restriction, and URI
validity never authorizes fetching a resource.

The optional format fixtures were fetched from the same upstream test-suite
commit. Their SHA-256 values are:

```text
uri.json          56eafa51d31e29d993cbaa0e53d2b93d196f162317896a943231befd442835dc
uri-template.json 02753aa3f3e38c29b3ed0231305ba77f0427f922aca9b974ed3351da07b6134e
```

Draft 2020-12 distinguishes format annotations from required assertions; the
explicit MCP checks are intentional policy, not a claim that every annotation
must be asserted. See the
[validation vocabulary](https://json-schema.org/draft/2020-12/json-schema-validation)
and [replacement-validator API](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6@v6.0.3).
Future schema revisions or added tool-schema vocabularies must fail qualification
if their required behavior is unsupported. No catch-all permissive fallback is
part of this resolution.

## G20 measurements

The benchmark compares upstream `jsonrpc.DecodeMessage` against the same decode
preceded by the strict JSON and depth check. Fixtures contain a JSON-RPC result
with a text payload at four sizes. Both paths decode identical bytes. These
figures isolate the cost of the extra raw boundary check; they exclude HTTP,
SSE framing, schema validation, user handlers, and the unimplemented generator.

Each row is the median of five samples. Times are microseconds per operation;
byte and allocation deltas compare the corresponding medians.

| Run | Frame bytes | SDK µs | Checked µs | Added µs | Added B/op | Added allocs/op |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: |
| Darwin/arm64, default CPU count | 1,069 | 5.836 | 8.456 | 2.620 | 27 | 0 |
| Darwin/arm64, default CPU count | 65,581 | 142.746 | 269.515 | 126.769 | 157 | 1 |
| Darwin/arm64, default CPU count | 262,189 | 593.409 | 1088.818 | 495.409 | 995 | 5 |
| Darwin/arm64, default CPU count | 1,048,493 | 1931.008 | 3941.796 | 2010.788 | 1,715 | 8 |
| Windows/amd64, default CPU count | 1,069 | 6.019 | 7.711 | 1.692 | 88 | 0 |
| Windows/amd64, default CPU count | 65,581 | 100.880 | 159.505 | 58.625 | 757 | 2 |
| Windows/amd64, default CPU count | 262,189 | 369.960 | 2523.603 | 2153.643 | 2,623 | 4 |
| Windows/amd64, default CPU count | 1,048,493 | 4860.512 | 8804.585 | 3944.073 | 4,460 | 7 |
| Windows/amd64, GOMAXPROCS=1 | 1,069 | 6.676 | 7.377 | 0.701 | 1 | 0 |
| Windows/amd64, GOMAXPROCS=1 | 65,581 | 370.456 | 600.735 | 230.279 | 16 | 0 |
| Windows/amd64, GOMAXPROCS=1 | 262,189 | 1489.516 | 2453.699 | 964.183 | 77 | 1 |
| Windows/amd64, GOMAXPROCS=1 | 1,048,493 | 4782.199 | 8596.591 | 3814.392 | 295 | 5 |

The default Windows run had a large timing change at 262,189 bytes. The repeated
GOMAXPROCS=1 run did not reproduce the 6.821 ratio, but still showed variability
in its smallest case. All samples and both runs are retained below. The cause
was not isolated, and no sample was removed to manufacture a passing result.
There is no speedup claim and no statistically established tail-latency claim.

The measured boundary passes the following proposed implementation budget for
each size on both hosts, including the first Windows run:

```text
median(checked ns/op) - median(SDK ns/op) <= 5,000 + 12 × frame_bytes
median(checked B/op) - median(SDK B/op) <= 8,192
median(checked allocs/op) - median(SDK allocs/op) <= 16
```

All twelve run/size combinations satisfy those bounds. The bound leaves
headroom above observed cost; it is an engineering acceptance target, not an
MCP requirement or a universal hardware guarantee. The budget check was also
fed a copied benchmark log with the checked-time samples deliberately raised.
It rejected the 1,069-byte case with a 17,829 ns delta against the 17,828 ns bound.
Implemented gates
must compare the same host, toolchain, fixture, and execution settings and
retain all samples. No failed correctness check can be offset by faster timing.

The MADR separately specifies resource defaults and full-implementation budgets.
Those are design limits to test during implementation, not measurements of
this scratch prototype. Measuring an unimplemented framework is impossible;
closing G20 means a defined, evidence-informed contract replaces unspecified
numbers and acceptance rules.

## Deliberately broken controls

A separate scratch copy disabled the strict syntactic guard and removed the
required-property constraint from a schema fixture. Running the original
assertions against that copy exited 1 with all five expected failures:

```text
FAIL strict rejection duplicate
FAIL strict rejection escaped_duplicate
FAIL strict rejection utf8
FAIL strict rejection surrogate
FAIL required plant escaped
```

The working tree and the successful prototype were untouched by these plants.
The final unmodified probes passed. The defective SDK candidate's behavior is
retained as a separate reproduction alongside the replacement's unchanged
precision requirements.

## Reproduction

Extract the Go source and module manifest below into a new scratch directory.
Use the published module checksums below and normal Go module verification.
Fetch the immutable MCP schema linked in the MADR as `mcp.schema.json` and
verify its digest. Fetch `uri.json` and `uri-template.json` from
`tests/draft2020-12/optional/format/` at the test-suite commit above and verify
their digests. `SCHEMA_SUITE` is the local directory of the pinned
`google/jsonschema-go` module's `jsonschema` package, not a floating checkout.

After explicit dependency preparation, run the tests with `GOTOOLCHAIN=local`
and `GOPROXY=off`. The benchmark commands were:

```text
go test -mod=readonly -run Test -count=1 -v .
go test -mod=readonly -run ^$ -bench ^BenchmarkBoundary$ -benchtime=100ms -count=5 .
go test -mod=readonly -run ^$ -bench ^BenchmarkBoundary$ -benchtime=300ms -count=5 -cpu=1 .
```

Use the fleet queue wrappers required by repository policy for benchmarks.
The diagnostic source is intentionally included in the report for replay and
review. The final SSE classification regression was added after the benchmarks;
the benchmark and strict-JSON functions were unchanged. It is not a production transport or validator implementation.

### Module manifest

```go
module github.com/maccavelli/mcplib

go 1.27.0

toolchain go1.27.4

require (
    github.com/google/jsonschema-go v0.4.3
    github.com/modelcontextprotocol/go-sdk v1.8.0
    github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
    github.com/yosida95/uritemplate/v3 v3.0.2
)

require (
    github.com/segmentio/asm v1.1.3 // indirect
    github.com/segmentio/encoding v0.5.4 // indirect
    golang.org/x/oauth2 v0.36.0 // indirect
    golang.org/x/sync v0.22.0 // indirect
    golang.org/x/sys v0.47.0 // indirect
    golang.org/x/text v0.14.0 // indirect
    golang.org/x/time v0.15.0 // indirect
    golang.org/x/tools v0.49.0 // indirect
)
```

### Module checksums

```text
github.com/dlclark/regexp2 v1.11.0 h1:G/nrcoOa7ZXlpoa/91N3X7mM3r8eIlMBBJZvsz/mxKI=
github.com/dlclark/regexp2 v1.11.0/go.mod h1:DHkYz0B9wPfa6wondMfaivmHpzrQ3v9q8cnmRbL6yW8=
github.com/golang-jwt/jwt/v5 v5.3.1 h1:kYf81DTWFe7t+1VvL7eS+jKFVWaUnK9cB1qbwn63YCY=
github.com/golang-jwt/jwt/v5 v5.3.1/go.mod h1:fxCRLWMO43lRc8nhHWY6LGqRcf+1gQWArsqaEUEa5bE=
github.com/google/go-cmp v0.7.0 h1:wk8382ETsv4JYUZwIsn6YpYiWiBsYLSJiTsyBybVuN8=
github.com/google/go-cmp v0.7.0/go.mod h1:pXiqmnSA92OHEEa9HXL2W4E7lf9JzCmGVUdgjX3N/iU=
github.com/google/jsonschema-go v0.4.3 h1:/DBOLZTfDow7pe2GmaJNhltueGTtDKICi8V8p+DQPd0=
github.com/google/jsonschema-go v0.4.3/go.mod h1:r5quNTdLOYEz95Ru18zA0ydNbBuYoo9tgaYcxEYhJVE=
github.com/modelcontextprotocol/go-sdk v1.8.0 h1:KIvahhYqwtbeniWVPs3TcXEA7b8jEtwfBpOTAI+Urx4=
github.com/modelcontextprotocol/go-sdk v1.8.0/go.mod h1:dL7u98E/zjJTGzEq+j30jQ8K2k1mb6LeAH4inEcSGts=
github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 h1:1EYB5IzjZawrrnELUi78f9fPu57HuXjmddZPjrls/28=
github.com/santhosh-tekuri/jsonschema/v6 v6.0.3/go.mod h1:JXeL+ps8p7/KNMjDQk3TCwPpBy0wYklyWTfbkIzdIFU=
github.com/segmentio/asm v1.1.3 h1:WM03sfUOENvvKexOLp+pCqgb/WDjsi7EK8gIsICtzhc=
github.com/segmentio/asm v1.1.3/go.mod h1:Ld3L4ZXGNcSLRg4JBsZ3//1+f/TjYl0Mzen/DQy1EJg=
github.com/segmentio/encoding v0.5.4 h1:OW1VRern8Nw6ITAtwSZ7Idrl3MXCFwXHPgqESYfvNt0=
github.com/segmentio/encoding v0.5.4/go.mod h1:HS1ZKa3kSN32ZHVZ7ZLPLXWvOVIiZtyJnO1gPH1sKt0=
github.com/yosida95/uritemplate/v3 v3.0.2 h1:Ed3Oyj9yrmi9087+NczuL5BwkIc4wvTb5zIM+UJPGz4=
github.com/yosida95/uritemplate/v3 v3.0.2/go.mod h1:ILOh0sOhIJR3+L/8afwt/kE++YT040gmv5BQTMR2HP4=
golang.org/x/oauth2 v0.36.0 h1:peZ/1z27fi9hUOFCAZaHyrpWG5lwe0RJEEEeH0ThlIs=
golang.org/x/oauth2 v0.36.0/go.mod h1:YDBUJMTkDnJS+A4BP4eZBjCqtokkg1hODuPjwiGPO7Q=
golang.org/x/sync v0.22.0 h1:SZjpbeLmrCk4xhRSZFNZW5gFUeCeFgjekvI/+gfScek=
golang.org/x/sync v0.22.0/go.mod h1:9xrNwdLfx4jkKbNva9FpL6vEN7evnE43NNNJQ2LF3+0=
golang.org/x/sys v0.47.0 h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=
golang.org/x/sys v0.47.0/go.mod h1:4GL1E5IUh+htKOUEOaiffhrAeqysfVGipDYzABqnCmw=
golang.org/x/text v0.14.0 h1:ScX5w1eTa3QqT8oi6+ziP7dTV1S2+ALU0bI+0zXKWiQ=
golang.org/x/text v0.14.0/go.mod h1:18ZOQIKpY8NJVqYksKHtTdi31H5itFRjB5/qKTNYzSU=
golang.org/x/time v0.15.0 h1:bbrp8t3bGUeFOx08pvsMYRTCVSMk89u4tKbNOZbp88U=
golang.org/x/time v0.15.0/go.mod h1:Y4YMaQmXwGQZoFaVFk4YpCt4FLQMYKZe9oeV/f4MSno=
golang.org/x/tools v0.49.0 h1:3NI7VXzL9+1WZD52Dx2ttoPwD5DWrFGpl9mFZDlmisI=
golang.org/x/tools v0.49.0/go.mod h1:SJNXV9DBKT0UbdttsQjbfJlAE/q+y36++zo3uL3N0Oo=
```

### Diagnostic source

Save this exact block as `qualification_test.go`. SHA-256:
`7e9a8796cd0c1d3acb5ecf81584f76a33287c4651dad8367c10d54fbd187d4ec`.

```go
package qualification

import (
 "bufio"
 "bytes"
 "context"
 "encoding/base64"
 "encoding/json"
 "encoding/json/jsontext"
 jsonv2 "encoding/json/v2"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/http/httptest"
 "net/netip"
 "net/url"
 "os"
 "path/filepath"
 "regexp"
 "strconv"
 "strings"
 "sync/atomic"
 "testing"
 "time"
 "unicode/utf8"

 "github.com/google/jsonschema-go/jsonschema"
 exact "github.com/santhosh-tekuri/jsonschema/v6"
 "github.com/modelcontextprotocol/go-sdk/jsonrpc"
 "github.com/modelcontextprotocol/go-sdk/mcp"
 "github.com/yosida95/uritemplate/v3"
)

const frameLimit = 1024 * 1024
const depthLimit = 64
var errStrict = errors.New("strict JSON rejection")

func strictJSON(b []byte) error {
 if len(b)>frameLimit { return fmt.Errorf("frame limit: %d",len(b)) }
 if !jsontext.Value(b).IsValid() { return errStrict }
 depth:=0; quoted:=false; escaped:=false
 for _,c:=range b {
  if quoted { if escaped {escaped=false} else if c=='\\' {escaped=true} else if c=='"' {quoted=false};continue }
  switch c {case '"':quoted=true;case '{','[':depth++;if depth>depthLimit{return errors.New("depth limit")};case '}',']':depth--}
 }
 return nil
}

// A diagnostic reader, not production code. It releases only validated frames.
type guardedReader struct { r *bufio.Reader; source io.ReadCloser; pending []byte; sse bool; skipLF bool; reject func(error); terminal error }
func guard(r io.ReadCloser,sse bool)*guardedReader{return &guardedReader{r:bufio.NewReader(r),source:r,sse:sse}}
func(g *guardedReader)Close()error{return g.source.Close()}
func(g *guardedReader)line()(string,int,error){
 var b []byte; consumed:=0
 for {c,e:=g.r.ReadByte();if e!=nil {if len(b)>0 && e==io.EOF{return "",consumed,io.ErrUnexpectedEOF};return "",consumed,e};consumed++
  if g.skipLF {g.skipLF=false;if c=='\n'{continue}}
  if c=='\r'{g.skipLF=true;return string(b),consumed,nil};if c=='\n'{return string(b),consumed,nil}
  b=append(b,c);if len(b)>frameLimit{return "",consumed,errors.New("line limit")}
 }
}
func(g *guardedReader)frame()([]byte,error){
 if !g.sse {
  line,n,e:=g.line();if e!=nil{return nil,e};if n>frameLimit{return nil,errors.New("frame limit")}
  b:=[]byte(line);if e:=strictJSON(b);e!=nil{return nil,e};return append(b,'\n'),nil
 }
 var out strings.Builder;var data []string; event:="";size:=0
 for {
  line,n,e:=g.line();size+=n;if e!=nil{return nil,e};if size>frameLimit{return nil,errors.New("event limit")}
  if !utf8.ValidString(line){return nil,errStrict};out.WriteString(line);out.WriteByte('\n')
  if line==""{
   payload:=strings.Join(data,"\n")
   if (strings.TrimSpace(event)==""||strings.TrimSpace(event)=="message")&&payload!="" {if e:=strictJSON([]byte(payload));e!=nil{return nil,e}}
   return []byte(out.String()),nil
  }
  key,value,_:=strings.Cut(line,":");value=strings.TrimPrefix(value," ")
  switch key{case "data":data=append(data,value);case "event":event=value}
 }
}
func(g *guardedReader)Read(b []byte)(int,error){
 if len(b)==0{return 0,nil};if g.terminal!=nil{return 0,g.terminal};if len(g.pending)==0{v,e:=g.frame();if e!=nil{g.terminal=e;if e!=io.EOF{g.source.Close();if g.reject!=nil{g.reject(e)}};return 0,e};g.pending=v}
 n:=copy(b,g.pending);g.pending=g.pending[n:];return n,nil
}

type rtFunc func(*http.Request)(*http.Response,error)
func(f rtFunc)RoundTrip(r *http.Request)(*http.Response,error){return f(r)}
func checkedHTTP(next http.RoundTripper,reject func(error))http.RoundTripper{return rtFunc(func(r *http.Request)(*http.Response,error){
 if r.Body!=nil&&r.Method=="POST" {b,e:=io.ReadAll(io.LimitReader(r.Body,frameLimit+1));r.Body.Close();if e!=nil{return nil,e};if e=strictJSON(b);e!=nil{return nil,e};r=r.Clone(r.Context());r.Body=io.NopCloser(bytes.NewReader(b))}
 resp,e:=next.RoundTrip(r);if e!=nil{return nil,e}
 ct:=strings.Split(resp.Header.Get("Content-Type"),";")[0]
 if ct=="text/event-stream" {g:=guard(resp.Body,true);g.reject=reject;resp.Body=g} else if ct=="application/json" {
  // Guard the whole body before exposing any bytes to the SDK.
  b,e:=io.ReadAll(io.LimitReader(resp.Body,frameLimit+1));resp.Body.Close();if e!=nil{return nil,e};if e=strictJSON(b);e!=nil{return nil,e};resp.Body=io.NopCloser(bytes.NewReader(b))
 }
 return resp,nil
})}
func checkedHandler(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
 if r.Method=="POST" {b,e:=io.ReadAll(io.LimitReader(r.Body,frameLimit+1));r.Body.Close();if e!=nil{http.Error(w,"read",400);return};if e=strictJSON(b);e!=nil{http.Error(w,"invalid JSON",400);return};r.Body=io.NopCloser(bytes.NewReader(b))}
 next.ServeHTTP(w,r)
})}

func TestStrictTransport(t *testing.T){
 good:=`{"jsonrpc":"2.0","id":1,"result":{"value":1}}`
 bads:=map[string][]byte{
  "duplicate":[]byte(`{"jsonrpc":"2.0","id":1,"result":{"value":1,"value":2}}`),
  "escaped_duplicate":[]byte(`{"jsonrpc":"2.0","id":1,"result":{"value":1,"\u0076alue":2}}`),
  "utf8":append(append([]byte(`{"jsonrpc":"2.0","id":1,"result":{"value":"`),0xff),[]byte(`"}}`)...),
  "surrogate":[]byte(`{"jsonrpc":"2.0","id":1,"result":{"value":"\ud800"}}`),
  "trailing":[]byte(good+`{}`),
  "oversize":[]byte(`{"s":"`+strings.Repeat("x",frameLimit)+`"}`),
  "depth":[]byte(strings.Repeat("[",depthLimit+1)+"0"+strings.Repeat("]",depthLimit+1)),
 }
 for name,b:=range bads{t.Run(name,func(t *testing.T){if strictJSON(b)==nil{t.Fatal("FAIL strict rejection",name)}})}
 t.Run("stdio_sdk",func(t *testing.T){
  for name,b:=range map[string][]byte{"valid":[]byte(good),"duplicate":bads["duplicate"],"utf8":bads["utf8"]}{
   for _,checked:=range []bool{false,true}{
    r:=io.NopCloser(bytes.NewReader(append(bytes.Clone(b),'\n')));if checked{r=guard(r,false)}
    c,e:=(&mcp.IOTransport{Reader:r,Writer:nopWriteCloser{io.Discard}}).Connect(context.Background());if e!=nil{t.Fatal(e)}
    ctx,cancel:=context.WithTimeout(context.Background(),time.Second);_,e=c.Read(ctx);cancel();c.Close()
    if name=="valid"&&e!=nil{t.Errorf("FAIL valid stdio: %v",e)}
    if name!="valid"&&checked&&e==nil{t.Errorf("FAIL guarded stdio accepted %s",name)}
    t.Logf("stdio %s checked=%v rejected=%v",name,checked,e!=nil)
   }
  }
 })
 t.Run("http_sdk_client",func(t *testing.T){
  for _,sse:=range []bool{false,true}{for name,b:=range map[string][]byte{"valid":[]byte(good),"duplicate":bads["duplicate"],"utf8":bads["utf8"]}{
   name,b:=name,b;t.Run(fmt.Sprintf("sse=%v/%s",sse,name),func(t *testing.T){
    raw:=b;ct:="application/json";if sse{raw=append(append([]byte("event: message\ndata: "),b...),[]byte("\n\n")...);ct="text/event-stream"}
    if sse&&name!="valid"{raw=append([]byte("id: 7\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\",\"params\":{}}\n\n"),raw...)}
    var conn mcp.Connection;var requests atomic.Int32
    base:=rtFunc(func(r *http.Request)(*http.Response,error){requests.Add(1);return &http.Response{StatusCode:200,Header:http.Header{"Content-Type":{ct}},Body:io.NopCloser(bytes.NewReader(raw)),Request:r},nil})
    ctx,cancel:=context.WithTimeout(context.Background(),time.Second);defer cancel()
    tr:=&mcp.StreamableClientTransport{Endpoint:"http://fixture.invalid/mcp",HTTPClient:&http.Client{Transport:checkedHTTP(base,func(error){if conn!=nil{conn.Close()}})},MaxRetries:5,DisableStandaloneSSE:true}
    c,e:=tr.Connect(ctx);if e!=nil{t.Fatal(e)};conn=c;defer c.Close();id,idErr:=jsonrpc.MakeID(float64(1));if idErr!=nil{t.Fatal(idErr)}
    e=c.Write(ctx,&jsonrpc.Request{ID:id,Method:"tools/list",Params:[]byte(`{}`)})
    for e==nil{var msg jsonrpc.Message;msg,e=c.Read(ctx);if e!=nil{break};if _,ok:=msg.(*jsonrpc.Request);ok{continue};if response,ok:=msg.(*jsonrpc.Response);ok&&response.Error!=nil{e=response.Error};break};if requests.Load()!=1{t.Errorf("FAIL unexpected HTTP retry count=%d",requests.Load())}
    if name=="valid"&&e!=nil{t.Errorf("FAIL valid HTTP: %v",e)}
    if name!="valid"&&(e==nil||errors.Is(e,context.DeadlineExceeded)){t.Errorf("FAIL invalid HTTP not promptly rejected: %v",e)}
    t.Logf("HTTP %s sse=%v rejected=%v error=%v",name,sse,e!=nil,e)
   })
  }}
 })
 t.Run("sse_framing",func(t *testing.T){
  for _,sep:=range []string{"\n","\r\n","\r"}{
   text:=strings.Join([]string{": heartbeat","","id: 7","retry: 12","event: message","data: {\"jsonrpc\":\"2.0\",","data: \"id\":1,\"result\":{}}","",""},sep)
   b,e:=io.ReadAll(guard(io.NopCloser(strings.NewReader(text)),true));if e!=nil{t.Errorf("FAIL SSE framing %q: %v",sep,e)}
   if !bytes.Contains(b,[]byte("id: 7\nretry: 12\n")){t.Error("FAIL SSE metadata lost")}
  }
 })
 t.Run("sdk_event_name_trimming",func(t *testing.T){
  for _,event:=range []string{"message "," message", "\t"}{
   raw:="event: "+event+"\ndata: "+string(bads["duplicate"])+"\n\n"
   buf:=make([]byte,1);n,e:=guard(io.NopCloser(strings.NewReader(raw)),true).Read(buf)
   if n!=0||e==nil{t.Errorf("FAIL SDK event-name trimming bypass %q",event)}
  }
 })
 t.Run("zero_invalid_byte_exposure",func(t *testing.T){
  for _,sse:=range []bool{false,true}{raw:=append(bytes.Clone(bads["duplicate"]),'\n');if sse{raw=append(append([]byte("data: "),bads["duplicate"]...),[]byte("\n\n")...)};g:=guard(io.NopCloser(bytes.NewReader(raw)),sse);buf:=make([]byte,1);n,e:=g.Read(buf);if n!=0||e==nil{t.Errorf("FAIL invalid bytes exposed sse=%v n=%d err=%v",sse,n,e)}}
 })
 t.Run("http_sdk_server",func(t *testing.T){
  for _,checked:=range []bool{false,true}{var hits atomic.Int32
   srv:=mcp.NewServer(&mcp.Implementation{Name:"fixture",Version:"1"},&mcp.ServerOptions{SupportedProtocolVersions:[]string{"2026-07-28"}})
   h:=mcp.NewStreamableHTTPHandler(func(*http.Request)*mcp.Server{hits.Add(1);return srv},&mcp.StreamableHTTPOptions{Stateless:true,JSONResponse:true})
   var handler http.Handler=h;if checked{handler=checkedHandler(h)}
   body:=`{"jsonrpc":"2.0","id":1,"method":"tools/list","method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
   req:=httptest.NewRequest("POST","http://fixture.invalid/mcp",strings.NewReader(body));req.Header.Set("Content-Type","application/json");req.Header.Set("Accept","application/json, text/event-stream");req.Header.Set("MCP-Protocol-Version","2026-07-28");req.Header.Set("MCP-Method","tools/list")
   w:=httptest.NewRecorder();handler.ServeHTTP(w,req)
   if checked&&hits.Load()!=0{t.Error("FAIL SDK saw invalid request")};if !checked&&(hits.Load()==0||w.Code!=200){t.Errorf("FAIL bypass control sdk_hits=%d status=%d body=%s",hits.Load(),w.Code,w.Body.String())}
   t.Logf("HTTP server checked=%v sdk_hits=%d status=%d",checked,hits.Load(),w.Code)
  }
 })
 t.Run("guard_plant",func(t *testing.T){
  // Deliberately bypass the guard: the same rejection assertion must fail.
  c,e:=(&mcp.IOTransport{Reader:io.NopCloser(bytes.NewReader(append(bytes.Clone(bads["duplicate"]),'\n'))),Writer:nopWriteCloser{io.Discard}}).Connect(context.Background());if e!=nil{t.Fatal(e)};defer c.Close()
  ctx,cancel:=context.WithTimeout(context.Background(),time.Second);defer cancel();_,e=c.Read(ctx)
  if e!=nil{t.Fatal("FAIL plant did not demonstrate a permissive control",e)}
  t.Log("PLANT CAUGHT: disabled raw guard lets duplicate members reach SDK decoding")
 })
}
type nopWriteCloser struct{io.Writer}
func(nopWriteCloser)Close()error{return nil}

type offlineLoader struct{root string}
func(l offlineLoader)Load(uri string)(any,error){
 u,e:=url.Parse(uri);if e!=nil{return nil,e};if u.Host!="localhost:1234"{return nil,fmt.Errorf("offline resolver refused %s",uri)}
 b,e:=os.ReadFile(filepath.Join(l.root,"testdata","remotes",u.Path));if e!=nil{return nil,e};return exact.UnmarshalJSON(bytes.NewReader(b))
}
func compiler(root string)*exact.Compiler {
 c:=exact.NewCompiler();c.DefaultDraft(exact.Draft2020);c.UseLoader(offlineLoader{root});return c
}
func TestSchemaCoverage(t *testing.T){
 suite:=os.Getenv("SCHEMA_SUITE");if suite==""{t.Fatal("FAIL missing SCHEMA_SUITE")}
 raw,e:=os.ReadFile("mcp.schema.json");if e!=nil{t.Fatal(e)};doc,e:=exact.UnmarshalJSON(bytes.NewReader(raw));if e!=nil{t.Fatal(e)}
 c:=compiler(suite);if e=c.AddResource("https://fixture.invalid/mcp.json",doc);e!=nil{t.Fatal(e)}
 defs:=doc.(map[string]any)["$defs"].(map[string]any)
 for name:=range defs{t.Run("compile/"+name,func(t *testing.T){if _,e:=c.Compile("https://fixture.invalid/mcp.json#/$defs/"+name);e!=nil{t.Errorf("FAIL definition %s: %v",name,e)}})}
 files:=[]string{"defs","ref","type","const","enum","properties","required","additionalProperties","items","maxItems","minimum","maximum","allOf","anyOf"};total:=0
 for _,name:=range files{t.Run("suite/"+name,func(t *testing.T){
  b,e:=os.ReadFile(filepath.Join(suite,"testdata","draft2020-12",name+".json"));if e!=nil{t.Fatal(e)}
  var groups []struct{Description string;Schema any;Tests []struct{Description string;Data any;Valid bool}}
  d:=json.NewDecoder(bytes.NewReader(b));d.UseNumber();if e=d.Decode(&groups);e!=nil{t.Fatal(e)}
  for gi,g:=range groups{cc:=compiler(suite);uri:=fmt.Sprintf("https://fixture.invalid/%s/%d.json",name,gi);if e:=cc.AddResource(uri,g.Schema);e!=nil{t.Errorf("FAIL add %s: %v",g.Description,e);continue};rs,e:=cc.Compile(uri);if e!=nil{t.Errorf("FAIL %s/%s resolve: %v",name,g.Description,e);continue}
   for _,tc:=range g.Tests{total++;e:=rs.Validate(tc.Data);if (e==nil)!=tc.Valid{t.Errorf("FAIL %s/%s/%s valid=%v error=%v",name,g.Description,tc.Description,tc.Valid,e)}}
  }
 })}
 t.Logf("ASSERTION FIXTURES=%d DEFINITIONS=%d",total,len(defs))
 t.Run("precision",func(t *testing.T){
  for _,raw:=range []string{`{"const":9007199254740993}`,`{"enum":[9007199254740993]}`,`{"type":"integer","minimum":9007199254740993,"maximum":9007199254740993}`} {
   doc,e:=exact.UnmarshalJSON(strings.NewReader(raw));if e!=nil{t.Fatal(e)};cc:=compiler(suite);if e=cc.AddResource("https://fixture.invalid/exact",doc);e!=nil{t.Fatal(e)};rs,e:=cc.Compile("https://fixture.invalid/exact");if e!=nil{t.Fatal(e)}
   if e=rs.Validate(json.Number("9007199254740993"));e!=nil{t.Errorf("FAIL exact number: %v",e)}
   if e=rs.Validate(json.Number("9007199254740992"));e==nil{t.Error("FAIL adjacent integer accepted")}
  }
 })
 t.Run("schema_plant",func(t *testing.T){cc:=compiler(suite);cc.AddResource("https://fixture.invalid/plant",map[string]any{"type":"object","required":[]any{"x"}});rs,e:=cc.Compile("https://fixture.invalid/plant");if e!=nil{t.Fatal(e)};if rs.Validate(map[string]any{})==nil{t.Fatal("FAIL required plant escaped")};t.Log("PLANT CAUGHT: missing required property rejected")})
}
func TestSDKValidatorLosslessBoundary(t *testing.T){
 s:=jsonschema.Schema{Type:"integer"};rs,e:=s.Resolve(nil);if e!=nil{t.Fatal(e)}
 if e=rs.Validate(json.Number("1"));e==nil{t.Fatal("SDK behavior changed: reassess replacement rationale")};t.Logf("SDK candidate failure reproduced: %v",e)
 var sc jsonschema.Schema;if e=json.Unmarshal([]byte(`{"const":9007199254740993}`),&sc);e!=nil{t.Fatal(e)};r,e:=sc.Resolve(nil);if e!=nil{t.Fatal(e)}
 if r.Validate(json.Number("9007199254740993"))==nil||r.Validate(json.Number("9007199254740992"))!=nil{t.Fatal("SDK constant behavior changed: reassess rationale")};t.Log("SDK candidate incorrectly accepts adjacent rounded integer; exact adapter test retains original precision assertions")
}

var uriRE=regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*):(?://([^/?#]*))?([^?#]*)(?:\?([^#]*))?(?:#(.*))?$`)
func validChars(s,extra string)bool{for i:=0;i<len(s);i++{c:=s[i];if c=='%'{if i+2>=len(s){return false};if _,e:=strconv.ParseUint(s[i+1:i+3],16,8);e!=nil{return false};i+=2;continue};if c>='a'&&c<='z'||c>='A'&&c<='Z'||c>='0'&&c<='9'||strings.ContainsRune("-._~!$&'()*+,;="+extra,rune(c)){continue};return false};return true}
func validURI(s string)bool{
 m:=uriRE.FindStringSubmatch(s);if m==nil||!validChars(m[3],":@/")||!validChars(m[4],":@/?")||!validChars(m[5],":@/?"){return false}
 u,e:=url.Parse(s);if e!=nil||!u.IsAbs(){return false}
 authority:=m[2];if i:=strings.LastIndex(authority,"@");i>=0{if !validChars(authority[:i],":"){return false};authority=authority[i+1:]}
 if strings.HasPrefix(authority,"["){i:=strings.IndexByte(authority,']');if i<0{return false};ip:=authority[1:i];if _,e=netip.ParseAddr(ip);e!=nil{if !regexp.MustCompile(`^[vV][0-9A-Fa-f]+\.[A-Za-z0-9._~!$&'()*+,;=:-]+$`).MatchString(ip){return false}};authority="host"+authority[i+1:]}
 host,port,hasPort:=strings.Cut(authority,":");if !validChars(host,""){return false};if hasPort{for _,c:=range port{if c<'0'||c>'9'{return false}}};return true
}
func TestFormats(t *testing.T){
 for _,s:=range []string{"file:///tmp/a","urn:example:a","data:text/plain;base64,QQ==","https://[2001:db8::1]/a?q=x#f","mcp-resource://host/a%20b"}{if !validURI(s){t.Errorf("FAIL valid URI %q",s)}}
 for _,s:=range []string{"relative/path","https://x/a b","https://[bad]/","x:%zz","https://x/é","https://x/a\\b"}{if validURI(s){t.Errorf("FAIL invalid URI %q",s)}}
 for _,s:=range []string{"mcp://host/{id}","https://host/{+path}{?q,limit}"}{if _,e:=uritemplate.New(s);e!=nil{t.Errorf("FAIL URI template %q: %v",s,e)}}
 for _,s:=range []string{"https://host/{id","https://host/{?}","https://host/{id:abc}"}{if _,e:=uritemplate.New(s);e==nil{t.Errorf("FAIL invalid URI template %q",s)}}
 for _,s:=range []string{"", "QQ==", "AAEC"}{if _,e:=base64.StdEncoding.Strict().DecodeString(s);e!=nil{t.Errorf("FAIL valid base64 %q",s)}}
 for _,s:=range []string{"!", "Q===", "QR=="}{if _,e:=base64.StdEncoding.Strict().DecodeString(s);e==nil{t.Errorf("FAIL invalid base64 %q",s)}}
 var v any;if e:=jsonv2.Unmarshal([]byte(`{"x":"ok"}`),&v);e!=nil{t.Fatal(e)}
}

func BenchmarkBoundary(b *testing.B){
 for _,n:=range []int{1024,64*1024,256*1024,frameLimit-128}{
  raw:=[]byte(`{"jsonrpc":"2.0","id":1,"result":{"text":"`+strings.Repeat("x",n)+`"}}`)
  for _,checked:=range []bool{false,true}{b.Run(fmt.Sprintf("bytes=%d/checked=%v",len(raw),checked),func(b *testing.B){b.ReportAllocs();b.SetBytes(int64(len(raw)));for b.Loop(){if checked{if e:=strictJSON(raw);e!=nil{b.Fatal(e)}};if _,e:=jsonrpc.DecodeMessage(raw);e!=nil{b.Fatal(e)}}})}
 }
}

func TestFormatFixtures(t *testing.T){
 total:=0
 for _,name:=range []string{"uri","uri-template"}{
  b,e:=os.ReadFile(name+".json");if e!=nil{t.Fatal(e)}
  var groups []struct{Tests []struct{Description string;Data any;Valid bool}};if e=json.Unmarshal(b,&groups);e!=nil{t.Fatal(e)}
  for _,g:=range groups{for _,tc:=range g.Tests{total++;valid:=true;if value,ok:=tc.Data.(string);ok{if name=="uri"{valid=validURI(value)}else{_,e:=uritemplate.New(value);valid=e==nil}};if valid!=tc.Valid{t.Errorf("FAIL format %s/%s want=%v got=%v",name,tc.Description,tc.Valid,valid)}}}
 }
 t.Logf("OFFICIAL FORMAT FIXTURES=%d",total)
 c:=compiler(os.Getenv("SCHEMA_SUITE"));c.AssertFormat()
 c.RegisterFormat(&exact.Format{Name:"uri",Validate:func(v any)error{if s,ok:=v.(string);ok&&!validURI(s){return errors.New("invalid URI")};return nil}})
 c.RegisterFormat(&exact.Format{Name:"uri-template",Validate:func(v any)error{if s,ok:=v.(string);ok{_,e:=uritemplate.New(s);return e};return nil}})
 c.RegisterFormat(&exact.Format{Name:"byte",Validate:func(v any)error{if s,ok:=v.(string);ok{_,e:=base64.StdEncoding.Strict().DecodeString(s);return e};return nil}})
 for _,tc:=range []struct{name,good,bad string}{{"uri","urn:test:a","relative/path"},{"uri-template","mcp://host/{id}","mcp://host/{id"},{"byte","QQ==","!"}}{
  uri:="https://fixture.invalid/format/"+tc.name;c.AddResource(uri,map[string]any{"type":"string","format":tc.name});sch,e:=c.Compile(uri);if e!=nil{t.Fatal(e)}
  if e:=sch.Validate(tc.good);e!=nil{t.Errorf("FAIL compiled format %s valid: %v",tc.name,e)};if e:=sch.Validate(tc.bad);e==nil{t.Errorf("FAIL compiled format %s invalid accepted",tc.name)}
 }
}
```

## Raw benchmark samples

Tabs in the module manifest and benchmark output are expanded to spaces for
Markdown formatting. All benchmark fields and sample values are preserved.

### Darwin/arm64, default CPU count

```text
goos: darwin
goarch: arm64
cpu: Apple M1 Pro
BenchmarkBoundary/bytes=1069/checked=false-10              20364          5860 ns/op     182.43 MB/s       34177 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false-10              20401          5827 ns/op     183.46 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false-10              20544          5759 ns/op     185.61 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false-10              20323          5837 ns/op     183.16 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false-10              20630          5836 ns/op     183.19 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-10               14281          8260 ns/op     129.41 MB/s       34203 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-10               14781          8176 ns/op     130.76 MB/s       34203 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-10               14228          8456 ns/op     126.42 MB/s       34203 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-10               14149          8556 ns/op     124.94 MB/s       34203 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-10               10000         13947 ns/op      76.65 MB/s       34203 B/op          7 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-10               804        145936 ns/op     449.38 MB/s      303986 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-10               835        142322 ns/op     460.79 MB/s      303981 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-10               843        142746 ns/op     459.42 MB/s      303985 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-10               850        141286 ns/op     464.17 MB/s      303979 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-10               849        145080 ns/op     452.03 MB/s      303983 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-10                432        274704 ns/op     238.73 MB/s      304154 B/op         18 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-10                435        269593 ns/op     243.26 MB/s      304140 B/op         18 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-10                440        269515 ns/op     243.33 MB/s      304099 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-10                448        265419 ns/op     247.09 MB/s      304149 B/op         18 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-10                459        261804 ns/op     250.50 MB/s      304130 B/op         18 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-10              201        593409 ns/op     441.83 MB/s     1288140 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-10              201        594523 ns/op     441.01 MB/s     1288200 B/op         29 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-10              201        600272 ns/op     436.78 MB/s     1288207 B/op         29 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-10              204        588426 ns/op     445.58 MB/s     1288208 B/op         29 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-10              202        586233 ns/op     447.24 MB/s     1288193 B/op         29 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-10               100       1089753 ns/op     240.59 MB/s     1289225 B/op         33 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-10               100       1083795 ns/op     241.92 MB/s     1289181 B/op         34 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-10               100       1090640 ns/op     240.40 MB/s     1289222 B/op         34 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-10               100       1088818 ns/op     240.80 MB/s     1289183 B/op         33 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-10               100       1082984 ns/op     242.10 MB/s     1289195 B/op         34 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-10              56       1945393 ns/op     538.96 MB/s     3115914 B/op         36 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-10              56       1931008 ns/op     542.98 MB/s     3115892 B/op         35 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-10              63       1926751 ns/op     544.18 MB/s     3115881 B/op         35 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-10              61       1916389 ns/op     547.12 MB/s     3115846 B/op         35 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-10              60       1936977 ns/op     541.30 MB/s     3115880 B/op         35 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-10               28       3948890 ns/op     265.52 MB/s     3117596 B/op         43 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-10               30       3950256 ns/op     265.42 MB/s     3117428 B/op         39 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-10               30       3941796 ns/op     265.99 MB/s     3117618 B/op         44 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-10               28       3935862 ns/op     266.39 MB/s     3117882 B/op         47 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-10               30       3934182 ns/op     266.51 MB/s     3117392 B/op         41 allocs/op
PASS
ok      github.com/maccavelli/mcplib    4.898s
```

### Windows/amd64, default CPU count

```text
goos: windows
goarch: amd64
cpu: Intel(R) Core(TM) i9-14900HX
BenchmarkBoundary/bytes=1069/checked=false-32              20586          6266 ns/op     170.60 MB/s       34177 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false-32              20202          6019 ns/op     177.60 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false-32              19684          6016 ns/op     177.69 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false-32              20174          5997 ns/op     178.25 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false-32              19689          6113 ns/op     174.88 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-32               15188          7862 ns/op     135.96 MB/s       34264 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-32               15343          8493 ns/op     125.87 MB/s       34265 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-32               15409          7711 ns/op     138.63 MB/s       34265 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-32               15456          7351 ns/op     145.42 MB/s       34251 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true-32               16438          7479 ns/op     142.93 MB/s       34262 B/op          7 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-32              1179        101628 ns/op     645.30 MB/s      304491 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-32              1245        100880 ns/op     650.09 MB/s      304495 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-32              1278         98454 ns/op     666.11 MB/s      304489 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-32              1234         98983 ns/op     662.55 MB/s      304492 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false-32              1209        103355 ns/op     634.52 MB/s      304402 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-32                752        159505 ns/op     411.15 MB/s      305114 B/op         19 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-32                764        159509 ns/op     411.14 MB/s      305242 B/op         19 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-32                788        169970 ns/op     385.84 MB/s      305273 B/op         19 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-32                764        157607 ns/op     416.11 MB/s      305254 B/op         19 allocs/op
BenchmarkBoundary/bytes=65581/checked=true-32                762        158749 ns/op     413.11 MB/s      305248 B/op         19 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-32              238        429278 ns/op     610.77 MB/s     1289396 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-32              328        367855 ns/op     712.75 MB/s     1289749 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-32              332        369960 ns/op     708.70 MB/s     1289829 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-32              314        365719 ns/op     716.91 MB/s     1289774 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=false-32              332        812405 ns/op     322.73 MB/s     1289716 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-32                56       2375139 ns/op     110.39 MB/s     1292141 B/op         34 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-32                61       2523603 ns/op     103.89 MB/s     1292484 B/op         33 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-32                58       2602247 ns/op     100.75 MB/s     1292453 B/op         32 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-32                44       2629582 ns/op      99.71 MB/s     1292372 B/op         32 allocs/op
BenchmarkBoundary/bytes=262189/checked=true-32                60       2493797 ns/op     105.14 MB/s     1292290 B/op         31 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-32              30       4871060 ns/op     215.25 MB/s     3119250 B/op         34 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-32              28       4777018 ns/op     219.49 MB/s     3118850 B/op         35 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-32              26       4685808 ns/op     223.76 MB/s     3118974 B/op         35 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-32              26       4860512 ns/op     215.72 MB/s     3119096 B/op         34 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false-32              25       5343028 ns/op     196.24 MB/s     3118894 B/op         34 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-32               14       8740486 ns/op     119.96 MB/s     3123641 B/op         44 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-32               13       8804585 ns/op     119.08 MB/s     3122936 B/op         41 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-32               13       9602062 ns/op     109.19 MB/s     3123916 B/op         39 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-32               14       9006164 ns/op     116.42 MB/s     3121361 B/op         41 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true-32               13       8311031 ns/op     126.16 MB/s     3123434 B/op         43 allocs/op
PASS
ok      github.com/maccavelli/mcplib    5.201s
```

### Windows/amd64, GOMAXPROCS=1

```text
goos: windows
goarch: amd64
cpu: Intel(R) Core(TM) i9-14900HX
BenchmarkBoundary/bytes=1069/checked=false             57672          5999 ns/op     178.19 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false             58051          6791 ns/op     157.42 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false             50138          6917 ns/op     154.55 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false             53042          6676 ns/op     160.13 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=false             62050          5982 ns/op     178.71 MB/s       34176 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true              47334          7314 ns/op     146.16 MB/s       34177 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true              45414          7377 ns/op     144.92 MB/s       34177 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true              48993          7200 ns/op     148.47 MB/s       34177 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true              48340         18247 ns/op      58.58 MB/s       34177 B/op          7 allocs/op
BenchmarkBoundary/bytes=1069/checked=true              14510         24234 ns/op      44.11 MB/s       34177 B/op          7 allocs/op
BenchmarkBoundary/bytes=65581/checked=false             1011        370456 ns/op     177.03 MB/s      303807 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false              997        382987 ns/op     171.24 MB/s      303808 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false              997        360511 ns/op     181.91 MB/s      303807 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false             1006        378152 ns/op     173.42 MB/s      303807 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=false              952        365348 ns/op     179.50 MB/s      303807 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=true               600        605740 ns/op     108.27 MB/s      303824 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=true               600        597500 ns/op     109.76 MB/s      303823 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=true               604        603957 ns/op     108.59 MB/s      303825 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=true               609        600735 ns/op     109.17 MB/s      303823 B/op         17 allocs/op
BenchmarkBoundary/bytes=65581/checked=true               594        599715 ns/op     109.35 MB/s      303823 B/op         17 allocs/op
BenchmarkBoundary/bytes=262189/checked=false             242       1472919 ns/op     178.01 MB/s     1287332 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=false             230       1523318 ns/op     172.12 MB/s     1287332 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=false             237       1489516 ns/op     176.02 MB/s     1287332 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=false             243       1480251 ns/op     177.12 MB/s     1287332 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=false             241       1506458 ns/op     174.04 MB/s     1287332 B/op         28 allocs/op
BenchmarkBoundary/bytes=262189/checked=true              139       2525638 ns/op     103.81 MB/s     1287409 B/op         29 allocs/op
BenchmarkBoundary/bytes=262189/checked=true              146       2453699 ns/op     106.85 MB/s     1287404 B/op         29 allocs/op
BenchmarkBoundary/bytes=262189/checked=true              147       2443193 ns/op     107.31 MB/s     1287409 B/op         29 allocs/op
BenchmarkBoundary/bytes=262189/checked=true              141       2486771 ns/op     105.43 MB/s     1287415 B/op         29 allocs/op
BenchmarkBoundary/bytes=262189/checked=true              146       2433735 ns/op     107.73 MB/s     1287404 B/op         29 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false             87       4814772 ns/op     217.77 MB/s     3114421 B/op         33 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false             64       4700366 ns/op     223.07 MB/s     3114419 B/op         33 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false             92       4797785 ns/op     218.54 MB/s     3114420 B/op         33 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false             80       4782199 ns/op     219.25 MB/s     3114421 B/op         33 allocs/op
BenchmarkBoundary/bytes=1048493/checked=false             87       4768784 ns/op     219.87 MB/s     3114421 B/op         33 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true              44       8598984 ns/op     121.93 MB/s     3114737 B/op         39 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true              46       8575504 ns/op     122.27 MB/s     3114667 B/op         37 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true              44       8552641 ns/op     122.59 MB/s     3114737 B/op         39 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true              42       8627974 ns/op     121.52 MB/s     3114716 B/op         38 allocs/op
BenchmarkBoundary/bytes=1048493/checked=true              46       8596591 ns/op     121.97 MB/s     3114648 B/op         37 allocs/op
PASS
ok      github.com/maccavelli/mcplib    15.221s
```
