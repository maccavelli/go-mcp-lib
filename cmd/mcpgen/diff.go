package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/load"
	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/ownership"
)

type semanticChange struct {
	Scope   string `json:"scope"`
	Pointer string `json:"pointer"`
	Kind    string `json:"kind"`
	Impact  string `json:"impact"`
	Before  any    `json:"before"`
	After   any    `json:"after"`
}

type diffChanges struct {
	From     string           `json:"from"`
	To       string           `json:"to"`
	Added    []string         `json:"added"`
	Removed  []string         `json:"removed"`
	Changed  []string         `json:"changed"`
	Semantic []semanticChange `json:"semantic"`
}

func (c *diffChanges) data() map[string]any {
	return map[string]any{"from": c.From, "to": c.To, "added": c.Added, "removed": c.Removed, "changed": c.Changed, "semantic": c.Semantic,
		"compatibility": "review required for removals, signature changes and changed assertions; no automatic compatibility approval"}
}

type contract struct {
	revision  string
	hashes    map[string]string
	documents map[string]any
}

func readContract(file string) (_ *contract, failure error) {
	root, err := os.OpenRoot(filepath.Dir(file))
	if err != nil {
		return nil, err
	}
	defer func() { failure = errors.Join(failure, root.Close()) }()
	body, err := ownership.Read(root, filepath.Base(file))
	if err != nil {
		return nil, err
	}
	hashes, legacy, err := ownership.Parse(body)
	if err != nil {
		return nil, err
	}
	if legacy {
		return nil, fmt.Errorf("diff: hashless legacy output has no complete semantic contract")
	}
	var manifest ownership.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	result := &contract{revision: manifest.Revision, hashes: hashes, documents: map[string]any{}}
	for _, name := range manifest.Files {
		body, err := ownership.Read(root, name)
		if err != nil || ownership.Digest(body) != hashes[name] {
			return nil, fmt.Errorf("diff: sidecar/output identity differs: %s", name)
		}
		if strings.HasSuffix(name, ".json") {
			if _, err := load.New().LoadBytes(name, body); err != nil {
				return nil, err
			}
			var value any
			decoder := json.NewDecoder(bytes.NewReader(body))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				return nil, err
			}
			result.documents[name] = value
		}
	}
	for _, name := range []string{"schema.json", "api.json", "coverage.json", "bindings.json"} {
		if result.documents[name] == nil {
			return nil, fmt.Errorf("diff: missing complete contract: %s", name)
		}
	}
	return result, nil
}

func diffManifests(fromPath, toPath string) (*diffChanges, error) {
	from, err := readContract(fromPath)
	if err != nil {
		return nil, err
	}
	to, err := readContract(toPath)
	if err != nil {
		return nil, err
	}
	changes := &diffChanges{From: from.revision, To: to.revision, Added: []string{}, Removed: []string{}, Changed: []string{}, Semantic: []semanticChange{}}
	for _, name := range ownership.Sorted(to.hashes) {
		if from.hashes[name] == "" {
			changes.Added = append(changes.Added, name)
		} else if from.hashes[name] != to.hashes[name] {
			changes.Changed = append(changes.Changed, name)
		}
	}
	for _, name := range ownership.Sorted(from.hashes) {
		if to.hashes[name] == "" {
			changes.Removed = append(changes.Removed, name)
		}
	}
	for _, name := range []string{"schema.json", "api.json", "coverage.json", "bindings.json"} {
		if err := changes.compare(name, "#", from.documents[name], true, to.documents[name], true); err != nil {
			return nil, err
		}
	}
	return changes, nil
}

func (c *diffChanges) compare(scope, pointer string, before any, had bool, after any, has bool) error {
	if reflect.DeepEqual(before, after) && had == has {
		return nil
	}
	left, lok := before.(map[string]any)
	right, rok := after.(map[string]any)
	if lok && rok && had && has {
		keys := map[string]bool{}
		for key := range left {
			keys[key] = true
		}
		for key := range right {
			keys[key] = true
		}
		for _, key := range ownership.Sorted(keys) {
			a, had := left[key]
			b, has := right[key]
			escaped := strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
			if err := c.compare(scope, pointer+"/"+escaped, a, had, b, has); err != nil {
				return err
			}
		}
		return nil
	}
	// Arrays are compared as a whole: order, removed alternatives, method and
	// capability inventories, and exact numeric constraints remain observable.
	if len(c.Semantic) >= 4096 {
		return fmt.Errorf("diff: semantic change budget exceeded")
	}
	kind, impact := "changed", "potentially-breaking; review required"
	if !had {
		kind, impact = "added", "review required"
	} else if !has {
		kind = "removed"
	}
	c.Semantic = append(c.Semantic, semanticChange{scope, pointer, kind, impact, before, after})
	sort.SliceStable(c.Semantic, func(i, j int) bool {
		a, b := c.Semantic[i], c.Semantic[j]
		return a.Scope < b.Scope || a.Scope == b.Scope && a.Pointer < b.Pointer
	})
	return nil
}
