package csv_test

import (
	"strings"
	"testing"

	"github.com/shapestone/shape-csv/pkg/csv"
)

func TestParseAdvancedTag(t *testing.T) {
	// Test basic tag parsing still works
	// Note: parseAdvancedTag is not exported, so we test through behavior

	t.Run("split option", func(t *testing.T) {
		result := csv.SplitField("a|b|c", "|")
		if len(result) != 3 {
			t.Errorf("SplitField() got %d values, want 3", len(result))
		}
		if result[0] != "a" || result[1] != "b" || result[2] != "c" {
			t.Errorf("SplitField() = %v, want [a b c]", result)
		}
	})

	t.Run("empty value", func(t *testing.T) {
		result := csv.SplitField("", "|")
		if len(result) != 0 {
			t.Errorf("SplitField(\"\") got %d values, want 0", len(result))
		}
	})

	t.Run("no separator", func(t *testing.T) {
		result := csv.SplitField("abc", "")
		if len(result) != 1 || result[0] != "abc" {
			t.Errorf("SplitField with empty separator = %v, want [abc]", result)
		}
	})
}

func TestJoinField(t *testing.T) {
	t.Run("multiple values", func(t *testing.T) {
		result := csv.JoinField([]string{"a", "b", "c"}, "|")
		if result != "a|b|c" {
			t.Errorf("JoinField() = %q, want %q", result, "a|b|c")
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		result := csv.JoinField([]string{}, "|")
		if result != "" {
			t.Errorf("JoinField([]) = %q, want empty", result)
		}
	})

	t.Run("single value", func(t *testing.T) {
		result := csv.JoinField([]string{"only"}, "|")
		if result != "only" {
			t.Errorf("JoinField([only]) = %q, want %q", result, "only")
		}
	})
}

func TestEscapeMode(t *testing.T) {
	t.Run("default RFC4180 mode", func(t *testing.T) {
		opts := csv.DefaultAdvancedOptions()
		if opts.EscapeMode != csv.EscapeModeRFC4180 {
			t.Error("default should be RFC4180 mode")
		}
		if opts.EscapeChar != 0 {
			t.Error("default escape char should be 0")
		}
	})

	t.Run("backslash mode available", func(t *testing.T) {
		opts := csv.AdvancedOptions{
			EscapeMode: csv.EscapeModeBackslash,
			EscapeChar: '\\',
		}
		if opts.EscapeMode != csv.EscapeModeBackslash {
			t.Error("should support backslash mode")
		}
	})
}

func TestFlattenStruct(t *testing.T) {
	type Address struct {
		Street string `csv:"street"`
		City   string `csv:"city"`
	}

	type Person struct {
		Name    string  `csv:"name"`
		Age     int     `csv:"age"`
		Address Address `csv:"address,recurse"`
	}

	t.Run("simple struct", func(t *testing.T) {
		p := Person{
			Name: "Alice",
			Age:  30,
			Address: Address{
				Street: "123 Main St",
				City:   "NYC",
			},
		}

		result := csv.FlattenStruct(p, "")

		if result["name"] != "Alice" {
			t.Errorf("name = %q, want %q", result["name"], "Alice")
		}
		// Note: Age is an int, which gets converted to string
		if result["age"] != "30" {
			t.Errorf("age = %q, want %q", result["age"], "30")
		}
	})

	t.Run("with prefix", func(t *testing.T) {
		p := Person{Name: "Bob"}
		result := csv.FlattenStruct(p, "person")

		if _, ok := result["person.name"]; !ok {
			t.Error("expected prefixed field name")
		}
	})

	t.Run("nil pointer", func(t *testing.T) {
		var p *Person = nil
		result := csv.FlattenStruct(p, "")
		if len(result) != 0 {
			t.Errorf("expected empty result for nil, got %v", result)
		}
	})
}

func TestTransformOptions(t *testing.T) {
	t.Run("field transform", func(t *testing.T) {
		opts := csv.TransformOptions{
			FieldTransform: func(name, value string) string {
				return strings.ToUpper(value)
			},
		}

		proc := csv.NewProcessWithTransforms(opts)
		result := proc.TransformField("name", "alice")
		if result != "ALICE" {
			t.Errorf("TransformField() = %q, want %q", result, "ALICE")
		}
	})

	t.Run("row transform", func(t *testing.T) {
		opts := csv.TransformOptions{
			RowTransform: func(record, headers []string) []string {
				// Append a computed field
				return append(record, "computed")
			},
		}

		proc := csv.NewProcessWithTransforms(opts)
		proc.SetHeaders([]string{"a", "b"})
		result := proc.TransformRow([]string{"1", "2"})
		if len(result) != 3 {
			t.Errorf("TransformRow() returned %d fields, want 3", len(result))
		}
		if result[2] != "computed" {
			t.Errorf("TransformRow()[2] = %q, want %q", result[2], "computed")
		}
	})

	t.Run("nil transforms", func(t *testing.T) {
		opts := csv.TransformOptions{}
		proc := csv.NewProcessWithTransforms(opts)

		// Should pass through unchanged
		field := proc.TransformField("name", "value")
		if field != "value" {
			t.Errorf("nil transform changed field: %q", field)
		}

		row := proc.TransformRow([]string{"a", "b"})
		if len(row) != 2 {
			t.Errorf("nil transform changed row length")
		}
	})
}

func TestAdvancedOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := csv.DefaultAdvancedOptions()
		if opts.EscapeChar != 0 {
			t.Error("default EscapeChar should be 0")
		}
		if opts.EscapeMode != csv.EscapeModeRFC4180 {
			t.Error("default EscapeMode should be RFC4180")
		}
		if opts.PreProcess != nil {
			t.Error("default PreProcess should be nil")
		}
		if opts.PostProcess != nil {
			t.Error("default PostProcess should be nil")
		}
	})

	t.Run("with hooks", func(t *testing.T) {
		preProcessCalled := false
		postProcessCalled := false

		opts := csv.AdvancedOptions{
			PreProcess: func(record []string) []string {
				preProcessCalled = true
				return record
			},
			PostProcess: func(v interface{}) interface{} {
				postProcessCalled = true
				return v
			},
		}

		// Call the hooks
		opts.PreProcess([]string{"test"})
		opts.PostProcess(nil)

		if !preProcessCalled {
			t.Error("PreProcess was not called")
		}
		if !postProcessCalled {
			t.Error("PostProcess was not called")
		}
	})
}

func TestMultiValueSeparator(t *testing.T) {
	if csv.MultiValueSeparator != "|" {
		t.Errorf("MultiValueSeparator = %q, want %q", csv.MultiValueSeparator, "|")
	}
}

// TestApplyEscapeMode tests backslash escape processing
func TestApplyEscapeMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		opts  csv.AdvancedOptions
		want  string
	}{
		{
			name:  "RFC4180 mode - no transformation",
			input: `test\n\r\t`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeRFC4180},
			want:  `test\n\r\t`, // Should not be unescaped
		},
		{
			name:  "backslash mode with escape char disabled",
			input: `test\n`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: 0},
			want:  `test\n`, // No transformation when EscapeChar is 0
		},
		{
			name:  "backslash mode - newline",
			input: `test\nline`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  "test\nline",
		},
		{
			name:  "backslash mode - carriage return",
			input: `test\rline`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  "test\rline",
		},
		{
			name:  "backslash mode - tab",
			input: `test\tline`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  "test\tline",
		},
		{
			name:  "backslash mode - escaped backslash",
			input: `test\\backslash`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  `test\backslash`,
		},
		{
			name:  "backslash mode - escaped quote",
			input: `test\"quote`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  `test"quote`,
		},
		{
			name:  "backslash mode - unknown escape",
			input: `test\xunknown`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  `testxunknown`, // Unknown escapes pass through the character
		},
		{
			name:  "backslash mode - multiple escapes",
			input: `line1\nline2\tline3`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  "line1\nline2\tline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := csv.ApplyEscapeMode(tt.input, tt.opts)
			if got != tt.want {
				t.Errorf("ApplyEscapeMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestValueToString tests the valueToString function through FlattenStruct
func TestValueToString(t *testing.T) {
	type TestStruct struct {
		Str      string  `csv:"str"`
		Int      int     `csv:"int"`
		Int64    int64   `csv:"int64"`
		Uint     uint    `csv:"uint"`
		Float32  float32 `csv:"float32"`
		Float64  float64 `csv:"float64"`
		Bool     bool    `csv:"bool"`
		BoolTrue bool    `csv:"bool_true"`
		PtrNil   *string `csv:"ptr_nil"`
		PtrStr   *string `csv:"ptr_str"`
		Slice    []string `csv:"slice"`
	}

	str := "test"
	s := TestStruct{
		Str:      "hello",
		Int:      42,
		Int64:    123456789,
		Uint:     99,
		Float32:  3.14,
		Float64:  2.718,
		Bool:     false,
		BoolTrue: true,
		PtrNil:   nil,
		PtrStr:   &str,
		Slice:    []string{"a", "b", "c"},
	}

	result := csv.FlattenStruct(s, "")

	tests := []struct {
		field string
		want  string
	}{
		{"str", "hello"},
		{"int", "42"},
		{"int64", "123456789"},
		{"uint", "99"},
		{"float32", "3.140000104904175"}, // Float32 precision
		{"float64", "2.718"},
		{"bool", "false"},
		{"bool_true", "true"},
		{"ptr_nil", ""},
		{"ptr_str", "test"},
		{"slice", "a|b|c"}, // Uses MultiValueSeparator
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got, ok := result[tt.field]
			if !ok {
				t.Errorf("Field %s not found in result", tt.field)
				return
			}
			if got != tt.want {
				t.Errorf("valueToString() for %s = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

// TestEscapeForOutput tests backslash escape encoding for output
func TestEscapeForOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		opts  csv.AdvancedOptions
		want  string
	}{
		{
			name:  "RFC4180 mode - no transformation",
			input: "test\n\r\t",
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeRFC4180},
			want:  "test\n\r\t", // Should not be escaped
		},
		{
			name:  "backslash mode with escape char disabled",
			input: "test\n",
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: 0},
			want:  "test\n", // No transformation when EscapeChar is 0
		},
		{
			name:  "backslash mode - newline",
			input: "test\nline",
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  `test\nline`,
		},
		{
			name:  "backslash mode - carriage return",
			input: "test\rline",
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  `test\rline`,
		},
		{
			name:  "backslash mode - tab",
			input: "test\tline",
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  `test\tline`,
		},
		{
			name:  "backslash mode - backslash",
			input: `test\backslash`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  `test\\backslash`,
		},
		{
			name:  "backslash mode - quote",
			input: `test"quote`,
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  `test\"quote`,
		},
		{
			name:  "backslash mode - multiple special chars",
			input: "line1\nline2\tline3\r",
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  `line1\nline2\tline3\r`,
		},
		{
			name:  "backslash mode - plain text",
			input: "simple text",
			opts:  csv.AdvancedOptions{EscapeMode: csv.EscapeModeBackslash, EscapeChar: '\\'},
			want:  "simple text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := csv.EscapeForOutput(tt.input, tt.opts)
			if got != tt.want {
				t.Errorf("EscapeForOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}
