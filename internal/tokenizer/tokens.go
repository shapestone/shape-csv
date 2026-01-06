// Package tokenizer provides CSV tokenization using Shape's tokenizer framework.
package tokenizer

// Token type constants for CSV format.
// These correspond to the terminals in the CSV grammar (RFC 4180).
//
// Note: The tokenizer emits simple character-level tokens. The parser is
// responsible for interpreting quotes and determining field boundaries.
const (
	// Structural tokens
	TokenComma   = "Comma"   // , (field separator)
	TokenDQuote  = "DQuote"  // " (quote delimiter)
	TokenNewline = "Newline" // \n or \r\n (line terminator)

	// Field content token
	TokenField = "Field" // Field content (any non-delimiter character)

	// Special token
	TokenEOF = "EOF" // End of file
)
