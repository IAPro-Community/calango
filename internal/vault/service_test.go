package vault

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
)

// memRepo is an in-memory Repo keyed by owner and name. It records every envelope it
// receives so tests can check that no plain value reached storage.
type memRepo struct {
	items    map[string]Stored
	envelope []string
}

func newMemRepo() *memRepo { return &memRepo{items: map[string]Stored{}} }

func key(owner, name string) string { return owner + "/" + name }

func (r *memRepo) PutItem(_ context.Context, owner string, item Stored, at time.Time) (Item, error) {
	r.envelope = append(r.envelope, item.Ciphertext)
	k := key(owner, item.Name)
	created := at
	if existing, ok := r.items[k]; ok {
		created = existing.CreatedAt
	}
	item.CreatedAt, item.UpdatedAt = created, at
	r.items[k] = item
	return item.Item, nil
}

func (r *memRepo) GetItem(_ context.Context, owner, name string) (Stored, error) {
	item, ok := r.items[key(owner, name)]
	if !ok {
		return Stored{}, core.ErrNotFound
	}
	return item, nil
}

func (r *memRepo) ListItems(_ context.Context, owner string) ([]Item, error) {
	var out []Item
	for k, item := range r.items {
		if strings.HasPrefix(k, owner+"/") {
			out = append(out, item.Item)
		}
	}
	return out, nil
}

func (r *memRepo) ListStored(_ context.Context, owner string) ([]Stored, error) {
	var out []Stored
	for k, item := range r.items {
		if strings.HasPrefix(k, owner+"/") {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *memRepo) ReplaceCiphertext(_ context.Context, owner, name, ciphertext string, keyVersion int) error {
	k := key(owner, name)
	item, ok := r.items[k]
	if !ok {
		return core.ErrNotFound
	}
	item.Ciphertext, item.KeyVersion = ciphertext, keyVersion
	r.items[k] = item
	return nil
}

func (r *memRepo) DeleteItem(_ context.Context, owner, name string) error {
	k := key(owner, name)
	if _, ok := r.items[k]; !ok {
		return core.ErrNotFound
	}
	delete(r.items, k)
	return nil
}

func newTestService(t *testing.T, repo *memRepo, keys MasterKeys) *Service {
	t.Helper()
	fixed := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	svc, err := NewService(repo, keys, "owner-1", &recordingAudit{}, func() time.Time { return fixed }, fixedID)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func TestServiceSetRevealDelete(t *testing.T) {
	repo := newMemRepo()
	svc := newTestService(t, repo, testKeys(t, map[int]byte{1: 0x11}))
	ctx := context.Background()

	item, err := svc.Set(ctx, "openai-key", "sk-canary-987654")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if item.Name != "openai-key" || item.CreatedAt.IsZero() {
		t.Fatalf("Set returned %+v", item)
	}
	got, err := svc.Reveal(ctx, "openai-key")
	if err != nil || got != "sk-canary-987654" {
		t.Fatalf("Reveal = %q, %v", got, err)
	}
	if err := svc.Delete(ctx, "openai-key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Reveal(ctx, "openai-key"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("Reveal after Delete: err = %v, want core.ErrNotFound", err)
	}
}

func TestServiceNeverStoresPlainValue(t *testing.T) {
	repo := newMemRepo()
	svc := newTestService(t, repo, testKeys(t, map[int]byte{1: 0x11}))
	if _, err := svc.Set(context.Background(), "token", "canary-plain-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	for _, ciphertext := range repo.envelope {
		if strings.Contains(ciphertext, "canary-plain-value") {
			t.Fatal("repository received the plain value")
		}
	}
}

func TestServiceRejectsBadNameAndEmptyValue(t *testing.T) {
	svc := newTestService(t, newMemRepo(), testKeys(t, map[int]byte{1: 0x11}))
	ctx := context.Background()
	for _, name := range []string{"", "Upper", "has space", "-leading", strings.Repeat("a", 64)} {
		if _, err := svc.Set(ctx, name, "value"); !errors.Is(err, ErrInvalidName) {
			t.Fatalf("Set(%q): err = %v, want ErrInvalidName", name, err)
		}
	}
	if _, err := svc.Set(ctx, "token", ""); !errors.Is(err, ErrEmptyValue) {
		t.Fatalf("Set with empty value: err = %v, want ErrEmptyValue", err)
	}
}

func TestServiceListReturnsNamesAndDatesOnly(t *testing.T) {
	svc := newTestService(t, newMemRepo(), testKeys(t, map[int]byte{1: 0x11}))
	ctx := context.Background()
	for _, name := range []string{"b-token", "a-token"} {
		if _, err := svc.Set(ctx, name, "value-"+name); err != nil {
			t.Fatalf("Set: %v", err)
		}
	}
	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List returned %d items, want 2", len(items))
	}
}

func TestServiceRotateRewrapsOldVersionOnly(t *testing.T) {
	repo := newMemRepo()
	ctx := context.Background()
	oldKeys := testKeys(t, map[int]byte{1: 0x11})
	if _, err := newTestService(t, repo, oldKeys).Set(ctx, "token", "rotate-me"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	bothKeys := testKeys(t, map[int]byte{1: 0x11, 2: 0x22})
	svc := newTestService(t, repo, bothKeys)
	rotated, err := svc.Rotate(ctx)
	if err != nil || rotated != 1 {
		t.Fatalf("Rotate = %d, %v; want 1, nil", rotated, err)
	}
	if again, err := svc.Rotate(ctx); err != nil || again != 0 {
		t.Fatalf("second Rotate = %d, %v; want 0, nil", again, err)
	}
	if got, err := svc.Reveal(ctx, "token"); err != nil || got != "rotate-me" {
		t.Fatalf("Reveal after Rotate = %q, %v", got, err)
	}
	if item := repo.items[key("owner-1", "token")]; item.KeyVersion != 2 {
		t.Fatalf("key version after Rotate = %d, want 2", item.KeyVersion)
	}
}

func TestServiceRequiresOwner(t *testing.T) {
	if _, err := NewService(newMemRepo(), testKeys(t, map[int]byte{1: 0x11}), "", &recordingAudit{}, nil, fixedID); err == nil {
		t.Fatal("NewService accepted an empty owner")
	}
}
