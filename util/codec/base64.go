package codec

import (
	"encoding/base64"
	"encoding/hex"
)

// Base64Encode encodes data to standard base64.
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode decodes standard base64 data.
func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// Base64DecodeWithDefault decodes base64 data, returning defaultValue on error.
func Base64DecodeWithDefault(data string, defaultValue []byte) []byte {
	decoded, err := Base64Decode(data)
	if err != nil {
		return defaultValue
	}
	return decoded
}

// Base64URLEncode encodes data to URL-safe base64 (no padding).
func Base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// Base64URLDecode decodes URL-safe base64 data (no padding).
func Base64URLDecode(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}

// Base64EncodeString encodes a string to standard base64.
func Base64EncodeString(s string) string {
	return Base64Encode([]byte(s))
}

// Base64DecodeString decodes standard base64 to a string.
func Base64DecodeString(data string) (string, error) {
	decoded, err := Base64Decode(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// HexEncode encodes data to hexadecimal string.
func HexEncode(data []byte) string {
	return hex.EncodeToString(data)
}

// HexDecode decodes a hexadecimal string to bytes.
func HexDecode(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// HexEncodeString encodes a string to hexadecimal.
func HexEncodeString(s string) string {
	return hex.EncodeToString([]byte(s))
}

// HexDecodeString decodes a hexadecimal string to a regular string.
func HexDecodeString(s string) (string, error) {
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
