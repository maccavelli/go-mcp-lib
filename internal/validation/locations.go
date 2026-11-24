package validation

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

// Protocol-location rules mandated at specific schema locations. These
// are assertions the framework enforces on its own contract, distinct
// from format annotations the application-schema profile may declare.
//
// The accepted application-schema profile is JSON Schema Draft 2020-12
// with the standard vocabulary plus this package's URI/URI-template
// location rules. Regex semantics are Go RE2 (host language); schemas
// requiring other engines are rejected at declaration, not silently
// applied with incompatible semantics.

// uriPattern approximates RFC 3986 URI: scheme : hier-part, with a
// practical RE2-compatible character class.
var uriPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// CheckRootURI enforces the Root.uri location rule: an absolute URI
// (not a relative reference) of the required shape. Empty is invalid;
// the schema models Root.uri as required.
func CheckRootURI(value string) error {
	if value == "" {
		return fmt.Errorf("validation: Root.uri is empty")
	}
	if !uriPattern.MatchString(value) {
		return fmt.Errorf("validation: Root.uri %q is not an absolute URI", clip(value))
	}
	return nil
}

// CheckURI enforces an absolute-URI location (e.g. Resource.uri).
func CheckURI(value string) error {
	if value == "" {
		return fmt.Errorf("validation: URI is empty")
	}
	if !uriPattern.MatchString(value) {
		return fmt.Errorf("validation: %q is not an absolute URI", clip(value))
	}
	return nil
}

// CheckURITemplate enforces RFC 6570 level 1-2 template shape at
// template locations (e.g. ResourceTemplate.uriTemplate): a URI-shaped
// prefix with {var} expansions, each expansion a simple varspec with an
// optional numeric prefix modifier or explode marker.
var templateVar = regexp.MustCompile(`^\{[A-Za-z0-9_]+(:[0-9]+)?(,\*|[*])?\}$`)
var templateVars = regexp.MustCompile(`\{[^{}]*\}`)

func CheckURITemplate(value string) error {
	if value == "" {
		return fmt.Errorf("validation: uriTemplate is empty")
	}
	stripped := templateVars.ReplaceAllString(value, "x")
	if !uriPattern.MatchString(stripped) {
		return fmt.Errorf("validation: uriTemplate %q is not URI-shaped after expansion", clip(value))
	}
	for _, expansion := range templateVars.FindAllString(value, -1) {
		if !templateVar.MatchString(expansion) {
			return fmt.Errorf(
				"validation: uriTemplate expansion %s is not a supported RFC 6570 simple varspec; supported: {var}, {var:NNN}, {var*}",
				clip(expansion),
			)
		}
	}
	if strings.Count(value, "{") != strings.Count(value, "}") {
		return fmt.Errorf("validation: uriTemplate %q has unbalanced braces", clip(value))
	}
	return nil
}

// CheckBase64 enforces standard base64 (padded, canonical alphabet)
// at content-encoding locations.
func CheckBase64(value string) error {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("validation: not canonical base64: %v", err)
	}
	// Canonical re-encoding must reproduce the input exactly.
	if base64.StdEncoding.EncodeToString(decoded) != strings.TrimSpace(value) {
		return fmt.Errorf("validation: base64 is not canonical")
	}
	return nil
}

// RejectUnknownRegexSemantics returns an actionable error for schema
// declarations that require regex semantics beyond the accepted
// profile, instead of silently applying incompatible behavior.
func RejectUnknownRegexSemantics(pattern, location string) error {
	return fmt.Errorf(
		"validation: %s: pattern %q uses semantics outside the accepted application-schema profile (Go RE2, no backreferences or lookaround); simplify the pattern or see docs for the supported profile",
		location,
		clip(pattern),
	)
}

func clip(value string) string {
	if len(value) > 80 {
		return value[:80] + "…"
	}
	return value
}
