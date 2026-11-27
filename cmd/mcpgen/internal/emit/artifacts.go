package emit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/model"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

// APIDeclaration describes a public declaration without copying implementation
// bodies or private fields. Imports resolve the qualifiers in its signature.
type APIDeclaration struct {
	Name       string                    `json:"name"`
	Kind       string                    `json:"kind"`
	File       string                    `json:"file"`
	Pointer    string                    `json:"schema-pointer,omitempty"`
	Signature  string                    `json:"signature"`
	Imports    map[string]string         `json:"imports"`
	SDKTargets map[string]sdkmap.TypeAPI `json:"sdk-targets,omitempty"`
}

// API is the generated protocol package's declaration contract. Handwritten
// runtime reachability and live behavior remain separately qualified.
type API struct {
	Version      int              `json:"version"`
	Revision     string           `json:"revision"`
	Package      string           `json:"package"`
	SchemaSHA256 string           `json:"schema-sha256"`
	SDKModule    string           `json:"sdk-module"`
	SDKVersion   string           `json:"sdk-version"`
	SDKSum       string           `json:"sdk-module-sum"`
	SDKContext   string           `json:"sdk-context-sha256"`
	Declarations []APIDeclaration `json:"declarations"`
}

type nodeCoverage struct {
	Pointer  string            `json:"schema-pointer"`
	Keywords map[string]string `json:"keywords"`
}

type capabilityCoverage struct {
	Scope   string `json:"scope"`
	Name    string `json:"name"`
	Pointer string `json:"schema-pointer"`
}

// Coverage records source bindings, never a claim of runtime conformance.
type Coverage struct {
	Version       int                  `json:"version"`
	Revision      string               `json:"revision"`
	SchemaSHA256  string               `json:"schema-sha256"`
	Definitions   []Binding            `json:"definitions"`
	Nodes         []nodeCoverage       `json:"nodes"`
	Methods       []profile.Method     `json:"methods"`
	Capabilities  []capabilityCoverage `json:"capabilities"`
	Qualification map[string]string    `json:"qualification"`
}

func (c *compiler) contractArtifacts(files []File) ([]File, error) {
	sum := sha256.Sum256(c.opts.Model.Document)
	identity := hex.EncodeToString(sum[:])
	api := API{Version: 1, Revision: c.opts.Revision, Package: c.opts.PackagePath,
		SchemaSHA256: identity, SDKModule: c.opts.Dispositions.SDKModule,
		SDKVersion: c.opts.Dispositions.SDKVersion, SDKSum: c.opts.Dispositions.SDKSum,
		SDKContext: c.opts.Dispositions.SourceSHA256, Declarations: []APIDeclaration{}}
	for _, file := range files {
		if !strings.HasSuffix(file.Name, ".go") {
			continue
		}
		declarations, err := c.apiDeclarations(file)
		if err != nil {
			return nil, err
		}
		api.Declarations = append(api.Declarations, declarations...)
	}
	sort.Slice(api.Declarations, func(i, j int) bool { return api.Declarations[i].Name < api.Declarations[j].Name })
	byName := make(map[string]*APIDeclaration, len(api.Declarations))
	for i := range api.Declarations {
		declaration := &api.Declarations[i]
		if byName[declaration.Name] != nil {
			return nil, fmt.Errorf("emit: duplicate API declaration %s", declaration.Name)
		}
		byName[declaration.Name] = declaration
	}
	for _, binding := range c.bindings {
		declaration := byName[binding.GoName]
		if declaration == nil || declaration.Kind != "type" || declaration.Pointer != binding.Pointer {
			return nil, fmt.Errorf("emit: missing API binding %s", binding.GoName)
		}
		if binding.Disposition == "alias" {
			surfaces := c.opts.Dispositions.TypeAPI[binding.Definition]
			if len(surfaces) != 3 {
				return nil, fmt.Errorf("emit: incomplete compiler API for alias %s", binding.GoName)
			}
			for _, target := range []string{"darwin/arm64", "linux/amd64", "windows/amd64"} {
				if surface := surfaces[target]; surface.Identity == "" || surface.Underlying == "" {
					return nil, fmt.Errorf("emit: missing alias API target %s: %s", binding.GoName, target)
				}
			}
			declaration.SDKTargets = surfaces
		}
	}
	coverage, err := c.coverage(identity)
	if err != nil {
		return nil, err
	}
	output := []File{}
	for _, item := range []struct {
		name  string
		value any
	}{{"api.json", api}, {"coverage.json", coverage}} {
		body, err := json.MarshalIndent(item.value, "", "  ")
		if err != nil {
			return nil, err
		}
		output = append(output, File{item.name, append(body, '\n')})
	}
	return output, nil
}

func (c *compiler) apiDeclarations(file File) ([]APIDeclaration, error) {
	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, file.Name, file.Content, 0)
	if err != nil {
		return nil, err
	}
	imports := map[string]string{}
	for _, spec := range tree.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = path
	}
	output := []APIDeclaration{}
	appendDeclaration := func(name, kind, receiver string, declaration ast.Node) error {
		var body bytes.Buffer
		if err := format.Node(&body, fset, declaration); err != nil {
			return err
		}
		pointer := c.claimed[strings.ToLower(name)]
		if receiver != "" {
			pointer = c.claimed[strings.ToLower(receiver)]
		}
		if !strings.HasPrefix(pointer, "#") {
			pointer = ""
		}
		output = append(output, APIDeclaration{Name: name, Kind: kind, File: file.Name,
			Pointer: pointer, Signature: body.String(), Imports: imports})
		return nil
	}
	for _, declaration := range tree.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if !declaration.Name.IsExported() {
				continue
			}
			name, receiver := declaration.Name.Name, ""
			if declaration.Recv != nil {
				receiver = receiverName(declaration.Recv.List[0].Type)
				if !ast.IsExported(receiver) {
					continue
				}
				name = receiver + "." + name
			}
			declaration.Body = nil
			if err := appendDeclaration(name, "func", receiver, declaration); err != nil {
				return nil, err
			}
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if !spec.Name.IsExported() {
						continue
					}
					if structure, ok := spec.Type.(*ast.StructType); ok {
						fields := []*ast.Field{}
						for _, field := range structure.Fields.List {
							if len(field.Names) == 0 && ast.IsExported(receiverName(field.Type)) || len(field.Names) > 0 && field.Names[0].IsExported() {
								fields = append(fields, field)
							}
						}
						structure.Fields.List = fields
					}
					if err := appendDeclaration(spec.Name.Name, "type", "", &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{spec}}); err != nil {
						return nil, err
					}
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						if name.IsExported() {
							if err := appendDeclaration(name.Name, declaration.Tok.String(), "", &ast.GenDecl{Tok: declaration.Tok, Specs: []ast.Spec{spec}}); err != nil {
								return nil, err
							}
						}
					}
				}
			}
		}
	}
	return output, nil
}

func receiverName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.StarExpr:
		return receiverName(expression.X)
	case *ast.IndexExpr:
		return receiverName(expression.X)
	case *ast.IndexListExpr:
		return receiverName(expression.X)
	case *ast.SelectorExpr:
		return expression.Sel.Name
	}
	return ""
}

func (c *compiler) coverage(identity string) (*Coverage, error) {
	result := &Coverage{Version: 1, Revision: c.opts.Revision, SchemaSHA256: identity,
		Definitions: c.bindings, Nodes: []nodeCoverage{}, Methods: append([]profile.Method{}, c.opts.Methods...),
		Capabilities: []capabilityCoverage{}, Qualification: map[string]string{
			"scope": "generated source bindings only", "codec-runtime": "pending R4",
			"canonical-runtime": "pending R5: TestRepairA19CanonicalRuntime"}}
	nodes := map[string]*model.Node{}
	var visit func(*model.Node)
	visit = func(node *model.Node) {
		if node == nil || nodes[node.Pointer] != nil {
			return
		}
		nodes[node.Pointer] = node
		for _, child := range node.Properties {
			visit(child)
		}
		visit(node.Items)
		visit(node.Additional)
		for _, children := range node.Compositions {
			for _, child := range children {
				visit(child)
			}
		}
	}
	visit(c.opts.Model.Root)
	for _, definition := range c.opts.Model.Definitions {
		visit(definition)
	}
	for _, pointer := range sortedKeys(nodes) {
		keywords := map[string]string{}
		for name, class := range nodes[pointer].Keywords {
			switch class {
			case model.ClassAssertion:
				keywords[name] = "assertion"
			case model.ClassAnnotation:
				keywords[name] = "annotation"
			default:
				return nil, fmt.Errorf("emit: unsupported coverage keyword %s at %s", name, pointer)
			}
		}
		result.Nodes = append(result.Nodes, nodeCoverage{pointer, keywords})
	}
	expectedMethods := map[string]map[string]bool{}
	for _, definition := range c.opts.Model.Definitions {
		if field := definition.Properties["method"]; field != nil {
			if name, ok := field.Const.(string); ok {
				if expectedMethods[name] == nil {
					expectedMethods[name] = map[string]bool{}
				}
				expectedMethods[name][definition.Pointer] = true
			}
		}
	}
	if len(expectedMethods) != len(result.Methods) {
		return nil, fmt.Errorf("emit: method coverage inventory differs")
	}
	seen := map[string]bool{}
	for _, method := range result.Methods {
		if !expectedMethods[method.Name][method.Pointer] || seen[method.Name] || nodes[method.Pointer] == nil {
			return nil, fmt.Errorf("emit: missing or duplicate method coverage %s", method.Name)
		}
		seen[method.Name] = true
	}
	sort.Slice(result.Methods, func(i, j int) bool { return result.Methods[i].Name < result.Methods[j].Name })
	for _, scope := range []string{"client", "server"} {
		name := "ClientCapabilities"
		if scope == "server" {
			name = "ServerCapabilities"
		}
		var properties func(*model.Node, string)
		properties = func(node *model.Node, prefix string) {
			if node == nil {
				return
			}
			for _, name := range sortedKeys(node.Properties) {
				child := node.Properties[name]
				key := prefix + name
				result.Capabilities = append(result.Capabilities, capabilityCoverage{scope, key, child.Pointer})
				properties(child, key+".")
			}
		}
		properties(c.opts.Model.Definitions[name], "")
	}
	return result, nil
}
