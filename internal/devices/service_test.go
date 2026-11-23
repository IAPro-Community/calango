package devices

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/store"
)

const (
	testOwner = "owner-1"
	testActor = "desk-1"
)

// newTestService wires the service to an in-memory store. The clock is read through
// the pointer, so tests move time by assigning to *clock.
func newTestService(t *testing.T, clock *time.Time) *Service {
	t.Helper()
	database, err := store.OpenMemory(context.Background())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	svc, err := NewService(database, func() time.Time { return *clock }, core.NewID)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

func issueCode(t *testing.T, svc *Service, scope core.DeviceScope) string {
	t.Helper()
	issued, err := svc.CreatePairingCode(context.Background(), testOwner, testActor, scope)
	if err != nil {
		t.Fatalf("create pairing code: %v", err)
	}
	return issued.Code
}

func TestPairingCodeIsSingleUse(t *testing.T) {
	clock := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	svc := newTestService(t, &clock)
	code := issueCode(t, svc, core.ScopeFull)
	ctx := context.Background()

	if _, err := svc.Pair(ctx, code, "Phone", "ios"); err != nil {
		t.Fatalf("first pair: %v", err)
	}
	_, err := svc.Pair(ctx, code, "Tablet", "android")
	if !errors.Is(err, core.ErrUnauthorized) {
		t.Fatalf("second pair with the same code: err = %v, want ErrUnauthorized", err)
	}
}

func TestPairingCodeExpires(t *testing.T) {
	clock := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	svc := newTestService(t, &clock)
	code := issueCode(t, svc, core.ScopeFull)

	clock = clock.Add(core.PairingCodeTTL)
	_, err := svc.Pair(context.Background(), code, "Phone", "ios")
	if !errors.Is(err, core.ErrUnauthorized) {
		t.Fatalf("pair at expiry: err = %v, want ErrUnauthorized", err)
	}
}

func TestWrongCodeIsRejected(t *testing.T) {
	clock := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	svc := newTestService(t, &clock)
	code := issueCode(t, svc, core.ScopeFull)
	other := "B" + code[1:]
	if code[0] == 'B' {
		other = "A" + code[1:]
	}

	_, err := svc.Pair(context.Background(), other, "Phone", "ios")
	if !errors.Is(err, core.ErrUnauthorized) {
		t.Fatalf("pair with wrong code: err = %v, want ErrUnauthorized", err)
	}
}

func TestInvalidInputDoesNotConsumeCode(t *testing.T) {
	clock := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	svc := newTestService(t, &clock)
	code := issueCode(t, svc, core.ScopeFull)
	ctx := context.Background()

	if _, err := svc.Pair(ctx, code, "   ", "ios"); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("blank name: err = %v, want ErrInvalidInput", err)
	}
	if _, err := svc.Pair(ctx, code, "Phone", "tv"); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("unknown platform: err = %v, want ErrInvalidInput", err)
	}
	if _, err := svc.Pair(ctx, code, "Phone", "ios"); err != nil {
		t.Fatalf("pair after rejected input: %v", err)
	}
}

func TestPairingCodeIsNormalized(t *testing.T) {
	clock := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	svc := newTestService(t, &clock)
	code := issueCode(t, svc, core.ScopeFull)
	typed := strings.ToLower(code[:4] + "-" + code[4:] + " ")

	if _, err := svc.Pair(context.Background(), typed, "Phone", "ios"); err != nil {
		t.Fatalf("pair with typed code %q: %v", typed, err)
	}
}

func TestDeviceAuthenticatesAndRevocationStopsIt(t *testing.T) {
	clock := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	svc := newTestService(t, &clock)
	ctx := context.Background()
	paired, err := svc.Pair(ctx, issueCode(t, svc, core.ScopeFull), "Phone", "ios")
	if err != nil {
		t.Fatalf("pair: %v", err)
	}

	dev, err := svc.Authenticate(ctx, paired.Token)
	if err != nil {
		t.Fatalf("authenticate with issued token: %v", err)
	}
	if dev.ID != paired.DeviceID || dev.Scope != core.ScopeFull {
		t.Fatalf("authenticated device = %+v, want id %s scope full", dev, paired.DeviceID)
	}
	if dev.LastSeenAt.IsZero() {
		t.Fatal("authenticate did not record last_seen_at")
	}

	if err := svc.Revoke(ctx, testOwner, testActor, paired.DeviceID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := svc.Authenticate(ctx, paired.Token); !errors.Is(err, core.ErrUnauthorized) {
		t.Fatalf("authenticate after revoke: err = %v, want ErrUnauthorized", err)
	}

	list, err := svc.List(ctx, testOwner)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Active() {
		t.Fatalf("list = %+v, want the one revoked device", list)
	}
}

func TestScopeComesFromPairingCode(t *testing.T) {
	clock := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	svc := newTestService(t, &clock)
	ctx := context.Background()
	paired, err := svc.Pair(ctx, issueCode(t, svc, core.ScopeApproveOnly), "Watch", "web")
	if err != nil {
		t.Fatalf("pair: %v", err)
	}
	if paired.Scope != core.ScopeApproveOnly {
		t.Fatalf("paired scope = %s, want approve_only", paired.Scope)
	}
	dev, err := svc.Authenticate(ctx, paired.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if dev.Scope != core.ScopeApproveOnly {
		t.Fatalf("authenticated scope = %s, want approve_only", dev.Scope)
	}
}

func TestAuthenticateRejectsEmptyAndUnknownTokens(t *testing.T) {
	clock := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	svc := newTestService(t, &clock)
	ctx := context.Background()

	for _, token := range []string{"", "not-a-real-token"} {
		if _, err := svc.Authenticate(ctx, token); !errors.Is(err, core.ErrUnauthorized) {
			t.Fatalf("authenticate(%q): err = %v, want ErrUnauthorized", token, err)
		}
	}
}

func TestRevokeIsScopedToOwner(t *testing.T) {
	clock := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	svc := newTestService(t, &clock)
	ctx := context.Background()
	paired, err := svc.Pair(ctx, issueCode(t, svc, core.ScopeFull), "Phone", "ios")
	if err != nil {
		t.Fatalf("pair: %v", err)
	}

	if err := svc.Revoke(ctx, "other-owner", testActor, paired.DeviceID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("revoke under another owner: err = %v, want ErrNotFound", err)
	}
	if err := svc.Revoke(ctx, testOwner, testActor, paired.DeviceID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := svc.Revoke(ctx, testOwner, testActor, paired.DeviceID); err != nil {
		t.Fatalf("second revoke should be a no-op: %v", err)
	}
}
