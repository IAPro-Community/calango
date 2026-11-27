package api

import (
	"net/http"

	"calango/internal/agents"
	"calango/internal/core"
)

// agentBody is the request shape shared by create and replace (AgentInput in the contract).
type agentBody struct {
	Name            string         `json:"name"`
	Role            string         `json:"role"`
	Instructions    string         `json:"instructions"`
	ReasoningEffort string         `json:"reasoning_effort"`
	RunOn           string         `json:"run_on"`
	TokenLimitMonth int64          `json:"token_limit_month"`
	ApprovalLevel   string         `json:"approval_level"`
	MaestroEnabled  bool           `json:"maestro_enabled"`
	CapacityRouting bool           `json:"capacity_routing"`
	Providers       []providerBody `json:"providers"`
}

// providerBody is one step of an agent's provider chain. Account is set on
// responses only; the server derives it from account_id.
type providerBody struct {
	AccountID string `json:"account_id"`
	Model     string `json:"model"`
	Account   string `json:"account,omitempty"`
}

// agentResponse is the Agent schema: the input fields plus live state.
type agentResponse struct {
	ID string `json:"id"`
	agentBody
	Paused          bool  `json:"paused"`
	TokensUsedMonth int64 `json:"tokens_used_month"`
}

func (b agentBody) toCore() core.Agent {
	return core.Agent{
		Name:            b.Name,
		Role:            b.Role,
		Instructions:    b.Instructions,
		Providers:       providerRefsOf(b.Providers),
		ReasoningEffort: core.ReasoningEffort(b.ReasoningEffort),
		RunOn:           core.RunOn(b.RunOn),
		TokenLimitMonth: b.TokenLimitMonth,
		ApprovalLevel:   core.ApprovalLevel(b.ApprovalLevel),
		MaestroEnabled:  b.MaestroEnabled,
		CapacityRouting: b.CapacityRouting,
	}
}

func agentResponseOf(v agents.View) agentResponse {
	return agentResponse{
		ID: v.ID,
		agentBody: agentBody{
			Name:            v.Name,
			Role:            v.Role,
			Instructions:    v.Instructions,
			Providers:       providerBodiesOf(v.Providers),
			ReasoningEffort: string(v.ReasoningEffort),
			RunOn:           string(v.RunOn),
			TokenLimitMonth: v.TokenLimitMonth,
			ApprovalLevel:   string(v.ApprovalLevel),
			MaestroEnabled:  v.MaestroEnabled,
			CapacityRouting: v.CapacityRouting,
		},
		Paused:          v.Paused,
		TokensUsedMonth: v.TokensUsedMonth,
	}
}

// providerRefsOf converts the request chain. Kind and label are not part of the
// input: the store reads them from the account when the agent is saved.
func providerRefsOf(in []providerBody) []core.ProviderRef {
	refs := make([]core.ProviderRef, 0, len(in))
	for _, p := range in {
		refs = append(refs, core.ProviderRef{AccountID: p.AccountID, Model: p.Model})
	}
	return refs
}

func providerBodiesOf(in []core.ProviderRef) []providerBody {
	out := make([]providerBody, 0, len(in))
	for _, r := range in {
		out = append(out, providerBody{AccountID: r.AccountID, Model: r.Model, Account: r.Key()})
	}
	return out
}

func (h *handler) registerAgents(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/agents", h.need(core.PermRead, h.listAgents))
	mux.HandleFunc("POST /v1/agents", h.need(core.PermWrite, h.createAgent))
	mux.HandleFunc("GET /v1/agents/{id}", h.need(core.PermRead, h.getAgent))
	mux.HandleFunc("PUT /v1/agents/{id}", h.need(core.PermWrite, h.replaceAgent))
	mux.HandleFunc("DELETE /v1/agents/{id}", h.need(core.PermWrite, h.deleteAgent))
	mux.HandleFunc("POST /v1/agents/pause-all", h.need(core.PermWrite, h.pauseAllAgents))
	mux.HandleFunc("POST /v1/agents/resume-all", h.need(core.PermWrite, h.resumeAllAgents))
}

func (h *handler) listAgents(w http.ResponseWriter, r *http.Request) {
	views, err := h.agents.List(r.Context(), h.ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]agentResponse, 0, len(views))
	for _, v := range views {
		out = append(out, agentResponseOf(v))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) createAgent(w http.ResponseWriter, r *http.Request) {
	var body agentBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	v, err := h.agents.Create(r.Context(), h.ownerID, body.toCore())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, agentResponseOf(v))
}

func (h *handler) getAgent(w http.ResponseWriter, r *http.Request) {
	v, err := h.agents.Get(r.Context(), h.ownerID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agentResponseOf(v))
}

func (h *handler) replaceAgent(w http.ResponseWriter, r *http.Request) {
	var body agentBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	v, err := h.agents.Replace(r.Context(), h.ownerID, r.PathValue("id"), body.toCore())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agentResponseOf(v))
}

func (h *handler) deleteAgent(w http.ResponseWriter, r *http.Request) {
	if err := h.agents.Delete(r.Context(), h.ownerID, r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) pauseAllAgents(w http.ResponseWriter, r *http.Request) {
	if err := h.agents.PauseAll(r.Context()); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) resumeAllAgents(w http.ResponseWriter, r *http.Request) {
	if err := h.agents.ResumeAll(r.Context()); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
