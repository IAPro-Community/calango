package api

import (
	"net"
	"net/http"
	"sync"
	"time"

	"calango/internal/core"
)

const (
	// pairAttemptsPerWindow bounds POST /v1/pair attempts per origin, valid or not.
	pairAttemptsPerWindow = 5
	pairWindow            = time.Minute
	// limiterSweepAt is how many origins may be tracked before idle ones are dropped.
	limiterSweepAt = 1024
)

// attemptLimiter is a sliding-window counter per origin. Origins are the remote host
// of the connection; a reverse proxy in front of the node makes them all one origin.
type attemptLimiter struct {
	mu     sync.Mutex
	now    func() time.Time
	max    int
	window time.Duration
	hits   map[string][]time.Time
}

func newAttemptLimiter(max int, window time.Duration, now func() time.Time) *attemptLimiter {
	return &attemptLimiter{now: now, max: max, window: window, hits: make(map[string][]time.Time)}
}

// allow records an attempt from origin and reports whether it is within the limit.
// Rejected attempts are not recorded, so the window reopens as the oldest one ages out.
func (l *attemptLimiter) allow(origin string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if len(l.hits) > limiterSweepAt {
		for o := range l.hits {
			l.trim(o, now)
		}
	}
	recent := l.trim(origin, now)
	if len(recent) >= l.max {
		return false
	}
	l.hits[origin] = append(recent, now)
	return true
}

// trim keeps the attempts of origin inside the window, drops the origin when none
// remain, and returns the kept attempts.
func (l *attemptLimiter) trim(origin string, now time.Time) []time.Time {
	cutoff := now.Add(-l.window)
	var kept []time.Time
	for _, t := range l.hits[origin] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.hits, origin)
	} else {
		l.hits[origin] = kept
	}
	return kept
}

func originOf(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// pairRequest is the body of POST /v1/pair.
type pairRequest struct {
	Code       string `json:"code"`
	DeviceName string `json:"device_name"`
	Platform   string `json:"platform"`
}

// pairResponse is returned once; the token cannot be read again.
type pairResponse struct {
	DeviceID string `json:"device_id"`
	Token    string `json:"token"`
	Scope    string `json:"scope"`
}

// pairingCodeRequest is the optional body of POST /v1/pairing-codes.
type pairingCodeRequest struct {
	Scope core.DeviceScope `json:"scope"`
}

// pairingCodeResponse is the PairingCode schema.
type pairingCodeResponse struct {
	Code      string `json:"code"`
	Scope     string `json:"scope"`
	ExpiresAt string `json:"expires_at"`
}

// deviceResponse is the Device schema. Token digests never leave the node.
type deviceResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Platform   string `json:"platform"`
	Scope      string `json:"scope"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at,omitempty"`
	RevokedAt  string `json:"revoked_at,omitempty"`
}

func deviceResponseOf(d core.Device) deviceResponse {
	return deviceResponse{
		ID:         d.ID,
		Name:       d.Name,
		Platform:   d.Platform,
		Scope:      string(d.Scope),
		CreatedAt:  formatTime(d.CreatedAt),
		LastSeenAt: formatTime(d.LastSeenAt),
		RevokedAt:  formatTime(d.RevokedAt),
	}
}

// formatTime renders an RFC 3339 UTC time, or "" for the zero time.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (h *handler) registerDevices(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/pairing-codes", h.need(core.PermWrite, h.createPairingCode))
	mux.HandleFunc("GET /v1/devices", h.need(core.PermWrite, h.listDevices))
	mux.HandleFunc("DELETE /v1/devices/{id}", h.need(core.PermWrite, h.revokeDevice))
}

// pair is public: it is the only way a new device gets a token. Every attempt counts
// against the origin's limit, valid or not.
func (h *handler) pair(w http.ResponseWriter, r *http.Request) {
	if !h.pairLimit.allow(originOf(r)) {
		w.Header().Set("Retry-After", "60")
		writeError(w, http.StatusTooManyRequests, codeRateLimited, "too many pairing attempts; wait a minute")
		return
	}
	var body pairRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	paired, err := h.devices.Pair(r.Context(), body.Code, body.DeviceName, body.Platform)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pairResponse{
		DeviceID: paired.DeviceID,
		Token:    paired.Token,
		Scope:    string(paired.Scope),
	})
}

func (h *handler) createPairingCode(w http.ResponseWriter, r *http.Request) {
	var body pairingCodeRequest
	if err := decodeOptionalJSON(w, r, &body); err != nil {
		writeServiceError(w, err)
		return
	}
	issued, err := h.devices.CreatePairingCode(r.Context(), h.ownerID, principalOf(r.Context()).deviceID, body.Scope)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pairingCodeResponse{
		Code:      issued.Code,
		Scope:     string(issued.Scope),
		ExpiresAt: formatTime(issued.ExpiresAt),
	})
}

func (h *handler) listDevices(w http.ResponseWriter, r *http.Request) {
	list, err := h.devices.List(r.Context(), h.ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]deviceResponse, 0, len(list))
	for _, d := range list {
		out = append(out, deviceResponseOf(d))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) revokeDevice(w http.ResponseWriter, r *http.Request) {
	err := h.devices.Revoke(r.Context(), h.ownerID, principalOf(r.Context()).deviceID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
