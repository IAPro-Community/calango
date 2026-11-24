package core

import (
	"fmt"
	"strings"
	"time"
)

// Conversation kinds. Only dm exists in V1; channel arrives with phase 6.
const (
	ConversationDM = "dm"
)

// Author kinds for messages.
const (
	AuthorPerson = "person"
	AuthorAgent  = "agent"
	AuthorSystem = "system"
)

// RunStatus is the lifecycle state of one run.
type RunStatus string

const (
	RunQueued          RunStatus = "queued"
	RunRunning         RunStatus = "running"
	RunDone            RunStatus = "done"
	RunFailed          RunStatus = "failed"
	RunStopped         RunStatus = "stopped"
	RunWaitingApproval RunStatus = "waiting_approval"
)

// Terminal reports whether the run has finished and will not change again.
func (s RunStatus) Terminal() bool {
	return s == RunDone || s == RunFailed || s == RunStopped
}

// Run trigger values.
const (
	TriggerMessage = "message"
	TriggerRoutine = "routine"
	TriggerTask    = "task"
)

// Step kinds recorded in run_steps.
const (
	StepThought         = "thought"
	StepToolCall        = "tool_call"
	StepToolResult      = "tool_result"
	StepText            = "text"
	StepProviderAttempt = "provider_attempt"
)

// Error codes stored in runs.error_code.
const (
	ErrCodeInterrupted        = "interrupted"
	ErrCodeAgentPaused        = "agent_paused"
	ErrCodeBudgetExceeded     = "budget_exceeded"
	ErrCodeProviderError      = "provider_error"
	ErrCodeHarnessSignedOut   = "harness_signed_out"
	ErrCodeStoreError         = "store_error"
	ErrCodeNoProvider         = "no_provider"
	ErrCodeMaestroUnavailable = "maestro_unavailable"
)

// MaxMessageBytes bounds one message body.
const MaxMessageBytes = 64 * 1024

// MaxInstructionBytes bounds the extra direction given on retry.
const MaxInstructionBytes = 8 * 1024

// Conversation is a DM or, later, a channel.
type Conversation struct {
	ID        string
	Kind      string
	Title     string
	CreatedAt time.Time
}

// Message is one entry in a conversation.
type Message struct {
	ID             string
	ConversationID string
	AuthorKind     string
	AuthorID       string
	Body           string
	CreatedAt      time.Time
}

// Run is one execution of an agent for one source message.
type Run struct {
	ID              string
	AgentID         string
	ConversationID  string
	SourceMessageID string
	IdempotencyKey  string
	Trigger         string
	Status          RunStatus
	Instructions    string
	RetryOf         string
	SteerOf         string
	TokensIn        int64
	TokensOut       int64
	StartedAt       *time.Time
	EndedAt         *time.Time
	ErrorCode       string
}

// Step is one recorded event of a run. Seq orders steps within a run.
type Step struct {
	ID        string
	RunID     string
	Seq       int64
	Kind      string
	Payload   string
	CreatedAt time.Time
}

// ValidateMessageBody rejects empty or oversized messages.
func ValidateMessageBody(body string) error {
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("%w: message body is empty", ErrInvalidInput)
	}
	if len(body) > MaxMessageBytes {
		return fmt.Errorf("%w: message body exceeds %d bytes", ErrInvalidInput, MaxMessageBytes)
	}
	return nil
}

// ValidateInstructions rejects empty or oversized retry instructions.
func ValidateInstructions(text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("%w: instructions are empty", ErrInvalidInput)
	}
	if len(text) > MaxInstructionBytes {
		return fmt.Errorf("%w: instructions exceed %d bytes", ErrInvalidInput, MaxInstructionBytes)
	}
	return nil
}
