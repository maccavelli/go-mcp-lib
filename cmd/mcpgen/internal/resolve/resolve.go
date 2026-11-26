// Package resolve resolves schema references against registered resources.
// URI identifiers never trigger network or filesystem acquisition.
package resolve

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
)

type resource struct {
	canonical string
	root      any
}

type edge struct{ base, reference, pointer, origin string }

// Registry records local resources, anchors and bounded reference work.
type Registry struct {
	Documents  map[string]*load.Document
	Edges      int
	Limits     load.Limits
	resources  map[string]resource
	anchors    map[string]*Ref
	references []edge
}

// NewRegistry returns an empty registry with no external loader.
func NewRegistry() *Registry {
	return &Registry{Documents: make(map[string]*load.Document), Limits: load.DefaultLimits(),
		resources: make(map[string]resource), anchors: make(map[string]*Ref)}
}

// Ref identifies a resource location without expanding recursive values.
type Ref struct {
	Canonical string
	Pointer   string
	Value     any
	base      string
}

func uri(base, reference string) (*url.URL, error) {
	if strings.ContainsAny(reference, " \t\r\n") {
		return nil, fmt.Errorf("resolve: whitespace in URI reference")
	}
	b, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("resolve: invalid base URI")
	}
	r, err := url.Parse(reference)
	if err != nil {
		return nil, fmt.Errorf("resolve: invalid URI reference")
	}
	u := b.ResolveReference(r)
	u.Scheme, u.Host = strings.ToLower(u.Scheme), strings.ToLower(u.Host)
	return u, nil
}

// Add atomically indexes IDs, anchors and references at schema locations.
func (r *Registry) Add(canonical string, doc *load.Document) error {
	if doc == nil {
		return fmt.Errorf("resolve: nil document")
	}
	u, err := uri("", canonical)
	if err != nil || u.Fragment != "" {
		return fmt.Errorf("resolve: invalid document identity")
	}
	canonical = u.String()
	if _, exists := r.Documents[canonical]; exists {
		return fmt.Errorf("resolve: duplicate canonical URI")
	}
	resources, anchors := make(map[string]resource), make(map[string]*Ref)
	prefixes := make(map[string]string)
	var references []edge
	var rootResource resource
	var walk func(any, string, string, int, bool) error
	walk = func(value any, base, pointer string, depth int, root bool) error {
		if depth > r.Limits.MaxDepth {
			return fmt.Errorf("resolve: schema indexing depth exceeded at %s", pointer)
		}
		node, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		id, hasID := node["$id"]
		if hasID {
			text, ok := id.(string)
			if !ok {
				return fmt.Errorf("resolve: invalid $id at %s", pointer)
			}
			identifier, err := uri(base, text)
			if err != nil || identifier.Fragment != "" {
				return fmt.Errorf("resolve: invalid resource identity at %s", pointer)
			}
			base = identifier.String()
		}
		if root || hasID {
			if _, exists := r.resources[base]; exists {
				return fmt.Errorf("resolve: duplicate resource identity at %s", pointer)
			}
			if _, exists := resources[base]; exists {
				return fmt.Errorf("resolve: duplicate resource identity at %s", pointer)
			}
			resources[base] = resource{base, value}
			prefixes[base] = pointer
			if root {
				rootResource = resources[base]
			}
		}
		if name, present := node["$anchor"]; present {
			text, ok := name.(string)
			if !ok || !validAnchor(text) {
				return fmt.Errorf("resolve: invalid anchor at %s", pointer)
			}
			key := base + "#" + text
			if anchors[key] != nil || r.anchors[key] != nil {
				return fmt.Errorf("resolve: duplicate anchor at %s", pointer)
			}
			anchors[key] = &Ref{Canonical: base, Pointer: strings.TrimPrefix(pointer, prefixes[base]), Value: value, base: base}
		}
		for _, keyword := range []string{"$dynamicRef", "$dynamicAnchor"} {
			if _, present := node[keyword]; present {
				return fmt.Errorf("resolve: unqualified %s at %s", keyword, pointer)
			}
		}
		if reference, present := node["$ref"]; present {
			text, ok := reference.(string)
			if !ok {
				return fmt.Errorf("resolve: invalid $ref at %s", pointer)
			}
			references = append(references, edge{base, text, pointer + "/$ref", base + "#" + strings.TrimPrefix(pointer, prefixes[base])})
			if len(references)+len(r.references) > r.Limits.MaxReferenceEdges {
				return fmt.Errorf("resolve: reference edge budget exhausted")
			}
		}
		for _, keyword := range []string{"$defs", "properties", "patternProperties", "dependentSchemas"} {
			children, ok := node[keyword].(map[string]any)
			if !ok {
				continue
			}
			names := make([]string, 0, len(children))
			for name := range children {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				if err := walk(children[name], base, pointer+"/"+keyword+"/"+escape(name), depth+1, false); err != nil {
					return err
				}
			}
		}
		for _, keyword := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
			children, _ := node[keyword].([]any)
			for i, child := range children {
				if err := walk(child, base, pointer+"/"+keyword+"/"+strconv.Itoa(i), depth+1, false); err != nil {
					return err
				}
			}
		}
		for _, keyword := range []string{"items", "additionalProperties", "additionalItems", "contains", "unevaluatedProperties", "unevaluatedItems", "propertyNames", "not", "if", "then", "else"} {
			if child, present := node[keyword]; present {
				if err := walk(child, base, pointer+"/"+keyword, depth+1, false); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(doc.Root, canonical, "", 0, true); err != nil {
		return err
	}
	if rootResource.root == nil {
		return fmt.Errorf("resolve: document root must be a schema object")
	}
	if _, exists := r.resources[canonical]; exists {
		return fmt.Errorf("resolve: document identity conflicts with a registered resource")
	}
	if existing, exists := resources[canonical]; exists && existing.canonical != rootResource.canonical {
		return fmt.Errorf("resolve: document identity conflicts with a nested resource")
	}
	for key, value := range resources {
		r.resources[key] = value
	}
	r.resources[canonical] = rootResource
	for key, value := range anchors {
		r.anchors[key] = value
	}
	r.references = append(r.references, references...)
	r.Documents[canonical] = doc
	return nil
}

func validAnchor(text string) bool {
	for i, c := range text {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || i > 0 && (c >= '0' && c <= '9' || c == '-' || c == '.') {
			continue
		}
		return false
	}
	return text != ""
}

func escape(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "~", "~0"), "/", "~1")
}

// Validate checks all graph edges without expanding recursive schemas.
func (r *Registry) Validate() error {
	graph := make(map[string]string, len(r.references))
	for _, reference := range r.references {
		resolved, err := r.resolve(reference.base, reference.reference)
		if err != nil {
			return fmt.Errorf("resolve: %s: %w", reference.pointer, err)
		}
		switch resolved.Value.(type) {
		case map[string]any, bool:
		default:
			return fmt.Errorf("resolve: reference target is not a schema at %s", reference.pointer)
		}
		graph[reference.origin] = resolved.Canonical + "#" + resolved.Pointer
	}
	// Each schema location has at most one $ref. Compute heights of the
	// functional graph in linear time, treating each cycle as one component.
	// This bounds acyclic depth without repeatedly expanding shared targets.
	heights := make(map[string]int)
	names := make([]string, 0, len(graph))
	for name := range graph {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, start := range names {
		if heights[start] != 0 {
			continue
		}
		positions := make(map[string]int)
		var path []string
		current := start
		for heights[current] == 0 {
			if at, repeated := positions[current]; repeated {
				for _, member := range path[at:] {
					heights[member] = 1
				}
				path = path[:at]
				break
			}
			positions[current] = len(path)
			path = append(path, current)
			next, exists := graph[current]
			if !exists {
				heights[current] = 1
				path = path[:len(path)-1]
				break
			}
			current = next
		}
		for i := len(path) - 1; i >= 0; i-- {
			height := heights[graph[path[i]]] + 1
			if height > r.Limits.MaxDepth {
				return fmt.Errorf("resolve: acyclic reference depth exceeds %d", r.Limits.MaxDepth)
			}
			heights[path[i]] = height
		}
	}
	return nil
}

// Resolve resolves a pointer or anchor in a registered resource, without IO.
func (r *Registry) Resolve(from, reference string) (*Ref, error) {
	if r.Edges >= r.Limits.MaxReferenceEdges {
		return nil, fmt.Errorf("resolve: reference edge budget exhausted")
	}
	r.Edges++
	return r.resolve(from, reference)
}

func (r *Registry) resolve(from, reference string) (*Ref, error) {
	u, err := uri(from, reference)
	if err != nil {
		return nil, err
	}
	fragment := u.Fragment
	u.Fragment, u.RawFragment = "", ""
	canonical := u.String()
	res, found := r.resources[canonical]
	if !found {
		return nil, fmt.Errorf("resolve: unregistered resource")
	}
	if fragment != "" && !strings.HasPrefix(fragment, "/") {
		anchor := r.anchors[res.canonical+"#"+fragment]
		if anchor == nil {
			return nil, fmt.Errorf("resolve: missing anchor")
		}
		copy := *anchor
		return &copy, nil
	}
	value, base := res.root, res.canonical
	canonical = res.canonical
	var relative []string
	if fragment != "" {
		for _, token := range strings.Split(fragment[1:], "/") {
			relative = append(relative, token)
			decoded, err := decodeToken(token)
			if err != nil {
				return nil, err
			}
			switch node := value.(type) {
			case map[string]any:
				child, exists := node[decoded]
				if !exists {
					return nil, fmt.Errorf("resolve: missing pointer target")
				}
				value = child
			case []any:
				index, ok := arrayIndex(decoded)
				if !ok || index >= len(node) {
					return nil, fmt.Errorf("resolve: invalid or out-of-range array index")
				}
				value = node[index]
			default:
				return nil, fmt.Errorf("resolve: pointer cannot descend into scalar")
			}
			if child, ok := value.(map[string]any); ok {
				if id, ok := child["$id"].(string); ok {
					identifier, err := uri(base, id)
					if err != nil {
						return nil, err
					}
					base = identifier.String()
					canonical = base
					relative = nil
				}
			}
		}
	}
	pointer := ""
	if len(relative) != 0 {
		pointer = "/" + strings.Join(relative, "/")
	}
	return &Ref{Canonical: canonical, Pointer: pointer, Value: value, base: base}, nil
}

func decodeToken(token string) (string, error) {
	for i := 0; i < len(token); i++ {
		if token[i] == '~' {
			if i+1 == len(token) || token[i+1] != '0' && token[i+1] != '1' {
				return "", fmt.Errorf("resolve: invalid JSON Pointer escape")
			}
			i++
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~"), nil
}

func arrayIndex(token string) (int, bool) {
	if token == "" || len(token) > 1 && token[0] == '0' {
		return 0, false
	}
	for _, c := range token {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	index, err := strconv.Atoi(token)
	return index, err == nil
}

// RefuseNetwork guards acquisition; registered HTTP identifiers are local.
func RefuseNetwork(reference string) error {
	u, err := uri("", reference)
	if err != nil {
		return err
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		return fmt.Errorf("resolve: network acquisition refused")
	}
	return nil
}

// ResolveChain follows references, retaining cycles as existing node identities.
func (r *Registry) ResolveChain(from, reference string, maxDepth int) ([]*Ref, error) {
	if maxDepth <= 0 || maxDepth > r.Limits.MaxDepth {
		maxDepth = r.Limits.MaxDepth
	}
	seen := make(map[string]int)
	var chain []*Ref
	for {
		resolved, err := r.resolve(from, reference)
		if err != nil {
			return nil, err
		}
		key := resolved.Canonical + "#" + resolved.Pointer
		if at, repeated := seen[key]; repeated {
			if at+1 > maxDepth {
				return nil, fmt.Errorf("resolve: acyclic reference depth exceeds %d", maxDepth)
			}
			return chain, nil
		}
		if r.Edges >= r.Limits.MaxReferenceEdges {
			return nil, fmt.Errorf("resolve: reference edge budget exhausted")
		}
		r.Edges++
		seen[key] = len(chain)
		chain = append(chain, resolved)
		node, ok := resolved.Value.(map[string]any)
		if !ok {
			break
		}
		next, ok := node["$ref"].(string)
		if !ok {
			break
		}
		from, reference = resolved.base, next
	}
	if len(chain) > maxDepth {
		return nil, fmt.Errorf("resolve: reference chain exceeds %d hops", maxDepth)
	}
	return chain, nil
}
