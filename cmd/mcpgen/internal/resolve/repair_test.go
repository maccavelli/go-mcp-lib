package resolve

import (
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

func TestRepairA06References(t *testing.T) {
	t.Run("empty-member-is-not-root", func(t *testing.T) {
		reg := registry(t, "https://example.invalid/schema.json", `{"":{"type":"integer"}}`)
		root, err := reg.Resolve("https://example.invalid/schema.json", "#")
		if err != nil || root.Pointer != "" {
			t.Fatalf("FAIL root: %+v %v", root, err)
		}
		member, err := reg.Resolve("https://example.invalid/schema.json", "#/")
		if err != nil || member.Pointer != "/" || member.Value.(map[string]any)["type"] != "integer" {
			t.Fatalf("FAIL empty member: %+v %v", member, err)
		}
	})
	t.Run("registered-https-and-suffix", func(t *testing.T) {
		reg := registry(t, "https://example.invalid/schema.json", `{"$defs":{"A":{"$ref":"other.json#/$defs/B"}}}`)
		doc, err := load.New().LoadBytes("other", []byte(`{"$defs":{"B":{"type":"string"}}}`))
		if err != nil {
			t.Fatal(err)
		}
		if err := reg.Add("https://example.invalid/other.json", doc); err != nil {
			t.Fatal(err)
		}
		if err := reg.Validate(); err != nil {
			t.Fatal(err)
		}
		chain, err := reg.ResolveChain("https://example.invalid/schema.json", "#/$defs/A", 0)
		if err != nil || len(chain) != 2 || chain[1].Canonical != "https://example.invalid/other.json" {
			t.Fatalf("FAIL relative resource: %+v %v", chain, err)
		}
	})
	t.Run("nested-id-and-anchor", func(t *testing.T) {
		reg := registry(t, "https://example.invalid/root.json", `{"$defs":{"A":{"$id":"nested.json","$anchor":"entry","$ref":"#/$defs/B","$defs":{"B":{"type":"boolean"}}}}}`)
		if err := reg.Validate(); err != nil {
			t.Fatal(err)
		}
		chain, err := reg.ResolveChain("https://example.invalid/root.json", "nested.json#entry", 0)
		if err != nil || len(chain) != 2 || chain[1].Value.(map[string]any)["type"] != "boolean" {
			t.Fatalf("FAIL nested resource: %+v %v", chain, err)
		}
		pointer, err := reg.Resolve("https://example.invalid/root.json", "#/$defs/A")
		if err != nil || pointer.Canonical != "https://example.invalid/nested.json" || pointer.Pointer != "" {
			t.Fatalf("FAIL nested resource identity: %+v %v", pointer, err)
		}
	})
	t.Run("acyclic-depth-in-validation", func(t *testing.T) {
		reg := registry(t, "schema", `{"$defs":{"A":{"$ref":"#/$defs/B"},"B":{"$ref":"#/$defs/C"},"C":{"type":"string"}}}`)
		reg.Limits.MaxDepth = 2
		if err := reg.Validate(); err == nil {
			t.Fatal("FAIL over-depth graph accepted by production validation")
		}
	})
	t.Run("cycle-is-one-component", func(t *testing.T) {
		reg := registry(t, "schema", `{"$defs":{"A":{"$ref":"#/$defs/B"},"B":{"$ref":"#/$defs/C"},"C":{"$ref":"#/$defs/A"}}}`)
		reg.Limits.MaxDepth = 1
		if err := reg.Validate(); err != nil {
			t.Fatalf("FAIL cycle treated as acyclic depth: %v", err)
		}
		chain, err := reg.ResolveChain("schema", "#/$defs/A", 1)
		if err != nil || len(chain) != 3 {
			t.Fatalf("FAIL cycle identities lost: %d %v", len(chain), err)
		}
	})
	t.Run("non-schema-reference", func(t *testing.T) {
		reg := registry(t, "schema", `{"$defs":{"A":{"$ref":"#/title"}},"title":"annotation"}`)
		if err := reg.Validate(); err == nil {
			t.Fatal("FAIL reference to annotation accepted as a schema")
		}
	})
	t.Run("recursive-edge", func(t *testing.T) {
		reg := registry(t, "schema", `{"$defs":{"A":{"$ref":"#/$defs/B"},"B":{"$ref":"#/$defs/A"}}}`)
		if err := reg.Validate(); err != nil {
			t.Fatal(err)
		}
		chain, err := reg.ResolveChain("schema", "#/$defs/A", 0)
		if err != nil || len(chain) != 2 {
			t.Fatalf("FAIL recursive graph: %+v %v", chain, err)
		}
	})
	t.Run("bad-targets", func(t *testing.T) {
		reg := registry(t, "schema", `{"a":[1],"~":{"/":true}}`)
		for _, reference := range []string{"#/a/999999999999999999999999999999999", "#/a/-1", "#/a/01", "#/~2", "#/~", "#/%ZZ", "other.json", "#missing"} {
			if _, err := reg.Resolve("schema", reference); err == nil {
				t.Errorf("FAIL invalid reference accepted: %q", reference)
			}
		}
		if ref, err := reg.Resolve("schema", "#/%7E0/%7E1"); err != nil || ref.Value != true {
			t.Fatalf("FAIL percent and pointer escape order: %+v %v", ref, err)
		}
	})
	t.Run("duplicate-id-is-atomic", func(t *testing.T) {
		reg := NewRegistry()
		doc, err := load.New().LoadBytes("duplicate", []byte(`{"$defs":{"A":{"$id":"same"},"B":{"$id":"same"}}}`))
		if err != nil {
			t.Fatal(err)
		}
		if err := reg.Add("schema", doc); err == nil || len(reg.Documents) != 0 || len(reg.resources) != 0 {
			t.Fatalf("FAIL failed registration changed registry: %v", err)
		}
	})
}
