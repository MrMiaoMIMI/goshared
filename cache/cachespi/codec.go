package cachespi

import "encoding/json"

// Codec defines how cache values are serialized and deserialized.
//
// Redis cache always stores bytes, so Redis codecs must serialize values into
// bytes. Reference semantics are not supported by Redis. In-memory cache uses
// Codec when InMemoryConfig.Codec is set to CodecJSON; otherwise it stores Go
// values directly by reference.
type Codec interface {
	Marshal(value any) ([]byte, error)
	Unmarshal(data []byte, receiver any) error
}

// JSONCodec serializes cache values with encoding/json.
//
// Redis cache uses JSONCodec by default. In-memory cache does not use a codec by
// default; configure JSONCodec explicitly to get copy semantics similar to Redis.
type JSONCodec struct{}

func (JSONCodec) Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func (JSONCodec) Unmarshal(data []byte, receiver any) error {
	return json.Unmarshal(data, receiver)
}
