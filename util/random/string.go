package random

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/MrMiaoMIMI/goshared/util/hash"
)

// GenRandomHashKey generates a cryptographically random SHA-256 hash key.
func GenRandomHashKey() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(fmt.Sprintf("hash: failed to read random bytes: %v", err))
	}
	return hash.SHA256(b)
}

// GenRandomBytes generates n cryptographically random bytes.
// Returns an error if n < 0.
func GenRandomBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("hash: GenRandomBytes: n must be non-negative, got %d", n)
	}
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	return b, err
}

// GenRandomHex generates a random hex string of the given byte length.
func GenRandomHex(byteLen int) (string, error) {
	b, err := GenRandomBytes(byteLen)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
