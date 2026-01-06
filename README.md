# shape-csv

![Build Status](https://github.com/shapestone/shape-csv/actions/workflows/ci.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/shapestone/shape-csv)](https://goreportcard.com/report/github.com/shapestone/shape-csv)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![codecov](https://codecov.io/gh/shapestone/shape-csv/branch/main/graph/badge.svg)](https://codecov.io/gh/shapestone/shape-csv)
![Go Version](https://img.shields.io/github/go-mod/go-version/shapestone/shape-csv)
[![GoDoc](https://pkg.go.dev/badge/github.com/shapestone/shape-csv.svg)](https://pkg.go.dev/github.com/shapestone/shape-csv)

[![CodeQL](https://github.com/shapestone/shape-csv/actions/workflows/codeql.yml/badge.svg)](https://github.com/shapestone/shape-csv/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/shapestone/shape-csv/badge)](https://securityscorecards.dev/viewer/?uri=github.com/shapestone/shape-csv)

A high-performance CSV parser for Go that generates Shape AST representations. Part of the Shape ecosystem for unified data format handling.

## Features

- **RFC 4180 Compliant**: Full support for quoted fields, escaped quotes, and embedded newlines
- **High Performance**: a bit faster than encoding/csv with fewer allocations
- **Type-Safe Unmarshal/Marshal**: Convert CSV to/from Go structs with struct tags
- **Custom Interfaces**: Implement `Marshaler`/`Unmarshaler` for complex types
- **Streaming Support**: Memory-efficient parsing of large files with `Scanner`
- **DOM API**: `Document` type for programmatic CSV manipulation
- **Dialect Detection**: Auto-detect delimiter and header presence with `Sniffer`
- **Schema Validation**: Define and validate CSV structure with required fields, types, and constraints
- **Type Converters**: Built-in converters for int, float, bool, date, time with type inference
- **Error Recovery**: Skip/warn/error modes for malformed CSV with structured `ParseError`
- **Size Limits**: `MaxFieldSize` and `MaxRecordSize` to prevent memory exhaustion
- **Position Tracking**: `FieldPos()` and `InputOffset()` for precise error reporting
- **Configurable Parsing**: encoding/csv compatible options plus `EscapeChar` for backslash escaping
- **Multi-Value Fields**: Split delimited values into slices with `csv:"field,split=|"`
- **Nested Structs**: Flatten embedded structs with `csv:",recurse"`
- **Transformation Hooks**: Pre/post processing hooks for custom field and row transformations
- **Thread-Safe**: All functions are safe for concurrent use

## Installation

```bash
go get github.com/shapestone/shape-csv
```

## Quick Start

### Parse CSV to AST

```go
import "github.com/shapestone/shape-csv/pkg/csv"

csvData := `name,age,city
Alice,30,New York
Bob,25,Los Angeles`

node, err := csv.Parse(csvData)
if err != nil {
    log.Fatal(err)
}
// node is *ast.ArrayDataNode containing records
```

### Unmarshal to Structs

```go
type Person struct {
    Name string `csv:"name"`
    Age  int    `csv:"age"`
    City string `csv:"city"`
}

var people []Person
err := csv.Unmarshal([]byte(csvData), &people)
// people now contains structured data
```

### Marshal from Structs

```go
people := []Person{
    {Name: "Alice", Age: 30, City: "NYC"},
    {Name: "Bob", Age: 25, City: "LA"},
}

output, err := csv.Marshal(people)
// output is CSV bytes with header row
```

### Streaming Large Files

```go
file, _ := os.Open("large.csv")
defer file.Close()

scanner := csv.NewScanner(file)
scanner.SetHasHeaders(true)

for scanner.Scan() {
    record := scanner.Record()
    name, _ := record.GetByName("name")
    // Process each record
}
```

### Validation

```go
if err := csv.Validate(csvData); err != nil {
    fmt.Println("Invalid CSV:", err)
}
```

## API Reference

### Core Parsing

| Function | Description |
|----------|-------------|
| `Parse(string)` | Parse CSV string to AST |
| `ParseReader(io.Reader)` | Parse CSV from any reader |
| `Validate(string)` | Validate CSV without AST (fast) |
| `ValidateReader(io.Reader)` | Validate CSV from reader |

### Marshal/Unmarshal

| Function | Description |
|----------|-------------|
| `Unmarshal([]byte, interface{})` | CSV bytes to Go structs |
| `Marshal(interface{})` | Go structs to CSV bytes |

### DOM API

| Type/Function | Description |
|---------------|-------------|
| `Document` | In-memory CSV document |
| `NewDocument()` | Create empty document |
| `ParseDocument(string)` | Parse string to Document |
| `Record` | Single CSV record |

### Streaming

| Type/Function | Description |
|---------------|-------------|
| `Scanner` | Streaming CSV reader |
| `NewScanner(io.Reader)` | Create scanner from reader |
| `SetHasHeaders(bool)` | Configure header handling |
| `Scan()` | Advance to next record |
| `Record()` | Get current record |

### Conversion

| Function | Description |
|----------|-------------|
| `NodeToInterface(ast.SchemaNode)` | AST to `[][]string` |
| `InterfaceToNode(interface{})` | `[][]string` to AST |
| `NodeToRecords(ast.SchemaNode)` | AST to `[][]string` (convenience) |
| `RecordsToNode([][]string)` | `[][]string` to AST (convenience) |
| `Render(ast.SchemaNode)` | AST to CSV bytes |

### CSV Dialect Detection (Sniffer)

Automatically detect CSV format from sample data:

```go
sample := "name;age;city\nAlice;30;NYC"
sniffer := csv.NewSniffer(sample)

delimiter := sniffer.DetectDelimiter() // ';'
hasHeader := sniffer.HasHeader()       // true
```

### Schema Validation

Define and validate CSV structure:

```go
schema := csv.NewSchema().
    AddRequiredColumn("name", csv.ColumnTypeString).
    AddColumn(csv.ColumnDefinition{
        Name:          "age",
        Type:          csv.ColumnTypeInt,
        Required:      true,
    }).
    AddColumn(csv.ColumnDefinition{
        Name:          "status",
        Type:          csv.ColumnTypeString,
        AllowedValues: []string{"active", "inactive"},
    })

data := [][]string{
    {"name", "age", "status"},
    {"Alice", "30", "active"},
}

result := csv.ValidateSchema(data, schema)
if !result.Valid {
    fmt.Println(result.AllErrors())
}
```

Generate schema from struct:

```go
type Person struct {
    Name  string `csv:"name"`
    Age   int    `csv:"age,required"`
}

schema, _ := csv.SchemaFromStruct(Person{})
```

### Type Converters

Built-in converters for common types:

```go
registry := csv.NewConverterRegistry()

// Built-in: int, float, bool, date, time, datetime
conv, _ := registry.Get("date")
value, _ := conv.Convert("2024-01-15")

// Type inference
typeName, value := csv.InferType("123")  // "int", int64(123)

// Null value detection
isNull := csv.IsNullValue("N/A", csv.DefaultNullValues)  // true
```

### Header Converters

Transform header names:

```go
csv.LowercaseHeader("FirstName")  // "firstname"
csv.UppercaseHeader("name")       // "NAME"
csv.SnakeCaseHeader("firstName")  // "first_name"
```

### Column Selection

Filter columns by name or index:

```go
selector := csv.ColumnSelector{
    UseCols: []string{"name", "email"},
    // or by index:
    // UseColIndexes: []int{0, 2},
}

if selector.ShouldInclude("name", 0) {
    // process column
}
```

### Transformation Hooks

Apply custom transformations during processing:

```go
opts := csv.TransformOptions{
    FieldTransform: func(name, value string) string {
        return strings.TrimSpace(value)
    },
    RowTransform: func(record, headers []string) []string {
        // Add computed field
        return append(record, "computed")
    },
}

proc := csv.NewProcessWithTransforms(opts)
proc.SetHeaders(headers)
transformed := proc.TransformRow(record)
```

### Parser Options

Configure parsing behavior (encoding/csv compatible):

```go
opts := csv.DefaultReaderOptions()
opts.Comma = '\t'           // Tab-separated
opts.Comment = '#'          // Skip comment lines
opts.LazyQuotes = true      // Lenient quote parsing
opts.TrimLeadingSpace = true
opts.EscapeChar = '\\'      // Backslash escaping (alternative to RFC 4180 doubling)

node, err := csv.ParseWithOptions(data, opts)
```

### Writer Options

Configure output format:

```go
opts := csv.DefaultWriterOptions()
opts.Comma = ';'       // Semicolon-separated
opts.UseCRLF = true    // Windows line endings

output, err := csv.RenderWithOptions(node, opts)
```

### Error Recovery

Handle malformed CSV gracefully:

```go
opts := csv.DefaultReaderOptions()
opts.OnBadLine = csv.BadLineSkip    // Skip bad lines (or BadLineWarn, BadLineError)
opts.MaxFieldSize = 1024 * 1024     // 1MB max field size
opts.MaxRecordSize = 10 * 1024 * 1024 // 10MB max record size

// Structured errors with position info
if err != nil {
    if parseErr, ok := err.(*csv.ParseError); ok {
        fmt.Printf("Error at line %d, column %d: %v\n",
            parseErr.Line, parseErr.Column, parseErr.Err)
    }
}
```

### Position Tracking

Track byte positions during streaming:

```go
scanner := csv.NewScanner(reader)
for scanner.Scan() {
    line, col := scanner.FieldPos(0)  // Position of first field
    offset := scanner.InputOffset()    // Byte offset in input
    // ...
}
```

### Custom Marshaler/Unmarshaler

Implement custom serialization for complex types:

```go
type Money struct {
    Amount   int
    Currency string
}

func (m Money) MarshalCSV() ([]byte, error) {
    return []byte(fmt.Sprintf("%s %d", m.Currency, m.Amount)), nil
}

func (m *Money) UnmarshalCSV(data []byte) error {
    _, err := fmt.Sscanf(string(data), "%s %d", &m.Currency, &m.Amount)
    return err
}
```

## Struct Tags

Use `csv` struct tags to control field mapping:

```go
type Record struct {
    ID        int     `csv:"id"`
    FullName  string  `csv:"full_name"`
    Score     float64 `csv:"score,omitempty"`
    Internal  string  `csv:"-"` // Ignored
}
```

Supported tag options:
- `csv:"name"` - Map to column name
- `csv:"name,omitempty"` - Omit if empty when marshaling
- `csv:"-"` - Skip this field
- `csv:"name,split=|"` - Split multi-value fields by separator
- `csv:"name,converter=int"` - Use named type converter
- `csv:",recurse"` - Flatten nested structs

## Performance

shape-csv is faster than encoding/csv with significantly fewer allocations:

- **2.5x faster** for small files, **1.1x faster** for large files
- **5,000x fewer allocations** for large files (4 vs 20,039)
- **21% less memory** usage

Run `make bench-vs-stdlib` to see current benchmarks.

## RFC 4180 Compliance

This parser implements RFC 4180 with the following rules:

- Fields separated by commas (`,`)
- Records separated by CRLF (`\r\n`) or LF (`\n`)
- Fields containing commas, newlines, or quotes must be quoted
- Quotes within quoted fields escaped as `""`
- Optional header row

## Examples

See the `examples/` directory:

- `examples/main.go` - Basic parsing and marshaling
- `examples/parse_reader/main.go` - Streaming large files
- `examples/encoding_csv_api/main.go` - Drop-in replacement patterns

Run examples:
```bash
go run examples/main.go
go run examples/parse_reader/main.go
go run examples/encoding_csv_api/main.go
```

## Development

```bash
make test      # Run tests
make lint      # Run linter
make build     # Build packages
make bench     # Run benchmarks
make coverage  # Generate coverage report
make all       # Run all checks
```

## License

Apache License 2.0

Copyright 2025-2026 Shapestone
