// Package crypto provides encryption and decryption utilities using AES-GCM.
// AES-GCM is an authenticated encryption scheme that provides both confidentiality
// and integrity protection.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	ErrInvalidKeySize   = errors.New("crypto: key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256")
	ErrCiphertextShort  = errors.New("crypto: ciphertext too short")
	ErrDecryptionFailed = errors.New("crypto: decryption failed")
)

// AESEncrypt encrypts plaintext using AES-GCM with the given key.
// The key must be 16, 24, or 32 bytes long (AES-128, AES-192, AES-256).
// Returns the nonce prepended to the ciphertext.
func AESEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKeySize, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: failed to generate nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// AESDecrypt decrypts ciphertext that was encrypted with AESEncrypt.
// Expects the nonce to be prepended to the ciphertext.
func AESDecrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKeySize, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextShort
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// AESEncryptString encrypts a plaintext string and returns a base64-encoded ciphertext.
func AESEncryptString(key []byte, plaintext string) (string, error) {
	encrypted, err := AESEncrypt(key, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// AESDecryptString decrypts a base64-encoded ciphertext string.
func AESDecryptString(key []byte, encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: invalid base64 input: %w", err)
	}
	plaintext, err := AESDecrypt(key, ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// GenerateAESKey generates a random AES key of the specified byte size.
// Valid sizes are 16 (AES-128), 24 (AES-192), or 32 (AES-256).
func GenerateAESKey(size int) ([]byte, error) {
	if size != 16 && size != 24 && size != 32 {
		return nil, ErrInvalidKeySize
	}
	key := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("crypto: failed to generate key: %w", err)
	}
	return key, nil
}
