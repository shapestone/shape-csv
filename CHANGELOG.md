# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- `csv.Validate` and `csv.ValidateReader` check column counts again: every record must have as many fields as the first, so `a,b\n1,2,3` and `a,b\n1` are invalid (as in shape v0.4.0 and `encoding/csv`). The check landed in the fast validator, which keeps its strict handling of bad quotes (`a,"b"c`, `a,b"c`, unclosed quotes). Blank lines are still skipped and quoted fields may span lines.
- The field-count error now reports the line the offending record starts on. The tokenizer-based parser (`ParseWithOptions` with `FieldsPerRecord` set) used to say "line 1" for every defect.

### Added
- `csv.ParseLine(line string) ([]string, error)`: parses one RFC 4180 record into fields. It replaces `ParseLine` from shape v0.4.0 and adds an error for an unclosed quote, a bare quote in an unquoted field, text after a closing quote, or more than one record. It also accepts quoted fields that contain line breaks and one trailing line terminator.

### Changed
- The field-count error from `csv.Validate` is a `*csv.ParseError` wrapping `csv.ErrFieldCount`, so `errors.Is(err, csv.ErrFieldCount)` works.
- Behaviour that differs from shape v0.4.0 `Validate`, kept on purpose: empty input is valid (as in `encoding/csv`; v0.4.0 rejected it), and multi-line quoted fields are valid records (v0.4.0 split on every newline and rejected them).
