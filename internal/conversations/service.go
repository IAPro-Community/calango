// Package conversations runs DMs with agents: messages, runs, recorded steps,
// stop, retry with instructions and the per-conversation parallel limit. Every
// model call goes through the gateway.
package conversations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"calango/internal/core"
	"calango/internal/gateway"
)

const (
	defaultParallelLimit   = 3
	defaultMaxOutputTokens = 4096
	maxIdempotencyKeyBytes = 200
)

// Repository is the persistence the service needs. The SQLite store implements it.
type Repository interface {
	InsertConversation(ctx context.Context, c core.Conversation, agentID string) error
	ConversationAgent(ctx context.Context, conversationID string) (string, error)
	AgentByID(ctx context.Context, id string) (core.Agent, error)
	GetAgent(ctx context.Context, ownerID, id string) (core.Agent, error)
	GlobalPaused(ctx context.Context) (bool, error)
	AdmitRun(ctx context.Context, msg *core.Message, run core.Run) error
	GetMessage(ctx context.Context, id string) (core.Message, error)
	ListMessages(ctx context.Context, conversationID string) ([]core.Message, error)
	GetRun(ctx context.Context, id string) (core.Run, error)
	GetRunByIdempotencyKey(ctx context.Context, key string) (core.Run, error)
	ClaimRun(ctx context.Context, id string, at time.Time) (bool, error)
	FinishRun(ctx context.Context, id string, status core.RunStatus, at time.Time, errCode string, tokensIn, tokensOut int64) error
	AppendStep(ctx context.Context, st core.Step) error
	ListSteps(ctx context.Context, runID string) ([]core.Step, error)
	CountRunningRuns(ctx context.Context, conversationID string) (int, error)
	OldestQueuedRun(ctx context.Context, conversationID string) (core.Run, error)
	ConversationsWithQueuedRuns(ctx context.Context) ([]string, error)
	InterruptRunningRuns(ctx context.Context, at time.Time) (int64, error)
}

// Gateway is the only way to call a model. *gateway.Gateway implements it.
type Gateway interface {
	Call(ctx context.Context, who gateway.Caller, req gateway.Request, stream func(gateway.Chunk)) (gateway.Usage, error)
}

// MemoryContext supplies the remembered facts placed before a run's prompt.
// *memory.Service implements it. A nil MemoryContext disables injection.
type MemoryContext interface {
	Context(ctx context.Context, ownerID, agentID string) (string, error)
}

// Config wires a Service. Now and NewID are injected so tests are deterministic.
type Config struct {
	Repo    Repository
	Gateway Gateway
	// Memory supplies the remembered facts injected before each run. Nil disables injection.
	Memory MemoryContext
	// Provider names the gateway provider used for every run in V1.
	Provider string
	// MaxOutputTokens bounds each model call. Zero means the default.
	MaxOutputTokens int64
	// ParallelLimit caps running runs per conversation. Zero means the default (3).
	ParallelLimit int
	Now           func() time.Time
	NewID         func(time.Time) (string, error)
}

// Service coordinates conversations and their runs.
type Service struct {
	cfg Config

	// mu serializes admission, dispatch and run completion, so the parallel
	// limit and idempotency checks cannot race.
	mu     sync.Mutex
	active map[string]*activeRun
	wg     sync.WaitGroup
}

type activeRun struct {
	cancel        context.CancelFunc
	stopRequested bool
}

// RunDetail is a run with its recorded steps.
type RunDetail struct {
	Run   core.Run
	Steps []core.Step
}

// NewService validates cfg and applies defaults.
func NewService(cfg Config) (*Service, error) {
	if cfg.Repo == nil || cfg.Gateway == nil || cfg.Now == nil || cfg.NewID == nil {
		return nil, errors.New("conversations service needs a repository, gateway, clock and id generator")
	}
	if strings.TrimSpace(cfg.Provider) == "" {
		return nil, errors.New("conversations service needs a provider name")
	}
	if cfg.ParallelLimit == 0 {
		cfg.ParallelLimit = defaultParallelLimit
	}
	if cfg.ParallelLimit < 1 {
		return nil, fmt.Errorf("parallel limit %d must be at least 1", cfg.ParallelLimit)
	}
	if cfg.MaxOutputTokens == 0 {
		cfg.MaxOutputTokens = defaultMaxOutputTokens
	}
	if cfg.MaxOutputTokens < 1 {
		return nil, fmt.Errorf("max output tokens %d must be positive", cfg.MaxOutputTokens)
	}
	return &Service{cfg: cfg, active: make(map[string]*activeRun)}, nil
}

// CreateDM opens a DM with one of the owner's agents. An empty title uses the agent's name.
func (s *Service) CreateDM(ctx context.Context, ownerID, agentID, title string) (core.Conversation, error) {
	agent, err := s.cfg.Repo.GetAgent(ctx, ownerID, agentID)
	if err != nil {
		return core.Conversation{}, err
	}
	now := s.cfg.Now().UTC()
	id, err := s.cfg.NewID(now)
	if err != nil {
		return core.Conversation{}, fmt.Errorf("generate conversation id: %w", err)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = agent.Name
	}
	c := core.Conversation{ID: id, Kind: core.ConversationDM, Title: title, CreatedAt: now}
	if err := s.cfg.Repo.InsertConversation(ctx, c, agent.ID); err != nil {
		return core.Conversation{}, err
	}
	return c, nil
}

// SendMessage stores a message from ownerID and admits one run for the DM's
// agent. Repeating the same idempotency key returns the stored message and run.
// A paused node stores the message and records a run failed with agent_paused.
func (s *Service) SendMessage(ctx context.Context, ownerID, conversationID, body, idempotencyKey string) (core.Message, core.Run, error) {
	if err := validateKey(idempotencyKey); err != nil {
		return core.Message{}, core.Run{}, err
	}
	if err := core.ValidateMessageBody(body); err != nil {
		return core.Message{}, core.Run{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.cfg.Repo.GetRunByIdempotencyKey(ctx, idempotencyKey)
	switch {
	case err == nil:
		if existing.ConversationID != conversationID {
			return core.Message{}, core.Run{}, fmt.Errorf("%w: idempotency key belongs to another conversation", core.ErrConflict)
		}
		if _, err := s.authorize(ctx, ownerID, conversationID); err != nil {
			return core.Message{}, core.Run{}, err
		}
		msg, err := s.cfg.Repo.GetMessage(ctx, existing.SourceMessageID)
		return msg, existing, err
	case !errors.Is(err, core.ErrNotFound):
		return core.Message{}, core.Run{}, err
	}

	agent, err := s.authorize(ctx, ownerID, conversationID)
	if err != nil {
		return core.Message{}, core.Run{}, err
	}
	now := s.cfg.Now().UTC()
	msgID, err := s.cfg.NewID(now)
	if err != nil {
		return core.Message{}, core.Run{}, fmt.Errorf("generate message id: %w", err)
	}
	runID, err := s.cfg.NewID(now)
	if err != nil {
		return core.Message{}, core.Run{}, fmt.Errorf("generate run id: %w", err)
	}
	msg := core.Message{
		ID: msgID, ConversationID: conversationID, AuthorKind: core.AuthorPerson,
		AuthorID: ownerID, Body: body, CreatedAt: now,
	}
	run := core.Run{
		ID: runID, AgentID: agent.ID, ConversationID: conversationID, SourceMessageID: msgID,
		IdempotencyKey: idempotencyKey, Trigger: core.TriggerMessage, Status: core.RunQueued,
	}
	run, err = s.admitLocked(ctx, &msg, run, now)
	return msg, run, err
}

// Stop ends one run. A running run is cancelled and ends as stopped once its
// provider call returns; a queued run ends at once. Other runs are untouched.
func (s *Service) Stop(ctx context.Context, ownerID, runID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, err := s.cfg.Repo.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if _, err := s.authorize(ctx, ownerID, run.ConversationID); err != nil {
		return err
	}
	if a := s.active[runID]; a != nil {
		a.stopRequested = true
		a.cancel()
		return nil
	}
	if run.Status == core.RunQueued {
		return s.cfg.Repo.FinishRun(ctx, runID, core.RunStopped, s.cfg.Now().UTC(), "", 0, 0)
	}
	return fmt.Errorf("%w: run %s is %s", core.ErrConflict, runID, run.Status)
}

// Retry redoes a stopped or failed run in the same conversation, against the
// same source message, with extra instructions. It is a new run linked by retry_of.
func (s *Service) Retry(ctx context.Context, ownerID, runID, instructions, idempotencyKey string) (core.Run, error) {
	if err := validateKey(idempotencyKey); err != nil {
		return core.Run{}, err
	}
	if err := core.ValidateInstructions(instructions); err != nil {
		return core.Run{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.cfg.Repo.GetRunByIdempotencyKey(ctx, idempotencyKey)
	switch {
	case err == nil:
		if existing.RetryOf != runID {
			return core.Run{}, fmt.Errorf("%w: idempotency key belongs to another retry", core.ErrConflict)
		}
		if _, err := s.authorize(ctx, ownerID, existing.ConversationID); err != nil {
			return core.Run{}, err
		}
		return existing, nil
	case !errors.Is(err, core.ErrNotFound):
		return core.Run{}, err
	}

	old, err := s.cfg.Repo.GetRun(ctx, runID)
	if err != nil {
		return core.Run{}, err
	}
	if _, err := s.authorize(ctx, ownerID, old.ConversationID); err != nil {
		return core.Run{}, err
	}
	if old.Status != core.RunStopped && old.Status != core.RunFailed {
		return core.Run{}, fmt.Errorf("%w: only stopped or failed runs can be retried, run %s is %s",
			core.ErrConflict, runID, old.Status)
	}

	now := s.cfg.Now().UTC()
	newID, err := s.cfg.NewID(now)
	if err != nil {
		return core.Run{}, fmt.Errorf("generate run id: %w", err)
	}
	run := core.Run{
		ID: newID, AgentID: old.AgentID, ConversationID: old.ConversationID,
		SourceMessageID: old.SourceMessageID, IdempotencyKey: idempotencyKey,
		Trigger: core.TriggerMessage, Status: core.RunQueued,
		Instructions: strings.TrimSpace(instructions), RetryOf: old.ID,
	}
	return s.admitLocked(ctx, nil, run, now)
}

// ListMessages returns a conversation's messages if ownerID owns its agent.
func (s *Service) ListMessages(ctx context.Context, ownerID, conversationID string) ([]core.Message, error) {
	if _, err := s.authorize(ctx, ownerID, conversationID); err != nil {
		return nil, err
	}
	return s.cfg.Repo.ListMessages(ctx, conversationID)
}

// GetRun returns a run with its steps if ownerID owns its agent.
func (s *Service) GetRun(ctx context.Context, ownerID, runID string) (RunDetail, error) {
	run, err := s.cfg.Repo.GetRun(ctx, runID)
	if err != nil {
		return RunDetail{}, err
	}
	if _, err := s.authorize(ctx, ownerID, run.ConversationID); err != nil {
		return RunDetail{}, err
	}
	steps, err := s.cfg.Repo.ListSteps(ctx, runID)
	if err != nil {
		return RunDetail{}, err
	}
	return RunDetail{Run: run, Steps: steps}, nil
}

// Start must run once at process startup, before any SendMessage. Runs left in
// running state by a crash end as failed/interrupted. Queued runs are dispatched again.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, err := s.cfg.Repo.InterruptRunningRuns(ctx, s.cfg.Now().UTC())
	if err != nil {
		return err
	}
	if n > 0 {
		slog.Warn("interrupted runs from previous process", "count", n)
	}
	conversations, err := s.cfg.Repo.ConversationsWithQueuedRuns(ctx)
	if err != nil {
		return err
	}
	for _, id := range conversations {
		if err := s.pumpLocked(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// Wait blocks until every run started so far has finished.
func (s *Service) Wait() {
	s.wg.Wait()
}

// authorize returns the conversation's agent if ownerID owns it. A foreign
// conversation reports core.ErrNotFound, so ids do not leak across owners.
func (s *Service) authorize(ctx context.Context, ownerID, conversationID string) (core.Agent, error) {
	agentID, err := s.cfg.Repo.ConversationAgent(ctx, conversationID)
	if err != nil {
		return core.Agent{}, err
	}
	return s.cfg.Repo.GetAgent(ctx, ownerID, agentID)
}

// admitLocked stores the run (with its message, if any) and dispatches it. A
// paused node stores the run as failed with agent_paused instead, so a repeated
// request with the same key still returns the same resource.
func (s *Service) admitLocked(ctx context.Context, msg *core.Message, run core.Run, now time.Time) (core.Run, error) {
	paused, err := s.cfg.Repo.GlobalPaused(ctx)
	if err != nil {
		return core.Run{}, err
	}
	if paused {
		run.Status = core.RunFailed
		run.EndedAt = &now
		run.ErrorCode = core.ErrCodeAgentPaused
	}
	if err := s.cfg.Repo.AdmitRun(ctx, msg, run); err != nil {
		return core.Run{}, err
	}
	if run.Status == core.RunQueued {
		if err := s.pumpLocked(ctx, run.ConversationID); err != nil {
			return core.Run{}, err
		}
	}
	return s.cfg.Repo.GetRun(ctx, run.ID)
}

// pumpLocked starts queued runs of a conversation while it has free slots.
// Callers hold s.mu.
func (s *Service) pumpLocked(ctx context.Context, conversationID string) error {
	running, err := s.cfg.Repo.CountRunningRuns(ctx, conversationID)
	if err != nil {
		return err
	}
	for running < s.cfg.ParallelLimit {
		next, err := s.cfg.Repo.OldestQueuedRun(ctx, conversationID)
		if errors.Is(err, core.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		claimed, err := s.cfg.Repo.ClaimRun(ctx, next.ID, s.cfg.Now().UTC())
		if err != nil {
			return err
		}
		if !claimed {
			return fmt.Errorf("run %s left the queue while dispatching", next.ID)
		}
		running++
		s.start(next)
	}
	return nil
}

// start registers the run as active before its goroutine begins, so Stop can
// always find it once it is running. Callers hold s.mu.
func (s *Service) start(run core.Run) {
	ctx, cancel := context.WithCancel(context.Background())
	s.active[run.ID] = &activeRun{cancel: cancel}
	s.wg.Add(1)
	go s.execute(ctx, run)
}

func (s *Service) execute(ctx context.Context, run core.Run) {
	defer s.wg.Done()

	status, errCode, usage := s.call(ctx, run)

	s.mu.Lock()
	defer s.mu.Unlock()

	if a := s.active[run.ID]; a != nil {
		delete(s.active, run.ID)
		a.cancel()
		if a.stopRequested && status != core.RunDone {
			status, errCode = core.RunStopped, ""
		}
	}
	if err := s.cfg.Repo.FinishRun(context.Background(), run.ID, status, s.cfg.Now().UTC(),
		errCode, usage.TokensIn, usage.TokensOut); err != nil {
		slog.Error("record run result", "run", run.ID, "err", err)
	}
	if err := s.pumpLocked(context.Background(), run.ConversationID); err != nil {
		slog.Error("dispatch queued runs", "conversation", run.ConversationID, "err", err)
	}
}

// call performs the model call for one run and records its streamed text as
// steps. It returns the terminal status, the error code and the usage.
func (s *Service) call(ctx context.Context, run core.Run) (core.RunStatus, string, gateway.Usage) {
	msg, err := s.cfg.Repo.GetMessage(context.Background(), run.SourceMessageID)
	if err != nil {
		slog.Error("load source message", "run", run.ID, "err", err)
		return core.RunFailed, core.ErrCodeStoreError, gateway.Usage{}
	}
	agent, err := s.cfg.Repo.AgentByID(context.Background(), run.AgentID)
	if err != nil {
		slog.Error("load run agent", "run", run.ID, "err", err)
		return core.RunFailed, core.ErrCodeStoreError, gateway.Usage{}
	}

	prompt := msg.Body
	if run.Instructions != "" {
		prompt += "\n\nInstructions for this retry:\n" + run.Instructions
	}
	if s.cfg.Memory != nil {
		block, err := s.cfg.Memory.Context(context.Background(), agent.OwnerID, agent.ID)
		if err != nil {
			slog.Error("load memory context", "run", run.ID, "err", err)
			return core.RunFailed, core.ErrCodeStoreError, gateway.Usage{}
		}
		if block != "" {
			prompt = block + "\n\n" + prompt
		}
	}

	var stepErr error
	var seq int64
	stream := func(c gateway.Chunk) {
		if stepErr != nil {
			return
		}
		payload, err := json.Marshal(map[string]string{"text": c.Text})
		if err != nil {
			stepErr = fmt.Errorf("encode step: %w", err)
			return
		}
		now := s.cfg.Now().UTC()
		id, err := s.cfg.NewID(now)
		if err != nil {
			stepErr = fmt.Errorf("generate step id: %w", err)
			return
		}
		stepErr = s.cfg.Repo.AppendStep(context.Background(), core.Step{
			ID: id, RunID: run.ID, Seq: seq, Kind: core.StepText, Payload: string(payload), CreatedAt: now,
		})
		seq++
	}

	usage, err := s.cfg.Gateway.Call(ctx, gateway.Caller{AgentID: agent.ID}, gateway.Request{
		Provider:        s.cfg.Provider,
		Model:           agent.Model,
		Prompt:          prompt,
		MaxOutputTokens: s.cfg.MaxOutputTokens,
	}, stream)

	switch {
	case stepErr != nil:
		slog.Error("record run step", "run", run.ID, "err", stepErr)
		return core.RunFailed, core.ErrCodeStoreError, usage
	case errors.Is(err, gateway.ErrBudgetExceeded):
		return core.RunFailed, core.ErrCodeBudgetExceeded, usage
	case err != nil:
		slog.Warn("model call failed", "run", run.ID, "err", err)
		return core.RunFailed, core.ErrCodeProviderError, usage
	}
	return core.RunDone, "", usage
}

func validateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("%w: idempotency key is required", core.ErrInvalidInput)
	}
	if len(key) > maxIdempotencyKeyBytes {
		return fmt.Errorf("%w: idempotency key exceeds %d bytes", core.ErrInvalidInput, maxIdempotencyKeyBytes)
	}
	return nil
}
