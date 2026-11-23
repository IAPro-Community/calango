package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestUsageIsReadScopedAndEmptyIsAnArray(t *testing.T) {
	h := newTestHandler(t)
	read := pairDevice(t, h, issuePairingCode(t, h, "read"))
	approveOnly := pairDevice(t, h, issuePairingCode(t, h, "approve_only"))

	rec := call(t, h, http.MethodGet, "/v1/usage", testToken, "")
	expectStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Fatalf("empty usage body = %q, want []", got)
	}
	expectStatus(t, call(t, h, http.MethodGet, "/v1/usage", read.Token, ""), http.StatusOK)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/usage", approveOnly.Token, ""), http.StatusForbidden)
}

func TestUsageRejectsBadParameters(t *testing.T) {
	h := newTestHandler(t)
	cases := map[string]string{
		"group_by=provider": "/v1/usage?group_by=provider",
		"from not RFC 3339": "/v1/usage?from=yesterday",
		"to not after from": "/v1/usage?from=2026-05-02T00:00:00Z&to=2026-05-01T00:00:00Z",
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			rec := call(t, h, http.MethodGet, path, testToken, "")
			expectStatus(t, rec, http.StatusBadRequest)
			if code := errorCode(t, rec); code != codeInvalidInput {
				t.Fatalf("error code = %q, want %q", code, codeInvalidInput)
			}
		})
	}
}
