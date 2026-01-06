// Package csv provides CSV dialect detection and header sniffing.
package csv

import (
	"regexp"
	"strings"
	"unicode"
)

// Sniffer detects CSV dialect (delimiter, headers, etc.)
type Sniffer struct {
	sample    string
	delimiter rune
	hasHeader bool
	analyzed  bool
}

// NewSniffer creates a new Sniffer with a sample of CSV data.
// For best results, provide at least 2-3 lines of data.
func NewSniffer(sample string) *Sniffer {
	return &Sniffer{
		sample:   sample,
		analyzed: false,
	}
}

// analyze performs dialect detection on the sample.
func (s *Sniffer) analyze() {
	if s.analyzed {
		return
	}
	s.delimiter = s.detectDelimiter()
	s.hasHeader = s.detectHeader()
	s.analyzed = true
}

// DetectDelimiter returns the detected field delimiter.
// Common delimiters checked: comma, tab, semicolon, pipe.
func (s *Sniffer) DetectDelimiter() rune {
	s.analyze()
	return s.delimiter
}

// detectDelimiter performs the actual delimiter detection.
func (s *Sniffer) detectDelimiter() rune {
	if s.sample == "" {
		return ','
	}

	delimiters := []rune{',', '\t', ';', '|'}
	scores := make(map[rune]int)

	lines := strings.Split(s.sample, "\n")
	if len(lines) == 0 {
		return ','
	}

	// Count occurrences of each delimiter per line
	for _, delim := range delimiters {
		counts := make([]int, 0, len(lines))
		for _, line := range lines {
			if line == "" {
				continue
			}
			count := countDelimiter(line, delim)
			counts = append(counts, count)
		}

		// Score based on consistency across lines
		if len(counts) > 0 && counts[0] > 0 {
			consistent := true
			for i := 1; i < len(counts); i++ {
				if counts[i] != counts[0] {
					consistent = false
					break
				}
			}
			if consistent {
				scores[delim] = counts[0] * 10 // Bonus for consistency
			} else {
				scores[delim] = counts[0]
			}
		}
	}

	// Return delimiter with highest score
	best := ','
	bestScore := 0
	for delim, score := range scores {
		if score > bestScore {
			best = delim
			bestScore = score
		}
	}

	return best
}

// countDelimiter counts occurrences of a delimiter, ignoring quoted sections.
func countDelimiter(line string, delim rune) int {
	count := 0
	inQuotes := false

	for _, ch := range line {
		if ch == '"' {
			inQuotes = !inQuotes
		} else if ch == delim && !inQuotes {
			count++
		}
	}

	return count
}

// HasHeader returns true if the first row appears to be a header.
func (s *Sniffer) HasHeader() bool {
	s.analyze()
	return s.hasHeader
}

// detectHeader uses heuristics to determine if first row is a header.
func (s *Sniffer) detectHeader() bool {
	lines := strings.Split(s.sample, "\n")
	if len(lines) < 2 {
		return false // Need at least 2 lines to compare
	}

	firstLine := lines[0]
	secondLine := ""
	for _, line := range lines[1:] {
		if line != "" {
			secondLine = line
			break
		}
	}

	if secondLine == "" {
		return false
	}

	// Split by detected delimiter
	delim := s.detectDelimiter()
	firstFields := splitByDelimiter(firstLine, delim)
	secondFields := splitByDelimiter(secondLine, delim)

	if len(firstFields) == 0 || len(secondFields) == 0 {
		return false
	}

	// Heuristics:
	// 1. Headers are typically non-numeric
	// 2. Headers often contain underscores or are camelCase
	// 3. Headers don't usually contain special characters like @ or #

	headerScore := 0
	dataScore := 0

	for _, field := range firstFields {
		field = strings.TrimSpace(field)
		if isLikelyHeader(field) {
			headerScore++
		}
		if isLikelyData(field) {
			dataScore++
		}
	}

	return headerScore > dataScore
}

// isLikelyHeader checks if a field looks like a header name.
func isLikelyHeader(s string) bool {
	if s == "" {
		return false
	}

	// Headers are typically text, not numbers
	if isNumeric(s) {
		return false
	}

	// Headers often match common naming patterns
	headerPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`),      // snake_case or identifier
		regexp.MustCompile(`^[a-zA-Z]+[A-Z][a-zA-Z]*$`),     // camelCase
		regexp.MustCompile(`^[A-Z][a-z]+([ ][A-Z][a-z]+)*$`), // Title Case
	}

	for _, pattern := range headerPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}

	return false
}

// isLikelyData checks if a field looks like data rather than a header.
func isLikelyData(s string) bool {
	if s == "" {
		return false
	}

	// Data is often numeric
	if isNumeric(s) {
		return true
	}

	// Data might contain email-like patterns
	if strings.Contains(s, "@") {
		return true
	}

	// Data might be a date
	datePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
		regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`),
	}
	for _, pattern := range datePatterns {
		if pattern.MatchString(s) {
			return true
		}
	}

	return false
}

// isNumeric checks if a string represents a number.
func isNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	// Allow leading minus for negative numbers
	if s[0] == '-' {
		s = s[1:]
	}

	// Check if all remaining chars are digits or decimal point
	hasDot := false
	for _, ch := range s {
		if ch == '.' {
			if hasDot {
				return false
			}
			hasDot = true
		} else if !unicode.IsDigit(ch) {
			return false
		}
	}

	return len(s) > 0
}

// splitByDelimiter splits a line by delimiter, respecting quotes.
func splitByDelimiter(line string, delim rune) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, ch := range line {
		if ch == '"' {
			inQuotes = !inQuotes
			current.WriteRune(ch)
		} else if ch == delim && !inQuotes {
			fields = append(fields, current.String())
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}

	// Don't forget the last field
	fields = append(fields, current.String())

	return fields
}

// HeaderConverter is a function that transforms header names.
type HeaderConverter func(string) string

// LowercaseHeader converts headers to lowercase.
func LowercaseHeader(s string) string {
	return strings.ToLower(s)
}

// UppercaseHeader converts headers to uppercase.
func UppercaseHeader(s string) string {
	return strings.ToUpper(s)
}

// SnakeCaseHeader converts headers to snake_case.
func SnakeCaseHeader(s string) string {
	var result strings.Builder
	prevWasSpace := false
	for i, ch := range s {
		if ch == ' ' {
			if result.Len() > 0 && !prevWasSpace {
				result.WriteRune('_')
			}
			prevWasSpace = true
			continue
		}
		if unicode.IsUpper(ch) && i > 0 && !prevWasSpace {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(ch))
		prevWasSpace = false
	}
	return result.String()
}

// ColumnSelector specifies which columns to include.
type ColumnSelector struct {
	// UseCols selects columns by name.
	UseCols []string
	// UseColIndexes selects columns by index (0-based).
	UseColIndexes []int
}

// ShouldInclude checks if a column should be included.
func (c *ColumnSelector) ShouldInclude(name string, index int) bool {
	// If both are empty, include all columns
	if len(c.UseCols) == 0 && len(c.UseColIndexes) == 0 {
		return true
	}

	// Check by name
	for _, col := range c.UseCols {
		if col == name {
			return true
		}
	}

	// Check by index
	for _, idx := range c.UseColIndexes {
		if idx == index {
			return true
		}
	}

	return false
}
