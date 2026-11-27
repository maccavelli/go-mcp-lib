package codec

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"testing"
)

type v2Fixture struct{ seen bool }

func (value *v2Fixture) UnmarshalJSONFrom(decoder *jsontext.Decoder) error {
	if _, err := decoder.ReadValue(); err != nil {
		return err
	}
	value.seen = true
	return nil
}

func TestRepairA15V2EntryPoints(t *testing.T) {
	bounds := DefaultBounds()
	var dynamic any
	if err := Unmarshal([]byte(`{"n":9007199254740993,"empty":[],"null":null}`), &dynamic, bounds); err != nil {
		t.Fatal(err)
	}
	object, ok := dynamic.(map[string]any)
	if !ok || object["n"] != jsonv1.Number("9007199254740993") || object["empty"] == nil || object["null"] != nil {
		t.Fatalf("FAIL native v2 exact dynamic semantics: %#v", dynamic)
	}
	if _, err := Marshal(dynamic, bounds); err != nil {
		t.Fatalf("FAIL native v2 marshal: %v", err)
	}
	var typed struct {
		Value int `json:"value"`
	}
	if err := Unmarshal([]byte(`{"VALUE":1}`), &typed, bounds); err != nil || typed.Value != 0 {
		t.Fatalf("FAIL native v2 case-sensitive decoding: %+v %v", typed, err)
	}
	var custom v2Fixture
	if err := Unmarshal([]byte(`{"bad":1,"bad":2}`), &custom, bounds); err == nil || custom.seen {
		t.Fatal("FAIL raw duplicate bypass reached custom v2 decoder")
	}
	if err := Unmarshal([]byte(`{}`), &custom, bounds); err != nil || !custom.seen {
		t.Fatalf("FAIL native v2 custom hook: %v", err)
	}
	for _, bounds := range []Bounds{{}, {MaxFrameBytes: 65 << 20, MaxDepth: 64, MaxTokenBytes: 256, MaxExponent: 1024}} {
		if _, err := Marshal(nil, bounds); err == nil {
			t.Error("FAIL native v2 marshal invalid bounds accepted")
		}
	}
}
