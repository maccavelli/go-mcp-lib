package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/testenv"
)

// TestEmitProtocolIntoScratch emits the full protocol package into a
// scratch tree and proves the generated files exist with manifest
// ownership. Generation is deterministic, so the committed protocol/
// bytes in the repository are this same emission.
func TestEmitProtocolIntoScratch(t *testing.T) {
	sdkDir := testenv.SDKDir(t)
	out := t.TempDir()
	manifest, err := SchemaToProtocol(
		"../../schema/inputs/2026-07-28/schema.json",
		sdkDir, "2026-07-28", out,
	)
	if err != nil {
		t.Fatalf("FAIL emit: %v", err)
	}
	if len(manifest.Files) != 3 {
		t.Fatalf("FAIL manifest files: %v", manifest.Files)
	}
	for _, name := range manifest.Files {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("FAIL %s not written", name)
		}
	}
	// Deterministic: a second emission is byte-identical.
	again := t.TempDir()
	secondManifest, err := SchemaToProtocol(
		"../../schema/inputs/2026-07-28/schema.json",
		sdkDir, "2026-07-28", again,
	)
	if err != nil {
		t.Fatalf("FAIL second emit: %v", err)
	}
	for _, name := range manifest.Files {
		a, _ := os.ReadFile(filepath.Join(out, name))
		b, _ := os.ReadFile(filepath.Join(again, name))
		if string(a) != string(b) {
			t.Fatalf("FAIL nondeterministic emission of %s", name)
		}
	}
	if secondManifest.Revision != manifest.Revision {
		t.Fatal("FAIL manifest revision drift")
	}
}
