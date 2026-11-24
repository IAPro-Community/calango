package conversations_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
)

// waitForText blocks until runID has recorded a text step, so a steer lands mid-reply.
func waitForText(t *testing.T, e *env, runID string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		detail, err := e.svc.GetRun(context.Background(), owner, runID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		for _, st := range detail.Steps {
			if st.Kind == core.StepText {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("run never streamed text")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestSteerQueuedRunStopsItAndQueuesGuidedContinuation(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"x"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 1)
	ctx := context.Background()

	_, a, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "first", "key-a")
	if err != nil {
		t.Fatalf("send first: %v", err)
	}
	_, b, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "second", "key-b")
	if err != nil {
		t.Fatalf("send second: %v", err)
	}
	if b.Status != core.RunQueued {
		t.Fatalf("second run status = %q, want queued behind the first", b.Status)
	}

	cont, err := e.svc.Steer(ctx, owner, b.ID, "use bullets", "steer-b")
	if err != nil {
		t.Fatalf("steer queued run: %v", err)
	}
	if cont.SteerOf != b.ID || cont.Instructions != "use bullets" || cont.Status != core.RunQueued {
		t.Fatalf("continuation = %+v, want queued steer of %s", cont, b.ID)
	}
	waitStatus(t, e, b.ID, core.RunStopped)

	fp.Release()
	waitStatus(t, e, a.ID, core.RunDone)
	waitStatus(t, e, cont.ID, core.RunDone)

	prompts := fp.Prompts()
	if len(prompts) != 2 {
		t.Fatalf("provider calls = %d, want 2 (first run and continuation only)", len(prompts))
	}
	for _, want := range []string{"second", "Guidance from the owner:\nuse bullets"} {
		if !strings.Contains(prompts[1], want) {
			t.Fatalf("continuation prompt = %q, want it to contain %q", prompts[1], want)
		}
	}
}

func TestSteerRunningRunInterruptsAndContinuesWithPartialOutput(t *testing.T) {
	fp := &fakeProvider{head: []string{"partial "}, chunks: []string{"done"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 1)
	ctx := context.Background()

	_, a, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "explain", "key-a")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	waitForText(t, e, a.ID)

	cont, err := e.svc.Steer(ctx, owner, a.ID, "shorter please", "steer-a")
	if err != nil {
		t.Fatalf("steer running run: %v", err)
	}
	if cont.SteerOf != a.ID || cont.Status != core.RunQueued {
		t.Fatalf("continuation = %+v, want queued steer of %s", cont, a.ID)
	}

	again, err := e.svc.Steer(ctx, owner, a.ID, "shorter please", "steer-a")
	if err != nil || again.ID != cont.ID {
		t.Fatalf("repeated steer = %+v, %v; want continuation %s", again, err, cont.ID)
	}
	if _, err := e.svc.Steer(ctx, owner, a.ID, "other guidance", "steer-other"); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("second guidance error = %v, want ErrConflict", err)
	}

	waitStatus(t, e, a.ID, core.RunStopped)
	fp.Release()
	waitStatus(t, e, cont.ID, core.RunDone)

	prompts := fp.Prompts()
	if len(prompts) != 2 {
		t.Fatalf("provider calls = %d, want 2", len(prompts))
	}
	for _, want := range []string{"explain", "partial ", "Guidance from the owner:\nshorter please"} {
		if !strings.Contains(prompts[1], want) {
			t.Fatalf("continuation prompt = %q, want it to contain %q", prompts[1], want)
		}
	}
}

func TestSteerRejectsFinishedRunAndInvalidGuidance(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"x"}}
	e := newEnv(t, fp, 1000000, 0)
	ctx := context.Background()

	_, a, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "first", "key-a")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	waitStatus(t, e, a.ID, core.RunDone)

	if _, err := e.svc.Steer(ctx, owner, a.ID, "too late", "steer-late"); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("steer finished run error = %v, want ErrConflict", err)
	}
	if _, err := e.svc.Steer(ctx, owner, a.ID, "", "steer-empty"); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("steer empty guidance error = %v, want ErrInvalidInput", err)
	}
}
