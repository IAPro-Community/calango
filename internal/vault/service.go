// Derived from vstorm-co/agenticos (Apache-2.0) at a4b0b3c: backend/app/core/vault.py. Modified.
//
// Named secrets for one owner. Values are sealed before they reach the repository; the
// API lists names and dates only, and Reveal is for in-process callers.

package vault

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"calango/internal/core"
)

// ErrNotFound is the shared domain not-found error, so the API maps it to 404.
var ErrNotFound = core.ErrNotFound

// ErrInvalidName is returned for names outside the vault naming rule. It wraps
// core.ErrInvalidInput, so the API answers 400.
var ErrInvalidName = fmt.Errorf("%w: vault name must be 1-63 characters of a-z, 0-9, '.', '-' or '_', starting with a letter or digit", core.ErrInvalidInput)

// ErrEmptyValue is returned when a secret is empty. An empty credential is always a
// mistake.
var ErrEmptyValue = fmt.Errorf("%w: vault secret cannot be empty", core.ErrInvalidInput)

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,62}$`)

// Item is the metadata the API may return. It never carries the value.
type Item struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Stored is an item with its envelope, as the repository keeps it.
type Stored struct {
	Item
	Ciphertext string
	KeyVersion int
}

// Repo persists sealed items. Implementations return ErrNotFound for unknown names.
type Repo interface {
	// PutItem inserts or replaces the item. created_at is kept on replace; updated_at is at.
	PutItem(ctx context.Context, owner string, item Stored, at time.Time) (Item, error)
	// GetItem returns the stored item, including its envelope.
	GetItem(ctx context.Context, owner, name string) (Stored, error)
	// ListItems returns the metadata of every item, without envelopes.
	ListItems(ctx context.Context, owner string) ([]Item, error)
	// ListStored returns every item with its envelope, for rotation.
	ListStored(ctx context.Context, owner string) ([]Stored, error)
	// ReplaceCiphertext updates the envelope and key version of an existing item. It
	// does not change created_at or updated_at, because rotation is not a user edit.
	ReplaceCiphertext(ctx context.Context, owner, name, ciphertext string, keyVersion int) error
	// DeleteItem removes the item.
	DeleteItem(ctx context.Context, owner, name string) error
}

// Auditor appends one event to the audit chain. Vault events carry names and actors
// only; a value must never be passed in Facts.
type Auditor interface {
	AppendAudit(ctx context.Context, ev core.AuditEvent) error
}

// Service seals and opens secrets for one owner with the master keys it was given.
type Service struct {
	repo    Repo
	keys    MasterKeys
	owner   string
	auditor Auditor
	now     func() time.Time
	newID   func(time.Time) (string, error)
}

// NewService returns a Service. now defaults to time.Now when nil. auditor and newID
// are required: a secret must never change or be revealed without an audit record.
func NewService(repo Repo, keys MasterKeys, owner string, auditor Auditor, now func() time.Time, newID func(time.Time) (string, error)) (*Service, error) {
	if owner == "" {
		return nil, errors.New("vault: owner is required")
	}
	if auditor == nil || newID == nil {
		return nil, errors.New("vault: auditor and id generator are required")
	}
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, keys: keys, owner: owner, auditor: auditor, now: now, newID: newID}, nil
}

// scope binds ciphertexts to this owner.
func (s *Service) scope() string { return "owner:" + s.owner }

// audit records one attempt before the action runs. If the record cannot be written the
// action does not run. An attempt whose action then fails is on record; an action with
// no record is not acceptable.
func (s *Service) audit(ctx context.Context, action, name string) error {
	at := s.now().UTC()
	id, err := s.newID(at)
	if err != nil {
		return fmt.Errorf("vault: new audit id: %w", err)
	}
	ev := core.AuditEvent{
		ID: id, At: at, ActorKind: "owner", ActorID: s.owner,
		Action: action, Subject: "vault:" + name, Facts: map[string]any{"name": name},
	}
	if err := s.auditor.AppendAudit(ctx, ev); err != nil {
		return fmt.Errorf("vault: audit %s %q: %w", action, name, err)
	}
	return nil
}

// Set seals value under name, replacing any existing secret with that name.
func (s *Service) Set(ctx context.Context, name, value string) (Item, error) {
	if !namePattern.MatchString(name) {
		return Item{}, ErrInvalidName
	}
	if value == "" {
		return Item{}, ErrEmptyValue
	}
	if err := s.audit(ctx, "vault.set", name); err != nil {
		return Item{}, err
	}
	version := s.keys.Current()
	ciphertext, err := Seal([]byte(value), s.scope(), s.keys, version)
	if err != nil {
		return Item{}, err
	}
	at := s.now().UTC()
	item, err := s.repo.PutItem(ctx, s.owner, Stored{
		Item:       Item{Name: name, CreatedAt: at, UpdatedAt: at},
		Ciphertext: ciphertext,
		KeyVersion: version,
	}, at)
	if err != nil {
		return Item{}, fmt.Errorf("vault: store %q: %w", name, err)
	}
	return item, nil
}

// Reveal returns the plain value. Only in-process callers that need the credential
// may use it; the value must never reach an API response, a log or a run step.
func (s *Service) Reveal(ctx context.Context, name string) (string, error) {
	if !namePattern.MatchString(name) {
		return "", ErrInvalidName
	}
	if err := s.audit(ctx, "vault.reveal", name); err != nil {
		return "", err
	}
	stored, err := s.repo.GetItem(ctx, s.owner, name)
	if err != nil {
		return "", err
	}
	plaintext, err := Open(stored.Ciphertext, s.scope(), s.keys, stored.KeyVersion)
	if err != nil {
		return "", fmt.Errorf("vault: reveal %q: %w", name, err)
	}
	return string(plaintext), nil
}

// List returns metadata for every secret of the owner.
func (s *Service) List(ctx context.Context) ([]Item, error) {
	return s.repo.ListItems(ctx, s.owner)
}

// Delete removes the named secret. A name outside the naming rule is refused before the
// audit write, so the trail never records arbitrary strings.
func (s *Service) Delete(ctx context.Context, name string) error {
	if !namePattern.MatchString(name) {
		return ErrInvalidName
	}
	if err := s.audit(ctx, "vault.delete", name); err != nil {
		return err
	}
	return s.repo.DeleteItem(ctx, s.owner, name)
}

// Rotate re-wraps every secret that is not under the current master key version. It
// returns how many were re-wrapped. Each re-wrap is audited before it is stored, as
// vault.rewrap with the name only. Payloads are not re-encrypted. It stops at the first
// envelope that cannot be opened, because that row needs an operator.
func (s *Service) Rotate(ctx context.Context) (int, error) {
	stored, err := s.repo.ListStored(ctx, s.owner)
	if err != nil {
		return 0, err
	}
	// Validate every envelope before changing any item. A restored key can
	// have the right version and still be the wrong key. Refuse startup or
	// rotation without modifying the restored database in that case.
	for _, item := range stored {
		plain, err := Open(item.Ciphertext, s.scope(), s.keys, item.KeyVersion)
		if err != nil {
			return 0, fmt.Errorf("vault: verify stored envelope before rotation: %w", err)
		}
		clear(plain)
	}
	current := s.keys.Current()
	rotated := 0
	for _, item := range stored {
		if item.KeyVersion == current {
			continue
		}
		next, err := Rewrap(item.Ciphertext, s.scope(), s.keys, item.KeyVersion, current)
		if err != nil {
			return rotated, fmt.Errorf("vault: rotate %q: %w", item.Name, err)
		}
		if err := s.audit(ctx, "vault.rewrap", item.Name); err != nil {
			return rotated, err
		}
		if err := s.repo.ReplaceCiphertext(ctx, s.owner, item.Name, next, current); err != nil {
			return rotated, fmt.Errorf("vault: store rotated %q: %w", item.Name, err)
		}
		rotated++
	}
	return rotated, nil
}
