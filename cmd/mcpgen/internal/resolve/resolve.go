// Package resolve implements local reference resolution for loaded schema
// documents.
//
// Resolution is strictly local: canonical URIs plus fragments and JSON
// Pointers resolve against a registry of loaded documents. Network
// references are refused even when a document contains an HTTP URI.
// Reference cycles become graph edges and are tolerated; the depth limit
// applies to resolution chains, not to node revisits.
package resolve

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

// Registry holds loaded documents keyed by canonical URI.
type Registry struct {
	Documents map[string]*load.Document
	Edges     int
	Limits    load.Limits
}

// NewRegistry returns an empty registry with the default limits.
func NewRegistry() *Registry {
	return &Registry{
		Documents: make(map[string]*load.Document),
		Limits:    load.DefaultLimits(),
	}
}

// Add registers a document under its canonical URI.
func (r *Registry) Add(canonical string, doc *load.Document) error {
	if _, exists := r.Documents[canonical]; exists {
		return fmt.Errorf("resolve: duplicate canonical URI %s", canonical)
	}
	r.Documents[canonical] = doc
	return nil
}

// Ref is a resolved reference target.
type Ref struct {
	Canonical string
	Pointer   string
	Value     any
}

// Resolve resolves a reference string against the registry. The reference
// may be a bare JSON Pointer ("#/..."), a canonical URI with fragment, or
// a same-document fragment. A cycle-tolerant chain budget is enforced.
func (r *Registry) Resolve(from string, ref string) (*Ref, error) {
	if r.Edges >= r.Limits.MaxReferenceEdges {
		return nil, fmt.Errorf("resolve: reference edge budget exhausted (%d)", r.Limits.MaxReferenceEdges)
	}
	r.Edges++
	canonical := from
	pointer := ""
	switch {
	case strings.HasPrefix(ref, "#/"):
		pointer = strings.TrimPrefix(ref, "#")
	case strings.HasPrefix(ref, "#"):
		pointer = "/"
	case strings.Contains(ref, "#"):
		split := strings.SplitN(ref, "#", 2)
		canonical = strings.TrimSuffix(split[0], ".json")
		pointer = strings.TrimPrefix(split[1], "")
		if pointer != "" && !strings.HasPrefix(pointer, "/") {
			pointer = "/" + pointer
		}
	default:
		canonical = strings.TrimSuffix(ref, ".json")
	}
	doc, ok := r.Documents[canonical]
	if !ok {
		return nil, fmt.Errorf("resolve: %s: unregistered canonical URI %s", from, canonical)
	}
	if pointer == "" || pointer == "/" {
		return &Ref{Canonical: canonical, Pointer: "/", Value: doc.Root}, nil
	}
	value, err := r.walk(doc, canonical, pointer)
	if err != nil {
		return nil, err
	}
	return &Ref{Canonical: canonical, Pointer: pointer, Value: value}, nil
}

// walk follows a JSON Pointer with cycle tolerance: revisiting a node in a
// pointer path is impossible (pointers are strictly descending), but
// reference chains through $ref are bounded by the depth limit rather
// than by node identity.
func (r *Registry) walk(doc *load.Document, canonical, pointer string) (any, error) {
	if err := validatePointer(pointer); err != nil {
		return nil, err
	}
	tokens := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	var current = doc.Root
	for i, token := range tokens {
		decoded := decodeToken(token)
		switch node := current.(type) {
		case map[string]any:
			next, ok := node[decoded]
			if !ok {
				return nil, fmt.Errorf(
					"resolve: %s: pointer /%s: no member %q",
					canonical, strings.Join(tokens[:i+1], "/"), decoded,
				)
			}
			current = next
		case []any:
			index, ok := arrayIndex(decoded)
			if !ok {
				return nil, fmt.Errorf(
					"resolve: %s: pointer /%s: %q is not a valid array index",
					canonical, strings.Join(tokens[:i+1], "/"), decoded,
				)
			}
			if index >= len(node) {
				return nil, fmt.Errorf(
					"resolve: %s: pointer /%s: index %d out of bounds",
					canonical, strings.Join(tokens[:i+1], "/"), index,
				)
			}
			current = node[index]
		default:
			return nil, fmt.Errorf(
				"resolve: %s: pointer /%s: cannot descend into %T",
				canonical, strings.Join(tokens[:i+1], "/"), current,
			)
		}
	}
	return current, nil
}

func validatePointer(pointer string) error {
	if !strings.HasPrefix(pointer, "/") {
		return fmt.Errorf("resolve: pointer %q must start with /", pointer)
	}
	return nil
}

func decodeToken(token string) string {
	replaced := strings.ReplaceAll(token, "~1", "/")
	return strings.ReplaceAll(replaced, "~0", "~")
}

func arrayIndex(token string) (int, bool) {
	if token == "" {
		return 0, false
	}
	value := 0
	for _, c := range token {
		if c < '0' || c > '9' {
			return 0, false
		}
		value = value*10 + int(c-'0')
	}
	if token != "0" && strings.HasPrefix(token, "0") {
		return 0, false
	}
	return value, true
}

// RefuseNetwork reports whether a reference string names a network
// target. Callers must refuse to resolve any such reference.
func RefuseNetwork(ref string) error {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return fmt.Errorf("resolve: network reference refused: %s", redact(ref))
	}
	return nil
}

// redact keeps error messages free of full external URLs while remaining
// diagnostic: the scheme and host are shown, the path is truncated.
func redact(ref string) string {
	parsed, err := url.Parse(ref)
	if err != nil {
		return "<unparseable reference>"
	}
	path := parsed.Path
	if len(path) > 12 {
		path = path[:12] + "…"
	}
	return parsed.Scheme + "://" + parsed.Host + path
}

// ResolveChain follows a chain of $ref values from an initial reference
// and enforces the depth limit. Resolve itself enforces the edge budget.
func (r *Registry) ResolveChain(from, ref string, maxDepth int) ([]*Ref, error) {
	if maxDepth <= 0 || maxDepth > r.Limits.MaxDepth {
		maxDepth = r.Limits.MaxDepth
	}
	chain := make([]*Ref, 0, 4)
	current := ref
	for range maxDepth {
		if err := RefuseNetwork(current); err != nil {
			return nil, err
		}
		resolved, err := r.Resolve(from, current)
		if err != nil {
			return nil, err
		}
		chain = append(chain, resolved)
		next, ok := nextRef(resolved.Value)
		if !ok {
			return chain, nil
		}
		current = next
	}
	return nil, fmt.Errorf("resolve: %s: reference chain exceeds %d hops", from, maxDepth)
}

func nextRef(value any) (string, bool) {
	obj, ok := value.(map[string]any)
	if !ok {
		return "", false
	}
	ref, ok := obj["$ref"].(string)
	return ref, ok
}
