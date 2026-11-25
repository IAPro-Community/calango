package vault

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

// isolatedKeyring never touches the actual Calango service name.
type isolatedKeyring struct{ service string }

func (k isolatedKeyring) Get(_, account string) (string, error) {
	return (OSKeyring{}).Get(k.service, account)
}

func (k isolatedKeyring) Set(_, account, secret string) error {
	return (OSKeyring{}).Set(k.service, account, secret)
}

func TestNativeKeyringRoundTrip(t *testing.T) {
	if os.Getenv("CALANGO_TEST_OS_KEYRING") != "1" {
		t.Skip("requires an explicitly authorized interactive OS keyring")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	service := "calango-test-" + hex.EncodeToString(nonce[:])
	account := MasterKeyAccount(1)
	kr := isolatedKeyring{service: service}
	t.Cleanup(func() {
		if err := keyring.Delete(service, account); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			t.Error("cleanup of isolated test credential failed")
		}
	})
	first, err := LoadOrCreateKeyringKey(kr, 1)
	if err != nil {
		t.Fatal("native keyring could not create the isolated test key")
	}
	defer clear(first)
	second, err := LoadOrCreateKeyringKey(kr, 1)
	if err != nil {
		t.Fatal("native keyring could not reload the isolated test key")
	}
	defer clear(second)
	if !bytes.Equal(first, second) {
		t.Fatal("native keyring did not return the same key")
	}
	if err := keyring.Delete(service, account); err != nil {
		t.Fatal("native keyring could not remove the isolated test key")
	}
	if _, err := kr.Get(keyringService, account); !errors.Is(err, ErrKeyNotFound) {
		t.Fatal("deleted test credential is still accessible or keyring failed")
	}
}
