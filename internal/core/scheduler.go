package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Routine is a unit of scheduled work that runs at a fixed interval.
type Routine struct {
	ID    string
	Every time.Duration
	Run   func(ctx context.Context) error

	next time.Time
}

// Scheduler keeps routines and reports which ones are due. The clock is injected so
// tests do not sleep.
type Scheduler struct {
	mu       sync.Mutex
	now      func() time.Time
	routines map[string]*Routine
}

// NewScheduler creates a scheduler. A nil clock falls back to time.Now.
func NewScheduler(now func() time.Time) *Scheduler {
	if now == nil {
		now = time.Now
	}
	return &Scheduler{now: now, routines: make(map[string]*Routine)}
}

// Add registers a routine. The first run is due immediately.
func (s *Scheduler) Add(r *Routine) error {
	if r == nil || r.ID == "" {
		return errors.New("routine id is required")
	}
	if r.Every <= 0 {
		return fmt.Errorf("routine %s: interval must be positive", r.ID)
	}
	if r.Run == nil {
		return fmt.Errorf("routine %s: run function is required", r.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.routines[r.ID]; exists {
		return fmt.Errorf("routine %s already exists", r.ID)
	}
	r.next = s.now()
	s.routines[r.ID] = r
	return nil
}

// Due returns the routines whose next run is at or before now, and schedules each one
// for now+Every. A routine that is due is returned once per due time, not once per call.
func (s *Scheduler) Due() []*Routine {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	var due []*Routine
	for _, r := range s.routines {
		if !now.Before(r.next) {
			due = append(due, r)
			r.next = now.Add(r.Every)
		}
	}
	return due
}

// Run ticks every tick duration, starts the due routines in their own goroutines, and
// returns when ctx is cancelled. Errors from routines are passed to onError when it is set.
func (s *Scheduler) Run(ctx context.Context, tick time.Duration, onError func(id string, err error)) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, r := range s.Due() {
				go func(r *Routine) {
					if err := r.Run(ctx); err != nil && onError != nil {
						onError(r.ID, err)
					}
				}(r)
			}
		}
	}
}
