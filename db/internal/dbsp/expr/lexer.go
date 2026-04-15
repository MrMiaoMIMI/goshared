package expr

import (
	"fmt"
	"strconv"
	"unicode"
)

type TokenKind int

const (
	TokenEOF TokenKind = iota
	TokenInt
	TokenString
	TokenIdent
	TokenColRef // @{name}
	TokenVarRef // ${name}
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenPercent
	TokenLParen
	TokenRParen
	TokenComma
	TokenAssign     // =
	TokenDeclAssign // :=
)

type Token struct {
	Kind TokenKind
	Val  string
	Pos  int
}

func (t Token) String() string {
	switch t.Kind {
	case TokenEOF:
		return "EOF"
	case TokenInt:
		return fmt.Sprintf("INT(%s)", t.Val)
	case TokenString:
		return fmt.Sprintf("STR(%q)", t.Val)
	case TokenIdent:
		return fmt.Sprintf("IDENT(%s)", t.Val)
	case TokenColRef:
		return fmt.Sprintf("COL(@{%s})", t.Val)
	case TokenVarRef:
		return fmt.Sprintf("VAR(${%s})", t.Val)
	default:
		return fmt.Sprintf("TOK(%d,%q)", t.Kind, t.Val)
	}
}

type Lexer struct {
	input []rune
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: []rune(input), pos: 0}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) advance() rune {
	ch := l.input[l.pos]
	l.pos++
	return ch
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
		l.pos++
	}
	return string(l.input[start:l.pos])
}

func (l *Lexer) readBracedName() (string, error) {
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '}' {
		l.pos++
	}
	if l.pos >= len(l.input) {
		return "", fmt.Errorf("unclosed '{' at position %d", start-1)
	}
	name := string(l.input[start:l.pos])
	l.pos++ // skip '}'
	return name, nil
}

func (l *Lexer) readString() (string, error) {
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		if l.input[l.pos] == '\\' {
			l.pos++ // skip escape
		}
		l.pos++
	}
	if l.pos >= len(l.input) {
		return "", fmt.Errorf("unclosed string at position %d", start-1)
	}
	val := string(l.input[start:l.pos])
	l.pos++ // skip closing '"'
	return val, nil
}

func (l *Lexer) readNumber() string {
	start := l.pos
	for l.pos < len(l.input) && unicode.IsDigit(l.input[l.pos]) {
		l.pos++
	}
	return string(l.input[start:l.pos])
}

// Tokenize produces all tokens from the input (expression context, not template).
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	for {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			tokens = append(tokens, Token{Kind: TokenEOF, Pos: l.pos})
			return tokens, nil
		}

		startPos := l.pos
		ch := l.peek()

		switch {
		case ch == '@':
			l.advance()
			if l.peek() != '{' {
				return nil, fmt.Errorf("expected '{' after '@' at position %d", startPos)
			}
			l.advance() // skip '{'
			name, err := l.readBracedName()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{Kind: TokenColRef, Val: name, Pos: startPos})

		case ch == '$':
			l.advance()
			if l.peek() != '{' {
				return nil, fmt.Errorf("expected '{' after '$' at position %d", startPos)
			}
			l.advance() // skip '{'
			name, err := l.readBracedName()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{Kind: TokenVarRef, Val: name, Pos: startPos})

		case ch == '"':
			l.advance()
			val, err := l.readString()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{Kind: TokenString, Val: val, Pos: startPos})

		case unicode.IsDigit(ch):
			numStr := l.readNumber()
			tokens = append(tokens, Token{Kind: TokenInt, Val: numStr, Pos: startPos})

		case isIdentStartChar(ch):
			ident := l.readIdentifier()
			tokens = append(tokens, Token{Kind: TokenIdent, Val: ident, Pos: startPos})

		case ch == '+':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenPlus, Val: "+", Pos: startPos})
		case ch == '-':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenMinus, Val: "-", Pos: startPos})
		case ch == '*':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenStar, Val: "*", Pos: startPos})
		case ch == '/':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenSlash, Val: "/", Pos: startPos})
		case ch == '%':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenPercent, Val: "%", Pos: startPos})
		case ch == '(':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenLParen, Val: "(", Pos: startPos})
		case ch == ')':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenRParen, Val: ")", Pos: startPos})
		case ch == ',':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenComma, Val: ",", Pos: startPos})
		case ch == ':':
			l.advance()
			if l.peek() == '=' {
				l.advance()
				tokens = append(tokens, Token{Kind: TokenDeclAssign, Val: ":=", Pos: startPos})
			} else {
				return nil, fmt.Errorf("unexpected ':' at position %d (expected ':=')", startPos)
			}
		case ch == '=':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenAssign, Val: "=", Pos: startPos})
		default:
			return nil, fmt.Errorf("unexpected character %q at position %d", ch, startPos)
		}
	}
}

// TokenizeExpression tokenizes a standalone expression (e.g., RHS of an expand_expr).
func TokenizeExpression(input string) ([]Token, error) {
	l := NewLexer(input)
	return l.Tokenize()
}

// intFromString parses an integer string to int64.
func intFromString(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func isIdentStartChar(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}
