// Package testenv resolves prepared test inputs through Go module metadata.
package testenv

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// SDKDir locates the runtime module's exact SDK pin without network access.
// Missing preparation is a test failure, never a skipped required test.
func SDKDir(t testing.TB) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "tools", "versions.json")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("FAIL prepared SDK: repository root not found")
		}
		root = parent
	}
	body, err := os.ReadFile(filepath.Join(root, "tools", "versions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pins struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"runtime-dependencies"`
	}
	if err := json.Unmarshal(body, &pins); err != nil {
		t.Fatal(err)
	}
	const module = "github.com/modelcontextprotocol/go-sdk"
	command := exec.Command("go", "list", "-m", "-json", module)
	command.Dir = root
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off",
		"GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=readonly")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("FAIL prepared SDK metadata: %v: %s", err, output)
	}
	var metadata struct {
		Dir     string
		Version string
		Replace json.RawMessage
	}
	if err := json.Unmarshal(output, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Dir == "" || metadata.Version != pins.Dependencies[module].Version || len(metadata.Replace) != 0 {
		t.Fatal("FAIL prepared SDK: missing directory, wrong version or replacement")
	}
	directory := filepath.Join(metadata.Dir, "mcp")
	if info, err := os.Stat(directory); err != nil || !info.IsDir() {
		t.Fatalf("FAIL prepared SDK source unavailable: %v", err)
	}
	return directory
}
