//go:build !windows

package auth

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The POSIX permission check applies only outside Windows (see readToken).

func TestImportFileTokenRejectsGroupOrWorldReadableFile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	secrets := newTestVault(t)
	path := writeLegacyToken(t, dir, "secret-token-value-0123456789", 0o644)

	_, err := ImportFileToken(ctx, dir, secrets)
	if err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Fatalf("err = %v, want a refusal that names chmod 600", err)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("file must not be removed on refusal: %v", statErr)
	}
	if _, revealErr := secrets.Reveal(ctx, VaultTokenName); revealErr == nil {
		t.Fatal("a refused file must not reach the vault")
	}
}

func TestImportFileTokenRejectsSymlink(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	secrets := newTestVault(t)
	// The link target is owner-only, so only the link itself can cause the refusal.
	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.WriteFile(target, []byte("linked-token-value-0123456789\n"), 0o600); err != nil {
		t.Fatalf("write link target: %v", err)
	}
	path := filepath.Join(dir, TokenFileName)
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err := ImportFileToken(ctx, dir, secrets)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("err = %v, want a refusal for a symlink", err)
	}
	if _, statErr := os.Lstat(path); statErr != nil {
		t.Fatalf("link must not be removed on refusal: %v", statErr)
	}
	if _, statErr := os.Stat(target); statErr != nil {
		t.Fatalf("link target must not be touched: %v", statErr)
	}
	if _, revealErr := secrets.Reveal(ctx, VaultTokenName); revealErr == nil {
		t.Fatal("a refused link must not reach the vault")
	}
}
