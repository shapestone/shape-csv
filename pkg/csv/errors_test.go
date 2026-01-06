package csv_test

import (
	"errors"
	"testing"

	"github.com/shapestone/shape-csv/pkg/csv"
)

func TestBadLineMode_String(t *testing.T) {
	tests := []struct {
		mode csv.BadLineMode
		want string
	}{
		{csv.BadLineModeError, "error"},
		{csv.BadLineModeWarn, "warn"},
		{csv.BadLineModeSkip, "skip"},
		{csv.BadLineMode(99), "BadLineMode(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("BadLineMode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	t.Run("same line", func(t *testing.T) {
		err := &csv.ParseError{
			StartLine: 5,
			Line:      5,
			Column:    10,
			Err:       csv.ErrQuote,
		}

		got := err.Error()
		want := "parse error on line 5, column 10: bare \" in non-quoted-field"
		if got != want {
			t.Errorf("ParseError.Error() = %q, want %q", got, want)
		}
	})

	t.Run("different lines", func(t *testing.T) {
		err := &csv.ParseError{
			StartLine: 3,
			Line:      5,
			Column:    1,
			Err:       errors.New("unclosed quote"),
		}

		got := err.Error()
		want := "parse error on line 5 (started line 3), column 1: unclosed quote"
		if got != want {
			t.Errorf("ParseError.Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		underlying := errors.New("test error")
		err := &csv.ParseError{
			Line: 1,
			Err:  underlying,
		}

		if !errors.Is(err, underlying) {
			t.Error("ParseError.Unwrap() should return the underlying error")
		}
	})
}

func TestDefaultErrorRecoveryOptions(t *testing.T) {
	opts := csv.DefaultErrorRecoveryOptions()

	if opts.OnBadLine != csv.BadLineModeError {
		t.Errorf("DefaultErrorRecoveryOptions().OnBadLine = %v, want BadLineModeError", opts.OnBadLine)
	}
	if opts.BadLineCallback != nil {
		t.Error("DefaultErrorRecoveryOptions().BadLineCallback should be nil")
	}
	if opts.WarningCallback != nil {
		t.Error("DefaultErrorRecoveryOptions().WarningCallback should be nil")
	}
	if opts.MaxFieldSize != 0 {
		t.Errorf("DefaultErrorRecoveryOptions().MaxFieldSize = %d, want 0", opts.MaxFieldSize)
	}
	if opts.MaxRecordSize != 0 {
		t.Errorf("DefaultErrorRecoveryOptions().MaxRecordSize = %d, want 0", opts.MaxRecordSize)
	}
}

func TestCommonErrors(t *testing.T) {
	// Test that common errors are defined
	if csv.ErrQuote == nil {
		t.Error("ErrQuote should not be nil")
	}
	if csv.ErrFieldCount == nil {
		t.Error("ErrFieldCount should not be nil")
	}
	if csv.ErrFieldTooLarge == nil {
		t.Error("ErrFieldTooLarge should not be nil")
	}
	if csv.ErrRecordTooLarge == nil {
		t.Error("ErrRecordTooLarge should not be nil")
	}
}
