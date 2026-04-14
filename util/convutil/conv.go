// Package convutil provides type conversion utility functions.
package convutil

import (
	"fmt"
	"strconv"
)

// ToInt64 converts any numeric or string value to int64.
func ToInt64(v any) (int64, error) {
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint64:
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("convutil: cannot convert %T to int64", v)
	}
}

// ToInt64Or converts v to int64, returning fallback on error.
func ToInt64Or(v any, fallback int64) int64 {
	n, err := ToInt64(v)
	if err != nil {
		return fallback
	}
	return n
}

// ToFloat64 converts any numeric or string value to float64.
func ToFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("convutil: cannot convert %T to float64", v)
	}
}

// ToFloat64Or converts v to float64, returning fallback on error.
func ToFloat64Or(v any, fallback float64) float64 {
	n, err := ToFloat64(v)
	if err != nil {
		return fallback
	}
	return n
}

// ToString converts any value to its string representation.
func ToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.FormatInt(int64(val), 10)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return ""
	case fmt.Stringer:
		return val.String()
	case error:
		return val.Error()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToBool converts a value to bool.
// "true", "1", "yes", "on" → true; "false", "0", "no", "off", "" → false.
func ToBool(v any) (bool, error) {
	switch val := v.(type) {
	case bool:
		return val, nil
	case int, int8, int16, int32, int64:
		n, _ := ToInt64(val)
		return n != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		n, _ := ToInt64(val)
		return n != 0, nil
	case string:
		switch val {
		case "true", "1", "yes", "on", "TRUE", "True", "YES", "Yes", "ON", "On":
			return true, nil
		case "false", "0", "no", "off", "", "FALSE", "False", "NO", "No", "OFF", "Off":
			return false, nil
		default:
			return false, fmt.Errorf("convutil: cannot convert string %q to bool", val)
		}
	default:
		return false, fmt.Errorf("convutil: cannot convert %T to bool", v)
	}
}

// ToBoolOr converts v to bool, returning fallback on error.
func ToBoolOr(v any, fallback bool) bool {
	b, err := ToBool(v)
	if err != nil {
		return fallback
	}
	return b
}
