package simd

import (
	"bytes"
	"strings"
	"testing"
)

// Benchmark data generators

func generateSimpleCSV(rows, cols int) []byte {
	var buf bytes.Buffer
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if j > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString("field")
		}
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func generateQuotedCSV(rows, cols int) []byte {
	var buf bytes.Buffer
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if j > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(`"field value"`)
		}
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func generateMixedCSV(rows, cols int) []byte {
	var buf bytes.Buffer
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if j > 0 {
				buf.WriteByte(',')
			}
			if j%2 == 0 {
				buf.WriteString("field")
			} else {
				buf.WriteString(`"quoted field"`)
			}
		}
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func generateLongFieldCSV(rows int) []byte {
	var buf bytes.Buffer
	longField := strings.Repeat("a", 200)
	for i := 0; i < rows; i++ {
		buf.WriteString(longField)
		buf.WriteString(",")
		buf.WriteString(longField)
		buf.WriteString(",")
		buf.WriteString(longField)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// Benchmarks: Small files (< 1KB)

func BenchmarkSIMD_Small_Simple(b *testing.B) {
	data := generateSimpleCSV(10, 5) // ~60 bytes
	benchmarkParser(b, data, true)
}

func BenchmarkFallback_Small_Simple(b *testing.B) {
	data := generateSimpleCSV(10, 5)
	benchmarkParser(b, data, false)
}

func BenchmarkSIMD_Small_Quoted(b *testing.B) {
	data := generateQuotedCSV(10, 5) // ~160 bytes
	benchmarkParser(b, data, true)
}

func BenchmarkFallback_Small_Quoted(b *testing.B) {
	data := generateQuotedCSV(10, 5)
	benchmarkParser(b, data, false)
}

// Benchmarks: Medium files (1KB - 100KB)

func BenchmarkSIMD_Medium_Simple(b *testing.B) {
	data := generateSimpleCSV(1000, 10) // ~11KB
	benchmarkParser(b, data, true)
}

func BenchmarkFallback_Medium_Simple(b *testing.B) {
	data := generateSimpleCSV(1000, 10)
	benchmarkParser(b, data, false)
}

func BenchmarkSIMD_Medium_Quoted(b *testing.B) {
	data := generateQuotedCSV(1000, 10) // ~27KB
	benchmarkParser(b, data, true)
}

func BenchmarkFallback_Medium_Quoted(b *testing.B) {
	data := generateQuotedCSV(1000, 10)
	benchmarkParser(b, data, false)
}

func BenchmarkSIMD_Medium_Mixed(b *testing.B) {
	data := generateMixedCSV(1000, 10) // ~19KB
	benchmarkParser(b, data, true)
}

func BenchmarkFallback_Medium_Mixed(b *testing.B) {
	data := generateMixedCSV(1000, 10)
	benchmarkParser(b, data, false)
}

// Benchmarks: Large files (> 100KB)

func BenchmarkSIMD_Large_Simple(b *testing.B) {
	data := generateSimpleCSV(10000, 20) // ~220KB
	benchmarkParser(b, data, true)
}

func BenchmarkFallback_Large_Simple(b *testing.B) {
	data := generateSimpleCSV(10000, 20)
	benchmarkParser(b, data, false)
}

func BenchmarkSIMD_Large_Quoted(b *testing.B) {
	data := generateQuotedCSV(10000, 20) // ~540KB
	benchmarkParser(b, data, true)
}

func BenchmarkFallback_Large_Quoted(b *testing.B) {
	data := generateQuotedCSV(10000, 20)
	benchmarkParser(b, data, false)
}

func BenchmarkSIMD_Large_LongFields(b *testing.B) {
	data := generateLongFieldCSV(1000) // ~600KB
	benchmarkParser(b, data, true)
}

func BenchmarkFallback_Large_LongFields(b *testing.B) {
	data := generateLongFieldCSV(1000)
	benchmarkParser(b, data, false)
}

// Benchmarks: Stage-specific operations

func BenchmarkStage1_Fallback(b *testing.B) {
	data := make([]byte, ChunkSize)
	for i := 0; i < ChunkSize; i++ {
		data[i] = 'a'
	}
	// Add structural characters at various positions
	data[5] = ','
	data[15] = '"'
	data[25] = ','
	data[35] = '"'
	data[45] = '\n'

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = detectStructuralCharsFallback(data, ',')
	}
}

func BenchmarkStage1_SIMD(b *testing.B) {
	if !HasAVX2() {
		b.Skip("AVX2 not available")
	}

	data := make([]byte, ChunkSize)
	for i := 0; i < ChunkSize; i++ {
		data[i] = 'a'
	}
	data[5] = ','
	data[15] = '"'
	data[25] = ','
	data[35] = '"'
	data[45] = '\n'

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = detectStructuralChars(data, ',')
	}
}

func BenchmarkStage2_QuoteStateTracking(b *testing.B) {
	// Simulate quote mask with several quote pairs
	quoteMask := uint64(0)
	quoteMask |= (1 << 5)  // Quote at position 5
	quoteMask |= (1 << 15) // Quote at position 15
	quoteMask |= (1 << 25) // Quote at position 25
	quoteMask |= (1 << 35) // Quote at position 35

	tracker := NewQuoteStateTracker()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = tracker.ProcessQuoteMask(quoteMask)
	}
}

func BenchmarkMaskIteration(b *testing.B) {
	// Create a mask with 10 set bits
	mask := uint64(0)
	for i := 0; i < 10; i++ {
		mask |= (1 << uint(i*6))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		iter := NewMaskIterator(mask)
		for iter.HasNext() {
			_ = iter.Next()
		}
	}
}

// Benchmark utilities for bit operations

func BenchmarkTrailingZeros(b *testing.B) {
	values := []uint64{
		0b00001000,
		0b00010000,
		0b00100000,
		0b01000000,
		0b10000000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range values {
			_ = trailingZeros64(v)
		}
	}
}

func BenchmarkPopcount(b *testing.B) {
	values := []uint64{
		0b00001111,
		0b01010101,
		0b11111111,
		0xFFFFFFFF,
		0xFFFFFFFFFFFFFFFF,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range values {
			_ = popcount64(v)
		}
	}
}

// Helper function to run parser benchmarks

func benchmarkParser(b *testing.B, data []byte, useSIMD bool) {
	// Skip SIMD benchmark if not available
	if useSIMD && !HasAVX2() {
		b.Skip("AVX2 not available")
	}

	opts := DefaultParseOptions()
	opts.UseSIMD = useSIMD

	parser := NewParser(opts)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}
