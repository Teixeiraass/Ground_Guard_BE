package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken returns the sha256 hex digest of a token. Tokens sent by email
// (verification / password reset) are stored hashed, never in plain text,
// the same way passwords are never stored in plain text.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
