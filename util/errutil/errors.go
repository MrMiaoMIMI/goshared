// Package errutil provides error handling utility functions.
package errutil

import (
	"errors"
	"fmt"
	"strings"
)

// MultiError collects multiple errors into one.
type MultiError struct {
	errs []error
}

// NewMultiError creates a new empty MultiError.
func NewMultiError() *MultiError {
	return &MultiError{}
}

// Add appends a non-nil error.
func (m *MultiError) Add(err error) {
	if err != nil {
		m.errs = append(m.errs, err)
	}
}

// HasErrors returns true if any errors were collected.
func (m *MultiError) HasErrors() bool {
	return len(m.errs) > 0
}

// Errors returns all collected errors.
func (m *MultiError) Errors() []error {
	return m.errs
}

// ErrorOrNil returns nil if no errors were collected, otherwise returns self.
func (m *MultiError) ErrorOrNil() error {
	if !m.HasErrors() {
		return nil
	}
	return m
}

// Error implements the error interface.
func (m *MultiError) Error() string {
	msgs := make([]string, len(m.errs))
	for i, err := range m.errs {
		msgs[i] = err.Error()
	}
	return fmt.Sprintf("%d errors: [%s]", len(m.errs), strings.Join(msgs, "; "))
}

// Unwrap returns the list of underlying errors (Go 1.20+ multi-error support).
func (m *MultiError) Unwrap() []error {
	return m.errs
}

// IgnoreErr calls fn and silently discards any error.
// Useful for deferred cleanup calls where errors are not critical.
func IgnoreErr(fn func() error) {
	_ = fn()
}

// Must panics if err is non-nil, otherwise returns v.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// MustNoErr panics if err is non-nil.
func MustNoErr(err error) {
	if err != nil {
		panic(err)
	}
}

// FirstError returns the first non-nil error from the list.
func FirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Recover calls fn and converts any panic into an error.
func Recover(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("panic: %v", r)
			}
		}
	}()
	fn()
	return
}

// IsAny checks if err matches any of the target errors.
func IsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// WrapIf wraps err with a message only if err is non-nil.
func WrapIf(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// WrapfIf wraps err with a formatted message only if err is non-nil.
func WrapfIf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}
