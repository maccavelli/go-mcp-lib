package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEmitProtocolIntoRepository emits the protocol package into the
// repository's protocol/ directory. It runs only when EMIT_PROTOCOL_DIR
// is set, because it mutates the tree; ordinary test runs never trigger
// it, and the committed bytes are validated by the repository gates.
func TestEmitProtocolIntoRepository(t *testing.T) {
	target := os.Getenv("EMIT_PROTOCOL_DIR")
	if target == "" {
		t.Skip("set EMIT_PROTOCOL_DIR to emit into the repository")
	}
	sdkDir := os.Getenv("TEST_SDK_DIR")
	if sdkDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		sdkDir = filepath.Join(
			home, "go", "pkg", "mod", "github.com",
			"modelcontextprotocol", "go-sdk@v1.8.0", "mcp",
		)
	}
	if _, err := os.Stat(sdkDir); err != nil {
		t.Skipf("pinned SDK source not present: %v", err)
	}
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
	t.Logf("emitted %d files into %s: %v", len(manifest.Files), target, manifest.Files)
}
