// Package mcp is the owned runtime API of go-mcp-lib: server, client,
// and connection handles that preserve SDK execution behind a
// validating bridge.
//
// The owned surface never exposes deprecated SDK symbols. Handles
// wrap the pinned upstream SDK; options are validated once at
// construction and immutable thereafter; every blocking operation
// takes a context first.
package mcp

import (
	"errors"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

// ServerOptions are validated once at construction and immutable
// thereafter. Zero fields mean SDK defaults; the framework never
// silently re-fills them.
type ServerOptions struct {
	// Implementation identifies the server on the wire. Required.
	Implementation *mcplib.Implementation

	// Instructions are optional server instructions for clients.
	Instructions string

	// PageSize caps single-page list results; zero means the SDK
	// default.
	PageSize int

	// KeepAlive, when non-zero, enables the ping interval.
	KeepAlive time.Duration

	// KeepAliveFailureThreshold: consecutive ping failures tolerated
	// before close (values < 2 behave as 1).
	KeepAliveFailureThreshold int
}

// validate rejects impossible configurations at construction, before
// any session exists.
func (o *ServerOptions) validate() error {
	if o == nil || o.Implementation == nil {
		return errors.New("mcp: ServerOptions.Implementation is required")
	}
	if o.PageSize < 0 {
		return errors.New("mcp: ServerOptions.PageSize must not be negative")
	}
	if o.KeepAlive < 0 {
		return errors.New("mcp: ServerOptions.KeepAlive must not be negative")
	}
	if o.KeepAlive == 0 && o.KeepAliveFailureThreshold != 0 {
		return errors.New(
			"mcp: ServerOptions.KeepAliveFailureThreshold has no effect without KeepAlive",
		)
	}
	return nil
}

// Server is an owned MCP server handle. The SDK server it wraps is
// private; all interaction goes through the owned API.
type Server struct {
	sdk *sdk.Server

	opts ServerOptions

	// registered guards duplicate registration checks; admission is
	// instance-scoped (G13).
	registered map[string]registration

	// owned middleware chain state; see middleware.go for ordering.
	userChain    []Middleware
	observeChain Middleware
	authChain    Middleware
	admitChain   Middleware

	// output owns declared-output validation before emission (S6.4).
	output *outputValidator

	mu sync.Mutex
}

type registration struct {
	kind string // "tool", "prompt", "resource", "resource-template"
	name string
}

// NewServer validates options and constructs the owned handle. It
// returns a construction error, never a half-built server.
func NewServer(opts *ServerOptions) (*Server, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	impl := &sdk.Implementation{
		Name:    opts.Implementation.Name,
		Title:   opts.Implementation.Title,
		Version: opts.Implementation.Version,
	}
	sdkOpts := &sdk.ServerOptions{
		Instructions:              opts.Instructions,
		PageSize:                  opts.PageSize,
		KeepAlive:                 opts.KeepAlive,
		KeepAliveFailureThreshold: opts.KeepAliveFailureThreshold,
	}
	return &Server{
		sdk:        sdk.NewServer(impl, sdkOpts),
		opts:       *opts,
		registered: make(map[string]registration),
		output:     newOutputValidator(),
	}, nil
}

// Options returns the validated immutable options.
func (s *Server) Options() ServerOptions {
	return s.opts
}
