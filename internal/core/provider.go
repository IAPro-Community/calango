package core

import (
	"fmt"
	"regexp"
	"time"
)

// Provider kinds. An account's kind selects the adapter that runs its calls.
const (
	ProviderKindCodex     = "codex"
	ProviderKindAnthropic = "anthropic"
)

// MaxProvidersPerAgent bounds an agent's fallback chain.
const MaxProvidersPerAgent = 8

// DefaultAccountLabel names the account that keeps the pre-chain behaviour: the
// shared Codex home, and the anthropic-api-key vault item.
const DefaultAccountLabel = "default"

var accountLabelPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// ProviderAccount is a named login to one provider. It holds no secret: the key
// lives in the vault and the Codex login lives in its own home directory.
type ProviderAccount struct {
	ID        string
	OwnerID   string
	Kind      string
	Label     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Name is the key the gateway routes by: "<kind>/<label>".
func (a ProviderAccount) Name() string { return ProviderKeyOf(a.Kind, a.Label) }

// ProviderKeyOf builds the gateway key for an account.
func ProviderKeyOf(kind, label string) string { return kind + "/" + label }

// Validate checks the kind and label. Errors wrap ErrInvalidInput.
func (a ProviderAccount) Validate() error {
	switch a.Kind {
	case ProviderKindCodex, ProviderKindAnthropic:
	default:
		return fmt.Errorf("%w: kind must be codex or anthropic", ErrInvalidInput)
	}
	if !accountLabelPattern.MatchString(a.Label) {
		return fmt.Errorf("%w: label must be 1-32 lowercase letters, digits or hyphens", ErrInvalidInput)
	}
	return nil
}

// ProviderRef is one step of an agent's chain. Kind and Label are filled when
// the chain is read; Validate only looks at the account id and model.
type ProviderRef struct {
	AccountID string
	Kind      string
	Label     string
	Model     string
}

// Key is the gateway key of the referenced account.
func (r ProviderRef) Key() string { return ProviderKeyOf(r.Kind, r.Label) }
