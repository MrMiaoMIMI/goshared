package codec

import "encoding/base64"

func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

func Base64DecodeWithDefaultValue(data string, defaultValue []byte) []byte {
	decoded, err := Base64Decode(data)
	if err != nil {
		return defaultValue
	}
	return decoded
}
