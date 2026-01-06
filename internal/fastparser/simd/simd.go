// Package simd provides SIMD-accelerated CSV parsing using AVX2 instructions.
//
// The implementation uses a two-stage architecture inspired by simdjson:
// Stage 1: Structural character detection using SIMD (quotes, delimiters, newlines)
// Stage 2: Field extraction from bitmasks
//
// On x86-64 systems with AVX2 support, this provides 4-10x faster parsing.
// On other systems, it falls back to a pure Go implementation.
package simd

import (
	"sync"
)

// Bitmasks holds the structural character positions for a 64-byte chunk.
// Each bit position corresponds to a byte in the chunk.
type Bitmasks struct {
	Quotes     uint64 // Positions of quote characters (")
	Delimiters uint64 // Positions of delimiter characters (,)
	Newlines   uint64 // Positions of newline characters (\r or \n)
}

// ChunkSize is the size of chunks processed by the SIMD parser (64 bytes).
// This is chosen to:
// - Process 2x YMM registers (32 bytes each) per iteration
// - Align well with cache line boundaries (64 bytes)
// - Produce convenient 64-bit bitmasks
const ChunkSize = 64

// QuoteState tracks whether we are currently inside or outside quoted fields.
// This state must be maintained across chunk boundaries.
type QuoteState struct {
	Inside bool // true if we are currently inside a quoted field
}

// cpuFeatures holds the detected CPU capabilities.
type cpuFeatures struct {
	hasAVX2   bool
	hasSSE4_2 bool
}

var (
	// cpuCaps holds the detected CPU features (initialized once).
	cpuCaps     cpuFeatures
	cpuCapsOnce sync.Once
)

// detectCPUFeatures initializes CPU feature detection.
// This is called once at startup.
func detectCPUFeatures() {
	cpuCapsOnce.Do(func() {
		// CPU feature detection is implemented in platform-specific files:
		// - cpuinfo_amd64.go for x86-64
		// - cpuinfo_other.go for other platforms
		cpuCaps = getCPUFeatures()
	})
}

// HasAVX2 returns true if the CPU supports AVX2 instructions.
func HasAVX2() bool {
	detectCPUFeatures()
	return cpuCaps.hasAVX2
}

// HasSSE4_2 returns true if the CPU supports SSE 4.2 instructions.
func HasSSE4_2() bool {
	detectCPUFeatures()
	return cpuCaps.hasSSE4_2
}

// ParseOptions controls SIMD parser behavior.
type ParseOptions struct {
	UseSIMD   bool // Manually control SIMD usage (default: auto-detect)
	Delimiter byte // CSV delimiter (default: ',')
}

// DefaultParseOptions returns the default parsing options.
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		UseSIMD:   true, // Auto-detect and use SIMD if available
		Delimiter: ',',
	}
}

// Parser is the main SIMD-accelerated CSV parser.
type Parser struct {
	opts       ParseOptions
	useSIMD    bool // Resolved SIMD usage after feature detection
	quoteState QuoteState
}

// NewParser creates a new SIMD parser with the given options.
func NewParser(opts ParseOptions) *Parser {
	p := &Parser{
		opts: opts,
	}

	// Resolve SIMD usage based on options and CPU capabilities
	if opts.UseSIMD {
		p.useSIMD = HasAVX2()
	}

	return p
}

// Parse parses CSV data and returns records.
// This is the main entry point for SIMD-accelerated parsing.
func (p *Parser) Parse(data []byte) ([][]string, error) {
	if p.useSIMD {
		return p.parseSIMD(data)
	}
	return p.parseFallback(data)
}

// parseSIMD implements the SIMD-accelerated parsing path.
func (p *Parser) parseSIMD(data []byte) ([][]string, error) {
	records := make([][]string, 0, 16)

	// Process data in 64-byte chunks
	offset := 0
	for offset < len(data) {
		// Calculate chunk size (may be less than 64 bytes at end)
		chunkSize := ChunkSize
		if offset+chunkSize > len(data) {
			chunkSize = len(data) - offset
		}

		// Stage 1: Detect structural characters using SIMD
		var masks Bitmasks
		if chunkSize == ChunkSize {
			// Full chunk - use SIMD
			masks = detectStructuralChars(data[offset:offset+ChunkSize], p.opts.Delimiter)
		} else {
			// Partial chunk at end - use fallback
			masks = detectStructuralCharsFallback(data[offset:offset+chunkSize], p.opts.Delimiter)
		}

		// Stage 2: Extract fields from bitmasks
		chunkRecords, newOffset, err := p.extractFields(data, offset, masks)
		if err != nil {
			return nil, err
		}

		records = append(records, chunkRecords...)
		offset = newOffset
	}

	return records, nil
}

// parseFallback implements the pure Go fallback parsing path.
func (p *Parser) parseFallback(data []byte) ([][]string, error) {
	// For now, delegate to the existing fastparser
	// TODO: Implement a pure Go two-stage parser that mirrors the SIMD approach
	records := make([][]string, 0, 16)

	offset := 0
	for offset < len(data) {
		// Process in chunks using pure Go
		chunkSize := ChunkSize
		if offset+chunkSize > len(data) {
			chunkSize = len(data) - offset
		}

		masks := detectStructuralCharsFallback(data[offset:offset+chunkSize], p.opts.Delimiter)

		chunkRecords, newOffset, err := p.extractFields(data, offset, masks)
		if err != nil {
			return nil, err
		}

		records = append(records, chunkRecords...)
		offset = newOffset
	}

	return records, nil
}

// extractFields extracts CSV fields from structural character bitmasks.
// This is Stage 2 of the two-stage architecture.
func (p *Parser) extractFields(data []byte, offset int, masks Bitmasks) ([][]string, int, error) {
	extractor := NewFieldExtractor(data, p.opts.Delimiter)
	extractor.quoteState = p.quoteState

	// Calculate chunk size
	chunkSize := ChunkSize
	if offset+chunkSize > len(data) {
		chunkSize = len(data) - offset
	}

	// Extract fields from the chunk
	records, newOffset, err := extractor.ExtractFields(offset, chunkSize, masks)

	// Update parser state for next chunk
	p.quoteState = extractor.quoteState

	return records, newOffset, err
}
