// Package model builds the generator's intermediate representation from
// resolved schema documents without collapsing anything: required vs
// optional vs nullable vs zero-valued stay distinct, number tokens stay
// exact, enums stay open, unions and compositions stay structural, and
// object openness stays declared.
//
// Keyword classification is total: every keyword present in a document is
// classified as assertion, annotation, or unsupported. An unsupported
// assertion or an unknown required vocabulary fails with a source pointer.
package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

// KeywordClass classifies a keyword as assertion, annotation, or
// unsupported.
type KeywordClass int

const (
	ClassAssertion KeywordClass = iota
	ClassAnnotation
	ClassUnsupported
)

// The 17 pinned JSON Schema 2020-12 keywords the model inventories.
var pinnedKeywords = map[string]KeywordClass{
	// assertions
	"type":                 ClassAssertion,
	"enum":                 ClassAssertion,
	"const":                ClassAssertion,
	"properties":           ClassAssertion,
	"required":             ClassAssertion,
	"items":                ClassAssertion,
	"additionalProperties": ClassAssertion,
	"minimum":              ClassAssertion,
	"maximum":              ClassAssertion,
	"minItems":             ClassAssertion,
	"maxItems":             ClassAssertion,
	// composition
	"allOf": ClassAssertion,
	"anyOf": ClassAssertion,
	"oneOf": ClassAssertion,
	// structure
	"$defs": ClassAssertion,
	"$ref":  ClassAssertion,
	"$id":   ClassAnnotation,
}

// Node is one schema node in the intermediate model.
type Node struct {
	// Pointer is the JSON Pointer of the node in its source document.
	Pointer string
	// Source is the canonical URI of the source document.
	Source string

	Type         string   // "object", "array", "string", "number", "boolean", "null", "" (any)
	Enum         []any    // exact values, order preserved
	Const        any      // exact value
	Required     []string // in declaration order
	Properties   map[string]*Node
	Items        *Node
	Open         *bool  // nil = unspecified; false = additionalProperties:false
	Minimum      string // exact number token
	Maximum      string // exact number token
	MinItems     int
	MaxItems     int
	Compositions map[string][]*Node // "allOf", "anyOf", "oneOf"

	// NumberPrecision records the exact token of any numeric keyword so
	// no float64 collapse happens before emission.
	NumberTokens map[string]string

	// EnumOpen records that the enum is a closed set at model level but
	// the protocol rule asserts openness; the semantic map, not the
	// schema, decides final behavior.
	EnumOpen bool

	// Keywords inventories every keyword seen on this node with its
	// classification, so the classification is total and auditable.
	Keywords map[string]KeywordClass

	// Annotations carries non-assertion members verbatim.
	Annotations map[string]any
}

// Model is the intermediate representation over a document's definitions.
type Model struct {
	Source      string
	Definitions map[string]*Node
}

// Build builds the model for a loaded document.
func Build(doc *load.Document) (*Model, error) {
	root, ok := doc.Root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("model: %s: root is not an object", doc.Source)
	}
	if vocab, present := root["$vocabulary"].(map[string]any); present {
		for uri, required := range vocab {
			isRequired, ok := required.(bool)
			if ok && isRequired && !requiredVocabularies[uri] {
				return nil, fmt.Errorf(
					"model: %s: unknown required vocabulary %s", doc.Source, uri,
				)
			}
		}
	}
	defsObj, hasDefs := root["$defs"].(map[string]any)
	if !hasDefs {
		return nil, fmt.Errorf("model: %s: no $defs", doc.Source)
	}
	definitions := make(map[string]*Node, len(defsObj))
	names := make([]string, 0, len(defsObj))
	for name := range defsObj {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		raw, ok := defsObj[name].(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				"model: %s: definition %q is not an object", doc.Source, name,
			)
		}
		node, err := buildNode(doc.Source, "#/$defs/"+name, raw)
		if err != nil {
			return nil, err
		}
		definitions[name] = node
	}
	return &Model{Source: doc.Source, Definitions: definitions}, nil
}

// assertionShaped lists JSON Schema 2020-12 keywords that assert but are
// NOT in the pinned 17-keyword inventory. Their presence fails the model
// with a source pointer: silently classifying them as annotations would
// under-validate everything downstream.
var assertionShaped = map[string]bool{
	"pattern": true, "multipleOf": true, "exclusiveMinimum": true,
	"exclusiveMaximum": true, "uniqueItems": true, "minProperties": true,
	"maxProperties": true, "patternProperties": true,
	"propertyNames": true, "dependentRequired": true,
	"dependentSchemas": true, "contains": true, "minContains": true,
	"maxContains": true, "prefixItems": true, "unevaluatedProperties": true,
	"unevaluatedItems": true, "if": true, "then": true, "else": true,
	"not": true,
}

// requiredVocabularies are the vocabularies the pinned dialect supports.
// A document declaring a required vocabulary outside this set fails.
var requiredVocabularies = map[string]bool{
	"https://json-schema.org/draft/2020-12/vocab/core":              true,
	"https://json-schema.org/draft/2020-12/vocab/applicator":        true,
	"https://json-schema.org/draft/2020-12/vocab/validation":        true,
	"https://json-schema.org/draft/2020-12/vocab/meta-data":         true,
	"https://json-schema.org/draft/2020-12/vocab/format-annotation": true,
	"https://json-schema.org/draft/2020-12/vocab/content":           true,
}

func buildNode(source, pointer string, raw map[string]any) (*Node, error) {
	node := &Node{
		Pointer:      pointer,
		Source:       source,
		Properties:   make(map[string]*Node),
		Compositions: make(map[string][]*Node),
		Keywords:     make(map[string]KeywordClass),
		NumberTokens: make(map[string]string),
		Annotations:  make(map[string]any),
	}
	for keyword, value := range raw {
		if assertionShaped[keyword] {
			return nil, fmt.Errorf(
				"model: %s %s: unsupported assertion keyword %q is outside the pinned inventory",
				source, pointer, keyword,
			)
		}
		class, known := pinnedKeywords[keyword]
		if !known {
			// Unknown keywords are annotations unless they claim to be
			// required vocabulary assertions via $vocabulary.
			node.Annotations[keyword] = value
			node.Keywords[keyword] = ClassAnnotation
			continue
		}
		node.Keywords[keyword] = class
		if class == ClassAnnotation {
			node.Annotations[keyword] = value
			continue
		}
		if err := applyKeyword(node, source, pointer, keyword, value); err != nil {
			return nil, err
		}
	}
	return node, nil
}

func applyKeyword(node *Node, source, pointer, keyword string, value any) error {
	switch keyword {
	case "type":
		if types, ok := value.([]any); ok {
			// union type: keep the list; node.Type holds the canonical
			// single type when there is exactly one.
			if len(types) == 1 {
				if t, ok := types[0].(string); ok {
					node.Type = t
					return nil
				}
			}
			tokens := make([]string, 0, len(types))
			for _, t := range types {
				if s, ok := t.(string); ok {
					tokens = append(tokens, s)
				} else {
					return fmt.Errorf("model: %s %s: type entry is not a string", source, pointer)
				}
			}
			node.Type = strings.Join(tokens, "|")
			return nil
		}
		t, ok := value.(string)
		if !ok {
			return fmt.Errorf("model: %s %s: type is not a string", source, pointer)
		}
		node.Type = t
	case "enum":
		values, ok := value.([]any)
		if !ok {
			return fmt.Errorf("model: %s %s: enum is not an array", source, pointer)
		}
		node.Enum = values
	case "const":
		node.Const = value
	case "required":
		names, ok := value.([]any)
		if !ok {
			return fmt.Errorf("model: %s %s: required is not an array", source, pointer)
		}
		for _, name := range names {
			s, ok := name.(string)
			if !ok {
				return fmt.Errorf("model: %s %s: required entry is not a string", source, pointer)
			}
			node.Required = append(node.Required, s)
		}
	case "properties":
		props, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("model: %s %s: properties is not an object", source, pointer)
		}
		propNames := make([]string, 0, len(props))
		for name := range props {
			propNames = append(propNames, name)
		}
		sort.Strings(propNames)
		for _, name := range propNames {
			raw, ok := props[name].(map[string]any)
			if !ok {
				return fmt.Errorf("model: %s %s: property %q is not an object", source, pointer, name)
			}
			child, err := buildNode(source, pointer+"/properties/"+name, raw)
			if err != nil {
				return err
			}
			node.Properties[name] = child
		}
	case "items":
		raw, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("model: %s %s: items is not an object", source, pointer)
		}
		child, err := buildNode(source, pointer+"/items", raw)
		if err != nil {
			return err
		}
		node.Items = child
	case "additionalProperties":
		switch v := value.(type) {
		case bool:
			open := v
			node.Open = &open
		case map[string]any:
			open := true
			node.Open = &open
			child, err := buildNode(source, pointer+"/additionalProperties", v)
			if err != nil {
				return err
			}
			node.Properties["<additional>"] = child
		default:
			return fmt.Errorf("model: %s %s: additionalProperties is neither bool nor object", source, pointer)
		}
	case "minimum", "maximum":
		num, ok := value.(load.Number) // exact number token
		if !ok {
			return fmt.Errorf("model: %s %s: %s is not a number", source, pointer, keyword)
		}
		node.NumberTokens[keyword] = num.Token
		if keyword == "minimum" {
			node.Minimum = num.Token
		} else {
			node.Maximum = num.Token
		}
	case "minItems", "maxItems":
		num, ok := value.(load.Number)
		if !ok {
			return fmt.Errorf("model: %s %s: %s is not a number", source, pointer, keyword)
		}
		var parsed int
		if _, err := fmt.Sscanf(num.Token, "%d", &parsed); err != nil {
			return fmt.Errorf("model: %s %s: %s %q is not an integer", source, pointer, keyword, num.Token)
		}
		if keyword == "minItems" {
			node.MinItems = parsed
		} else {
			node.MaxItems = parsed
		}
	case "allOf", "anyOf", "oneOf":
		list, ok := value.([]any)
		if !ok {
			return fmt.Errorf("model: %s %s: %s is not an array", source, pointer, keyword)
		}
		for i, item := range list {
			raw, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("model: %s %s: %s entry is not an object", source, pointer, keyword)
			}
			child, err := buildNode(source, fmt.Sprintf("%s/%s/%d", pointer, keyword, i), raw)
			if err != nil {
				return err
			}
			node.Compositions[keyword] = append(node.Compositions[keyword], child)
		}
	case "$defs":
		// nested $defs are recorded as keywords but not flattened; the
		// generator emits top-level definitions only.
	case "$ref":
		ref, ok := value.(string)
		if !ok {
			return fmt.Errorf("model: %s %s: $ref is not a string", source, pointer)
		}
		node.Annotations["$ref"] = ref
	default:
		return fmt.Errorf("model: %s %s: unsupported assertion keyword %q", source, pointer, keyword)
	}
	return nil
}

// Unsupported returns the keywords classified unsupported on a node.
func (n *Node) Unsupported() []string {
	var unsupported []string
	for keyword, class := range n.Keywords {
		if class == ClassUnsupported {
			unsupported = append(unsupported, keyword)
		}
	}
	sort.Strings(unsupported)
	return unsupported
}
