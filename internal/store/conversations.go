package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"calango/internal/core"
)

const runColumns = `id, agent_id, conversation_id, source_message_id, idempotency_key, trigger,
	status, instructions, retry_of, steer_of, tokens_in, tokens_out, started_at, ended_at, error_code`

// InsertConversation stores a conversation and its agent member in one transaction.
func (s *Store) InsertConversation(ctx context.Context, c core.Conversation, agentID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin conversation insert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	at := c.CreatedAt.UTC().Format(usageTimeLayout)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO conversations (id, kind, title, created_at) VALUES (?, ?, ?, ?)`,
		c.ID, c.Kind, c.Title, at); err != nil {
		return fmt.Errorf("insert conversation %s: %w", c.ID, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO conversation_members (conversation_id, member_kind, member_id, joined_at)
		VALUES (?, 'agent', ?, ?)`,
		c.ID, agentID, at); err != nil {
		return fmt.Errorf("insert conversation member: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit conversation insert: %w", err)
	}
	return nil
}

// ListConversations returns the conversations whose agent partner belongs to ownerID,
// newest first.
func (s *Store) ListConversations(ctx context.Context, ownerID string) ([]core.Conversation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id, c.kind, c.title, c.created_at
		FROM conversations c
		JOIN conversation_members m ON m.conversation_id = c.id AND m.member_kind = 'agent'
		JOIN agents a ON a.id = m.member_id
		WHERE a.owner_id = ?
		ORDER BY c.created_at DESC, c.id DESC`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []core.Conversation
	for rows.Next() {
		var c core.Conversation
		var at string
		if err := rows.Scan(&c.ID, &c.Kind, &c.Title, &at); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		if c.CreatedAt, err = parseStoredTime(at); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}
	return out, nil
}

// ConversationAgent returns the agent partner of a DM.
func (s *Store) ConversationAgent(ctx context.Context, conversationID string) (string, error) {
	var agentID string
	err := s.db.QueryRowContext(ctx,
		`SELECT member_id FROM conversation_members WHERE conversation_id = ? AND member_kind = 'agent'`,
		conversationID).Scan(&agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("%w: conversation %s", core.ErrNotFound, conversationID)
	}
	if err != nil {
		return "", fmt.Errorf("read conversation agent: %w", err)
	}
	return agentID, nil
}

// AgentByID returns an agent without an owner check. Only internal code that
// has already authorized the request may call it.
func (s *Store) AgentByID(ctx context.Context, id string) (core.Agent, error) {
	a, err := scanAgent(s.db.QueryRowContext(ctx,
		`SELECT `+agentColumns+` FROM agents WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Agent{}, fmt.Errorf("%w: agent %s", core.ErrNotFound, id)
	}
	if err != nil {
		return core.Agent{}, fmt.Errorf("read agent %s: %w", id, err)
	}
	if a.Providers, err = loadProviderChain(ctx, s.db, id); err != nil {
		return core.Agent{}, err
	}
	return a, nil
}

// AdmitRun stores an optional message and a run in one transaction. The run's
// idempotency key is unique, so a duplicate request fails and stores nothing.
func (s *Store) AdmitRun(ctx context.Context, msg *core.Message, run core.Run) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin run admission: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if msg != nil {
		if _, err := tx.ExecContext(ctx, `INSERT INTO messages
			(id, conversation_id, thread_id, author_kind, author_id, body, created_at)
			VALUES (?, ?, NULL, ?, ?, ?, ?)`,
			msg.ID, msg.ConversationID, msg.AuthorKind, msg.AuthorID, msg.Body,
			msg.CreatedAt.UTC().Format(usageTimeLayout)); err != nil {
			return fmt.Errorf("insert message %s: %w", msg.ID, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO runs (`+runColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.AgentID, run.ConversationID, nullIfEmpty(run.SourceMessageID),
		run.IdempotencyKey, run.Trigger, string(run.Status), run.Instructions,
		nullIfEmpty(run.RetryOf), nullIfEmpty(run.SteerOf), run.TokensIn, run.TokensOut,
		timeValue(run.StartedAt), timeValue(run.EndedAt), nullIfEmpty(run.ErrorCode)); err != nil {
		return fmt.Errorf("insert run %s: %w", run.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit run admission: %w", err)
	}
	return nil
}

// GetMessage returns one message by id.
func (s *Store) GetMessage(ctx context.Context, id string) (core.Message, error) {
	var m core.Message
	var at string
	err := s.db.QueryRowContext(ctx, `SELECT id, conversation_id, author_kind, author_id, body, created_at
		FROM messages WHERE id = ?`, id).
		Scan(&m.ID, &m.ConversationID, &m.AuthorKind, &m.AuthorID, &m.Body, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Message{}, fmt.Errorf("%w: message %s", core.ErrNotFound, id)
	}
	if err != nil {
		return core.Message{}, fmt.Errorf("read message %s: %w", id, err)
	}
	m.CreatedAt, err = parseStoredTime(at)
	if err != nil {
		return core.Message{}, err
	}
	return m, nil
}

// ListMessages returns a conversation's messages in creation order.
func (s *Store) ListMessages(ctx context.Context, conversationID string) ([]core.Message, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, conversation_id, author_kind, author_id, body, created_at
		FROM messages WHERE conversation_id = ? ORDER BY created_at, id`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []core.Message
	for rows.Next() {
		var m core.Message
		var at string
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.AuthorKind, &m.AuthorID, &m.Body, &at); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		if m.CreatedAt, err = parseStoredTime(at); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}
	return out, nil
}

// GetRun returns one run by id.
func (s *Store) GetRun(ctx context.Context, id string) (core.Run, error) {
	r, err := scanRun(s.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM runs WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Run{}, fmt.Errorf("%w: run %s", core.ErrNotFound, id)
	}
	if err != nil {
		return core.Run{}, fmt.Errorf("read run %s: %w", id, err)
	}
	return r, nil
}

// GetRunByIdempotencyKey returns the run created for a client key.
func (s *Store) GetRunByIdempotencyKey(ctx context.Context, key string) (core.Run, error) {
	r, err := scanRun(s.db.QueryRowContext(ctx,
		`SELECT `+runColumns+` FROM runs WHERE idempotency_key = ?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Run{}, fmt.Errorf("%w: idempotency key", core.ErrNotFound)
	}
	if err != nil {
		return core.Run{}, fmt.Errorf("read run by idempotency key: %w", err)
	}
	return r, nil
}

// ClaimRun moves a queued run to running only when its agent has no active run.
// The conditional update prevents overlapping claims across service instances.
func (s *Store) ClaimRun(ctx context.Context, id string, at time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE runs SET status = 'running', started_at = ?
		WHERE id = ? AND status = 'queued'
		AND NOT EXISTS (SELECT 1 FROM runs active
			WHERE active.agent_id = runs.agent_id AND active.status IN ('running', 'waiting_approval'))`,
		at.UTC().Format(usageTimeLayout), id)
	if err != nil {
		return false, fmt.Errorf("claim run %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count claimed runs: %w", err)
	}
	return n == 1, nil
}

// SetRunStatus moves a run between running and waiting_approval and reports
// whether it did. A run waiting on a person keeps its conversation slot. A run
// that is no longer in the expected state reports false.
func (s *Store) SetRunStatus(ctx context.Context, id string, from, to core.RunStatus) (bool, error) {
	if !(from == core.RunRunning && to == core.RunWaitingApproval || from == core.RunWaitingApproval && to == core.RunRunning) {
		return false, fmt.Errorf("run %s: status change %s to %s is not allowed", id, from, to)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status = ? WHERE id = ? AND status = ?`,
		string(to), id, string(from))
	if err != nil {
		return false, fmt.Errorf("set run %s status to %s: %w", id, to, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count status changes for run %s: %w", id, err)
	}
	return n == 1, nil
}

// FinishRun records the terminal status, end time, error code and tokens of a run.
func (s *Store) FinishRun(ctx context.Context, id string, status core.RunStatus, at time.Time,
	errCode string, tokensIn, tokensOut int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE runs
		SET status = ?, ended_at = ?, error_code = ?, tokens_in = ?, tokens_out = ?
		WHERE id = ?`,
		string(status), at.UTC().Format(usageTimeLayout), nullIfEmpty(errCode), tokensIn, tokensOut, id)
	if err != nil {
		return fmt.Errorf("finish run %s: %w", id, err)
	}
	return requireOneRow(res, "run "+id)
}

// FinishRunWithMessage records the run result and its optional reply atomically.
func (s *Store) FinishRunWithMessage(ctx context.Context, id string, status core.RunStatus, at time.Time,
	errCode string, tokensIn, tokensOut int64, message *core.Message) error {
	if message == nil {
		return s.FinishRun(ctx, id, status, at, errCode, tokensIn, tokensOut)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin run completion: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `UPDATE runs
		SET status = ?, ended_at = ?, error_code = ?, tokens_in = ?, tokens_out = ?
		WHERE id = ?`,
		string(status), at.UTC().Format(usageTimeLayout), nullIfEmpty(errCode), tokensIn, tokensOut, id)
	if err != nil {
		return fmt.Errorf("finish run %s: %w", id, err)
	}
	if err := requireOneRow(res, "run "+id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO messages
		(id, conversation_id, thread_id, author_kind, author_id, body, created_at)
		VALUES (?, ?, NULL, ?, ?, ?, ?)`,
		message.ID, message.ConversationID, message.AuthorKind, message.AuthorID, message.Body,
		message.CreatedAt.UTC().Format(usageTimeLayout)); err != nil {
		return fmt.Errorf("insert reply message %s: %w", message.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit run completion %s: %w", id, err)
	}
	return nil
}

// CountRunningRuns counts runs that hold a conversation slot: running or waiting_approval.
func (s *Store) CountRunningRuns(ctx context.Context, conversationID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM runs WHERE conversation_id = ? AND status IN ('running', 'waiting_approval')`,
		conversationID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count running runs: %w", err)
	}
	return n, nil
}

// CountActiveAgentRuns counts runs holding an agent slot across conversations:
// running or waiting_approval.
func (s *Store) CountActiveAgentRuns(ctx context.Context, agentID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM runs WHERE agent_id = ? AND status IN ('running', 'waiting_approval')`,
		agentID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count active agent runs: %w", err)
	}
	return n, nil
}

// OldestQueuedAgentRun returns the earliest queued run admitted for an agent.
// Rowid preserves insertion order even when ids share a timestamp.
func (s *Store) OldestQueuedAgentRun(ctx context.Context, agentID string) (core.Run, error) {
	r, err := scanRun(s.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM runs
		WHERE agent_id = ? AND status = 'queued' ORDER BY rowid LIMIT 1`, agentID))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Run{}, fmt.Errorf("%w: no queued run for agent", core.ErrNotFound)
	}
	if err != nil {
		return core.Run{}, fmt.Errorf("read oldest queued agent run: %w", err)
	}
	return r, nil
}

// OldestQueuedRun returns the earliest queued run admitted to a conversation.
// Rowid preserves insertion order even when ids share a timestamp.
func (s *Store) OldestQueuedRun(ctx context.Context, conversationID string) (core.Run, error) {
	r, err := scanRun(s.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM runs
		WHERE conversation_id = ? AND status = 'queued' ORDER BY rowid LIMIT 1`, conversationID))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Run{}, fmt.Errorf("%w: no queued run", core.ErrNotFound)
	}
	if err != nil {
		return core.Run{}, fmt.Errorf("read oldest queued run: %w", err)
	}
	return r, nil
}

// ConversationsWithQueuedRuns lists conversations that have queued runs.
func (s *Store) ConversationsWithQueuedRuns(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT conversation_id FROM runs WHERE status = 'queued'`)
	if err != nil {
		return nil, fmt.Errorf("list conversations with queued runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan conversation id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}
	return out, nil
}

// InterruptRunningRuns ends every run left in running or waiting_approval state as
// failed with error_code interrupted. It runs once at startup, before any dispatch.
func (s *Store) InterruptRunningRuns(ctx context.Context, at time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE runs
		SET status = 'failed', ended_at = ?, error_code = ?
		WHERE status IN ('running', 'waiting_approval')`,
		at.UTC().Format(usageTimeLayout), core.ErrCodeInterrupted)
	if err != nil {
		return 0, fmt.Errorf("interrupt running runs: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count interrupted runs: %w", err)
	}
	return n, nil
}

// AppendStep stores one step. The caller assigns Seq; a run has one writer.
func (s *Store) AppendStep(ctx context.Context, st core.Step) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO run_steps (id, run_id, seq, kind, payload_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		st.ID, st.RunID, st.Seq, st.Kind, st.Payload, st.CreatedAt.UTC().Format(usageTimeLayout))
	if err != nil {
		return fmt.Errorf("append step %d of run %s: %w", st.Seq, st.RunID, err)
	}
	return nil
}

// ListSteps returns a run's steps in sequence order.
func (s *Store) ListSteps(ctx context.Context, runID string) ([]core.Step, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, run_id, seq, kind, payload_json, created_at
		FROM run_steps WHERE run_id = ? ORDER BY seq`, runID)
	if err != nil {
		return nil, fmt.Errorf("list steps of run %s: %w", runID, err)
	}
	defer func() { _ = rows.Close() }()

	var out []core.Step
	for rows.Next() {
		var st core.Step
		var at string
		if err := rows.Scan(&st.ID, &st.RunID, &st.Seq, &st.Kind, &st.Payload, &at); err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}
		if st.CreatedAt, err = parseStoredTime(at); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate steps: %w", err)
	}
	return out, nil
}

func scanRun(row rowScanner) (core.Run, error) {
	var (
		r                                              core.Run
		sourceID, retryOf, steerOf, started, ended, ec sql.NullString
		status                                         string
	)
	err := row.Scan(&r.ID, &r.AgentID, &r.ConversationID, &sourceID, &r.IdempotencyKey, &r.Trigger,
		&status, &r.Instructions, &retryOf, &steerOf, &r.TokensIn, &r.TokensOut, &started, &ended, &ec)
	if err != nil {
		return core.Run{}, err
	}
	r.Status = core.RunStatus(status)
	r.SourceMessageID = sourceID.String
	r.RetryOf = retryOf.String
	r.SteerOf = steerOf.String
	r.ErrorCode = ec.String
	if r.StartedAt, err = optionalTime(started); err != nil {
		return core.Run{}, err
	}
	if r.EndedAt, err = optionalTime(ended); err != nil {
		return core.Run{}, err
	}
	return r, nil
}

func timeValue(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(usageTimeLayout)
}

func optionalTime(v sql.NullString) (*time.Time, error) {
	if !v.Valid {
		return nil, nil
	}
	t, err := parseStoredTime(v.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func parseStoredTime(s string) (time.Time, error) {
	t, err := time.Parse(usageTimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse stored time %q: %w", s, err)
	}
	return t, nil
}
