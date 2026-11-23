package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

// NewDeviceToken returns a fresh 256-bit bearer token for a paired device.
// The caller shows it once and stores only Digest(token).
func NewDeviceToken() (string, error) {
	return newToken()
}

// Digest returns the hex SHA-256 of a secret. Device tokens and pairing codes are
// stored only as digests, so a leaked database does not leak usable credentials.
func Digest(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
