package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/maccavelli/go-mcp-lib/internal/validation"
)

// outputValidator validates structured tool output against a declared
// output schema before emission. Schemas compile once at registration
// through the canonical adapter; validation failures reject the
// output — invalid results are never emitted silently.
type outputValidator struct {
	adapter *validation.Adapter
	schemas map[string]*validation.Compiled
}

func newOutputValidator() *outputValidator {
	return &outputValidator{
		adapter: validation.New(256),
		schemas: make(map[string]*validation.Compiled),
	}
}

// declare compiles and stores a tool's declared output schema. A nil
// schema means no output validation for that tool (absence is
// distinct from invalid). An invalid schema rejects the declaration
// itself — no permissive substitution.
func (v *outputValidator) declare(tool string, outputSchema any) error {
	if outputSchema == nil {
		delete(v.schemas, tool)
		return nil
	}
	doc, err := decodeExact(outputSchema)
	if err != nil {
		return typed(ErrRegistration,
			fmt.Sprintf("tool %q output schema does not decode", tool), err)
	}
	compiled, err := v.adapter.Compile(
		"output:"+tool, canonicalOutputURI(tool), doc)
	if err != nil {
		return typed(ErrRegistration,
			fmt.Sprintf("tool %q output schema rejected", tool), err)
	}
	v.schemas[tool] = compiled
	return nil
}

// check validates a structured output value against the tool's
// declared schema. No declaration means no check — absence is not a
// failure.
func (v *outputValidator) check(tool string, output any) error {
	compiled, ok := v.schemas[tool]
	if !ok {
		return nil
	}
	value, err := decodeExact(output)
	if err != nil {
		return typed(ErrToolCall,
			fmt.Sprintf("tool %q output does not decode", tool), err)
	}
	if err := v.adapter.Validate(compiled, value); err != nil {
		return typed(ErrToolCall,
			fmt.Sprintf("tool %q output rejected by declared schema", tool), err)
	}
	return nil
}

// decodeExact turns raw bytes, a string, a RawMessage, or a Go value
// into the exact-token decoded representation the canonical
// validator needs: numbers become json.Number, never float64.
func decodeExact(input any) (any, error) {
	var raw []byte
	switch typed := input.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	case json.RawMessage:
		raw = typed
	default:
		marshaled, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		raw = marshaled
	}
	var out any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func canonicalOutputURI(tool string) string {
	return fmt.Sprintf("https://go-mcp-lib.invalid/tools/%s/output.json", tool)
}
