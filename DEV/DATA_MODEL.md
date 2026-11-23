# Data model

Database: SQLite per node (`modernc.org/sqlite`). IDs are ULID text. Dates in UTC, RFC 3339. This is the target version for phases 1–3; migrations create the tables as the modules come in.

## Tables

### Identity and devices
- `accounts(id, email, plan, created_at)` — local mirror of the account (source of truth in Calango Cloud).
- `nodes(id, account_id, kind['pc'|'vps'], name, tailnet_addr, last_seen_at, version)`
- `devices(id, account_id, name, platform['desktop'|'ios'|'android'|'web'], token_hash, scopes, created_at, last_seen_at, revoked_at)`
- `pairing_codes(code_hash, expires_at, used_at)`

### Agents
- `agents(id, owner_id, name, role, instructions, harness, model, reasoning_effort, run_on['pc'|'vps'|'auto'], token_limit_month, approval_level['always'|'sensitive'|'trusted'], paused, created_at, updated_at)`
- `agent_tools(agent_id, connector_id, tool_name, access['none'|'read'|'write'])`

### Conversations
- `conversations(id, kind['dm'|'channel'], title, created_at)`
- `conversation_members(conversation_id, member_kind['person'|'agent'], member_id)`
- `messages(id, conversation_id, thread_id, author_kind, author_id, body, created_at)`
- `runs(id, agent_id, conversation_id, source_message_id, idempotency_key, trigger['message'|'routine'|'task'], node_id, status['queued'|'running'|'done'|'failed'|'stopped'|'waiting_approval'], tokens_in, tokens_out, cost_micros, cost_status['known'|'estimated'|'unknown'], started_at, ended_at, error_code)` — the idempotency key is unique per owner/operation.
- `run_steps(id, run_id, seq, kind['thought'|'tool_call'|'tool_result'|'text'], payload_json, created_at)`

### Routines and tasks
- `routines(id, agent_id, name, trigger_kind['cron'|'interval'|'event'], schedule, event_filter_json, run_on, quiet_start, quiet_end, max_retries, notify_on['always'|'news'|'failure'], paused, next_run_at)`
- `routine_runs(id, routine_id, occurrence_id, scheduled_for, run_id, attempt, status, had_news, created_at)` — `occurrence_id` is unique per routine and scheduled time.
- `tasks(id, title, body, status['todo'|'doing'|'done'], assignee_kind, assignee_id, suggested_by_agent_id, project_id, created_at, updated_at)`

### Approvals and audit
- `approvals(id, run_id, agent_id, action, target, args_json, action_digest, expires_at, status['pending'|'approved'|'denied'|'expired'], outcome['completed'|'failed'|'not_run'|null], decided_by_device_id, decided_at, seq)` — the digest identifies the canonical snapshot that was executed.
- `approval_rules(agent_id, action, decision['always_allow'])`
- `audit_log(id, at, actor_kind, actor_id, action, subject, facts_json)` — append-only.

### Memory, skills, connectors, vault
- `memories(id, scope['account'|'agent'|'project'], scope_id, kind['fact'|'preference'|'decision'], text, source_run_id, created_at, updated_at, deleted_at)`
- `skills(id, name, description, body_md, source['user'|'agent'|'import'], approved)`
- `connectors(id, kind['mcp_stdio'|'mcp_http'], name, config_json, secret_ref)`
- `vault_items(id, name, secret_ref, created_at)` — the value stays in the OS vault; only the reference is here.

### Usage
- `usage(id, at, agent_id, routine_id, project_id, node_id, provider, model, tokens_in, tokens_out, cost_micros NULL, cost_status['known'|'estimated'|'unknown'], pricing_version)` — unknown pricing is not zero cost.

### ADE
- `projects(id, name, repo_path, remote_url, run_on, created_at)`
- `worktrees(id, project_id, task_id, branch, path, node_id, status)`

### Replication
- `events(id, node_id, seq, at, type, entity, entity_id, payload_json)` — append-only log; `(node_id, seq)` is unique.
- `peers(node_id, last_applied_seq)`

## Real-time events (`/v1/events`, WebSocket)

`message.created`, `run.started`, `run.step`, `run.finished`, `approval.requested`, `approval.decided`, `routine.fired`, `routine.failed`, `task.updated`, `node.online`, `node.offline`, `usage.threshold`.

Format: `{ "type": "...", "at": "...", "node_id": "...", "data": { ... } }`. The client may reconnect and read again, because the sequence of steps and the persisted transitions are the source of truth.
