package validation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRepairA16ValidationProfile(t *testing.T) {
	t.Run("content-bound-cache", func(t *testing.T) {
		a := New(8)
		first := compileOK(t, a, "reused", `{"type":"string"}`)
		second := compileOK(t, a, "reused", `{"type":"integer"}`)
		if first == second || second.Validate("x") == nil || first.Validate("x") != nil {
			t.Error("FAIL changed schema reused stale compilation")
		}
	})
	t.Run("immutable-schema-and-registry", func(t *testing.T) {
		a := New(8)
		doc := map[string]any{"type": "string"}
		if err := a.Register("https://fixture.invalid/remote", doc); err != nil {
			t.Fatal(err)
		}
		doc["type"] = "integer"
		compiled := compileOK(t, a, "reference", `{"$ref":"https://fixture.invalid/remote"}`)
		if compiled.Validate("x") != nil || compiled.Validate(json.Number("1")) == nil {
			t.Error("FAIL registered schema retained mutable caller input")
		}
		local, err := a.Compile("local", "https://fixture.invalid/local", doc)
		if err != nil {
			t.Fatal(err)
		}
		doc["type"] = "null"
		if local.Validate(json.Number("1")) != nil || local.Validate(nil) == nil {
			t.Error("FAIL compiled schema retained mutable caller input")
		}
		// Adding a resource changes the immutable registry/cache identity.
		if err := a.Register("https://fixture.invalid/other", true); err != nil {
			t.Fatal(err)
		}
		again := compileOK(t, a, "reference", `{"$ref":"https://fixture.invalid/remote"}`)
		if again == compiled {
			t.Error("FAIL registry identity absent from cache key")
		}
	})
	t.Run("no-external-io", func(t *testing.T) {
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			_, _ = w.Write([]byte(`{"type":"string"}`))
		}))
		defer server.Close()
		path := filepath.Join(t.TempDir(), "schema.json")
		if err := os.WriteFile(path, []byte(`{"type":"string"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, ref := range []string{server.URL + "/schema", "file:///" + strings.TrimPrefix(filepath.ToSlash(path), "/")} {
			a := New(4)
			_, err := a.Compile("external", "https://fixture.invalid/root", map[string]any{"$ref": ref})
			if err == nil || !strings.Contains(err.Error(), "external loading is prohibited") {
				t.Errorf("FAIL external reference was not refused: %v", err)
			}
		}
		if requests.Load() != 0 {
			t.Error("FAIL validator performed external network I/O")
		}
	})
	t.Run("draft-vocabulary-and-pointer", func(t *testing.T) {
		for _, tc := range []struct{ body, pointer string }{
			{`{"$schema":"http://json-schema.org/draft-07/schema#"}`, "/$schema"},
			{`{"properties":{"x":{"$schema":"http://json-schema.org/draft-07/schema#"}}}`, "/properties/x/$schema"},
			{`{"$vocabulary":{"https://fixture.invalid/unqualified":true}}`, "/$vocabulary/https:~1~1fixture.invalid~1unqualified"},
			{`{"$vocabulary":{"core":true}}`, "/$vocabulary/core"},
			{`{"properties":{"x":{"pattern":"(?=a)"}}}`, "/properties/x/pattern"},
			{`{"patternProperties":{"(a)\\1":{}}}`, "/patternProperties/"},
			{`{"pattern":"(?i)a"}`, "/pattern"},
			{`{"pattern":"\\p{L}"}`, "/pattern"},
			{`{"pattern":"[[:alpha:]]"}`, "/pattern"},
			{`{"pattern":"\\x{41}"}`, "/pattern"},
			{`{"hidden":{"pattern":"(?=a)"},"$ref":"#/hidden"}`, "/hidden/pattern"},
			{`{"a/b~c":{"pattern":"(?=a)"},"$ref":"#/a~1b~0c"}`, "/a~1b~0c/pattern"},
		} {
			a := New(4)
			_, err := a.Compile("profile", "https://fixture.invalid/profile", mustDoc(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.pointer) {
				t.Errorf("FAIL unqualified profile declaration %s: %v", tc.body, err)
			}
		}
		a := New(8)
		annotation := compileOK(t, a, "annotation", `{"type":"string","format":"uri","$vocabulary":{"https://fixture.invalid/optional":false}}`)
		if err := annotation.Validate("not a URI"); err != nil {
			t.Errorf("FAIL application format annotation became assertion: %v", err)
		}
		instanceKeywords := compileOK(t, a, "instance-keywords", `{"const":{"$schema":"anything","pattern":"(?=a)"}}`)
		if err := instanceKeywords.Validate(map[string]any{"$schema": "anything", "pattern": "(?=a)"}); err != nil {
			t.Errorf("FAIL instance-valued keyword misidentified as schema: %v", err)
		}
	})
	t.Run("qualified-pattern-semantics", func(t *testing.T) {
		a := New(16)
		for _, tc := range []struct{ pattern, good, bad string }{
			{`^a+$`, "aaa", "aaa\n"}, {`^f.o$`, "foo", "f\ro"},
			{`^f.o$`, "féo", "f\u2028o"}, {`^f.o$`, "f😀o", "f\u2029o"},
			{`^[a-z]+$`, "abc", "ABC"}, {`^á`, "ármányos", "élmény"},
		} {
			schema := compileOK(t, a, fmt.Sprintf("pattern-%d", len(a.compiled)), fmt.Sprintf(`{"pattern":%q}`, tc.pattern))
			if schema.Validate(tc.good) != nil || schema.Validate(tc.bad) == nil {
				t.Errorf("FAIL qualified pattern %q: good=%q bad=%q", tc.pattern, tc.good, tc.bad)
			}
		}
	})
	t.Run("concurrent-compile", func(t *testing.T) {
		a := New(1)
		const count = 24
		var wait sync.WaitGroup
		results := make(chan *Compiled, count)
		for range count {
			wait.Go(func() {
				compiled, err := a.Compile("shared", "https://fixture.invalid/shared", map[string]any{"type": "string"})
				if err != nil {
					t.Errorf("FAIL concurrent first compile: %v", err)
				}
				results <- compiled
			})
		}
		wait.Wait()
		close(results)
		var first *Compiled
		for current := range results {
			if first == nil {
				first = current
			}
			if current == nil || current != first || current.Validate("ok") != nil {
				t.Error("FAIL concurrent compilation did not reuse immutable schema")
			}
		}
		if len(a.compiled) != 1 || len(a.hits) != 1 {
			t.Error("FAIL concurrent cache retention")
		}
	})
	t.Run("fail-closed-and-retention", func(t *testing.T) {
		if (*Compiled)(nil).Validate(nil) == nil || new(Compiled).Validate(nil) == nil {
			t.Error("FAIL missing compiled handle accepted")
		}
		a := New(1)
		if err := a.Register("https://fixture.invalid/one", true); err != nil {
			t.Fatal(err)
		}
		if a.Register("https://fixture.invalid/two", true) == nil {
			t.Error("FAIL resource registry retention unbounded")
		}
	})
}

func TestRepairA16RequiredAssertion(t *testing.T) {
	a := New(4)
	compiled := compileOK(t, a, "required-assertion", `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`)
	if compiled.Validate(map[string]any{"name": "present"}) != nil {
		t.Error("FAIL required assertion valid control")
	}
	if compiled.Validate(map[string]any{}) == nil {
		t.Error("FAIL required assertion removed")
	}
}

func TestRepairA16ExactNeighbors(t *testing.T) {
	for _, raw := range []string{
		`{"const":9007199254740993}`, `{"enum":[9007199254740993]}`,
		`{"type":"integer","minimum":9007199254740993,"maximum":9007199254740993}`,
	} {
		a := New(4)
		compiled := compileOK(t, a, "neighbor", raw)
		for _, tc := range []struct {
			body  string
			valid bool
		}{{"9007199254740993", true}, {"9007199254740992", false}, {"9007199254740994", false}} {
			instance := mustDoc(t, tc.body)
			if (compiled.Validate(instance) == nil) != tc.valid {
				t.Errorf("FAIL exact adjacent integer %s for %s", tc.body, raw)
			}
		}
	}
}

func TestRepairA17Locations(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "schema", "inputs", "2026-07-28", "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	a := New(32)
	const canonical = "https://fixture.invalid/mcp.json"
	if err := a.RegisterProtocol(canonical, body); err != nil {
		t.Fatal(err)
	}
	if New(4).RegisterProtocol(canonical, []byte(`{}`)) == nil {
		t.Error("FAIL unreviewed protocol document activated assertions")
	}
	for _, tc := range []struct{ pointer, good, bad string }{
		{"/$defs/Root/properties/uri", "file:///workspace", "https://example.invalid/workspace"},
		{"/$defs/Resource/properties/uri", "urn:example:a", "https://example.invalid/a b"},
		{"/$defs/ResourceTemplate/properties/uriTemplate", "https://example.invalid/{+path}{?q,limit}", "https://example.invalid/{?}"},
		{"/$defs/ImageContent/properties/data", "QQ==", "QR=="},
	} {
		compiled := compileOK(t, a, fmt.Sprintf("location-%d", len(a.compiled)), fmt.Sprintf(`{"$ref":%q}`, canonical+"#"+tc.pointer))
		if err := compiled.Validate(tc.good); err != nil {
			t.Errorf("FAIL valid mandated location %s: %v", tc.pointer, err)
		}
		if err := compiled.Validate(tc.bad); err == nil {
			t.Errorf("FAIL mandated location assertion bypassed: %s", tc.pointer)
		}
	}
	for _, uri := range []string{"relative/path", "https://example.invalid/a%", "https://[bad]/", "https://example.invalid/é", "https://example.invalid/a\\b"} {
		if CheckURI(uri) == nil {
			t.Errorf("FAIL strict URI accepted %q", uri)
		}
	}
	for _, value := range []string{"QR==", " QQ==", "QQ==\n", "QQ==\r\n", "Q===", "!"} {
		if CheckBase64(value) == nil {
			t.Errorf("FAIL canonical base64 accepted %q", value)
		}
	}
}
