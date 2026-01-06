package csv

import (
	"strings"
	"testing"
)

// TestNewScanner tests creating a new Scanner
func TestNewScanner(t *testing.T) {
	csvData := "name,age\nAlice,30\nBob,25\n"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader)
	if scanner == nil {
		t.Fatal("NewScanner() returned nil")
	}
}

// TestStreamRecords tests streaming CSV records one at a time
func TestStreamRecords(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		hasHeaders bool
		want       []Record
		wantErr    bool
	}{
		{
			name:       "simple CSV with headers",
			input:      "name,age\nAlice,30\nBob,25\n",
			hasHeaders: true,
			want: []Record{
				{fields: []string{"Alice", "30"}, headers: []string{"name", "age"}},
				{fields: []string{"Bob", "25"}, headers: []string{"name", "age"}},
			},
		},
		{
			name:       "CSV without headers",
			input:      "Alice,30\nBob,25\n",
			hasHeaders: false,
			want: []Record{
				{fields: []string{"Alice", "30"}},
				{fields: []string{"Bob", "25"}},
			},
		},
		{
			name:       "empty CSV",
			input:      "",
			hasHeaders: false,
			want:       []Record{},
		},
		{
			name:       "single record with headers",
			input:      "name,age\nAlice,30\n",
			hasHeaders: true,
			want: []Record{
				{fields: []string{"Alice", "30"}, headers: []string{"name", "age"}},
			},
		},
		{
			name:       "CSV with empty fields",
			input:      "a,b,c\n1,,3\n,,\n",
			hasHeaders: true,
			want: []Record{
				{fields: []string{"1", "", "3"}, headers: []string{"a", "b", "c"}},
				{fields: []string{"", "", ""}, headers: []string{"a", "b", "c"}},
			},
		},
		{
			name:       "CSV with quoted fields",
			input:      "name,description\nItem1,\"Has, comma\"\nItem2,\"Has \"\"quotes\"\"\"\n",
			hasHeaders: true,
			want: []Record{
				{fields: []string{"Item1", "Has, comma"}, headers: []string{"name", "description"}},
				{fields: []string{"Item2", "Has \"quotes\""}, headers: []string{"name", "description"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			scanner := NewScanner(reader).SetHasHeaders(tt.hasHeaders)

			var got []Record
			for scanner.Scan() {
				record := scanner.Record()
				got = append(got, record)
			}

			if err := scanner.Err(); (err != nil) != tt.wantErr {
				t.Errorf("Scanner.Err() = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("Scanner got %d records, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if len(got[i].fields) != len(tt.want[i].fields) {
					t.Errorf("Record %d has %d fields, want %d", i, len(got[i].fields), len(tt.want[i].fields))
					continue
				}
				for j := range got[i].fields {
					if got[i].fields[j] != tt.want[i].fields[j] {
						t.Errorf("Record %d field %d = %q, want %q", i, j, got[i].fields[j], tt.want[i].fields[j])
					}
				}

				// Check headers
				if len(got[i].headers) != len(tt.want[i].headers) {
					t.Errorf("Record %d has %d headers, want %d", i, len(got[i].headers), len(tt.want[i].headers))
					continue
				}
				for j := range got[i].headers {
					if got[i].headers[j] != tt.want[i].headers[j] {
						t.Errorf("Record %d header %d = %q, want %q", i, j, got[i].headers[j], tt.want[i].headers[j])
					}
				}
			}
		})
	}
}

// TestScannerHeaders tests accessing headers
func TestScannerHeaders(t *testing.T) {
	csvData := "name,age,city\nAlice,30,NYC\nBob,25,LA\n"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader).SetHasHeaders(true)

	// Scan first record
	if !scanner.Scan() {
		t.Fatal("Scanner.Scan() returned false for first record")
	}

	headers := scanner.Headers()
	want := []string{"name", "age", "city"}

	if len(headers) != len(want) {
		t.Fatalf("Scanner.Headers() returned %d headers, want %d", len(headers), len(want))
	}

	for i := range headers {
		if headers[i] != want[i] {
			t.Errorf("Header %d = %q, want %q", i, headers[i], want[i])
		}
	}
}

// TestScannerGetByName tests accessing fields by header name
func TestScannerGetByName(t *testing.T) {
	csvData := "name,age,city\nAlice,30,NYC\nBob,25,LA\n"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader).SetHasHeaders(true)

	// Scan first record
	if !scanner.Scan() {
		t.Fatal("Scanner.Scan() returned false for first record")
	}

	record := scanner.Record()

	// Test getting fields by name
	tests := []struct {
		name  string
		want  string
		found bool
	}{
		{"name", "Alice", true},
		{"age", "30", true},
		{"city", "NYC", true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := record.GetByName(tt.name)
			if ok != tt.found {
				t.Errorf("GetByName(%q) found = %v, want %v", tt.name, ok, tt.found)
			}
			if ok && got != tt.want {
				t.Errorf("GetByName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestScannerNoHeaders tests Scanner without headers
func TestScannerNoHeaders(t *testing.T) {
	csvData := "Alice,30,NYC\nBob,25,LA\n"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader).SetHasHeaders(false)

	var count int
	for scanner.Scan() {
		count++
		record := scanner.Record()

		// Headers should be empty
		if len(record.headers) != 0 {
			t.Errorf("Record.headers should be empty, got %v", record.headers)
		}

		// GetByName should always fail
		if _, ok := record.GetByName("name"); ok {
			t.Error("GetByName should fail when no headers are set")
		}
	}

	if count != 2 {
		t.Errorf("Scanner counted %d records, want 2", count)
	}
}

// TestScannerError tests Scanner error handling
func TestScannerError(t *testing.T) {
	// Create a CSV with unclosed quote (parse error)
	csvData := "name,age\nAlice,\"30\nBob,25"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader).SetHasHeaders(true)

	// Scan should fail due to parse error
	for scanner.Scan() {
		// Keep scanning
	}

	if err := scanner.Err(); err == nil {
		t.Error("Scanner.Err() should return error for invalid CSV")
	}
}

// TestScannerEOF tests Scanner behavior at EOF
func TestScannerEOF(t *testing.T) {
	csvData := "name,age\nAlice,30\n"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader).SetHasHeaders(true)

	// Scan single record
	if !scanner.Scan() {
		t.Fatal("Scanner.Scan() returned false for first record")
	}

	// Next scan should return false (EOF)
	if scanner.Scan() {
		t.Error("Scanner.Scan() should return false at EOF")
	}

	// Error should be nil at EOF
	if err := scanner.Err(); err != nil {
		t.Errorf("Scanner.Err() = %v at EOF, want nil", err)
	}
}

// TestScannerReuse tests that Scanner cannot be reused after error
func TestScannerReuse(t *testing.T) {
	csvData := "name,age\nAlice,30\nBob,25\n"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader).SetHasHeaders(true)

	// Scan all records
	count := 0
	for scanner.Scan() {
		count++
	}

	if count != 2 {
		t.Errorf("Scanner counted %d records, want 2", count)
	}

	// Try to scan again (should return false)
	if scanner.Scan() {
		t.Error("Scanner.Scan() should return false after EOF")
	}
}

// TestScannerLargeFile tests Scanner with a large CSV file
func TestScannerLargeFile(t *testing.T) {
	// Create a large CSV in memory
	var sb strings.Builder
	sb.WriteString("id,name,value\n")
	for i := 0; i < 1000; i++ {
		sb.WriteString("1,test,value\n")
	}

	reader := strings.NewReader(sb.String())
	scanner := NewScanner(reader).SetHasHeaders(true)

	count := 0
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Scanner.Err() = %v", err)
	}

	if count != 1000 {
		t.Errorf("Scanner counted %d records, want 1000", count)
	}
}

// TestScannerSetReuseRecord tests the SetReuseRecord option
func TestScannerSetReuseRecord(t *testing.T) {
	csvData := "name,age\nAlice,30\nBob,25\nCarol,35\n"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader).SetHasHeaders(true).SetReuseRecord(true)

	// Scan multiple records
	var records []Record
	for scanner.Scan() {
		record := scanner.Record()
		// Store a copy since records may be reused
		recordCopy := Record{
			fields:  append([]string{}, record.fields...),
			headers: record.headers,
		}
		records = append(records, recordCopy)
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Scanner.Err() = %v", err)
	}

	// Verify we got all records
	if len(records) != 3 {
		t.Errorf("Got %d records, want 3", len(records))
	}

	// Verify each record has correct data
	expectedNames := []string{"Alice", "Bob", "Carol"}
	for i, name := range expectedNames {
		if val, _ := records[i].Get(0); val != name {
			t.Errorf("Record %d name = %s, want %s", i, val, name)
		}
	}
}

// TestScannerRecordBeforeScan tests calling Record() before Scan()
func TestScannerRecordBeforeScan(t *testing.T) {
	csvData := "name,age\nAlice,30\n"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader).SetHasHeaders(true)

	// Call Record() before any Scan()
	record := scanner.Record()
	if record.Len() != 0 {
		t.Errorf("Record() before Scan() should return empty record, got len=%d", record.Len())
	}
}

// TestScannerRecordOutOfBounds tests calling Record() when index is out of bounds
func TestScannerRecordOutOfBounds(t *testing.T) {
	csvData := "name,age\nAlice,30\n"
	reader := strings.NewReader(csvData)

	scanner := NewScanner(reader).SetHasHeaders(true)

	// Scan the single record
	if !scanner.Scan() {
		t.Fatal("Scanner.Scan() returned false")
	}

	_ = scanner.Record()

	// Try to scan beyond EOF
	if scanner.Scan() {
		t.Error("Scanner.Scan() should return false at EOF")
	}

	// Record() should return empty record
	record := scanner.Record()
	if record.Len() != 0 {
		t.Errorf("Record() after EOF should return empty record, got len=%d", record.Len())
	}
}

// BenchmarkScanner benchmarks the Scanner
func BenchmarkScanner(b *testing.B) {
	// Create CSV data
	var sb strings.Builder
	sb.WriteString("col1,col2,col3,col4,col5\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("value1,value2,value3,value4,value5\n")
	}
	csvData := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(csvData)
		scanner := NewScanner(reader).SetHasHeaders(true)

		for scanner.Scan() {
			_ = scanner.Record()
		}

		if err := scanner.Err(); err != nil {
			b.Fatalf("Scanner.Err() = %v", err)
		}
	}
}
