package harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	parser "github.com/IAPro-Community/IAPro-SDK/providers/parsers/codex"
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

func (q QuotaAdapter) Identity(owner, profile string) (sdk.Account, error) {
	if q.Codex == nil || q.Codex.home == "" || owner == "" || profile == "" {
		return sdk.Account{}, sdk.ErrUnsupported
	}
	resolved, err := filepath.EvalSymlinks(q.Codex.home)
	if err != nil || !sameQuotaHome(resolved, q.Codex.home) {
		return sdk.Account{}, sdk.ErrIdentityMismatch
	}
	path := filepath.Join(resolved, "auth.json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxQuotaAuthBytes {
		return sdk.Account{}, sdk.ErrIdentityMismatch
	}
	f, err := os.Open(path)
	if err != nil {
		return sdk.Account{}, sdk.ErrIdentityMismatch
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return sdk.Account{}, sdk.ErrIdentityMismatch
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxQuotaAuthBytes+1))
	defer clear(raw)
	if err != nil || len(raw) > maxQuotaAuthBytes {
		return sdk.Account{}, sdk.ErrIdentityMismatch
	}
	id, err := sdkcodex.ExternalAccountID(raw)
	if err != nil {
		return sdk.Account{}, sdk.ErrIdentityMismatch
	}
	generation := sha256.Sum256(raw)
	return sdk.Account{OwnerID: owner, Scope: model.AccountScope{
		ProviderID: string(model.ProviderCodex), ProfileID: profile, AccountID: id,
		IdentityVersion: hex.EncodeToString(generation[:]), CredentialScope: "dedicated-codex-home",
	}}, nil
}

func (q QuotaAdapter) InspectAuth(ctx context.Context, account sdk.Account) (sdk.Authentication, error) {
	release, err := lockQuotaHome(ctx, q.Codex.home)
	if err != nil {
		return sdk.Authentication{}, err
	}
	defer release()
	current, err := q.Identity(account.OwnerID, account.Scope.ProfileID)
	if err != nil || current != account {
		return sdk.Authentication{}, sdk.ErrIdentityMismatch
	}
	status, err := q.Codex.Available(ctx)
	if err != nil || !status.Installed || !status.SignedIn {
		return sdk.Authentication{}, sdk.ErrIdentityMismatch
	}
	return sdk.Authentication{Scope: account.Scope, Authenticated: true, ObservedAt: q.Now()}, nil
}

func (q QuotaAdapter) GetUsage(ctx context.Context, account sdk.Account) (model.UsageSnapshot, error) {
	release, err := lockQuotaHome(ctx, q.Codex.home)
	if err != nil {
		return model.UsageSnapshot{}, err
	}
	defer release()
	current, err := q.Identity(account.OwnerID, account.Scope.ProfileID)
	if err != nil || current != account {
		return model.UsageSnapshot{}, sdk.ErrIdentityMismatch
	}
	rpc, err := sdkcodex.StartStdioRPC(ctx, func(runCtx context.Context) (*exec.Cmd, error) {
		cmd := q.Codex.commandWithHome(runCtx, q.Codex.home, "app-server")
		cmd.Dir = q.Codex.workDir
		return cmd, nil
	})
	if err != nil {
		return model.UsageSnapshot{}, err
	}
	snap, err := parser.Probe(ctx, rpc, account.Scope, account.Scope.AccountID, q.Codex.home, sameQuotaHome, q.Now)
	if err != nil {
		return model.UsageSnapshot{}, err
	}
	current, err = q.Identity(account.OwnerID, account.Scope.ProfileID)
	if err != nil || current != account {
		return model.UsageSnapshot{}, sdk.ErrIdentityMismatch
	}
	return snap, nil
}

func sameQuotaHome(a, b string) bool {
	x, err := os.Stat(a)
	if err != nil || !x.IsDir() {
		return false
	}
	y, err := os.Stat(b)
	return err == nil && y.IsDir() && os.SameFile(x, y)
}
