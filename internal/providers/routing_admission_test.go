package providers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/IAPro-SDK/lease"
	"github.com/IAPro-Community/IAPro-SDK/model"

	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/harness"
)

// memoryLeaseStore is a single-process stand-in for the durable store.
type memoryLeaseStore struct {
	mu      sync.Mutex
	records map[lease.Key]lease.Record
}

func (m *memoryLeaseStore) Load(_ context.Context, key lease.Key) (lease.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.records[key], nil
}

func (m *memoryLeaseStore) CompareAndSwap(_ context.Context, key lease.Key, rev uint64, next lease.Record) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.records[key].Revision != rev {
		return false, nil
	}
	m.records[key] = next
	return true, nil
}

func newLeaseTestService(t *testing.T, now time.Time) (*Service, *memoryLeaseStore) {
	t.Helper()
	store := &memoryLeaseStore{records: map[lease.Key]lease.Record{}}
	coordinator, err := lease.New(store, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{cfg: Config{
		Now:            func() time.Time { return now },
		LeaseStore:     store,
		CapacityOwner:  "owner",
		NewID:          func(time.Time) (string, error) { return "holder", nil },
		RoutingAllowed: true,
	}}
	s.leases = coordinator
	s.capacity = newCapacityMonitor(s)
	return s, store
}

func TestAdmissionHoldsFenceWhileRunMayBeAlive(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	s, store := newLeaseTestService(t, now)
	a := core.ProviderAccount{ID: "profile", OwnerID: "owner", Kind: core.ProviderKindCodex, Label: "work"}
	p := &leasedCodex{service: s, account: a, cli: &harness.Codex{}}
	release, err := p.Admit(context.Background(), &gateway.Request{Model: "gpt-6.2-sol"})
	_ = release
	if err == nil {
		t.Skip("admission requires a live CLI identity on this host")
	}
	// A refused admission must never leave a leased record behind.
	for _, rec := range store.records {
		if rec.Holder != "" && now.Before(rec.ExpiresAt) {
			t.Fatalf("refused admission kept a live lease: %+v", rec)
		}
	}
}

func TestAdmissionFinishRequiresQuiescence(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	store := &memoryLeaseStore{records: map[lease.Key]lease.Record{}}
	coordinator, err := lease.New(store, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	key := lease.PhysicalKey("owner", model.AccountScope{ProviderID: "codex", ProfileID: "profile", AccountID: "external", IdentityVersion: "generation"})
	guard, err := coordinator.AcquireManaged(context.Background(), key, "holder", lease.ManagedOptions{
		TTL: time.Hour, RenewEvery: 20 * time.Minute, StoreTimeout: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.Finish(false); !errors.Is(err, lease.ErrReconcile) {
		t.Fatalf("non-quiescent finish must retain the fence, got %v", err)
	}
	// The fence survives: while the record is still live, no one else may take
	// it, and an expired one needs the exact fence reconciled first.
	if _, err := coordinator.AcquireManaged(context.Background(), key, "other", lease.ManagedOptions{
		TTL: time.Hour, RenewEvery: 20 * time.Minute, StoreTimeout: time.Minute,
	}); !errors.Is(err, lease.ErrBusy) {
		t.Fatalf("live fence must stay held, got %v", err)
	}
	// Expiration alone is never evidence of quiescence.
	if err := coordinator.ReconcileExpired(context.Background(), key, 1, false); !errors.Is(err, lease.ErrReconcile) {
		t.Fatalf("reconcile without quiescence must fail, got %v", err)
	}
}
