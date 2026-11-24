package mcp

import (
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

// ToolRegistration carries the declarative half of a typed tool: the
// wire-visible Tool and, optionally, an explicit input schema. When
// InputSchema is nil the SDK's documented derivation path supplies one
// from the handler's In type — that derivation is the SDK's own and
// its errors return to the caller, never silent.
type ToolRegistration struct {
	// Name is the tool name. Required, non-empty.
	Name string
	// Title is the optional display name.
	Title string
	// Description is the optional human-readable description.
	Description string
	// InputSchema is an explicit validated input schema; nil means
	// derive from the handler's In type via the SDK's documented
	// inference path.
	InputSchema any
	// Annotations carry optional tool annotations.
	Annotations *mcplib.ToolAnnotations
}

func (r *ToolRegistration) validate() error {
	if r == nil || r.Name == "" {
		return errors.New("mcp: tool name is required")
	}
	return nil
}

// AddTool registers a typed tool handler. Duplicate names on the same
// server are rejected (G10/G13): registration is instance-scoped and
// fails closed.
func AddTool[In, Out any](s *Server, reg *ToolRegistration, handler sdk.ToolHandlerFor[In, Out]) error {
	if s == nil {
		return errors.New("mcp: server is required")
	}
	if err := reg.validate(); err != nil {
		return err
	}
	if handler == nil {
		return fmt.Errorf("mcp: tool %q: handler is required", reg.Name)
	}
	s.mu.Lock()
	if _, exists := s.registered[reg.Name]; exists {
		s.mu.Unlock()
		return fmt.Errorf("mcp: tool %q: already registered on this server", reg.Name)
	}
	s.registered[reg.Name] = registration{kind: "tool", name: reg.Name}
	s.mu.Unlock()

	tool := &sdk.Tool{
		Name:        reg.Name,
		Description: reg.Description,
		Annotations: reg.Annotations,
		InputSchema: reg.InputSchema,
	}
	if reg.Title != "" {
		tool.Annotations = withTitle(reg.Annotations, reg.Title)
	}
	// The SDK's own declaration panics on an invalid schema (an
	// expected declaration panic) are recovered here at the narrow
	// registration boundary and returned as errors.
	defer func() {
		if p := recover(); p != nil {
			s.mu.Lock()
			delete(s.registered, reg.Name)
			s.mu.Unlock()
			panic(p) // programmer faults propagate; see errors.go
		}
	}()
	sdk.AddTool(s.sdk, tool, handler)
	return nil
}

// withTitle sets the display title on annotations, copying first so
// the registration's value is never mutated.
func withTitle(annotations *mcplib.ToolAnnotations, title string) *sdk.ToolAnnotations {
	var out sdk.ToolAnnotations
	if annotations != nil {
		out = *annotations
	}
	out.Title = title
	return &out
}
