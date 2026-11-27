package codec

import (
	"bytes"
	"strings"
	"testing"
)

func TestRepairA15RawCodec(t *testing.T) {
	valid := []string{`null`, `false`, `0`, `-0`, `1.0`, `1e+00001024`, `1e-00001024`, `""`,
		`"\ud83d\ude00"`, `{"a":[],"b":{}}`, `{"a":1,"nested":{"a":2}}`, " \n{}\r\t "}
	invalid := []string{"", " \n\t", `"`, `"unfinished`, `"\`, `"\u12`, `"\ud800"`, `"\udc00"`,
		`"\ud800x"`, `"\ud800\u0061"`, `n`, `nu`, `nul`, `tru`, `fals`, `-`, `+1`, `01`, `-01`,
		`1.`, `1e`, `1e+`, `1e-`, `[`, `[0`, `[0,`, `{`, `{"a"`, `{"a":`, `{"a":0`,
		`{}[]`, `true false`, `0x10`, `1e1025`, `1e-1025`, `NaN`, `Infinity`,
		`{"a":1,"\u0061":2}`, `{"":1,"":2}`}
	for _, input := range valid {
		t.Run("valid/"+input, func(t *testing.T) {
			body := []byte(input)
			before := bytes.Clone(body)
			if _, err := ValidateRaw(body, DefaultBounds()); err != nil {
				t.Fatalf("FAIL strict raw valid input %q: %v", input, err)
			}
			if !bytes.Equal(body, before) {
				t.Fatal("FAIL strict raw validation mutated caller bytes")
			}
		})
	}
	for _, input := range invalid {
		t.Run("invalid/"+input, func(t *testing.T) {
			if _, err := ValidateRaw([]byte(input), DefaultBounds()); err == nil {
				t.Fatalf("FAIL strict raw accepted malformed input %q", input)
			}
		})
	}
	t.Run("invalid-UTF8", func(t *testing.T) {
		if _, err := ValidateRaw([]byte{'"', 0xff, '"'}, DefaultBounds()); err == nil {
			t.Fatal("FAIL strict raw accepted invalid UTF-8")
		}
	})
	t.Run("duplicates-structured-name", func(t *testing.T) {
		names, err := DuplicateNames([]byte(`{"a/b~x\"":1,"\u0061/b~x\"":2}`))
		if err != nil || len(names) != 1 || names[0] != "a/b~x\"" {
			t.Fatalf("FAIL structured duplicate name: %q %v", names, err)
		}
		for _, input := range []string{"", `"unfinished`, `1e`, `[] true`} {
			if _, err := DuplicateNames([]byte(input)); err == nil {
				t.Errorf("FAIL duplicate inspector accepted malformed input %q", input)
			}
		}
	})
	t.Run("bounds", func(t *testing.T) {
		defaults := DefaultBounds()
		if defaults != (Bounds{4 << 20, 64, 256, 1024}) {
			t.Fatalf("FAIL frozen default bounds: %+v", defaults)
		}
		for _, depth := range []int{64, 256} {
			bounds := defaults
			bounds.MaxDepth = depth
			at := strings.Repeat("[", depth) + strings.Repeat("]", depth)
			if _, err := ValidateRaw([]byte(at), bounds); err != nil {
				t.Errorf("FAIL strict raw depth %d rejected: %v", depth, err)
			}
			if _, err := ValidateRaw([]byte("["+at+"]"), bounds); err == nil {
				t.Errorf("FAIL strict raw depth %d accepted", depth+1)
			}
		}
		bounds := defaults
		bounds.MaxFrameBytes = 16
		if _, err := ValidateRaw([]byte(`"12345678901234"`), bounds); err != nil {
			t.Fatalf("FAIL exact frame limit: %v", err)
		}
		if _, err := ValidateRaw([]byte(`"123456789012345"`), bounds); err == nil {
			t.Fatal("FAIL strict raw frame limit bypass")
		}
		for _, size := range []int{255, 256, 257} {
			_, err := ValidateRaw([]byte("0."+strings.Repeat("1", size-2)), defaults)
			if (err == nil) != (size <= 256) {
				t.Errorf("FAIL strict raw numeric token limit %d: %v", size, err)
			}
		}
		bad := []Bounds{{0, 64, 256, 1024}, {-1, 64, 256, 1024}, {64<<20 + 1, 64, 256, 1024},
			{4 << 20, 0, 256, 1024}, {4 << 20, 257, 256, 1024}, {4 << 20, 64, 0, 1024},
			{4 << 20, 64, 257, 1024}, {4 << 20, 64, 256, 0}, {4 << 20, 64, 256, 1025}}
		for _, bounds := range bad {
			if _, err := ValidateRaw([]byte(`null`), bounds); err == nil {
				t.Errorf("FAIL strict raw invalid bounds accepted: %+v", bounds)
			}
		}
	})
}
