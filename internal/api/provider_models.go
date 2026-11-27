package api

import (
	"net/http"
	"sort"

	"calango/internal/core"
)

// registerProviderModels exposes the model catalog of one provider account.
// The daemon owns the Codex home and environment; the client never runs the CLI.
func (h *handler) registerProviderModels(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/provider-accounts/{id}/models", h.need(core.PermRead, h.listProviderModels))
}

// providerModelBody is one selectable model. The desktop only offers what the
// provider CLI itself lists.
type providerModelBody struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	DefaultReasoning string `json:"default_reasoning,omitempty"`
}

func (h *handler) listProviderModels(w http.ResponseWriter, r *http.Request) {
	views, err := h.accounts.CodexModels(r.Context(), h.ownerID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	// Anthropic accounts have a fixed model string; an empty list keeps the
	// desktop free form instead of inventing a catalog.
	out := make([]providerModelBody, 0, len(views))
	for _, m := range views {
		out = append(out, providerModelBody{ID: m.Slug, Label: m.DisplayName, DefaultReasoning: m.DefaultReasoning})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	writeJSON(w, http.StatusOK, out)
}
