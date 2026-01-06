// Package csv provides a user-friendly DOM API for CSV manipulation.
//
// The DOM API provides type-safe, fluent interfaces for building and manipulating
// CSV documents without requiring type assertions or working with raw AST nodes.
//
// # Document Type
//
// Document represents a CSV file with optional headers and data records:
//
//	doc := csv.NewDocument().
//		SetHeaders([]string{"name", "age"}).
//		AddRecord([]string{"Alice", "30"}).
//		AddRecord([]string{"Bob", "25"})
//
// # Record Type
//
// Record represents a single row in a CSV file with typed access:
//
//	record, _ := doc.GetRecord(0)
//	name, _ := record.Get(0)           // Get by index
//	age, _ := record.GetByName("age")  // Get by header name
//
// # Type-Safe Access
//
// Access values without type assertions:
//
//	record, ok := doc.GetRecord(0)     // Get first record
//	field, ok := record.Get(0)         // Get first field
//	field, ok := record.GetByName("name")  // Get field by header name
//
// # Round-trip Support
//
// Parse CSV and render back to CSV:
//
//	doc, _ := csv.ParseDocument("name,age\nAlice,30")
//	csvStr, _ := doc.CSV()  // Render back to CSV string
package csv

import (
	"fmt"
	"strings"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-csv/internal/fastparser"
)

// Document represents a CSV file with a fluent API for manipulation.
// All setter methods return *Document to enable method chaining.
//
// A Document consists of:
//   - Optional headers (first row that names the columns)
//   - Data records (remaining rows)
type Document struct {
	headers []string
	records [][]string
}

// Record represents a single row in a CSV file.
// It provides type-safe access to field values by index or by header name.
type Record struct {
	fields  []string
	headers []string // Reference to document headers for name-based access
}

// NewDocument creates a new empty Document.
func NewDocument() *Document {
	return &Document{
		headers: []string{},
		records: make([][]string, 0),
	}
}

// ParseDocument parses CSV string into a Document with a fluent API.
// Returns an error if the input is not valid CSV.
//
// The first row can be treated as headers using SetHeaders() if needed.
// By default, all rows are treated as data records.
//
// Example:
//
//	doc, err := csv.ParseDocument("name,age\nAlice,30\nBob,25")
//	if err != nil {
//	    // handle error
//	}
//	// Optionally designate first row as headers
//	if doc.RecordCount() > 0 {
//	    headers := doc.Records()[0].Fields()
//	    doc.SetHeaders(headers)
//	    // Remove first record since it's now headers
//	}
func ParseDocument(input string) (*Document, error) {
	// Parse CSV using fast parser
	records, err := fastparser.Parse([]byte(input))
	if err != nil {
		return nil, err
	}

	doc := NewDocument()
	for _, record := range records {
		doc.AddRecord(record)
	}

	return doc, nil
}

// SetHeaders sets the column headers for this CSV document.
// Headers are used by Record.GetByName() to access fields by name.
// Returns the Document for method chaining.
func (d *Document) SetHeaders(headers []string) *Document {
	d.headers = headers
	return d
}

// AddRecord adds a data record (row) to the document.
// Returns the Document for method chaining.
func (d *Document) AddRecord(fields []string) *Document {
	d.records = append(d.records, fields)
	return d
}

// Headers returns the column headers.
// Returns an empty slice if no headers have been set.
func (d *Document) Headers() []string {
	return d.headers
}

// Records returns all data records as Record objects.
// Each Record provides type-safe access to field values.
func (d *Document) Records() []Record {
	records := make([]Record, len(d.records))
	for i, fields := range d.records {
		records[i] = Record{
			fields:  fields,
			headers: d.headers,
		}
	}
	return records
}

// RecordCount returns the number of data records in the document.
// This does not include the header row.
func (d *Document) RecordCount() int {
	return len(d.records)
}

// GetRecord returns the record at the specified index.
// Returns (Record, false) if the index is out of bounds.
// Index is 0-based (0 = first data record, not the header).
func (d *Document) GetRecord(index int) (Record, bool) {
	if index < 0 || index >= len(d.records) {
		return Record{}, false
	}

	return Record{
		fields:  d.records[index],
		headers: d.headers,
	}, true
}

// CSV renders the Document back to a CSV string.
// This includes headers (if set) followed by all data records.
//
// Example:
//
//	doc := csv.NewDocument().
//	    SetHeaders([]string{"name", "age"}).
//	    AddRecord([]string{"Alice", "30"})
//	csvStr, _ := doc.CSV()
//	// Output: name,age\nAlice,30\n
func (d *Document) CSV() (string, error) {
	var sb strings.Builder

	// Write headers if set
	if len(d.headers) > 0 {
		if err := writeRecord(&sb, d.headers); err != nil {
			return "", err
		}
	}

	// Write records
	for _, record := range d.records {
		if err := writeRecord(&sb, record); err != nil {
			return "", err
		}
	}

	return sb.String(), nil
}

// writeRecord writes a single record to the string builder in CSV format.
// Handles quoting of fields that contain commas, quotes, or newlines.
func writeRecord(sb *strings.Builder, fields []string) error {
	for i, field := range fields {
		if i > 0 {
			sb.WriteByte(',')
		}

		// Check if field needs quoting
		needsQuoting := strings.ContainsAny(field, ",\"\n\r")

		if needsQuoting {
			sb.WriteByte('"')
			// Escape quotes by doubling them
			for _, ch := range field {
				if ch == '"' {
					sb.WriteString(`""`)
				} else {
					sb.WriteRune(ch)
				}
			}
			sb.WriteByte('"')
		} else {
			sb.WriteString(field)
		}
	}

	sb.WriteByte('\n')
	return nil
}

// ============================================================================
// Record Methods (type-safe field access)
// ============================================================================

// Get gets the field value at the specified index.
// Returns (value, false) if the index is out of bounds.
// Index is 0-based.
func (r Record) Get(index int) (string, bool) {
	if index < 0 || index >= len(r.fields) {
		return "", false
	}
	return r.fields[index], true
}

// GetByName gets the field value by header name.
// Returns (value, false) if the header name is not found or if no headers are set.
//
// Example:
//
//	record, _ := doc.GetRecord(0)
//	name, ok := record.GetByName("name")
//	if !ok {
//	    // Header "name" not found or no headers set
//	}
func (r Record) GetByName(name string) (string, bool) {
	if len(r.headers) == 0 {
		return "", false
	}

	// Find the index of the header
	for i, header := range r.headers {
		if header == name {
			return r.Get(i)
		}
	}

	return "", false
}

// Fields returns all field values in the record.
// This returns a copy of the fields slice.
func (r Record) Fields() []string {
	// Return a copy to prevent external modification
	fields := make([]string, len(r.fields))
	copy(fields, r.fields)
	return fields
}

// Len returns the number of fields in the record.
func (r Record) Len() int {
	return len(r.fields)
}

// ============================================================================
// AST Conversion (for integration with AST-based APIs)
// ============================================================================

// ToAST converts the Document to an AST ArrayDataNode.
// This is useful for integration with other Shape parsers.
func (d *Document) ToAST() (*ast.ArrayDataNode, error) {
	allRecords := make([]ast.SchemaNode, 0, len(d.records)+1)

	// Add headers as first record if set
	if len(d.headers) > 0 {
		headerNodes := make([]ast.SchemaNode, len(d.headers))
		for i, h := range d.headers {
			headerNodes[i] = ast.NewLiteralNode(h, ast.ZeroPosition())
		}
		allRecords = append(allRecords, ast.NewArrayDataNode(headerNodes, ast.ZeroPosition()))
	}

	// Add data records
	for _, record := range d.records {
		fieldNodes := make([]ast.SchemaNode, len(record))
		for i, f := range record {
			fieldNodes[i] = ast.NewLiteralNode(f, ast.ZeroPosition())
		}
		allRecords = append(allRecords, ast.NewArrayDataNode(fieldNodes, ast.ZeroPosition()))
	}

	return ast.NewArrayDataNode(allRecords, ast.ZeroPosition()), nil
}

// FromAST creates a Document from an AST ArrayDataNode.
// This is useful for integration with other Shape parsers.
func FromAST(node ast.SchemaNode) (*Document, error) {
	arrayNode, ok := node.(*ast.ArrayDataNode)
	if !ok {
		return nil, fmt.Errorf("expected *ast.ArrayDataNode, got %T", node)
	}

	doc := NewDocument()
	elements := arrayNode.Elements()

	for _, elem := range elements {
		recordNode, ok := elem.(*ast.ArrayDataNode)
		if !ok {
			return nil, fmt.Errorf("expected record to be *ast.ArrayDataNode, got %T", elem)
		}

		fields := make([]string, 0, recordNode.Len())
		for _, fieldNode := range recordNode.Elements() {
			literalNode, ok := fieldNode.(*ast.LiteralNode)
			if !ok {
				return nil, fmt.Errorf("expected field to be *ast.LiteralNode, got %T", fieldNode)
			}

			value, ok := literalNode.Value().(string)
			if !ok {
				return nil, fmt.Errorf("expected field value to be string, got %T", literalNode.Value())
			}

			fields = append(fields, value)
		}

		doc.AddRecord(fields)
	}

	return doc, nil
}
