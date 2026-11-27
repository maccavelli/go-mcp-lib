// Package validation owns the bounded Draft 2020-12 validation profile. Schema
// snapshots and exact numeric projections are immutable; references resolve
// exclusively from the local registry. SDK validation is a separate boundary.
package validation

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type resource struct {
	document any
	body     []byte
	protocol bool
}

// Adapter retains immutable registered documents and content-keyed compilations.
// Its mutex serializes publication, including concurrent first compilations.
type Adapter struct {
	mu        sync.Mutex
	resources map[string]resource
	compiled  map[string]*Compiled
	hits      map[string]int
	maxCache  int
}

// New creates bounded registry and compilation caches; nonpositive uses 256.
func New(maxCache int) *Adapter {
	if maxCache <= 0 {
		maxCache = 256
	}
	return &Adapter{resources: make(map[string]resource), compiled: make(map[string]*Compiled),
		hits: make(map[string]int), maxCache: maxCache}
}

func snapshot(document any) (resource, error) {
	body, err := json.Marshal(document)
	if err != nil {
		return resource{}, fmt.Errorf("validation: snapshot: %w", err)
	}
	var owned any
	if err := exactUnmarshal(string(body), &owned); err != nil {
		return resource{}, fmt.Errorf("validation: snapshot: %w", err)
	}
	return resource{document: owned, body: body}, nil
}

// Register snapshots a document before retaining it under its canonical URI.
func (a *Adapter) Register(canonical string, doc any) error {
	owned, err := snapshot(doc)
	if err != nil {
		return err
	}
	return a.register(canonical, owned)
}

func (a *Adapter) register(canonical string, owned resource) error {
	if err := CheckURI(canonical); err != nil {
		return fmt.Errorf("validation: canonical resource URI: %w", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.resources[canonical]; exists {
		return fmt.Errorf("validation: duplicate canonical URI %s", canonical)
	}
	if len(a.resources) >= a.maxCache {
		return fmt.Errorf("validation: resource registry full (%d entries)", a.maxCache)
	}
	a.resources[canonical] = owned
	return nil
}

// Compile snapshots a schema and caches by caller identity, canonical URI,
// schema bytes, the complete registry snapshot and the fixed profile version.
// Changed content under a reused caller identity cannot reuse a stale schema.
func (a *Adapter) Compile(identity, canonical string, doc any) (*Compiled, error) {
	owned, err := snapshot(doc)
	if err != nil {
		return nil, err
	}
	if err := CheckURI(canonical); err != nil {
		return nil, fmt.Errorf("validation: canonical schema URI: %w", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	key := a.cacheKey(identity, canonical, owned.body)
	if cached, ok := a.compiled[key]; ok {
		a.hits[identity]++
		return cached, nil
	}
	if len(a.compiled) >= a.maxCache {
		return nil, fmt.Errorf("validation: compilation cache full (%d entries); raise the bound deliberately, not silently", a.maxCache)
	}
	if err := checkProfile(owned.document, canonical); err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(registryLoader(a.resources))
	compiler.UseRegexpEngine(compilePattern)
	installProtocolRules(compiler, a.resources)
	if err := compiler.AddResource(canonical, owned.document); err != nil {
		return nil, fmt.Errorf("validation: compile %s: %w", identity, shortErr(err))
	}
	schema, err := compiler.Compile(canonical)
	if err != nil {
		return nil, fmt.Errorf("validation: compile %s: %w", identity, shortErr(err))
	}
	wrapped, err := wrapCompiled(schema)
	if err != nil {
		return nil, err
	}
	a.compiled[key] = wrapped
	// Bound this diagnostic map by the same successful compilation population.
	if _, ok := a.hits[identity]; !ok {
		a.hits[identity] = 0
	}
	return wrapped, nil
}

func (a *Adapter) cacheKey(identity, canonical string, body []byte) string {
	// JSON encoding length-delimits every component, including caller strings.
	parts := []any{"draft2020-exact-local-protocol-v1", identity, canonical, string(body)}
	for _, uri := range sortedKeys(a.resources) {
		r := a.resources[uri]
		parts = append(parts, uri, string(r.body), r.protocol)
	}
	encoded, err := json.Marshal(parts) // only strings and booleans
	if err != nil {
		panic(err) // no unsupported values can enter this private tuple
	}
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type registryLoader map[string]resource

func (r registryLoader) Load(uri string) (any, error) {
	entry, ok := r[uri]
	if !ok {
		return nil, fmt.Errorf("validation: unregistered reference %q; external loading is prohibited", clip(uri))
	}
	if err := checkProfile(entry.document, uri); err != nil {
		return nil, err
	}
	return entry.document, nil
}

// Validate checks a decoded instance against the immutable compiled schema.
func (a *Adapter) Validate(schema *Compiled, instance any) error {
	return schema.Validate(instance)
}
