package tokenizer

import (
	"github.com/shapestone/shape-core/pkg/tokenizer"
)

// Options configures the tokenizer behavior.
type Options struct {
	// Comma is the field delimiter. Default: ','
	Comma rune
}

// DefaultOptions returns default tokenizer options.
func DefaultOptions() Options {
	return Options{
		Comma: ',',
	}
}

// NewTokenizer creates a tokenizer for CSV format with default comma delimiter.
// The tokenizer matches CSV tokens in order of specificity.
//
// CSV tokenization works differently from JSON because field content
// depends on context (inside or outside quotes). We tokenize at the
// character level:
// 1. Newlines (CRLF before LF to match longer sequence first)
// 2. Comma (or custom delimiter)
// 3. Double quote
// 4. Field content (any non-delimiter character)
func NewTokenizer() tokenizer.Tokenizer {
	return NewTokenizerWithOptions(DefaultOptions())
}

// NewTokenizerWithOptions creates a tokenizer with custom options.
func NewTokenizerWithOptions(opts Options) tokenizer.Tokenizer {
	return tokenizer.NewTokenizerWithoutWhitespace(
		// Newlines (CRLF before LF for greedy matching)
		tokenizer.StringMatcherFunc(TokenNewline, "\r\n"),
		tokenizer.StringMatcherFunc(TokenNewline, "\n"),

		// Structural tokens - use custom delimiter
		tokenizer.StringMatcherFunc(TokenComma, string(opts.Comma)),
		tokenizer.StringMatcherFunc(TokenDQuote, `"`),

		// Field content (everything else)
		// The parser handles the distinction between quoted and unquoted fields
		FieldContentMatcherWithDelim(opts.Comma),
	)
}

// NewTokenizerWithStream creates a tokenizer for CSV format using a pre-configured stream.
// This is used internally to support streaming from io.Reader.
func NewTokenizerWithStream(stream tokenizer.Stream) tokenizer.Tokenizer {
	return NewTokenizerWithStreamAndOptions(stream, DefaultOptions())
}

// NewTokenizerWithStreamAndOptions creates a tokenizer from a stream with custom options.
func NewTokenizerWithStreamAndOptions(stream tokenizer.Stream, opts Options) tokenizer.Tokenizer {
	tok := NewTokenizerWithOptions(opts)
	tok.InitializeFromStream(stream)
	return tok
}

// FieldContentMatcher creates a matcher for field content with default comma delimiter.
// Matches runs of characters that are not comma, quote, CR, or LF.
func FieldContentMatcher() tokenizer.Matcher {
	return FieldContentMatcherWithDelim(',')
}

// FieldContentMatcherWithDelim creates a matcher for field content with a custom delimiter.
// Matches runs of characters that are not the delimiter, quote, CR, or LF.
//
// Grammar:
//   Field = Character+ ;
//   Character = <any character except delimiter, quote, CR, LF> ;
//
// Note: This matcher only produces TokenField. The distinction between
// quoted and unquoted fields must be handled at the parser level by tracking
// whether we're inside quotes.
//
// Performance: Uses ByteStream for fast ASCII scanning when available.
func FieldContentMatcherWithDelim(delim rune) tokenizer.Matcher {
	return func(stream tokenizer.Stream) *tokenizer.Token {
		// Try ByteStream fast path (only if delimiter is ASCII)
		if delim < 128 {
			if byteStream, ok := stream.(tokenizer.ByteStream); ok {
				return fieldContentMatcherByteWithDelim(byteStream, byte(delim))
			}
		}

		// Fallback to rune-based matcher
		return fieldContentMatcherRuneWithDelim(stream, delim)
	}
}

// fieldContentMatcherByteWithDelim uses ByteStream for optimal performance.
func fieldContentMatcherByteWithDelim(stream tokenizer.ByteStream, delim byte) *tokenizer.Token {
	startPos := stream.BytePosition()

	for {
		b, ok := stream.PeekByte()
		if !ok {
			break
		}

		// Stop at delimiters
		if b == delim || b == '"' || b == '\n' || b == '\r' {
			break
		}

		// Consume the character
		stream.NextByte()
	}

	if stream.BytePosition() == startPos {
		return nil
	}

	value := stream.SliceFrom(startPos)
	return tokenizer.NewToken(TokenField, []rune(string(value)))
}

// fieldContentMatcherRuneWithDelim is the fallback rune-based implementation.
func fieldContentMatcherRuneWithDelim(stream tokenizer.Stream, delim rune) *tokenizer.Token {
	var value []rune

	for {
		r, ok := stream.PeekChar()
		if !ok {
			break
		}

		// Stop at delimiters
		if r == delim || r == '"' || r == '\n' || r == '\r' {
			break
		}

		// Consume the character
		stream.NextChar()
		value = append(value, r)
	}

	if len(value) == 0 {
		return nil
	}

	return tokenizer.NewToken(TokenField, value)
}
