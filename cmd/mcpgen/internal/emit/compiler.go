package emit

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/model"
)

type compiler struct {
	opts                      Options
	aliases, owned, constants *ast.File
	names                     map[string]string
	claimed                   map[string]string
	bindings                  []Binding
	usesJSON                  bool
}

func newCompiler(opts Options) (*compiler, error) {
	pkg := opts.PackagePath[strings.LastIndex(opts.PackagePath, "/")+1:]
	if !token.IsIdentifier(pkg) || token.Lookup(pkg).IsKeyword() {
		return nil, fmt.Errorf("emit: invalid package name")
	}
	c := &compiler{opts: opts, names: make(map[string]string), claimed: make(map[string]string), aliases: newFile(pkg, opts.Revision), owned: newFile(pkg, opts.Revision), constants: newFile(pkg, opts.Revision)}
	// Reserve the complete static support namespace, including unexported names.
	for _, name := range []string{"Field", "Present", "Null", "Number", "NewNumber", "ProtocolRevision", "MethodNames", "SchemaBinding", "SchemaBindings", "validateBinding", "marshalOwned", "unmarshalOwned", "marshalUnion", "unmarshalUnion", "decodeVariant", "protocolSchema", "protocolBindings", "schemaState"} {
		c.claimed[strings.ToLower(name)] = "codec support"
	}
	if len(opts.Dispositions.Dispositions) != len(opts.Model.Definitions) {
		return nil, fmt.Errorf("emit: disposition inventory differs from definitions")
	}
	for _, name := range sortedKeys(opts.Model.Definitions) {
		goName := name
		if opts.Names != nil {
			goName = opts.Names[name]
		}
		if err := c.claim(goName, opts.Model.Definitions[name].Pointer); err != nil {
			return nil, err
		}
		c.names[name] = goName
	}
	return c, nil
}

func (c *compiler) claim(name, pointer string) error {
	first, _ := utf8.DecodeRuneInString(name)
	if !token.IsIdentifier(name) || !unicode.IsUpper(first) {
		return fmt.Errorf("emit: %s: invalid exported identifier %q", pointer, name)
	}
	key := strings.ToLower(name)
	if previous, exists := c.claimed[key]; exists {
		return fmt.Errorf("emit: %s: identifier collision %q with %s", pointer, name, previous)
	}
	c.claimed[key] = pointer
	return nil
}

func sortedKeys[V any](values map[string]V) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *compiler) compile() error {
	hasAlias := false
	for _, name := range sortedKeys(c.opts.Model.Definitions) {
		node := c.opts.Model.Definitions[name]
		goName := c.names[name]
		disposition := c.opts.Dispositions.Dispositions[name]
		c.bindings = append(c.bindings, Binding{name, node.Pointer, goName, disposition})
		comment := goName + " binds " + node.Pointer + "; mapping revision: " + c.opts.Revision + "."
		switch disposition {
		case "alias":
			c.aliases.Decls = append(c.aliases.Decls, typeDecl(goName, selector("mcp", name), comment, true))
			// Both directions require assignability between named types, not just
			// a permitted explicit conversion between identical underlying types.
			for _, pair := range [][2]ast.Expr{{selector("mcp", name), ident(goName)}, {ident(goName), selector("mcp", name)}} {
				c.aliases.Decls = append(c.aliases.Decls, &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ident("_")}, Type: pair[0], Values: []ast.Expr{&ast.StarExpr{X: call(ident("new"), pair[1])}}}}})
			}
			hasAlias = true
		case "owned", "owned-codec":
			if err := c.declare(goName, node, comment); err != nil {
				return err
			}
		default:
			return fmt.Errorf("emit: %s: unsupported disposition %q", node.Pointer, disposition)
		}
	}
	if hasAlias {
		c.aliases.Decls = append([]ast.Decl{imported("mcp", "github.com/modelcontextprotocol/go-sdk/mcp")}, c.aliases.Decls...)
	}
	imports := []ast.Decl{}
	if len(c.owned.Decls) > 0 {
		imports = append(imports, imported("", "encoding/json/jsontext"))
	}
	if c.usesJSON {
		imports = append(imports, imported("json", "encoding/json/v2"))
	}
	c.owned.Decls = append(imports, c.owned.Decls...)
	c.constants.Decls = append(c.constants.Decls, &ast.GenDecl{Tok: token.CONST, Doc: doc("ProtocolRevision identifies the reviewed schema manifest."), Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ident("ProtocolRevision")}, Values: []ast.Expr{literal(c.opts.Revision)}}}})
	methods := map[string]string{}
	for _, name := range sortedKeys(c.opts.Model.Definitions) {
		node := c.opts.Model.Definitions[name]
		if prop := node.Properties["method"]; prop != nil {
			if method, ok := prop.Const.(string); ok {
				methods[method] = method
			}
		}
	}
	values := make([]ast.Expr, 0, len(methods))
	for _, method := range sortedKeys(methods) {
		values = append(values, literal(method))
	}
	// A fresh slice prevents consumers from mutating the global registry.
	c.constants.Decls = append(c.constants.Decls, &ast.FuncDecl{Doc: doc("MethodNames returns the canonical schema method inventory."), Name: ident("MethodNames"), Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{field("", &ast.ArrayType{Elt: ident("string")})}}}, Body: &ast.BlockStmt{List: []ast.Stmt{returned(&ast.CompositeLit{Type: &ast.ArrayType{Elt: ident("string")}, Elts: values})}}})
	return nil
}

func (c *compiler) declare(name string, node *model.Node, comment string) error {
	if len(node.Compositions["allOf"]) > 0 {
		projected, err := c.objectIntersection(node)
		if err != nil {
			return err
		}
		node = projected
	}
	if isUnion(node) {
		return c.declareUnion(name, node, comment)
	}
	typ, err := c.shape(name, node)
	if err != nil {
		return err
	}
	c.owned.Decls = append(c.owned.Decls, typeDecl(name, typ, comment, false))
	c.owned.Decls = append(c.owned.Decls, c.ownedHooks(name, node)...)
	return nil
}

func isUnion(node *model.Node) bool {
	return strings.Contains(node.Type, "|") || len(node.Compositions["anyOf"]) > 0 || len(node.Compositions["oneOf"]) > 0
}

func (c *compiler) reference(node *model.Node) (ast.Expr, bool, error) {
	ref, ok := node.Annotations["$ref"].(string)
	if !ok {
		return nil, false, nil
	}
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return nil, true, fmt.Errorf("emit: %s: reference has no reviewed declaration binding", node.Pointer)
	}
	name := strings.TrimPrefix(ref, prefix)
	name = strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~")
	goName, ok := c.names[name]
	if !ok {
		return nil, true, fmt.Errorf("emit: %s: missing reference binding %q", node.Pointer, ref)
	}
	return ident(goName), true, nil
}

func (c *compiler) use(name string, node *model.Node) (ast.Expr, error) {
	if typ, ok, err := c.reference(node); ok {
		return typ, err
	}
	if isUnion(node) || len(node.Compositions["allOf"]) > 0 || node.Type == "object" || len(node.Properties) > 0 {
		if err := c.claim(name, node.Pointer); err != nil {
			return nil, err
		}
		if err := c.declare(name, node, name+" binds "+node.Pointer+"; mapping revision: "+c.opts.Revision+"."); err != nil {
			return nil, err
		}
		return ident(name), nil
	}
	return c.shape(name, node)
}

func (c *compiler) shape(name string, node *model.Node) (ast.Expr, error) {
	if typ, ok, err := c.reference(node); ok {
		return typ, err
	}
	switch node.Type {
	case "object":
		members := []*ast.Field{}
		seen := map[string]string{"marshaljsonto": "codec method", "unmarshaljsonfrom": "codec method"}
		for _, propName := range sortedKeys(node.Properties) {
			prop := node.Properties[propName]
			goName := exportedName(propName)
			first, _ := utf8.DecodeRuneInString(goName)
			if !token.IsIdentifier(goName) || !unicode.IsUpper(first) {
				return nil, fmt.Errorf("emit: %s: invalid field name %q", prop.Pointer, propName)
			}
			key := strings.ToLower(goName)
			if prior, ok := seen[key]; ok {
				return nil, fmt.Errorf("emit: %s: field collision %q with %s", prop.Pointer, goName, prior)
			}
			seen[key] = prop.Pointer
			// Inline field schemas have their own namespace. Request.params and
			// the distinct top-level RequestParams are both real schema bindings.
			typ, err := c.use(name+"Fields"+goName, prop)
			if err != nil {
				return nil, err
			}
			required := false
			for _, req := range node.Required {
				if req == propName {
					required = true
				}
			}
			tag := propName
			if !required {
				typ = generic("Field", typ)
				tag += ",omitzero"
			}
			member := field(goName, typ)
			member.Tag = literal("json:" + strconv.Quote(tag))
			members = append(members, member)
		}
		if node.Open == nil || *node.Open {
			if prior, ok := seen["additional"]; ok {
				return nil, fmt.Errorf("emit: %s: additional properties field collision with %s", node.Pointer, prior)
			}
			value := selector("jsontext", "Value")
			if extra := node.Additional; extra != nil {
				var err error
				value, err = c.use(name+"Additional", extra)
				if err != nil {
					return nil, err
				}
			}
			member := field("Additional", &ast.MapType{Key: ident("string"), Value: value})
			member.Tag = literal(`json:",embed"`)
			members = append(members, member)
		}
		return &ast.StructType{Fields: &ast.FieldList{List: members}}, nil
	case "array":
		item := selector("jsontext", "Value")
		if node.Items != nil {
			var err error
			item, err = c.use(name+"Item", node.Items)
			if err != nil {
				return nil, err
			}
		}
		return &ast.ArrayType{Elt: item}, nil
	case "string":
		return ident("string"), nil
	case "number", "integer":
		return ident("Number"), nil
	case "boolean":
		return ident("bool"), nil
	case "null", "":
		return selector("jsontext", "Value"), nil
	default:
		return nil, fmt.Errorf("emit: %s: unsupported type %q", node.Pointer, node.Type)
	}
}

func (c *compiler) ownedHooks(name string, node *model.Node) []ast.Decl {
	pointer := node.Pointer
	plain := &ast.DeclStmt{Decl: typeDecl("plain", ident(name), "", false)}
	plain2 := &ast.DeclStmt{Decl: typeDecl("plain", ident(name), "", false)}
	var base ast.Expr = ident("plain")
	if node.Type == "number" || node.Type == "integer" {
		base = ident("Number")
	}
	if node.Type == "null" || node.Type == "" {
		if _, ref := node.Annotations["$ref"]; !ref {
			base = selector("jsontext", "Value")
		}
	}
	marshalBody := []ast.Stmt{returned(call(ident("marshalOwned"), ident("encoder"), literal(pointer), call(base, ident("value"))))}
	unmarshalBody := []ast.Stmt{returned(call(ident("unmarshalOwned"), ident("decoder"), literal(pointer), call(&ast.ParenExpr{X: &ast.StarExpr{X: base}}, ident("value"))))}
	if id, ok := base.(*ast.Ident); ok && id.Name == "plain" {
		marshalBody = append([]ast.Stmt{plain}, marshalBody...)
		unmarshalBody = append([]ast.Stmt{plain2}, unmarshalBody...)
	}
	marshal := method(name, false, "MarshalJSONTo", []*ast.Field{field("encoder", &ast.StarExpr{X: selector("jsontext", "Encoder")})}, []*ast.Field{field("", ident("error"))}, marshalBody)
	unmarshal := method(name, true, "UnmarshalJSONFrom", []*ast.Field{field("decoder", &ast.StarExpr{X: selector("jsontext", "Decoder")})}, []*ast.Field{field("", ident("error"))}, unmarshalBody)
	return []ast.Decl{marshal, unmarshal}
}

func method(receiver string, pointer bool, name string, params, results []*ast.Field, body []ast.Stmt) *ast.FuncDecl {
	var typ ast.Expr = ident(receiver)
	if pointer {
		typ = &ast.StarExpr{X: typ}
	}
	comment := name + " returns the requested typed union branch or a validation error."
	switch name {
	case "MarshalJSONTo":
		comment = "MarshalJSONTo validates and encodes the owned value using its schema contract."
	case "UnmarshalJSONFrom":
		comment = "UnmarshalJSONFrom decodes and validates the owned value using its schema contract."
	}
	return &ast.FuncDecl{Doc: doc(comment), Recv: &ast.FieldList{List: []*ast.Field{field("value", typ)}}, Name: ident(name), Type: &ast.FuncType{Params: &ast.FieldList{List: params}, Results: &ast.FieldList{List: results}}, Body: &ast.BlockStmt{List: body}}
}

func (c *compiler) declareUnion(name string, node *model.Node, comment string) error {
	c.usesJSON = true
	c.owned.Decls = append(c.owned.Decls, typeDecl(name, &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{field("value", selector("jsontext", "Value"))}}}, comment, false))
	c.owned.Decls = append(c.owned.Decls, method(name, false, "MarshalJSONTo", []*ast.Field{field("encoder", &ast.StarExpr{X: selector("jsontext", "Encoder")})}, []*ast.Field{field("", ident("error"))}, []ast.Stmt{returned(call(ident("marshalUnion"), ident("encoder"), literal(node.Pointer), &ast.SelectorExpr{X: ident("value"), Sel: ident("value")}))}))
	c.owned.Decls = append(c.owned.Decls, method(name, true, "UnmarshalJSONFrom", []*ast.Field{field("decoder", &ast.StarExpr{X: selector("jsontext", "Decoder")})}, []*ast.Field{field("", ident("error"))}, []ast.Stmt{returned(call(ident("unmarshalUnion"), ident("decoder"), literal(node.Pointer), &ast.UnaryExpr{Op: token.AND, X: &ast.SelectorExpr{X: ident("value"), Sel: ident("value")}}))}))
	branches := node.Compositions["anyOf"]
	if len(branches) == 0 {
		branches = node.Compositions["oneOf"]
	}
	if strings.Contains(node.Type, "|") {
		for _, kind := range strings.Split(node.Type, "|") {
			branches = append(branches, &model.Node{Pointer: node.Pointer, Type: kind})
		}
	}
	seen := map[string]bool{}
	for i, branch := range branches {
		variant := "Variant" + strconv.Itoa(i+1)
		if ref, ok := branch.Annotations["$ref"].(string); ok {
			variant = c.names[strings.TrimPrefix(ref, "#/$defs/")]
		} else if branch.Type != "" && !strings.Contains(branch.Type, "|") {
			variant = exportedName(branch.Type)
		}
		if seen[strings.ToLower(variant)] {
			return fmt.Errorf("emit: %s: union accessor collision %q", branch.Pointer, variant)
		}
		seen[strings.ToLower(variant)] = true
		typ, err := c.use(name+variant, branch)
		if err != nil {
			return err
		}
		constructor := name + "From" + variant
		if err := c.claim(constructor, branch.Pointer); err != nil {
			return err
		}
		body := []ast.Stmt{&ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ident("result")}, Type: ident(name)}}}},
			&ast.AssignStmt{Lhs: []ast.Expr{ident("body"), ident("err")}, Tok: token.DEFINE, Rhs: []ast.Expr{call(selector("json", "Marshal"), ident("input"))}},
			&ast.IfStmt{Cond: &ast.BinaryExpr{X: ident("err"), Op: token.NEQ, Y: ident("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ident("result"), ident("err")}}}}},
			&ast.AssignStmt{Lhs: []ast.Expr{ident("err")}, Tok: token.ASSIGN, Rhs: []ast.Expr{call(selector("json", "Unmarshal"), ident("body"), &ast.UnaryExpr{Op: token.AND, X: ident("result")})}},
			&ast.ReturnStmt{Results: []ast.Expr{ident("result"), ident("err")}}}
		c.owned.Decls = append(c.owned.Decls, &ast.FuncDecl{Doc: doc(constructor + " constructs the typed branch at " + branch.Pointer + "."), Name: ident(constructor), Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{field("input", typ)}}, Results: &ast.FieldList{List: []*ast.Field{field("", ident(name)), field("", ident("error"))}}}, Body: &ast.BlockStmt{List: body}})
		c.owned.Decls = append(c.owned.Decls, method(name, false, "As"+variant, nil, []*ast.Field{field("", typ), field("", ident("error"))}, []ast.Stmt{returned(call(generic("decodeVariant", typ), &ast.SelectorExpr{X: ident("value"), Sel: ident("value")}, literal(branch.Pointer), literal(branch.Type)))}))
	}
	return nil
}
