// Command calango runs the Calango daemon: the HTTP API.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"calango/internal/agents"
	"calango/internal/api"
	"calango/internal/approvals"
	"calango/internal/auth"
	"calango/internal/config"
	"calango/internal/core"
	"calango/internal/devices"
	"calango/internal/memory"
	"calango/internal/store"
	"calango/internal/usage"
	"calango/internal/vault"
)

const (
	// localOwnerID owns every record of the V1 single-person node until accounts exist.
	localOwnerID = "owner"
	// localDeviceID is recorded as the decider of approvals made with the local token.
	localDeviceID = "local-desktop"
	// approvalTTL is how long a pending approval waits before it expires.
	approvalTTL = 24 * time.Hour
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	database, err := store.Open(ctx, filepath.Join(cfg.DataDir, "calango.db"))
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("close store: %v", err)
		}
	}()

	keys, err := vault.LoadMasterKeys(cfg.VaultKey, cfg.VaultKeyFile, vault.OSKeyring{})
	if err != nil {
		log.Fatalf("vault master key: %v", err)
	}
	secrets, err := vault.NewService(database, keys, localOwnerID, database, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("vault service: %v", err)
	}
	imported, err := auth.ImportFileToken(ctx, cfg.DataDir, secrets)
	if err != nil {
		log.Fatalf("api token import: %v", err)
	}
	if imported {
		log.Printf("moved the api token from %s into the vault", filepath.Join(cfg.DataDir, auth.TokenFileName))
	}
	token, err := auth.LoadOrCreateVaultToken(ctx, secrets)
	if err != nil {
		log.Fatalf("local api token: %v", err)
	}
	localToken, err := auth.NewLocalToken(token, secrets)
	if err != nil {
		log.Fatalf("local api token: %v", err)
	}
	agentSvc, err := agents.NewService(database, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("agents service: %v", err)
	}
	memorySvc, err := memory.NewService(database, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("memory service: %v", err)
	}
	approvalSvc, err := approvals.NewService(database, approvalTTL, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("approvals service: %v", err)
	}
	devicesSvc, err := devices.NewService(database, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("devices service: %v", err)
	}
	usageSvc, err := usage.NewService(database)
	if err != nil {
		log.Fatalf("usage service: %v", err)
	}
	handler, err := api.NewHandler(api.Config{
		Agents:    agentSvc,
		Memories:  memorySvc,
		Approvals: approvalSvc,
		Devices:   devicesSvc,
		Usage:     usageSvc,
		Vault:     secrets,
		Audit:     database,
		Token:     localToken,
		OwnerID:   localOwnerID,
		DeviceID:  localDeviceID,
	})
	if err != nil {
		log.Fatalf("api handler: %v", err)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("calango listening on %s (api token in the vault)", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
