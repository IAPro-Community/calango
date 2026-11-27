//go:build windows

package auth

import (
	"context"
	"os"
	"testing"
)

// On Windows os.Stat reports 0666 for any writable file, so the legacy file looks group-
// and world-readable. The import must accept it anyway (see readToken).
func TestImportFileTokenAcceptsLegacyFileOnWindows(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	secrets := newTestVault(t)
	const token = "windows-token-value-0123456789abcdef"
	path := writeLegacyToken(t, dir, token, 0o600)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat legacy token: %v", err)
	}
	if info.Mode().Perm()&0o077 == 0 {
		t.Fatalf("mode %v has no group or world bits; the test no longer covers the Windows case", info.Mode().Perm())
	}

	imported, err := ImportFileToken(ctx, dir, secrets)
	if err != nil || !imported {
		t.Fatalf("import = %v, %v; want true, nil", imported, err)
	}
	got, err := secrets.Reveal(ctx, VaultTokenName)
	if err != nil || got != token {
		t.Fatalf("vault token = %q (err %v), want the legacy value", got, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("legacy file still present after import: %v", err)
	}
}
