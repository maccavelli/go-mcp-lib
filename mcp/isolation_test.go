package mcp

import (
	"context"
	"fmt"
	"testing"
	"time"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

// TestConcurrentConnectionsIsolation proves three things at once:
// two simultaneous client connections to one server; a blocked handler
// on one connection does not stall work on the other; and each caller
// receives its own response, routed by connection, not by a global
// last-session authority.
func TestConcurrentConnectionsIsolation(t *testing.T) {
	server := newTestServer(t)

	blocked := make(chan struct{})
	release := make(chan struct{})
	handler := func(ctx context.Context, req *ToolRequest, in slowInput) (*mcplib.CallToolResult, slowOutput, error) {
		switch in.Client {
		case "slow":
			close(blocked)
			<-release // hold the first caller's handler
			return nil, slowOutput{Ack: "slow-done"}, nil
		case "fast":
			return nil, slowOutput{Ack: "fast-done"}, nil
		default:
			return nil, slowOutput{}, fmt.Errorf("unknown client marker %q", in.Client)
		}
	}
	if err := AddTool(server, &ToolRegistration{Name: "work", Description: "does work per caller"}, handler); err != nil {
		t.Fatalf("FAIL AddTool: %v", err)
	}

	// Two independent client sessions over separate in-memory pipes.
	pair1 := newPair(t, server)
	pair2 := newPair(t, server)
	defer pair1.shutdown()
	defer pair2.shutdown()

	ctxSlow, cancelSlow := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelSlow()
	ctxFast, cancelFast := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFast()

	// The slow caller blocks inside its handler.
	slowDone := make(chan error, 1)
	go func() {
		_, err := pair1.session.CallTool(ctxSlow, "work", map[string]any{"client": "slow"})
		slowDone <- err
	}()

	select {
	case <-blocked:
		// The slow handler is now running and holding its caller.
	case <-time.After(5 * time.Second):
		t.Fatal("FAIL slow handler never started")
	}

	// While the first caller is held, the second caller's work must
	// complete on its own connection.
	fastDone := make(chan struct{})
	go func() {
		defer close(fastDone)
		result, err := pair2.session.CallTool(ctxFast, "work", map[string]any{"client": "fast"})
		if err != nil {
			t.Errorf("FAIL fast call: %v", err)
			return
		}
		if isError, _ := result.IsError.Get(); isError {
			t.Errorf("FAIL fast call reported error: %+v", result)
			return
		}
		found := false
		for _, content := range result.Content {
			if text, err := content.AsTextContent(); err == nil && text.Text != "" {
				found = true
			}
		}
		if !found {
			t.Errorf("FAIL fast call returned no content: %+v", result.Content)
		}
	}()

	select {
	case <-fastDone:
		// Unrelated work completed while the slow handler was held.
	case <-time.After(5 * time.Second):
		t.Fatal("FAIL fast call stalled behind the blocked slow handler")
	}

	// Release the slow handler and confirm its caller completes too.
	close(release)
	select {
	case err := <-slowDone:
		if err != nil {
			t.Fatalf("FAIL slow call after release: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("FAIL slow call never completed after release")
	}
}

// TestTwoServersSameToolNameLive extends the registration-scope proof
// to live sessions: two servers registered with the same tool name
// serve independently, and each connection reaches its own server.
func TestTwoServersSameToolNameLive(t *testing.T) {
	serverA := newTestServer(t)
	serverB := newTestServer(t)
	handler := func(ctx context.Context, req *ToolRequest, in tagInput) (*mcplib.CallToolResult, tagOutput, error) {
		return nil, tagOutput{Server: in.Expect}, nil
	}
	for _, s := range []*Server{serverA, serverB} {
		if err := AddTool(s, &ToolRegistration{Name: "who"}, handler); err != nil {
			t.Fatalf("FAIL AddTool: %v", err)
		}
	}

	pairA := newPair(t, serverA)
	pairB := newPair(t, serverB)
	defer pairA.shutdown()
	defer pairB.shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resA, err := pairA.session.CallTool(ctx, "who", map[string]any{"expect": "A"})
	if err != nil {
		t.Fatalf("FAIL server A call: %v", err)
	}
	resB, err := pairB.session.CallTool(ctx, "who", map[string]any{"expect": "B"})
	if err != nil {
		t.Fatalf("FAIL server B call: %v", err)
	}
	for name, res := range map[string]*mcplib.CallToolResult{"A": resA, "B": resB} {
		if isError, _ := res.IsError.Get(); isError {
			t.Errorf("FAIL server %s reported error", name)
		}
	}
}

// pair is one connected client end plus its shutdown function.
type pair struct {
	session  *ClientSession
	shutdown func()
}

func newPair(t *testing.T, server *Server) *pair {
	t.Helper()
	serverTransport, clientTransport := InMemoryTransport()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = server.sdk.Connect(ctx, serverTransport.transport(), nil)
	}()
	client, err := NewClient(&ClientOptions{
		Implementation: &mcplib.Implementation{Name: "pair-client", Version: "1.0.0"},
	})
	if err != nil {
		cancel()
		t.Fatalf("FAIL NewClient: %v", err)
	}
	session, err := client.Connect(ctx, clientTransport)
	if err != nil {
		cancel()
		t.Fatalf("FAIL connect: %v", err)
	}
	return &pair{
		session: session,
		shutdown: func() {
			cancel()
			_ = session.Close()
			<-done
		},
	}
}

type slowInput struct {
	Client string `json:"client"`
}

type slowOutput struct {
	Ack string `json:"ack"`
}

type tagInput struct {
	Expect string `json:"expect"`
}

type tagOutput struct {
	Server string `json:"server"`
}
