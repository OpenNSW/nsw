package jsonform

import (
	"fmt"
)

type VisitFunc func(path string, node *JSONSchema, parent *JSONSchema) error

func Traverse(schema *JSONSchema, visit VisitFunc) error {
	return traverseRecursive(schema, "", nil, visit)
}

func traverseRecursive(
	node *JSONSchema,
	path string,
	parent *JSONSchema,
	visit VisitFunc,
) error {

	if node == nil {
		return nil
	}

	// Execute callback on current node
	if err := visit(path, node, parent); err != nil {
		return err
	}

	// Handle object properties
	if node.Type == "object" {
		for key, child := range node.Properties {
			childPath := key
			if path != "" {
				childPath = fmt.Sprintf("%s.%s", path, key)
			}

			if err := traverseRecursive(&child, childPath, node, visit); err != nil {
				return err
			}
		}
	}

	// Handle array items
	if node.Type == "array" && node.Items != nil {
		childPath := path + "[]"
		if err := traverseRecursive(node.Items, childPath, node, visit); err != nil {
			return err
		}
	}

	return nil
}
