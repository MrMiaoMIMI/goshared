// Package convutil provides small, explicit conversions for scalar values.
package convutil

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Signed is any signed integer type.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned is any unsigned integer type.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Float is any floating point type.
type Float interface {
	~float32 | ~float64
}

// Number is any integer or floating point type.
type Number interface {
	Signed | Unsigned | Float
}

// Scalar is a supported target type for To.
type Scalar interface {
	Number | ~bool | ~string
}

// To converts v into T.
//
// Supported target types are strings, bools, integers, unsigned integers, and
// floats, including user-defined aliases of those types.
func To[T Scalar](v any) (T, error) {
	var zero T
	target := reflect.TypeOf(zero)

	switch target.Kind() {
	case reflect.String:
		return cast[T](String(v)), nil
	case reflect.Bool:
		b, err := parseBool(v)
		if err != nil {
			return zero, err
		}
		return cast[T](b), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := parseInt64(v, target.Bits())
		if err != nil {
			return zero, err
		}
		return cast[T](n), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, err := parseUint64(v, target.Bits())
		if err != nil {
			return zero, err
		}
		return cast[T](n), nil
	case reflect.Float32, reflect.Float64:
		n, err := parseFloat64(v, target.Bits())
		if err != nil {
			return zero, err
		}
		return cast[T](n), nil
	default:
		return zero, fmt.Errorf("convutil: unsupported target type %s", target)
	}
}

// ToOr converts v into T, returning fallback if conversion fails.
func ToOr[T Scalar](v any, fallback T) T {
	value, err := To[T](v)
	if err != nil {
		return fallback
	}
	return value
}

// String returns a readable string representation of v.
func String(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case []byte:
		return string(val)
	case fmt.Stringer:
		return val.String()
	case error:
		return val.Error()
	case bool:
		return strconv.FormatBool(val)
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func cast[T Scalar](v any) T {
	var zero T
	target := reflect.TypeOf(zero)
	return reflect.ValueOf(v).Convert(target).Interface().(T)
}

func parseInt64(v any, bits int) (int64, error) {
	if v == nil {
		return 0, fmt.Errorf("convutil: cannot convert <nil> to int")
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := rv.Int()
		if n < minSigned(bits) || n > maxSigned(bits) {
			return 0, fmt.Errorf("convutil: %d overflows int%d", n, bits)
		}
		return n, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n := rv.Uint()
		if n > uint64(maxSigned(bits)) {
			return 0, fmt.Errorf("convutil: %d overflows int%d", n, bits)
		}
		return int64(n), nil
	case reflect.Float32, reflect.Float64:
		n := rv.Float()
		if n < float64(minSigned(bits)) || n > float64(maxSigned(bits)) {
			return 0, fmt.Errorf("convutil: %v overflows int%d", n, bits)
		}
		return int64(n), nil
	case reflect.Bool:
		if rv.Bool() {
			return 1, nil
		}
		return 0, nil
	case reflect.String:
		return strconv.ParseInt(strings.TrimSpace(rv.String()), 10, bits)
	default:
		return 0, fmt.Errorf("convutil: cannot convert %T to int", v)
	}
}

func parseUint64(v any, bits int) (uint64, error) {
	if v == nil {
		return 0, fmt.Errorf("convutil: cannot convert <nil> to uint")
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := rv.Int()
		if n < 0 {
			return 0, fmt.Errorf("convutil: cannot convert negative value %d to uint", n)
		}
		if uint64(n) > maxUnsigned(bits) {
			return 0, fmt.Errorf("convutil: %d overflows uint%d", n, bits)
		}
		return uint64(n), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n := rv.Uint()
		if n > maxUnsigned(bits) {
			return 0, fmt.Errorf("convutil: %d overflows uint%d", n, bits)
		}
		return n, nil
	case reflect.Float32, reflect.Float64:
		n := rv.Float()
		if n < 0 || n > float64(maxUnsigned(bits)) {
			return 0, fmt.Errorf("convutil: %v overflows uint%d", n, bits)
		}
		return uint64(n), nil
	case reflect.Bool:
		if rv.Bool() {
			return 1, nil
		}
		return 0, nil
	case reflect.String:
		return strconv.ParseUint(strings.TrimSpace(rv.String()), 10, bits)
	default:
		return 0, fmt.Errorf("convutil: cannot convert %T to uint", v)
	}
}

func parseFloat64(v any, bits int) (float64, error) {
	if v == nil {
		return 0, fmt.Errorf("convutil: cannot convert <nil> to float")
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(rv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return rv.Float(), nil
	case reflect.Bool:
		if rv.Bool() {
			return 1, nil
		}
		return 0, nil
	case reflect.String:
		return strconv.ParseFloat(strings.TrimSpace(rv.String()), bits)
	default:
		return 0, fmt.Errorf("convutil: cannot convert %T to float", v)
	}
}

func parseBool(v any) (bool, error) {
	if v == nil {
		return false, fmt.Errorf("convutil: cannot convert <nil> to bool")
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() != 0, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() != 0, nil
	case reflect.Float32, reflect.Float64:
		return rv.Float() != 0, nil
	case reflect.String:
		switch strings.ToLower(strings.TrimSpace(rv.String())) {
		case "true", "t", "1", "yes", "y", "on":
			return true, nil
		case "false", "f", "0", "no", "n", "off", "":
			return false, nil
		default:
			return false, fmt.Errorf("convutil: cannot convert string %q to bool", rv.String())
		}
	default:
		return false, fmt.Errorf("convutil: cannot convert %T to bool", v)
	}
}

func maxSigned(bits int) int64 {
	if bits >= 64 {
		return math.MaxInt64
	}
	return int64(1)<<(bits-1) - 1
}

func minSigned(bits int) int64 {
	if bits >= 64 {
		return math.MinInt64
	}
	return -int64(1) << (bits - 1)
}

func maxUnsigned(bits int) uint64 {
	if bits >= 64 {
		return math.MaxUint64
	}
	return uint64(1)<<bits - 1
}
