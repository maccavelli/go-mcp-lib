package load

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRepairA05StrictLoader(t *testing.T) {
	for name, body := range map[string]string{
		"plus": `{"n":+1}`, "leading-zero": `{"n":01}`,
		"fraction": `{"n":1.}`, "exponent": `{"n":1e+}`,
		"double-sign": `{"n":1-2}`, "trailing-value": `{} {}`,
		"trailing-literal": `{}true`, "truncated-object": `{"n":`,
		"unpaired-high": `{"s":"\ud800"}`, "unpaired-low": `{"s":"\udc00"}`,
		"escaped-duplicate": `{"a":1,"\u0061":2}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := New().LoadBytes(name, []byte(body)); err == nil {
				t.Fatalf("FAIL strict loader accepted deliberately broken %s", name)
			}
		})
	}
	t.Run("valid-lossless", func(t *testing.T) {
		body := []byte(`{"n":-1.2300e+42,"s":"\ud83d\ude00"}`)
		doc, err := New().LoadBytes("valid", body)
		if err != nil {
			t.Fatal(err)
		}
		object := doc.Root.(map[string]any)
		if object["n"].(Number).Token != "-1.2300e+42" || object["s"] != "😀" {
			t.Fatalf("FAIL lossless value: %#v", object)
		}
		body[0] = '['
		if doc.Bytes[0] != '{' {
			t.Fatal("FAIL caller bytes mutate the retained document")
		}
	})
	t.Run("file-bound", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "large.json")
		if err := os.WriteFile(path, []byte(strings.Repeat(" ", 1024)), 0o600); err != nil {
			t.Fatal(err)
		}
		loader := New()
		loader.Limits.MaxDocumentBytes = 16
		if _, err := loader.LoadFile(path); err == nil {
			t.Fatal("FAIL bounded reader accepted oversized file")
		}
	})
	t.Run("depth-and-pointer", func(t *testing.T) {
		loader := New()
		loader.Limits.MaxDepth = 2
		_, err := loader.LoadBytes("depth", []byte(`{"a":{"b":{}}}`))
		diagnostic, ok := err.(Diagnostic)
		if !ok || diagnostic.Code != CodeDepthExceeded || diagnostic.Pointer != "/a/b" {
			t.Fatalf("FAIL depth diagnostic: %v", err)
		}
	})
	t.Run("unicode-diagnostic-bound", func(t *testing.T) {
		message := bounded(strings.Repeat("語", 200))
		if len(message) > 256 || !utf8.ValidString(message) {
			t.Fatal("FAIL diagnostic splits Unicode or exceeds byte allowance")
		}
	})
}
