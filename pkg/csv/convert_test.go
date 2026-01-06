package csv

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

// TestNodeToInterface tests converting AST nodes to [][]string
func TestNodeToInterface(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    [][]string
		wantErr bool
	}{
		{
			name:  "simple CSV",
			input: "name,age\nAlice,30\nBob,25\n",
			want: [][]string{
				{"name", "age"},
				{"Alice", "30"},
				{"Bob", "25"},
			},
		},
		{
			name:  "empty CSV",
			input: "",
			want:  [][]string{},
		},
		{
			name:  "single record",
			input: "name,age\n",
			want: [][]string{
				{"name", "age"},
			},
		},
		{
			name:  "with empty fields",
			input: "a,b,c\n1,,3\n,,\n",
			want: [][]string{
				{"a", "b", "c"},
				{"1", "", "3"},
				{"", "", ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse CSV to AST
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Convert AST to interface
			got := NodeToInterface(node)

			// Check type
			gotSlice, ok := got.([][]string)
			if !ok {
				t.Fatalf("NodeToInterface() returned %T, want [][]string", got)
			}

			// Compare
			if len(gotSlice) != len(tt.want) {
				t.Errorf("NodeToInterface() got %d records, want %d", len(gotSlice), len(tt.want))
				return
			}

			for i := range gotSlice {
				if len(gotSlice[i]) != len(tt.want[i]) {
					t.Errorf("NodeToInterface() record %d has %d fields, want %d", i, len(gotSlice[i]), len(tt.want[i]))
					continue
				}
				for j := range gotSlice[i] {
					if gotSlice[i][j] != tt.want[i][j] {
						t.Errorf("NodeToInterface() record %d field %d = %q, want %q", i, j, gotSlice[i][j], tt.want[i][j])
					}
				}
			}
		})
	}
}

// TestInterfaceToNode tests converting [][]string to AST nodes
func TestInterfaceToNode(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name: "simple data",
			input: [][]string{
				{"name", "age"},
				{"Alice", "30"},
				{"Bob", "25"},
			},
		},
		{
			name:  "empty data",
			input: [][]string{},
		},
		{
			name: "single record",
			input: [][]string{
				{"name", "age"},
			},
		},
		{
			name: "with empty fields",
			input: [][]string{
				{"a", "b", "c"},
				{"1", "", "3"},
				{"", "", ""},
			},
		},
		{
			name:    "invalid type - not slice",
			input:   map[string]string{"key": "value"},
			wantErr: true,
		},
		{
			name:    "invalid type - slice of wrong type",
			input:   []int{1, 2, 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to AST
			node, err := InterfaceToNode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("InterfaceToNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Check that we got an ArrayDataNode
			arrayNode, ok := node.(*ast.ArrayDataNode)
			if !ok {
				t.Errorf("InterfaceToNode() returned %T, want *ast.ArrayDataNode", node)
				return
			}

			// Convert back to interface and compare
			got := NodeToInterface(arrayNode)
			gotSlice, ok := got.([][]string)
			if !ok {
				t.Fatalf("NodeToInterface() returned %T, want [][]string", got)
			}

			wantSlice := tt.input.([][]string)
			if len(gotSlice) != len(wantSlice) {
				t.Errorf("Round trip: got %d records, want %d", len(gotSlice), len(wantSlice))
				return
			}

			for i := range gotSlice {
				if len(gotSlice[i]) != len(wantSlice[i]) {
					t.Errorf("Round trip: record %d has %d fields, want %d", i, len(gotSlice[i]), len(wantSlice[i]))
					continue
				}
				for j := range gotSlice[i] {
					if gotSlice[i][j] != wantSlice[i][j] {
						t.Errorf("Round trip: record %d field %d = %q, want %q", i, j, gotSlice[i][j], wantSlice[i][j])
					}
				}
			}
		})
	}
}

// TestConvertRoundTrip tests that NodeToInterface and InterfaceToNode are inverse operations
func TestConvertRoundTrip(t *testing.T) {
	original := [][]string{
		{"name", "age", "city"},
		{"Alice", "30", "NYC"},
		{"Bob", "25", "LA"},
		{"Charlie", "35", "SF"},
	}

	// Convert to AST
	node, err := InterfaceToNode(original)
	if err != nil {
		t.Fatalf("InterfaceToNode() error = %v", err)
	}

	// Convert back to interface
	result := NodeToInterface(node)
	resultSlice, ok := result.([][]string)
	if !ok {
		t.Fatalf("NodeToInterface() returned %T, want [][]string", result)
	}

	// Compare
	if len(resultSlice) != len(original) {
		t.Fatalf("Round trip: got %d records, want %d", len(resultSlice), len(original))
	}

	for i := range resultSlice {
		if len(resultSlice[i]) != len(original[i]) {
			t.Errorf("Round trip: record %d has %d fields, want %d", i, len(resultSlice[i]), len(original[i]))
			continue
		}
		for j := range resultSlice[i] {
			if resultSlice[i][j] != original[i][j] {
				t.Errorf("Round trip: record %d field %d = %q, want %q", i, j, resultSlice[i][j], original[i][j])
			}
		}
	}
}

// TestNodeToRecords tests the convenience wrapper for getting [][]string
func TestNodeToRecords(t *testing.T) {
	t.Run("multiple records", func(t *testing.T) {
		node, err := Parse("name,age\nAlice,30\nBob,25\n")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		records := NodeToRecords(node)
		if len(records) != 3 {
			t.Errorf("NodeToRecords() got %d records, want 3", len(records))
		}
		if records[0][0] != "name" {
			t.Errorf("NodeToRecords() first field = %q, want 'name'", records[0][0])
		}
	})

	t.Run("empty", func(t *testing.T) {
		node, err := Parse("")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		records := NodeToRecords(node)
		if len(records) != 0 {
			t.Errorf("NodeToRecords() got %d records, want 0", len(records))
		}
	})

	t.Run("nil node", func(t *testing.T) {
		records := NodeToRecords(nil)
		if len(records) != 0 {
			t.Errorf("NodeToRecords(nil) got %d records, want 0", len(records))
		}
	})
}

// TestRecordsToNode tests the convenience wrapper for creating nodes from [][]string
func TestRecordsToNode(t *testing.T) {
	t.Run("multiple records", func(t *testing.T) {
		records := [][]string{
			{"name", "age"},
			{"Alice", "30"},
		}

		node, err := RecordsToNode(records)
		if err != nil {
			t.Fatalf("RecordsToNode() error = %v", err)
		}

		// Verify round-trip
		result := NodeToRecords(node)
		if len(result) != 2 {
			t.Errorf("Round trip: got %d records, want 2", len(result))
		}
	})

	t.Run("empty records", func(t *testing.T) {
		records := [][]string{}

		node, err := RecordsToNode(records)
		if err != nil {
			t.Fatalf("RecordsToNode() error = %v", err)
		}

		result := NodeToRecords(node)
		if len(result) != 0 {
			t.Errorf("Round trip: got %d records, want 0", len(result))
		}
	})
}

// TestInterfaceToNodeEdgeCases tests edge cases for InterfaceToNode
func TestInterfaceToNodeEdgeCases(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		node, err := InterfaceToNode(nil)
		if err != nil {
			t.Fatalf("InterfaceToNode(nil) error = %v", err)
		}
		// Should return a LiteralNode with empty string
		if node == nil {
			t.Error("InterfaceToNode(nil) should return a node")
		}
	})

	t.Run("single string", func(t *testing.T) {
		node, err := InterfaceToNode("test")
		if err != nil {
			t.Fatalf("InterfaceToNode(string) error = %v", err)
		}
		// Should return a LiteralNode
		litNode, ok := node.(*ast.LiteralNode)
		if !ok {
			t.Errorf("InterfaceToNode(string) returned %T, want *ast.LiteralNode", node)
		} else if litNode.Value() != "test" {
			t.Errorf("LiteralNode value = %v, want 'test'", litNode.Value())
		}
	})

	t.Run("single string slice - record", func(t *testing.T) {
		node, err := InterfaceToNode([]string{"a", "b", "c"})
		if err != nil {
			t.Fatalf("InterfaceToNode([]string) error = %v", err)
		}
		// Should return an ArrayDataNode representing a record
		arrayNode, ok := node.(*ast.ArrayDataNode)
		if !ok {
			t.Errorf("InterfaceToNode([]string) returned %T, want *ast.ArrayDataNode", node)
		} else if arrayNode.Len() != 3 {
			t.Errorf("ArrayDataNode length = %d, want 3", arrayNode.Len())
		}
	})

	t.Run("interface slice with strings", func(t *testing.T) {
		input := []interface{}{"a", "b", "c"}
		node, err := InterfaceToNode(input)
		if err != nil {
			t.Fatalf("InterfaceToNode([]interface{} of strings) error = %v", err)
		}
		arrayNode, ok := node.(*ast.ArrayDataNode)
		if !ok {
			t.Errorf("InterfaceToNode returned %T, want *ast.ArrayDataNode", node)
		} else if arrayNode.Len() != 3 {
			t.Errorf("ArrayDataNode length = %d, want 3", arrayNode.Len())
		}
	})

	t.Run("interface slice with []string", func(t *testing.T) {
		input := []interface{}{
			[]string{"a", "b"},
			[]string{"c", "d"},
		}
		node, err := InterfaceToNode(input)
		if err != nil {
			t.Fatalf("InterfaceToNode([]interface{} of []string) error = %v", err)
		}
		arrayNode, ok := node.(*ast.ArrayDataNode)
		if !ok {
			t.Errorf("InterfaceToNode returned %T, want *ast.ArrayDataNode", node)
		} else if arrayNode.Len() != 2 {
			t.Errorf("ArrayDataNode length = %d, want 2", arrayNode.Len())
		}
	})

	t.Run("interface slice with []interface{}", func(t *testing.T) {
		input := []interface{}{
			[]interface{}{"a", "b"},
			[]interface{}{"c", "d"},
		}
		node, err := InterfaceToNode(input)
		if err != nil {
			t.Fatalf("InterfaceToNode(nested []interface{}) error = %v", err)
		}
		arrayNode, ok := node.(*ast.ArrayDataNode)
		if !ok {
			t.Errorf("InterfaceToNode returned %T, want *ast.ArrayDataNode", node)
		} else if arrayNode.Len() != 2 {
			t.Errorf("ArrayDataNode length = %d, want 2", arrayNode.Len())
		}
	})

	t.Run("empty interface slice", func(t *testing.T) {
		input := []interface{}{}
		node, err := InterfaceToNode(input)
		if err != nil {
			t.Fatalf("InterfaceToNode(empty []interface{}) error = %v", err)
		}
		arrayNode, ok := node.(*ast.ArrayDataNode)
		if !ok {
			t.Errorf("InterfaceToNode returned %T, want *ast.ArrayDataNode", node)
		} else if arrayNode.Len() != 0 {
			t.Errorf("ArrayDataNode length = %d, want 0", arrayNode.Len())
		}
	})

	t.Run("interface slice with int - unsupported", func(t *testing.T) {
		input := []interface{}{1, 2, 3}
		_, err := InterfaceToNode(input)
		if err == nil {
			t.Error("InterfaceToNode([]interface{} of ints) should return error")
		}
	})

	t.Run("interface slice with non-string in string slice", func(t *testing.T) {
		input := []interface{}{123, 456}
		node, err := InterfaceToNode(input)
		// Should convert non-strings to string representation
		if err == nil {
			t.Error("InterfaceToNode([]interface{} of ints) should return error for non-string types")
		}
		_ = node
	})
}

// TestNodeToInterfaceEdgeCases tests edge cases for NodeToInterface
func TestNodeToInterfaceEdgeCases(t *testing.T) {
	t.Run("literal node", func(t *testing.T) {
		litNode := ast.NewLiteralNode("test", ast.ZeroPosition())
		result := NodeToInterface(litNode)
		if str, ok := result.(string); !ok || str != "test" {
			t.Errorf("NodeToInterface(LiteralNode) = %v (%T), want 'test' (string)", result, result)
		}
	})

	t.Run("literal node with nil value", func(t *testing.T) {
		litNode := ast.NewLiteralNode(nil, ast.ZeroPosition())
		result := NodeToInterface(litNode)
		// When literal node has nil value, it returns "<nil>" string representation
		if str, ok := result.(string); !ok || str != "<nil>" {
			t.Errorf("NodeToInterface(LiteralNode with nil) = %v (%T), want '<nil>' (string)", result, result)
		}
	})

	t.Run("literal node with non-string value", func(t *testing.T) {
		litNode := ast.NewLiteralNode(123, ast.ZeroPosition())
		result := NodeToInterface(litNode)
		if str, ok := result.(string); !ok || str != "123" {
			t.Errorf("NodeToInterface(LiteralNode with int) = %v (%T), want '123' (string)", result, result)
		}
	})

	t.Run("empty array node", func(t *testing.T) {
		arrayNode := ast.NewArrayDataNode([]ast.SchemaNode{}, ast.ZeroPosition())
		result := NodeToInterface(arrayNode)
		if slice, ok := result.([][]string); !ok || len(slice) != 0 {
			t.Errorf("NodeToInterface(empty ArrayDataNode) = %v (%T), want empty [][]string", result, result)
		}
	})

	t.Run("nil node", func(t *testing.T) {
		result := NodeToInterface(nil)
		if result != nil {
			t.Errorf("NodeToInterface(nil) = %v, want nil", result)
		}
	})
}
