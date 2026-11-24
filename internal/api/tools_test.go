package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestConnectorsAndGrantsThroughHTTP(t *testing.T) {
	h := newTestHandler(t)
	agentID := createTestAgent(t, h)

	expectStatus(t, call(t, h, http.MethodGet, "/v1/connectors", "", ""), http.StatusUnauthorized)

	// The URL points at a closed port, so the connector cannot answer.
	rec := call(t, h, http.MethodPost, "/v1/connectors", testToken,
		`{"name":"notes","transport":"http","url":"http://127.0.0.1:9/mcp"}`)
	expectStatus(t, rec, http.StatusCreated)
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("created connector body %q: %v", rec.Body.String(), err)
	}
	expectStatus(t, call(t, h, http.MethodPost, "/v1/connectors", testToken,
		`{"name":"bad","transport":"ftp","url":"http://127.0.0.1:9/mcp"}`), http.StatusBadRequest)

	list := call(t, h, http.MethodGet, "/v1/connectors", testToken, "")
	expectStatus(t, list, http.StatusOK)
	if !strings.Contains(list.Body.String(), `"name":"notes"`) {
		t.Fatalf("connectors = %s, want notes", list.Body.String())
	}

	grantPath := "/v1/agents/" + agentID + "/tools/" + created.ID
	expectStatus(t, call(t, h, http.MethodPut, grantPath, testToken, `{"access":"read"}`), http.StatusNoContent)
	expectStatus(t, call(t, h, http.MethodPut, grantPath, testToken, `{"access":"write"}`), http.StatusNoContent)
	expectStatus(t, call(t, h, http.MethodPut, grantPath, testToken, `{"access":"admin"}`), http.StatusBadRequest)

	grants := call(t, h, http.MethodGet, "/v1/agents/"+agentID+"/tools", testToken, "")
	expectStatus(t, grants, http.StatusOK)
	if !strings.Contains(grants.Body.String(), `"access":"write"`) {
		t.Fatalf("grants = %s, want write access", grants.Body.String())
	}

	expectStatus(t, call(t, h, http.MethodPut, grantPath, testToken, `{"access":"none"}`), http.StatusNoContent)
	grants = call(t, h, http.MethodGet, "/v1/agents/"+agentID+"/tools", testToken, "")
	if got := strings.TrimSpace(grants.Body.String()); got != "[]" {
		t.Fatalf("grants after none = %s, want []", got)
	}

	unreachable := call(t, h, http.MethodGet, "/v1/connectors/"+created.ID+"/tools", testToken, "")
	expectStatus(t, unreachable, http.StatusBadGateway)
	if got := errorCode(t, unreachable); got != codeConnectorUnavailable {
		t.Fatalf("error code = %q, want %s", got, codeConnectorUnavailable)
	}

	expectStatus(t, call(t, h, http.MethodDelete, "/v1/connectors/"+created.ID, testToken, ""), http.StatusNoContent)
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/connectors/"+created.ID, testToken, ""), http.StatusNotFound)
}

func TestReadScopeCannotManageConnectors(t *testing.T) {
	h := newTestHandler(t)
	reader := pairDevice(t, h, issuePairingCode(t, h, "read"))

	expectStatus(t, call(t, h, http.MethodGet, "/v1/connectors", reader.Token, ""), http.StatusOK)
	rec := call(t, h, http.MethodPost, "/v1/connectors", reader.Token,
		`{"name":"notes","transport":"http","url":"http://127.0.0.1:9/mcp"}`)
	expectStatus(t, rec, http.StatusForbidden)
	if got := errorCode(t, rec); got != "forbidden" {
		t.Fatalf("error code = %q, want forbidden", got)
	}
}
