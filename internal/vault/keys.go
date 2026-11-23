// Derived from vstorm-co/agenticos (Apache-2.0) at a4b0b3c: backend/app/core/vault.py. Modified.
//
// Sources of the master key. Desktop keeps it in the OS keyring; the VPS takes it from
// a deployment secret (ADR 0012). Neither source falls back to plain text.

package vault

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

// ErrKeyNotFound is returned by a Keyring when no entry exists for the account.
var ErrKeyNotFound = errors.New("vault: keyring entry not found")

// ErrKeyringUnavailable is returned when the OS keyring cannot be read or written.
// On Linux this usually means Secret Service is locked or absent. The caller must
// tell the owner to unlock it and retry; storing secrets without it is refused.
var ErrKeyringUnavailable = errors.New("vault: OS keyring unavailable; unlock it and retry, secrets are not stored in plain text")

// keyringService is the service name under which master keys are stored.
const keyringService = "calango"

// Keyring is the seam over the OS credential store.
type Keyring interface {
	Get(service, account string) (string, error)
	Set(service, account, secret string) error
}

// OSKeyring uses the operating system's credential store through go-keyring:
// Keychain on macOS, Credential Manager on Windows, Secret Service on Linux.
type OSKeyring struct{}

// Get returns the secret stored for the account, or ErrKeyNotFound.
func (OSKeyring) Get(service, account string) (string, error) {
	secret, err := keyring.Get(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrKeyNotFound
	}
	return secret, err
}

// Set stores the secret for the account in the OS keyring.
func (OSKeyring) Set(service, account, secret string) error {
	return keyring.Set(service, account, secret)
}

// MasterKeyAccount is the keyring account that holds the master key for a version.
func MasterKeyAccount(version int) string {
	return fmt.Sprintf("vault-master-v%d", version)
}

// LoadOrCreateKeyringKey returns the master key for a version from the keyring. On the
// first call it generates a random key and stores it. Any keyring error other than
// "not found" is returned as ErrKeyringUnavailable: the caller does not get a key and
// must not store secrets.
func LoadOrCreateKeyringKey(kr Keyring, version int) ([]byte, error) {
	account := MasterKeyAccount(version)
	encoded, err := kr.Get(keyringService, account)
	if err == nil {
		return ParseMasterKey(encoded)
	}
	if !errors.Is(err, ErrKeyNotFound) {
		return nil, fmt.Errorf("%w: %v", ErrKeyringUnavailable, err)
	}

	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("vault: generate master key: %w", err)
	}
	if err := kr.Set(keyringService, account, base64.StdEncoding.EncodeToString(key)); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyringUnavailable, err)
	}
	return key, nil
}

// ParseMasterKey decodes a base64 master key from a deployment secret or the keyring.
// The error never includes the input.
func ParseMasterKey(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, errors.New("vault: master key is not valid base64")
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("vault: master key must decode to %d bytes", keySize)
	}
	return key, nil
}
