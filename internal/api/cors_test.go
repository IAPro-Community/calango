package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testDesktopOrigin = "app://calango"

func TestCORSPreflightFromListedOriginNeedsNoToken(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodOptions, "/v1/agents", nil)
	req.Header.Set("Origin", testDesktopOrigin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testDesktopOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, testDesktopOrigin)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Fatalf("Access-Control-Allow-Headers = %q, want Authorization", got)
	}
}

func TestCORSListedOriginStillNeedsToken(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/agents", nil)
	req.Header.Set("Origin", testDesktopOrigin)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 without token", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testDesktopOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, testDesktopOrigin)
	}
}

func TestCORSRefusesUnlistedOrigin(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodOptions, "/v1/agents", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q for unlisted origin, want none", got)
	}
	if rec.Code == http.StatusNoContent {
		t.Fatalf("unlisted origin got a CORS preflight answer")
	}
}
