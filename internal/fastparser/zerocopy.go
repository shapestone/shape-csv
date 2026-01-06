package fastparser

import (
	"errors"
	"fmt"
)

// ParseZeroCopy parses CSV data and returns [][]byte slices pointing into the original buffer.
// This is the most memory-efficient parsing mode:
//   - Unquoted fields: returned slices point directly into original data (zero allocations)
//   - Quoted fields without escapes: returned slices point into original data
//   - Quoted fields with escaped quotes (""): new memory allocated only for unescaping
//
// IMPORTANT: The returned slices share memory with the input buffer.
// Do not modify the input buffer while using the returned data.
//
// Performance: Near-zero allocations for simple CSV without escaped quotes.
func ParseZeroCopy(data []byte) ([][][]byte, error) {
	if len(data) == 0 {
		return [][][]byte{}, nil
	}

	p := &zeroCopyParser{
		data:   data,
		pos:    0,
		length: len(data),
	}

	return p.parse()
}

// zeroCopyParser implements zero-copy CSV parsing returning []byte slices.
type zeroCopyParser struct {
	data   []byte
	pos    int
	length int
}

// parse parses the entire CSV file into [][]byte records.
func (p *zeroCopyParser) parse() ([][][]byte, error) {
	records := make([][][]byte, 0, 16)

	// Track field count from first record for pre-allocation
	var capacityHint int

	for p.pos < p.length {
		// Skip empty lines
		if p.isNewline() {
			p.skipNewline()
			continue
		}

		record, err := p.parseRecord(capacityHint)
		if err != nil {
			return nil, err
		}

		// Use first record's field count as capacity hint for subsequent records
		if capacityHint == 0 && len(record) > 0 {
			capacityHint = len(record)
		}

		records = append(records, record)
	}

	return records, nil
}

// parseRecord parses a single CSV record.
func (p *zeroCopyParser) parseRecord(capacityHint int) ([][]byte, error) {
	var fields [][]byte
	if capacityHint > 0 {
		fields = make([][]byte, 0, capacityHint)
	} else {
		fields = make([][]byte, 0, 8)
	}

	for {
		field, err := p.parseField()
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)

		// Check what comes next
		if p.pos >= p.length {
			// End of file
			return fields, nil
		}

		c := p.data[p.pos]
		if c == ',' {
			// More fields coming
			p.pos++
			continue
		}

		if c == '\r' || c == '\n' {
			// End of record
			p.skipNewline()
			return fields, nil
		}

		return nil, fmt.Errorf("unexpected character '%c' at position %d", c, p.pos)
	}
}

// parseField parses a single CSV field and returns a []byte slice.
func (p *zeroCopyParser) parseField() ([]byte, error) {
	if p.pos >= p.length {
		// Empty field at end of file
		return []byte{}, nil
	}

	// Check if field is quoted
	if p.data[p.pos] == '"' {
		return p.parseQuotedField()
	}

	return p.parseUnquotedField()
}

// parseUnquotedField parses an unquoted CSV field.
// Returns a slice pointing directly into the original data (zero-copy).
func (p *zeroCopyParser) parseUnquotedField() ([]byte, error) {
	start := p.pos

	for p.pos < p.length {
		c := p.data[p.pos]

		// Field terminators
		if c == ',' || c == '\r' || c == '\n' {
			break
		}

		// Quote in middle of unquoted field is an error
		if c == '"' {
			return nil, fmt.Errorf("quote character in unquoted field at position %d", p.pos)
		}

		p.pos++
	}

	// Return slice pointing into original data (zero-copy)
	return p.data[start:p.pos], nil
}

// parseQuotedField parses a quoted CSV field.
// If the field contains no escaped quotes, returns a slice pointing into original data.
// If the field contains escaped quotes (""), allocates new memory for unescaping.
func (p *zeroCopyParser) parseQuotedField() ([]byte, error) {
	// Skip opening quote
	p.pos++

	start := p.pos
	hasEscapedQuotes := false

	// First pass: scan to find closing quote and check for escaped quotes
	scanPos := p.pos
	foundClosingQuote := false

	for scanPos < p.length {
		c := p.data[scanPos]

		if c == '"' {
			scanPos++
			// Check if next character is also a quote (escaped quote)
			if scanPos < p.length && p.data[scanPos] == '"' {
				hasEscapedQuotes = true
				scanPos++
				continue
			}

			// Found closing quote
			foundClosingQuote = true
			if !hasEscapedQuotes {
				// No escaped quotes - can return zero-copy slice
				result := p.data[start : scanPos-1]
				p.pos = scanPos
				return result, nil
			}

			// Has escaped quotes - need to allocate and unescape
			break
		}

		scanPos++
	}

	if !foundClosingQuote {
		return nil, errors.New("unclosed quoted field")
	}

	// Second pass: allocate buffer and copy with unescaping
	buf := make([]byte, 0, scanPos-start)

	// Reset position for second pass
	copyStart := p.pos

	for p.pos < p.length {
		c := p.data[p.pos]

		if c == '"' {
			// Add everything up to this quote
			if p.pos > copyStart {
				buf = append(buf, p.data[copyStart:p.pos]...)
			}
			p.pos++

			// Check if next character is also a quote (escaped quote)
			if p.pos < p.length && p.data[p.pos] == '"' {
				// Escaped quote - add single quote to buffer
				buf = append(buf, '"')
				p.pos++
				copyStart = p.pos
				continue
			}

			// Closing quote - we're done
			return buf, nil
		}

		p.pos++
	}

	return nil, errors.New("unclosed quoted field")
}

// isNewline checks if current position is at a newline.
func (p *zeroCopyParser) isNewline() bool {
	if p.pos >= p.length {
		return false
	}
	c := p.data[p.pos]
	return c == '\r' || c == '\n'
}

// skipNewline skips a newline sequence (LF or CRLF).
func (p *zeroCopyParser) skipNewline() {
	if p.pos >= p.length {
		return
	}

	// Handle CRLF
	if p.data[p.pos] == '\r' {
		p.pos++
		if p.pos < p.length && p.data[p.pos] == '\n' {
			p.pos++
		}
		return
	}

	// Handle LF
	if p.data[p.pos] == '\n' {
		p.pos++
	}
}

// ScannerOptions configures Scanner behavior.
type ScannerOptions struct {
	// ReuseRecord causes Scanner to reuse the same []string slice for each record.
	// This reduces allocations but means you must copy data if you need to retain it.
	// Matches the pattern used by encoding/csv.Reader.ReuseRecord.
	ReuseRecord bool
}

// Scanner provides a streaming interface for parsing CSV data.
// It's similar to bufio.Scanner but specialized for CSV.
//
// Example usage:
//
//	scanner := NewScanner(data, ScannerOptions{})
//	for scanner.Scan() {
//	    record := scanner.Record()
//	    // Process record...
//	}
//	if err := scanner.Err(); err != nil {
//	    // Handle error
//	}
type Scanner struct {
	data   []byte
	pos    int
	length int

	// Current record
	record []string

	// Options
	reuseRecord bool

	// Error state
	err error

	// Parser state
	currentRecord []string
}

// NewScanner creates a new CSV scanner.
func NewScanner(data []byte, opts ScannerOptions) *Scanner {
	return &Scanner{
		data:        data,
		pos:         0,
		length:      len(data),
		reuseRecord: opts.ReuseRecord,
	}
}

// Scan advances to the next record.
// Returns true if a record was found, false if end-of-data or error.
// After Scan returns false, check Err() to distinguish between end-of-data and error.
func (s *Scanner) Scan() bool {
	if s.err != nil {
		return false
	}

	if s.pos >= s.length {
		return false
	}

	// Skip empty lines
	for s.pos < s.length && s.isNewline() {
		s.skipNewline()
	}

	if s.pos >= s.length {
		return false
	}

	// Parse next record
	record, err := s.parseRecord()
	if err != nil {
		s.err = err
		return false
	}

	s.currentRecord = record
	return true
}

// Record returns the current record as []string.
// The returned slice is valid until the next call to Scan.
// If ReuseRecord option is enabled, the same slice is reused between calls.
func (s *Scanner) Record() []string {
	return s.currentRecord
}

// Err returns the first error encountered during scanning, if any.
func (s *Scanner) Err() error {
	return s.err
}

// parseRecord parses a single CSV record.
func (s *Scanner) parseRecord() ([]string, error) {
	var fields []string

	if s.reuseRecord && s.record != nil {
		// Reuse existing slice
		fields = s.record[:0]
	} else {
		// Allocate new slice
		fields = make([]string, 0, 8)
	}

	for {
		field, err := s.parseField()
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)

		// Check what comes next
		if s.pos >= s.length {
			// End of file
			if s.reuseRecord {
				s.record = fields
			}
			return fields, nil
		}

		c := s.data[s.pos]
		if c == ',' {
			// More fields coming
			s.pos++
			continue
		}

		if c == '\r' || c == '\n' {
			// End of record
			s.skipNewline()
			if s.reuseRecord {
				s.record = fields
			}
			return fields, nil
		}

		return nil, fmt.Errorf("unexpected character '%c' at position %d", c, s.pos)
	}
}

// parseField parses a single CSV field.
func (s *Scanner) parseField() (string, error) {
	if s.pos >= s.length {
		// Empty field at end of file
		return "", nil
	}

	// Check if field is quoted
	if s.data[s.pos] == '"' {
		return s.parseQuotedField()
	}

	return s.parseUnquotedField()
}

// parseUnquotedField parses an unquoted CSV field.
func (s *Scanner) parseUnquotedField() (string, error) {
	start := s.pos

	for s.pos < s.length {
		c := s.data[s.pos]

		// Field terminators
		if c == ',' || c == '\r' || c == '\n' {
			break
		}

		// Quote in middle of unquoted field is an error
		if c == '"' {
			return "", fmt.Errorf("quote character in unquoted field at position %d", s.pos)
		}

		s.pos++
	}

	// Use unsafe string conversion to avoid allocation
	return unsafeString(s.data[start:s.pos]), nil
}

// parseQuotedField parses a quoted CSV field.
func (s *Scanner) parseQuotedField() (string, error) {
	// Skip opening quote
	s.pos++

	// Get a buffer from the pool
	buf := getBuffer()
	defer putBuffer(buf)

	start := s.pos

	for s.pos < s.length {
		c := s.data[s.pos]

		if c == '"' {
			// Add everything up to this quote
			buf = append(buf, s.data[start:s.pos]...)
			s.pos++

			// Check if next character is also a quote (escaped quote)
			if s.pos < s.length && s.data[s.pos] == '"' {
				// Escaped quote - add single quote to buffer
				buf = append(buf, '"')
				s.pos++
				start = s.pos
				continue
			}

			// Closing quote - we're done
			return string(buf), nil
		}

		s.pos++
	}

	return "", errors.New("unclosed quoted field")
}

// isNewline checks if current position is at a newline.
func (s *Scanner) isNewline() bool {
	if s.pos >= s.length {
		return false
	}
	c := s.data[s.pos]
	return c == '\r' || c == '\n'
}

// skipNewline skips a newline sequence (LF or CRLF).
func (s *Scanner) skipNewline() {
	if s.pos >= s.length {
		return
	}

	// Handle CRLF
	if s.data[s.pos] == '\r' {
		s.pos++
		if s.pos < s.length && s.data[s.pos] == '\n' {
			s.pos++
		}
		return
	}

	// Handle LF
	if s.data[s.pos] == '\n' {
		s.pos++
	}
}
