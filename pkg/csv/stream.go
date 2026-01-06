package csv

import (
	"io"

	"github.com/shapestone/shape-csv/internal/fastparser"
)

// Scanner provides a streaming interface for reading CSV records one at a time.
// This is memory-efficient for large CSV files as it processes records incrementally
// rather than loading the entire file into memory.
//
// Example usage:
//
//	file, _ := os.Open("data.csv")
//	defer file.Close()
//
//	scanner := csv.NewScanner(file).SetHasHeaders(true)
//	for scanner.Scan() {
//	    record := scanner.Record()
//	    name, _ := record.GetByName("name")
//	    fmt.Println(name)
//	}
//	if err := scanner.Err(); err != nil {
//	    // handle error
//	}
type Scanner struct {
	reader      io.Reader
	hasHeaders  bool
	reuseRecord bool
	headers     []string
	records     [][]string
	index       int
	err         error
	parsed      bool
	lastRecord  Record // reused when reuseRecord is true
}

// NewScanner creates a new Scanner that reads CSV from the given io.Reader.
// By default, the scanner assumes no headers. Use SetHasHeaders(true) to treat
// the first row as headers.
//
// Example:
//
//	scanner := csv.NewScanner(reader)
func NewScanner(reader io.Reader) *Scanner {
	return &Scanner{
		reader:     reader,
		hasHeaders: false,
		index:      -1,
		parsed:     false,
	}
}

// SetHasHeaders sets whether the first row should be treated as headers.
// If true, the first row will be used as column names for GetByName() access.
// Returns the Scanner for method chaining.
//
// Example:
//
//	scanner := csv.NewScanner(reader).SetHasHeaders(true)
func (s *Scanner) SetHasHeaders(hasHeaders bool) *Scanner {
	s.hasHeaders = hasHeaders
	return s
}

// SetReuseRecord sets whether the scanner should reuse the Record struct.
// When true, successive calls to Record() may return the same Record struct
// with updated field values. This can reduce memory allocations but means
// that previous Record values may be overwritten.
// Returns the Scanner for method chaining.
//
// Example:
//
//	scanner := csv.NewScanner(reader).SetReuseRecord(true)
func (s *Scanner) SetReuseRecord(reuse bool) *Scanner {
	s.reuseRecord = reuse
	return s
}

// Scan advances the scanner to the next record.
// It returns false when there are no more records or an error occurs.
// After Scan returns false, the Err method will return any error that occurred.
//
// Example:
//
//	for scanner.Scan() {
//	    record := scanner.Record()
//	    // process record
//	}
//	if err := scanner.Err(); err != nil {
//	    // handle error
//	}
func (s *Scanner) Scan() bool {
	// Parse on first call
	if !s.parsed {
		if err := s.parse(); err != nil {
			s.err = err
			return false
		}
		s.parsed = true
	}

	// Move to next record
	s.index++

	// Check if we have more records
	return s.index < len(s.records)
}

// Record returns the current record.
// This should only be called after Scan() returns true.
//
// The returned Record provides type-safe access to field values
// by index or by header name (if headers are set).
//
// When ReuseRecord is enabled, the returned Record may share memory with
// previous calls. Copy the Record if you need to retain its values.
func (s *Scanner) Record() Record {
	if s.index < 0 || s.index >= len(s.records) {
		return Record{fields: []string{}, headers: s.headers}
	}

	if s.reuseRecord {
		// Reuse the lastRecord struct, just update the fields
		s.lastRecord.fields = s.records[s.index]
		s.lastRecord.headers = s.headers
		return s.lastRecord
	}

	return Record{
		fields:  s.records[s.index],
		headers: s.headers,
	}
}

// Err returns the error, if any, that was encountered during scanning.
// It returns nil if no error occurred or at EOF.
func (s *Scanner) Err() error {
	return s.err
}

// Headers returns the column headers if SetHasHeaders(true) was called.
// Returns an empty slice if no headers were set.
// This is available after the first call to Scan().
func (s *Scanner) Headers() []string {
	return s.headers
}

// parse reads and parses the entire CSV file.
// In a true streaming implementation, this would parse incrementally,
// but for now we use the fast parser to read all records.
func (s *Scanner) parse() error {
	// Read all data from reader
	data, err := io.ReadAll(s.reader)
	if err != nil {
		return err
	}

	// Parse CSV
	allRecords, err := fastparser.Parse(data)
	if err != nil {
		return err
	}

	// Handle headers
	if s.hasHeaders && len(allRecords) > 0 {
		s.headers = allRecords[0]
		s.records = allRecords[1:]
	} else {
		s.headers = []string{}
		s.records = allRecords
	}

	return nil
}
