package license

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

// State is where a node's license stands at one instant.
type State string

const (
	// StateNone: no license is installed. The node runs Free.
	StateNone State = "none"
	// StateActive: the signature is valid and exp has not passed.
	StateActive State = "active"
	// StateGrace: exp has passed but the tolerance has not; the plan still applies.
	StateGrace State = "grace"
	// StateExpired: the tolerance has passed. The node runs Free and keeps its data.
	StateExpired State = "expired"
	// StateInvalid: the license cannot be trusted. The node runs Free and keeps its data.
	StateInvalid State = "invalid"
)

// Reasons for StateInvalid.
const (
	ReasonMalformed   = "malformed"
	ReasonSignature   = "signature"
	ReasonClaims      = "claims"
	ReasonNoPublicKey = "no_public_key"
	ReasonUnreadable  = "unreadable"
)

// Status is the evaluated license. Plan is the effective plan: the licensed plan while
// active or in grace, Free otherwise.
type Status struct {
	State State
	Plan  string
	// Reason explains StateInvalid; empty otherwise.
	Reason string
	// Claims is set whenever the signature verified, including after expiry, so the
	// interface can say which license lapsed.
	Claims    *Claims
	CheckedAt time.Time
}

// Paid reports whether the effective plan is above Free.
func (s Status) Paid() bool { return s.Plan != PlanFree }

// Evaluate verifies token and places it in time. An empty token is StateNone. A nil
// verifier means no public key is configured, so any token is invalid.
func Evaluate(v *Verifier, token string, now time.Time) Status {
	st := Status{State: StateNone, Plan: PlanFree, CheckedAt: now.UTC()}
	if isBlank(token) {
		return st
	}
	if v == nil {
		st.State, st.Reason = StateInvalid, ReasonNoPublicKey
		return st
	}
	c, err := v.Verify(token)
	if err != nil {
		st.State, st.Reason = StateInvalid, reasonOf(err)
		return st
	}
	st.Claims = &c
	switch {
	case now.Before(c.Expires()):
		st.State, st.Plan = StateActive, c.Plan
	case now.Before(c.GraceEnds()):
		st.State, st.Plan = StateGrace, c.Plan
	default:
		st.State = StateExpired
	}
	return st
}

func reasonOf(err error) string {
	switch {
	case errors.Is(err, ErrSignature):
		return ReasonSignature
	case errors.Is(err, ErrClaims):
		return ReasonClaims
	default:
		return ReasonMalformed
	}
}

func isBlank(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

// Manager reads the license file and evaluates it on demand. It only reads: it never
// writes, renames or removes the file or anything else in the data directory, so an
// expired or invalid license can only lower the plan.
type Manager struct {
	path     string
	verifier *Verifier
	now      func() time.Time
}

// NewManager returns a manager for the license at path. A nil key is allowed: every
// installed license is then invalid and the node runs Free.
func NewManager(path string, key ed25519.PublicKey, now func() time.Time) (*Manager, error) {
	if path == "" {
		return nil, errors.New("license: file path is required")
	}
	if now == nil {
		return nil, errors.New("license: clock is required")
	}
	m := &Manager{path: path, now: now}
	if key != nil {
		v, err := NewVerifier(key)
		if err != nil {
			return nil, err
		}
		m.verifier = v
	}
	return m, nil
}

// Path returns the license file location.
func (m *Manager) Path() string { return m.path }

// Status reads the file and evaluates it at the manager's clock. The file is read on
// every call, so a renewed license applies without a restart. A missing file is
// StateNone; an unreadable one is StateInvalid. It never returns an error: any failure
// lowers the plan to Free.
func (m *Manager) Status() Status {
	now := m.now()
	token, err := readToken(m.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Evaluate(m.verifier, "", now)
		}
		return Status{State: StateInvalid, Plan: PlanFree, Reason: ReasonUnreadable, CheckedAt: now.UTC()}
	}
	return Evaluate(m.verifier, token, now)
}

func readToken(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	// One byte past the limit is enough for Verify to refuse the token as malformed.
	data, err := io.ReadAll(io.LimitReader(f, MaxTokenBytes+1))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// embeddedPublicKey is the license service's public key in standard base64. Release
// builds set it with -ldflags "-X calango/internal/license.embeddedPublicKey=<key>".
// Development builds leave it empty and may configure a key instead.
var embeddedPublicKey = ""

// Key sources reported by ResolvePublicKey.
const (
	KeySourceEmbedded   = "embedded"
	KeySourceConfigured = "configured"
	KeySourceNone       = "none"
)

// ResolvePublicKey picks the verification key. The embedded key always wins, so a
// release build cannot be pointed at another issuer by configuration; configured is
// used only when nothing is embedded. Both empty returns a nil key and KeySourceNone.
func ResolvePublicKey(configured string) (ed25519.PublicKey, string, error) {
	if embeddedPublicKey != "" {
		key, err := ParsePublicKey(embeddedPublicKey)
		if err != nil {
			return nil, "", fmt.Errorf("embedded license key: %w", err)
		}
		return key, KeySourceEmbedded, nil
	}
	if configured == "" {
		return nil, KeySourceNone, nil
	}
	key, err := ParsePublicKey(configured)
	if err != nil {
		return nil, "", err
	}
	return key, KeySourceConfigured, nil
}
