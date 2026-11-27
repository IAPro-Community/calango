package gateway

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type admittingProvider struct {
	fakeProvider
	ledger *memLedger
	admit  int
	finish int
	seen   error
}

func (p *admittingProvider) Admit(_ context.Context, _ *Request) (func(error) error, error) {
	p.admit++
	return func(err error) error {
		p.finish++
		p.seen = err
		if len(p.ledger.snapshot()) != 1 {
			return errors.New("admission released before ledger")
		}
		return nil
	}, nil
}

func TestAdmissionRunsAfterBudgetAndBeforeLedgerFinalization(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{"agent": 20}}
	p := &admittingProvider{ledger: ledger}
	g, err := New(Config{Ledger: ledger, Providers: []Provider{p}, Now: func() time.Time { return testNow },
		NewID: func(time.Time) (string, error) { return "usage-id", nil }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 21), nil); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatal(err)
	}
	if p.admit != 0 || p.calls != 0 {
		t.Fatal("budget denial must precede admission")
	}
	if _, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil); err != nil {
		t.Fatal(err)
	}
	if p.admit != 1 || p.finish != 1 || p.calls != 1 || p.seen != nil {
		t.Fatalf("unexpected admission lifecycle: %+v", p)
	}
}

func TestZeroTariffRequiresAccountBinding(t *testing.T) {
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	table := PriceTable{Version: "v1", Models: map[string]Price{
		"codex/work/m":  {AccountIdentitySHA256: a},
		"codex/other/m": {},
		"codex/paid/m":  {InputPerMillion: 1, AccountIdentitySHA256: a},
	}}
	if !table.ConfirmedZero("codex/work", "m", a) ||
		table.ConfirmedZero("codex/work", "m", b) ||
		table.ConfirmedZero("codex/other", "m", a) ||
		table.ConfirmedZero("codex/paid", "m", a) {
		t.Fatal("incorrect automatic cost gate")
	}
}
