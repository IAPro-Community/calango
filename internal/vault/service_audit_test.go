package vault

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
)

// recordingAudit keeps every event it is given. When err is set, it refuses them all.
type recordingAudit struct {
	events []core.AuditEvent
	err    error
}

func (a *recordingAudit) AppendAudit(_ context.Context, ev core.AuditEvent) error {
	if a.err != nil {
		return a.err
	}
	a.events = append(a.events, ev)
	return nil
}

// fixedID returns a 26-character id, the length the audit table requires.
func fixedID(time.Time) (string, error) { return strings.Repeat("0", 26), nil }

func TestServiceAuditsEachActionWithNameOnly(t *testing.T) {
	const secret = "super-secret-value"
	repo := newMemRepo()
	aud := &recordingAudit{}
	svc, err := NewService(repo, testKeys(t, map[int]byte{1: 0x11}), "owner-1", aud, nil, fixedID)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.Set(ctx, "api.key", secret); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := svc.Reveal(ctx, "api.key"); err != nil {
		t.Fatalf("Reveal: %v", err)
	}
	if err := svc.Delete(ctx, "api.key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	want := []string{"vault.set", "vault.reveal", "vault.delete"}
	if len(aud.events) != len(want) {
		t.Fatalf("got %d events, want %d", len(aud.events), len(want))
	}
	for i, ev := range aud.events {
		if ev.Action != want[i] {
			t.Errorf("event %d action = %q, want %q", i, ev.Action, want[i])
		}
		if ev.Subject != "vault:api.key" || ev.ActorKind != "owner" || ev.ActorID != "owner-1" {
			t.Errorf("event %d subject/actor = %q %q %q", i, ev.Subject, ev.ActorKind, ev.ActorID)
		}
		if len(ev.Facts) != 1 || ev.Facts["name"] != "api.key" {
			t.Errorf("event %d facts = %v, want only the name", i, ev.Facts)
		}
		raw, err := json.Marshal(ev)
		if err != nil {
			t.Fatalf("marshal event %d: %v", i, err)
		}
		if strings.Contains(string(raw), secret) {
			t.Errorf("event %d carries the plain value: %s", i, raw)
		}
	}
}

func TestServiceFailsClosedWhenAuditIsRefused(t *testing.T) {
	repo := newMemRepo()
	aud := &recordingAudit{}
	svc, err := NewService(repo, testKeys(t, map[int]byte{1: 0x11}), "owner-1", aud, nil, fixedID)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.Set(ctx, "api.key", "value-1"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	refused := errors.New("audit chain offline")
	aud.err = refused

	if _, err := svc.Set(ctx, "other.key", "value-2"); !errors.Is(err, refused) {
		t.Fatalf("Set error = %v, want audit refusal", err)
	}
	if _, err := repo.GetItem(ctx, "owner-1", "other.key"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Set wrote a secret without an audit record: %v", err)
	}

	value, err := svc.Reveal(ctx, "api.key")
	if !errors.Is(err, refused) || value != "" {
		t.Fatalf("Reveal = (%q, %v), want empty value and audit refusal", value, err)
	}

	if err := svc.Delete(ctx, "api.key"); !errors.Is(err, refused) {
		t.Fatalf("Delete error = %v, want audit refusal", err)
	}
	if _, err := repo.GetItem(ctx, "owner-1", "api.key"); err != nil {
		t.Fatalf("Delete removed the secret without an audit record: %v", err)
	}
}

func TestServiceRequiresAuditorAndIDSource(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11})
	if _, err := NewService(newMemRepo(), keys, "owner-1", nil, nil, fixedID); err == nil {
		t.Fatal("NewService accepted a nil auditor")
	}
	if _, err := NewService(newMemRepo(), keys, "owner-1", &recordingAudit{}, nil, nil); err == nil {
		t.Fatal("NewService accepted a nil id source")
	}
}
