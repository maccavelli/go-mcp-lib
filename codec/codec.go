// Package codec implements the framework's owned strict JSON entry
// points.
//
// The codec rejects before any lossy decode: duplicate and
// escaped-equivalent member names, invalid UTF-8, trailing values, depth
// beyond the bound, frame size beyond the limit, and numeric tokens over
// 256 bytes or with exponent magnitude over 1,024. Absence, null, empty,
// and zero stay distinct; numbers are preserved as exact tokens, never
// blanket float64.
package codec

import (
	"encoding/json/jsontext"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Bounds are codec safeguards, not protocol rules.
type Bounds struct {
	MaxFrameBytes int // per JSON value; default 4 MiB
	MaxDepth      int // nesting; default 256
	MaxTokenBytes int // per numeric token; default 256
	MaxExponent   int // exponent magnitude; default 1,024
}

// DefaultBounds are the frozen codec safeguards.
func DefaultBounds() Bounds {
	return Bounds{
		MaxFrameBytes: 4 << 20,
		MaxDepth:      256,
		MaxTokenBytes: 256,
		MaxExponent:   1024,
	}
}

// Error is a stable codec rejection with position information.
type Error struct {
	Offset int
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("codec: offset %d: %s", e.Offset, e.Detail)
}

// ValidateRaw checks raw bytes before any decode: duplicate and
// escaped-equivalent member names (rejected natively by the jsontext
// decoder), UTF-8 validity, trailing values, depth, frame size, and
// numeric token/exponent bounds. It returns the byte length consumed.
func ValidateRaw(data []byte, bounds Bounds) (int, error) {
	if len(data) > bounds.MaxFrameBytes {
		return 0, &Error{0, fmt.Sprintf("frame is %d bytes; limit is %d", len(data), bounds.MaxFrameBytes)}
	}
	if !utf8.Valid(data) {
		return 0, &Error{0, "frame is not valid UTF-8"}
	}
	dec := jsontext.NewDecoder(strings.NewReader(string(data)))
	depth := 0
	consumed := 0
	for {
		tok, err := dec.ReadToken()
		if err != nil {
			if strings.Contains(err.Error(), "EOF") {
				break
			}
			if strings.Contains(err.Error(), "duplicate object member") {
				name := betweenQuotes(err.Error())
				return 0, &Error{int(dec.InputOffset()), fmt.Sprintf("duplicate or escaped-equivalent member name %q", name)}
			}
			return 0, &Error{int(dec.InputOffset()), boundedErr(err)}
		}
		consumed = int(dec.InputOffset())
		switch tok.Kind() {
		case '{', '[':
			depth++
			if depth > bounds.MaxDepth {
				return 0, &Error{consumed, fmt.Sprintf("depth %d exceeds limit %d", depth, bounds.MaxDepth)}
			}
		case '}', ']':
			depth--
		case '0':
			token := tok.Clone().String()
			if len(token) > bounds.MaxTokenBytes {
				return 0, &Error{consumed - len(token), fmt.Sprintf("numeric token is %d bytes; limit is %d", len(token), bounds.MaxTokenBytes)}
			}
			if exp, ok := exponentMagnitude(token); ok && exp > bounds.MaxExponent {
				return 0, &Error{consumed - len(token), fmt.Sprintf("exponent magnitude %d exceeds limit %d", exp, bounds.MaxExponent)}
			}
		}
		if depth == 0 && consumed > 0 {
			rest := strings.TrimLeft(string(data[consumed:]), " \t\r\n")
			if rest != "" {
				return 0, &Error{consumed, "trailing value after top-level JSON"}
			}
			return consumed, nil
		}
	}
	if depth != 0 {
		return 0, &Error{consumed, "unterminated container"}
	}
	return consumed, nil
}

// DuplicateNames reports duplicate or escaped-equivalent member names.
// The jsontext decoder rejects both natively; this function surfaces the
// rejection with the offending name extracted.
func DuplicateNames(data []byte) ([]string, error) {
	dec := jsontext.NewDecoder(strings.NewReader(string(data)))
	var dupes []string
	for {
		_, err := dec.ReadToken()
		if err != nil {
			if strings.Contains(err.Error(), "EOF") {
				return dupes, nil
			}
			if strings.Contains(err.Error(), "duplicate object member") {
				dupes = append(dupes, betweenQuotes(err.Error()))
				return dupes, nil
			}
			return dupes, err
		}
	}
}

// betweenQuotes extracts the quoted name from a decoder error message.
func betweenQuotes(message string) string {
	at := strings.IndexByte(message, '"')
	if at < 0 {
		return "?"
	}
	rest := message[at+1:]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return "?"
	}
	return rest[:end]
}

// exponentMagnitude extracts the magnitude of a numeric token's exponent,
// reporting whether one exists.
func exponentMagnitude(token string) (int, bool) {
	lower := strings.ToLower(token)
	at := strings.IndexByte(lower, 'e')
	if at < 0 {
		return 0, false
	}
	mag := 0
	for i := at + 1; i < len(lower); i++ {
		c := lower[i]
		if c == '+' || c == '-' {
			continue
		}
		if c < '0' || c > '9' {
			break
		}
		mag = mag*10 + int(c-'0')
		if mag > 100000 {
			return mag, true
		}
	}
	return mag, true
}

func boundedErr(err error) string {
	message := err.Error()
	if len(message) > 200 {
		message = message[:200] + "…"
	}
	return message
}
