//go:build go1.18
// +build go1.18

package parser

import (
	"testing"
)

// FuzzParser tests the parser with random inputs to find edge cases and panics.
// Run with: go test -fuzz=FuzzParser -fuzztime=30s ./internal/parser
func FuzzParser(f *testing.F) {
	// Add seed corpus with valid CSV samples
	seeds := []string{
		"",
		"a",
		"a,b,c",
		"a,b,c\n",
		"a,b\nc,d",
		"\"quoted\"",
		"\"with,comma\"",
		"\"with\"\"quote\"",
		"\"multi\nline\"",
		"a,\"b\",c",
		"\r\n",
		"a\r\nb",
		"a,b,c\r\nd,e,f",
		",,",
		"\"\"",
		"\"\"\"\"",
		"a,\"b,c\",d",
		"\"a\"\"b\"",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic, regardless of input
		p := NewParser(input)
		_, _ = p.Parse()
	})
}
