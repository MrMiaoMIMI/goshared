// Package validator provides small composable validation rules.
package validator

import (
	"cmp"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Error represents one validation failure.
type Error struct {
	Field   string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Errors is a collection of validation failures.
type Errors []*Error

func (e Errors) Error() string {
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

// Err returns nil when there are no validation failures.
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}
	return e
}

// Rule is one validation rule. It returns nil when the value is valid.
type Rule func() *Error

// Validate runs rules and returns all failures as one error.
func Validate(rules ...Rule) error {
	return Collect(rules...).Err()
}

// Collect runs rules and returns all failures.
func Collect(rules ...Rule) Errors {
	errs := make(Errors, 0)
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if err := rule(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// RequiredString checks that value is not empty after trimming spaces.
func RequiredString(field, value string) Rule {
	return func() *Error {
		if strings.TrimSpace(value) == "" {
			return fail(field, "is required")
		}
		return nil
	}
}

// Required checks that value is not the zero value of T.
func Required[T comparable](field string, value T) Rule {
	return func() *Error {
		var zero T
		if value == zero {
			return fail(field, "is required")
		}
		return nil
	}
}

// MinLen checks that value has at least minLen runes.
func MinLen(field, value string, minLen int) Rule {
	return func() *Error {
		if utf8.RuneCountInString(value) < minLen {
			return fail(field, "must be at least %d characters", minLen)
		}
		return nil
	}
}

// MaxLen checks that value has at most maxLen runes.
func MaxLen(field, value string, maxLen int) Rule {
	return func() *Error {
		if utf8.RuneCountInString(value) > maxLen {
			return fail(field, "must be at most %d characters", maxLen)
		}
		return nil
	}
}

// Min checks that value is greater than or equal to min.
func Min[T cmp.Ordered](field string, value, min T) Rule {
	return func() *Error {
		if value < min {
			return fail(field, "must be at least %v", min)
		}
		return nil
	}
}

// Max checks that value is less than or equal to max.
func Max[T cmp.Ordered](field string, value, max T) Rule {
	return func() *Error {
		if value > max {
			return fail(field, "must be at most %v", max)
		}
		return nil
	}
}

// Range checks that value is within [min, max].
func Range[T cmp.Ordered](field string, value, min, max T) Rule {
	return func() *Error {
		if value < min || value > max {
			return fail(field, "must be between %v and %v", min, max)
		}
		return nil
	}
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Email checks that value is a valid email address when it is not empty.
func Email(field, value string) Rule {
	return func() *Error {
		if value != "" && !emailRegex.MatchString(value) {
			return fail(field, "must be a valid email address")
		}
		return nil
	}
}

// Match checks that value matches expr.
func Match(field, value string, expr *regexp.Regexp, message string) Rule {
	return func() *Error {
		if expr == nil {
			return fail(field, "invalid regex")
		}
		if !expr.MatchString(value) {
			return failMessage(field, message)
		}
		return nil
	}
}

// MatchString checks that value matches pattern.
func MatchString(field, value, pattern, message string) Rule {
	return func() *Error {
		matched, err := regexp.MatchString(pattern, value)
		if err != nil {
			return fail(field, "invalid regex pattern %q: %v", pattern, err)
		}
		if !matched {
			return failMessage(field, message)
		}
		return nil
	}
}

// In checks that value equals one of allowed.
func In[T comparable](field string, value T, allowed ...T) Rule {
	return func() *Error {
		for _, item := range allowed {
			if value == item {
				return nil
			}
		}
		return fail(field, "must be one of: %s", joinValues(allowed))
	}
}

// Custom adds a caller-defined validation failure.
func Custom(field string, valid bool, message string) Rule {
	return func() *Error {
		if !valid {
			return failMessage(field, message)
		}
		return nil
	}
}

func fail(field, format string, args ...any) *Error {
	return &Error{Field: field, Message: fmt.Sprintf(format, args...)}
}

func failMessage(field, message string) *Error {
	return &Error{Field: field, Message: message}
}

func joinValues[T any](values []T) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprint(value)
	}
	return strings.Join(parts, ", ")
}
