// Package codec supplies bounded strict JSON syntax and owned JSON v2 codecs.
// Raw checks retain numeric lexemes and reject duplicate names, invalid Unicode,
// incomplete/trailing values and resource-limit violations before decoding.
package codec

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Bounds are explicit positive codec safeguards, not protocol rules.
type Bounds struct {
	MaxFrameBytes int // default 4 MiB; hard cap 64 MiB
	MaxDepth      int // default 64; maximum 256
	MaxTokenBytes int // numeric token bytes; maximum 256
	MaxExponent   int // exponent magnitude; maximum 1,024
}

// DefaultBounds returns the accepted undeclared codec safeguards.
func DefaultBounds() Bounds {
	return Bounds{MaxFrameBytes: 4 << 20, MaxDepth: 64, MaxTokenBytes: 256, MaxExponent: 1024}
}

// Error is a stable codec rejection with byte position information.
type Error struct {
	Offset int
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("codec: offset %d: %s", e.Offset, e.Detail)
}

func validBounds(bounds Bounds) error {
	if bounds.MaxFrameBytes <= 0 || bounds.MaxFrameBytes > 64<<20 ||
		bounds.MaxDepth <= 0 || bounds.MaxDepth > 256 ||
		bounds.MaxTokenBytes <= 0 || bounds.MaxTokenBytes > 256 ||
		bounds.MaxExponent <= 0 || bounds.MaxExponent > 1024 {
		return &Error{0, "invalid bounds: require positive frame/depth/token/exponent limits within 64 MiB/256/256/1024"}
	}
	return nil
}

// ValidateRaw accepts exactly one complete JSON value followed by actual EOF.
// The returned offset ends the value, excluding its trailing whitespace.
func ValidateRaw(data []byte, bounds Bounds) (int, error) {
	consumed, _, err := scanRaw(data, bounds)
	return consumed, err
}

// DuplicateNames reports the first native duplicate-name rejection. Other
// malformed, empty, incomplete or over-limit inputs return an error.
func DuplicateNames(data []byte) ([]string, error) {
	_, duplicates, err := scanRaw(data, DefaultBounds())
	if len(duplicates) != 0 {
		return duplicates, nil
	}
	return nil, err
}

func scanRaw(data []byte, bounds Bounds) (int, []string, error) {
	if err := validBounds(bounds); err != nil {
		return 0, nil, err
	}
	if len(data) > bounds.MaxFrameBytes {
		return 0, nil, &Error{0, fmt.Sprintf("frame is %d bytes; limit is %d", len(data), bounds.MaxFrameBytes)}
	}
	decoder := jsontext.NewDecoder(bytes.NewBuffer(data))
	started := false
	for {
		token, err := decoder.ReadToken()
		if err != nil {
			if errors.Is(err, io.EOF) && !started {
				return 0, nil, &Error{int(decoder.InputOffset()), "one complete JSON value required"}
			}
			var syntax *jsontext.SyntacticError
			if errors.As(err, &syntax) && errors.Is(err, jsontext.ErrDuplicateName) {
				name := syntax.JSONPointer.LastToken()
				display := name
				if len(display) > 80 {
					display = display[:80] + "…"
				}
				return 0, []string{name}, &Error{int(syntax.ByteOffset), fmt.Sprintf("duplicate or escaped-equivalent member name %q", display)}
			}
			return 0, nil, &Error{int(decoder.InputOffset()), boundedErr(err)}
		}
		started = true
		consumed := int(decoder.InputOffset())
		if decoder.StackDepth() > bounds.MaxDepth {
			return 0, nil, &Error{consumed, fmt.Sprintf("depth %d exceeds limit %d", decoder.StackDepth(), bounds.MaxDepth)}
		}
		if token.Kind() == '0' {
			lexeme := token.String()
			if len(lexeme) > bounds.MaxTokenBytes {
				return 0, nil, &Error{consumed - len(lexeme), fmt.Sprintf("numeric token is %d bytes; limit is %d", len(lexeme), bounds.MaxTokenBytes)}
			}
			if exponentMagnitude(lexeme) > bounds.MaxExponent {
				return 0, nil, &Error{consumed - len(lexeme), fmt.Sprintf("exponent magnitude exceeds limit %d", bounds.MaxExponent)}
			}
		}
		if decoder.StackDepth() != 0 {
			continue
		}
		// A complete root is necessary but insufficient: consume the actual
		// stream terminator so malformed or additional values cannot pass.
		if _, err := decoder.ReadToken(); !errors.Is(err, io.EOF) {
			return 0, nil, &Error{consumed, "trailing value or malformed content after top-level JSON"}
		}
		return consumed, nil, nil
	}
}

func exponentMagnitude(token string) int {
	at := strings.IndexAny(token, "eE")
	if at < 0 {
		return 0
	}
	exponent := token[at+1:]
	if exponent[0] == '+' || exponent[0] == '-' {
		exponent = exponent[1:]
	}
	magnitude := 0
	for _, digit := range exponent {
		magnitude = magnitude*10 + int(digit-'0')
		if magnitude > 1024 {
			return magnitude // stop before any machine-integer overflow
		}
	}
	return magnitude
}

func boundedErr(err error) string {
	message := err.Error()
	if len(message) > 200 {
		message = message[:200] + "…"
	}
	return message
}
