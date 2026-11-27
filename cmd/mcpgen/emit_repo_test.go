package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/testenv"
)

// TestEmitProtocolIntoRepositoryScratch always exercises a repository-shaped
// output directory. Environment variables cannot redirect it into the source tree.
func TestEmitProtocolIntoRepositoryScratch(t *testing.T) {
	target := filepath.Join(t.TempDir(), "protocol")
	sdkDir := testenv.SDKDir(t)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest, err := SchemaToProtocol(
		"../../schema/inputs/2026-07-28/schema.json",
		sdkDir, "2026-07-28", target,
	)
	if err != nil {
		t.Fatalf("FAIL emit: %v", err)
	}
	if !reflect.DeepEqual(manifest.Files, []string{"aliases.go", "api.json", "bindings.json", "codecs.go", "constants.go", "coverage.json", "owned.go", "schema.json"}) {
		t.Fatalf("FAIL incomplete emitted inventory: %v", manifest.Files)
	}
	for _, name := range manifest.Files {
		if body, err := os.ReadFile(filepath.Join(target, name)); err != nil || len(body) == 0 {
			t.Fatalf("FAIL missing or empty emission %s: %v", name, err)
		}
	}
	t.Logf("emitted %d files into %s: %v", len(manifest.Files), target, manifest.Files)
}
