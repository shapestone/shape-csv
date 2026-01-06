package simd

import (
	"strings"
	"testing"
)

// TestCPUFeatureDetection tests CPU feature detection.
func TestCPUFeatureDetection(t *testing.T) {
	// Just ensure detection doesn't panic
	hasAVX2 := HasAVX2()
	hasSSE42 := HasSSE4_2()

	t.Logf("CPU Features: AVX2=%v, SSE4.2=%v", hasAVX2, hasSSE42)

	// On x86-64, at least one should be available (most modern CPUs have SSE4.2)
	// On other platforms, both should be false
}

// TestBitmaskStructuralDetectionFallback tests pure Go structural character detection.
func TestBitmaskStructuralDetectionFallback(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		delimiter byte
		wantQuote []int
		wantDelim []int
		wantNewl  []int
	}{
		{
			name:      "simple fields",
			input:     "a,b,c\n",
			delimiter: ',',
			wantQuote: []int{},
			wantDelim: []int{1, 3},
			wantNewl:  []int{5},
		},
		{
			name:      "quoted field",
			input:     `"hello",world` + "\n",
			delimiter: ',',
			wantQuote: []int{0, 6},
			wantDelim: []int{7},
			wantNewl:  []int{13},
		},
		{
			name:      "escaped quotes",
			input:     `"he""llo",world` + "\n",
			delimiter: ',',
			wantQuote: []int{0, 3, 4, 8},
			wantDelim: []int{9},
			wantNewl:  []int{15},
		},
		{
			name:      "CRLF newline",
			input:     "a,b\r\n",
			delimiter: ',',
			wantQuote: []int{},
			wantDelim: []int{1},
			wantNewl:  []int{3, 4},
		},
		{
			name:      "custom delimiter",
			input:     "a|b|c\n",
			delimiter: '|',
			wantQuote: []int{},
			wantDelim: []int{1, 3},
			wantNewl:  []int{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pad input to chunk size if needed
			data := []byte(tt.input)
			if len(data) < ChunkSize {
				padded := make([]byte, ChunkSize)
				copy(padded, data)
				data = padded
			}

			masks := detectStructuralCharsFallback(data[:len(tt.input)], tt.delimiter)

			// Check quotes
			gotQuote := extractBitPositions(masks.Quotes, len(tt.input))
			if !equalIntSlices(gotQuote, tt.wantQuote) {
				t.Errorf("Quote positions: got %v, want %v", gotQuote, tt.wantQuote)
			}

			// Check delimiters
			gotDelim := extractBitPositions(masks.Delimiters, len(tt.input))
			if !equalIntSlices(gotDelim, tt.wantDelim) {
				t.Errorf("Delimiter positions: got %v, want %v", gotDelim, tt.wantDelim)
			}

			// Check newlines
			gotNewl := extractBitPositions(masks.Newlines, len(tt.input))
			if !equalIntSlices(gotNewl, tt.wantNewl) {
				t.Errorf("Newline positions: got %v, want %v", gotNewl, tt.wantNewl)
			}
		})
	}
}

// TestQuoteStateTracker tests quote state tracking across chunks.
func TestQuoteStateTracker(t *testing.T) {
	tests := []struct {
		name       string
		quoteMask  uint64
		initialIn  bool
		wantInside []int
		wantFinal  bool
	}{
		{
			name:       "simple quote pair",
			quoteMask:  0b00001001, // Bits 0 and 3
			initialIn:  false,
			wantInside: []int{1, 2, 3}, // Inside from after opening quote until closing quote
			wantFinal:  false,
		},
		{
			name:       "unterminated quote",
			quoteMask:  0b00000001, // Bit 0 only
			initialIn:  false,
			wantInside: []int{1, 2, 3, 4, 5, 6, 7}, // Inside after opening quote
			wantFinal:  true,
		},
		{
			name:       "continue from previous chunk",
			quoteMask:  0b00000010, // Bit 1 (closing quote)
			initialIn:  true,
			wantInside: []int{0, 1}, // Inside until and including closing quote position
			wantFinal:  false,
		},
		{
			name:       "escaped quotes",
			quoteMask:  0b00000011, // Bits 0 and 1 (adjacent)
			initialIn:  false,
			wantInside: []int{}, // Adjacent quotes cancel out
			wantFinal:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := &QuoteStateTracker{insideQuote: tt.initialIn}
			insideMask, err := tracker.ProcessQuoteMask(tt.quoteMask)
			if err != nil {
				t.Fatalf("ProcessQuoteMask failed: %v", err)
			}

			// Check inside positions (limited to first 8 bits for test)
			gotInside := extractBitPositions(insideMask, 8)
			if !equalIntSlices(gotInside, tt.wantInside) {
				t.Errorf("Inside positions: got %v, want %v", gotInside, tt.wantInside)
				t.Logf("Inside mask: %064b", insideMask)
			}

			// Check final state
			if tracker.GetState() != tt.wantFinal {
				t.Errorf("Final state: got %v, want %v", tracker.GetState(), tt.wantFinal)
			}
		})
	}
}

// TestMaskIterator tests bitmask iteration utilities.
func TestMaskIterator(t *testing.T) {
	mask := uint64(0b00010101) // Bits 0, 2, 4

	iter := NewMaskIterator(mask)

	// Should iterate in order of bit position
	expected := []int{0, 2, 4}
	got := []int{}

	for iter.HasNext() {
		pos := iter.Next()
		got = append(got, pos)
	}

	if !equalIntSlices(got, expected) {
		t.Errorf("Iteration order: got %v, want %v", got, expected)
	}

	// Should return -1 when exhausted
	if pos := iter.Next(); pos != -1 {
		t.Errorf("Expected -1 when exhausted, got %d", pos)
	}
}

// TestBitUtilities tests bit manipulation utilities.
func TestBitUtilities(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
		want  int
	}{
		{"trailing zeros 1", 0b00001000, 3},
		{"trailing zeros 2", 0b00010000, 4},
		{"trailing zeros 3", 0b00000001, 0},
		{"trailing zeros empty", 0, 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trailingZeros64(tt.value)
			if got != tt.want {
				t.Errorf("trailingZeros64(%064b) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}

	// Test popcount
	popcountTests := []struct {
		value uint64
		want  int
	}{
		{0b00000000, 0},
		{0b00000001, 1},
		{0b00000011, 2},
		{0b10101010, 4},
		{0xFFFFFFFFFFFFFFFF, 64},
	}

	for _, tt := range popcountTests {
		got := popcount64(tt.value)
		if got != tt.want {
			t.Errorf("popcount64(%064b) = %d, want %d", tt.value, got, tt.want)
		}
	}
}

// TestFieldExtractor tests the field extraction stage.
func TestFieldExtractor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "simple CSV",
			input: "a,b,c\n",
			want:  [][]string{{"a", "b", "c"}},
		},
		{
			name:  "multiple rows",
			input: "a,b\nc,d\n",
			want:  [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:  "quoted field",
			input: `"hello",world` + "\n",
			want:  [][]string{{"hello", "world"}},
		},
		{
			name:  "quoted with comma",
			input: `"hello, world",test` + "\n",
			want:  [][]string{{"hello, world", "test"}},
		},
		{
			name:  "escaped quotes",
			input: `"he""llo",world` + "\n",
			want:  [][]string{{"he\"llo", "world"}},
		},
		{
			name:  "empty fields",
			input: "a,,c\n",
			want:  [][]string{{"a", "", "c"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(tt.input)
			masks := detectStructuralCharsFallback(data, ',')

			extractor := NewFieldExtractor(data, ',')
			records, _, err := extractor.ExtractFields(0, len(data), masks)
			if err != nil {
				t.Fatalf("ExtractFields failed: %v", err)
			}

			if !equalRecords(records, tt.want) {
				t.Errorf("Records:\ngot:  %v\nwant: %v", records, tt.want)
			}
		})
	}
}

// TestParserBasic tests the basic SIMD parser functionality.
func TestParserBasic(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  [][]string
	}{
		{
			name:  "simple CSV",
			input: "a,b,c\n",
			want:  [][]string{{"a", "b", "c"}},
		},
		{
			name:  "multiple rows",
			input: "a,b\nc,d\ne,f\n",
			want:  [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}},
		},
		{
			name:  "quoted fields",
			input: `"hello","world"` + "\n",
			want:  [][]string{{"hello", "world"}},
		},
		{
			name:  "mixed quoted and unquoted",
			input: `a,"b",c` + "\n",
			want:  [][]string{{"a", "b", "c"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultParseOptions()
			opts.UseSIMD = false // Force fallback for consistent testing

			parser := NewParser(opts)
			records, err := parser.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !equalRecords(records, tt.want) {
				t.Errorf("Records:\ngot:  %v\nwant: %v", records, tt.want)
			}
		})
	}
}

// TestParserLargeInput tests parsing of input larger than one chunk.
func TestParserLargeInput(t *testing.T) {
	// Create CSV larger than 64 bytes
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString("field1,field2,field3\n")
	}
	input := sb.String()

	opts := DefaultParseOptions()
	opts.UseSIMD = false

	parser := NewParser(opts)
	records, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 20 rows
	if len(records) != 20 {
		t.Errorf("Expected 20 records, got %d", len(records))
	}

	// Each row should have 3 fields
	for i, record := range records {
		if len(record) != 3 {
			t.Errorf("Record %d: expected 3 fields, got %d", i, len(record))
		}
	}
}

// TestParserWithSIMD tests SIMD parser if AVX2 is available.
func TestParserWithSIMD(t *testing.T) {
	if !HasAVX2() {
		t.Skip("AVX2 not available on this CPU")
	}

	input := "a,b,c\nd,e,f\n"
	want := [][]string{{"a", "b", "c"}, {"d", "e", "f"}}

	opts := DefaultParseOptions()
	opts.UseSIMD = true

	parser := NewParser(opts)
	records, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !equalRecords(records, want) {
		t.Errorf("Records:\ngot:  %v\nwant: %v", records, want)
	}
}

// Benchmark functions

// BenchmarkStructuralDetectionFallback benchmarks pure Go structural detection.
func BenchmarkStructuralDetectionFallback(b *testing.B) {
	data := make([]byte, ChunkSize)
	for i := 0; i < ChunkSize; i++ {
		data[i] = 'a'
	}
	// Add some structural characters
	data[10] = ','
	data[20] = '"'
	data[30] = '\n'

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detectStructuralCharsFallback(data, ',')
	}
}

// BenchmarkStructuralDetectionSIMD benchmarks SIMD structural detection.
func BenchmarkStructuralDetectionSIMD(b *testing.B) {
	if !HasAVX2() {
		b.Skip("AVX2 not available on this CPU")
	}

	data := make([]byte, ChunkSize)
	for i := 0; i < ChunkSize; i++ {
		data[i] = 'a'
	}
	data[10] = ','
	data[20] = '"'
	data[30] = '\n'

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detectStructuralChars(data, ',')
	}
}

// Helper functions

// extractBitPositions extracts the positions of set bits in a mask.
func extractBitPositions(mask uint64, limit int) []int {
	positions := []int{}
	for i := 0; i < limit && i < 64; i++ {
		if (mask & (1 << uint(i))) != 0 {
			positions = append(positions, i)
		}
	}
	return positions
}

// equalIntSlices compares two int slices.
func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// equalRecords compares two record slices.
func equalRecords(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}
