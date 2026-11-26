package emit

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/model"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

func projectionCompiler(t *testing.T, body []byte) *compiler {
	t.Helper()
	doc, err := load.New().LoadBytes("fixture", body)
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
	c, err := newCompiler(Options{Revision: "fixture", Model: m, Dispositions: dispositions, PackagePath: "example.invalid/protocol"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRepairA10IntersectionProjection(t *testing.T) {
	body, err := os.ReadFile("../../../../schema/inputs/2026-07-28/schema.json")
	if err != nil {
		t.Fatal(err)
	}
	c := projectionCompiler(t, body)
	before, err := json.Marshal(c.opts.Model)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"HeaderMismatchError", "MissingRequiredClientCapabilityError", "UnsupportedProtocolVersionError"} {
		t.Run(name, func(t *testing.T) {
			node := c.opts.Model.Definitions[name].Properties["error"]
			projected, err := c.objectIntersection(node)
			if err != nil {
				t.Fatal(err)
			}
			expected := []string{"code", "message"}
			if name != "HeaderMismatchError" {
				expected = []string{"code", "data", "message"}
			}
			if !reflect.DeepEqual(projected.Required, expected) {
				t.Errorf("FAIL intersection requiredness: %s: %v", name, projected.Required)
			}
			if projected.Pointer != node.Pointer || projected.Type != "object" || projected.Properties["message"].Type != "string" || projected.Properties["code"].Type != "integer" {
				t.Errorf("FAIL intersection lost original binding or base fields: %s", name)
			}
			if name != "HeaderMismatchError" && projected.Properties["data"].Type != "object" {
				t.Errorf("FAIL intersection data stayed untyped: %s", name)
			}
		})
	}
	after, err := json.Marshal(c.opts.Model)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("FAIL intersection modified the source model")
	}
	cases := []struct{ name, body, want string }{
		{"non-object", `{"$defs":{"Example":{"allOf":[{"type":"object"},{"type":"string"}]}}}`, "no qualified object representation"},
		{"incompatible-field", `{"$defs":{"Example":{"allOf":[{"type":"object","properties":{"x":{"type":"string"}}},{"type":"object","properties":{"x":{"type":"integer"}}}]}}}`, "no qualified field representation"},
		{"recursive", `{"$defs":{"Example":{"allOf":[{"$ref":"#/$defs/Example"},{"type":"object"}]}}}`, "recursive or over-depth"},
		{"missing-ref", `{"$defs":{"Example":{"allOf":[{"$ref":"#/$defs/Missing"},{"type":"object"}]}}}`, "missing reference binding"},
		{"unconstrained", `{"$defs":{"Example":{"allOf":[{},{}]}}}`, "does not constrain an object"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			c := projectionCompiler(t, []byte(test.body))
			if _, err := c.objectIntersection(c.opts.Model.Definitions["Example"]); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("FAIL unsupported intersection accepted: %s: %v", test.name, err)
			}
		})
	}
	t.Run("shared-reference-dag", func(t *testing.T) {
		definitions := map[string]*model.Node{"N0": {Pointer: "#/$defs/N0", Type: "object"}}
		for i := 1; i <= 12; i++ {
			name := "N" + strconv.Itoa(i)
			ref := &model.Node{Pointer: "#/$defs/" + name + "/allOf/0", Annotations: map[string]any{"$ref": "#/$defs/N" + strconv.Itoa(i-1)}}
			definitions[name] = &model.Node{Pointer: "#/$defs/" + name, Compositions: map[string][]*model.Node{"allOf": {ref, ref}}}
		}
		c := &compiler{opts: Options{Model: &model.Model{Definitions: definitions}}}
		var leaves []*model.Node
		if err := c.objectLeaves(definitions["N12"], map[*model.Node]bool{}, map[*model.Node]bool{}, 0, &leaves); err != nil {
			t.Fatal(err)
		}
		if len(leaves) != 25 {
			t.Errorf("FAIL shared intersection DAG expanded repeatedly: %d", len(leaves))
		}
	})
}
