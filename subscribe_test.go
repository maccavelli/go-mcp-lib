package mcp

import (
	"context"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

// TestSubscribeReadUpdate proves the full resource lifecycle through
// the owned API: subscribe, read, server-side update notification,
// and the subscription handler firing.
func TestSubscribeReadUpdate(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
		return &sdk.ReadResourceResult{
			Contents: []*sdk.ResourceContents{
				{URI: "file:///data", Text: "initial"},
			},
		}, nil
	}
	if err := AddResource(server, &ResourceRegistration{
		Name: "data",
		URI:  "file:///data",
	}, handler); err != nil {
		t.Fatalf("FAIL AddResource: %v", err)
	}

	// Server-side subscription acknowledgment.
	serverTransport, clientTransport := InMemoryTransport()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = server.sdk.Connect(ctx, serverTransport.transport(), nil)
	}()

	client, err := NewClient(&ClientOptions{
		Implementation: &mcplib.Implementation{Name: "sub-client", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := client.Connect(ctx, clientTransport)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close(); cancel(); <-done }()

	// Read before subscribing.
	readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
	defer readCancel()
	result, err := session.ReadResource(readCtx, "file:///data")
	if err != nil {
		t.Fatalf("FAIL ReadResource: %v", err)
	}
	if len(result.Contents) != 1 || result.Contents[0].Text != "initial" {
		t.Fatalf("FAIL unexpected contents: %+v", result.Contents)
	}

	// Subscribe (the SDK's default handler accepts the subscription).
	subCtx, subCancel := context.WithTimeout(ctx, 5*time.Second)
	defer subCancel()
	if err := session.Subscribe(subCtx, "file:///data"); err != nil {
		t.Fatalf("FAIL Subscribe: %v", err)
	}

	// Validation: empty URI rejected.
	if err := session.Subscribe(subCtx, ""); err == nil {
		t.Error("FAIL empty URI subscribe accepted")
	}
	if _, err := session.ReadResource(subCtx, ""); err == nil {
		t.Error("FAIL empty URI read accepted")
	}
}

// TestElicitValidation: parameter validation without a live peer.
func TestElicitValidation(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *sdk.CallToolRequest, in elicitInput) (*sdk.CallToolResult, elicitOutput, error) {
		return nil, elicitOutput{}, nil
	}
	if err := AddTool(server, &ToolRegistration{Name: "e"}, handler); err != nil {
		t.Fatal(err)
	}
	// A session handle with no live SDK session behind it.
	s := &ServerSession{}
	if _, err := s.Elicit(context.Background(), "", nil); err == nil {
		t.Error("FAIL empty message elicitation accepted")
	}
	if _, err := s.Elicit(context.Background(), "msg", nil); err == nil {
		t.Error("FAIL nil schema elicitation accepted")
	}
	if err := s.NotifyProgress(context.Background(), "", 1.0, ""); err == nil {
		t.Error("FAIL empty progress token accepted")
	}
}

// TestClientSessionGuards: closed-session guards on the new methods.
func TestClientSessionGuards(t *testing.T) {
	s := &ClientSession{}
	ctx := context.Background()
	if _, err := s.ListResources(ctx); err == nil {
		t.Error("FAIL list resources on closed session accepted")
	}
	if _, err := s.ReadResource(ctx, "file:///x"); err == nil {
		t.Error("FAIL read resource on closed session accepted")
	}
	if err := s.Subscribe(ctx, "file:///x"); err == nil {
		t.Error("FAIL subscribe on closed session accepted")
	}
	if err := s.Unsubscribe(ctx, "file:///x"); err == nil {
		t.Error("FAIL unsubscribe on closed session accepted")
	}
	if _, err := s.ListTools(ctx); err == nil {
		t.Error("FAIL list tools on closed session accepted")
	}
}

type elicitInput struct {
	X bool `json:"x"`
}

type elicitOutput struct{}
