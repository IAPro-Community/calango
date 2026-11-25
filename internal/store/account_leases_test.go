package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/IAPro-SDK/lease"
)

func TestAccountLeasePersistsFenceAcrossRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "calango.db")
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	key := lease.Key{OwnerID: "owner-a", Provider: "codex", AccountID: "verified-account"}
	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	coordinator, err := lease.New(db.AccountLeases(), clock)
	if err != nil {
		t.Fatal(err)
	}
	first, err := coordinator.Acquire(ctx, key, "run-a", time.Minute, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	coordinator, err = lease.New(reopened.AccountLeases(), clock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.Acquire(ctx, key, "run-b", time.Minute, 0); !errors.Is(err, lease.ErrBusy) {
		t.Fatalf("active lease after restart: %v", err)
	}
	now = now.Add(time.Minute)
	if _, err := coordinator.Acquire(ctx, key, "run-b", time.Minute, 0); !errors.Is(err, lease.ErrReconcile) {
		t.Fatalf("expired lease without reconciliation: %v", err)
	}
	second, err := coordinator.Acquire(ctx, key, "run-b", time.Minute, first.Fence)
	if err != nil || second.Fence != first.Fence+1 {
		t.Fatalf("reconciled acquisition: %+v, %v", second, err)
	}
	if err := coordinator.Release(ctx, key, first, true); !errors.Is(err, lease.ErrStale) {
		t.Fatalf("stale holder release: %v", err)
	}
	if err := coordinator.Release(ctx, key, second, false); !errors.Is(err, lease.ErrReconcile) {
		t.Fatalf("release without quiescence: %v", err)
	}
	if err := coordinator.Release(ctx, key, second, true); err != nil {
		t.Fatal(err)
	}
	third, err := coordinator.Acquire(ctx, key, "run-c", time.Minute, 0)
	if err != nil || third.Fence != second.Fence+1 {
		t.Fatalf("acquisition after release: %+v, %v", third, err)
	}
}

func TestAccountLeaseConcurrentCASAndOwnerIsolation(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "calango.db")
	a, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	b, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	key := lease.Key{OwnerID: "owner-a", Provider: "codex", AccountID: "same-account"}
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	record := lease.Record{Holder: "run", Fence: 1, Revision: 1, ExpiresAt: now.Add(time.Minute)}
	type outcome struct {
		ok  bool
		err error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	for _, db := range []*Store{a, b} {
		go func(s *Store) {
			<-start
			ok, err := s.AccountLeases().CompareAndSwap(ctx, key, 0, record)
			results <- outcome{ok, err}
		}(db)
	}
	close(start)
	wins := 0
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.ok {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("CAS winners = %d, want 1", wins)
	}
	other := key
	other.OwnerID = "owner-b"
	if got, err := b.AccountLeases().Load(ctx, other); err != nil || got != (lease.Record{}) {
		t.Fatalf("other owner: %+v, %v", got, err)
	}
	if ok, err := b.AccountLeases().CompareAndSwap(ctx, other, 0, record); err != nil || !ok {
		t.Fatalf("other owner CAS: %t, %v", ok, err)
	}
	next := record
	next.Revision = 2
	next.Fence = 2
	if ok, err := a.AccountLeases().CompareAndSwap(ctx, key, 1, next); err != nil || !ok {
		t.Fatalf("advance CAS: %t, %v", ok, err)
	}
	stale := record
	stale.Revision = 3
	if ok, err := b.AccountLeases().CompareAndSwap(ctx, key, 2, stale); err != nil || ok {
		t.Fatalf("regressed fence CAS: %t, %v", ok, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if ok, err := b.AccountLeases().CompareAndSwap(canceled, key, 2, stale); err == nil || ok {
		t.Fatalf("canceled CAS: %t, %v", ok, err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if ok, err := b.AccountLeases().CompareAndSwap(ctx, key, 2, stale); err == nil || ok {
		t.Fatalf("closed store CAS: %t, %v", ok, err)
	}
	got, err := a.AccountLeases().Load(ctx, key)
	if err != nil || got.Fence != 2 || got.Revision != 2 {
		t.Fatalf("persisted generation: %+v, %v", got, err)
	}
}
