package simd

import (
	"fmt"
)

// FieldExtractor extracts CSV fields from structural character bitmasks.
// This is Stage 2 of the two-stage SIMD architecture.
type FieldExtractor struct {
	data       []byte
	quoteState QuoteState
	delimiter  byte

	// Current parsing state
	currentField []byte
	currentRow   []string
	rows         [][]string
}

// NewFieldExtractor creates a new field extractor.
func NewFieldExtractor(data []byte, delimiter byte) *FieldExtractor {
	return &FieldExtractor{
		data:      data,
		delimiter: delimiter,
		rows:      make([][]string, 0, 16),
	}
}

// ExtractFields extracts fields from a chunk of data using bitmasks.
// Returns the extracted records and the number of bytes consumed.
func (e *FieldExtractor) ExtractFields(offset int, chunkSize int, masks Bitmasks) ([][]string, int, error) {
	// Process each byte position in the chunk
	for i := 0; i < chunkSize; i++ {
		bitPos := uint(i)
		bytePos := offset + i

		// Check if this position has any structural characters
		hasQuote := (masks.Quotes & (1 << bitPos)) != 0
		hasDelimiter := (masks.Delimiters & (1 << bitPos)) != 0
		hasNewline := (masks.Newlines & (1 << bitPos)) != 0

		// Get the actual byte
		c := e.data[bytePos]

		// Handle quote
		if hasQuote {
			if !e.quoteState.Inside {
				// Opening quote
				e.quoteState.Inside = true
				continue // Skip the quote character
			}

			// We're inside a quote - check if this is an escaped quote
			if bytePos+1 < len(e.data) && e.data[bytePos+1] == '"' {
				// Escaped quote - add single quote to field
				e.currentField = append(e.currentField, '"')
				// Skip next quote (handled by clearing adjacent bits in bitmask)
				i++
				continue
			}

			// Closing quote
			e.quoteState.Inside = false
			continue // Skip the quote character
		}

		// Handle delimiter (only if not inside quotes)
		if hasDelimiter && !e.quoteState.Inside {
			// End of field
			e.finishField()
			continue
		}

		// Handle newline (only if not inside quotes)
		if hasNewline && !e.quoteState.Inside {
			// End of field and end of row
			e.finishField()
			e.finishRow()

			// Handle CRLF - skip LF if we just processed CR
			if c == '\r' && bytePos+1 < len(e.data) && e.data[bytePos+1] == '\n' {
				i++ // Skip the LF
			}
			continue
		}

		// Regular character - add to current field
		e.currentField = append(e.currentField, c)
	}

	// Return extracted rows
	result := e.rows
	e.rows = make([][]string, 0, 16) // Reset for next chunk
	return result, offset + chunkSize, nil
}

// finishField completes the current field and adds it to the current row.
func (e *FieldExtractor) finishField() {
	field := string(e.currentField)
	e.currentRow = append(e.currentRow, field)
	e.currentField = e.currentField[:0] // Reset field buffer
}

// finishRow completes the current row and adds it to the results.
func (e *FieldExtractor) finishRow() {
	if len(e.currentRow) > 0 {
		e.rows = append(e.rows, e.currentRow)
		e.currentRow = make([]string, 0, len(e.currentRow)) // Reset row buffer
	}
}

// QuoteStateTracker manages quote state across chunk boundaries using cumulative XOR.
// This is based on the technique from simdjson and minio/simdcsv.
type QuoteStateTracker struct {
	insideQuote bool
}

// NewQuoteStateTracker creates a new quote state tracker.
func NewQuoteStateTracker() *QuoteStateTracker {
	return &QuoteStateTracker{}
}

// ProcessQuoteMask processes a quote bitmask and returns:
// - A bitmask indicating which positions are inside quotes
// - The updated quote state for the next chunk
//
// Algorithm:
// 1. Use cumulative XOR to toggle quote state at each quote position
// 2. Handle escaped quotes ("") by clearing adjacent bits
// 3. Track state across chunk boundaries
func (t *QuoteStateTracker) ProcessQuoteMask(quoteMask uint64) (insideQuoteMask uint64, err error) {
	// Handle escaped quotes by detecting adjacent quote bits
	// If bit N and N+1 are both set, they form an escaped quote
	// Clear both bits to ignore them
	escapedQuotes := quoteMask & (quoteMask >> 1)
	quoteMask &^= escapedQuotes       // Clear lower bit of pair
	quoteMask &^= (escapedQuotes << 1) // Clear upper bit of pair

	// Use cumulative XOR to track inside/outside quote state
	// At each quote position, toggle the state
	// The result is a bitmask where 1 = inside quote, 0 = outside quote

	// Start with initial state
	state := uint64(0)
	if t.insideQuote {
		state = 1
	}

	// Process each bit position
	insideQuoteMask = 0
	for i := uint(0); i < 64; i++ {
		// Set the inside quote mask for this position BEFORE toggling
		// This ensures the quote character itself is not marked as inside
		if state != 0 {
			insideQuoteMask |= (1 << i)
		}

		// Check if this position has a quote and toggle state
		if (quoteMask & (1 << i)) != 0 {
			// Toggle state
			state ^= 1
		}
	}

	// Update state for next chunk
	t.insideQuote = (state != 0)

	return insideQuoteMask, nil
}

// GetState returns the current quote state.
func (t *QuoteStateTracker) GetState() bool {
	return t.insideQuote
}

// MaskIterator provides utilities for iterating over bitmasks.
type MaskIterator struct {
	mask uint64
}

// NewMaskIterator creates a new mask iterator.
func NewMaskIterator(mask uint64) *MaskIterator {
	return &MaskIterator{mask: mask}
}

// Next returns the position of the next set bit, or -1 if no more bits.
func (m *MaskIterator) Next() int {
	if m.mask == 0 {
		return -1
	}

	// Find position of lowest set bit (trailing zeros count)
	pos := trailingZeros64(m.mask)

	// Clear that bit
	m.mask &^= (1 << uint(pos))

	return pos
}

// HasNext returns true if there are more set bits.
func (m *MaskIterator) HasNext() bool {
	return m.mask != 0
}

// Count returns the number of set bits remaining.
func (m *MaskIterator) Count() int {
	return popcount64(m.mask)
}

// trailingZeros64 counts the number of trailing zero bits.
// This is equivalent to finding the position of the lowest set bit.
func trailingZeros64(x uint64) int {
	if x == 0 {
		return 64
	}

	// De Bruijn sequence method (fast on most CPUs)
	const debruijn64 = 0x03f79d71b4cb0a89
	var deBruijnIdx = [64]int{
		0, 1, 56, 2, 57, 49, 28, 3, 61, 58, 42, 50, 38, 29, 17, 4,
		62, 47, 59, 36, 45, 43, 51, 22, 53, 39, 33, 30, 24, 18, 12, 5,
		63, 55, 48, 27, 60, 41, 37, 16, 46, 35, 44, 21, 52, 32, 23, 11,
		54, 26, 40, 15, 34, 20, 31, 10, 25, 14, 19, 9, 13, 8, 7, 6,
	}

	return deBruijnIdx[((x&-x)*debruijn64)>>58]
}

// popcount64 counts the number of set bits (population count).
func popcount64(x uint64) int {
	// Brian Kernighan's algorithm
	count := 0
	for x != 0 {
		x &= x - 1 // Clear lowest set bit
		count++
	}
	return count
}

// ValidateMasks performs sanity checks on the bitmasks.
func ValidateMasks(masks Bitmasks) error {
	// Check for any impossible combinations
	// For example, a quote and delimiter at the same position inside a quote
	// is an error in the input data

	// For now, just do basic validation
	// TODO: Add more sophisticated validation

	if masks.Quotes == 0 && masks.Delimiters == 0 && masks.Newlines == 0 {
		// All empty is fine (chunk with no structural characters)
		return nil
	}

	return nil
}

// PrintMasks is a debugging utility to visualize bitmasks.
func PrintMasks(masks Bitmasks, data []byte) {
	fmt.Printf("Data:       ")
	for i := 0; i < len(data) && i < 64; i++ {
		c := data[i]
		if c >= 32 && c < 127 {
			fmt.Printf("%c", c)
		} else if c == '\n' {
			fmt.Print("\\n")
		} else if c == '\r' {
			fmt.Print("\\r")
		} else {
			fmt.Print("·")
		}
	}
	fmt.Println()

	fmt.Printf("Quotes:     ")
	printBitmask(masks.Quotes, 64)
	fmt.Println()

	fmt.Printf("Delimiters: ")
	printBitmask(masks.Delimiters, 64)
	fmt.Println()

	fmt.Printf("Newlines:   ")
	printBitmask(masks.Newlines, 64)
	fmt.Println()
}

// printBitmask prints a bitmask as a visual representation.
func printBitmask(mask uint64, width int) {
	for i := 0; i < width; i++ {
		if (mask & (1 << uint(i))) != 0 {
			fmt.Print("1")
		} else {
			fmt.Print("·")
		}
	}
}
