package mcp_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	mcp "github.com/maccavelli/go-mcp-lib/mcp"
	p "github.com/maccavelli/go-mcp-lib/protocol"
)

// This external consumer imports only the owned packages. Its registry is
// reconciled with actual compiled-source declarations so an unreviewed export
// cannot silently fall outside the transitive signature check.
func TestOwnedPublicAPI(t *testing.T) {
	registry := map[string]reflect.Type{
		"Client": reflect.TypeFor[mcp.Client](), "ClientOptions": reflect.TypeFor[mcp.ClientOptions](),
		"ClientSession": reflect.TypeFor[mcp.ClientSession](), "Server": reflect.TypeFor[mcp.Server](),
		"ServerOptions": reflect.TypeFor[mcp.ServerOptions](), "ServerSession": reflect.TypeFor[mcp.ServerSession](),
		"Transport": reflect.TypeFor[mcp.Transport](), "StdioTransport": reflect.TypeFor[mcp.StdioTransport](),
		"ToolRegistration": reflect.TypeFor[mcp.ToolRegistration](), "PromptRegistration": reflect.TypeFor[mcp.PromptRegistration](),
		"ResourceRegistration": reflect.TypeFor[mcp.ResourceRegistration](), "ResourceTemplateRegistration": reflect.TypeFor[mcp.ResourceTemplateRegistration](),
		"TypedError": reflect.TypeFor[mcp.TypedError](), "PinnedDependencies": reflect.TypeFor[mcp.PinnedDependencies](),
		"Request": reflect.TypeFor[mcp.Request](), "RequestContext": reflect.TypeFor[mcp.RequestContext[p.JSONValue]](),
		"ToolRequest": reflect.TypeFor[mcp.ToolRequest](), "PromptRequest": reflect.TypeFor[mcp.PromptRequest](),
		"ResourceRequest": reflect.TypeFor[mcp.ResourceRequest](), "MethodHandler": reflect.TypeFor[mcp.MethodHandler](),
		"Middleware": reflect.TypeFor[mcp.Middleware](), "ToolHandler": reflect.TypeFor[mcp.ToolHandler[any, any]](),
		"PromptHandler": reflect.TypeFor[mcp.PromptHandler](), "ResourceHandler": reflect.TypeFor[mcp.ResourceHandler](),
		"NewServer": reflect.TypeOf(mcp.NewServer), "NewClient": reflect.TypeOf(mcp.NewClient),
		"Pins": reflect.TypeOf(mcp.Pins), "InMemoryTransport": reflect.TypeOf(mcp.InMemoryTransport),
		"AddTool": reflect.TypeOf(mcp.AddTool[any, any]), "AddPrompt": reflect.TypeOf(mcp.AddPrompt),
		"AddResource": reflect.TypeOf(mcp.AddResource), "AddResourceTemplate": reflect.TypeOf(mcp.AddResourceTemplate),
		"ErrProtocol": reflect.TypeOf(mcp.ErrProtocol), "ErrRegistration": reflect.TypeOf(mcp.ErrRegistration),
		"ErrToolCall": reflect.TypeOf(mcp.ErrToolCall), "ErrTransport": reflect.TypeOf(mcp.ErrTransport),
		"ErrResourceLimit": reflect.TypeOf(mcp.ErrResourceLimit), "ErrOverload": reflect.TypeOf(mcp.ErrOverload),
		"ErrCancelled": reflect.TypeOf(mcp.ErrCancelled),
	}
	actual := publicDeclarations(t)
	for name := range actual {
		if registry[name] == nil {
			t.Errorf("FAIL unreviewed public declaration: %s", name)
		}
	}
	for name := range registry {
		if !actual[name] {
			t.Errorf("FAIL stale public declaration probe: %s", name)
		}
	}
	// The compiler-backed SDK audit separately qualifies these alias closures.
	// Their transitive SDK data types form the allowed SDK subset here.
	aliases := []reflect.Type{
		reflect.TypeFor[p.Annotations](), reflect.TypeFor[p.ElicitResult](), reflect.TypeFor[p.Icon](),
		reflect.TypeFor[p.Implementation](), reflect.TypeFor[p.ProgressNotificationParams](),
		reflect.TypeFor[p.Prompt](), reflect.TypeFor[p.PromptArgument](), reflect.TypeFor[p.Resource](),
		reflect.TypeFor[p.ResourceContents](), reflect.TypeFor[p.ResourceTemplate](),
		reflect.TypeFor[p.ResourceUpdatedNotificationParams](), reflect.TypeFor[p.Role](),
		reflect.TypeFor[p.Tool](), reflect.TypeFor[p.ToolAnnotations](),
	}
	allowed := map[reflect.Type]bool{}
	seen := map[reflect.Type]bool{}
	for _, typ := range aliases {
		walkPublicType(typ, seen, func(value reflect.Type) {
			if strings.HasPrefix(value.PkgPath(), "github.com/modelcontextprotocol/go-sdk/") {
				allowed[value] = true
			}
		})
	}
	for name, typ := range registry {
		walkPublicType(typ, map[reflect.Type]bool{}, func(value reflect.Type) {
			if strings.HasPrefix(value.PkgPath(), "github.com/modelcontextprotocol/go-sdk/") && !allowed[value] {
				t.Errorf("FAIL public SDK reachability: %s reaches %s", name, value)
			}
		})
	}
}

func publicDeclarations(t *testing.T) map[string]bool {
	t.Helper()
	selection, err := build.Default.ImportDir(".", build.IgnoreVendor)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]bool{}
	for _, name := range selection.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Recv == nil && declaration.Name.IsExported() {
					result[declaration.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						if spec.Name.IsExported() {
							result[spec.Name.Name] = true
						}
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							if name.IsExported() {
								result[name.Name] = true
							}
						}
					}
				}
			}
		}
	}
	return result
}

func walkPublicType(typ reflect.Type, seen map[reflect.Type]bool, visit func(reflect.Type)) {
	if typ == nil || seen[typ] {
		return
	}
	seen[typ] = true
	visit(typ)
	path := typ.PkgPath()
	if path != "" && !strings.HasPrefix(path, "github.com/maccavelli/go-mcp-lib") && !strings.HasPrefix(path, "github.com/modelcontextprotocol/go-sdk/") {
		return
	}
	walk := func(value reflect.Type) { walkPublicType(value, seen, visit) }
	for _, receiver := range []reflect.Type{typ, reflect.PointerTo(typ)} {
		for index := 0; index < receiver.NumMethod(); index++ {
			method := receiver.Method(index)
			if method.PkgPath == "" {
				walk(method.Type)
			}
		}
	}
	switch typ.Kind() {
	case reflect.Pointer, reflect.Array, reflect.Slice, reflect.Chan:
		walk(typ.Elem())
	case reflect.Map:
		walk(typ.Key())
		walk(typ.Elem())
	case reflect.Struct:
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			if field.IsExported() {
				walk(field.Type)
			}
		}
	case reflect.Func:
		for index := 0; index < typ.NumIn(); index++ {
			walk(typ.In(index))
		}
		for index := 0; index < typ.NumOut(); index++ {
			walk(typ.Out(index))
		}
	}
}
