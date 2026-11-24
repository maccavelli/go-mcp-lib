package mcp

import "testing"

// TestPins asserts the exact upstream pins the runtime module builds on.
// The require list in go.mod is grown organically as stages import the
// dependencies; this test is the runtime-side record of the plan pins.
func TestPins(t *testing.T) {
	got := Pins()
	want := map[string]string{
		"github.com/modelcontextprotocol/go-sdk":   "v1.8.0",
		"github.com/santhosh-tekuri/jsonschema/v6": "v6.0.3",
		"github.com/yosida95/uritemplate/v3":       "v3.0.2",
	}
	for module, version := range want {
		if got[module] != version {
			t.Errorf("pin %s: got %q want %q", module, got[module], version)
		}
	}
	if len(got) != len(want) {
		t.Errorf("pins: got %d entries want %d", len(got), len(want))
	}
}
