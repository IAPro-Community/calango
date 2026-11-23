package api

import (
	"fmt"
	"net/http"
	"time"

	"calango/internal/core"
	"calango/internal/usage"
)

// usageRow is the UsageRow schema.
type usageRow struct {
	Key                 string `json:"key"`
	TokensIn            int64  `json:"tokens_in"`
	TokensOut           int64  `json:"tokens_out"`
	CostMicros          int64  `json:"cost_micros"`
	CostKnownMicros     int64  `json:"cost_known_micros"`
	CostEstimatedMicros int64  `json:"cost_estimated_micros"`
	UnknownCostRows     int64  `json:"cost_unknown_rows"`
}

func (h *handler) registerUsage(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/usage", h.need(core.PermRead, h.getUsage))
}

func (h *handler) getUsage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	from, err := parseOptionalTime("from", query.Get("from"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	to, err := parseOptionalTime("to", query.Get("to"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	rows, err := h.usage.Summary(r.Context(), h.ownerID, usage.Query{
		From:    from,
		To:      to,
		GroupBy: usage.Group(query.Get("group_by")),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]usageRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, usageRow{
			Key:                 row.Key,
			TokensIn:            row.TokensIn,
			TokensOut:           row.TokensOut,
			CostMicros:          row.CostMicros,
			CostKnownMicros:     row.CostKnownMicros,
			CostEstimatedMicros: row.CostEstimatedMicros,
			UnknownCostRows:     row.UnknownCostRows,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// parseOptionalTime returns the zero time when raw is empty.
func parseOptionalTime(name, raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s must be an RFC 3339 time", core.ErrInvalidInput, name)
	}
	return t, nil
}
