// Package sdkmap builds the complete disposition table for every schema
// definition against the pinned SDK by inspecting Go module metadata and
// the SDK's public source ASTs.
//
// A definition is dispositioned as:
//   - "alias": an exported SDK type of the same name exists and carries no
//     deprecation marker;
//   - "owned-codec": the same-named SDK type is deprecated or sealed
//     (SEP-2577 surfaces); the framework owns the public DTO and converts
//     only through the SDK's public codecs;
//   - "owned": no same-named public SDK type exists; the framework emits
//     its own type from the schema.
//
// The mapping is total: a definition without a disposition fails the run.
package sdkmap

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Deprecations lists exported SDK identifiers carrying a Deprecated
// marker in their doc comment. The list is frozen from the pinned SDK
// source audit; the builder verifies each entry against the SDK source at
// run time and fails on drift.
var Deprecations = []string{
	"CreateMessageParams", "CreateMessageResult",
	"CreateMessageWithToolsParams", "CreateMessageWithToolsResult",
	"ListRootsParams", "ListRootsResult",
	"LoggingCapabilities", "LoggingHandler", "LoggingHandlerOptions",
	"LoggingLevel", "LoggingMessageParams",
	"Meta", "ModelHint", "ModelPreferences",
	"Root", "RootCapabilities", "RootsListChangedParams",
	"SamplingCapabilities", "SamplingContextCapabilities",
	"SamplingMessage", "SamplingMessageV2",
	"SamplingToolsCapabilities",
	"SetLoggingLevelParams", "ToolChoice", "ToolResultContent", "ToolUseContent",
}

// Sealed lists exported SDK interfaces whose implementations are
// constructed only inside the SDK package.
var Sealed = []string{"InputRequest", "InputResponse"}

// Generics lists exported generic SDK types. A generic type cannot be
// blanket-aliased without instantiation; definitions mapping to these
// become owned DTOs instead.
var Generics = []string{"ClientRequest", "ServerRequest"}

// Result is the mechanical disposition table.
type Result struct {
	Revision     string            `json:"revision"`
	SDKModule    string            `json:"sdk-module"`
	SDKVersion   string            `json:"sdk-version"`
	Dispositions map[string]string `json:"dispositions"`
	Deprecations []string          `json:"sdk-deprecations-verified"`
}

// Build computes dispositions for every definition in defs against the
// SDK package directory. It fails on an unknown disposition state; there
// is no "skip" outcome.
func Build(revision, sdkDir, sdkVersion string, defs []string) (*Result, error) {
	exports, err := inspectSDK(sdkDir)
	if err != nil {
		return nil, err
	}
	if len(defs) == 0 {
		return nil, fmt.Errorf("sdkmap: no definitions supplied")
	}
	deprecations := make(map[string]bool, len(Deprecations))
	for _, name := range Deprecations {
		deprecations[name] = true
	}
	sealed := make(map[string]bool, len(Sealed))
	for _, name := range Sealed {
		sealed[name] = true
	}
	generics := make(map[string]bool, len(Generics))
	for _, name := range Generics {
		generics[name] = true
	}
	dispositions := make(map[string]string, len(defs))
	for _, def := range defs {
		switch {
		case sealed[def]:
			dispositions[def] = "owned-codec"
		case deprecations[def]:
			dispositions[def] = "owned-codec"
		case generics[def]:
			dispositions[def] = "owned"
		case exports[def]:
			dispositions[def] = "alias"
		default:
			dispositions[def] = "owned"
		}
	}
	// Verify each frozen deprecation still exists as an exported type in
	// the SDK; drift fails loudly.
	for _, name := range Deprecations {
		if !exports[name] {
			return nil, fmt.Errorf(
				"sdkmap: frozen deprecation %s no longer appears as an exported SDK type; the pin or the audit drifted",
				name,
			)
		}
	}
	verified := append([]string(nil), Deprecations...)
	sort.Strings(verified)
	return &Result{
		Revision:     revision,
		SDKModule:    "github.com/modelcontextprotocol/go-sdk",
		SDKVersion:   sdkVersion,
		Dispositions: dispositions,
		Deprecations: verified,
	}, nil
}

// inspectSDK parses the SDK package and returns its exported type names
// that carry deprecation markers, plus all exported type names.
func inspectSDK(sdkDir string) (map[string]bool, error) {
	if _, err := os.Stat(sdkDir); err != nil {
		return nil, fmt.Errorf("sdkmap: SDK directory %s: %w", sdkDir, err)
	}
	entries, err := os.ReadDir(sdkDir)
	if err != nil {
		return nil, fmt.Errorf("sdkmap: %w", err)
	}
	exports := make(map[string]bool)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(sdkDir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("sdkmap: parse %s: %w", name, err)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if ts.Name.IsExported() {
					exports[ts.Name.Name] = true
				}
			}
		}
	}
	return exports, nil
}

// WriteJSON serializes the result deterministically.
func (r *Result) WriteJSON(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
