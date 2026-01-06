package csv

import (
	"strings"
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

// TestRender tests the basic Render function
func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple CSV",
			input: "name,age\nAlice,30\nBob,25\n",
			want:  "name,age\nAlice,30\nBob,25\n",
		},
		{
			name:  "empty CSV",
			input: "",
			want:  "",
		},
		{
			name:  "single record",
			input: "name,age\n",
			want:  "name,age\n",
		},
		{
			name:  "with empty fields",
			input: "a,b,c\n1,,3\n,,\n",
			want:  "a,b,c\n1,,3\n,,\n",
		},
		{
			name:  "preserves whitespace in unquoted fields",
			input: "a,b,c\n1,2,3\n",
			want:  "a,b,c\n1,2,3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse CSV to AST
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Render AST back to CSV
			got, err := Render(node)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("Render() mismatch:\ngot:\n%q\nwant:\n%q", gotStr, tt.want)
			}
		})
	}
}

// TestRenderEscaping tests Render with fields that need escaping
func TestRenderEscaping(t *testing.T) {
	tests := []struct {
		name  string
		input [][]string
		want  string
	}{
		{
			name: "fields with commas",
			input: [][]string{
				{"name", "description"},
				{"Item1", "Has, comma"},
				{"Item2", "Normal"},
			},
			want: "name,description\nItem1,\"Has, comma\"\nItem2,Normal\n",
		},
		{
			name: "fields with quotes",
			input: [][]string{
				{"name", "description"},
				{"Item1", `Has "quotes"`},
				{"Item2", "Normal"},
			},
			want: "name,description\nItem1,\"Has \"\"quotes\"\"\"\nItem2,Normal\n",
		},
		{
			name: "fields with newlines",
			input: [][]string{
				{"name", "description"},
				{"Item1", "Has\nnewline"},
				{"Item2", "Normal"},
			},
			want: "name,description\nItem1,\"Has\nnewline\"\nItem2,Normal\n",
		},
		{
			name: "fields with carriage returns",
			input: [][]string{
				{"name", "description"},
				{"Item1", "Has\rCR"},
				{"Item2", "Normal"},
			},
			want: "name,description\nItem1,\"Has\rCR\"\nItem2,Normal\n",
		},
		{
			name: "fields with multiple special characters",
			input: [][]string{
				{"name", "description"},
				{"Item1", "Has, comma and \"quotes\" and\nnewline"},
			},
			want: "name,description\nItem1,\"Has, comma and \"\"quotes\"\" and\nnewline\"\n",
		},
		{
			name: "empty fields",
			input: [][]string{
				{"a", "b", "c"},
				{"1", "", "3"},
				{"", "", ""},
			},
			want: "a,b,c\n1,,3\n,,\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to AST
			node, err := InterfaceToNode(tt.input)
			if err != nil {
				t.Fatalf("InterfaceToNode() error = %v", err)
			}

			// Render AST to CSV
			got, err := Render(node)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("Render() mismatch:\ngot:\n%q\nwant:\n%q", gotStr, tt.want)
			}
		})
	}
}

// TestRenderRoundTrip tests that Parse and Render are inverse operations
func TestRenderRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple CSV",
			input: "name,age,city\nAlice,30,NYC\nBob,25,LA\nCharlie,35,SF\n",
		},
		{
			name:  "CSV with quoted fields",
			input: "name,description\nItem1,\"Has, comma\"\nItem2,\"Has \"\"quotes\"\"\"\nItem3,\"Has\nnewline\"\n",
		},
		{
			name:  "CSV with empty fields",
			input: "a,b,c\n1,,3\n,2,\n,,\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse CSV to AST
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Render AST back to CSV
			rendered, err := Render(node)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			renderedStr := string(rendered)
			if renderedStr != tt.input {
				t.Errorf("Round trip mismatch:\ngot:\n%q\nwant:\n%q", renderedStr, tt.input)
			}
		})
	}
}

// TestRenderNilNode tests Render with nil node
func TestRenderNilNode(t *testing.T) {
	got, err := Render(nil)
	if err != nil {
		t.Fatalf("Render(nil) error = %v", err)
	}

	gotStr := string(got)
	if gotStr != "" {
		t.Errorf("Render(nil) = %q, want empty string", gotStr)
	}
}

// TestRenderInvalidNode tests Render with unexpected node types
func TestRenderInvalidNode(t *testing.T) {
	// Create an ObjectNode (which is not expected for CSV)
	pos := ast.Position{}
	objectNode := ast.NewObjectNode(map[string]ast.SchemaNode{
		"key": ast.NewLiteralNode("value", pos),
	}, pos)

	_, err := Render(objectNode)
	if err == nil {
		t.Error("Render(ObjectNode) should return error, got nil")
	}
}

// TestRenderWhitespace tests that Render preserves field content exactly
func TestRenderWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input [][]string
		want  string
	}{
		{
			name: "leading and trailing spaces",
			input: [][]string{
				{"name", "value"},
				{" space ", "  test  "},
			},
			want: "name,value\n space ,  test  \n",
		},
		{
			name: "tabs",
			input: [][]string{
				{"name", "value"},
				{"tab\there", "normal"},
			},
			want: "name,value\ntab\there,normal\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to AST
			node, err := InterfaceToNode(tt.input)
			if err != nil {
				t.Fatalf("InterfaceToNode() error = %v", err)
			}

			// Render AST to CSV
			got, err := Render(node)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("Render() mismatch:\ngot:\n%q\nwant:\n%q", gotStr, tt.want)
			}
		})
	}
}

// BenchmarkRender benchmarks the Render function
func BenchmarkRender(b *testing.B) {
	// Create a CSV with 100 records of 5 fields each
	var sb strings.Builder
	sb.WriteString("col1,col2,col3,col4,col5\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("value1,value2,value3,value4,value5\n")
	}
	csvData := sb.String()

	// Parse once
	node, err := Parse(csvData)
	if err != nil {
		b.Fatalf("Parse() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Render(node)
		if err != nil {
			b.Fatalf("Render() error = %v", err)
		}
	}
}
