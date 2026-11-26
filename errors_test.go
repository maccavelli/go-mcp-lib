package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

func TestTypedErrorClassification(t *testing.T) {
	base := errors.New("boom")
	err := typed(ErrTransport, "pipe broke", base)

	if !errors.Is(err, ErrTransport) {
		t.Error("FAIL errors.Is kind sentinel")
	}
	if !errors.Is(err, base) {
		t.Error("FAIL errors.Is underlying cause")
	}
	var typedErr *TypedError
	if !errors.As(err, &typedErr) {
		t.Fatal("FAIL errors.As TypedError")
	}
	if typedErr.Kind != ErrTransport || typedErr.Msg != "pipe broke" {
		t.Errorf("FAIL fields: %+v", typedErr)
	}
	var sentinel *TypedError
	if errors.As(err, &sentinel) && sentinel.Cause != base {
		t.Errorf("FAIL cause lost: %+v", sentinel)
	}
}

func TestCallToolCancelledClassifies(t *testing.T) {
	server := newTestServer(t)
	stalled := make(chan struct{})
	release := make(chan struct{})
	handler := func(ctx context.Context, req *ToolRequest, in stallInput) (*mcplib.CallToolResult, stallOutput, error) {
		close(stalled)
		<-release
		return nil, stallOutput{}, ctx.Err()
	}
	if err := AddTool(server, &ToolRegistration{Name: "stall"}, handler); err != nil {
		t.Fatal(err)
	}
	session, shutdown := startPair(t, server)
	defer shutdown()

	// The caller's deadline expires while the handler is held.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	callDone := make(chan error, 1)
	go func() {
		_, err := session.CallTool(ctx, "stall", map[string]any{"ignored": true})
		callDone <- err
	}()

	select {
	case <-stalled:
		// Handler running; let the deadline expire.
	case <-time.After(5 * time.Second):
		t.Fatal("FAIL handler never started")
	}

	var err error
	select {
	case err = <-callDone:
		close(release)
		// Deadline fired while the handler was held.
	case <-time.After(10 * time.Second):
		close(release)
		t.Fatal("FAIL cancelled call never returned")
	}
	if err == nil {
		t.Fatal("FAIL cancelled call returned nil error")
	}
	if !errors.Is(err, ErrCancelled) {
		t.Errorf("FAIL cancellation not classified: %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("FAIL context cause lost: %v", err)
	}
}

func TestRegistrationErrorClassifies(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *ToolRequest, in stallInput) (*mcplib.CallToolResult, stallOutput, error) {
		return nil, stallOutput{}, nil
	}
	if err := AddTool(server, &ToolRegistration{Name: "x"}, handler); err != nil {
		t.Fatal(err)
	}
	err := AddTool(server, &ToolRegistration{Name: "x"}, handler)
	if !errors.Is(err, ErrRegistration) {
		t.Errorf("FAIL duplicate not classified: %v", err)
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Errorf("FAIL message lost: %v", err)
	}
}

type stallInput struct {
	Ignored bool `json:"ignored"`
}

type stallOutput struct{}
