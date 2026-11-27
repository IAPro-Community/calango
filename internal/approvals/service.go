// Package approvals decides whether a sensitive action may run. A person
// approves an exact snapshot; execution runs only that snapshot and records a
// receipt. Every state change is audited in the same transaction.
package approvals

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"calango/internal/core"
	"calango/internal/store"
)

// DefaultTTL is how long a pending approval waits for a decision.
const DefaultTTL = 24 * time.Hour

// Repository is the persistence the service needs. *store.Store implements it.
type Repository interface {
	GetAgent(ctx context.Context, ownerID, id string) (core.Agent, error)
	GetRun(ctx context.Context, id string) (core.Run, error)
	InsertApproval(ctx context.Context, a core.Approval, ev core.AuditEvent) error
	GetApproval(ctx context.Context, ownerID, id string) (core.Approval, error)
	ListApprovals(ctx context.Context, ownerID string, status core.ApprovalStatus) ([]core.Approval, error)
	ListDueApprovals(ctx context.Context, now time.Time) ([]core.Approval, error)
	ListUnreceiptedApprovals(ctx context.Context) ([]core.Approval, error)
	NextApprovalSeq(ctx context.Context, runID string) (int64, error)
	DecideApproval(ctx context.Context, d store.ApprovalDecision) error
	CompleteApproval(ctx context.Context, id string, outcome core.ApprovalOutcome, reason string, ev core.AuditEvent) error
	HasRule(ctx context.Context, agentID string, kind core.ActionKind) (bool, error)
	AppendAudit(ctx context.Context, ev core.AuditEvent) error
}

// Service owns the approval lifecycle. One mutex serializes state changes; it is
// released while an action runs, so long actions do not block other decisions.
type Service struct {
	repo  Repository
	ttl   time.Duration
	now   func() time.Time
	newID func(time.Time) (string, error)

	mu        sync.Mutex
	executing map[string]struct{}
	executors map[core.ActionKind]Executor
}

// NewService builds the service. ttl <= 0 uses DefaultTTL.
func NewService(repo Repository, ttl time.Duration, now func() time.Time, newID func(time.Time) (string, error)) (*Service, error) {
	if repo == nil || now == nil || newID == nil {
		return nil, errors.New("approvals service needs a repository, a clock and an id generator")
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Service{repo: repo, ttl: ttl, now: now, newID: newID,
		executing: make(map[string]struct{}), executors: make(map[core.ActionKind]Executor)}, nil
}

// Request opens an approval for a sensitive action of a run. If the agent has an
// "always allow" rule for the kind, the approval is approved at once with no card,
// and the auto-approval is still audited.
func (s *Service) Request(ctx context.Context, ownerID, agentID, runID string, action core.Action) (core.Approval, error) {
	n, err := action.Normalize()
	if err != nil {
		return core.Approval{}, err
	}
	if _, err := s.repo.GetAgent(ctx, ownerID, agentID); err != nil {
		return core.Approval{}, fmt.Errorf("approval agent: %w", err)
	}
	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return core.Approval{}, fmt.Errorf("approval run: %w", err)
	}
	if run.AgentID != agentID {
		return core.Approval{}, fmt.Errorf("%w: run belongs to another agent", core.ErrInvalidInput)
	}
	s.mu.Lock()
	a, err := s.requestLocked(ctx, agentID, runID, n)
	s.mu.Unlock()
	if err == nil && a.Status == core.ApprovalApproved {
		s.dispatch(ownerID, a)
	}
	return a, err
}

// Executor runs one approved snapshot. It records its own result; a returned error
// becomes the receipt's failed outcome. It runs in its own goroutine.
type Executor func(ctx context.Context, ownerID, approvalID string, action core.Action) error

// SetExecutor registers the executor for one kind of action. Approved actions of
// that kind run through it, both when a person approves and when a rule approves.
func (s *Service) SetExecutor(kind core.ActionKind, fn Executor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executors[kind] = fn
}

// dispatch starts the executor of an approved action in the background. The caller
// must not hold s.mu: Execute takes it.
func (s *Service) dispatch(ownerID string, a core.Approval) {
	s.mu.Lock()
	fn := s.executors[a.Action.Kind]
	s.mu.Unlock()
	if fn == nil {
		return
	}
	go func() {
		_, err := s.Execute(context.Background(), ownerID, a.ID, a.Action,
			func(ctx context.Context, act core.Action) error {
				return fn(ctx, ownerID, a.ID, act)
			})
		if err != nil {
			slog.Warn("approved action did not run", "approval", a.ID, "err", err)
		}
	}()
}

func (s *Service) requestLocked(ctx context.Context, agentID, runID string, n core.Action) (core.Approval, error) {
	digest, err := n.Digest()
	if err != nil {
		return core.Approval{}, err
	}
	now := s.now()
	id, err := s.newID(now)
	if err != nil {
		return core.Approval{}, fmt.Errorf("new approval id: %w", err)
	}
	seq, err := s.repo.NextApprovalSeq(ctx, runID)
	if err != nil {
		return core.Approval{}, err
	}
	ruled, err := s.repo.HasRule(ctx, agentID, n.Kind)
	if err != nil {
		return core.Approval{}, err
	}

	a := core.Approval{
		ID: id, RunID: runID, AgentID: agentID, Seq: seq, Action: n, Digest: digest,
		Status: core.ApprovalPending, ExpiresAt: now.Add(s.ttl), CreatedAt: now,
	}
	action := "approval.requested"
	if ruled {
		decided := now
		a.Status = core.ApprovalApproved
		a.DecidedAt = &decided
		action = "approval.auto_approved"
	}
	ev, err := s.event(id, now, action, map[string]any{
		"kind": string(n.Kind), "digest": digest, "run_id": runID,
	})
	if err != nil {
		return core.Approval{}, err
	}
	if err := s.repo.InsertApproval(ctx, a, ev); err != nil {
		return core.Approval{}, err
	}
	return a, nil
}

// Decide records a person's decision on a pending approval. deviceID is the paired
// device that decides; it is required, so the agent cannot approve its own action.
// remember stores an "always allow" rule for the agent and kind; it applies only to approve.
// An approved action is handed to its executor at once.
func (s *Service) Decide(ctx context.Context, ownerID, id, deviceID string, approve, remember bool) (core.Approval, error) {
	a, err := s.decide(ctx, ownerID, id, deviceID, approve, remember)
	if err == nil && a.Status == core.ApprovalApproved {
		s.dispatch(ownerID, a)
	}
	return a, err
}

// decide validates and records the decision under the lock.
func (s *Service) decide(ctx context.Context, ownerID, id, deviceID string, approve, remember bool) (core.Approval, error) {
	if deviceID == "" {
		return core.Approval{}, fmt.Errorf("%w: a paired device must decide", core.ErrInvalidInput)
	}
	if remember && !approve {
		return core.Approval{}, fmt.Errorf("%w: only an approval can be remembered", core.ErrInvalidInput)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	a, err := s.repo.GetApproval(ctx, ownerID, id)
	if err != nil {
		return core.Approval{}, err
	}
	if a.Status != core.ApprovalPending {
		return core.Approval{}, fmt.Errorf("%w: approval is %s", core.ErrConflict, a.Status)
	}
	now := s.now()
	if !now.Before(a.ExpiresAt) {
		if err := s.expireLocked(ctx, a, now); err != nil {
			return core.Approval{}, err
		}
		return core.Approval{}, fmt.Errorf("%w: approval expired", core.ErrConflict)
	}

	d := store.ApprovalDecision{ID: id, DeviceID: deviceID, At: now}
	if approve {
		d.Status = core.ApprovalApproved
		if remember {
			d.RuleAgentID = a.AgentID
			d.RuleAction = a.Action.Kind
		}
	} else {
		d.Status = core.ApprovalDenied
		d.Outcome = core.OutcomeNotRun
	}
	ev, err := s.event(id, now, "approval.decided", map[string]any{
		"decision": string(d.Status), "digest": a.Digest, "kind": string(a.Action.Kind), "remember": remember,
	})
	if err != nil {
		return core.Approval{}, err
	}
	ev.ActorKind, ev.ActorID = "device", deviceID
	d.Audit = ev

	if err := s.repo.DecideApproval(ctx, d); err != nil {
		return core.Approval{}, err
	}
	return s.repo.GetApproval(ctx, ownerID, id)
}

// Execute runs the approved snapshot, not the action passed in. If the presented
// action differs from what was approved, nothing runs: the attempt is audited, a new
// approval is requested for the presented action, and ErrDigestMismatch is returned.
// The receipt is completed or failed (with the error text) and is written once.
func (s *Service) Execute(ctx context.Context, ownerID, id string, presented core.Action, run func(context.Context, core.Action) error) (core.Approval, error) {
	presentedNorm, err := presented.Normalize()
	if err != nil {
		return core.Approval{}, err
	}
	presentedDigest, err := presentedNorm.Digest()
	if err != nil {
		return core.Approval{}, err
	}

	s.mu.Lock()
	a, err := s.repo.GetApproval(ctx, ownerID, id)
	if err != nil {
		s.mu.Unlock()
		return core.Approval{}, err
	}
	if a.Status != core.ApprovalApproved {
		s.mu.Unlock()
		return core.Approval{}, fmt.Errorf("%w: approval is %s", core.ErrConflict, a.Status)
	}
	if a.Outcome != "" {
		s.mu.Unlock()
		return core.Approval{}, fmt.Errorf("%w: approval already has outcome %s", core.ErrConflict, a.Outcome)
	}
	if _, busy := s.executing[id]; busy {
		s.mu.Unlock()
		return core.Approval{}, fmt.Errorf("%w: approval is already executing", core.ErrConflict)
	}
	stored, err := a.Action.Digest()
	if err != nil {
		s.mu.Unlock()
		return core.Approval{}, fmt.Errorf("approval %s snapshot is invalid: %w", id, err)
	}
	if stored != a.Digest {
		s.mu.Unlock()
		return core.Approval{}, fmt.Errorf("approval %s snapshot does not match its digest", id)
	}

	if presentedDigest != a.Digest {
		next, err := s.refuseLocked(ctx, a, presentedNorm, presentedDigest)
		s.mu.Unlock()
		if err != nil {
			return core.Approval{}, err
		}
		return next, fmt.Errorf("%w: new approval %s requested", core.ErrDigestMismatch, next.ID)
	}

	s.executing[id] = struct{}{}
	s.mu.Unlock()

	runErr := run(ctx, a.Action)

	// Keep the execution fenced until its receipt is durably stored. If
	// recording fails, startup reconciliation must mark the outcome unknown;
	// never allow the same action to execute twice in this process.
	s.mu.Lock()
	outcome, reason := core.OutcomeCompleted, ""
	if runErr != nil {
		outcome, reason = core.OutcomeFailed, runErr.Error()
	}
	ev, evErr := s.event(id, s.now(), "approval.executed", map[string]any{
		"outcome": string(outcome), "digest": a.Digest,
	})
	if evErr != nil {
		s.mu.Unlock()
		return core.Approval{}, evErr
	}
	// Persist the receipt even if ctx was cancelled during the run.
	persistErr := s.repo.CompleteApproval(context.WithoutCancel(ctx), id, outcome, reason, ev)
	if persistErr == nil {
		delete(s.executing, id)
	}
	s.mu.Unlock()
	if persistErr != nil {
		return core.Approval{}, fmt.Errorf("record receipt for approval %s: %w", id, persistErr)
	}
	return s.repo.GetApproval(ctx, ownerID, id)
}

// refuseLocked audits a refused execution and opens a new request for the presented
// action. If the audit cannot be written, nothing new is requested and the error is
// returned: the refusal itself must be on record.
func (s *Service) refuseLocked(ctx context.Context, a core.Approval, presented core.Action, presentedDigest string) (core.Approval, error) {
	now := s.now()
	ev, err := s.event(a.ID, now, "approval.blocked", map[string]any{
		"approved_digest": a.Digest, "presented_digest": presentedDigest,
	})
	if err != nil {
		return core.Approval{}, err
	}
	if err := s.repo.AppendAudit(context.WithoutCancel(ctx), ev); err != nil {
		return core.Approval{}, fmt.Errorf("audit blocked execution of %s: %w", a.ID, err)
	}
	return s.requestLocked(ctx, a.AgentID, a.RunID, presented)
}

// Sweep expires every pending approval past its deadline. It returns how many expired.
func (s *Service) Sweep(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	due, err := s.repo.ListDueApprovals(ctx, now)
	if err != nil {
		return 0, err
	}
	for _, a := range due {
		if err := s.expireLocked(ctx, a, now); err != nil {
			return 0, err
		}
	}
	return len(due), nil
}

// reconcileReason is recorded on approved actions that have no receipt at startup.
const reconcileReason = "no receipt was recorded before the process stopped; the outcome is unknown and the action was not retried"

// Reconcile closes approved actions that have no receipt. They were approved before
// the process stopped, so whether they ran is unknown: each is recorded as failed
// and audited, and never retried. Call it once at startup, before any executor
// runs. It returns how many were closed.
func (s *Service) Reconcile(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	open, err := s.repo.ListUnreceiptedApprovals(ctx)
	if err != nil {
		return 0, err
	}
	now := s.now()
	for _, a := range open {
		ev, err := s.event(a.ID, now, "approval.reconciled", map[string]any{"digest": a.Digest})
		if err != nil {
			return 0, err
		}
		if err := s.repo.CompleteApproval(ctx, a.ID, core.OutcomeFailed, reconcileReason, ev); err != nil {
			return 0, fmt.Errorf("reconcile approval %s: %w", a.ID, err)
		}
	}
	return len(open), nil
}

// List returns the owner's approvals filtered by status; empty status returns all.
func (s *Service) List(ctx context.Context, ownerID string, status core.ApprovalStatus) ([]core.Approval, error) {
	return s.repo.ListApprovals(ctx, ownerID, status)
}

// Get returns one approval of the owner.
func (s *Service) Get(ctx context.Context, ownerID, id string) (core.Approval, error) {
	return s.repo.GetApproval(ctx, ownerID, id)
}

func (s *Service) expireLocked(ctx context.Context, a core.Approval, now time.Time) error {
	ev, err := s.event(a.ID, now, "approval.expired", map[string]any{"digest": a.Digest})
	if err != nil {
		return err
	}
	err = s.repo.DecideApproval(ctx, store.ApprovalDecision{
		ID: a.ID, Status: core.ApprovalExpired, Outcome: core.OutcomeNotRun, At: now, Audit: ev,
	})
	if err != nil {
		return fmt.Errorf("expire approval %s: %w", a.ID, err)
	}
	return nil
}

func (s *Service) event(subject string, at time.Time, action string, facts map[string]any) (core.AuditEvent, error) {
	id, err := s.newID(at)
	if err != nil {
		return core.AuditEvent{}, fmt.Errorf("new audit id: %w", err)
	}
	return core.AuditEvent{ID: id, At: at, ActorKind: "system", Action: action, Subject: subject, Facts: facts}, nil
}
