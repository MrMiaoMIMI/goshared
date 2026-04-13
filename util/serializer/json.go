package serializer

import "encoding/json"

func JsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func JsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func JsonBytes(v any) []byte {
	jsonBytes, _ := JsonMarshal(v)
	return jsonBytes
}
