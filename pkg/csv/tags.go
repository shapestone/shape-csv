package csv

import (
	"reflect"
	"strings"
)

// fieldInfo contains parsed information from a struct field's csv tag
type fieldInfo struct {
	name      string // CSV field name (empty means use Go field name)
	omitEmpty bool   // omitempty option
	skip      bool   // skip this field (tag is "-")
}

// parseTag parses a struct field's csv tag value
// Format: "fieldname" or "fieldname,option1,option2"
// Options: omitempty
// Special: "-" means skip field
func parseTag(tag string) fieldInfo {
	info := fieldInfo{}

	if tag == "-" {
		info.name = "-"
		info.skip = true
		return info
	}

	parts := strings.Split(tag, ",")
	if len(parts) > 0 {
		info.name = parts[0]
	}

	// Parse options
	for i := 1; i < len(parts); i++ {
		switch strings.TrimSpace(parts[i]) {
		case "omitempty":
			info.omitEmpty = true
		}
	}

	return info
}

// getFieldInfo extracts field information from a struct field
// Returns fieldInfo with the CSV name and options
func getFieldInfo(field reflect.StructField) fieldInfo {
	tag := field.Tag.Get("csv")

	info := parseTag(tag)

	// If no name specified in tag, use the Go field name
	if info.name == "" && !info.skip {
		info.name = field.Name
	}

	return info
}

// isEmptyValue reports whether v is empty according to omitempty rules
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
