package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestAgentMaestroSettingPersistsAcrossCreateAndReplace(t *testing.T) {
	h := newTestHandler(t)
	body := fmt.Sprintf(`{"name":"Maestro agent","token_limit_month":1000,"maestro_enabled":true,%s}`, testChain)
	created := call(t, h, http.MethodPost, "/v1/agents", testToken, body)
	expectStatus(t, created, http.StatusCreated)

	var agent agentResponse
	if err := json.Unmarshal(created.Body.Bytes(), &agent); err != nil {
		t.Fatalf("decode created agent: %v", err)
	}
	if !agent.MaestroEnabled {
		t.Fatal("created agent did not retain maestro_enabled=true")
	}

	body = fmt.Sprintf(`{"name":"Maestro agent","token_limit_month":1000,"maestro_enabled":false,%s}`, testChain)
	updated := call(t, h, http.MethodPut, "/v1/agents/"+agent.ID, testToken, body)
	expectStatus(t, updated, http.StatusOK)
	if err := json.Unmarshal(updated.Body.Bytes(), &agent); err != nil {
		t.Fatalf("decode replaced agent: %v", err)
	}
	if agent.MaestroEnabled {
		t.Fatal("replaced agent retained maestro_enabled=true")
	}
}
