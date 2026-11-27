package validation

import (
	"strings"
	"testing"
)

func TestRootURI(t *testing.T) {
	for _, uri := range []string{
		"file:///workspace/root",
		"file://server/share/workspace",
	} {
		if err := CheckRootURI(uri); err != nil {
			t.Errorf("FAIL valid Root.uri rejected: %v", err)
		}
	}
	for _, uri := range []string{
		"",                                  // empty: Root.uri is required
		"/workspace",                        // relative reference, not absolute
		"workspace",                         // bare path
		"https://example.invalid/workspace", // roots must use file URIs
		"file:workspace",                    // opaque relative file path
		"file:/workspace",                   // pinned protocol requires file://
	} {
		if err := CheckRootURI(uri); err == nil {
			t.Errorf("FAIL Root.uri %q accepted", uri)
		}
	}
}

func TestURITemplate(t *testing.T) {
	for _, template := range []string{
		"file:///logs/{name}",
		"https://example.invalid/users/{userId}/posts/{postId}",
		"file:///docs/{page:3}",
		"file:///x/{+ref}",
		"logs/{name}",
		"https://example.invalid/{+path}{?q,limit}",
		"", // RFC 6570 permits a template with no literals or expressions
	} {
		if err := CheckURITemplate(template); err != nil {
			t.Errorf("FAIL valid template rejected: %v", err)
		}
	}
	for _, template := range []string{
		"file:///x/{a b}",     // invalid varspec
		"file:///x/{unclosed", // unbalanced
	} {
		if err := CheckURITemplate(template); err == nil {
			t.Errorf("FAIL template %q accepted", template)
		}
	}
}

func TestBase64(t *testing.T) {
	if err := CheckBase64("aGVsbG8="); err != nil {
		t.Fatalf("FAIL valid base64 rejected: %v", err)
	}
	for _, value := range []string{
		"aGVsbG8",    // missing padding
		"aGVsbG8===", // extra padding
		"!!!!",       // not base64 alphabet
	} {
		if err := CheckBase64(value); err == nil {
			t.Errorf("FAIL base64 %q accepted", value)
		}
	}
}

func TestRejectUnknownRegexSemantics(t *testing.T) {
	err := RejectUnknownRegexSemantics(`(a+)\1`, "Tool.inputSchema.properties.x.pattern")
	if err == nil || !strings.Contains(err.Error(), "accepted application-schema profile") {
		t.Fatalf("FAIL actionable pointer missing: %v", err)
	}
}
