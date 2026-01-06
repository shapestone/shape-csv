// Package fastparser DFA implementation
//
// This file implements a DFA (Deterministic Finite Automaton) based CSV parser
// as an alternative to the byte-by-byte parser in parser.go.
//
// PERFORMANCE NOTE:
// Benchmark results show that the DFA approach is actually slower than the
// original branch-based parser for CSV parsing (~40-73% slower):
//
//	BenchmarkParse_Medium-10       56928    20524 ns/op    23425 B/op    106 allocs/op
//	BenchmarkParseDFA_Medium-10    33818    35665 ns/op    28761 B/op   1106 allocs/op
//
// Reasons:
// 1. Modern CPUs have excellent branch prediction for regular patterns like CSV
// 2. Table lookups add indirection overhead (2 memory loads per character)
// 3. The original parser is already highly optimized with direct comparisons
// 4. CSV parsing benefits from early-exit optimizations that are harder in DFA
//
// The DFA implementation is kept for:
// - Educational value (demonstrates table-driven parsing)
// - Alternative implementation for validation
// - Potential future optimizations (SIMD-assisted DFA)
//
// For production use, prefer Parse() over ParseDFA().
package fastparser

import (
	"errors"
	"fmt"
)

// charClass represents character classes for DFA state machine
type charClass uint8

const (
	classQuote charClass = iota // "
	classComma                   // ,
	classCR                      // \r
	classLF                      // \n
	classOther                   // everything else
	numCharClasses
)

// dfaState represents states in the DFA
type dfaState uint8

const (
	stateStart dfaState = iota
	stateInUnquotedField
	stateInQuotedField
	stateAfterQuote
	stateEndField
	stateEndRecord
	stateError
	numStates
)

// dfaAction represents actions to perform during state transitions
type dfaAction uint8

const (
	actionNone dfaAction = iota
	actionAddChar         // Add character to current field
	actionEndField        // End current field
	actionEndRecord       // End current record
	actionEscapedQuote    // Add escaped quote to field
	actionSkip            // Skip character (e.g., opening quote)
	actionError           // Error condition
	numActions
)

// transition represents a state transition in the DFA
type transition struct {
	nextState dfaState
	action    dfaAction
}

// charClassTable is a 256-entry lookup table for character classification
// This fits in L1 cache and eliminates branches for character classification
var charClassTable [256]charClass

// dfaTransitions is the DFA state transition table
// [currentState][charClass] -> (nextState, action)
var dfaTransitions [numStates][numCharClasses]transition

// init initializes the DFA tables at package initialization
func init() {
	initCharClassTable()
	initDFATransitions()
}

// initCharClassTable initializes the character classification lookup table
func initCharClassTable() {
	// Default all characters to classOther
	for i := 0; i < 256; i++ {
		charClassTable[i] = classOther
	}

	// Set special characters
	charClassTable['"'] = classQuote
	charClassTable[','] = classComma
	charClassTable['\r'] = classCR
	charClassTable['\n'] = classLF
}

// initDFATransitions initializes the DFA state transition table
func initDFATransitions() {
	// stateStart: beginning of a field
	dfaTransitions[stateStart][classQuote] = transition{stateInQuotedField, actionSkip}
	dfaTransitions[stateStart][classComma] = transition{stateEndField, actionEndField}
	dfaTransitions[stateStart][classCR] = transition{stateEndRecord, actionNone}
	dfaTransitions[stateStart][classLF] = transition{stateEndRecord, actionNone}
	dfaTransitions[stateStart][classOther] = transition{stateInUnquotedField, actionAddChar}

	// stateInUnquotedField: reading an unquoted field
	dfaTransitions[stateInUnquotedField][classQuote] = transition{stateError, actionError}
	dfaTransitions[stateInUnquotedField][classComma] = transition{stateEndField, actionEndField}
	dfaTransitions[stateInUnquotedField][classCR] = transition{stateEndRecord, actionNone}
	dfaTransitions[stateInUnquotedField][classLF] = transition{stateEndRecord, actionNone}
	dfaTransitions[stateInUnquotedField][classOther] = transition{stateInUnquotedField, actionAddChar}

	// stateInQuotedField: reading a quoted field
	dfaTransitions[stateInQuotedField][classQuote] = transition{stateAfterQuote, actionNone}
	dfaTransitions[stateInQuotedField][classComma] = transition{stateInQuotedField, actionAddChar}
	dfaTransitions[stateInQuotedField][classCR] = transition{stateInQuotedField, actionAddChar}
	dfaTransitions[stateInQuotedField][classLF] = transition{stateInQuotedField, actionAddChar}
	dfaTransitions[stateInQuotedField][classOther] = transition{stateInQuotedField, actionAddChar}

	// stateAfterQuote: just read a quote inside a quoted field
	dfaTransitions[stateAfterQuote][classQuote] = transition{stateInQuotedField, actionEscapedQuote}
	dfaTransitions[stateAfterQuote][classComma] = transition{stateEndField, actionEndField}
	dfaTransitions[stateAfterQuote][classCR] = transition{stateEndRecord, actionNone}
	dfaTransitions[stateAfterQuote][classLF] = transition{stateEndRecord, actionNone}
	dfaTransitions[stateAfterQuote][classOther] = transition{stateError, actionError}

	// stateEndField and stateEndRecord are terminal states within a loop iteration
	// They don't have meaningful transitions (handled by parser logic)
	for c := charClass(0); c < numCharClasses; c++ {
		dfaTransitions[stateEndField][c] = transition{stateError, actionError}
		dfaTransitions[stateEndRecord][c] = transition{stateError, actionError}
	}

	// stateError has no valid transitions
	for c := charClass(0); c < numCharClasses; c++ {
		dfaTransitions[stateError][c] = transition{stateError, actionError}
	}
}

// ParseDFA parses CSV data using a DFA (Deterministic Finite Automaton).
// This is optimized for performance using pre-computed transition tables.
func ParseDFA(data []byte) ([][]string, error) {
	if len(data) == 0 {
		return [][]string{}, nil
	}

	records := make([][]string, 0, 16)
	var currentRecord []string
	var currentField []byte
	var capacityHint int

	state := stateStart
	pos := 0
	length := len(data)

	// Helper to save current field
	saveField := func() {
		currentRecord = append(currentRecord, string(currentField))
		currentField = currentField[:0]
	}

	// Helper to save current record
	saveRecord := func() {
		if len(currentRecord) > 0 {
			if capacityHint == 0 {
				capacityHint = len(currentRecord)
			}
			records = append(records, currentRecord)
			currentRecord = nil
		}
	}

	// Initialize first record
	if capacityHint > 0 {
		currentRecord = make([]string, 0, capacityHint)
	} else {
		currentRecord = getFieldSlice()
	}

	// Main parsing loop
	for pos < length {
		char := data[pos]
		class := charClassTable[char]

		// Handle empty lines (skip CR/LF when not in a quoted field and no field started)
		if state == stateStart && (class == classCR || class == classLF) {
			// Check if we have any fields in current record
			if len(currentRecord) == 0 && len(currentField) == 0 {
				// Empty line, skip it
				pos++
				continue
			}
		}

		// Get transition
		trans := dfaTransitions[state][class]

		// Handle errors
		if trans.action == actionError {
			// Return field slice to pool on error if we got it from pool
			if capacityHint == 0 && currentRecord != nil {
				putFieldSlice(currentRecord)
			}
			if state == stateInUnquotedField && class == classQuote {
				return nil, fmt.Errorf("quote character in unquoted field at position %d", pos)
			}
			if state == stateAfterQuote && class == classOther {
				return nil, fmt.Errorf("invalid character after closing quote at position %d", pos)
			}
			return nil, fmt.Errorf("parse error at position %d", pos)
		}

		// Execute action
		switch trans.action {
		case actionAddChar:
			currentField = append(currentField, char)

		case actionEscapedQuote:
			currentField = append(currentField, '"')

		case actionEndField:
			saveField()

		case actionSkip:
			// Just skip the character (e.g., opening quote)

		case actionNone:
			// No action needed (e.g., closing quote)
		}

		// Update state
		state = trans.nextState

		// Move to next character
		pos++

		// Handle end of field
		if state == stateEndField {
			state = stateStart
		}

		// Handle end of record
		if state == stateEndRecord {
			saveField()
			// Skip LF after CR in CRLF
			if pos < length && char == '\r' && data[pos] == '\n' {
				pos++
			}
			saveRecord()
			state = stateStart
			// Initialize next record
			if pos < length {
				if capacityHint > 0 {
					currentRecord = make([]string, 0, capacityHint)
				} else {
					currentRecord = getFieldSlice()
				}
			}
		}
	}

	// Handle EOF
	if state == stateInQuotedField {
		// Return field slice to pool on error if we got it from pool
		if capacityHint == 0 && currentRecord != nil {
			putFieldSlice(currentRecord)
		}
		return nil, errors.New("unclosed quoted field")
	}

	// Save final field and record if any
	if state == stateInUnquotedField || state == stateAfterQuote {
		// We were in the middle of a field, save it
		saveField()
	} else if state == stateStart && currentRecord != nil && (len(currentRecord) > 0 || len(currentField) > 0) {
		// We just ended a field with a delimiter, but haven't started the next one
		// This means there's an empty final field to save
		saveField()
	}

	if len(currentRecord) > 0 {
		saveRecord()
	} else if capacityHint == 0 && currentRecord != nil {
		putFieldSlice(currentRecord)
	}

	return records, nil
}
