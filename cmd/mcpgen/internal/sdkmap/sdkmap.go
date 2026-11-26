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
	"os"
	"sort"
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
	"ModelHint", "ModelPreferences",
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

// OwnershipPolicy is independent of SDK deprecation facts. Meta is a modern
// SDK type, but framework-owned metadata must retain the framework's wire API.
// Sampling/roots codec ownership remains pinned by Deprecations and Sealed.
var OwnershipPolicy = map[string]string{"Meta": "owned"}

// Result is the mechanical disposition table.
type Result struct {
	Revision     string            `json:"revision"`
	SDKModule    string            `json:"sdk-module"`
	SDKVersion   string            `json:"sdk-version"`
	Dispositions map[string]string `json:"dispositions"`
	Deprecations []string          `json:"sdk-deprecations-verified"`
	AliasAudit   map[string]string `json:"alias-audit"`
	SourceSHA256 string            `json:"sdk-context-sha256"`
	SDKSum       string            `json:"sdk-module-sum"`
	SDKExports   []string          `json:"sdk-exports-verified"`
}

// Build computes dispositions for every definition in defs against the
// SDK package directory. It fails on an unknown disposition state; there
// is no "skip" outcome.
func Build(revision, sdkDir, sdkVersion string, defs []string) (*Result, error) {
	root, err := RuntimeModule(".")
	if err != nil {
		return nil, err
	}
	return BuildInModule(revision, sdkDir, sdkVersion, root, defs)
}

// BuildInModule audits the SDK under the runtime module's exact dependency graph.
func BuildInModule(revision, sdkDir, sdkVersion, root string, defs []string) (*Result, error) {
	if revision == "" || sdkVersion != "v1.8.0" {
		return nil, fmt.Errorf("sdkmap: unsupported or missing mapping identity")
	}
	if len(defs) == 0 {
		return nil, fmt.Errorf("sdkmap: no definitions supplied")
	}
	audit, err := auditSDK(root, sdkDir)
	if err != nil {
		return nil, err
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
		case OwnershipPolicy[def] != "":
			dispositions[def] = OwnershipPolicy[def]
		case sealed[def]:
			dispositions[def] = "owned-codec"
		case deprecations[def]:
			dispositions[def] = "owned-codec"
		case generics[def]:
			dispositions[def] = "owned"
		case audit.available[def]:
			if audit.reasons[def] != "" {
				dispositions[def] = "owned"
			} else {
				dispositions[def] = "alias"
			}
		default:
			dispositions[def] = "owned"
		}
	}
	// Verify source deprecation facts separately from ownership policy.
	for _, name := range Deprecations {
		if !audit.available[name] {
			return nil, fmt.Errorf("sdkmap: frozen deprecated type %s is unavailable on a supported target", name)
		}
		if !audit.deprecated[name] {
			return nil, fmt.Errorf("sdkmap: frozen deprecated type %s has no source deprecation marker on a supported target", name)
		}
	}
	verified := append([]string(nil), Deprecations...)
	sort.Strings(verified)
	reasons := make(map[string]string)
	for _, name := range sortedReasons(audit.reasons) {
		reasons[name] = audit.reasons[name]
	}
	exports := make([]string, 0, len(audit.available))
	for name, available := range audit.available {
		if available {
			exports = append(exports, name)
		}
	}
	sort.Strings(exports)
	return &Result{
		Revision:     revision,
		SDKModule:    "github.com/modelcontextprotocol/go-sdk",
		SDKVersion:   sdkVersion,
		Dispositions: dispositions,
		Deprecations: verified,
		AliasAudit:   reasons,
		SourceSHA256: audit.digest,
		SDKSum:       audit.sum,
		SDKExports:   exports,
	}, nil
}

// WriteJSON serializes the result deterministically.
func (r *Result) WriteJSON(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
