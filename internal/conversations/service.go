// Package conversations runs DMs with agents: messages, runs, recorded steps,
// stop, retry with instructions, one active run per agent and per-conversation
// parallel limits. Every model call goes through the gateway.
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
	"calango/internal/events"
	"calango/internal/gateway"
	"calango/internal/harness"

	"github.com/IAPro-Community/IAPro-SDK/routing"
)

const (
	defaultParallelLimit   = 3
	defaultMaxOutputTokens = 4096
	maxIdempotencyKeyBytes = 200
)

// Repository is the persistence the service needs. The SQLite store implements it.
type Repository interface {
	InsertConversation(ctx context.Context, c core.Conversation, agentID string) error
	ListConversations(ctx context.Context, ownerID string) ([]core.Conversation, error)
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
	FinishRunWithMessage(ctx context.Context, id string, status core.RunStatus, at time.Time, errCode string, tokensIn, tokensOut int64, message *core.Message) error
	AppendStep(ctx context.Context, st core.Step) error
	ListSteps(ctx context.Context, runID string) ([]core.Step, error)
	CountRunningRuns(ctx context.Context, conversationID string) (int, error)
	CountActiveAgentRuns(ctx context.Context, agentID string) (int, error)
	OldestQueuedAgentRun(ctx context.Context, agentID string) (core.Run, error)
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

// Tools opens the MCP endpoint for one run. *mcpgw.Gateway implements it. The
// endpoint is nil when the agent has no grants and the gateway offers no built-in
// tools; release must be called when the run's model calls end. A nil Tools disables
// tool access.
type Tools interface {
	Open(ctx context.Context, ownerID, agentID, runID string) (*gateway.ToolEndpoint, func(), error)
}

// Publisher receives the changes the service makes visible to clients. *events.Hub implements it.
type Publisher interface {
	Publish(events.Event)
}

// CapacityRouter selects once before the first dispatch; strict mode never
// advances to another account after a gateway attempt.
type CapacityRouter interface {
	SelectCapacityRoute(context.Context, core.Agent, string) (core.ProviderRef, *gateway.CapacityRoute, error)
}

// Config wires a Service. Now and NewID are injected so tests are deterministic.
type Config struct {
	Repo           Repository
	Gateway        Gateway
	CapacityRouter CapacityRouter
	// Memory supplies the remembered facts injected before each run. Nil disables injection.
	Memory MemoryContext
	// Tools gives each run its MCP endpoint. Nil disables tool access.
	Tools Tools
	// MaestroHome resolves the active isolated Codex home for linked agents.
	MaestroHome func() (string, error)
	// Events receives message and run changes for connected clients. Nil disables publishing.
	Events Publisher
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

	// mu serializes admission, dispatch and run completion, so per-agent and
	// per-conversation limits and idempotency checks cannot race.
	mu     sync.Mutex
	active map[string]*activeRun
	wg     sync.WaitGroup
}

type activeRun struct {
	cancel        context.CancelFunc
	stopRequested bool
	steer         *core.Run
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

// ListConversations returns the conversations of ownerID's agents, newest first.
func (s *Service) ListConversations(ctx context.Context, ownerID string) ([]core.Conversation, error) {
	return s.cfg.Repo.ListConversations(ctx, ownerID)
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
		if a.steer != nil {
			return fmt.Errorf("%w: run %s is being steered", core.ErrConflict, runID)
		}
		a.stopRequested = true
		a.cancel()
		return nil
	}
	if run.Status == core.RunQueued {
		if err := s.cfg.Repo.FinishRun(ctx, runID, core.RunStopped, s.cfg.Now().UTC(), "", 0, 0); err != nil {
			return err
		}
		s.publishRun(events.TypeRunFinished, runID, run.AgentID)
		return s.pumpAgentLocked(ctx, run.AgentID)
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

// Steer gives a queued or running run the owner's guidance and returns the
// continuation run that carries it (steer_of). The steered run stops. A queued
// run stops at once and its continuation is queued; a running run is cancelled
// and its continuation is queued once it has ended, so the continuation sees
// the output streamed so far. A running run accepts one guidance; repeating the
// same key returns the same continuation.
func (s *Service) Steer(ctx context.Context, ownerID, runID, guidance, idempotencyKey string) (core.Run, error) {
	if err := validateKey(idempotencyKey); err != nil {
		return core.Run{}, err
	}
	if err := core.ValidateInstructions(guidance); err != nil {
		return core.Run{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.cfg.Repo.GetRunByIdempotencyKey(ctx, idempotencyKey)
	switch {
	case err == nil:
		if existing.SteerOf != runID {
			return core.Run{}, fmt.Errorf("%w: idempotency key belongs to another run", core.ErrConflict)
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
	now := s.cfg.Now().UTC()
	a := s.active[runID]
	switch {
	case a != nil && a.stopRequested:
		return core.Run{}, fmt.Errorf("%w: run %s is stopping", core.ErrConflict, runID)
	case a != nil && a.steer != nil:
		if a.steer.IdempotencyKey == idempotencyKey {
			return *a.steer, nil
		}
		return core.Run{}, fmt.Errorf("%w: run %s already has guidance", core.ErrConflict, runID)
	case a != nil:
		cont, err := s.continuation(old, guidance, idempotencyKey, now)
		if err != nil {
			return core.Run{}, err
		}
		a.steer = &cont
		a.cancel()
		return cont, nil
	case old.Status == core.RunQueued:
		if err := s.cfg.Repo.FinishRun(ctx, runID, core.RunStopped, now, "", 0, 0); err != nil {
			return core.Run{}, err
		}
		s.publishRun(events.TypeRunFinished, runID, old.AgentID)
		cont, err := s.continuation(old, guidance, idempotencyKey, now)
		if err != nil {
			return core.Run{}, err
		}
		return s.admitLocked(ctx, nil, cont, now)
	default:
		return core.Run{}, fmt.Errorf("%w: only queued or running runs can be steered, run %s is %s",
			core.ErrConflict, runID, old.Status)
	}
}

// continuation builds the run that carries guidance for old. It has the same
// source message, so it answers the same request, and steer_of points at old.
func (s *Service) continuation(old core.Run, guidance, key string, now time.Time) (core.Run, error) {
	id, err := s.cfg.NewID(now)
	if err != nil {
		return core.Run{}, fmt.Errorf("generate run id: %w", err)
	}
	return core.Run{
		ID: id, AgentID: old.AgentID, ConversationID: old.ConversationID,
		SourceMessageID: old.SourceMessageID, IdempotencyKey: key,
		Trigger: core.TriggerMessage, Status: core.RunQueued,
		Instructions: strings.TrimSpace(guidance), SteerOf: old.ID,
	}, nil
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

// waitPollInterval is how often WaitRun checks a run's status.
const waitPollInterval = 250 * time.Millisecond

// WaitRun blocks until runID is neither queued nor running, then returns its detail.
// Any final status counts, including failed and stopped; the caller decides what it means.
// ctx ends the wait early with ctx's error.
func (s *Service) WaitRun(ctx context.Context, ownerID, runID string) (RunDetail, error) {
	ticker := time.NewTicker(waitPollInterval)
	defer ticker.Stop()
	for {
		detail, err := s.GetRun(ctx, ownerID, runID)
		if err != nil {
			return RunDetail{}, err
		}
		// A run waiting for a decision is not finished; it resumes as running.
		if status := detail.Run.Status; status != core.RunQueued && status != core.RunRunning && status != core.RunWaitingApproval {
			return detail, nil
		}
		select {
		case <-ctx.Done():
			return RunDetail{}, ctx.Err()
		case <-ticker.C:
		}
	}
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
	seenAgents := make(map[string]struct{}, len(conversations))
	for _, id := range conversations {
		agentID, err := s.cfg.Repo.ConversationAgent(ctx, id)
		if err != nil {
			return err
		}
		if _, seen := seenAgents[agentID]; seen {
			continue
		}
		seenAgents[agentID] = struct{}{}
		if err := s.pumpAgentLocked(ctx, agentID); err != nil {
			return err
		}
	}
	return nil
}

// Wait blocks until every run started so far has finished.
func (s *Service) Wait() {
	s.wg.Wait()
}

// publish sends a change to the owner's connected clients. It does nothing when no publisher is configured.
func (s *Service) publish(ownerID, typ string, data any) {
	if s.cfg.Events == nil {
		return
	}
	s.cfg.Events.Publish(events.Event{Type: typ, OwnerID: ownerID, Data: data})
}

// publishRun sends the stored state of runID as typ, for the owner of agentID.
func (s *Service) publishRun(typ, runID, agentID string) {
	if s.cfg.Events == nil {
		return
	}
	ctx := context.Background()
	agent, err := s.cfg.Repo.AgentByID(ctx, agentID)
	if err != nil {
		slog.Error("load event owner", "run", runID, "err", err)
		return
	}
	run, err := s.cfg.Repo.GetRun(ctx, runID)
	if err != nil {
		slog.Error("load event run", "run", runID, "err", err)
		return
	}
	s.publish(agent.OwnerID, typ, run)
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
	if msg != nil {
		s.publish(msg.AuthorID, events.TypeMessageCreated, *msg)
	}
	if run.Status == core.RunQueued {
		if err := s.pumpAgentLocked(ctx, run.AgentID); err != nil {
			return core.Run{}, err
		}
	}
	return s.cfg.Repo.GetRun(ctx, run.ID)
}

// pumpAgentLocked starts the oldest queued run for an agent when neither the
// agent nor the run's conversation currently holds a slot. Callers hold s.mu.
func (s *Service) pumpAgentLocked(ctx context.Context, agentID string) error {
	active, err := s.cfg.Repo.CountActiveAgentRuns(ctx, agentID)
	if err != nil {
		return err
	}
	if active > 0 {
		return nil
	}
	next, err := s.cfg.Repo.OldestQueuedAgentRun(ctx, agentID)
	if errors.Is(err, core.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	running, err := s.cfg.Repo.CountRunningRuns(ctx, next.ConversationID)
	if err != nil {
		return err
	}
	if running >= s.cfg.ParallelLimit {
		return nil
	}
	claimed, err := s.cfg.Repo.ClaimRun(ctx, next.ID, s.cfg.Now().UTC())
	if err != nil {
		return err
	}
	if !claimed {
		return fmt.Errorf("run %s left the queue while dispatching", next.ID)
	}
	s.start(next)
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

	var steer *core.Run
	if a := s.active[run.ID]; a != nil {
		delete(s.active, run.ID)
		a.cancel()
		steer = a.steer
		if (a.stopRequested || steer != nil) && status != core.RunDone {
			status, errCode = core.RunStopped, ""
		}
	}

	finishedAt := s.cfg.Now().UTC()
	var reply *core.Message
	if status == core.RunDone && run.Trigger == core.TriggerMessage {
		var err error
		reply, err = s.agentReply(run, finishedAt)
		switch {
		case err != nil:
			slog.Error("prepare agent reply", "run", run.ID, "err", err)
			status, errCode = core.RunFailed, core.ErrCodeStoreError
		case reply == nil:
			status, errCode = core.RunFailed, core.ErrCodeProviderError
		}
	}
	if err := s.cfg.Repo.FinishRunWithMessage(context.Background(), run.ID, status, finishedAt,
		errCode, usage.TokensIn, usage.TokensOut, reply); err != nil {
		slog.Error("record run result", "run", run.ID, "err", err)
	} else if reply != nil {
		s.publishAgentMessage(*reply)
	}
	s.publishRun(events.TypeRunFinished, run.ID, run.AgentID)
	if steer != nil {
		if _, err := s.admitLocked(context.Background(), nil, *steer, s.cfg.Now().UTC()); err != nil {
			slog.Error("queue steered run", "run", run.ID, "err", err)
		}
	}
	if err := s.pumpAgentLocked(context.Background(), run.AgentID); err != nil {
		slog.Error("dispatch queued runs", "agent", run.AgentID, "err", err)
	}
}

func (s *Service) agentReply(run core.Run, at time.Time) (*core.Message, error) {
	steps, err := s.cfg.Repo.ListSteps(context.Background(), run.ID)
	if err != nil {
		return nil, fmt.Errorf("list completed run steps: %w", err)
	}
	body, err := ReplyText(steps)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(body) == "" || len(body) > core.MaxMessageBytes {
		return nil, nil
	}
	id, err := s.cfg.NewID(at)
	if err != nil {
		return nil, fmt.Errorf("generate reply id: %w", err)
	}
	return &core.Message{
		ID: id, ConversationID: run.ConversationID, AuthorKind: core.AuthorAgent,
		AuthorID: run.AgentID, Body: body, CreatedAt: at,
	}, nil
}

func (s *Service) publishAgentMessage(message core.Message) {
	if s.cfg.Events == nil {
		return
	}
	agent, err := s.cfg.Repo.AgentByID(context.Background(), message.AuthorID)
	if err != nil {
		slog.Error("load agent reply owner", "message", message.ID, "err", err)
		return
	}
	s.publish(agent.OwnerID, events.TypeMessageCreated, message)
}

// call runs the agent's provider chain for one run (ADR 0016). Entries run in
// order; the next entry runs only when the failure is one another provider may
// avoid and no output has reached the client. Each attempt is one gateway call
// and one provider_attempt step. Text and attempt steps share one sequence. It
// returns the terminal status, the error code and the usage summed over attempts.
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
	s.publishRun(events.TypeRunStarted, run.ID, agent.ID)

	var maestroHome string
	if agent.MaestroEnabled {
		for _, ref := range agent.Providers {
			if ref.Kind != "codex" {
				continue
			}
			if s.cfg.MaestroHome == nil {
				return core.RunFailed, core.ErrCodeMaestroUnavailable, gateway.Usage{}
			}
			maestroHome, err = s.cfg.MaestroHome()
			if err != nil || maestroHome == "" {
				slog.Warn("linked Maestro home is unavailable", "run", run.ID)
				return core.RunFailed, core.ErrCodeMaestroUnavailable, gateway.Usage{}
			}
			break
		}
	}
	if len(agent.Providers) == 0 {
		return core.RunFailed, core.ErrCodeNoProvider, gateway.Usage{}
	}

	prompt := msg.Body
	if run.SteerOf != "" {
		partial, err := s.partialOutput(context.Background(), run.SteerOf)
		if err != nil {
			slog.Error("load steered output", "run", run.ID, "err", err)
			return core.RunFailed, core.ErrCodeStoreError, gateway.Usage{}
		}
		prompt = steeredPrompt(msg.Body, partial, run.Instructions)
	} else if run.Instructions != "" {
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

	var (
		stepErr error
		seq     int64
		emitted bool
		total   gateway.Usage
	)
	appendStep := func(kind string, payload map[string]string) {
		if stepErr != nil {
			return
		}
		raw, err := json.Marshal(payload)
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
		st := core.Step{
			ID: id, RunID: run.ID, Seq: seq, Kind: kind, Payload: string(raw), CreatedAt: now,
		}
		stepErr = s.cfg.Repo.AppendStep(context.Background(), st)
		if stepErr == nil {
			s.publish(agent.OwnerID, events.TypeRunStep, st)
		}
		seq++
	}
	stream := func(c gateway.Chunk) {
		emitted = true
		appendStep(core.StepText, map[string]string{"text": c.Text})
	}

	// One endpoint per run: its token stays valid until the run's model calls end.
	var endpoint *gateway.ToolEndpoint
	if s.cfg.Tools != nil {
		ep, release, err := s.cfg.Tools.Open(ctx, agent.OwnerID, agent.ID, run.ID)
		if err != nil {
			slog.Error("open run tools", "run", run.ID, "err", err)
			return core.RunFailed, core.ErrCodeStoreError, gateway.Usage{}
		}
		defer release()
		endpoint = ep
	}

	var route *gateway.CapacityRoute
	chain := agent.Providers
	if agent.CapacityRouting {
		if s.cfg.CapacityRouter == nil {
			return core.RunFailed, core.ErrCodeProviderError, total
		}
		selected, decision, err := s.cfg.CapacityRouter.SelectCapacityRoute(ctx, agent, run.ID)
		if err != nil || decision == nil {
			appendStep(core.StepProviderAttempt, map[string]string{
				"account": chain[0].Key(), "model": chain[0].Model,
				"outcome": "blocked", "error_code": core.ErrCodeProviderError,
				"routing_reason": "capacity_intervention_required",
			})
			if stepErr != nil {
				return core.RunFailed, core.ErrCodeStoreError, total
			}
			return core.RunFailed, core.ErrCodeProviderError, total
		}
		route = decision
		chain = []core.ProviderRef{selected}
		appendStep(core.StepProviderAttempt, map[string]string{
			"account": selected.Key(), "model": selected.Model, "outcome": "selected",
			"error_code": "", "routing_reason": route.Reason, "source_account": route.SourceKey,
		})
		if stepErr != nil {
			return core.RunFailed, core.ErrCodeStoreError, total
		}
	}

	var code string
	for _, ref := range chain {
		request := gateway.Request{
			Provider:        ref.Key(),
			Model:           ref.Model,
			Prompt:          prompt,
			MaxOutputTokens: s.cfg.MaxOutputTokens,
			Tools:           endpoint,
			CapacityRoute:   route,
		}
		if agent.MaestroEnabled && ref.Kind == "codex" {
			request.CodexHome = maestroHome
		}
		attempt, err := s.cfg.Gateway.Call(ctx, gateway.Caller{AgentID: agent.ID}, request, stream)
		total = addUsage(total, attempt)
		code = attemptErrorCode(err)
		outcome := "done"
		if err != nil {
			outcome = "failed"
		}
		appendStep(core.StepProviderAttempt, map[string]string{
			"account": ref.Key(), "model": ref.Model, "outcome": outcome, "error_code": code,
		})
		if stepErr != nil {
			slog.Error("record run step", "run", run.ID, "err", stepErr)
			return core.RunFailed, core.ErrCodeStoreError, total
		}
		if err == nil {
			return core.RunDone, "", total
		}
		slog.Warn("provider attempt failed", "run", run.ID, "account", ref.Key(), "error_code", code)
		if route != nil || !canAdvanceChain(ctx, err, emitted, gateway.AttemptMayHaveEffects(err)) {
			break
		}
	}
	return core.RunFailed, code, total
}

// partialOutput returns the text a run streamed before it ended.
func (s *Service) partialOutput(ctx context.Context, runID string) (string, error) {
	steps, err := s.cfg.Repo.ListSteps(ctx, runID)
	if err != nil {
		return "", err
	}
	return ReplyText(steps)
}

// ReplyText joins the text steps of a run in order. It is the reply the model wrote so far.
func ReplyText(steps []core.Step) (string, error) {
	var b strings.Builder
	for _, st := range steps {
		if st.Kind != core.StepText {
			continue
		}
		var text struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(st.Payload), &text); err != nil {
			return "", fmt.Errorf("decode step %s: %w", st.ID, err)
		}
		b.WriteString(text.Text)
	}
	return b.String(), nil
}

// steeredPrompt is the prompt of a continuation: the source message, what the
// interrupted run already said, and the owner's guidance.
func steeredPrompt(body, partial, guidance string) string {
	var b strings.Builder
	b.WriteString(body)
	if partial != "" {
		b.WriteString("\n\nYour reply so far, which the owner interrupted:\n")
		b.WriteString(partial)
	}
	b.WriteString("\n\nGuidance from the owner:\n")
	b.WriteString(guidance)
	return b.String()
}

// canAdvanceChain delegates the safety boundary to IAPro-SDK. The host's
// configured account chain is the explicit authorization; SDK does not
// discover or select accounts outside that chain. A native CLI failure after
// dispatch is never retried automatically without reconciliation.
func canAdvanceChain(ctx context.Context, err error, emitted, effectsPossible bool) bool {
	beforeDispatch := errors.Is(err, harness.ErrNotInstalled) || errors.Is(err, harness.ErrSignedOut)
	return routing.CanAdvanceChain(routing.ChainBoundary{
		Authorized:      true,
		Retryable:       beforeDispatch || errors.Is(err, gateway.ErrUnavailable),
		Canceled:        ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded),
		BudgetOrLedger:  errors.Is(err, gateway.ErrBudgetExceeded) || errors.Is(err, gateway.ErrLedger),
		OutputEmitted:   emitted,
		BeforeDispatch:  beforeDispatch,
		EffectsPossible: effectsPossible,
	})
}

// attemptErrorCode maps one failed attempt to the run's error code; "" means success.
func attemptErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, harness.ErrSignedOut):
		return core.ErrCodeHarnessSignedOut
	case errors.Is(err, gateway.ErrBudgetExceeded):
		return core.ErrCodeBudgetExceeded
	default:
		return core.ErrCodeProviderError
	}
}

func addUsage(a, b gateway.Usage) gateway.Usage {
	return gateway.Usage{
		TokensIn:  a.TokensIn + b.TokensIn,
		TokensOut: a.TokensOut + b.TokensOut,
		Estimated: a.Estimated || b.Estimated,
	}
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
