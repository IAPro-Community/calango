package vault

import (
	"fmt"
	"os"
	"strings"
)

// masterKeyVersion is the only version the node creates today. Rotation adds versions
// alongside it; see Service.Rotate.
const masterKeyVersion = 1

// LoadMasterKeys returns the master keys for this process. The callers pass the values
// that internal/config read from CALANGO_VAULT_KEY or CALANGO_VAULT_KEY_FILE; both empty
// means the OS keyring is the source. It fails rather than falling back to a weaker
// source: with no key, the node cannot open its secrets and must not start.
func LoadMasterKeys(inlineKey, keyFile string, kr Keyring) (MasterKeys, error) {
	if keyFile != "" {
		raw, err := os.ReadFile(keyFile)
		if err != nil {
			return MasterKeys{}, fmt.Errorf("vault: read master key file: %w", err)
		}
		return keysFromEncoded(string(raw))
	}
	if inlineKey != "" {
		return keysFromEncoded(inlineKey)
	}
	key, err := LoadOrCreateKeyringKey(kr, masterKeyVersion)
	if err != nil {
		return MasterKeys{}, err
	}
	return NewMasterKeys(map[int][]byte{masterKeyVersion: key})
}

func keysFromEncoded(encoded string) (MasterKeys, error) {
	key, err := ParseMasterKey(strings.TrimSpace(encoded))
	if err != nil {
		return MasterKeys{}, err
	}
	return NewMasterKeys(map[int][]byte{masterKeyVersion: key})
}
