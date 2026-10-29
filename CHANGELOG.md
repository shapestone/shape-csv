# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed (breaking for v0.9.x callers of `Validate`)
- **`csv.Validate` and `csv.ValidateReader` now reject ragged rows.** Every record must have as many fields as the first one, so `a,b\n1,2,3` and `a,b\n1` are invalid. v0.9.x accepted them, so any code that validated ragged CSV and treated it as valid will start getting errors. This restores shape v0.4.0 and `encoding/csv` behaviour. `ValidateWithOptions` with default options still allows ragged records. Blank lines are skipped and quoted fields may span lines.
- Line breaks are `\n`, `\r\n` and a lone `\r`. A lone `\r` ends a record and counts as one line in error positions. `encoding/csv` instead keeps a lone `\r` inside the field, so the two disagree on such input.
- Errors from `Validate`/`ValidateReader` are now `*csv.ParseError` values. A field-count failure wraps `csv.ErrFieldCount` ("wrong number of fields (got N, expected M)"); malformed quoting (unclosed quote, bare quote in an unquoted field, text after a closing quote) wraps `csv.ErrQuote`. Both carry the line (and, for quotes, column) of the problem, so `errors.Is` and `errors.As` work.
- Behaviour that differs from shape v0.4.0 `Validate`, kept on purpose: empty input is valid (as in `encoding/csv`; v0.4.0 rejected it), and multi-line quoted fields are valid records (v0.4.0 split on every newline and rejected them). Quote errors such as `a,"b"c`, `a,b"c` and an unclosed quote stay invalid, which v0.4.0 accepted.

### Fixed
- The field-count error from the tokenizer-based parser (`ParseWithOptions` with `FieldsPerRecord` set) reported "line 1" for every defect; it now reports the line the offending record starts on.

### Added
- `csv.ParseLine(line string) ([]string, error)`: parses one RFC 4180 record into fields. It replaces `ParseLine` from shape v0.4.0 and adds an error for an unclosed quote, a bare quote in an unquoted field, text after a closing quote, or more than one record. It also accepts quoted fields that contain line breaks and one trailing line terminator.
