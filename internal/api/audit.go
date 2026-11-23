package api

import (
	"context"
	"net/http"

	"calango/internal/audit"
	"calango/internal/core"
)

// AuditVerifier recomputes the audit chain. *store.Store implements it.
type AuditVerifier interface {
	VerifyAudit(ctx context.Context) (audit.Verification, error)
}

// auditVerification is the AuditVerification schema.
type auditVerification struct {
	OK             bool        `json:"ok"`
	EntriesChecked int         `json:"entries_checked"`
	FirstBreak     *auditBreak `json:"first_break"`
}

// auditBreak is the first_break object of AuditVerification.
type auditBreak struct {
	Seq    int64  `json:"seq"`
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

func (h *handler) registerAudit(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/audit/verify", h.need(core.PermRead, h.verifyAudit))
}

func (h *handler) verifyAudit(w http.ResponseWriter, r *http.Request) {
	v, err := h.audit.VerifyAudit(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := auditVerification{OK: v.FirstBreak == nil, EntriesChecked: v.EntriesChecked}
	if b := v.FirstBreak; b != nil {
		out.FirstBreak = &auditBreak{Seq: b.Seq, ID: b.ID, Reason: b.Reason}
	}
	writeJSON(w, http.StatusOK, out)
}
