package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
)

func TestRepairA14ProfileFlag(t *testing.T) {
	for name, command := range map[string]func([]string, *printer) error{"inspect": cmdInspect, "check": cmdCheck, "generate": cmdGenerate} {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			sentinel := filepath.Join(repo, "sentinel.txt")
			if err := os.WriteFile(sentinel, []byte("unrelated"), 0o644); err != nil {
				t.Fatal(err)
			}
			err := command([]string{"--repo", repo, "--manifest", "../escape.json"}, newPrinter("json"))
			var classified *profile.Error
			if !errors.As(err, &classified) || classified.Code != "unsafe-profile" {
				t.Errorf("FAIL CLI profile flag contract: %v", err)
			}
			body, err := os.ReadFile(sentinel)
			if err != nil || string(body) != "unrelated" {
				t.Error("FAIL rejected profile flag mutated source")
			}
			if _, err := os.Stat(filepath.Join(repo, "protocol")); !os.IsNotExist(err) {
				t.Error("FAIL rejected profile flag created outputs")
			}
		})
	}
}
