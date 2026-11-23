package store

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestConcurrentAuditAppendsKeepTheChainIntact drives many writers at one file-backed
// database. Every append must succeed (no SQLITE_BUSY surfaces to the caller) and the
// chain must verify with every entry present.
func TestConcurrentAuditAppendsKeepTheChainIntact(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, filepath.Join(t.TempDir(), "calango.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	const writers = 8
	const perWriter = 25
	at := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	errs := make(chan error, writers*perWriter)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				// Audit ids are 26 characters (schema CHECK), so pad the counter.
				id := fmt.Sprintf("%026d", w*perWriter+i)
				if err := st.AppendAudit(ctx, auditEvent(id, at, "concurrency.test")); err != nil {
					errs <- fmt.Errorf("writer %d append %d: %w", w, i, err)
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	v, err := st.VerifyAudit(ctx)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if v.FirstBreak != nil {
		t.Fatalf("chain broke: %+v", *v.FirstBreak)
	}
	if want := writers * perWriter; v.EntriesChecked != want {
		t.Fatalf("entries checked = %d, want %d", v.EntriesChecked, want)
	}
}
