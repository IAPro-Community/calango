package api

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"calango/internal/core"
	"calango/internal/events"
	"calango/internal/maestro"
	"calango/internal/providers"
)

const (
	// eventWriteWait bounds one write. A client that cannot take a frame in time is disconnected.
	eventWriteWait = 10 * time.Second
	// eventPongWait is how long a client may stay silent before it counts as gone.
	eventPongWait = 60 * time.Second
	// eventPingPeriod keeps idle connections open through proxies. It must stay below eventPongWait.
	eventPingPeriod = 25 * time.Second
)

// eventsUpgrader accepts any origin. Access control is the bearer token in the
// Authorization header: browsers do not send it on cross-site WebSocket handshakes.
var eventsUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// eventFrame is one JSON message on /v1/events.
type eventFrame struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// stepEventData carries a step with its run id, so a client can route it without a lookup.
type stepEventData struct {
	RunID string `json:"run_id"`
	stepResponse
}

func (h *handler) registerEvents(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/events", h.need(core.PermRead, h.streamEvents))
}

// streamEvents upgrades the request to a WebSocket and pushes the owner's
// conversation events until the client leaves, goes silent, or falls too far
// behind. Nothing is replayed: after a reconnect the client reloads
// conversations and runs over the REST API.
func (h *handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	// Subscribe before the upgrade. The client receives 101 only after the
	// subscription exists, so no event published after the handshake is missed.
	sub := h.events.Subscribe(h.ownerID)
	defer sub.Close()

	conn, err := eventsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade has already answered the request with an HTTP error.
		return
	}
	defer conn.Close()

	// Clients send nothing. Reading exists to process pongs and to notice closes.
	if err := conn.SetReadDeadline(time.Now().Add(eventPongWait)); err != nil {
		return
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(eventPongWait))
	})
	gone := make(chan struct{})
	go func() {
		defer close(gone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(eventPingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-gone:
			return
		case ev, ok := <-sub.C:
			if !ok {
				_ = conn.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "too slow: reconnect and reload state"),
					time.Now().Add(eventWriteWait))
				return
			}
			data, known := eventDataOf(ev)
			if !known {
				continue
			}
			if err := conn.SetWriteDeadline(time.Now().Add(eventWriteWait)); err != nil {
				return
			}
			if err := conn.WriteJSON(eventFrame{Type: ev.Type, Data: data}); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(eventWriteWait)); err != nil {
				return
			}
		}
	}
}

// eventDataOf converts a published change into its API shape. Unknown payloads are skipped.
func eventDataOf(ev events.Event) (any, bool) {
	switch d := ev.Data.(type) {
	case core.Message:
		return messageOf(d), true
	case core.Run:
		return runOf(d), true
	case core.Step:
		return stepEventData{
			RunID: d.RunID,
			stepResponse: stepResponse{
				ID: d.ID, Seq: d.Seq, Kind: d.Kind, Payload: d.Payload, CreatedAt: d.CreatedAt,
			},
		}, true
	case providers.CapacityView:
		if ev.Type == events.TypeCapacityChanged {
			return d, true
		}
	case maestro.Status, maestro.Release:
		switch ev.Type {
		case events.TypeMaestroUpdated, events.TypeMaestroUpdateAvailable:
			return d, true
		}
	case map[string]string:
		if ev.Type == events.TypeMaestroUpdateFailed && d["result"] == "failed" {
			return d, true
		}
	}
	return nil, false
}
