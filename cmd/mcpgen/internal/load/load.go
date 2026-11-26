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
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
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
	CodeNumericLimit       Code = "numeric-limit"
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
	cut := max - len("…")
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
	if l.Limits.MaxDocumentBytes <= 0 || l.Limits.MaxDocumentBytes > DefaultLimits().MaxDocumentBytes {
		return nil, Diagnostic{Phase: "load", Source: path, Code: CodeDocumentTooLarge, Message: "invalid document byte limit"}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, Diagnostic{
			Phase: "load", Source: path, Code: CodeMissingFile,
			Message: bounded(err.Error()),
		}
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, l.Limits.MaxDocumentBytes+1))
	if err != nil {
		return nil, Diagnostic{Phase: "load", Source: path, Code: CodeInvalidJSON, Message: bounded(err.Error())}
	}
	return l.LoadBytes(path, data)
}

// LoadBytes loads and validates a schema document from bytes.
func (l *Loader) LoadBytes(source string, data []byte) (*Document, error) {
	if l.Limits.MaxDocumentBytes <= 0 || l.Limits.MaxDocumentBytes > DefaultLimits().MaxDocumentBytes {
		return nil, Diagnostic{Phase: "load", Source: source, Code: CodeDocumentTooLarge, Message: "invalid document byte limit"}
	}
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
	if l.Limits.MaxDepth <= 0 || l.Limits.MaxDepth > DefaultLimits().MaxDepth || l.Limits.MaxDefinitions <= 0 || l.Limits.MaxDefinitions > DefaultLimits().MaxDefinitions || l.Limits.MaxDiagnostics <= 0 {
		return nil, Diagnostic{Phase: "load", Source: source, Code: CodeInvalidJSON, Message: "invalid loader limits"}
	}
	root, err := parseStrict(data, l.Limits.MaxDepth)
	if err != nil {
		code := CodeInvalidJSON
		pointer := ""
		var se strictError
		if errors.As(err, &se) {
			pointer = se.pointer
			if se.code != "" {
				code = se.code
			}
		}
		var syntax *jsontext.SyntacticError
		if errors.As(err, &syntax) {
			pointer = string(syntax.JSONPointer)
		}
		if errors.Is(err, jsontext.ErrDuplicateName) {
			code = CodeDuplicateMember
		}
		return nil, Diagnostic{
			Phase: "parse", Source: source, Code: code,
			Pointer: pointer,
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
	if dialect, present := obj["$schema"]; present && dialect != "https://json-schema.org/draft/2020-12/schema" {
		return nil, Diagnostic{Phase: "load", Source: source, Pointer: "/$schema", Code: CodeInvalidJSON, Message: "unsupported schema dialect"}
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
	return &Document{Source: source, Bytes: bytes.Clone(data), Definitions: count, Root: root}, nil
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

// parseStrict uses the standard JSON grammar and retains exact numeric tokens.
func parseStrict(data []byte, maxDepth int) (any, error) {
	decoder := jsontext.NewDecoder(bytes.NewReader(data))
	value, err := readValue(decoder, 0, maxDepth)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.ReadToken(); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, err
		}
		return nil, strictError{message: "trailing JSON value"}
	}
	return value, nil
}

func readValue(decoder *jsontext.Decoder, depth, maxDepth int) (any, error) {
	token, err := decoder.ReadToken()
	if err != nil {
		return nil, err
	}
	switch token.Kind() {
	case '{', '[':
		depth++
		if depth > maxDepth {
			return nil, strictError{pointer: string(decoder.StackPointer()), message: "JSON nesting limit exceeded", code: CodeDepthExceeded}
		}
		if token.Kind() == '{' {
			object := make(map[string]any)
			for decoder.PeekKind() != '}' {
				key, err := decoder.ReadToken()
				if err != nil {
					return nil, err
				}
				name := strings.Clone(key.String())
				value, err := readValue(decoder, depth, maxDepth)
				if err != nil {
					return nil, err
				}
				object[name] = value
			}
			_, err := decoder.ReadToken()
			return object, err
		}
		array := make([]any, 0)
		for decoder.PeekKind() != ']' {
			value, err := readValue(decoder, depth, maxDepth)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		_, err := decoder.ReadToken()
		return array, err
	case '"':
		return strings.Clone(token.String()), nil
	case '0':
		raw := token.String()
		if len(raw) > 256 {
			return nil, strictError{pointer: string(decoder.StackPointer()), message: "numeric token exceeds 256 bytes", code: CodeNumericLimit}
		}
		if at := strings.IndexAny(raw, "eE"); at >= 0 {
			exponent, err := strconv.ParseInt(raw[at+1:], 10, 64)
			if err != nil || exponent < -1024 || exponent > 1024 {
				return nil, strictError{pointer: string(decoder.StackPointer()), message: "numeric exponent exceeds magnitude 1024", code: CodeNumericLimit}
			}
		}
		return Number{Token: strings.Clone(raw)}, nil
	case 't', 'f':
		return token.Bool(), nil
	case 'n':
		return nil, nil
	default:
		return nil, strictError{pointer: string(decoder.StackPointer()), message: "unexpected JSON token"}
	}
}
