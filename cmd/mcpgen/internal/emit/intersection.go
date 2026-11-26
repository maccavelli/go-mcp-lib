package emit

import (
	"fmt"
	"strings"

	"github.com/maccavelli/go-mcp-lib/cmd/mcpgen/internal/model"
)

// objectIntersection projects a structural object intersection into fields.
// This is not a schema normalization: generated hooks retain the original
// pointer and validate all branches in the unchanged embedded document.
func (c *compiler) objectIntersection(node *model.Node) (*model.Node, error) {
	leaves := []*model.Node{}
	if err := c.objectLeaves(node, map[*model.Node]bool{}, map[*model.Node]bool{}, 0, &leaves); err != nil {
		return nil, err
	}
	objectConstrained := false
	for _, leaf := range leaves {
		objectConstrained = objectConstrained || leaf.Type == "object"
	}
	if !objectConstrained {
		return nil, fmt.Errorf("emit: %s: intersection does not constrain an object", node.Pointer)
	}
	projected := &model.Node{
		Pointer: node.Pointer, Source: node.Source, Type: "object",
		Properties: make(map[string]*model.Node),
	}
	required := map[string]bool{}
	for _, leaf := range leaves {
		for _, name := range sortedKeys(leaf.Properties) {
			child := leaf.Properties[name]
			if previous := projected.Properties[name]; previous != nil {
				var err error
				child, err = c.fieldIntersection(previous, child)
				if err != nil {
					return nil, err
				}
			}
			projected.Properties[name] = child
		}
		for _, name := range leaf.Required {
			required[name] = true
		}
		if leaf.Open != nil && !*leaf.Open {
			closed := false
			projected.Open = &closed
		}
		if leaf.Additional != nil {
			additional := leaf.Additional
			if projected.Additional != nil {
				var err error
				additional, err = c.fieldIntersection(projected.Additional, additional)
				if err != nil {
					return nil, err
				}
			}
			projected.Additional = additional
		}
	}
	projected.Required = sortedKeys(required)
	return projected, nil
}

func (c *compiler) objectLeaves(node *model.Node, active, visited map[*model.Node]bool, depth int, leaves *[]*model.Node) error {
	if depth >= 256 || active[node] {
		return fmt.Errorf("emit: %s: recursive or over-depth object intersection cannot be projected", node.Pointer)
	}
	// Intersecting the same schema twice is idempotent. Expand shared DAG nodes
	// once; active-path detection still rejects recursive structural projection.
	if visited[node] {
		return nil
	}
	visited[node] = true
	active[node] = true
	defer delete(active, node)
	if node.Type != "" && node.Type != "object" || isUnion(node) {
		return fmt.Errorf("emit: %s: intersection branch has no qualified object representation", node.Pointer)
	}
	if _, referenced := node.Annotations["$ref"]; referenced {
		target, err := c.referenceNode(node)
		if err != nil {
			return err
		}
		if err := c.objectLeaves(target, active, visited, depth+1, leaves); err != nil {
			return err
		}
	}
	*leaves = append(*leaves, node)
	for _, branch := range node.Compositions["allOf"] {
		if err := c.objectLeaves(branch, active, visited, depth+1, leaves); err != nil {
			return err
		}
	}
	return nil
}

// fieldIntersection selects a lossless field representation. Constraints remain
// on the complete object schema. Complex incompatible shapes fail explicitly;
// they never become any or an arbitrarily selected branch.
func (c *compiler) fieldIntersection(left, right *model.Node) (*model.Node, error) {
	if leftRef, ok := left.Annotations["$ref"].(string); ok {
		if rightRef, ok := right.Annotations["$ref"].(string); ok && leftRef == rightRef {
			return left, nil
		}
	}
	if untypedField(left) {
		return right, nil
	}
	if untypedField(right) {
		return left, nil
	}
	if !isUnion(left) && !isUnion(right) && len(left.Compositions["allOf"]) == 0 && len(right.Compositions["allOf"]) == 0 {
		if left.Type == right.Type && (left.Type == "string" || left.Type == "boolean" || left.Type == "null" || left.Type == "integer" || left.Type == "number") {
			return left, nil
		}
		if left.Type == "number" && right.Type == "integer" {
			return right, nil
		}
		if left.Type == "integer" && right.Type == "number" {
			return left, nil
		}
	}
	return nil, fmt.Errorf("emit: %s: intersection with %s has no qualified field representation", left.Pointer, right.Pointer)
}

func untypedField(node *model.Node) bool {
	_, referenced := node.Annotations["$ref"]
	return node.Type == "" && !referenced && len(node.Properties) == 0 && node.Items == nil && node.Additional == nil && len(node.Compositions) == 0
}

func (c *compiler) referenceNode(node *model.Node) (*model.Node, error) {
	ref, ok := node.Annotations["$ref"].(string)
	const prefix = "#/$defs/"
	if !ok || !strings.HasPrefix(ref, prefix) {
		return nil, fmt.Errorf("emit: %s: reference has no reviewed declaration binding", node.Pointer)
	}
	name := strings.TrimPrefix(ref, prefix)
	name = strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~")
	target := c.opts.Model.Definitions[name]
	if target == nil {
		return nil, fmt.Errorf("emit: %s: missing reference binding %q", node.Pointer, ref)
	}
	return target, nil
}
