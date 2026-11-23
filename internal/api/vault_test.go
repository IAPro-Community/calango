package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// canary is a value that must never appear in any response or audit record.
const canary = "canary-secret-7f3a9c"

type vaultItemBody struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func TestVaultRoutesNeverReturnTheValue(t *testing.T) {
	h, _ := newTestEnv(t)

	rec := call(t, h, http.MethodPut, "/v1/vault/github", testToken, `{"value":"`+canary+`"}`)
	expectStatus(t, rec, http.StatusOK)
	if strings.Contains(rec.Body.String(), canary) {
		t.Fatalf("PUT response carries the secret value: %s", rec.Body.String())
	}
	var item vaultItemBody
	if err := json.Unmarshal(rec.Body.Bytes(), &item); err != nil {
		t.Fatalf("decode PUT response: %v", err)
	}
	if item.Name != "github" || item.CreatedAt == "" || item.UpdatedAt == "" {
		t.Fatalf("PUT response = %+v, want name and dates", item)
	}

	rec = call(t, h, http.MethodGet, "/v1/vault", testToken, "")
	expectStatus(t, rec, http.StatusOK)
	if strings.Contains(rec.Body.String(), canary) {
		t.Fatalf("GET response carries the secret value: %s", rec.Body.String())
	}
	var list []vaultItemBody
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if len(list) != 1 || list[0].Name != "github" {
		t.Fatalf("GET list = %+v, want one item named github", list)
	}
}

func TestVaultSetReplacesTheSameName(t *testing.T) {
	h, _ := newTestEnv(t)
	expectStatus(t, call(t, h, http.MethodPut, "/v1/vault/github", testToken, `{"value":"first"}`), http.StatusOK)
	expectStatus(t, call(t, h, http.MethodPut, "/v1/vault/github", testToken, `{"value":"second"}`), http.StatusOK)

	rec := call(t, h, http.MethodGet, "/v1/vault", testToken, "")
	var list []vaultItemBody
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("GET list has %d items after replace, want 1", len(list))
	}
}

func TestVaultRejectsBadNamesAndEmptyValues(t *testing.T) {
	h, _ := newTestEnv(t)
	cases := []struct{ path, body string }{
		{"/v1/vault/BAD", `{"value":"x"}`},
		{"/v1/vault/github", `{"value":""}`},
		{"/v1/vault/github", `{}`},
	}
	for _, c := range cases {
		rec := call(t, h, http.MethodPut, c.path, testToken, c.body)
		expectStatus(t, rec, http.StatusBadRequest)
		if got := errorCode(t, rec); got != codeInvalidInput {
			t.Fatalf("PUT %s %s: code = %q, want %q", c.path, c.body, got, codeInvalidInput)
		}
	}
}

func TestVaultDeleteRemovesTheNameAndUnknownIsNotFound(t *testing.T) {
	h, _ := newTestEnv(t)
	expectStatus(t, call(t, h, http.MethodPut, "/v1/vault/github", testToken, `{"value":"x"}`), http.StatusOK)
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/vault/github", testToken, ""), http.StatusNoContent)

	rec := call(t, h, http.MethodGet, "/v1/vault", testToken, "")
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Fatalf("GET after delete = %s, want []", got)
	}

	rec = call(t, h, http.MethodDelete, "/v1/vault/github", testToken, "")
	expectStatus(t, rec, http.StatusNotFound)
	if got := errorCode(t, rec); got != codeNotFound {
		t.Fatalf("DELETE unknown: code = %q, want %q", got, codeNotFound)
	}
}

func TestVaultRoutesAreFullScopeOnly(t *testing.T) {
	h, _ := newTestEnv(t)
	for _, scope := range []string{"read", "approve_only"} {
		device := pairDevice(t, h, issuePairingCode(t, h, scope))
		expectStatus(t, call(t, h, http.MethodGet, "/v1/vault", device.Token, ""), http.StatusForbidden)
		expectStatus(t, call(t, h, http.MethodPut, "/v1/vault/github", device.Token, `{"value":"x"}`), http.StatusForbidden)
		expectStatus(t, call(t, h, http.MethodDelete, "/v1/vault/github", device.Token, ""), http.StatusForbidden)
	}
}

// The API token lives in the vault under a reserved name. Overwriting or deleting it
// through these routes would change or lock out the token without a rotation.
func TestVaultReservesTheAPITokenName(t *testing.T) {
	h, _ := newTestEnv(t)
	rec := call(t, h, http.MethodPut, "/v1/vault/api-token", testToken, `{"value":"x"}`)
	expectStatus(t, rec, http.StatusConflict)
	if got := errorCode(t, rec); got != codeConflict {
		t.Fatalf("PUT api-token: code = %q, want %q", got, codeConflict)
	}
	rec = call(t, h, http.MethodDelete, "/v1/vault/api-token", testToken, "")
	expectStatus(t, rec, http.StatusConflict)
}

// Every set and delete is on the audit chain with the name only, never the value.
func TestVaultWritesAreAuditedWithoutTheValue(t *testing.T) {
	h, database := newTestEnv(t)
	expectStatus(t, call(t, h, http.MethodPut, "/v1/vault/github", testToken, `{"value":"`+canary+`"}`), http.StatusOK)
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/vault/github", testToken, ""), http.StatusNoContent)

	ctx := context.Background()
	sets, err := database.ListAudit(ctx, "vault.set")
	if err != nil {
		t.Fatalf("list vault.set: %v", err)
	}
	deletes, err := database.ListAudit(ctx, "vault.delete")
	if err != nil {
		t.Fatalf("list vault.delete: %v", err)
	}
	if len(sets) != 1 || len(deletes) != 1 {
		t.Fatalf("audit rows: set=%d delete=%d, want 1 each", len(sets), len(deletes))
	}
	if sets[0].Subject != "vault:github" {
		t.Fatalf("set subject = %q, want vault:github", sets[0].Subject)
	}
	raw, err := json.Marshal(append(sets, deletes...))
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}
	if strings.Contains(string(raw), canary) {
		t.Fatalf("audit rows carry the secret value: %s", raw)
	}
}
