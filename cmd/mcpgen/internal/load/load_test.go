package load

import (
	"strings"
	"testing"
)

func loadOrDie(t *testing.T, source, body string) *Document {
	t.Helper()
	doc, err := New().LoadBytes(source, []byte(body))
	if err != nil {
		t.Fatalf("FAIL load %s: %v", source, err)
	}
	return doc
}

func TestLoadPinnedSchemaShape(t *testing.T) {
	if testing.Short() {
		t.Skip("pinned schema file lives in the repository root")
	}
	doc, err := New().LoadFile("../../../../schema/inputs/2026-07-28/schema.json")
	if err != nil {
		t.Fatalf("FAIL pinned schema: %v", err)
	}
	if doc.Definitions != 155 {
		t.Errorf("FAIL definitions: got %d want 155", doc.Definitions)
	}
}

func TestDuplicateMemberRejected(t *testing.T) {
	_, err := New().LoadBytes("dup", []byte(`{"a":1,"a":2}`))
	if err == nil {
		t.Fatal("FAIL duplicate member accepted")
	}
	if d, ok := err.(Diagnostic); !ok || d.Code != CodeDuplicateMember {
		t.Fatalf("FAIL diagnostic: %v", err)
	}
}

func TestExactNumberTokens(t *testing.T) {
	doc := loadOrDie(t, "nums", `{"n":1.100,"z":0.30000000000000004}`)
	obj := doc.Root.(map[string]any)
	n, okN := obj["n"].(Number)
	z, okZ := obj["z"].(Number)
	if !okN || !okZ || n.Token != "1.100" || z.Token != "0.30000000000000004" {
		t.Fatalf("FAIL exact tokens: %v %v", obj["n"], obj["z"])
	}
}

func TestInvalidUTF8Rejected(t *testing.T) {
	_, err := New().LoadBytes("bad", append([]byte(`{"a":"`), 0xff, 0xfe, '"', '}'))
	if d, ok := err.(Diagnostic); !ok || d.Code != CodeInvalidUTF8 {
		t.Fatalf("FAIL utf8: %v", err)
	}
}

func TestDocumentLimit(t *testing.T) {
	loader := New()
	loader.Limits.MaxDocumentBytes = 8
	_, err := loader.LoadBytes("big", []byte(`{"pad":"1234567890"}`))
	if d, ok := err.(Diagnostic); !ok || d.Code != CodeDocumentTooLarge {
		t.Fatalf("FAIL limit: %v", err)
	}
}

func TestDefinitionsLimit(t *testing.T) {
	loader := New()
	loader.Limits.MaxDefinitions = 1
	_, err := loader.LoadBytes("defs", []byte(`{"$defs":{"a":{},"b":{}}}`))
	if d, ok := err.(Diagnostic); !ok || d.Code != CodeTooManyDefinitions {
		t.Fatalf("FAIL defs limit: %v", err)
	}
}

func TestParseFailuresAreStable(t *testing.T) {
	for _, body := range []string{``, `{`, `{"a"}`, `[1,]`, `tru`, `{"a":+}`, `nul`} {
		_, err := New().LoadBytes("malformed", []byte(body))
		if err == nil {
			t.Errorf("FAIL malformed accepted: %q", body)
			continue
		}
		if _, ok := err.(Diagnostic); !ok {
			t.Errorf("FAIL non-diagnostic error for %q: %v", body, err)
		}
	}
}

func TestRootMustBeObject(t *testing.T) {
	if _, err := New().LoadBytes("arr", []byte(`[1,2]`)); err == nil {
		t.Fatal("FAIL array root accepted")
	}
}

func TestDiagnosticsBounded(t *testing.T) {
	long := strings.Repeat("x", 600)
	d := Diagnostic{Phase: "load", Message: long}
	if len(d.Error()) > 512 {
		t.Fatalf("FAIL unbounded message: %d", len(d.Error()))
	}
}
