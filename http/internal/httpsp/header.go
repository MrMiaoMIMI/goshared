package httpsp

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// encodeHeaderStruct converts a struct with `header` tags into http.Header.
//
// Supported field types: string, int*, uint*, float*, bool.
// Tag format: `header:"Header-Name"` or `header:"Header-Name,omitempty"`.
// A tag of "-" causes the field to be skipped.
func encodeHeaderStruct(v any) (http.Header, error) {
	headers := make(http.Header)
	val := reflect.ValueOf(v)

	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return headers, nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("header: expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	for i := range typ.NumField() {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		tag := field.Tag.Get("header")
		if tag == "" || tag == "-" {
			continue
		}

		name, omitEmpty := parseHeaderTag(tag)

		if omitEmpty && fieldVal.IsZero() {
			continue
		}

		strVal, err := formatFieldValue(fieldVal)
		if err != nil {
			continue
		}

		headers.Set(name, strVal)
	}

	return headers, nil
}

func parseHeaderTag(tag string) (name string, omitEmpty bool) {
	parts := strings.SplitN(tag, ",", 2)
	name = parts[0]
	if len(parts) > 1 && parts[1] == "omitempty" {
		omitEmpty = true
	}
	return
}
