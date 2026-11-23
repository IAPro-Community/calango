package config

import "testing"

func TestLoadUsesDefaultAddress(t *testing.T) {
	t.Setenv("CALANGO_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Addr != defaultAddr {
		t.Fatalf("Load().Addr = %q, want %q", cfg.Addr, defaultAddr)
	}
}

func TestLoadReadsConfiguredAddress(t *testing.T) {
	t.Setenv("CALANGO_ADDR", "localhost:8080")

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
			if _, err := Load(); err == nil {
				t.Fatalf("Load() with CALANGO_ADDR=%q succeeded, want validation error", addr)
			}
		})
	}
}
