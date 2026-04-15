package expr

import (
	"fmt"
)

// EvalContext holds variables and column values for expression evaluation.
type EvalContext struct {
	vars map[string]Value
	cols map[string]Value
}

func NewContext() *EvalContext {
	return &EvalContext{
		vars: make(map[string]Value),
		cols: make(map[string]Value),
	}
}

func (c *EvalContext) SetVar(name string, val Value) {
	c.vars[name] = val
}

func (c *EvalContext) GetVar(name string) (Value, bool) {
	v, ok := c.vars[name]
	return v, ok
}

func (c *EvalContext) SetCol(name string, val Value) {
	c.cols[name] = val
}

func (c *EvalContext) GetCol(name string) (Value, bool) {
	v, ok := c.cols[name]
	return v, ok
}

// CopyTo copies all vars and cols from this context to the target.
// Existing entries in target are NOT overwritten.
func (c *EvalContext) CopyTo(target *EvalContext) {
	for k, v := range c.vars {
		if _, exists := target.vars[k]; !exists {
			target.vars[k] = v
		}
	}
	for k, v := range c.cols {
		if _, exists := target.cols[k]; !exists {
			target.cols[k] = v
		}
	}
}

// LoadColumnsFromMap loads column values from a map[string]any.
// Supports int64, int, uint64, string as value types.
func (c *EvalContext) LoadColumnsFromMap(m map[string]any) error {
	for k, v := range m {
		val, err := anyToValue(v)
		if err != nil {
			return fmt.Errorf("column %q: %w", k, err)
		}
		c.SetCol(k, val)
	}
	return nil
}

func anyToValue(v any) (Value, error) {
	switch val := v.(type) {
	case int64:
		return IntValue(val), nil
	case int:
		return IntValue(int64(val)), nil
	case uint64:
		return IntValue(int64(val)), nil
	case int32:
		return IntValue(int64(val)), nil
	case uint32:
		return IntValue(int64(val)), nil
	case string:
		return StrValue(val), nil
	default:
		return Value{}, fmt.Errorf("unsupported type %T", v)
	}
}

// Eval evaluates an AST expression within the given context.
func Eval(node Expr, ctx *EvalContext) (Value, error) {
	switch n := node.(type) {
	case *IntLit:
		return IntValue(n.Value), nil

	case *StrLit:
		return StrValue(n.Value), nil

	case *ColRef:
		v, ok := ctx.GetCol(n.Name)
		if !ok {
			return Value{}, fmt.Errorf("column @{%s} not found in ShardingKey", n.Name)
		}
		return v, nil

	case *VarRef:
		v, ok := ctx.GetVar(n.Name)
		if !ok {
			return Value{}, fmt.Errorf("variable ${%s} not defined", n.Name)
		}
		return v, nil

	case *BinaryOp:
		left, err := Eval(n.Left, ctx)
		if err != nil {
			return Value{}, err
		}
		right, err := Eval(n.Right, ctx)
		if err != nil {
			return Value{}, err
		}
		return evalBinaryOp(n.Op, left, right)

	case *FuncCall:
		args := make([]Value, len(n.Args))
		for i, arg := range n.Args {
			val, err := Eval(arg, ctx)
			if err != nil {
				return Value{}, err
			}
			args[i] = val
		}
		fn, ok := LookupFunc(n.Name)
		if !ok {
			return Value{}, fmt.Errorf("unknown function %s()", n.Name)
		}
		return fn(args)

	default:
		return Value{}, fmt.Errorf("unknown AST node type %T", node)
	}
}

func evalBinaryOp(op TokenKind, left, right Value) (Value, error) {
	lv, err := left.Int64()
	if err != nil {
		return Value{}, fmt.Errorf("left operand: %w", err)
	}
	rv, err := right.Int64()
	if err != nil {
		return Value{}, fmt.Errorf("right operand: %w", err)
	}
	switch op {
	case TokenPlus:
		return IntValue(lv + rv), nil
	case TokenMinus:
		return IntValue(lv - rv), nil
	case TokenStar:
		return IntValue(lv * rv), nil
	case TokenSlash:
		if rv == 0 {
			return Value{}, fmt.Errorf("division by zero")
		}
		return IntValue(lv / rv), nil
	case TokenPercent:
		if rv == 0 {
			return Value{}, fmt.Errorf("modulo by zero")
		}
		return IntValue(lv % rv), nil
	default:
		return Value{}, fmt.Errorf("unknown operator %d", op)
	}
}
