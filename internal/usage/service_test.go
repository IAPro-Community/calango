package usage

import (
	"context"
	"errors"
	"testing"
	"time"

	"calango/internal/core"
)

type fakeRepo struct {
	owner string
	query Query
	calls int
}

func (f *fakeRepo) UsageTotals(_ context.Context, owner string, q Query) ([]Row, error) {
	f.owner, f.query = owner, q
	f.calls++
	return nil, nil
}

func TestSummaryDefaultsToAgentGroupAndPassesOwner(t *testing.T) {
	repo := &fakeRepo{}
	svc, err := NewService(repo)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := svc.Summary(context.Background(), "owner-1", Query{}); err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if repo.owner != "owner-1" || repo.query.GroupBy != GroupAgent {
		t.Fatalf("repository got (owner=%q, group=%q), want (owner-1, agent)", repo.owner, repo.query.GroupBy)
	}
}

func TestSummaryRejectsBadQueriesBeforeReadingUsage(t *testing.T) {
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	cases := map[string]Query{
		"unknown group":    {GroupBy: Group("provider")},
		"to before from":   {From: base, To: base.Add(-time.Hour)},
		"to equal to from": {From: base, To: base},
	}
	for name, q := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepo{}
			svc, err := NewService(repo)
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}
			if _, err := svc.Summary(context.Background(), "owner-1", q); !errors.Is(err, core.ErrInvalidInput) {
				t.Fatalf("Summary() error = %v, want ErrInvalidInput", err)
			}
			if repo.calls != 0 {
				t.Fatalf("repository called %d times for an invalid query", repo.calls)
			}
		})
	}
}

func TestNewServiceRequiresRepository(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) returned no error")
	}
}
