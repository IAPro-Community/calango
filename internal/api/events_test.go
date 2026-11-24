package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// eventsURL returns the WebSocket address of /v1/events on the test server.
func eventsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/v1/events"
}

// dialEvents opens /v1/events with the bearer token and fails the test on error.
func dialEvents(t *testing.T, srv *httptest.Server, token string) *websocket.Conn {
	t.Helper()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	conn, _, err := websocket.DefaultDialer.Dial(eventsURL(srv), header)
	if err != nil {
		t.Fatalf("dial events: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// readFrame returns the next event type and data, waiting at most five seconds.
func readFrame(t *testing.T, conn *websocket.Conn) (string, json.RawMessage) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("read deadline: %v", err)
	}
	var frame struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := conn.ReadJSON(&frame); err != nil {
		t.Fatalf("read event: %v", err)
	}
	return frame.Type, frame.Data
}

func TestEventsRequireTheBearerToken(t *testing.T) {
	h, _ := newTestEnv(t)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	conn, resp, err := websocket.DefaultDialer.Dial(eventsURL(srv), nil)
	if err == nil {
		_ = conn.Close()
		t.Fatal("dial without token succeeded")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("response = %v, want 401", resp)
	}
}

func TestEventsStreamMessageThenRunChanges(t *testing.T) {
	h, _ := newTestEnv(t)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	convID := createDM(t, h)
	conn := dialEvents(t, srv, testToken)

	sent := sendMessage(t, h, convID, "events-key-1", "oi")

	if typ, _ := readFrame(t, conn); typ != "message.created" {
		t.Fatalf("first event = %q, want message.created", typ)
	}
	typ, data := readFrame(t, conn)
	if typ != "run.started" {
		t.Fatalf("second event = %q, want run.started", typ)
	}
	var started runResponse
	if err := json.Unmarshal(data, &started); err != nil {
		t.Fatalf("decode run.started: %v", err)
	}
	if started.ID != sent.Run.ID || started.Status != "running" {
		t.Fatalf("run.started = %+v, want run %s running", started, sent.Run.ID)
	}

	expectStatus(t, call(t, h, http.MethodPost, "/v1/runs/"+sent.Run.ID+"/stop", testToken, ""), http.StatusNoContent)

	typ, data = readFrame(t, conn)
	// Stopping records the cancelled provider attempt as a step before run.finished.
	for typ == "run.step" {
		typ, data = readFrame(t, conn)
	}
	if typ != "run.finished" {
		t.Fatalf("third event = %q, want run.finished", typ)
	}
	var finished runResponse
	if err := json.Unmarshal(data, &finished); err != nil {
		t.Fatalf("decode run.finished: %v", err)
	}
	if finished.ID != sent.Run.ID || finished.Status != "stopped" {
		t.Fatalf("run.finished = %+v, want run %s stopped", finished, sent.Run.ID)
	}
}
