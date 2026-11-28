package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestProviderAccountsCreateListAndDelete(t *testing.T) {
	h, _ := newTestEnv(t)

	created := call(t, h, http.MethodPost, "/v1/provider-accounts", testToken, `{"kind":"anthropic","label":"work"}`)
	expectStatus(t, created, http.StatusCreated)
	var acc struct {
		ID         string `json:"id"`
		Key        string `json:"key"`
		Setup      string `json:"setup"`
		SecretItem string `json:"secret_item"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &acc); err != nil {
		t.Fatalf("decode account: %v", err)
	}
	if acc.Key != "anthropic/work" || acc.Setup != "PUT /v1/vault/anthropic-work-api-key with the API key" || acc.SecretItem != "anthropic-work-api-key" {
		t.Fatalf("account = %+v", acc)
	}

	expectStatus(t, call(t, h, http.MethodPost, "/v1/provider-accounts", testToken, `{"kind":"anthropic","label":"work"}`), http.StatusConflict)

	listed := call(t, h, http.MethodGet, "/v1/provider-accounts", testToken, "")
	expectStatus(t, listed, http.StatusOK)
	var all []struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &all); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	keys := map[string]bool{}
	for _, a := range all {
		keys[a.Key] = true
	}
	if !keys["anthropic/default"] || !keys["anthropic/work"] {
		t.Fatalf("listed keys = %v, want anthropic/default and anthropic/work", keys)
	}

	agent := call(t, h, http.MethodPost, "/v1/agents", testToken,
		fmt.Sprintf(`{"name":"Calango","token_limit_month":100000,"providers":[{"account_id":%q,"model":"claude-test"}]}`, acc.ID))
	expectStatus(t, agent, http.StatusCreated)
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/provider-accounts/"+acc.ID, testToken, ""), http.StatusConflict)

	spare := call(t, h, http.MethodPost, "/v1/provider-accounts", testToken, `{"kind":"anthropic","label":"spare"}`)
	expectStatus(t, spare, http.StatusCreated)
	var spareAcc struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(spare.Body.Bytes(), &spareAcc); err != nil {
		t.Fatalf("decode spare account: %v", err)
	}
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/provider-accounts/"+spareAcc.ID, testToken, ""), http.StatusNoContent)
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/provider-accounts/"+spareAcc.ID, testToken, ""), http.StatusNotFound)
}

func TestProviderCapacityUnsupportedAndOwnerScoped(t *testing.T) {
	h, _ := newTestEnv(t)
	created := call(t, h, http.MethodPost, "/v1/provider-accounts", testToken, `{"kind":"anthropic","label":"quota-test"}`)
	expectStatus(t, created, http.StatusCreated)
	var acc providerAccountResponse
	if err := json.Unmarshal(created.Body.Bytes(), &acc); err != nil {
		t.Fatal(err)
	}
	if acc.Capacity.Status != "UNSUPPORTED" || len(acc.Capacity.Windows) != 0 {
		t.Fatalf("unexpected capacity: %+v", acc.Capacity)
	}
	refreshed := call(t, h, http.MethodPost, "/v1/provider-accounts/"+acc.ID+"/capacity/refresh", testToken, "")
	expectStatus(t, refreshed, http.StatusOK)
	if err := json.Unmarshal(refreshed.Body.Bytes(), &acc); err != nil {
		t.Fatal(err)
	}
	if acc.Capacity.Status != "UNSUPPORTED" || acc.Capacity.AccountID != acc.ID {
		t.Fatalf("unexpected refresh: %+v", acc.Capacity)
	}
	expectStatus(t, call(t, h, http.MethodPost, "/v1/provider-accounts/missing/capacity/refresh", testToken, ""), http.StatusNotFound)
	expectStatus(t, call(t, h, http.MethodPost, "/v1/provider-accounts/"+acc.ID+"/capacity/refresh", "", ""), http.StatusUnauthorized)
}

func TestProviderKindsOnlyAdvertisesEnabledAdapters(t *testing.T) {
	h, _ := newTestEnv(t)
	resp := call(t, h, http.MethodGet, "/v1/provider-kinds", testToken, "")
	expectStatus(t, resp, http.StatusOK)
	var kinds []struct {
		Kind        string `json:"kind"`
		DisplayName string `json:"display_name"`
		AuthKind    string `json:"auth_kind"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &kinds); err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, k := range kinds {
		if k.DisplayName == "" {
			t.Fatalf("missing provider display name: %+v", k)
		}
		seen[k.Kind] = k.AuthKind
	}
	if len(seen) != 2 || seen["codex"] != "oauth_cli" || seen["anthropic"] != "api_key" {
		t.Fatalf("unexpected enabled catalog: %+v", kinds)
	}
	expectStatus(t, call(t, h, http.MethodGet, "/v1/provider-kinds", "", ""), http.StatusUnauthorized)
}
