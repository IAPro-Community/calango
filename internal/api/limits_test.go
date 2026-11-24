package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestJSONRequestBodyLimitRejectsOversizedPayload(t *testing.T) {
	h := newTestHandler(t)
	body := `{"name":"` + strings.Repeat("x", maxBodyBytes) + `"}`

	rec := call(t, h, http.MethodPost, "/v1/agents", testToken, body)
	expectStatus(t, rec, http.StatusBadRequest)
}
