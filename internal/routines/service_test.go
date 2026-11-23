package routines

import (
	"context"
	"errors"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/store"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

type fakeExec struct {
	calls   []core.Occurrence
	errs    []error // errs[i] is returned by the i-th call; nil means success
	hadNews bool
}

func (f *fakeExec) Execute(_ context.Context, _ core.Routine, o core.Occurrence) (Result, error) {
	i := len(f.calls)
	f.calls = append(f.calls, o)
	if i < len(f.errs) && f.errs[i] != nil {
		return Result{}, f.errs[i]
	}
	return Result{HadNews: f.hadNews}, nil
}

type env struct {
	t      *testing.T
	ctx    context.Context
	st     *store.Store
	clk    *clock
	exec   *fakeExec
	svc    *Service
	events []Event
	agent  string
}

func newEnv(t *testing.T, exec *fakeExec, start time.Time) *env {
	t.Helper()
	ctx := context.Background()
	st, err := store.OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	clk := &clock{t: start}
	e := &env{t: t, ctx: ctx, st: st, clk: clk, exec: exec}
	agentID, err := core.NewID(start)
	if err != nil {
		t.Fatal(err)
	}
	e.agent = agentID
	if err := st.InsertAgent(ctx, core.Agent{
		ID: agentID, OwnerID: "owner", Name: "Assistant", TokenLimitMonth: 1000,
		RunOn: core.RunOnPC, ApprovalLevel: core.ApprovalSensitive,
		CreatedAt: start, UpdatedAt: start,
	}); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(st, core.RunOnPC, time.UTC, exec, func(ev Event) { e.events = append(e.events, ev) }, clk.now, core.NewID)
	if err != nil {
		t.Fatal(err)
	}
	e.svc = svc
	return e
}

func (e *env) routine(mutate func(*core.Routine)) core.Routine {
	e.t.Helper()
	r := core.Routine{
		AgentID: e.agent, Name: "digest", Prompt: "summarise",
		TriggerKind: core.TriggerCron, Schedule: "0 9 * * *",
		MaxRetries: 2, NotifyOn: core.NotifyAlways,
	}
	if mutate != nil {
		mutate(&r)
	}
	created, err := e.svc.Create(e.ctx, "owner", r)
	if err != nil {
		e.t.Fatalf("create routine: %v", err)
	}
	return created
}

func (e *env) tick() int {
	e.t.Helper()
	n, err := e.svc.Tick(e.ctx)
	if err != nil {
		e.t.Fatalf("tick: %v", err)
	}
	return n
}

func (e *env) occurrences(routineID string) []core.Occurrence {
	e.t.Helper()
	occs, err := e.svc.Occurrences(e.ctx, "owner", routineID)
	if err != nil {
		e.t.Fatalf("occurrences: %v", err)
	}
	return occs
}

func at(h, m int) time.Time {
	return time.Date(2026, 1, 1, h, m, 0, 0, time.UTC)
}

func TestCronRoutineFiresOncePerScheduledTime(t *testing.T) {
	exec := &fakeExec{}
	e := newEnv(t, exec, at(8, 0))
	r := e.routine(nil)

	e.clk.t = at(9, 0)
	if n := e.tick(); n != 1 {
		t.Fatalf("first tick executed %d, want 1", n)
	}
	if n := e.tick(); n != 0 {
		t.Fatalf("same instant executed %d, want 0", n)
	}
	occs := e.occurrences(r.ID)
	if len(occs) != 1 || occs[0].Status != core.OccurrenceDone || occs[0].Attempt != 1 {
		t.Fatalf("occurrences = %+v, want one done attempt 1", occs)
	}
}

func TestMissedRunsCoalesceIntoOneExecution(t *testing.T) {
	exec := &fakeExec{}
	e := newEnv(t, exec, at(0, 30))
	r := e.routine(func(r *core.Routine) { r.Schedule = "0 * * * *" }) // hourly

	e.clk.t = at(3, 10) // 01:00, 02:00 and 03:00 were missed while the PC was off
	if n := e.tick(); n != 1 {
		t.Fatalf("executed %d after outage, want 1", n)
	}
	occs := e.occurrences(r.ID)
	if len(occs) != 1 || !occs[0].ScheduledFor.Equal(at(1, 0)) {
		t.Fatalf("occurrences = %+v, want one for 01:00", occs)
	}
	stored, err := e.svc.Get(e.ctx, "owner", r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.NextRunAt.Equal(at(4, 0)) {
		t.Fatalf("next run = %s, want 04:00", stored.NextRunAt)
	}
}

func TestTransientFailureRetriesSameOccurrenceWithoutFailureEvent(t *testing.T) {
	exec := &fakeExec{errs: []error{errors.New("provider timeout")}}
	e := newEnv(t, exec, at(8, 0))
	r := e.routine(nil)

	e.clk.t = at(9, 0)
	e.tick() // attempt 1 fails, retry after 1 minute
	e.clk.t = at(9, 0).Add(30 * time.Second)
	if n := e.tick(); n != 0 {
		t.Fatalf("retry ran before backoff: %d", n)
	}
	e.clk.t = at(9, 1)
	if n := e.tick(); n != 1 {
		t.Fatalf("retry executed %d, want 1", n)
	}

	occs := e.occurrences(r.ID)
	if len(occs) != 1 {
		t.Fatalf("occurrences = %d, want 1 record for the retried time", len(occs))
	}
	if occs[0].Status != core.OccurrenceDone || occs[0].Attempt != 2 || occs[0].LastError != "" {
		t.Fatalf("occurrence = %+v, want done on attempt 2 with cleared error", occs[0])
	}
	if len(exec.calls) != 2 || exec.calls[0].OccurrenceID != exec.calls[1].OccurrenceID {
		t.Fatalf("retry must reuse the occurrence ID, calls = %+v", exec.calls)
	}
	for _, ev := range e.events {
		if ev.Kind == EventFailed {
			t.Fatalf("transient failure notified the owner: %+v", ev)
		}
	}
}

func TestExhaustedRetriesFailOnceAndNotifyOwner(t *testing.T) {
	boom := errors.New("harness unavailable")
	exec := &fakeExec{errs: []error{boom, boom, boom, boom}}
	e := newEnv(t, exec, at(8, 0))
	r := e.routine(func(r *core.Routine) { r.MaxRetries = 1 })

	e.clk.t = at(9, 0)
	e.tick() // attempt 1, retry at 09:01
	e.clk.t = at(9, 1)
	e.tick() // attempt 2, exhausted

	occs := e.occurrences(r.ID)
	if len(occs) != 1 || occs[0].Status != core.OccurrenceFailed || occs[0].Attempt != 2 {
		t.Fatalf("occurrences = %+v, want one failed attempt 2", occs)
	}
	var failed []Event
	for _, ev := range e.events {
		if ev.Kind == EventFailed {
			failed = append(failed, ev)
		}
	}
	if len(failed) != 1 || failed[0].Error != boom.Error() {
		t.Fatalf("failure events = %+v, want exactly one with the error", failed)
	}

	e.clk.t = at(9, 30)
	if n := e.tick(); n != 0 {
		t.Fatalf("failed occurrence ran again: %d", n)
	}
}

func TestQuietWindowPostponesWithoutLosingTheRun(t *testing.T) {
	exec := &fakeExec{}
	e := newEnv(t, exec, at(20, 0))
	night := 23 * 60
	morning := 7 * 60
	r := e.routine(func(r *core.Routine) {
		r.Schedule = "0 23 * * *"
		r.QuietStart = &night
		r.QuietEnd = &morning
	})

	e.clk.t = at(23, 0)
	if n := e.tick(); n != 0 {
		t.Fatalf("ran inside quiet window: %d", n)
	}
	occs := e.occurrences(r.ID)
	if len(occs) != 1 || occs[0].Status != core.OccurrencePending {
		t.Fatalf("occurrences = %+v, want one pending", occs)
	}
	if !occs[0].NextAttemptAt.Equal(time.Date(2026, 1, 2, 7, 0, 0, 0, time.UTC)) {
		t.Fatalf("postponed to %s, want 07:00 next day", occs[0].NextAttemptAt)
	}

	e.clk.t = time.Date(2026, 1, 2, 7, 0, 0, 0, time.UTC)
	if n := e.tick(); n != 1 {
		t.Fatalf("executed %d after window, want 1", n)
	}
}

func TestNewsPolicyNotifiesOnlyWhenThereIsNews(t *testing.T) {
	exec := &fakeExec{hadNews: false}
	e := newEnv(t, exec, at(8, 0))
	e.routine(func(r *core.Routine) { r.NotifyOn = core.NotifyNews })

	e.clk.t = at(9, 0)
	e.tick()
	if len(e.events) != 0 {
		t.Fatalf("news policy notified without news: %+v", e.events)
	}

	exec.hadNews = true
	e.clk.t = time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)
	e.tick()
	if len(e.events) != 1 || e.events[0].Kind != EventFired {
		t.Fatalf("events = %+v, want one fired event", e.events)
	}
}

func TestFailurePolicySilentOnSuccess(t *testing.T) {
	exec := &fakeExec{}
	e := newEnv(t, exec, at(8, 0))
	e.routine(func(r *core.Routine) { r.NotifyOn = core.NotifyFailure })

	e.clk.t = at(9, 0)
	e.tick()
	if len(e.events) != 0 {
		t.Fatalf("failure policy notified on success: %+v", e.events)
	}
}

// blockingExec stands in for a process that dies mid-execution: it signals that the run
// started and then waits until its context is cancelled.
type blockingExec struct {
	started chan struct{}
	block   bool
}

func (b *blockingExec) Execute(ctx context.Context, _ core.Routine, _ core.Occurrence) (Result, error) {
	if !b.block {
		return Result{}, nil
	}
	b.started <- struct{}{}
	<-ctx.Done()
	return Result{}, ctx.Err()
}

func TestRestartRequeuesInterruptedOccurrenceWithSameRecord(t *testing.T) {
	e := newEnv(t, &fakeExec{}, at(8, 0))
	r := e.routine(nil)
	exec := &blockingExec{started: make(chan struct{}, 1), block: true}

	first, err := NewService(e.st, core.RunOnPC, time.UTC, exec, nil, e.clk.now, core.NewID)
	if err != nil {
		t.Fatal(err)
	}
	e.clk.t = at(9, 0)
	ctxFirst, cancelFirst := context.WithCancel(e.ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = first.Tick(ctxFirst) // the in-flight run errors out when the test "kills" it
	}()
	<-exec.started

	// Restarted process: the occurrence is still marked running in the store.
	exec.block = false
	second, err := NewService(e.st, core.RunOnPC, time.UTC, exec, nil, e.clk.now, core.NewID)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := second.Start(e.ctx); err != nil || n != 1 {
		t.Fatalf("start requeued %d (err %v), want 1 interrupted occurrence", n, err)
	}
	if n, err := second.Tick(e.ctx); err != nil || n != 1 {
		t.Fatalf("second tick executed %d (err %v), want 1", n, err)
	}
	cancelFirst()
	<-done

	occs := e.occurrences(r.ID)
	if len(occs) != 1 {
		t.Fatalf("occurrences = %d, want the same single record", len(occs))
	}
	if occs[0].Status != core.OccurrenceDone || occs[0].Attempt != 2 {
		t.Fatalf("occurrence = %+v, want done on attempt 2", occs[0])
	}
}

func TestPausedAndGlobalPauseStopExecution(t *testing.T) {
	exec := &fakeExec{}
	e := newEnv(t, exec, at(8, 0))
	r := e.routine(nil)

	if _, err := e.svc.SetPaused(e.ctx, "owner", r.ID, true); err != nil {
		t.Fatal(err)
	}
	e.clk.t = at(9, 0)
	if n := e.tick(); n != 0 {
		t.Fatalf("paused routine ran: %d", n)
	}

	if _, err := e.svc.SetPaused(e.ctx, "owner", r.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := e.st.SetGlobalPaused(e.ctx, true, e.clk.t); err != nil {
		t.Fatal(err)
	}
	e.clk.t = at(10, 0)
	if n := e.tick(); n != 0 {
		t.Fatalf("global pause still ran: %d", n)
	}
}

func TestResumeDoesNotFireTimesMissedWhilePaused(t *testing.T) {
	exec := &fakeExec{}
	e := newEnv(t, exec, at(8, 0))
	r := e.routine(func(r *core.Routine) { r.Schedule = "0 * * * *" })

	if _, err := e.svc.SetPaused(e.ctx, "owner", r.ID, true); err != nil {
		t.Fatal(err)
	}
	e.clk.t = at(12, 30)
	resumed, err := e.svc.SetPaused(e.ctx, "owner", r.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if !resumed.NextRunAt.Equal(at(13, 0)) {
		t.Fatalf("next run after resume = %s, want 13:00", resumed.NextRunAt)
	}
	if n := e.tick(); n != 0 {
		t.Fatalf("resume fired missed times: %d", n)
	}
}

func TestRunOnRoutesOnlyToThisNode(t *testing.T) {
	exec := &fakeExec{}
	e := newEnv(t, exec, at(8, 0))
	e.routine(func(r *core.Routine) { r.RunOn = core.RunOnVPS })
	e.routine(func(r *core.Routine) { r.RunOn = core.RunOnAuto })

	e.clk.t = at(9, 0)
	if n := e.tick(); n != 1 {
		t.Fatalf("executed %d, want only the auto routine", n)
	}
}

func TestCreateRejectsAgentOwnedByAnotherOwner(t *testing.T) {
	exec := &fakeExec{}
	e := newEnv(t, exec, at(8, 0))
	_, err := e.svc.Create(e.ctx, "intruder", core.Routine{
		AgentID: e.agent, Name: "x", Prompt: "y", TriggerKind: core.TriggerCron, Schedule: "0 9 * * *",
	})
	if !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestNewServiceRejectsUnknownNode(t *testing.T) {
	st, err := store.OpenMemory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	_, err = NewService(st, core.RunOnAuto, time.UTC, &fakeExec{}, nil, time.Now, core.NewID)
	if !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}
