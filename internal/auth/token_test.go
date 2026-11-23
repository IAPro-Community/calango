package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOrCreateTokenCreatesPrivateFileAndReusesIt(t *testing.T) {
	dir := t.TempDir()

	first, err := LoadOrCreateToken(dir)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if len(first) < 32 {
		t.Fatalf("token has %d characters; want at least 32", len(first))
	}

	info, err := os.Stat(filepath.Join(dir, TokenFileName))
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("token file mode = %o, want 600", perm)
	}

	second, err := LoadOrCreateToken(dir)
	if err != nil {
		t.Fatalf("reload token: %v", err)
	}
	if second != first {
		t.Fatal("second call must return the stored token, not a new one")
	}
}

func TestLoadOrCreateTokenRejectsGroupOrWorldReadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, TokenFileName)
	if err := os.WriteFile(path, []byte("secret-token-value-0123456789"), 0o644); err != nil {
		t.Fatalf("write token: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod token: %v", err)
	}

	_, err := LoadOrCreateToken(dir)
	if err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Fatalf("err = %v, want a refusal that names chmod 600", err)
	}
}

func TestLoadOrCreateTokenRejectsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, TokenFileName), nil, 0o600); err != nil {
		t.Fatalf("write empty token: %v", err)
	}

	if _, err := LoadOrCreateToken(dir); err == nil {
		t.Fatal("empty token file must be rejected, not used as a bearer secret")
	}
}
