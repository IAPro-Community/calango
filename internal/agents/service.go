// Package agents manages agent configuration, monthly budgets and the node-wide pause.
package agents

import (
	"context"
	"errors"
	"fmt"
	"time"

	"calango/internal/core"
)

// Repository is the persistence the service needs. The SQLite store implements it.
type Repository interface {
	InsertAgent(ctx context.Context, a core.Agent) error
	GetAgent(ctx context.Context, ownerID, id string) (core.Agent, error)
	ListAgents(ctx context.Context, ownerID string) ([]core.Agent, error)
	ReplaceAgent(ctx context.Context, a core.Agent) error
	DeleteAgent(ctx context.Context, ownerID, id string) error
	GlobalPaused(ctx context.Context) (bool, error)
	SetGlobalPaused(ctx context.Context, paused bool, at time.Time) error
	MonthlyTokens(ctx context.Context, agentID string, from, to time.Time) (int64, error)
}

// View is an agent as clients see it: configuration plus live state.
type View struct {
	core.Agent
	// Paused is true while the node-wide pause is on. Per-agent pause is not in V1.
	Paused bool
	// TokensUsedMonth is the agent's spend in the current UTC month.
	TokensUsedMonth int64
}

// Service applies the agent rules on top of a Repository.
type Service struct {
	repo  Repository
	now   func() time.Time
	newID func(time.Time) (string, error)
}

// NewService builds a Service. now and newID are injected so tests control time and ids.
func NewService(repo Repository, now func() time.Time, newID func(time.Time) (string, error)) (*Service, error) {
	if repo == nil || now == nil || newID == nil {
		return nil, errors.New("agents service needs a repository, a clock and an id generator")
	}
	return &Service{repo: repo, now: now, newID: newID}, nil
}

// Create validates and stores a new agent owned by ownerID.
func (s *Service) Create(ctx context.Context, ownerID string, in core.Agent) (View, error) {
	now := s.now()
	a := in
	a.OwnerID = ownerID
	a.Normalize()
	if err := a.Validate(); err != nil {
		return View{}, err
	}
	id, err := s.newID(now)
	if err != nil {
		return View{}, fmt.Errorf("generate agent id: %w", err)
	}
	a.ID = id
	a.CreatedAt = now
	a.UpdatedAt = now
	if err := s.repo.InsertAgent(ctx, a); err != nil {
		return View{}, err
	}
	return s.view(ctx, a)
}

// Get returns one agent owned by ownerID.
func (s *Service) Get(ctx context.Context, ownerID, id string) (View, error) {
	a, err := s.repo.GetAgent(ctx, ownerID, id)
	if err != nil {
		return View{}, err
	}
	return s.view(ctx, a)
}

// List returns the owner's agents, oldest first.
func (s *Service) List(ctx context.Context, ownerID string) ([]View, error) {
	agents, err := s.repo.ListAgents(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	views := make([]View, 0, len(agents))
	for _, a := range agents {
		v, err := s.view(ctx, a)
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, nil
}

// Replace overwrites the editable fields of an existing agent. The next run uses the new
// configuration; a run already in progress keeps the one it started with.
func (s *Service) Replace(ctx context.Context, ownerID, id string, in core.Agent) (View, error) {
	current, err := s.repo.GetAgent(ctx, ownerID, id)
	if err != nil {
		return View{}, err
	}
	updated := current
	updated.Name = in.Name
	updated.Role = in.Role
	updated.Instructions = in.Instructions
	updated.Harness = in.Harness
	updated.Model = in.Model
	updated.ReasoningEffort = in.ReasoningEffort
	updated.RunOn = in.RunOn
	updated.TokenLimitMonth = in.TokenLimitMonth
	updated.ApprovalLevel = in.ApprovalLevel
	updated.Normalize()
	if err := updated.Validate(); err != nil {
		return View{}, err
	}
	updated.UpdatedAt = s.now()
	if err := s.repo.ReplaceAgent(ctx, updated); err != nil {
		return View{}, err
	}
	return s.view(ctx, updated)
}

// Delete removes an agent without history. Agents with runs or usage are refused with core.ErrConflict.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.repo.DeleteAgent(ctx, ownerID, id)
}

// PauseAll turns on the node-wide pause: no agent starts a new run until ResumeAll.
func (s *Service) PauseAll(ctx context.Context) error {
	return s.repo.SetGlobalPaused(ctx, true, s.now())
}

// ResumeAll turns the node-wide pause off.
func (s *Service) ResumeAll(ctx context.Context) error {
	return s.repo.SetGlobalPaused(ctx, false, s.now())
}

func (s *Service) view(ctx context.Context, a core.Agent) (View, error) {
	paused, err := s.repo.GlobalPaused(ctx)
	if err != nil {
		return View{}, err
	}
	from, to := currentMonth(s.now())
	used, err := s.repo.MonthlyTokens(ctx, a.ID, from, to)
	if err != nil {
		return View{}, fmt.Errorf("read monthly usage for agent %s: %w", a.ID, err)
	}
	return View{Agent: a, Paused: paused, TokensUsedMonth: used}, nil
}

// currentMonth returns the UTC window [first day, first day of next month) containing t.
func currentMonth(t time.Time) (time.Time, time.Time) {
	t = t.UTC()
	from := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return from, from.AddDate(0, 1, 0)
}
