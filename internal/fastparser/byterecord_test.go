package fastparser

import (
	"reflect"
	"testing"
)

func TestByteRecord_BasicOperations(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		offsets []int
		want    []string
	}{
		{
			name:    "single field",
			data:    []byte("hello"),
			offsets: []int{0, 5},
			want:    []string{"hello"},
		},
		{
			name:    "three fields",
			data:    []byte("abcdefghi"),
			offsets: []int{0, 3, 6, 9},
			want:    []string{"abc", "def", "ghi"},
		},
		{
			name:    "empty fields",
			data:    []byte("abc"),
			offsets: []int{0, 0, 3, 3},
			want:    []string{"", "abc", ""},
		},
		{
			name:    "all empty",
			data:    []byte(""),
			offsets: []int{0, 0, 0},
			want:    []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := NewByteRecord(tt.data, tt.offsets)

			// Test NumFields
			if got := record.NumFields(); got != len(tt.want) {
				t.Errorf("NumFields() = %d, want %d", got, len(tt.want))
			}

			// Test Field()
			for i, want := range tt.want {
				if got := record.Field(i); got != want {
					t.Errorf("Field(%d) = %q, want %q", i, got, want)
				}
			}

			// Test Fields()
			if got := record.Fields(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Fields() = %v, want %v", got, tt.want)
			}

			// Test FieldBytes()
			for i, want := range tt.want {
				gotBytes := record.FieldBytes(i)
				if got := string(gotBytes); got != want {
					t.Errorf("FieldBytes(%d) = %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestByteRecord_OutOfBounds(t *testing.T) {
	record := NewByteRecord([]byte("abc"), []int{0, 3})

	// Test negative index
	if got := record.Field(-1); got != "" {
		t.Errorf("Field(-1) = %q, want empty string", got)
	}
	if got := record.FieldBytes(-1); got != nil {
		t.Errorf("FieldBytes(-1) = %v, want nil", got)
	}

	// Test index >= NumFields
	if got := record.Field(1); got != "" {
		t.Errorf("Field(1) = %q, want empty string", got)
	}
	if got := record.FieldBytes(1); got != nil {
		t.Errorf("FieldBytes(1) = %v, want nil", got)
	}
}

func TestParseByteRecords_BasicParsing(t *testing.T) {
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
		{
			name:  "quoted empty field",
			input: `""`,
			want:  [][]string{{""}},
		},
		{
			name:    "unclosed quoted field",
			input:   `"hello`,
			wantErr: true,
		},
		{
			name:    "quote in middle of unquoted field",
			input:   `hel"lo`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseByteRecords([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseByteRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Convert ByteRecords to [][]string for comparison
				gotStrings := make([][]string, len(got))
				for i, record := range got {
					gotStrings[i] = record.Fields()
				}

				if !reflect.DeepEqual(gotStrings, tt.want) {
					t.Errorf("ParseByteRecords() = %v, want %v", gotStrings, tt.want)
				}
			}
		})
	}
}

func TestParseByteRecords_RFC4180Examples(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name: "RFC 4180 Example 1",
			input: `aaa,bbb,ccc
zzz,yyy,xxx`,
			want: [][]string{
				{"aaa", "bbb", "ccc"},
				{"zzz", "yyy", "xxx"},
			},
		},
		{
			name:  "RFC 4180 Example 6 - embedded comma",
			input: `"aaa","b,bb","ccc"`,
			want:  [][]string{{"aaa", "b,bb", "ccc"}},
		},
		{
			name:  "RFC 4180 Example 7 - embedded newline",
			input: "\"aaa\",\"b\nbb\",\"ccc\"",
			want:  [][]string{{"aaa", "b\nbb", "ccc"}},
		},
		{
			name:  "RFC 4180 Example 8 - embedded double-quote",
			input: `"aaa","b""bb","ccc"`,
			want:  [][]string{{"aaa", `b"bb`, "ccc"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseByteRecords([]byte(tt.input))
			if err != nil {
				t.Errorf("ParseByteRecords() error = %v", err)
				return
			}

			// Convert ByteRecords to [][]string for comparison
			gotStrings := make([][]string, len(got))
			for i, record := range got {
				gotStrings[i] = record.Fields()
			}

			if !reflect.DeepEqual(gotStrings, tt.want) {
				t.Errorf("ParseByteRecords() = %v, want %v", gotStrings, tt.want)
			}
		})
	}
}

func TestParseByteRecords_FieldBytesNoAllocation(t *testing.T) {
	// Test that FieldBytes returns a slice that shares the original data
	input := []byte("abc,def,ghi")
	records, err := ParseByteRecords(input)
	if err != nil {
		t.Fatalf("ParseByteRecords() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	record := records[0]

	// Get field bytes
	field0 := record.FieldBytes(0)
	field1 := record.FieldBytes(1)
	field2 := record.FieldBytes(2)

	// Verify content
	if string(field0) != "abc" {
		t.Errorf("field0 = %q, want %q", field0, "abc")
	}
	if string(field1) != "def" {
		t.Errorf("field1 = %q, want %q", field1, "def")
	}
	if string(field2) != "ghi" {
		t.Errorf("field2 = %q, want %q", field2, "ghi")
	}

	// Verify the slices point to the record's internal data
	// (This is an implementation detail test, but important for the pattern)
	recordData := record.data
	if &field0[0] != &recordData[0] {
		t.Error("field0 does not share memory with record data")
	}
	if &field1[0] != &recordData[3] {
		t.Error("field1 does not share memory with record data")
	}
	if &field2[0] != &recordData[6] {
		t.Error("field2 does not share memory with record data")
	}
}

func TestUnmarshalBytes_ToStringSlice(t *testing.T) {
	input := []byte("a,b,c\nd,e,f\ng,h,i")

	var records [][]string
	err := UnmarshalBytes(input, &records)
	if err != nil {
		t.Fatalf("UnmarshalBytes() error = %v", err)
	}

	want := [][]string{
		{"a", "b", "c"},
		{"d", "e", "f"},
		{"g", "h", "i"},
	}

	if !reflect.DeepEqual(records, want) {
		t.Errorf("UnmarshalBytes() = %v, want %v", records, want)
	}
}

func TestUnmarshalBytes_ToStruct(t *testing.T) {
	type Person struct {
		Name  string `csv:"name"`
		Age   int    `csv:"age"`
		Email string `csv:"email"`
	}

	input := []byte("Name,Age,Email\nJohn,30,john@example.com\nJane,25,jane@example.com")

	var people []Person
	err := UnmarshalBytes(input, &people)
	if err != nil {
		t.Fatalf("UnmarshalBytes() error = %v", err)
	}

	want := []Person{
		{Name: "John", Age: 30, Email: "john@example.com"},
		{Name: "Jane", Age: 25, Email: "jane@example.com"},
	}

	if !reflect.DeepEqual(people, want) {
		t.Errorf("UnmarshalBytes() = %v, want %v", people, want)
	}
}

func TestUnmarshalBytes_EmptyData(t *testing.T) {
	var records [][]string
	err := UnmarshalBytes([]byte(""), &records)
	if err != nil {
		t.Fatalf("UnmarshalBytes() error = %v", err)
	}

	if len(records) != 0 {
		t.Errorf("expected empty slice, got %v", records)
	}
}

func TestUnmarshalBytes_Errors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		target  interface{}
		wantErr string
	}{
		{
			name:    "nil target",
			data:    []byte("a,b,c"),
			target:  nil,
			wantErr: "UnmarshalBytes(nil)",
		},
		{
			name:    "non-pointer",
			data:    []byte("a,b,c"),
			target:  [][]string{},
			wantErr: "non-pointer",
		},
		{
			name:    "nil pointer",
			data:    []byte("a,b,c"),
			target:  (*[][]string)(nil),
			wantErr: "nil *[][]string",
		},
		{
			name:    "non-slice",
			data:    []byte("a,b,c"),
			target:  new(string),
			wantErr: "pointer to slice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalBytes(tt.data, tt.target)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestUnmarshalBytes_StructWithTypes(t *testing.T) {
	type Record struct {
		Name   string  `csv:"name"`
		Age    int     `csv:"age"`
		Score  float64 `csv:"score"`
		Active bool    `csv:"active"`
	}

	input := []byte("name,age,score,active\nAlice,30,95.5,true\nBob,25,87.3,false")

	var records []Record
	err := UnmarshalBytes(input, &records)
	if err != nil {
		t.Fatalf("UnmarshalBytes() error = %v", err)
	}

	want := []Record{
		{Name: "Alice", Age: 30, Score: 95.5, Active: true},
		{Name: "Bob", Age: 25, Score: 87.3, Active: false},
	}

	if !reflect.DeepEqual(records, want) {
		t.Errorf("UnmarshalBytes() = %v, want %v", records, want)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > len(substr)+1 && containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestByteRecord_NumFieldsEdgeCases tests NumFields with empty and varying field counts.
func TestByteRecord_NumFieldsEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		offsets []int
		want    int
	}{
		{
			name:    "empty record - no offsets",
			data:    []byte{},
			offsets: []int{},
			want:    0,
		},
		{
			name:    "empty record - single offset",
			data:    []byte{},
			offsets: []int{0},
			want:    0,
		},
		{
			name:    "single empty field",
			data:    []byte{},
			offsets: []int{0, 0},
			want:    1,
		},
		{
			name:    "two empty fields",
			data:    []byte{},
			offsets: []int{0, 0, 0},
			want:    2,
		},
		{
			name:    "single field",
			data:    []byte("a"),
			offsets: []int{0, 1},
			want:    1,
		},
		{
			name:    "ten fields",
			data:    []byte("0123456789"),
			offsets: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			want:    10,
		},
		{
			name:    "hundred fields",
			data:    make([]byte, 100),
			offsets: func() []int {
				offsets := make([]int, 101)
				for i := range offsets {
					offsets[i] = i
				}
				return offsets
			}(),
			want: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := NewByteRecord(tt.data, tt.offsets)
			got := record.NumFields()
			if got != tt.want {
				t.Errorf("NumFields() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestByteRecord_SkipNewlineEdgeCases tests skipNewline with various line endings.
func TestByteRecord_SkipNewlineEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "CRLF line endings",
			input: "a,b\r\nc,d\r\n",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "bare CR line endings",
			input: "a,b\rc,d\r",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "bare LF line endings",
			input: "a,b\nc,d\n",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "mixed line endings",
			input: "a,b\nc,d\re,f\r\ng,h",
			want:  [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}, {"g", "h"}},
		},
		{
			name:  "multiple consecutive CR",
			input: "a,b\r\r\rc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "multiple consecutive CRLF",
			input: "a,b\r\n\r\n\r\nc,d",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseByteRecords([]byte(tt.input))
			if err != nil {
				t.Errorf("ParseByteRecords() error = %v", err)
				return
			}

			// Convert ByteRecords to [][]string for comparison
			gotStrings := make([][]string, len(got))
			for i, record := range got {
				gotStrings[i] = record.Fields()
			}

			if !reflect.DeepEqual(gotStrings, tt.want) {
				t.Errorf("ParseByteRecords() = %v, want %v", gotStrings, tt.want)
			}
		})
	}
}
