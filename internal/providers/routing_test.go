package providers

import (
	"context"
	"testing"
	"time"

	"github.com/IAPro-Community/IAPro-SDK/model"

	"calango/internal/core"
	"calango/internal/gateway"
)

func TestAccountLeaseKeyUsesPhysicalIdentity(t *testing.T) {
	a := model.AccountScope{ProviderID: "codex", AccountID: "external", ProfileID: "profile-a", IdentityVersion: "v1"}
	b := a
	b.ProfileID, b.IdentityVersion = "profile-b", "v2"
	if accountLeaseKey("owner", a) != accountLeaseKey("owner", b) {
		t.Fatal("aliases or credential rotation created a second lease")
	}
	if accountLeaseKey("owner", a) == accountLeaseKey("other", a) {
		t.Fatal("owner isolation lost")
	}
}

func TestStrictRoutingDisabledFailsClosed(t *testing.T) {
	s := &Service{cfg: Config{Now: time.Now, Prices: gateway.PriceTable{}}}
	agent := core.Agent{OwnerID: "owner", CapacityRouting: true, Providers: []core.ProviderRef{{AccountID: "a"}, {AccountID: "b"}}}
	if _, route, err := s.SelectCapacityRoute(context.Background(), agent, "run"); err == nil || route != nil {
		t.Fatal("disabled capacity router admitted a run")
	}
}
