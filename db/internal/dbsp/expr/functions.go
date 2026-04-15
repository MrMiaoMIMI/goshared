package expr

import (
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
)

// Func is a built-in function implementation.
type Func func(args []Value) (Value, error)

var (
	funcsMu   sync.RWMutex
	funcRegistry = map[string]Func{
		"fill":   builtinFill,
		"str":    builtinStr,
		"hash":   builtinHash,
		"mod":    builtinMod,
		"div":    builtinDiv,
		"lower":  builtinLower,
		"upper":  builtinUpper,
		"concat": builtinConcat,
	}
)

// RegisterFunc registers a custom built-in function.
func RegisterFunc(name string, fn Func) {
	funcsMu.Lock()
	defer funcsMu.Unlock()
	funcRegistry[name] = fn
}

// LookupFunc returns the function registered under the given name.
func LookupFunc(name string) (Func, bool) {
	funcsMu.RLock()
	defer funcsMu.RUnlock()
	fn, ok := funcRegistry[name]
	return fn, ok
}

func checkArity(name string, args []Value, expected int) error {
	if len(args) != expected {
		return fmt.Errorf("%s() expects %d arguments, got %d", name, expected, len(args))
	}
	return nil
}

// fill(value, width) -- zero-pad integer to width
func builtinFill(args []Value) (Value, error) {
	if err := checkArity("fill", args, 2); err != nil {
		return Value{}, err
	}
	val, err := args[0].Int64()
	if err != nil {
		return Value{}, fmt.Errorf("fill(): first argument: %w", err)
	}
	width, err := args[1].Int64()
	if err != nil {
		return Value{}, fmt.Errorf("fill(): second argument: %w", err)
	}
	s := fmt.Sprintf("%0*d", width, val)
	return StrValue(s), nil
}

// str(value) -- convert to string
func builtinStr(args []Value) (Value, error) {
	if err := checkArity("str", args, 1); err != nil {
		return Value{}, err
	}
	return StrValue(args[0].String()), nil
}

// hash(value) -- FNV-1a hash to non-negative int64
func builtinHash(args []Value) (Value, error) {
	if err := checkArity("hash", args, 1); err != nil {
		return Value{}, err
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(args[0].String()))
	return IntValue(int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)), nil
}

// mod(a, b) -- non-negative modulo (always returns 0..b-1 for b > 0)
func builtinMod(args []Value) (Value, error) {
	if err := checkArity("mod", args, 2); err != nil {
		return Value{}, err
	}
	a, err := args[0].Int64()
	if err != nil {
		return Value{}, fmt.Errorf("mod(): first argument: %w", err)
	}
	b, err := args[1].Int64()
	if err != nil {
		return Value{}, fmt.Errorf("mod(): second argument: %w", err)
	}
	if b == 0 {
		return Value{}, fmt.Errorf("mod(): division by zero")
	}
	r := a % b
	if r < 0 {
		r += b
	}
	return IntValue(r), nil
}

// div(a, b) -- alias for a / b
func builtinDiv(args []Value) (Value, error) {
	if err := checkArity("div", args, 2); err != nil {
		return Value{}, err
	}
	a, err := args[0].Int64()
	if err != nil {
		return Value{}, fmt.Errorf("div(): first argument: %w", err)
	}
	b, err := args[1].Int64()
	if err != nil {
		return Value{}, fmt.Errorf("div(): second argument: %w", err)
	}
	if b == 0 {
		return Value{}, fmt.Errorf("div(): division by zero")
	}
	return IntValue(a / b), nil
}

// lower(value) -- lowercase string
func builtinLower(args []Value) (Value, error) {
	if err := checkArity("lower", args, 1); err != nil {
		return Value{}, err
	}
	return StrValue(strings.ToLower(args[0].String())), nil
}

// upper(value) -- uppercase string
func builtinUpper(args []Value) (Value, error) {
	if err := checkArity("upper", args, 1); err != nil {
		return Value{}, err
	}
	return StrValue(strings.ToUpper(args[0].String())), nil
}

// concat(a, b, ...) -- concatenate strings
func builtinConcat(args []Value) (Value, error) {
	if len(args) < 2 {
		return Value{}, fmt.Errorf("concat() expects at least 2 arguments, got %d", len(args))
	}
	var sb strings.Builder
	for _, a := range args {
		sb.WriteString(a.String())
	}
	return StrValue(sb.String()), nil
}
