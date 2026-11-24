package codec

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func asError(err error, target **Error) bool {
	return errors.As(err, target)
}

func mustValidate(t *testing.T, body string) {
	t.Helper()
	if _, err := ValidateRaw([]byte(body), DefaultBounds()); err != nil {
		t.Fatalf("FAIL validate %q: %v", body, err)
	}
}

func mustReject(t *testing.T, body string, why string) {
	t.Helper()
	if _, err := ValidateRaw([]byte(body), DefaultBounds()); err == nil {
		t.Fatalf("FAIL accepted %q (%s)", body, why)
	}
}

func TestAccepts(t *testing.T) {
	for _, body := range []string{
		`{}`, `[]`, `{"a":1}`, `[1,[2,[3]]]`, `"x"`, `null`, `true`,
		`{"a":{"b":[1,2,{"c":null}]}}`, `1e308`, `-0.5`,
	} {
		mustValidate(t, body)
	}
}

func TestTrailingValue(t *testing.T) {
	mustReject(t, `{} {}`, "trailing object")
	mustReject(t, `1 2`, "trailing number")
	mustReject(t, `[] null`, "trailing null")
}

func TestDepthBound(t *testing.T) {
	bounds := DefaultBounds()
	inner := strings.Repeat("[", bounds.MaxDepth) + strings.Repeat("]", bounds.MaxDepth)
	if _, err := ValidateRaw([]byte(inner), bounds); err != nil {
		t.Fatalf("FAIL exact bound rejected: %v", err)
	}
	over := strings.Repeat("[", bounds.MaxDepth+1) + strings.Repeat("]", bounds.MaxDepth+1)
	if _, err := ValidateRaw([]byte(over), bounds); err == nil {
		t.Fatal("FAIL one past depth bound accepted")
	}
}

func TestFrameBound(t *testing.T) {
	bounds := Bounds{MaxFrameBytes: 16, MaxDepth: 4, MaxTokenBytes: 8, MaxExponent: 4}
	mustValidate(t, `{"a":1}`)
	if _, err := ValidateRaw([]byte(`{"pad":"1234567890123456"}`), bounds); err == nil {
		t.Fatal("FAIL frame over bound accepted")
	}
}

func TestNumericTokenBound(t *testing.T) {
	use := DefaultBounds()
	// 256-byte numeric token: at the bound.
	atBound := "0." + strings.Repeat("1", 254)
	if len(atBound) != 256 {
		t.Fatalf("fixture math: %d", len(atBound))
	}
	if _, err := ValidateRaw([]byte(atBound), use); err != nil {
		t.Fatalf("FAIL token at bound rejected: %v", err)
	}
	overBound := "0." + strings.Repeat("1", 256)
	if _, err := ValidateRaw([]byte(overBound), use); err == nil {
		t.Fatal("FAIL token one past bound accepted")
	}
}

func TestExponentBound(t *testing.T) {
	mustValidate(t, `1e1024`)
	mustReject(t, `1e1025`, "exponent over bound")
	mustValidate(t, `1e-1024`)
	mustReject(t, `1e-1025`, "negative exponent over bound")
	mustValidate(t, `1E+100`)
	// Strings containing digits never trip numeric bounds.
	mustValidate(t, `"`+strings.Repeat("1", 300)+`"`)
}

func TestDuplicateNames(t *testing.T) {
	dupes, err := DuplicateNames([]byte(`{"a":1,"a":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(dupes) != 1 || dupes[0] != "a" {
		t.Fatalf("FAIL dupes: %v", dupes)
	}
}

func TestValidateRawClassifiesDuplicates(t *testing.T) {
	// ValidateRaw must reject duplicates with the classified codec
	// error, not merely propagate a raw decoder error: the classification
	// is what the duplicate-bypass mutation removes.
	_, err := ValidateRaw([]byte(`{"a":1,"a":2}`), DefaultBounds())
	if err == nil {
		t.Fatal("FAIL duplicate accepted")
	}
	var cErr *Error
	if ok := asError(err, &cErr); !ok {
		t.Fatalf("FAIL unclassified error: %v", err)
	}
	if !strings.Contains(cErr.Detail, "duplicate or escaped-equivalent member name") {
		t.Fatalf("FAIL unclassified detail: %s", cErr.Detail)
	}
	_, err = ValidateRaw([]byte(`{"a":1,"\u0061":2}`), DefaultBounds())
	if err == nil {
		t.Fatal("FAIL escaped-equivalent accepted")
	}
	if ok := asError(err, &cErr); !ok || !strings.Contains(cErr.Detail, "duplicate or escaped-equivalent member name") {
		t.Fatalf("FAIL escaped-equivalent detail: %v", err)
	}
}

func TestEscapedEquivalentNames(t *testing.T) {
	// "a" and "\u0061" are the same name escaped differently.
	dupes, err := DuplicateNames([]byte(`{"a":1,"\u0061":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(dupes) != 1 {
		t.Fatalf("FAIL escaped-equivalent not caught: %v", dupes)
	}
}

func TestNestedDuplicates(t *testing.T) {
	// The decoder rejects at the first duplicate.
	dupes, err := DuplicateNames([]byte(`{"x":{"b":1,"b":2},"y":[{"c":1,"c":2}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(dupes) != 1 || dupes[0] != "b" {
		t.Fatalf("FAIL nested dupes: %v", dupes)
	}
}

func TestNoFalseDuplicates(t *testing.T) {
	dupes, err := DuplicateNames([]byte(`{"a":1,"b":2,"c":{"a":3}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(dupes) != 0 {
		t.Fatalf("FAIL false dupes: %v", dupes)
	}
}

func TestInvalidUTF8Rejected(t *testing.T) {
	bad := append([]byte(`{"a":"`), 0xff, 0xfe, '"', '}')
	if _, err := ValidateRaw(bad, DefaultBounds()); err == nil {
		t.Fatal("FAIL invalid UTF-8 accepted")
	}
}

func TestDistinctAbsenceNullEmptyZero(t *testing.T) {
	// The codec's contract: absence, null, empty, and zero are distinct
	// states, verified through exact-token decode.
	var probe struct {
		Absent *int        `json:"absent,omitempty"`
		Null   *int        `json:"null"`
		Empty  []int       `json:"empty"`
		Zero   int         `json:"zero"`
		Num    json.Number `json:"num"`
	}
	if err := json.Unmarshal([]byte(`{"null":null,"empty":[],"zero":0,"num":0.30000000000000004}`), &probe); err != nil {
		t.Fatal(err)
	}
	if probe.Absent != nil {
		t.Fatal("FAIL absent became present")
	}
	if probe.Null != nil {
		t.Fatal("FAIL null became non-nil")
	}
	if probe.Empty == nil || len(probe.Empty) != 0 {
		t.Fatal("FAIL empty collapsed")
	}
	if probe.Zero != 0 {
		t.Fatal("FAIL zero lost")
	}
	if string(probe.Num) != "0.30000000000000004" {
		t.Fatalf("FAIL exact number lost: %s", probe.Num)
	}
}
