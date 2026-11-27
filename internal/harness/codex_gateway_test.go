package harness

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"calango/internal/gateway"
)

// memLedger is a minimal gateway.Ledger for tests.
type memLedger struct {
	mu      sync.Mutex
	limit   int64
	records []gateway.Record
}

func (l *memLedger) AgentTokenLimit(context.Context, string) (int64, error) { return l.limit, nil }

func (l *memLedger) MonthlyTokens(context.Context, string, time.Time, time.Time) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var total int64
	for _, r := range l.records {
		total += r.TokensIn + r.TokensOut
	}
	return total, nil
}

func (l *memLedger) RecordUsage(_ context.Context, rec gateway.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, rec)
	return nil
}

func gatewayWith(t *testing.T, c *Codex, ledger *memLedger) *gateway.Gateway {
	t.Helper()
	g, err := gateway.New(gateway.Config{
		Ledger:    ledger,
		Providers: []gateway.Provider{c},
		Now:       time.Now,
		NewID:     func(time.Time) (string, error) { return "usage-1", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestCodexRunThroughGatewayRecordsEstimatedUnknownCost(t *testing.T) {
	c, _ := newFakeCodex(t)
	ledger := &memLedger{limit: 1_000_000}
	g := gatewayWith(t, c, ledger)

	var got string
	_, err := g.Call(context.Background(), gateway.Caller{AgentID: "agent-1"}, request("say hi"), func(ch gateway.Chunk) {
		got += ch.Text
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got != "hello from fake codex" {
		t.Fatalf("streamed %q", got)
	}
	if len(ledger.records) != 1 {
		t.Fatalf("records = %d, want 1", len(ledger.records))
	}
	rec := ledger.records[0]
	if rec.Provider != CodexName || rec.TokensIn <= 0 || rec.TokensOut <= 0 {
		t.Fatalf("record = %+v, want codex tokens", rec)
	}
	// No price is configured for codex: the cost is unknown, never zero.
	if rec.CostStatus != gateway.CostUnknown || rec.CostMicros != nil {
		t.Fatalf("cost = %s %v, want unknown and nil", rec.CostStatus, rec.CostMicros)
	}
}

func TestCodexSignedOutSurfacesThroughGateway(t *testing.T) {
	c, dir := newFakeCodex(t)
	if err := os.WriteFile(filepath.Join(dir, "FAKE_SIGNED_OUT"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	g := gatewayWith(t, c, &memLedger{limit: 1_000_000})

	_, err := g.Call(context.Background(), gateway.Caller{AgentID: "agent-1"}, request("hi"), nil)
	if !errors.Is(err, ErrSignedOut) {
		t.Fatalf("err = %v, want ErrSignedOut through the gateway", err)
	}
}
