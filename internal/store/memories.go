package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"calango/internal/core"
)

const memoryColumns = `id, owner_id, scope, agent_id, kind, text, created_at, updated_at, deleted_at`

// InsertMemory stores a new memory. The caller validates it first.
func (s *Store) InsertMemory(ctx context.Context, m core.Memory) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO memories (`+memoryColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		m.ID, m.OwnerID, string(m.Scope), nullIfEmpty(m.AgentID), string(m.Kind), m.Text,
		m.CreatedAt.UTC().Format(usageTimeLayout), m.UpdatedAt.UTC().Format(usageTimeLayout))
	if err != nil {
		return fmt.Errorf("insert memory %s: %w", m.ID, err)
	}
	return nil
}

// GetMemory returns an active memory only when it belongs to ownerID.
func (s *Store) GetMemory(ctx context.Context, ownerID, id string) (core.Memory, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+memoryColumns+` FROM memories WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`,
		id, ownerID)
	m, err := scanMemory(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Memory{}, fmt.Errorf("memory %s: %w", id, core.ErrNotFound)
	}
	if err != nil {
		return core.Memory{}, fmt.Errorf("get memory %s: %w", id, err)
	}
	return m, nil
}

// ListMemories returns the owner's active memories, most recently updated first.
func (s *Store) ListMemories(ctx context.Context, ownerID string) ([]core.Memory, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+memoryColumns+` FROM memories
		WHERE owner_id = ? AND deleted_at IS NULL ORDER BY updated_at DESC, id DESC`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()

	var out []core.Memory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	return out, nil
}

// UpdateMemoryText replaces the text of an active memory owned by ownerID.
func (s *Store) UpdateMemoryText(ctx context.Context, ownerID, id, text string, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE memories SET text = ?, updated_at = ?
		WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`,
		text, at.UTC().Format(usageTimeLayout), id, ownerID)
	if err != nil {
		return fmt.Errorf("update memory %s: %w", id, err)
	}
	return requireOneRow(res, "memory "+id)
}

// DeleteMemory marks an active memory as deleted. It stays in the table until PurgeMemories.
func (s *Store) DeleteMemory(ctx context.Context, ownerID, id string, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE memories SET deleted_at = ?
		WHERE id = ? AND owner_id = ? AND deleted_at IS NULL`,
		at.UTC().Format(usageTimeLayout), id, ownerID)
	if err != nil {
		return fmt.Errorf("delete memory %s: %w", id, err)
	}
	return requireOneRow(res, "memory "+id)
}

// PurgeMemories physically removes memories deleted before the cutoff and returns how many.
func (s *Store) PurgeMemories(ctx context.Context, deletedBefore time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM memories WHERE deleted_at IS NOT NULL AND deleted_at < ?`,
		deletedBefore.UTC().Format(usageTimeLayout))
	if err != nil {
		return 0, fmt.Errorf("purge memories: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge memories: %w", err)
	}
	return n, nil
}

func scanMemory(row rowScanner) (core.Memory, error) {
	var (
		m                  core.Memory
		scope, kind        string
		agentID, deletedAt sql.NullString
		createdAt          string
		updatedAt          string
	)
	if err := row.Scan(&m.ID, &m.OwnerID, &scope, &agentID, &kind, &m.Text, &createdAt, &updatedAt, &deletedAt); err != nil {
		return core.Memory{}, err
	}
	m.Scope = core.MemoryScope(scope)
	m.Kind = core.MemoryKind(kind)
	m.AgentID = agentID.String

	var err error
	if m.CreatedAt, err = time.Parse(usageTimeLayout, createdAt); err != nil {
		return core.Memory{}, fmt.Errorf("parse created_at: %w", err)
	}
	if m.UpdatedAt, err = time.Parse(usageTimeLayout, updatedAt); err != nil {
		return core.Memory{}, fmt.Errorf("parse updated_at: %w", err)
	}
	if m.DeletedAt, err = optionalTime(deletedAt); err != nil {
		return core.Memory{}, fmt.Errorf("parse deleted_at: %w", err)
	}
	return m, nil
}
