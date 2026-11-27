package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/IAPro-Community/IAPro-SDK/lease"
)

// AccountLeaseStore adapts the node database to the SDK coordinator. It does
// not authorize a dispatch or prove that an expired process has stopped.
type AccountLeaseStore struct {
	store *Store
}

var _ lease.Store = AccountLeaseStore{}

func (s *Store) AccountLeases() AccountLeaseStore { return AccountLeaseStore{store: s} }

func validLeaseKey(key lease.Key) bool {
	return key.OwnerID != "" && key.Provider != "" && key.AccountID != ""
}

func (s AccountLeaseStore) Load(ctx context.Context, key lease.Key) (lease.Record, error) {
	if !validLeaseKey(key) {
		return lease.Record{}, lease.ErrStale
	}
	var record lease.Record
	var expiry string
	err := s.store.db.QueryRowContext(ctx, `SELECT holder, fence, revision, expires_at
		FROM account_leases WHERE owner_id = ? AND provider = ? AND account_id = ?`,
		key.OwnerID, key.Provider, key.AccountID).Scan(&record.Holder, &record.Fence, &record.Revision, &expiry)
	if errors.Is(err, sql.ErrNoRows) {
		return lease.Record{}, nil
	}
	if err != nil {
		return lease.Record{}, fmt.Errorf("load account lease: %w", err)
	}
	if expiry != "" {
		if record.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiry); err != nil {
			return lease.Record{}, errors.New("invalid stored lease expiry")
		}
	}
	if record.Fence == 0 || record.Revision < record.Fence || (record.Holder == "") != record.ExpiresAt.IsZero() {
		return lease.Record{}, lease.ErrStale
	}
	return record, nil
}

// CompareAndSwap uses one conditional SQL write, including for initial inserts.
// FULL synchronization protects committed fencing generations across crashes;
// the connection may retain this stronger durability setting when reused.
func (s AccountLeaseStore) CompareAndSwap(ctx context.Context, key lease.Key, revision uint64, next lease.Record) (bool, error) {
	if !validLeaseKey(key) || revision >= math.MaxInt64 || next.Revision != revision+1 ||
		next.Fence == 0 || next.Fence > next.Revision || (next.Holder == "") != next.ExpiresAt.IsZero() {
		return false, lease.ErrStale
	}
	expiry := ""
	if !next.ExpiresAt.IsZero() {
		encoded, err := next.ExpiresAt.UTC().MarshalText()
		if err != nil {
			return false, lease.ErrStale
		}
		expiry = string(encoded)
	}
	conn, err := s.store.db.Conn(ctx)
	if err != nil {
		return false, fmt.Errorf("open account lease connection: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `PRAGMA synchronous=FULL`); err != nil {
		return false, fmt.Errorf("configure account lease durability: %w", err)
	}
	var result sql.Result
	if revision == 0 {
		result, err = conn.ExecContext(ctx, `INSERT INTO account_leases
			(owner_id, provider, account_id, holder, fence, revision, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (owner_id, provider, account_id) DO NOTHING`,
			key.OwnerID, key.Provider, key.AccountID, next.Holder, next.Fence, next.Revision, expiry)
	} else {
		result, err = conn.ExecContext(ctx, `UPDATE account_leases SET holder = ?, fence = ?, revision = ?, expires_at = ?
			WHERE owner_id = ? AND provider = ? AND account_id = ? AND revision = ? AND fence <= ?`,
			next.Holder, next.Fence, next.Revision, expiry, key.OwnerID, key.Provider, key.AccountID, revision, next.Fence)
	}
	if err != nil {
		return false, fmt.Errorf("write account lease: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check account lease write: %w", err)
	}
	return count == 1, nil
}
