package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepairA14ProfileSelection(t *testing.T) {
	repo := t.TempDir()
	for name := range requiredFiles {
		body, err := os.ReadFile(filepath.Join("../../../..", name))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	body, err := os.ReadFile("../../../../schema/generation-profile.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "reviewed.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if inputs, err := LoadNamed(repo, Revision, "reviewed.json"); err != nil || len(inputs.Mapping.Dispositions) != 155 {
		t.Fatalf("FAIL named profile selection: %v", err)
	}
	for _, name := range []string{"../escape.json", "/absolute.json", "a/../reviewed.json", "a%2freviewed.json", "a\\reviewed.json", "missing.json"} {
		if _, err := LoadNamed(repo, Revision, name); err == nil {
			t.Errorf("FAIL unsafe/absent profile accepted: %s", name)
		}
	}
	if err := os.Symlink("reviewed.json", filepath.Join(repo, "alias.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNamed(repo, Revision, "alias.json"); err == nil {
		t.Error("FAIL aliased profile accepted")
	}
	if err := os.WriteFile(filepath.Join(repo, "reviewed.json"), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNamed(repo, Revision, "reviewed.json"); err == nil {
		t.Error("FAIL selected profile pin bypassed")
	}
}
