package vault

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func encodedKey(b byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{b}, keySize))
}

func TestLoadMasterKeysKeyFileTakesPrecedenceOverInlineKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.key")
	if err := os.WriteFile(path, []byte(encodedKey(0x11)+"\n"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	keys, err := LoadMasterKeys(KeySources{Key: encodedKey(0x22), KeyFile: path}, &fakeKeyring{entries: map[string]string{}})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if keys.Current() != 1 {
		t.Fatalf("current = %d, want 1", keys.Current())
	}

	// A value sealed with the file key must open with a keyring built from that key only.
	sealed, err := Seal([]byte("token"), "owner:o", keys, 1)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	fromFile, err := NewMasterKeys(map[int][]byte{1: bytes.Repeat([]byte{0x11}, keySize)})
	if err != nil {
		t.Fatalf("file keys: %v", err)
	}
	if _, err := Open(sealed, "owner:o", fromFile, 1); err != nil {
		t.Fatalf("open with file key: %v", err)
	}
}

func TestLoadMasterKeysMissingKeyFileDoesNotFallBack(t *testing.T) {
	kr := &fakeKeyring{entries: map[string]string{}}
	missing := filepath.Join(t.TempDir(), "absent.key")

	_, err := LoadMasterKeys(KeySources{Key: encodedKey(0x22), KeyFile: missing}, kr)
	if err == nil || !strings.Contains(err.Error(), "master key file") {
		t.Fatalf("err = %v, want an error about the master key file", err)
	}
	if kr.sets != 0 {
		t.Fatal("a failing key file must not create a keyring key")
	}
}

func TestLoadMasterKeysInlineKeyWhenNoFile(t *testing.T) {
	kr := &fakeKeyring{entries: map[string]string{}}

	keys, err := LoadMasterKeys(KeySources{Key: encodedKey(0x33)}, kr)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !keys.Has(1) {
		t.Fatal("inline key must be registered as version 1")
	}
	if kr.sets != 0 {
		t.Fatal("inline key must not touch the keyring")
	}
}

func TestLoadMasterKeysInvalidInlineKeyNeverEchoesInput(t *testing.T) {
	secret := "not-base64-and-secret!"

	_, err := LoadMasterKeys(KeySources{Key: secret}, &fakeKeyring{entries: map[string]string{}})
	if err == nil {
		t.Fatal("invalid key must be rejected")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaks the key input: %v", err)
	}
}

func TestLoadMasterKeysUsesKeyringWhenNothingConfigured(t *testing.T) {
	kr := &fakeKeyring{entries: map[string]string{}}

	keys, err := LoadMasterKeys(KeySources{}, kr)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !keys.Has(1) || kr.sets != 1 {
		t.Fatalf("keyring not used: has=%v sets=%d", keys.Has(1), kr.sets)
	}
}

func TestLoadMasterKeysKeyringUnavailableFailsClosed(t *testing.T) {
	kr := &fakeKeyring{entries: map[string]string{}, err: errors.New("dbus: no secret service")}

	_, err := LoadMasterKeys(KeySources{}, kr)
	if !errors.Is(err, ErrKeyringUnavailable) {
		t.Fatalf("err = %v, want ErrKeyringUnavailable", err)
	}
}

func TestLoadMasterKeysPreviousKeyOpensOldVersionDuringRotation(t *testing.T) {
	kr := &fakeKeyring{entries: map[string]string{}}
	keys, err := LoadMasterKeys(KeySources{Key: encodedKey(0x22), Version: 2, Previous: encodedKey(0x11)}, kr)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if keys.Current() != 2 || !keys.Has(1) {
		t.Fatalf("current=%d has1=%v, want current 2 with version 1 still loaded", keys.Current(), keys.Has(1))
	}
	if kr.sets != 0 {
		t.Fatal("explicit keys must not touch the keyring")
	}
}

func TestLoadMasterKeysPreviousNeedsVersionTwo(t *testing.T) {
	_, err := LoadMasterKeys(KeySources{Key: encodedKey(0x22), Previous: encodedKey(0x11)}, &fakeKeyring{entries: map[string]string{}})
	if err == nil || !strings.Contains(err.Error(), "at least 2") {
		t.Fatalf("err = %v, want refusal for version 1", err)
	}
}

func TestLoadMasterKeysVersionNeedsExplicitKey(t *testing.T) {
	_, err := LoadMasterKeys(KeySources{Version: 2}, &fakeKeyring{entries: map[string]string{}})
	if err == nil || !strings.Contains(err.Error(), "OS keyring") {
		t.Fatalf("err = %v, want refusal to version a keyring key", err)
	}
}
