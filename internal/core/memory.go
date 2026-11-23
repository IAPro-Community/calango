package core

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxMemoryChars bounds one memory. Longer material belongs in a document, not in every prompt.
const MaxMemoryChars = 2000

// MemoryScope says which runs a memory is injected into.
type MemoryScope string

const (
	// ScopeAccount memories are injected into every agent of the owner.
	ScopeAccount MemoryScope = "account"
	// ScopeAgent memories are injected only into runs of AgentID.
	ScopeAgent MemoryScope = "agent"
)

// MemoryKind classifies a memory for the person reading it.
type MemoryKind string

const (
	KindFact       MemoryKind = "fact"
	KindPreference MemoryKind = "preference"
	KindDecision   MemoryKind = "decision"
)

// Memory is something the agent keeps across runs. Deleted memories keep their
// row until the retention period ends, but they never reach a prompt.
type Memory struct {
	ID        string
	OwnerID   string
	Scope     MemoryScope
	AgentID   string // set only for ScopeAgent
	Kind      MemoryKind
	Text      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// Normalize trims the text and defaults the kind to fact.
func (m *Memory) Normalize() {
	m.Text = strings.TrimSpace(m.Text)
	if m.Kind == "" {
		m.Kind = KindFact
	}
}

// Validate checks a memory that has been normalized.
func (m Memory) Validate() error {
	if m.Text == "" {
		return fmt.Errorf("%w: memory text is required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(m.Text) > MaxMemoryChars {
		return fmt.Errorf("%w: memory text exceeds %d characters", ErrInvalidInput, MaxMemoryChars)
	}
	switch m.Kind {
	case KindFact, KindPreference, KindDecision:
	default:
		return fmt.Errorf("%w: unknown memory kind %q", ErrInvalidInput, m.Kind)
	}
	switch m.Scope {
	case ScopeAccount:
		if m.AgentID != "" {
			return fmt.Errorf("%w: account memory cannot name an agent", ErrInvalidInput)
		}
	case ScopeAgent:
		if m.AgentID == "" {
			return fmt.Errorf("%w: agent memory needs an agent", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unknown memory scope %q", ErrInvalidInput, m.Scope)
	}
	return nil
}

// EstimateTokens approximates the prompt cost of text: one token per four
// characters, rounded up. It is only used to fit memory into a budget.
func EstimateTokens(text string) int {
	return (utf8.RuneCountInString(text) + 3) / 4
}
