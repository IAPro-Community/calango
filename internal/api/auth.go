package api

import (
	"net/http"

	"calango/internal/core"
)

// registerAuth mounts the routes that manage the local bearer token.
func (h *handler) registerAuth(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/rotate-token", h.need(core.PermWrite, h.rotateToken))
}

// rotatedTokenResponse is the only response that carries a token. The old token stops
// matching as soon as this response is built, so the caller must store the new one now.
type rotatedTokenResponse struct {
	Token string `json:"token"`
}

func (h *handler) rotateToken(w http.ResponseWriter, r *http.Request) {
	token, err := h.localToken.Rotate(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rotatedTokenResponse{Token: token})
}
