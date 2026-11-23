package vault

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// fakeKeyring is an in-memory Keyring. err, when set, fails every call.
type fakeKeyring struct {
	entries map[string]string
	err     error
	sets    int
}

func (f *fakeKeyring) Get(service, account string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	secret, ok := f.entries[service+"/"+account]
	if !ok {
		return "", ErrKeyNotFound
	}
	return secret, nil
}

func (f *fakeKeyring) Set(service, account, secret string) error {
	if f.err != nil {
		return f.err
	}
	f.sets++
	f.entries[service+"/"+account] = secret
	return nil
}

func TestLoadOrCreateKeyringKeyCreatesOnceAndReuses(t *testing.T) {
	kr := &fakeKeyring{entries: map[string]string{}}

	first, err := LoadOrCreateKeyringKey(kr, 1)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if len(first) != keySize {
		t.Fatalf("created key has %d bytes, want %d", len(first), keySize)
	}
	if _, ok := kr.entries["calango/vault-master-v1"]; !ok {
		t.Fatal("master key was not stored under calango/vault-master-v1")
	}

	second, err := LoadOrCreateKeyringKey(kr, 1)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("second load returned a different key")
	}
	if kr.sets != 1 {
		t.Fatalf("keyring Set called %d times, want 1", kr.sets)
	}
}

func TestLoadOrCreateKeyringKeyRefusesWhenKeyringFails(t *testing.T) {
	kr := &fakeKeyring{entries: map[string]string{}, err: errors.New("secret service locked")}

	key, err := LoadOrCreateKeyringKey(kr, 1)
	if !errors.Is(err, ErrKeyringUnavailable) {
		t.Fatalf("err = %v, want ErrKeyringUnavailable", err)
	}
	if key != nil {
		t.Fatal("a key was returned although the keyring failed")
	}
	if kr.sets != 0 {
		t.Fatal("a key was written although the keyring failed")
	}
}

func TestLoadOrCreateKeyringKeyRejectsCorruptEntry(t *testing.T) {
	kr := &fakeKeyring{entries: map[string]string{"calango/vault-master-v1": base64.StdEncoding.EncodeToString([]byte("short"))}}
	if _, err := LoadOrCreateKeyringKey(kr, 1); err == nil {
		t.Fatal("a corrupt keyring entry was accepted")
	}
}

func TestParseMasterKeyRejectsBadInputWithoutEchoingIt(t *testing.T) {
	bad := []string{"%%%not-base64", base64.StdEncoding.EncodeToString(make([]byte, 16))}
	for _, input := range bad {
		_, err := ParseMasterKey(input)
		if err == nil {
			t.Fatalf("ParseMasterKey(%q) accepted bad input", input)
		}
		if strings.Contains(err.Error(), input) {
			t.Fatalf("error echoes the key material: %v", err)
		}
	}
	good := base64.StdEncoding.EncodeToString(make([]byte, keySize))
	if key, err := ParseMasterKey(" " + good + "\n"); err != nil || len(key) != keySize {
		t.Fatalf("ParseMasterKey(valid) = %d bytes, %v", len(key), err)
	}
}
