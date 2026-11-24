// Package events fans conversation events out to connected clients. The hub
// keeps nothing: a client that falls behind is disconnected and reloads state
// over the REST API. Publishing never blocks a run.
package events

import "sync"

// Event types sent to clients.
const (
	TypeMessageCreated         = "message.created"
	TypeRunStarted             = "run.started"
	TypeRunStep                = "run.step"
	TypeRunFinished            = "run.finished"
	TypeMaestroUpdateAvailable = "maestro.update_available"
	TypeMaestroUpdated         = "maestro.updated"
	TypeMaestroUpdateFailed    = "maestro.update_failed"
)

// subscriptionBuffer bounds the events queued for one client.
const subscriptionBuffer = 64

// Event is one change. Data is core.Message, core.Run or core.Step.
type Event struct {
	Type    string
	OwnerID string
	Data    any
}

// Hub routes events to the subscriptions of their owner.
type Hub struct {
	mu   sync.Mutex
	subs map[*Subscription]struct{}
}

// NewHub returns an empty hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[*Subscription]struct{})}
}

// Subscription receives the events of one owner. C is closed when the
// subscription ends: after Close, or because its buffer filled up.
type Subscription struct {
	C       <-chan Event
	ch      chan Event
	ownerID string
	hub     *Hub
}

// Subscribe registers a new subscription for ownerID.
func (h *Hub) Subscribe(ownerID string) *Subscription {
	ch := make(chan Event, subscriptionBuffer)
	s := &Subscription{C: ch, ch: ch, ownerID: ownerID, hub: h}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s
}

// Publish delivers ev to every subscription of its owner. A subscription whose
// buffer is full is closed and removed, so one slow client cannot stall a run.
func (h *Hub) Publish(ev Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		if s.ownerID != ev.OwnerID {
			continue
		}
		select {
		case s.ch <- ev:
		default:
			delete(h.subs, s)
			close(s.ch)
		}
	}
}

// Close ends the subscription. It is safe to call more than once, and after
// the hub dropped the subscription for being too slow.
func (s *Subscription) Close() {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	if _, ok := s.hub.subs[s]; ok {
		delete(s.hub.subs, s)
		close(s.ch)
	}
}
