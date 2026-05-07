// Package serializer provides JSON serialization helpers.
package serializer

import (
	"bytes"
	"encoding/json"
)

// Marshal marshals v to JSON bytes.
func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal unmarshals JSON bytes into v.
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// String marshals v to a JSON string.
func String(v any) (string, error) {
	data, err := Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Pretty marshals v to indented JSON bytes.
func Pretty(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// PrettyString marshals v to an indented JSON string.
func PrettyString(v any) (string, error) {
	data, err := Pretty(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalUseNumber unmarshals JSON using json.Number for numeric values.
func UnmarshalUseNumber(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return dec.Decode(v)
}

// Copy deep-copies src into dst by marshaling and unmarshaling JSON.
func Copy(src, dst any) error {
	data, err := Marshal(src)
	if err != nil {
		return err
	}
	return Unmarshal(data, dst)
}

// Clone deep-copies src through JSON and returns a typed copy.
func Clone[T any](src T) (T, error) {
	var dst T
	err := Copy(src, &dst)
	return dst, err
}

// Valid reports whether data is valid JSON.
func Valid(data []byte) bool {
	return json.Valid(data)
}
