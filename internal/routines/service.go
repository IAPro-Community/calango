// Package routines runs scheduled agent work. Each scheduled time becomes a persisted
// occurrence with a deterministic ID, so restarts and retries reuse it. Missed times
// coalesce into one run, failed attempts retry with backoff, and quiet windows postpone.
package routines

import (
	"context"
	"fmt"
	"sync"
	"time"

	"calango/internal/core"
	"calango/internal/store"
)

// Event kinds sent to the owner.
const (
	EventFired  = "routine.fired"
	EventFailed = "routine.failed"
)

// Result is what one successful execution reports.
type Result struct {
	RunID   string
	HadNews bool
}

// Executor runs a routine's prompt once for one occurrence. Implementations create an agent
// run, so model calls still pass through the gateway.
type Executor interface {
	Execute(ctx context.Context, r core.Routine, o core.Occurrence) (Result, error)
}

// Event tells the owner that an occurrence fired or failed for good.
type Event struct {
	Kind         string
	RoutineID    string
	OccurrenceID string
	Attempt      int
	Error        string
}

// Service owns routine lifecycle and delivery. Ticks are serialized by a mutex.
type Service struct {
	mu     sync.Mutex
	store  *store.Store
	node   core.RunOn
	loc    *time.Location
	exec   Executor
	notify func(Event)
	now    func() time.Time
	newID  func(time.Time) (string, error)
}

// NewService builds the service for one node. node is the place this process runs: pc or
// vps. Routines with run_on=auto run on any node; routines bound to the other node are
// left untouched. notify may be nil.
func NewService(st *store.Store, node core.RunOn, loc *time.Location, exec Executor, notify func(Event),
	now func() time.Time, newID func(time.Time) (string, error)) (*Service, error) {
	if node != core.RunOnPC && node != core.RunOnVPS {
		return nil, fmt.Errorf("%w: node must be pc or vps", core.ErrInvalidInput)
	}
	if loc == nil || exec == nil || now == nil || newID == nil {
		return nil, fmt.Errorf("routines: location, executor, clock and id generator are required")
	}
	if notify == nil {
		notify = func(Event) {}
	}
	return &Service{store: st, node: node, loc: loc, exec: exec, notify: notify, now: now, newID: newID}, nil
}

// Start requeues occurrences a previous process left running. Their attempt was already
// counted, so the occurrence runs again with the same ID.
func (s *Service) Start(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.RequeueInterruptedOccurrences(ctx, s.now())
}

// Create stores a routine owned by ownerID and schedules its first occurrence.
func (s *Service) Create(ctx context.Context, ownerID string, r core.Routine) (core.Routine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	id, err := s.newID(now)
	if err != nil {
		return core.Routine{}, fmt.Errorf("routine id: %w", err)
	}
	r.ID = id
	r.OwnerID = ownerID
	r.Paused = false
	r.Normalize()
	r.CreatedAt = now
	r.UpdatedAt = now
	if _, err := s.store.GetAgent(ctx, ownerID, r.AgentID); err != nil {
		return core.Routine{}, err
	}
	if err := r.Validate(s.loc); err != nil {
		return core.Routine{}, err
	}
	if r.NextRunAt, err = r.NextAfter(now, s.loc); err != nil {
		return core.Routine{}, err
	}
	if err := s.store.InsertRoutine(ctx, r); err != nil {
		return core.Routine{}, err
	}
	return r, nil
}

// Get returns a routine owned by ownerID.
func (s *Service) Get(ctx context.Context, ownerID, id string) (core.Routine, error) {
	return s.store.GetRoutine(ctx, ownerID, id)
}

// List returns the owner's routines.
func (s *Service) List(ctx context.Context, ownerID string) ([]core.Routine, error) {
	return s.store.ListRoutines(ctx, ownerID)
}

// Occurrences returns the delivery history of one routine owned by ownerID.
func (s *Service) Occurrences(ctx context.Context, ownerID, id string) ([]core.Occurrence, error) {
	if _, err := s.store.GetRoutine(ctx, ownerID, id); err != nil {
		return nil, err
	}
	return s.store.ListOccurrences(ctx, id)
}

// Delete removes a routine and its history.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.DeleteRoutine(ctx, ownerID, id)
}

// SetPaused pauses or resumes a routine. Resuming schedules the next time after now, so
// the times missed while paused do not fire.
func (s *Service) SetPaused(ctx context.Context, ownerID, id string, paused bool) (core.Routine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.store.GetRoutine(ctx, ownerID, id)
	if err != nil {
		return core.Routine{}, err
	}
	if r.Paused == paused {
		return r, nil
	}
	now := s.now()
	r.Paused = paused
	r.UpdatedAt = now
	if !paused {
		if r.NextRunAt, err = r.NextAfter(now, s.loc); err != nil {
			return core.Routine{}, err
		}
	}
	if err := s.store.UpdateRoutine(ctx, r); err != nil {
		return core.Routine{}, err
	}
	return r, nil
}

// Tick schedules due routines and runs due occurrences. It returns how many executions
// ran. A node-wide pause stops everything.
func (s *Service) Tick(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	paused, err := s.store.GlobalPaused(ctx)
	if err != nil {
		return 0, err
	}
	if paused {
		return 0, nil
	}
	now := s.now()
	if err := s.scheduleDue(ctx, now); err != nil {
		return 0, err
	}
	return s.runDue(ctx, now)
}

// Run calls Tick every interval until ctx is done. Tick errors go to onError when set.
func (s *Service) Run(ctx context.Context, every time.Duration, onError func(error)) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.Tick(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}

func (s *Service) runsHere(r core.Routine) bool {
	return r.RunOn == core.RunOnAuto || r.RunOn == s.node
}

// scheduleDue records one occurrence for each due routine. Times that were missed while
// the node was off collapse into that single occurrence, at the earliest missed time.
func (s *Service) scheduleDue(ctx context.Context, now time.Time) error {
	due, err := s.store.ListDueRoutines(ctx, now)
	if err != nil {
		return err
	}
	for _, r := range due {
		if !s.runsHere(r) {
			continue
		}
		scheduled := r.NextRunAt
		id, err := s.newID(now)
		if err != nil {
			return fmt.Errorf("occurrence id: %w", err)
		}
		if _, err := s.store.InsertOccurrence(ctx, core.Occurrence{
			ID:            id,
			RoutineID:     r.ID,
			OccurrenceID:  core.OccurrenceID(r.ID, scheduled),
			ScheduledFor:  scheduled,
			Status:        core.OccurrencePending,
			NextAttemptAt: scheduled,
			CreatedAt:     now,
			UpdatedAt:     now,
		}); err != nil {
			return err
		}
		if r.NextRunAt, err = r.NextAfter(now, s.loc); err != nil {
			return err
		}
		r.UpdatedAt = now
		if err := s.store.UpdateRoutine(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) runDue(ctx context.Context, now time.Time) (int, error) {
	due, err := s.store.ListDueOccurrences(ctx, now)
	if err != nil {
		return 0, err
	}
	executed := 0
	for _, o := range due {
		r, err := s.store.GetRoutineByID(ctx, o.RoutineID)
		if err != nil {
			return executed, err
		}
		if r.Paused || !s.runsHere(r) {
			continue
		}
		if until := core.PostponeOutsideQuietWindow(now, s.loc, r.QuietStart, r.QuietEnd); until.After(now) {
			if err := s.store.PostponeOccurrence(ctx, o.ID, until, now); err != nil {
				return executed, err
			}
			continue
		}
		claimed, ok, err := s.store.ClaimOccurrence(ctx, o.ID, now)
		if err != nil {
			return executed, err
		}
		if !ok {
			continue
		}
		if err := s.execute(ctx, r, claimed); err != nil {
			return executed, err
		}
		executed++
	}
	return executed, nil
}

// execute runs one claimed occurrence and records its outcome. The attempt number is
// already on the occurrence. Attempts 1..MaxRetries+1 are allowed in total.
func (s *Service) execute(ctx context.Context, r core.Routine, o core.Occurrence) error {
	res, execErr := s.exec.Execute(ctx, r, o)
	finished := s.now()
	o.UpdatedAt = finished
	o.NextAttemptAt = finished

	if execErr == nil {
		hadNews := res.HadNews
		o.Status = core.OccurrenceDone
		o.RunID = res.RunID
		o.HadNews = &hadNews
		o.LastError = ""
		if err := s.store.FinishOccurrence(ctx, o); err != nil {
			return err
		}
		if r.NotifyOn == core.NotifyAlways || (r.NotifyOn == core.NotifyNews && hadNews) {
			s.notify(Event{Kind: EventFired, RoutineID: r.ID, OccurrenceID: o.OccurrenceID, Attempt: o.Attempt})
		}
		return nil
	}

	o.LastError = execErr.Error()
	if o.Attempt <= r.MaxRetries {
		o.Status = core.OccurrencePending
		o.NextAttemptAt = finished.Add(core.RetryBackoff(o.Attempt))
		return s.store.FinishOccurrence(ctx, o)
	}
	o.Status = core.OccurrenceFailed
	if err := s.store.FinishOccurrence(ctx, o); err != nil {
		return err
	}
	s.notify(Event{Kind: EventFailed, RoutineID: r.ID, OccurrenceID: o.OccurrenceID, Attempt: o.Attempt, Error: o.LastError})
	return nil
}
