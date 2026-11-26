package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestDeclareOutputSchemaValidation: an invalid output schema
// rejects the registration itself; nothing permissive substitutes.
func TestDeclareOutputSchemaValidation(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *ToolRequest, in outInput) (*mcplib.CallToolResult, outOutput, error) {
		return nil, outOutput{Count: 1}, nil
	}
	// Invalid schema: type is not a JSON Schema type.
	err := AddTool(server, &ToolRegistration{
		Name:         "bad",
		OutputSchema: map[string]any{"type": "not-a-type"},
	}, handler)
	if err == nil {
		t.Fatal("FAIL invalid output schema accepted at registration")
	}
	if !errors.Is(err, ErrRegistration) {
		t.Fatalf("FAIL not classified as registration error: %v", err)
	}
	if !strings.Contains(err.Error(), "output schema rejected") {
		t.Fatalf("FAIL wrong message: %v", err)
	}
	// The failed registration must not linger.
	if err := AddTool(server, &ToolRegistration{Name: "bad"}, handler); err != nil {
		t.Fatalf("FAIL retry after rejection: %v", err)
	}
}

// TestOutputValidationBeforeEmission: a handler whose structured
// output violates the declared schema is refused before the result
// leaves the chain.
func TestOutputValidationBeforeEmission(t *testing.T) {
	server := newTestServer(t)
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{"type": "integer", "minimum": float64(0)},
		},
		"required":             []any{"count"},
		"additionalProperties": false,
	}
	handler := func(ctx context.Context, req *ToolRequest, in outInput) (*mcplib.CallToolResult, outOutput, error) {
		return nil, outOutput{Count: -1}, nil // violates minimum
	}
	if err := AddTool(server, &ToolRegistration{
		Name:         "emit-check",
		OutputSchema: schema,
	}, handler); err != nil {
		t.Fatalf("FAIL AddTool: %v", err)
	}

	snapshot := server.chainSnapshot()
	final := sdk.MethodHandler(func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
		// Simulate the SDK dispatch result: structured content that
		// violates the declared minimum.
		return &sdk.CallToolResult{
			StructuredContent: map[string]any{"count": -1},
		}, nil
	})
	wrapped := snapshot.wrap(final)
	callReq := &sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{Name: "emit-check"}}
	_, err := wrapped(context.Background(), "tools/call", callReq)
	if err == nil {
		t.Fatal("FAIL violating structured output passed emission")
	}
	if !errors.Is(err, ErrToolCall) {
		t.Fatalf("FAIL emission rejection not typed: %v", err)
	}
	if !strings.Contains(err.Error(), "rejected by declared schema") {
		t.Fatalf("FAIL wrong rejection: %v", err)
	}
}

// TestOutputPassesWithoutDeclaration and valid declared outputs both
// reach the emission boundary unblocked.
func TestValidOutputPassesEmission(t *testing.T) {
	server := newTestServer(t)
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{"type": "integer"},
		},
		"required": []any{"count"},
	}
	handler := func(ctx context.Context, req *ToolRequest, in outInput) (*mcplib.CallToolResult, outOutput, error) {
		return nil, outOutput{Count: 1}, nil
	}
	if err := AddTool(server, &ToolRegistration{
		Name:         "emit-ok",
		OutputSchema: schema,
	}, handler); err != nil {
		t.Fatalf("FAIL AddTool: %v", err)
	}
	snapshot := server.chainSnapshot()
	final := sdk.MethodHandler(func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
		return &sdk.CallToolResult{
			StructuredContent: map[string]any{"count": 1},
		}, nil
	})
	wrapped := snapshot.wrap(final)
	callReq := &sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{Name: "emit-ok"}}
	result, err := wrapped(context.Background(), "tools/call", callReq)
	if err != nil {
		t.Fatalf("FAIL valid structured output rejected: %v", err)
	}
	if result == nil {
		t.Fatal("FAIL result lost at emission")
	}
}

// TestNoDeclarationNoCheck: absence of a declared schema is not a
// failure; the emission layer passes results through.
func TestNoDeclarationNoCheck(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *ToolRequest, in outInput) (*mcplib.CallToolResult, outOutput, error) {
		return nil, outOutput{}, nil
	}
	if err := AddTool(server, &ToolRegistration{Name: "bare"}, handler); err != nil {
		t.Fatal(err)
	}
	if _, ok := server.output.schemas["bare"]; ok {
		t.Fatal("FAIL undeclared tool gained a schema entry")
	}
	if err := server.output.check("bare", map[string]any{"anything": true}); err != nil {
		t.Fatalf("FAIL undeclared tool checked: %v", err)
	}
}

type outInput struct {
	X bool `json:"x"`
}

type outOutput struct {
	Count int `json:"count"`
}
