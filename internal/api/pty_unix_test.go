//go:build unix

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"calango/internal/pty"
)

// newPTYEnv returns a test server whose terminal runs /bin/sh in a temporary
// workspace, at most two sessions at a time.
func newPTYEnv(t *testing.T) (*httptest.Server, http.Handler, string) {
	t.Helper()
	root := t.TempDir()
	dirs, err := pty.NewRootDir(root)
	if err != nil {
		t.Fatal(err)
	}
	mgr, err := pty.NewManager(pty.Config{
		Dirs:        dirs,
		Shell:       func() (string, []string) { return "/bin/sh", nil },
		Env:         append(pty.FilterEnv(os.Environ()), "PS1=$ "),
		MaxSessions: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	gw := &blockingGateway{done: make(chan struct{})}
	h, _, _ := newTestEnvWithGateway(t, nil, gw, func() { close(gw.done) }, func(c *Config) { c.Terminal = mgr })
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, h, dirs.Root()
}

func createPTY(t *testing.T, h http.Handler, token, body string) ptySessionResponse {
	t.Helper()
	rec := call(t, h, http.MethodPost, "/v1/pty", token, body)
	expectStatus(t, rec, http.StatusCreated)
	var out ptySessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	return out
}

func ptyURL(srv *httptest.Server, id string) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/v1/pty/" + id
}

func dialPTY(srv *httptest.Server, id, token string) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	return websocket.DefaultDialer.Dial(ptyURL(srv, id), header)
}

func mustDialPTY(t *testing.T, srv *httptest.Server, id string) *websocket.Conn {
	t.Helper()
	conn, _, err := dialPTY(srv, id, testToken)
	if err != nil {
		t.Fatalf("dial pty: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func sendInput(t *testing.T, conn *websocket.Conn, s string) {
	t.Helper()
	if err := conn.WriteMessage(websocket.BinaryMessage, []byte(s)); err != nil {
		t.Fatalf("send input: %v", err)
	}
}

// readPTYUntil reads binary output until pattern matches, and returns the match.
// It fails on a text frame other than exit, or after five seconds.
func readPTYUntil(t *testing.T, conn *websocket.Conn, pattern string) []string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	var buf bytes.Buffer
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		if m := re.FindStringSubmatch(buf.String()); m != nil {
			return m
		}
		kind, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read pty (want %q, got %q): %v", pattern, buf.String(), err)
		}
		if kind == websocket.BinaryMessage {
			buf.Write(data)
		}
	}
}

// readExit skips output until the exit control frame and returns it.
func readExit(t *testing.T, conn *websocket.Conn) ptyControl {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		kind, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read until exit: %v", err)
		}
		if kind != websocket.TextMessage {
			continue
		}
		var ctl ptyControl
		if err := json.Unmarshal(data, &ctl); err != nil || ctl.Type != "exit" {
			t.Fatalf("control frame %q", data)
		}
		return ctl
	}
}

func TestPTYRoutesRequireTheBearerToken(t *testing.T) {
	srv, h, _ := newPTYEnv(t)
	expectStatus(t, call(t, h, http.MethodPost, "/v1/pty", "", ""), http.StatusUnauthorized)
	expectStatus(t, call(t, h, http.MethodPost, "/v1/pty", "wrong-token", ""), http.StatusUnauthorized)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/pty", "", ""), http.StatusUnauthorized)

	session := createPTY(t, h, testToken, "")
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/pty/"+session.ID, "", ""), http.StatusUnauthorized)
	for _, token := range []string{"", "wrong-token"} {
		conn, resp, err := dialPTY(srv, session.ID, token)
		if err == nil {
			_ = conn.Close()
			t.Fatalf("dial with token %q succeeded", token)
		}
		if resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("dial with token %q: response %v, want 401", token, resp)
		}
	}
}

func TestPTYNeedsWriteScope(t *testing.T) {
	srv, h, _ := newPTYEnv(t)
	session := createPTY(t, h, testToken, "")
	reader := pairDevice(t, h, issuePairingCode(t, h, "read"))

	expectStatus(t, call(t, h, http.MethodPost, "/v1/pty", reader.Token, ""), http.StatusForbidden)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/pty", reader.Token, ""), http.StatusForbidden)
	conn, resp, err := dialPTY(srv, session.ID, reader.Token)
	if err == nil {
		_ = conn.Close()
		t.Fatal("read-scope device attached to a terminal")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("response %v, want 403", resp)
	}
}

func TestPTYSessionIsBoundToItsDevice(t *testing.T) {
	srv, h, _ := newPTYEnv(t)
	session := createPTY(t, h, testToken, "")
	other := pairDevice(t, h, issuePairingCode(t, h, ""))

	_, resp, err := dialPTY(srv, session.ID, other.Token)
	if err == nil || resp == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("other device attach: err %v response %v, want 404", err, resp)
	}
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/pty/"+session.ID, other.Token, ""), http.StatusNotFound)
	list := call(t, h, http.MethodGet, "/v1/pty", other.Token, "")
	expectStatus(t, list, http.StatusOK)
	if strings.Contains(list.Body.String(), session.ID) {
		t.Fatalf("other device lists the session: %s", list.Body.String())
	}
}

func TestPTYEchoAndResizeOverWebSocket(t *testing.T) {
	srv, h, root := newPTYEnv(t)
	if err := os.Mkdir(root+"/proj", 0o700); err != nil {
		t.Fatal(err)
	}
	session := createPTY(t, h, testToken, `{"cwd":"proj","cols":90,"rows":30}`)
	if session.Cwd != root+"/proj" || session.Cols != 90 || session.Rows != 30 || session.Attached {
		t.Fatalf("session = %+v", session)
	}
	conn := mustDialPTY(t, srv, session.ID)

	sendInput(t, conn, "echo api-$((6*7)); pwd\n")
	readPTYUntil(t, conn, `(?s)api-42.*`+regexp.QuoteMeta(root+"/proj"))

	if err := conn.WriteJSON(ptyControl{Type: "resize", Cols: 120, Rows: 40}); err != nil {
		t.Fatal(err)
	}
	sendInput(t, conn, "stty size\n")
	readPTYUntil(t, conn, `40 120`)

	list := call(t, h, http.MethodGet, "/v1/pty", testToken, "")
	expectStatus(t, list, http.StatusOK)
	var items []ptySessionResponse
	if err := json.Unmarshal(list.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != session.ID || !items[0].Attached || items[0].Cols != 120 {
		t.Fatalf("list = %+v", items)
	}
}

func TestPTYSecondClientIsRefused(t *testing.T) {
	srv, h, _ := newPTYEnv(t)
	session := createPTY(t, h, testToken, "")
	mustDialPTY(t, srv, session.ID)
	_, resp, err := dialPTY(srv, session.ID, testToken)
	if err == nil || resp == nil || resp.StatusCode != http.StatusConflict {
		t.Fatalf("second attach: err %v response %v, want 409", err, resp)
	}
}

func TestPTYCreateValidatesInput(t *testing.T) {
	_, h, _ := newPTYEnv(t)
	for _, body := range []string{
		`{"cwd":"/"}`, `{"cwd":".."}`, `{"cwd":"missing"}`,
		`{"cols":0,"rows":10}`, `{"cols":5000,"rows":10}`, `{"shell":"/bin/bash"}`, `not json`,
	} {
		rec := call(t, h, http.MethodPost, "/v1/pty", testToken, body)
		expectStatus(t, rec, http.StatusBadRequest)
		if got := errorCode(t, rec); got != "invalid_input" {
			t.Fatalf("body %s: code %q", body, got)
		}
	}
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/pty/not-an-id", testToken, ""), http.StatusNotFound)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/pty/"+strings.Repeat("A", 43), testToken, ""), http.StatusNotFound)
}

func TestPTYSessionLimit(t *testing.T) {
	_, h, _ := newPTYEnv(t)
	first := createPTY(t, h, testToken, "")
	createPTY(t, h, testToken, "")
	rec := call(t, h, http.MethodPost, "/v1/pty", testToken, "")
	expectStatus(t, rec, http.StatusTooManyRequests)
	if got := errorCode(t, rec); got != "rate_limited" {
		t.Fatalf("code %q", got)
	}
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/pty/"+first.ID, testToken, ""), http.StatusNoContent)
	createPTY(t, h, testToken, "")
}

func TestPTYDeleteKillsTheTreeAndClosesTheSocket(t *testing.T) {
	srv, h, _ := newPTYEnv(t)
	session := createPTY(t, h, testToken, "")
	conn := mustDialPTY(t, srv, session.ID)
	sendInput(t, conn, "sleep 300 & echo BG=$!\n")
	bg, err := strconv.Atoi(readPTYUntil(t, conn, `BG=(\d+)`)[1])
	if err != nil {
		t.Fatal(err)
	}

	expectStatus(t, call(t, h, http.MethodDelete, "/v1/pty/"+session.ID, testToken, ""), http.StatusNoContent)
	ctl := readExit(t, conn)
	if ctl.Reason != pty.ReasonTerminated || ctl.ExitCode != nil {
		t.Fatalf("exit frame = %+v", ctl)
	}
	if _, _, err := conn.ReadMessage(); !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		t.Fatalf("after exit: %v, want normal close", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for syscall.Kill(bg, 0) == nil {
		if data, _ := os.ReadFile("/proc/" + strconv.Itoa(bg) + "/stat"); bytes.Contains(data, []byte(") Z ")) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("background job %d survived DELETE", bg)
		}
		time.Sleep(20 * time.Millisecond)
	}
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/pty/"+session.ID, testToken, ""), http.StatusNotFound)
}

func TestPTYShellExitSendsExitCode(t *testing.T) {
	srv, h, _ := newPTYEnv(t)
	session := createPTY(t, h, testToken, "")
	conn := mustDialPTY(t, srv, session.ID)
	sendInput(t, conn, "exit 7\n")
	ctl := readExit(t, conn)
	if ctl.Reason != pty.ReasonExited || ctl.ExitCode == nil || *ctl.ExitCode != 7 {
		t.Fatalf("exit frame = %+v", ctl)
	}
}

func TestPTYMalformedControlFrameClosesTheSocket(t *testing.T) {
	srv, h, _ := newPTYEnv(t)
	session := createPTY(t, h, testToken, "")
	conn := mustDialPTY(t, srv, session.ID)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"resize","cols":0,"rows":0}`)); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, _, err := conn.ReadMessage()
		if err == nil {
			continue
		}
		if !websocket.IsCloseError(err, websocket.CloseUnsupportedData) {
			t.Fatalf("err = %v, want close 1003", err)
		}
		break
	}
	// The session survives a bad client and can be attached again once the server
	// has detached the first socket.
	deadline := time.Now().Add(5 * time.Second)
	for {
		again, resp, err := dialPTY(srv, session.ID, testToken)
		if err == nil {
			_ = again.Close()
			return
		}
		if resp == nil || resp.StatusCode != http.StatusConflict || time.Now().After(deadline) {
			t.Fatalf("reattach: err %v response %v", err, resp)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
