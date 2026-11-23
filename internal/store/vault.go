package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"calango/internal/core"
	"calango/internal/vault"
)

// Store implements vault.Repo over the vault_items table.
var _ vault.Repo = (*Store)(nil)

// PutItem inserts the item or replaces its envelope. created_at survives a replace.
func (s *Store) PutItem(ctx context.Context, owner string, item vault.Stored, at time.Time) (vault.Item, error) {
	stamp := at.UTC().Format(usageTimeLayout)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO vault_items
		(owner_id, name, ciphertext, key_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (owner_id, name) DO UPDATE SET
			ciphertext = excluded.ciphertext,
			key_version = excluded.key_version,
			updated_at = excluded.updated_at`,
		owner, item.Name, item.Ciphertext, item.KeyVersion, stamp, stamp); err != nil {
		return vault.Item{}, fmt.Errorf("upsert vault item: %w", err)
	}
	stored, err := s.GetItem(ctx, owner, item.Name)
	if err != nil {
		return vault.Item{}, err
	}
	return stored.Item, nil
}

// GetItem returns the item with its envelope, or core.ErrNotFound.
func (s *Store) GetItem(ctx context.Context, owner, name string) (vault.Stored, error) {
	row := s.db.QueryRowContext(ctx, `SELECT name, ciphertext, key_version, created_at, updated_at
		FROM vault_items WHERE owner_id = ? AND name = ?`, owner, name)
	stored, err := scanVaultItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return vault.Stored{}, fmt.Errorf("%w: secret %s", core.ErrNotFound, name)
	}
	return stored, err
}

// ListItems returns metadata for every item of the owner, ordered by name.
func (s *Store) ListItems(ctx context.Context, owner string) ([]vault.Item, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, created_at, updated_at
		FROM vault_items WHERE owner_id = ? ORDER BY name`, owner)
	if err != nil {
		return nil, fmt.Errorf("list vault items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []vault.Item
	for rows.Next() {
		var name, createdAt, updatedAt string
		if err := rows.Scan(&name, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan vault item: %w", err)
		}
		item, err := vaultItemMeta(name, createdAt, updatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vault items: %w", err)
	}
	return items, nil
}

// ListStored returns every item of the owner with its envelope, ordered by name.
func (s *Store) ListStored(ctx context.Context, owner string) ([]vault.Stored, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, ciphertext, key_version, created_at, updated_at
		FROM vault_items WHERE owner_id = ? ORDER BY name`, owner)
	if err != nil {
		return nil, fmt.Errorf("list stored vault items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stored []vault.Stored
	for rows.Next() {
		item, err := scanVaultItem(rows)
		if err != nil {
			return nil, err
		}
		stored = append(stored, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stored vault items: %w", err)
	}
	return stored, nil
}

// ReplaceCiphertext updates the envelope of an existing item and keeps its dates.
func (s *Store) ReplaceCiphertext(ctx context.Context, owner, name, ciphertext string, keyVersion int) error {
	res, err := s.db.ExecContext(ctx, `UPDATE vault_items SET ciphertext = ?, key_version = ?
		WHERE owner_id = ? AND name = ?`, ciphertext, keyVersion, owner, name)
	if err != nil {
		return fmt.Errorf("replace vault ciphertext: %w", err)
	}
	return requireOneRow(res, name)
}

// DeleteItem removes the item, or returns core.ErrNotFound.
func (s *Store) DeleteItem(ctx context.Context, owner, name string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM vault_items WHERE owner_id = ? AND name = ?`, owner, name)
	if err != nil {
		return fmt.Errorf("delete vault item: %w", err)
	}
	return requireOneRow(res, name)
}

func scanVaultItem(row rowScanner) (vault.Stored, error) {
	var name, ciphertext, createdAt, updatedAt string
	var keyVersion int
	if err := row.Scan(&name, &ciphertext, &keyVersion, &createdAt, &updatedAt); err != nil {
		return vault.Stored{}, err
	}
	meta, err := vaultItemMeta(name, createdAt, updatedAt)
	if err != nil {
		return vault.Stored{}, err
	}
	return vault.Stored{Item: meta, Ciphertext: ciphertext, KeyVersion: keyVersion}, nil
}

func vaultItemMeta(name, createdAt, updatedAt string) (vault.Item, error) {
	created, err := time.Parse(usageTimeLayout, createdAt)
	if err != nil {
		return vault.Item{}, fmt.Errorf("parse vault created_at: %w", err)
	}
	updated, err := time.Parse(usageTimeLayout, updatedAt)
	if err != nil {
		return vault.Item{}, fmt.Errorf("parse vault updated_at: %w", err)
	}
	return vault.Item{Name: name, CreatedAt: created, UpdatedAt: updated}, nil
}
