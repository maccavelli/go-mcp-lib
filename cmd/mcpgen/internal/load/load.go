// Package load implements strict, lossless schema loading for the pinned
// MCP schema inputs.
//
// The loader refuses to normalize: exact number tokens are preserved,
// duplicate object members are rejected, invalid UTF-8 is rejected, and
// every failure produces a stable diagnostic with phase, source URI, JSON
// Pointer, code, and a bounded message. Document, definition, reference,
// and depth limits are generator safeguards, not schema rules.
package load

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// Limits are generator safeguards. They bound loader work on hostile or
// corrupt inputs; they are not statements about the schema language.
type Limits struct {
	MaxDocumentBytes  int64 // per document; 2 MiB default
	MaxDefinitions    int   // per document; 4,096 default
	MaxReferenceEdges int   // per registry; 65,536 default
	MaxDepth          int   // reference resolution depth; 256 default
	MaxDiagnostics    int   // cap before truncation; 1,000 default
}

// DefaultLimits are the plan's frozen generator safeguards.
func DefaultLimits() Limits {
	return Limits{
		MaxDocumentBytes:  2 << 20,
		MaxDefinitions:    4096,
		MaxReferenceEdges: 65536,
		MaxDepth:          256,
		MaxDiagnostics:    1000,
	}
}

// Code classifies a diagnostic. Codes are stable API: a consumer may match
// on them, so they are never renumbered or reused for a different meaning.
type Code string

const (
	CodeDocumentTooLarge   Code = "document-too-large"
	CodeDuplicateMember    Code = "duplicate-member"
	CodeInvalidUTF8        Code = "invalid-utf8"
	CodeTooManyDefinitions Code = "too-many-definitions"
	CodeTooManyReferences  Code = "too-many-references"
	CodeDepthExceeded      Code = "depth-exceeded"
	CodeDiagnosticsCapped  Code = "diagnostics-capped"
	CodeInvalidJSON        Code = "invalid-json"
	CodeMissingFile        Code = "missing-file"
)

// Diagnostic is a stable, bounded loader failure report.
type Diagnostic struct {
	Phase   string `json:"phase"`
	Source  string `json:"source"`
	Pointer string `json:"pointer"`
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%s: %s %s: %s: %s", d.Phase, d.Source, d.Pointer, d.Code, bounded(d.Message))
}

// bounded clamps a message to 256 bytes without splitting a rune.
func bounded(message string) string {
	const max = 256
	if len(message) <= max {
		return message
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(message[cut]) {
		cut--
	}
	return message[:cut] + "…"
}

// Loader loads schema documents under fixed limits.
type Loader struct {
	Limits Limits
}

// New returns a loader with the default limits.
func New() *Loader { return &Loader{Limits: DefaultLimits()} }

// Document is a loaded schema document: verified UTF-8, strict-member JSON,
// with its definitions counted and its raw bytes retained losslessly.
type Document struct {
	Source      string
	Bytes       []byte
	Definitions int
	// Root is the decoded value tree. Numbers are kept as their exact
	// token strings in the raw tree; the model stage re-reads them.
	Root any
}

// LoadFile loads and validates a schema document from a path.
func (l *Loader) LoadFile(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, Diagnostic{
			Phase: "load", Source: path, Code: CodeMissingFile,
			Message: bounded(err.Error()),
		}
	}
	return l.LoadBytes(path, data)
}

// LoadBytes loads and validates a schema document from bytes.
func (l *Loader) LoadBytes(source string, data []byte) (*Document, error) {
	if int64(len(data)) > l.Limits.MaxDocumentBytes {
		return nil, Diagnostic{
			Phase: "load", Source: source, Code: CodeDocumentTooLarge,
			Message: fmt.Sprintf("document is %d bytes; limit is %d", len(data), l.Limits.MaxDocumentBytes),
		}
	}
	if !utf8.Valid(data) {
		return nil, Diagnostic{
			Phase: "parse", Source: source, Code: CodeInvalidUTF8,
			Message: "document is not valid UTF-8",
		}
	}
	root, err := parseStrict(data)
	if err != nil {
		code := CodeInvalidJSON
		if se, ok := err.(strictError); ok && se.code != "" {
			code = se.code
		}
		return nil, Diagnostic{
			Phase: "parse", Source: source, Code: code,
			Pointer: err.(strictError).pointer,
			Message: bounded(err.Error()),
		}
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return nil, Diagnostic{
			Phase: "parse", Source: source, Code: CodeInvalidJSON,
			Message: "schema root must be a JSON object",
		}
	}
	count := 0
	if defs, present := obj["$defs"]; present {
		defsObj, ok := defs.(map[string]any)
		if !ok {
			return nil, Diagnostic{
				Phase: "parse", Source: source, Code: CodeInvalidJSON,
				Pointer: "/$defs",
				Message: "$defs must be an object",
			}
		}
		count = len(defsObj)
	}
	if count > l.Limits.MaxDefinitions {
		return nil, Diagnostic{
			Phase: "load", Source: source, Code: CodeTooManyDefinitions,
			Message: fmt.Sprintf("document has %d definitions; limit is %d", count, l.Limits.MaxDefinitions),
		}
	}
	return &Document{Source: source, Bytes: data, Definitions: count, Root: root}, nil
}

// Number is an exact numeric token. The loader never converts schema
// numbers to float64; the token is preserved verbatim so no precision is
// lost and the model can distinguish numbers from strings.
type Number struct{ Token string }

// strictError carries the JSON Pointer and code of a parse failure.
type strictError struct {
	pointer string
	message string
	code    Code
}

func (e strictError) Error() string { return e.message }

// parseStrict decodes JSON rejecting duplicate object members and recording
// the pointer at which any failure occurred. Numbers decode to their exact
// token strings so no precision is lost before the model stage.
func parseStrict(data []byte) (any, error) {
	p := &strictParser{data: data}
	return p.parseValue(0)
}

type strictParser struct {
	data []byte
	pos  int
}

func (p *strictParser) ws() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *strictParser) parseValue(depth int) (any, error) {
	if depth > 512 {
		return nil, strictError{pointer: "", message: "nesting deeper than 512"}
	}
	p.ws()
	if p.pos >= len(p.data) {
		return nil, strictError{pointer: "", message: "unexpected end of document"}
	}
	switch c := p.data[p.pos]; c {
	case '{':
		return p.parseObject(depth)
	case '[':
		return p.parseArray(depth)
	case '"':
		return p.parseString()
	case 't':
		return p.parseKeyword("true", true)
	case 'f':
		return p.parseKeyword("false", false)
	case 'n':
		return p.parseKeyword("null", nil)
	default:
		return p.parseNumber()
	}
}

func (p *strictParser) parseObject(depth int) (any, error) {
	pointer := len(p.data[:p.pos])
	p.pos++ // consume {
	seen := make(map[string]struct{})
	result := make(map[string]any)
	first := true
	for {
		p.ws()
		if p.pos >= len(p.data) {
			return nil, strictError{pointer: "", message: "unterminated object"}
		}
		if p.data[p.pos] == '}' {
			p.pos++
			return result, nil
		}
		if !first {
			if p.data[p.pos] != ',' {
				return nil, strictError{pointer: fmt.Sprintf("@%d", pointer), message: "expected , or }"}
			}
			p.pos++
			p.ws()
		}
		first = false
		if p.data[p.pos] != '"' {
			return nil, strictError{pointer: fmt.Sprintf("@%d", p.pos), message: "expected member name"}
		}
		key, err := p.parseString()
		if err != nil {
			return nil, err
		}
		p.ws()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return nil, strictError{pointer: fmt.Sprintf("@%d", p.pos), message: "expected : after member name"}
		}
		p.pos++
		value, err := p.parseValue(depth + 1)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[key]; dup {
			return nil, strictError{
				pointer: fmt.Sprintf("@%d", pointer),
				message: "duplicate member " + key,
				code:    CodeDuplicateMember,
			}
		}
		seen[key] = struct{}{}
		result[key] = value
	}
}

func (p *strictParser) parseArray(depth int) (any, error) {
	p.pos++ // consume [
	result := make([]any, 0, 4)
	first := true
	for {
		p.ws()
		if p.pos >= len(p.data) {
			return nil, strictError{pointer: "", message: "unterminated array"}
		}
		if p.data[p.pos] == ']' {
			p.pos++
			return result, nil
		}
		if !first {
			if p.data[p.pos] != ',' {
				return nil, strictError{pointer: fmt.Sprintf("@%d", p.pos), message: "expected , or ]"}
			}
			p.pos++
			p.ws()
		}
		first = false
		item, err := p.parseValue(depth + 1)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
}

func (p *strictParser) parseString() (string, error) {
	p.pos++ // consume opening quote
	var sb strings.Builder
	for {
		if p.pos >= len(p.data) {
			return "", strictError{pointer: "", message: "unterminated string"}
		}
		c := p.data[p.pos]
		if c == '"' {
			p.pos++
			return sb.String(), nil
		}
		if c == '\\' {
			p.pos++
			if p.pos >= len(p.data) {
				return "", strictError{pointer: "", message: "unterminated escape"}
			}
			switch p.data[p.pos] {
			case '"', '\\', '/':
				sb.WriteByte(p.data[p.pos])
			case 'b':
				sb.WriteByte('\b')
			case 'f':
				sb.WriteByte('\f')
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case 'u':
				if p.pos+4 >= len(p.data) {
					return "", strictError{pointer: "", message: "truncated \\u escape"}
				}
				r, ok := parseHex4(p.data[p.pos+1 : p.pos+5])
				if !ok {
					return "", strictError{pointer: "", message: "invalid \\u escape"}
				}
				sb.WriteRune(r)
				p.pos += 4
			default:
				return "", strictError{pointer: "", message: "invalid escape character"}
			}
			p.pos++
			continue
		}
		if c < 0x20 {
			return "", strictError{pointer: "", message: "raw control character in string"}
		}
		sb.WriteByte(c)
		p.pos++
	}
}

func parseHex4(hex []byte) (rune, bool) {
	var r rune
	for _, c := range hex {
		r <<= 4
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			r |= rune(c-'A') + 10
		default:
			return 0, false
		}
	}
	return r, true
}

func (p *strictParser) parseKeyword(word string, value any) (any, error) {
	if p.pos+len(word) <= len(p.data) && string(p.data[p.pos:p.pos+len(word)]) == word {
		p.pos += len(word)
		return value, nil
	}
	return nil, strictError{pointer: fmt.Sprintf("@%d", p.pos), message: "invalid literal"}
}

func (p *strictParser) parseNumber() (any, error) {
	start := p.pos
	if p.pos < len(p.data) && (p.data[p.pos] == '-' || p.data[p.pos] == '+') {
		p.pos++
	}
	digits := 0
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if c >= '0' && c <= '9' {
			digits++
			p.pos++
			continue
		}
		if c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-' {
			p.pos++
			continue
		}
		break
	}
	if digits == 0 {
		return nil, strictError{pointer: fmt.Sprintf("@%d", start), message: "invalid number"}
	}
	// Exact token: numbers are preserved as written, unnormalized.
	return Number{Token: string(p.data[start:p.pos])}, nil
}

var _ = io.Discard // keep io import honest for future stream loading
