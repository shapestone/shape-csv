package csv

import (
	"fmt"
	"reflect"
)

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

	if rv.Len() == 0 {
		return []byte{}, nil
	}

	elemType := rv.Type().Elem()
	isPtr := elemType.Kind() == reflect.Ptr
	if isPtr {
		elemType = elemType.Elem()
	}

	if elemType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("csv: Marshal expects slice of structs, got slice of %s", elemType)
	}

	// Get cached compiled encoder for this struct type
	enc := getCSVStructEncoder(elemType)

	// Estimate capacity: header + avg row size * num rows
	estimatedSize := len(enc.headerRow) + rv.Len()*len(enc.headerRow)
	buf := make([]byte, 0, estimatedSize)

	// Write pre-computed header
	buf = append(buf, enc.headerRow...)

	// Write data rows
	for rowIdx := 0; rowIdx < rv.Len(); rowIdx++ {
		row := rv.Index(rowIdx)

		// Handle pointer to struct
		if isPtr {
			if row.IsNil() {
				continue
			}
			row = row.Elem()
		}

		for i, field := range enc.fields {
			if i > 0 {
				buf = append(buf, ',')
			}

			fv := row.Field(field.index)

			// Handle omitempty: field column stays but value is empty
			if field.omitEmpty && field.emptyFn(fv) {
				continue
			}

			buf = field.writer(buf, fv)
		}
		buf = append(buf, '\n')
	}

	return buf, nil
}

