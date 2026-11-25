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

const agentColumns = `id, owner_id, name, role, instructions,
	reasoning_effort, run_on, token_limit_month, approval_level, maestro_enabled, capacity_routing, created_at, updated_at`

// InsertAgent stores a new agent and its provider chain in one transaction. The
// caller validates the agent first.
func (s *Store) InsertAgent(ctx context.Context, a core.Agent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin agent insert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `INSERT INTO agents (`+agentColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.OwnerID, a.Name, a.Role, a.Instructions,
		effortValue(a.ReasoningEffort), string(a.RunOn), a.TokenLimitMonth, string(a.ApprovalLevel), a.MaestroEnabled, a.CapacityRouting,
		a.CreatedAt.UTC().Format(usageTimeLayout), a.UpdatedAt.UTC().Format(usageTimeLayout)); err != nil {
		return fmt.Errorf("insert agent %s: %w", a.ID, err)
	}
	if err := writeProviderChain(ctx, tx, a); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit agent insert %s: %w", a.ID, err)
	}
	return nil
}

// queryer is satisfied by *sql.DB and *sql.Tx.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// writeProviderChain replaces the provider chain of a. Every account must belong
// to a's owner; the database foreign keys alone would not check ownership.
func writeProviderChain(ctx context.Context, tx *sql.Tx, a core.Agent) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_providers WHERE agent_id = ?`, a.ID); err != nil {
		return fmt.Errorf("clear provider chain of agent %s: %w", a.ID, err)
	}
	for i, p := range a.Providers {
		var owner string
		err := tx.QueryRowContext(ctx, `SELECT owner_id FROM provider_accounts WHERE id = ?`, p.AccountID).Scan(&owner)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && owner != a.OwnerID) {
			return fmt.Errorf("%w: providers[%d] account is not one of yours", core.ErrInvalidInput, i)
		}
		if err != nil {
			return fmt.Errorf("read provider account %s: %w", p.AccountID, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO agent_providers (agent_id, position, account_id, model) VALUES (?, ?, ?, ?)`,
			a.ID, i, p.AccountID, p.Model); err != nil {
			return fmt.Errorf("insert provider %d of agent %s: %w", i, a.ID, err)
		}
	}
	return nil
}

// loadProviderChain returns the agent's provider chain in position order.
func loadProviderChain(ctx context.Context, q queryer, agentID string) ([]core.ProviderRef, error) {
	rows, err := q.QueryContext(ctx, `SELECT p.account_id, a.kind, a.label, p.model
		FROM agent_providers p JOIN provider_accounts a ON a.id = p.account_id
		WHERE p.agent_id = ? ORDER BY p.position`, agentID)
	if err != nil {
		return nil, fmt.Errorf("read provider chain of agent %s: %w", agentID, err)
	}
	defer rows.Close()

	var chain []core.ProviderRef
	for rows.Next() {
		var ref core.ProviderRef
		if err := rows.Scan(&ref.AccountID, &ref.Kind, &ref.Label, &ref.Model); err != nil {
			return nil, fmt.Errorf("scan provider of agent %s: %w", agentID, err)
		}
		chain = append(chain, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read provider chain of agent %s: %w", agentID, err)
	}
	return chain, nil
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
	if a.Providers, err = loadProviderChain(ctx, s.db, id); err != nil {
		return core.Agent{}, err
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
	// The rows are closed here; loading each chain needs a second query, and the
	// pool may hold a single connection.
	for i := range agents {
		if agents[i].Providers, err = loadProviderChain(ctx, s.db, agents[i].ID); err != nil {
			return nil, err
		}
	}
	return agents, nil
}

// ReplaceAgent overwrites the editable fields of an existing agent and its
// provider chain in one transaction. created_at is kept.
func (s *Store) ReplaceAgent(ctx context.Context, a core.Agent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin agent replace: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `UPDATE agents SET
		name = ?, role = ?, instructions = ?, reasoning_effort = ?,
		run_on = ?, token_limit_month = ?, approval_level = ?, maestro_enabled = ?, capacity_routing = ?, updated_at = ?
		WHERE id = ? AND owner_id = ?`,
		a.Name, a.Role, a.Instructions, effortValue(a.ReasoningEffort),
		string(a.RunOn), a.TokenLimitMonth, string(a.ApprovalLevel), a.MaestroEnabled, a.CapacityRouting, a.UpdatedAt.UTC().Format(usageTimeLayout),
		a.ID, a.OwnerID)
	if err != nil {
		return fmt.Errorf("replace agent %s: %w", a.ID, err)
	}
	if err := requireOneRow(res, "agent "+a.ID); err != nil {
		return err
	}
	if err := writeProviderChain(ctx, tx, a); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit agent replace %s: %w", a.ID, err)
	}
	return nil
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
	if err := row.Scan(&a.ID, &a.OwnerID, &a.Name, &a.Role, &a.Instructions,
		&effort, &runOn, &a.TokenLimitMonth, &approval, &a.MaestroEnabled, &a.CapacityRouting, &createdAt, &updatedAt); err != nil {
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
