// Package emit compiles reviewed schema bindings into Go syntax trees.
package emit

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/model"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

// Options configures an emission. Names are the independently reviewed names;
// a nil map preserves definition names for isolated fixture models.
type Options struct {
	Revision     string
	Model        *model.Model
	Dispositions *sdkmap.Result
	PackagePath  string
	Names        map[string]string
}

// File contains final bytes, including formatting, covered by the manifest.
type File struct {
	Name    string
	Content []byte
}

// Manifest records exact inventory and content identities. Hashless historical
// manifests are not evidence that files may be overwritten or removed.
type Manifest struct {
	Version  int               `json:"version"`
	Revision string            `json:"revision"`
	Files    []string          `json:"files"`
	SHA256   map[string]string `json:"sha256"`
}

// Binding is the correspondence between a schema and its public name.
type Binding struct {
	Definition  string `json:"definition"`
	Pointer     string `json:"pointer"`
	GoName      string `json:"go-name"`
	Disposition string `json:"disposition"`
}

//go:embed support.go.txt
var supportSource string

// Render emits every definition or returns an error before producing output.
func Render(opts Options) ([]File, *Manifest, error) {
	if opts.Model == nil || opts.Dispositions == nil || opts.Revision == "" || opts.PackagePath == "" {
		return nil, nil, fmt.Errorf("emit: model, dispositions, revision and package path are required")
	}
	c, err := newCompiler(opts)
	if err != nil {
		return nil, nil, err
	}
	if err := c.compile(); err != nil {
		return nil, nil, err
	}
	files := make([]File, 0, 6)
	for _, item := range []struct {
		name string
		tree *ast.File
	}{
		{"aliases.go", c.aliases}, {"owned.go", c.owned}, {"constants.go", c.constants},
	} {
		body, err := formatAST(item.tree)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, File{item.name, body})
	}
	// Generic support is parsed into an AST. Only its package identifier changes;
	// schema text is never interpolated into Go source.
	fset := token.NewFileSet()
	support, err := parser.ParseFile(fset, "codecs.go", supportSource, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("emit: codec support: %w", err)
	}
	support.Name.Name = path.Base(opts.PackagePath)
	var formatted bytes.Buffer
	if err := format.Node(&formatted, fset, support); err != nil {
		return nil, nil, err
	}
	files = append(files, File{"codecs.go", formatted.Bytes()})
	bindings, err := json.MarshalIndent(c.bindings, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	files = append(files, File{"bindings.json", append(bindings, '\n')})
	if len(opts.Model.Document) == 0 {
		return nil, nil, fmt.Errorf("emit: original schema bytes are required")
	}
	files = append(files, File{"schema.json", bytes.Clone(opts.Model.Document)})
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	manifest := &Manifest{Version: 2, Revision: opts.Revision, SHA256: make(map[string]string)}
	for _, file := range files {
		manifest.Files = append(manifest.Files, file.Name)
		sum := sha256.Sum256(file.Content)
		manifest.SHA256[file.Name] = hex.EncodeToString(sum[:])
	}
	return files, manifest, nil
}

func formatAST(file *ast.File) ([]byte, error) {
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), file); err != nil {
		return nil, fmt.Errorf("emit: format syntax tree: %w", err)
	}
	return FormatGo("generated.go", out.Bytes())
}

// FormatGo verifies syntax and returns canonical bytes; JSON artifacts bypass it.
func FormatGo(name string, content []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("emit: %s does not parse: %w", name, err)
	}
	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		return nil, fmt.Errorf("emit: %s does not format: %w", name, err)
	}
	return out.Bytes(), nil
}

func ident(name string) *ast.Ident { return ast.NewIdent(name) }
func literal(value string) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(value)}
}
func selector(pkg, name string) ast.Expr                { return &ast.SelectorExpr{X: ident(pkg), Sel: ident(name)} }
func call(fun ast.Expr, args ...ast.Expr) *ast.CallExpr { return &ast.CallExpr{Fun: fun, Args: args} }
func field(name string, typ ast.Expr) *ast.Field {
	f := &ast.Field{Type: typ}
	if name != "" {
		f.Names = []*ast.Ident{ident(name)}
	}
	return f
}
func doc(text string) *ast.CommentGroup {
	if text == "" {
		return nil
	}
	return &ast.CommentGroup{List: []*ast.Comment{{Text: "// " + text}}}
}
func generic(name string, typ ast.Expr) ast.Expr { return &ast.IndexExpr{X: ident(name), Index: typ} }
func returned(expr ast.Expr) ast.Stmt            { return &ast.ReturnStmt{Results: []ast.Expr{expr}} }
func typeDecl(name string, typ ast.Expr, comment string, alias bool) ast.Decl {
	spec := &ast.TypeSpec{Name: ident(name), Type: typ}
	if alias {
		spec.Assign = token.Pos(1)
	}
	return &ast.GenDecl{Doc: doc(comment), Tok: token.TYPE, Specs: []ast.Spec{spec}}
}
func imported(name, importPath string) ast.Decl {
	spec := &ast.ImportSpec{Path: literal(importPath)}
	if name != "" {
		spec.Name = ident(name)
	}
	return &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}
}
func newFile(pkg, revision string) *ast.File {
	return &ast.File{Doc: doc("Code generated by mcpgen; mapping revision: " + revision + ". DO NOT EDIT."), Name: ident(pkg)}
}

// exportedName keeps wire spelling in tags; normalized-name collisions fail.
func exportedName(value string) string {
	initialisms := map[string]string{"id": "ID", "uri": "URI", "url": "URL", "json": "JSON", "jsonrpc": "JSONRPC", "rpc": "RPC", "api": "API", "mime": "MIME", "ttl": "TTL", "http": "HTTP", "https": "HTTPS", "uuid": "UUID", "utf8": "UTF8", "xml": "XML"}
	parts := strings.FieldsFunc(value, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var words []string
	for _, part := range parts {
		runes := []rune(part)
		start := 0
		for i := 1; i < len(runes); i++ {
			boundary := unicode.IsUpper(runes[i]) && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) || (i+1 < len(runes) && unicode.IsUpper(runes[i-1]) && unicode.IsLower(runes[i+1])))
			if boundary {
				words = append(words, string(runes[start:i]))
				start = i
			}
		}
		words = append(words, string(runes[start:]))
	}
	for i, word := range words {
		if replacement, ok := initialisms[strings.ToLower(word)]; ok {
			words[i] = replacement
		} else {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, "")
}
