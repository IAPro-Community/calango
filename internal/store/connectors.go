package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sqlite "modernc.org/sqlite"

	"calango/internal/core"
)

const connectorColumns = `id, owner_id, name, transport, command, args, url, created_at, updated_at`

// CreateConnector stores a connector. A name the owner already uses is core.ErrConflict.
func (s *Store) CreateConnector(ctx context.Context, c core.Connector) error {
	args, err := json.Marshal(c.Args)
	if err != nil {
		return fmt.Errorf("encode connector args: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO connectors (`+connectorColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, c.ID, c.OwnerID, c.Name, string(c.Transport),
		c.Command, string(args), c.URL, c.CreatedAt.UTC().Format(usageTimeLayout),
		c.UpdatedAt.UTC().Format(usageTimeLayout))
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: connector %q already exists", core.ErrConflict, c.Name)
		}
		return fmt.Errorf("insert connector %s: %w", c.ID, err)
	}
	return nil
}

// ListConnectors returns the owner's connectors, oldest first.
func (s *Store) ListConnectors(ctx context.Context, ownerID string) ([]core.Connector, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+connectorColumns+` FROM connectors WHERE owner_id = ? ORDER BY created_at, id`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list connectors: %w", err)
	}
	defer rows.Close()

	var out []core.Connector
	for rows.Next() {
		c, err := scanConnector(rows)
		if err != nil {
			return nil, fmt.Errorf("scan connector: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connectors: %w", err)
	}
	return out, nil
}

// GetConnector returns one of the owner's connectors. Another owner's id is not found.
func (s *Store) GetConnector(ctx context.Context, ownerID, id string) (core.Connector, error) {
	c, err := scanConnector(s.db.QueryRowContext(ctx,
		`SELECT `+connectorColumns+` FROM connectors WHERE id = ? AND owner_id = ?`, id, ownerID))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Connector{}, fmt.Errorf("%w: connector %s", core.ErrNotFound, id)
	}
	if err != nil {
		return core.Connector{}, fmt.Errorf("read connector %s: %w", id, err)
	}
	return c, nil
}

// DeleteConnector removes a connector and, through the foreign key, every grant to it.
func (s *Store) DeleteConnector(ctx context.Context, ownerID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM connectors WHERE id = ? AND owner_id = ?`, id, ownerID)
	if err != nil {
		return fmt.Errorf("delete connector %s: %w", id, err)
	}
	return requireOneRow(res, "connector "+id)
}

// SetAgentTool stores or replaces an agent's grant on a connector.
func (s *Store) SetAgentTool(ctx context.Context, t core.AgentTool) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_tools (agent_id, connector_id, access)
		VALUES (?, ?, ?)
		ON CONFLICT (agent_id, connector_id) DO UPDATE SET access = excluded.access`,
		t.AgentID, t.ConnectorID, string(t.Access))
	if err != nil {
		return fmt.Errorf("set grant for agent %s: %w", t.AgentID, err)
	}
	return nil
}

// RemoveAgentTool deletes a grant. Removing a grant that does not exist does nothing.
func (s *Store) RemoveAgentTool(ctx context.Context, agentID, connectorID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM agent_tools WHERE agent_id = ? AND connector_id = ?`, agentID, connectorID)
	if err != nil {
		return fmt.Errorf("remove grant for agent %s: %w", agentID, err)
	}
	return nil
}

// ListAgentTools returns the grants of one agent, ordered by connector id.
func (s *Store) ListAgentTools(ctx context.Context, agentID string) ([]core.AgentTool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT agent_id, connector_id, access FROM agent_tools WHERE agent_id = ? ORDER BY connector_id`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list grants: %w", err)
	}
	defer rows.Close()

	var out []core.AgentTool
	for rows.Next() {
		var t core.AgentTool
		var access string
		if err := rows.Scan(&t.AgentID, &t.ConnectorID, &access); err != nil {
			return nil, fmt.Errorf("scan grant: %w", err)
		}
		t.Access = core.ToolAccess(access)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate grants: %w", err)
	}
	return out, nil
}

func scanConnector(row rowScanner) (core.Connector, error) {
	var c core.Connector
	var transport, args, created, updated string
	if err := row.Scan(&c.ID, &c.OwnerID, &c.Name, &transport, &c.Command, &args, &c.URL,
		&created, &updated); err != nil {
		return core.Connector{}, err
	}
	c.Transport = core.ConnectorTransport(transport)
	if err := json.Unmarshal([]byte(args), &c.Args); err != nil {
		return core.Connector{}, fmt.Errorf("decode connector args: %w", err)
	}
	var err error
	if c.CreatedAt, err = time.Parse(usageTimeLayout, created); err != nil {
		return core.Connector{}, err
	}
	if c.UpdatedAt, err = time.Parse(usageTimeLayout, updated); err != nil {
		return core.Connector{}, err
	}
	return c, nil
}

// isUniqueViolation reports whether err is a UNIQUE constraint failure from SQLite.
func isUniqueViolation(err error) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed")
}
