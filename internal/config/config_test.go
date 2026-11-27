package config

import (
	"path/filepath"
	"testing"
)

func TestLoadUsesDefaultAddress(t *testing.T) {
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Addr != defaultAddr {
		t.Fatalf("Load().Addr = %q, want %q", cfg.Addr, defaultAddr)
	}
	if !filepath.IsAbs(cfg.DataDir) || filepath.Base(cfg.DataDir) != "calango" {
		t.Fatalf("Load().DataDir = %q, want an absolute Calango data directory", cfg.DataDir)
	}
}

func TestLoadReadsConfiguredAddress(t *testing.T) {
	t.Setenv("CALANGO_ADDR", "localhost:8080")
	t.Setenv("CALANGO_DATA_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Addr != "localhost:8080" {
		t.Fatalf("Load().Addr = %q, want localhost:8080", cfg.Addr)
	}
}

func TestLoadRejectsInvalidAddress(t *testing.T) {
	for _, addr := range []string{
		"localhost",
		"localhost:http",
		"localhost:0",
		"localhost:65536",
		":7317",
		"invalid host:7317",
		"-invalid:7317",
		" localhost:7317",
	} {
		t.Run(addr, func(t *testing.T) {
			t.Setenv("CALANGO_ADDR", addr)
			t.Setenv("CALANGO_DATA_DIR", "")
			if _, err := Load(); err == nil {
				t.Fatalf("Load() with CALANGO_ADDR=%q succeeded, want validation error", addr)
			}
		})
	}
}

func TestLoadReadsConfiguredDataDir(t *testing.T) {
	expected := t.TempDir()
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", expected)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DataDir != expected {
		t.Fatalf("Load().DataDir = %q, want %q", cfg.DataDir, expected)
	}
}

func TestLoadRejectsDataDirWithSurroundingWhitespace(t *testing.T) {
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", " "+t.TempDir())

	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted CALANGO_DATA_DIR with surrounding whitespace")
	}
}

func TestPriceTableConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prices.json")
	t.Setenv("CALANGO_PRICE_TABLE_FILE", path)
	cfg, err := Load()
	if err != nil || cfg.PriceTableFile != path {
		t.Fatalf("valid tariff configuration rejected: %v", err)
	}
	for _, invalid := range []string{"relative.json", " " + path, path + " "} {
		t.Setenv("CALANGO_PRICE_TABLE_FILE", invalid)
		if _, err := Load(); err == nil {
			t.Fatal("accepted invalid tariff path")
		}
		cfg, err := LoadForPrintToken()
		if err != nil || cfg.PriceTableFile != "" {
			t.Fatalf("print-token depends on tariff: %v", err)
		}
	}
	cfg = Config{Addr: defaultAddr, DataDir: t.TempDir(), PriceTableFile: path + "\x00"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("accepted NUL in tariff path")
	}
}
