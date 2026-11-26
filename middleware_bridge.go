package mcp

import (
	"context"
	legacyjson "encoding/json"
	json "encoding/json/v2"
	"errors"
	"net/http"
	"reflect"
	"sync"

	"github.com/maccavelli/go-mcp-lib/internal/sdkbridge"
	p "github.com/maccavelli/go-mcp-lib/protocol"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func ownedMiddlewareRequest[P any](source sdk.Request) (Request, error) {
	params := source.GetParams()
	value, err := sdkbridge.FromSDK[P](&params)
	if err != nil {
		return nil, typed(ErrProtocol, "middleware parameters", err)
	}
	result := &RequestContext[P]{Params: value}
	if session, ok := source.GetSession().(*sdk.ServerSession); ok {
		result.session = &ServerSession{sdk: session}
	}
	if version, ok := source.(interface{ ProtocolVersion() string }); ok {
		result.protocolVersion = version.ProtocolVersion()
	}
	result.headers = copiedRequestHeaders(source)
	return result, nil
}

func copiedRequestHeaders(source sdk.Request) http.Header {
	if extra := source.GetExtra(); extra != nil {
		return extra.Header.Clone()
	}
	return nil
}

func middlewareRequest(method string, source sdk.Request) (Request, error) {
	if source == nil || reflect.ValueOf(source).IsNil() {
		return nil, typed(ErrProtocol, "nil middleware request", nil)
	}
	switch method {
	case "tools/call":
		return ownedMiddlewareRequest[p.CallToolRequestParams](source)
	case "prompts/get":
		return ownedMiddlewareRequest[p.GetPromptRequestParams](source)
	case "resources/read":
		return ownedMiddlewareRequest[p.ReadResourceRequestParams](source)
	case "tools/list", "prompts/list", "resources/list", "resources/templates/list":
		return ownedMiddlewareRequest[p.PaginatedRequestParams](source)
	case "completion/complete":
		return ownedMiddlewareRequest[p.CompleteRequestParams](source)
	case "server/discover":
		return ownedMiddlewareRequest[p.RequestParams](source)
	case "subscriptions/listen":
		return ownedMiddlewareRequest[p.SubscriptionsListenRequestParams](source)
	case "notifications/cancelled":
		return ownedMiddlewareRequest[p.CancelledNotificationParams](source)
	case "notifications/progress":
		return ownedMiddlewareRequest[p.ProgressNotificationParams](source)
	case "notifications/message":
		return ownedMiddlewareRequest[p.LoggingMessageNotificationParams](source)
	case "notifications/resources/updated":
		return ownedMiddlewareRequest[p.ResourceUpdatedNotificationParams](source)
	case "notifications/subscriptions/acknowledged":
		return ownedMiddlewareRequest[p.SubscriptionsAcknowledgedNotificationParams](source)
	case "notifications/tools/list_changed", "notifications/prompts/list_changed", "notifications/resources/list_changed":
		return ownedMiddlewareRequest[p.NotificationParams](source)
	case "elicitation/create":
		return ownedMiddlewareRequest[p.ElicitRequestParams](source)
	case "sampling/createMessage":
		return ownedMiddlewareRequest[p.CreateMessageRequestParams](source)
	case "roots/list":
		return ownedMiddlewareRequest[p.ListRootsRequestFieldsParams](source)
	default:
		// The SDK's auxiliary methods and extension payloads have no canonical
		// parameter binding in this schema. Keep them opaque and SDK-free.
		return ownedMiddlewareRequest[p.JSONValue](source)
	}
}

func cloneMiddlewareRequest(source sdk.Request, parameters any) (sdk.Request, error) {
	body, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}
	original := reflect.ValueOf(source)
	clone := reflect.New(original.Elem().Type())
	clone.Elem().Set(original.Elem())
	field := clone.Elem().FieldByName("Params")
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.Pointer {
		return nil, errors.New("mcp: unsupported SDK parameter representation")
	}
	value := reflect.New(field.Type().Elem())
	if err := legacyjson.Unmarshal(body, value.Interface()); err != nil {
		return nil, err
	}
	field.Set(value)
	result, ok := clone.Interface().(sdk.Request)
	if !ok {
		return nil, errors.New("mcp: unsupported SDK request representation")
	}
	return result, nil
}

func middlewareResult(method string, source sdk.Result) (any, error) {
	if source == nil {
		return nil, nil
	}
	if value := reflect.ValueOf(source); value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, typed(ErrProtocol, "nil SDK middleware result", nil)
	}
	// The SDK calls setCompleteResultType after receiving middleware returns.
	// Project that documented SDK finalization onto a copy for owned validation;
	// this never fills fields on received wire input or on arbitrary results.
	source = projectSDKCompleteResult(source)
	if input, ok := source.(interface{ NeedsInput() bool }); ok && input.NeedsInput() {
		return sdkbridge.FromSDK[p.InputRequiredResult](&source)
	}
	switch method {
	case "tools/call":
		return sdkbridge.FromSDK[p.CallToolResult](&source)
	case "tools/list":
		return sdkbridge.FromSDK[p.ListToolsResult](&source)
	case "prompts/get":
		return sdkbridge.FromSDK[p.GetPromptResult](&source)
	case "prompts/list":
		return sdkbridge.FromSDK[p.ListPromptsResult](&source)
	case "resources/read":
		return sdkbridge.FromSDK[p.ReadResourceResult](&source)
	case "resources/list":
		return sdkbridge.FromSDK[p.ListResourcesResult](&source)
	case "resources/templates/list":
		return sdkbridge.FromSDK[p.ListResourceTemplatesResult](&source)
	case "completion/complete":
		return sdkbridge.FromSDK[p.CompleteResult](&source)
	case "server/discover":
		return sdkbridge.FromSDK[p.DiscoverResult](&source)
	case "subscriptions/listen":
		return sdkbridge.FromSDK[p.SubscriptionsListenResult](&source)
	default:
		return sdkbridge.FromSDK[p.JSONValue](&source)
	}
}

func projectSDKCompleteResult(source sdk.Result) sdk.Result {
	switch value := source.(type) {
	case *sdk.DiscoverResult:
		copy := *value
		copy.ResultType = "complete"
		return &copy
	case *sdk.CompleteResult:
		copy := *value
		copy.ResultType = "complete"
		return &copy
	case *sdk.ListToolsResult:
		copy := *value
		copy.ResultType = "complete"
		return &copy
	case *sdk.ListPromptsResult:
		copy := *value
		copy.ResultType = "complete"
		return &copy
	case *sdk.ListResourcesResult:
		copy := *value
		copy.ResultType = "complete"
		return &copy
	case *sdk.ListResourceTemplatesResult:
		copy := *value
		copy.ResultType = "complete"
		return &copy
	case *sdk.SubscriptionsListenResult:
		copy := *value
		copy.ResultType = "complete"
		return &copy
	default:
		return source
	}
}

func restoreMiddlewareResult(method string, source any, prototype reflect.Type) (sdk.Result, error) {
	if source == nil {
		return nil, nil
	}
	if value := reflect.ValueOf(source); value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, typed(ErrProtocol, "nil middleware result value", nil)
	}
	// Validate the concrete owned result against the method, including the
	// distinct input-required shape, before constructing a sealed SDK result.
	var expected any
	switch method {
	case "tools/call":
		expected = (*p.CallToolResult)(nil)
		if _, ok := source.(*p.InputRequiredResult); ok {
			expected = (*p.InputRequiredResult)(nil)
		}
		if reflect.TypeOf(source) != reflect.TypeOf(expected) {
			return nil, typed(ErrProtocol, "wrong tools/call result type", nil)
		}
		return sdkbridge.ToSDK[sdk.CallToolResult](&source)
	case "prompts/get":
		expected = (*p.GetPromptResult)(nil)
	case "resources/read":
		expected = (*p.ReadResourceResult)(nil)
	case "tools/list":
		expected = (*p.ListToolsResult)(nil)
	case "prompts/list":
		expected = (*p.ListPromptsResult)(nil)
	case "resources/list":
		expected = (*p.ListResourcesResult)(nil)
	case "resources/templates/list":
		expected = (*p.ListResourceTemplatesResult)(nil)
	case "completion/complete":
		expected = (*p.CompleteResult)(nil)
	case "server/discover":
		expected = (*p.DiscoverResult)(nil)
	case "subscriptions/listen":
		expected = (*p.SubscriptionsListenResult)(nil)
	default:
		expected = (*p.JSONValue)(nil)
	}
	if method == "prompts/get" || method == "resources/read" {
		if _, ok := source.(*p.InputRequiredResult); ok {
			expected = (*p.InputRequiredResult)(nil)
		}
	}
	if reflect.TypeOf(source) != reflect.TypeOf(expected) {
		return nil, typed(ErrProtocol, "wrong middleware result type", nil)
	}
	// Standard short-circuits have a known SDK representation. Opaque extension
	// results can only transform the sealed representation returned by dispatch.
	switch method {
	case "prompts/get":
		return sdkbridge.ToSDK[sdk.GetPromptResult](&source)
	case "resources/read":
		return sdkbridge.ToSDK[sdk.ReadResourceResult](&source)
	case "tools/list":
		return sdkbridge.ToSDK[sdk.ListToolsResult](&source)
	case "prompts/list":
		return sdkbridge.ToSDK[sdk.ListPromptsResult](&source)
	case "resources/list":
		return sdkbridge.ToSDK[sdk.ListResourcesResult](&source)
	case "resources/templates/list":
		return sdkbridge.ToSDK[sdk.ListResourceTemplatesResult](&source)
	case "completion/complete":
		return sdkbridge.ToSDK[sdk.CompleteResult](&source)
	case "server/discover":
		return sdkbridge.ToSDK[sdk.DiscoverResult](&source)
	case "subscriptions/listen":
		return sdkbridge.ToSDK[sdk.SubscriptionsListenResult](&source)
	}
	if prototype == nil || prototype.Kind() != reflect.Pointer {
		return nil, typed(ErrProtocol, "opaque result needs a dispatched SDK representation", nil)
	}
	body, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	value := reflect.New(prototype.Elem())
	if err := legacyjson.Unmarshal(body, value.Interface()); err != nil {
		return nil, err
	}
	result, ok := value.Interface().(sdk.Result)
	if !ok {
		return nil, typed(ErrProtocol, "unsupported SDK result representation", nil)
	}
	return result, nil
}

func receivingMiddleware(link Middleware) sdk.Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, source sdk.Request) (sdk.Result, error) {
			request, err := middlewareRequest(method, source)
			if err != nil {
				return nil, err
			}
			parameterType := reflect.TypeOf(request.Parameters())
			var mu sync.Mutex
			var prototype reflect.Type
			ownedNext := func(ctx context.Context, forwarded string, request Request) (any, error) {
				if forwarded != method || request == nil || reflect.TypeOf(request.Parameters()) != parameterType {
					return nil, typed(ErrProtocol, "middleware changed request method or parameter type", nil)
				}
				translated, err := cloneMiddlewareRequest(source, request.Parameters())
				if err != nil {
					return nil, typed(ErrProtocol, "middleware parameters", err)
				}
				result, err := next(ctx, method, translated)
				if err != nil {
					return nil, err
				}
				mu.Lock()
				prototype = reflect.TypeOf(result)
				mu.Unlock()
				return middlewareResult(method, result)
			}
			wrapped := link(ownedNext)
			if wrapped == nil {
				return nil, typed(ErrProtocol, "middleware returned a nil handler", nil)
			}
			result, err := wrapped(ctx, method, request)
			if err != nil {
				return nil, err
			}
			mu.Lock()
			template := prototype
			mu.Unlock()
			return restoreMiddlewareResult(method, result, template)
		}
	}
}
