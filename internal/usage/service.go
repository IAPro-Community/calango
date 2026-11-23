// Package usage reports what the node spent: tokens and cost grouped by one
// dimension. Cost is never invented. Rows with unknown cost add nothing to the sum
// and are counted apart, so a zero total is never mistaken for a free call.
package usage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"calango/internal/core"
)

// Group is the dimension a summary is grouped by.
type Group string

// Groups accepted by the summary. The store maps each one to a fixed column.
const (
	GroupAgent   Group = "agent"
	GroupRoutine Group = "routine"
	GroupProject Group = "project"
	GroupNode    Group = "node"
	GroupModel   Group = "model"
)

// Query selects usage rows. From is inclusive and To is exclusive; a zero time leaves
// that side unbounded. An empty GroupBy means GroupAgent.
type Query struct {
	From    time.Time
	To      time.Time
	GroupBy Group
}

// Row is the total of one group. Key is empty when the row has no value for the group.
type Row struct {
	Key             string
	TokensIn        int64
	TokensOut       int64
	CostMicros      int64
	UnknownCostRows int64
}

// Repository is the persistence the service needs. The SQLite store implements it.
type Repository interface {
	UsageTotals(ctx context.Context, ownerID string, q Query) ([]Row, error)
}

// Service answers usage queries for one owner.
type Service struct {
	repo Repository
}

// NewService validates its dependency.
func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, errors.New("usage service needs a repository")
	}
	return &Service{repo: repo}, nil
}

// Summary returns one row per group value, ordered by key.
func (s *Service) Summary(ctx context.Context, ownerID string, q Query) ([]Row, error) {
	if q.GroupBy == "" {
		q.GroupBy = GroupAgent
	}
	switch q.GroupBy {
	case GroupAgent, GroupRoutine, GroupProject, GroupNode, GroupModel:
	default:
		return nil, fmt.Errorf("%w: group_by must be agent, routine, project, node or model", core.ErrInvalidInput)
	}
	if !q.From.IsZero() && !q.To.IsZero() && !q.To.After(q.From) {
		return nil, fmt.Errorf("%w: to must be after from", core.ErrInvalidInput)
	}
	rows, err := s.repo.UsageTotals(ctx, ownerID, q)
	if err != nil {
		return nil, fmt.Errorf("usage summary: %w", err)
	}
	return rows, nil
}
