package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	sqlite "modernc.org/sqlite"

	"calango/internal/core"
)

const agentColumns = `id, owner_id, name, role, instructions, harness, model,
	reasoning_effort, run_on, token_limit_month, approval_level, created_at, updated_at`

// InsertAgent stores a new agent. The caller validates it first.
func (s *Store) InsertAgent(ctx context.Context, a core.Agent) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO agents (`+agentColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.OwnerID, a.Name, a.Role, a.Instructions, a.Harness, a.Model,
		effortValue(a.ReasoningEffort), string(a.RunOn), a.TokenLimitMonth, string(a.ApprovalLevel),
		a.CreatedAt.UTC().Format(usageTimeLayout), a.UpdatedAt.UTC().Format(usageTimeLayout))
	if err != nil {
		return fmt.Errorf("insert agent %s: %w", a.ID, err)
	}
	return nil
}

// GetAgent returns the agent only when it belongs to ownerID.
func (s *Store) GetAgent(ctx context.Context, ownerID, id string) (core.Agent, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+agentColumns+` FROM agents WHERE id = ? AND owner_id = ?`, id, ownerID)
	a, err := scanAgent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Agent{}, fmt.Errorf("%w: agent %s", core.ErrNotFound, id)
	}
	if err != nil {
		return core.Agent{}, fmt.Errorf("read agent %s: %w", id, err)
	}
	return a, nil
}

// ListAgents returns the owner's agents, oldest first.
func (s *Store) ListAgents(ctx context.Context, ownerID string) ([]core.Agent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+agentColumns+` FROM agents WHERE owner_id = ? ORDER BY created_at, id`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []core.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	return agents, nil
}

// ReplaceAgent overwrites the editable fields of an existing agent. created_at is kept.
func (s *Store) ReplaceAgent(ctx context.Context, a core.Agent) error {
	res, err := s.db.ExecContext(ctx, `UPDATE agents SET
		name = ?, role = ?, instructions = ?, harness = ?, model = ?, reasoning_effort = ?,
		run_on = ?, token_limit_month = ?, approval_level = ?, updated_at = ?
		WHERE id = ? AND owner_id = ?`,
		a.Name, a.Role, a.Instructions, a.Harness, a.Model, effortValue(a.ReasoningEffort),
		string(a.RunOn), a.TokenLimitMonth, string(a.ApprovalLevel), a.UpdatedAt.UTC().Format(usageTimeLayout),
		a.ID, a.OwnerID)
	if err != nil {
		return fmt.Errorf("replace agent %s: %w", a.ID, err)
	}
	return requireOneRow(res, "agent "+a.ID)
}

// DeleteAgent removes an agent. It returns core.ErrConflict when the agent has runs or
// usage, because history must stay attributable.
func (s *Store) DeleteAgent(ctx context.Context, ownerID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM agents WHERE id = ? AND owner_id = ?`, id, ownerID)
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && strings.Contains(sqliteErr.Error(), "FOREIGN KEY constraint failed") {
		return fmt.Errorf("%w: agent %s has runs or usage history", core.ErrConflict, id)
	}
	if err != nil {
		return fmt.Errorf("delete agent %s: %w", id, err)
	}
	return requireOneRow(res, "agent "+id)
}

// GlobalPaused reports the node-wide pause switch.
func (s *Store) GlobalPaused(ctx context.Context) (bool, error) {
	var paused int
	if err := s.db.QueryRowContext(ctx, `SELECT global_paused FROM node_state WHERE id = 1`).Scan(&paused); err != nil {
		return false, fmt.Errorf("read global pause: %w", err)
	}
	return paused == 1, nil
}

// SetGlobalPaused turns the node-wide pause switch on or off.
func (s *Store) SetGlobalPaused(ctx context.Context, paused bool, at time.Time) error {
	value := 0
	if paused {
		value = 1
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE node_state SET global_paused = ?, updated_at = ? WHERE id = 1`,
		value, at.UTC().Format(usageTimeLayout)); err != nil {
		return fmt.Errorf("set global pause: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAgent(row rowScanner) (core.Agent, error) {
	var (
		a                    core.Agent
		effort               sql.NullString
		runOn, approval      string
		createdAt, updatedAt string
	)
	if err := row.Scan(&a.ID, &a.OwnerID, &a.Name, &a.Role, &a.Instructions, &a.Harness, &a.Model,
		&effort, &runOn, &a.TokenLimitMonth, &approval, &createdAt, &updatedAt); err != nil {
		return core.Agent{}, err
	}
	a.ReasoningEffort = core.ReasoningEffort(effort.String)
	a.RunOn = core.RunOn(runOn)
	a.ApprovalLevel = core.ApprovalLevel(approval)
	var err error
	if a.CreatedAt, err = time.Parse(usageTimeLayout, createdAt); err != nil {
		return core.Agent{}, fmt.Errorf("parse created_at: %w", err)
	}
	if a.UpdatedAt, err = time.Parse(usageTimeLayout, updatedAt); err != nil {
		return core.Agent{}, fmt.Errorf("parse updated_at: %w", err)
	}
	return a, nil
}

// effortValue maps the empty reasoning effort to SQL NULL (provider default).
func effortValue(effort core.ReasoningEffort) sql.NullString {
	return sql.NullString{String: string(effort), Valid: effort != ""}
}

func requireOneRow(res sql.Result, what string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("count affected rows for %s: %w", what, err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", core.ErrNotFound, what)
	}
	return nil
}
