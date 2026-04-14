// Package validator provides lightweight struct and value validation utilities.
package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a single validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// ErrorOrNil returns nil if no errors, otherwise returns self.
func (e ValidationErrors) ErrorOrNil() error {
	if !e.HasErrors() {
		return nil
	}
	return e
}

// Validator builds validation rules fluently.
type Validator struct {
	errors ValidationErrors
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{}
}

// Required checks that the string value is not empty.
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, &ValidationError{Field: field, Message: "is required"})
	}
	return v
}

// RequiredAny checks that the value is not nil/zero.
func (v *Validator) RequiredAny(field string, value any) *Validator {
	if value == nil || reflect.ValueOf(value).IsZero() {
		v.errors = append(v.errors, &ValidationError{Field: field, Message: "is required"})
	}
	return v
}

// MinLen checks that the string has at least minLen characters.
func (v *Validator) MinLen(field, value string, minLen int) *Validator {
	if utf8.RuneCountInString(value) < minLen {
		v.errors = append(v.errors, &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d characters", minLen),
		})
	}
	return v
}

// MaxLen checks that the string has at most maxLen characters.
func (v *Validator) MaxLen(field, value string, maxLen int) *Validator {
	if utf8.RuneCountInString(value) > maxLen {
		v.errors = append(v.errors, &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at most %d characters", maxLen),
		})
	}
	return v
}

// Min checks that the numeric value is at least min.
func (v *Validator) Min(field string, value, min int64) *Validator {
	if value < min {
		v.errors = append(v.errors, &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d", min),
		})
	}
	return v
}

// Max checks that the numeric value is at most max.
func (v *Validator) Max(field string, value, max int64) *Validator {
	if value > max {
		v.errors = append(v.errors, &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at most %d", max),
		})
	}
	return v
}

// Range checks that the numeric value is within [min, max].
func (v *Validator) Range(field string, value, min, max int64) *Validator {
	if value < min || value > max {
		v.errors = append(v.errors, &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be between %d and %d", min, max),
		})
	}
	return v
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Email checks that the value is a valid email address.
func (v *Validator) Email(field, value string) *Validator {
	if value != "" && !emailRegex.MatchString(value) {
		v.errors = append(v.errors, &ValidationError{Field: field, Message: "must be a valid email address"})
	}
	return v
}

// MatchRegex checks that the value matches the given regex pattern.
// If the pattern is invalid, a validation error is added.
func (v *Validator) MatchRegex(field, value, pattern, message string) *Validator {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		v.errors = append(v.errors, &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("invalid regex pattern %q: %v", pattern, err),
		})
		return v
	}
	if !matched {
		v.errors = append(v.errors, &ValidationError{Field: field, Message: message})
	}
	return v
}

// In checks that the value is one of the allowed values.
func (v *Validator) In(field string, value string, allowed ...string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.errors = append(v.errors, &ValidationError{
		Field:   field,
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")),
	})
	return v
}

// Custom adds a custom validation rule.
func (v *Validator) Custom(field string, valid bool, message string) *Validator {
	if !valid {
		v.errors = append(v.errors, &ValidationError{Field: field, Message: message})
	}
	return v
}

// Validate returns all collected validation errors.
func (v *Validator) Validate() ValidationErrors {
	return v.errors
}

// ValidateOrNil returns nil if no errors, otherwise returns ValidationErrors.
func (v *Validator) ValidateOrNil() error {
	return v.errors.ErrorOrNil()
}
