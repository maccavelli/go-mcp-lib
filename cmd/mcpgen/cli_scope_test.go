package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairA13CLIScope(t *testing.T) {
	repo := profileFixture(t)
	nonce := strings.Repeat("0", 32)
	if err := os.WriteFile(filepath.Join(repo, ".cli-fixture-capability"), []byte(nonce), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cliCapability(repo, nonce); err != nil {
		t.Fatalf("FAIL CLI fixture control: %v", err)
	}
	if err := cliCapability(repo, strings.Repeat("1", 32)); err == nil {
		t.Error("FAIL CLI fixture nonce guard accepted mismatch")
	}
	if err := os.Remove(filepath.Join(repo, ".cli-fixture-capability")); err != nil {
		t.Fatal(err)
	}
	if err := cliCapability(repo, nonce); err == nil {
		t.Error("FAIL CLI fixture nonce guard accepted absence")
	}
}
