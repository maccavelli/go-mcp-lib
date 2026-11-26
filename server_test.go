package mcp

import (
	"context"
	"strings"
	"testing"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	server, err := NewServer(&ServerOptions{
		Implementation: &mcplib.Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		},
	})
	if err != nil {
		t.Fatalf("FAIL NewServer: %v", err)
	}
	return server
}

func TestServerOptionsValidation(t *testing.T) {
	cases := []struct {
		name string
		opts *ServerOptions
		want string
	}{
		{"nil options", nil, "Implementation is required"},
		{"missing implementation", &ServerOptions{}, "Implementation is required"},
		{"negative page size", &ServerOptions{
			Implementation: &mcplib.Implementation{Name: "x", Version: "1"},
			PageSize:       -1,
		}, "PageSize must not be negative"},
		{"threshold without keepalive", &ServerOptions{
			Implementation:            &mcplib.Implementation{Name: "x", Version: "1"},
			KeepAliveFailureThreshold: 3,
		}, "has no effect without KeepAlive"},
	}
	for _, tc := range cases {
		_, err := NewServer(tc.opts)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("FAIL %s: got %v, want substring %q", tc.name, err, tc.want)
		}
	}
}

func TestAddToolDuplicateRejected(t *testing.T) {
	server := newTestServer(t)
	addHandler := func(ctx context.Context, req *ToolRequest, in map[string]any) (*mcplib.CallToolResult, map[string]any, error) {
		return nil, nil, nil
	}
	if err := AddTool(server, &ToolRegistration{Name: "dup"}, addHandler); err != nil {
		t.Fatalf("FAIL first registration: %v", err)
	}
	err := AddTool(server, &ToolRegistration{Name: "dup"}, addHandler)
	if err == nil {
		t.Fatal("FAIL duplicate registration accepted")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("FAIL wrong error: %v", err)
	}
}

func TestAddToolValidation(t *testing.T) {
	server := newTestServer(t)
	if err := AddTool[map[string]any, map[string]any](server, &ToolRegistration{}, nil); err == nil {
		t.Error("FAIL empty name accepted")
	}
	handler := func(ctx context.Context, req *ToolRequest, in map[string]any) (*mcplib.CallToolResult, map[string]any, error) {
		return nil, nil, nil
	}
	if err := AddTool[map[string]any, map[string]any](server, &ToolRegistration{Name: "x"}, nil); err == nil {
		t.Error("FAIL nil handler accepted")
	}
	if err := AddTool(nil, &ToolRegistration{Name: "x"}, handler); err == nil {
		t.Error("FAIL nil server accepted")
	}
}

func TestConcurrentServersSameToolName(t *testing.T) {
	// G13: admission is instance-scoped; two servers may register the
	// same tool name independently.
	a, b := newTestServer(t), newTestServer(t)
	handler := func(ctx context.Context, req *ToolRequest, in map[string]any) (*mcplib.CallToolResult, map[string]any, error) {
		return nil, nil, nil
	}
	if err := AddTool(a, &ToolRegistration{Name: "shared"}, handler); err != nil {
		t.Fatalf("FAIL server a: %v", err)
	}
	if err := AddTool(b, &ToolRegistration{Name: "shared"}, handler); err != nil {
		t.Fatalf("FAIL server b: %v — instance scoping broken", err)
	}
}
