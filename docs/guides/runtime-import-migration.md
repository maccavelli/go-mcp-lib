# Migrate runtime imports to the functional package

Use the public package for the capability needed:

| Capability | Canonical import |
| :--- | :--- |
| MCP server/client runtime | `github.com/maccavelli/go-mcp-lib/mcp` |
| Strict raw JSON | `github.com/maccavelli/go-mcp-lib/codec` |
| Generated wire contracts | `github.com/maccavelli/go-mcp-lib/protocol` |

Replace a former module-root runtime import with:

```go
import mcp "github.com/maccavelli/go-mcp-lib/mcp"
```

Keep the existing `mcp` selector and runtime API calls. The module identity and
version dependency remain `github.com/maccavelli/go-mcp-lib`; moving its runtime
package does not introduce another Go module. Codec and protocol imports remain
unchanged. The root package is absent, so a legacy-root import fails to build.

The SDK bridge is private under `mcp/internal/sdkbridge/`. Consumers use owned
runtime APIs rather than importing that implementation. Shared validation is
also private and remains at root scope because both runtime and generated
protocol code use it. Do not import the maintainer generator from applications.

Run `make pre-add-check` to validate both modules with the prepared pinned tools.
The command selects a fleet host and queues heavy work. Native qualification
covers Windows/amd64, Darwin/arm64 and Linux/amd64, with additional WSL coverage;
cross-compiling alone does not establish native test results. Ordinary checks
use CGO off. Explicit race runs use the eligible host's queue and CGO on.

`make generate-check` verifies the unchanged generated contract and both modules'
three-target compilation. Protocol regeneration/recovery commands are in the
[generation guide](generate-protocol.md). The
[layout decision](../decisions/0014-MADR-organize-mcp-runtime-package.md) and
[execution PLAN](../decisions/0014-PLAN-organize-mcp-runtime-package.md) record
the package move, consumer checks and preserved behavior.
