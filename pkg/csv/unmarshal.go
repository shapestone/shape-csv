package csv

import (
	"github.com/shapestone/shape-csv/internal/fastparser"
)

// Unmarshal parses the CSV-encoded data and stores the result in the value pointed to by v.
//
// This function uses a high-performance fast path that bypasses AST construction for
// optimal performance. If you need the AST for advanced features, use
// Parse() followed by conversion or manual AST traversal.
//
// Unmarshal supports two target types:
//
// 1. [][]string - Returns raw CSV records (fastest, comparable to encoding/csv):
//
//	var records [][]string
//	err := csv.Unmarshal(data, &records)
//	// records[0] is the header row, records[1:] are data rows
//
// 2. []struct - Maps CSV to struct fields using headers:
//
//	type Person struct {
//	    Name string `csv:"name"`
//	    Age  int    `csv:"age"`
//	}
//	var people []Person
//	err := csv.Unmarshal(data, &people)
//
// For struct mapping, Unmarshal matches incoming CSV headers to struct fields
// using the following rules:
//   - Exact match with csv tag name (case-insensitive)
//   - Exact match with struct field name (case-insensitive)
//
// Unmarshal will only set exported struct fields.
//
// The csv tag format is:
//
//	Field int `csv:"column_name"`           // Map to CSV column "column_name"
//	Field int `csv:"column_name,omitempty"` // Map to CSV column, omit if empty when marshaling
//	Field int `csv:"-"`                      // Always ignore this field
//	Field int                                // Use struct field name as column name
//
// Supported field types:
//   - string
//   - int, int8, int16, int32, int64
//   - uint, uint8, uint16, uint32, uint64
//   - float32, float64
//   - bool (accepts: true/false, 1/0, t/f, T/F, TRUE/FALSE)
//   - pointers to any of the above (nil for empty values)
//
// If a CSV column is not found in the struct, it is ignored.
// If a struct field is not found in the CSV, it is left with its zero value.
func Unmarshal(data []byte, v interface{}) error {
	// Fast path: Direct parsing without AST construction (4-5x faster)
	return fastparser.Unmarshal(data, v)
}
