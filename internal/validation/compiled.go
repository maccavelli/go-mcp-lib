package validation

import "fmt"

// Compiled is the opaque handle for a compiled schema. It exists so
// other packages in this module can hold compiled schemas without
// importing the validator's concrete types into their exported API.
type Compiled struct {
	schema compiledSchema
}

type compiledSchema interface {
	validateInstance(value any) error
}

// Validate checks a decoded instance against the compiled schema.
func (c *Compiled) Validate(value any) error {
	if c == nil || c.schema == nil {
		return nil
	}
	return c.schema.validateInstance(value)
}

// WrapCompiled adapts the concrete compiled representation. It is
// intentionally unexported-typed at the boundary: the validator's own
// Compile hands back a Compiled, never a concrete type.
func wrapCompiled(schema any) (*Compiled, error) {
	if s, ok := schema.(interface {
		Validate(v any) error
	}); ok {
		return &Compiled{schema: concrete{s: s}}, nil
	}
	return nil, fmt.Errorf("validation: unsupported compiled representation")
}

type concrete struct {
	s interface{ Validate(v any) error }
}

func (c concrete) validateInstance(value any) error {
	return c.s.Validate(value)
}
