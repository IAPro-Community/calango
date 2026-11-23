// Package devices pairs clients to the node and authenticates their tokens. A
// pairing code is shown once, works once and expires after core.PairingCodeTTL. The
// device token is returned once and only its digest is stored. Revoking a device makes
// its next call fail. Every change is audited.
package devices

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"calango/internal/auth"
	"calango/internal/core"
)

// Repository is the persistence the service needs. *store.Store implements it.
type Repository interface {
	InsertPairingCode(ctx context.Context, digest string, pc core.PairingCode, ev core.AuditEvent) error
	PairDevice(ctx context.Context, digest string, at time.Time, d core.Device, ev core.AuditEvent) (core.Device, error)
	DeviceByDigest(ctx context.Context, digest string) (core.Device, error)
	TouchDevice(ctx context.Context, id string, at time.Time) error
	ListDevices(ctx context.Context, ownerID string) ([]core.Device, error)
	RevokeDevice(ctx context.Context, ownerID, id string, at time.Time, ev core.AuditEvent) error
}

// Service issues pairing codes, pairs devices and authenticates their tokens.
type Service struct {
	repo  Repository
	now   func() time.Time
	newID func(time.Time) (string, error)
}

// NewService builds the service. now and newID are injected so time and ids are testable.
func NewService(repo Repository, now func() time.Time, newID func(time.Time) (string, error)) (*Service, error) {
	if repo == nil || now == nil || newID == nil {
		return nil, errors.New("devices service needs a repository, a clock and an id generator")
	}
	return &Service{repo: repo, now: now, newID: newID}, nil
}

// IssuedCode is a pairing code. Code is returned to the caller once and never stored.
type IssuedCode struct {
	Code      string
	Scope     core.DeviceScope
	ExpiresAt time.Time
}

// CreatePairingCode issues a single-use code for a device of the given scope. An empty
// scope means full. actorID is the device that asked, recorded in the audit log.
func (s *Service) CreatePairingCode(ctx context.Context, ownerID, actorID string, scope core.DeviceScope) (IssuedCode, error) {
	if scope == "" {
		scope = core.ScopeFull
	}
	if err := scope.Validate(); err != nil {
		return IssuedCode{}, err
	}
	code, err := core.NewPairingCode()
	if err != nil {
		return IssuedCode{}, err
	}
	at := s.now()
	pc := core.PairingCode{OwnerID: ownerID, Scope: scope, ExpiresAt: at.Add(core.PairingCodeTTL)}
	ev, err := s.audit(at, actorID, "pairing_code.created", "pairing_code", map[string]any{"scope": string(scope)})
	if err != nil {
		return IssuedCode{}, err
	}
	if err := s.repo.InsertPairingCode(ctx, auth.Digest(code), pc, ev); err != nil {
		return IssuedCode{}, fmt.Errorf("store pairing code: %w", err)
	}
	return IssuedCode{Code: code, Scope: scope, ExpiresAt: pc.ExpiresAt}, nil
}

// Paired is the result of a successful pairing. Token is returned once.
type Paired struct {
	DeviceID string
	Token    string
	Scope    core.DeviceScope
}

// Pair exchanges a pairing code for a device token. Input is checked before the code
// is consumed, so a mistyped name or platform does not burn the code.
func (s *Service) Pair(ctx context.Context, code, name, platform string) (Paired, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > core.MaxDeviceNameLength {
		return Paired{}, fmt.Errorf("%w: device_name must have 1 to %d characters", core.ErrInvalidInput, core.MaxDeviceNameLength)
	}
	if err := core.ValidatePlatform(platform); err != nil {
		return Paired{}, err
	}
	normalized := core.NormalizePairingCode(code)
	if !core.ValidPairingCode(normalized) {
		return Paired{}, fmt.Errorf("%w: pairing code is invalid, used or expired", core.ErrUnauthorized)
	}

	token, err := auth.NewDeviceToken()
	if err != nil {
		return Paired{}, err
	}
	at := s.now()
	id, err := s.newID(at)
	if err != nil {
		return Paired{}, fmt.Errorf("devices: device id: %w", err)
	}
	ev, err := s.audit(at, id, "device.paired", id, map[string]any{"platform": platform})
	if err != nil {
		return Paired{}, err
	}
	d := core.Device{
		ID:          id,
		Name:        name,
		Platform:    platform,
		TokenDigest: auth.Digest(token),
		CreatedAt:   at,
	}
	stored, err := s.repo.PairDevice(ctx, auth.Digest(normalized), at, d, ev)
	if err != nil {
		return Paired{}, fmt.Errorf("pair device: %w", err)
	}
	return Paired{DeviceID: stored.ID, Token: token, Scope: stored.Scope}, nil
}

// Authenticate returns the active device that owns token and records the access. An
// empty, unknown or revoked token is core.ErrUnauthorized.
func (s *Service) Authenticate(ctx context.Context, token string) (core.Device, error) {
	if token == "" {
		return core.Device{}, fmt.Errorf("%w: missing token", core.ErrUnauthorized)
	}
	d, err := s.repo.DeviceByDigest(ctx, auth.Digest(token))
	if errors.Is(err, core.ErrNotFound) {
		return core.Device{}, fmt.Errorf("%w: unknown token", core.ErrUnauthorized)
	}
	if err != nil {
		return core.Device{}, fmt.Errorf("authenticate: %w", err)
	}
	if !d.Active() {
		return core.Device{}, fmt.Errorf("%w: device revoked", core.ErrUnauthorized)
	}
	at := s.now()
	if err := s.repo.TouchDevice(ctx, d.ID, at); err != nil {
		return core.Device{}, err
	}
	d.LastSeenAt = at
	return d, nil
}

// List returns the owner's devices, revoked ones included.
func (s *Service) List(ctx context.Context, ownerID string) ([]core.Device, error) {
	return s.repo.ListDevices(ctx, ownerID)
}

// Revoke disables a device of ownerID. actorID is the device that asked. Revoking a
// device twice is not an error; an unknown device is core.ErrNotFound.
func (s *Service) Revoke(ctx context.Context, ownerID, actorID, deviceID string) error {
	at := s.now()
	ev, err := s.audit(at, actorID, "device.revoked", deviceID, nil)
	if err != nil {
		return err
	}
	return s.repo.RevokeDevice(ctx, ownerID, deviceID, at, ev)
}

func (s *Service) audit(at time.Time, actorID, action, subject string, facts map[string]any) (core.AuditEvent, error) {
	id, err := s.newID(at)
	if err != nil {
		return core.AuditEvent{}, fmt.Errorf("devices: audit id: %w", err)
	}
	return core.AuditEvent{
		ID:        id,
		At:        at,
		ActorKind: "device",
		ActorID:   actorID,
		Action:    action,
		Subject:   subject,
		Facts:     facts,
	}, nil
}
