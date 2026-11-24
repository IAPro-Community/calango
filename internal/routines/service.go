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

// Result is what one execution reports. ConversationID is set as soon as the routine has one,
// even when the execution fails, so the next attempt reuses it instead of opening another.
// Finish ends the routine.
type Result struct {
	RunID          string
	ConversationID string
	HadNews        bool
	Finish         bool
}

// Executor runs a routine's prompt once for one occurrence. Implementations create an agent
// run, so model calls still pass through the gateway.
type Executor interface {
	Execute(ctx context.Context, r core.Routine, o core.Occurrence) (Result, error)
}

// Event tells the owner that an occurrence fired or failed for good.
type Event struct {
	Kind         string
	OwnerID      string
	RoutineID    string
	OccurrenceID string
	Attempt      int
	Error        string
}

// Patch is a partial edit. A nil field keeps its stored value. Quiet bounds are set together;
// ClearQuiet removes the window and wins over both bounds.
type Patch struct {
	Name        *string
	Prompt      *string
	TriggerKind *core.TriggerKind
	Schedule    *string
	RunOn       *core.RunOn
	QuietStart  *int
	QuietEnd    *int
	ClearQuiet  bool
	MaxRetries  *int
	NotifyOn    *core.NotifyOn
	Paused      *bool
}

// Service owns routine lifecycle and delivery. mu guards the stored state of routines and
// occurrences. runMu serializes ticks. Executions run without mu, so an edit made while a run
// is in flight is not blocked by it; the outcome is written against a fresh copy of the routine.
type Service struct {
	mu     sync.Mutex
	runMu  sync.Mutex
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

// Update applies a partial edit, including pause and resume. The next time is recomputed when
// the trigger or schedule changes, or when a paused routine resumes. Occurrences already
// scheduled keep their times; resuming does not replay the times missed while paused.
func (s *Service) Update(ctx context.Context, ownerID, id string, p Patch) (core.Routine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.store.GetRoutine(ctx, ownerID, id)
	if err != nil {
		return core.Routine{}, err
	}
	if r.EndedAt != nil {
		return core.Routine{}, fmt.Errorf("%w: routine %s has ended", core.ErrConflict, id)
	}
	now := s.now()
	if p.Name != nil {
		r.Name = *p.Name
	}
	if p.Prompt != nil {
		r.Prompt = *p.Prompt
	}
	if p.TriggerKind != nil {
		r.TriggerKind = *p.TriggerKind
	}
	if p.Schedule != nil {
		r.Schedule = *p.Schedule
	}
	if p.RunOn != nil {
		r.RunOn = *p.RunOn
	}
	if p.MaxRetries != nil {
		r.MaxRetries = *p.MaxRetries
	}
	if p.NotifyOn != nil {
		r.NotifyOn = *p.NotifyOn
	}
	reschedule := p.TriggerKind != nil || p.Schedule != nil
	if p.Paused != nil {
		reschedule = reschedule || (r.Paused && !*p.Paused)
		r.Paused = *p.Paused
	}
	switch {
	case p.ClearQuiet:
		r.QuietStart, r.QuietEnd = nil, nil
	case p.QuietStart != nil || p.QuietEnd != nil:
		r.QuietStart, r.QuietEnd = p.QuietStart, p.QuietEnd
	}
	if err := r.Validate(s.loc); err != nil {
		return core.Routine{}, err
	}
	if reschedule {
		if r.NextRunAt, err = r.NextAfter(now, s.loc); err != nil {
			return core.Routine{}, err
		}
	}
	r.UpdatedAt = now
	if err := s.store.UpdateRoutine(ctx, r); err != nil {
		return core.Routine{}, err
	}
	return r, nil
}

// SetPaused pauses or resumes a routine. It is Update with only the pause field set.
func (s *Service) SetPaused(ctx context.Context, ownerID, id string, paused bool) (core.Routine, error) {
	return s.Update(ctx, ownerID, id, Patch{Paused: &paused})
}

// RunNow schedules one extra occurrence of a routine. It runs at the next tick, so the quiet
// window and the node's pause still apply. A paused or ended routine cannot run now.
func (s *Service) RunNow(ctx context.Context, ownerID, id string) (core.Occurrence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.store.GetRoutine(ctx, ownerID, id)
	if err != nil {
		return core.Occurrence{}, err
	}
	switch {
	case r.EndedAt != nil:
		return core.Occurrence{}, fmt.Errorf("%w: routine %s has ended", core.ErrConflict, id)
	case r.Paused:
		return core.Occurrence{}, fmt.Errorf("%w: routine %s is paused; resume it first", core.ErrConflict, id)
	}
	now := s.now()
	occID, err := s.newID(now)
	if err != nil {
		return core.Occurrence{}, fmt.Errorf("occurrence id: %w", err)
	}
	o := core.Occurrence{
		ID:            occID,
		RoutineID:     r.ID,
		OccurrenceID:  r.ID + "@manual:" + occID,
		ScheduledFor:  now,
		Status:        core.OccurrencePending,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := s.store.InsertOccurrence(ctx, o); err != nil {
		return core.Occurrence{}, err
	}
	return o, nil
}

// Tick schedules due routines and runs due occurrences. It returns how many executions
// ran. A node-wide pause stops everything. Ticks run one at a time.
func (s *Service) Tick(ctx context.Context) (int, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	now, ok, err := s.schedule(ctx)
	if err != nil || !ok {
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

// schedule records the occurrences that are due. It reports false when a node-wide pause
// stops the tick.
func (s *Service) schedule(ctx context.Context) (time.Time, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	paused, err := s.store.GlobalPaused(ctx)
	if err != nil {
		return time.Time{}, false, err
	}
	if paused {
		return time.Time{}, false, nil
	}
	now := s.now()
	if err := s.scheduleDue(ctx, now); err != nil {
		return time.Time{}, false, err
	}
	return now, true, nil
}

func (s *Service) runsHere(r core.Routine) bool {
	return r.RunOn == core.RunOnAuto || r.RunOn == s.node
}

// scheduleDue records one occurrence for each due routine. Times that were missed while
// the node was off collapse into that single occurrence, at the earliest missed time.
// The caller holds mu.
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

// runDue executes the due occurrences one after another. Each execution runs without mu.
func (s *Service) runDue(ctx context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	due, err := s.store.ListDueOccurrences(ctx, now)
	s.mu.Unlock()
	if err != nil {
		return 0, err
	}
	executed := 0
	for _, o := range due {
		r, claimed, ok, err := s.claim(ctx, o, now)
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

// claim takes a due occurrence for execution. It returns false when the occurrence stays
// pending: its routine is paused, ended or bound to another node, the quiet window postponed
// it, or another worker holds it.
func (s *Service) claim(ctx context.Context, o core.Occurrence, now time.Time) (core.Routine, core.Occurrence, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.store.GetRoutineByID(ctx, o.RoutineID)
	if err != nil {
		return core.Routine{}, core.Occurrence{}, false, err
	}
	if r.Paused || r.EndedAt != nil || !s.runsHere(r) {
		return core.Routine{}, core.Occurrence{}, false, nil
	}
	if until := core.PostponeOutsideQuietWindow(now, s.loc, r.QuietStart, r.QuietEnd); until.After(now) {
		return core.Routine{}, core.Occurrence{}, false, s.store.PostponeOccurrence(ctx, o.ID, until, now)
	}
	claimed, ok, err := s.store.ClaimOccurrence(ctx, o.ID, now)
	if err != nil || !ok {
		return core.Routine{}, core.Occurrence{}, false, err
	}
	return r, claimed, true, nil
}

// execute runs one claimed occurrence and records its outcome. The attempt number is
// already on the occurrence. Attempts 1..MaxRetries+1 are allowed in total.
func (s *Service) execute(ctx context.Context, r core.Routine, o core.Occurrence) error {
	res, execErr := s.exec.Execute(ctx, r, o)

	s.mu.Lock()
	defer s.mu.Unlock()
	finished := s.now()
	fresh, err := s.store.GetRoutineByID(ctx, r.ID)
	if err != nil {
		return err
	}
	routineChanged := false
	if res.ConversationID != "" && fresh.ConversationID == "" {
		fresh.ConversationID = res.ConversationID
		routineChanged = true
	}
	if execErr == nil && res.Finish && fresh.EndedAt == nil {
		fresh.EndedAt = &finished
		routineChanged = true
	}
	if routineChanged {
		fresh.UpdatedAt = finished
		if err := s.store.UpdateRoutine(ctx, fresh); err != nil {
			return err
		}
	}

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
		if fresh.NotifyOn == core.NotifyAlways || (fresh.NotifyOn == core.NotifyNews && hadNews) {
			s.notify(Event{Kind: EventFired, OwnerID: fresh.OwnerID, RoutineID: fresh.ID, OccurrenceID: o.OccurrenceID, Attempt: o.Attempt})
		}
		return nil
	}

	o.LastError = execErr.Error()
	if o.Attempt <= fresh.MaxRetries {
		o.Status = core.OccurrencePending
		o.NextAttemptAt = finished.Add(core.RetryBackoff(o.Attempt))
		return s.store.FinishOccurrence(ctx, o)
	}
	o.Status = core.OccurrenceFailed
	if err := s.store.FinishOccurrence(ctx, o); err != nil {
		return err
	}
	s.notify(Event{Kind: EventFailed, OwnerID: fresh.OwnerID, RoutineID: fresh.ID, OccurrenceID: o.OccurrenceID, Attempt: o.Attempt, Error: o.LastError})
	return nil
}
