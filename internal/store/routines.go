package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"calango/internal/core"
)

const routineColumns = `id, owner_id, agent_id, name, prompt, trigger_kind, schedule, run_on,
	quiet_start, quiet_end, max_retries, notify_on, paused, next_run_at, created_at, updated_at,
	conversation_id, ended_at`

const occurrenceColumns = `id, routine_id, occurrence_id, scheduled_for, run_id, attempt, status,
	had_news, last_error, next_attempt_at, created_at, updated_at`

// InsertRoutine stores a new routine. The caller validates it first.
func (s *Store) InsertRoutine(ctx context.Context, r core.Routine) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO routines (`+routineColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.OwnerID, r.AgentID, r.Name, r.Prompt, string(r.TriggerKind), r.Schedule, string(r.RunOn),
		intPtrValue(r.QuietStart), intPtrValue(r.QuietEnd), r.MaxRetries, string(r.NotifyOn), boolInt(r.Paused),
		r.NextRunAt.UTC().Format(usageTimeLayout), r.CreatedAt.UTC().Format(usageTimeLayout),
		r.UpdatedAt.UTC().Format(usageTimeLayout), nullIfEmpty(r.ConversationID), timeValue(r.EndedAt))
	if err != nil {
		return fmt.Errorf("insert routine %s: %w", r.ID, err)
	}
	return nil
}

// GetRoutine returns the routine only when it belongs to ownerID.
func (s *Store) GetRoutine(ctx context.Context, ownerID, id string) (core.Routine, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+routineColumns+` FROM routines WHERE id = ? AND owner_id = ?`, id, ownerID)
	return scanRoutineOrNotFound(row, id)
}

// GetRoutineByID returns a routine without an owner check. Only the scheduler uses it.
func (s *Store) GetRoutineByID(ctx context.Context, id string) (core.Routine, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+routineColumns+` FROM routines WHERE id = ?`, id)
	return scanRoutineOrNotFound(row, id)
}

// ListRoutines returns the owner's routines, oldest first.
func (s *Store) ListRoutines(ctx context.Context, ownerID string) ([]core.Routine, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+routineColumns+` FROM routines WHERE owner_id = ? ORDER BY created_at, id`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list routines: %w", err)
	}
	defer rows.Close()

	var out []core.Routine
	for rows.Next() {
		r, err := scanRoutine(rows)
		if err != nil {
			return nil, fmt.Errorf("scan routine: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list routines: %w", err)
	}
	return out, nil
}

// ListDueRoutines returns active routines whose next scheduled time is at or before now.
func (s *Store) ListDueRoutines(ctx context.Context, now time.Time) ([]core.Routine, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+routineColumns+` FROM routines WHERE paused = 0 AND ended_at IS NULL AND next_run_at <= ? ORDER BY next_run_at, id`,
		now.UTC().Format(usageTimeLayout))
	if err != nil {
		return nil, fmt.Errorf("list due routines: %w", err)
	}
	defer rows.Close()

	var out []core.Routine
	for rows.Next() {
		r, err := scanRoutine(rows)
		if err != nil {
			return nil, fmt.Errorf("scan routine: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list due routines: %w", err)
	}
	return out, nil
}

// UpdateRoutine overwrites the mutable fields of an existing routine, including its
// schedule position. created_at is kept.
func (s *Store) UpdateRoutine(ctx context.Context, r core.Routine) error {
	res, err := s.db.ExecContext(ctx, `UPDATE routines SET
		name = ?, prompt = ?, trigger_kind = ?, schedule = ?, run_on = ?, quiet_start = ?, quiet_end = ?,
		max_retries = ?, notify_on = ?, paused = ?, next_run_at = ?, updated_at = ?, conversation_id = ?, ended_at = ?
		WHERE id = ? AND owner_id = ?`,
		r.Name, r.Prompt, string(r.TriggerKind), r.Schedule, string(r.RunOn), intPtrValue(r.QuietStart),
		intPtrValue(r.QuietEnd), r.MaxRetries, string(r.NotifyOn), boolInt(r.Paused),
		r.NextRunAt.UTC().Format(usageTimeLayout), r.UpdatedAt.UTC().Format(usageTimeLayout),
		nullIfEmpty(r.ConversationID), timeValue(r.EndedAt),
		r.ID, r.OwnerID)
	if err != nil {
		return fmt.Errorf("update routine %s: %w", r.ID, err)
	}
	return requireOneRow(res, "routine "+r.ID)
}

// DeleteRoutine removes a routine and its occurrence history.
func (s *Store) DeleteRoutine(ctx context.Context, ownerID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM routines WHERE id = ? AND owner_id = ?`, id, ownerID)
	if err != nil {
		return fmt.Errorf("delete routine %s: %w", id, err)
	}
	return requireOneRow(res, "routine "+id)
}

// InsertOccurrence records a scheduled time. It returns false when the occurrence already
// exists, which is how a redelivered schedule reuses the stored record.
func (s *Store) InsertOccurrence(ctx context.Context, o core.Occurrence) (bool, error) {
	res, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO routine_runs (`+occurrenceColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.ID, o.RoutineID, o.OccurrenceID, o.ScheduledFor.UTC().Format(usageTimeLayout),
		nullIfEmpty(o.RunID), o.Attempt, string(o.Status), nullBool(o.HadNews), o.LastError,
		o.NextAttemptAt.UTC().Format(usageTimeLayout), o.CreatedAt.UTC().Format(usageTimeLayout),
		o.UpdatedAt.UTC().Format(usageTimeLayout))
	if err != nil {
		return false, fmt.Errorf("insert occurrence %s: %w", o.OccurrenceID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count inserted occurrences: %w", err)
	}
	return n == 1, nil
}

// GetOccurrence returns the occurrence with the given routine and deterministic ID.
func (s *Store) GetOccurrence(ctx context.Context, routineID, occurrenceID string) (core.Occurrence, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+occurrenceColumns+`
		FROM routine_runs WHERE routine_id = ? AND occurrence_id = ?`, routineID, occurrenceID)
	o, err := scanOccurrence(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Occurrence{}, fmt.Errorf("%w: occurrence %s", core.ErrNotFound, occurrenceID)
	}
	if err != nil {
		return core.Occurrence{}, fmt.Errorf("read occurrence %s: %w", occurrenceID, err)
	}
	return o, nil
}

// ListOccurrences returns a routine's occurrences, oldest scheduled time first.
func (s *Store) ListOccurrences(ctx context.Context, routineID string) ([]core.Occurrence, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+occurrenceColumns+`
		FROM routine_runs WHERE routine_id = ? ORDER BY scheduled_for, id`, routineID)
	if err != nil {
		return nil, fmt.Errorf("list occurrences: %w", err)
	}
	defer rows.Close()

	var out []core.Occurrence
	for rows.Next() {
		o, err := scanOccurrence(rows)
		if err != nil {
			return nil, fmt.Errorf("scan occurrence: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list occurrences: %w", err)
	}
	return out, nil
}

// ListDueOccurrences returns pending occurrences whose next attempt is at or before now.
func (s *Store) ListDueOccurrences(ctx context.Context, now time.Time) ([]core.Occurrence, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+occurrenceColumns+`
		FROM routine_runs WHERE status = 'pending' AND next_attempt_at <= ? ORDER BY next_attempt_at, id`,
		now.UTC().Format(usageTimeLayout))
	if err != nil {
		return nil, fmt.Errorf("list due occurrences: %w", err)
	}
	defer rows.Close()

	var out []core.Occurrence
	for rows.Next() {
		o, err := scanOccurrence(rows)
		if err != nil {
			return nil, fmt.Errorf("scan occurrence: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list due occurrences: %w", err)
	}
	return out, nil
}

// ClaimOccurrence moves a due pending occurrence to running and counts the attempt. It
// returns false when another worker already claimed it or it is not due yet.
func (s *Store) ClaimOccurrence(ctx context.Context, id string, now time.Time) (core.Occurrence, bool, error) {
	ts := now.UTC().Format(usageTimeLayout)
	res, err := s.db.ExecContext(ctx, `UPDATE routine_runs
		SET status = 'running', attempt = attempt + 1, updated_at = ?
		WHERE id = ? AND status = 'pending' AND next_attempt_at <= ?`, ts, id, ts)
	if err != nil {
		return core.Occurrence{}, false, fmt.Errorf("claim occurrence %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return core.Occurrence{}, false, fmt.Errorf("count claimed occurrences: %w", err)
	}
	if n == 0 {
		return core.Occurrence{}, false, nil
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+occurrenceColumns+` FROM routine_runs WHERE id = ?`, id)
	o, err := scanOccurrence(row)
	if err != nil {
		return core.Occurrence{}, false, fmt.Errorf("read claimed occurrence %s: %w", id, err)
	}
	return o, true, nil
}

// PostponeOccurrence moves a pending occurrence to a later time without counting an attempt.
// Quiet windows use it so a run is delayed, not lost.
func (s *Store) PostponeOccurrence(ctx context.Context, id string, until, now time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE routine_runs
		SET next_attempt_at = ?, updated_at = ? WHERE id = ? AND status = 'pending'`,
		until.UTC().Format(usageTimeLayout), now.UTC().Format(usageTimeLayout), id)
	if err != nil {
		return fmt.Errorf("postpone occurrence %s: %w", id, err)
	}
	return requireTransition(res, "occurrence "+id)
}

// FinishOccurrence stores the outcome of a running occurrence. It only applies while the
// occurrence is still running, so a stale worker cannot overwrite a newer result.
func (s *Store) FinishOccurrence(ctx context.Context, o core.Occurrence) error {
	res, err := s.db.ExecContext(ctx, `UPDATE routine_runs SET
		status = ?, run_id = ?, had_news = ?, last_error = ?, next_attempt_at = ?, updated_at = ?
		WHERE id = ? AND status = 'running'`,
		string(o.Status), nullIfEmpty(o.RunID), nullBool(o.HadNews), o.LastError,
		o.NextAttemptAt.UTC().Format(usageTimeLayout), o.UpdatedAt.UTC().Format(usageTimeLayout), o.ID)
	if err != nil {
		return fmt.Errorf("finish occurrence %s: %w", o.ID, err)
	}
	return requireTransition(res, "occurrence "+o.ID)
}

// RequeueInterruptedOccurrences turns occurrences left running by a previous process back
// into due pending ones. Their attempt was already counted when they were claimed.
func (s *Store) RequeueInterruptedOccurrences(ctx context.Context, now time.Time) (int64, error) {
	ts := now.UTC().Format(usageTimeLayout)
	res, err := s.db.ExecContext(ctx, `UPDATE routine_runs
		SET status = 'pending', last_error = 'interrupted', next_attempt_at = ?, updated_at = ?
		WHERE status = 'running'`, ts, ts)
	if err != nil {
		return 0, fmt.Errorf("requeue interrupted occurrences: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count requeued occurrences: %w", err)
	}
	return n, nil
}

func scanRoutineOrNotFound(row rowScanner, id string) (core.Routine, error) {
	r, err := scanRoutine(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Routine{}, fmt.Errorf("%w: routine %s", core.ErrNotFound, id)
	}
	if err != nil {
		return core.Routine{}, fmt.Errorf("read routine %s: %w", id, err)
	}
	return r, nil
}

func scanRoutine(row rowScanner) (core.Routine, error) {
	var (
		r                               core.Routine
		trigger, runOn, notify          string
		quietStart, quietEnd            sql.NullInt64
		paused                          int
		nextRunAt, createdAt, updatedAt string
		conversationID, endedAt         sql.NullString
	)
	if err := row.Scan(&r.ID, &r.OwnerID, &r.AgentID, &r.Name, &r.Prompt, &trigger, &r.Schedule, &runOn,
		&quietStart, &quietEnd, &r.MaxRetries, &notify, &paused, &nextRunAt, &createdAt, &updatedAt,
		&conversationID, &endedAt); err != nil {
		return core.Routine{}, err
	}
	r.TriggerKind = core.TriggerKind(trigger)
	r.RunOn = core.RunOn(runOn)
	r.NotifyOn = core.NotifyOn(notify)
	r.Paused = paused == 1
	r.QuietStart = intFromNull(quietStart)
	r.QuietEnd = intFromNull(quietEnd)
	var err error
	if r.NextRunAt, err = time.Parse(usageTimeLayout, nextRunAt); err != nil {
		return core.Routine{}, fmt.Errorf("parse next_run_at: %w", err)
	}
	if r.CreatedAt, err = time.Parse(usageTimeLayout, createdAt); err != nil {
		return core.Routine{}, fmt.Errorf("parse created_at: %w", err)
	}
	if r.UpdatedAt, err = time.Parse(usageTimeLayout, updatedAt); err != nil {
		return core.Routine{}, fmt.Errorf("parse updated_at: %w", err)
	}
	r.ConversationID = conversationID.String
	if r.EndedAt, err = optionalTime(endedAt); err != nil {
		return core.Routine{}, fmt.Errorf("parse ended_at: %w", err)
	}
	return r, nil
}

func scanOccurrence(row rowScanner) (core.Occurrence, error) {
	var (
		o                                           core.Occurrence
		routineID                                   string
		scheduledFor, nextAttemptAt, createdAt, upd string
		runID                                       sql.NullString
		status                                      string
		hadNews                                     sql.NullInt64
	)
	if err := row.Scan(&o.ID, &routineID, &o.OccurrenceID, &scheduledFor, &runID, &o.Attempt, &status,
		&hadNews, &o.LastError, &nextAttemptAt, &createdAt, &upd); err != nil {
		return core.Occurrence{}, err
	}
	o.RoutineID = routineID
	o.RunID = runID.String
	o.Status = core.OccurrenceStatus(status)
	if hadNews.Valid {
		v := hadNews.Int64 == 1
		o.HadNews = &v
	}
	var err error
	if o.ScheduledFor, err = time.Parse(usageTimeLayout, scheduledFor); err != nil {
		return core.Occurrence{}, fmt.Errorf("parse scheduled_for: %w", err)
	}
	if o.NextAttemptAt, err = time.Parse(usageTimeLayout, nextAttemptAt); err != nil {
		return core.Occurrence{}, fmt.Errorf("parse next_attempt_at: %w", err)
	}
	if o.CreatedAt, err = time.Parse(usageTimeLayout, createdAt); err != nil {
		return core.Occurrence{}, fmt.Errorf("parse created_at: %w", err)
	}
	if o.UpdatedAt, err = time.Parse(usageTimeLayout, upd); err != nil {
		return core.Occurrence{}, fmt.Errorf("parse updated_at: %w", err)
	}
	return o, nil
}

func intPtrValue(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func intFromNull(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullBool(b *bool) any {
	if b == nil {
		return nil
	}
	return boolInt(*b)
}
