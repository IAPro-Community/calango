package routines

import (
	"errors"
	"strings"
	"testing"

	"calango/internal/core"
)

func TestFinishEndsRoutineAndStopsFutureRuns(t *testing.T) {
	exec := &fakeExec{finish: true}
	e := newEnv(t, exec, at(8, 0))
	r := e.routine(nil)

	e.clk.t = at(9, 0)
	if n := e.tick(); n != 1 {
		t.Fatalf("first tick executed %d, want 1", n)
	}
	ended, err := e.svc.Get(e.ctx, "owner", r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ended.EndedAt == nil {
		t.Fatal("finish signal did not end the routine")
	}

	e.clk.t = at(9, 0).AddDate(0, 0, 1)
	if n := e.tick(); n != 0 {
		t.Fatalf("ended routine ran again: %d", n)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("executor calls = %d, want 1", len(exec.calls))
	}
}

func TestEndedRoutineRejectsPauseRunNowAndUpdate(t *testing.T) {
	exec := &fakeExec{finish: true}
	e := newEnv(t, exec, at(8, 0))
	r := e.routine(nil)
	e.clk.t = at(9, 0)
	e.tick()

	if _, err := e.svc.SetPaused(e.ctx, "owner", r.ID, true); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("pause of ended routine: err = %v, want ErrConflict", err)
	}
	if _, err := e.svc.RunNow(e.ctx, "owner", r.ID); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("run now of ended routine: err = %v, want ErrConflict", err)
	}
	name := "renamed"
	if _, err := e.svc.Update(e.ctx, "owner", r.ID, Patch{Name: &name}); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("update of ended routine: err = %v, want ErrConflict", err)
	}
}

func TestFinishOnFailedAttemptIsIgnored(t *testing.T) {
	exec := &fakeExec{finish: true, errs: []error{errors.New("provider unavailable")}}
	e := newEnv(t, exec, at(8, 0))
	r := e.routine(func(r *core.Routine) { r.MaxRetries = 0 })

	e.clk.t = at(9, 0)
	e.tick()
	got, err := e.svc.Get(e.ctx, "owner", r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.EndedAt != nil {
		t.Fatal("finish on a failed attempt ended the routine")
	}
	occs := e.occurrences(r.ID)
	if len(occs) != 1 || occs[0].Status != core.OccurrenceFailed {
		t.Fatalf("occurrences = %+v, want one failed occurrence", occs)
	}
}

func TestConversationSurvivesFailedAttempt(t *testing.T) {
	convID, err := core.NewID(at(8, 0))
	if err != nil {
		t.Fatal(err)
	}
	exec := &fakeExec{conversation: convID, errs: []error{errors.New("run failed")}}
	e := newEnv(t, exec, at(8, 0))
	if err := e.st.InsertConversation(e.ctx, core.Conversation{ID: convID, Kind: "dm", CreatedAt: at(8, 0)}, e.agent); err != nil {
		t.Fatal(err)
	}
	r := e.routine(nil)

	e.clk.t = at(9, 0)
	e.tick()
	got, err := e.svc.Get(e.ctx, "owner", r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConversationID != convID {
		t.Fatalf("conversation id = %q after a failed attempt, want %q", got.ConversationID, convID)
	}
}

func TestRunNowFiresAsManualOccurrenceOnNextTick(t *testing.T) {
	exec := &fakeExec{}
	e := newEnv(t, exec, at(8, 0))
	r := e.routine(nil)

	o, err := e.svc.RunNow(e.ctx, "owner", r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(o.OccurrenceID, "@manual:") {
		t.Fatalf("manual occurrence id = %q, want an @manual: id", o.OccurrenceID)
	}
	if n := e.tick(); n != 1 {
		t.Fatalf("manual run executed %d, want 1", n)
	}
	if len(exec.calls) != 1 || exec.calls[0].OccurrenceID != o.OccurrenceID {
		t.Fatalf("executor calls = %+v, want the manual occurrence", exec.calls)
	}

	if _, err := e.svc.SetPaused(e.ctx, "owner", r.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.RunNow(e.ctx, "owner", r.ID); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("run now of paused routine: err = %v, want ErrConflict", err)
	}
}

func TestUpdateScheduleRecomputesNextRun(t *testing.T) {
	e := newEnv(t, &fakeExec{}, at(8, 0))
	r := e.routine(nil) // 09:00 daily

	schedule := "0 12 * * *"
	updated, err := e.svc.Update(e.ctx, "owner", r.ID, Patch{Schedule: &schedule})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.NextRunAt.Equal(at(12, 0)) {
		t.Fatalf("next run after schedule change = %s, want 12:00", updated.NextRunAt)
	}
}
