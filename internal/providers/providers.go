// Package providers manages provider accounts: named logins to Codex or
// Anthropic (ADR 0016). It builds the gateway provider for an account when a
// call needs it. The service stores no secret: an Anthropic key stays in the
// vault and a Codex login stays in the account's own CODEX_HOME.
package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"calango/internal/anthropic"
	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/harness"

	"github.com/IAPro-Community/IAPro-SDK/lease"
	sdk "github.com/IAPro-Community/IAPro-SDK/providers"
	sdkspec "github.com/IAPro-Community/IAPro-SDK/providers/spec"
)

// Repository is the persistence the service needs. The SQLite store implements it.
type Repository interface {
	InsertProviderAccount(ctx context.Context, a core.ProviderAccount) error
	ListProviderAccounts(ctx context.Context, ownerID string) ([]core.ProviderAccount, error)
	GetProviderAccountByName(ctx context.Context, ownerID, kind, label string) (core.ProviderAccount, error)
	DeleteProviderAccount(ctx context.Context, ownerID, id string) error
}

// Config wires a Service.
type Config struct {
	Repo    Repository
	Secrets anthropic.Secrets
	// CodexBin is the Codex CLI binary; empty means "codex".
	CodexBin string
	// CodexWorkDir is the working root of every Codex run. It must exist.
	CodexWorkDir string
	// AccountsDir holds one CODEX_HOME per non-default Codex account.
	AccountsDir string
	// AnthropicBaseURL is empty for the production API.
	AnthropicBaseURL string
	Now              func() time.Time
	NewID            func(time.Time) (string, error)
	CapacityInterval time.Duration
	CapacityOwner    string
	OnCapacity       func(string, CapacityView)
	RoutingAllowed   bool
	LeaseStore       lease.Store
	Prices           gateway.PriceTable
}

// Service manages accounts and resolves them to gateway providers.
type Service struct {
	cfg      Config
	capacity *capacityMonitor
	leases   *lease.Coordinator
	manager  *sdk.AccountManager[providerFactory]
}

type providerFactory func(context.Context, core.ProviderAccount) (gateway.Provider, error)

// NewService validates cfg and returns a Service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Repo == nil || cfg.Now == nil || cfg.NewID == nil {
		return nil, errors.New("providers service needs a repository, a clock and an id generator")
	}
	if cfg.Secrets == nil {
		return nil, errors.New("providers service needs a secrets source")
	}
	if cfg.CodexWorkDir == "" || cfg.AccountsDir == "" {
		return nil, errors.New("providers service needs the codex work directory and the accounts directory")
	}
	s := &Service{cfg: cfg, manager: sdk.NewAccountManager[providerFactory]()}
	if err := s.manager.Register(core.ProviderKindCodex, s.resolveCodex); err != nil {
		return nil, err
	}
	if err := s.manager.Register(core.ProviderKindAnthropic, s.resolveAnthropic); err != nil {
		return nil, err
	}
	if cfg.CapacityInterval > 0 && cfg.CapacityOwner != "" {
		s.capacity = newCapacityMonitor(s)
	}
	if cfg.RoutingAllowed {
		if s.capacity == nil {
			return nil, errors.New("capacity routing requires a configured monitor")
		}
		var err error
		s.leases, err = lease.New(cfg.LeaseStore, cfg.Now)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// View is an account with the step that finishes its sign-in. Setup names the
// step only; it never contains a secret.
type View struct {
	core.ProviderAccount
	// Key is the gateway name the agent chain refers to: "<kind>/<label>".
	Key        string
	Setup      string
	SecretItem string
	Capacity   CapacityView
}

// Create stores a new account. A Codex account gets its own home directory.
func (s *Service) Create(ctx context.Context, ownerID, kind, label string) (View, error) {
	// Preserve the HTTP/domain invalid-input contract and explicit host allowlist.
	a := core.ProviderAccount{OwnerID: ownerID, Kind: kind, Label: label}
	if err := a.Validate(); err != nil {
		return View{}, err
	}
	req := sdk.ProfileRegistration{OwnerID: ownerID, ProviderID: kind, Label: label}
	return sdk.RegisterProfile(ctx, req, func(provider string) error {
		if _, ok := s.manager.Lookup(provider); !ok {
			return fmt.Errorf("%w: provider is not enabled", core.ErrInvalidInput)
		}
		return nil
	}, func(label string) error {
		if !sdk.ValidPortableLabel(label) {
			return fmt.Errorf("%w: invalid account label", core.ErrInvalidInput)
		}
		return nil
	}, s.createAccountRecord)
}

func (s *Service) createAccountRecord(ctx context.Context, req sdk.ProfileRegistration) (View, error) {
	now := s.cfg.Now()
	a := core.ProviderAccount{OwnerID: req.OwnerID, Kind: req.ProviderID, Label: req.Label, CreatedAt: now, UpdatedAt: now}
	id, err := s.cfg.NewID(now)
	if err != nil {
		return View{}, fmt.Errorf("generate provider account id: %w", err)
	}
	a.ID = id
	if home := s.codexHome(a); home != "" {
		if err := os.MkdirAll(home, 0o700); err != nil {
			return View{}, fmt.Errorf("create codex home for %s: %w", a.Name(), err)
		}
	}
	if err := s.cfg.Repo.InsertProviderAccount(ctx, a); err != nil {
		return View{}, err
	}
	return s.view(a), nil
}

// List returns the owner's accounts, oldest first.
func (s *Service) List(ctx context.Context, ownerID string) ([]View, error) {
	accounts, err := s.cfg.Repo.ListProviderAccounts(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	views := make([]View, 0, len(accounts))
	for _, a := range accounts {
		views = append(views, s.view(a))
	}
	return views, nil
}

// Delete removes an account and, for Codex, its home directory with the login
// in it. The default Codex home is shared with the CLI and is never removed.
// Accounts used by an agent's chain are refused with core.ErrConflict.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	accounts, err := s.cfg.Repo.ListProviderAccounts(ctx, ownerID)
	if err != nil {
		return err
	}
	var target *core.ProviderAccount
	for i := range accounts {
		if accounts[i].ID == id {
			target = &accounts[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("%w: provider account %s", core.ErrNotFound, id)
	}
	if s.capacity != nil {
		select {
		case s.capacity.gate <- struct{}{}:
			defer func() { <-s.capacity.gate }()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := s.cfg.Repo.DeleteProviderAccount(ctx, ownerID, id); err != nil {
		return err
	}
	if s.capacity != nil {
		s.capacity.forget(*target)
	}
	if home := s.codexHome(*target); home != "" {
		if err := os.RemoveAll(home); err != nil {
			return fmt.Errorf("remove codex home of %s: %w", target.Name(), err)
		}
	}
	return nil
}

// Resolve returns the gateway provider for the owner's account with key
// "<kind>/<label>". The gateway calls it for each request, so a deleted account
// stops resolving at once.
func (s *Service) Resolve(ctx context.Context, ownerID, key string) (gateway.Provider, error) {
	kind, label, err := sdk.ParseAccountKey(key)
	if err != nil {
		return nil, fmt.Errorf("%w: provider key %q", core.ErrInvalidInput, key)
	}
	a, err := s.cfg.Repo.GetProviderAccountByName(ctx, ownerID, kind, label)
	if err != nil {
		return nil, err
	}
	factory, ok := s.manager.Lookup(a.Kind)
	if !ok {
		return nil, fmt.Errorf("providers: unknown kind %q", a.Kind)
	}
	return factory(ctx, a)
}

func (s *Service) resolveCodex(_ context.Context, a core.ProviderAccount) (gateway.Provider, error) {
	p, err := harness.NewCodex(s.cfg.CodexBin, s.cfg.CodexWorkDir, s.codexHome(a))
	if err != nil {
		return nil, err
	}
	if s.capacity != nil && a.OwnerID == s.cfg.CapacityOwner {
		p.CoordinateQuota()
	}
	if s.capacity != nil && a.OwnerID == s.cfg.CapacityOwner && a.Label != core.DefaultAccountLabel {
		return &leasedCodex{service: s, account: a, cli: p}, nil
	}
	return p, nil
}

func (s *Service) resolveAnthropic(_ context.Context, a core.ProviderAccount) (gateway.Provider, error) {
	return anthropic.NewAccount(s.cfg.Secrets, keyItem(a), s.cfg.AnthropicBaseURL)
}

// codexHome returns the CODEX_HOME of a Codex account, or "" for the default
// home that the Codex CLI already uses.
func (s *Service) codexHome(a core.ProviderAccount) string {
	if a.Kind != core.ProviderKindCodex || a.Label == core.DefaultAccountLabel {
		return ""
	}
	return filepath.Join(s.cfg.AccountsDir, core.ProviderKindCodex, a.Label)
}

// keyItem returns the vault item that holds an Anthropic account's key. The
// default account keeps the item name used before accounts existed.
func keyItem(a core.ProviderAccount) string {
	if a.Label == core.DefaultAccountLabel {
		return anthropic.KeyName
	}
	return "anthropic-" + a.Label + "-api-key"
}

func (s *Service) view(a core.ProviderAccount) View {
	v := View{ProviderAccount: a, Key: a.Name(), Setup: s.setup(a), Capacity: s.capacityView(a)}
	if a.Kind == core.ProviderKindAnthropic {
		v.SecretItem = keyItem(a)
	}
	return v
}

// ProviderKind describes a host-enabled registration, not all SDK specs.
// Discovering a CLI does not authorize its installation or execution.
type ProviderKind struct {
	Kind        string `json:"kind"`
	DisplayName string `json:"display_name"`
	AuthKind    string `json:"auth_kind"`
}

func (s *Service) ListProviderKinds() []ProviderKind {
	entries := s.manager.List()
	out := make([]ProviderKind, 0, len(entries))
	for _, entry := range entries {
		if entry.ID == core.ProviderKindAnthropic {
			// Direct API provider, intentionally distinct from the Claude CLI.
			out = append(out, ProviderKind{Kind: entry.ID, DisplayName: "Anthropic", AuthKind: string(sdkspec.AuthKindAPIKey)})
			continue
		}
		if spec, ok := sdkspec.Get(entry.ID); ok && spec.DisplayName != "" && spec.AuthKind != "" {
			out = append(out, ProviderKind{Kind: entry.ID, DisplayName: spec.DisplayName, AuthKind: string(spec.AuthKind)})
		}
	}
	return out
}

func (s *Service) setup(a core.ProviderAccount) string {
	if a.Kind == core.ProviderKindCodex {
		spec := sdkspec.MustGet(core.ProviderKindCodex)
		command := strings.Join(append([]string{"codex"}, spec.LoginCommand()...), " ")
		if home := s.codexHome(a); home != "" {
			return "CODEX_HOME=" + home + " " + command
		}
		return command
	}
	return "PUT /v1/vault/" + keyItem(a) + " with the API key"
}
