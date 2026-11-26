package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

func profileFixture(t *testing.T) string {
	t.Helper()
	source, err := sdkmap.RuntimeModule(".")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, name := range []string{
		"go.mod", "go.sum", "tools/versions.json", "schema/generation-profile.json",
		"schema/inputs/2026-07-28/schema.json", "schema/inputs/2026-07-28/manifest.json",
		"schema/sdk-map.json", "schema/semantics.json",
	} {
		body, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestRepairA08ProvenanceAndMapping(t *testing.T) {
	if _, err := profile.Load(profileFixture(t), profile.Revision); err != nil {
		t.Fatalf("FAIL profile control: %v", err)
	}
	for _, name := range []string{"missing-profile", "missing-sdk-map", "missing-semantics", "altered-schema", "forged-profile", "symlink-escape"} {
		t.Run(name, func(t *testing.T) {
			root := profileFixture(t)
			mapPath := filepath.Join(root, "schema/sdk-map.json")
			switch name {
			case "missing-profile", "missing-sdk-map", "missing-semantics":
				path := map[string]string{"missing-profile": "schema/generation-profile.json", "missing-sdk-map": "schema/sdk-map.json", "missing-semantics": "schema/semantics.json"}[name]
				if err := os.Remove(filepath.Join(root, path)); err != nil {
					t.Fatal(err)
				}
			case "altered-schema":
				path := filepath.Join(root, profile.SchemaFile)
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(body, ' '), 0o600); err != nil {
					t.Fatal(err)
				}
			case "forged-profile":
				path := filepath.Join(root, "schema/generation-profile.json")
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var manifest map[string]any
				if err := json.Unmarshal(body, &manifest); err != nil {
					t.Fatal(err)
				}
				mapBytes, err := os.ReadFile(mapPath)
				if err != nil {
					t.Fatal(err)
				}
				mapBytes = append(mapBytes, ' ')
				if err := os.WriteFile(mapPath, mapBytes, 0o600); err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(mapBytes)
				identity := manifest["files"].(map[string]any)["schema/sdk-map.json"].(map[string]any)
				identity["sha256"], identity["bytes"] = hex.EncodeToString(sum[:]), len(mapBytes)
				body, err = json.Marshal(manifest)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
			case "symlink-escape":
				body, err := os.ReadFile(mapPath)
				if err != nil {
					t.Fatal(err)
				}
				outside := filepath.Join(t.TempDir(), "outside.json")
				if err := os.WriteFile(outside, body, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(mapPath); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, mapPath); err != nil {
					t.Fatalf("FAIL required symlink fixture unavailable: %v", err)
				}
			}
			output := filepath.Join(root, "candidate")
			if err := os.Mkdir(output, 0o700); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(output, "handwritten.txt")
			if err := os.WriteFile(marker, []byte("preserve"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := SchemaToProtocol(filepath.Join(root, profile.SchemaFile), "unprepared SDK must not be reached", profile.Revision, output)
			var diagnostic *profile.Error
			if !errors.As(err, &diagnostic) {
				t.Errorf("FAIL profile bypass reached SDK or output: %s: %v", name, err)
			}
			if _, err := checkTree(root); !errors.As(err, &diagnostic) {
				t.Errorf("FAIL first generation bypassed reviewed inputs: %s: %v", name, err)
			}
			entries, err := os.ReadDir(output)
			if err != nil || len(entries) != 1 {
				t.Errorf("FAIL profile rejection changed destination inventory: %v", err)
			}
			body, err := os.ReadFile(marker)
			if err != nil || string(body) != "preserve" {
				t.Errorf("FAIL profile rejection changed handwritten bytes: %v", err)
			}
		})
	}
}
