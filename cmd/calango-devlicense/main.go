// Command calango-devlicense creates development key pairs and test licenses for
// internal/license. It is not the license service: production keys never live in this
// repository, and the node binary does not include this command.
//
//	calango-devlicense keygen -out dev-license.key
//	calango-devlicense issue -key dev-license.key -account acct-dev -plan pro -days 7 > calango.license
//
// Run the node with CALANGO_LICENSE_PUBLIC_KEY set to the printed public key (only
// builds without an embedded key accept it) and CALANGO_LICENSE_FILE pointing at the
// issued file.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"calango/internal/license"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr, time.Now); err != nil {
		fmt.Fprintln(os.Stderr, "calango-devlicense:", err)
		os.Exit(2)
	}
}

func run(args []string, stdout, stderr io.Writer, now func() time.Time) error {
	if len(args) == 0 {
		return errors.New("usage: calango-devlicense keygen|issue [flags]")
	}
	switch args[0] {
	case "keygen":
		return keygen(args[1:], stdout, stderr)
	case "issue":
		return issue(args[1:], stdout, stderr, now)
	default:
		return fmt.Errorf("unknown command %q; want keygen or issue", args[0])
	}
}

// keygen writes a new private key seed (base64, mode 0600) and prints the public key.
func keygen(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "file for the private key seed; must not exist")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("keygen: -out is required")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("keygen: %w", err)
	}
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("keygen: %w", err)
	}
	if _, err := fmt.Fprintln(f, base64.StdEncoding.EncodeToString(priv.Seed())); err != nil {
		_ = f.Close()
		return fmt.Errorf("keygen: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("keygen: %w", err)
	}
	_, err = fmt.Fprintln(stdout, license.EncodePublicKey(pub))
	return err
}

// issue signs a test license with a key from keygen and prints the token.
func issue(args []string, stdout, stderr io.Writer, now func() time.Time) error {
	fs := flag.NewFlagSet("issue", flag.ContinueOnError)
	fs.SetOutput(stderr)
	keyFile := fs.String("key", "", "private key seed written by keygen")
	account := fs.String("account", "acct-dev", "account_id claim")
	plan := fs.String("plan", license.PlanPro, "plan claim: pro, team or business")
	features := fs.String("features", "", "comma-separated features claim")
	nodes := fs.Int("nodes", 1, "nodes_max claim; -1 for unlimited")
	days := fs.Int("days", 7, "validity in days from now; negative issues an already expired license")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *keyFile == "" {
		return errors.New("issue: -key is required")
	}
	raw, err := os.ReadFile(*keyFile)
	if err != nil {
		return fmt.Errorf("issue: %w", err)
	}
	seed, err := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return errors.New("issue: key file is not a base64 Ed25519 seed")
	}
	at := now().UTC().Truncate(time.Second)
	exp := at.Add(time.Duration(*days) * 24 * time.Hour)
	iat := at
	if exp.Before(iat) {
		iat = exp
	}
	var list []string
	for _, f := range strings.Split(*features, ",") {
		if f = strings.TrimSpace(f); f != "" {
			list = append(list, f)
		}
	}
	token, err := license.Sign(ed25519.NewKeyFromSeed(seed), license.Claims{
		AccountID: *account, Plan: *plan, Features: list, NodesMax: *nodes,
		IssuedAt: iat.Unix(), ExpiresAt: exp.Unix(),
	})
	if err != nil {
		return fmt.Errorf("issue: %w", err)
	}
	_, err = fmt.Fprintln(stdout, token)
	return err
}
