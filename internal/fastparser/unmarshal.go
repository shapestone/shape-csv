package fastparser

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Unmarshal parses CSV data and unmarshals it into a slice of structs or [][]string.
//
// For [][]string, it returns all records including the header row:
//
//	var records [][]string
//	err := Unmarshal([]byte(csvData), &records)
//	// records[0] is the header row, records[1:] are data rows
//
// For slice of structs, the first row is treated as headers:
//
//	type Person struct {
//	    Name string `csv:"name"`
//	    Age  int    `csv:"age"`
//	}
//
//	var people []Person
//	err := Unmarshal([]byte(csvData), &people)
//
// Struct tags:
//   - Use `csv:"fieldname"` to specify the CSV column name
//   - If no tag is provided, the field name is used (case-insensitive matching)
//
// Supported types:
//   - string
//   - int, int8, int16, int32, int64
//   - uint, uint8, uint16, uint32, uint64
//   - float32, float64
//   - bool (accepts: true/false, 1/0, t/f, T/F, TRUE/FALSE)
func Unmarshal(data []byte, v interface{}) error {
	// Validate input
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || v == nil {
		return errors.New("csv: Unmarshal(nil)")
	}

	if rv.Kind() != reflect.Ptr {
		return errors.New("csv: Unmarshal(non-pointer " + rv.Type().String() + ")")
	}

	if rv.IsNil() {
		return errors.New("csv: Unmarshal(nil " + rv.Type().String() + ")")
	}

	// Get the element type
	elem := rv.Elem()
	if elem.Kind() != reflect.Slice {
		return errors.New("csv: Unmarshal expects pointer to slice, got " + elem.Type().String())
	}

	// Get the slice element type
	sliceElemType := elem.Type().Elem()

	// Fast path: [][]string - return raw records
	if sliceElemType.Kind() == reflect.Slice && sliceElemType.Elem().Kind() == reflect.String {
		records, err := Parse(data)
		if err != nil {
			return err
		}
		elem.Set(reflect.ValueOf(records))
		return nil
	}

	// Struct path: slice of structs
	if sliceElemType.Kind() != reflect.Struct {
		return errors.New("csv: Unmarshal expects [][]string or slice of structs, got slice of " + sliceElemType.String())
	}

	// Parse CSV
	records, err := Parse(data)
	if err != nil {
		return err
	}

	// Empty data
	if len(records) == 0 {
		elem.Set(reflect.MakeSlice(elem.Type(), 0, 0))
		return nil
	}

	// First row is header
	headers := records[0]
	dataRows := records[1:]

	// Get cached struct info (includes field map and pre-computed setters)
	info := getStructInfo(sliceElemType, headers)

	// Create result slice
	result := reflect.MakeSlice(elem.Type(), 0, len(dataRows))

	// Process each data row
	for rowIdx, row := range dataRows {
		// Create new struct instance
		structVal := reflect.New(sliceElemType).Elem()

		// Populate fields using cached setters
		for colIdx, value := range row {
			if colIdx >= len(headers) {
				// Extra columns beyond headers - ignore
				continue
			}

			// Look up field index for this column
			fieldIdx, ok := info.fieldMap[colIdx]
			if !ok {
				// Column not mapped to any struct field - skip
				continue
			}

			// Get the pre-computed setter for this column
			setter, ok := info.setters[colIdx]
			if !ok {
				// No setter for this column - skip
				continue
			}

			// Get the struct field
			field := structVal.Field(fieldIdx)

			// Use pre-computed setter instead of switch-based setFieldValue
			if err := setter(field, value, rowIdx, colIdx); err != nil {
				return err
			}
		}

		// Append to result
		result = reflect.Append(result, structVal)
	}

	// Set the result
	elem.Set(result)
	return nil
}

// UnmarshalBytes parses CSV data using ByteRecord offset tracking and unmarshals
// it into a slice of structs or [][]string.
//
// This function uses the BurntSushi pattern for memory efficiency:
//   - Fields are tracked as byte offsets, not copied strings
//   - String conversion happens only when accessing struct fields
//   - Significantly reduces allocations for large datasets
//
// For [][]string, it returns all records including the header row:
//
//	var records [][]string
//	err := UnmarshalBytes([]byte(csvData), &records)
//	// records[0] is the header row, records[1:] are data rows
//
// For slice of structs, the first row is treated as headers:
//
//	type Person struct {
//	    Name string `csv:"name"`
//	    Age  int    `csv:"age"`
//	}
//
//	var people []Person
//	err := UnmarshalBytes([]byte(csvData), &people)
//
// This function provides the same functionality as Unmarshal() but with
// better memory efficiency for large datasets.
func UnmarshalBytes(data []byte, v interface{}) error {
	// Validate input
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || v == nil {
		return errors.New("csv: UnmarshalBytes(nil)")
	}

	if rv.Kind() != reflect.Ptr {
		return errors.New("csv: UnmarshalBytes(non-pointer " + rv.Type().String() + ")")
	}

	if rv.IsNil() {
		return errors.New("csv: UnmarshalBytes(nil " + rv.Type().String() + ")")
	}

	// Get the element type
	elem := rv.Elem()
	if elem.Kind() != reflect.Slice {
		return errors.New("csv: UnmarshalBytes expects pointer to slice, got " + elem.Type().String())
	}

	// Get the slice element type
	sliceElemType := elem.Type().Elem()

	// Parse CSV using ByteRecord
	byteRecords, err := ParseByteRecords(data)
	if err != nil {
		return err
	}

	// Fast path: [][]string - convert ByteRecords to string slices
	if sliceElemType.Kind() == reflect.Slice && sliceElemType.Elem().Kind() == reflect.String {
		records := make([][]string, len(byteRecords))
		for i, br := range byteRecords {
			records[i] = br.Fields()
		}
		elem.Set(reflect.ValueOf(records))
		return nil
	}

	// Struct path: slice of structs
	if sliceElemType.Kind() != reflect.Struct {
		return errors.New("csv: UnmarshalBytes expects [][]string or slice of structs, got slice of " + sliceElemType.String())
	}

	// Empty data
	if len(byteRecords) == 0 {
		elem.Set(reflect.MakeSlice(elem.Type(), 0, 0))
		return nil
	}

	// First row is header
	headerRecord := byteRecords[0]
	headers := headerRecord.Fields()
	dataRecords := byteRecords[1:]

	// Get cached struct info (includes field map and pre-computed setters)
	info := getStructInfo(sliceElemType, headers)

	// Create result slice
	result := reflect.MakeSlice(elem.Type(), 0, len(dataRecords))

	// Process each data row
	for rowIdx, record := range dataRecords {
		// Create new struct instance
		structVal := reflect.New(sliceElemType).Elem()

		// Populate fields using cached setters
		for colIdx := 0; colIdx < record.NumFields(); colIdx++ {
			if colIdx >= len(headers) {
				// Extra columns beyond headers - ignore
				continue
			}

			// Look up field index for this column
			fieldIdx, ok := info.fieldMap[colIdx]
			if !ok {
				// Column not mapped to any struct field - skip
				continue
			}

			// Get the pre-computed setter for this column
			setter, ok := info.setters[colIdx]
			if !ok {
				// No setter for this column - skip
				continue
			}

			// Get the struct field
			field := structVal.Field(fieldIdx)

			// Get field value as string (lazy conversion)
			value := record.Field(colIdx)

			// Use pre-computed setter instead of switch-based setFieldValue
			if err := setter(field, value, rowIdx, colIdx); err != nil {
				return err
			}
		}

		// Append to result
		result = reflect.Append(result, structVal)
	}

	// Set the result
	elem.Set(result)
	return nil
}

// parseBool parses a boolean value from a string.
// Accepts: true/false, 1/0, t/f, T/F, TRUE/FALSE (case-insensitive).
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "t":
		return true, nil
	case "false", "0", "f":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %q", s)
	}
}
