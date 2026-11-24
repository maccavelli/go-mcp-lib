package main

import (
	"reflect"
	"testing"
)

// TestStageNames pins the generator's stage vocabulary: the nine stages of
// the framework plan, and no others.
func TestStageNames(t *testing.T) {
	got := stageNames()
	want := []string{
		"acquire", "diagnostic", "emit", "load", "model",
		"publish", "qualify", "resolve", "sdkmap",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("stageNames: got %v want %v", got, want)
	}
}
