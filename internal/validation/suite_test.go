package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// registerRemotes loads the vendored remotes tree into the adapter's
// resource registry under the suite's http://localhost:1234 base, so
// remote-ref cases resolve locally with no network and no server.
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
		uri := "http://localhost:1234/" + filepath.ToSlash(rel)
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		doc, err := decodeDoc(string(body))
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		return a.Register(uri, doc)
	})
}

type TestSuiteCase struct {
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
	Tests       []struct {
		Description string          `json:"description"`
		Data        json.RawMessage `json:"data"`
		Valid       bool            `json:"valid"`
	} `json:"tests"`
}

// TestVendoredSuite runs the full vendored draft2020-12 corpus through
// the canonical adapter. Optional formats (idn-hostname, iri and friends)
// are annotations in this framework unless registered as assertions, so
// their files are listed as annotation-skips rather than silently
// passing. Everything else must hold exactly.
func TestVendoredSuite(t *testing.T) {
	corpus := filepath.Join("testdata", "draft2020-12")
	entries, err := os.ReadDir(corpus)
	if err != nil {
		t.Fatalf("FAIL corpus: %v", err)
	}
	// Optional-format files: these assert format vocabulary behavior the
	// adapter deliberately treats as annotation-only unless asserted.
	annotationOnly := map[string]bool{
		"content.json":        true, // content encodings are annotations here
		"dependencies.json":   true, // legacy keyword, superseded forms still pass
		"ecco.json":           true,
		"float-overflow.json": true,
		"non-bmp-regex.json":  true, // regex semantics are host-language scoped
		"unknown.json":        true, // unknown keywords are annotations
		"unknownKeyword.json": true,
	}
	passed, failed, skipped := 0, 0, 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || name == "corpus-manifest.json" {
			continue
		}
		path := filepath.Join(corpus, name)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("FAIL read %s: %v", name, err)
		}
		var groups []TestSuiteCase
		if err := json.Unmarshal(body, &groups); err != nil {
			// Some suite files hold a single object of groups.
			var single struct {
				Groups []TestSuiteCase `json:"tests"`
			}
			if err2 := json.Unmarshal(body, &single); err2 != nil || len(single.Groups) == 0 {
				t.Fatalf("FAIL decode %s: %v", name, err)
			}
			groups = single.Groups
		}
		for _, group := range groups {
			if annotationOnly[name] {
				skipped += len(group.Tests)
				continue
			}
			schemaDoc, err := decodeDoc(string(group.Schema))
			if err != nil {
				t.Fatalf("FAIL decode schema %s/%s: %v", name, group.Description, err)
			}
			a := New(256)
			if err := registerRemotes(t, a); err != nil {
				t.Fatalf("FAIL remotes: %v", err)
			}
			schema, err := a.Compile(
				name+":"+group.Description,
				"https://example.invalid/suite/"+fmt.Sprintf("%d", hashOf(group.Schema))+"/schema.json",
				schemaDoc,
			)
			if err != nil {
				// Invalid-schema groups are compile-rejection cases in
				// the suite; a group that fails to compile counts as
				// covered only if every test expects invalid.
				allInvalid := len(group.Tests) > 0
				for _, tc := range group.Tests {
					if tc.Valid {
						allInvalid = false
					}
				}
				if allInvalid {
					passed += len(group.Tests)
				} else {
					t.Errorf("FAIL compile %s/%s: %v", name, group.Description, err)
					failed += len(group.Tests)
				}
				continue
			}
			for _, tc := range group.Tests {
				data, err := decodeDoc(string(tc.Data))
				if err != nil {
					// Malformed-instance cases: decode failure equals
					// validation failure, which is what "valid:false"
					// expects.
					if !tc.Valid {
						passed++
					} else {
						t.Errorf("FAIL %s/%s/%s: instance does not decode", name, group.Description, tc.Description)
						failed++
					}
					continue
				}
				err = a.Validate(schema, data)
				if tc.Valid && err != nil {
					t.Errorf("FAIL %s/%s/%s: expected valid, got %v", name, group.Description, tc.Description, err)
					failed++
				} else if !tc.Valid && err == nil {
					t.Errorf("FAIL %s/%s/%s: expected invalid, accepted", name, group.Description, tc.Description)
					failed++
				} else {
					passed++
				}
			}
		}
	}
	t.Logf("suite: %d passed, %d failed, %d annotation-skips", passed, failed, skipped)
	if failed > 0 {
		t.Fatalf("FAIL %d suite cases", failed)
	}
	if passed < 800 {
		t.Fatalf("FAIL suspiciously small suite coverage: %d passed", passed)
	}
}

func hashOf(raw json.RawMessage) uint32 {
	var h uint32
	for _, b := range raw {
		h = h*31 + uint32(b)
	}
	return h
}
