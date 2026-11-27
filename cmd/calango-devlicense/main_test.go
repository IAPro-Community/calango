package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"calango/internal/license"
)

func TestKeygenAndIssueProduceAVerifiableLicense(t *testing.T) {
	at := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { return at }
	keyFile := filepath.Join(t.TempDir(), "dev.key")

	var pubOut bytes.Buffer
	if err := run([]string{"keygen", "-out", keyFile}, &pubOut, io.Discard, clock); err != nil {
		t.Fatalf("keygen: %v", err)
	}
	info, err := os.Stat(keyFile)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	// Windows não tem bits de permissão POSIX; o modo lido lá não diz nada.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("key file mode = %v; want 0600", info.Mode().Perm())
	}
	if err := run([]string{"keygen", "-out", keyFile}, io.Discard, io.Discard, clock); err == nil {
		t.Fatal("keygen overwrote an existing key file")
	}
	pub, err := license.ParsePublicKey(pubOut.String())
	if err != nil {
		t.Fatalf("printed public key: %v", err)
	}

	var tokenOut bytes.Buffer
	args := []string{"issue", "-key", keyFile, "-account", "acct-dev", "-plan", "team", "-features", "vps, push", "-nodes", "-1", "-days", "7"}
	if err := run(args, &tokenOut, io.Discard, clock); err != nil {
		t.Fatalf("issue: %v", err)
	}
	v, err := license.NewVerifier(pub)
	if err != nil {
		t.Fatal(err)
	}
	st := license.Evaluate(v, tokenOut.String(), at)
	if st.State != license.StateActive || st.Plan != "team" || st.Claims.NodesMax != -1 ||
		strings.Join(st.Claims.Features, ",") != "vps,push" || !st.Claims.Expires().Equal(at.Add(7*24*time.Hour)) {
		t.Fatalf("issued license = %+v", st)
	}

	tokenOut.Reset()
	if err := run([]string{"issue", "-key", keyFile, "-days", "-8"}, &tokenOut, io.Discard, clock); err != nil {
		t.Fatalf("issue expired: %v", err)
	}
	if st := license.Evaluate(v, tokenOut.String(), at); st.State != license.StateExpired || st.Plan != license.PlanFree {
		t.Fatalf("expired license = %s/%s", st.State, st.Plan)
	}
}

func TestRunRejectsBadInput(t *testing.T) {
	clock := time.Now
	bad := filepath.Join(t.TempDir(), "bad.key")
	if err := os.WriteFile(bad, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"no command":   nil,
		"unknown":      {"sign"},
		"keygen no -o": {"keygen"},
		"issue no key": {"issue"},
		"bad key":      {"issue", "-key", bad},
		"bad plan":     {"issue", "-key", bad, "-plan", "gold"},
	} {
		if err := run(args, io.Discard, io.Discard, clock); err == nil {
			t.Fatalf("%s: run(%v) succeeded", name, args)
		}
	}
}
