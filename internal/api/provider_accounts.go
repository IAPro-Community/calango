package api

import (
	"net/http"
	"time"

	"calango/internal/core"
	"calango/internal/providers"
)

// providerAccountBody is the POST /v1/provider-accounts request.
type providerAccountBody struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

// providerAccountResponse is the ProviderAccount schema. It never carries a secret;
// setup names the step that finishes the sign-in.
type providerAccountResponse struct {
	ID         string                 `json:"id"`
	Kind       string                 `json:"kind"`
	Label      string                 `json:"label"`
	Key        string                 `json:"key"`
	Setup      string                 `json:"setup"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Capacity   providers.CapacityView `json:"capacity"`
	SecretItem string                 `json:"secret_item,omitempty"`
}

func providerAccountResponseOf(v providers.View) providerAccountResponse {
	return providerAccountResponse{
		ID:         v.ID,
		Kind:       v.Kind,
		Label:      v.Label,
		Key:        v.Key,
		Setup:      v.Setup,
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
		Capacity:   v.Capacity,
		SecretItem: v.SecretItem,
	}
}

func (h *handler) registerProviderAccounts(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/provider-kinds", h.need(core.PermRead, h.listProviderKinds))
	mux.HandleFunc("GET /v1/provider-accounts", h.need(core.PermRead, h.listProviderAccounts))
	mux.HandleFunc("POST /v1/provider-accounts", h.need(core.PermWrite, h.createProviderAccount))
	mux.HandleFunc("DELETE /v1/provider-accounts/{id}", h.need(core.PermWrite, h.deleteProviderAccount))
	mux.HandleFunc("POST /v1/provider-accounts/{id}/capacity/refresh", h.need(core.PermWrite, h.refreshProviderCapacity))
}

func (h *handler) listProviderKinds(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.accounts.ListProviderKinds())
}

func (h *handler) listProviderAccounts(w http.ResponseWriter, r *http.Request) {
	views, err := h.accounts.List(r.Context(), h.ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]providerAccountResponse, 0, len(views))
	for _, v := range views {
		out = append(out, providerAccountResponseOf(v))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) createProviderAccount(w http.ResponseWriter, r *http.Request) {
	var body providerAccountBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	v, err := h.accounts.Create(r.Context(), h.ownerID, body.Kind, body.Label)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, providerAccountResponseOf(v))
}

func (h *handler) deleteProviderAccount(w http.ResponseWriter, r *http.Request) {
	if err := h.accounts.Delete(r.Context(), h.ownerID, r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) refreshProviderCapacity(w http.ResponseWriter, r *http.Request) {
	v, err := h.accounts.RefreshCapacity(r.Context(), h.ownerID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, providerAccountResponseOf(v))
}
