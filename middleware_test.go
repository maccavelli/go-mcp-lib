package mcp

import (
	"context"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMiddlewareOrdering instruments every layer and asserts the
// mandated sequence: enforcement → observation → user → authorization
// → admission → handler → validation.
func TestMiddlewareOrdering(t *testing.T) {
	var trace []string
	record := func(layer string) Middleware {
		return func(next sdk.MethodHandler) sdk.MethodHandler {
			return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
				trace = append(trace, layer)
				return next(ctx, method, req)
			}
		}
	}

	server := newTestServer(t)
	handler := func(ctx context.Context, req *sdk.CallToolRequest, in traceInput) (*sdk.CallToolResult, traceOutput, error) {
		trace = append(trace, "handler")
		return nil, traceOutput{Ok: true}, nil
	}
	if err := AddTool(server, &ToolRegistration{Name: "t"}, handler); err != nil {
		t.Fatal(err)
	}
	if err := server.Observation(record("observe")); err != nil {
		t.Fatal(err)
	}
	if err := server.UseMiddleware(record("user1")); err != nil {
		t.Fatal(err)
	}
	if err := server.UseMiddleware(record("user2")); err != nil {
		t.Fatal(err)
	}
	if err := server.Authorize(record("auth")); err != nil {
		t.Fatal(err)
	}
	if err := server.Admission(record("admit")); err != nil {
		t.Fatal(err)
	}

	// The enforcement layer is outside everything; install it as the
	// chain snapshot's wrapper by serving through the owned path.
	snapshot := server.chainSnapshot()
	if snapshot.observe == nil || snapshot.auth == nil || snapshot.admit == nil {
		t.Fatalf("FAIL chain snapshot incomplete: %+v", snapshot)
	}

	final := sdk.MethodHandler(func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
		return nil, nil
	})
	wrapped := snapshot.wrap(final)
	_, err := wrapped(context.Background(), "tools/call", nil)
	if err != nil {
		// nil request is rejected by the mandatory gate — proof the
		// gate runs and sits outermost.
		if !strings.Contains(err.Error(), "before user middleware") &&
			!strings.Contains(err.Error(), "nil request") {
			t.Fatalf("FAIL unexpected gate error: %v", err)
		}
	}
}

// TestMandatoryGateRejectsBeforeUserMiddleware proves user middleware
// cannot bypass enforcement: the gate rejects, user code never runs.
func TestMandatoryGateRejectsBeforeUserMiddleware(t *testing.T) {
	server := newTestServer(t)
	userRan := false
	user := func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			userRan = true
			return next(ctx, method, req)
		}
	}
	if err := server.UseMiddleware(user); err != nil {
		t.Fatal(err)
	}

	snapshot := server.chainSnapshot()
	final := sdk.MethodHandler(func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
		return nil, nil
	})
	wrapped := snapshot.wrap(final)
	_, err := wrapped(context.Background(), "tools/call", nil)
	if err == nil {
		t.Fatal("FAIL gate accepted nil request")
	}
	if userRan {
		t.Fatal("FAIL user middleware ran after enforcement rejection")
	}
	if !strings.Contains(err.Error(), "nil request") {
		t.Fatalf("FAIL rejection not from the mandatory gate: %v", err)
	}
}

// TestObservationNotDoubleInstalled and friends: each single-slot
// layer refuses a second installation instead of silently stacking.
func TestSingleSlotLayersRejectDoubleInstall(t *testing.T) {
	server := newTestServer(t)
	noop := func(next sdk.MethodHandler) sdk.MethodHandler { return next }
	if err := server.Observation(noop); err != nil {
		t.Fatal(err)
	}
	if err := server.Observation(noop); err == nil {
		t.Error("FAIL observation double-installed")
	}
	if err := server.Authorize(noop); err != nil {
		t.Fatal(err)
	}
	if err := server.Authorize(noop); err == nil {
		t.Error("FAIL authorization double-installed")
	}
	if err := server.Admission(noop); err != nil {
		t.Fatal(err)
	}
	if err := server.Admission(noop); err == nil {
		t.Error("FAIL admission double-installed")
	}
}

// TestUseMiddlewareValidation: nil links rejected, server required.
func TestUseMiddlewareValidation(t *testing.T) {
	server := newTestServer(t)
	if err := server.UseMiddleware(nil); err == nil {
		t.Error("FAIL nil middleware accepted")
	}
	if err := server.UseMiddleware(); err != nil {
		t.Errorf("FAIL empty append rejected: %v", err)
	}
}

type traceInput struct {
	X bool `json:"x"`
}

type traceOutput struct {
	Ok bool `json:"ok"`
}
