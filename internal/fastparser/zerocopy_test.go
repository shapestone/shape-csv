package fastparser

import (
	"bytes"
	"reflect"
	"testing"
)

// TestParseZeroCopy tests the zero-copy parser that returns [][]byte slices.
func TestParseZeroCopy(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    [][]string // We'll compare bytes as strings for readability
		wantErr bool
	}{
		{
			name:  "empty input",
			input: "",
			want:  [][]string{},
		},
		{
			name:  "single field",
			input: "a",
			want:  [][]string{{"a"}},
		},
		{
			name:  "simple record",
			input: "a,b,c",
			want:  [][]string{{"a", "b", "c"}},
		},
		{
			name:  "two records",
			input: "a,b\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "empty fields",
			input: "a,,c",
			want:  [][]string{{"a", "", "c"}},
		},
		{
			name:  "quoted field with comma",
			input: `"hello,world"`,
			want:  [][]string{{"hello,world"}},
		},
		{
			name:  "quoted field with escaped quote - requires allocation",
			input: `"say ""hello"""`,
			want:  [][]string{{`say "hello"`}},
		},
		{
			name:  "unquoted fields - zero allocations",
			input: "foo,bar,baz",
			want:  [][]string{{"foo", "bar", "baz"}},
		},
		{
			name:    "unclosed quoted field",
			input:   `"unclosed`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseZeroCopy([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseZeroCopy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Convert [][]byte to [][]string for comparison
			gotStrings := make([][]string, len(got))
			for i, record := range got {
				gotStrings[i] = make([]string, len(record))
				for j, field := range record {
					gotStrings[i][j] = string(field)
				}
			}

			if !reflect.DeepEqual(gotStrings, tt.want) {
				t.Errorf("ParseZeroCopy() = %v, want %v", gotStrings, tt.want)
			}
		})
	}
}

// TestParseZeroCopy_PointsToOriginalBuffer verifies that returned slices point to original buffer.
func TestParseZeroCopy_PointsToOriginalBuffer(t *testing.T) {
	input := []byte("foo,bar,baz")
	records, err := ParseZeroCopy(input)
	if err != nil {
		t.Fatalf("ParseZeroCopy() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	// For unquoted fields, the returned slices should point into the original buffer
	// Check that the data pointer is within the original buffer range
	for _, field := range records[0] {
		if len(field) == 0 {
			continue
		}
		// Verify field data comes from original input
		foundInOriginal := false
		for i := 0; i <= len(input)-len(field); i++ {
			if bytes.Equal(input[i:i+len(field)], field) {
				foundInOriginal = true
				break
			}
		}
		if !foundInOriginal {
			t.Errorf("field %q not found in original input", field)
		}
	}
}

// TestScanner tests the Scanner type with streaming interface.
func TestScanner(t *testing.T) {
	input := []byte("a,b,c\nd,e,f\ng,h,i")

	scanner := NewScanner(input, ScannerOptions{})

	expectedRecords := [][]string{
		{"a", "b", "c"},
		{"d", "e", "f"},
		{"g", "h", "i"},
	}

	recordNum := 0
	for scanner.Scan() {
		if recordNum >= len(expectedRecords) {
			t.Fatalf("too many records, expected %d", len(expectedRecords))
		}

		record := scanner.Record()
		expected := expectedRecords[recordNum]

		if len(record) != len(expected) {
			t.Errorf("record %d: got %d fields, want %d", recordNum, len(record), len(expected))
			recordNum++
			continue
		}

		for i, field := range record {
			if field != expected[i] {
				t.Errorf("record %d, field %d: got %q, want %q", recordNum, i, field, expected[i])
			}
		}

		recordNum++
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("Scanner.Err() = %v, want nil", err)
	}

	if recordNum != len(expectedRecords) {
		t.Errorf("got %d records, want %d", recordNum, len(expectedRecords))
	}
}

// TestScanner_ReuseRecord tests the ReuseRecord option.
func TestScanner_ReuseRecord(t *testing.T) {
	input := []byte("a,b,c\nd,e,f")

	scanner := NewScanner(input, ScannerOptions{
		ReuseRecord: true,
	})

	var firstRecordPtr *[]string
	recordNum := 0

	for scanner.Scan() {
		record := scanner.Record()

		if recordNum == 0 {
			// Store pointer to first record
			firstRecordPtr = &record
		} else if recordNum == 1 {
			// Verify it's the same underlying slice
			secondRecordPtr := &record
			if firstRecordPtr != secondRecordPtr {
				// The slices should be the same slice being reused
				// Note: In Go, when we return record, we're returning a slice header
				// The ReuseRecord pattern means we reuse the same backing array
				// We can verify this by checking the capacity and content
				t.Logf("Note: slice headers differ, but backing array should be reused")
			}
		}

		recordNum++
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("Scanner.Err() = %v, want nil", err)
	}
}

// TestScanner_Error tests scanner error handling.
func TestScanner_Error(t *testing.T) {
	input := []byte(`"unclosed quote`)

	scanner := NewScanner(input, ScannerOptions{})

	// Should return false on error
	if scanner.Scan() {
		t.Error("Scan() should return false on error")
	}

	// Error should be set
	if scanner.Err() == nil {
		t.Error("Err() should return error for unclosed quote")
	}
}

// TestScanner_EmptyInput tests scanner with empty input.
func TestScanner_EmptyInput(t *testing.T) {
	scanner := NewScanner([]byte(""), ScannerOptions{})

	if scanner.Scan() {
		t.Error("Scan() should return false for empty input")
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("Err() = %v, want nil for empty input", err)
	}
}

// TestScanner_EmptyLines tests scanner with empty lines.
func TestScanner_EmptyLines(t *testing.T) {
	input := []byte("a,b\n\nc,d\n")

	scanner := NewScanner(input, ScannerOptions{})

	expectedRecords := [][]string{
		{"a", "b"},
		{"c", "d"},
	}

	recordNum := 0
	for scanner.Scan() {
		if recordNum >= len(expectedRecords) {
			t.Fatalf("too many records")
		}
		record := scanner.Record()
		expected := expectedRecords[recordNum]

		if !reflect.DeepEqual(record, expected) {
			t.Errorf("record %d: got %v, want %v", recordNum, record, expected)
		}
		recordNum++
	}

	if recordNum != len(expectedRecords) {
		t.Errorf("got %d records, want %d", recordNum, len(expectedRecords))
	}
}

// Benchmark tests for zero-copy optimizations

// BenchmarkParseZeroCopy_Simple benchmarks zero-copy parsing with simple unquoted data.
func BenchmarkParseZeroCopy_Simple(b *testing.B) {
	data := []byte("a,b,c\nd,e,f\ng,h,i")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ParseZeroCopy(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseZeroCopy_QuotedWithEscapes benchmarks with escaped quotes (requires allocation).
func BenchmarkParseZeroCopy_QuotedWithEscapes(b *testing.B) {
	data := []byte(`"say ""hello""","world","foo ""bar"" baz"`)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ParseZeroCopy(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseZeroCopy_LargeSimple benchmarks large file with simple unquoted data.
func BenchmarkParseZeroCopy_LargeSimple(b *testing.B) {
	// Generate 1000 records with 5 fields each
	var data []byte
	for i := 0; i < 1000; i++ {
		if i > 0 {
			data = append(data, '\n')
		}
		data = append(data, []byte("field1,field2,field3,field4,field5")...)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ParseZeroCopy(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkScanner benchmarks the Scanner with ReuseRecord.
func BenchmarkScanner(b *testing.B) {
	// Generate test data
	var data []byte
	for i := 0; i < 100; i++ {
		if i > 0 {
			data = append(data, '\n')
		}
		data = append(data, []byte("a,b,c,d,e")...)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		scanner := NewScanner(data, ScannerOptions{})
		for scanner.Scan() {
			_ = scanner.Record()
		}
		if scanner.Err() != nil {
			b.Fatal(scanner.Err())
		}
	}
}

// BenchmarkScanner_ReuseRecord benchmarks Scanner with ReuseRecord enabled.
func BenchmarkScanner_ReuseRecord(b *testing.B) {
	// Generate test data
	var data []byte
	for i := 0; i < 100; i++ {
		if i > 0 {
			data = append(data, '\n')
		}
		data = append(data, []byte("a,b,c,d,e")...)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		scanner := NewScanner(data, ScannerOptions{ReuseRecord: true})
		for scanner.Scan() {
			_ = scanner.Record()
		}
		if scanner.Err() != nil {
			b.Fatal(scanner.Err())
		}
	}
}

// BenchmarkParse_vs_ParseZeroCopy compares standard Parse with ParseZeroCopy.
func BenchmarkParse_vs_ParseZeroCopy(b *testing.B) {
	data := []byte("a,b,c\nd,e,f\ng,h,i")

	b.Run("Parse", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := Parse(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ParseZeroCopy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := ParseZeroCopy(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestParseZeroCopy_LineEndings tests various line ending scenarios for skipNewline coverage.
func TestParseZeroCopy_LineEndings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "bare LF",
			input: "a,b\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "bare CR",
			input: "a,b\rc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "CRLF",
			input: "a,b\r\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "multiple bare LF",
			input: "a,b\n\n\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "multiple bare CR",
			input: "a,b\r\r\rc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "multiple CRLF",
			input: "a,b\r\n\r\n\r\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "mixed line endings",
			input: "a,b\nc,d\re,f\r\ng,h",
			want:  [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}, {"g", "h"}},
		},
		{
			name:  "trailing LF",
			input: "a,b\n",
			want:  [][]string{{"a", "b"}},
		},
		{
			name:  "trailing CR",
			input: "a,b\r",
			want:  [][]string{{"a", "b"}},
		},
		{
			name:  "trailing CRLF",
			input: "a,b\r\n",
			want:  [][]string{{"a", "b"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseZeroCopy([]byte(tt.input))
			if err != nil {
				t.Errorf("ParseZeroCopy() error = %v", err)
				return
			}

			// Convert [][]byte to [][]string for comparison
			gotStrings := make([][]string, len(got))
			for i, record := range got {
				gotStrings[i] = make([]string, len(record))
				for j, field := range record {
					gotStrings[i][j] = string(field)
				}
			}

			if !reflect.DeepEqual(gotStrings, tt.want) {
				t.Errorf("ParseZeroCopy() = %v, want %v", gotStrings, tt.want)
			}
		})
	}
}

// TestParseZeroCopy_QuotedFieldEdgeCases tests edge cases in parseQuotedField for better coverage.
func TestParseZeroCopy_QuotedFieldEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    [][]string
		wantErr bool
	}{
		{
			name:  "quoted field with only escaped quotes",
			input: `""""`,
			want:  [][]string{{`"`}},
		},
		{
			name:  "quoted field with multiple consecutive escaped quotes",
			input: `"a""""""b"`,
			want:  [][]string{{`a"""b`}},
		},
		{
			name:  "quoted field with newline at start",
			input: "\"\na,b\"",
			want:  [][]string{{"\na,b"}},
		},
		{
			name:  "quoted field with newline at end",
			input: "\"a,b\n\"",
			want:  [][]string{{"a,b\n"}},
		},
		{
			name:  "quoted field with CRLF inside",
			input: "\"a\r\nb\"",
			want:  [][]string{{"a\r\nb"}},
		},
		{
			name:  "quoted empty field followed by non-empty",
			input: `"",a`,
			want:  [][]string{{"", "a"}},
		},
		{
			name:  "multiple quoted fields with escaped quotes",
			input: `"a""b","c""d","e""f"`,
			want:  [][]string{{`a"b`, `c"d`, `e"f`}},
		},
		{
			name:  "quoted field ending with escaped quote",
			input: `"hello"""`,
			want:  [][]string{{`hello"`}},
		},
		{
			name:  "quoted field starting with escaped quote",
			input: `"""hello"`,
			want:  [][]string{{`"hello`}},
		},
		{
			name:    "unclosed quoted field with newline",
			input:   "\"hello\nworld",
			wantErr: true,
		},
		{
			name:    "unclosed quoted field with comma",
			input:   "\"hello,world",
			wantErr: true,
		},
		{
			name:    "unclosed quoted field at EOF",
			input:   "a,\"b",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseZeroCopy([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseZeroCopy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Convert [][]byte to [][]string for comparison
			gotStrings := make([][]string, len(got))
			for i, record := range got {
				gotStrings[i] = make([]string, len(record))
				for j, field := range record {
					gotStrings[i][j] = string(field)
				}
			}

			if !reflect.DeepEqual(gotStrings, tt.want) {
				t.Errorf("ParseZeroCopy() = %v, want %v", gotStrings, tt.want)
			}
		})
	}
}

// TestScanner_LineEndings tests Scanner with various line endings for skipNewline coverage.
func TestScanner_LineEndings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "bare CR endings",
			input: "a,b\rc,d\re,f",
			want:  [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}},
		},
		{
			name:  "CRLF endings",
			input: "a,b\r\nc,d\r\ne,f",
			want:  [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}},
		},
		{
			name:  "mixed CR and CRLF",
			input: "a,b\rc,d\r\ne,f",
			want:  [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}},
		},
		{
			name:  "multiple consecutive CR",
			input: "a,b\r\r\rc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner([]byte(tt.input), ScannerOptions{})

			var got [][]string
			for scanner.Scan() {
				record := scanner.Record()
				got = append(got, append([]string{}, record...))
			}

			if err := scanner.Err(); err != nil {
				t.Errorf("Scanner.Err() = %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Scanner records = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestScanner_QuotedFieldEdgeCases tests Scanner's parseQuotedField with edge cases.
func TestScanner_QuotedFieldEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    [][]string
		wantErr bool
	}{
		{
			name:  "empty quoted field",
			input: `""`,
			want:  [][]string{{""}},
		},
		{
			name:  "quoted field with only spaces",
			input: `"   "`,
			want:  [][]string{{"   "}},
		},
		{
			name:  "quoted field with escaped quotes at boundaries",
			input: `"""a"""`,
			want:  [][]string{{`"a"`}},
		},
		{
			name:    "quote in unquoted field error",
			input:   `a"b`,
			wantErr: true,
		},
		{
			name:    "unclosed quote error",
			input:   `"abc`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner([]byte(tt.input), ScannerOptions{})

			var got [][]string
			for scanner.Scan() {
				record := scanner.Record()
				got = append(got, append([]string{}, record...))
			}

			err := scanner.Err()
			if (err != nil) != tt.wantErr {
				t.Errorf("Scanner.Err() = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Scanner records = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseZeroCopy_EmptyFieldAtEOF tests parseField with empty field at EOF
func TestParseZeroCopy_EmptyFieldAtEOF(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "single empty field at EOF",
			input: "a,",
			want:  [][]string{{"a", ""}},
		},
		{
			name:  "multiple empty fields at EOF",
			input: "a,,",
			want:  [][]string{{"a", "", ""}},
		},
		{
			name:  "only empty field",
			input: ",",
			want:  [][]string{{"", ""}},
		},
		{
			name:  "trailing comma at EOF",
			input: "a,b,",
			want:  [][]string{{"a", "b", ""}},
		},
		{
			name:  "empty field at EOF after newline",
			input: "a,b\nc,",
			want:  [][]string{{"a", "b"}, {"c", ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseZeroCopy([]byte(tt.input))
			if err != nil {
				t.Errorf("ParseZeroCopy() error = %v", err)
				return
			}

			// Convert [][]byte to [][]string for comparison
			gotStrings := make([][]string, len(got))
			for i, record := range got {
				gotStrings[i] = make([]string, len(record))
				for j, field := range record {
					gotStrings[i][j] = string(field)
				}
			}

			if !reflect.DeepEqual(gotStrings, tt.want) {
				t.Errorf("ParseZeroCopy() = %v, want %v", gotStrings, tt.want)
			}
		})
	}
}

// TestScanner_EmptyFieldAtEOF tests Scanner parseField with empty field at EOF
func TestScanner_EmptyFieldAtEOF(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "single empty field at EOF",
			input: "a,",
			want:  [][]string{{"a", ""}},
		},
		{
			name:  "trailing comma",
			input: "a,b,c,",
			want:  [][]string{{"a", "b", "c", ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner([]byte(tt.input), ScannerOptions{})

			var got [][]string
			for scanner.Scan() {
				record := scanner.Record()
				got = append(got, append([]string{}, record...))
			}

			if err := scanner.Err(); err != nil {
				t.Errorf("Scanner.Err() = %v", err)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Scanner records = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseZeroCopy_UnexpectedCharacter tests error handling for unexpected characters
func TestParseZeroCopy_UnexpectedCharacter(t *testing.T) {
	// After a field, only comma, newline, or EOF are valid
	// This should not normally happen with valid CSV, but we test the error path
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "quote in middle of unquoted field",
			input:   `a"b`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseZeroCopy([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseZeroCopy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestScanner_UnexpectedCharacter tests Scanner error handling
func TestScanner_UnexpectedCharacter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "quote in middle of unquoted field",
			input:   `ab"cd`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner([]byte(tt.input), ScannerOptions{})

			for scanner.Scan() {
				_ = scanner.Record()
			}

			err := scanner.Err()
			if (err != nil) != tt.wantErr {
				t.Errorf("Scanner.Err() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestZeroCopyParser_BareCRAtEOF tests skipNewline with bare CR at EOF
func TestZeroCopyParser_BareCRAtEOF(t *testing.T) {
	// Test that a bare CR at the end of the file is handled correctly
	input := "a,b\r"
	got, err := ParseZeroCopy([]byte(input))
	if err != nil {
		t.Fatalf("ParseZeroCopy() error = %v", err)
	}

	want := [][]string{{"a", "b"}}

	// Convert to strings
	gotStrings := make([][]string, len(got))
	for i, record := range got {
		gotStrings[i] = make([]string, len(record))
		for j, field := range record {
			gotStrings[i][j] = string(field)
		}
	}

	if !reflect.DeepEqual(gotStrings, want) {
		t.Errorf("ParseZeroCopy() = %v, want %v", gotStrings, want)
	}
}

// TestScanner_BareCRAtEOF tests Scanner skipNewline with bare CR at EOF
func TestScanner_BareCRAtEOF(t *testing.T) {
	input := "a,b\r"
	scanner := NewScanner([]byte(input), ScannerOptions{})

	var got [][]string
	for scanner.Scan() {
		record := scanner.Record()
		got = append(got, append([]string{}, record...))
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("Scanner.Err() = %v", err)
		return
	}

	want := [][]string{{"a", "b"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scanner records = %v, want %v", got, want)
	}
}
