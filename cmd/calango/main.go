// Command calango runs the Calango daemon: HTTP API plus the routine scheduler.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"calango/internal/api"
	"calango/internal/core"
)

func main() {
	addr := os.Getenv("CALANGO_ADDR")
	if addr == "" {
		addr = "127.0.0.1:7317"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	scheduler := core.NewScheduler(time.Now)
	go scheduler.Run(ctx, time.Second, func(id string, err error) {
		log.Printf("routine %s failed: %v", id, err)
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("calango listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
