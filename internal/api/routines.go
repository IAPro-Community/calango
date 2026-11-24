package api

import (
	"net/http"
	"time"

	"calango/internal/core"
	"calango/internal/routines"
)

func (h *handler) registerRoutines(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/routines", h.need(core.PermRead, h.listRoutines))
	mux.HandleFunc("POST /v1/routines", h.need(core.PermWrite, h.createRoutine))
	mux.HandleFunc("GET /v1/routines/{id}", h.need(core.PermRead, h.getRoutine))
	mux.HandleFunc("PATCH /v1/routines/{id}", h.need(core.PermWrite, h.patchRoutine))
	mux.HandleFunc("DELETE /v1/routines/{id}", h.need(core.PermWrite, h.deleteRoutine))
	mux.HandleFunc("POST /v1/routines/{id}/run", h.need(core.PermWrite, h.runRoutineNow))
	mux.HandleFunc("GET /v1/routines/{id}/runs", h.need(core.PermRead, h.listRoutineRuns))
}

type routineBody struct {
	AgentID     string           `json:"agent_id"`
	Name        string           `json:"name"`
	Prompt      string           `json:"prompt"`
	TriggerKind core.TriggerKind `json:"trigger_kind"`
	Schedule    string           `json:"schedule"`
	RunOn       core.RunOn       `json:"run_on"`
	QuietStart  *int             `json:"quiet_start"`
	QuietEnd    *int             `json:"quiet_end"`
	MaxRetries  int              `json:"max_retries"`
	NotifyOn    core.NotifyOn    `json:"notify_on"`
}

func (b routineBody) toCore() core.Routine {
	return core.Routine{
		AgentID:     b.AgentID,
		Name:        b.Name,
		Prompt:      b.Prompt,
		TriggerKind: b.TriggerKind,
		Schedule:    b.Schedule,
		RunOn:       b.RunOn,
		QuietStart:  b.QuietStart,
		QuietEnd:    b.QuietEnd,
		MaxRetries:  b.MaxRetries,
		NotifyOn:    b.NotifyOn,
	}
}

// routinePatchBody is a partial edit. Absent fields keep their stored value. clear_quiet
// removes the quiet window and wins over quiet_start and quiet_end.
type routinePatchBody struct {
	Name        *string           `json:"name"`
	Prompt      *string           `json:"prompt"`
	TriggerKind *core.TriggerKind `json:"trigger_kind"`
	Schedule    *string           `json:"schedule"`
	RunOn       *core.RunOn       `json:"run_on"`
	QuietStart  *int              `json:"quiet_start"`
	QuietEnd    *int              `json:"quiet_end"`
	ClearQuiet  bool              `json:"clear_quiet"`
	MaxRetries  *int              `json:"max_retries"`
	NotifyOn    *core.NotifyOn    `json:"notify_on"`
	Paused      *bool             `json:"paused"`
}

func (b routinePatchBody) toPatch() routines.Patch {
	return routines.Patch{
		Name:        b.Name,
		Prompt:      b.Prompt,
		TriggerKind: b.TriggerKind,
		Schedule:    b.Schedule,
		RunOn:       b.RunOn,
		QuietStart:  b.QuietStart,
		QuietEnd:    b.QuietEnd,
		ClearQuiet:  b.ClearQuiet,
		MaxRetries:  b.MaxRetries,
		NotifyOn:    b.NotifyOn,
		Paused:      b.Paused,
	}
}

type routineResponse struct {
	ID             string           `json:"id"`
	AgentID        string           `json:"agent_id"`
	Name           string           `json:"name"`
	Prompt         string           `json:"prompt"`
	TriggerKind    core.TriggerKind `json:"trigger_kind"`
	Schedule       string           `json:"schedule"`
	RunOn          core.RunOn       `json:"run_on"`
	QuietStart     *int             `json:"quiet_start"`
	QuietEnd       *int             `json:"quiet_end"`
	MaxRetries     int              `json:"max_retries"`
	NotifyOn       core.NotifyOn    `json:"notify_on"`
	Paused         bool             `json:"paused"`
	ConversationID string           `json:"conversation_id,omitempty"`
	EndedAt        *time.Time       `json:"ended_at"`
	NextRunAt      time.Time        `json:"next_run_at"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

func routineResponseOf(r core.Routine) routineResponse {
	return routineResponse{
		ID:             r.ID,
		AgentID:        r.AgentID,
		Name:           r.Name,
		Prompt:         r.Prompt,
		TriggerKind:    r.TriggerKind,
		Schedule:       r.Schedule,
		RunOn:          r.RunOn,
		QuietStart:     r.QuietStart,
		QuietEnd:       r.QuietEnd,
		MaxRetries:     r.MaxRetries,
		NotifyOn:       r.NotifyOn,
		Paused:         r.Paused,
		ConversationID: r.ConversationID,
		EndedAt:        r.EndedAt,
		NextRunAt:      r.NextRunAt,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

type occurrenceResponse struct {
	ID            string                `json:"id"`
	OccurrenceID  string                `json:"occurrence_id"`
	ScheduledFor  time.Time             `json:"scheduled_for"`
	Status        core.OccurrenceStatus `json:"status"`
	Attempt       int                   `json:"attempt"`
	RunID         string                `json:"run_id,omitempty"`
	HadNews       *bool                 `json:"had_news"`
	LastError     string                `json:"last_error,omitempty"`
	NextAttemptAt time.Time             `json:"next_attempt_at"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

func occurrenceResponseOf(o core.Occurrence) occurrenceResponse {
	return occurrenceResponse{
		ID:            o.ID,
		OccurrenceID:  o.OccurrenceID,
		ScheduledFor:  o.ScheduledFor,
		Status:        o.Status,
		Attempt:       o.Attempt,
		RunID:         o.RunID,
		HadNews:       o.HadNews,
		LastError:     o.LastError,
		NextAttemptAt: o.NextAttemptAt,
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
	}
}

func (h *handler) listRoutines(w http.ResponseWriter, r *http.Request) {
	items, err := h.routines.List(r.Context(), h.ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]routineResponse, 0, len(items))
	for _, it := range items {
		out = append(out, routineResponseOf(it))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) createRoutine(w http.ResponseWriter, r *http.Request) {
	var body routineBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	v, err := h.routines.Create(r.Context(), h.ownerID, body.toCore())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, routineResponseOf(v))
}

func (h *handler) getRoutine(w http.ResponseWriter, r *http.Request) {
	v, err := h.routines.Get(r.Context(), h.ownerID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, routineResponseOf(v))
}

func (h *handler) patchRoutine(w http.ResponseWriter, r *http.Request) {
	var body routinePatchBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	v, err := h.routines.Update(r.Context(), h.ownerID, r.PathValue("id"), body.toPatch())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, routineResponseOf(v))
}

func (h *handler) deleteRoutine(w http.ResponseWriter, r *http.Request) {
	if err := h.routines.Delete(r.Context(), h.ownerID, r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// runRoutineNow queues one extra occurrence. It returns 202: the run happens at the next tick.
func (h *handler) runRoutineNow(w http.ResponseWriter, r *http.Request) {
	o, err := h.routines.RunNow(r.Context(), h.ownerID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, occurrenceResponseOf(o))
}

func (h *handler) listRoutineRuns(w http.ResponseWriter, r *http.Request) {
	items, err := h.routines.Occurrences(r.Context(), h.ownerID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]occurrenceResponse, 0, len(items))
	for _, it := range items {
		out = append(out, occurrenceResponseOf(it))
	}
	writeJSON(w, http.StatusOK, out)
}
