// Package csv provides advanced CSV processing features.
package csv

import (
	"reflect"
	"strconv"
	"strings"
)

// EscapeMode specifies how escape characters are handled.
type EscapeMode int

const (
	// EscapeModeRFC4180 uses RFC 4180 double-quote escaping (default).
	EscapeModeRFC4180 EscapeMode = iota
	// EscapeModeBackslash uses backslash escaping for special characters.
	EscapeModeBackslash
)

// AdvancedOptions configures advanced CSV processing features.
type AdvancedOptions struct {
	// EscapeChar is the escape character for EscapeModeBackslash mode.
	// Default: 0 (disabled, use RFC 4180 double-quote escaping)
	EscapeChar rune

	// EscapeMode specifies the escape handling mode.
	// Default: EscapeModeRFC4180
	EscapeMode EscapeMode

	// PreProcess is called for each record before field processing.
	// Can modify fields before they are assigned to struct fields.
	PreProcess func([]string) []string

	// PostProcess is called for each unmarshaled struct after field assignment.
	// Can modify the struct before it's added to the result slice.
	PostProcess func(interface{}) interface{}
}

// DefaultAdvancedOptions returns default advanced options.
func DefaultAdvancedOptions() AdvancedOptions {
	return AdvancedOptions{
		EscapeChar: 0,
		EscapeMode: EscapeModeRFC4180,
	}
}

// MultiValueSeparator is the default separator for multi-value fields.
const MultiValueSeparator = "|"

// advancedFieldInfo extends fieldInfo with advanced options.
type advancedFieldInfo struct {
	fieldInfo
	// split is the separator for multi-value fields (empty = no split)
	split string
	// recurse indicates nested struct should be flattened
	recurse bool
	// converter is the name of a registered converter
	converter string
}

// parseAdvancedTag parses a struct field's csv tag with advanced options.
// Format: "fieldname,option1,option2,split=|,converter=myconv"
func parseAdvancedTag(tag string) advancedFieldInfo {
	info := advancedFieldInfo{
		fieldInfo: parseTag(tag),
	}

	if tag == "-" {
		return info
	}

	parts := strings.Split(tag, ",")

	// Parse advanced options
	for i := 1; i < len(parts); i++ {
		opt := strings.TrimSpace(parts[i])

		if strings.HasPrefix(opt, "split=") {
			info.split = strings.TrimPrefix(opt, "split=")
		} else if strings.HasPrefix(opt, "converter=") {
			info.converter = strings.TrimPrefix(opt, "converter=")
		} else if opt == "recurse" {
			info.recurse = true
		}
	}

	return info
}

// SplitField splits a field value by the given separator.
// Returns a slice of strings.
func SplitField(value string, separator string) []string {
	if value == "" {
		return []string{}
	}
	if separator == "" {
		return []string{value}
	}
	return strings.Split(value, separator)
}

// JoinField joins a slice of values with the given separator.
func JoinField(values []string, separator string) string {
	return strings.Join(values, separator)
}

// ApplyEscapeMode processes a field value according to escape mode.
// Converts backslash-escaped sequences (\n, \r, \t, \\, \") to actual characters.
func ApplyEscapeMode(value string, opts AdvancedOptions) string {
	if opts.EscapeMode != EscapeModeBackslash || opts.EscapeChar == 0 {
		return value
	}

	// Unescape backslash-escaped characters
	var result strings.Builder
	escapeChar := opts.EscapeChar
	escaped := false

	for _, ch := range value {
		if escaped {
			// Handle escaped character
			switch ch {
			case 'n':
				result.WriteRune('\n')
			case 'r':
				result.WriteRune('\r')
			case 't':
				result.WriteRune('\t')
			case escapeChar:
				result.WriteRune(escapeChar)
			case '"':
				result.WriteRune('"')
			default:
				result.WriteRune(ch)
			}
			escaped = false
		} else if ch == escapeChar {
			escaped = true
		} else {
			result.WriteRune(ch)
		}
	}

	return result.String()
}

// EscapeForOutput applies escaping for output according to escape mode.
// Converts special characters to backslash-escaped sequences for output.
func EscapeForOutput(value string, opts AdvancedOptions) string {
	if opts.EscapeMode != EscapeModeBackslash || opts.EscapeChar == 0 {
		return value
	}

	var result strings.Builder
	escapeChar := opts.EscapeChar

	for _, ch := range value {
		switch ch {
		case '\n':
			result.WriteRune(escapeChar)
			result.WriteRune('n')
		case '\r':
			result.WriteRune(escapeChar)
			result.WriteRune('r')
		case '\t':
			result.WriteRune(escapeChar)
			result.WriteRune('t')
		case escapeChar:
			result.WriteRune(escapeChar)
			result.WriteRune(escapeChar)
		case '"':
			result.WriteRune(escapeChar)
			result.WriteRune('"')
		default:
			result.WriteRune(ch)
		}
	}

	return result.String()
}

// FlattenStruct flattens a nested struct into a flat map of field names to values.
// Uses a prefix for nested field names (e.g., "Address.Street").
func FlattenStruct(v interface{}, prefix string) map[string]string {
	result := make(map[string]string)
	flattenValue(reflect.ValueOf(v), prefix, result)
	return result
}

func flattenValue(v reflect.Value, prefix string, result map[string]string) {
	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get field name from tag or struct field
		info := getFieldInfo(field)
		if info.skip {
			continue
		}

		name := info.name
		if prefix != "" {
			name = prefix + "." + name
		}

		// Check for recurse option
		advInfo := parseAdvancedTag(field.Tag.Get("csv"))
		if advInfo.recurse && fieldVal.Kind() == reflect.Struct {
			flattenValue(fieldVal, name, result)
			continue
		}

		// Convert value to string
		result[name] = valueToString(fieldVal)
	}
}

func valueToString(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.String {
			strs := make([]string, v.Len())
			for i := 0; i < v.Len(); i++ {
				strs[i] = v.Index(i).String()
			}
			return strings.Join(strs, MultiValueSeparator)
		}
	}

	return ""
}

// TransformOptions configures field transformation hooks.
type TransformOptions struct {
	// FieldTransform is called for each field value during unmarshal.
	// Can modify the value before type conversion.
	FieldTransform func(fieldName, value string) string

	// RowTransform is called for each row before field assignment.
	// Can modify or filter the record.
	RowTransform func(record []string, headers []string) []string
}

// ProcessWithTransforms applies transformations during CSV processing.
type ProcessWithTransforms struct {
	transforms TransformOptions
	headers    []string
}

// NewProcessWithTransforms creates a processor with transformation hooks.
func NewProcessWithTransforms(opts TransformOptions) *ProcessWithTransforms {
	return &ProcessWithTransforms{
		transforms: opts,
	}
}

// SetHeaders sets the header row for field name lookups.
func (p *ProcessWithTransforms) SetHeaders(headers []string) {
	p.headers = make([]string, len(headers))
	copy(p.headers, headers)
}

// TransformRow applies row transformation.
func (p *ProcessWithTransforms) TransformRow(record []string) []string {
	if p.transforms.RowTransform == nil {
		return record
	}
	return p.transforms.RowTransform(record, p.headers)
}

// TransformField applies field transformation.
func (p *ProcessWithTransforms) TransformField(fieldName, value string) string {
	if p.transforms.FieldTransform == nil {
		return value
	}
	return p.transforms.FieldTransform(fieldName, value)
}
