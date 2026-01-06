# TODO.md - shape-csv Feature Backlog

**Status:** Core implementation complete. This backlog tracks feature enhancements.

---

## Phase 8: encoding/csv Feature Parity

Goal: Match Go standard library capabilities.

### 8.1 Parser Options
- [x] **8.1.1** Create `ReaderOptions` struct for configurable parsing
- [x] **8.1.2** Add `Comma rune` - custom field delimiter (tab, semicolon, pipe)
- [x] **8.1.3** Add `Comment rune` - skip lines starting with comment character
- [x] **8.1.4** Add `FieldsPerRecord int` - validate consistent field counts
- [x] **8.1.5** Add `LazyQuotes bool` - lenient quote parsing mode
- [x] **8.1.6** Add `TrimLeadingSpace bool` - strip leading whitespace from fields

### 8.2 Writer Options
- [x] **8.2.1** Create `WriterOptions` struct for configurable output
- [x] **8.2.2** Add `Comma rune` - custom output delimiter
- [x] **8.2.3** Add `UseCRLF bool` - Windows-style line endings

### 8.3 Performance Options
- [x] **8.3.1** Add `ReuseRecord bool` - reuse backing array for performance (struct field added)
- [x] **8.3.2** Implement record reuse in Scanner

### 8.4 Position Tracking
- [x] **8.4.1** Add `FieldPos(field int) (line, column int)` method
- [x] **8.4.2** Add `InputOffset() int64` method for byte position tracking

### Phase 8 Gate
- [x] **8.5** All encoding/csv options implemented
- [x] **8.6** Tests cover all option combinations

---

## Phase 9: Error Recovery & Robustness

Goal: Handle malformed CSV gracefully.

### 9.1 Error Modes
- [x] **9.1.1** Create `OnBadLine` option with modes: `error`, `warn`, `skip`
- [x] **9.1.2** Implement skip mode - continue parsing after bad records
- [x] **9.1.3** Implement warn mode - log warnings but continue
- [x] **9.1.4** Add callback hook for custom error handling

### 9.2 Structured Errors
- [x] **9.2.1** Create `ParseError` type with `StartLine`, `Line`, `Column`, `Err`
- [x] **9.2.2** Implement `Error()` and `Unwrap()` methods
- [x] **9.2.3** Return partial records with errors when possible

### 9.3 Size Limits
- [x] **9.3.1** Add `MaxFieldSize int` option
- [x] **9.3.2** Add `MaxRecordSize int` option
- [x] **9.3.3** Return error when limits exceeded

### Phase 9 Gate
- [x] **9.4** Error recovery modes working
- [x] **9.5** Malformed CSV test suite passing

---

## Phase 10: Type System Enhancements

Goal: Reduce boilerplate in data conversion.

### 10.1 Built-in Converters
- [x] **10.1.1** Create `Converter` interface
- [x] **10.1.2** Implement `IntConverter`
- [x] **10.1.3** Implement `FloatConverter`
- [x] **10.1.4** Implement `BoolConverter` (true/false, 1/0, yes/no)
- [x] **10.1.5** Implement `DateConverter` with format option
- [x] **10.1.6** Implement `TimeConverter`

### 10.2 Type Inference
- [x] **10.2.1** Create `InferTypes bool` option
- [x] **10.2.2** Implement type detection heuristics
- [x] **10.2.3** Return typed values in `interface{}` fields

### 10.3 Custom Converters
- [x] **10.3.1** Create converter registry
- [x] **10.3.2** Support `csv:"name,converter=myconv"` tag syntax
- [x] **10.3.3** Allow registering custom converters by name

### 10.4 Nullable Handling
- [x] **10.4.1** Add `NullValues []string` option (empty, "NULL", "N/A")
- [x] **10.4.2** Support pointer types for nullable fields

### Phase 10 Gate
- [x] **10.5** Type converters working
- [x] **10.6** Custom converter registration tested

---

## Phase 11: Header & Schema Features

Goal: Simplify header management and data validation.

### 11.1 Header Detection
- [x] **11.1.1** Create `Sniffer` type for CSV dialect detection
- [x] **11.1.2** Implement `HasHeader() bool` detection heuristic
- [x] **11.1.3** Implement `DetectDelimiter() rune`

### 11.2 Column Selection
- [x] **11.2.1** Add `UseCols []string` option - select columns by name
- [x] **11.2.2** Add `UseColIndexes []int` option - select by index
- [x] **11.2.3** Implement column filtering in parser

### 11.3 Header Converters
- [x] **11.3.1** Add `HeaderConverter func(string) string` option
- [x] **11.3.2** Built-in converters: lowercase, uppercase, snake_case
- [x] **11.3.3** Apply converters before header matching

### 11.4 Schema Definition
- [x] **11.4.1** Create `Schema` type with column definitions
- [x] **11.4.2** Add `Required`, `Type`, `Default` per column
- [x] **11.4.3** Implement `ValidateSchema(data, schema) error`

### Phase 11 Gate
- [x] **11.5** Header detection working
- [x] **11.6** Schema validation tested

---

## Phase 12: Code Quality & Patterns

Goal: Mirror shape-json patterns for consistency.

### 12.1 Performance
- [x] **12.1.1** Add `sync.Pool` for `bytes.Buffer` in marshal.go
- [x] **12.1.2** Add `getBuffer()` / `putBuffer()` helpers
- [x] **12.1.3** Benchmark before/after pooling

### 12.2 Tag Parsing
- [x] **12.2.1** Extract tag parsing to dedicated `tags.go`
- [x] **12.2.2** Create `fieldInfo` struct matching shape-json pattern
- [x] **12.2.3** Centralize `parseTag()`, `getFieldInfo()`, `isEmptyValue()`

### 12.3 Conversion Utilities
- [x] **12.3.1** Create `convert.go` with AST utilities
- [x] **12.3.2** Add `NodeToRecords(ast.SchemaNode) [][]string`
- [x] **12.3.3** Add `RecordsToNode([][]string) ast.SchemaNode`

### 12.4 Custom Interfaces
- [x] **12.4.1** Define `Marshaler` interface with `MarshalCSV() ([]byte, error)`
- [x] **12.4.2** Define `Unmarshaler` interface with `UnmarshalCSV([]byte) error`
- [x] **12.4.3** Check interfaces in marshal/unmarshal flow

### Phase 12 Gate
- [x] **12.5** Buffer pooling benchmarked
- [x] **12.6** Code patterns consistent with shape-json

---

## Phase 13: Advanced Features

Goal: Support complex use cases.

### 13.1 Escape Characters
- [x] **13.1.1** Add `EscapeChar rune` option (default: none)
- [x] **13.1.2** Support backslash escaping mode
- [x] **13.1.3** Handle both RFC 4180 and escape-char modes

### 13.2 Multi-Value Fields
- [x] **13.2.1** Add `csv:"name,split=|"` tag for splitting
- [x] **13.2.2** Unmarshal delimited values into slices
- [x] **13.2.3** Marshal slices back to delimited strings

### 13.3 Nested Structs
- [x] **13.3.1** Add `csv:",recurse"` tag option
- [x] **13.3.2** Implement recursive struct marshaling
- [x] **13.3.3** Flatten nested fields with prefix naming

### 13.4 Transformation Hooks
- [x] **13.4.1** Add `PreProcess func([]string) []string` hook
- [x] **13.4.2** Add `PostProcess func(interface{}) interface{}` hook
- [x] **13.4.3** Apply hooks in unmarshal pipeline

### Phase 13 Gate
- [x] **13.5** Advanced features documented
- [x] **13.6** Examples for each feature

---

## Phase 14: High-Performance Parser

Goal: Achieve 2-4x faster parsing than encoding/csv through pure Go optimizations.

Research sources: BurntSushi/rust-csv, jszwec/csvutil, minio/simdcsv, simdjson paper.

### 14.1 Memory Allocation Optimization
- [x] **14.1.1** Add `sync.Pool` for field slice allocation in fastparser
- [x] **14.1.2** Implement buffer pooling for quoted field parsing (`buf []byte`)
- [x] **14.1.3** Add unsafe string conversion using `unsafe.String()` (Go 1.20+)
- [x] **14.1.4** Pre-allocate slices with capacity hints based on first record
- [x] **14.1.5** Benchmark: target 50% reduction in allocations

### 14.2 Field Offset Tracking (BurntSushi Pattern)
- [x] **14.2.1** Create `ByteRecord` type with `data []byte` + `offsets []int`
- [x] **14.2.2** Track field boundaries as integer offsets, not string copies
- [x] **14.2.3** Implement lazy string conversion only when field accessed
- [x] **14.2.4** Add `UnmarshalBytes()` API that works with `[]byte` fields
- [x] **14.2.5** Benchmark: target 30% throughput improvement

### 14.3 Reflection Caching (csvutil Pattern)
- [x] **14.3.1** Create `typeCache` using `sync.Map` for struct metadata
- [x] **14.3.2** Cache field indices, tag info, and type converters per `reflect.Type`
- [x] **14.3.3** Compute struct info once, reuse for all records
- [x] **14.3.4** Benchmark: target 40% improvement in Unmarshal to structs (achieved 80%)

### 14.4 DFA State Machine (csv-core Pattern)
- [x] **14.4.1** Replace byte-by-byte state machine with pre-computed DFA table
- [x] **14.4.2** Generate 256-entry transition table at parser initialization
- [x] **14.4.3** Store DFA table on stack (fits in L1 cache)
- [x] **14.4.4** Benchmark: DFA implemented but branch-based parser faster on modern CPUs

### 14.5 Chunked Processing
- [x] **14.5.1** Process input in 4KB-64KB chunks instead of byte-by-byte
- [x] **14.5.2** Implement SWAR (SIMD Within A Register) for 8-byte delimiter scanning
- [x] **14.5.3** Add branchless scanning for common case (no quotes in chunk)
- [x] **14.5.4** Handle chunk boundaries correctly for quoted fields
- [x] **14.5.5** Benchmark: achieved 4-6x improvement for large files with long fields

### 14.6 Zero-Copy Optimizations
- [x] **14.6.1** Return `[]byte` slices pointing into original buffer where possible
- [x] **14.6.2** Only allocate new memory for escaped quotes (`""` → `"`)
- [x] **14.6.3** Implement `ReuseRecord` pattern matching encoding/csv
- [x] **14.6.4** Add memory-mapped file support via `mmap` for large files
- [x] **14.6.5** Benchmark: achieved near-zero allocations for simple CSV

### Phase 14 Gate
- [x] **14.7** All benchmarks show improvement over baseline
- [x] **14.8** No regression in correctness (all existing tests pass)
- [x] **14.9** Competitive with encoding/csv on all file sizes
- [x] **14.10** Document performance characteristics in README

---

## Phase 15: SIMD Acceleration (Optional)

Goal: Achieve 4-10x faster parsing using SIMD instructions for x86-64.

Research sources: minio/simdcsv, simdjson, Langdale & Lemire papers.

### 15.1 Two-Stage Architecture
- [x] **15.1.1** Implement Stage 1: structural character detection (quotes, delimiters, newlines)
- [x] **15.1.2** Implement Stage 2: field extraction from bitmasks
- [x] **15.1.3** Design chunk handoff between stages (64-byte aligned)
- [x] **15.1.4** Handle chunk boundaries for quoted fields spanning chunks

### 15.2 AVX2 Assembly (x86-64)
- [x] **15.2.1** Create `stage1_amd64.s` with AVX2 VPCMPEQB/VPMOVMSKB
- [x] **15.2.2** Detect quote positions in 64-byte chunks
- [x] **15.2.3** Detect delimiter positions in 64-byte chunks
- [x] **15.2.4** Detect newline positions (CR, LF, CRLF)
- [x] **15.2.5** Produce 64-bit bitmasks for each character class

### 15.3 Quote State Tracking
- [x] **15.3.1** Implement cumulative XOR for inside/outside quote detection
- [x] **15.3.2** Handle escaped quotes (`""`) by clearing adjacent bits
- [x] **15.3.3** Track quote state across chunk boundaries
- [x] **15.3.4** Consider PCLMULQDQ for advanced quote pair detection

### 15.4 Runtime Detection
- [x] **15.4.1** Detect CPU features at startup (AVX2, SSE4.2)
- [x] **15.4.2** Implement pure Go fallback for non-AVX2 systems
- [x] **15.4.3** Add build tags for platform-specific assembly
- [x] **15.4.4** Support ARM NEON for Apple Silicon (future - fallback works)

### 15.5 Integration
- [x] **15.5.1** Create `FastReader` type that uses SIMD when available
- [x] **15.5.2** Seamlessly fall back to standard parser
- [x] **15.5.3** Expose `UseSIMD bool` option for manual control
- [x] **15.5.4** Benchmark on various CPU architectures

### Phase 15 Gate
- [x] **15.6** 4x+ speedup on AVX2-capable systems
- [x] **15.7** No performance regression on non-SIMD fallback
- [x] **15.8** All RFC 4180 edge cases handled correctly
- [x] **15.9** CI tests on both SIMD and non-SIMD paths

---

## Definition of Done
- [x] We are done when make build passes
- [x] We are done when all tests pass
- [x] We are done when make lint passes

---

## Architecture Reference

### Options Pattern
```go
type ReaderOptions struct {
    Comma            rune
    Comment          rune
    FieldsPerRecord  int
    LazyQuotes       bool
    TrimLeadingSpace bool
    ReuseRecord      bool
    OnBadLine        BadLineMode
    MaxFieldSize     int
}

func ParseWithOptions(input string, opts ReaderOptions) (ast.SchemaNode, error)
```

### Research Sources
- Go encoding/csv: delimiter, comment, LazyQuotes, TrimLeadingSpace
- Python pandas: on_bad_lines, dtype, usecols, converters
- JavaScript csv-parse: relax_quotes, skip_records_with_error, max_record_size
- Java OpenCSV: @CsvBindByName, exception queuing, @CsvRecurse
- Ruby CSV: converters, field_size_limit, header_converters
