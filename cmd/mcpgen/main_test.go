package main

import (
	"reflect"
	"testing"
)

// TestCommandNames pins the CLI command vocabulary: the six stage-4
// commands plus diagnostic, and no others.
func TestCommandNames(t *testing.T) {
	got := commandNames()
	want := []string{
		"check", "diagnostic", "diff", "generate",
		"inspect", "recover", "update",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("commandNames: got %v want %v", got, want)
	}
}

// TestExitCodesAreContract pins the exit-code contract.
func TestExitCodesAreContract(t *testing.T) {
	if exitOK != 0 || exitFail != 1 || exitUsage != 2 {
		t.Fatalf("exit codes drifted: ok=%d fail=%d usage=%d", exitOK, exitFail, exitUsage)
	}
}
