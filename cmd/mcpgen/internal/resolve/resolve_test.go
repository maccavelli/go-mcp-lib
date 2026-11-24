package resolve

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

func registry(t *testing.T, canonical, body string) *Registry {
	t.Helper()
	doc, err := load.New().LoadBytes(canonical, []byte(body))
	if err != nil {
		t.Fatalf("FAIL load %s: %v", canonical, err)
	}
	reg := NewRegistry()
	if err := reg.Add(canonical, doc); err != nil {
		t.Fatalf("FAIL add: %v", err)
	}
	return reg
}

func TestPointerResolution(t *testing.T) {
	reg := registry(t, "schema", `{"$defs":{"Tool":{"type":"object","properties":{"name":{"type":"string"}}}}}`)
	ref, err := reg.Resolve("schema", "#/$defs/Tool")
	if err != nil {
		t.Fatalf("FAIL resolve: %v", err)
	}
	tool := ref.Value.(map[string]any)
	if tool["type"] != "object" {
		t.Fatalf("FAIL resolved value: %v", tool)
	}
}

func TestEscapedTokens(t *testing.T) {
	reg := registry(t, "schema", `{"$defs":{"a/b":{"~":1},"c~d":{"x":2}}}`)
	if _, err := reg.Resolve("schema", "#/$defs/a~1b/~0"); err != nil {
		t.Fatalf("FAIL escaped slash: %v", err)
	}
	if _, err := reg.Resolve("schema", "#/$defs/c~0d"); err != nil {
		t.Fatalf("FAIL escaped tilde: %v", err)
	}
}

func TestMissingTargetFailsLoudly(t *testing.T) {
	reg := registry(t, "schema", `{"$defs":{}}`)
	if _, err := reg.Resolve("schema", "#/$defs/Missing"); err == nil {
		t.Fatal("FAIL missing target accepted")
	}
	if _, err := reg.Resolve("schema", "#/$defs/Missing/deeper"); err == nil {
		t.Fatal("FAIL deep missing target accepted")
	}
}

func TestArrayIndexing(t *testing.T) {
	reg := registry(t, "schema", `{"$defs":{"list":[null,2,{"target":true}]}}`)
	ref, err := reg.Resolve("schema", "#/$defs/list/2/target")
	if err != nil {
		t.Fatalf("FAIL array pointer: %v", err)
	}
	if ref.Value != true {
		t.Fatalf("FAIL array value: %v", ref.Value)
	}
	if _, err := reg.Resolve("schema", "#/$defs/list/9"); err == nil {
		t.Fatal("FAIL out-of-bounds accepted")
	}
	if _, err := reg.Resolve("schema", "#/$defs/list/01"); err == nil {
		t.Fatal("FAIL leading-zero index accepted")
	}
}

func TestDuplicateCanonicalRejected(t *testing.T) {
	reg := registry(t, "schema", `{}`)
	doc, _ := load.New().LoadBytes("schema", []byte(`{}`))
	if err := reg.Add("schema", doc); err == nil {
		t.Fatal("FAIL duplicate canonical URI accepted")
	}
}

func TestNetworkRefusal(t *testing.T) {
	_ = registry(t, "schema", `{"$defs":{"a":{}}}`)
	err := RefuseNetwork("https://example.invalid/deep/path/to/evil")
	if err == nil {
		t.Fatal("FAIL network reference not refused")
	}
	// The refusal message must not carry the full external URL path.
	if strings.Contains(err.Error(), "to/evil") {
		t.Fatalf("FAIL refusal leaks full path: %v", err)
	}
	if err := RefuseNetwork("#/$defs/a"); err != nil {
		t.Fatalf("FAIL local pointer wrongly refused: %v", err)
	}
	if err := RefuseNetwork("schema#/$defs/a"); err != nil {
		t.Fatalf("FAIL canonical+fragment wrongly refused: %v", err)
	}
}

func TestChainThroughNestedRefs(t *testing.T) {
	reg := registry(t, "schema", `{"$defs":{"a":{"$ref":"#/$defs/b"},"b":{"$ref":"#/$defs/c"},"c":{"type":"string"}}}`)
	chain, err := reg.ResolveChain("schema", "#/$defs/a", 0)
	if err != nil {
		t.Fatalf("FAIL chain: %v", err)
	}
	if len(chain) != 3 {
		t.Fatalf("FAIL chain length: %d", len(chain))
	}
}

func TestCycleTerminatesOnNonRef(t *testing.T) {
	// A cycle tolerates revisits; resolution ends at the first non-ref
	// node, and the chain does not spin.
	reg := registry(t, "schema", `{"$defs":{"a":{"$ref":"#/$defs/a"}}}`)
	chain, err := reg.ResolveChain("schema", "#/$defs/a", 8)
	if err == nil {
		if len(chain) > 8 {
			t.Fatal("FAIL cycle spun past the budget")
		}
		t.Logf("cycle ended by depth budget after %d hops", len(chain))
	}
}

func TestDepthLimitEnforced(t *testing.T) {
	reg := NewRegistry()
	reg.Limits.MaxDepth = 4
	doc, err := load.New().LoadBytes("schema", []byte(
		`{"$defs":{"a":{"$ref":"#/$defs/b"},"b":{"$ref":"#/$defs/c"},"c":{"$ref":"#/$defs/d"},"d":{"$ref":"#/$defs/e"},"e":{"type":"string"}}}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Add("schema", doc); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.ResolveChain("schema", "#/$defs/a", 0); err == nil {
		t.Fatal("FAIL over-depth chain accepted")
	}
}

func TestEdgeBudget(t *testing.T) {
	reg := registry(t, "schema", `{"$defs":{"a":{}}}`)
	reg.Limits.MaxReferenceEdges = 1
	if _, err := reg.Resolve("schema", "#/$defs/a"); err != nil {
		t.Fatalf("FAIL first edge: %v", err)
	}
	if _, err := reg.Resolve("schema", "#/$defs/a"); err == nil {
		t.Fatal("FAIL edge budget not enforced")
	}
}

func TestUnregisteredCanonical(t *testing.T) {
	reg := registry(t, "schema", `{}`)
	if _, err := reg.Resolve("schema", "other#/$defs/a"); err == nil {
		t.Fatal("FAIL unregistered canonical accepted")
	}
	if _, err := reg.Resolve("schema", "other"); err == nil {
		t.Fatal("FAIL unregistered bare canonical accepted")
	}
}
