// Package mcp placeholders keep the pinned module graph explicit until the
// first generated contracts land. This file is deleted in stage 2 when a
// real package imports the SDK; the require pins are load-bearing and must
// not be pruned by go mod tidy before then.
package mcp

// PinnedDependencies names the exact upstream pins the framework builds on.
// It exists so the module's require list survives go mod tidy while no
// source file imports the SDK yet, and so tests can assert the pins.
type PinnedDependencies struct{}

// Pins returns the exact upstream versions the runtime module builds on.
func Pins() map[string]string {
	return map[string]string{
		"github.com/modelcontextprotocol/go-sdk":   "v1.8.0",
		"github.com/santhosh-tekuri/jsonschema/v6": "v6.0.3",
		"github.com/yosida95/uritemplate/v3":       "v3.0.2",
	}
}
