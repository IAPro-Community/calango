package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	sdkcodex "github.com/IAPro-Community/IAPro-SDK/providers/codex"
	codexparser "github.com/IAPro-Community/IAPro-SDK/providers/parsers/codex"

	"calango/internal/core"
	"calango/internal/harness"
)

// Get returns one account owned by ownerID.
func (s *Service) Get(ctx context.Context, ownerID, id string) (View, error) {
	accounts, err := s.cfg.Repo.ListProviderAccounts(ctx, ownerID)
	if err != nil {
		return View{}, err
	}
	for _, a := range accounts {
		if a.ID == id {
			return s.view(a), nil
		}
	}
	return View{}, fmt.Errorf("%w: provider account %s", core.ErrNotFound, id)
}

// CodexModels lists the models the account's CLI can use. The catalog comes
// from the CLI itself (read-only probe), never from a table duplicated here.
// The probe runs with the account's isolated home so the answer matches the
// login that will execute the turn.
func (s *Service) CodexModels(ctx context.Context, ownerID, id string) ([]codexparser.Model, error) {
	view, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}
	if view.Kind != core.ProviderKindCodex {
		return nil, nil
	}
	cli, err := harness.NewCodex(s.cfg.CodexBin, s.cfg.CodexWorkDir, s.codexHome(view.ProviderAccount))
	if err != nil {
		return nil, err
	}
	env := append(os.Environ(), cli.Env()...)
	launch := sdkcodex.LaunchCatalog(cli.Bin(), env, s.cfg.CodexWorkDir, sdkcodex.CatalogBundled)
	cmd, err := launch(ctx)
	if err != nil || cmd == nil {
		return nil, fmt.Errorf("%w: codex model catalog", core.ErrInvalidInput)
	}
	out, err := runProbe(cmd)
	if err != nil {
		return nil, fmt.Errorf("%w: codex model catalog", core.ErrInvalidInput)
	}
	models, err := sdkcodex.DecodeCatalog(out)
	if err != nil {
		return nil, fmt.Errorf("%w: codex model catalog", core.ErrInvalidInput)
	}
	return models, nil
}

// runProbe runs a read-only probe. The error stays opaque because CLI output
// can echo paths or tokens from the account environment.
func runProbe(cmd *exec.Cmd) ([]byte, error) {
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("probe exit %d", exitErr.ExitCode())
		}
		return nil, err
	}
	return out, nil
}
