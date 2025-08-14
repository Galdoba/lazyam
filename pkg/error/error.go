package lazyerror

import "fmt"

// LazyError - reprezents custom error type to defirentiate experced errors from unexpected.
type LazyError struct {
	originalError error
	isExpected    bool
	format        string
	args          []any
}

// NewExpected - create new expected LazyError.
func NewExpected(err error, format string, args ...any) *LazyError {
	return &LazyError{
		originalError: err,
		isExpected:    true,
		format:        format,
		args:          args,
	}

}

// NewExpected - create new expected LazyError.
func NewUnexpected(format string, args ...any) *LazyError {
	return &LazyError{
		originalError: fmt.Errorf(format, args...),
		isExpected:    false,
		format:        format,
		args:          args,
	}

}

// Error - Fullfill Error interface. Funntion returns original error if format is empty, or formatted string if not.
func (lerr *LazyError) Error() string {
	if lerr.isExpected {
		return fmt.Sprintf(lerr.format, lerr.args...)
	}
	return lerr.originalError.Error()
}

// FormatArgs - return format strinng and arguments of LazyError for logging.
func (lerr *LazyError) FormatArgs() (string, []any) {
	return lerr.format, lerr.args
}

func (lerr *LazyError) IsExpected() bool {
	return lerr.isExpected
}
