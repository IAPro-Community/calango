package vault

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRotateAuditsEachRewrapWithNameOnly(t *testing.T) {
	repo := newMemRepo()
	ctx := context.Background()
	if _, err := newTestService(t, repo, testKeys(t, map[int]byte{1: 0x11})).Set(ctx, "token", "rotate-me"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	aud := &recordingAudit{}
	svc, err := NewService(repo, testKeys(t, map[int]byte{1: 0x11, 2: 0x22}), "owner-1", aud,
		func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }, fixedID)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if rotated, err := svc.Rotate(ctx); err != nil || rotated != 1 {
		t.Fatalf("Rotate = %d, %v; want 1, nil", rotated, err)
	}
	if len(aud.events) != 1 {
		t.Fatalf("audit events = %d, want one vault.rewrap", len(aud.events))
	}
	ev := aud.events[0]
	if ev.Action != "vault.rewrap" || ev.Subject != "vault:token" {
		t.Fatalf("event = %s %s, want vault.rewrap vault:token", ev.Action, ev.Subject)
	}
	if len(ev.Facts) != 1 || ev.Facts["name"] != "token" {
		t.Fatalf("facts = %v, want the name only", ev.Facts)
	}
}

func TestRotateStoresNothingWhenAuditFails(t *testing.T) {
	repo := newMemRepo()
	ctx := context.Background()
	if _, err := newTestService(t, repo, testKeys(t, map[int]byte{1: 0x11})).Set(ctx, "token", "rotate-me"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	aud := &recordingAudit{err: errors.New("audit disk full")}
	svc, err := NewService(repo, testKeys(t, map[int]byte{1: 0x11, 2: 0x22}), "owner-1", aud,
		func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }, fixedID)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if _, err := svc.Rotate(ctx); err == nil {
		t.Fatal("Rotate succeeded although the audit write failed")
	}
	if item := repo.items[key("owner-1", "token")]; item.KeyVersion != 1 {
		t.Fatalf("key version = %d after failed audit, want the old version 1", item.KeyVersion)
	}
}
