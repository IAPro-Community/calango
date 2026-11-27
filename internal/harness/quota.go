package harness

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/IAPro-SDK/model"
	sdk "github.com/IAPro-Community/IAPro-SDK/providers"
	sdkcodex "github.com/IAPro-Community/IAPro-SDK/providers/codex"
)

// A dedicated home is coordinated inside this daemon. This is not an OS-wide
// lock: opt-in requires an exclusive Calango-managed login, not a shared CLI home.
type homeSlot struct {
	token chan struct{}
	refs  int
}

var quotaHomes = struct {
	sync.Mutex
	slots map[string]*homeSlot
}{slots: make(map[string]*homeSlot)}

func lockQuotaHome(ctx context.Context, home string) (func(), error) {
	home, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, errors.New("quota home unavailable")
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return nil, errors.New("quota home unavailable")
	}
	if runtime.GOOS == "windows" {
		home = strings.ToLower(home)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	quotaHomes.Lock()
	slot := quotaHomes.slots[home]
	if slot == nil {
		slot = &homeSlot{token: make(chan struct{}, 1)}
		quotaHomes.slots[home] = slot
	}
	slot.refs++
	quotaHomes.Unlock()
	drop := func() {
		quotaHomes.Lock()
		defer quotaHomes.Unlock()
		slot.refs--
		if slot.refs == 0 {
			delete(quotaHomes.slots, home)
		}
	}
	select {
	case slot.token <- struct{}{}:
		return func() { <-slot.token; drop() }, nil
	case <-ctx.Done():
		drop()
		return nil, ctx.Err()
	}
}

// QuotaAdapter reads only the dedicated, explicitly authorized Codex home.
// No secrets, provider identity or auth generation are returned to the client.
type QuotaAdapter struct {
	Codex *Codex
	Now   func() time.Time
}

const maxQuotaAuthBytes = 4 * 1024 * 1024

func (q QuotaAdapter) shared() (*sdkcodex.DedicatedQuota, error) {
	if q.Codex == nil || q.Codex.home == "" {
		return nil, sdk.ErrUnsupported
	}
	now := q.Now
	if now == nil {
		now = time.Now
	}
	return sdkcodex.NewDedicatedQuota(sdkcodex.DedicatedQuotaConfig{
		Home:     q.Codex.home,
		ReadAuth: q.readAuth,
		Lock: func(ctx context.Context) (func(), error) {
			return lockQuotaHome(ctx, q.Codex.home)
		},
		Login: func(ctx context.Context) (*exec.Cmd, error) {
			return q.Codex.commandWithHome(ctx, q.Codex.home, "login", "status"), nil
		},
		AppServer: func(ctx context.Context) (*exec.Cmd, error) {
			cmd := q.Codex.commandWithHome(ctx, q.Codex.home, "app-server")
			cmd.Dir = q.Codex.workDir
			return cmd, nil
		},
		SameHome: sameQuotaHome,
		Now:      now,
	})
}

// readAuth enforces Calango's authorized profile filesystem boundary. The
// returned owned buffer is wiped by the SDK identity adapter.
func (q QuotaAdapter) readAuth(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	homeInfo, err := os.Lstat(q.Codex.home)
	if err != nil || !homeInfo.IsDir() || homeInfo.Mode()&os.ModeSymlink != 0 {
		return nil, sdk.ErrIdentityMismatch
	}
	resolved, err := filepath.EvalSymlinks(q.Codex.home)
	if err != nil || !sameQuotaHome(resolved, q.Codex.home) {
		return nil, sdk.ErrIdentityMismatch
	}
	path := filepath.Join(resolved, "auth.json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxQuotaAuthBytes {
		return nil, sdk.ErrIdentityMismatch
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, sdk.ErrIdentityMismatch
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, sdk.ErrIdentityMismatch
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxQuotaAuthBytes+1))
	if err != nil || len(raw) > maxQuotaAuthBytes {
		clear(raw)
		return nil, sdk.ErrIdentityMismatch
	}
	return raw, nil
}

func (q QuotaAdapter) Identity(owner, profile string) (sdk.Account, error) {
	adapter, err := q.shared()
	if err != nil {
		return sdk.Account{}, err
	}
	return adapter.Identity(context.Background(), owner, profile)
}

func (q QuotaAdapter) InspectAuth(ctx context.Context, account sdk.Account) (sdk.Authentication, error) {
	adapter, err := q.shared()
	if err != nil {
		return sdk.Authentication{}, err
	}
	return adapter.InspectAuth(ctx, account)
}

func (q QuotaAdapter) GetUsage(ctx context.Context, account sdk.Account) (model.UsageSnapshot, error) {
	adapter, err := q.shared()
	if err != nil {
		return model.UsageSnapshot{}, err
	}
	return adapter.GetUsage(ctx, account)
}

func sameQuotaHome(a, b string) bool {
	x, err := os.Stat(a)
	if err != nil || !x.IsDir() {
		return false
	}
	y, err := os.Stat(b)
	return err == nil && y.IsDir() && os.SameFile(x, y)
}
