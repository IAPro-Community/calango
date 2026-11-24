package api

import (
	"context"
	"errors"
	"net/http"

	"calango/internal/core"
	"calango/internal/events"
	"calango/internal/maestro"
)

func (h *handler) registerMaestro(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/maestro", h.need(core.PermRead, h.getMaestro))
	mux.HandleFunc("POST /v1/maestro/install", h.need(core.PermWrite, h.installMaestro))
	mux.HandleFunc("POST /v1/maestro/update", h.need(core.PermWrite, h.updateMaestro))
	mux.HandleFunc("POST /v1/maestro/rollback", h.need(core.PermWrite, h.rollbackMaestro))
	mux.HandleFunc("PATCH /v1/maestro/settings", h.need(core.PermWrite, h.patchMaestroSettings))
}

func (h *handler) getMaestro(w http.ResponseWriter, r *http.Request) {
	status, err := h.maestro.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "maestro_status_failed", "Unable to read Maestro status")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

type maestroInstallRequest struct {
	Version string `json:"version"`
}

func (h *handler) installMaestro(w http.ResponseWriter, r *http.Request) {
	var body maestroInstallRequest
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	if err := h.maestro.Install(r.Context(), body.Version); err != nil {
		writeMaestroError(w, err)
		h.publishMaestro(events.TypeMaestroUpdateFailed, map[string]string{"result": "failed"})
		return
	}
	status, err := h.maestro.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "maestro_status_failed", "Unable to read Maestro status")
		return
	}
	h.publishMaestro(events.TypeMaestroUpdated, status)
	writeJSON(w, http.StatusOK, status)
}

func (h *handler) updateMaestro(w http.ResponseWriter, r *http.Request) {
	if err := decodeOptionalJSON(w, r, struct{}{}); err != nil {
		return
	}
	release, err := h.maestro.Update(r.Context())
	if err != nil {
		writeMaestroError(w, err)
		h.publishMaestro(events.TypeMaestroUpdateFailed, map[string]string{"result": "failed"})
		return
	}
	status, statusErr := h.maestro.Status(r.Context())
	if statusErr != nil {
		writeError(w, http.StatusInternalServerError, "maestro_status_failed", "Unable to read Maestro status")
		return
	}
	if release != nil {
		h.publishMaestro(events.TypeMaestroUpdated, status)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": status, "updated": release != nil})
}

func (h *handler) rollbackMaestro(w http.ResponseWriter, r *http.Request) {
	if err := decodeOptionalJSON(w, r, struct{}{}); err != nil {
		return
	}
	if err := h.maestro.Rollback(r.Context()); err != nil {
		writeMaestroError(w, err)
		return
	}
	status, err := h.maestro.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "maestro_status_failed", "Unable to read Maestro status")
		return
	}
	h.publishMaestro(events.TypeMaestroUpdated, status)
	writeJSON(w, http.StatusOK, status)
}

type maestroSettingsRequest struct {
	AutoUpdate *bool `json:"auto_update"`
}

func (h *handler) patchMaestroSettings(w http.ResponseWriter, r *http.Request) {
	var body maestroSettingsRequest
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	if body.AutoUpdate == nil {
		writeError(w, http.StatusBadRequest, "auto_update_required", "auto_update is required")
		return
	}
	if err := h.maestro.SetAutoUpdate(*body.AutoUpdate); err != nil {
		writeError(w, http.StatusInternalServerError, "maestro_settings_failed", "Unable to save Maestro settings")
		return
	}
	status, err := h.maestro.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "maestro_status_failed", "Unable to read Maestro status")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *handler) publishMaestro(kind string, data any) {
	if h.events != nil {
		h.events.Publish(events.Event{Type: kind, OwnerID: h.ownerID, Data: data})
	}
}

func writeMaestroError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code, message := "maestro_operation_failed", "Maestro operation failed"
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		status = http.StatusRequestTimeout
	case errors.Is(err, maestro.ErrInvalidVersion):
		status, code, message = http.StatusBadRequest, "invalid_version", "Only stable Maestro V1 versions may be installed"
	case errors.Is(err, maestro.ErrActiveRun):
		status, code, message = http.StatusConflict, "active_run", "Maestro cannot change versions while a run is active"
	case errors.Is(err, maestro.ErrNoRollback):
		status, code, message = http.StatusConflict, "no_previous_version", "No previous verified Maestro version is available"
	}
	writeError(w, status, code, message)
}
