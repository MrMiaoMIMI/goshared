// Package idutil provides functions for generating various types of unique identifiers.
package idutil

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// UUID generates a random UUID v4 string (xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx).
// Uses crypto/rand for cryptographic randomness.
func UUID() string {
	var uuid [16]byte
	if _, err := io.ReadFull(rand.Reader, uuid[:]); err != nil {
		panic(fmt.Sprintf("idutil: failed to generate UUID: %v", err))
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// ShortID generates a short URL-safe ID of the given byte length (output is 2x length in hex).
// Default length is 8 bytes → 16 hex characters.
func ShortID(byteLen int) string {
	if byteLen <= 0 {
		byteLen = 8
	}
	b := make([]byte, byteLen)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(fmt.Sprintf("idutil: failed to generate ShortID: %v", err))
	}
	return hex.EncodeToString(b)
}

const nanoIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_-"

// NanoID generates a NanoID-like URL-safe random ID of the given length.
// Uses a 64-character alphabet: [A-Za-z0-9_-].
// Default length is 21 characters.
func NanoID(length int) string {
	if length <= 0 {
		length = 21
	}
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		panic(fmt.Sprintf("idutil: failed to generate NanoID: %v", err))
	}
	for i := range bytes {
		bytes[i] = nanoIDAlphabet[bytes[i]&63]
	}
	return string(bytes)
}

// ObjectID generates a 24-character hex string similar to MongoDB ObjectID.
// Format: 4-byte timestamp + 5-byte random + 3-byte counter.
func ObjectID() string {
	var b [12]byte
	binary.BigEndian.PutUint32(b[0:4], uint32(time.Now().Unix()))
	if _, err := io.ReadFull(rand.Reader, b[4:9]); err != nil {
		panic(fmt.Sprintf("idutil: failed to generate ObjectID: %v", err))
	}
	counter := objectIDCounter.Add(1)
	b[9] = byte(counter >> 16)
	b[10] = byte(counter >> 8)
	b[11] = byte(counter)
	return hex.EncodeToString(b[:])
}

var objectIDCounter atomic.Uint32

func init() {
	var seed [4]byte
	if _, err := io.ReadFull(rand.Reader, seed[:]); err != nil {
		panic(fmt.Sprintf("idutil: failed to seed ObjectID counter: %v", err))
	}
	objectIDCounter.Store(binary.BigEndian.Uint32(seed[:]))
}

// Snowflake generates a 64-bit snowflake-like ID.
// Layout: 41 bits timestamp (ms) + 10 bits machine + 12 bits sequence.
// This is a simplified version; for production use with multiple machines,
// set the machine ID via SetSnowflakeMachineID before calling Snowflake.
func Snowflake() int64 {
	return snowflakeGen.Next()
}

// SetSnowflakeMachineID sets the machine ID for snowflake ID generation (0-1023).
// Must be called before any Snowflake() calls (typically in init or main).
func SetSnowflakeMachineID(id int64) {
	if id < 0 || id > 1023 {
		panic("idutil: machine ID must be between 0 and 1023")
	}
	snowflakeGen.machineID.Store(id)
}

var snowflakeGen = &snowflakeGenerator{epoch: 1700000000000}

type snowflakeGenerator struct {
	epoch     int64
	machineID atomic.Int64
	sequence  atomic.Int64
	lastTime  atomic.Int64
}

func (g *snowflakeGenerator) Next() int64 {
	mid := g.machineID.Load()
	for {
		now := time.Now().UnixMilli() - g.epoch
		last := g.lastTime.Load()

		if now > last {
			if g.lastTime.CompareAndSwap(last, now) {
				g.sequence.Store(0)
				return (now << 22) | (mid << 12) | 0
			}
			continue
		}

		seq := g.sequence.Add(1)
		if seq < 4096 {
			return (last << 22) | (mid << 12) | seq
		}

		time.Sleep(time.Millisecond)
	}
}

// PrefixedID generates an ID with a prefix, e.g. "usr_a1b2c3d4e5f6".
func PrefixedID(prefix string, byteLen int) string {
	return prefix + "_" + ShortID(byteLen)
}
