package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"calango/internal/core"
)

const deviceColumns = `id, owner_id, name, platform, scope, token_digest, created_at, last_seen_at, revoked_at`

// InsertPairingCode stores the digest of a new pairing code and audits it. Codes that
// expired before ev.At are removed in the same transaction.
func (s *Store) InsertPairingCode(ctx context.Context, digest string, pc core.PairingCode, ev core.AuditEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin pairing code insert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM pairing_codes WHERE expires_at < ?`,
		ev.At.UTC().Format(usageTimeLayout)); err != nil {
		return fmt.Errorf("purge expired pairing codes: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO pairing_codes (digest, owner_id, scope, expires_at)
		VALUES (?, ?, ?, ?)`, digest, pc.OwnerID, string(pc.Scope),
		pc.ExpiresAt.UTC().Format(usageTimeLayout)); err != nil {
		return fmt.Errorf("insert pairing code: %w", err)
	}
	if err := insertAudit(ctx, tx, ev); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit pairing code insert: %w", err)
	}
	return nil
}

// PairDevice consumes a pairing code and inserts the device in one transaction. The
// code must be unused and unexpired at `at`. The update is conditional, so two
// concurrent attempts with the same code cannot both succeed. OwnerID and Scope come
// from the code, not from d. It returns the stored device.
func (s *Store) PairDevice(ctx context.Context, digest string, at time.Time, d core.Device, ev core.AuditEvent) (core.Device, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return core.Device{}, fmt.Errorf("begin device pairing: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stamp := at.UTC().Format(usageTimeLayout)
	res, err := tx.ExecContext(ctx, `UPDATE pairing_codes SET used_at = ?
		WHERE digest = ? AND used_at IS NULL AND expires_at > ?`, stamp, digest, stamp)
	if err != nil {
		return core.Device{}, fmt.Errorf("consume pairing code: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return core.Device{}, fmt.Errorf("consume pairing code: %w", err)
	}
	if n == 0 {
		return core.Device{}, fmt.Errorf("%w: pairing code is invalid, used or expired", core.ErrUnauthorized)
	}

	var scope string
	if err := tx.QueryRowContext(ctx, `SELECT owner_id, scope FROM pairing_codes WHERE digest = ?`,
		digest).Scan(&d.OwnerID, &scope); err != nil {
		return core.Device{}, fmt.Errorf("read pairing code: %w", err)
	}
	d.Scope = core.DeviceScope(scope)

	if _, err := tx.ExecContext(ctx, `INSERT INTO devices (`+deviceColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL, NULL)`, d.ID, d.OwnerID, d.Name, d.Platform,
		string(d.Scope), d.TokenDigest, d.CreatedAt.UTC().Format(usageTimeLayout)); err != nil {
		return core.Device{}, fmt.Errorf("insert device %s: %w", d.ID, err)
	}
	if err := insertAudit(ctx, tx, ev); err != nil {
		return core.Device{}, err
	}
	if err := tx.Commit(); err != nil {
		return core.Device{}, fmt.Errorf("commit device pairing: %w", err)
	}
	return d, nil
}

// DeviceByDigest returns the device whose token digest matches. Revoked devices are
// returned too; the caller decides what they may do.
func (s *Store) DeviceByDigest(ctx context.Context, digest string) (core.Device, error) {
	d, err := scanDevice(s.db.QueryRowContext(ctx, `SELECT `+deviceColumns+` FROM devices
		WHERE token_digest = ?`, digest))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Device{}, fmt.Errorf("%w: device", core.ErrNotFound)
	}
	if err != nil {
		return core.Device{}, fmt.Errorf("load device by token: %w", err)
	}
	return d, nil
}

// TouchDevice records that a device authenticated at `at`.
func (s *Store) TouchDevice(ctx context.Context, id string, at time.Time) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE devices SET last_seen_at = ? WHERE id = ?`,
		at.UTC().Format(usageTimeLayout), id); err != nil {
		return fmt.Errorf("touch device %s: %w", id, err)
	}
	return nil
}

// ListDevices returns the owner's devices, oldest first, revoked ones included.
func (s *Store) ListDevices(ctx context.Context, ownerID string) ([]core.Device, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+deviceColumns+` FROM devices
		WHERE owner_id = ? ORDER BY created_at, id`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []core.Device{}
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read devices: %w", err)
	}
	return out, nil
}

// RevokeDevice disables a device of ownerID and audits the change. Revoking a device
// that is already revoked does nothing. An unknown device is core.ErrNotFound.
func (s *Store) RevokeDevice(ctx context.Context, ownerID, id string, at time.Time, ev core.AuditEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin device revoke: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `UPDATE devices SET revoked_at = ?
		WHERE id = ? AND owner_id = ? AND revoked_at IS NULL`,
		at.UTC().Format(usageTimeLayout), id, ownerID)
	if err != nil {
		return fmt.Errorf("revoke device %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke device %s: %w", id, err)
	}
	if n == 0 {
		var found int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices WHERE id = ? AND owner_id = ?`,
			id, ownerID).Scan(&found); err != nil {
			return fmt.Errorf("check device %s: %w", id, err)
		}
		if found == 0 {
			return fmt.Errorf("%w: device %s", core.ErrNotFound, id)
		}
		return nil
	}
	if err := insertAudit(ctx, tx, ev); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit device revoke: %w", err)
	}
	return nil
}

// scanDevice reads one row in deviceColumns order. Scan errors are returned unwrapped
// so callers can match sql.ErrNoRows.
func scanDevice(row rowScanner) (core.Device, error) {
	var d core.Device
	var scope, created string
	var lastSeen, revoked sql.NullString
	if err := row.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Platform, &scope, &d.TokenDigest,
		&created, &lastSeen, &revoked); err != nil {
		return core.Device{}, err
	}
	d.Scope = core.DeviceScope(scope)
	var err error
	if d.CreatedAt, err = time.Parse(usageTimeLayout, created); err != nil {
		return core.Device{}, fmt.Errorf("parse device created_at: %w", err)
	}
	if lastSeen.Valid {
		if d.LastSeenAt, err = time.Parse(usageTimeLayout, lastSeen.String); err != nil {
			return core.Device{}, fmt.Errorf("parse device last_seen_at: %w", err)
		}
	}
	if revoked.Valid {
		if d.RevokedAt, err = time.Parse(usageTimeLayout, revoked.String); err != nil {
			return core.Device{}, fmt.Errorf("parse device revoked_at: %w", err)
		}
	}
	return d, nil
}
