package codec

import "testing"

type namedString string

func TestBase64EncodeDecode(t *testing.T) {
	encoded := Base64Encode(namedString("hello"))
	got, err := Base64DecodeString(encoded)
	if err != nil {
		t.Fatalf("Base64DecodeString: %v", err)
	}
	if got != "hello" {
		t.Fatalf("decoded = %q, want hello", got)
	}
}

func TestBase64URLEncodeDecode(t *testing.T) {
	encoded := Base64URLEncode([]byte("hello?"))
	got, err := Base64URLDecodeString(encoded)
	if err != nil {
		t.Fatalf("Base64URLDecodeString: %v", err)
	}
	if got != "hello?" {
		t.Fatalf("decoded = %q, want hello?", got)
	}
}

func TestHexEncodeDecode(t *testing.T) {
	encoded := HexEncode("hello")
	got, err := HexDecodeString(encoded)
	if err != nil {
		t.Fatalf("HexDecodeString: %v", err)
	}
	if got != "hello" {
		t.Fatalf("decoded = %q, want hello", got)
	}
}
