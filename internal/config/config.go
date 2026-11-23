// Package config loads and validates process configuration for the Calango node.
package config

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

const defaultAddr = "127.0.0.1:7317"

// Config contains the runtime settings shared by the Calango entry point.
type Config struct {
	Addr string
}

// Load reads supported CALANGO_* environment variables and validates them.
func Load() (Config, error) {
	addr := os.Getenv("CALANGO_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	cfg := Config{Addr: addr}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("CALANGO_ADDR: %w", err)
	}
	return cfg, nil
}

// Validate checks the address before the server or background workers start.
func (c Config) Validate() error {
	if c.Addr == "" || strings.TrimSpace(c.Addr) != c.Addr {
		return fmt.Errorf("must be a non-empty host:port without surrounding whitespace")
	}

	host, portText, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return fmt.Errorf("must be a valid host:port: %w", err)
	}
	if host == "" {
		return fmt.Errorf("host is required; use 0.0.0.0 to listen on all interfaces")
	}
	if !validHost(host) {
		return fmt.Errorf("host must be an IP address or valid DNS name")
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("port must be an integer from 1 to 65535")
	}
	return nil
}

func validHost(host string) bool {
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}

	host = strings.TrimSuffix(host, ".")
	if host == "" || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}
