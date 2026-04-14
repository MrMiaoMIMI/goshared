package serializer

import (
	"bytes"
	"encoding/json"
)

// JsonMarshal marshals v to JSON bytes.
func JsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// JsonUnmarshal unmarshals JSON bytes into v.
func JsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// JsonBytes marshals v to JSON bytes, returning nil on error.
func JsonBytes(v any) []byte {
	b, _ := JsonMarshal(v)
	return b
}

// JsonString marshals v to a JSON string.
func JsonString(v any) string {
	return string(JsonBytes(v))
}

// JsonPretty marshals v to indented JSON bytes.
func JsonPretty(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// JsonPrettyString marshals v to an indented JSON string.
func JsonPrettyString(v any) string {
	b, err := JsonPretty(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// JsonSafeUnmarshal unmarshals JSON bytes into v, using json.Number for numeric values
// to avoid float64 precision loss.
func JsonSafeUnmarshal(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return dec.Decode(v)
}

// JsonClone deep-copies src into dst by marshaling and unmarshaling.
func JsonClone(src, dst any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// JsonValid reports whether data is valid JSON.
func JsonValid(data []byte) bool {
	return json.Valid(data)
}
