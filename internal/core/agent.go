// Package core holds the domain model of Calango. It has no network or disk I/O.
package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Domain errors. Adapters map them to API error codes; wrap them with context.
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrInvalidInput = errors.New("invalid input")
)

// RunOn says where an agent runs.
type RunOn string

// Where an agent runs.
const (
	RunOnPC   RunOn = "pc"
	RunOnVPS  RunOn = "vps"
	RunOnAuto RunOn = "auto"
)

// ApprovalLevel says which actions ask the person before they run.
type ApprovalLevel string

// Approval levels, from strictest to loosest.
const (
	ApprovalAlways    ApprovalLevel = "always"
	ApprovalSensitive ApprovalLevel = "sensitive"
	ApprovalTrusted   ApprovalLevel = "trusted"
)

// ReasoningEffort uses the same four names for every provider.
type ReasoningEffort string

// Reasoning effort levels.
const (
	EffortLow    ReasoningEffort = "low"
	EffortMedium ReasoningEffort = "medium"
	EffortHigh   ReasoningEffort = "high"
	EffortMax    ReasoningEffort = "max"
)

// Agent is a personal assistant configured by one person. It holds configuration
// only: budget decisions belong to internal/gateway.
type Agent struct {
	ID              string
	OwnerID         string
	Name            string
	Role            string
	Instructions    string
	Providers       []ProviderRef   // ordered fallback chain; the first entry is tried first
	ReasoningEffort ReasoningEffort // empty means the provider default
	RunOn           RunOn
	TokenLimitMonth int64
	ApprovalLevel   ApprovalLevel
	MaestroEnabled  bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Normalize trims the name and fills the defaults: run on this PC and sensitive approval.
func (a *Agent) Normalize() {
	a.Name = strings.TrimSpace(a.Name)
	if a.RunOn == "" {
		a.RunOn = RunOnPC
	}
	if a.ApprovalLevel == "" {
		a.ApprovalLevel = ApprovalSensitive
	}
}

// Validate checks the configuration. Errors wrap ErrInvalidInput and name the field.
// A token limit is required: without one, the gateway refuses every call.
func (a Agent) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if a.TokenLimitMonth <= 0 {
		return fmt.Errorf("%w: token_limit_month must be positive", ErrInvalidInput)
	}
	switch a.RunOn {
	case RunOnPC, RunOnVPS, RunOnAuto:
	default:
		return fmt.Errorf("%w: run_on must be pc, vps or auto", ErrInvalidInput)
	}
	switch a.ApprovalLevel {
	case ApprovalAlways, ApprovalSensitive, ApprovalTrusted:
	default:
		return fmt.Errorf("%w: approval_level must be always, sensitive or trusted", ErrInvalidInput)
	}
	switch a.ReasoningEffort {
	case "", EffortLow, EffortMedium, EffortHigh, EffortMax:
	default:
		return fmt.Errorf("%w: reasoning_effort must be low, medium, high or max", ErrInvalidInput)
	}
	if n := len(a.Providers); n == 0 || n > MaxProvidersPerAgent {
		return fmt.Errorf("%w: providers must list 1 to %d accounts", ErrInvalidInput, MaxProvidersPerAgent)
	}
	for i, p := range a.Providers {
		if p.AccountID == "" {
			return fmt.Errorf("%w: providers[%d].account_id is required", ErrInvalidInput, i)
		}
		if strings.TrimSpace(p.Model) == "" {
			return fmt.Errorf("%w: providers[%d].model is required", ErrInvalidInput, i)
		}
	}
	return nil
}
