// Package fastparser implements a high-performance CSV parser without AST construction.
//
// This parser is optimized for the common case of parsing CSV directly into Go types.
// It bypasses tokenization and AST construction, parsing directly from bytes to values.
//
// Performance targets (vs encoding/csv):
//   - Faster or equal parsing speed
//   - Less memory usage
//   - Fewer allocations
package fastparser

import (
	"errors"
	"fmt"
)

// Parse parses CSV data directly from bytes to [][]string without AST construction.
// This is the fastest way to parse CSV when you don't need the AST.
//
// The parser follows RFC 4180:
//   - Fields are separated by commas
//   - Records are separated by newlines (LF or CRLF)
//   - Fields may be quoted with double quotes
//   - Quoted fields may contain commas, newlines, and escaped quotes ("")
//   - Empty lines are skipped
//
// Returns a slice of records, where each record is a slice of field values.
func Parse(data []byte) ([][]string, error) {
	if len(data) == 0 {
		return [][]string{}, nil
	}

	p := &parser{
		data:   data,
		pos:    0,
		length: len(data),
	}

	return p.parse()
}

// parser implements a high-performance CSV parser.
type parser struct {
	data   []byte
	pos    int
	length int
}

// parse parses the entire CSV file using a single backing array for all fields.
func (p *parser) parse() ([][]string, error) {
	// Estimate capacity based on data size
	// Assume average field size of 8 bytes + 1 comma = 9 bytes per field
	estimatedFields := p.length / 9
	if estimatedFields < 64 {
		estimatedFields = 64
	}

	// Pre-allocate backing array and records slice
	backingArray := make([]string, 0, estimatedFields)
	records := make([][]string, 0, estimatedFields/8)

	// Track field count from first record
	fieldsPerRecord := 0
	recordStart := 0

	for p.pos < p.length {
		// Skip empty lines
		if p.isNewline() {
			p.skipNewline()
			continue
		}

		recordStart = len(backingArray)

		// Parse all fields in this record
		for {
			field, err := p.parseField()
			if err != nil {
				return nil, err
			}
			backingArray = append(backingArray, field)

			// Check what comes next
			if p.pos >= p.length {
				break
			}

			c := p.data[p.pos]
			if c == ',' {
				p.pos++
				continue
			}

			if c == '\r' || c == '\n' {
				p.skipNewline()
				break
			}

			return nil, fmt.Errorf("unexpected character '%c' at position %d", c, p.pos)
		}

		// Add the record as a slice of the backing array
		recordEnd := len(backingArray)
		records = append(records, backingArray[recordStart:recordEnd:recordEnd])

		// Use first record's field count for capacity hints
		if fieldsPerRecord == 0 {
			fieldsPerRecord = recordEnd - recordStart
		}
	}

	return records, nil
}

// parseField parses a single CSV field.
func (p *parser) parseField() (string, error) {
	if p.pos >= p.length {
		return "", nil
	}

	if p.data[p.pos] == '"' {
		return p.parseQuotedField()
	}

	return p.parseUnquotedField()
}

// parseQuotedField parses a quoted CSV field.
func (p *parser) parseQuotedField() (string, error) {
	p.pos++ // Skip opening quote
	start := p.pos

	// Scan for closing quote, checking for escapes
	for i := p.pos; i < p.length; i++ {
		if p.data[i] == '"' {
			if i+1 < p.length && p.data[i+1] == '"' {
				// Escaped quote - need slow path
				return p.parseQuotedFieldSlow(start)
			}
			// Simple quoted field - zero copy
			result := unsafeString(p.data[start:i])
			p.pos = i + 1
			return result, nil
		}
	}

	return "", errors.New("unclosed quoted field")
}

// parseQuotedFieldSlow handles quoted fields with escaped quotes.
func (p *parser) parseQuotedFieldSlow(start int) (string, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	for p.pos < p.length {
		c := p.data[p.pos]

		if c == '"' {
			buf = append(buf, p.data[start:p.pos]...)
			p.pos++

			if p.pos < p.length && p.data[p.pos] == '"' {
				buf = append(buf, '"')
				p.pos++
				start = p.pos
				continue
			}

			return string(buf), nil
		}

		p.pos++
	}

	return "", errors.New("unclosed quoted field")
}

// parseUnquotedField parses an unquoted CSV field using zero-copy.
func (p *parser) parseUnquotedField() (string, error) {
	start := p.pos

	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ',' || c == '\r' || c == '\n' {
			break
		}
		if c == '"' {
			return "", fmt.Errorf("quote character in unquoted field at position %d", p.pos)
		}
		p.pos++
	}

	return unsafeString(p.data[start:p.pos]), nil
}

// isNewline checks if current position is at a newline.
func (p *parser) isNewline() bool {
	if p.pos >= p.length {
		return false
	}
	c := p.data[p.pos]
	return c == '\r' || c == '\n'
}

// skipNewline skips a newline sequence (LF or CRLF).
func (p *parser) skipNewline() {
	if p.pos >= p.length {
		return
	}
	if p.data[p.pos] == '\r' {
		p.pos++
		if p.pos < p.length && p.data[p.pos] == '\n' {
			p.pos++
		}
		return
	}
	if p.data[p.pos] == '\n' {
		p.pos++
	}
}
