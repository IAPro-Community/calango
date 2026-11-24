package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type routineJSON struct {
	ID     string `json:"id"`
	Paused bool   `json:"paused"`
}

func decodeRoutine(t *testing.T, body []byte) routineJSON {
	t.Helper()
	var r routineJSON
	if err := json.Unmarshal(body, &r); err != nil {
		t.Fatalf("decode routine: %v", err)
	}
	return r
}

func createTestAgent(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := call(t, h, http.MethodPost, "/v1/agents", testToken,
		fmt.Sprintf(`{"name":"Routine agent","token_limit_month":100000,%s}`, testChain))
	expectStatus(t, rec, http.StatusCreated)
	var agent struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &agent); err != nil {
		t.Fatalf("decode agent: %v", err)
	}
	return agent.ID
}

func routineBodyFor(agentID string) string {
	return fmt.Sprintf(`{"agent_id":%q,"name":"Morning brief","prompt":"Summarise overnight mail.","trigger_kind":"cron","schedule":"0 8 * * *","run_on":"pc","notify_on":"news"}`, agentID)
}

func TestRoutineLifecycleThroughHTTP(t *testing.T) {
	h := newTestHandler(t)
	agentID := createTestAgent(t, h)

	created := call(t, h, http.MethodPost, "/v1/routines", testToken, routineBodyFor(agentID))
	expectStatus(t, created, http.StatusCreated)
	r := decodeRoutine(t, created.Body.Bytes())
	if r.ID == "" || r.Paused {
		t.Fatalf("created routine = %+v; id must be set and routine must start active", r)
	}
	path := "/v1/routines/" + r.ID

	expectStatus(t, call(t, h, http.MethodGet, path, testToken, ""), http.StatusOK)

	patched := call(t, h, http.MethodPatch, path, testToken, `{"name":"Evening brief","paused":true}`)
	expectStatus(t, patched, http.StatusOK)
	if got := decodeRoutine(t, patched.Body.Bytes()); !got.Paused {
		t.Fatalf("patch paused=true returned paused=false: %s", patched.Body.String())
	}
	if !strings.Contains(patched.Body.String(), `"name":"Evening brief"`) {
		t.Fatalf("patch did not change name: %s", patched.Body.String())
	}

	list := call(t, h, http.MethodGet, "/v1/routines", testToken, "")
	expectStatus(t, list, http.StatusOK)
	if strings.Count(list.Body.String(), `"id"`) != 1 {
		t.Fatalf("list = %s, want exactly one routine", list.Body.String())
	}

	expectStatus(t, call(t, h, http.MethodDelete, path, testToken, ""), http.StatusNoContent)
	gone := call(t, h, http.MethodGet, path, testToken, "")
	expectStatus(t, gone, http.StatusNotFound)
	if got := errorCode(t, gone); got != "not_found" {
		t.Fatalf("code = %q, want not_found", got)
	}
}

func TestRunNowQueuesOneManualOccurrence(t *testing.T) {
	h := newTestHandler(t)
	agentID := createTestAgent(t, h)
	created := call(t, h, http.MethodPost, "/v1/routines", testToken, routineBodyFor(agentID))
	expectStatus(t, created, http.StatusCreated)
	path := "/v1/routines/" + decodeRoutine(t, created.Body.Bytes()).ID

	run := call(t, h, http.MethodPost, path+"/run", testToken, "")
	expectStatus(t, run, http.StatusAccepted)
	if !strings.Contains(run.Body.String(), "@manual:") {
		t.Fatalf("run-now occurrence id must be manual: %s", run.Body.String())
	}

	runs := call(t, h, http.MethodGet, path+"/runs", testToken, "")
	expectStatus(t, runs, http.StatusOK)
	if strings.Count(runs.Body.String(), `"occurrence_id"`) != 1 {
		t.Fatalf("runs = %s, want exactly one occurrence", runs.Body.String())
	}

	expectStatus(t, call(t, h, http.MethodPost, "/v1/routines/missing/run", testToken, ""), http.StatusNotFound)
}

func TestRoutineBodiesAreValidatedStrictly(t *testing.T) {
	h := newTestHandler(t)
	agentID := createTestAgent(t, h)

	unknownField := call(t, h, http.MethodPost, "/v1/routines", testToken,
		fmt.Sprintf(`{"agent_id":%q,"name":"A","prompt":"p","trigger_kind":"cron","schedule":"0 8 * * *","bogus":true}`, agentID))
	expectStatus(t, unknownField, http.StatusBadRequest)

	badCron := call(t, h, http.MethodPost, "/v1/routines", testToken,
		fmt.Sprintf(`{"agent_id":%q,"name":"A","prompt":"p","trigger_kind":"cron","schedule":"not a cron"}`, agentID))
	expectStatus(t, badCron, http.StatusBadRequest)

	emptyPrompt := call(t, h, http.MethodPost, "/v1/routines", testToken,
		fmt.Sprintf(`{"agent_id":%q,"name":"A","prompt":"  ","trigger_kind":"cron","schedule":"0 8 * * *"}`, agentID))
	expectStatus(t, emptyPrompt, http.StatusBadRequest)

	halfWindow := call(t, h, http.MethodPost, "/v1/routines", testToken,
		fmt.Sprintf(`{"agent_id":%q,"name":"A","prompt":"p","trigger_kind":"cron","schedule":"0 8 * * *","quiet_start":60}`, agentID))
	expectStatus(t, halfWindow, http.StatusBadRequest)
}

func TestRoutineRoutesRequireBearerTokenAndWriteScope(t *testing.T) {
	h := newTestHandler(t)
	agentID := createTestAgent(t, h)

	expectStatus(t, call(t, h, http.MethodGet, "/v1/routines", "", ""), http.StatusUnauthorized)

	reader := pairDevice(t, h, issuePairingCode(t, h, "read"))
	expectStatus(t, call(t, h, http.MethodGet, "/v1/routines", reader.Token, ""), http.StatusOK)
	rec := call(t, h, http.MethodPost, "/v1/routines", reader.Token, routineBodyFor(agentID))
	expectStatus(t, rec, http.StatusForbidden)
	if got := errorCode(t, rec); got != "forbidden" {
		t.Fatalf("error code = %q, want forbidden", got)
	}
}
