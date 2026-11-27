package auth

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/store"
	"calango/internal/vault"
)

const testOwner = "owner-1"

// newTestVault returns a vault service over an in-memory store, as the node wires it.
func newTestVault(t *testing.T) *vault.Service {
	t.Helper()
	ctx := context.Background()
	db, err := store.OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	keys, err := vault.NewMasterKeys(map[int][]byte{1: bytes.Repeat([]byte{0x42}, 32)})
	if err != nil {
		t.Fatalf("master keys: %v", err)
	}
	svc, err := vault.NewService(db, keys, testOwner, db, time.Now, core.NewID)
	if err != nil {
		t.Fatalf("vault service: %v", err)
	}
	return svc
}

func writeLegacyToken(t *testing.T, dir, token string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(dir, TokenFileName)
	if err := os.WriteFile(path, []byte(token+"\n"), mode); err != nil {
		t.Fatalf("write legacy token: %v", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod legacy token: %v", err)
	}
	return path
}

// skipLegacyImportOnWindows pula os testes que esperam importar o arquivo legado. No Windows,
// os.Stat reporta 0666 para qualquer arquivo gravável (as ACLs não viram bits POSIX), então
// readToken recusa todo arquivo legado: bug registrado em DEV/CI_NATIVA.md (issue #120).
func skipLegacyImportOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("bug conhecido no Windows: a checagem de permissão POSIX recusa todo api-token legado (DEV/CI_NATIVA.md)")
	}
}

func TestLoadOrCreateVaultTokenCreatesOnceAndReuses(t *testing.T) {
	ctx := context.Background()
	secrets := newTestVault(t)

	first, err := LoadOrCreateVaultToken(ctx, secrets)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if len(first) < 32 {
		t.Fatalf("token has %d characters; want at least 32", len(first))
	}
	second, err := LoadOrCreateVaultToken(ctx, secrets)
	if err != nil {
		t.Fatalf("reload token: %v", err)
	}
	if second != first {
		t.Fatal("second call must return the stored token, not a new one")
	}
	stored, err := secrets.Reveal(ctx, VaultTokenName)
	if err != nil || stored != first {
		t.Fatalf("vault holds %q (err %v), want the token", stored, err)
	}
}

func TestImportFileTokenMovesLegacyFileIntoVault(t *testing.T) {
	skipLegacyImportOnWindows(t)
	ctx := context.Background()
	dir := t.TempDir()
	secrets := newTestVault(t)
	path := writeLegacyToken(t, dir, "legacy-token-value-0123456789abcdef", 0o600)

	imported, err := ImportFileToken(ctx, dir, secrets)
	if err != nil || !imported {
		t.Fatalf("import = %v, %v; want true, nil", imported, err)
	}
	got, err := secrets.Reveal(ctx, VaultTokenName)
	if err != nil || got != "legacy-token-value-0123456789abcdef" {
		t.Fatalf("vault token = %q (err %v), want the legacy value", got, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("legacy file still present after import: %v", err)
	}

	again, err := ImportFileToken(ctx, dir, secrets)
	if err != nil || again {
		t.Fatalf("second import = %v, %v; want false, nil", again, err)
	}
}

func TestImportFileTokenRefusesConflictingVaultTokenAndKeepsBoth(t *testing.T) {
	skipLegacyImportOnWindows(t)
	ctx := context.Background()
	dir := t.TempDir()
	secrets := newTestVault(t)
	if _, err := secrets.Set(ctx, VaultTokenName, "vault-token-value-0123456789abcdef"); err != nil {
		t.Fatalf("seed vault: %v", err)
	}
	path := writeLegacyToken(t, dir, "legacy-token-value-0123456789abcdef", 0o600)

	imported, err := ImportFileToken(ctx, dir, secrets)
	if err == nil || imported {
		t.Fatalf("import = %v, %v; want a refusal", imported, err)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("legacy file must survive a refused import: %v", statErr)
	}
	got, err := secrets.Reveal(ctx, VaultTokenName)
	if err != nil || got != "vault-token-value-0123456789abcdef" {
		t.Fatalf("vault token changed to %q (err %v)", got, err)
	}
}

func TestImportFileTokenFinishesInterruptedImport(t *testing.T) {
	skipLegacyImportOnWindows(t)
	ctx := context.Background()
	dir := t.TempDir()
	secrets := newTestVault(t)
	const token = "same-token-value-0123456789abcdef"
	if _, err := secrets.Set(ctx, VaultTokenName, token); err != nil {
		t.Fatalf("seed vault: %v", err)
	}
	path := writeLegacyToken(t, dir, token, 0o600)

	imported, err := ImportFileToken(ctx, dir, secrets)
	if err != nil || !imported {
		t.Fatalf("import = %v, %v; want true, nil", imported, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file with an already-imported token must be removed: %v", err)
	}
}

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
}

func TestImportFileTokenWithoutFileIsNoop(t *testing.T) {
	imported, err := ImportFileToken(context.Background(), t.TempDir(), newTestVault(t))
	if err != nil || imported {
		t.Fatalf("import = %v, %v; want false, nil", imported, err)
	}
}

func TestImportFileTokenRejectsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, TokenFileName), nil, 0o600); err != nil {
		t.Fatalf("write empty token: %v", err)
	}

	if _, err := ImportFileToken(context.Background(), dir, newTestVault(t)); err == nil {
		t.Fatal("empty token file must be rejected, not used as a bearer secret")
	}
}

func TestLocalTokenRotationSurvivesARestart(t *testing.T) {
	ctx := context.Background()
	secrets := newTestVault(t)
	first, err := LoadOrCreateVaultToken(ctx, secrets)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	live, err := NewLocalToken(first, secrets)
	if err != nil {
		t.Fatalf("local token: %v", err)
	}

	rotated, err := live.Rotate(ctx)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if live.Matches(first) {
		t.Fatal("old token still matches after rotation")
	}
	if !live.Matches(rotated) {
		t.Fatal("rotated token does not match")
	}

	// A restart reloads the token from the vault and must find the rotated one.
	reloaded, err := LoadOrCreateVaultToken(ctx, secrets)
	if err != nil {
		t.Fatalf("reload token: %v", err)
	}
	if reloaded != rotated {
		t.Fatal("restart loaded the old token; the vault and the live token diverged")
	}
}

func TestLocalTokenKeepsOldTokenWhenVaultWriteFails(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	keys, err := vault.NewMasterKeys(map[int][]byte{1: bytes.Repeat([]byte{0x42}, 32)})
	if err != nil {
		t.Fatalf("master keys: %v", err)
	}
	secrets, err := vault.NewService(db, keys, testOwner, db, time.Now, core.NewID)
	if err != nil {
		t.Fatalf("vault service: %v", err)
	}
	const seed = "seed-token-0123456789"
	live, err := NewLocalToken(seed, secrets)
	if err != nil {
		t.Fatalf("local token: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	// The vault write fails, so rotation must fail and the old token must keep working.
	if _, err := live.Rotate(ctx); err == nil {
		t.Fatal("rotate succeeded although the vault write failed")
	}
	if !live.Matches(seed) {
		t.Fatal("old token stopped matching although no new token was stored")
	}
}
