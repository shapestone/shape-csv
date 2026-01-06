package fastparser

import (
	"github.com/shapestone/shape-csv/internal/fastparser/simd"
)

// ParseSIMD parses CSV data using SIMD acceleration when available.
// This function automatically detects CPU features and uses AVX2 if supported,
// or falls back to pure Go implementation on systems without AVX2.
//
// The SIMD parser implements a two-stage architecture:
// - Stage 1: Structural character detection (quotes, delimiters, newlines)
// - Stage 2: Field extraction from bitmasks
//
// On x86-64 systems with AVX2, this can provide 4-10x faster parsing.
//
// Returns a slice of records, where each record is a slice of field values.
func ParseSIMD(data []byte) ([][]string, error) {
	opts := simd.DefaultParseOptions()
	parser := simd.NewParser(opts)
	return parser.Parse(data)
}

// ParseSIMDWithOptions parses CSV data using SIMD with custom options.
// This allows manual control over SIMD usage and delimiter configuration.
func ParseSIMDWithOptions(data []byte, opts simd.ParseOptions) ([][]string, error) {
	parser := simd.NewParser(opts)
	return parser.Parse(data)
}

// HasSIMDSupport returns true if the current CPU supports SIMD acceleration (AVX2).
// This can be used to determine whether the SIMD parser will provide acceleration.
func HasSIMDSupport() bool {
	return simd.HasAVX2()
}
