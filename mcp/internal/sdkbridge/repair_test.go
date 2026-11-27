package sdkbridge

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"reflect"
	"testing"

	"github.com/maccavelli/go-mcp-lib/codec"
	"github.com/maccavelli/go-mcp-lib/protocol"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func exactValue(t *testing.T, body []byte) any {
	t.Helper()
	var value any
	if err := codec.Unmarshal(body, &value, codec.DefaultBounds()); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestRepairA16SDKDifferentials(t *testing.T) {
	t.Run("real-sdk-custom-resource-link", func(t *testing.T) {
		frame := []byte(`{"resultType":"complete","content":[{"type":"resource_link","name":"fixture","uri":"file:///fixture","size":9007199254740993}]}`)
		var upstream sdk.CallToolResult
		if err := jsonv1.Unmarshal(frame, &upstream); err != nil {
			t.Fatal(err)
		}
		link, ok := upstream.Content[0].(*sdk.ResourceLink)
		if !ok || link.Size == nil || *link.Size != 9007199254740993 {
			t.Fatal("FAIL actual SDK resource-link/exact integer decode")
		}
		owned, err := FromSDK[protocol.CallToolResult](&upstream)
		if err != nil {
			t.Fatal(err)
		}
		body, err := codec.Marshal(owned, codec.DefaultBounds())
		if err != nil || !reflect.DeepEqual(exactValue(t, body), exactValue(t, frame)) {
			t.Fatalf("FAIL SDK custom wire semantics: %s: %v", body, err)
		}
		again, err := ToSDK[sdk.CallToolResult](owned)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := jsonv1.Marshal(again)
		if err != nil || !reflect.DeepEqual(exactValue(t, encoded), exactValue(t, frame)) {
			t.Fatalf("FAIL SDK custom wire roundtrip: %s: %v", encoded, err)
		}
	})
	t.Run("presence-case-unknown-and-exact-raw", func(t *testing.T) {
		for _, tc := range []struct {
			frame   string
			present bool
		}{
			{`{"uri":"file:///fixture"}`, false},
			{`{"uri":"file:///fixture","name":""}`, true},
			{`{"uri":"file:///fixture","name":"shown","extra":{"n":9007199254740993}}`, true},
		} {
			var owned protocol.Root
			if err := codec.Unmarshal([]byte(tc.frame), &owned, codec.DefaultBounds()); err != nil {
				t.Fatal(err)
			}
			if owned.Name.IsPresent() != tc.present {
				t.Error("FAIL generated field presence normalized")
			}
			body, err := codec.Marshal(owned, codec.DefaultBounds())
			if err != nil || !reflect.DeepEqual(exactValue(t, body), exactValue(t, []byte(tc.frame))) {
				t.Errorf("FAIL generated unknown/raw/zero preservation: %s: %v", body, err)
			}
		}
		for _, bad := range []string{`{"URI":"file:///fixture"}`, `{"uri":"file:///fixture","name":null}`, `{"uri":"https://example.invalid/fixture"}`} {
			var owned protocol.Root
			if codec.Unmarshal([]byte(bad), &owned, codec.DefaultBounds()) == nil {
				t.Errorf("FAIL generated required/case/location assertion accepted %s", bad)
			}
		}
	})
	t.Run("v1-v2-collection-policy", func(t *testing.T) {
		type collections struct {
			List []string          `json:"list"`
			Map  map[string]string `json:"map"`
		}
		legacy, err := jsonv1.Marshal(collections{})
		if err != nil {
			t.Fatal(err)
		}
		owned, err := codec.Marshal(collections{}, codec.DefaultBounds())
		if err != nil {
			t.Fatal(err)
		}
		if string(legacy) != `{"list":null,"map":null}` || string(owned) != `{"list":[],"map":{}}` {
			t.Errorf("FAIL v1/v2 nil collection distinction: %s / %s", legacy, owned)
		}
		fromSDK, err := FromSDK[jsontext.Value](&collections{})
		if err != nil || fromSDK == nil || !bytes.Equal(*fromSDK, legacy) {
			t.Errorf("FAIL bridge legacy collection policy: %v: %v", fromSDK, err)
		}
		empty, err := codec.Marshal(collections{List: []string{}, Map: map[string]string{}}, codec.DefaultBounds())
		if err != nil || !bytes.Equal(empty, owned) {
			t.Errorf("FAIL native v2 empty collection encoding: %s: %v", empty, err)
		}
	})
	t.Run("custom-marshalers-and-byte-versus-semantic-identity", func(t *testing.T) {
		legacy, err := jsonv1.Marshal(differentialHooks{})
		if err != nil {
			t.Fatal(err)
		}
		owned, err := codec.Marshal(differentialHooks{}, codec.DefaultBounds())
		if err != nil {
			t.Fatal(err)
		}
		// The pinned v1 compatibility package also prefers MarshalJSONTo when
		// a value implements both interfaces. Test real legacy-only hooks too.
		if string(legacy) != `"native"` || string(owned) != `"native"` {
			t.Errorf("FAIL codec hook ownership: %s / %s", legacy, owned)
		}
		fromSDK, err := FromSDK[string](&legacyOnlyHooks{})
		if err != nil || fromSDK == nil || *fromSDK != "legacy" {
			t.Errorf("FAIL bridge bypassed SDK legacy marshaler: %v %v", fromSDK, err)
		}
		legacyOnly, err := codec.Marshal(legacyOnlyHooks{}, codec.DefaultBounds())
		if err != nil || string(legacyOnly) != `"legacy"` {
			t.Errorf("FAIL native codec ignored legacy-only marshaler: %s %v", legacyOnly, err)
		}
		raw := jsontext.Value(`{"n":9007199254740993,"s":"<>&"}`)
		body, err := codec.Marshal(raw, codec.DefaultBounds())
		if err != nil || !bytes.Equal(body, raw) {
			t.Errorf("FAIL required raw byte identity: %s: %v", body, err)
		}
		formatted := []byte("{ \"n\" : 9007199254740993, \"s\":\"<>&\" }")
		if bytes.Equal(body, formatted) || !reflect.DeepEqual(exactValue(t, body), exactValue(t, formatted)) {
			t.Error("FAIL byte identity conflated with semantic equality")
		}
	})
	t.Run("document-sdk-dynamic-number-boundary", func(t *testing.T) {
		frame := []byte(`{"resultType":"complete","content":[],"structuredContent":{"n":9007199254740993}}`)
		var owned protocol.CallToolResult
		if err := codec.Unmarshal(frame, &owned, codec.DefaultBounds()); err != nil {
			t.Fatal(err)
		}
		raw, present := owned.StructuredContent.Get()
		if !present || string(raw) != `{"n":9007199254740993}` {
			t.Fatal("FAIL owned dynamic payload rounded")
		}
		var upstream sdk.CallToolResult
		if err := jsonv1.Unmarshal(frame, &upstream); err != nil {
			t.Fatal(err)
		}
		value, ok := upstream.StructuredContent.(map[string]any)["n"].(float64)
		if !ok || value != 9007199254740992 {
			t.Fatal("FAIL pinned SDK dynamic-number behavior changed; reassess boundary")
		}
		// This is a differential finding, never evidence for a lossless SDK
		// transport claim. Owned raw values remain the canonical validation input.
	})
}

type differentialHooks struct{}

type legacyOnlyHooks struct{}

func (legacyOnlyHooks) MarshalJSON() ([]byte, error) { return []byte(`"legacy"`), nil }

func (differentialHooks) MarshalJSON() ([]byte, error) { return []byte(`"legacy"`), nil }
func (differentialHooks) MarshalJSONTo(encoder *jsontext.Encoder) error {
	return json.MarshalEncode(encoder, "native")
}
