package ownership

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRepairA12Names(t *testing.T) {
	for _, name := range []string{"../escape.go", "/absolute", "a/b", "a\\b", "a%2fb", "CON", "nul.txt", "COM1.go", "trailing.", ".hidden", "a:b", ""} {
		if err := Name(name); err == nil {
			t.Errorf("FAIL unsafe output name accepted: %q", name)
		}
	}
	for _, name := range []string{"owned.go", "api.json", "fixture_test.go"} {
		if err := Name(name); err != nil {
			t.Errorf("FAIL portable output rejected: %s %v", name, err)
		}
	}
}

func TestRepairA12Inventories(t *testing.T) {
	valid := Manifest{Version: 2, Revision: "fixture", Files: []string{"a.go"}, SHA256: map[string]string{"a.go": Digest([]byte("a"))}}
	for _, name := range []string{"valid", "hashless", "unhashed", "case-fold", "traversal", "unsorted", "wrong-version"} {
		t.Run(name, func(t *testing.T) {
			manifest := valid
			manifest.SHA256 = map[string]string{"a.go": Digest([]byte("a"))}
			switch name {
			case "hashless":
				manifest.Version = 0
				manifest.SHA256 = nil
			case "unhashed":
				manifest.SHA256["a.go"] = "unknown"
			case "case-fold":
				manifest.Files = []string{"A.go", "a.go"}
				manifest.SHA256["A.go"] = manifest.SHA256["a.go"]
			case "traversal":
				manifest.Files = []string{"../a.go"}
				manifest.SHA256 = map[string]string{"../a.go": Digest([]byte("a"))}
			case "unsorted":
				manifest.Files = []string{"z.go", "a.go"}
				manifest.SHA256["z.go"] = Digest([]byte("z"))
			case "wrong-version":
				manifest.Version = 3
			}
			body, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			_, legacy, err := Parse(body)
			if (err == nil) != (name == "valid") || legacy {
				t.Errorf("FAIL ownership manifest case %s: legacy=%t error=%v", name, legacy, err)
			}
		})
	}
	root := t.TempDir()
	manifest, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{ManifestName: manifest, "a.go": []byte("a")} {
		if err := os.WriteFile(filepath.Join(root, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	}()
	if _, _, err := Existing(r, ""); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("manual"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Existing(r, ""); err == nil {
		t.Error("FAIL changed previous output accepted")
	}
}
