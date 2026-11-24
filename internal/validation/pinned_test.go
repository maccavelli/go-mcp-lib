package validation

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestCompileAllPinnedDefinitions compiles all 155 definitions of the
// committed pinned schema through the canonical adapter, proving every
// definition is a valid Draft 2020-12 subschema.
func TestCompileAllPinnedDefinitions(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schema", "inputs", "2026-07-28", "schema.json"))
	if err != nil {
		t.Fatalf("FAIL pinned schema: %v", err)
	}
	doc, err := decodeDoc(string(body))
	if err != nil {
		t.Fatalf("FAIL decode: %v", err)
	}
	root := doc.(map[string]any)
	defsObj := root["$defs"].(map[string]any)
	if len(defsObj) != 155 {
		t.Fatalf("FAIL definition count: %d", len(defsObj))
	}
	a := New(256)
	// Register the full document as the local registry root so internal
	// #/$defs/... references resolve during compilation.
	if err := a.Register("https://example.invalid/mcp.json", doc); err != nil {
		t.Fatalf("FAIL register root: %v", err)
	}
	names := make([]string, 0, len(defsObj))
	for name := range defsObj {
		names = append(names, name)
	}
	sort.Strings(names)
	compiled := 0
	for _, name := range names {
		// Each definition compiles as its own root, wrapped so $defs
		// references resolve through the registered document.
		wrapper := map[string]any{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"$ref":    "https://example.invalid/mcp.json#/$defs/" + name,
		}
		if _, err := a.Compile("def:"+name, "https://example.invalid/defs/"+name+".json", wrapper); err != nil {
			t.Errorf("FAIL compile %s: %v", name, err)
			continue
		}
		compiled++
	}
	if compiled != 155 {
		t.Fatalf("FAIL compiled %d of 155 definitions", compiled)
	}
}

// TestValidateRepresentativeInstances validates representative instances
// against representative compiled definitions: the exact-token contract
// at validation time.
func TestValidateRepresentativeInstances(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schema", "inputs", "2026-07-28", "schema.json"))
	if err != nil {
		t.Fatalf("FAIL pinned schema: %v", err)
	}
	doc, err := decodeDoc(string(body))
	if err != nil {
		t.Fatal(err)
	}
	a := New(256)
	if err := a.Register("https://example.invalid/mcp.json", doc); err != nil {
		t.Fatal(err)
	}
	schema, err := a.Compile("tool", "https://example.invalid/tool-root.json", map[string]any{
		"$ref": "https://example.invalid/mcp.json#/$defs/Tool",
	})
	if err != nil {
		t.Fatalf("FAIL compile Tool: %v", err)
	}
	// A minimal valid Tool instance: name and inputSchema required.
	valid := map[string]any{
		"name":        "probe",
		"inputSchema": map[string]any{"type": "object"},
	}
	if err := a.Validate(schema, valid); err != nil {
		t.Fatalf("FAIL valid Tool rejected: %v", err)
	}
	invalid := map[string]any{"inputSchema": map[string]any{"type": "object"}}
	if err := a.Validate(schema, invalid); err == nil {
		t.Fatal("FAIL Tool without name accepted")
	}
}
