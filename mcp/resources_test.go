package mcp

import (
	"context"
	"strings"
	"testing"

	mcplib "github.com/maccavelli/go-mcp-lib/protocol"
)

func TestAddPromptValidation(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *PromptRequest) (*mcplib.GetPromptResult, error) {
		return nil, nil
	}
	if err := AddPrompt(server, &PromptRegistration{}, handler); err == nil {
		t.Error("FAIL empty prompt name accepted")
	}
	if err := AddPrompt(server, &PromptRegistration{Name: "p"}, nil); err == nil {
		t.Error("FAIL nil prompt handler accepted")
	}
	dupArgs := &PromptRegistration{
		Name:      "p",
		Arguments: []*mcplib.PromptArgument{{Name: "a"}, {Name: "a"}},
	}
	if err := AddPrompt(server, dupArgs, handler); err == nil ||
		!strings.Contains(err.Error(), "duplicate argument") {
		t.Errorf("FAIL duplicate argument accepted: %v", err)
	}
	ok := &PromptRegistration{Name: "p", Arguments: []*mcplib.PromptArgument{{Name: "a"}}}
	if err := AddPrompt(server, ok, handler); err != nil {
		t.Fatalf("FAIL valid prompt rejected: %v", err)
	}
	if err := AddPrompt(server, &PromptRegistration{Name: "p"}, handler); err == nil {
		t.Error("FAIL duplicate prompt name accepted")
	}
}

func TestAddResourceValidation(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *ResourceRequest) (*mcplib.ReadResourceResult, error) {
		return nil, nil
	}
	cases := []struct {
		name string
		reg  *ResourceRegistration
		want string
	}{
		{"empty name", &ResourceRegistration{URI: "file:///x"}, "resource name is required"},
		{"missing uri", &ResourceRegistration{Name: "r"}, "resource URI is required"},
		{"relative uri", &ResourceRegistration{Name: "r", URI: "/plain/path"}, "not an absolute URI"},
	}
	for _, tc := range cases {
		if err := AddResource(server, tc.reg, handler); err == nil ||
			!strings.Contains(err.Error(), tc.want) {
			t.Errorf("FAIL %s: got %v, want %q", tc.name, err, tc.want)
		}
	}
	ok := &ResourceRegistration{Name: "r", URI: "file:///data"}
	if err := AddResource(server, ok, handler); err != nil {
		t.Fatalf("FAIL valid resource rejected: %v", err)
	}
	dup := &ResourceRegistration{Name: "r2", URI: "file:///data"}
	if err := AddResource(server, dup, handler); err == nil ||
		!strings.Contains(err.Error(), "already registered") {
		t.Errorf("FAIL duplicate resource URI accepted: %v", err)
	}
}

func TestAddResourceTemplateValidation(t *testing.T) {
	server := newTestServer(t)
	handler := func(ctx context.Context, req *ResourceRequest) (*mcplib.ReadResourceResult, error) {
		return nil, nil
	}
	cases := []struct {
		name string
		reg  *ResourceTemplateRegistration
		want string
	}{
		{"empty name", &ResourceTemplateRegistration{URITemplate: "file:///x/{a}"}, "template name is required"},
		{"empty template", &ResourceTemplateRegistration{Name: "t"}, "uriTemplate is empty"},
		{"invalid modifier", &ResourceTemplateRegistration{Name: "t", URITemplate: "file:///x/{a:no}"}, "invalid RFC 6570"},
		{"unbalanced", &ResourceTemplateRegistration{Name: "t", URITemplate: "file:///x/{a"}, "invalid RFC 6570"},
		{"invalid varspec", &ResourceTemplateRegistration{Name: "t", URITemplate: "logs/{?}"}, "invalid RFC 6570"},
	}
	for _, tc := range cases {
		if err := AddResourceTemplate(server, tc.reg, handler); err == nil ||
			!strings.Contains(err.Error(), tc.want) {
			t.Errorf("FAIL %s: got %v, want %q", tc.name, err, tc.want)
		}
	}
	ok := &ResourceTemplateRegistration{Name: "t", URITemplate: "file:///logs/{name}"}
	if err := AddResourceTemplate(server, ok, handler); err != nil {
		t.Fatalf("FAIL valid template rejected: %v", err)
	}
	dup := &ResourceTemplateRegistration{Name: "t2", URITemplate: "file:///logs/{name}"}
	if err := AddResourceTemplate(server, dup, handler); err == nil ||
		!strings.Contains(err.Error(), "already registered") {
		t.Errorf("FAIL duplicate template accepted: %v", err)
	}
	for _, template := range []string{"file:///x", "logs/{a}", "https://example.invalid/{+path}{?q,limit}"} {
		if err := AddResourceTemplate(server, &ResourceTemplateRegistration{Name: "standard", URITemplate: template}, handler); err != nil {
			t.Errorf("FAIL RFC 6570 declaration %q: %v", template, err)
		}
	}
}
