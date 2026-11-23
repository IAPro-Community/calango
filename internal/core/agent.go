// Package core holds the domain model of Calango. It has no network or disk I/O.
package core

import (
	"errors"
	"fmt"
)

// ErrBudgetExceeded is returned when a charge would push an agent past its token limit.
var ErrBudgetExceeded = errors.New("token budget exceeded")

// Agent is a personal assistant that spends model tokens against a budget.
type Agent struct {
	ID           string
	Name         string
	Instructions string
	// TokenLimit is the maximum tokens the agent may spend. Zero means no limit is set,
	// so Charge refuses every spend until a positive limit is configured.
	TokenLimit int64
	tokensUsed int64
}

// NewAgent validates the fields that every agent needs.
func NewAgent(id, name string, tokenLimit int64) (*Agent, error) {
	if id == "" {
		return nil, errors.New("agent id is required")
	}
	if name == "" {
		return nil, errors.New("agent name is required")
	}
	if tokenLimit <= 0 {
		return nil, fmt.Errorf("agent %s: token limit must be positive", id)
	}
	return &Agent{ID: id, Name: name, TokenLimit: tokenLimit}, nil
}

// TokensUsed reports how many tokens the agent has spent so far.
func (a *Agent) TokensUsed() int64 {
	return a.tokensUsed
}

// Charge records n spent tokens. It refuses, without recording anything, when the spend
// would exceed TokenLimit.
func (a *Agent) Charge(n int64) error {
	if n < 0 {
		return fmt.Errorf("charge must be non-negative, got %d", n)
	}
	if n > a.TokenLimit-a.tokensUsed {
		return ErrBudgetExceeded
	}
	a.tokensUsed += n
	return nil
}
