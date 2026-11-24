package config

import (
	"strings"
	"testing"
)

// setVaultEnv clears every vault variable so each test states only what it sets.
func setVaultEnv(t *testing.T, env map[string]string) {
	t.Helper()
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", "")
	for _, name := range []string{
		"CALANGO_VAULT_KEY", "CALANGO_VAULT_KEY_FILE", "CALANGO_VAULT_KEY_VERSION",
		"CALANGO_VAULT_KEY_PREVIOUS", "CALANGO_VAULT_KEY_PREVIOUS_FILE",
	} {
		t.Setenv(name, env[name])
	}
}

func TestLoadReadsRotationSettings(t *testing.T) {
	setVaultEnv(t, map[string]string{
		"CALANGO_VAULT_KEY_FILE":          "/run/secrets/calango-vault-v2",
		"CALANGO_VAULT_KEY_VERSION":       "2",
		"CALANGO_VAULT_KEY_PREVIOUS_FILE": "/run/secrets/calango-vault-v1",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.VaultKeyVersion != 2 || cfg.VaultKeyPreviousFile != "/run/secrets/calango-vault-v1" || cfg.VaultKeyPrevious != "" {
		t.Fatalf("rotation settings = version %d previous file %q", cfg.VaultKeyVersion, cfg.VaultKeyPreviousFile)
	}
}

func TestLoadDefaultsVaultKeyVersionToOne(t *testing.T) {
	setVaultEnv(t, map[string]string{"CALANGO_VAULT_KEY_FILE": "/run/secrets/calango-vault"})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.VaultKeyVersion != 1 {
		t.Fatalf("VaultKeyVersion = %d, want 1", cfg.VaultKeyVersion)
	}
}

func TestLoadRejectsMalformedVaultKeyVersion(t *testing.T) {
	for _, raw := range []string{"0", "-1", "two"} {
		setVaultEnv(t, map[string]string{
			"CALANGO_VAULT_KEY_FILE":    "/run/secrets/calango-vault",
			"CALANGO_VAULT_KEY_VERSION": raw,
		})
		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "CALANGO_VAULT_KEY_VERSION") {
			t.Fatalf("version %q: err = %v, want a CALANGO_VAULT_KEY_VERSION refusal", raw, err)
		}
	}
}

func TestLoadRejectsBothPreviousKeySources(t *testing.T) {
	setVaultEnv(t, map[string]string{
		"CALANGO_VAULT_KEY_FILE":          "/run/secrets/calango-vault-v2",
		"CALANGO_VAULT_KEY_VERSION":       "2",
		"CALANGO_VAULT_KEY_PREVIOUS":      "c2VjcmV0LXZhbHVlLWRvLW5vdC1wcmludA==",
		"CALANGO_VAULT_KEY_PREVIOUS_FILE": "/run/secrets/calango-vault-v1",
	})

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("err = %v, want a mutually exclusive refusal", err)
	}
	if strings.Contains(err.Error(), "c2VjcmV0") {
		t.Fatalf("error leaks the previous key: %v", err)
	}
}

func TestLoadRejectsPreviousKeyWithoutCurrentKey(t *testing.T) {
	setVaultEnv(t, map[string]string{
		"CALANGO_VAULT_KEY_PREVIOUS": "c2VjcmV0LXZhbHVlLWRvLW5vdC1wcmludA==",
	})

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CALANGO_VAULT_KEY_PREVIOUS: needs") {
		t.Fatalf("err = %v, want a refusal without a current key", err)
	}
}

func TestLoadRejectsPreviousKeyAtVersionOne(t *testing.T) {
	setVaultEnv(t, map[string]string{
		"CALANGO_VAULT_KEY_FILE":     "/run/secrets/calango-vault",
		"CALANGO_VAULT_KEY_PREVIOUS": "c2VjcmV0LXZhbHVlLWRvLW5vdC1wcmludA==",
	})

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "at least 2") {
		t.Fatalf("err = %v, want a refusal at version 1", err)
	}
}

func TestLoadRejectsVersionTwoWithoutExplicitKey(t *testing.T) {
	setVaultEnv(t, map[string]string{"CALANGO_VAULT_KEY_VERSION": "2"})

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "OS keyring") {
		t.Fatalf("err = %v, want a refusal to version the keyring key", err)
	}
}
