package httpsp

import (
	"fmt"
	"reflect"
	"strings"
)

// extractJSONBody extracts fields with `json` struct tags from v and returns
// them as a map suitable for JSON marshaling. Only exported fields explicitly
// tagged with `json:"name"` are included; fields tagged with `json:"-"` are
// skipped.
func extractJSONBody(v any) (map[string]any, error) {
	val := reflect.ValueOf(v)

	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("json body: expected struct, got %s", val.Kind())
	}

	body := make(map[string]any)
	typ := val.Type()
	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}

		name, omitEmpty := parseJSONTag(tag)
		fieldVal := val.Field(i)

		if omitEmpty && fieldVal.IsZero() {
			continue
		}

		body[name] = fieldVal.Interface()
	}

	if len(body) == 0 {
		return nil, nil
	}
	return body, nil
}

func parseJSONTag(tag string) (name string, omitEmpty bool) {
	parts := strings.SplitN(tag, ",", 2)
	name = parts[0]
	if len(parts) > 1 && strings.Contains(parts[1], "omitempty") {
		omitEmpty = true
	}
	return
}
