// Package profile verifies independently pinned generation inputs before any
// candidate output is created. Ownership policy and SDK source facts are separate.
package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/sdkmap"
)

const profileFile = "schema/generation-profile.json"

// SchemaFile is the single reviewed source document for this profile.
const SchemaFile = "schema/inputs/2026-07-28/schema.json"

var requiredFiles = map[string]int{
	SchemaFile: 0, "schema/inputs/2026-07-28/manifest.json": 1,
	"schema/sdk-map.json": 2, "schema/semantics.json": 2,
	"tools/versions.json": 0, "go.mod": 0, "go.sum": 0,
}

// Error identifies a bounded profile failure using a logical input name.
type Error struct{ Input, Code, Detail string }

func (e *Error) Error() string {
	text := fmt.Sprintf("profile: %s: %s: %s", e.Input, e.Code, e.Detail)
	if len(text) <= 256 {
		return text
	}
	cut := 253
	for !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut] + "..."
}

func failure(input, code, detail string) error { return &Error{input, code, detail} }

type fileIdentity struct {
	SHA256  string `json:"sha256"`
	Bytes   int    `json:"bytes"`
	Version int    `json:"version"`
}

type manifest struct {
	Version  int                     `json:"version"`
	Revision string                  `json:"revision"`
	Files    map[string]fileIdentity `json:"files"`
}

// Mapping is the complete reviewed ownership and Go-name input.
type Mapping struct {
	Version      int               `json:"version"`
	Revision     string            `json:"revision"`
	Module       string            `json:"module"`
	SDKVersion   string            `json:"version-pinned"`
	SDKSum       string            `json:"sdk-module-sum"`
	SDKContext   string            `json:"sdk-context-sha256"`
	Dispositions map[string]string `json:"dispositions"`
	Reasons      map[string]string `json:"reasons"`
	GoNames      map[string]string `json:"go-names"`
}

// Method binds a canonical wire name to its schema and direction policy.
type Method struct {
	Name               string `json:"name"`
	Definition         string `json:"definition"`
	Pointer            string `json:"schema-pointer"`
	Direction          string `json:"direction"`
	SDKSymbol          string `json:"sdk-symbol"`
	SDKReference       string `json:"sdk-reference"`
	Capability         string `json:"capability"`
	CapabilityScope    string `json:"capability-scope"`
	CapabilityPointer  string `json:"capability-schema-pointer"`
	Completion         string `json:"completion"`
	Cancellation       *bool  `json:"cancellation"`
	CancellationScope  string `json:"cancellation-scope"`
	RetrySafe          *bool  `json:"retry-safe"`
	RetryPolicy        string `json:"retry-policy"`
	Fixture            string `json:"fixture"`
	NormativeSection   string `json:"normative-section"`
	NormativeReference string `json:"normative-reference"`
	ContractSource     string `json:"contract-source"`
	QualificationOwner string `json:"qualification-owner"`
	Conditions         string `json:"conditions"`
}

type semantics struct {
	Methods []Method `json:"methods"`
}

// Inputs holds verified immutable document snapshots for one generation.
type Inputs struct {
	Schema  *load.Document
	Mapping Mapping
	Methods []Method
}

func digest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func read(root *os.Root, name string) ([]byte, error) {
	file, err := root.Open(name)
	if err != nil {
		code := "unreadable-input"
		if errors.Is(err, os.ErrNotExist) {
			code = "missing-input"
		}
		return nil, failure(name, code, "required reviewed input cannot be opened")
	}
	defer func() { _ = file.Close() }()
	const limit = 2 << 20
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, failure(name, "unreadable-input", "input read failed")
	}
	if len(body) > limit {
		return nil, failure(name, "input-too-large", "input exceeds 2 MiB")
	}
	return body, nil
}

func decode(name string, body []byte, output any) error {
	if _, err := load.New().LoadBytes(name, body); err != nil {
		return failure(name, "invalid-input", err.Error())
	}
	if err := json.Unmarshal(body, output); err != nil {
		return failure(name, "invalid-input", "reviewed document has an invalid shape")
	}
	return nil
}

// Load verifies the profile's source pin, then every required member's bytes and
// format version. os.Root keeps local resource acquisition inside the repository.
func Load(directory, revision string) (*Inputs, error) {
	if revision != Revision {
		return nil, failure(profileFile, "unsupported-revision", "revision is not reviewed")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, failure(profileFile, "unreadable-root", "repository root cannot be opened")
	}
	defer func() { _ = root.Close() }()
	body, err := read(root, profileFile)
	if err != nil {
		return nil, err
	}
	if digest(body) != ProfileSHA256 {
		return nil, failure(profileFile, "unreviewed-input", "profile differs from its independent source pin")
	}
	var profile manifest
	if err := decode(profileFile, body, &profile); err != nil {
		return nil, err
	}
	if profile.Version != 1 || profile.Revision != Revision || len(profile.Files) != len(requiredFiles) {
		return nil, failure(profileFile, "invalid-input", "profile identity or member inventory differs")
	}
	names := make([]string, 0, len(requiredFiles))
	for name := range requiredFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	documents := make(map[string][]byte)
	for _, name := range names {
		identity, found := profile.Files[name]
		if !found || identity.Version != requiredFiles[name] {
			return nil, failure(name, "invalid-input", "required profile member or version differs")
		}
		body, err := read(root, name)
		if err != nil {
			return nil, err
		}
		if len(body) != identity.Bytes || digest(body) != identity.SHA256 {
			return nil, failure(name, "unreviewed-input", "input differs from its reviewed identity")
		}
		if strings.HasSuffix(name, ".json") {
			var header struct {
				Version  int    `json:"version"`
				Revision string `json:"revision"`
			}
			if err := decode(name, body, &header); err != nil {
				return nil, err
			}
			if identity.Version != 0 && (header.Version != identity.Version || header.Revision != Revision) {
				return nil, failure(name, "invalid-input", "member version or revision differs")
			}
		}
		documents[name] = body
	}
	if digest(documents[SchemaFile]) != SchemaSHA256 {
		return nil, failure(SchemaFile, "unreviewed-input", "schema differs from the independent protocol pin")
	}
	doc, err := load.New().LoadBytes(SchemaFile, documents[SchemaFile])
	if err != nil {
		return nil, err
	}
	inputs := &Inputs{Schema: doc}
	if err := json.Unmarshal(documents["schema/sdk-map.json"], &inputs.Mapping); err != nil {
		return nil, failure("schema/sdk-map.json", "invalid-input", "mapping shape differs")
	}
	var rules semantics
	if err := json.Unmarshal(documents["schema/semantics.json"], &rules); err != nil {
		return nil, failure("schema/semantics.json", "invalid-input", "semantic map shape differs")
	}
	inputs.Methods = rules.Methods
	if err := inputs.validate(); err != nil {
		return nil, err
	}
	return inputs, nil
}

func (i *Inputs) validate() error {
	if i == nil || i.Schema == nil {
		return failure(SchemaFile, "invalid-input", "schema snapshot is missing")
	}
	root, _ := i.Schema.Root.(map[string]any)
	definitions, ok := root["$defs"].(map[string]any)
	if !ok || len(definitions) != 155 || len(i.Mapping.Dispositions) != len(definitions) || len(i.Mapping.GoNames) != len(definitions) || len(i.Mapping.Reasons) != len(definitions) {
		return failure("schema/sdk-map.json", "invalid-mapping", "every definition requires a disposition, name and reason")
	}
	names := make([]string, 0, len(definitions))
	for name := range definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	seenNames := make(map[string]bool)
	expectedMethods := make(map[string]map[string]bool)
	for _, name := range names {
		disposition := i.Mapping.Dispositions[name]
		if disposition != "alias" && disposition != "owned" && disposition != "owned-codec" || i.Mapping.Reasons[name] == "" {
			return failure("schema/sdk-map.json", "invalid-mapping", "definition lacks a supported reviewed disposition: "+name)
		}
		goName := i.Mapping.GoNames[name]
		first, _ := utf8.DecodeRuneInString(goName)
		if !token.IsIdentifier(goName) || !unicode.IsUpper(first) || seenNames[strings.ToLower(goName)] {
			return failure("schema/sdk-map.json", "invalid-mapping", "invalid or colliding Go name: "+name)
		}
		seenNames[strings.ToLower(goName)] = true
		definition, _ := definitions[name].(map[string]any)
		properties, _ := definition["properties"].(map[string]any)
		method, _ := properties["method"].(map[string]any)
		if wireName, ok := method["const"].(string); ok {
			if expectedMethods[wireName] == nil {
				expectedMethods[wireName] = make(map[string]bool)
			}
			expectedMethods[wireName][name] = true
		}
	}
	if len(i.Methods) != len(expectedMethods) {
		return failure("schema/semantics.json", "invalid-mapping", "canonical method inventory differs")
	}
	seenMethods := make(map[string]bool)
	for _, method := range i.Methods {
		if !expectedMethods[method.Name][method.Definition] || method.Direction == "" || method.Pointer != "#/$defs/"+method.Definition || seenMethods[method.Name] {
			return failure("schema/semantics.json", "invalid-mapping", "missing, duplicate or unbound method: "+method.Name)
		}
		if !completeMethodPolicy(method, definitions) {
			return failure("schema/semantics.json", "invalid-mapping", "incomplete or inconsistent method policy: "+method.Name)
		}
		seenMethods[method.Name] = true
	}
	return nil
}

func completeMethodPolicy(method Method, definitions map[string]any) bool {
	if !strings.HasPrefix(method.SDKSymbol, "mcp.") || !token.IsIdentifier(strings.TrimPrefix(method.SDKSymbol, "mcp.")) ||
		method.SDKReference != "https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/protocol.go" ||
		method.NormativeSection == "" || !strings.HasPrefix(method.NormativeReference, "https://modelcontextprotocol.io/specification/"+Revision+"/") ||
		method.ContractSource != SchemaFile+method.Pointer || method.QualificationOwner != "R5: TestRepairA19CanonicalRuntime" ||
		method.Cancellation == nil || method.RetrySafe == nil || method.RetryPolicy != "caller-controlled; no automatic retries" ||
		method.Fixture == "" || method.Conditions == "" {
		return false
	}
	completion, cancellationScope := "single-result", "client-request"
	switch {
	case strings.HasSuffix(method.Definition, "Notification"):
		completion, cancellationScope = "notification", "not-applicable"
		if *method.Cancellation || *method.RetrySafe {
			return false
		}
	case method.Name == "subscriptions/listen":
		completion = "subscription-stream"
	case method.Name == "tools/call" || method.Name == "prompts/get" || method.Name == "resources/read":
		completion = "single-result-or-input-required"
	case method.Name == "elicitation/create" || method.Name == "sampling/createMessage" || method.Name == "roots/list":
		cancellationScope = "parent-context"
	}
	if method.Completion != completion || method.CancellationScope != cancellationScope {
		return false
	}
	switch method.CapabilityScope {
	case "none":
		return method.Capability == "none" && method.CapabilityPointer == "none"
	case "request":
		return method.Name == "notifications/progress" && method.Capability == "progressToken" && method.CapabilityPointer == "#/$defs/RequestMetaObject/properties/progressToken"
	case "server", "client":
		definition := "ServerCapabilities"
		if method.CapabilityScope == "client" {
			definition = "ClientCapabilities"
		}
		pointer := "#/$defs/" + definition
		value, _ := definitions[definition].(map[string]any)
		for _, member := range strings.Split(method.Capability, ".") {
			properties, _ := value["properties"].(map[string]any)
			next, ok := properties[member].(map[string]any)
			if !ok {
				return false
			}
			value = next
			pointer += "/properties/" + member
		}
		return pointer == method.CapabilityPointer
	default:
		return false
	}
}

// ApplySDK rejects SDK-context drift and unsafe aliases, then applies reviewed
// ownership policy. A protocol-level codec DTO need not have a same-name SDK type.
func (i *Inputs) ApplySDK(actual *sdkmap.Result) (*sdkmap.Result, error) {
	if actual == nil || actual.Revision != Revision || actual.SDKModule != i.Mapping.Module || actual.SDKVersion != i.Mapping.SDKVersion || actual.SDKSum != i.Mapping.SDKSum || actual.SourceSHA256 != i.Mapping.SDKContext || len(actual.Dispositions) != len(i.Mapping.Dispositions) {
		return nil, failure("schema/sdk-map.json", "sdk-context-drift", "SDK source/module identity differs")
	}
	exports := make(map[string]bool, len(actual.SDKExports))
	for _, name := range actual.SDKExports {
		exports[name] = true
	}
	for _, method := range i.Methods {
		if !exports[strings.TrimPrefix(method.SDKSymbol, "mcp.")] {
			return nil, failure("schema/semantics.json", "sdk-symbol-unavailable", "method SDK type is unavailable on a supported target: "+method.Name)
		}
	}
	result := *actual
	result.Dispositions = make(map[string]string, len(i.Mapping.Dispositions))
	for name, disposition := range i.Mapping.Dispositions {
		observed, found := actual.Dispositions[name]
		if !found || disposition == "alias" && observed != "alias" || observed == "owned-codec" && disposition != "owned-codec" {
			return nil, failure("schema/sdk-map.json", "unsafe-mapping", "SDK alias or required codec ownership differs: "+name)
		}
		result.Dispositions[name] = disposition
	}
	return &result, nil
}
