package validation

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func registerRemotes(t *testing.T, a *Adapter) error {
	t.Helper()
	base := filepath.Join("testdata", "draft2020-12", "remotes")
	return filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		doc, err := decodeDoc(string(body))
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		return a.Register("http://localhost:1234/"+filepath.ToSlash(rel), doc)
	})
}

// TestSuiteCase is the independent upstream schema/instance/outcome fixture.
type TestSuiteCase struct {
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
	Tests       []struct {
		Description string          `json:"description"`
		Data        json.RawMessage `json:"data"`
		Valid       bool            `json:"valid"`
	} `json:"tests"`
}

type originalCase struct {
	File  string `json:"file"`
	Group string `json:"group"`
	Test  string `json:"test"`
	Valid bool   `json:"valid"`
}

func fixtureID(file, group, test string) string {
	return file + "\x00" + group + "\x00" + test
}

// TestVendoredSuite validates every applicable immutable fixture, with no
// compile-error-as-invalid-instance substitution or count-only coverage gate.
func TestVendoredSuite(t *testing.T) {
	base := filepath.Join("testdata", "draft2020-12")
	read := func(path string, out any) {
		t.Helper()
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, out); err != nil {
			t.Fatal(err)
		}
	}
	var manifest struct {
		Files map[string]string `json:"files"`
	}
	read(filepath.Join(base, "corpus-manifest.json"), &manifest)
	for _, filename := range sortedKeys(manifest.Files) {
		body, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(filename)))
		if err != nil {
			t.Errorf("FAIL fixture missing %s: %v", filename, err)
			continue
		}
		if fmt.Sprintf("%x", sha256.Sum256(body)) != manifest.Files[filename] {
			t.Errorf("FAIL fixture identity %s", filename)
		}
	}
	var original struct {
		Files map[string]struct {
			VendoredSHA string `json:"vendored-sha256"`
		} `json:"files"`
		Cases []originalCase `json:"cases"`
	}
	read(filepath.Join("testdata", "original-assertions.json"), &original)
	oracle := make(map[string]originalCase)
	for _, row := range original.Cases {
		id := fixtureID(row.File, row.Group, row.Test)
		if _, duplicate := oracle[id]; duplicate {
			t.Fatalf("FAIL duplicate original fixture identity %q", id)
		}
		oracle[id] = row
	}
	if len(oracle) != 421 || len(original.Files) != 14 {
		t.Fatal("FAIL original qualification inventory changed")
	}
	for filename, entry := range original.Files {
		if manifest.Files[filename] != entry.VendoredSHA {
			t.Errorf("FAIL reconciled fixture file changed: %s", filename)
		}
	}
	// Custom dialects may remove required standard vocabularies. This profile
	// intentionally requires the canonical complete Draft 2020-12 dialect.
	rejected := map[string]string{
		"vocabulary.json\x00schema that uses custom metaschema with with no validation vocabulary": "/$schema",
		"vocabulary.json\x00ignore unrecognized optional vocabulary":                               "/$schema",
	}
	seen := make(map[string]bool)
	validated, excluded := 0, 0
	for _, filename := range sortedKeys(manifest.Files) {
		if !strings.HasSuffix(filename, ".json") || strings.Contains(filename, "/") {
			continue
		}
		var groups []TestSuiteCase
		read(filepath.Join(base, filename), &groups)
		for groupIndex, group := range groups {
			t.Run(filename+"/"+group.Description, func(t *testing.T) {
				a := New(256)
				if err := registerRemotes(t, a); err != nil {
					t.Fatal(err)
				}
				doc := mustDoc(t, string(group.Schema))
				compiled, err := a.Compile(filename+":"+group.Description,
					fmt.Sprintf("https://example.invalid/suite/%s/%d/schema.json", filename, groupIndex), doc)
				if pointer, outOfProfile := rejected[filename+"\x00"+group.Description]; outOfProfile {
					if err == nil || !strings.Contains(err.Error(), pointer) {
						t.Errorf("FAIL explicit out-of-profile dialect not rejected: %v", err)
					}
					excluded += len(group.Tests)
					t.Logf("profile rejection; %d upstream instance outcomes nonapplicable (custom dialect)", len(group.Tests))
					return
				}
				if err != nil {
					t.Fatalf("FAIL compile fixture (no instance outcome claimed): %v", err)
				}
				for _, tc := range group.Tests {
					t.Run(tc.Description, func(t *testing.T) {
						id := fixtureID(filename, group.Description, tc.Description)
						if row, required := oracle[id]; required {
							if row.Valid != tc.Valid {
								t.Error("FAIL original independent outcome changed")
							}
							seen[id] = true
						}
						value := mustDoc(t, string(tc.Data))
						if validationErr := a.Validate(compiled, value); (validationErr == nil) != tc.Valid {
							t.Errorf("FAIL instance: expected valid=%v, error=%v", tc.Valid, validationErr)
						}
						validated++
					})
				}
			})
		}
	}
	for id := range oracle {
		if !seen[id] {
			t.Errorf("FAIL original fixture unexecuted: %q", id)
		}
	}
	t.Logf("original assertion identities executed=%d; total instance outcomes=%d; explicit custom-dialect exclusions=%d", len(seen), validated, excluded)
}
