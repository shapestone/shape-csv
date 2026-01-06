# SIMD-Accelerated CSV Parser

This package implements SIMD (Single Instruction, Multiple Data) acceleration for CSV parsing using AVX2 instructions on x86-64 platforms.

## Overview

The SIMD parser uses a **two-stage architecture** inspired by [simdjson](https://github.com/simdjson/simdjson) and [minio/simdcsv](https://github.com/minio/simdcsv):

1. **Stage 1: Structural Character Detection**
   - Uses AVX2 SIMD instructions to scan 64-byte chunks in parallel
   - Detects quotes (`"`), delimiters (`,`), and newlines (`\r`, `\n`)
   - Produces 64-bit bitmasks for each character class

2. **Stage 2: Field Extraction**
   - Processes bitmasks to extract actual CSV fields
   - Handles quote state tracking using cumulative XOR
   - Manages escaped quotes (`""`) and chunk boundaries

## Performance

On x86-64 systems with AVX2 support:
- **4-10x faster** than byte-by-byte parsing for large files
- **64 bytes processed per SIMD instruction** (2x YMM registers)
- **Zero-copy** for simple fields (no quotes or escapes)
- **Automatic fallback** to pure Go on non-AVX2 systems

## CPU Requirements

### AVX2 Support (x86-64)
- Intel Haswell (2013) or newer
- AMD Excavator (2015) or newer
- Automatically detected at runtime

### Fallback (All Platforms)
- Pure Go implementation with two-stage architecture
- Same correctness guarantees as SIMD path
- 2-3x slower than AVX2 but still faster than byte-by-byte

### Future: ARM NEON (Apple Silicon)
- Planned support for ARM NEON instructions
- Will use same two-stage architecture
- Currently uses pure Go fallback

## Usage

### Basic Usage

```go
import "github.com/shapestone/shape-csv/internal/fastparser/simd"

// Auto-detect CPU features and use SIMD if available
opts := simd.DefaultParseOptions()
parser := simd.NewParser(opts)
records, err := parser.Parse(data)
```

### Manual Control

```go
// Force SIMD on (will panic if CPU doesn't support AVX2)
opts := simd.ParseOptions{
    UseSIMD:   true,
    Delimiter: ',',
}
parser := simd.NewParser(opts)
records, err := parser.Parse(data)

// Force pure Go fallback
opts := simd.ParseOptions{
    UseSIMD:   false,
    Delimiter: ',',
}
parser := simd.NewParser(opts)
records, err := parser.Parse(data)
```

### Feature Detection

```go
import "github.com/shapestone/shape-csv/internal/fastparser/simd"

if simd.HasAVX2() {
    fmt.Println("AVX2 acceleration available")
} else {
    fmt.Println("Using pure Go fallback")
}
```

## Architecture Details

### Stage 1: Structural Detection (AVX2)

The AVX2 implementation processes 64 bytes per iteration:

```assembly
; Load 64 bytes (2x YMM registers)
VMOVDQU 0(SI), Y0    ; Load bytes 0-31
VMOVDQU 32(SI), Y1   ; Load bytes 32-63

; Compare against quote character
VPBROADCASTB X2, Y2  ; Broadcast '"' to all 32 bytes
VPCMPEQB Y0, Y2, Y3  ; Compare: Y3 = (Y0 == '"') ? 0xFF : 0x00
VPMOVMSKB Y3, R8     ; Extract high bits to 32-bit mask

; Combine two 32-bit masks into 64-bit mask
MOVL R8, R10         ; Lower 32 bits
SHLQ $32, R9         ; Upper 32 bits << 32
ORQ R9, R10          ; 64-bit mask
```

### Stage 2: Quote State Tracking

Uses **cumulative XOR** to determine inside/outside quote state:

```
Input:     " hello " world
Quotes:    1 00000 1 00000   (bitmask)
XOR scan:  1 11111 0 00000   (inside quotes when 1)
```

Escaped quotes (`""`) are handled by detecting adjacent bits:

```
Input:     " he""llo " world
Quotes:    1 00 11 000 1 00000
Escaped:   0 00 11 000 0 00000   (clear adjacent pairs)
Result:    1 00 00 000 1 00000   (two quotes become zero)
```

### Chunk Boundaries

Quote state is tracked across 64-byte chunk boundaries:

```
Chunk 1: [" hello ]  -> ends inside quote (state = true)
Chunk 2: [ world "]  -> starts inside quote (state = true)
```

The parser maintains `QuoteState` between chunks to handle this correctly.

## Implementation Files

- `simd.go` - Main parser and two-stage architecture
- `stage1_amd64.s` - AVX2 assembly for structural detection (x86-64)
- `stage1_amd64.go` - Go wrapper for AVX2 assembly
- `stage1_other.go` - Stub for non-x86-64 platforms
- `stage1_fallback.go` - Pure Go structural detection
- `stage2.go` - Field extraction and quote state tracking
- `cpuinfo_amd64.go` - CPU feature detection (x86-64)
- `cpuinfo_other.go` - Stub for non-x86-64 platforms
- `cpuid_amd64.s` - CPUID instruction wrapper

## Testing

### Unit Tests

```bash
go test -v ./internal/fastparser/simd/
```

### Benchmarks

```bash
# Run all SIMD benchmarks
go test -bench=. ./internal/fastparser/simd/

# Compare SIMD vs fallback
go test -bench='(SIMD|Fallback)' ./internal/fastparser/simd/

# Benchmark specific stages
go test -bench='Stage' ./internal/fastparser/simd/
```

### Expected Results (AVX2 System)

```
BenchmarkStructuralDetectionSIMD-8    50000000    25 ns/op    (2.5 GB/s)
BenchmarkStructuralDetectionFallback-8 10000000   120 ns/op   (533 MB/s)
BenchmarkSIMD_Large_Simple-8            5000    220000 ns/op
BenchmarkFallback_Large_Simple-8        1000    950000 ns/op
```

## References

- [simdjson: Parsing gigabytes of JSON per second](https://github.com/simdjson/simdjson)
- [Parsing Gigabytes of JSON per Second (paper)](https://arxiv.org/abs/1902.08318)
- [minio/simdcsv](https://github.com/minio/simdcsv)
- [Intel Intrinsics Guide - AVX2](https://www.intel.com/content/www/us/en/docs/intrinsics-guide/index.html#avxnewtechs=AVX2)

## Future Enhancements

1. **ARM NEON Support** - SIMD acceleration for Apple Silicon
2. **PCLMULQDQ for Quote Tracking** - Advanced carry-less multiplication for faster quote state
3. **Multi-threading** - Parallel chunk processing for multi-core systems
4. **Adaptive Chunk Sizing** - Optimize chunk size based on cache and data characteristics
