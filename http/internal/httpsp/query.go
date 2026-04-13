package httpsp

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// encodeQueryStruct converts a struct with `url` tags into url.Values.
//
// Supported field types: string, int*, uint*, float*, bool.
// Tag format: `url:"param_name"` or `url:"param_name,omitempty"`.
// A tag of "-" causes the field to be skipped.
func encodeQueryStruct(v any) (url.Values, error) {
	values := make(url.Values)
	val := reflect.ValueOf(v)

	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return values, nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("query: expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	for i := range typ.NumField() {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		tag := field.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}

		name, omitEmpty := parseURLTag(tag)

		if omitEmpty && fieldVal.IsZero() {
			continue
		}

		strVal, err := formatFieldValue(fieldVal)
		if err != nil {
			continue
		}

		values.Set(name, strVal)
	}

	return values, nil
}

func parseURLTag(tag string) (name string, omitEmpty bool) {
	parts := strings.SplitN(tag, ",", 2)
	name = parts[0]
	if len(parts) > 1 && parts[1] == "omitempty" {
		omitEmpty = true
	}
	return
}

func formatFieldValue(v reflect.Value) (string, error) {
	switch v.Kind() {
	case reflect.String:
		return v.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	default:
		return "", fmt.Errorf("query: unsupported type %s", v.Kind())
	}
}
