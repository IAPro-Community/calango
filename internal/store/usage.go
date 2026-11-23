package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/usage"
)

// usageTimeLayout is fixed width in UTC, so SQL text comparison on usage.at
// matches chronological order.
const usageTimeLayout = "2006-01-02T15:04:05.000Z07:00"

var _ gateway.Ledger = (*Store)(nil)

// AgentTokenLimit returns the agent's monthly token limit.
func (s *Store) AgentTokenLimit(ctx context.Context, agentID string) (int64, error) {
	var limit int64
	err := s.db.QueryRowContext(ctx,
		`SELECT token_limit_month FROM agents WHERE id = ?`, agentID).Scan(&limit)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%w: %s", gateway.ErrUnknownAgent, agentID)
	}
	if err != nil {
		return 0, fmt.Errorf("read agent token limit: %w", err)
	}
	return limit, nil
}

// MonthlyTokens sums tokens for the agent with from <= at < to.
func (s *Store) MonthlyTokens(ctx context.Context, agentID string, from, to time.Time) (int64, error) {
	var total int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(tokens_in + tokens_out), 0) FROM usage
		WHERE agent_id = ? AND at >= ? AND at < ?`,
		agentID,
		from.UTC().Format(usageTimeLayout),
		to.UTC().Format(usageTimeLayout),
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum monthly tokens: %w", err)
	}
	return total, nil
}

// RecordUsage appends one usage row. The schema rejects a cost that does not
// match its status (unknown cost must be NULL; known or estimated must not be).
func (s *Store) RecordUsage(ctx context.Context, rec gateway.Record) error {
	var cost any
	if rec.CostMicros != nil {
		cost = *rec.CostMicros
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO usage (id, at, agent_id, routine_id, project_id, node_id, provider, model,
			tokens_in, tokens_out, cost_micros, cost_status, pricing_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ID,
		rec.At.UTC().Format(usageTimeLayout),
		rec.Caller.AgentID,
		nullIfEmpty(rec.Caller.RoutineID),
		nullIfEmpty(rec.Caller.ProjectID),
		nullIfEmpty(rec.Caller.NodeID),
		rec.Provider,
		rec.Model,
		rec.TokensIn,
		rec.TokensOut,
		cost,
		string(rec.CostStatus),
		nullIfEmpty(rec.PricingVersion),
	)
	if err != nil {
		return fmt.Errorf("record usage: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// usageGroupColumns maps each group to its usage column. Only these fixed expressions
// reach the SQL text; a group value from a request is never interpolated.
var usageGroupColumns = map[usage.Group]string{
	usage.GroupAgent:   "u.agent_id",
	usage.GroupRoutine: "u.routine_id",
	usage.GroupProject: "u.project_id",
	usage.GroupNode:    "u.node_id",
	usage.GroupModel:   "u.model",
}

// UsageTotals sums the owner's usage within [q.From, q.To). Unknown-cost rows add
// nothing to the cost sum; they are counted in UnknownCostRows instead.
func (s *Store) UsageTotals(ctx context.Context, ownerID string, q usage.Query) ([]usage.Row, error) {
	column, ok := usageGroupColumns[q.GroupBy]
	if !ok {
		return nil, fmt.Errorf("usage group %q: %w", q.GroupBy, core.ErrInvalidInput)
	}
	key := "COALESCE(" + column + ", '')"
	where := []string{"a.owner_id = ?"}
	args := []any{ownerID}
	if !q.From.IsZero() {
		where = append(where, "u.at >= ?")
		args = append(args, q.From.UTC().Format(usageTimeLayout))
	}
	if !q.To.IsZero() {
		where = append(where, "u.at < ?")
		args = append(args, q.To.UTC().Format(usageTimeLayout))
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+key+`,
			COALESCE(SUM(u.tokens_in), 0), COALESCE(SUM(u.tokens_out), 0),
			COALESCE(SUM(u.cost_micros), 0), COUNT(CASE WHEN u.cost_status = 'unknown' THEN 1 END)
		FROM usage u JOIN agents a ON a.id = u.agent_id
		WHERE `+strings.Join(where, " AND ")+`
		GROUP BY `+key+`
		ORDER BY `+key, args...)
	if err != nil {
		return nil, fmt.Errorf("query usage: %w", err)
	}
	defer rows.Close()

	var out []usage.Row
	for rows.Next() {
		var r usage.Row
		if err := rows.Scan(&r.Key, &r.TokensIn, &r.TokensOut, &r.CostMicros, &r.UnknownCostRows); err != nil {
			return nil, fmt.Errorf("scan usage row: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read usage rows: %w", err)
	}
	return out, nil
}
