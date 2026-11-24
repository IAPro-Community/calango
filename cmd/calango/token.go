package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"calango/internal/auth"
	"calango/internal/config"
	"calango/internal/core"
	"calango/internal/store"
	"calango/internal/vault"
)

// vaultKeySources maps the configured master key sources to the vault loader.
func vaultKeySources(cfg config.Config) vault.KeySources {
	return vault.KeySources{
		Key: cfg.VaultKey, KeyFile: cfg.VaultKeyFile, Version: cfg.VaultKeyVersion,
		Previous: cfg.VaultKeyPrevious, PreviousFile: cfg.VaultKeyPreviousFile,
	}
}

// loadLocalToken returns the local API token from the vault, creating it on first use.
// A legacy token file is moved into the vault first; imported reports whether that happened.
func loadLocalToken(ctx context.Context, cfg config.Config, secrets *vault.Service) (token string, imported bool, err error) {
	imported, err = auth.ImportFileToken(ctx, cfg.DataDir, secrets)
	if err != nil {
		return "", false, fmt.Errorf("api token import: %w", err)
	}
	token, err = auth.LoadOrCreateVaultToken(ctx, secrets)
	if err != nil {
		return "", false, err
	}
	return token, imported, nil
}

// printToken writes the local API token to stdout and nothing else. The desktop shell runs
// `calango print-token` to get it, so the token never reaches a log or an HTTP route.
// It does not rotate master keys: the running daemon owns that at startup.
func printToken(ctx context.Context, cfg config.Config) error {
	database, err := store.Open(ctx, filepath.Join(cfg.DataDir, "calango.db"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("close store: %v", err)
		}
	}()
	keys, err := vault.LoadMasterKeys(vaultKeySources(cfg), vault.OSKeyring{})
	if err != nil {
		return fmt.Errorf("vault master key: %w", err)
	}
	secrets, err := vault.NewService(database, keys, localOwnerID, database, time.Now, core.NewID)
	if err != nil {
		return fmt.Errorf("vault service: %w", err)
	}
	token, _, err := loadLocalToken(ctx, cfg, secrets)
	if err != nil {
		return err
	}
	_, err = io.WriteString(os.Stdout, token)
	return err
}
