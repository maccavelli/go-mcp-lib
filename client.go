package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/go-mcp-lib/internal/sdkbridge"
	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

// ClientOptions are validated once at construction and immutable
// thereafter. Only modern (non-deprecated) protocol features are
// configurable here; the SDK's deprecated sampling entry points are
// deliberately absent from the owned surface.
type ClientOptions struct {
	// Implementation identifies the client on the wire. Required.
	Implementation *mcplib.Implementation
}

func (o *ClientOptions) validate() error {
	if o == nil || o.Implementation == nil {
		return errors.New("mcp: ClientOptions.Implementation is required")
	}
	return nil
}

// Client is an owned MCP client handle wrapping the SDK client.
type Client struct {
	sdk  *sdk.Client
	opts ClientOptions
}

// NewClient validates options and constructs the owned handle.
func NewClient(opts *ClientOptions) (*Client, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	impl := &sdk.Implementation{
		Name:    opts.Implementation.Name,
		Title:   opts.Implementation.Title,
		Version: opts.Implementation.Version,
	}
	return &Client{
		sdk:  sdk.NewClient(impl, &sdk.ClientOptions{}),
		opts: *opts,
	}, nil
}

// Options returns the validated immutable options.
func (c *Client) Options() ClientOptions {
	return c.opts
}

// Connect attaches the client to a transport and performs the
// initialization handshake with the caller's deadline.
func (c *Client) Connect(ctx context.Context, t Transport) (*ClientSession, error) {
	if t == nil {
		return nil, errors.New("mcp: transport is required")
	}
	sess, err := c.sdk.Connect(ctx, t.transport(), nil)
	if err != nil {
		return nil, typed(ErrTransport, "connect", err)
	}
	return &ClientSession{sdk: sess}, nil
}

// ClientSession is an owned client-side connection handle.
type ClientSession struct {
	sdk *sdk.ClientSession
}

// ID returns the connection's stable session identifier.
func (s *ClientSession) ID() string {
	if s == nil || s.sdk == nil {
		return ""
	}
	return s.sdk.ID()
}

// Ping checks liveness with the caller's deadline.
func (s *ClientSession) Ping(ctx context.Context) error {
	if s == nil || s.sdk == nil {
		return errors.New("mcp: session is closed")
	}
	if err := s.sdk.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mcp: ping: %w", err)
	}
	return nil
}

// Close tears the connection down. It is safe to call more than once.
func (s *ClientSession) Close() error {
	if s == nil || s.sdk == nil {
		return nil
	}
	if err := s.sdk.Close(); err != nil {
		return fmt.Errorf("mcp: close: %w", err)
	}
	return nil
}

// ListTools discovers the server's tools with the caller's deadline.
func (s *ClientSession) ListTools(ctx context.Context) (*mcplib.ListToolsResult, error) {
	if s == nil || s.sdk == nil {
		return nil, errors.New("mcp: session is closed")
	}
	result, err := s.sdk.ListTools(ctx, nil)
	if err != nil {
		return nil, typed(ErrToolCall, "list tools", err)
	}
	return convertClientResult[mcplib.ListToolsResult]("list tools", result)
}

// ListResources discovers the server's static resources.
func (s *ClientSession) ListResources(ctx context.Context) (*mcplib.ListResourcesResult, error) {
	if s == nil || s.sdk == nil {
		return nil, errors.New("mcp: session is closed")
	}
	result, err := s.sdk.ListResources(ctx, nil)
	if err != nil {
		return nil, typed(ErrToolCall, "list resources", err)
	}
	return convertClientResult[mcplib.ListResourcesResult]("list resources", result)
}

// ReadResource reads one resource by URI.
func (s *ClientSession) ReadResource(ctx context.Context, uri string) (*mcplib.ReadResourceResult, error) {
	if s == nil || s.sdk == nil {
		return nil, errors.New("mcp: session is closed")
	}
	if uri == "" {
		return nil, errors.New("mcp: resource URI is required")
	}
	result, err := s.sdk.ReadResource(ctx, &sdk.ReadResourceParams{URI: uri})
	if err != nil {
		return nil, typed(ErrToolCall, "read resource", err)
	}
	return convertClientResult[mcplib.ReadResourceResult]("read resource", result)
}

// Subscribe asks the server to notify this session of resource
// updates for a URI.
func (s *ClientSession) Subscribe(ctx context.Context, uri string) error {
	if s == nil || s.sdk == nil {
		return errors.New("mcp: session is closed")
	}
	if uri == "" {
		return errors.New("mcp: resource URI is required")
	}
	err := s.sdk.Subscribe(ctx, &sdk.SubscribeParams{URI: uri})
	if err != nil {
		return typed(ErrToolCall, "subscribe", err)
	}
	return nil
}

// Unsubscribe stops update notifications for a URI.
func (s *ClientSession) Unsubscribe(ctx context.Context, uri string) error {
	if s == nil || s.sdk == nil {
		return errors.New("mcp: session is closed")
	}
	if uri == "" {
		return errors.New("mcp: resource URI is required")
	}
	err := s.sdk.Unsubscribe(ctx, &sdk.UnsubscribeParams{URI: uri})
	if err != nil {
		return typed(ErrToolCall, "unsubscribe", err)
	}
	return nil
}

// CallTool invokes a tool with the caller's deadline.
func (s *ClientSession) CallTool(ctx context.Context, name string, arguments map[string]any) (*mcplib.CallToolResult, error) {
	if s == nil || s.sdk == nil {
		return nil, errors.New("mcp: session is closed")
	}
	if name == "" {
		return nil, errors.New("mcp: tool name is required")
	}
	result, err := s.sdk.CallTool(ctx, &sdk.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, typed(ErrCancelled,
				fmt.Sprintf("tool %q", name), ctx.Err())
		}
		return nil, typed(ErrToolCall, name, err)
	}
	return convertClientResult[mcplib.CallToolResult](name, result)
}

func convertClientResult[Owned, SDK any](operation string, source *SDK) (*Owned, error) {
	result, err := sdkbridge.FromSDK[Owned](source)
	if err != nil {
		return nil, typed(ErrToolCall, operation, err)
	}
	return result, nil
}
