package emit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/model"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/testenv"
)

func pinnedModel(t *testing.T) (*model.Model, *sdkmap.Result) {
	t.Helper()
	body, err := os.ReadFile("../../../../schema/inputs/2026-07-28/schema.json")
	if err != nil {
		t.Fatalf("FAIL schema: %v", err)
	}
	doc, err := load.New().LoadBytes("pinned", body)
	if err != nil {
		t.Fatalf("FAIL load: %v", err)
	}
	m, err := model.Build(doc)
	if err != nil {
		t.Fatalf("FAIL model: %v", err)
	}
	dir := testenv.SDKDir(t)
	var names []string
	for name := range m.Definitions {
		names = append(names, name)
	}
	dispositions, err := sdkmap.Build("2026-07-28", dir, "v1.8.0", names)
	if err != nil {
		t.Fatalf("FAIL sdkmap: %v", err)
	}
	return m, dispositions
}

func renderAll(t *testing.T) ([]File, *Manifest) {
	t.Helper()
	return reviewedOutput(t)
}

func TestRenderDeterministic(t *testing.T) {
	first, firstManifest := renderAll(t)
	second, secondManifest := renderAll(t)
	if firstManifest.Revision != secondManifest.Revision {
		t.Fatal("FAIL manifest revision drift")
	}
	if len(first) != len(second) {
		t.Fatalf("FAIL file count drift: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Name != second[i].Name {
			t.Fatalf("FAIL file name drift: %s vs %s", first[i].Name, second[i].Name)
		}
		if !bytes.Equal(first[i].Content, second[i].Content) {
			t.Fatalf("FAIL nondeterministic emission of %s", first[i].Name)
		}
	}
}

func TestEveryDeclarationCarriesSourcePointer(t *testing.T) {
	files, _ := renderAll(t)
	for _, file := range files {
		content := string(file.Content)
		// Each type declaration's doc comment must reference the schema
		// pointer or the mapping revision.
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "type ") {
				// find the doc comment block above this line
				lines := strings.Split(content, "\n")
				for j, l := range lines {
					if l == line {
						comment := collectDoc(lines[:j])
						if !strings.Contains(comment, "#/$defs/") && !strings.Contains(comment, "mapping revision") {
							t.Errorf("FAIL %s: declaration %q lacks source pointer", file.Name, line)
						}
						break
					}
				}
			}
		}
	}
}

func collectDoc(lines []string) string {
	var doc []string
	for j := len(lines) - 1; j >= 0; j-- {
		trimmed := strings.TrimSpace(lines[j])
		if strings.HasPrefix(trimmed, "//") {
			doc = append([]string{trimmed}, doc...)
			continue
		}
		break
	}
	return strings.Join(doc, "\n")
}

func TestDeprecatedNeverAliased(t *testing.T) {
	files, _ := renderAll(t)
	var aliases string
	for _, file := range files {
		if file.Name == "aliases.go" {
			aliases = string(file.Content)
		}
	}
	if aliases == "" {
		t.Fatal("FAIL no aliases emitted")
	}
	for _, deprecated := range []string{
		"CreateMessageResult", "ListRootsResult", "CreateMessageParams",
		"SamplingMessage", "ModelPreferences", "Root", "ToolChoice",
	} {
		if bytes.Contains([]byte(aliases), []byte("type "+deprecated+" = mcp.")) {
			t.Errorf("FAIL deprecated %s aliased directly", deprecated)
		}
	}
}

func TestOwnedDTOPublicAPI(t *testing.T) {
	files, _ := renderAll(t)
	var owned string
	for _, file := range files {
		if file.Name == "owned.go" {
			owned = string(file.Content)
		}
	}
	if !strings.Contains(owned, "type CreateMessageResult struct") {
		t.Fatal("FAIL CreateMessageResult not emitted as an owned DTO")
	}
	if !strings.Contains(owned, "type ListRootsResult struct") {
		t.Fatal("FAIL ListRootsResult not emitted as an owned DTO")
	}
}

func TestKeywordNamesSafe(t *testing.T) {
	for wire, want := range map[string]string{"type": "Type", "$schema": "Schema", "requestId": "RequestID", "ttlMs": "TTLMs", "toolUseId": "ToolUseID", "mimeType": "MIMEType", "uriTemplate": "URITemplate", "HTTPServer": "HTTPServer", "io.modelcontextprotocol/subscriptionId": "IoModelcontextprotocolSubscriptionID"} {
		if got := exportedName(wire); got != want {
			t.Errorf("FAIL Go field initialism: %s: %s, want %s", wire, got, want)
		}
	}
	files, _ := renderAll(t)
	for _, file := range files {
		content := string(file.Content)
		// struct fields like Type_ or Func_ must exist where the schema
		// names collide with Go keywords; the raw keyword must not
		// appear as a field name start.
		for _, kw := range []string{"Type", "Func", "Range", "Select"} {
			if strings.Contains(content, "\t"+kw+" json") {
				t.Errorf("FAIL %s: raw keyword %q used as field", file.Name, kw)
			}
		}
	}
}

func TestGeneratedHeader(t *testing.T) {
	files, _ := renderAll(t)
	for _, file := range files {
		if !strings.HasSuffix(file.Name, ".go") {
			continue
		}
		if err := generatedDocumentation(file.Content); err != nil {
			t.Errorf("FAIL generated documentation %s: %v", file.Name, err)
		}
	}
}

func generatedDocumentation(body []byte) error {
	syntax, err := parser.ParseFile(token.NewFileSet(), "generated.go", body, parser.ParseComments)
	if err != nil {
		return err
	}
	if !ast.IsGenerated(syntax) || !bytes.HasPrefix(body, []byte("// Code generated by mcpgen")) {
		return fmt.Errorf("standard leading generated marker required")
	}
	for _, declaration := range syntax.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if ok && method.Name.IsExported() && (method.Doc == nil || !strings.HasPrefix(method.Doc.Text(), method.Name.Name+" ")) {
			return fmt.Errorf("exported function or method %s requires documentation", method.Name.Name)
		}
	}
	return nil
}

func TestGeneratedDocumentationControls(t *testing.T) {
	header := "// Code generated by mcpgen; DO NOT EDIT.\n\n"
	valid := header + "package fixture\ntype Value struct{}\n// Method describes a fixture operation.\nfunc (Value) Method() {}\n"
	if err := generatedDocumentation([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"old-header-position": "package // Code generated by mcpgen; DO NOT EDIT.\nfixture\n",
		"missing-method-doc":  strings.Replace(valid, "// Method describes a fixture operation.\n", "", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if err := generatedDocumentation([]byte(source)); err == nil {
				t.Errorf("FAIL generated documentation accepted %s", name)
			} else {
				t.Logf("CONTROL %s: FAIL generated documentation: %v", name, err)
			}
		})
	}
}

func TestFormatGoAcceptsEmission(t *testing.T) {
	files, _ := renderAll(t)
	for _, file := range files {
		if !strings.HasSuffix(file.Name, ".go") {
			continue
		}
		if _, err := FormatGo(file.Name, file.Content); err != nil {
			t.Errorf("FAIL %s: %v", file.Name, err)
		}
	}
}
