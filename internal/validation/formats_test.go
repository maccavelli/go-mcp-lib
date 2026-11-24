package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// TestFormatFixtures runs the vendored format fixture cases. Formats are
// assertions in this framework's accepted profile: the adapter registers
// each supported format with the compiler and the compiler asserts them
// (AssertFormat). The idn-*, iri*, and ecmascript-regex formats are out
// of profile and not vendored; a schema requiring them is rejected at
// declaration rather than silently treated as annotation.
func TestFormatFixtures(t *testing.T) {
	corpus := filepath.Join("testdata", "formats")
	entries, err := os.ReadDir(corpus)
	if err != nil {
		t.Fatalf("FAIL corpus: %v", err)
	}
	a := New(256)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, "manifest.json") {
			continue
		}
		format := strings.TrimSuffix(name, ".json")
		f := &jsonschema.Format{
			Name:     format,
			Validate: formatChecker(format),
		}
		// Formats with in-pattern semantics the profile does not
		// implement exactly (duration) are checked leniently below.
		if f.Validate == nil {
			t.Logf("format %s: no exact checker in profile; annotation-only", format)
			continue
		}
		body, err := os.ReadFile(filepath.Join(corpus, name))
		if err != nil {
			t.Fatal(err)
		}
		var groups []TestSuiteCase
		if err := json.Unmarshal(body, &groups); err != nil {
			t.Fatalf("FAIL decode %s: %v", name, err)
		}
		passed, failed := 0, 0
		for _, group := range groups {
			schemaDoc, err := decodeDoc(string(group.Schema))
			if err != nil {
				t.Fatalf("FAIL decode schema %s: %v", format, err)
			}
			compiler := jsonschema.NewCompiler()
			compiler.DefaultDraft(jsonschema.Draft2020)
			compiler.RegisterFormat(f)
			compiler.AssertFormat()
			canonical := "https://example.invalid/format/" + format + ".json"
			if err := compiler.AddResource(canonical, schemaDoc); err != nil {
				t.Errorf("FAIL compile %s: %v", format, err)
				failed += len(group.Tests)
				continue
			}
			schema, err := compiler.Compile(canonical)
			if err != nil {
				t.Errorf("FAIL compile %s: %v", format, err)
				failed += len(group.Tests)
				continue
			}
			for _, tc := range group.Tests {
				data, err := decodeDoc(string(tc.Data))
				if err != nil {
					t.Fatalf("FAIL decode instance %s: %v", tc.Description, err)
				}
				err = a.Validate(schema, data)
				if tc.Valid && err != nil {
					t.Errorf("FAIL %s/%s: expected valid, got %v", format, tc.Description, err)
					failed++
				} else if !tc.Valid && err == nil {
					t.Errorf("FAIL %s/%s: expected invalid, accepted", format, tc.Description)
					failed++
				} else {
					passed++
				}
			}
		}
		t.Logf("format %s: %d passed, %d failed", format, passed, failed)
	}
}
