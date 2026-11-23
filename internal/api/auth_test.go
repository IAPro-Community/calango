package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"calango/internal/auth"
	"calango/internal/core"
	"calango/internal/store"
	"calango/internal/vault"
)

func TestRotateTokenRetiresTheOldToken(t *testing.T) {
	h := newTestHandler(t)

	rec := call(t, h, http.MethodPost, "/v1/auth/rotate-token", testToken, "")
	expectStatus(t, rec, http.StatusOK)
	var rotated struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("decode rotated token: %v", err)
	}
	if rotated.Token == "" || rotated.Token == testToken {
		t.Fatalf("rotation returned %q, want a new non-empty token", rotated.Token)
	}

	expectStatus(t, call(t, h, http.MethodGet, "/v1/agents", testToken, ""), http.StatusUnauthorized)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/agents", rotated.Token, ""), http.StatusOK)
}

func TestRotateTokenNeedsFullScope(t *testing.T) {
	h := newTestHandler(t)
	for _, scope := range []string{"read", "approve_only"} {
		paired := pairDevice(t, h, issuePairingCode(t, h, scope))
		expectStatus(t, call(t, h, http.MethodPost, "/v1/auth/rotate-token", paired.Token, ""), http.StatusForbidden)
	}
	expectStatus(t, call(t, h, http.MethodGet, "/v1/agents", testToken, ""), http.StatusOK)
}

// newTestLocalToken wraps testToken over a throwaway vault for tests that build a handler by hand.
func newTestLocalToken(t *testing.T) *auth.LocalToken {
	t.Helper()
	database, err := store.OpenMemory(context.Background())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	keys, err := vault.NewMasterKeys(map[int][]byte{1: bytes.Repeat([]byte{7}, 32)})
	if err != nil {
		t.Fatalf("master keys: %v", err)
	}
	secrets, err := vault.NewService(database, keys, testOwner, database, time.Now, core.NewID)
	if err != nil {
		t.Fatalf("vault service: %v", err)
	}
	token, err := auth.NewLocalToken(testToken, secrets)
	if err != nil {
		t.Fatalf("local token: %v", err)
	}
	return token
}
