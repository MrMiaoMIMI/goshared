// Package strutil provides string utility functions.
package strutil

import (
	"strings"
	"unicode"
)

// IsEmpty returns true if the string is empty.
func IsEmpty(s string) bool {
	return len(s) == 0
}

// IsBlank returns true if the string is empty or contains only whitespace.
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// DefaultIfEmpty returns defaultVal if s is empty, otherwise s.
func DefaultIfEmpty(s, defaultVal string) string {
	if s == "" {
		return defaultVal
	}
	return s
}

// DefaultIfBlank returns defaultVal if s is blank, otherwise s.
func DefaultIfBlank(s, defaultVal string) string {
	if IsBlank(s) {
		return defaultVal
	}
	return s
}

// Truncate truncates s to at most maxLen characters.
// Returns empty string if maxLen <= 0.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// TruncateWithSuffix truncates s and appends suffix (e.g. "...") if truncated.
func TruncateWithSuffix(s string, maxLen int, suffix string) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	suffixRunes := []rune(suffix)
	if maxLen <= len(suffixRunes) {
		return string(suffixRunes[:maxLen])
	}
	return string(runes[:maxLen-len(suffixRunes)]) + suffix
}

// CamelToSnake converts CamelCase to snake_case.
func CamelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// SnakeToCamel converts snake_case to CamelCase (PascalCase).
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		result.WriteString(string(runes))
	}
	return result.String()
}

// SnakeToLowerCamel converts snake_case to lowerCamelCase.
func SnakeToLowerCamel(s string) string {
	camel := SnakeToCamel(s)
	if camel == "" {
		return ""
	}
	runes := []rune(camel)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// Reverse returns s with its characters reversed.
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ContainsAny returns true if s contains any of the substrings.
func ContainsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ContainsAll returns true if s contains all of the substrings.
func ContainsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// RemoveWhitespace removes all whitespace characters from s.
func RemoveWhitespace(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// PadLeft pads s on the left with padChar to reach totalLen.
// Returns s unchanged if totalLen <= len(s) or totalLen <= 0.
func PadLeft(s string, totalLen int, padChar rune) string {
	if totalLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) >= totalLen {
		return s
	}
	return strings.Repeat(string(padChar), totalLen-len(runes)) + s
}

// PadRight pads s on the right with padChar to reach totalLen.
// Returns s unchanged if totalLen <= len(s) or totalLen <= 0.
func PadRight(s string, totalLen int, padChar rune) string {
	if totalLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) >= totalLen {
		return s
	}
	return s + strings.Repeat(string(padChar), totalLen-len(runes))
}

// MaskMiddle masks the middle portion of a string, useful for hiding sensitive data.
// Keeps the first `keep` and last `keep` characters, replaces the rest with maskChar.
func MaskMiddle(s string, keep int, maskChar rune) string {
	runes := []rune(s)
	if len(runes) <= keep*2 {
		return s
	}
	for i := keep; i < len(runes)-keep; i++ {
		runes[i] = maskChar
	}
	return string(runes)
}

// SplitAndTrim splits s by sep and trims whitespace from each element.
// Empty elements after trimming are removed.
func SplitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// SubstringBefore returns the substring before the first occurrence of sep.
// Returns the original string if sep is not found.
func SubstringBefore(s, sep string) string {
	if idx := strings.Index(s, sep); idx >= 0 {
		return s[:idx]
	}
	return s
}

// SubstringAfter returns the substring after the first occurrence of sep.
// Returns empty string if sep is not found.
func SubstringAfter(s, sep string) string {
	if idx := strings.Index(s, sep); idx >= 0 {
		return s[idx+len(sep):]
	}
	return ""
}

// SubstringBeforeLast returns the substring before the last occurrence of sep.
func SubstringBeforeLast(s, sep string) string {
	if idx := strings.LastIndex(s, sep); idx >= 0 {
		return s[:idx]
	}
	return s
}

// SubstringAfterLast returns the substring after the last occurrence of sep.
func SubstringAfterLast(s, sep string) string {
	if idx := strings.LastIndex(s, sep); idx >= 0 {
		return s[idx+len(sep):]
	}
	return ""
}

// HasPrefixAny returns true if s has any of the given prefixes.
func HasPrefixAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// HasSuffixAny returns true if s has any of the given suffixes.
func HasSuffixAny(s string, suffixes ...string) bool {
	for _, sf := range suffixes {
		if strings.HasSuffix(s, sf) {
			return true
		}
	}
	return false
}

// CountWords returns the number of words in a string (split by whitespace).
func CountWords(s string) int {
	return len(strings.Fields(s))
}

// Capitalize uppercases the first character of s.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// Uncapitalize lowercases the first character of s.
func Uncapitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// RepeatJoin repeats the string n times with the given separator.
func RepeatJoin(s string, n int, sep string) string {
	if n <= 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = s
	}
	return strings.Join(parts, sep)
}

// ToKebabCase converts CamelCase or snake_case to kebab-case.
func ToKebabCase(s string) string {
	snake := CamelToSnake(s)
	return strings.ReplaceAll(snake, "_", "-")
}

// IsNumeric returns true if the string contains only digits.
func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsAlpha returns true if the string contains only letters.
func IsAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsAlphaNumeric returns true if the string contains only letters and digits.
func IsAlphaNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// WrapWith wraps s with the given prefix and suffix.
func WrapWith(s, prefix, suffix string) string {
	return prefix + s + suffix
}

// Quote wraps s in double quotes.
func Quote(s string) string {
	return "\"" + s + "\""
}

// Ellipsis truncates s to maxLen and appends "..." if truncated.
func Ellipsis(s string, maxLen int) string {
	return TruncateWithSuffix(s, maxLen, "...")
}
