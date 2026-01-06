package fastparser

import (
	"errors"
	"fmt"
)

// ByteRecord represents a CSV record using the BurntSushi offset tracking pattern.
// Instead of storing field values as strings, it stores the raw byte data and
// integer offsets marking field boundaries. This enables:
//   - Zero-copy field access via FieldBytes()
//   - Lazy string conversion via Field()
//   - Significant memory reduction for large datasets
//
// The offsets array contains the start position of each field.
// Field i spans from offsets[i] to offsets[i+1] (or end of data for last field).
type ByteRecord struct {
	data    []byte // The raw CSV record data
	offsets []int  // Start position of each field in data
}

// NewByteRecord creates a ByteRecord from raw data and field offsets.
// The offsets array should contain the start position of each field.
// For a record with N fields, offsets should have N elements, plus one
// sentinel value at the end marking the end of the last field.
//
// Example:
//   data = "abc,def,ghi"
//   offsets = [0, 4, 8, 11]  // fields start at 0, 4, 8; 11 is end marker
func NewByteRecord(data []byte, offsets []int) *ByteRecord {
	return &ByteRecord{
		data:    data,
		offsets: offsets,
	}
}

// NumFields returns the number of fields in the record.
func (r *ByteRecord) NumFields() int {
	if len(r.offsets) == 0 {
		return 0
	}
	// offsets array has N+1 elements for N fields (last is end marker)
	return len(r.offsets) - 1
}

// Field returns the i-th field as a string.
// This performs lazy string conversion - the string is only created when this method is called.
// Returns empty string if index is out of bounds.
func (r *ByteRecord) Field(i int) string {
	if i < 0 || i >= r.NumFields() {
		return ""
	}

	start := r.offsets[i]
	end := r.offsets[i+1]

	// Use unsafe string conversion to avoid allocation
	// This is safe because the underlying data is immutable
	return unsafeString(r.data[start:end])
}

// FieldBytes returns the i-th field as a []byte slice without allocation.
// The returned slice shares the underlying array with the ByteRecord.
// Returns nil if index is out of bounds.
//
// IMPORTANT: The returned slice must not be modified, as it shares memory
// with the original CSV data.
func (r *ByteRecord) FieldBytes(i int) []byte {
	if i < 0 || i >= r.NumFields() {
		return nil
	}

	start := r.offsets[i]
	end := r.offsets[i+1]

	return r.data[start:end]
}

// Fields returns all fields as strings.
// This is a convenience method that calls Field() for each field.
func (r *ByteRecord) Fields() []string {
	fields := make([]string, r.NumFields())
	for i := range fields {
		fields[i] = r.Field(i)
	}
	return fields
}

// ParseByteRecords parses CSV data into ByteRecords with offset tracking.
// This is more memory-efficient than Parse() because it doesn't create
// string copies for each field. Use this when you want to:
//   - Access fields as []byte without allocation (FieldBytes)
//   - Defer string conversion until needed (Field)
//   - Minimize memory usage for large datasets
//
// The parser follows RFC 4180:
//   - Fields are separated by commas
//   - Records are separated by newlines (LF or CRLF)
//   - Fields may be quoted with double quotes
//   - Quoted fields may contain commas, newlines, and escaped quotes ("")
//   - Empty lines are skipped
//
// Returns a slice of ByteRecords.
func ParseByteRecords(data []byte) ([]*ByteRecord, error) {
	if len(data) == 0 {
		return []*ByteRecord{}, nil
	}

	p := &byteRecordParser{
		data:   data,
		pos:    0,
		length: len(data),
	}

	return p.parse()
}

// byteRecordParser implements CSV parsing with offset tracking.
type byteRecordParser struct {
	data   []byte
	pos    int
	length int

	// Accumulator for building the current record's data
	recordData []byte

	// Accumulator for building the current record's offsets
	offsets []int
}

// parse parses the entire CSV file into ByteRecords.
func (p *byteRecordParser) parse() ([]*ByteRecord, error) {
	records := make([]*ByteRecord, 0, 16)

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
		if capacityHint == 0 && record.NumFields() > 0 {
			capacityHint = record.NumFields()
		}

		records = append(records, record)
	}

	return records, nil
}

// parseRecord parses a single CSV record with offset tracking.
func (p *byteRecordParser) parseRecord(capacityHint int) (*ByteRecord, error) {
	// Reset accumulators for this record
	if capacityHint > 0 {
		p.recordData = make([]byte, 0, capacityHint*10) // estimate 10 bytes per field
		p.offsets = make([]int, 0, capacityHint+1)
	} else {
		p.recordData = make([]byte, 0, 64)
		p.offsets = make([]int, 0, 8)
	}

	for {
		// Mark the start of this field
		fieldStart := len(p.recordData)
		p.offsets = append(p.offsets, fieldStart)

		// Parse the field and append to recordData
		err := p.parseField()
		if err != nil {
			return nil, err
		}

		// Check what comes next
		if p.pos >= p.length {
			// End of file - add final offset marker
			p.offsets = append(p.offsets, len(p.recordData))
			return NewByteRecord(p.recordData, p.offsets), nil
		}

		c := p.data[p.pos]
		if c == ',' {
			// More fields coming
			p.pos++
			continue
		}

		if c == '\r' || c == '\n' {
			// End of record - add final offset marker
			p.skipNewline()
			p.offsets = append(p.offsets, len(p.recordData))
			return NewByteRecord(p.recordData, p.offsets), nil
		}

		return nil, fmt.Errorf("unexpected character '%c' at position %d", c, p.pos)
	}
}

// parseField parses a single CSV field and appends its data to recordData.
func (p *byteRecordParser) parseField() error {
	if p.pos >= p.length {
		// Empty field at end of file
		return nil
	}

	// Check if field is quoted
	if p.data[p.pos] == '"' {
		return p.parseQuotedField()
	}

	return p.parseUnquotedField()
}

// parseQuotedField parses a quoted CSV field and appends to recordData.
func (p *byteRecordParser) parseQuotedField() error {
	// Skip opening quote
	p.pos++

	start := p.pos

	for p.pos < p.length {
		c := p.data[p.pos]

		if c == '"' {
			// Add everything up to this quote
			p.recordData = append(p.recordData, p.data[start:p.pos]...)
			p.pos++

			// Check if next character is also a quote (escaped quote)
			if p.pos < p.length && p.data[p.pos] == '"' {
				// Escaped quote - add single quote to buffer
				p.recordData = append(p.recordData, '"')
				p.pos++
				start = p.pos
				continue
			}

			// Closing quote - we're done
			return nil
		}

		p.pos++
	}

	return errors.New("unclosed quoted field")
}

// parseUnquotedField parses an unquoted CSV field and appends to recordData.
func (p *byteRecordParser) parseUnquotedField() error {
	start := p.pos

	for p.pos < p.length {
		c := p.data[p.pos]

		// Field terminators
		if c == ',' || c == '\r' || c == '\n' {
			break
		}

		// Quote in middle of unquoted field is an error
		if c == '"' {
			return fmt.Errorf("quote character in unquoted field at position %d", p.pos)
		}

		p.pos++
	}

	// Append field data
	p.recordData = append(p.recordData, p.data[start:p.pos]...)
	return nil
}

// isNewline checks if current position is at a newline.
func (p *byteRecordParser) isNewline() bool {
	if p.pos >= p.length {
		return false
	}
	c := p.data[p.pos]
	return c == '\r' || c == '\n'
}

// skipNewline skips a newline sequence (LF or CRLF).
func (p *byteRecordParser) skipNewline() {
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
