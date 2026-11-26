package mcp

import (
	"context"
	legacyjson "encoding/json"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"testing"
	"time"

	p "github.com/maccavelli/go-mcp-lib/protocol"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestOwnedMiddlewareLive(t *testing.T) {
	server := newTestServer(t)
	if err := AddTool(server, &ToolRegistration{Name: "echo"}, func(_ context.Context, request *ToolRequest, input echoInput) (*p.CallToolResult, echoOutput, error) {
		if request.Params.Name != "echo" {
			t.Error("FAIL owned middleware change did not reach handler")
		}
		return nil, echoOutput{Reply: input.Message}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := server.UseMiddleware(func(next MethodHandler) MethodHandler {
		return func(ctx context.Context, method string, request Request) (any, error) {
			if method == "tools/call" {
				call, ok := request.(*ToolRequest)
				if !ok || call.Params.Name != "alias" || call.Session() == nil {
					t.Error("FAIL owned middleware request or connection")
				} else {
					call.Params.Name = "echo"
				}
			}
			result, err := next(ctx, method, request)
			if err == nil && method == "tools/call" {
				if _, ok := result.(*p.CallToolResult); !ok {
					t.Error("FAIL middleware result is not the owned DTO")
				}
			}
			return result, err
		}
	}); err != nil {
		t.Fatal(err)
	}
	session, shutdown := startPair(t, server)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := session.CallTool(ctx, "alias", map[string]any{"message": "owned middleware"})
	if err != nil || len(result.Content) != 1 {
		t.Fatalf("FAIL owned middleware dispatch: %+v %v", result, err)
	}
	text, err := result.Content[0].AsTextContent()
	if err != nil || text.Text != `{"reply":"owned middleware"}` {
		t.Errorf("FAIL owned middleware round trip: %+v %v", text, err)
	}
}

func TestOwnedMiddlewareBoundary(t *testing.T) {
	for _, name := range []string{"forward-copy", "parameter-type", "wrong-result-type", "short-circuit", "nil-handler"} {
		t.Run(name, func(t *testing.T) {
			source := &sdk.CallToolRequest{
				Params: &sdk.CallToolParamsRaw{Name: "echo", Meta: sdk.Meta{
					"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
					"io.modelcontextprotocol/clientCapabilities": map[string]any{},
				}},
				Extra: &sdk.RequestExtra{Header: http.Header{"X-Fixture": {"original"}}},
			}
			content, err := p.ContentBlockFromTextContent(p.TextContent{Type: "text", Text: "short"})
			if err != nil {
				t.Fatal(err)
			}
			complete := &p.CallToolResult{ResultType: "complete", Content: []p.ContentBlock{content}}
			calls := 0
			final := func(_ context.Context, _ string, request sdk.Request) (sdk.Result, error) {
				calls++
				if name == "forward-copy" && request.GetParams().(*sdk.CallToolParamsRaw).Name != "changed" {
					t.Error("FAIL forwarded parameters did not reach SDK dispatch")
				}
				var result sdk.CallToolResult
				if err := legacyjson.Unmarshal([]byte(`{"resultType":"complete","content":[{"type":"text","text":"sdk"}]}`), &result); err != nil {
					t.Fatal(err)
				}
				return &result, nil
			}
			link := func(next MethodHandler) MethodHandler {
				if name == "nil-handler" {
					return nil
				}
				return func(ctx context.Context, method string, request Request) (any, error) {
					switch name {
					case "forward-copy":
						request.(*ToolRequest).Params.Name = "changed"
						headers := request.Headers()
						headers.Set("X-Fixture", "changed")
						if request.Headers().Get("X-Fixture") != "original" {
							t.Error("FAIL middleware headers are not a snapshot")
						}
					case "parameter-type":
						var params p.ReadResourceRequestParams
						if err := json.Unmarshal([]byte(`{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}},"uri":"file:///fixture"}`), &params); err != nil {
							t.Fatal(err)
						}
						return next(ctx, method, &ResourceRequest{Params: &params})
					case "wrong-result-type":
						return &p.ListToolsResult{ResultType: "complete", CacheScope: "private", TTLMs: "0"}, nil
					case "short-circuit":
						return complete, nil
					}
					return next(ctx, method, request)
				}
			}
			result, err := receivingMiddleware(link)(final)(context.Background(), "tools/call", source)
			switch name {
			case "parameter-type", "wrong-result-type", "nil-handler":
				if !errors.Is(err, ErrProtocol) || calls != 0 {
					t.Errorf("FAIL middleware %s guard: calls=%d result=%+v error=%v", name, calls, result, err)
				}
			case "forward-copy":
				if err != nil || calls != 1 || source.Params.Name != "echo" || source.Extra.Header.Get("X-Fixture") != "original" {
					t.Errorf("FAIL middleware copy boundary: calls=%d result=%+v error=%v", calls, result, err)
				}
			case "short-circuit":
				if err != nil || calls != 0 || result.(*sdk.CallToolResult).Content[0].(*sdk.TextContent).Text != "short" {
					t.Errorf("FAIL middleware short circuit: calls=%d result=%+v error=%v", calls, result, err)
				}
			}
		})
	}
}

func TestSDKResultProjection(t *testing.T) {
	source := &sdk.DiscoverResult{
		SupportedVersions: []string{"2026-07-28"},
		Capabilities:      &sdk.ServerCapabilities{},
		Cacheable:         sdk.Cacheable{CacheScope: "private", TTLMs: 0},
	}
	value, err := middlewareResult("server/discover", source)
	if err != nil {
		t.Fatalf("FAIL SDK complete-result projection: %v", err)
	}
	if value.(*p.DiscoverResult).ResultType != "complete" || source.ResultType != "" {
		t.Error("FAIL SDK complete-result projection changed the original or lost the marker")
	}
	if _, err := middlewareResult("server/discover", (*sdk.DiscoverResult)(nil)); !errors.Is(err, ErrProtocol) {
		t.Errorf("FAIL nil SDK projection accepted: %v", err)
	}
}
