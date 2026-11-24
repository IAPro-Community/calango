package config

import (
	"net/netip"
	"strings"
	"testing"
)

func TestTrustedProxiesParsesAddressesAndRanges(t *testing.T) {
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", t.TempDir())
	t.Setenv("CALANGO_TRUSTED_PROXIES", " 10.0.0.1 , 192.168.0.0/16 ,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.TrustedProxies) != 2 {
		t.Fatalf("TrustedProxies = %v, want 2 entries", cfg.TrustedProxies)
	}
	contains := func(raw string) bool {
		addr := netip.MustParseAddr(raw)
		for _, p := range cfg.TrustedProxies {
			if p.Contains(addr) {
				return true
			}
		}
		return false
	}
	if !contains("10.0.0.1") || !contains("192.168.44.2") {
		t.Fatalf("listed proxies are not trusted: %v", cfg.TrustedProxies)
	}
	if contains("10.0.0.2") {
		t.Fatalf("10.0.0.2 is trusted but was not listed: %v", cfg.TrustedProxies)
	}
}

func TestTrustedProxiesRejectsMalformedEntries(t *testing.T) {
	t.Setenv("CALANGO_ADDR", "")
	t.Setenv("CALANGO_DATA_DIR", t.TempDir())
	t.Setenv("CALANGO_TRUSTED_PROXIES", "10.0.0.0/33")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "CALANGO_TRUSTED_PROXIES") {
		t.Fatalf("Load() error = %v, want a CALANGO_TRUSTED_PROXIES error", err)
	}
}
