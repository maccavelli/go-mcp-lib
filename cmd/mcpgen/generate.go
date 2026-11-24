package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/emit"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/model"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

// SchemaToProtocol is the in-process pipeline from the pinned schema to
// the generated protocol package: load, model, disposition, emit. It is
// the engine behind `mcpgen generate` in stage 4; stage 3 exposes it to
// qualification directly.
func SchemaToProtocol(schemaPath, sdkDir, revision, outDir string) (*emit.Manifest, error) {
	doc, err := load.New().LoadFile(schemaPath)
	if err != nil {
		return nil, err
	}
	m, err := model.Build(doc)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(m.Definitions))
	for name := range m.Definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	dispositions, err := sdkmap.Build(revision, sdkDir, "v1.8.0", names)
	if err != nil {
		return nil, err
	}
	files, manifest, err := emit.Render(emit.Options{
		Revision:     revision,
		Model:        m,
		Dispositions: dispositions,
		PackagePath:  "github.com/maccavelli/go-mcp-lib/protocol",
	})
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	for _, file := range files {
		formatted, err := emit.FormatGo(file.Name, file.Content)
		if err != nil {
			return nil, fmt.Errorf("generate: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, file.Name), formatted, 0o644); err != nil {
			return nil, err
		}
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "generated-manifest.json"), append(manifestData, '\n'), 0o644); err != nil {
		return nil, err
	}
	return manifest, nil
}
