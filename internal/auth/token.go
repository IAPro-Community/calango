// Package auth holds the local bearer token that protects the node API in phase 1.
// Device tokens and pairing (F1-T15) replace it later; until then the token
// identifies the desktop that runs the node. The token lives in the vault (ADR 0012).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"calango/internal/vault"
)

// VaultTokenName is the vault item that holds the local API token.
const VaultTokenName = "api-token"

// TokenFileName is the legacy token file under the data directory. Nodes that ran
// before the vault existed keep it until ImportFileToken moves it.
const TokenFileName = "api-token"

// LoadOrCreateVaultToken returns the API token stored in the vault, creating it on first use.
func LoadOrCreateVaultToken(ctx context.Context, secrets *vault.Service) (string, error) {
	token, err := secrets.Reveal(ctx, VaultTokenName)
	if err == nil {
		return token, nil
	}
	if !errors.Is(err, vault.ErrNotFound) {
		return "", fmt.Errorf("read api token from vault: %w", err)
	}
	token, err = newToken()
	if err != nil {
		return "", err
	}
	if _, err := secrets.Set(ctx, VaultTokenName, token); err != nil {
		return "", fmt.Errorf("store api token in vault: %w", err)
	}
	return token, nil
}

// ImportFileToken moves a token from the legacy 0600 file in dir into the vault and then
// removes the file. It reports whether a file was imported. When the vault already holds
// a different token, it refuses and leaves both untouched: silently dropping either token
// would break a client that still uses it.
func ImportFileToken(ctx context.Context, dir string, secrets *vault.Service) (bool, error) {
	path := filepath.Join(dir, TokenFileName)
	token, err := readToken(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	existing, err := secrets.Reveal(ctx, VaultTokenName)
	switch {
	case err == nil && existing == token:
		// Already imported by an earlier start that stopped before removing the file.
	case err == nil:
		return false, fmt.Errorf("both the vault and %s hold an api token; keep the one clients use and delete the other", path)
	case errors.Is(err, vault.ErrNotFound):
		if _, err := secrets.Set(ctx, VaultTokenName, token); err != nil {
			return false, fmt.Errorf("import api token into vault: %w", err)
		}
	default:
		return false, fmt.Errorf("read api token from vault: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return false, fmt.Errorf("remove imported api token file: %w", err)
	}
	return true, nil
}

// readToken reads the legacy file and enforces its owner-only permissions.
func readToken(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat api token: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("api token %s is readable by other users; run chmod 600", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read api token: %w", err)
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", fmt.Errorf("api token %s is empty", path)
	}
	return token, nil
}

func newToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate api token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

// LocalToken is the live local bearer token. Rotate stores a new token in the vault and
// swaps it in memory, so the old token stops matching on the next request.
type LocalToken struct {
	secrets *vault.Service
	// rotate serialises Rotate so the vault and memory always end on the same token.
	rotate sync.Mutex
	mu     sync.RWMutex
	value  string
}

// NewLocalToken wraps the token loaded from the vault. secrets is required for Rotate.
func NewLocalToken(value string, secrets *vault.Service) (*LocalToken, error) {
	if value == "" {
		return nil, errors.New("auth: local token is empty")
	}
	if secrets == nil {
		return nil, errors.New("auth: vault is required")
	}
	return &LocalToken{secrets: secrets, value: value}, nil
}

// Matches reports whether candidate is the current token, compared in constant time.
func (t *LocalToken) Matches(candidate string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(t.value)) == 1
}

// Rotate generates a new token, stores it in the vault and makes it the live token.
// The vault write comes first and fails closed: if it fails the old token stays live.
func (t *LocalToken) Rotate(ctx context.Context) (string, error) {
	t.rotate.Lock()
	defer t.rotate.Unlock()
	next, err := newToken()
	if err != nil {
		return "", err
	}
	if _, err := t.secrets.Set(ctx, VaultTokenName, next); err != nil {
		return "", fmt.Errorf("store rotated api token in vault: %w", err)
	}
	t.mu.Lock()
	t.value = next
	t.mu.Unlock()
	return next, nil
}
