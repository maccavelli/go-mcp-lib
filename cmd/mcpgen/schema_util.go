package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

// schemaDoc is the minimal schema reader for disposition computation.
type schemaDoc struct {
	Defs map[string]json.RawMessage `json:"$defs"`
}

func loadSchema(path string) (*schemaDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	// The strict loader validates before decode; reuse it.
	doc, err := load.New().LoadBytes(path, data)
	if err != nil {
		return nil, err
	}
	root, ok := doc.Root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema: root is not an object")
	}
	defs, ok := root["$defs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema: no $defs")
	}
	out := &schemaDoc{Defs: make(map[string]json.RawMessage, len(defs))}
	for name := range defs {
		out.Defs[name] = nil
	}
	return out, nil
}
