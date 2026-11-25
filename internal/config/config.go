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
	// VaultKeyVersion is the version of VaultKey or VaultKeyFile; zero means 1.
	VaultKeyVersion int
	// VaultKeyPrevious and VaultKeyPreviousFile hold the key of version VaultKeyVersion-1
	// while secrets are re-wrapped during a rotation. Remove them afterwards.
	VaultKeyPrevious     string
	VaultKeyPreviousFile string
	// TrustedProxies are the peers whose X-Forwarded-For header counts toward the pairing limit.
	TrustedProxies []netip.Prefix
	// PriceTableFile is an optional versioned USD tariff file, not a credential.
	PriceTableFile string
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
	vaultKeyVersion := 1
	if raw := os.Getenv("CALANGO_VAULT_KEY_VERSION"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			return Config{}, fmt.Errorf("CALANGO_VAULT_KEY_VERSION: must be an integer of at least 1")
		}
		vaultKeyVersion = v
	}
	trustedProxies, err := parseTrustedProxies(os.Getenv("CALANGO_TRUSTED_PROXIES"))
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		Addr: addr, DataDir: dataDir,
		VaultKey: vaultKey, VaultKeyFile: vaultKeyFile, VaultKeyVersion: vaultKeyVersion,
		VaultKeyPrevious: os.Getenv("CALANGO_VAULT_KEY_PREVIOUS"), VaultKeyPreviousFile: os.Getenv("CALANGO_VAULT_KEY_PREVIOUS_FILE"),
		TrustedProxies: trustedProxies,
		PriceTableFile: os.Getenv("CALANGO_PRICE_TABLE_FILE"),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// parseTrustedProxies reads CALANGO_TRUSTED_PROXIES: comma-separated IP addresses or CIDR ranges.
func parseTrustedProxies(raw string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			p, err := netip.ParsePrefix(item)
			if err != nil {
				return nil, fmt.Errorf("CALANGO_TRUSTED_PROXIES: %q is not a CIDR range", item)
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(item)
		if err != nil {
			return nil, fmt.Errorf("CALANGO_TRUSTED_PROXIES: %q is not an IP address or CIDR range", item)
		}
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
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
	if c.PriceTableFile != "" && (!filepath.IsAbs(c.PriceTableFile) || strings.TrimSpace(c.PriceTableFile) != c.PriceTableFile || strings.ContainsRune(c.PriceTableFile, 0)) {
		return fmt.Errorf("CALANGO_PRICE_TABLE_FILE: must be an absolute path without surrounding whitespace or NUL")
	}
	if err := c.validateVaultKeys(); err != nil {
		return err
	}
	return nil
}

// validateVaultKeys checks the key versions and the rotation pair. Errors name variables, never values.
func (c Config) validateVaultKeys() error {
	version := c.VaultKeyVersion
	if version == 0 {
		version = 1
	}
	if version < 1 {
		return fmt.Errorf("CALANGO_VAULT_KEY_VERSION: must be an integer of at least 1")
	}
	hasCurrent := c.VaultKey != "" || c.VaultKeyFile != ""
	if version > 1 && !hasCurrent {
		return fmt.Errorf("CALANGO_VAULT_KEY_VERSION: needs CALANGO_VAULT_KEY or CALANGO_VAULT_KEY_FILE; the OS keyring holds version 1 only")
	}
	if c.VaultKeyPrevious != "" && c.VaultKeyPreviousFile != "" {
		return fmt.Errorf("CALANGO_VAULT_KEY_PREVIOUS and CALANGO_VAULT_KEY_PREVIOUS_FILE are mutually exclusive; set one")
	}
	if c.VaultKeyPrevious == "" && c.VaultKeyPreviousFile == "" {
		return nil
	}
	if !hasCurrent {
		return fmt.Errorf("CALANGO_VAULT_KEY_PREVIOUS: needs CALANGO_VAULT_KEY or CALANGO_VAULT_KEY_FILE for the current key")
	}
	if version < 2 {
		return fmt.Errorf("CALANGO_VAULT_KEY_PREVIOUS: needs CALANGO_VAULT_KEY_VERSION of at least 2")
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
