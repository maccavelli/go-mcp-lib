// Package sdkbridge converts private SDK values through their wire codecs.
package sdkbridge

import (
	legacyjson "encoding/json"
	json "encoding/json/v2"
	"errors"
	"fmt"
)

// FromSDK honors the SDK's legacy JSON hooks before invoking the owned hooks.
// A nil SDK success is an invalid result, rather than an empty owned value.
func FromSDK[Owned, SDK any](source *SDK) (*Owned, error) {
	if source == nil {
		return nil, errors.New("mcp: SDK returned a nil result")
	}
	body, err := legacyjson.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("mcp: encode SDK value: %w", err)
	}
	var result Owned
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("mcp: decode owned value: %w", err)
	}
	return &result, nil
}

// ToSDK validates the owned wire representation before invoking the SDK hooks.
func ToSDK[SDK, Owned any](source *Owned) (*SDK, error) {
	if source == nil {
		return nil, errors.New("mcp: owned value is nil")
	}
	body, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("mcp: encode owned value: %w", err)
	}
	var result SDK
	if err := legacyjson.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("mcp: decode SDK value: %w", err)
	}
	return &result, nil
}
