package validation

import (
	"strings"
	"testing"
)

func TestRootURI(t *testing.T) {
	for _, uri := range []string{
		"file:///workspace/root",
		"https://example.invalid/workspace",
	} {
		if err := CheckRootURI(uri); err != nil {
			t.Errorf("FAIL valid Root.uri rejected: %v", err)
		}
	}
	for _, uri := range []string{
		"",           // empty: Root.uri is required
		"/workspace", // relative reference, not absolute
		"workspace",  // bare path
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
	} {
		if err := CheckURITemplate(template); err != nil {
			t.Errorf("FAIL valid template rejected: %v", err)
		}
	}
	for _, template := range []string{
		"",                    // empty
		"logs/{name}",         // no scheme
		"file:///x/{+ref}",    // reserved expansion unsupported in profile
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
