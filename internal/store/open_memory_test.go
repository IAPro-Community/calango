package store

import (
	"context"
	"testing"
	"time"

	"calango/internal/core"
)

func TestOpenMemoryCreatesIsolatedDatabases(t *testing.T) {
	ctx := context.Background()
	first, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open first memory store: %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })

	second, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open second memory store: %v", err)
	}
	t.Cleanup(func() { _ = second.Close() })

	now := time.Unix(1, 0).UTC()
	accountID, err := core.NewID(now)
	if err != nil {
		t.Fatalf("create account ID: %v", err)
	}
	if err := first.InsertProviderAccount(ctx, core.ProviderAccount{
		ID: accountID, OwnerID: "owner", Kind: "codex", Label: "default",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("insert account into first store: %v", err)
	}

	accounts, err := second.ListProviderAccounts(ctx, "owner")
	if err != nil {
		t.Fatalf("list accounts from second store: %v", err)
	}
	if len(accounts) != 0 {
		t.Fatalf("second store saw %d account(s) from the first store", len(accounts))
	}
}
