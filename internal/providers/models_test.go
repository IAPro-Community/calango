package providers

import (
	"context"
	"testing"

	"calango/internal/core"
)

type catalogRepo struct {
	accounts []core.ProviderAccount
}

func (r catalogRepo) ListProviderAccounts(context.Context, string) ([]core.ProviderAccount, error) {
	return r.accounts, nil
}

func (r catalogRepo) InsertProviderAccount(context.Context, core.ProviderAccount) error {
	return nil
}

func (r catalogRepo) GetProviderAccountByName(context.Context, string, string, string) (core.ProviderAccount, error) {
	return core.ProviderAccount{}, core.ErrNotFound
}

func (r catalogRepo) DeleteProviderAccount(context.Context, string, string) error {
	return nil
}

func TestCodexModelsRejectsUnknownAccount(t *testing.T) {
	s := &Service{cfg: Config{Repo: catalogRepo{accounts: []core.ProviderAccount{
		{ID: "a", OwnerID: "owner", Kind: core.ProviderKindAnthropic, Label: "default"},
	}}}}
	if _, err := s.CodexModels(context.Background(), "owner", "missing"); err == nil {
		t.Fatal("unknown account must not be probed")
	}
	models, err := s.CodexModels(context.Background(), "owner", "a")
	if err != nil || len(models) != 0 {
		t.Fatalf("anthropic account must have no catalog: %v %v", models, err)
	}
}
