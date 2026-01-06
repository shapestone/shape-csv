// Package parser implements LL(1) recursive descent parsing for CSV format.
// Each production rule in the grammar (docs/grammar/csv.ebnf) corresponds to a parse function.
package parser

import (
	"fmt"
	"strings"

	"github.com/shapestone/shape-core/pkg/ast"
	shapetokenizer "github.com/shapestone/shape-core/pkg/tokenizer"
	"github.com/shapestone/shape-csv/internal/tokenizer"
)

// BadLineMode specifies how to handle malformed lines.
type BadLineMode int

const (
	// BadLineModeError returns an error on malformed lines (default).
	BadLineModeError BadLineMode = iota
	// BadLineModeWarn logs a warning but continues parsing.
	BadLineModeWarn
	// BadLineModeSkip silently skips malformed lines.
	BadLineModeSkip
)

// Options configures the parser behavior.
type Options struct {
	// Comma is the field delimiter. Default: ','
	Comma rune
	// Comment is the comment character. Lines starting with this are skipped. Default: 0 (disabled)
	Comment rune
	// FieldsPerRecord validates field count. 0=first record sets count, negative=no validation
	FieldsPerRecord int
	// LazyQuotes allows quotes in unquoted fields
	LazyQuotes bool
	// TrimLeadingSpace trims leading whitespace from fields
	TrimLeadingSpace bool
	// OnBadLine specifies how to handle malformed lines. Default: BadLineModeError
	OnBadLine BadLineMode
	// MaxFieldSize is the maximum allowed size for a single field in bytes. 0 means no limit.
	MaxFieldSize int
	// MaxRecordSize is the maximum allowed size for a single record in bytes. 0 means no limit.
	MaxRecordSize int
	// WarningCallback is invoked for warnings when OnBadLine is BadLineModeWarn
	WarningCallback func(line int, message string)
}

// DefaultOptions returns default parser options.
// Note: FieldsPerRecord defaults to -1 (no validation) for backward compatibility.
// Set to 0 for encoding/csv-compatible behavior where first record sets expected count.
func DefaultOptions() Options {
	return Options{
		Comma:            ',',
		Comment:          0,
		FieldsPerRecord:  -1, // No validation by default for backward compatibility
		LazyQuotes:       false,
		TrimLeadingSpace: false,
	}
}

// Parser implements LL(1) recursive descent parsing for CSV.
// It maintains a single token lookahead for predictive parsing.
type Parser struct {
	tokenizer      *shapetokenizer.Tokenizer
	current        *shapetokenizer.Token
	hasToken       bool
	opts           Options
	expectedFields int // Set from first record when FieldsPerRecord is 0
	currentLine    int
	currentColumn  int
}

// NewParser creates a new CSV parser for the given input string.
// For parsing from io.Reader, use NewParserFromStream instead.
func NewParser(input string) *Parser {
	return NewParserWithOptions(input, DefaultOptions())
}

// NewParserWithOptions creates a new CSV parser with custom options.
func NewParserWithOptions(input string, opts Options) *Parser {
	return newParserWithStreamAndOptions(shapetokenizer.NewStream(input), opts)
}

// NewParserFromStream creates a new CSV parser using a pre-configured stream.
// This allows parsing from io.Reader using tokenizer.NewStreamFromReader.
func NewParserFromStream(stream shapetokenizer.Stream) *Parser {
	return NewParserFromStreamWithOptions(stream, DefaultOptions())
}

// NewParserFromStreamWithOptions creates a new CSV parser from a stream with custom options.
func NewParserFromStreamWithOptions(stream shapetokenizer.Stream, opts Options) *Parser {
	return newParserWithStreamAndOptions(stream, opts)
}

// newParserWithStreamAndOptions is the internal constructor that accepts a stream and options.
func newParserWithStreamAndOptions(stream shapetokenizer.Stream, opts Options) *Parser {
	// Create tokenizer with matching delimiter option
	tokOpts := tokenizer.Options{
		Comma: opts.Comma,
	}
	tok := tokenizer.NewTokenizerWithStreamAndOptions(stream, tokOpts)

	p := &Parser{
		tokenizer:      &tok,
		opts:           opts,
		expectedFields: opts.FieldsPerRecord,
		currentLine:    1,
		currentColumn:  1,
	}
	p.advance() // Load first token
	return p
}

// Parse parses the input and returns an AST representing the CSV file.
//
// Grammar:
//
//	File = [ Header ] { Record } ;
//
// Returns *ast.ArrayDataNode - an array of records, where each record is an ArrayDataNode of fields.
// For CSV data, each field is a LiteralNode containing a string value.
func (p *Parser) Parse() (ast.SchemaNode, error) {
	records := make([]ast.SchemaNode, 0, 16)
	recordNum := 0

	// Parse records until EOF
	for p.hasToken {
		// Skip empty lines (just newlines)
		if p.peek() != nil && p.peek().Kind() == tokenizer.TokenNewline {
			p.advance()
			continue
		}

		// Check for EOF
		if !p.hasToken {
			break
		}

		// Skip comment lines if comment character is set
		if p.opts.Comment != 0 && p.isCommentLine() {
			p.skipLine()
			continue
		}

		record, err := p.parseRecord()
		if err != nil {
			// Handle error based on OnBadLine mode
			if err := p.handleBadLine(err); err != nil {
				return nil, err
			}
			// Skip to next line and continue
			p.skipLine()
			continue
		}

		// Validate field count
		fieldCount := len(record.Elements())
		if p.opts.FieldsPerRecord >= 0 {
			if recordNum == 0 && p.opts.FieldsPerRecord == 0 {
				// First record sets expected count
				p.expectedFields = fieldCount
			} else if p.expectedFields > 0 && fieldCount != p.expectedFields {
				fieldErr := fmt.Errorf("record on line %d: wrong number of fields (got %d, expected %d)",
					p.currentLine, fieldCount, p.expectedFields)
				if err := p.handleBadLine(fieldErr); err != nil {
					return nil, err
				}
				// Skip this record
				continue
			}
		}

		// Check record size limit
		if p.opts.MaxRecordSize > 0 {
			recordSize := p.calculateRecordSize(record)
			if recordSize > p.opts.MaxRecordSize {
				sizeErr := fmt.Errorf("record on line %d exceeds maximum size (%d > %d)",
					p.currentLine, recordSize, p.opts.MaxRecordSize)
				if err := p.handleBadLine(sizeErr); err != nil {
					return nil, err
				}
				continue
			}
		}

		records = append(records, record)
		recordNum++
	}

	return ast.NewArrayDataNode(records, ast.ZeroPosition()), nil
}

// handleBadLine handles a parsing error based on OnBadLine mode.
// Returns nil if parsing should continue, or the error if it should stop.
func (p *Parser) handleBadLine(err error) error {
	switch p.opts.OnBadLine {
	case BadLineModeSkip:
		// Silently skip
		return nil
	case BadLineModeWarn:
		// Log warning and continue
		if p.opts.WarningCallback != nil {
			p.opts.WarningCallback(p.currentLine, err.Error())
		}
		return nil
	default:
		// BadLineModeError - return the error
		return err
	}
}

// calculateRecordSize calculates the total size of a record in bytes.
func (p *Parser) calculateRecordSize(record *ast.ArrayDataNode) int {
	size := 0
	for _, elem := range record.Elements() {
		if lit, ok := elem.(*ast.LiteralNode); ok {
			if s, ok := lit.Value().(string); ok {
				size += len(s)
			}
		}
	}
	return size
}

// parseRecord parses a single CSV record.
//
// Grammar:
//
//	Record = Field { "," Field } LineTerminator ;
//
// Returns *ast.ArrayDataNode representing the record (array of field values).
func (p *Parser) parseRecord() (*ast.ArrayDataNode, error) {
	startPos := p.position()
	fields := make([]ast.SchemaNode, 0, 8)

	// Parse first field
	field, err := p.parseField()
	if err != nil {
		return nil, err
	}
	fields = append(fields, field)

	// Parse additional fields: { "," Field }
	for p.peek() != nil && p.peek().Kind() == tokenizer.TokenComma {
		p.advance() // consume comma

		field, err := p.parseField()
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	// Consume line terminator (newline or EOF)
	if p.peek() != nil && p.peek().Kind() == tokenizer.TokenNewline {
		p.advance()
	}
	// EOF is also a valid line terminator (no need to advance)

	return ast.NewArrayDataNode(fields, startPos), nil
}

// parseField parses a single CSV field.
//
// Grammar:
//
//	Field = QuotedField | UnquotedField ;
//
// Returns *ast.LiteralNode with string value.
func (p *Parser) parseField() (*ast.LiteralNode, error) {
	startPos := p.position()

	// Handle TrimLeadingSpace: skip leading whitespace tokens before the actual field
	if p.opts.TrimLeadingSpace {
		for p.peek() != nil && p.peek().Kind() == tokenizer.TokenField {
			value := p.peek().ValueString()
			trimmed := strings.TrimLeft(value, " \t")
			if len(trimmed) == len(value) {
				// No leading space to trim
				break
			}
			if len(trimmed) == 0 {
				// Entire token was whitespace, skip it
				p.advance()
				continue
			}
			// Partial whitespace - need to handle this in parseUnquotedField
			break
		}
	}

	var field *ast.LiteralNode
	var err error

	// Check if field starts with quote
	if p.peek() != nil && p.peek().Kind() == tokenizer.TokenDQuote {
		field, err = p.parseQuotedField()
	} else {
		// Otherwise parse unquoted field
		field, err = p.parseUnquotedField(startPos)
	}

	if err != nil {
		return nil, err
	}

	// Check field size limit
	if p.opts.MaxFieldSize > 0 && field != nil {
		if s, ok := field.Value().(string); ok {
			if len(s) > p.opts.MaxFieldSize {
				return nil, fmt.Errorf("field at %s exceeds maximum size (%d > %d)",
					startPos.String(), len(s), p.opts.MaxFieldSize)
			}
		}
	}

	return field, nil
}

// parseQuotedField parses a quoted CSV field.
//
// Grammar:
//
//	QuotedField = '"' { QuotedChar | EscapedQuote } '"' ;
//	EscapedQuote = '""' ;
//
// Returns *ast.LiteralNode with unescaped string value.
// Handles embedded quotes (escaped as ""), commas, and newlines.
func (p *Parser) parseQuotedField() (*ast.LiteralNode, error) {
	startPos := p.position()

	// Consume opening quote
	if err := p.expect(tokenizer.TokenDQuote); err != nil {
		return nil, err
	}

	var value strings.Builder

	// Parse field content until closing quote
	for {
		token := p.peek()
		if token == nil || !p.hasToken {
			return nil, fmt.Errorf("unclosed quoted field at %s", startPos.String())
		}

		kind := token.Kind()

		if kind == tokenizer.TokenDQuote {
			// Found a quote - could be closing quote or escaped quote
			p.advance() // consume the quote

			// Check if next token is also a quote (escaped quote: "")
			nextToken := p.peek()
			if nextToken != nil && nextToken.Kind() == tokenizer.TokenDQuote {
				// Escaped quote - add single quote to value
				value.WriteByte('"')
				p.advance() // consume second quote
			} else {
				// Closing quote - we're done
				return ast.NewLiteralNode(value.String(), startPos), nil
			}
		} else if kind == tokenizer.TokenField {
			// Field content
			value.WriteString(token.ValueString())
			p.advance()
		} else if kind == tokenizer.TokenComma {
			// Delimiter inside quoted field - treat as literal
			value.WriteRune(p.opts.Comma)
			p.advance()
		} else if kind == tokenizer.TokenNewline {
			// Newline inside quoted field - treat as literal
			// Detect CRLF vs LF by checking the token value
			tokenValue := token.ValueString()
			if tokenValue == "\r\n" {
				value.WriteString("\r\n")
			} else {
				value.WriteByte('\n')
			}
			p.advance()
		} else {
			return nil, fmt.Errorf("unexpected token %s in quoted field at %s", kind, p.positionStr())
		}
	}
}

// parseUnquotedField parses an unquoted CSV field.
//
// Grammar:
//
//	UnquotedField = { UnquotedChar } ;
//	UnquotedChar = [^,"\r\n] ;
//
// Returns *ast.LiteralNode with string value.
// Unquoted fields cannot contain commas, quotes, or newlines (unless LazyQuotes is enabled).
func (p *Parser) parseUnquotedField(startPos ast.Position) (*ast.LiteralNode, error) {
	// Empty field (e.g., between commas: "a,,b")
	token := p.peek()
	if token == nil || token.Kind() == tokenizer.TokenComma || token.Kind() == tokenizer.TokenNewline {
		return ast.NewLiteralNode("", startPos), nil
	}

	// With LazyQuotes, we need to consume all content until the next comma or newline
	if p.opts.LazyQuotes {
		var value strings.Builder

		for p.peek() != nil {
			tok := p.peek()
			if tok.Kind() == tokenizer.TokenComma || tok.Kind() == tokenizer.TokenNewline {
				break
			}
			if tok.Kind() == tokenizer.TokenField {
				value.WriteString(tok.ValueString())
			} else if tok.Kind() == tokenizer.TokenDQuote {
				value.WriteByte('"')
			}
			p.advance()
		}

		result := value.String()
		if p.opts.TrimLeadingSpace {
			result = p.trimLeadingSpace(result)
		}
		return ast.NewLiteralNode(result, startPos), nil
	}

	// Strict mode: quotes are not allowed in unquoted fields
	if token.Kind() == tokenizer.TokenField {
		value := token.ValueString()
		p.advance()

		// Apply TrimLeadingSpace if enabled
		value = p.trimLeadingSpace(value)

		// Check for invalid quote in middle of unquoted field
		if strings.ContainsRune(value, '"') {
			return nil, fmt.Errorf("quote character in unquoted field at %s", startPos.String())
		}

		return ast.NewLiteralNode(value, startPos), nil
	}

	// Quote at start of what should be unquoted field
	if token.Kind() == tokenizer.TokenDQuote {
		return nil, fmt.Errorf("quote character in unquoted field at %s", startPos.String())
	}

	// Empty field at end of line
	return ast.NewLiteralNode("", startPos), nil
}

// Helper methods

// peek returns current token without advancing.
func (p *Parser) peek() *shapetokenizer.Token {
	return p.current
}

// advance moves to next token.
func (p *Parser) advance() {
	token, ok := p.tokenizer.NextToken()
	if ok {
		p.current = token
		p.hasToken = true
	} else {
		p.hasToken = false
		p.current = nil
	}
}

// expect consumes token of expected kind or returns error.
func (p *Parser) expect(kind string) error {
	if p.peek() == nil || p.peek().Kind() != kind {
		return fmt.Errorf("expected %s at %s, got %s",
			kind, p.positionStr(), p.peek().Kind())
	}
	p.advance()
	return nil
}

// position returns current position for AST nodes.
func (p *Parser) position() ast.Position {
	if p.hasToken && p.current != nil {
		return ast.NewPosition(
			p.current.Offset(),
			p.current.Row(),
			p.current.Column(),
		)
	}
	return ast.ZeroPosition()
}

// positionStr returns current position as a string for error messages.
func (p *Parser) positionStr() string {
	return p.position().String()
}

// isCommentLine checks if the current line starts with the comment character.
func (p *Parser) isCommentLine() bool {
	token := p.peek()
	if token == nil || token.Kind() != tokenizer.TokenField {
		return false
	}
	value := token.ValueString()
	if len(value) == 0 {
		return false
	}
	return rune(value[0]) == p.opts.Comment
}

// skipLine advances past all tokens until the next newline or EOF.
func (p *Parser) skipLine() {
	for p.hasToken {
		if p.peek() != nil && p.peek().Kind() == tokenizer.TokenNewline {
			p.advance()
			return
		}
		p.advance()
	}
}

// trimLeadingSpace removes leading whitespace from a string if TrimLeadingSpace is enabled.
func (p *Parser) trimLeadingSpace(s string) string {
	if !p.opts.TrimLeadingSpace {
		return s
	}
	return strings.TrimLeft(s, " \t")
}
