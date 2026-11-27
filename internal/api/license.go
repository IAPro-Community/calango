package api

import (
	"net/http"
	"time"

	"calango/internal/core"
	"calango/internal/license"
)

// LicenseReader reports the node's evaluated license. It never fails: problems lower
// the plan to Free.
type LicenseReader interface {
	Status() license.Status
}

// licenseStatus is the LicenseStatus schema.
type licenseStatus struct {
	State       license.State `json:"state"`
	Plan        string        `json:"plan"`
	Reason      string        `json:"reason,omitempty"`
	AccountID   string        `json:"account_id,omitempty"`
	LicensePlan string        `json:"license_plan,omitempty"`
	Features    []string      `json:"features,omitempty"`
	NodesMax    *int          `json:"nodes_max,omitempty"`
	IssuedAt    *time.Time    `json:"issued_at,omitempty"`
	ExpiresAt   *time.Time    `json:"expires_at,omitempty"`
	GraceEndsAt *time.Time    `json:"grace_ends_at,omitempty"`
	CheckedAt   time.Time     `json:"checked_at"`
}

func (h *handler) registerLicense(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/license", h.need(core.PermRead, h.getLicense))
}

func (h *handler) getLicense(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, licenseView(h.license.Status()))
}

func licenseView(st license.Status) licenseStatus {
	out := licenseStatus{State: st.State, Plan: st.Plan, Reason: st.Reason, CheckedAt: st.CheckedAt}
	if c := st.Claims; c != nil {
		expires, graceEnds := c.Expires(), c.GraceEnds()
		nodes := c.NodesMax
		out.AccountID, out.LicensePlan, out.Features = c.AccountID, c.Plan, c.Features
		out.NodesMax, out.ExpiresAt, out.GraceEndsAt = &nodes, &expires, &graceEnds
		if c.IssuedAt > 0 {
			issued := time.Unix(c.IssuedAt, 0).UTC()
			out.IssuedAt = &issued
		}
	}
	return out
}
