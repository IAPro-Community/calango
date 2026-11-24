package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// createDM creates an agent and a DM with it, returning the conversation id.
func createDM(t *testing.T, h http.Handler) string {
	t.Helper()
	agent := call(t, h, http.MethodPost, "/v1/agents", testToken, fmt.Sprintf(`{"name":"Calango","token_limit_month":100000,%s}`, testChain))
	expectStatus(t, agent, http.StatusCreated)
	var a struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(agent.Body.Bytes(), &a); err != nil {
		t.Fatalf("decode agent: %v", err)
	}
	dm := call(t, h, http.MethodPost, "/v1/conversations", testToken, `{"agent_id":"`+a.ID+`","title":"Oi"}`)
	expectStatus(t, dm, http.StatusCreated)
	var c struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(dm.Body.Bytes(), &c); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}
	return c.ID
}

// postWithKey sends an authorised POST, with an Idempotency-Key header when key is not empty.
func postWithKey(t *testing.T, h http.Handler, path, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// sendMessage posts a message with the given idempotency key and decodes the response.
func sendMessage(t *testing.T, h http.Handler, convID, key, body string) sendMessageResponse {
	t.Helper()
	rec := postWithKey(t, h, "/v1/conversations/"+convID+"/messages", key, `{"body":"`+body+`"}`)
	expectStatus(t, rec, http.StatusCreated)
	var out sendMessageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode send message: %v", err)
	}
	return out
}

// waitRunStatus polls GET /v1/runs/{id} until the run reaches want or the deadline passes.
func waitRunStatus(t *testing.T, h http.Handler, runID, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		rec := call(t, h, http.MethodGet, "/v1/runs/"+runID, testToken, "")
		expectStatus(t, rec, http.StatusOK)
		var detail runDetailResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
			t.Fatalf("decode run: %v", err)
		}
		if detail.Status == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("run %s status = %q, want %q", runID, detail.Status, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestConversationsCreateListAndSendMessage(t *testing.T) {
	h := newTestHandler(t)
	convID := createDM(t, h)

	list := call(t, h, http.MethodGet, "/v1/conversations", testToken, "")
	expectStatus(t, list, http.StatusOK)
	var convs []conversationResponse
	if err := json.Unmarshal(list.Body.Bytes(), &convs); err != nil {
		t.Fatalf("decode conversations: %v", err)
	}
	if len(convs) != 1 || convs[0].ID != convID || convs[0].Kind != "dm" {
		t.Fatalf("conversations = %+v, want the one DM", convs)
	}

	missingKey := postWithKey(t, h, "/v1/conversations/"+convID+"/messages", "", `{"body":"oi"}`)
	expectStatus(t, missingKey, http.StatusBadRequest)
	if code := errorCode(t, missingKey); code != "invalid_input" {
		t.Fatalf("missing key code = %q, want invalid_input", code)
	}

	first := sendMessage(t, h, convID, "send-1", "oi")
	if first.Message.Body != "oi" || first.Message.AuthorKind != "person" {
		t.Fatalf("message = %+v, want person message with body oi", first.Message)
	}
	if first.Run.SourceMessageID != first.Message.ID {
		t.Fatalf("run source = %q, want message %q", first.Run.SourceMessageID, first.Message.ID)
	}

	again := sendMessage(t, h, convID, "send-1", "oi")
	if again.Run.ID != first.Run.ID || again.Message.ID != first.Message.ID {
		t.Fatalf("repeated key returned %+v, want the same message and run as %+v", again, first)
	}

	messages := call(t, h, http.MethodGet, "/v1/conversations/"+convID+"/messages", testToken, "")
	expectStatus(t, messages, http.StatusOK)
	var msgs []messageResponse
	if err := json.Unmarshal(messages.Body.Bytes(), &msgs); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1 after a repeated key", len(msgs))
	}
}

func TestConversationRunStopAndRetry(t *testing.T) {
	h := newTestHandler(t)
	convID := createDM(t, h)
	sent := sendMessage(t, h, convID, "send-1", "oi")
	runID := sent.Run.ID

	expectStatus(t, call(t, h, http.MethodPost, "/v1/runs/"+runID+"/stop", testToken, ""), http.StatusNoContent)
	waitRunStatus(t, h, runID, "stopped")

	again := call(t, h, http.MethodPost, "/v1/runs/"+runID+"/stop", testToken, "")
	expectStatus(t, again, http.StatusConflict)
	if code := errorCode(t, again); code != "conflict" {
		t.Fatalf("stopping a finished run code = %q, want conflict", code)
	}

	noKey := postWithKey(t, h, "/v1/runs/"+runID+"/retry", "", `{"instructions":"mais curto"}`)
	expectStatus(t, noKey, http.StatusBadRequest)

	retry := postWithKey(t, h, "/v1/runs/"+runID+"/retry", "retry-1", `{"instructions":"mais curto"}`)
	expectStatus(t, retry, http.StatusCreated)
	var r runResponse
	if err := json.Unmarshal(retry.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode retry: %v", err)
	}
	if r.RetryOf != runID || r.Instructions != "mais curto" || r.SourceMessageID != sent.Message.ID {
		t.Fatalf("retry = %+v, want retry_of %q with instructions and the same source message", r, runID)
	}

	expectStatus(t, call(t, h, http.MethodGet, "/v1/runs/missing", testToken, ""), http.StatusNotFound)
}

func TestConversationRunSteerQueuesGuidedContinuation(t *testing.T) {
	h := newTestHandler(t)
	convID := createDM(t, h)
	sent := sendMessage(t, h, convID, "send-1", "oi")

	noKey := postWithKey(t, h, "/v1/runs/"+sent.Run.ID+"/steer", "", `{"guidance":"mais curto"}`)
	expectStatus(t, noKey, http.StatusBadRequest)

	steer := postWithKey(t, h, "/v1/runs/"+sent.Run.ID+"/steer", "steer-1", `{"guidance":"mais curto"}`)
	expectStatus(t, steer, http.StatusAccepted)
	var r runResponse
	if err := json.Unmarshal(steer.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode steer: %v", err)
	}
	if r.SteerOf != sent.Run.ID || r.Instructions != "mais curto" || r.SourceMessageID != sent.Message.ID {
		t.Fatalf("steer run = %+v, want steer_of %q with the guidance and the same source message", r, sent.Run.ID)
	}

	again := postWithKey(t, h, "/v1/runs/"+sent.Run.ID+"/steer", "steer-1", `{"guidance":"mais curto"}`)
	expectStatus(t, again, http.StatusAccepted)
	var same runResponse
	if err := json.Unmarshal(again.Body.Bytes(), &same); err != nil {
		t.Fatalf("decode repeated steer: %v", err)
	}
	if same.ID != r.ID {
		t.Fatalf("repeated steer id = %q, want %q", same.ID, r.ID)
	}

	expectStatus(t, postWithKey(t, h, "/v1/runs/missing/steer", "steer-2", `{"guidance":"x"}`), http.StatusNotFound)
}

func TestRunDetailDoesNotExposeProviderErrorCanary(t *testing.T) {
	const canary = "provider-error-canary-8f0c"
	h, _, _ := newTestEnvWithGateway(
		t,
		nil,
		&failingGateway{err: fmt.Errorf("%s", canary)},
		nil,
	)
	convID := createDM(t, h)
	sent := sendMessage(t, h, convID, "secret-canary", "hello")
	waitRunStatus(t, h, sent.Run.ID, "failed")

	rec := call(t, h, http.MethodGet, "/v1/runs/"+sent.Run.ID, testToken, "")
	expectStatus(t, rec, http.StatusOK)
	if strings.Contains(rec.Body.String(), canary) {
		t.Fatalf("run detail leaked provider error canary: %s", rec.Body.String())
	}
}
