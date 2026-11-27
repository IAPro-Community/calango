package license

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var issued = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)

// testKey derives a fixed key pair from seed so tests never need a real key.
func testKey(seed byte) (ed25519.PublicKey, ed25519.PrivateKey) {
	priv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, ed25519.SeedSize))
	return priv.Public().(ed25519.PublicKey), priv
}

func testClaims() Claims {
	return Claims{
		AccountID: "acct-1",
		Plan:      PlanPro,
		Features:  []string{"vps"},
		NodesMax:  1,
		IssuedAt:  issued.Unix(),
		ExpiresAt: issued.Add(7 * 24 * time.Hour).Unix(),
	}
}

func signed(t *testing.T, priv ed25519.PrivateKey, c Claims) string {
	t.Helper()
	token, err := Sign(priv, c)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	return token
}

func verifier(t *testing.T, pub ed25519.PublicKey) *Verifier {
	t.Helper()
	v, err := NewVerifier(pub)
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	return v
}

func TestVerifyAcceptsSignedLicense(t *testing.T) {
	pub, priv := testKey(1)
	c, err := verifier(t, pub).Verify(signed(t, priv, testClaims()))
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if c.AccountID != "acct-1" || c.Plan != PlanPro || c.NodesMax != 1 || c.Version != 1 {
		t.Fatalf("Verify() claims = %+v", c)
	}
}

func TestVerifyRefusesWrongKey(t *testing.T) {
	_, priv := testKey(1)
	other, _ := testKey(2)
	_, err := verifier(t, other).Verify(signed(t, priv, testClaims()))
	if !errors.Is(err, ErrSignature) {
		t.Fatalf("Verify() with another key error = %v, want ErrSignature", err)
	}
}

func TestVerifyRefusesTamperedPayload(t *testing.T) {
	pub, priv := testKey(1)
	token := signed(t, priv, testClaims())
	parts := strings.Split(token, ".")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	// Upgrade the plan and the node limit without re-signing.
	forged := strings.Replace(string(payload), `"plan":"pro"`, `"plan":"business"`, 1)
	forged = strings.Replace(forged, `"nodes_max":1`, `"nodes_max":-1`, 1)
	if forged == string(payload) {
		t.Fatal("tamper did not change the payload")
	}
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(forged))
	if _, err := verifier(t, pub).Verify(strings.Join(parts, ".")); !errors.Is(err, ErrSignature) {
		t.Fatalf("Verify() tampered payload error = %v, want ErrSignature", err)
	}
}

func TestVerifyRefusesTamperedSignature(t *testing.T) {
	pub, priv := testKey(1)
	parts := strings.Split(signed(t, priv, testClaims()), ".")
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	sig[0] ^= 0x01
	parts[2] = base64.RawURLEncoding.EncodeToString(sig)
	if _, err := verifier(t, pub).Verify(strings.Join(parts, ".")); !errors.Is(err, ErrSignature) {
		t.Fatalf("Verify() tampered signature error = %v, want ErrSignature", err)
	}
}

func TestVerifyRefusesSignatureMovedToAnotherPrefix(t *testing.T) {
	pub, priv := testKey(1)
	parts := strings.Split(signed(t, priv, testClaims()), ".")
	parts[0] = "calango-license-v2"
	if _, err := verifier(t, pub).Verify(strings.Join(parts, ".")); !errors.Is(err, ErrMalformed) {
		t.Fatalf("Verify() other prefix error = %v, want ErrMalformed", err)
	}
}

func TestVerifyRefusesMalformedTokens(t *testing.T) {
	pub, priv := testKey(1)
	good := strings.Split(signed(t, priv, testClaims()), ".")
	for name, token := range map[string]string{
		"empty":           "",
		"two parts":       good[0] + "." + good[1],
		"four parts":      strings.Join(good, ".") + ".x",
		"bad payload b64": good[0] + ".!!!." + good[2],
		"padded payload":  good[0] + "." + good[1] + "==." + good[2],
		"short signature": good[0] + "." + good[1] + "." + base64.RawURLEncoding.EncodeToString([]byte("short")),
		"too long":        good[0] + "." + strings.Repeat("A", MaxTokenBytes) + "." + good[2],
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier(t, pub).Verify(token); !errors.Is(err, ErrMalformed) {
				t.Fatalf("Verify(%q) error = %v, want ErrMalformed", name, err)
			}
		})
	}
}

// signRaw signs an arbitrary payload, to prove claim checks run after the signature.
func signRaw(priv ed25519.PrivateKey, payload string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	sig := ed25519.Sign(priv, signingInput(encoded))
	return TokenPrefix + "." + encoded + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifyRefusesSignedButInvalidClaims(t *testing.T) {
	pub, priv := testKey(1)
	for name, payload := range map[string]string{
		"not json":        `plan=pro`,
		"version 2":       `{"v":2,"account_id":"a","plan":"pro","nodes_max":1,"iat":1,"exp":2}`,
		"no account":      `{"v":1,"plan":"pro","nodes_max":1,"iat":1,"exp":2}`,
		"unknown plan":    `{"v":1,"account_id":"a","plan":"enterprise","nodes_max":1,"iat":1,"exp":2}`,
		"free plan":       `{"v":1,"account_id":"a","plan":"free","nodes_max":1,"iat":1,"exp":2}`,
		"zero nodes":      `{"v":1,"account_id":"a","plan":"pro","nodes_max":0,"iat":1,"exp":2}`,
		"no exp":          `{"v":1,"account_id":"a","plan":"pro","nodes_max":1,"iat":1}`,
		"iat after exp":   `{"v":1,"account_id":"a","plan":"pro","nodes_max":1,"iat":3,"exp":2}`,
		"empty feature":   `{"v":1,"account_id":"a","plan":"pro","features":[" "],"nodes_max":1,"iat":1,"exp":2}`,
		"trailing object": `{"v":1,"account_id":"a","plan":"pro","nodes_max":1,"iat":1,"exp":2}{}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier(t, pub).Verify(signRaw(priv, payload)); !errors.Is(err, ErrClaims) {
				t.Fatalf("Verify(%s) error = %v, want ErrClaims", name, err)
			}
		})
	}
}

func TestSignRefusesInvalidClaims(t *testing.T) {
	_, priv := testKey(1)
	c := testClaims()
	c.Plan = "gold"
	if _, err := Sign(priv, c); !errors.Is(err, ErrClaims) {
		t.Fatalf("Sign() unknown plan error = %v, want ErrClaims", err)
	}
}

func TestEvaluateStatesAcrossTime(t *testing.T) {
	pub, priv := testKey(1)
	v := verifier(t, pub)
	token := signed(t, priv, testClaims())
	exp := issued.Add(7 * 24 * time.Hour)
	for _, tc := range []struct {
		name  string
		at    time.Time
		state State
		plan  string
	}{
		{"at issue", issued, StateActive, PlanPro},
		{"one second before exp", exp.Add(-time.Second), StateActive, PlanPro},
		{"at exp", exp, StateGrace, PlanPro},
		{"six days into tolerance", exp.Add(6 * 24 * time.Hour), StateGrace, PlanPro},
		{"last second of tolerance", exp.Add(GracePeriod - time.Second), StateGrace, PlanPro},
		{"tolerance over", exp.Add(GracePeriod), StateExpired, PlanFree},
		{"offline ten days after a seven-day license", issued.Add(17 * 24 * time.Hour), StateExpired, PlanFree},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := Evaluate(v, token, tc.at)
			if st.State != tc.state || st.Plan != tc.plan {
				t.Fatalf("Evaluate() at %s = %s/%s, want %s/%s", tc.at, st.State, st.Plan, tc.state, tc.plan)
			}
			if st.Claims == nil || st.Claims.AccountID != "acct-1" {
				t.Fatalf("Evaluate() claims = %+v, want them kept for the warning", st.Claims)
			}
		})
	}
}

func TestEvaluateFallsBackToFree(t *testing.T) {
	pub, priv := testKey(1)
	other, _ := testKey(2)
	token := signed(t, priv, testClaims())
	for _, tc := range []struct {
		name   string
		v      *Verifier
		token  string
		state  State
		reason string
	}{
		{"no license", verifier(t, pub), "  \n", StateNone, ""},
		{"no public key", nil, token, StateInvalid, ReasonNoPublicKey},
		{"wrong key", verifier(t, other), token, StateInvalid, ReasonSignature},
		{"garbage", verifier(t, pub), "not-a-license", StateInvalid, ReasonMalformed},
		{"bad claims", verifier(t, pub), signRaw(priv, `{"v":1}`), StateInvalid, ReasonClaims},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := Evaluate(tc.v, tc.token, issued)
			if st.State != tc.state || st.Reason != tc.reason || st.Plan != PlanFree || st.Paid() || st.Claims != nil {
				t.Fatalf("Evaluate() = %+v, want %s/%s on Free without claims", st, tc.state, tc.reason)
			}
		})
	}
}

func TestEvaluateAcceptsTrailingNewline(t *testing.T) {
	pub, priv := testKey(1)
	st := Evaluate(verifier(t, pub), signed(t, priv, testClaims())+"\n", issued)
	if st.State != StateActive {
		t.Fatalf("Evaluate() with trailing newline = %s, want active", st.State)
	}
}

// snapshot records every file under dir with its contents and mode.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		entry := info.Mode().String()
		if !d.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry += ":" + string(data)
		}
		out[path] = entry
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	return out
}

func TestManagerDowngradesWithoutTouchingData(t *testing.T) {
	pub, priv := testKey(1)
	dir := t.TempDir()
	path := filepath.Join(dir, "calango.license")
	if err := os.WriteFile(path, []byte(signed(t, priv, testClaims())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "calango.db"), []byte("user data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "workspace", "project"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "workspace", "project", "notes.md"), []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, dir)

	now := issued
	m, err := NewManager(path, pub, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if st := m.Status(); st.State != StateActive || st.Plan != PlanPro {
		t.Fatalf("Status() at issue = %s/%s, want active/pro", st.State, st.Plan)
	}
	now = issued.Add(10 * 24 * time.Hour)
	if st := m.Status(); st.State != StateGrace || st.Plan != PlanPro {
		t.Fatalf("Status() in tolerance = %s/%s, want grace/pro", st.State, st.Plan)
	}
	now = issued.Add(30 * 24 * time.Hour)
	st := m.Status()
	if st.State != StateExpired || st.Plan != PlanFree || st.Claims == nil {
		t.Fatalf("Status() after tolerance = %+v, want expired on Free with claims", st)
	}
	if after := snapshot(t, dir); len(after) != len(before) {
		t.Fatalf("data directory changed: before %v, after %v", before, after)
	} else {
		for path, entry := range before {
			if after[path] != entry {
				t.Fatalf("%s changed after downgrade: %q -> %q", path, entry, after[path])
			}
		}
	}

	// Installing a renewed license restores the plan without a restart.
	renewed := testClaims()
	renewed.IssuedAt = now.Unix()
	renewed.ExpiresAt = now.Add(7 * 24 * time.Hour).Unix()
	if err := os.WriteFile(path, []byte(signed(t, priv, renewed)), 0o600); err != nil {
		t.Fatal(err)
	}
	if st := m.Status(); st.State != StateActive || st.Plan != PlanPro {
		t.Fatalf("Status() after renewal = %s/%s, want active/pro", st.State, st.Plan)
	}
}

func TestManagerMissingAndUnreadableFiles(t *testing.T) {
	pub, _ := testKey(1)
	dir := t.TempDir()
	m, err := NewManager(filepath.Join(dir, "absent.license"), pub, func() time.Time { return issued })
	if err != nil {
		t.Fatal(err)
	}
	if st := m.Status(); st.State != StateNone || st.Plan != PlanFree {
		t.Fatalf("Status() without file = %+v, want none/free", st)
	}
	// A directory in place of the file cannot be read as a token.
	if err := os.Mkdir(filepath.Join(dir, "dir.license"), 0o700); err != nil {
		t.Fatal(err)
	}
	m, err = NewManager(filepath.Join(dir, "dir.license"), pub, func() time.Time { return issued })
	if err != nil {
		t.Fatal(err)
	}
	if st := m.Status(); st.State != StateInvalid || st.Reason != ReasonUnreadable || st.Plan != PlanFree {
		t.Fatalf("Status() unreadable = %+v, want invalid/unreadable on Free", st)
	}
}

func TestManagerWithoutKeyRunsFree(t *testing.T) {
	_, priv := testKey(1)
	path := filepath.Join(t.TempDir(), "calango.license")
	if err := os.WriteFile(path, []byte(signed(t, priv, testClaims())), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(path, nil, func() time.Time { return issued })
	if err != nil {
		t.Fatalf("NewManager() without key error = %v", err)
	}
	if st := m.Status(); st.State != StateInvalid || st.Reason != ReasonNoPublicKey || st.Plan != PlanFree {
		t.Fatalf("Status() without key = %+v, want invalid/no_public_key on Free", st)
	}
}

func TestNewManagerValidatesInput(t *testing.T) {
	pub, _ := testKey(1)
	clock := func() time.Time { return issued }
	if _, err := NewManager("", pub, clock); err == nil {
		t.Fatal("NewManager() without path succeeded")
	}
	if _, err := NewManager("x", pub, nil); err == nil {
		t.Fatal("NewManager() without clock succeeded")
	}
	if _, err := NewManager("x", ed25519.PublicKey{1, 2, 3}, clock); !errors.Is(err, ErrPublicKey) {
		t.Fatalf("NewManager() short key error = %v, want ErrPublicKey", err)
	}
}

func TestParsePublicKey(t *testing.T) {
	pub, _ := testKey(1)
	got, err := ParsePublicKey(EncodePublicKey(pub) + "\n")
	if err != nil || !got.Equal(pub) {
		t.Fatalf("ParsePublicKey() = %x, %v", got, err)
	}
	for _, bad := range []string{"", "not base64", base64.StdEncoding.EncodeToString([]byte("short"))} {
		if _, err := ParsePublicKey(bad); !errors.Is(err, ErrPublicKey) {
			t.Fatalf("ParsePublicKey(%q) error = %v, want ErrPublicKey", bad, err)
		}
	}
}

func TestResolvePublicKeyPrefersEmbedded(t *testing.T) {
	embedded, _ := testKey(1)
	configured, _ := testKey(2)
	t.Cleanup(func() { embeddedPublicKey = "" })

	embeddedPublicKey = ""
	if key, src, err := ResolvePublicKey(""); err != nil || key != nil || src != KeySourceNone {
		t.Fatalf("ResolvePublicKey() nothing = %x, %s, %v", key, src, err)
	}
	if key, src, err := ResolvePublicKey(EncodePublicKey(configured)); err != nil || !key.Equal(configured) || src != KeySourceConfigured {
		t.Fatalf("ResolvePublicKey() configured = %x, %s, %v", key, src, err)
	}
	if _, _, err := ResolvePublicKey("bad"); !errors.Is(err, ErrPublicKey) {
		t.Fatalf("ResolvePublicKey(bad) error = %v", err)
	}

	embeddedPublicKey = EncodePublicKey(embedded)
	if key, src, err := ResolvePublicKey(EncodePublicKey(configured)); err != nil || !key.Equal(embedded) || src != KeySourceEmbedded {
		t.Fatalf("ResolvePublicKey() embedded = %x, %s, %v; the embedded key must win", key, src, err)
	}
	embeddedPublicKey = "broken"
	if _, _, err := ResolvePublicKey(""); !errors.Is(err, ErrPublicKey) {
		t.Fatalf("ResolvePublicKey() broken embedded error = %v", err)
	}
}
