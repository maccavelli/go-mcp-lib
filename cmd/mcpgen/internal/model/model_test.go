package model

import (
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

func build(t *testing.T, body string) *Model {
	t.Helper()
	doc, err := load.New().LoadBytes("schema", []byte(body))
	if err != nil {
		t.Fatalf("FAIL load: %v", err)
	}
	m, err := Build(doc)
	if err != nil {
		t.Fatalf("FAIL model: %v", err)
	}
	return m
}

func TestRequiredOptionalNullDistinct(t *testing.T) {
	m := build(t, `{"$defs":{"Tool":{"type":"object","required":["name"],"properties":{
		"name":{"type":"string"},
		"inputSchema":{"type":"object"},
		"nullable":{"type":["string","null"]},
		"zero":{"type":"integer","default":0}
	}}}}`)
	tool := m.Definitions["Tool"]
	if len(tool.Required) != 1 || tool.Required[0] != "name" {
		t.Fatalf("FAIL required: %v", tool.Required)
	}
	if _, ok := tool.Properties["inputSchema"]; !ok {
		t.Fatal("FAIL optional property collapsed")
	}
	nullable := tool.Properties["nullable"]
	if nullable.Type != "string|null" {
		t.Fatalf("FAIL nullable union collapsed: %q", nullable.Type)
	}
	zero := tool.Properties["zero"]
	if _, collapsed := zero.Annotations["default"]; !collapsed {
		t.Fatal("FAIL default annotation dropped")
	}
}

func TestExactNumberPrecision(t *testing.T) {
	m := build(t, `{"$defs":{"N":{"type":"number","minimum":0.30000000000000004,"maximum":1e300}}}`)
	n := m.Definitions["N"]
	if n.Minimum != "0.30000000000000004" {
		t.Fatalf("FAIL minimum token: %q", n.Minimum)
	}
	if n.Maximum != "1e300" {
		t.Fatalf("FAIL maximum token: %q", n.Maximum)
	}
}

func TestObjectOpennessPreserved(t *testing.T) {
	m := build(t, `{"$defs":{
		"Closed":{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":false},
		"Open":{"type":"object","properties":{"a":{"type":"string"}}},
		"Schemaed":{"type":"object","additionalProperties":{"type":"string"}}
	}}`)
	closed := m.Definitions["Closed"]
	if closed.Open == nil || *closed.Open {
		t.Fatal("FAIL closed object lost additionalProperties:false")
	}
	open := m.Definitions["Open"]
	if open.Open != nil {
		t.Fatal("FAIL unspecified openness collapsed to a value")
	}
	schemaed := m.Definitions["Schemaed"]
	if schemaed.Open == nil || !*schemaed.Open {
		t.Fatal("FAIL schemaed additionalProperties lost")
	}
	if _, ok := schemaed.Properties["<additional>"]; !ok {
		t.Fatal("FAIL additionalProperties schema dropped")
	}
}

func TestCompositionsStructural(t *testing.T) {
	m := build(t, `{"$defs":{"U":{"anyOf":[{"type":"string"},{"type":"object","required":["uri"]}]}}}`)
	u := m.Definitions["U"]
	alternatives := u.Compositions["anyOf"]
	if len(alternatives) != 2 {
		t.Fatalf("FAIL anyOf collapsed: %d", len(alternatives))
	}
	if alternatives[0].Type != "string" || alternatives[1].Type != "object" {
		t.Fatal("FAIL anyOf alternatives mangled")
	}
}

func TestEnumOpennessRecorded(t *testing.T) {
	m := build(t, `{"$defs":{"Role":{"type":"string","enum":["user","assistant"]}}}`)
	role := m.Definitions["Role"]
	if len(role.Enum) != 2 || role.Enum[0] != "user" {
		t.Fatalf("FAIL enum: %v", role.Enum)
	}
}

func TestKeywordClassificationTotal(t *testing.T) {
	m := build(t, `{"$defs":{"K":{"type":"object","title":"a title","description":"described","x-vendor":"kept","properties":{"a":{"type":"string"}}}}}`)
	k := m.Definitions["K"]
	for _, keyword := range []string{"type", "properties", "title", "description", "x-vendor"} {
		if _, ok := k.Keywords[keyword]; !ok {
			t.Fatalf("FAIL keyword %q not inventoried", keyword)
		}
	}
	if k.Keywords["x-vendor"] != ClassAnnotation {
		t.Fatal("FAIL unknown keyword defaulted to assertion")
	}
	if _, kept := k.Annotations["title"]; !kept {
		t.Fatal("FAIL annotation dropped")
	}
}

func TestPinnedKeywordInventory(t *testing.T) {
	if len(pinnedKeywords) != 17 {
		t.Fatalf("FAIL pinned keyword inventory: %d want 17", len(pinnedKeywords))
	}
}

func TestBrokenDefinitionsFailWithPointer(t *testing.T) {
	doc, err := load.New().LoadBytes("schema", []byte(`{"$defs":{"Bad":{"type":42}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(doc); err == nil {
		t.Fatal("FAIL non-string type accepted")
	} else {
		if got := err.Error(); !contains(got, "#/$defs/Bad") {
			t.Fatalf("FAIL error lacks source pointer: %s", got)
		}
	}
}

func TestNoDefsFails(t *testing.T) {
	doc, err := load.New().LoadBytes("schema", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(doc); err == nil {
		t.Fatal("FAIL empty document accepted")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(haystack) > 0 && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
