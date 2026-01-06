//go:build go1.18
// +build go1.18

package tokenizer

import (
	"testing"
)

// FuzzTokenizer tests the tokenizer with random inputs to find edge cases and panics.
// Run with: go test -fuzz=FuzzTokenizer -fuzztime=30s ./internal/tokenizer
func FuzzTokenizer(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		"",
		"a",
		",",
		"\n",
		"\r\n",
		"\"",
		"\"\"",
		"a,b,c",
		"\"quoted\"",
		"\"with,comma\"",
		"\"with\"\"quote\"",
		"a\nb\nc",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// The tokenizer should never panic, regardless of input
		tok := NewTokenizer()
		tok.Initialize(input)
		for {
			token, ok := tok.NextToken()
			if !ok {
				break
			}
			// Consume all tokens without panicking
			_ = token
		}
	})
}
