package store

import (
	"context"
	"database/sql"
	"fmt"

	"calango/internal/audit"
)

// ExportAudit returns the whole chain and its checkpoint from one read transaction. The
// export is a consistent snapshot: an append committing meanwhile is either fully in or
// fully out, so the checkpoint always matches the entries it is returned with.
func (s *Store) ExportAudit(ctx context.Context) (audit.Export, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return audit.Export{}, fmt.Errorf("begin audit export: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	cp, err := loadCheckpoint(ctx, tx)
	if err != nil {
		return audit.Export{}, err
	}
	records, err := loadChain(ctx, tx)
	if err != nil {
		return audit.Export{}, err
	}
	return audit.Export{Records: records, Checkpoint: cp}, nil
}
