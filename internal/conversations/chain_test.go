package conversations_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/gateway"
)

// sendAndWait sends one message under key and waits for its run to reach want.
func sendAndWait(t *testing.T, e *env, key string, want core.RunStatus) core.Run {
	t.Helper()
	_, run, err := e.svc.SendMessage(context.Background(), owner, e.conv.ID, "oi", key)
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	return waitStatus(t, e, run.ID, want)
}

// attempts returns the payload of each provider_attempt step of a run, in order.
func attempts(t *testing.T, e *env, runID string) []map[string]string {
	t.Helper()
	detail, err := e.svc.GetRun(context.Background(), owner, runID)
	if err != nil {
		t.Fatalf("get run detail: %v", err)
	}
	var out []map[string]string
	for _, s := range detail.Steps {
		if s.Kind != core.StepProviderAttempt {
			continue
		}
		var payload map[string]string
		if err := json.Unmarshal([]byte(s.Payload), &payload); err != nil {
			t.Fatalf("decode attempt payload %s: %v", s.Payload, err)
		}
		out = append(out, payload)
	}
	return out
}

func TestFallsBackToNextAccountBeforeOutput(t *testing.T) {
	primary := &fakeProvider{failure: fmt.Errorf("%w: status 503", gateway.ErrUnavailable)}
	backup := &fakeProvider{chunks: []string{"olá"}}
	set := providerSet{"fake/primary": primary, "fake/backup": backup}
	chain := []chainEntry{{label: "primary", model: "m-1"}, {label: "backup", model: "m-2"}}
	e := newEnvWith(t, set, chain, 1000000, 0)

	run := sendAndWait(t, e, "k-fallback", core.RunDone)

	want := []map[string]string{
		{"account": "fake/primary", "model": "m-1", "outcome": "failed", "error_code": core.ErrCodeProviderError},
		{"account": "fake/backup", "model": "m-2", "outcome": "done", "error_code": ""},
	}
	if got := attempts(t, e, run.ID); !reflect.DeepEqual(got, want) {
		t.Fatalf("attempts = %v, want %v", got, want)
	}
	if n := len(backup.Prompts()); n != 1 {
		t.Fatalf("backup calls = %d, want 1", n)
	}
}

func TestDoesNotFallBackOnceOutputStreamed(t *testing.T) {
	primary := &fakeProvider{chunks: []string{"parcial"}, tail: fmt.Errorf("%w: stream cut", gateway.ErrUnavailable)}
	backup := &fakeProvider{chunks: []string{"nunca"}}
	set := providerSet{"fake/primary": primary, "fake/backup": backup}
	chain := []chainEntry{{label: "primary", model: "m-1"}, {label: "backup", model: "m-2"}}
	e := newEnvWith(t, set, chain, 1000000, 0)

	run := sendAndWait(t, e, "k-tail", core.RunFailed)

	if run.ErrorCode != core.ErrCodeProviderError {
		t.Fatalf("error code = %q, want %q", run.ErrorCode, core.ErrCodeProviderError)
	}
	if got := attempts(t, e, run.ID); len(got) != 1 || got[0]["outcome"] != "failed" {
		t.Fatalf("attempts = %v, want one failed attempt", got)
	}
	if n := len(backup.Prompts()); n != 0 {
		t.Fatalf("backup calls = %d, want 0 after output started", n)
	}
}

func TestBudgetExhaustionNeverFallsBack(t *testing.T) {
	primary := &fakeProvider{chunks: []string{"x"}}
	backup := &fakeProvider{chunks: []string{"y"}}
	set := providerSet{"fake/primary": primary, "fake/backup": backup}
	chain := []chainEntry{{label: "primary", model: "m-1"}, {label: "backup", model: "m-2"}}
	e := newEnvWith(t, set, chain, 1, 0)

	run := sendAndWait(t, e, "k-budget", core.RunFailed)

	if run.ErrorCode != core.ErrCodeBudgetExceeded {
		t.Fatalf("error code = %q, want %q", run.ErrorCode, core.ErrCodeBudgetExceeded)
	}
	if got := attempts(t, e, run.ID); len(got) != 1 || got[0]["error_code"] != core.ErrCodeBudgetExceeded {
		t.Fatalf("attempts = %v, want one budget_exceeded attempt", got)
	}
	if n := len(primary.Prompts()) + len(backup.Prompts()); n != 0 {
		t.Fatalf("provider calls = %d, want 0 when the budget blocks the call", n)
	}
}

func TestChainStopsOnPossibleEffectsWithoutOutput(t *testing.T) {
	primary := &fakeProvider{failure: gateway.ErrUnavailable, effects: true}
	backup := &fakeProvider{chunks: []string{"never"}}
	set := providerSet{"fake/primary": primary, "fake/backup": backup}
	chain := []chainEntry{{label: "primary", model: "m-1"}, {label: "backup", model: "m-2"}}
	e := newEnvWith(t, set, chain, 1000000, 0)
	run := sendAndWait(t, e, "k-effects", core.RunFailed)
	if len(primary.Prompts()) != 1 || len(backup.Prompts()) != 0 || len(attempts(t, e, run.ID)) != 1 {
		t.Fatal("ambiguous effect must stop the entire chain")
	}
}

func TestChainStopsOnUsageIDFailure(t *testing.T) {
	primary := &fakeProvider{failure: gateway.ErrUnavailable}
	backup := &fakeProvider{chunks: []string{"never"}}
	set := providerSet{"fake/primary": primary, "fake/backup": backup}
	chain := []chainEntry{{label: "primary", model: "m-1"}, {label: "backup", model: "m-2"}}
	e := newEnvWithUsageID(t, set, chain, 1000000, 0, func(time.Time) (string, error) {
		return "", errors.New("usage id failure")
	})
	run := sendAndWait(t, e, "k-ledger-id", core.RunFailed)
	if len(primary.Prompts()) != 0 || len(backup.Prompts()) != 0 || len(attempts(t, e, run.ID)) != 1 {
		t.Fatal("ledger ID failure must prevent dispatch and fallback")
	}
}
