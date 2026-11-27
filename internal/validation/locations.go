package validation

import (
	"encoding/base64"
	"fmt"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	"github.com/yosida95/uritemplate/v3"
)

var uriParts = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*):(?://([^/?#]*))?([^?#]*)(?:\?([^#]*))?(?:#(.*))?$`)
var ipvFuture = regexp.MustCompile(`^[vV][0-9A-Fa-f]+\.[A-Za-z0-9._~!$&'()*+,;=:-]+$`)

// CheckURI checks absolute RFC 3986 syntax, including percent escapes and
// bracketed IP literals, without fetching, normalizing or applying access policy.
func CheckURI(value string) error {
	if !validURI(value) {
		return fmt.Errorf("validation: %q is not an absolute URI (RFC 3986)", clip(value))
	}
	return nil
}

func validURI(value string) bool {
	parts := uriParts.FindStringSubmatch(value)
	if parts == nil || !validURIChars(parts[3], ":@/") || !validURIChars(parts[4], ":@/?") || !validURIChars(parts[5], ":@/?") {
		return false
	}
	authority := parts[2]
	if at := strings.LastIndexByte(authority, '@'); at >= 0 {
		if !validURIChars(authority[:at], ":") {
			return false
		}
		authority = authority[at+1:]
	}
	if strings.HasPrefix(authority, "[") {
		end := strings.IndexByte(authority, ']')
		if end < 0 {
			return false
		}
		literal := authority[1:end]
		address, err := netip.ParseAddr(literal)
		if (err != nil || !address.Is6() || address.Zone() != "") && !ipvFuture.MatchString(literal) {
			return false
		}
		authority = "host" + authority[end+1:]
	}
	host, port, hasPort := strings.Cut(authority, ":")
	if !validURIChars(host, "") {
		return false
	}
	if hasPort {
		for _, char := range port {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	// Without authority a path cannot begin with two slashes (RFC 3986 3.3).
	return strings.HasPrefix(value[len(parts[1])+1:], "//") || !strings.HasPrefix(parts[3], "//")
}

func validURIChars(value, extra string) bool {
	for i := 0; i < len(value); i++ {
		char := value[i]
		if char == '%' {
			if i+2 >= len(value) || !hexDigit(value[i+1]) || !hexDigit(value[i+2]) {
				return false
			}
			i += 2
			continue
		}
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("-._~!$&'()*+,;="+extra, rune(char)) {
			continue
		}
		return false
	}
	return true
}

func hexDigit(char byte) bool {
	return char >= '0' && char <= '9' || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F'
}

// CheckRootURI enforces MCP Root.uri's file URI and absolute-path requirement.
// Host components permit remote file roots; no filesystem access is attempted.
func CheckRootURI(value string) error {
	if err := CheckURI(value); err != nil {
		return err
	}
	parsed, err := url.Parse(value)
	if err != nil || !strings.HasPrefix(value, "file://") || parsed.Opaque != "" || !strings.HasPrefix(parsed.Path, "/") {
		return fmt.Errorf("validation: Root.uri must be an absolute file URI: %q", clip(value))
	}
	return nil
}

// CheckURITemplate uses the pinned RFC 6570 parser, including all operators and
// relative templates. URI templates have their own grammar, not URI grammar.
func CheckURITemplate(value string) error {
	if _, err := uritemplate.New(value); err != nil {
		return fmt.Errorf("validation: invalid RFC 6570 template %q: %w", clip(value), err)
	}
	return nil
}

// CheckBase64 requires canonical standard padded base64, including zero pad
// bits. Comparing the original input rejects ignored CR/LF and whitespace.
func CheckBase64(value string) error {
	decoded, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil {
		return fmt.Errorf("validation: invalid base64: %w", err)
	}
	if base64.StdEncoding.EncodeToString(decoded) != value {
		return fmt.Errorf("validation: base64 is not canonical")
	}
	return nil
}

// RejectUnknownRegexSemantics reports an unqualified declaration with its pointer.
func RejectUnknownRegexSemantics(pattern, location string) error {
	return fmt.Errorf("validation: %s: pattern %q uses semantics outside the accepted application-schema profile (qualified RE2/ECMA subset); simplify the pattern or see docs for the supported profile", location, clip(pattern))
}

func clip(value string) string {
	if len(value) > 80 {
		return value[:80] + "…"
	}
	return value
}
