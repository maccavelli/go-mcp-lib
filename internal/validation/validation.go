// Package validation is the private canonical validator adapter over
// the pinned github.com/santhosh-tekuri/jsonschema/v6.
//
// Compilations are cached per immutable schema identity with retention
// bounds; concurrent access is safe. Draft 2020-12 is explicit; local
// reference resolution uses resources the caller registers. Numeric
// values stay exact: validation operates on decoded values with
// json.Number tokens, never float64. This is the canonical path; the
// SDK's internal validation is distinct and never substituted here.
package validation

import (
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Adapter caches compiled schemas by immutable identity (the raw schema
// bytes' hash or another caller-chosen stable key).
type Adapter struct {
	mu        sync.Mutex
	resources map[string]any
	compiled  map[string]*Compiled
	hits      map[string]int
	maxCache  int
}

// New creates an adapter with an empty local resource registry and a
// bounded compilation cache.
func New(maxCache int) *Adapter {
	if maxCache <= 0 {
		maxCache = 256
	}
	return &Adapter{
		resources: make(map[string]any),
		compiled:  make(map[string]*Compiled),
		hits:      make(map[string]int),
		maxCache:  maxCache,
	}
}

// Register adds a local document to the reference registry under its
// canonical URI.
func (a *Adapter) Register(canonical string, doc any) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.resources[canonical]; exists {
		return fmt.Errorf("validation: duplicate canonical URI %s", canonical)
	}
	a.resources[canonical] = doc
	return nil
}

// Compile validates a schema document and caches it under identity.
// Draft 2020-12 is forced when the document does not declare one.
// The returned Compiled is opaque; callers never see the validator's
// concrete types.
func (a *Adapter) Compile(identity, canonical string, doc any) (*Compiled, error) {
	a.mu.Lock()
	if schema, ok := a.compiled[identity]; ok {
		a.hits[identity]++
		a.mu.Unlock()
		return schema, nil
	}
	a.mu.Unlock()
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	a.mu.Lock()
	for uri, res := range a.resources {
		if err := compiler.AddResource(uri, res); err != nil {
			a.mu.Unlock()
			return nil, fmt.Errorf("validation: resource %s: %w", uri, shortErr(err))
		}
	}
	if err := compiler.AddResource(canonical, doc); err != nil {
		a.mu.Unlock()
		return nil, fmt.Errorf("validation: compile %s: %w", identity, shortErr(err))
	}
	a.mu.Unlock()
	schema, err := compiler.Compile(canonical)
	if err != nil {
		return nil, fmt.Errorf("validation: compile %s: %w", identity, shortErr(err))
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.compiled) >= a.maxCache {
		return nil, fmt.Errorf(
			"validation: compilation cache full (%d entries); raise the bound deliberately, not silently",
			a.maxCache,
		)
	}
	wrapped, err := wrapCompiled(schema)
	if err != nil {
		return nil, fmt.Errorf("validation: compile %s: %w", identity, err)
	}
	a.compiled[identity] = wrapped
	return wrapped, nil
}

// Validate checks a decoded instance against a compiled schema.
func (a *Adapter) Validate(schema *Compiled, instance any) error {
	return schema.Validate(instance)
}

func shortErr(err error) error {
	message := err.Error()
	if len(message) > 300 {
		return fmt.Errorf("%s…", message[:300])
	}
	return err
}
