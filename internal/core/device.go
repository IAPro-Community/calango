package core

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// ErrUnauthorized means a credential is unknown, revoked, already used or expired.
// Messages never say which, so a caller cannot probe pairing codes.
var ErrUnauthorized = errors.New("unauthorized")

// PairingCodeTTL is how long a pairing code stays usable after it is issued.
const PairingCodeTTL = 5 * time.Minute

// PairingCodeLength is the number of characters in a pairing code.
const PairingCodeLength = 8

// pairingCodeAlphabet has 32 symbols with the look-alikes I, O, 0 and 1 left out.
// 32 divides 256, so masking a random byte with 31 gives every symbol the same odds.
const pairingCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// MaxDeviceNameLength bounds the name a device gives itself when it pairs.
const MaxDeviceNameLength = 100

// DeviceScope limits what a paired device may call.
type DeviceScope string

// Device scopes. The desktop's local token always acts with full scope.
const (
	ScopeFull        DeviceScope = "full"
	ScopeApproveOnly DeviceScope = "approve_only"
	ScopeRead        DeviceScope = "read"
)

// Validate returns ErrInvalidInput unless s is a known scope.
func (s DeviceScope) Validate() error {
	switch s {
	case ScopeFull, ScopeApproveOnly, ScopeRead:
		return nil
	}
	return fmt.Errorf("%w: unknown device scope %q", ErrInvalidInput, s)
}

// Permission is what a route needs from the scope of the caller.
type Permission string

// Permissions. Routes declare one; scopes grant a set of them.
const (
	PermRead    Permission = "read"
	PermWrite   Permission = "write"
	PermApprove Permission = "approve"
)

// Allows reports whether scope s grants permission p.
func (s DeviceScope) Allows(p Permission) bool {
	switch s {
	case ScopeFull:
		return true
	case ScopeApproveOnly:
		return p == PermApprove
	case ScopeRead:
		return p == PermRead
	}
	return false
}

// ValidatePlatform returns ErrInvalidInput unless p is a known platform.
func ValidatePlatform(p string) error {
	switch p {
	case "desktop", "ios", "android", "web":
		return nil
	}
	return fmt.Errorf("%w: unknown platform %q", ErrInvalidInput, p)
}

// Device is a paired client. TokenDigest holds the SHA-256 of its token; the token
// itself is never stored.
type Device struct {
	ID          string
	OwnerID     string
	Name        string
	Platform    string
	Scope       DeviceScope
	TokenDigest string
	CreatedAt   time.Time
	LastSeenAt  time.Time // zero when the device has never authenticated
	RevokedAt   time.Time // zero while the device is active
}

// Active reports whether the device may still authenticate.
func (d Device) Active() bool { return d.RevokedAt.IsZero() }

// PairingCode is a single-use grant to pair one device under an owner with a scope.
// Its digest is stored by the repository, never the code itself.
type PairingCode struct {
	OwnerID   string
	Scope     DeviceScope
	ExpiresAt time.Time
}

// NewPairingCode returns a random pairing code of PairingCodeLength characters.
func NewPairingCode() (string, error) {
	var raw [PairingCodeLength]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("core: read random pairing code: %w", err)
	}
	out := make([]byte, PairingCodeLength)
	for i, b := range raw {
		out[i] = pairingCodeAlphabet[b&31]
	}
	return string(out), nil
}

// NormalizePairingCode drops spaces and dashes and upper-cases the code, so people
// can type it in any case or grouping.
func NormalizePairingCode(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToUpper(r)
	}, s)
}

// ValidPairingCode reports whether s, already normalized, has the code's shape.
func ValidPairingCode(s string) bool {
	if len(s) != PairingCodeLength {
		return false
	}
	for i := range len(s) {
		if strings.IndexByte(pairingCodeAlphabet, s[i]) < 0 {
			return false
		}
	}
	return true
}
