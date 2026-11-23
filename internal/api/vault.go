package api

import (
	"context"
	"fmt"
	"net/http"

	"calango/internal/auth"
	"calango/internal/core"
	"calango/internal/vault"
)

// VaultStore is the part of the vault the API may call. Reveal is absent on purpose:
// no route returns a secret value.
type VaultStore interface {
	List(ctx context.Context) ([]vault.Item, error)
	Set(ctx context.Context, name, value string) (vault.Item, error)
	Delete(ctx context.Context, name string) error
}

// vaultItemResponse is the VaultItem schema. It carries no value.
type vaultItemResponse struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func vaultItemOf(it vault.Item) vaultItemResponse {
	return vaultItemResponse{
		Name:      it.Name,
		CreatedAt: formatTime(it.CreatedAt),
		UpdatedAt: formatTime(it.UpdatedAt),
	}
}

// vaultSetRequest is the body of PUT /v1/vault/{name}.
type vaultSetRequest struct {
	Value string `json:"value"`
}

func (h *handler) registerVault(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/vault", h.need(core.PermWrite, h.listVault))
	mux.HandleFunc("PUT /v1/vault/{name}", h.need(core.PermWrite, h.setVault))
	mux.HandleFunc("DELETE /v1/vault/{name}", h.need(core.PermWrite, h.deleteVault))
}

// reservedVaultName refuses the item that holds the local API token. Replacing or
// deleting it here would change or lock out the token without a rotation; the only
// route that changes it is POST /v1/auth/rotate-token.
func reservedVaultName(name string) error {
	if name == auth.VaultTokenName {
		return fmt.Errorf("%w: %q is reserved for the local API token", core.ErrConflict, name)
	}
	return nil
}

func (h *handler) listVault(w http.ResponseWriter, r *http.Request) {
	items, err := h.vault.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]vaultItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, vaultItemOf(it))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) setVault(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := reservedVaultName(name); err != nil {
		writeServiceError(w, err)
		return
	}
	var body vaultSetRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	it, err := h.vault.Set(r.Context(), name, body.Value)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vaultItemOf(it))
}

func (h *handler) deleteVault(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := reservedVaultName(name); err != nil {
		writeServiceError(w, err)
		return
	}
	if err := h.vault.Delete(r.Context(), name); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
