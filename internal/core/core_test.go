package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewAgentValidates(t *testing.T) {
	if _, err := NewAgent("", "x", 10); err == nil {
		t.Error("empty id should fail")
	}
	if _, err := NewAgent("a", "", 10); err == nil {
		t.Error("empty name should fail")
	}
	if _, err := NewAgent("a", "x", 0); err == nil {
		t.Error("zero limit should fail")
	}
	if _, err := NewAgent("a", "x", 10); err != nil {
		t.Errorf("valid agent failed: %v", err)
	}
}

func TestChargeRespectsBudget(t *testing.T) {
	a, _ := NewAgent("a", "x", 100)
	if err := a.Charge(60); err != nil {
		t.Fatalf("first charge: %v", err)
	}
	if err := a.Charge(50); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected ErrBudgetExceeded, got %v", err)
	}
	if a.TokensUsed() != 60 {
		t.Errorf("refused charge must not be recorded, used=%d", a.TokensUsed())
	}
	if err := a.Charge(40); err != nil {
		t.Errorf("charge that exactly reaches the limit should pass: %v", err)
	}
	if err := a.Charge(-1); err == nil {
		t.Error("negative charge should fail")
	}
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func TestSchedulerDueOnInterval(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)}
	s := NewScheduler(clock.now)
	noop := func(context.Context) error { return nil }
	if err := s.Add(&Routine{ID: "digest", Every: time.Hour, Run: noop}); err != nil {
		t.Fatal(err)
	}

	if got := s.Due(); len(got) != 1 {
		t.Fatalf("first call should be due immediately, got %d", len(got))
	}
	if got := s.Due(); len(got) != 0 {
		t.Fatalf("second call at same instant should not be due, got %d", len(got))
	}
	clock.t = clock.t.Add(59 * time.Minute)
	if got := s.Due(); len(got) != 0 {
		t.Fatalf("59 minutes in, should not be due, got %d", len(got))
	}
	clock.t = clock.t.Add(time.Minute)
	if got := s.Due(); len(got) != 1 {
		t.Fatalf("60 minutes in, should be due, got %d", len(got))
	}
}

func TestSchedulerRejectsBadRoutines(t *testing.T) {
	s := NewScheduler(nil)
	noop := func(context.Context) error { return nil }
	cases := []*Routine{
		nil,
		{ID: "", Every: time.Second, Run: noop},
		{ID: "a", Every: 0, Run: noop},
		{ID: "b", Every: time.Second, Run: nil},
	}
	for i, r := range cases {
		if err := s.Add(r); err == nil {
			t.Errorf("case %d: expected error", i)
		}
	}
	if err := s.Add(&Routine{ID: "dup", Every: time.Second, Run: noop}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(&Routine{ID: "dup", Every: time.Second, Run: noop}); err == nil {
		t.Error("duplicate id should fail")
	}
}
