package expr

import (
	"fmt"
	"strconv"
)

type valueKind int

const (
	KindInt64 valueKind = iota
	KindString
)

type Value struct {
	kind   valueKind
	intVal int64
	strVal string
}

func IntValue(v int64) Value   { return Value{kind: KindInt64, intVal: v} }
func StrValue(v string) Value  { return Value{kind: KindString, strVal: v} }

func (v Value) Kind() valueKind { return v.kind }
func (v Value) IsInt() bool     { return v.kind == KindInt64 }
func (v Value) IsString() bool  { return v.kind == KindString }

func (v Value) Int64() (int64, error) {
	switch v.kind {
	case KindInt64:
		return v.intVal, nil
	case KindString:
		n, err := strconv.ParseInt(v.strVal, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert string %q to int64", v.strVal)
		}
		return n, nil
	}
	return 0, fmt.Errorf("unknown value kind")
}

func (v Value) MustInt64() int64 {
	n, err := v.Int64()
	if err != nil {
		panic(err)
	}
	return n
}

func (v Value) String() string {
	switch v.kind {
	case KindInt64:
		return strconv.FormatInt(v.intVal, 10)
	case KindString:
		return v.strVal
	}
	return ""
}
