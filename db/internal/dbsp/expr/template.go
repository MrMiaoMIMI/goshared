package expr

import (
	"fmt"
	"strings"
)

// Template represents a parsed name_expr template.
// Only ${var} interpolation is supported in templates.
// All computation (column access, function calls, arithmetic) belongs in expand_exprs.
type Template struct {
	parts []templatePart
}

type templatePart interface {
	templatePart()
}

type literalPart struct {
	text string
}

type varPart struct {
	name string
}

func (*literalPart) templatePart() {}
func (*varPart) templatePart()     {}

// ParseTemplate parses a name_expr template string.
// Only ${var} references are allowed. All other text is literal.
func ParseTemplate(input string) (*Template, error) {
	runes := []rune(input)
	var parts []templatePart
	var literal []rune

	i := 0
	for i < len(runes) {
		ch := runes[i]

		if ch == '$' && i+1 < len(runes) && runes[i+1] == '{' {
			if len(literal) > 0 {
				parts = append(parts, &literalPart{text: string(literal)})
				literal = nil
			}

			i += 2 // skip '${'
			braceContent, end, err := extractBraced(runes, i)
			if err != nil {
				return nil, fmt.Errorf("in template %q: %w", input, err)
			}
			i = end

			varName := strings.TrimSpace(braceContent)
			if varName == "" {
				return nil, fmt.Errorf("empty variable name in template %q", input)
			}
			if !isSimpleIdentifier(varName) {
				return nil, fmt.Errorf("template only supports simple ${var} references, got ${%s} in %q", braceContent, input)
			}
			parts = append(parts, &varPart{name: varName})
			continue
		}

		literal = append(literal, ch)
		i++
	}

	if len(literal) > 0 {
		parts = append(parts, &literalPart{text: string(literal)})
	}

	return &Template{parts: parts}, nil
}

// isSimpleIdentifier checks if a string is a valid Go-like identifier (letters, digits, underscores).
func isSimpleIdentifier(s string) bool {
	if s == "" {
		return false
	}
	runes := []rune(s)
	if !isIdentStartChar(runes[0]) {
		return false
	}
	for _, r := range runes[1:] {
		if !isIdentChar(r) {
			return false
		}
	}
	return true
}

// extractBraced reads content between { and }, handling nesting.
// pos should point right after the opening '{'.
// Returns content string, position after closing '}', and any error.
func extractBraced(runes []rune, pos int) (string, int, error) {
	depth := 1
	start := pos
	for pos < len(runes) {
		switch runes[pos] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return string(runes[start:pos]), pos + 1, nil
			}
		}
		pos++
	}
	return "", pos, fmt.Errorf("unclosed '{' starting at position %d", start-1)
}

// Eval evaluates the template with the given context, producing a string.
func (t *Template) Eval(ctx *EvalContext) (string, error) {
	var sb strings.Builder
	for _, part := range t.parts {
		switch p := part.(type) {
		case *literalPart:
			sb.WriteString(p.text)
		case *varPart:
			v, ok := ctx.GetVar(p.name)
			if !ok {
				return "", fmt.Errorf("variable ${%s} not defined", p.name)
			}
			sb.WriteString(v.String())
		}
	}
	return sb.String(), nil
}

// CollectVarRefs returns all ${var} names referenced in the template.
func (t *Template) CollectVarRefs() []string {
	var refs []string
	for _, part := range t.parts {
		if vp, ok := part.(*varPart); ok {
			refs = append(refs, vp.name)
		}
	}
	return refs
}
