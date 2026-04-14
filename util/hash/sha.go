package hash

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
)

// SHA256 returns the hex-encoded SHA-256 hash of the data.
func SHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// SHA256String returns the hex-encoded SHA-256 hash of a string.
func SHA256String(s string) string {
	return SHA256([]byte(s))
}

// SHA512 returns the hex-encoded SHA-512 hash of the data.
func SHA512(data []byte) string {
	h := sha512.Sum512(data)
	return hex.EncodeToString(h[:])
}

// SHA512String returns the hex-encoded SHA-512 hash of a string.
func SHA512String(s string) string {
	return SHA512([]byte(s))
}

// MD5 returns the hex-encoded MD5 hash of the data.
// Note: MD5 is cryptographically broken; use only for checksums, not security.
func MD5(data []byte) string {
	h := md5.Sum(data)
	return hex.EncodeToString(h[:])
}

// MD5String returns the hex-encoded MD5 hash of a string.
func MD5String(s string) string {
	return MD5([]byte(s))
}

// HMACSHA256 returns the hex-encoded HMAC-SHA256 of data using the given key.
func HMACSHA256(key, data []byte) string {
	return computeHMAC(sha256.New, key, data)
}

// HMACSHA512 returns the hex-encoded HMAC-SHA512 of data using the given key.
func HMACSHA512(key, data []byte) string {
	return computeHMAC(sha512.New, key, data)
}

func computeHMAC(hashFn func() hash.Hash, key, data []byte) string {
	mac := hmac.New(hashFn, key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// GenHashKey generates a SHA-256 hash from multiple salt strings concatenated.
func GenHashKey(salts ...string) string {
	h := sha256.New()
	for _, salt := range salts {
		h.Write([]byte(salt))
	}
	return hex.EncodeToString(h.Sum(nil))
}
