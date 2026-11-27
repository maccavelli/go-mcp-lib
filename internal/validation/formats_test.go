package validation

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestFormatFixtures distinguishes annotations from mandatory protocol syntax.
// Every vendored instance is tested through the production adapter. The original
// 36 URI/template cases additionally exercise the production semantic checkers.
func TestFormatFixtures(t *testing.T) {
	base := filepath.Join("testdata", "formats")
	manifestBody, err := os.ReadFile(filepath.Join(base, "formats-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Formats map[string]struct {
			Cases int    `json:"cases"`
			Hash  string `json:"hash"`
		} `json:"formats"`
	}
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatal(err)
	}
	assertions, annotations := 0, 0
	for _, format := range sortedKeys(manifest.Formats) {
		entry := manifest.Formats[format]
		if entry.Cases == 0 {
			continue // license, not an instance fixture
		}
		body, err := os.ReadFile(filepath.Join(base, format+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(body)) != entry.Hash {
			t.Errorf("FAIL format fixture identity %s", format)
			continue
		}
		var groups []TestSuiteCase
		if err := json.Unmarshal(body, &groups); err != nil {
			t.Fatal(err)
		}
		count := 0
		for i, group := range groups {
			a := New(4)
			schema := compileOK(t, a, fmt.Sprintf("format-%s-%d", format, i), string(group.Schema))
			for _, tc := range group.Tests {
				count++
				t.Run(format+"/"+group.Description+"/"+tc.Description, func(t *testing.T) {
					instance := mustDoc(t, string(tc.Data))
					if err := a.Validate(schema, instance); err != nil {
						t.Errorf("FAIL generic format annotation asserted: %v", err)
					}
					check := map[string]func(string) error{"uri": CheckURI, "uri-template": CheckURITemplate}[format]
					if check == nil {
						annotations++
						return
					}
					assertions++
					text, isString := instance.(string)
					valid := !isString || check(text) == nil
					if valid != tc.Valid {
						t.Errorf("FAIL production %s syntax: valid=%v want=%v", format, valid, tc.Valid)
					}
				})
			}
		}
		if count != entry.Cases {
			t.Errorf("FAIL format case identity %s: %d want %d", format, count, entry.Cases)
		}
	}
	if assertions != 36 {
		t.Errorf("FAIL original format reconciliation: %d want 36", assertions)
	}
	t.Logf("original mandatory syntax fixtures=%d; additional annotation-only fixtures=%d", assertions, annotations)
}
