package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"calango/internal/core"
)

// approvalResponse is the Approval schema. Args is always a JSON object.
type approvalResponse struct {
	ID      string          `json:"id"`
	RunID   string          `json:"run_id,omitempty"`
	AgentID string          `json:"agent_id"`
	Action  string          `json:"action"`
	Target  string          `json:"target"`
	Args    json.RawMessage `json:"args"`
	Status  string          `json:"status"`
	Outcome string          `json:"outcome,omitempty"`
}

func approvalResponseOf(a core.Approval) approvalResponse {
	args := a.Action.Args
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	return approvalResponse{
		ID:      a.ID,
		RunID:   a.RunID,
		AgentID: a.AgentID,
		Action:  string(a.Action.Kind),
		Target:  a.Action.Target,
		Args:    args,
		Status:  string(a.Status),
		Outcome: string(a.Outcome),
	}
}

func (h *handler) registerApprovals(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/approvals", h.need(core.PermApprove, h.listApprovals))
	mux.HandleFunc("POST /v1/approvals/{id}/decide", h.need(core.PermApprove, h.decideApproval))
}

func (h *handler) listApprovals(w http.ResponseWriter, r *http.Request) {
	var status core.ApprovalStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		parsed, err := parseApprovalStatus(raw)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		status = parsed
	}
	list, err := h.approvals.List(r.Context(), h.ownerID, status)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]approvalResponse, 0, len(list))
	for _, a := range list {
		out = append(out, approvalResponseOf(a))
	}
	writeJSON(w, http.StatusOK, out)
}

// decideApproval records the decision of the device that presents the token. That
// device is the decider, so a run can never approve its own action.
func (h *handler) decideApproval(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Decision string `json:"decision"`
		Remember bool   `json:"remember"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	var approve bool
	switch body.Decision {
	case "approve":
		approve = true
	case "deny":
		approve = false
	default:
		writeServiceError(w, fmt.Errorf("%w: decision must be approve or deny", core.ErrInvalidInput))
		return
	}
	a, err := h.approvals.Decide(r.Context(), h.ownerID, r.PathValue("id"), principalOf(r.Context()).deviceID, approve, body.Remember)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, approvalResponseOf(a))
}

func parseApprovalStatus(s string) (core.ApprovalStatus, error) {
	switch st := core.ApprovalStatus(s); st {
	case core.ApprovalPending, core.ApprovalApproved, core.ApprovalDenied, core.ApprovalExpired:
		return st, nil
	}
	return "", fmt.Errorf("%w: status must be pending, approved, denied or expired", core.ErrInvalidInput)
}
