package api

import (
	"errors"
	"net/http"

	"calango/internal/core"
	"calango/internal/skills"
)

// Error codes of the skill import routes.
const (
	codeLicenseConfirmation = "license_confirmation_required"
	codeReplaceRequired     = "replace_required"
	codeSkillChanged        = "skill_changed"
)

func (h *handler) registerSkills(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/skill-imports/discover", h.need(core.PermRead, h.discoverSkills))
	mux.HandleFunc("GET /v1/skill-imports", h.need(core.PermRead, h.listImportedSkills))
	mux.HandleFunc("POST /v1/skill-imports", h.need(core.PermWrite, h.importSkill))
	mux.HandleFunc("DELETE /v1/skill-imports/{id}", h.need(core.PermWrite, h.removeImportedSkill))
}

func (h *handler) discoverSkills(w http.ResponseWriter, r *http.Request) {
	d, err := h.skills.Discover(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *handler) listImportedSkills(w http.ResponseWriter, r *http.Request) {
	list, err := h.skills.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// skillImportBody is the SkillImportInput schema.
type skillImportBody struct {
	SourceID       string `json:"source_id"`
	Hash           string `json:"hash"`
	ConfirmLicense bool   `json:"confirm_license"`
	Replace        bool   `json:"replace"`
}

type skillImportResponse struct {
	Skill  skills.Imported `json:"skill"`
	Result string          `json:"result"`
}

func (h *handler) importSkill(w http.ResponseWriter, r *http.Request) {
	var body skillImportBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	imported, result, err := h.skills.Import(r.Context(), skills.ImportRequest{
		SourceID: body.SourceID, Hash: body.Hash, ConfirmLicense: body.ConfirmLicense, Replace: body.Replace,
	})
	switch {
	case errors.Is(err, skills.ErrLicenseConfirmation):
		writeError(w, http.StatusConflict, codeLicenseConfirmation, "this skill declares no usable license; confirm to import it")
		return
	case errors.Is(err, skills.ErrReplaceRequired):
		writeError(w, http.StatusConflict, codeReplaceRequired, "another version is imported; set replace to update it")
		return
	case errors.Is(err, skills.ErrChanged):
		writeError(w, http.StatusConflict, codeSkillChanged, "the skill changed since it was discovered; discover again")
		return
	case err != nil:
		writeServiceError(w, err)
		return
	}
	status := http.StatusOK
	if result == skills.ResultCreated {
		status = http.StatusCreated
	}
	writeJSON(w, status, skillImportResponse{Skill: imported, Result: result})
}

func (h *handler) removeImportedSkill(w http.ResponseWriter, r *http.Request) {
	if err := h.skills.Remove(r.Context(), r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
