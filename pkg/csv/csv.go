// Package csv provides CSV format parsing and AST generation.
//
// This package implements a complete CSV parser following RFC 4180.
// It parses CSV data into Shape's unified AST representation.
//
// Grammar: See docs/grammar/csv.ebnf for the complete EBNF specification.
//
// This parser uses LL(1) recursive descent parsing (see Shape ADR 0004).
// Each production rule in the grammar corresponds to a parse function in internal/parser/parser.go.
//
// # Thread Safety
//
// All functions in this package are safe for concurrent use by multiple goroutines.
// Each function call creates its own parser instance with no shared mutable state.
//
//	// Safe: Concurrent parsing
//	go func() { csv.Parse(input1) }()
//	go func() { csv.Parse(input2) }()
//	go func() { csv.Unmarshal(data, &v) }()
//
// # Parsing APIs
//
// The package provides two parsing functions:
//
//   - Parse(string) - Parses CSV from a string in memory
//   - ParseReader(io.Reader) - Parses CSV from any io.Reader with streaming support
//
// Use Parse() for small CSV documents that are already in memory as strings.
// Use ParseReader() for large files, network streams, or any io.Reader source.
//
// # Example usage with Parse:
//
//	csvStr := "name,age\nAlice,30\nBob,25"
//	node, err := csv.Parse(csvStr)
//	if err != nil {
//	    // handle error
//	}
//	// node is now a *ast.ArrayDataNode representing the CSV data
//
// # Example usage with ParseReader:
//
//	file, err := os.Open("data.csv")
//	if err != nil {
//	    // handle error
//	}
//	defer file.Close()
//
//	node, err := csv.ParseReader(file)
//	if err != nil {
//	    // handle error
//	}
//	// node is now a *ast.ArrayDataNode representing the CSV data
package csv

import (
	"errors"
	"fmt"
	"io"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-core/pkg/tokenizer"
	"github.com/shapestone/shape-csv/internal/fastparser"
	"github.com/shapestone/shape-csv/internal/parser"
)

// Parse parses CSV format into an AST from a string.
//
// The input is a complete CSV document with optional header and data rows.
//
// Returns an ast.ArrayDataNode representing the parsed CSV:
//   - *ast.ArrayDataNode for the file (array of records)
//   - Each record is an *ast.ArrayDataNode of fields
//   - Each field is an *ast.LiteralNode containing a string value
//
// For parsing large files or streaming data, use ParseReader instead.
//
// Example:
//
//	node, err := csv.Parse("name,age\nAlice,30\nBob,25")
//	arrayNode := node.(*ast.ArrayDataNode)
//	records := arrayNode.Elements()
//	// records[0] is the header row
//	// records[1] is the first data row
func Parse(input string) (ast.SchemaNode, error) {
	p := parser.NewParser(input)
	return p.Parse()
}

// ParseReader parses CSV format into an AST from an io.Reader.
//
// This function is designed for parsing large CSV files or streaming data with
// constant memory usage. It uses a buffered stream implementation that reads data
// in chunks, making it suitable for files that don't fit entirely in memory.
//
// The reader can be any io.Reader implementation:
//   - os.File for reading from files
//   - strings.Reader for reading from strings
//   - bytes.Buffer for reading from byte slices
//   - Network streams, compressed streams, etc.
//
// Returns an ast.ArrayDataNode representing the parsed CSV:
//   - *ast.ArrayDataNode for the file (array of records)
//   - Each record is an *ast.ArrayDataNode of fields
//   - Each field is an *ast.LiteralNode containing a string value
//
// Example parsing from a file:
//
//	file, err := os.Open("data.csv")
//	if err != nil {
//	    // handle error
//	}
//	defer file.Close()
//
//	node, err := csv.ParseReader(file)
//	if err != nil {
//	    // handle error
//	}
//	// node is now a *ast.ArrayDataNode representing the CSV data
//
// Example parsing from a string:
//
//	reader := strings.NewReader("name,age\nAlice,30")
//	node, err := csv.ParseReader(reader)
func ParseReader(reader io.Reader) (ast.SchemaNode, error) {
	stream := tokenizer.NewStreamFromReader(reader)
	p := parser.NewParserFromStream(stream)
	return p.Parse()
}

// Format returns the format identifier for this parser.
// Returns "CSV" to identify this as the CSV data format parser.
func Format() string {
	return "CSV"
}

// Validate checks if the input string is valid CSV.
//
// This function uses a high-performance fast path that bypasses AST construction.
//
// Returns nil if the input is valid CSV.
// Returns an error with details about why the CSV is invalid.
//
// This is the idiomatic Go approach - check the error:
//
//	if err := csv.Validate(input); err != nil {
//	    // Invalid CSV
//	    fmt.Println("Invalid CSV:", err)
//	}
//	// Valid CSV - err is nil
//
// Valid CSV includes:
//   - Simple fields: name,age
//   - Quoted fields: "name","age"
//   - Empty fields: a,,c
//   - Escaped quotes: "field with ""quotes"""
//   - Newlines in quoted fields: "field\nwith\nnewlines"
//
// Every record must have the same number of fields as the first one, as in
// encoding/csv and shape v0.4.0: "a,b\n1,2,3" and "a,b\n1" are invalid. The
// error is a *ParseError wrapping ErrFieldCount, with the line on which the
// offending record starts. Malformed quoting is a *ParseError wrapping
// ErrQuote, with the line and column of the problem. Line breaks are \n,
// \r\n and a lone \r (unlike encoding/csv, which keeps a lone \r inside the
// field); each counts as one line for error positions. Blank lines are skipped. Use ValidateWithOptions
// to allow ragged records.
func Validate(input string) error {
	return validateBytes([]byte(input))
}

// validateBytes runs the fast validator and converts a field-count failure
// into a *ParseError that wraps ErrFieldCount.
func validateBytes(data []byte) error {
	err := fastparser.Validate(data)
	var se *fastparser.SyntaxError
	if errors.As(err, &se) {
		return &ParseError{
			StartLine: se.Line,
			Line:      se.Line,
			Column:    se.Column,
			Err:       fmt.Errorf("%w: %v", ErrQuote, se.Err),
		}
	}
	var fc *fastparser.FieldCountError
	if errors.As(err, &fc) {
		return &ParseError{
			StartLine: fc.Line,
			Line:      fc.Line,
			Column:    1,
			Err:       fmt.Errorf("%w (got %d, expected %d)", ErrFieldCount, fc.Got, fc.Want),
		}
	}
	return err
}

// ParseLine parses a single CSV record (RFC 4180) into its fields.
//
// It is the counterpart of ParseLine in shape v0.4.0, with an added error for
// malformed input: an unclosed quote, a quote inside an unquoted field,
// characters after a closing quote, or more than one record. Quoted fields
// may contain commas, line breaks and doubled quotes; one trailing line
// terminator is ignored. An empty line yields a single empty field.
//
// Example:
//
//	fields, err := csv.ParseLine(`"Smith, John",30,NYC`)
//	// fields == []string{"Smith, John", "30", "NYC"}
func ParseLine(line string) ([]string, error) {
	return fastparser.ParseLine(line)
}

// ValidateReader checks if the input from an io.Reader is valid CSV.
//
// This function uses a high-performance fast path that bypasses AST construction.
//
// Returns nil if the input is valid CSV.
// Returns an error with details about why the CSV is invalid.
// This reads the entire input from the reader.
//
// This is the idiomatic Go approach - check the error:
//
//	file, _ := os.Open("data.csv")
//	defer file.Close()
//	if err := csv.ValidateReader(file); err != nil {
//	    // Invalid CSV
//	    fmt.Println("Invalid CSV:", err)
//	}
//	// Valid CSV - err is nil
//
// It applies the same rules as Validate, including the field-count check.
// For large streams, consider using ParseReader directly and handling errors.
func ValidateReader(reader io.Reader) error {
	// Fast path: Read all data and parse without AST
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	return validateBytes(data)
}
