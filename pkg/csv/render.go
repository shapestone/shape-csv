// Package csv provides AST rendering to CSV bytes.
//
// This file implements the core CSV rendering functionality, converting
// Shape AST nodes back into CSV byte representations.
package csv

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/shapestone/shape-core/pkg/ast"
)

// Render converts an AST node to CSV bytes.
//
// The node should be the result of Parse() or ParseReader().
// Returns CSV bytes following RFC 4180 format.
//
// Rendering handles:
//   - Automatic quoting of fields containing commas, quotes, or newlines
//   - Proper escaping of quotes (doubled)
//   - Preservation of empty fields
//   - Consistent line endings (LF)
//
// Example:
//
//	node, _ := csv.Parse("name,age\nAlice,30\nBob,25\n")
//	bytes, _ := csv.Render(node)
//	// bytes: name,age\nAlice,30\nBob,25\n
func Render(node ast.SchemaNode) ([]byte, error) {
	if node == nil {
		return []byte{}, nil
	}

	var buf bytes.Buffer

	if err := renderNode(node, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// renderNode recursively renders an AST node to the buffer.
func renderNode(node ast.SchemaNode, buf *bytes.Buffer) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.ArrayDataNode:
		return renderArrayData(n, buf)
	case *ast.LiteralNode:
		return renderLiteral(n, buf)
	default:
		return fmt.Errorf("unsupported node type for CSV rendering: %T", node)
	}
}

// renderArrayData renders an ArrayDataNode as CSV.
// This handles both the file level (array of records) and record level (array of fields).
func renderArrayData(node *ast.ArrayDataNode, buf *bytes.Buffer) error {
	elements := node.Elements()
	if len(elements) == 0 {
		return nil
	}

	// Check if this is a file (array of arrays) or a record (array of literals)
	switch elements[0].(type) {
	case *ast.ArrayDataNode:
		// File level - array of records
		for i, elem := range elements {
			if i > 0 {
				buf.WriteByte('\n')
			}
			if err := renderNode(elem, buf); err != nil {
				return err
			}
		}
		// Add final newline
		buf.WriteByte('\n')
		return nil

	case *ast.LiteralNode:
		// Record level - array of fields
		for i, elem := range elements {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := renderNode(elem, buf); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unexpected element type in array: %T", elements[0])
	}
}

// renderLiteral renders a LiteralNode as a CSV field.
func renderLiteral(node *ast.LiteralNode, buf *bytes.Buffer) error {
	value := node.Value()

	// CSV fields are strings
	var fieldValue string
	if s, ok := value.(string); ok {
		fieldValue = s
	} else if value == nil {
		fieldValue = ""
	} else {
		fieldValue = fmt.Sprintf("%v", value)
	}

	// Write field with proper escaping
	writeCSVField(buf, fieldValue)
	return nil
}

// writeCSVField writes a CSV field to the buffer with proper escaping.
// Fields containing commas, quotes, newlines, or carriage returns are quoted.
// Quotes within quoted fields are escaped by doubling them.
func writeCSVField(buf *bytes.Buffer, value string) {
	writeCSVFieldWithDelim(buf, value, ',')
}

// writeCSVFieldWithDelim writes a CSV field with a custom delimiter.
func writeCSVFieldWithDelim(buf *bytes.Buffer, value string, delim rune) {
	// Check if field needs quoting (contains delimiter, quotes, newlines, or carriage returns)
	needsQuoting := strings.ContainsRune(value, delim) || strings.ContainsAny(value, "\"\n\r")

	if needsQuoting {
		buf.WriteByte('"')
		// Escape quotes by doubling them
		for _, ch := range value {
			if ch == '"' {
				buf.WriteString(`""`)
			} else {
				buf.WriteRune(ch)
			}
		}
		buf.WriteByte('"')
	} else {
		buf.WriteString(value)
	}
}

// renderWithOptions converts an AST node to CSV bytes with custom options.
func renderWithOptions(node ast.SchemaNode, opts WriterOptions) ([]byte, error) {
	if node == nil {
		return []byte{}, nil
	}

	var buf bytes.Buffer
	lineEnding := "\n"
	if opts.UseCRLF {
		lineEnding = "\r\n"
	}

	if err := renderNodeWithOptions(node, &buf, opts.Comma, lineEnding); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// renderNodeWithOptions recursively renders an AST node with custom delimiter and line ending.
func renderNodeWithOptions(node ast.SchemaNode, buf *bytes.Buffer, delim rune, lineEnding string) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.ArrayDataNode:
		return renderArrayDataWithOptions(n, buf, delim, lineEnding)
	case *ast.LiteralNode:
		return renderLiteralWithDelim(n, buf, delim)
	default:
		return fmt.Errorf("unsupported node type for CSV rendering: %T", node)
	}
}

// renderArrayDataWithOptions renders an ArrayDataNode with custom delimiter and line ending.
func renderArrayDataWithOptions(node *ast.ArrayDataNode, buf *bytes.Buffer, delim rune, lineEnding string) error {
	elements := node.Elements()
	if len(elements) == 0 {
		return nil
	}

	// Check if this is a file (array of arrays) or a record (array of literals)
	switch elements[0].(type) {
	case *ast.ArrayDataNode:
		// File level - array of records
		for i, elem := range elements {
			if i > 0 {
				buf.WriteString(lineEnding)
			}
			if err := renderNodeWithOptions(elem, buf, delim, lineEnding); err != nil {
				return err
			}
		}
		// Add final line ending
		buf.WriteString(lineEnding)
		return nil

	case *ast.LiteralNode:
		// Record level - array of fields
		for i, elem := range elements {
			if i > 0 {
				buf.WriteRune(delim)
			}
			if err := renderNodeWithOptions(elem, buf, delim, lineEnding); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unexpected element type in array: %T", elements[0])
	}
}

// renderLiteralWithDelim renders a LiteralNode with a custom delimiter.
func renderLiteralWithDelim(node *ast.LiteralNode, buf *bytes.Buffer, delim rune) error {
	value := node.Value()

	// CSV fields are strings
	var fieldValue string
	if s, ok := value.(string); ok {
		fieldValue = s
	} else if value == nil {
		fieldValue = ""
	} else {
		fieldValue = fmt.Sprintf("%v", value)
	}

	// Write field with proper escaping
	writeCSVFieldWithDelim(buf, fieldValue, delim)
	return nil
}
