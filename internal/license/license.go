// Package license verifies Ed25519-signed licenses offline and derives the node's
// effective plan. A license that is missing, invalid or past its tolerance falls back
// to Free: the node keeps running and no data is changed or removed.
package license

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// TokenPrefix opens every license token and is part of the signed input.
const TokenPrefix = "calango-license-v1"

// GracePeriod is how long a license keeps its plan after exp, so an offline node
// does not lose paid features while it cannot renew.
const GracePeriod = 7 * 24 * time.Hour

// MaxTokenBytes bounds a token read from disk or configuration.
const MaxTokenBytes = 8 << 10

// claimsVersion is the only claims layout this binary understands.
const claimsVersion = 1

// Plans a license can grant. Free needs no license.
const (
	PlanFree     = "free"
	PlanPro      = "pro"
	PlanTeam     = "team"
	PlanBusiness = "business"
)

// Errors returned by Verify. Each one makes the license invalid; none stops the node.
var (
	ErrMalformed   = errors.New("license: malformed token")
	ErrSignature   = errors.New("license: signature does not match the public key")
	ErrClaims      = errors.New("license: invalid claims")
	ErrNoPublicKey = errors.New("license: no public key configured")
	ErrPublicKey   = errors.New("license: invalid public key")
)

// Claims is the signed body of a license, as issued by the Calango license service.
// Times are Unix seconds.
type Claims struct {
	Version   int      `json:"v"`
	AccountID string   `json:"account_id"`
	Plan      string   `json:"plan"`
	Features  []string `json:"features"`
	// NodesMax is the number of VPS nodes the account may run; -1 means unlimited.
	NodesMax  int   `json:"nodes_max"`
	IssuedAt  int64 `json:"iat"`
	ExpiresAt int64 `json:"exp"`
}

// Expires returns exp as a time.
func (c Claims) Expires() time.Time { return time.Unix(c.ExpiresAt, 0).UTC() }

// GraceEnds returns the instant after which the license no longer grants its plan.
func (c Claims) GraceEnds() time.Time { return c.Expires().Add(GracePeriod) }

func (c Claims) validate() error {
	switch {
	case c.Version != claimsVersion:
		return fmt.Errorf("%w: unsupported version %d", ErrClaims, c.Version)
	case strings.TrimSpace(c.AccountID) == "":
		return fmt.Errorf("%w: account_id is required", ErrClaims)
	case c.Plan != PlanPro && c.Plan != PlanTeam && c.Plan != PlanBusiness:
		return fmt.Errorf("%w: unknown plan", ErrClaims)
	case c.NodesMax != -1 && c.NodesMax < 1:
		return fmt.Errorf("%w: nodes_max must be -1 or at least 1", ErrClaims)
	case c.ExpiresAt <= 0:
		return fmt.Errorf("%w: exp is required", ErrClaims)
	case c.IssuedAt < 0 || c.IssuedAt > c.ExpiresAt:
		return fmt.Errorf("%w: iat must not be after exp", ErrClaims)
	}
	for _, f := range c.Features {
		if strings.TrimSpace(f) == "" {
			return fmt.Errorf("%w: empty feature", ErrClaims)
		}
	}
	return nil
}

// ParsePublicKey decodes a standard base64 Ed25519 public key (32 bytes).
func ParsePublicKey(text string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(text))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, ErrPublicKey
	}
	return ed25519.PublicKey(raw), nil
}

// EncodePublicKey is the inverse of ParsePublicKey.
func EncodePublicKey(key ed25519.PublicKey) string {
	return base64.StdEncoding.EncodeToString(key)
}

// Verifier checks license tokens against one public key, without network.
type Verifier struct {
	key ed25519.PublicKey
}

// NewVerifier returns a verifier for key.
func NewVerifier(key ed25519.PublicKey) (*Verifier, error) {
	if len(key) != ed25519.PublicKeySize {
		return nil, ErrPublicKey
	}
	return &Verifier{key: bytes.Clone(key)}, nil
}

// Verify checks the token's form and signature and returns its claims. It does not
// look at time; Evaluate decides between active, grace and expired.
func (v *Verifier) Verify(token string) (Claims, error) {
	token = strings.TrimSpace(token)
	if len(token) > MaxTokenBytes {
		return Claims{}, fmt.Errorf("%w: token too long", ErrMalformed)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != TokenPrefix {
		return Claims{}, fmt.Errorf("%w: want %s.<payload>.<signature>", ErrMalformed, TokenPrefix)
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil || len(payload) == 0 {
		return Claims{}, fmt.Errorf("%w: payload is not base64url", ErrMalformed)
	}
	sig, err := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	if err != nil || len(sig) != ed25519.SignatureSize {
		return Claims{}, fmt.Errorf("%w: signature is not a base64url Ed25519 signature", ErrMalformed)
	}
	if !ed25519.Verify(v.key, signingInput(parts[1]), sig) {
		return Claims{}, ErrSignature
	}
	var c Claims
	dec := json.NewDecoder(bytes.NewReader(payload))
	if err := dec.Decode(&c); err != nil {
		return Claims{}, fmt.Errorf("%w: payload is not a claims object", ErrClaims)
	}
	if dec.More() {
		return Claims{}, fmt.Errorf("%w: trailing data after claims", ErrClaims)
	}
	if err := c.validate(); err != nil {
		return Claims{}, err
	}
	return c, nil
}

// Sign builds a token for claims. Only the license service and the development
// tool hold a private key; the node never signs.
func Sign(priv ed25519.PrivateKey, c Claims) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", errors.New("license: invalid private key")
	}
	if c.Version == 0 {
		c.Version = claimsVersion
	}
	if err := c.validate(); err != nil {
		return "", err
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("license: encode claims: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	sig := ed25519.Sign(priv, signingInput(encoded))
	return TokenPrefix + "." + encoded + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// signingInput is what the signature covers: the prefix and the encoded payload, so a
// signature cannot be moved to another token format.
func signingInput(encodedPayload string) []byte {
	return []byte(TokenPrefix + "." + encodedPayload)
}
