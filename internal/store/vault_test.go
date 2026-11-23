package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/vault"
)

func TestVaultItemLifecycleKeepsCreatedAtAndRotatesWithoutTouchingDates(t *testing.T) {
	ctx := context.Background()
	s, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer func() { _ = s.Close() }()

	first := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	second := first.Add(time.Hour)
	if _, err := s.PutItem(ctx, "owner-1", vault.Stored{
		Item:       vault.Item{Name: "openai-key"},
		Ciphertext: "env-1",
		KeyVersion: 1,
	}, first); err != nil {
		t.Fatalf("PutItem insert: %v", err)
	}
	item, err := s.PutItem(ctx, "owner-1", vault.Stored{
		Item:       vault.Item{Name: "openai-key"},
		Ciphertext: "env-2",
		KeyVersion: 1,
	}, second)
	if err != nil {
		t.Fatalf("PutItem replace: %v", err)
	}
	if !item.CreatedAt.Equal(first) || !item.UpdatedAt.Equal(second) {
		t.Fatalf("replace dates = created %v updated %v; want %v and %v", item.CreatedAt, item.UpdatedAt, first, second)
	}

	stored, err := s.GetItem(ctx, "owner-1", "openai-key")
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if stored.Ciphertext != "env-2" || stored.KeyVersion != 1 {
		t.Fatalf("GetItem = %+v, want the replaced envelope", stored)
	}

	if err := s.ReplaceCiphertext(ctx, "owner-1", "openai-key", "env-3", 2); err != nil {
		t.Fatalf("ReplaceCiphertext: %v", err)
	}
	rotated, err := s.GetItem(ctx, "owner-1", "openai-key")
	if err != nil {
		t.Fatalf("GetItem after rotate: %v", err)
	}
	if rotated.KeyVersion != 2 || !rotated.UpdatedAt.Equal(second) {
		t.Fatalf("rotation changed the wrong fields: %+v", rotated)
	}

	if err := s.DeleteItem(ctx, "owner-1", "openai-key"); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if _, err := s.GetItem(ctx, "owner-1", "openai-key"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("GetItem after delete: err = %v, want core.ErrNotFound", err)
	}
}

func TestVaultItemsAreScopedByOwnerAndNotFoundMapsToDomainError(t *testing.T) {
	ctx := context.Background()
	s, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer func() { _ = s.Close() }()

	at := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	if _, err := s.PutItem(ctx, "owner-1", vault.Stored{Item: vault.Item{Name: "b-token"}, Ciphertext: "x", KeyVersion: 1}, at); err != nil {
		t.Fatalf("PutItem: %v", err)
	}
	if _, err := s.PutItem(ctx, "owner-1", vault.Stored{Item: vault.Item{Name: "a-token"}, Ciphertext: "y", KeyVersion: 1}, at); err != nil {
		t.Fatalf("PutItem: %v", err)
	}

	items, err := s.ListItems(ctx, "owner-1")
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) != 2 || items[0].Name != "a-token" || items[1].Name != "b-token" {
		t.Fatalf("ListItems = %+v, want a-token then b-token", items)
	}

	other, err := s.ListItems(ctx, "owner-2")
	if err != nil {
		t.Fatalf("ListItems other owner: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("another owner sees %d items", len(other))
	}
	if _, err := s.GetItem(ctx, "owner-2", "a-token"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("GetItem for another owner: err = %v, want core.ErrNotFound", err)
	}
	if err := s.ReplaceCiphertext(ctx, "owner-1", "missing", "z", 1); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("ReplaceCiphertext missing: err = %v, want core.ErrNotFound", err)
	}
	if err := s.DeleteItem(ctx, "owner-1", "missing"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("DeleteItem missing: err = %v, want core.ErrNotFound", err)
	}
}

func TestVaultItemRejectsPlainKeyVersionZero(t *testing.T) {
	ctx := context.Background()
	s, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer func() { _ = s.Close() }()

	at := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	if _, err := s.PutItem(ctx, "owner-1", vault.Stored{Item: vault.Item{Name: "token"}, Ciphertext: "x", KeyVersion: 0}, at); err == nil {
		t.Fatal("PutItem accepted key version 0")
	}
}
