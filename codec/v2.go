package codec

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	json "encoding/json/v2"
)

// Marshal encodes native JSON v2 semantics and checks the explicit raw bounds.
func Marshal(value any, bounds Bounds) ([]byte, error) {
	if err := validBounds(bounds); err != nil {
		return nil, err
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if _, err := ValidateRaw(body, bounds); err != nil {
		return nil, err
	}
	return body, nil
}

var exactAny = json.UnmarshalFromFunc(func(decoder *jsontext.Decoder, value *any) error {
	body, err := decoder.ReadValue()
	if err != nil {
		return err
	}
	// The raw guard already established syntax, uniqueness and resource bounds.
	// json.Number is the pinned validator's lossless dynamic-number contract;
	// typed fields and custom hooks still use native v2 unmarshaling.
	projection := jsonv1.NewDecoder(bytes.NewReader(body))
	projection.UseNumber()
	return projection.Decode(value)
})

// Unmarshal checks raw input before native JSON v2 decoding. Dynamic numbers
// remain exact json.Number tokens rather than passing through float64.
func Unmarshal(data []byte, value any, bounds Bounds) error {
	if _, err := ValidateRaw(data, bounds); err != nil {
		return err
	}
	return json.Unmarshal(data, value, json.WithUnmarshalers(exactAny))
}
