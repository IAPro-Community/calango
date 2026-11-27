package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"calango/internal/core"
	"calango/internal/pty"
)

const (
	// ptyMaxFrame bounds one client frame: typed input or a control message.
	ptyMaxFrame = 64 << 10
	// ptyQueuedFlush bounds how many queued output chunks are flushed after the session ends.
	ptyQueuedFlush = 256
)

// ptyUpgrader follows eventsUpgrader: the bearer token in the Authorization header is
// the credential, checked before the upgrade, so any origin is accepted.
var ptyUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// ptySessionRequest is the PtySessionInput schema.
type ptySessionRequest struct {
	Cwd  string `json:"cwd"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// ptySessionResponse is the PtySession schema. Terminal content never appears here.
type ptySessionResponse struct {
	ID        string    `json:"id"`
	Cwd       string    `json:"cwd"`
	Shell     string    `json:"shell"`
	Cols      uint16    `json:"cols"`
	Rows      uint16    `json:"rows"`
	Attached  bool      `json:"attached"`
	CreatedAt time.Time `json:"created_at"`
}

// ptyControl is a JSON text frame on /v1/pty/{id}. Clients send "resize"; the server
// sends "exit" before it closes the socket because the session ended.
type ptyControl struct {
	Type     string `json:"type"`
	Cols     uint16 `json:"cols,omitempty"`
	Rows     uint16 `json:"rows,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// registerPTY mounts the integrated terminal. A terminal runs arbitrary commands as
// the node's user, so every route needs write scope, like any other route that acts.
func (h *handler) registerPTY(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/pty", h.need(core.PermWrite, h.createPTY))
	mux.HandleFunc("GET /v1/pty", h.need(core.PermWrite, h.listPTY))
	mux.HandleFunc("GET /v1/pty/{id}", h.need(core.PermWrite, h.attachPTY))
	mux.HandleFunc("DELETE /v1/pty/{id}", h.need(core.PermWrite, h.deletePTY))
}

func ptySessionOf(info pty.Info) ptySessionResponse {
	return ptySessionResponse{
		ID: info.ID, Cwd: info.Dir, Shell: info.Shell, Cols: info.Cols, Rows: info.Rows,
		Attached: info.Attached, CreatedAt: info.CreatedAt.UTC(),
	}
}

func (h *handler) createPTY(w http.ResponseWriter, r *http.Request) {
	var req ptySessionRequest
	if err := decodeOptionalJSON(w, r, &req); err != nil {
		writeServiceError(w, err)
		return
	}
	info, err := h.terminal.Create(r.Context(), pty.StartRequest{
		OwnerID: h.ownerID, DeviceID: principalOf(r.Context()).deviceID,
		Dir: req.Cwd, Cols: req.Cols, Rows: req.Rows,
	})
	if err != nil {
		writePTYError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ptySessionOf(info))
}

func (h *handler) listPTY(w http.ResponseWriter, r *http.Request) {
	infos := h.terminal.List(h.ownerID, principalOf(r.Context()).deviceID)
	out := make([]ptySessionResponse, 0, len(infos))
	for _, info := range infos {
		out = append(out, ptySessionOf(info))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) deletePTY(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !pty.ValidID(id) {
		writePTYError(w, pty.ErrNotFound)
		return
	}
	if err := h.terminal.Terminate(h.ownerID, principalOf(r.Context()).deviceID, id); err != nil {
		writePTYError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// attachPTY upgrades to a WebSocket bound to one session. Binary frames carry raw
// terminal bytes both ways; text frames carry ptyControl JSON. Closing the socket
// detaches: the session keeps running until DELETE, exit or the idle timeout.
func (h *handler) attachPTY(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !pty.ValidID(id) {
		writePTYError(w, pty.ErrNotFound)
		return
	}
	// Attach before the upgrade, so not found and busy are plain HTTP errors.
	att, err := h.terminal.Attach(h.ownerID, principalOf(r.Context()).deviceID, id)
	if err != nil {
		writePTYError(w, err)
		return
	}
	defer att.Detach()

	conn, err := ptyUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(ptyMaxFrame)
	if err := conn.SetReadDeadline(time.Now().Add(eventPongWait)); err != nil {
		return
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(eventPongWait))
	})

	gone := make(chan struct{})
	go func() {
		defer close(gone)
		readPTYInput(conn, att)
	}()

	ticker := time.NewTicker(eventPingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-gone:
			return
		case chunk := <-att.Output():
			if writePTYFrame(conn, websocket.BinaryMessage, chunk) != nil {
				return
			}
		case <-att.Done():
			flushPTYOutput(conn, att)
			code := att.ExitCode()
			ctl := ptyControl{Type: "exit", Reason: att.Reason()}
			if code >= 0 {
				ctl.ExitCode = &code
			}
			if data, err := json.Marshal(ctl); err == nil {
				_ = writePTYFrame(conn, websocket.TextMessage, data)
			}
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "terminal session ended"),
				time.Now().Add(eventWriteWait))
			return
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(eventWriteWait)); err != nil {
				return
			}
		}
	}
}

// readPTYInput forwards client frames until the socket fails or a frame is malformed.
// Input is passed to the terminal unchanged; the command policy of issue #89 applies
// to agent-driven sessions at session start (pty.StartPolicy), not to typed keys.
func readPTYInput(conn *websocket.Conn, att *pty.Attachment) {
	for {
		kind, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		switch kind {
		case websocket.BinaryMessage:
			if _, err := att.Write(data); err != nil {
				return
			}
		case websocket.TextMessage:
			var ctl ptyControl
			if json.Unmarshal(data, &ctl) != nil || ctl.Type != "resize" || att.Resize(ctl.Cols, ctl.Rows) != nil {
				_ = conn.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "invalid control frame"),
					time.Now().Add(eventWriteWait))
				return
			}
		}
	}
}

func writePTYFrame(conn *websocket.Conn, kind int, data []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(eventWriteWait)); err != nil {
		return err
	}
	return conn.WriteMessage(kind, data)
}

// flushPTYOutput sends output that was queued before the session ended.
func flushPTYOutput(conn *websocket.Conn, att *pty.Attachment) {
	for range ptyQueuedFlush {
		select {
		case chunk := <-att.Output():
			if writePTYFrame(conn, websocket.BinaryMessage, chunk) != nil {
				return
			}
		default:
			return
		}
	}
}

// writePTYError maps terminal errors to the API's stable codes.
func writePTYError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pty.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, "terminal session not found")
	case errors.Is(err, pty.ErrBusy):
		writeError(w, http.StatusConflict, codeConflict, err.Error())
	case errors.Is(err, pty.ErrLimit):
		writeError(w, http.StatusTooManyRequests, codeRateLimited, err.Error())
	case errors.Is(err, pty.ErrDenied):
		writeError(w, http.StatusForbidden, codeForbidden, err.Error())
	case errors.Is(err, pty.ErrInvalidInput), errors.Is(err, core.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, codeInvalidInput, err.Error())
	case errors.Is(err, pty.ErrClosed):
		writeError(w, http.StatusServiceUnavailable, codeInternal, "terminal is shutting down")
	default:
		writeServiceError(w, err)
	}
}
