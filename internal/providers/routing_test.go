package providers

import (
	"context"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/gateway"
)

func TestStrictRoutingDisabledFailsClosed(t *testing.T) {
	s := &Service{cfg: Config{Now: time.Now, Prices: gateway.PriceTable{}}}
	agent := core.Agent{OwnerID: "owner", CapacityRouting: true, Providers: []core.ProviderRef{{AccountID: "a"}, {AccountID: "b"}}}
	if _, route, err := s.SelectCapacityRoute(context.Background(), agent, "run"); err == nil || route != nil {
		t.Fatal("disabled capacity router admitted a run")
	}
}
