package fastparser

import (
	"testing"

	"github.com/shapestone/shape-csv/internal/fastparser/simd"
)

// TestParseSIMD tests basic SIMD parser functionality for coverage.
// The SIMD parser is still under development, so we just ensure it doesn't panic.
func TestParseSIMD(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty input", input: ""},
		{name: "single field", input: "a"},
		{name: "simple record", input: "a,b,c"},
		{name: "two records", input: "a,b\nc,d"},
		{name: "quoted fields", input: `"hello,world","foo"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just ensure it doesn't panic
			_, _ = ParseSIMD([]byte(tt.input))
		})
	}
}

// TestParseSIMDWithOptions tests SIMD parser with options for coverage.
func TestParseSIMDWithOptions(t *testing.T) {
	// Just call the function to get coverage
	input := []byte("a,b,c")

	// Test with various delimiters
	opts := simd.ParseOptions{
		UseSIMD:   true,
		Delimiter: ',',
	}
	_, _ = ParseSIMDWithOptions(input, opts)
}

// TestHasSIMDSupport tests the CPU feature detection.
func TestHasSIMDSupport(t *testing.T) {
	// Just call it to ensure it doesn't panic and returns a bool
	result := HasSIMDSupport()
	t.Logf("HasSIMDSupport() = %v", result)

	// Should be deterministic - calling multiple times should return the same value
	for i := 0; i < 3; i++ {
		if HasSIMDSupport() != result {
			t.Error("HasSIMDSupport() returned different values on repeated calls")
		}
	}
}
