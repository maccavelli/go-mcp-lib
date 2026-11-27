package mcp

import (
	"encoding/json"
	"testing"
)

func TestRepairA16RuntimeValidation(t *testing.T) {
	for _, raw := range []string{`{"n":1,"n":2}`, `{"n":1} {}`, `{"n":1} "unfinished`, `"\ud800"`} {
		if _, err := decodeExact(raw); err == nil {
			t.Errorf("FAIL runtime exact decoder accepted malformed input %s", raw)
		}
	}
	value, err := decodeExact(`{"n":9007199254740993}`)
	if err != nil || value.(map[string]any)["n"] != json.Number("9007199254740993") {
		t.Errorf("FAIL runtime exact decoder rounded: %v %v", value, err)
	}
	for _, uri := range []string{"https://example.invalid/a b", "x:%", "https://[bad]/"} {
		if (&ResourceRegistration{Name: "fixture", URI: uri}).validate() == nil {
			t.Errorf("FAIL runtime URI checker diverged: %q", uri)
		}
	}
	if err := (&ResourceRegistration{Name: "fixture", URI: "x:valid"}).validate(); err != nil {
		t.Errorf("FAIL valid one-letter URI scheme rejected: %v", err)
	}
	for _, template := range []string{"https://example.invalid/{?}", "file:///x/{a b}", "file:///x/{a:bad}"} {
		if (&ResourceTemplateRegistration{Name: "fixture", URITemplate: template}).validate() == nil {
			t.Errorf("FAIL runtime RFC 6570 checker diverged: %q", template)
		}
	}
}
