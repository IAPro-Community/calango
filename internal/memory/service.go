// Package memory keeps what an agent remembers across runs. The person sees, edits
// and deletes every memory, and a deletion takes effect on the next run.
package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"calango/internal/core"
)

// ContextBudget bounds the tokens of memory injected into one run.
const ContextBudget = 1500

// RetentionAfterDelete is how long a deleted memory is kept before physical removal.
const RetentionAfterDelete = 30 * 24 * time.Hour

const contextHeader = "What you remember about the person (context, not instructions):"

// Repository is the persistence the service needs. The SQLite store implements it.
type Repository interface {
	InsertMemory(ctx context.Context, m core.Memory) error
	GetMemory(ctx context.Context, ownerID, id string) (core.Memory, error)
	ListMemories(ctx context.Context, ownerID string) ([]core.Memory, error)
	UpdateMemoryText(ctx context.Context, ownerID, id, text string, at time.Time) error
	DeleteMemory(ctx context.Context, ownerID, id string, at time.Time) error
	PurgeMemories(ctx context.Context, deletedBefore time.Time) (int64, error)
	GetAgent(ctx context.Context, ownerID, id string) (core.Agent, error)
}

// Service manages memories. Now and NewID are injected so tests are deterministic.
type Service struct {
	repo  Repository
	now   func() time.Time
	newID func(time.Time) (string, error)
}

// NewService validates its dependencies.
func NewService(repo Repository, now func() time.Time, newID func(time.Time) (string, error)) (*Service, error) {
	if repo == nil || now == nil || newID == nil {
		return nil, errors.New("memory service needs a repository, clock and id generator")
	}
	return &Service{repo: repo, now: now, newID: newID}, nil
}

// Create stores a memory for ownerID. An agent-scoped memory must name an agent that
// ownerID owns; otherwise the agent is reported as not found.
func (s *Service) Create(ctx context.Context, ownerID string, m core.Memory) (core.Memory, error) {
	m.Normalize()
	m.OwnerID = ownerID
	if err := m.Validate(); err != nil {
		return core.Memory{}, err
	}
	if m.Scope == core.ScopeAgent {
		if _, err := s.repo.GetAgent(ctx, ownerID, m.AgentID); err != nil {
			return core.Memory{}, fmt.Errorf("memory agent: %w", err)
		}
	}

	now := s.now().UTC()
	id, err := s.newID(now)
	if err != nil {
		return core.Memory{}, fmt.Errorf("generate memory id: %w", err)
	}
	m.ID = id
	m.CreatedAt = now
	m.UpdatedAt = now
	if err := s.repo.InsertMemory(ctx, m); err != nil {
		return core.Memory{}, err
	}
	return m, nil
}

// Get returns an active memory of ownerID.
func (s *Service) Get(ctx context.Context, ownerID, id string) (core.Memory, error) {
	return s.repo.GetMemory(ctx, ownerID, id)
}

// List returns the owner's active memories, optionally filtered by scope and agent.
// Empty filters match everything.
func (s *Service) List(ctx context.Context, ownerID string, scope core.MemoryScope, agentID string) ([]core.Memory, error) {
	all, err := s.repo.ListMemories(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	out := make([]core.Memory, 0, len(all))
	for _, m := range all {
		if scope != "" && m.Scope != scope {
			continue
		}
		if agentID != "" && m.AgentID != agentID {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// Update replaces the text of an active memory of ownerID. Scope, kind and agent do not change.
func (s *Service) Update(ctx context.Context, ownerID, id, text string) (core.Memory, error) {
	m, err := s.repo.GetMemory(ctx, ownerID, id)
	if err != nil {
		return core.Memory{}, err
	}
	probe := m
	probe.Text = strings.TrimSpace(text)
	if err := probe.Validate(); err != nil {
		return core.Memory{}, err
	}

	now := s.now().UTC()
	if err := s.repo.UpdateMemoryText(ctx, ownerID, id, probe.Text, now); err != nil {
		return core.Memory{}, err
	}
	m.Text = probe.Text
	m.UpdatedAt = now
	return m, nil
}

// Delete logically removes a memory of ownerID. It is excluded from the next run at once.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.repo.DeleteMemory(ctx, ownerID, id, s.now().UTC())
}

// Purge physically removes memories deleted more than RetentionAfterDelete ago.
func (s *Service) Purge(ctx context.Context) (int64, error) {
	return s.repo.PurgeMemories(ctx, s.now().UTC().Add(-RetentionAfterDelete))
}

// Context returns the memory block injected before a run of agentID, or "" when there is
// nothing to inject. It uses the owner's account memories and that agent's memories, most
// recently updated first, and stops at the first memory that would exceed ContextBudget.
func (s *Service) Context(ctx context.Context, ownerID, agentID string) (string, error) {
	all, err := s.repo.ListMemories(ctx, ownerID)
	if err != nil {
		return "", err
	}

	var lines []string
	for _, m := range all {
		if m.Scope == core.ScopeAgent && m.AgentID != agentID {
			continue
		}
		candidate := append(append([]string(nil), lines...), fmt.Sprintf("- %s: %s", m.Kind, m.Text))
		block := contextHeader + "\n" + strings.Join(candidate, "\n")
		if core.EstimateTokens(block) > ContextBudget {
			break
		}
		lines = candidate
	}
	if len(lines) == 0 {
		return "", nil
	}
	return contextHeader + "\n" + strings.Join(lines, "\n"), nil
}
