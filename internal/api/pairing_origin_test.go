package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

// proxyCall sends one POST /v1/pair from peer. An empty forwarded value sends no header.
func proxyCall(t *testing.T, h http.Handler, peer, forwarded string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"code":"AAAAAAAA","device_name":"Phone","platform":"ios"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/pair", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = peer
	if forwarded != "" {
		req.Header.Set("X-Forwarded-For", forwarded)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func trustedLAN() []netip.Prefix {
	return []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}
}

func TestPairingLimitFollowsForwardedClientBehindTrustedProxy(t *testing.T) {
	h, _ := newTestEnvWith(t, trustedLAN())
	for range pairAttemptsPerWindow {
		proxyCall(t, h, "192.0.2.10:4000", "198.51.100.7")
	}
	expectStatus(t, proxyCall(t, h, "192.0.2.10:4000", "198.51.100.7"), http.StatusTooManyRequests)
	// Another client behind the same proxy has its own bucket.
	expectStatus(t, proxyCall(t, h, "192.0.2.10:4000", "198.51.100.8"), http.StatusUnauthorized)
}

func TestPairingLimitIgnoresForwardedHeaderFromUntrustedPeer(t *testing.T) {
	h, _ := newTestEnvWith(t, trustedLAN())
	for i := range pairAttemptsPerWindow {
		proxyCall(t, h, "203.0.113.5:4000", fmt.Sprintf("198.51.100.%d", i+1))
	}
	expectStatus(t, proxyCall(t, h, "203.0.113.5:4000", "198.51.100.99"), http.StatusTooManyRequests)
}

func TestPairingLimitFallsBackToProxyWhenForwardedChainIsMalformed(t *testing.T) {
	h, _ := newTestEnvWith(t, trustedLAN())
	for i := range pairAttemptsPerWindow {
		proxyCall(t, h, "192.0.2.10:4000", fmt.Sprintf("garbage-%d", i))
	}
	expectStatus(t, proxyCall(t, h, "192.0.2.10:4000", "garbage-x"), http.StatusTooManyRequests)
}
