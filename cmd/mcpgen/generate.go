package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/emit"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/model"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/profile"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/resolve"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

// SchemaToProtocol is the in-process pipeline from the pinned schema to
// the generated protocol package: load, model, disposition, emit. It is
// the engine behind `mcpgen generate` in stage 4; stage 3 exposes it to
// qualification directly.
func SchemaToProtocol(schemaPath, sdkDir, revision, outDir string) (*emit.Manifest, error) {
	return schemaToProtocol(schemaPath, sdkDir, revision, outDir, nil)
}

func schemaToProtocol(schemaPath, sdkDir, revision, outDir string, inputs *profile.Inputs) (*emit.Manifest, error) {
	return schemaToProtocolAudit(schemaPath, sdkDir, revision, outDir, inputs, nil)
}

func schemaToProtocolAudit(schemaPath, sdkDir, revision, outDir string, inputs *profile.Inputs, audit func(*sdkmap.Result)) (*emit.Manifest, error) {
	absolute, err := filepath.Abs(schemaPath)
	if err != nil {
		return nil, err
	}
	moduleRoot, err := sdkmap.RuntimeModule(filepath.Dir(absolute))
	if err != nil {
		return nil, err
	}
	if absolute != filepath.Join(moduleRoot, filepath.FromSlash(profile.SchemaFile)) {
		return nil, fmt.Errorf("generate: schema path is not the reviewed input")
	}
	if inputs == nil {
		inputs, err = profile.Load(moduleRoot, revision)
		if err != nil {
			return nil, err
		}
	}
	doc := inputs.Schema
	canonical := (&url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(filepath.ToSlash(absolute), "/")}).String()
	registry := resolve.NewRegistry()
	if err := registry.Add(canonical, doc); err != nil {
		return nil, err
	}
	if err := registry.Validate(); err != nil {
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
	dispositions, err := sdkmap.BuildInModule(revision, sdkDir, "v1.8.0", moduleRoot, names)
	if err != nil {
		return nil, err
	}
	dispositions, err = inputs.ApplySDK(dispositions)
	if err != nil {
		return nil, err
	}
	if audit != nil {
		audit(dispositions)
	}
	files, manifest, err := emit.Render(emit.Options{
		Revision:     revision,
		Model:        m,
		Dispositions: dispositions,
		PackagePath:  "github.com/maccavelli/go-mcp-lib/protocol",
		Names:        inputs.Mapping.GoNames,
		Methods:      inputs.Methods,
	})
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(outDir, file.Name), file.Content, 0o644); err != nil {
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
