package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// ErrDigestMismatch means the action presented for execution is not the one
// the person approved. Execution must not run.
var ErrDigestMismatch = errors.New("approval digest mismatch")

// ActionKind names a sensitive action. The list is fixed in code.
type ActionKind string

const (
	ActionSendMessage    ActionKind = "send_message"
	ActionSendEmail      ActionKind = "send_email"
	ActionDelete         ActionKind = "delete"
	ActionPublish        ActionKind = "publish"
	ActionPay            ActionKind = "pay"
	ActionDeploy         ActionKind = "deploy"
	ActionMerge          ActionKind = "merge"
	ActionShellCommand   ActionKind = "shell_command"
	ActionConnectorWrite ActionKind = "connector_write"
)

var sensitiveKinds = map[ActionKind]bool{
	ActionSendMessage:    true,
	ActionSendEmail:      true,
	ActionDelete:         true,
	ActionPublish:        true,
	ActionPay:            true,
	ActionDeploy:         true,
	ActionMerge:          true,
	ActionShellCommand:   true,
	ActionConnectorWrite: true,
}

// IsSensitiveAction reports whether kind requires a person's approval.
func IsSensitiveAction(kind ActionKind) bool { return sensitiveKinds[kind] }

// Action is what the agent wants to do: kind, target and arguments.
type Action struct {
	Kind   ActionKind
	Target string
	// Args must be a JSON object. Normalize replaces it with its canonical form.
	Args json.RawMessage
}

// Normalize validates the action and returns it with canonical arguments:
// object keys sorted, insignificant whitespace removed, numbers kept verbatim.
// Two equal actions with different key order therefore share one digest.
func (a Action) Normalize() (Action, error) {
	if !IsSensitiveAction(a.Kind) {
		return Action{}, fmt.Errorf("%w: action kind %q does not need approval", ErrInvalidInput, a.Kind)
	}
	target := strings.TrimSpace(a.Target)
	if target == "" {
		return Action{}, fmt.Errorf("%w: action target is required", ErrInvalidInput)
	}
	args := a.Args
	if len(bytes.TrimSpace(args)) == 0 {
		args = json.RawMessage(`{}`)
	}
	dec := json.NewDecoder(bytes.NewReader(args))
	dec.UseNumber()
	var value map[string]any
	if err := dec.Decode(&value); err != nil {
		return Action{}, fmt.Errorf("%w: action args must be a JSON object: %v", ErrInvalidInput, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return Action{}, fmt.Errorf("%w: action args contain trailing data", ErrInvalidInput)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return Action{}, fmt.Errorf("canonicalize action args: %w", err)
	}
	return Action{Kind: a.Kind, Target: target, Args: canonical}, nil
}

// Digest returns the SHA-256 (hex) of the canonical action. The digest is the
// identity of the approved snapshot.
func (a Action) Digest() (string, error) {
	n, err := a.Normalize()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(struct {
		Action ActionKind      `json:"action"`
		Args   json.RawMessage `json:"args"`
		Target string          `json:"target"`
	}{n.Kind, n.Args, n.Target})
	if err != nil {
		return "", fmt.Errorf("encode action payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// ApprovalStatus is the decision state of an approval.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalDenied   ApprovalStatus = "denied"
	ApprovalExpired  ApprovalStatus = "expired"
)

// ApprovalOutcome is the receipt after the decision: what actually happened.
type ApprovalOutcome string

const (
	OutcomeCompleted ApprovalOutcome = "completed"
	OutcomeFailed    ApprovalOutcome = "failed"
	OutcomeNotRun    ApprovalOutcome = "not_run"
)

// Approval is one request for a person's decision on one action of one run.
type Approval struct {
	ID                string
	RunID             string
	AgentID           string
	Seq               int64
	Action            Action
	Digest            string
	Status            ApprovalStatus
	Outcome           ApprovalOutcome
	OutcomeReason     string
	DecidedByDeviceID string
	DecidedAt         *time.Time
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

// AuditEvent is one append-only record. Facts must never hold secrets or the
// full arguments of an action: store the digest and the kind instead.
type AuditEvent struct {
	ID        string
	At        time.Time
	ActorKind string
	ActorID   string
	Action    string
	Subject   string
	Facts     map[string]any
}
