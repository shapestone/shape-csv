package csv_test

import (
	"strings"
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-csv/pkg/csv"
)

func TestDefaultReaderOptions(t *testing.T) {
	opts := csv.DefaultReaderOptions()

	if opts.Comma != ',' {
		t.Errorf("DefaultReaderOptions().Comma = %q, want ','", opts.Comma)
	}
	if opts.Comment != 0 {
		t.Errorf("DefaultReaderOptions().Comment = %q, want 0", opts.Comment)
	}
	if opts.FieldsPerRecord != -1 {
		t.Errorf("DefaultReaderOptions().FieldsPerRecord = %d, want -1", opts.FieldsPerRecord)
	}
	if opts.LazyQuotes {
		t.Error("DefaultReaderOptions().LazyQuotes should be false")
	}
	if opts.TrimLeadingSpace {
		t.Error("DefaultReaderOptions().TrimLeadingSpace should be false")
	}
	if opts.ReuseRecord {
		t.Error("DefaultReaderOptions().ReuseRecord should be false")
	}
}

func TestDefaultWriterOptions(t *testing.T) {
	opts := csv.DefaultWriterOptions()

	if opts.Comma != ',' {
		t.Errorf("DefaultWriterOptions().Comma = %q, want ','", opts.Comma)
	}
	if opts.UseCRLF {
		t.Error("DefaultWriterOptions().UseCRLF should be false")
	}
}

func TestParseWithOptions_CustomDelimiter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		comma    rune
		wantRows int
		wantCols int
	}{
		{
			name:     "tab separated",
			input:    "name\tage\nAlice\t30",
			comma:    '\t',
			wantRows: 2,
			wantCols: 2,
		},
		{
			name:     "semicolon separated",
			input:    "a;b;c\n1;2;3",
			comma:    ';',
			wantRows: 2,
			wantCols: 3,
		},
		{
			name:     "pipe separated",
			input:    "x|y|z",
			comma:    '|',
			wantRows: 1,
			wantCols: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := csv.DefaultReaderOptions()
			opts.Comma = tt.comma

			node, err := csv.ParseWithOptions(tt.input, opts)
			if err != nil {
				t.Fatalf("ParseWithOptions() error = %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("ParseWithOptions() returned %T, want *ast.ArrayDataNode", node)
			}

			if arr.Len() != tt.wantRows {
				t.Errorf("got %d rows, want %d", arr.Len(), tt.wantRows)
			}

			if arr.Len() > 0 {
				firstRow, ok := arr.Elements()[0].(*ast.ArrayDataNode)
				if !ok {
					t.Fatalf("first row is %T, want *ast.ArrayDataNode", arr.Elements()[0])
				}
				if firstRow.Len() != tt.wantCols {
					t.Errorf("first row has %d cols, want %d", firstRow.Len(), tt.wantCols)
				}
			}
		})
	}
}

func TestParseWithOptions_Comment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		comment  rune
		wantRows int
	}{
		{
			name:     "hash comment",
			input:    "name,age\n# this is a comment\nAlice,30",
			comment:  '#',
			wantRows: 2, // header + 1 data row, comment skipped
		},
		{
			name:     "semicolon comment",
			input:    "a,b\n; comment\n1,2",
			comment:  ';',
			wantRows: 2,
		},
		{
			name:     "no comment char set",
			input:    "a,b\n# not a comment\n1,2",
			comment:  0,
			wantRows: 3, // # is treated as data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := csv.DefaultReaderOptions()
			opts.Comment = tt.comment

			node, err := csv.ParseWithOptions(tt.input, opts)
			if err != nil {
				t.Fatalf("ParseWithOptions() error = %v", err)
			}

			arr, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Fatalf("ParseWithOptions() returned %T, want *ast.ArrayDataNode", node)
			}

			if arr.Len() != tt.wantRows {
				t.Errorf("got %d rows, want %d", arr.Len(), tt.wantRows)
			}
		})
	}
}

func TestParseWithOptions_FieldsPerRecord(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		fieldsPerRecord int
		wantErr         bool
	}{
		{
			name:            "negative allows variable fields",
			input:           "a,b,c\n1,2\n3,4,5,6",
			fieldsPerRecord: -1,
			wantErr:         false,
		},
		{
			name:            "zero uses first record count",
			input:           "a,b,c\n1,2,3\n4,5,6",
			fieldsPerRecord: 0,
			wantErr:         false,
		},
		{
			name:            "zero rejects mismatched count",
			input:           "a,b,c\n1,2\n4,5,6",
			fieldsPerRecord: 0,
			wantErr:         true,
		},
		{
			name:            "positive enforces exact count",
			input:           "a,b\n1,2",
			fieldsPerRecord: 2,
			wantErr:         false,
		},
		{
			name:            "positive rejects wrong count",
			input:           "a,b,c\n1,2,3",
			fieldsPerRecord: 2,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := csv.DefaultReaderOptions()
			opts.FieldsPerRecord = tt.fieldsPerRecord

			_, err := csv.ParseWithOptions(tt.input, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWithOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseWithOptions_TrimLeadingSpace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		trim     bool
		wantVal  string
	}{
		{
			name:    "no trim",
			input:   "  value",
			trim:    false,
			wantVal: "  value",
		},
		{
			name:    "trim enabled",
			input:   "  value",
			trim:    true,
			wantVal: "value",
		},
		{
			name:    "trim with tabs",
			input:   "\t\tvalue",
			trim:    true,
			wantVal: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := csv.DefaultReaderOptions()
			opts.TrimLeadingSpace = tt.trim

			node, err := csv.ParseWithOptions(tt.input, opts)
			if err != nil {
				t.Fatalf("ParseWithOptions() error = %v", err)
			}

			arr := node.(*ast.ArrayDataNode)
			row := arr.Elements()[0].(*ast.ArrayDataNode)
			field := row.Elements()[0].(*ast.LiteralNode)
			got := field.Value().(string)

			if got != tt.wantVal {
				t.Errorf("got %q, want %q", got, tt.wantVal)
			}
		})
	}
}

func TestParseWithOptions_LazyQuotes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		lazy    bool
		wantErr bool
	}{
		{
			name:    "strict rejects quote in field",
			input:   `a"b,c`,
			lazy:    false,
			wantErr: true,
		},
		{
			name:    "lazy allows quote in field",
			input:   `a"b,c`,
			lazy:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := csv.DefaultReaderOptions()
			opts.LazyQuotes = tt.lazy

			_, err := csv.ParseWithOptions(tt.input, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWithOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenderWithOptions_CustomDelimiter(t *testing.T) {
	// Create a simple AST
	field1 := ast.NewLiteralNode("a", ast.ZeroPosition())
	field2 := ast.NewLiteralNode("b", ast.ZeroPosition())
	row := ast.NewArrayDataNode([]ast.SchemaNode{field1, field2}, ast.ZeroPosition())
	file := ast.NewArrayDataNode([]ast.SchemaNode{row}, ast.ZeroPosition())

	tests := []struct {
		name      string
		comma     rune
		wantDelim string
	}{
		{
			name:      "tab separated",
			comma:     '\t',
			wantDelim: "\t",
		},
		{
			name:      "semicolon separated",
			comma:     ';',
			wantDelim: ";",
		},
		{
			name:      "pipe separated",
			comma:     '|',
			wantDelim: "|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := csv.DefaultWriterOptions()
			opts.Comma = tt.comma

			out, err := csv.RenderWithOptions(file, opts)
			if err != nil {
				t.Fatalf("RenderWithOptions() error = %v", err)
			}

			if !strings.Contains(string(out), tt.wantDelim) {
				t.Errorf("output %q doesn't contain delimiter %q", string(out), tt.wantDelim)
			}
		})
	}
}

func TestRenderWithOptions_UseCRLF(t *testing.T) {
	// Create a two-row AST
	field1 := ast.NewLiteralNode("a", ast.ZeroPosition())
	field2 := ast.NewLiteralNode("b", ast.ZeroPosition())
	row1 := ast.NewArrayDataNode([]ast.SchemaNode{field1}, ast.ZeroPosition())
	row2 := ast.NewArrayDataNode([]ast.SchemaNode{field2}, ast.ZeroPosition())
	file := ast.NewArrayDataNode([]ast.SchemaNode{row1, row2}, ast.ZeroPosition())

	tests := []struct {
		name        string
		useCRLF     bool
		wantEnding  string
	}{
		{
			name:       "LF endings",
			useCRLF:    false,
			wantEnding: "a\nb\n",
		},
		{
			name:       "CRLF endings",
			useCRLF:    true,
			wantEnding: "a\r\nb\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := csv.DefaultWriterOptions()
			opts.UseCRLF = tt.useCRLF

			out, err := csv.RenderWithOptions(file, opts)
			if err != nil {
				t.Fatalf("RenderWithOptions() error = %v", err)
			}

			if string(out) != tt.wantEnding {
				t.Errorf("got %q, want %q", string(out), tt.wantEnding)
			}
		})
	}
}

func TestOptionsValidation(t *testing.T) {
	t.Run("reader options validation", func(t *testing.T) {
		tests := []struct {
			name    string
			opts    csv.ReaderOptions
			wantErr bool
		}{
			{
				name:    "valid default",
				opts:    csv.DefaultReaderOptions(),
				wantErr: false,
			},
			{
				name:    "invalid comma - newline",
				opts:    csv.ReaderOptions{Comma: '\n'},
				wantErr: true,
			},
			{
				name:    "invalid comma - quote",
				opts:    csv.ReaderOptions{Comma: '"'},
				wantErr: true,
			},
			{
				name:    "comment same as comma",
				opts:    csv.ReaderOptions{Comma: ',', Comment: ','},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.opts.Validate()
				if (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("writer options validation", func(t *testing.T) {
		tests := []struct {
			name    string
			opts    csv.WriterOptions
			wantErr bool
		}{
			{
				name:    "valid default",
				opts:    csv.DefaultWriterOptions(),
				wantErr: false,
			},
			{
				name:    "invalid comma - newline",
				opts:    csv.WriterOptions{Comma: '\n'},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.opts.Validate()
				if (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})
}

func TestParseReaderWithOptions(t *testing.T) {
	input := "name\tage\nAlice\t30"
	reader := strings.NewReader(input)

	opts := csv.DefaultReaderOptions()
	opts.Comma = '\t'

	node, err := csv.ParseReaderWithOptions(reader, opts)
	if err != nil {
		t.Fatalf("ParseReaderWithOptions() error = %v", err)
	}

	arr, ok := node.(*ast.ArrayDataNode)
	if !ok {
		t.Fatalf("ParseReaderWithOptions() returned %T, want *ast.ArrayDataNode", node)
	}

	if arr.Len() != 2 {
		t.Errorf("got %d rows, want 2", arr.Len())
	}
}

func TestValidateWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		opts    csv.ReaderOptions
		wantErr bool
	}{
		{
			name:  "valid with custom delimiter",
			input: "a\tb\tc",
			opts: csv.ReaderOptions{
				Comma:           '\t',
				FieldsPerRecord: -1,
			},
			wantErr: false,
		},
		{
			name:  "invalid - field count mismatch",
			input: "a,b,c\n1,2",
			opts: csv.ReaderOptions{
				Comma:           ',',
				FieldsPerRecord: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := csv.ValidateWithOptions(tt.input, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWithOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReaderPositionTracking(t *testing.T) {
	reader := csv.NewReader(csv.DefaultReaderOptions())

	// Initial state
	line, col := reader.FieldPos(0)
	if line != 1 || col != 1 {
		t.Errorf("FieldPos(0) = (%d, %d), want (1, 1)", line, col)
	}

	if reader.InputOffset() != 0 {
		t.Errorf("InputOffset() = %d, want 0", reader.InputOffset())
	}

	// Update position
	reader.SetOffset(5, 10, 100)

	line, col = reader.FieldPos(0)
	if line != 5 || col != 10 {
		t.Errorf("FieldPos(0) after SetOffset = (%d, %d), want (5, 10)", line, col)
	}

	if reader.InputOffset() != 100 {
		t.Errorf("InputOffset() after SetOffset = %d, want 100", reader.InputOffset())
	}
}
