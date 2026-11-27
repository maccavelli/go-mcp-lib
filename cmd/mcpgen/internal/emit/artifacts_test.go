package emit

import (
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
)

// This reviewed fixture is derived directly from the immutable source, outside
// ordinary gates and independently of the production model/coverage walker.
//
//go:embed testdata/schema-nodes.json
var schemaNodeInventory []byte

func TestRepairA10Artifacts(t *testing.T) {
	files, manifest := reviewedOutput(t)
	var api API
	var coverage Coverage
	for _, file := range files {
		var target any
		switch file.Name {
		case "api.json":
			target = &api
		case "coverage.json":
			target = &coverage
		default:
			continue
		}
		if err := json.Unmarshal(file.Content, target); err != nil {
			t.Fatal(err)
		}
		if len(manifest.SHA256[file.Name]) != 64 {
			t.Errorf("FAIL artifact ownership identity: %s", file.Name)
		}
	}
	if api.Version != 1 || coverage.Version != 1 || api.SchemaSHA256 != profile.SchemaSHA256 || coverage.SchemaSHA256 != profile.SchemaSHA256 || api.SDKVersion != "v1.8.0" || api.SDKContext == "" {
		t.Fatal("FAIL contract artifact provenance")
	}
	declarations := map[string]APIDeclaration{}
	for _, declaration := range api.Declarations {
		if declaration.Name == "" || declaration.Signature == "" || declarations[declaration.Name].Name != "" {
			t.Errorf("FAIL duplicate or empty API declaration: %s", declaration.Name)
		}
		declarations[declaration.Name] = declaration
	}
	for name, signature := range map[string]string{
		"Number":                            "type Number string",
		"Field.Get":                         "func (value Field[T]) Get() (T, bool)",
		"ContentBlockFromTextContent":       "func ContentBlockFromTextContent(input TextContent) (ContentBlock, error)",
		"CallToolRequest.UnmarshalJSONFrom": "func (value *CallToolRequest) UnmarshalJSONFrom(decoder *jsontext.Decoder) error",
	} {
		if declarations[name].Signature != signature {
			t.Errorf("FAIL independent API signature %s: %q", name, declarations[name].Signature)
		}
	}
	if strings.Contains(declarations["Field"].Signature, "present") || declarations["CallToolRequest.UnmarshalJSONFrom"].Imports["jsontext"] != "encoding/json/jsontext" {
		t.Error("FAIL API includes implementation storage or unresolved qualifier")
	}
	aliases := 0
	for _, binding := range coverage.Definitions {
		declaration := declarations[binding.GoName]
		if declaration.Kind != "type" || declaration.Pointer != binding.Pointer {
			t.Errorf("FAIL API definition binding: %s", binding.GoName)
		}
		if binding.Disposition == "alias" {
			aliases++
			for _, target := range []string{"darwin/arm64", "linux/amd64", "windows/amd64"} {
				if declaration.SDKTargets[target].Identity != "github.com/modelcontextprotocol/go-sdk/mcp."+binding.Definition || declaration.SDKTargets[target].Underlying == "" {
					t.Errorf("FAIL API compiler target: %s %s", binding.GoName, target)
				}
			}
		}
	}
	var expected struct {
		Version      int                 `json:"version"`
		SchemaSHA256 string              `json:"schema-sha256"`
		Nodes        map[string][]string `json:"nodes"`
	}
	if err := json.Unmarshal(schemaNodeInventory, &expected); err != nil {
		t.Fatal(err)
	}
	if expected.Version != 1 || expected.SchemaSHA256 != profile.SchemaSHA256 || len(expected.Nodes) != 840 {
		t.Fatal("FAIL independent schema node inventory identity")
	}
	if aliases != 14 || len(coverage.Definitions) != 155 || len(coverage.Nodes) != len(expected.Nodes) {
		t.Errorf("FAIL coverage node/definition inventory: aliases=%d definitions=%d nodes=%d", aliases, len(coverage.Definitions), len(coverage.Nodes))
	}
	nodes := map[string]nodeCoverage{}
	for _, node := range coverage.Nodes {
		if _, exists := nodes[node.Pointer]; exists {
			t.Errorf("FAIL duplicate coverage node %s", node.Pointer)
		}
		nodes[node.Pointer] = node
	}
	for pointer, keywords := range expected.Nodes {
		if actual, exists := nodes[pointer]; !exists || !slices.Equal(sortedKeys(actual.Keywords), keywords) {
			t.Errorf("FAIL coverage source location or keyword inventory: %s", pointer)
		}
	}
	if nodes["#"].Keywords["$defs"] != "assertion" || nodes["#"].Keywords["$schema"] != "annotation" || nodes["#/$defs/JSONValue"].Keywords["anyOf"] != "assertion" {
		t.Error("FAIL coverage keyword binding")
	}
	if len(coverage.Methods) != 21 || coverage.Qualification["codec-runtime"] != "pending R4" || !strings.HasPrefix(coverage.Qualification["canonical-runtime"], "pending R5") {
		t.Error("FAIL coverage method inventory or runtime qualification boundary")
	}
	foundTool, foundCapability, foundCancellation := false, false, false
	for _, method := range coverage.Methods {
		if method.Name == "notifications/cancelled" {
			foundCancellation = method.Pointer == "#/$defs/CancelledNotification" && nodes["#/$defs/ClientNotification"].Pointer != ""
		}
		if method.Name == "tools/call" {
			foundTool = method.Pointer == "#/$defs/CallToolRequest" && method.Capability == "tools" && method.QualificationOwner == "R5: TestRepairA19CanonicalRuntime"
		}
	}
	for _, capability := range coverage.Capabilities {
		if capability.Scope == "server" && capability.Name == "tools" && capability.Pointer == "#/$defs/ServerCapabilities/properties/tools" {
			foundCapability = true
		}
	}
	if !foundTool || !foundCapability || !foundCancellation {
		t.Error("FAIL coverage method/capability provenance")
	}
}

func TestRepairA10ArtifactInputs(t *testing.T) {
	m, mapping := pinnedModel(t)
	inputs, err := profile.Load("../../../..", profile.Revision)
	if err != nil {
		t.Fatal(err)
	}
	mapping, err = inputs.ApplySDK(mapping)
	if err != nil {
		t.Fatal(err)
	}
	opts := Options{Revision: profile.Revision, Model: m, Dispositions: mapping,
		Names: inputs.Mapping.GoNames, Methods: slices.Clone(inputs.Methods), PackagePath: "example.invalid/protocol"}
	first, _, err := Render(opts)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(opts.Methods)
	second, _, err := Render(opts)
	if err != nil {
		t.Fatal(err)
	}
	for i := range first {
		if first[i].Name != second[i].Name || string(first[i].Content) != string(second[i].Content) {
			t.Errorf("FAIL artifact nondeterminism: %s", first[i].Name)
		}
	}
	opts.Methods = opts.Methods[1:]
	if _, _, err := Render(opts); err == nil || !strings.Contains(err.Error(), "method coverage inventory") {
		t.Errorf("FAIL method coverage omission accepted: %v", err)
	}
	opts.Methods = inputs.Methods
	// Copy metadata before tampering: the SDK audit cache remains immutable.
	var copyMapping = *mapping
	encoded, err := json.Marshal(mapping.TypeAPI)
	if err != nil {
		t.Fatal(err)
	}
	// Unmarshal into a fresh map rather than reusing the shallow copy's map.
	copyMapping.TypeAPI = nil
	if err := json.Unmarshal(encoded, &copyMapping.TypeAPI); err != nil {
		t.Fatal(err)
	}
	delete(copyMapping.TypeAPI["Tool"], "windows/amd64")
	opts.Dispositions = &copyMapping
	if _, _, err := Render(opts); err == nil || !strings.Contains(err.Error(), "compiler API for alias Tool") {
		t.Errorf("FAIL alias compiler API omission accepted: %v", err)
	}
}
