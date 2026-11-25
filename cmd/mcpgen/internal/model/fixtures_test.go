package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

// fixtureExpectation is the handwritten expected block inside a fixture.
type fixtureExpectation struct {
	Fails              bool     `json:"fails"`
	Reason             string   `json:"reason"`
	Definitions        int      `json:"definitions"`
	Definition         string   `json:"definition"`
	ItemsDefinition    string   `json:"items-definition"`
	Recursive          bool     `json:"recursive"`
	Required           []string `json:"required"`
	Nullable           []string `json:"nullable"`
	DefaultsPreserved  []string `json:"defaults-preserved"`
	AllOfBranches      int      `json:"allOf-branches"`
	Merged             bool     `json:"merged"`
	MinimumToken       string   `json:"minimum-token"`
	MaximumToken       string   `json:"maximum-token"`
	MinItems           *int     `json:"minItems"`
	MaxItems           *int     `json:"maxItems"`
	PropertiesVerbatim []string `json:"properties-verbatim"`
	AnnotationsKept    []string `json:"annotations-kept"`
}

// TestFixtureCorpus drives every fixture under testdata/fixtures and checks
// the handwritten expected block. Expected outcomes are authored in the
// fixture files themselves, not generated from the model.
func TestFixtureCorpus(t *testing.T) {
	entries, err := os.ReadDir("../../testdata/fixtures")
	if err != nil {
		t.Fatalf("FAIL fixtures dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join("../../testdata/fixtures", name))
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Defs     map[string]json.RawMessage `json:"$defs"`
				Expected fixtureExpectation         `json:"expected"`
			}
			if err := json.Unmarshal(body, &doc); err != nil {
				t.Fatalf("FAIL fixture decode: %v", err)
			}
			loader := load.New()
			loaded, err := loader.LoadBytes(name, body)
			if err != nil {
				if doc.Expected.Fails {
					return // loader-level failure expected
				}
				t.Fatalf("FAIL load: %v", err)
			}
			m, err := Build(loaded)
			if err != nil {
				if doc.Expected.Fails {
					if doc.Expected.Reason != "" && !containsSub(err.Error(), doc.Expected.Reason) {
						t.Errorf("FAIL failure reason mismatch: %v (want %q)", err, doc.Expected.Reason)
					}
					t.Logf("expected failure: %v", err)
					return
				}
				t.Fatalf("FAIL model: %v", err)
			}
			if doc.Expected.Fails {
				t.Fatalf("FAIL fixture expected failure %q but the model succeeded", doc.Expected.Reason)
			}
			expect := doc.Expected
			if expect.Definitions > 0 && len(m.Definitions) != expect.Definitions {
				t.Errorf("FAIL definitions: got %d want %d", len(m.Definitions), expect.Definitions)
			}
			if expect.Recursive && !hasRecursiveRef(m) {
				t.Error("FAIL recursive reference collapsed")
			}
			if expect.MinimumToken != "" {
				node := fixtureNode(t, m, expect.Definition)
				if node.Minimum != expect.MinimumToken {
					t.Errorf("FAIL minimum token: got %q want %q", node.Minimum, expect.MinimumToken)
				}
			}
			if expect.MaximumToken != "" {
				node := fixtureNode(t, m, expect.Definition)
				if node.Maximum != expect.MaximumToken {
					t.Errorf("FAIL maximum token: got %q want %q", node.Maximum, expect.MaximumToken)
				}
			}
			if expect.MinItems != nil || expect.MaxItems != nil {
				n := fixtureNode(t, m, expect.ItemsDefinition)
				if expect.MinItems != nil {
					if _, present := n.Keywords["minItems"]; !present {
						t.Error("FAIL minItems: assertion absent")
					}
				}
				if expect.MaxItems != nil {
					if _, present := n.Keywords["maxItems"]; !present {
						t.Error("FAIL maxItems: assertion absent")
					}
				}
				if expect.MinItems != nil && n.MinItems != *expect.MinItems {
					t.Errorf("FAIL minItems: got %d want %d", n.MinItems, *expect.MinItems)
				}
				if expect.MaxItems != nil && n.MaxItems != *expect.MaxItems {
					t.Errorf("FAIL maxItems: got %d want %d", n.MaxItems, *expect.MaxItems)
				}
			}
			if expect.AllOfBranches > 0 {
				node := fixtureNode(t, m, expect.Definition)
				if got := len(node.Compositions["allOf"]); got != expect.AllOfBranches {
					t.Errorf("FAIL allOf branches: got %d want %d", got, expect.AllOfBranches)
				}
			}
			if !expect.Merged && expect.AllOfBranches > 1 {
				node := fixtureNode(t, m, expect.Definition)
				for _, branch := range node.Compositions["allOf"] {
					if len(branch.Required) == 0 && len(branch.Properties) == 0 {
						t.Error("FAIL allOf branches silently merged")
					}
				}
			}
			if len(expect.Nullable) > 0 {
				node := fixtureNode(t, m, expect.Definition)
				for _, name := range expect.Nullable {
					p, ok := node.Properties[name]
					if !ok || p.Type != "string|null" {
						t.Errorf("FAIL nullable %s: %+v", name, p)
					}
				}
			}
			if len(expect.Required) > 0 {
				node := fixtureNode(t, m, expect.Definition)
				for _, name := range expect.Required {
					found := false
					for _, r := range node.Required {
						if r == name {
							found = true
						}
					}
					if !found {
						t.Errorf("FAIL required %s missing", name)
					}
				}
			}
			if len(expect.DefaultsPreserved) > 0 {
				node := fixtureNode(t, m, expect.Definition)
				for _, name := range expect.DefaultsPreserved {
					p, ok := node.Properties[name]
					if !ok {
						t.Fatalf("FAIL property %s missing", name)
					}
					if _, has := p.Annotations["default"]; !has {
						t.Errorf("FAIL default on %s dropped", name)
					}
				}
			}
			if len(expect.PropertiesVerbatim) > 0 {
				node := fixtureNode(t, m, expect.Definition)
				for _, name := range expect.PropertiesVerbatim {
					if _, ok := node.Properties[name]; !ok {
						t.Errorf("FAIL property %s renamed or dropped by the model", name)
					}
				}
			}
			if len(expect.AnnotationsKept) > 0 {
				node := fixtureNode(t, m, expect.Definition)
				for _, name := range expect.AnnotationsKept {
					if _, ok := node.Annotations[name]; !ok {
						t.Errorf("FAIL annotation %s dropped", name)
					}
				}
			}
			_ = fmt.Sprint // silence when no branches used
		})
	}
}

func fixtureNode(t *testing.T, m *Model, name string) *Node {
	t.Helper()
	if name != "" {
		node, ok := m.Definitions[name]
		if !ok {
			t.Fatalf("FAIL fixture binding: definition %q does not exist", name)
		}
		return node
	}
	if len(m.Definitions) != 1 {
		t.Fatal("FAIL fixture binding: multiple definitions require an explicit name")
	}
	for _, node := range m.Definitions {
		return node
	}
	return nil
}

func hasRecursiveRef(m *Model) bool {
	var descend func(n *Node) bool
	descend = func(n *Node) bool {
		if n.Items != nil {
			if _, isRef := n.Items.Annotations["$ref"]; isRef {
				return true
			}
			return descend(n.Items)
		}
		return false
	}
	for _, node := range m.Definitions {
		if _, isRef := node.Annotations["$ref"]; isRef {
			return true
		}
		for _, child := range node.Properties {
			if descend(child) {
				return true
			}
		}
		if descend(node) {
			return true
		}
	}
	return false
}

func containsSub(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
