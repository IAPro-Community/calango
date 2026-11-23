package api

import (
	"fmt"
	"net/http"

	"calango/internal/core"
)

// memoryBody is the MemoryInput schema. scope_id names the agent for scope agent.
type memoryBody struct {
	Scope   string `json:"scope"`
	ScopeID string `json:"scope_id"`
	Kind    string `json:"kind"`
	Text    string `json:"text"`
}

type memoryResponse struct {
	ID      string `json:"id"`
	Scope   string `json:"scope"`
	ScopeID string `json:"scope_id,omitempty"`
	Kind    string `json:"kind"`
	Text    string `json:"text"`
}

func memoryResponseOf(m core.Memory) memoryResponse {
	resp := memoryResponse{ID: m.ID, Scope: string(m.Scope), Kind: string(m.Kind), Text: m.Text}
	if m.Scope == core.ScopeAgent {
		resp.ScopeID = m.AgentID
	}
	return resp
}

// parseScope accepts the V1 scopes. "project" is not in V1 until projects exist.
func parseScope(s string) (core.MemoryScope, error) {
	switch core.MemoryScope(s) {
	case core.ScopeAccount, core.ScopeAgent:
		return core.MemoryScope(s), nil
	}
	return "", fmt.Errorf("%w: scope must be account or agent", core.ErrInvalidInput)
}

func (h *handler) registerMemories(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/memories", h.need(core.PermRead, h.listMemories))
	mux.HandleFunc("POST /v1/memories", h.need(core.PermWrite, h.createMemory))
	mux.HandleFunc("PATCH /v1/memories/{id}", h.need(core.PermWrite, h.updateMemory))
	mux.HandleFunc("DELETE /v1/memories/{id}", h.need(core.PermWrite, h.deleteMemory))
}

func (h *handler) listMemories(w http.ResponseWriter, r *http.Request) {
	var scope core.MemoryScope
	if raw := r.URL.Query().Get("scope"); raw != "" {
		parsed, err := parseScope(raw)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		scope = parsed
	}
	list, err := h.memories.List(r.Context(), h.ownerID, scope, r.URL.Query().Get("scope_id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]memoryResponse, 0, len(list))
	for _, m := range list {
		out = append(out, memoryResponseOf(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) createMemory(w http.ResponseWriter, r *http.Request) {
	var body memoryBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	scope, err := parseScope(body.Scope)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if scope == core.ScopeAccount && body.ScopeID != "" {
		writeServiceError(w, fmt.Errorf("%w: scope_id is only used with scope agent", core.ErrInvalidInput))
		return
	}
	m, err := h.memories.Create(r.Context(), h.ownerID, core.Memory{
		Scope:   scope,
		AgentID: body.ScopeID,
		Kind:    core.MemoryKind(body.Kind),
		Text:    body.Text,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, memoryResponseOf(m))
}

func (h *handler) updateMemory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	m, err := h.memories.Update(r.Context(), h.ownerID, r.PathValue("id"), body.Text)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, memoryResponseOf(m))
}

func (h *handler) deleteMemory(w http.ResponseWriter, r *http.Request) {
	if err := h.memories.Delete(r.Context(), h.ownerID, r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
