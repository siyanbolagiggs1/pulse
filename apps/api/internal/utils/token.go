package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateSecureToken returns a cryptographically random hex token of the given byte length.
// 32 bytes → 64 char hex string.
func GenerateSecureToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
