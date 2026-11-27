package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// healthcheckTimeout bounds `calango healthcheck`, so a hung daemon fails the probe.
const healthcheckTimeout = 3 * time.Second

// healthcheckURL returns the loopback /healthz URL for the address the daemon listens on.
// A wildcard host (0.0.0.0, ::) or an empty one is probed on loopback; the port is kept.
func healthcheckURL(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("CALANGO_ADDR %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	} else if host == "::" {
		host = "::1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/healthz", nil
}

// healthcheck queries the running daemon's /healthz and fails unless it answers 200. The
// container image is distroless (no shell, curl or wget), so Docker and Compose health
// checks run `calango healthcheck`. It sends no token and reads no secret.
func healthcheck(ctx context.Context, addr string, client *http.Client) error {
	url, err := healthcheckURL(addr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, healthcheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %d", url, resp.StatusCode)
	}
	return nil
}
