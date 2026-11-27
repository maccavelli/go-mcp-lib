package emit

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/model"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/qualify"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

func reviewedOutput(t *testing.T) ([]File, *Manifest) {
	t.Helper()
	m, actual := pinnedModel(t)
	inputs, err := profile.Load("../../../..", profile.Revision)
	if err != nil {
		t.Fatal(err)
	}
	actual, err = inputs.ApplySDK(actual)
	if err != nil {
		t.Fatal(err)
	}
	files, manifest, err := Render(Options{Revision: profile.Revision, Model: m, Dispositions: actual, Names: inputs.Mapping.GoNames, Methods: inputs.Methods, PackagePath: "github.com/maccavelli/go-mcp-lib/protocol"})
	if err != nil {
		t.Fatalf("FAIL reviewed AST emission: %v", err)
	}
	return files, manifest
}

func TestRepairA10TotalBindings(t *testing.T) {
	files, manifest := reviewedOutput(t)
	declarations := map[string]int{}
	var bindings []Binding
	for _, file := range files {
		sum := sha256.Sum256(file.Content)
		if manifest.SHA256[file.Name] != hex.EncodeToString(sum[:]) {
			t.Errorf("FAIL manifest content identity: %s", file.Name)
		}
		if file.Name == "bindings.json" {
			if err := json.Unmarshal(file.Content, &bindings); err != nil {
				t.Fatal(err)
			}
		}
		if !strings.HasSuffix(file.Name, ".go") {
			continue
		}
		tree, err := parser.ParseFile(token.NewFileSet(), file.Name, file.Content, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(tree, func(node ast.Node) bool {
			if spec, ok := node.(*ast.TypeSpec); ok {
				declarations[spec.Name.Name]++
			}
			return true
		})
	}
	if manifest.Version != 2 || len(manifest.SHA256) != 8 || len(bindings) != 155 {
		t.Fatalf("FAIL incomplete bindings or hashed inventory: %d", len(bindings))
	}
	seen := map[string]bool{}
	for _, binding := range bindings {
		if binding.Pointer != "#/$defs/"+binding.Definition || declarations[binding.GoName] != 1 || seen[binding.Definition] {
			t.Errorf("FAIL missing or duplicate typed binding: %+v", binding)
		}
		seen[binding.Definition] = true
	}
	if declarations["RequestID"] != 1 || declarations["RequestId"] != 0 {
		t.Error("FAIL reviewed Go name was not consumed")
	}
	for _, name := range []string{"ContentBlock", "InputRequest", "InputResponse", "InputRequests", "InputResponses", "JSONObject", "JSONValue", "JSONArray", "ProgressToken"} {
		if declarations[name] != 1 {
			t.Errorf("FAIL required Option D binding: %s", name)
		}
	}
}

func TestRepairA10NameGuards(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"case-fold", `{"$defs":{"Thing":{"type":"string"},"THING":{"type":"string"}}}`, "identifier collision"},
		{"keyword", `{"$defs":{"type":{"type":"string"}}}`, "invalid exported identifier"},
		{"reserved-support", `{"$defs":{"Field":{"type":"string"}}}`, "identifier collision"},
		{"field-normalization", `{"$defs":{"Example":{"type":"object","properties":{"request_id":{"type":"string"},"request-id":{"type":"string"}}}}}`, "field collision"},
		{"codec-method", `{"$defs":{"Example":{"type":"object","properties":{"marshalJSONTo":{"type":"string"}}}}}`, "field collision"},
		{"additional-field", `{"$defs":{"Example":{"type":"object","properties":{"additional":{"type":"string"}}}}}`, "additional properties field collision"},
		{"auxiliary", `{"$defs":{"Example":{"type":"object","properties":{"nested":{"type":"object"}}},"ExampleFieldsNested":{"type":"string"}}}`, "identifier collision"},
		{"unbound-reference", `{"$defs":{"Example":{"$ref":"#/$defs/Missing"}}}`, "missing reference binding"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			doc, err := load.New().LoadBytes("fixture", []byte(test.body))
			if err != nil {
				t.Fatal(err)
			}
			m, err := model.Build(doc)
			if err != nil {
				t.Fatal(err)
			}
			dispositions := &sdkmap.Result{Dispositions: map[string]string{}}
			for name := range m.Definitions {
				dispositions.Dispositions[name] = "owned"
			}
			files, manifest, err := Render(Options{Revision: "fixture", Model: m, Dispositions: dispositions, PackagePath: "example.invalid/protocol"})
			if err == nil || !strings.Contains(err.Error(), test.want) || files != nil || manifest != nil {
				t.Errorf("FAIL emission guard %s: output=%d manifest=%v error=%v", test.name, len(files), manifest, err)
			}
		})
	}
}

//go:embed wire_test.go.txt
var wireTests []byte

//go:embed alias_wire_test.go.txt
var aliasWireTests []byte

// TestRepairA10GeneratedWire executes independently handwritten fixtures against
// the emitted package. This intentionally does not claim full runtime integration.
func TestRepairA10GeneratedWire(t *testing.T) {
	files, _ := reviewedOutput(t)
	root := t.TempDir()
	for _, relative := range []string{"go.mod", "go.sum"} {
		copyTestFile(t, filepath.Join("../../../..", relative), filepath.Join(root, relative))
	}
	entries, err := os.ReadDir("../../../../internal/validation")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".go") {
			copyTestFile(t, filepath.Join("../../../../internal/validation", entry.Name()), filepath.Join(root, "internal/validation", entry.Name()))
		}
	}
	protocol := filepath.Join(root, "protocol")
	if err := os.MkdirAll(protocol, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(protocol, file.Name), file.Content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copyTestFile(t, "../../../../protocol/doc.go", filepath.Join(protocol, "doc.go"))
	if err := os.WriteFile(filepath.Join(protocol, "wire_test.go"), wireTests, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protocol, "alias_wire_test.go"), aliasWireTests, 0o644); err != nil {
		t.Fatal(err)
	}
	cache, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	env := qualify.GoEnv(strings.TrimSpace(string(cache)))
	env = append(env, "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-vet=off", "-count=1", "-v", "./protocol")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("FAIL generated wire package: %v\n%s", err, output)
	}
	if !bytes.Contains(output, []byte("PASS: TestIndependentWire")) {
		t.Fatalf("FAIL generated wire fixture did not execute: %s", output)
	}
	t.Logf("generated wire fixtures: %s", output)
	// Generated sources are a different analyzer input from the generator. The
	// canonical gate passes its already selected pinned executable to this test.
	linter := os.Getenv("GO_MCP_LINTER")
	if linter == "" {
		resolved, err := exec.LookPath("golangci-lint")
		if err != nil {
			t.Fatalf("FAIL generated linter unavailable: %v", err)
		}
		linter = resolved
	}
	configuration, err := filepath.Abs("../../../../.golangci.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range [][]string{{"go", "vet", "./..."}, {linter, "run", "--config", configuration}} {
		check := exec.CommandContext(ctx, command[0], command[1:]...)
		check.Dir = root
		check.Env = cmd.Env
		output, err := check.CombinedOutput()
		if err != nil {
			t.Fatalf("FAIL generated candidate analyzer %s: %v\n%s", command[0], err, output)
		}
	}
	// Mutation controls use only this scratch package and must fail observably.
	source := filepath.Join(protocol, "codecs.go")
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	anchor := []byte("func validateBinding(pointer string, body []byte) error {")
	if bytes.Count(original, anchor) != 1 {
		t.Fatal("FAIL validation plant anchor changed")
	}
	planted := bytes.Replace(original, anchor, append(append([]byte{}, anchor...), []byte("\nif pointer != \"\" { return nil }\n")...), 1)
	if err := os.WriteFile(source, planted, 0o644); err != nil {
		t.Fatal(err)
	}
	broken := exec.CommandContext(ctx, "go", "test", "-vet=off", "-count=1", "-run", "^TestIndependentWire$/^(invalid-input|allof-errors)$", "./protocol")
	broken.Dir = root
	broken.Env = cmd.Env
	output, err = broken.CombinedOutput()
	if err == nil || !bytes.Contains(output, []byte("FAIL invalid wire accepted")) || !bytes.Contains(output, []byte("FAIL allOf branch assertion bypassed")) {
		t.Fatalf("FAIL generated validation bypass was not detected: %v\n%s", err, output)
	}
	t.Log("generated wire validation bypass caught: FAIL invalid wire accepted; FAIL allOf branch assertion bypassed")
	// Keep the codec corpus independent of serializer implementations. Dropping
	// SDK fields in a compilable scratch serializer must fail its literal fixtures.
	anchor = []byte("return legacyjson.Marshal(value)")
	if bytes.Count(aliasWireTests, anchor) != 1 {
		t.Fatal("FAIL alias wire plant anchor changed")
	}
	planted = bytes.Replace(aliasWireTests, anchor, []byte("if value != nil { return []byte(\"{}\"), nil }; return legacyjson.Marshal(value)"), 1)
	if err := os.WriteFile(filepath.Join(protocol, "alias_wire_test.go"), planted, 0o644); err != nil {
		t.Fatal(err)
	}
	broken = exec.CommandContext(ctx, "go", "test", "-vet=off", "-count=1", "-run", "^TestSDKAliasWireContract$", "./protocol")
	broken.Dir, broken.Env = root, cmd.Env
	output, err = broken.CombinedOutput()
	if err == nil || !bytes.Contains(output, []byte("FAIL SDK alias wire")) {
		t.Fatalf("FAIL SDK alias codec bypass was not detected: %v\n%s", err, output)
	}
	t.Log("SDK alias codec bypass caught: FAIL SDK alias wire")
}

func copyTestFile(t *testing.T, source, destination string) {
	t.Helper()
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRepairA10DocumentSnapshot(t *testing.T) {
	doc, err := load.New().LoadBytes("fixture", []byte(`{"$defs":{"Value":{"type":"string"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	m, err := model.Build(doc)
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), m.Document...)
	doc.Bytes[0] = '!'
	if !reflect.DeepEqual(m.Document, before) {
		t.Fatal("FAIL model schema bytes share mutable loader storage")
	}
}
