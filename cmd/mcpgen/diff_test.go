package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
)

func contractFixture(t *testing.T, changed bool) string {
	t.Helper()
	root := t.TempDir()
	values := map[string]string{"schema.json": `{"type":"object","maxProperties":10}`, "api.json": `{"declarations":[{"name":"Thing","signature":"type Thing string"}]}`, "coverage.json": `{"methods":["tools/call"],"capabilities":["tools"],"nodes":[{"keywords":{"type":"assertion"}}]}`, "bindings.json": `{"Thing":"owned"}`}
	if changed {
		values["schema.json"] = `{"type":"object","maxProperties":2}`
		values["api.json"] = `{"declarations":[]}`
		values["coverage.json"] = `{"methods":[],"capabilities":[],"nodes":[{"keywords":{"type":"assertion","pattern":"assertion"}}]}`
		values["bindings.json"] = `{"Thing":"owned-codec"}`
	}
	manifest := ownership.Manifest{Version: 2, Revision: "fixture", SHA256: map[string]string{}}
	for _, name := range ownership.Sorted(values) {
		body := []byte(values[name])
		manifest.Files = append(manifest.Files, name)
		manifest.SHA256[name] = ownership.Digest(body)
		if err := os.WriteFile(filepath.Join(root, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, ownership.ManifestName)
	if err := os.WriteFile(file, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestRepairA14SemanticDiff(t *testing.T) {
	before, after := contractFixture(t, false), contractFixture(t, true)
	changes, err := diffManifests(before, after)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, change := range changes.Semantic {
		seen[change.Scope+change.Pointer] = true
		if change.Impact == "" {
			t.Error("FAIL unclassified semantic change")
		}
	}
	for _, key := range []string{"schema.json#/maxProperties", "api.json#/declarations", "coverage.json#/methods", "coverage.json#/capabilities", "coverage.json#/nodes", "bindings.json#/Thing"} {
		if !seen[key] {
			t.Errorf("FAIL semantic diff omitted %s", key)
		}
	}
	if len(changes.Added) != 0 || len(changes.Removed) != 0 || len(changes.Changed) == 0 {
		t.Error("FAIL same-inventory changes lost")
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(after), "schema.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := diffManifests(before, after); err == nil {
		t.Error("FAIL changed sidecar accepted")
	}
}
