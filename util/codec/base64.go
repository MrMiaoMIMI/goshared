// Package codec provides small encoding helpers for byte and string payloads.
package codec

import (
	"encoding/base64"
	"encoding/hex"
)

// Bytes is any value that can be encoded as bytes.
type Bytes interface {
	~[]byte | ~string
}

// Base64Encode encodes data to standard padded base64.
func Base64Encode[T Bytes](data T) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// Base64Decode decodes standard padded base64.
func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// Base64DecodeString decodes standard padded base64 into a string.
func Base64DecodeString(data string) (string, error) {
	decoded, err := Base64Decode(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// Base64URLEncode encodes data to URL-safe unpadded base64.
func Base64URLEncode[T Bytes](data T) string {
	return base64.RawURLEncoding.EncodeToString([]byte(data))
}

// Base64URLDecode decodes URL-safe unpadded base64.
func Base64URLDecode(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}

// Base64URLDecodeString decodes URL-safe unpadded base64 into a string.
func Base64URLDecodeString(data string) (string, error) {
	decoded, err := Base64URLDecode(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// HexEncode encodes data to lowercase hexadecimal.
func HexEncode[T Bytes](data T) string {
	return hex.EncodeToString([]byte(data))
}

// HexDecode decodes a hexadecimal string.
func HexDecode(data string) ([]byte, error) {
	return hex.DecodeString(data)
}

// HexDecodeString decodes a hexadecimal string into a string.
func HexDecodeString(data string) (string, error) {
	decoded, err := HexDecode(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
