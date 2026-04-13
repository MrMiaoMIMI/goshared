package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func GenHashKey(salts ...string) string {
	hash := sha256.New()
	for _, salt := range salts {
		hash.Write([]byte(salt))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func GenRandomHashKey() string {
	return GenHashKey(fmt.Sprintf("%d", time.Now().UnixNano()))
}
