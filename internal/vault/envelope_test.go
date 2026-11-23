package vault

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func testKeys(t *testing.T, versions map[int]byte) MasterKeys {
	t.Helper()
	raw := make(map[int][]byte, len(versions))
	for version, fill := range versions {
		key := bytes.Repeat([]byte{fill}, keySize)
		raw[version] = key
	}
	keys, err := NewMasterKeys(raw)
	if err != nil {
		t.Fatalf("NewMasterKeys: %v", err)
	}
	return keys
}

const testScope = "owner:owner-1"

func TestSealOpenRoundTrip(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11})
	ciphertext, err := Seal([]byte("sk-live-123456"), testScope, keys, 1)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if strings.Contains(ciphertext, "sk-live-123456") {
		t.Fatal("ciphertext contains the plain value")
	}
	got, err := Open(ciphertext, testScope, keys, 1)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if string(got) != "sk-live-123456" {
		t.Fatalf("Open = %q, want the sealed value", got)
	}
}

func TestOpenRejectsOtherOwner(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11})
	ciphertext, err := Seal([]byte("token"), testScope, keys, 1)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := Open(ciphertext, "owner:owner-2", keys, 1); !errors.Is(err, ErrSealed) {
		t.Fatalf("Open for another owner: err = %v, want ErrSealed", err)
	}
}

func TestOpenRejectsWrongMasterKey(t *testing.T) {
	sealingKeys := testKeys(t, map[int]byte{1: 0x11})
	otherKeys := testKeys(t, map[int]byte{1: 0x22})
	ciphertext, err := Seal([]byte("token"), testScope, sealingKeys, 1)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := Open(ciphertext, testScope, otherKeys, 1); !errors.Is(err, ErrSealed) {
		t.Fatalf("Open with another master key: err = %v, want ErrSealed", err)
	}
}

func TestOpenRejectsVersionThatDoesNotMatch(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11, 2: 0x22})
	ciphertext, err := Seal([]byte("token"), testScope, keys, 1)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := Open(ciphertext, testScope, keys, 2); !errors.Is(err, ErrSealed) {
		t.Fatalf("Open with the wrong version: err = %v, want ErrSealed", err)
	}
}

func TestOpenReportsMissingMasterKeyVersion(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11})
	ciphertext, err := Seal([]byte("token"), testScope, keys, 1)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := Open(ciphertext, testScope, keys, 7); !errors.Is(err, ErrNoMasterKey) {
		t.Fatalf("Open with an unconfigured version: err = %v, want ErrNoMasterKey", err)
	}
}

func TestOpenRejectsTamperedPayload(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11})
	ciphertext, err := Seal([]byte("token"), testScope, keys, 1)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(ciphertext), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	payload, err := base64.StdEncoding.DecodeString(env["p"].(string))
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	payload[len(payload)-1] ^= 0x01
	env["p"] = base64.StdEncoding.EncodeToString(payload)
	tampered, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("encode envelope: %v", err)
	}
	if _, err := Open(string(tampered), testScope, keys, 1); !errors.Is(err, ErrSealed) {
		t.Fatalf("Open tampered payload: err = %v, want ErrSealed", err)
	}
}

func TestOpenRejectsMalformedEnvelope(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11})
	for _, input := range []string{"", "not json", `{"v":2,"k":"","p":""}`, `{"v":1,"k":"!!","p":"AA=="}`} {
		if _, err := Open(input, testScope, keys, 1); !errors.Is(err, ErrSealed) {
			t.Fatalf("Open(%q): err = %v, want ErrSealed", input, err)
		}
	}
}

func TestRewrapMovesToNewVersionWithoutTouchingPayload(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11, 2: 0x22})
	original, err := Seal([]byte("token"), testScope, keys, 1)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	rewrapped, err := Rewrap(original, testScope, keys, 1, 2)
	if err != nil {
		t.Fatalf("Rewrap: %v", err)
	}
	if payloadOf(t, original) != payloadOf(t, rewrapped) {
		t.Fatal("Rewrap changed the payload")
	}
	got, err := Open(rewrapped, testScope, keys, 2)
	if err != nil || string(got) != "token" {
		t.Fatalf("Open after Rewrap = %q, %v; want token", got, err)
	}
	if _, err := Open(rewrapped, testScope, keys, 1); !errors.Is(err, ErrSealed) {
		t.Fatalf("old version still opens after Rewrap: err = %v", err)
	}
}

func TestRewrapRejectsWrongOwner(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11, 2: 0x22})
	original, err := Seal([]byte("token"), testScope, keys, 1)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := Rewrap(original, "owner:owner-2", keys, 1, 2); !errors.Is(err, ErrSealed) {
		t.Fatalf("Rewrap for another owner: err = %v, want ErrSealed", err)
	}
}

func TestSealRejectsEmptyValueAndUnknownVersion(t *testing.T) {
	keys := testKeys(t, map[int]byte{1: 0x11})
	if _, err := Seal(nil, testScope, keys, 1); err == nil {
		t.Fatal("Seal accepted an empty value")
	}
	if _, err := Seal([]byte("token"), testScope, keys, 3); !errors.Is(err, ErrNoMasterKey) {
		t.Fatalf("Seal with unknown version: err = %v, want ErrNoMasterKey", err)
	}
}

func TestNewMasterKeysValidatesAndPicksHighestVersion(t *testing.T) {
	if _, err := NewMasterKeys(nil); err == nil {
		t.Fatal("NewMasterKeys accepted no keys")
	}
	if _, err := NewMasterKeys(map[int][]byte{1: make([]byte, 16)}); err == nil {
		t.Fatal("NewMasterKeys accepted a 16-byte key")
	}
	if _, err := NewMasterKeys(map[int][]byte{0: make([]byte, keySize)}); err == nil {
		t.Fatal("NewMasterKeys accepted version 0")
	}
	keys := testKeys(t, map[int]byte{1: 0x11, 3: 0x33, 2: 0x22})
	if keys.Current() != 3 {
		t.Fatalf("Current() = %d, want 3", keys.Current())
	}
	if !keys.Has(2) || keys.Has(4) {
		t.Fatal("Has reports the wrong versions")
	}
}

func payloadOf(t *testing.T, ciphertext string) string {
	t.Helper()
	var env envelope
	if err := json.Unmarshal([]byte(ciphertext), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env.Payload
}
