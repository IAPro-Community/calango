// Package auth holds the local bearer token that protects the node API in phase 1.
// Device tokens and pairing (F1-T15) replace it later; until then the token
// identifies the desktop that runs the node.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// TokenFileName is the file under the data directory that stores the token.
const TokenFileName = "api-token"

// LoadOrCreateToken returns the bearer token stored in dir, creating it on first use.
// The file holds 256 random bits and must be readable only by its owner.
func LoadOrCreateToken(dir string) (string, error) {
	path := filepath.Join(dir, TokenFileName)
	token, err := readToken(path)
	if err == nil {
		return token, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	token, err = newToken()
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			// Another start created it first; use that token.
			return readToken(path)
		}
		return "", fmt.Errorf("create api token: %w", err)
	}
	if _, err := f.WriteString(token + "\n"); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("write api token: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close api token: %w", err)
	}
	return token, nil
}

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
