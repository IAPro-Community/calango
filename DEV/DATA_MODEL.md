# Modelo de dados

Banco: SQLite por nó (`modernc.org/sqlite`). IDs são ULID em texto. Datas em UTC, RFC 3339. Esta é a versão alvo da fase 1–3; migrações criam as tabelas conforme os módulos entram.

## Tabelas

### Identidade e dispositivos
- `accounts(id, email, plan, created_at)` — espelho local da conta (fonte de verdade na Calango Cloud).
- `nodes(id, account_id, kind['pc'|'vps'], name, tailnet_addr, last_seen_at, version)`
- `devices(id, account_id, name, platform['desktop'|'ios'|'android'|'web'], token_hash, scopes, created_at, last_seen_at, revoked_at)`
- `pairing_codes(code_hash, expires_at, used_at)`

### Agentes
- `agents(id, owner_id, name, role, instructions, harness, model, reasoning_effort, run_on['pc'|'vps'|'auto'], token_limit_month, approval_level['always'|'sensitive'|'trusted'], paused, created_at, updated_at)`
- `agent_tools(agent_id, connector_id, tool_name, access['none'|'read'|'write'])`

### Conversas
- `conversations(id, kind['dm'|'channel'], title, created_at)`
- `conversation_members(conversation_id, member_kind['person'|'agent'], member_id)`
- `messages(id, conversation_id, thread_id, author_kind, author_id, body, created_at)`
- `runs(id, agent_id, conversation_id, trigger['message'|'routine'|'task'], node_id, status['queued'|'running'|'done'|'failed'|'stopped'|'waiting_approval'], tokens_in, tokens_out, cost_micros, started_at, ended_at, error_code)`
- `run_steps(id, run_id, seq, kind['thought'|'tool_call'|'tool_result'|'text'], payload_json, created_at)`

### Rotinas e tarefas
- `routines(id, agent_id, name, trigger_kind['cron'|'interval'|'event'], schedule, event_filter_json, run_on, quiet_start, quiet_end, max_retries, notify_on['always'|'news'|'failure'], paused, next_run_at)`
- `routine_runs(id, routine_id, run_id, attempt, status, had_news, created_at)`
- `tasks(id, title, body, status['todo'|'doing'|'done'], assignee_kind, assignee_id, suggested_by_agent_id, project_id, created_at, updated_at)`

### Aprovações e auditoria
- `approvals(id, run_id, agent_id, action, target, args_json, status['pending'|'approved'|'denied'|'expired'], outcome['completed'|'failed'|'not_run'|null], decided_by_device_id, decided_at, seq)`
- `approval_rules(agent_id, action, decision['always_allow'])`
- `audit_log(id, at, actor_kind, actor_id, action, subject, facts_json)` — append-only.

### Memória, skills, conectores, cofre
- `memories(id, scope['account'|'agent'|'project'], scope_id, kind['fact'|'preference'|'decision'], text, source_run_id, created_at, updated_at, deleted_at)`
- `skills(id, name, description, body_md, source['user'|'agent'|'import'], approved)`
- `connectors(id, kind['mcp_stdio'|'mcp_http'], name, config_json, secret_ref)`
- `vault_items(id, name, secret_ref, created_at)` — valor fica no cofre do SO; aqui só a referência.

### Uso
- `usage(id, at, agent_id, routine_id, project_id, node_id, provider, model, tokens_in, tokens_out, cost_micros)`

### ADE
- `projects(id, name, repo_path, remote_url, run_on, created_at)`
- `worktrees(id, project_id, task_id, branch, path, node_id, status)`

### Replicação
- `events(id, node_id, seq, at, type, entity, entity_id, payload_json)` — log append-only; `(node_id, seq)` único.
- `peers(node_id, last_applied_seq)`

## Eventos de tempo real (`/v1/events`, WebSocket)

`message.created`, `run.started`, `run.step`, `run.finished`, `approval.requested`, `approval.decided`, `routine.fired`, `routine.failed`, `task.updated`, `node.online`, `node.offline`, `usage.threshold`.

Formato: `{ "type": "...", "at": "...", "node_id": "...", "data": { ... } }`.
