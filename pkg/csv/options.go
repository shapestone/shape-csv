// Package csv provides configurable options for CSV parsing and writing.
package csv

import (
	"io"
	"unicode/utf8"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-core/pkg/tokenizer"
	"github.com/shapestone/shape-csv/internal/parser"
)

// ReaderOptions configures CSV parsing behavior.
// These options mirror the encoding/csv.Reader configuration.
type ReaderOptions struct {
	// Comma is the field delimiter.
	// It must be a valid rune and not \r, \n, or the Unicode replacement character (0xFFFD).
	// Default: ','
	Comma rune

	// Comment, if not 0, is the comment character. Lines beginning with the
	// Comment character without preceding whitespace are ignored.
	// Default: 0 (disabled)
	Comment rune

	// FieldsPerRecord is the expected number of fields per record.
	// If positive, each record must have exactly this many fields.
	// If 0, the first record determines the expected field count.
	// If negative, no field count validation is performed.
	// Default: 0
	FieldsPerRecord int

	// LazyQuotes controls whether a quote may appear in an unquoted field
	// and a non-doubled quote may appear in a quoted field.
	// Default: false
	LazyQuotes bool

	// TrimLeadingSpace controls whether leading white space in a field is ignored.
	// This is done even if the field delimiter (Comma) is white space.
	// Default: false
	TrimLeadingSpace bool

	// ReuseRecord controls whether calls to Read may return a slice sharing
	// the backing array of the previous call's returned slice for performance.
	// Default: false
	ReuseRecord bool
}

// DefaultReaderOptions returns the default reader configuration.
// Note: FieldsPerRecord defaults to -1 (no validation) for backward compatibility.
// Set to 0 for encoding/csv-compatible behavior where first record sets expected count.
func DefaultReaderOptions() ReaderOptions {
	return ReaderOptions{
		Comma:            ',',
		Comment:          0,
		FieldsPerRecord:  -1, // No validation by default for backward compatibility
		LazyQuotes:       false,
		TrimLeadingSpace: false,
		ReuseRecord:      false,
	}
}

// WriterOptions configures CSV writing behavior.
// These options mirror the encoding/csv.Writer configuration.
type WriterOptions struct {
	// Comma is the field delimiter.
	// Default: ','
	Comma rune

	// UseCRLF controls whether to use \r\n (true) or \n (false) as the line terminator.
	// Default: false (use \n)
	UseCRLF bool
}

// DefaultWriterOptions returns the default writer configuration.
func DefaultWriterOptions() WriterOptions {
	return WriterOptions{
		Comma:   ',',
		UseCRLF: false,
	}
}

// ParseWithOptions parses CSV format into an AST from a string with custom options.
//
// Example:
//
//	opts := csv.DefaultReaderOptions()
//	opts.Comma = '\t'  // Tab-separated
//	opts.TrimLeadingSpace = true
//	node, err := csv.ParseWithOptions("name\tage\nAlice\t30", opts)
func ParseWithOptions(input string, opts ReaderOptions) (ast.SchemaNode, error) {
	p := parser.NewParserWithOptions(input, parser.Options{
		Comma:            opts.Comma,
		Comment:          opts.Comment,
		FieldsPerRecord:  opts.FieldsPerRecord,
		LazyQuotes:       opts.LazyQuotes,
		TrimLeadingSpace: opts.TrimLeadingSpace,
	})
	return p.Parse()
}

// ParseReaderWithOptions parses CSV format into an AST from an io.Reader with custom options.
//
// Example:
//
//	opts := csv.DefaultReaderOptions()
//	opts.Comment = '#'  // Skip comment lines
//	node, err := csv.ParseReaderWithOptions(file, opts)
func ParseReaderWithOptions(reader io.Reader, opts ReaderOptions) (ast.SchemaNode, error) {
	stream := tokenizer.NewStreamFromReader(reader)
	p := parser.NewParserFromStreamWithOptions(stream, parser.Options{
		Comma:            opts.Comma,
		Comment:          opts.Comment,
		FieldsPerRecord:  opts.FieldsPerRecord,
		LazyQuotes:       opts.LazyQuotes,
		TrimLeadingSpace: opts.TrimLeadingSpace,
	})
	return p.Parse()
}

// ValidateWithOptions checks if the input string is valid CSV with custom options.
//
// Example:
//
//	opts := csv.DefaultReaderOptions()
//	opts.Comma = ';'  // Semicolon-separated
//	err := csv.ValidateWithOptions("a;b;c", opts)
func ValidateWithOptions(input string, opts ReaderOptions) error {
	_, err := ParseWithOptions(input, opts)
	return err
}

// RenderWithOptions converts an AST node to CSV bytes with custom options.
//
// Example:
//
//	opts := csv.DefaultWriterOptions()
//	opts.Comma = '\t'
//	opts.UseCRLF = true
//	bytes, err := csv.RenderWithOptions(node, opts)
func RenderWithOptions(node ast.SchemaNode, opts WriterOptions) ([]byte, error) {
	return renderWithOptions(node, opts)
}

// Reader wraps a ReaderOptions and provides position tracking.
// This mirrors encoding/csv.Reader's FieldPos and InputOffset methods.
type Reader struct {
	opts         ReaderOptions
	lastLine     int
	lastColumn   int
	inputOffset  int64
	fieldOffsets []int // byte offsets for each field in current record
}

// NewReader creates a Reader with the given options.
func NewReader(opts ReaderOptions) *Reader {
	return &Reader{
		opts:         opts,
		lastLine:     1,
		lastColumn:   1,
		inputOffset:  0,
		fieldOffsets: make([]int, 0),
	}
}

// FieldPos returns the line and column (in bytes) of the field with the given index
// in the record most recently returned by Read.
// Columns are 1-indexed; lines are 1-indexed.
func (r *Reader) FieldPos(field int) (line, column int) {
	return r.lastLine, r.lastColumn
}

// InputOffset returns the input byte offset of the end of the most recently read row.
func (r *Reader) InputOffset() int64 {
	return r.inputOffset
}

// SetOffset updates the reader's position tracking.
// This is used internally after parsing.
func (r *Reader) SetOffset(line, column int, offset int64) {
	r.lastLine = line
	r.lastColumn = column
	r.inputOffset = offset
}

// validDelim reports whether r is a valid field delimiter.
func validDelim(r rune) bool {
	return r != 0 && r != '"' && r != '\r' && r != '\n' && utf8.ValidRune(r) && r != utf8.RuneError
}

// ValidateOptions checks if the options are valid.
// Returns an error if the options are invalid.
func (o ReaderOptions) Validate() error {
	if !validDelim(o.Comma) {
		return &OptionsError{Field: "Comma", Message: "invalid delimiter"}
	}
	if o.Comment != 0 && !validDelim(o.Comment) {
		return &OptionsError{Field: "Comment", Message: "invalid comment character"}
	}
	if o.Comment == o.Comma {
		return &OptionsError{Field: "Comment", Message: "comment character same as delimiter"}
	}
	return nil
}

// Validate checks if the writer options are valid.
func (o WriterOptions) Validate() error {
	if !validDelim(o.Comma) {
		return &OptionsError{Field: "Comma", Message: "invalid delimiter"}
	}
	return nil
}

// OptionsError represents an invalid option configuration.
type OptionsError struct {
	Field   string
	Message string
}

func (e *OptionsError) Error() string {
	return "csv: invalid " + e.Field + ": " + e.Message
}
