package sdkbridge

import (
	"bytes"
	legacyjson "encoding/json"
	"encoding/json/jsontext"
	"errors"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSDKWireConversion(t *testing.T) {
	t.Run("custom-content-and-discriminator", func(t *testing.T) {
		var source sdk.CallToolResult
		if err := legacyjson.Unmarshal([]byte(`{"resultType":"input_required","content":[{"type":"text","text":"hello"}],"inputRequests":{},"requestState":"opaque"}`), &source); err != nil {
			t.Fatal(err)
		}
		body, err := FromSDK[jsontext.Value](&source)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`"resultType":"input_required"`, `"type":"text"`, `"text":"hello"`, `"requestState":"opaque"`} {
			if !bytes.Contains(*body, []byte(want)) {
				t.Errorf("FAIL SDK custom wire field lost: %s: %s", want, *body)
			}
		}
		roundTrip, err := ToSDK[sdk.CallToolResult](body)
		if err != nil || !roundTrip.NeedsInput() || len(roundTrip.Content) != 1 {
			t.Fatalf("FAIL SDK custom decode: %+v %v", roundTrip, err)
		}
	})
	t.Run("nil", func(t *testing.T) {
		if value, err := FromSDK[jsontext.Value]((*sdk.CallToolResult)(nil)); err == nil || value != nil {
			t.Error("FAIL nil SDK success accepted")
		}
		if value, err := ToSDK[sdk.CallToolResult]((*jsontext.Value)(nil)); err == nil || value != nil {
			t.Error("FAIL nil owned value accepted")
		}
	})
	t.Run("failure-propagation", func(t *testing.T) {
		source := make(chan int)
		if value, err := FromSDK[jsontext.Value](&source); err == nil || value != nil {
			t.Error("FAIL SDK encoding failure discarded")
		}
		body := jsontext.Value(`{"x":1,"x":2}`)
		if value, err := ToSDK[map[string]int](&body); err == nil || value != nil {
			t.Error("FAIL invalid owned wire accepted")
		}
		body = jsontext.Value(`{"x":"wrong"}`)
		if value, err := ToSDK[map[string]int](&body); err == nil || value != nil {
			t.Error("FAIL SDK decoding failure discarded")
		}
		_, err := FromSDK[rejectOwned](&struct{}{})
		if !errors.Is(err, errOwnedRejected) {
			t.Errorf("FAIL owned decoding error identity lost: %v", err)
		}
	})
}

var errOwnedRejected = errors.New("fixture owned rejection")

type rejectOwned struct{}

func (*rejectOwned) UnmarshalJSONFrom(*jsontext.Decoder) error { return errOwnedRejected }
