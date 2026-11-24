package model

import (
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

// TestBuildPinnedSchema loads and models the actual committed MCP schema
// end to end: 155 definitions, no collapse, total keyword classification.
func TestBuildPinnedSchema(t *testing.T) {
	loader := load.New()
	doc, err := loader.LoadFile("../../../../schema/inputs/2026-07-28/schema.json")
	if err != nil {
		t.Fatalf("FAIL pinned schema load: %v", err)
	}
	if doc.Definitions != 155 {
		t.Fatalf("FAIL definitions: %d", doc.Definitions)
	}
	m, err := Build(doc)
	if err != nil {
		t.Fatalf("FAIL pinned schema model: %v", err)
	}
	if len(m.Definitions) != 155 {
		t.Fatalf("FAIL modeled definitions: %d", len(m.Definitions))
	}
	// Spot-check structural fidelity on known definitions.
	tool := m.Definitions["Tool"]
	if tool == nil || tool.Type != "object" {
		t.Fatalf("FAIL Tool shape: %+v", tool)
	}
	if _, hasName := tool.Properties["name"]; !hasName {
		t.Fatal("FAIL Tool.name property missing")
	}
	req := false
	for _, name := range tool.Required {
		if name == "name" {
			req = true
		}
	}
	if !req {
		t.Fatal("FAIL Tool.name not required")
	}
	// Sampling deprecation context: CreateMessageRequest must survive the
	// model losslessly even though SEP-2577 deprecates the feature.
	cmr := m.Definitions["CreateMessageRequest"]
	if cmr == nil {
		t.Fatal("FAIL CreateMessageRequest definition missing from the model")
	}
}
