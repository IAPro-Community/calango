package vault

import (
	"context"
	"testing"
)

func TestRotateRejectsWrongRestoredKeyWithSameVersion(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepo()
	original := newTestService(t, repo, testKeys(t, map[int]byte{1: 0x11}))
	if _, err := original.Set(ctx, "test-secret", "synthetic-value"); err != nil {
		t.Fatal(err)
	}
	before := repo.items[key("owner-1", "test-secret")]
	wrong := newTestService(t, repo, testKeys(t, map[int]byte{1: 0x22}))
	if n, err := wrong.Rotate(ctx); err == nil || n != 0 {
		t.Fatalf("Rotate with wrong key = (%d, %v), want failure", n, err)
	}
	after := repo.items[key("owner-1", "test-secret")]
	if before.Ciphertext != after.Ciphertext || before.KeyVersion != after.KeyVersion {
		t.Fatal("failed recovery modified the original envelope")
	}
	if n, err := original.Rotate(ctx); err != nil || n != 0 {
		t.Fatalf("Rotate with correct key = (%d, %v), want success without rewrap", n, err)
	}
	got, err := original.Reveal(ctx, "test-secret")
	if err != nil || got != "synthetic-value" {
		t.Fatal("correct key did not restore the synthetic secret")
	}
}
