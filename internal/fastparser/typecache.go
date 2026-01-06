package fastparser

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// fieldSetter is a pre-computed function that sets a field value from a string.
// It takes the field value, the string to parse, row index, and column index.
type fieldSetter func(field reflect.Value, value string, rowIdx, colIdx int) error

// structInfo holds cached metadata about a struct type for a specific header layout.
type structInfo struct {
	// fieldMap maps column index to struct field index
	fieldMap map[int]int

	// setters maps column index to a pre-computed setter function
	setters map[int]fieldSetter
}

// cacheKey uniquely identifies a struct type + header combination
type cacheKey struct {
	typ        reflect.Type
	headerHash string
}

// Global cache for struct metadata
var (
	typeCache sync.Map // map[cacheKey]*structInfo
)

// getStructInfo retrieves or computes struct metadata for the given type and headers.
// Results are cached for performance.
func getStructInfo(structType reflect.Type, headers []string) *structInfo {
	// Generate cache key
	key := cacheKey{
		typ:        structType,
		headerHash: hashHeaders(headers),
	}

	// Check cache first
	if cached, ok := typeCache.Load(key); ok {
		return cached.(*structInfo)
	}

	// Compute struct info
	info := computeStructInfo(structType, headers)

	// Store in cache
	typeCache.Store(key, info)

	return info
}

// computeStructInfo builds the field map and setters for a struct type.
func computeStructInfo(structType reflect.Type, headers []string) *structInfo {
	info := &structInfo{
		fieldMap: make(map[int]int),
		setters:  make(map[int]fieldSetter),
	}

	// Build a map of CSV column names to struct field indices
	csvNameToFieldIdx := make(map[string]int)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get CSV column name from tag or field name
		csvName := field.Name
		tag := field.Tag.Get("csv")
		if tag != "" && tag != "-" {
			// Handle "name,omitempty" format
			if idx := strings.IndexByte(tag, ','); idx >= 0 {
				csvName = tag[:idx]
			} else {
				csvName = tag
			}
		}

		// Store with lowercase for case-insensitive matching
		csvNameToFieldIdx[strings.ToLower(csvName)] = i
	}

	// Match headers to fields and create setters
	for colIdx, header := range headers {
		headerLower := strings.ToLower(header)
		if fieldIdx, ok := csvNameToFieldIdx[headerLower]; ok {
			// Map column to field
			info.fieldMap[colIdx] = fieldIdx

			// Create pre-computed setter for this field
			field := structType.Field(fieldIdx)
			info.setters[colIdx] = createSetter(field.Type)
		}
	}

	return info
}

// createSetter returns a pre-computed setter function for the given field type.
// This avoids the need for a switch statement on every field set operation.
func createSetter(fieldType reflect.Type) fieldSetter {
	switch fieldType.Kind() {
	case reflect.String:
		return func(field reflect.Value, value string, rowIdx, colIdx int) error {
			field.SetString(value)
			return nil
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(field reflect.Value, value string, rowIdx, colIdx int) error {
			if value == "" {
				field.SetInt(0)
				return nil
			}
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("csv: cannot parse %q as int at row %d, column %d: %v", value, rowIdx+1, colIdx, err)
			}
			if field.OverflowInt(i) {
				return fmt.Errorf("csv: value %d overflows %s at row %d, column %d", i, field.Type(), rowIdx+1, colIdx)
			}
			field.SetInt(i)
			return nil
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(field reflect.Value, value string, rowIdx, colIdx int) error {
			if value == "" {
				field.SetUint(0)
				return nil
			}
			u, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return fmt.Errorf("csv: cannot parse %q as uint at row %d, column %d: %v", value, rowIdx+1, colIdx, err)
			}
			if field.OverflowUint(u) {
				return fmt.Errorf("csv: value %d overflows %s at row %d, column %d", u, field.Type(), rowIdx+1, colIdx)
			}
			field.SetUint(u)
			return nil
		}

	case reflect.Float32, reflect.Float64:
		return func(field reflect.Value, value string, rowIdx, colIdx int) error {
			if value == "" {
				field.SetFloat(0)
				return nil
			}
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("csv: cannot parse %q as float at row %d, column %d: %v", value, rowIdx+1, colIdx, err)
			}
			if field.OverflowFloat(f) {
				return fmt.Errorf("csv: value %v overflows %s at row %d, column %d", f, field.Type(), rowIdx+1, colIdx)
			}
			field.SetFloat(f)
			return nil
		}

	case reflect.Bool:
		return func(field reflect.Value, value string, rowIdx, colIdx int) error {
			if value == "" {
				field.SetBool(false)
				return nil
			}
			b, err := parseBool(value)
			if err != nil {
				return fmt.Errorf("csv: cannot parse %q as bool at row %d, column %d: %v", value, rowIdx+1, colIdx, err)
			}
			field.SetBool(b)
			return nil
		}

	default:
		return func(field reflect.Value, value string, rowIdx, colIdx int) error {
			return fmt.Errorf("csv: unsupported field type %s at row %d, column %d", field.Type(), rowIdx+1, colIdx)
		}
	}
}

// hashHeaders creates a stable hash string from headers for cache keying.
// This ensures different header orderings produce different cache keys.
func hashHeaders(headers []string) string {
	// Simple approach: join headers with a delimiter
	// For better performance, could use a faster hash function
	return strings.Join(headers, "\x00")
}

// clearStructCache clears the entire type cache.
// Useful for testing or if you want to free memory.
func clearStructCache() {
	typeCache.Range(func(key, value interface{}) bool {
		typeCache.Delete(key)
		return true
	})
}
