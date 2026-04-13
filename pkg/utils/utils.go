package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"time"
)

// GenerateID creates a unique ID using timestamp and random bytes
func GenerateID(prefix string) string {
	return prefix + "-" + hex.EncodeToString(randomBytes(8))
}

// Hash creates a SHA256 hash of the input string
func Hash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// randomBytes generates random bytes
func randomBytes(length int) []byte {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return b
}
