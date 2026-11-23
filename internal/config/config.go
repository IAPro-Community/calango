// Package config loads and validates process configuration for the Calango node.
package config

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultAddr = "127.0.0.1:7317"

type Config struct {
	Addr    string
	DataDir string
	// VaultKey and VaultKeyFile hold the vault master key when no OS keyring is used.
	// They are secrets: never log or print a Config.
	VaultKey     string
	VaultKeyFile string
}

// Load reads supported CALANGO_* environment variables and validates them.
func Load() (Config, error) {
	addr := os.Getenv("CALANGO_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	dataDir := os.Getenv("CALANGO_DATA_DIR")
	if dataDir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return Config{}, fmt.Errorf("CALANGO_DATA_DIR: find user config directory: %w", err)
		}
		dataDir = filepath.Join(configDir, "calango")
	}
	if strings.TrimSpace(dataDir) != dataDir || strings.ContainsRune(dataDir, 0) {
		return Config{}, fmt.Errorf("CALANGO_DATA_DIR: must not contain surrounding whitespace or NUL")
	}
	dataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return Config{}, fmt.Errorf("CALANGO_DATA_DIR: resolve path: %w", err)
	}

	vaultKey := os.Getenv("CALANGO_VAULT_KEY")
	vaultKeyFile := os.Getenv("CALANGO_VAULT_KEY_FILE")
	cfg := Config{Addr: addr, DataDir: dataDir, VaultKey: vaultKey, VaultKeyFile: vaultKeyFile}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks process configuration before the server or background workers start.
func (c Config) Validate() error {
	if err := validateAddr(c.Addr); err != nil {
		return fmt.Errorf("CALANGO_ADDR: %w", err)
	}
	if c.DataDir == "" || strings.TrimSpace(c.DataDir) != c.DataDir || strings.ContainsRune(c.DataDir, 0) {
		return fmt.Errorf("CALANGO_DATA_DIR: must be a non-empty path without surrounding whitespace or NUL")
	}
	if !filepath.IsAbs(c.DataDir) {
		return fmt.Errorf("CALANGO_DATA_DIR: path must be absolute")
	}
	if c.VaultKey != "" && c.VaultKeyFile != "" {
		return fmt.Errorf("CALANGO_VAULT_KEY and CALANGO_VAULT_KEY_FILE are mutually exclusive; set one")
	}
	return nil
}

func validateAddr(addr string) error {
	if addr == "" || strings.TrimSpace(addr) != addr {
		return fmt.Errorf("must be a non-empty host:port without surrounding whitespace")
	}

	host, portText, err := net.SplitHostPort(addr)
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
