package tokenizer

import (
	"strings"
	"testing"

	"github.com/shapestone/shape-core/pkg/tokenizer"
)

// TestTokenTypes tests that all token constants are defined and non-empty.
func TestTokenTypes(t *testing.T) {
	tests := []struct {
		name      string
		tokenType string
	}{
		{"comma token", TokenComma},
		{"double quote token", TokenDQuote},
		{"newline token", TokenNewline},
		{"field token", TokenField},
		{"EOF token", TokenEOF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tokenType == "" {
				t.Errorf("%s is empty", tt.name)
			}
		})
	}
}

// TestNewTokenizer_BasicTokens tests comprehensive tokenization scenarios.
func TestNewTokenizer_BasicTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			kind  string
			value string
		}
	}{
		{
			name:  "single comma",
			input: ",",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenComma, ","},
			},
		},
		{
			name:  "single field",
			input: "abc",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenField, "abc"},
			},
		},
		{
			name:  "newline LF",
			input: "\n",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenNewline, "\n"},
			},
		},
		{
			name:  "newline CRLF",
			input: "\r\n",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenNewline, "\r\n"},
			},
		},
		{
			name:  "simple row",
			input: "a,b,c",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenField, "a"},
				{TokenComma, ","},
				{TokenField, "b"},
				{TokenComma, ","},
				{TokenField, "c"},
			},
		},
		{
			name:  "quoted field",
			input: `"hello"`,
			expected: []struct {
				kind  string
				value string
			}{
				{TokenDQuote, `"`},
				{TokenField, "hello"},
				{TokenDQuote, `"`},
			},
		},
		{
			name:  "quoted field with comma",
			input: `"a,b"`,
			expected: []struct {
				kind  string
				value string
			}{
				{TokenDQuote, `"`},
				{TokenField, "a"},
				{TokenComma, ","},
				{TokenField, "b"},
				{TokenDQuote, `"`},
			},
		},
		{
			name:  "quoted field with escaped quote",
			input: `"say ""hello"""`,
			expected: []struct {
				kind  string
				value string
			}{
				{TokenDQuote, `"`},
				{TokenField, `say `},
				{TokenDQuote, `"`},
				{TokenDQuote, `"`},
				{TokenField, `hello`},
				{TokenDQuote, `"`},
				{TokenDQuote, `"`},
				{TokenDQuote, `"`},
			},
		},
		{
			name:  "empty fields",
			input: ",,",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenComma, ","},
				{TokenComma, ","},
			},
		},
		{
			name:  "row with empty field at start",
			input: ",b,c",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenComma, ","},
				{TokenField, "b"},
				{TokenComma, ","},
				{TokenField, "c"},
			},
		},
		{
			name:  "row with empty field in middle",
			input: "a,,c",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenField, "a"},
				{TokenComma, ","},
				{TokenComma, ","},
				{TokenField, "c"},
			},
		},
		{
			name:  "row with empty field at end",
			input: "a,b,",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenField, "a"},
				{TokenComma, ","},
				{TokenField, "b"},
				{TokenComma, ","},
			},
		},
		{
			name:  "empty quoted field",
			input: `""`,
			expected: []struct {
				kind  string
				value string
			}{
				{TokenDQuote, `"`},
				{TokenDQuote, `"`},
			},
		},
		{
			name:  "complete row with newline",
			input: "a,b,c\n",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenField, "a"},
				{TokenComma, ","},
				{TokenField, "b"},
				{TokenComma, ","},
				{TokenField, "c"},
				{TokenNewline, "\n"},
			},
		},
		{
			name:  "multiple rows",
			input: "a,b\nx,y\n",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenField, "a"},
				{TokenComma, ","},
				{TokenField, "b"},
				{TokenNewline, "\n"},
				{TokenField, "x"},
				{TokenComma, ","},
				{TokenField, "y"},
				{TokenNewline, "\n"},
			},
		},
		{
			name:  "quoted field with newline inside",
			input: "\"line1\nline2\"",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenDQuote, `"`},
				{TokenField, "line1"},
				{TokenNewline, "\n"},
				{TokenField, "line2"},
				{TokenDQuote, `"`},
			},
		},
		{
			name:  "field with special characters",
			input: "hello-world_123",
			expected: []struct {
				kind  string
				value string
			}{
				{TokenField, "hello-world_123"},
			},
		},
		{
			name:  "mixed quoted and unquoted",
			input: `a,"b",c`,
			expected: []struct {
				kind  string
				value string
			}{
				{TokenField, "a"},
				{TokenComma, ","},
				{TokenDQuote, `"`},
				{TokenField, "b"},
				{TokenDQuote, `"`},
				{TokenComma, ","},
				{TokenField, "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			for i, exp := range tt.expected {
				token, ok := tok.NextToken()
				if !ok {
					t.Fatalf("token %d: expected token, got none (expected %s: %q)", i, exp.kind, exp.value)
				}
				if token.Kind() != exp.kind {
					t.Errorf("token %d: expected kind %s, got %s (value: %q)", i, exp.kind, token.Kind(), token.ValueString())
				}
				if token.ValueString() != exp.value {
					t.Errorf("token %d: expected value %q, got %q (kind: %s)", i, exp.value, token.ValueString(), token.Kind())
				}
			}

			// Verify no extra tokens
			token, ok := tok.NextToken()
			if ok {
				t.Errorf("expected no more tokens, got %s: %q", token.Kind(), token.ValueString())
			}
		})
	}
}

// TestTokenizer_LargeCSV tests tokenizing a large CSV that crosses buffer boundaries.
func TestTokenizer_LargeCSV(t *testing.T) {
	// Create a CSV with many rows to test buffering
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString(`"field1","field2","field3"`)
		sb.WriteString("\n")
	}

	data := sb.String()
	reader := strings.NewReader(data)
	stream := tokenizer.NewStreamFromReader(reader)

	tok := NewTokenizerWithStream(stream)

	tokenCount := 0
	for {
		_, ok := tok.NextToken()
		if !ok {
			if !stream.IsEos() {
				t.Fatalf("Tokenization failed after %d tokens, but not at EOS", tokenCount)
			}
			break
		}
		tokenCount++
	}

	// Each row has: " field1 " , " field2 " , " field3 " \n
	// = " + field1 + " + , + " + field2 + " + , + " + field3 + " + \n = 12 tokens
	// 100 rows = 1200 tokens
	expectedTokens := 100 * 12
	if tokenCount != expectedTokens {
		t.Errorf("Expected %d tokens, got %d", expectedTokens, tokenCount)
	}
}
