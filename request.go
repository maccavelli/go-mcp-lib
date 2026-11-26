package mcp

import (
	"context"
	"net/http"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

// RequestContext carries an owned, independently decoded parameter snapshot and
// a connection-scoped handle. The underlying SDK request is never exposed.
type RequestContext[P any] struct {
	Params *P

	session         *ServerSession
	protocolVersion string
	headers         http.Header
}

// Request is the owned middleware view. Parameters returns a canonical protocol
// DTO; extension payloads use protocol.JSONValue. Concrete handler contexts keep
// their typed Params field. SDK requests cannot implement this sealed interface.
type Request interface {
	Parameters() any
	Session() *ServerSession
	ProtocolVersion() string
	Headers() http.Header
	ownedRequest()
}

func (*RequestContext[P]) ownedRequest() {}

// Parameters returns this request's independently decoded parameter snapshot.
func (r *RequestContext[P]) Parameters() any {
	if r == nil {
		return nil
	}
	return r.Params
}

// Headers returns a copy of the transport headers, when present.
func (r *RequestContext[P]) Headers() http.Header {
	if r == nil {
		return nil
	}
	return r.headers.Clone()
}

// MethodHandler receives owned requests and returns canonical protocol DTOs.
// The private receiving bridge checks parameter/result types before SDK dispatch.
type MethodHandler func(context.Context, string, Request) (any, error)

// Session returns the owned handle for this request's connection.
func (r *RequestContext[P]) Session() *ServerSession {
	if r == nil {
		return nil
	}
	return r.session
}

// ProtocolVersion returns the version associated with this request.
func (r *RequestContext[P]) ProtocolVersion() string {
	if r == nil {
		return ""
	}
	return r.protocolVersion
}

// ToolRequest is the owned context delivered to a typed tool handler.
type ToolRequest = RequestContext[mcplib.CallToolRequestParams]

// PromptRequest is the owned context delivered to a prompt handler.
type PromptRequest = RequestContext[mcplib.GetPromptRequestParams]

// ResourceRequest is the owned context delivered to a resource handler.
type ResourceRequest = RequestContext[mcplib.ReadResourceRequestParams]

// ToolHandler preserves SDK input inference and typed output fulfillment while
// keeping the request, session and explicit result on the owned API.
type ToolHandler[In, Out any] func(context.Context, *ToolRequest, In) (*mcplib.CallToolResult, Out, error)

// PromptHandler returns the owned response to a prompts/get request.
type PromptHandler func(context.Context, *PromptRequest) (*mcplib.GetPromptResult, error)

// ResourceHandler returns the owned response to a resources/read request.
type ResourceHandler func(context.Context, *ResourceRequest) (*mcplib.ReadResourceResult, error)
