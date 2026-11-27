package gateway

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

var testNow = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

type fakeProvider struct {
	usage   Usage
	err     error
	calls   int
	started chan struct{}
	release chan struct{}
}

func (p *fakeProvider) Name() string { return "fake" }

func (p *fakeProvider) Complete(_ context.Context, _ Request, stream func(Chunk)) (Usage, error) {
	p.calls++
	if p.started != nil {
		p.started <- struct{}{}
		<-p.release
	}
	if stream != nil {
		stream(Chunk{Text: "ok"})
	}
	return p.usage, p.err
}

type memLedger struct {
	mu      sync.Mutex
	limits  map[string]int64
	records []Record
}

func (l *memLedger) AgentTokenLimit(_ context.Context, agentID string) (int64, error) {
	limit, ok := l.limits[agentID]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnknownAgent, agentID)
	}
	return limit, nil
}

func (l *memLedger) MonthlyTokens(_ context.Context, agentID string, from, to time.Time) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var total int64
	for _, r := range l.records {
		if r.Caller.AgentID == agentID && !r.At.Before(from) && r.At.Before(to) {
			total += r.TokensIn + r.TokensOut
		}
	}
	return total, nil
}

func (l *memLedger) RecordUsage(_ context.Context, rec Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.records = append(l.records, rec)
	return nil
}

func (l *memLedger) snapshot() []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Record(nil), l.records...)
}

func newTestGateway(t *testing.T, ledger *memLedger, provider *fakeProvider, prices PriceTable, onThreshold func(ThresholdEvent)) *Gateway {
	t.Helper()

	ids := 0
	g, err := New(Config{
		Ledger:    ledger,
		Providers: []Provider{provider},
		Prices:    prices,
		Now:       func() time.Time { return testNow },
		NewID: func(time.Time) (string, error) {
			ids++
			return fmt.Sprintf("%026d", ids), nil
		},
		OnThreshold: onThreshold,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return g
}

// fakeRequest asks for maxOut output tokens with an empty prompt, so the
// pre-call estimate is exactly maxOut.
func fakeRequest(model string, maxOut int64) Request {
	return Request{Provider: "fake", Model: model, MaxOutputTokens: maxOut}
}

func TestCallRefusesBeforeProviderWhenEstimateExceedsRemainingBudget(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{"agent": 100}}
	ledger.records = []Record{{At: testNow, Caller: Caller{AgentID: "agent"}, TokensIn: 95}}
	provider := &fakeProvider{}
	g := newTestGateway(t, ledger, provider, PriceTable{}, nil)

	_, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil)

	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("Call() error = %v, want ErrBudgetExceeded", err)
	}
	if provider.calls != 0 {
		t.Fatalf("provider called %d times, want 0", provider.calls)
	}
	if n := len(ledger.snapshot()); n != 1 {
		t.Fatalf("ledger has %d records, want only the pre-existing one", n)
	}
}

func TestCallRecordsCostAsTokensTimesPrice(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{"agent": 10_000_000}}
	provider := &fakeProvider{usage: Usage{TokensIn: 1_000_000, TokensOut: 500_000}}
	prices := PriceTable{
		Version: "2026-11-01",
		Models: map[string]Price{
			"fake/model-a": {InputPerMillion: 3_000_000, OutputPerMillion: 15_000_000},
		},
	}
	g := newTestGateway(t, ledger, provider, prices, nil)

	if _, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("model-a", 10), nil); err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	rec := ledger.snapshot()[0]
	if rec.CostMicros == nil || *rec.CostMicros != 10_500_000 {
		t.Fatalf("CostMicros = %v, want 10500000 (1M*3 + 0.5M*15)", rec.CostMicros)
	}
	if rec.CostStatus != CostKnown {
		t.Fatalf("CostStatus = %q, want %q", rec.CostStatus, CostKnown)
	}
	if rec.PricingVersion != "2026-11-01" {
		t.Fatalf("PricingVersion = %q, want the table version", rec.PricingVersion)
	}
	if rec.ID == "" || rec.Caller.AgentID != "agent" {
		t.Fatalf("record = %+v, want id and caller set", rec)
	}
}

func TestCallRecordsUnknownPriceAsNullNotZero(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{"agent": 1000}}
	provider := &fakeProvider{usage: Usage{TokensIn: 10, TokensOut: 20}}
	g := newTestGateway(t, ledger, provider, PriceTable{}, nil)

	if _, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("unpriced", 10), nil); err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	rec := ledger.snapshot()[0]
	if rec.CostMicros != nil {
		t.Fatalf("CostMicros = %d, want nil for unpriced model", *rec.CostMicros)
	}
	if rec.CostStatus != CostUnknown {
		t.Fatalf("CostStatus = %q, want %q", rec.CostStatus, CostUnknown)
	}
}

func TestCallMarksEstimatedTokensAsEstimatedCost(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{"agent": 1000}}
	provider := &fakeProvider{usage: Usage{TokensIn: 1_000_000, Estimated: true}}
	prices := PriceTable{Version: "fixture", Models: map[string]Price{"fake/m": {InputPerMillion: 2_000_000}}}
	g := newTestGateway(t, ledger, provider, prices, nil)

	if _, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil); err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	rec := ledger.snapshot()[0]
	if rec.CostStatus != CostEstimated || rec.CostMicros == nil || *rec.CostMicros != 2_000_000 {
		t.Fatalf("record cost = (%q, %v), want (estimated, 2000000)", rec.CostStatus, rec.CostMicros)
	}
}

func TestCallChargesEstimateWhenProviderFailsAndReleasesReservation(t *testing.T) {
	boom := errors.New("provider unavailable")
	ledger := &memLedger{limits: map[string]int64{"agent": 20}}
	provider := &fakeProvider{err: boom}
	g := newTestGateway(t, ledger, provider, PriceTable{}, nil)

	_, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil)
	if !errors.Is(err, boom) {
		t.Fatalf("Call() error = %v, want provider error", err)
	}
	rec := ledger.snapshot()[0]
	if rec.TokensOut != 10 || rec.TokensIn != 0 {
		t.Fatalf("failed call recorded tokens in=%d out=%d, want the estimate 0/10", rec.TokensIn, rec.TokensOut)
	}

	// 10 charged + 10 requested = 20 fits the limit only if the failed call's
	// reservation was released.
	provider.err = nil
	if _, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil); err != nil {
		t.Fatalf("second Call() error = %v, want reservation released", err)
	}
}

func TestCallRefusesConcurrentCallThatWouldOverspend(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{"agent": 15}}
	provider := &fakeProvider{started: make(chan struct{}), release: make(chan struct{})}
	g := newTestGateway(t, ledger, provider, PriceTable{}, nil)

	done := make(chan error, 1)
	go func() {
		_, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil)
		done <- err
	}()
	<-provider.started

	// The first call holds 10 reserved tokens; a second 10-token call would
	// reach 20 > 15 even though nothing is recorded yet.
	_, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil)
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("second Call() error = %v, want ErrBudgetExceeded while first is in flight", err)
	}

	close(provider.release)
	if err := <-done; err != nil {
		t.Fatalf("first Call() error = %v", err)
	}
}

func TestCallRefusesUnknownAgentBeforeProvider(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{}}
	provider := &fakeProvider{}
	g := newTestGateway(t, ledger, provider, PriceTable{}, nil)

	_, err := g.Call(context.Background(), Caller{AgentID: "ghost"}, fakeRequest("m", 10), nil)

	if !errors.Is(err, ErrUnknownAgent) {
		t.Fatalf("Call() error = %v, want ErrUnknownAgent", err)
	}
	if provider.calls != 0 {
		t.Fatalf("provider called %d times, want 0", provider.calls)
	}
}

func TestCallEmitsThresholdsOnceAt80And100(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{"agent": 100}}
	provider := &fakeProvider{}
	var events []ThresholdEvent
	g := newTestGateway(t, ledger, provider, PriceTable{}, func(e ThresholdEvent) {
		events = append(events, e)
	})

	// Actual usage per call: 40, then 40 (total 80), then 20 (total 100).
	for _, used := range []int64{40, 40, 20} {
		provider.usage = Usage{TokensIn: used / 2, TokensOut: used - used/2}
		if _, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil); err != nil {
			t.Fatalf("Call() error = %v", err)
		}
	}

	if len(events) != 2 {
		t.Fatalf("got %d threshold events %+v, want 2 (80%% and 100%%)", len(events), events)
	}
	if events[0].Percent != 80 || events[0].UsedTokens != 80 || events[0].LimitTokens != 100 {
		t.Fatalf("first event = %+v, want 80%% at 80 of 100", events[0])
	}
	if events[1].Percent != 100 || events[1].UsedTokens != 100 {
		t.Fatalf("second event = %+v, want 100%% at 100", events[1])
	}
}

func TestNewRejectsDuplicateProviderNames(t *testing.T) {
	_, err := New(Config{
		Ledger:    &memLedger{},
		Providers: []Provider{&fakeProvider{}, &fakeProvider{}},
		Now:       func() time.Time { return testNow },
		NewID:     func(time.Time) (string, error) { return "", nil },
	})
	if err == nil {
		t.Fatal("New() accepted two providers named fake")
	}
}

func TestUsageIDFailureBlocksBeforeDispatch(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{"agent": 100}}
	provider := &fakeProvider{err: ErrUnavailable}
	g, err := New(Config{
		Ledger: ledger, Providers: []Provider{provider},
		Now:   func() time.Time { return testNow },
		NewID: func(time.Time) (string, error) { return "", errors.New("id failure") },
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil)
	if !errors.Is(err, ErrLedger) || provider.calls != 0 || len(ledger.snapshot()) != 0 {
		t.Fatalf("ledger ID failure must prevent dispatch: %v, calls=%d", err, provider.calls)
	}
}

func TestUnknownEffectsAreConservative(t *testing.T) {
	ledger := &memLedger{limits: map[string]int64{"agent": 100}}
	provider := &fakeProvider{err: ErrUnavailable}
	g := newTestGateway(t, ledger, provider, PriceTable{}, nil)
	_, err := g.Call(context.Background(), Caller{AgentID: "agent"}, fakeRequest("m", 10), nil)
	if !errors.Is(err, ErrUnavailable) || !AttemptMayHaveEffects(err) {
		t.Fatalf("unknown adapter must preserve failure and possible effects: %v", err)
	}
}
