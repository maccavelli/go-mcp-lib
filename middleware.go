package mcp

import (
	"context"
	"errors"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Middleware is an owned link in the server's receiving chain. It
// wraps a MethodHandler and returns the next one. Middleware sees the
// SDK request; owned guards run outside it by construction.
type Middleware func(sdk.MethodHandler) sdk.MethodHandler

// serverChain assembles the mandated ordering around the SDK's final
// handler. From outermost to innermost:
//
//  1. mandatory enforcement (raw/schema/version) — never skippable,
//     never reorderable; not user middleware at all
//  2. observation — surrounds the whole operation
//  3. application authorization — precedes side effects
//  4. admission — precedes handler execution
//  5. the SDK's own dispatch to the registered handler
//  6. result validation — after the handler and after any
//     transformation middleware the SDK runs
//
// User middleware is inserted between observation and authorization:
// it may observe and transform, but the mandatory layer sits outside
// it and admission sits inside it, so user code can neither bypass
// enforcement nor jump the admission queue.
type serverChain struct {
	observe  Middleware // observation: surrounds everything user-visible
	user     []Middleware
	auth     Middleware
	admit    Middleware
	validate Middleware // result validation after handler
}

// enforce is the mandatory outermost layer. It rejects before any
// user-visible middleware runs. It is not configurable.
func (c *serverChain) enforce(inner sdk.MethodHandler) sdk.MethodHandler {
	return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
		if err := mandatoryGate(ctx, method, req); err != nil {
			return nil, typed(ErrProtocol, "rejected before user middleware", err)
		}
		return inner(ctx, method, req)
	}
}

// wrap builds the full chain around the SDK's final handler.
func (c *serverChain) wrap(final sdk.MethodHandler) sdk.MethodHandler {
	next := final
	if c.validate != nil {
		next = c.validate(next)
	}
	if c.admit != nil {
		next = c.admit(next)
	}
	if c.auth != nil {
		next = c.auth(next)
	}
	for i := len(c.user) - 1; i >= 0; i-- {
		next = c.user[i](next)
	}
	if c.observe != nil {
		next = c.observe(next)
	}
	return c.enforce(next)
}

// outputValidation returns the chain's innermost validation link for
// a server: results validate after the handler and after every
// transformation middleware, before emission. Tools with a declared
// output schema have their structured content checked; undeclared
// tools pass through.
func (s *Server) outputValidation() Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			result, err := next(ctx, method, req)
			if err != nil {
				return result, err
			}
			// Only tools/call results carry structured output.
			call, ok := req.(*sdk.CallToolRequest)
			if !ok {
				return result, err
			}
			toolResult, ok := result.(*sdk.CallToolResult)
			if !ok || toolResult == nil || toolResult.StructuredContent == nil {
				return result, err
			}
			if verr := s.output.check(call.Params.Name, toolResult.StructuredContent); verr != nil {
				// Reject the emission: the result never leaves with a
				// declared-schema violation.
				return nil, verr
			}
			return result, err
		}
	}
}

// mandatoryGate is the raw/schema/version enforcement. It runs for
// every method, before user middleware can observe the request.
func mandatoryGate(ctx context.Context, method string, req sdk.Request) error {
	if method == "" {
		return errors.New("empty method")
	}
	// Raw-byte and schema validation occur in the guarded transports
	// (stage 7) before the SDK parses; this gate asserts the contract
	// at dispatch level and rejects structurally invalid requests.
	if req == nil {
		return errors.New("nil request")
	}
	return nil
}

// UseMiddleware appends user middleware to the server's receiving
// chain. The mandatory layer stays outside everything user-added;
// order among user middleware is call order.
func (s *Server) UseMiddleware(m ...Middleware) error {
	if s == nil {
		return errors.New("mcp: server is required")
	}
	for _, link := range m {
		if link == nil {
			return errors.New("mcp: middleware must not be nil")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userChain = append(s.userChain, m...)
	// Re-install the chain on the SDK so the new links take effect for
	// future connections.
	s.sdk.AddReceivingMiddleware(s.rebuildReceiving()...)
	return nil
}

// rebuildReceiving wraps the owned chain into SDK middleware form.
// The SDK chain receives only what sits inside the mandatory gate.
func (s *Server) rebuildReceiving() []sdk.Middleware {
	sdkLinks := make([]sdk.Middleware, 0, len(s.userChain))
	for _, link := range s.userChain {
		sdkLinks = append(sdkLinks, sdk.Middleware(link))
	}
	return sdkLinks
}

// Observation installs the observation layer, which surrounds every
// user-visible operation. Installing it twice returns an error rather
// than silently double-wrapping.
func (s *Server) Observation(m Middleware) error {
	if s == nil {
		return errors.New("mcp: server is required")
	}
	if m == nil {
		return errors.New("mcp: observation middleware must not be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.observeChain != nil {
		return errors.New("mcp: observation already installed")
	}
	s.observeChain = m
	return nil
}

// Authorize installs the application authorization layer, which
// precedes side effects. Replacing an installed policy is an error:
// authorization changes require a fresh server.
func (s *Server) Authorize(m Middleware) error {
	if s == nil {
		return errors.New("mcp: server is required")
	}
	if m == nil {
		return errors.New("mcp: authorization middleware must not be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.authChain != nil {
		return errors.New("mcp: authorization already installed")
	}
	s.authChain = m
	return nil
}

// Admission installs the admission layer, which precedes handler
// execution and controls concurrency. See the limits table for the
// default bounds (stage 8 wires the counts).
func (s *Server) Admission(m Middleware) error {
	if s == nil {
		return errors.New("mcp: server is required")
	}
	if m == nil {
		return errors.New("mcp: admission middleware must not be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.admitChain != nil {
		return errors.New("mcp: admission already installed")
	}
	s.admitChain = m
	return nil
}

// chainSnapshot renders the current owned chain for serving; the
// mandatory gate always wraps the result.
func (s *Server) chainSnapshot() serverChain {
	s.mu.Lock()
	defer s.mu.Unlock()
	return serverChain{
		observe:  s.observeChain,
		user:     append([]Middleware(nil), s.userChain...),
		auth:     s.authChain,
		admit:    s.admitChain,
		validate: s.outputValidation(),
	}
}
