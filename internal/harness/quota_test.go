package harness

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestQuotaIdentityGenerationAndHomeLock(t *testing.T) {
	home := t.TempDir()
	cli, err := NewCodex("codex", home, home)
	if err != nil {
		t.Fatal(err)
	}
	adapter := QuotaAdapter{Codex: cli, Now: time.Now}
	authPath := filepath.Join(home, "auth.json")
	write := func(value string) {
		t.Helper()
		if err := os.WriteFile(authPath, []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"tokens":{"account_id":"account-a","access_token":"synthetic"}}`)
	a, err := adapter.Identity("owner", "profile")
	if err != nil {
		t.Fatal(err)
	}
	write(`{"tokens":{"account_id":"account-a","access_token":"different-synthetic"}}`)
	b, err := adapter.Identity("owner", "profile")
	if err != nil || a.Scope.AccountID != b.Scope.AccountID || a.Scope.IdentityVersion == b.Scope.IdentityVersion {
		t.Fatal("credential change must change generation")
	}
	release, err := lockQuotaHome(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := lockQuotaHome(ctx, home); err == nil {
		t.Fatal("canceled waiter acquired the home lock")
	}
	release()
	releaseAgain, err := lockQuotaHome(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	releaseAgain()
}
