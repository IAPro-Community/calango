// Derived from vstorm-co/agenticos (Apache-2.0) at a4b0b3c: backend/app/core/vault.py. Modified.
//
// Envelope encryption for secrets at rest. A random data key seals each value.
// The data key is wrapped by a key derived from the master key, the owner scope
// and the master key version, so a ciphertext cannot be moved between owners or
// unwrapped with a different version. Rotation re-wraps the data key only.
// Algorithms differ from upstream (AES-256-GCM instead of Fernet); see ADR 0012.

package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// envelopeVersion is the only envelope format Calango writes or reads.
const envelopeVersion = 1

// keySize is the length of the master key, the data key and each derived key.
const keySize = 32

// ErrSealed is returned for every failure to open or rewrap an envelope. Wrong key,
// wrong owner, wrong version and tampering look the same to the caller on purpose.
var ErrSealed = errors.New("vault: secret cannot be opened")

// ErrNoMasterKey is returned when an envelope names a master key version that is not
// configured. Restore the key until every row has been rewrapped.
var ErrNoMasterKey = errors.New("vault: no master key configured for this version")

// MasterKeys maps each configured master key version to its key. The highest version
// seals new secrets.
type MasterKeys struct {
	keys    map[int][]byte
	current int
}

// NewMasterKeys validates the keys and picks the current version. Versions start at 1.
func NewMasterKeys(keys map[int][]byte) (MasterKeys, error) {
	if len(keys) == 0 {
		return MasterKeys{}, errors.New("vault: no master key configured")
	}
	current := 0
	copied := make(map[int][]byte, len(keys))
	for version, key := range keys {
		if version < 1 {
			return MasterKeys{}, fmt.Errorf("vault: master key version %d must be >= 1", version)
		}
		if len(key) != keySize {
			return MasterKeys{}, fmt.Errorf("vault: master key version %d must be %d bytes", version, keySize)
		}
		if version > current {
			current = version
		}
		copied[version] = append([]byte(nil), key...)
	}
	return MasterKeys{keys: copied, current: current}, nil
}

// Current returns the version new secrets are sealed under.
func (m MasterKeys) Current() int { return m.current }

// Has reports whether a version is configured.
func (m MasterKeys) Has(version int) bool {
	_, ok := m.keys[version]
	return ok
}

func (m MasterKeys) key(version int) ([]byte, error) {
	key, ok := m.keys[version]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrNoMasterKey, version)
	}
	return key, nil
}

// envelope is the JSON stored in the database. "k" is the wrapped data key, "p" the
// sealed payload. Both are nonce || ciphertext, base64 encoded.
type envelope struct {
	Version    int    `json:"v"`
	WrappedKey string `json:"k"`
	Payload    string `json:"p"`
}

// Seal encrypts plaintext for one owner under the given master key version.
func Seal(plaintext []byte, scope string, keys MasterKeys, version int) (string, error) {
	if len(plaintext) == 0 {
		return "", errors.New("vault: cannot store an empty secret")
	}
	masterKey, err := keys.key(version)
	if err != nil {
		return "", err
	}
	dataKey := make([]byte, keySize)
	if _, err := rand.Read(dataKey); err != nil {
		return "", fmt.Errorf("vault: generate data key: %w", err)
	}
	payload, err := aeadSeal(dataKey, plaintext, payloadAAD(scope))
	if err != nil {
		return "", err
	}
	return encodeEnvelope(dataKey, payload, masterKey, scope, version)
}

// Open decrypts an envelope sealed for this owner under the given version.
func Open(ciphertext string, scope string, keys MasterKeys, version int) ([]byte, error) {
	wrapped, payload, err := parseEnvelope(ciphertext)
	if err != nil {
		return nil, err
	}
	dataKey, err := unwrapDataKey(wrapped, scope, keys, version)
	if err != nil {
		return nil, err
	}
	plaintext, err := aeadOpen(dataKey, payload, payloadAAD(scope))
	if err != nil {
		return nil, ErrSealed
	}
	return plaintext, nil
}

// Rewrap moves an envelope to another master key version. Only the wrapped data key
// is re-sealed; the payload is untouched. The result is always in the current format.
func Rewrap(ciphertext string, scope string, keys MasterKeys, fromVersion, toVersion int) (string, error) {
	wrapped, payload, err := parseEnvelope(ciphertext)
	if err != nil {
		return "", err
	}
	dataKey, err := unwrapDataKey(wrapped, scope, keys, fromVersion)
	if err != nil {
		return "", err
	}
	masterKey, err := keys.key(toVersion)
	if err != nil {
		return "", err
	}
	return encodeEnvelope(dataKey, payload, masterKey, scope, toVersion)
}

// encodeEnvelope wraps dataKey under masterKey and returns the JSON envelope. The
// payload bytes are passed through unchanged.
func encodeEnvelope(dataKey, payload, masterKey []byte, scope string, version int) (string, error) {
	wrapped, err := aeadSeal(wrappingKey(masterKey, scope, version), dataKey, wrapAAD(scope, version))
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(envelope{
		Version:    envelopeVersion,
		WrappedKey: base64.StdEncoding.EncodeToString(wrapped),
		Payload:    base64.StdEncoding.EncodeToString(payload),
	})
	if err != nil {
		return "", fmt.Errorf("vault: encode envelope: %w", err)
	}
	return string(out), nil
}

func parseEnvelope(ciphertext string) (wrapped, payload []byte, err error) {
	var env envelope
	if err := json.Unmarshal([]byte(ciphertext), &env); err != nil {
		return nil, nil, ErrSealed
	}
	if env.Version != envelopeVersion {
		return nil, nil, ErrSealed
	}
	wrapped, err = base64.StdEncoding.DecodeString(env.WrappedKey)
	if err != nil {
		return nil, nil, ErrSealed
	}
	payload, err = base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return nil, nil, ErrSealed
	}
	return wrapped, payload, nil
}

func unwrapDataKey(wrapped []byte, scope string, keys MasterKeys, version int) ([]byte, error) {
	masterKey, err := keys.key(version)
	if err != nil {
		return nil, err
	}
	dataKey, err := aeadOpen(wrappingKey(masterKey, scope, version), wrapped, wrapAAD(scope, version))
	if err != nil || len(dataKey) != keySize {
		return nil, ErrSealed
	}
	return dataKey, nil
}

// wrappingKey derives the key that wraps a data key, bound to the scope and the version.
// It is HKDF-SHA256 (RFC 5869) with an all-zero salt and one output block, which is all
// a 32-byte key needs.
func wrappingKey(masterKey []byte, scope string, version int) []byte {
	extract := hmac.New(sha256.New, make([]byte, sha256.Size))
	extract.Write(masterKey)
	prk := extract.Sum(nil)

	expand := hmac.New(sha256.New, prk)
	expand.Write([]byte(fmt.Sprintf("%s|v%d", scope, version)))
	expand.Write([]byte{0x01})
	return expand.Sum(nil)
}

func payloadAAD(scope string) []byte { return []byte("payload|" + scope) }

func wrapAAD(scope string, version int) []byte {
	return []byte(fmt.Sprintf("wrap|%s|v%d", scope, version))
}

func aeadSeal(key, plaintext, aad []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("vault: generate nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, aad), nil
}

func aeadOpen(key, sealed, aad []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, ErrSealed
	}
	nonce, body := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, body, aad)
	if err != nil {
		return nil, ErrSealed
	}
	return plaintext, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("vault: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vault: gcm: %w", err)
	}
	return gcm, nil
}
