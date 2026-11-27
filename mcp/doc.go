// Package mcp provides owned MCP server/client handles, sessions, typed
// registrations and middleware over the pinned upstream SDK.
package mcp

// PinnedDependencies names the exact upstream pins the framework builds on.
// Runtime pin tests verify this dependency contract alongside the module graph.
type PinnedDependencies struct{}

// Pins returns the exact upstream versions the runtime module builds on.
func Pins() map[string]string {
	return map[string]string{
		"github.com/modelcontextprotocol/go-sdk":   "v1.8.0",
		"github.com/santhosh-tekuri/jsonschema/v6": "v6.0.3",
		"github.com/yosida95/uritemplate/v3":       "v3.0.2",
	}
}
