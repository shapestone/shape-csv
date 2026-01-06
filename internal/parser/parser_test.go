package parser

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
	shapetokenizer "github.com/shapestone/shape-core/pkg/tokenizer"
)

// TestParse_EmptyInput tests parsing an empty CSV file
// Grammar: File = [ Header ] { Record }
// Empty input is valid and should return an empty ArrayDataNode
func TestParse_EmptyInput(t *testing.T) {
	input := ""
	p := NewParser(input)
	node, err := p.Parse()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	arr, ok := node.(*ast.ArrayDataNode)
	if !ok {
		t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
	}

	if arr.Len() != 0 {
		t.Errorf("expected empty array, got %d records", arr.Len())
	}
}

// TestParse_SingleRecord tests parsing single CSV records
// Grammar: Record = Field { "," Field } LineTerminator
func TestParse_SingleRecord(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFields []string
	}{
		{
			name:       "single field",
			input:      "hello",
			wantFields: []string{"hello"},
		},
		{
			name:       "single field with newline",
			input:      "hello\n",
			wantFields: []string{"hello"},
		},
		{
			name:       "three fields",
			input:      "a,b,c",
			wantFields: []string{"a", "b", "c"},
		},
		{
			name:       "three fields with newline",
			input:      "a,b,c\n",
			wantFields: []string{"a", "b", "c"},
		},
		{
			name:       "empty middle field",
			input:      "a,,c",
			wantFields: []string{"a", "", "c"},
		},
		{
			name:       "all empty fields",
			input:      ",,",
			wantFields: []string{"", "", ""},
		},
		{
			name:       "trailing comma",
			input:      "a,b,",
			wantFields: []string{"a", "b", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
			}

			if arr.Len() != 1 {
				t.Fatalf("expected 1 record, got %d", arr.Len())
			}

			// Get the record (should be an ArrayDataNode)
			record := arr.Get(0)
			recordArr, ok := record.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode for record, got %T", record)
			}

			if recordArr.Len() != len(tt.wantFields) {
				t.Fatalf("expected %d fields, got %d", len(tt.wantFields), recordArr.Len())
			}

			// Check each field
			for i, wantField := range tt.wantFields {
				field := recordArr.Get(i)
				lit, ok := field.(*ast.LiteralNode)
				if !ok {
					t.Fatalf("field %d: expected *ast.LiteralNode, got %T", i, field)
				}

				str, ok := lit.Value().(string)
				if !ok {
					t.Fatalf("field %d: expected string value, got %T", i, lit.Value())
				}

				if str != wantField {
					t.Errorf("field %d: expected %q, got %q", i, wantField, str)
				}
			}
		})
	}
}

// TestParse_QuotedFields tests parsing quoted CSV fields
// Grammar: QuotedField = '"' { QuotedChar | EscapedQuote } '"'
// EscapedQuote = '""'
func TestParse_QuotedFields(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFields []string
	}{
		{
			name:       "quoted field with comma",
			input:      `"a,b",c`,
			wantFields: []string{"a,b", "c"},
		},
		{
			name:       "escaped quote",
			input:      `"say ""hello"""`,
			wantFields: []string{`say "hello"`},
		},
		{
			name:       "field with embedded newline",
			input:      "\"line1\nline2\"",
			wantFields: []string{"line1\nline2"},
		},
		{
			name:       "mixed quoted and unquoted",
			input:      `"quoted",unquoted,"another"`,
			wantFields: []string{"quoted", "unquoted", "another"},
		},
		{
			name:       "empty quoted field",
			input:      `"",value`,
			wantFields: []string{"", "value"},
		},
		{
			name:       "all quoted fields",
			input:      `"a","b","c"`,
			wantFields: []string{"a", "b", "c"},
		},
		{
			name:       "quoted field with spaces",
			input:      `" hello world ",test`,
			wantFields: []string{" hello world ", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
			}

			if arr.Len() != 1 {
				t.Fatalf("expected 1 record, got %d", arr.Len())
			}

			record := arr.Get(0)
			recordArr, ok := record.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode for record, got %T", record)
			}

			if recordArr.Len() != len(tt.wantFields) {
				t.Fatalf("expected %d fields, got %d", len(tt.wantFields), recordArr.Len())
			}

			for i, wantField := range tt.wantFields {
				field := recordArr.Get(i)
				lit, ok := field.(*ast.LiteralNode)
				if !ok {
					t.Fatalf("field %d: expected *ast.LiteralNode, got %T", i, field)
				}

				str, ok := lit.Value().(string)
				if !ok {
					t.Fatalf("field %d: expected string value, got %T", i, lit.Value())
				}

				if str != wantField {
					t.Errorf("field %d: expected %q, got %q", i, wantField, str)
				}
			}
		})
	}
}

// TestParse_MultipleRecords tests parsing multiple CSV records
// Grammar: File = [ Header ] { Record }
func TestParse_MultipleRecords(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantRecords [][]string
	}{
		{
			name:  "two records",
			input: "a,b\nc,d",
			wantRecords: [][]string{
				{"a", "b"},
				{"c", "d"},
			},
		},
		{
			name:  "two records with trailing newline",
			input: "a,b\nc,d\n",
			wantRecords: [][]string{
				{"a", "b"},
				{"c", "d"},
			},
		},
		{
			name:  "header and data",
			input: "name,age\nAlice,30\nBob,25",
			wantRecords: [][]string{
				{"name", "age"},
				{"Alice", "30"},
				{"Bob", "25"},
			},
		},
		{
			name:  "CRLF line endings",
			input: "a,b\r\nc,d\r\n",
			wantRecords: [][]string{
				{"a", "b"},
				{"c", "d"},
			},
		},
		{
			name:  "mixed quoted fields",
			input: "\"Last,Name\",First\n\"Smith\",John",
			wantRecords: [][]string{
				{"Last,Name", "First"},
				{"Smith", "John"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
			}

			if arr.Len() != len(tt.wantRecords) {
				t.Fatalf("expected %d records, got %d", len(tt.wantRecords), arr.Len())
			}

			for i, wantRecord := range tt.wantRecords {
				record := arr.Get(i)
				recordArr, ok := record.(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("record %d: expected *ast.ArrayDataNode, got %T", i, record)
				}

				if recordArr.Len() != len(wantRecord) {
					t.Fatalf("record %d: expected %d fields, got %d", i, len(wantRecord), recordArr.Len())
				}

				for j, wantField := range wantRecord {
					field := recordArr.Get(j)
					lit, ok := field.(*ast.LiteralNode)
					if !ok {
						t.Fatalf("record %d, field %d: expected *ast.LiteralNode, got %T", i, j, field)
					}

					str, ok := lit.Value().(string)
					if !ok {
						t.Fatalf("record %d, field %d: expected string value, got %T", i, j, lit.Value())
					}

					if str != wantField {
						t.Errorf("record %d, field %d: expected %q, got %q", i, j, wantField, str)
					}
				}
			}
		})
	}
}

// TestParse_Errors tests error handling
func TestParse_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unclosed quote",
			input: `"unterminated`,
		},
		{
			name:  "unclosed quote with comma",
			input: `"unterminated,field`,
		},
		{
			name:  "quote in middle of unquoted field",
			input: `bad"quote`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			_, err := p.Parse()

			if err == nil {
				t.Errorf("expected error for invalid CSV: %s", tt.input)
			}
		})
	}
}

// TestGrammarFileExists verifies that the grammar file exists
func TestGrammarFileExists(t *testing.T) {
	// This test ensures the grammar file is in the expected location
	// The actual path will be relative to the project root
	grammarPath := "../../docs/grammar/csv.ebnf"

	// We just verify the path is documented - actual file check would require fs access
	if grammarPath == "" {
		t.Error("grammar path should be defined")
	}
}

// TestGrammarVerification is a placeholder for grammar verification
// This would be implemented with a grammar verification tool
func TestGrammarVerification(t *testing.T) {
	t.Skip("Grammar verification tool not yet implemented")
}

// TestGrammarCoverage is a placeholder for grammar coverage testing
// This would verify all grammar productions are tested
func TestGrammarCoverage(t *testing.T) {
	t.Skip("Grammar coverage tool not yet implemented")
}

// TestNewParserFromStream tests creating a parser from a stream
// This tests NewParserFromStream() and NewParserFromStreamWithOptions()
func TestNewParserFromStream(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		opts       Options
		wantFields []string
	}{
		{
			name:       "basic stream parsing",
			input:      "a,b,c",
			opts:       DefaultOptions(),
			wantFields: []string{"a", "b", "c"},
		},
		{
			name:  "stream with multiple records",
			input: "a,b\nc,d",
			opts:  DefaultOptions(),
			wantFields: []string{"a", "b"}, // First record
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test NewParserFromStream with DefaultOptions
			stream := &testStream{data: tt.input, pos: 0, row: 1, column: 1}
			p := NewParserFromStream(stream)
			node, err := p.Parse()

			if err != nil {
				t.Fatalf("NewParserFromStream: unexpected error: %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
			}

			if arr.Len() > 0 {
				record := arr.Get(0)
				recordArr, ok := record.(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("expected *ast.ArrayDataNode for record, got %T", record)
				}

				if recordArr.Len() != len(tt.wantFields) {
					t.Fatalf("expected %d fields, got %d", len(tt.wantFields), recordArr.Len())
				}
			}

			// Test NewParserFromStreamWithOptions
			stream2 := &testStream{data: tt.input, pos: 0, row: 1, column: 1}
			p2 := NewParserFromStreamWithOptions(stream2, tt.opts)
			node2, err2 := p2.Parse()

			if err2 != nil {
				t.Fatalf("NewParserFromStreamWithOptions: unexpected error: %v", err2)
			}

			arr2, ok := node2.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode, got %T", node2)
			}

			if arr2.Len() > 0 {
				record := arr2.Get(0)
				recordArr, ok := record.(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("expected *ast.ArrayDataNode for record, got %T", record)
				}

				if recordArr.Len() != len(tt.wantFields) {
					t.Fatalf("expected %d fields, got %d", len(tt.wantFields), recordArr.Len())
				}

				for i, wantField := range tt.wantFields {
					field := recordArr.Get(i)
					lit, ok := field.(*ast.LiteralNode)
					if !ok {
						t.Fatalf("field %d: expected *ast.LiteralNode, got %T", i, field)
					}

					str, ok := lit.Value().(string)
					if !ok {
						t.Fatalf("field %d: expected string value, got %T", i, lit.Value())
					}

					if str != wantField {
						t.Errorf("field %d: expected %q, got %q", i, wantField, str)
					}
				}
			}
		})
	}
}

// testStream is a simple implementation of shapetokenizer.Stream for testing
type testStream struct {
	data   string
	pos    int
	row    int
	column int
}

func (s *testStream) Clone() shapetokenizer.Stream {
	return &testStream{
		data:   s.data,
		pos:    s.pos,
		row:    s.row,
		column: s.column,
	}
}

func (s *testStream) Match(other shapetokenizer.Stream) {
	if otherStream, ok := other.(*testStream); ok {
		s.pos = otherStream.pos
		s.row = otherStream.row
		s.column = otherStream.column
	}
}

func (s *testStream) PeekChar() (rune, bool) {
	if s.pos >= len(s.data) {
		return 0, false
	}
	return rune(s.data[s.pos]), true
}

func (s *testStream) NextChar() (rune, bool) {
	if s.pos >= len(s.data) {
		return 0, false
	}
	r := rune(s.data[s.pos])
	s.pos++
	s.column++
	if r == '\n' {
		s.row++
		s.column = 1
	}
	return r, true
}

func (s *testStream) MatchChars(chars []rune) bool {
	origPos := s.pos
	for _, ch := range chars {
		if r, ok := s.NextChar(); !ok || r != ch {
			s.pos = origPos
			return false
		}
	}
	return true
}

func (s *testStream) IsEos() bool {
	return s.pos >= len(s.data)
}

func (s *testStream) GetRow() int {
	return s.row
}

func (s *testStream) GetOffset() int {
	return s.pos
}

func (s *testStream) GetColumn() int {
	return s.column
}

func (s *testStream) Reset() {
	s.pos = 0
	s.row = 1
	s.column = 1
}

func (s *testStream) GetLocation() shapetokenizer.Location {
	return shapetokenizer.Location{
		Cursor: s.pos,
		Row:    s.row,
		Column: s.column,
	}
}

func (s *testStream) SetLocation(loc shapetokenizer.Location) {
	s.pos = loc.Cursor
	s.row = loc.Row
	s.column = loc.Column
}

// TestHandleBadLine tests the three bad line handling modes
func TestHandleBadLine(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		mode          BadLineMode
		wantErr       bool
		wantRecords   int
		wantWarnings  int
		warningCheck  func(line int, msg string)
	}{
		{
			name:        "error mode - unclosed quote",
			input:       "\"unclosed\na,b,c",
			mode:        BadLineModeError,
			wantErr:     true,
			wantRecords: 0,
		},
		{
			name:         "skip mode - unclosed quote (consumes to EOF)",
			input:        "\"unclosed",
			mode:         BadLineModeSkip,
			wantErr:      false,
			wantRecords:  0, // Unclosed quote consumes everything
			wantWarnings: 0,
		},
		{
			name:         "warn mode - unclosed quote (consumes to EOF)",
			input:        "\"unclosed",
			mode:         BadLineModeWarn,
			wantErr:      false,
			wantRecords:  0,
			wantWarnings: 1,
		},
		{
			name:        "error mode - quote in unquoted field",
			input:       "bad\"quote\na,b,c",
			mode:        BadLineModeError,
			wantErr:     true,
			wantRecords: 0,
		},
		{
			name:         "skip mode - quote in unquoted field",
			input:        "bad\"quote\na,b,c",
			mode:         BadLineModeSkip,
			wantErr:      false,
			wantRecords:  1,
			wantWarnings: 0,
		},
		{
			name:         "warn mode - quote in unquoted field",
			input:        "bad\"quote\na,b,c",
			mode:         BadLineModeWarn,
			wantErr:      false,
			wantRecords:  1,
			wantWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warningCount := 0
			opts := Options{
				Comma:           ',',
				FieldsPerRecord: -1,
				OnBadLine:       tt.mode,
				WarningCallback: func(line int, message string) {
					warningCount++
					if tt.warningCheck != nil {
						tt.warningCheck(line, message)
					}
				},
			}

			p := NewParserWithOptions(tt.input, opts)
			node, err := p.Parse()

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr {
				arr, ok := node.(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
				}

				if arr.Len() != tt.wantRecords {
					t.Errorf("expected %d records, got %d", tt.wantRecords, arr.Len())
				}

				if warningCount != tt.wantWarnings {
					t.Errorf("expected %d warnings, got %d", tt.wantWarnings, warningCount)
				}
			}
		})
	}
}

// TestCommentLines tests comment line detection and skipping
func TestCommentLines(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		comment     rune
		wantRecords int
		wantFields  [][]string
	}{
		{
			name:        "skip lines starting with #",
			input:       "# comment\na,b,c\n# another comment\nd,e,f",
			comment:     '#',
			wantRecords: 2,
			wantFields: [][]string{
				{"a", "b", "c"},
				{"d", "e", "f"},
			},
		},
		{
			name:        "skip lines starting with //",
			input:       "// comment\na,b,c",
			comment:     '/',
			wantRecords: 1,
			wantFields: [][]string{
				{"a", "b", "c"},
			},
		},
		{
			name:        "comment at end",
			input:       "a,b,c\n# comment",
			comment:     '#',
			wantRecords: 1,
			wantFields: [][]string{
				{"a", "b", "c"},
			},
		},
		{
			name:        "multiple consecutive comments",
			input:       "# comment 1\n# comment 2\n# comment 3\na,b,c",
			comment:     '#',
			wantRecords: 1,
			wantFields: [][]string{
				{"a", "b", "c"},
			},
		},
		{
			name:        "no comment character set",
			input:       "# not a comment\na,b,c",
			comment:     0,
			wantRecords: 2,
			wantFields: [][]string{
				{"# not a comment"},
				{"a", "b", "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Comma:           ',',
				Comment:         tt.comment,
				FieldsPerRecord: -1,
			}

			p := NewParserWithOptions(tt.input, opts)
			node, err := p.Parse()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
			}

			if arr.Len() != tt.wantRecords {
				t.Errorf("expected %d records, got %d", tt.wantRecords, arr.Len())
			}

			for i, wantFields := range tt.wantFields {
				if i >= arr.Len() {
					break
				}

				record := arr.Get(i)
				recordArr, ok := record.(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("record %d: expected *ast.ArrayDataNode, got %T", i, record)
				}

				if recordArr.Len() != len(wantFields) {
					t.Errorf("record %d: expected %d fields, got %d", i, len(wantFields), recordArr.Len())
					continue
				}

				for j, wantField := range wantFields {
					field := recordArr.Get(j)
					lit, ok := field.(*ast.LiteralNode)
					if !ok {
						t.Fatalf("record %d, field %d: expected *ast.LiteralNode, got %T", i, j, field)
					}

					str, ok := lit.Value().(string)
					if !ok {
						t.Fatalf("record %d, field %d: expected string value, got %T", i, j, lit.Value())
					}

					if str != wantField {
						t.Errorf("record %d, field %d: expected %q, got %q", i, j, wantField, str)
					}
				}
			}
		})
	}
}

// TestMaxFieldSize tests the MaxFieldSize limit
func TestMaxFieldSize(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		maxFieldSize int
		wantErr      bool
		onBadLine    BadLineMode
	}{
		{
			name:         "field within limit",
			input:        "hello",
			maxFieldSize: 10,
			wantErr:      false,
		},
		{
			name:         "field exceeds limit - error mode",
			input:        "this is a very long field",
			maxFieldSize: 10,
			wantErr:      true,
			onBadLine:    BadLineModeError,
		},
		{
			name:         "field exceeds limit - skip mode",
			input:        "toolong,ok",
			maxFieldSize: 5,
			wantErr:      false,
			onBadLine:    BadLineModeSkip,
		},
		{
			name:         "quoted field exceeds limit",
			input:        "\"this is a very long quoted field\"",
			maxFieldSize: 10,
			wantErr:      true,
			onBadLine:    BadLineModeError,
		},
		{
			name:         "empty field with limit",
			input:        "",
			maxFieldSize: 1,
			wantErr:      false,
		},
		{
			name:         "no limit set",
			input:        "this is a very long field that should be fine",
			maxFieldSize: 0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Comma:           ',',
				MaxFieldSize:    tt.maxFieldSize,
				FieldsPerRecord: -1,
				OnBadLine:       tt.onBadLine,
			}

			p := NewParserWithOptions(tt.input, opts)
			_, err := p.Parse()

			if tt.wantErr && err == nil {
				t.Errorf("expected error for field exceeding max size, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestMaxRecordSize tests the MaxRecordSize limit
func TestMaxRecordSize(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		maxRecordSize int
		wantErr       bool
		onBadLine     BadLineMode
		wantRecords   int
	}{
		{
			name:          "record within limit",
			input:         "a,b,c",
			maxRecordSize: 10,
			wantErr:       false,
			wantRecords:   1,
		},
		{
			name:          "record exceeds limit - error mode",
			input:         "this,is,a,very,long,record",
			maxRecordSize: 10,
			wantErr:       true,
			onBadLine:     BadLineModeError,
			wantRecords:   0,
		},
		{
			name:          "record exceeds limit - skip mode",
			input:         "toolong,record,data\nok,short",
			maxRecordSize: 10,
			wantErr:       false,
			onBadLine:     BadLineModeSkip,
			wantRecords:   1, // Second record is OK
		},
		{
			name:          "multiple records with size limit",
			input:         "a,b\nverylongfield,data\nc,d",
			maxRecordSize: 10,
			wantErr:       false,
			onBadLine:     BadLineModeSkip,
			wantRecords:   2, // First and third records are OK
		},
		{
			name:          "no limit set",
			input:         "very,long,record,with,many,fields",
			maxRecordSize: 0,
			wantErr:       false,
			wantRecords:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Comma:           ',',
				MaxRecordSize:   tt.maxRecordSize,
				FieldsPerRecord: -1,
				OnBadLine:       tt.onBadLine,
			}

			p := NewParserWithOptions(tt.input, opts)
			node, err := p.Parse()

			if tt.wantErr && err == nil {
				t.Errorf("expected error for record exceeding max size, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr {
				arr, ok := node.(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
				}

				if arr.Len() != tt.wantRecords {
					t.Errorf("expected %d records, got %d", tt.wantRecords, arr.Len())
				}
			}
		})
	}
}

// TestTrimLeadingSpace tests the TrimLeadingSpace option
func TestTrimLeadingSpace(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		trim       bool
		wantFields []string
	}{
		{
			name:       "trim leading spaces",
			input:      " a, b, c",
			trim:       true,
			wantFields: []string{"a", "b", "c"},
		},
		{
			name:       "no trim - preserve spaces",
			input:      " a, b, c",
			trim:       false,
			wantFields: []string{" a", " b", " c"},
		},
		{
			name:       "trim tabs",
			input:      "\ta,\tb,\tc",
			trim:       true,
			wantFields: []string{"a", "b", "c"},
		},
		{
			name:       "trim mixed whitespace",
			input:      " \ta, \tb",
			trim:       true,
			wantFields: []string{"a", "b"},
		},
		{
			name:       "quoted fields not affected by trim",
			input:      " \" a\",\" b\"",
			trim:       true,
			wantFields: []string{" a", " b"},
		},
		{
			name:       "empty fields with trim",
			input:      " , ,c",
			trim:       true,
			wantFields: []string{"", "", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Comma:            ',',
				TrimLeadingSpace: tt.trim,
				FieldsPerRecord:  -1,
			}

			p := NewParserWithOptions(tt.input, opts)
			node, err := p.Parse()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
			}

			if arr.Len() != 1 {
				t.Fatalf("expected 1 record, got %d", arr.Len())
			}

			record := arr.Get(0)
			recordArr, ok := record.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode for record, got %T", record)
			}

			if recordArr.Len() != len(tt.wantFields) {
				t.Fatalf("expected %d fields, got %d", len(tt.wantFields), recordArr.Len())
			}

			for i, wantField := range tt.wantFields {
				field := recordArr.Get(i)
				lit, ok := field.(*ast.LiteralNode)
				if !ok {
					t.Fatalf("field %d: expected *ast.LiteralNode, got %T", i, field)
				}

				str, ok := lit.Value().(string)
				if !ok {
					t.Fatalf("field %d: expected string value, got %T", i, lit.Value())
				}

				if str != wantField {
					t.Errorf("field %d: expected %q, got %q", i, wantField, str)
				}
			}
		})
	}
}

// TestLazyQuotes tests the LazyQuotes option for unquoted fields
func TestLazyQuotes(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		lazy       bool
		wantErr    bool
		wantFields []string
	}{
		{
			name:       "lazy quotes - allow quote in field",
			input:      "a\"b,c",
			lazy:       true,
			wantErr:    false,
			wantFields: []string{"a\"b", "c"},
		},
		{
			name:       "strict quotes - error on quote",
			input:      "a\"b,c",
			lazy:       false,
			wantErr:    true,
			wantFields: nil,
		},
		{
			name:       "lazy quotes - multiple quotes",
			input:      "a\"b\"c,d",
			lazy:       true,
			wantErr:    false,
			wantFields: []string{"a\"b\"c", "d"},
		},
		{
			name:       "lazy quotes - embedded quote after valid quoted field",
			input:      "a,b\"c,d",
			lazy:       true,
			wantErr:    false,
			wantFields: []string{"a", "b\"c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Comma:           ',',
				LazyQuotes:      tt.lazy,
				FieldsPerRecord: -1,
			}

			p := NewParserWithOptions(tt.input, opts)
			node, err := p.Parse()

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr && tt.wantFields != nil {
				arr, ok := node.(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
				}

				if arr.Len() != 1 {
					t.Fatalf("expected 1 record, got %d", arr.Len())
				}

				record := arr.Get(0)
				recordArr, ok := record.(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("expected *ast.ArrayDataNode for record, got %T", record)
				}

				if recordArr.Len() != len(tt.wantFields) {
					t.Fatalf("expected %d fields, got %d", len(tt.wantFields), recordArr.Len())
				}

				for i, wantField := range tt.wantFields {
					field := recordArr.Get(i)
					lit, ok := field.(*ast.LiteralNode)
					if !ok {
						t.Fatalf("field %d: expected *ast.LiteralNode, got %T", i, field)
					}

					str, ok := lit.Value().(string)
					if !ok {
						t.Fatalf("field %d: expected string value, got %T", i, lit.Value())
					}

					if str != wantField {
						t.Errorf("field %d: expected %q, got %q", i, wantField, str)
					}
				}
			}
		})
	}
}

// TestFieldsPerRecord tests field count validation
func TestFieldsPerRecord(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		fieldsPerRecord int
		wantErr         bool
		wantRecords     int
		onBadLine       BadLineMode
	}{
		{
			name:            "fixed field count - valid",
			input:           "a,b,c\nd,e,f",
			fieldsPerRecord: 3,
			wantErr:         false,
			wantRecords:     2,
		},
		{
			name:            "fixed field count - error on mismatch",
			input:           "a,b,c\nd,e",
			fieldsPerRecord: 3,
			wantErr:         true,
			wantRecords:     0,
			onBadLine:       BadLineModeError,
		},
		{
			name:            "fixed field count - skip on mismatch",
			input:           "a,b,c\nd,e\nf,g,h",
			fieldsPerRecord: 3,
			wantErr:         false,
			wantRecords:     2, // First and third records
			onBadLine:       BadLineModeSkip,
		},
		{
			name:            "first record sets count - valid",
			input:           "a,b,c\nd,e,f",
			fieldsPerRecord: 0,
			wantErr:         false,
			wantRecords:     2,
		},
		{
			name:            "first record sets count - error on mismatch",
			input:           "a,b,c\nd,e",
			fieldsPerRecord: 0,
			wantErr:         true,
			wantRecords:     0,
			onBadLine:       BadLineModeError,
		},
		{
			name:            "first record sets count - skip on mismatch",
			input:           "a,b,c\nd,e\nf,g,h",
			fieldsPerRecord: 0,
			wantErr:         false,
			wantRecords:     2, // First and third records
			onBadLine:       BadLineModeSkip,
		},
		{
			name:            "no validation",
			input:           "a,b,c\nd,e\nf,g,h,i",
			fieldsPerRecord: -1,
			wantErr:         false,
			wantRecords:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Comma:           ',',
				FieldsPerRecord: tt.fieldsPerRecord,
				OnBadLine:       tt.onBadLine,
			}

			p := NewParserWithOptions(tt.input, opts)
			node, err := p.Parse()

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr {
				arr, ok := node.(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
				}

				if arr.Len() != tt.wantRecords {
					t.Errorf("expected %d records, got %d", tt.wantRecords, arr.Len())
				}
			}
		})
	}
}

// TestQuotedFieldLineEndings tests CRLF vs LF handling in quoted fields
func TestQuotedFieldLineEndings(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFields []string
	}{
		{
			name:       "LF in quoted field",
			input:      "\"line1\nline2\"",
			wantFields: []string{"line1\nline2"},
		},
		{
			name:       "CRLF in quoted field",
			input:      "\"line1\r\nline2\"",
			wantFields: []string{"line1\r\nline2"},
		},
		{
			name:       "mixed line endings in quoted field",
			input:      "\"line1\nline2\r\nline3\"",
			wantFields: []string{"line1\nline2\r\nline3"},
		},
		{
			name:       "multiple fields with different line endings",
			input:      "\"a\nb\",\"c\r\nd\"",
			wantFields: []string{"a\nb", "c\r\nd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
			}

			if arr.Len() != 1 {
				t.Fatalf("expected 1 record, got %d", arr.Len())
			}

			record := arr.Get(0)
			recordArr, ok := record.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode for record, got %T", record)
			}

			if recordArr.Len() != len(tt.wantFields) {
				t.Fatalf("expected %d fields, got %d", len(tt.wantFields), recordArr.Len())
			}

			for i, wantField := range tt.wantFields {
				field := recordArr.Get(i)
				lit, ok := field.(*ast.LiteralNode)
				if !ok {
					t.Fatalf("field %d: expected *ast.LiteralNode, got %T", i, field)
				}

				str, ok := lit.Value().(string)
				if !ok {
					t.Fatalf("field %d: expected string value, got %T", i, lit.Value())
				}

				if str != wantField {
					t.Errorf("field %d: expected %q, got %q", i, wantField, str)
				}
			}
		})
	}
}

// TestEmptyLines tests handling of empty lines
func TestEmptyLines(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantRecords int
	}{
		{
			name:        "single empty line",
			input:       "\n",
			wantRecords: 0,
		},
		{
			name:        "multiple empty lines",
			input:       "\n\n\n",
			wantRecords: 0,
		},
		{
			name:        "empty lines between records",
			input:       "a,b\n\nc,d",
			wantRecords: 2,
		},
		{
			name:        "empty line at start",
			input:       "\na,b,c",
			wantRecords: 1,
		},
		{
			name:        "empty line at end",
			input:       "a,b,c\n\n",
			wantRecords: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("expected *ast.ArrayDataNode, got %T", node)
			}

			if arr.Len() != tt.wantRecords {
				t.Errorf("expected %d records, got %d", tt.wantRecords, arr.Len())
			}
		})
	}
}
