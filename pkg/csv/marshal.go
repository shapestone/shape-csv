package csv

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Buffer pool for marshaling to reduce allocations
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// getBuffer retrieves a buffer from the pool.
func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// putBuffer returns a buffer to the pool.
func putBuffer(buf *bytes.Buffer) {
	// Only return reasonably sized buffers to the pool
	if buf.Cap() < 64*1024 { // 64KB limit
		bufferPool.Put(buf)
	}
}

// Marshaler is the interface implemented by types that can marshal themselves into CSV.
type Marshaler interface {
	MarshalCSV() ([]byte, error)
}

// Unmarshaler is the interface implemented by types that can unmarshal CSV data.
type Unmarshaler interface {
	UnmarshalCSV([]byte) error
}

// Marshal returns the CSV encoding of v.
//
// Marshal traverses the value v, which must be a slice of structs.
// Each struct becomes a row in the CSV, with struct fields becoming columns.
//
// The encoding of each struct field can be customized by the format string
// stored under the "csv" key in the struct field's tag. The format string
// gives the name of the field, possibly followed by a comma-separated list
// of options. The name may be empty in order to specify options without
// overriding the default field name.
//
// The "omitempty" option specifies that the field should be omitted from the
// encoding if the field has an empty value, defined as false, 0, a nil pointer,
// a nil interface value, and any empty array, slice, map, or string.
// Note: In CSV, omitempty means the field is still included in the row, but
// with an empty value. This maintains consistent column structure.
//
// As a special case, if the field tag is "-", the field is always omitted.
//
// Examples of struct field tags and their meanings:
//
//	// Field appears in CSV as "myName"
//	Field int `csv:"myName"`
//
//	// Field appears in CSV as "Field" (default)
//	Field int
//
//	// Field appears in CSV as "myName", empty values appear as ""
//	Field int `csv:"myName,omitempty"`
//
//	// Field is ignored by this package
//	Field int `csv:"-"`
//
// Anonymous struct fields are currently not supported.
//
// Map and slice fields (other than []byte) are not supported.
//
// Pointer values encode as the value pointed to. A nil pointer encodes as
// an empty string.
//
// The CSV header row is auto-generated from struct field names or tags,
// and is sorted alphabetically for deterministic output.
func Marshal(v interface{}) ([]byte, error) {
	// Validate input
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || v == nil {
		return nil, fmt.Errorf("csv: Marshal(nil)")
	}

	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("csv: Marshal expects slice, got %s", rv.Type())
	}

	// Get slice element type (should be struct)
	if rv.Len() == 0 {
		// Empty slice produces empty output
		return []byte{}, nil
	}

	elemType := rv.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if elemType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("csv: Marshal expects slice of structs, got slice of %s", elemType)
	}

	// Build field information
	type fieldEntry struct {
		name      string
		index     int
		omitEmpty bool
	}

	var fields []fieldEntry
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		info := getFieldInfo(field)

		// Skip fields with "-" tag
		if info.skip {
			continue
		}

		fields = append(fields, fieldEntry{
			name:      info.name,
			index:     i,
			omitEmpty: info.omitEmpty,
		})
	}

	// Sort fields by name for deterministic output
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].name < fields[j].name
	})

	buf := getBuffer()
	defer putBuffer(buf)

	// Write header row
	for i, field := range fields {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeField(buf, field.name)
	}
	buf.WriteByte('\n')

	// Write data rows
	for rowIdx := 0; rowIdx < rv.Len(); rowIdx++ {
		row := rv.Index(rowIdx)

		// Handle pointer to struct
		if row.Kind() == reflect.Ptr {
			if row.IsNil() {
				// Skip nil pointers
				continue
			}
			row = row.Elem()
		}

		// Write each field
		for i, field := range fields {
			if i > 0 {
				buf.WriteByte(',')
			}

			fieldVal := row.Field(field.index)

			// Handle omitempty
			if field.omitEmpty && isEmptyValue(fieldVal) {
				// Write empty field (maintains column structure)
				continue
			}

			// Convert field value to string and write
			if err := marshalFieldValue(fieldVal, buf); err != nil {
				return nil, fmt.Errorf("csv: error marshaling field %s: %w", field.name, err)
			}
		}
		buf.WriteByte('\n')
	}

	// Make a copy of the bytes since we're returning the buffer to the pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// marshalFieldValue marshals a single field value to the buffer
func marshalFieldValue(rv reflect.Value, buf *bytes.Buffer) error {
	// Handle invalid values
	if !rv.IsValid() {
		return nil // Empty field
	}

	// Handle pointers
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil // Empty field for nil pointer
		}
		return marshalFieldValue(rv.Elem(), buf)
	}

	// Handle interface
	if rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil // Empty field
		}
		return marshalFieldValue(rv.Elem(), buf)
	}

	switch rv.Kind() {
	case reflect.String:
		writeField(buf, rv.String())
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		writeField(buf, strconv.FormatInt(rv.Int(), 10))
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		writeField(buf, strconv.FormatUint(rv.Uint(), 10))
		return nil

	case reflect.Float32, reflect.Float64:
		writeField(buf, strconv.FormatFloat(rv.Float(), 'g', -1, 64))
		return nil

	case reflect.Bool:
		writeField(buf, strconv.FormatBool(rv.Bool()))
		return nil

	default:
		return fmt.Errorf("unsupported type %s", rv.Type())
	}
}

// writeField writes a CSV field to the buffer with proper escaping
func writeField(buf *bytes.Buffer, value string) {
	// Check if field needs quoting
	needsQuoting := strings.ContainsAny(value, ",\"\n\r")

	if needsQuoting {
		buf.WriteByte('"')
		// Escape quotes by doubling them
		for _, ch := range value {
			if ch == '"' {
				buf.WriteString(`""`)
			} else {
				buf.WriteRune(ch)
			}
		}
		buf.WriteByte('"')
	} else {
		buf.WriteString(value)
	}
}
