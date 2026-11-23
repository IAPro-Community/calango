package core

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

const crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewID returns a 26-character ULID-style identifier: a 48-bit millisecond
// timestamp followed by 80 random bits, encoded in Crockford base32. Identifiers
// created later sort later as strings, so they are safe as index keys.
func NewID(now time.Time) (string, error) {
	ms := now.UnixMilli()
	if ms < 0 || ms >= 1<<48 {
		return "", fmt.Errorf("core: id timestamp %d outside 48-bit range", ms)
	}

	var raw [16]byte
	for i := range 6 {
		raw[i] = byte(ms >> (8 * (5 - i)))
	}
	if _, err := rand.Read(raw[6:]); err != nil {
		return "", fmt.Errorf("core: read random id bytes: %w", err)
	}

	n := new(big.Int).SetBytes(raw[:])
	base := big.NewInt(32)
	digit := new(big.Int)
	buf := make([]byte, 26)
	for i := len(buf) - 1; i >= 0; i-- {
		n.DivMod(n, base, digit)
		buf[i] = crockfordAlphabet[digit.Int64()]
	}
	return string(buf), nil
}
