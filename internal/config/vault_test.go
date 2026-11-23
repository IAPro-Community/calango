package config

import (
	"strings"
	"testing"
)

func TestLoadReadsVaultKeySettings(t *testing.T) {
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", "")
	t.Setenv("CALANGO_VAULT_KEY", "")
	t.Setenv("CALANGO_VAULT_KEY_FILE", "/run/secrets/calango-vault")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.VaultKeyFile != "/run/secrets/calango-vault" || cfg.VaultKey != "" {
		t.Fatalf("vault settings = key %q file %q", cfg.VaultKey, cfg.VaultKeyFile)
	}
}

func TestLoadRejectsBothVaultKeySources(t *testing.T) {
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", "")
	t.Setenv("CALANGO_VAULT_KEY", "c2VjcmV0LXZhbHVlLWRvLW5vdC1wcmludA==")
	t.Setenv("CALANGO_VAULT_KEY_FILE", "/run/secrets/calango-vault")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("err = %v, want a mutually exclusive refusal", err)
	}
	if strings.Contains(err.Error(), "c2VjcmV0") {
		t.Fatalf("error leaks the key value: %v", err)
	}
}
