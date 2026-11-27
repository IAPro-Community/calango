package api

import (
	"net/http"
	"testing"
)

func TestPrivateAPIResponsesMustNotBeCached(t *testing.T) {
	h := newTestHandler(t)
	for _, tc := range []struct {
		method, path, token, body string
	}{
		{http.MethodGet, "/v1/agents", testToken, ""},
		{http.MethodGet, "/v1/memories", testToken, ""},
		{http.MethodGet, "/v1/agents", "", ""},
		{http.MethodPost, "/v1/pair", "", "{}"},
	} {
		rec := call(t, h, tc.method, tc.path, tc.token, tc.body)
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s %s: Cache-Control=%q", tc.method, tc.path, got)
		}
	}
}
