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

// NotifyProgress reports progress on a long-running request to
// the connected client, scoped to this session's connection.
func (s *ServerSession) NotifyProgress(ctx context.Context, progressToken string, progress float64, message string) error {
	if s == nil || s.sdk == nil {
		return errors.New("mcp: session is closed")
	}
	if progressToken == "" {
		return errors.New("mcp: progress token is required")
	}
	params := &sdk.ProgressNotificationParams{
		Progress:      progress,
		ProgressToken: progressToken,
	}
	params.Message = message
	err := s.sdk.NotifyProgress(ctx, params)
	if err != nil {
		return typed(ErrTransport, "notify progress", err)
	}
	return nil
}

// Elicit asks the connected client's user for structured input. The
// request blocks with the caller's deadline and routes through this
// session only.
func (s *ServerSession) Elicit(ctx context.Context, message string, schema any) (*sdk.ElicitResult, error) {
	if s == nil || s.sdk == nil {
		return nil, errors.New("mcp: session is closed")
	}
	if message == "" {
		return nil, errors.New("mcp: elicitation message is required")
	}
	if schema == nil {
		return nil, errors.New("mcp: elicitation schema is required")
	}
	result, err := s.sdk.Elicit(ctx, &sdk.ElicitParams{
		Message:         message,
		RequestedSchema: schema,
	})
	if err != nil {
		return nil, typed(ErrToolCall, "elicit", err)
	}
	return result, nil
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
