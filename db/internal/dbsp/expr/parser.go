package expr

import (
	"fmt"
)

// Parser is a recursive-descent parser for expressions.
type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Kind: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *Parser) expect(kind TokenKind) (Token, error) {
	t := p.peek()
	if t.Kind != kind {
		return t, fmt.Errorf("expected token %d, got %s at position %d", kind, t, t.Pos)
	}
	return p.advance(), nil
}

// ParseExpression parses a full expression.
func (p *Parser) ParseExpression() (Expr, error) {
	return p.parseAddSub()
}

// parseAddSub handles + and - (lowest precedence).
func (p *Parser) parseAddSub() (Expr, error) {
	left, err := p.parseMulDivMod()
	if err != nil {
		return nil, err
	}
	for p.peek().Kind == TokenPlus || p.peek().Kind == TokenMinus {
		op := p.advance()
		right, err := p.parseMulDivMod()
		if err != nil {
			return nil, err
		}
		left = &BinaryOp{Op: op.Kind, Left: left, Right: right}
	}
	return left, nil
}

// parseMulDivMod handles *, /, % (higher precedence).
func (p *Parser) parseMulDivMod() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.peek().Kind == TokenStar || p.peek().Kind == TokenSlash || p.peek().Kind == TokenPercent {
		op := p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryOp{Op: op.Kind, Left: left, Right: right}
	}
	return left, nil
}

// parseUnary handles unary minus.
func (p *Parser) parseUnary() (Expr, error) {
	if p.peek().Kind == TokenMinus {
		p.advance()
		operand, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &BinaryOp{Op: TokenMinus, Left: &IntLit{Value: 0}, Right: operand}, nil
	}
	return p.parsePrimary()
}

// parsePrimary handles atoms: literals, refs, function calls, parens.
func (p *Parser) parsePrimary() (Expr, error) {
	t := p.peek()
	switch t.Kind {
	case TokenInt:
		p.advance()
		val, err := intFromString(t.Val)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q at position %d", t.Val, t.Pos)
		}
		return &IntLit{Value: val}, nil

	case TokenString:
		p.advance()
		return &StrLit{Value: t.Val}, nil

	case TokenColRef:
		p.advance()
		return &ColRef{Name: t.Val}, nil

	case TokenVarRef:
		p.advance()
		return &VarRef{Name: t.Val}, nil

	case TokenIdent:
		// Look ahead: if followed by '(', this is a function call
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Kind == TokenLParen {
			return p.parseFuncCall()
		}
		// Otherwise treat bare identifier as string literal
		p.advance()
		return &StrLit{Value: t.Val}, nil

	case TokenLParen:
		p.advance()
		expr, err := p.ParseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, fmt.Errorf("unclosed parenthesis at position %d", t.Pos)
		}
		return expr, nil

	default:
		return nil, fmt.Errorf("unexpected token %s at position %d", t, t.Pos)
	}
}

// parseFuncCall parses a function call: ident '(' args... ')'
func (p *Parser) parseFuncCall() (Expr, error) {
	nameToken := p.advance() // TokenIdent (function name)
	p.advance()              // skip '('

	var args []Expr
	if p.peek().Kind != TokenRParen {
		arg, err := p.ParseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		for p.peek().Kind == TokenComma {
			p.advance() // skip ','
			arg, err := p.ParseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}

	if _, err := p.expect(TokenRParen); err != nil {
		return nil, fmt.Errorf("expected ')' after function arguments for %q", nameToken.Val)
	}

	return &FuncCall{Name: nameToken.Val, Args: args}, nil
}

// ParseExpressionString is a convenience that tokenizes and parses an expression string.
func ParseExpressionString(input string) (Expr, error) {
	tokens, err := TokenizeExpression(input)
	if err != nil {
		return nil, fmt.Errorf("tokenize error: %w", err)
	}
	parser := NewParser(tokens)
	expr, err := parser.ParseExpression()
	if err != nil {
		return nil, fmt.Errorf("parse error in %q: %w", input, err)
	}
	if parser.peek().Kind != TokenEOF {
		return nil, fmt.Errorf("unexpected trailing tokens in %q at position %d", input, parser.peek().Pos)
	}
	return expr, nil
}
