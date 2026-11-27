package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

// startPair connects an owned server and client over the in-memory
// transport and returns both ends plus a shutdown function.
func startPair(t *testing.T, server *Server) (*ClientSession, func()) {
	t.Helper()
	serverTransport, clientTransport := InMemoryTransport()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = server.sdk.Connect(ctx, serverTransport.transport(), nil)
	}()

	client, err := NewClient(&ClientOptions{
		Implementation: &mcplib.Implementation{Name: "test-client", Version: "1.0.0"},
	})
	if err != nil {
		cancel()
		t.Fatalf("FAIL NewClient: %v", err)
	}
	session, err := client.Connect(ctx, clientTransport)
	if err != nil {
		cancel()
		t.Fatalf("FAIL client connect: %v", err)
	}
	shutdown := func() {
		cancel()
		_ = session.Close()
		<-done
	}
	return session, shutdown
}

func TestRoundTripToolCall(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *ToolRequest, in echoInput) (*mcplib.CallToolResult, echoOutput, error) {
		if req.Params.Name != "echo" || req.Session() == nil || req.ProtocolVersion() == "" {
			t.Error("FAIL owned tool context lost request or connection")
		}
		if in.Message != "ping" {
			return nil, echoOutput{}, nil
		}
		return nil, echoOutput{Reply: "pong: " + in.Message}, nil
	}
	if err := AddTool(server, &ToolRegistration{
		Name:        "echo",
		Description: "echoes the message back with a prefix",
	}, handler); err != nil {
		t.Fatalf("FAIL AddTool: %v", err)
	}

	session, shutdown := startPair(t, server)
	defer shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := session.CallTool(ctx, "echo", map[string]any{"message": "ping"})
	if err != nil {
		t.Fatalf("FAIL CallTool: %v", err)
	}
	if isError, _ := result.IsError.Get(); isError {
		t.Fatalf("FAIL tool reported error: %+v", result)
	}
	found := false
	for _, content := range result.Content {
		if text, err := content.AsTextContent(); err == nil &&
			strings.Contains(text.Text, "pong: ping") {
			found = true
		}
	}
	if !found {
		t.Fatalf("FAIL expected reply text in result: %+v", result.Content)
	}
}

func TestRoundTripListTools(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *ToolRequest, in echoInput) (*mcplib.CallToolResult, echoOutput, error) {
		return nil, echoOutput{}, nil
	}
	if err := AddTool(server, &ToolRegistration{Name: "one"}, handler); err != nil {
		t.Fatal(err)
	}
	if err := AddTool(server, &ToolRegistration{Name: "two"}, handler); err != nil {
		t.Fatal(err)
	}

	session, shutdown := startPair(t, server)
	defer shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := session.ListTools(ctx)
	if err != nil {
		t.Fatalf("FAIL ListTools: %v", err)
	}
	if len(result.Tools) != 2 {
		t.Fatalf("FAIL expected 2 tools, got %d", len(result.Tools))
	}
	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
		if tool.InputSchema == nil {
			t.Errorf("FAIL tool %q has no input schema", tool.Name)
		}
	}
	if !names["one"] || !names["two"] {
		t.Fatalf("FAIL missing tools: %v", names)
	}
}

func TestCallToolRejectsEmptyName(t *testing.T) {
	server := newTestServer(t)
	session, shutdown := startPair(t, server)
	defer shutdown()
	ctx := context.Background()
	if _, err := session.CallTool(ctx, "", nil); err == nil {
		t.Fatal("FAIL empty tool name accepted")
	}
}

func TestCallToolAfterClose(t *testing.T) {
	server := newTestServer(t)
	session, shutdown := startPair(t, server)
	defer shutdown()
	if err := session.Close(); err != nil {
		t.Fatalf("FAIL close: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := session.CallTool(ctx, "anything", nil)
	if err == nil {
		t.Fatal("FAIL call after close accepted")
	}
}

func TestCallToolDeadline(t *testing.T) {
	// Context-first blocking: a cancelled deadline must abort the call
	// rather than hang.
	server := newTestServer(t)
	slow := func(ctx context.Context, req *ToolRequest, in echoInput) (*mcplib.CallToolResult, echoOutput, error) {
		<-ctx.Done()
		return nil, echoOutput{}, ctx.Err()
	}
	if err := AddTool(server, &ToolRegistration{Name: "slow"}, slow); err != nil {
		t.Fatal(err)
	}
	session, shutdown := startPair(t, server)
	defer shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := session.CallTool(ctx, "slow", map[string]any{"message": "x"})
	if err == nil {
		t.Fatal("FAIL cancelled call returned success")
	}
}

// echoInput/echoOutput are the typed halves of the test tool.
type echoInput struct {
	Message string `json:"message"`
}

type echoOutput struct {
	Reply string `json:"reply"`
}
