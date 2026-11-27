package api

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/license"
)

func TestGetLicenseWithoutLicenseIsFree(t *testing.T) {
	h := newTestHandler(t)
	rec := call(t, h, http.MethodGet, "/v1/license", testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/license = %d: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["state"] != "none" || got["plan"] != "free" || got["checked_at"] == nil || got["account_id"] != nil {
		t.Fatalf("GET /v1/license body = %v", got)
	}
	if rec := call(t, h, http.MethodGet, "/v1/license", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/license without token = %d, want 401", rec.Code)
	}
}

func TestGetLicenseReportsGraceThenFree(t *testing.T) {
	priv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{3}, ed25519.SeedSize))
	issued := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	token, err := license.Sign(priv, license.Claims{
		AccountID: "acct-9", Plan: license.PlanPro, Features: []string{"vps"}, NodesMax: 1,
		IssuedAt: issued.Unix(), ExpiresAt: issued.Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "calango.license")
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	now := issued.Add(9 * 24 * time.Hour)
	mgr, err := license.NewManager(path, priv.Public().(ed25519.PublicKey), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	h := &handler{license: mgr}
	mux := http.NewServeMux()
	h.registerLicense(mux)
	get := func(scope core.DeviceScope) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/license", nil)
		req = req.WithContext(withPrincipal(req.Context(), principal{scope: scope}))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	var body licenseStatus
	rec := get(core.ScopeRead)
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/license = %d %s (%v)", rec.Code, rec.Body.String(), err)
	}
	if body.State != license.StateGrace || body.Plan != "pro" || body.AccountID != "acct-9" ||
		body.GraceEndsAt == nil || !body.GraceEndsAt.Equal(issued.Add(14*24*time.Hour)) {
		t.Fatalf("grace body = %+v", body)
	}

	now = issued.Add(15 * 24 * time.Hour)
	body = licenseStatus{}
	rec = get(core.ScopeRead)
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.State != license.StateExpired || body.Plan != "free" || body.LicensePlan != "pro" || body.ExpiresAt == nil {
		t.Fatalf("expired body = %+v", body)
	}
	if rec := get(core.ScopeApproveOnly); rec.Code != http.StatusForbidden {
		t.Fatalf("GET /v1/license with approve_only = %d, want 403", rec.Code)
	}
}
