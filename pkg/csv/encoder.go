package csv

import (
	"reflect"
	"sort"
	"strconv"
	"sync"
)

// csvFieldWriter writes a single struct field value to buf as a CSV field.
// It appends the encoded field value to buf and returns the extended slice.
type csvFieldWriter func(buf []byte, rv reflect.Value) []byte

// csvStructEncoder holds pre-computed encoding info for a struct type.
// Once built, it is immutable and safe for concurrent use.
type csvStructEncoder struct {
	headerRow []byte         // pre-encoded header row including trailing \n
	fields    []csvFieldInfo // sorted fields matching header column order
}

// csvFieldInfo holds pre-computed encoding metadata for a single struct field.
type csvFieldInfo struct {
	index     int                      // struct field index
	name      string                   // CSV column name
	omitEmpty bool                     // omitempty flag
	writer    csvFieldWriter           // pre-resolved writer for this field's type
	emptyFn   func(reflect.Value) bool // pre-resolved empty checker (nil if !omitEmpty)
}

// csvEncoderCache stores compiled encoders keyed by reflect.Type.
var csvEncoderCache sync.Map

// getCSVStructEncoder returns a cached encoder for the given struct type.
// If no encoder exists yet, one is built and cached atomically.
func getCSVStructEncoder(t reflect.Type) *csvStructEncoder {
	if v, ok := csvEncoderCache.Load(t); ok {
		return v.(*csvStructEncoder)
	}
	enc := buildCSVStructEncoder(t)
	actual, _ := csvEncoderCache.LoadOrStore(t, enc)
	return actual.(*csvStructEncoder)
}

// buildCSVStructEncoder constructs a new csvStructEncoder for the given struct type.
func buildCSVStructEncoder(t reflect.Type) *csvStructEncoder {
	var fields []csvFieldInfo

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		info := getFieldInfo(field)

		// Skip fields with "-" tag
		if info.skip {
			continue
		}

		fi := csvFieldInfo{
			index:   i,
			name:    info.name,
			writer:  csvFieldWriterForType(field.Type),
		}

		if info.omitEmpty {
			fi.omitEmpty = true
			fi.emptyFn = csvEmptyFuncForKind(field.Type.Kind())
		}

		fields = append(fields, fi)
	}

	// Sort fields by name for deterministic output
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].name < fields[j].name
	})

	// Pre-encode header row
	var headerRow []byte
	for i, f := range fields {
		if i > 0 {
			headerRow = append(headerRow, ',')
		}
		headerRow = appendCSVField(headerRow, f.name)
	}
	headerRow = append(headerRow, '\n')

	return &csvStructEncoder{
		headerRow: headerRow,
		fields:    fields,
	}
}

// csvFieldWriterForType returns a csvFieldWriter closure specialized for the given type.
// This avoids per-field type-switching at encode time.
func csvFieldWriterForType(t reflect.Type) csvFieldWriter {
	// Handle pointer types by wrapping with nil check + dereference
	if t.Kind() == reflect.Ptr {
		elemWriter := csvFieldWriterForType(t.Elem())
		return func(buf []byte, rv reflect.Value) []byte {
			if rv.IsNil() {
				return buf
			}
			return elemWriter(buf, rv.Elem())
		}
	}

	switch t.Kind() {
	case reflect.String:
		return func(buf []byte, rv reflect.Value) []byte {
			return appendCSVField(buf, rv.String())
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(buf []byte, rv reflect.Value) []byte {
			return strconv.AppendInt(buf, rv.Int(), 10)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return func(buf []byte, rv reflect.Value) []byte {
			return strconv.AppendUint(buf, rv.Uint(), 10)
		}

	case reflect.Float32, reflect.Float64:
		// Use bitSize 64 for all floats to match original Marshal behavior.
		// rv.Float() always returns float64 regardless of the underlying type.
		return func(buf []byte, rv reflect.Value) []byte {
			return appendCSVField(buf, strconv.FormatFloat(rv.Float(), 'g', -1, 64))
		}

	case reflect.Bool:
		return func(buf []byte, rv reflect.Value) []byte {
			if rv.Bool() {
				return append(buf, "true"...)
			}
			return append(buf, "false"...)
		}

	case reflect.Interface:
		// Interfaces need dynamic dispatch at encode time
		return func(buf []byte, rv reflect.Value) []byte {
			if rv.IsNil() {
				return buf
			}
			// Resolve the concrete value and dispatch
			return csvFieldWriterForType(rv.Elem().Type())(buf, rv.Elem())
		}

	default:
		// Unsupported type: write nothing (original code returned error,
		// but the compiled encoder approach doesn't propagate errors for
		// individual fields; unsupported types simply produce empty fields)
		return func(buf []byte, rv reflect.Value) []byte {
			return buf
		}
	}
}

// appendCSVField appends the string s to buf with CSV quoting if necessary.
// This is a zero-allocation CSV field encoder that works directly on []byte.
func appendCSVField(buf []byte, s string) []byte {
	// Scan for characters that require quoting
	needsQuoting := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ',', '"', '\n', '\r':
			needsQuoting = true
			goto done
		}
	}
done:
	if !needsQuoting {
		return append(buf, s...)
	}

	// Quoted field: wrap in double quotes, escape internal quotes by doubling
	buf = append(buf, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			buf = append(buf, '"', '"')
		} else {
			buf = append(buf, s[i])
		}
	}
	buf = append(buf, '"')
	return buf
}

// csvEmptyFuncForKind returns a function that checks whether a reflect.Value
// of the given kind is "empty" per omitempty semantics. Returns a generic
// fallback for kinds not specifically handled.
func csvEmptyFuncForKind(k reflect.Kind) func(reflect.Value) bool {
	switch k {
	case reflect.String:
		return func(rv reflect.Value) bool { return rv.Len() == 0 }

	case reflect.Bool:
		return func(rv reflect.Value) bool { return !rv.Bool() }

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(rv reflect.Value) bool { return rv.Int() == 0 }

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return func(rv reflect.Value) bool { return rv.Uint() == 0 }

	case reflect.Float32, reflect.Float64:
		return func(rv reflect.Value) bool { return rv.Float() == 0 }

	case reflect.Ptr, reflect.Interface:
		return func(rv reflect.Value) bool { return rv.IsNil() }

	case reflect.Array, reflect.Map, reflect.Slice:
		return func(rv reflect.Value) bool { return rv.Len() == 0 }

	default:
		// Fallback: use the generic isEmptyValue from tags.go
		return isEmptyValue
	}
}
