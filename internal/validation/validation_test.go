package validation

import (
	"encoding/json"
	"strings"
	"testing"
)

func compileOK(t *testing.T, a *Adapter, identity, body string) *Compiled {
	t.Helper()
	doc, err := decodeDoc(body)
	if err != nil {
		t.Fatalf("FAIL decode %s: %v", identity, err)
	}
	schema, err := a.Compile(identity, "https://example.invalid/"+identity+".json", doc)
	if err != nil {
		t.Fatalf("FAIL compile %s: %v", identity, err)
	}
	return schema
}

func TestCompileAndValidateBasic(t *testing.T) {
	a := New(16)
	schema := compileOK(t, a, "basic", `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`)
	if err := a.Validate(schema, map[string]any{"name": "x"}); err != nil {
		t.Fatalf("FAIL valid instance rejected: %v", err)
	}
	if err := a.Validate(schema, map[string]any{}); err == nil {
		t.Fatal("FAIL invalid instance accepted")
	}
}

func TestExactNumbers(t *testing.T) {
	a := New(16)
	schema := compileOK(t, a, "num", `{"type":"number","minimum":0.30000000000000004}`)
	// Decode wire bytes through the exact decoder: the instance's number
	// tokens must survive decoding for validation to compare them.
	var instance any
	if err := exactUnmarshal(`0.30000000000000004`, &instance); err != nil {
		t.Fatal(err)
	}
	if err := a.Validate(schema, instance); err != nil {
		t.Fatalf("FAIL exact minimum rejected: %v", err)
	}
	var lower any
	if err := exactUnmarshal(`0.3`, &lower); err != nil {
		t.Fatal(err)
	}
	if err := a.Validate(schema, lower); err == nil {
		t.Fatal("FAIL value below exact minimum accepted")
	}
	// The decoded value must be a json.Number, not a float64: this is the
	// assertion the rounding mutation breaks.
	if _, ok := instance.(json.Number); !ok {
		t.Fatalf("FAIL exact decode produced %T, not json.Number", instance)
	}
}

func TestCacheReuse(t *testing.T) {
	a := New(16)
	compileOK(t, a, "cached", `{"type":"string"}`)
	compileOK(t, a, "cached", `{"type":"string"}`)
	if a.hits["cached"] != 1 {
		t.Fatalf("FAIL cache hit not recorded: %d", a.hits["cached"])
	}
	if len(a.compiled) != 1 {
		t.Fatalf("FAIL cache grew on identity reuse: %d", len(a.compiled))
	}
}

func TestRetentionBound(t *testing.T) {
	a := New(2)
	compileOK(t, a, "one", `{"type":"string"}`)
	compileOK(t, a, "two", `{"type":"string"}`)
	if _, err := a.Compile("three", "https://example.invalid/three.json", mustDoc(t, `{"type":"string"}`)); err == nil {
		t.Fatal("FAIL retention bound not enforced")
	} else if !strings.Contains(err.Error(), "cache full") {
		t.Fatalf("FAIL unexpected bound error: %v", err)
	}
}

func TestInvalidSchemaRejected(t *testing.T) {
	a := New(16)
	if _, err := a.Compile("broken", "https://example.invalid/broken.json", mustDoc(t, `{"type":"not-a-type"}`)); err == nil {
		t.Fatal("FAIL invalid schema compiled")
	}
}

func TestLocalReferenceResolution(t *testing.T) {
	a := New(16)
	doc, err := decodeDoc(`{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Register("https://example.invalid/defs/basic.json", doc); err != nil {
		t.Fatal(err)
	}
	root := compileOK(t, a, "root", `{"type":"object","properties":{"inner":{"$ref":"https://example.invalid/defs/basic.json"}}}`)
	if err := a.Validate(root, map[string]any{"inner": map[string]any{"name": "ok"}}); err != nil {
		t.Fatalf("FAIL local reference unresolved: %v", err)
	}
	if err := a.Validate(root, map[string]any{"inner": map[string]any{}}); err == nil {
		t.Fatal("FAIL referenced schema not enforced")
	}
}

// decodeDoc decodes with exact number tokens preserved.
func decodeDoc(body string) (any, error) {
	var doc any
	if err := exactUnmarshal(body, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func mustDoc(t *testing.T, body string) any {
	t.Helper()
	doc, err := decodeDoc(body)
	if err != nil {
		t.Fatalf("FAIL decode: %v", err)
	}
	return doc
}
