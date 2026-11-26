package model

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

func TestRepairA07Vocabulary(t *testing.T) {
	for name, body := range map[string]string{
		"minLength":               `{"$defs":{"A":{"type":"string","minLength":1}}}`,
		"maxLength":               `{"$defs":{"A":{"type":"string","maxLength":2}}}`,
		"dynamic":                 `{"$defs":{"A":{"$dynamicRef":"#A"}}}`,
		"fractional-count":        `{"$defs":{"A":{"type":"array","minItems":1.5}}}`,
		"negative-count":          `{"$defs":{"A":{"type":"array","maxItems":-1}}}`,
		"invalid-vocabulary-flag": `{"$vocabulary":{"urn:example":1},"$defs":{"A":{}}}`,
		"invalid-vocabulary":      `{"$vocabulary":true,"$defs":{"A":{}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			doc, err := load.New().LoadBytes(name, []byte(body))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Build(doc); err == nil {
				t.Fatalf("FAIL unqualified or malformed keyword accepted: %s", name)
			}
		})
	}
	t.Run("integer-value-not-integer-spelling", func(t *testing.T) {
		m := build(t, `{"$defs":{"A":{"type":"array","minItems":1e0,"maxItems":2.0}}}`)
		node := m.Definitions["A"]
		if node.MinItems != 1 || node.MaxItems != 2 || node.NumberTokens["minItems"] != "1e0" {
			t.Fatal("FAIL exact integer-valued schema bounds lost")
		}
	})
	t.Run("annotations-stay-annotations", func(t *testing.T) {
		m := build(t, `{"$defs":{"A":{"type":"string","format":"application-format","x-example":{"kept":true}}}}`)
		if m.Definitions["A"].Keywords["format"] != ClassAnnotation || m.Definitions["A"].Annotations["x-example"] == nil {
			t.Fatal("FAIL arbitrary format/extension annotation was asserted or dropped")
		}
	})
	t.Run("escaped-source-pointer", func(t *testing.T) {
		doc, err := load.New().LoadBytes("pointer", []byte(`{"$defs":{"a/b":{"properties":{"c~d":{"type":42}}}}}`))
		if err != nil {
			t.Fatal(err)
		}
		_, err = Build(doc)
		if err == nil || !strings.Contains(err.Error(), "#/$defs/a~1b/properties/c~0d") {
			t.Fatalf("FAIL escaped diagnostic pointer: %v", err)
		}
	})
}
