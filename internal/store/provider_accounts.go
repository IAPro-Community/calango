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

const providerAccountColumns = `id, owner_id, kind, label, created_at, updated_at`

// InsertProviderAccount stores a new account. A duplicate kind and label for the
// same owner returns core.ErrConflict.
func (s *Store) InsertProviderAccount(ctx context.Context, a core.ProviderAccount) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO provider_accounts (`+providerAccountColumns+`)
		VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.OwnerID, a.Kind, a.Label,
		a.CreatedAt.UTC().Format(usageTimeLayout), a.UpdatedAt.UTC().Format(usageTimeLayout))
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed") {
		return fmt.Errorf("%w: provider account %s/%s already exists", core.ErrConflict, a.Kind, a.Label)
	}
	if err != nil {
		return fmt.Errorf("insert provider account %s: %w", a.ID, err)
	}
	return nil
}

// ListProviderAccounts returns the owner's accounts, oldest first.
func (s *Store) ListProviderAccounts(ctx context.Context, ownerID string) ([]core.ProviderAccount, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+providerAccountColumns+` FROM provider_accounts WHERE owner_id = ? ORDER BY created_at, id`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list provider accounts: %w", err)
	}
	defer rows.Close()

	var accounts []core.ProviderAccount
	for rows.Next() {
		a, err := scanProviderAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan provider account: %w", err)
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list provider accounts: %w", err)
	}
	return accounts, nil
}

// GetProviderAccountByName returns the owner's account with this kind and label.
func (s *Store) GetProviderAccountByName(ctx context.Context, ownerID, kind, label string) (core.ProviderAccount, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+providerAccountColumns+` FROM provider_accounts WHERE owner_id = ? AND kind = ? AND label = ?`,
		ownerID, kind, label)
	a, err := scanProviderAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.ProviderAccount{}, fmt.Errorf("%w: provider account %s/%s", core.ErrNotFound, kind, label)
	}
	if err != nil {
		return core.ProviderAccount{}, fmt.Errorf("read provider account %s/%s: %w", kind, label, err)
	}
	return a, nil
}

// DeleteProviderAccount removes the owner's account. It returns core.ErrConflict
// while an agent's provider chain still uses it.
func (s *Store) DeleteProviderAccount(ctx context.Context, ownerID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM provider_accounts WHERE id = ? AND owner_id = ?`, id, ownerID)
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && strings.Contains(sqliteErr.Error(), "FOREIGN KEY constraint failed") {
		return fmt.Errorf("%w: provider account %s is used by an agent", core.ErrConflict, id)
	}
	if err != nil {
		return fmt.Errorf("delete provider account %s: %w", id, err)
	}
	return requireOneRow(res, "provider account "+id)
}

func scanProviderAccount(row rowScanner) (core.ProviderAccount, error) {
	var (
		a                    core.ProviderAccount
		createdAt, updatedAt string
	)
	if err := row.Scan(&a.ID, &a.OwnerID, &a.Kind, &a.Label, &createdAt, &updatedAt); err != nil {
		return core.ProviderAccount{}, err
	}
	var err error
	if a.CreatedAt, err = time.Parse(usageTimeLayout, createdAt); err != nil {
		return core.ProviderAccount{}, fmt.Errorf("parse created_at: %w", err)
	}
	if a.UpdatedAt, err = time.Parse(usageTimeLayout, updatedAt); err != nil {
		return core.ProviderAccount{}, fmt.Errorf("parse updated_at: %w", err)
	}
	return a, nil
}
