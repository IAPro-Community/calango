package api

import (
	"errors"
	"log"
	"net/http"

	"calango/internal/core"
)

// Error codes returned in the "code" field. They match the Error schema in api/openapi.yaml.
const (
	codeNotFound     = "not_found"
	codeInvalidInput = "invalid_input"
	codeConflict     = "conflict"
	codeUnauthorized = "unauthorized"
	codeForbidden    = "forbidden"
	codeRateLimited  = "rate_limited"
	codeInternal     = "internal"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// writeServiceError maps domain errors to HTTP. Anything else is logged and
// reported as internal, so storage details never reach the client.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, core.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, err.Error())
	case errors.Is(err, core.ErrConflict):
		writeError(w, http.StatusConflict, codeConflict, err.Error())
	case errors.Is(err, core.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, codeInvalidInput, err.Error())
	case errors.Is(err, core.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, codeUnauthorized, err.Error())
	default:
		log.Printf("api: %v", err)
		writeError(w, http.StatusInternalServerError, codeInternal, "internal error")
	}
}
