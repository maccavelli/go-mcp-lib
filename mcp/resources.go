package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/go-mcp-lib/internal/validation"
	"github.com/maccavelli/go-mcp-lib/mcp/internal/sdkbridge"
	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

// PromptRegistration is the declarative half of a prompt.
type PromptRegistration struct {
	// Name is required, non-empty.
	Name string
	// Title is the optional display name.
	Title string
	// Description is optional.
	Description string
	// Arguments are the prompt's declared arguments.
	Arguments []*mcplib.PromptArgument
}

func (r *PromptRegistration) validate() error {
	if r == nil || r.Name == "" {
		return errors.New("mcp: prompt name is required")
	}
	seen := make(map[string]bool, len(r.Arguments))
	for _, arg := range r.Arguments {
		if arg == nil {
			return errors.New("mcp: prompt argument must not be nil")
		}
		if arg.Name == "" {
			return errors.New("mcp: prompt argument name is required")
		}
		if seen[arg.Name] {
			return fmt.Errorf("mcp: prompt %q: duplicate argument %q", r.Name, arg.Name)
		}
		seen[arg.Name] = true
	}
	return nil
}

// AddPrompt registers a prompt handler. Duplicate names on the same
// server are rejected; registration is instance-scoped.
func AddPrompt(s *Server, reg *PromptRegistration, handler PromptHandler) error {
	if s == nil {
		return errors.New("mcp: server is required")
	}
	if err := reg.validate(); err != nil {
		return err
	}
	if handler == nil {
		return fmt.Errorf("mcp: prompt %q: handler is required", reg.Name)
	}
	s.mu.Lock()
	if _, exists := s.registered[reg.Name]; exists {
		s.mu.Unlock()
		return fmt.Errorf("mcp: prompt %q: already registered on this server", reg.Name)
	}
	s.registered[reg.Name] = registration{kind: "prompt", name: reg.Name}
	s.mu.Unlock()

	prompt := &sdk.Prompt{
		Name:        reg.Name,
		Description: reg.Description,
		Arguments:   reg.Arguments,
	}
	s.sdk.AddPrompt(prompt, func(ctx context.Context, request *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
		params, err := sdkbridge.FromSDK[mcplib.GetPromptRequestParams](request.Params)
		if err != nil {
			return nil, typed(ErrProtocol, "prompt request", err)
		}
		owned := &PromptRequest{Params: params, session: &ServerSession{sdk: request.Session}, protocolVersion: request.ProtocolVersion(), headers: copiedRequestHeaders(request)}
		result, err := handler(ctx, owned)
		if err != nil {
			return nil, err
		}
		return sdkbridge.ToSDK[sdk.GetPromptResult](result)
	})
	return nil
}

// ResourceRegistration is the declarative half of a static resource.
type ResourceRegistration struct {
	// Name is required, non-empty.
	Name string
	// Title is the optional display name.
	Title string
	// URI is required, absolute (location rule).
	URI string
	// Description is optional.
	Description string
	// MIMEType is optional.
	MIMEType string
}

func (r *ResourceRegistration) validate() error {
	if r == nil || r.Name == "" {
		return errors.New("mcp: resource name is required")
	}
	if r.URI == "" {
		return errors.New("mcp: resource URI is required")
	}
	if err := validateAbsoluteURI(r.URI); err != nil {
		return err
	}
	return nil
}

// AddResource registers a static resource handler. Duplicate URIs on
// the same server are rejected; registration is instance-scoped.
func AddResource(s *Server, reg *ResourceRegistration, handler ResourceHandler) error {
	if s == nil {
		return errors.New("mcp: server is required")
	}
	if err := reg.validate(); err != nil {
		return err
	}
	if handler == nil {
		return fmt.Errorf("mcp: resource %q: handler is required", reg.Name)
	}
	s.mu.Lock()
	key := "resource:" + reg.URI
	if _, exists := s.registered[key]; exists {
		s.mu.Unlock()
		return fmt.Errorf("mcp: resource URI %q: already registered on this server", reg.URI)
	}
	s.registered[key] = registration{kind: "resource", name: reg.URI}
	s.mu.Unlock()

	resource := &sdk.Resource{
		Name:        reg.Name,
		Description: reg.Description,
		MIMEType:    reg.MIMEType,
		URI:         reg.URI,
	}
	s.sdk.AddResource(resource, resourceHandlerBridge(handler))
	return nil
}

// ResourceTemplateRegistration is the declarative half of a resource
// template. The URI template is validated by the pinned RFC 6570 parser.
type ResourceTemplateRegistration struct {
	// Name is required, non-empty.
	Name string
	// URITemplate is required and validated.
	URITemplate string
	// Description is optional.
	Description string
	// MIMEType is optional.
	MIMEType string
}

func (r *ResourceTemplateRegistration) validate() error {
	if r == nil || r.Name == "" {
		return errors.New("mcp: resource template name is required")
	}
	if err := validateURITemplateShape(r.URITemplate); err != nil {
		return err
	}
	return nil
}

// AddResourceTemplate registers a resource template handler. Duplicate
// templates on the same server are rejected.
func AddResourceTemplate(s *Server, reg *ResourceTemplateRegistration, handler ResourceHandler) error {
	if s == nil {
		return errors.New("mcp: server is required")
	}
	if err := reg.validate(); err != nil {
		return err
	}
	if handler == nil {
		return fmt.Errorf("mcp: resource template %q: handler is required", reg.Name)
	}
	s.mu.Lock()
	key := "resource-template:" + reg.URITemplate
	if _, exists := s.registered[key]; exists {
		s.mu.Unlock()
		return fmt.Errorf("mcp: resource template %q: already registered on this server", reg.URITemplate)
	}
	s.registered[key] = registration{kind: "resource-template", name: reg.URITemplate}
	s.mu.Unlock()

	template := &sdk.ResourceTemplate{
		Name:        reg.Name,
		Description: reg.Description,
		MIMEType:    reg.MIMEType,
		URITemplate: reg.URITemplate,
	}
	s.sdk.AddResourceTemplate(template, resourceHandlerBridge(handler))
	return nil
}

func resourceHandlerBridge(handler ResourceHandler) sdk.ResourceHandler {
	return func(ctx context.Context, request *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
		params, err := sdkbridge.FromSDK[mcplib.ReadResourceRequestParams](request.Params)
		if err != nil {
			return nil, typed(ErrProtocol, "resource request", err)
		}
		owned := &ResourceRequest{Params: params, session: &ServerSession{sdk: request.Session}, protocolVersion: request.ProtocolVersion(), headers: copiedRequestHeaders(request)}
		result, err := handler(ctx, owned)
		if err != nil {
			return nil, err
		}
		return sdkbridge.ToSDK[sdk.ReadResourceResult](result)
	}
}

// validateAbsoluteURI uses the canonical production location checker.
func validateAbsoluteURI(uri string) error {
	return validation.CheckURI(uri)
}

// validateURITemplateShape checks declaration presence and RFC 6570 syntax.
func validateURITemplateShape(template string) error {
	if template == "" {
		return errors.New("mcp: uriTemplate is empty")
	}
	return validation.CheckURITemplate(template)
}
