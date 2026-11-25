package sdkmap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/testenv"
)

func pinnedDefs(t *testing.T) []string {
	t.Helper()
	body, err := os.ReadFile("../../../../schema/inputs/2026-07-28/schema.json")
	if err != nil {
		t.Fatalf("FAIL pinned schema: %v", err)
	}
	var schema struct {
		Defs map[string]json.RawMessage `json:"$defs"`
	}
	if err := json.Unmarshal(body, &schema); err != nil {
		t.Fatalf("FAIL schema decode: %v", err)
	}
	defs := make([]string, 0, len(schema.Defs))
	for name := range schema.Defs {
		defs = append(defs, name)
	}
	if len(defs) != 155 {
		t.Fatalf("FAIL definition count: %d", len(defs))
	}
	return defs
}

func sdkDir(t *testing.T) string {
	t.Helper()
	return testenv.SDKDir(t)
}

func TestBuildTotalDispositions(t *testing.T) {
	dir := sdkDir(t)
	result, err := Build("2026-07-28", dir, "v1.8.0", pinnedDefs(t))
	if err != nil {
		t.Fatalf("FAIL build: %v", err)
	}
	if len(result.Dispositions) != 155 {
		t.Fatalf("FAIL totality: %d dispositions", len(result.Dispositions))
	}
	// Deprecated and sealed definitions must never alias.
	for _, name := range append(Deprecations, Sealed...) {
		if result.Dispositions[name] == "alias" {
			t.Fatalf("FAIL %s aliased a deprecated/sealed SDK type", name)
		}
	}
	counts := make(map[string]int)
	for _, d := range result.Dispositions {
		counts[d]++
	}
	t.Logf("dispositions: %v", counts)
	for _, required := range []string{"alias", "owned", "owned-codec"} {
		if counts[required] == 0 {
			t.Fatalf("FAIL disposition class %s never used; classification is broken", required)
		}
	}
}

func TestBuildFailsOnMissingDefinition(t *testing.T) {
	if _, err := Build("2026-07-28", sdkDir(t), "v1.8.0", nil); err == nil {
		t.Fatal("FAIL empty definitions accepted")
	}
}

func TestWriteJSONDeterministic(t *testing.T) {
	dir := sdkDir(t)
	result, err := Build("2026-07-28", dir, "v1.8.0", pinnedDefs(t))
	if err != nil {
		t.Fatalf("FAIL build: %v", err)
	}
	first := filepath.Join(t.TempDir(), "a.json")
	second := filepath.Join(t.TempDir(), "b.json")
	if err := result.WriteJSON(first); err != nil {
		t.Fatal(err)
	}
	if err := result.WriteJSON(second); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(first)
	b, _ := os.ReadFile(second)
	if string(a) != string(b) {
		t.Fatal("FAIL nondeterministic serialization")
	}
}

func TestKnownDispositions(t *testing.T) {
	dir := sdkDir(t)
	result, err := Build("2026-07-28", dir, "v1.8.0", pinnedDefs(t))
	if err != nil {
		t.Fatalf("FAIL build: %v", err)
	}
	want := map[string]string{
		// SEP-2577 surfaces: owned DTOs, codec conversion only.
		"CreateMessageResult": "owned-codec",
		"ListRootsResult":     "owned-codec",
		// Clean public types alias directly.
		"Tool":               "alias",
		"TextContent":        "alias",
		"Implementation":     "alias",
		"CallToolResult":     "alias",
		"ClientCapabilities": "alias",
	}
	for name, disposition := range want {
		if result.Dispositions[name] != disposition {
			t.Errorf("FAIL %s: got %q want %q", name, result.Dispositions[name], disposition)
		}
	}
}
