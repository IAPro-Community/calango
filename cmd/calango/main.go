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
	"calango/internal/anthropic"
	"calango/internal/api"
	"calango/internal/approvals"
	"calango/internal/auth"
	"calango/internal/config"
	"calango/internal/conversations"
	"calango/internal/core"
	"calango/internal/devices"
	"calango/internal/events"
	"calango/internal/gateway"
	"calango/internal/maestro"
	"calango/internal/mcpgw"
	"calango/internal/memory"
	"calango/internal/providers"
	"calango/internal/routines"
	"calango/internal/store"
	"calango/internal/tools"
	"calango/internal/usage"
	"calango/internal/vault"
)

const (
	localOwnerID         = "owner"
	localDeviceID        = "local-desktop"
	approvalTTL          = 24 * time.Hour
	routineTick          = 30 * time.Second
	maestroCheckInterval = 24 * time.Hour
)

// desktopOrigin is the origin of the Electron desktop shell (apps/desktop, scheme app).
const desktopOrigin = "app://calango"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	prices, err := gateway.LoadPriceTable(cfg.PriceTableFile)
	if err != nil {
		log.Fatalf("invalid price table: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(os.Args) > 1 && os.Args[1] == "print-token" {
		if err := printToken(ctx, cfg); err != nil {
			log.Fatalf("print-token: %v", err)
		}
		return
	}
	database, err := store.Open(ctx, filepath.Join(cfg.DataDir, "calango.db"))
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("close store: %v", err)
		}
	}()

	keys, err := vault.LoadMasterKeys(vaultKeySources(cfg), vault.OSKeyring{})
	if err != nil {
		log.Fatalf("vault master key: %v", err)
	}
	secrets, err := vault.NewService(database, keys, localOwnerID, database, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("vault service: %v", err)
	}
	rewrapped, err := secrets.Rotate(ctx)
	if err != nil {
		log.Fatalf("vault key rotation stopped after %d secrets: %v", rewrapped, err)
	}
	if rewrapped > 0 {
		log.Printf("vault: re-wrapped %d secrets under master key version %d", rewrapped, keys.Current())
	}
	token, imported, err := loadLocalToken(ctx, cfg, secrets)
	if err != nil {
		log.Fatalf("local api token: %v", err)
	}
	if imported {
		log.Printf("moved the api token from %s into the vault", filepath.Join(cfg.DataDir, auth.TokenFileName))
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
	go memorySvc.RunPurge(ctx, func(removed int64, err error) {
		if err != nil {
			log.Printf("memory purge: %v", err)
			return
		}
		if removed > 0 {
			log.Printf("memory purge: removed %d expired memories", removed)
		}
	})
	approvalSvc, err := approvals.NewService(database, approvalTTL, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("approvals service: %v", err)
	}
	toolsSvc, err := tools.NewService(database, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("tools service: %v", err)
	}
	defer toolsSvc.Close()
	mcpURL, err := mcpgw.LoopbackURL(cfg.Addr)
	if err != nil {
		log.Fatalf("mcp endpoint: %v", err)
	}
	mcpGW, err := mcpgw.New(mcpgw.Config{Tools: toolsSvc, Approvals: approvalSvc, Runs: database, Memory: memorySvc, URL: mcpURL})
	if err != nil {
		log.Fatalf("mcp gateway: %v", err)
	}
	approvalSvc.SetExecutor(core.ActionConnectorWrite, mcpGW.Execute)
	devicesSvc, err := devices.NewService(database, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("devices service: %v", err)
	}
	usageSvc, err := usage.NewService(database)
	if err != nil {
		log.Fatalf("usage service: %v", err)
	}
	workDir := filepath.Join(cfg.DataDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		log.Fatalf("conversations workspace: %v", err)
	}
	providerSvc, err := providers.NewService(providers.Config{
		Repo:             database,
		Secrets:          secrets,
		CodexBin:         "codex",
		CodexWorkDir:     workDir,
		AccountsDir:      filepath.Join(cfg.DataDir, "accounts"),
		AnthropicBaseURL: anthropic.DefaultBaseURL,
		Now:              time.Now,
		NewID:            core.NewID,
	})
	if err != nil {
		log.Fatalf("provider accounts: %v", err)
	}
	resolve := func(ctx context.Context, name string) (gateway.Provider, error) {
		return providerSvc.Resolve(ctx, localOwnerID, name)
	}
	gw, err := gateway.New(gateway.Config{
		Ledger:  database,
		Resolve: resolve,
		Prices:  prices,
		Now:     time.Now,
		NewID:   core.NewID,
	})
	if err != nil {
		log.Fatalf("gateway: %v", err)
	}
	hub := events.NewHub()
	maestroManager, err := maestro.New(maestro.Config{
		DataDir: cfg.DataDir,
		ActiveRuns: func(ctx context.Context) (bool, error) {
			agentViews, err := agentSvc.List(ctx, localOwnerID)
			if err != nil {
				return false, err
			}
			for _, agent := range agentViews {
				count, err := database.CountActiveAgentRuns(ctx, agent.ID)
				if err != nil {
					return false, err
				}
				if count > 0 {
					return true, nil
				}
			}
			return false, nil
		},
		Audit: func(ctx context.Context, from, to string, automatic bool, result string) error {
			id, err := core.NewID(time.Now())
			if err != nil {
				return err
			}
			return database.AppendAudit(ctx, core.AuditEvent{
				ID: id, At: time.Now(), ActorKind: "system",
				Action: "maestro.version_switch", Subject: "maestro",
				Facts: map[string]any{"from": from, "to": to, "auto": automatic, "result": result},
			})
		},
	})
	if err != nil {
		log.Fatalf("maestro manager: %v", err)
	}
	convSvc, err := conversations.NewService(conversations.Config{
		Repo:        database,
		Gateway:     gw,
		Memory:      memorySvc,
		Now:         time.Now,
		NewID:       core.NewID,
		Events:      hub,
		MaestroHome: maestroManager.CodexHome,
		Tools:       mcpGW,
	})
	if err != nil {
		log.Fatalf("conversations service: %v", err)
	}
	if n, err := approvalSvc.Reconcile(ctx); err != nil {
		log.Fatalf("approvals reconcile: %v", err)
	} else if n > 0 {
		log.Printf("approvals: closed %d approved actions without receipt", n)
	}
	if err := convSvc.Start(ctx); err != nil {
		log.Fatalf("conversations start: %v", err)
	}
	routinesSvc, err := routines.NewService(database, core.RunOnPC, time.Local, routines.NewChatExecutor(convSvc), func(ev routines.Event) {
		hub.Publish(events.Event{Type: ev.Kind, OwnerID: ev.OwnerID, Data: ev})
	}, time.Now, core.NewID)
	if err != nil {
		log.Fatalf("routines service: %v", err)
	}
	if n, err := routinesSvc.Start(ctx); err != nil {
		log.Fatalf("routines start: %v", err)
	} else if n > 0 {
		log.Printf("routines: requeued %d interrupted occurrences", n)
	}
	go routinesSvc.Run(ctx, routineTick, func(err error) { log.Printf("routines: %v", err) })
	go runMaestroScheduler(ctx, maestroManager, hub)

	handler, err := api.NewHandler(api.Config{
		Agents:         agentSvc,
		Accounts:       providerSvc,
		Memories:       memorySvc,
		Approvals:      approvalSvc,
		Devices:        devicesSvc,
		Usage:          usageSvc,
		Vault:          secrets,
		Audit:          database,
		Conversations:  convSvc,
		Events:         hub,
		Routines:       routinesSvc,
		Maestro:        maestroManager,
		Tools:          toolsSvc,
		Token:          localToken,
		OwnerID:        localOwnerID,
		DeviceID:       localDeviceID,
		TrustedProxies: cfg.TrustedProxies,
		CORSOrigins:    []string{desktopOrigin},
	})
	if err != nil {
		log.Fatalf("api handler: %v", err)
	}
	root := http.NewServeMux()
	root.Handle(mcpgw.Path, mcpGW.Handler())
	root.Handle("/", handler)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           root,
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

func runMaestroScheduler(ctx context.Context, manager *maestro.Manager, hub *events.Hub) {
	check := func() {
		status, err := manager.Status(ctx)
		if err != nil {
			log.Printf("maestro status unavailable")
			return
		}
		if status.AutoUpdate {
			installed, err := manager.RunAutoUpdate(ctx)
			if err != nil {
				log.Printf("maestro automatic update failed")
				hub.Publish(events.Event{Type: events.TypeMaestroUpdateFailed, OwnerID: localOwnerID, Data: map[string]string{"result": "failed"}})
				return
			}
			if installed != nil {
				status, err := manager.Status(ctx)
				if err != nil {
					log.Printf("maestro status after update unavailable")
					return
				}
				hub.Publish(events.Event{Type: events.TypeMaestroUpdated, OwnerID: localOwnerID, Data: status})
			}
			return
		}
		release, err := manager.CheckUpdate(ctx)
		if err != nil {
			log.Printf("maestro registry check failed")
			return
		}
		if release != nil {
			hub.Publish(events.Event{Type: events.TypeMaestroUpdateAvailable, OwnerID: localOwnerID, Data: *release})
		}
	}
	check()
	ticker := time.NewTicker(maestroCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}
