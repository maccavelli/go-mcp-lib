package mcp

import (
	"errors"
	"fmt"
)

// Error kinds are sentinel values for errors.Is classification. Each
// carries a bounded, sanitized public message; diagnostics stay with
// the error chain, never the wire.
var (
	// ErrProtocol is a wire-level protocol violation: malformed
	// frames, invalid JSON-RPC shapes, unsupported revisions.
	ErrProtocol = errors.New("mcp: protocol error")

	// ErrRegistration is a rejected registration: invalid schema,
	// duplicate name, invalid handler signature.
	ErrRegistration = errors.New("mcp: registration error")

	// ErrToolCall is a failed tool invocation: schema rejection,
	// handler failure, or cancellation.
	ErrToolCall = errors.New("mcp: tool call error")

	// ErrTransport is a transport-level failure: broken pipe,
	// connection closed, TLS trust.
	ErrTransport = errors.New("mcp: transport error")

	// ErrResourceLimit is a resource limit hit: body size, event
	// size, depth, token bounds.
	ErrResourceLimit = errors.New("mcp: resource limit error")

	// ErrOverload is admission overflow: handler or queue saturation.
	ErrOverload = errors.New("mcp: overload error")

	// ErrCancelled is an operation cancelled by its caller context.
	ErrCancelled = errors.New("mcp: cancelled")
)

// TypedError classifies an error with a kind and bounded public
// message. Wrap with %w so errors.Is reaches both the sentinel and
// the underlying cause.
type TypedError struct {
	Kind  error
	Cause error
	Msg   string
}

func (e *TypedError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%v: %s: %v", e.Kind, e.Msg, e.Cause)
	}
	return fmt.Sprintf("%v: %s", e.Kind, e.Msg)
}

func (e *TypedError) Unwrap() error { return e.Cause }

// Is matches the kind sentinel or the cause chain.
func (e *TypedError) Is(target error) bool {
	return errors.Is(e.Kind, target) || errors.Is(e.Cause, target)
}

// As exposes the cause for errors.As, and itself for TypedError.
func (e *TypedError) As(target any) bool {
	if typed, ok := target.(**TypedError); ok {
		*typed = e
		return true
	}
	return false
}

// classify maps transport and registration call sites to typed
// errors. Callers wrap their specific context message.
func typed(kind error, msg string, cause error) *TypedError {
	return &TypedError{Kind: kind, Msg: msg, Cause: cause}
}
