package config

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultsLicenseFileToDataDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", dir)
	t.Setenv("CALANGO_LICENSE_FILE", "")
	t.Setenv("CALANGO_LICENSE_PUBLIC_KEY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if want := filepath.Join(dir, DefaultLicenseFileName); cfg.LicenseFile != want {
		t.Fatalf("Load().LicenseFile = %q, want %q", cfg.LicenseFile, want)
	}
	if cfg.LicensePublicKey != "" {
		t.Fatalf("Load().LicensePublicKey = %q, want empty", cfg.LicensePublicKey)
	}
}

func TestLoadReadsLicenseSettings(t *testing.T) {
	file := filepath.Join(t.TempDir(), "license")
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", t.TempDir())
	t.Setenv("CALANGO_LICENSE_FILE", file)
	t.Setenv("CALANGO_LICENSE_PUBLIC_KEY", key+"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LicenseFile != file || cfg.LicensePublicKey != key {
		t.Fatalf("Load() license = %q, %q", cfg.LicenseFile, cfg.LicensePublicKey)
	}
}

func TestLoadRejectsInvalidLicenseSettings(t *testing.T) {
	for name, env := range map[string][2]string{
		"relative file":   {"license", ""},
		"padded file":     {" " + filepath.Join(t.TempDir(), "license"), ""},
		"key not base64":  {"", "not a key"},
		"key wrong size":  {"", base64.StdEncoding.EncodeToString([]byte("short"))},
		"key with spaces": {"", strings.Repeat("A", 20) + " " + strings.Repeat("A", 23)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("CALANGO_ADDR", "")
			t.Setenv("CALANGO_DATA_DIR", t.TempDir())
			t.Setenv("CALANGO_LICENSE_FILE", env[0])
			t.Setenv("CALANGO_LICENSE_PUBLIC_KEY", env[1])
			_, err := Load()
			if err == nil {
				t.Fatalf("Load() with %s succeeded, want validation error", name)
			}
			if env[1] != "" && strings.Contains(err.Error(), env[1]) {
				t.Fatalf("Load() error echoes the value: %v", err)
			}
		})
	}
}
