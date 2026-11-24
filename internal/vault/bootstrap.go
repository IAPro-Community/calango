package vault

import (
	"fmt"
	"os"
	"strings"
)

// masterKeyVersion is the version of the OS keyring key and the default explicit key.
const masterKeyVersion = 1

// KeySources names where the master keys come from. The current key is Key, or the file
// at KeyFile; Version is its version (zero means 1). Previous or PreviousFile hold the key
// of version Version-1 while secrets are re-wrapped during a rotation. All empty means the
// OS keyring, which holds version 1 only. Callers pass the values internal/config read.
type KeySources struct {
	Key          string
	KeyFile      string
	Version      int
	Previous     string
	PreviousFile string
}

// LoadMasterKeys returns the master keys for this process. It fails rather than falling
// back to a weaker source: with no key, the node cannot open its secrets and must not start.
func LoadMasterKeys(src KeySources, kr Keyring) (MasterKeys, error) {
	if src.Key == "" && src.KeyFile == "" {
		if src.Version > 1 || src.Previous != "" || src.PreviousFile != "" {
			return MasterKeys{}, fmt.Errorf("vault: versioned and previous master keys need an explicit key, not the OS keyring")
		}
		key, err := LoadOrCreateKeyringKey(kr, masterKeyVersion)
		if err != nil {
			return MasterKeys{}, err
		}
		return NewMasterKeys(map[int][]byte{masterKeyVersion: key})
	}

	version := src.Version
	if version == 0 {
		version = masterKeyVersion
	}
	current, err := readMasterKey(src.Key, src.KeyFile)
	if err != nil {
		return MasterKeys{}, err
	}
	keys := map[int][]byte{version: current}
	if src.Previous != "" || src.PreviousFile != "" {
		if version < 2 {
			return MasterKeys{}, fmt.Errorf("vault: previous master key needs a current version of at least 2")
		}
		previous, err := readMasterKey(src.Previous, src.PreviousFile)
		if err != nil {
			return MasterKeys{}, err
		}
		keys[version-1] = previous
	}
	return NewMasterKeys(keys)
}

// readMasterKey decodes the key in file when a file is set, otherwise the inline value.
func readMasterKey(inline, file string) ([]byte, error) {
	if file != "" {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("vault: read master key file: %w", err)
		}
		return ParseMasterKey(strings.TrimSpace(string(raw)))
	}
	return ParseMasterKey(strings.TrimSpace(inline))
}
