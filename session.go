package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerSession is an owned server-side connection handle. The SDK
// session stays inside; the public surface is context-first and never
// exposes a mutable global routing authority.
type ServerSession struct {
	sdk *sdk.ServerSession
}

// ID returns the connection's stable session identifier.
func (s *ServerSession) ID() string {
	if s == nil || s.sdk == nil {
		return ""
	}
	return s.sdk.ID()
}

// Ping checks liveness with the caller's deadline.
func (s *ServerSession) Ping(ctx context.Context) error {
	if s == nil || s.sdk == nil {
		return errors.New("mcp: session is closed")
	}
	if err := s.sdk.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mcp: ping: %w", err)
	}
	return nil
}

// Close tears the connection down. It is safe to call more than once.
func (s *ServerSession) Close() error {
	if s == nil || s.sdk == nil {
		return nil
	}
	if err := s.sdk.Close(); err != nil {
		return fmt.Errorf("mcp: close: %w", err)
	}
	return nil
}
