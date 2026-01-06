// Package csv provides conversion between AST nodes and Go native types.
package csv

import (
	"fmt"

	"github.com/shapestone/shape-core/pkg/ast"
)

// NodeToInterface converts an AST node to native Go types.
//
// For CSV, this converts:
//   - *ast.ArrayDataNode (file) → [][]string (slice of records)
//   - *ast.ArrayDataNode (record) → []string (slice of fields)
//   - *ast.LiteralNode (field) → string (field value)
//
// This function recursively processes nested structures.
//
// Example:
//
//	node, _ := csv.Parse("name,age\nAlice,30\n")
//	data := csv.NodeToInterface(node)
//	// data is [][]string{{"name","age"}, {"Alice","30"}}
func NodeToInterface(node ast.SchemaNode) interface{} {
	switch n := node.(type) {
	case *ast.LiteralNode:
		val := n.Value()
		// CSV fields are always strings
		if s, ok := val.(string); ok {
			return s
		}
		// Fallback: convert to string representation
		return fmt.Sprintf("%v", val)

	case *ast.ArrayDataNode:
		// Check if this is a file (array of arrays) or a record (array of literals)
		elements := n.Elements()
		if len(elements) == 0 {
			// Empty array could be either - return empty [][]string for CSV file
			return [][]string{}
		}

		// Check first element to determine type
		switch elements[0].(type) {
		case *ast.ArrayDataNode:
			// This is a file (array of records)
			records := make([][]string, len(elements))
			for i, elem := range elements {
				recordData := NodeToInterface(elem)
				if record, ok := recordData.([]string); ok {
					records[i] = record
				} else {
					// Unexpected type
					records[i] = []string{}
				}
			}
			return records

		case *ast.LiteralNode:
			// This is a record (array of fields)
			fields := make([]string, len(elements))
			for i, elem := range elements {
				fieldData := NodeToInterface(elem)
				if field, ok := fieldData.(string); ok {
					fields[i] = field
				} else {
					fields[i] = fmt.Sprintf("%v", fieldData)
				}
			}
			return fields

		default:
			// Unknown node type
			return []string{}
		}

	default:
		return nil
	}
}

// InterfaceToNode converts native Go types to AST nodes.
//
// For CSV, this converts:
//   - [][]string (slice of records) → *ast.ArrayDataNode (file)
//   - []string (slice of fields) → *ast.ArrayDataNode (record)
//   - string (field value) → *ast.LiteralNode
//
// This function recursively processes nested structures.
//
// Example:
//
//	data := [][]string{
//	    {"name", "age"},
//	    {"Alice", "30"},
//	}
//	node, _ := csv.InterfaceToNode(data)
//	// node is an *ast.ArrayDataNode representing the CSV data
func InterfaceToNode(v interface{}) (ast.SchemaNode, error) {
	// Use empty position since we're creating nodes programmatically
	pos := ast.Position{}

	if v == nil {
		return ast.NewLiteralNode("", pos), nil
	}

	switch val := v.(type) {
	// Handle strings (CSV fields)
	case string:
		return ast.NewLiteralNode(val, pos), nil

	// Handle [][]string (CSV file - slice of records)
	case [][]string:
		records := make([]ast.SchemaNode, len(val))
		for i, record := range val {
			recordNode, err := InterfaceToNode(record)
			if err != nil {
				return nil, fmt.Errorf("record %d: %w", i, err)
			}
			records[i] = recordNode
		}
		return ast.NewArrayDataNode(records, pos), nil

	// Handle []string (CSV record - slice of fields)
	case []string:
		fields := make([]ast.SchemaNode, len(val))
		for i, field := range val {
			fields[i] = ast.NewLiteralNode(field, pos)
		}
		return ast.NewArrayDataNode(fields, pos), nil

	// Handle []interface{} (generic slice)
	case []interface{}:
		// Try to convert to [][]string or []string
		if len(val) == 0 {
			return ast.NewArrayDataNode([]ast.SchemaNode{}, pos), nil
		}

		// Check first element type
		switch val[0].(type) {
		case []interface{}, []string:
			// Looks like a file (slice of records)
			records := make([]ast.SchemaNode, len(val))
			for i, item := range val {
				recordNode, err := InterfaceToNode(item)
				if err != nil {
					return nil, fmt.Errorf("record %d: %w", i, err)
				}
				records[i] = recordNode
			}
			return ast.NewArrayDataNode(records, pos), nil

		case string:
			// Looks like a record (slice of fields)
			fields := make([]ast.SchemaNode, len(val))
			for i, item := range val {
				if s, ok := item.(string); ok {
					fields[i] = ast.NewLiteralNode(s, pos)
				} else {
					fields[i] = ast.NewLiteralNode(fmt.Sprintf("%v", item), pos)
				}
			}
			return ast.NewArrayDataNode(fields, pos), nil

		default:
			return nil, fmt.Errorf("unsupported slice element type: %T", val[0])
		}

	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
}

// NodeToRecords converts an AST node to a slice of string records.
// This is a convenience wrapper around NodeToInterface that returns
// the data in [][]string format suitable for CSV processing.
//
// Example:
//
//	node, _ := csv.Parse("name,age\nAlice,30\n")
//	records := csv.NodeToRecords(node)
//	// records is [][]string{{"name","age"}, {"Alice","30"}}
func NodeToRecords(node ast.SchemaNode) [][]string {
	data := NodeToInterface(node)
	if records, ok := data.([][]string); ok {
		return records
	}
	// If it's a single record, wrap it
	if record, ok := data.([]string); ok {
		return [][]string{record}
	}
	return [][]string{}
}

// RecordsToNode converts a slice of string records to an AST node.
// This is a convenience wrapper around InterfaceToNode for [][]string.
//
// Example:
//
//	records := [][]string{
//	    {"name", "age"},
//	    {"Alice", "30"},
//	}
//	node, _ := csv.RecordsToNode(records)
func RecordsToNode(records [][]string) (ast.SchemaNode, error) {
	return InterfaceToNode(records)
}
