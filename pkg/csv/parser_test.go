package csv_test

import (
	"strings"
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-csv/pkg/csv"
)

// TestParse tests the Parse function (5.1.1-5.1.2)
func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "simple csv",
			input:   "name,age\nAlice,30\nBob,25",
			wantErr: false,
		},
		{
			name:    "quoted fields",
			input:   `"name","age"\n"Alice","30"`,
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: false,
		},
		{
			name:    "single field",
			input:   "value",
			wantErr: false,
		},
		{
			name:    "unclosed quote",
			input:   `"unclosed`,
			wantErr: true,
		},
		{
			name:    "escaped quotes",
			input:   `"field with ""quotes"" inside"`,
			wantErr: false,
		},
		{
			name:    "empty fields",
			input:   "a,,c\n,b,",
			wantErr: false,
		},
		{
			name:    "newlines in quoted fields",
			input:   "\"field\nwith\nnewlines\",normal",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := csv.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if node == nil {
					t.Error("Parse() returned nil node for valid input")
					return
				}

				// Verify node is an ArrayDataNode
				arrayNode, ok := node.(*ast.ArrayDataNode)
				if !ok {
					t.Errorf("Parse() returned %T, expected *ast.ArrayDataNode", node)
					return
				}

				// For non-empty input, verify we got records
				if tt.input != "" && arrayNode.Len() == 0 {
					t.Error("Parse() returned empty array for non-empty input")
				}
			}
		})
	}
}

// TestParseReader tests the ParseReader function (5.1.3-5.1.4)
func TestParseReader(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "simple csv from reader",
			input:   "name,age\nAlice,30\nBob,25",
			wantErr: false,
		},
		{
			name:    "large csv from reader",
			input:   generateLargeCSV(1000), // Test streaming with 1000 rows
			wantErr: false,
		},
		{
			name:    "empty reader",
			input:   "",
			wantErr: false,
		},
		{
			name:    "quoted fields from reader",
			input:   `"name","age"\n"Alice","30"`,
			wantErr: false,
		},
		{
			name:    "unclosed quote from reader",
			input:   `"unclosed`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			node, err := csv.ParseReader(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if node == nil {
					t.Error("ParseReader() returned nil node for valid input")
					return
				}

				// Verify node is an ArrayDataNode
				arrayNode, ok := node.(*ast.ArrayDataNode)
				if !ok {
					t.Errorf("ParseReader() returned %T, expected *ast.ArrayDataNode", node)
					return
				}

				// For non-empty input, verify we got records
				if tt.input != "" && arrayNode.Len() == 0 {
					t.Error("ParseReader() returned empty array for non-empty input")
				}
			}
		})
	}
}

// TestValidate tests the Validate function (5.1.5-5.1.6)
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid simple csv",
			input:   "name,age\nAlice,30",
			wantErr: false,
		},
		{
			name:    "valid quoted fields",
			input:   `"name","age"`,
			wantErr: false,
		},
		{
			name:    "valid empty input",
			input:   "",
			wantErr: false,
		},
		{
			name:    "valid escaped quotes",
			input:   `"field with ""quotes"""`,
			wantErr: false,
		},
		{
			name:    "invalid unclosed quote",
			input:   `"unclosed`,
			wantErr: true,
		},
		{
			name:    "invalid quote in unquoted field",
			input:   `field"with"quote`,
			wantErr: true,
		},
		{
			name:    "valid multiline quoted field",
			input:   "\"field\nwith\nnewlines\"",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := csv.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateReader tests the ValidateReader function (5.1.7-5.1.8)
func TestValidateReader(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid csv from reader",
			input:   "name,age\nAlice,30",
			wantErr: false,
		},
		{
			name:    "valid quoted from reader",
			input:   `"name","age"`,
			wantErr: false,
		},
		{
			name:    "valid empty reader",
			input:   "",
			wantErr: false,
		},
		{
			name:    "invalid unclosed quote from reader",
			input:   `"unclosed`,
			wantErr: true,
		},
		{
			name:    "large valid csv from reader",
			input:   generateLargeCSV(500),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			err := csv.ValidateReader(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateReader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFormat tests the Format function
func TestFormat(t *testing.T) {
	format := csv.Format()
	if format != "CSV" {
		t.Errorf("Format() = %q, want %q", format, "CSV")
	}
}

// Helper function to generate large CSV for testing
func generateLargeCSV(rows int) string {
	var sb strings.Builder
	sb.WriteString("id,name,age,email\n")
	for i := 0; i < rows; i++ {
		sb.WriteString("1,Alice,30,alice@example.com\n")
	}
	return sb.String()
}
