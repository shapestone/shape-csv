package fastparser

import (
	"reflect"
	"strings"
	"testing"
)

func TestChunkedParser_BasicParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    [][]string
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
			name:  "two records with CRLF",
			input: "a,b\r\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "empty fields",
			input: "a,,c",
			want:  [][]string{{"a", "", "c"}},
		},
		{
			name:  "all empty fields",
			input: ",,",
			want:  [][]string{{"", "", ""}},
		},
		{
			name:  "quoted field with comma",
			input: `"hello,world"`,
			want:  [][]string{{"hello,world"}},
		},
		{
			name:  "quoted field with escaped quote",
			input: `"say ""hello"""`,
			want:  [][]string{{`say "hello"`}},
		},
		{
			name:  "quoted field with newline",
			input: "\"hello\nworld\"",
			want:  [][]string{{"hello\nworld"}},
		},
		{
			name:  "quoted field with CRLF",
			input: "\"hello\r\nworld\"",
			want:  [][]string{{"hello\r\nworld"}},
		},
		{
			name:  "mixed quoted and unquoted",
			input: `a,"b,c",d`,
			want:  [][]string{{"a", "b,c", "d"}},
		},
		{
			name:  "trailing newline",
			input: "a,b\n",
			want:  [][]string{{"a", "b"}},
		},
		{
			name:  "empty lines skipped",
			input: "a,b\n\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseChunked([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseChunked() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseChunked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChunkedParser_ChunkBoundaries(t *testing.T) {
	// Test cases specifically designed to test chunk boundary handling
	tests := []struct {
		name      string
		input     string
		chunkSize int
		want      [][]string
	}{
		{
			name:      "field split across chunk boundary",
			input:     "abcdefghijklmnop,xyz",
			chunkSize: 8,
			want:      [][]string{{"abcdefghijklmnop", "xyz"}},
		},
		{
			name:      "delimiter at chunk boundary",
			input:     "abcdefgh,ijklmnop",
			chunkSize: 8,
			want:      [][]string{{"abcdefgh", "ijklmnop"}},
		},
		{
			name:      "quoted field split across chunk boundary",
			input:     `"abcdefghijklmnop",xyz`,
			chunkSize: 8,
			want:      [][]string{{"abcdefghijklmnop", "xyz"}},
		},
		{
			name:      "escaped quote at chunk boundary",
			input:     `"abc""defgh",xyz`,
			chunkSize: 8,
			want:      [][]string{{`abc"defgh`, "xyz"}},
		},
		{
			name:      "newline at chunk boundary",
			input:     "abcdefgh\nijklmnop",
			chunkSize: 8,
			want:      [][]string{{"abcdefgh"}, {"ijklmnop"}},
		},
		{
			name:      "CRLF split across chunk boundary",
			input:     "abcdefg\r\nijklmnop",
			chunkSize: 8,
			want:      [][]string{{"abcdefg"}, {"ijklmnop"}},
		},
		{
			name:      "multiple records with small chunks",
			input:     "a,b,c\nd,e,f\ng,h,i",
			chunkSize: 4,
			want:      [][]string{{"a", "b", "c"}, {"d", "e", "f"}, {"g", "h", "i"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a chunked parser with specific chunk size
			p := &chunkedParser{
				data:      []byte(tt.input),
				pos:       0,
				length:    len(tt.input),
				chunkSize: tt.chunkSize,
			}
			got, err := p.parse()
			if err != nil {
				t.Errorf("parse() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChunkedParser_LargeFiles(t *testing.T) {
	// Test with large input to ensure chunking works correctly
	tests := []struct {
		name      string
		numRows   int
		numCols   int
		wantRows  int
		wantCols  int
		chunkSize int
	}{
		{
			name:      "1000 rows x 10 cols",
			numRows:   1000,
			numCols:   10,
			wantRows:  1000,
			wantCols:  10,
			chunkSize: 4096,
		},
		{
			name:      "100 rows x 100 cols",
			numRows:   100,
			numCols:   100,
			wantRows:  100,
			wantCols:  100,
			chunkSize: 8192,
		},
		{
			name:      "10000 rows x 5 cols with small chunks",
			numRows:   10000,
			numCols:   5,
			wantRows:  10000,
			wantCols:  5,
			chunkSize: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate large CSV data
			var sb strings.Builder
			for i := 0; i < tt.numRows; i++ {
				for j := 0; j < tt.numCols; j++ {
					if j > 0 {
						sb.WriteByte(',')
					}
					sb.WriteString("field")
				}
				sb.WriteByte('\n')
			}

			input := []byte(sb.String())
			p := &chunkedParser{
				data:      input,
				pos:       0,
				length:    len(input),
				chunkSize: tt.chunkSize,
			}

			got, err := p.parse()
			if err != nil {
				t.Errorf("parse() error = %v", err)
				return
			}

			if len(got) != tt.wantRows {
				t.Errorf("got %d rows, want %d rows", len(got), tt.wantRows)
			}

			for i, row := range got {
				if len(row) != tt.wantCols {
					t.Errorf("row %d: got %d cols, want %d cols", i, len(row), tt.wantCols)
					break
				}
			}
		})
	}
}

func TestChunkedParser_QuotedFieldsAcrossBoundaries(t *testing.T) {
	// Test specifically for quoted fields that span multiple chunks
	tests := []struct {
		name      string
		input     string
		chunkSize int
		want      [][]string
	}{
		{
			name:      "long quoted field with commas",
			input:     `"this,is,a,very,long,field,that,spans,multiple,chunks",next`,
			chunkSize: 8,
			want:      [][]string{{"this,is,a,very,long,field,that,spans,multiple,chunks", "next"}},
		},
		{
			name:      "quoted field with newlines across chunks",
			input:     "\"line1\nline2\nline3\nline4\",next",
			chunkSize: 8,
			want:      [][]string{{"line1\nline2\nline3\nline4", "next"}},
		},
		{
			name:      "multiple escaped quotes across chunks",
			input:     `"a""b""c""d""e""f""g""h",next`,
			chunkSize: 8,
			want:      [][]string{{`a"b"c"d"e"f"g"h`, "next"}},
		},
		{
			name:      "quoted field ending at chunk boundary",
			input:     `"abcdefg",next`,
			chunkSize: 8,
			want:      [][]string{{"abcdefg", "next"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &chunkedParser{
				data:      []byte(tt.input),
				pos:       0,
				length:    len(tt.input),
				chunkSize: tt.chunkSize,
			}
			got, err := p.parse()
			if err != nil {
				t.Errorf("parse() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSWAR_HasDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		delimiter byte
		want      bool
	}{
		{
			name:      "comma present",
			data:      []byte("abc,defg"),
			delimiter: ',',
			want:      true,
		},
		{
			name:      "comma not present",
			data:      []byte("abcdefgh"),
			delimiter: ',',
			want:      false,
		},
		{
			name:      "LF present",
			data:      []byte("abcd\nefg"),
			delimiter: '\n',
			want:      true,
		},
		{
			name:      "LF not present",
			data:      []byte("abcdefgh"),
			delimiter: '\n',
			want:      false,
		},
		{
			name:      "quote present",
			data:      []byte("abc\"defg"),
			delimiter: '"',
			want:      true,
		},
		{
			name:      "quote not present",
			data:      []byte("abcdefgh"),
			delimiter: '"',
			want:      false,
		},
		{
			name:      "delimiter at start",
			data:      []byte(",abcdefg"),
			delimiter: ',',
			want:      true,
		},
		{
			name:      "delimiter at end",
			data:      []byte("abcdefg,"),
			delimiter: ',',
			want:      true,
		},
		{
			name:      "multiple delimiters",
			data:      []byte("ab,de,gh"),
			delimiter: ',',
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.data) != 8 {
				t.Fatalf("test data must be exactly 8 bytes, got %d", len(tt.data))
			}

			// Convert to uint64
			var val uint64
			for i := 0; i < 8; i++ {
				val |= uint64(tt.data[i]) << (i * 8)
			}

			got := hasDelimiter(val, tt.delimiter)
			if got != tt.want {
				t.Errorf("hasDelimiter(%q, %q) = %v, want %v", tt.data, tt.delimiter, got, tt.want)
			}
		})
	}
}

func TestSWAR_FindDelimiterPos(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		delimiter byte
		wantPos   int
	}{
		{
			name:      "comma at position 3",
			data:      []byte("abc,defg"),
			delimiter: ',',
			wantPos:   3,
		},
		{
			name:      "comma at position 0",
			data:      []byte(",bcdefgh"),
			delimiter: ',',
			wantPos:   0,
		},
		{
			name:      "comma at position 7",
			data:      []byte("abcdefg,"),
			delimiter: ',',
			wantPos:   7,
		},
		{
			name:      "LF at position 4",
			data:      []byte("abcd\nfgh"),
			delimiter: '\n',
			wantPos:   4,
		},
		{
			name:      "no delimiter",
			data:      []byte("abcdefgh"),
			delimiter: ',',
			wantPos:   -1,
		},
		{
			name:      "multiple delimiters - returns first",
			data:      []byte("ab,de,gh"),
			delimiter: ',',
			wantPos:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.data) != 8 {
				t.Fatalf("test data must be exactly 8 bytes, got %d", len(tt.data))
			}

			// Convert to uint64
			var val uint64
			for i := 0; i < 8; i++ {
				val |= uint64(tt.data[i]) << (i * 8)
			}

			got := findDelimiterPos(val, tt.delimiter)
			if got != tt.wantPos {
				t.Errorf("findDelimiterPos(%q, %q) = %d, want %d", tt.data, tt.delimiter, got, tt.wantPos)
			}
		})
	}
}

func TestChunkedParser_ComparisonWithStandardParser(t *testing.T) {
	// Test that chunked parser produces identical results to standard parser
	tests := []string{
		"a,b,c",
		"a,b,c\nd,e,f",
		`"quoted,field",normal,field`,
		`"field with ""quotes""",another`,
		"field1,field2,field3\nfield4,field5,field6\nfield7,field8,field9",
		"a,,c\n,b,\n,,",
		"\"multi\nline\nfield\",normal",
		"a,b\r\nc,d\r\ne,f",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			standard, err := Parse([]byte(input))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			chunked, err := ParseChunked([]byte(input))
			if err != nil {
				t.Fatalf("ParseChunked() error = %v", err)
			}

			if !reflect.DeepEqual(standard, chunked) {
				t.Errorf("results differ:\nstandard = %v\nchunked  = %v", standard, chunked)
			}
		})
	}
}

// TestChunkedParser_ErrorPaths tests error handling in parseRecord.
func TestChunkedParser_ErrorPaths(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "unclosed quoted field",
			input:   `"hello world`,
			wantErr: true,
		},
		{
			name:    "unclosed quoted field at end",
			input:   `a,b,"c`,
			wantErr: true,
		},
		{
			name:    "quote in unquoted field",
			input:   `hel"lo,world`,
			wantErr: true,
		},
		{
			name:    "unclosed quote with escaped quotes",
			input:   `"hello""world`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseChunked([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseChunked() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestChunkedParser_LineEndingEdgeCases tests various line ending scenarios.
func TestChunkedParser_LineEndingEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "bare CR line endings",
			input: "a,b\rc,d\re,f",
			want:  [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}},
		},
		{
			name:  "multiple consecutive CR",
			input: "a,b\r\r\rc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "CRLF at chunk boundary",
			input: "abcdefg\r\nijklmno",
			want:  [][]string{{"abcdefg"}, {"ijklmno"}},
		},
		{
			name:  "CR without LF at end",
			input: "a,b,c\r",
			want:  [][]string{{"a", "b", "c"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseChunked([]byte(tt.input))
			if err != nil {
				t.Errorf("ParseChunked() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseChunked() = %v, want %v", got, tt.want)
			}
		})
	}
}
