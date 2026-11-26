# Verification

Updated: 2026-11-23.

## Go core (verified in this step)

Environment: Go 1.27.4 linux/amd64. `go.mod` keeps Go 1.22 as the minimum version.

| Check | Result |
|---|---|
| `GOCACHE=/tmp/calango-go-cache go test ./... -count=1` | ok; tests in `internal/core` and `internal/config` |
| `GOCACHE=/tmp/calango-go-cache go test -race ./... -count=1` | ok |
| `GOCACHE=/tmp/calango-go-cache go vet ./...` | ok, no output |
| `gofmt -l .` | ok, no files |
| `git diff --check` | ok |
| Build `calango` linux/amd64 and linux/arm64 | ok |
| Build `calango` windows/amd64 and windows/arm64 | ok |
| Build `calango` darwin/amd64 and darwin/arm64 | ok |
| `CALANGO_ADDR=invalid` on the binary | fails before the server starts, with a host/port error |
| Binary on `127.0.0.1:7399`, `GET /healthz` | 200, body `{"status":"ok"}` |
| Test process stopped | ok |

Windows/macOS builds were cross-compiled on Linux; they do not replace native tests or installers. The GitHub workflow has not run yet because no push has been authorized.

## Docker (history and current state)

The builds listed below were made on 2026-11-22, as recorded in the worklog. In this step, `docker version` showed client 29.1.3, but the sandbox denied access to the `/var/run/docker.sock` socket; therefore the builds were not revalidated.

| Check | Result |
|---|---|
| `docker build -t calango-site .` (root Dockerfile), history | ok, image `nginxinc/nginx-unprivileged:1.27-alpine` |
| `nginx -t` inside the site container | ok |
| `GET /healthz` on the site | 200 |
| `GET /` on the site | 200, `text/html` |
| CSP headers, `X-Frame-Options: DENY`, `nosniff` | present |
| `docker build -f deploy/Dockerfile.node -t calango/calango .`, history | ok, 13.3 MB |
| `GET /healthz` on the node (`calango/calango`, nonroot user) | 200, `{"status":"ok"}` |
| `docker run` with bridge networking | fails: `operation not supported` when creating the veth on this machine |
| Build and run with `--network host` | ok (both images) |
| `hadolint` on the Dockerfiles | not run (not installed) |

Note: Docker bridge networking does not work on this machine (veth error). Run tests used `--network host`. On normal machines, `docker run -p` should work; this has not been verified here yet.

## Contract and tokens

| Check | Result |
|---|---|
| `api/openapi.yaml` is valid YAML and all `$ref`s resolve (15 routes, 11 schemas) | ok (PyYAML) |
| YAML parse of the workflow and OpenAPI in this step | ok (PyYAML) |
| Redocly CLI `2.81.0 lint --extends minimal --max-problems=0 api/openapi.yaml` | ok, no problems or warnings (OpenAPI 3.1) |
| `packages/tokens/tokens.json` is valid JSON in this step | ok |
| `node scripts/generate-tokens.mjs` | ok; generated CSS and TypeScript/Dart constants |
| `node scripts/generate-tokens.mjs --check` | ok; three artifacts present and no drift |
| `node --check scripts/generate-tokens.mjs` | ok |
| i18n catalog generation/drift | ok since F0-T07: `node scripts/generate-i18n.mjs --check` in CI; screens consuming it come in F1-T26 |
| Links in `DEV/INDEX.md` and `ROADMAP.md` to ADRs and specs exist | ok |

## Site (`site/index.html`)

Served at `http://localhost:4377`, embedded browser.

| Check | Result |
|---|---|
| Console without errors | ok |
| Logo images load | ok |
| Demo: 5 scenes with the expected state | ok, checked via DOM |
| Notification text readable in dark mode | fixed |
| Contrast measured by script (light and dark), transitions disabled | text failures fixed; false positives remain (phone gradient and nav with `oklab`, which the script does not interpret) |
| ADE section title in dark mode | was invisible; fixed and checked in a capture |
| Title marker (`.sweep`) in light and dark | readable after reveal |
| Horizontal overflow (measured in a fixed-width iframe) | fixed: `scrollWidth` equals the window width at 320, 360, 390, 414, 600, 768 and 1024 px. Causes: grid columns without `minmax(0,1fr)` (how-to/contribute and demo), wall notes, rotated tape |
| "How it works" diagram on phone | larger captions, shorter subtexts, labels outside the PC baseline |
| Full visual review in light and dark, all sections | partial: hero (light) and ADE (dark) checked |

## Orquestrador Maestro

| Check | Result |
|---|---|
| Stable version of the V1 line identified (1.1.1, released 2026-10-18) | ok, from GitHub releases |
| Apache-2.0 license confirmed | ok (`LICENSE` file) |
| Installation via npm and `verify` in Calango | not run |

## Language and site i18n (2026-11-23)

| Check | Result |
|---|---|
| Redocly lint `api/openapi.yaml` (`@redocly/cli@2.81.0`, minimal) after translated descriptions | ok, valid |
| Site: switching PT / EN / ES updates text, `<html lang>` and title, no console errors | ok (preview on `localhost:4377`) |
| Site: no entries with empty `data-i18n` text in English | ok (172 keys) |
| Site: horizontal overflow at 375 px and 1024 px | ok (`scrollWidth` equals viewport) |
| Site: rotating hero word not clipped (height increased) | fixed, not re-captured after the last CSS change |
| Not run: GitHub Actions run, Docker build on this machine (bridge networking not validated), full visual review of every section in every locale | pending |

## Site header (2026-11-23)

| Check | Result |
|---|---|
| Document horizontal overflow (`scrollWidth` vs viewport) at 1568, 1024, 1021, 1020, 1001, 960, 390 and 320 px, EN / ES / PT-BR | ok, no unclipped element past the viewport. Before the fix: ES 320 px overflowed 345 px (plan cards) |
| Menu visible from 1021 px, hidden at 1020 px and below | ok |
| Gaps brand → menu → CTA at 1021 px ES | 22 px / 22 px |
| Gaps at 1024 px EN / ES / PT-BR (brand → menu) | 65 / 24 / 31 px |
| Brand box equals text width at 1001 px ES (before: 115 px box for 132 px text) | ok (`flex-shrink:0`) |
| CTA label one line (button height 37 px) at 1024 px ES and PT-BR | ok |
| `?lang=pt-BR`: `<html lang="pt-BR">`, PT button pressed | ok |
| Visual capture at 1568, 1024, 1021, 1020, 1001, 390 and 320 px | ok, dark theme only |
| Not run: touch devices, Firefox and Safari, light theme header, real phone | pending |

## Store and startup (F1-T01/T02, 2026-11-23)

Environment: Go 1.27.4 linux/amd64; minimum Go 1.22.

| Check | Result |
|---|---|
| `go test ./internal/store -count=1` | ok: schema and tables, constraints (budget, unknown cost, JSON steps), audit log append-only, WAL, `foreign_keys=1`, file mode 0600, migration checksum, rollback on failure |
| `go test -race ./internal/store/... -count=1` | ok |
| `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `gofmt -l .` | ok |
| `go mod tidy` and `go mod verify` | ok; `modernc.org/sqlite v1.35.0` is a direct dependency |
| Smoke: `CALANGO_ADDR=127.0.0.1:17317 CALANGO_DATA_DIR=<tmp>` run of `calango` | `/healthz` returned `{"status":"ok"}`, `calango.db` created, clean SIGTERM shutdown |
| Limit | driver pinned to v1.35.0: newer releases require Go 1.23+, above the project's minimum |

## Gateway (F1-T03, 2026-11-23)

Environment: Go 1.27.4 linux/amd64.

| Check | Result |
|---|---|
| `go test -race ./internal/core/... ./internal/gateway/... ./internal/store/... -count=1` | ok: 9 gateway tests (refusal before the provider when the estimate exceeds the remaining budget, cost = tokens × price, unknown price stored as NULL, estimated status, failed call charged the estimate and its reservation released, concurrent call refused while the first is in flight, unknown agent, 80% and 100% thresholds emitted once, duplicate provider rejected); SQLite usage ledger tests (monthly window sums, unknown cost NULL); ID shape and time order |
| `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `gofmt -l .` | ok |
| No provider SDK imported outside `internal/gateway` | ok (no provider SDK exists yet) |
| Limits | reservations live in memory: correct for one process only (V1). If `NewID` fails, the usage row is not written. The price table is empty: no verified prices in the repo yet, so every model is recorded as `unknown` until a price is added with its source and date. Not yet wired into `cmd/calango` (no provider exists) |

## Agents (F1-T06, 2026-11-23)

Environment: Go 1.27.4 linux/amd64.

| Check | Result |
|---|---|
| `go test -race ./internal/core/... ./internal/store/... ./internal/agents/... -count=1` | ok: agent validation (complete configuration accepted; table of invalid configurations rejected); migration probe tests using the next free version and rollback; schema table list includes `node_state`; agents service: refuses unusable configuration, stores every field and defaults, PUT keeps identity, owner isolation, delete refused with usage history (`conflict`), delete without history, global pause covers new agents until resume, monthly usage counts only the current UTC month |
| `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `gofmt -l .` | ok |
| `npx @redocly/cli@2.81.0 lint api/openapi.yaml` | valid; 16 warnings from the `operation-4xx-response` rule, same class as existing operations (including `listAgents`); no errors |
| Limits | No HTTP handlers yet (device auth first). Pause does not stop runs yet (F1-T07 enforces it). The RESTRICT violation is matched by message text "FOREIGN KEY constraint failed": this driver build reports extended code 1811, not 787. `agents.paused` stays 0; per-agent pause is not in V1. `ListAgents` runs one usage query per agent (N+1, acceptable for a personal node). |

## Conversations and runs (F1-T07, 2026-11-23)

Environment: Go 1.27.4 linux/amd64.

| Check | Result |
|---|---|
| `go test -race ./internal/core/... ./internal/store/... ./internal/conversations/... -count=1` | ok: 8 service tests with a fake provider and the real gateway over in-memory SQLite: message runs to done with ordered steps and tokens; repeated key returns the same message and run and calls the model once; stop ends only its run and the queued run starts when the slot frees; paused node records `failed/agent_paused` without calling the model; restart after crash records `failed/interrupted`; retry of a stopped run carries the instructions in the prompt, refuses a running run; exhausted budget fails before the provider; another owner gets `not_found`. Store schema test lists `conversation_members` |
| `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `gofmt -l .` | ok (gofmt reformatted the struct alignment in `internal/store/conversations.go`) |
| Limits | Guide/steer not implemented: it needs a harness that emits steps (F1-T04/T05, blocked by D3). Per-agent "one reply at a time" not implemented. Reconnection is a transport concern (`/v1/events`, F1-T13). Idempotency for messages lives in `runs.idempotency_key` (no key column on `messages`). Run cost stays `unknown`: the gateway does not return cost to the caller; real cost is in `usage` (F1-T17). Reservations are in memory: one process only. User stops log a `model call failed` warning because the cancelled call returns an error; the status is still `stopped`. No HTTP handlers yet |

## Approvals (F1-T08, 2026-11-23)

Environment: Go 1.27.4 linux/amd64.

| Check | Result |
|---|---|
| `go test -race ./internal/core/... ./internal/store/... ./internal/approvals/... -count=1` | ok: canonical action (key order and whitespace share a digest; target or args change it; non-sensitive kind, empty target, non-object or trailing args rejected). Database enforces immutable snapshot, final status and final receipt (triggers), and a decision happens once (second decision is `conflict`, not audited). Service: device required; approve then execute runs the snapshot once with receipt `completed`; failure recorded with reason; changed action refused with `approval.blocked` and a new request; "always approve" skips the card and audits `approval.auto_approved`; deny is final; expiry at 24 h refuses the decision and `Sweep` marks it expired; other owner gets `not_found`; non-sensitive action rejected |
| `go test ./internal/conversations/... -race -count=5` | ok. Fixed a flaky retry test: it stopped the first run before it reached the provider (cancelled during loading), so it expected two prompts and got one. It now waits for the first provider call before stopping |
| `go test ./... -count=1`, `go vet ./...`, `gofmt -l .` | ok |
| Limits | No HTTP routes or card yet (F1-T13). The run does not enter `waiting_approval` yet. An approved action without a receipt after a crash is not reconciled. `Request` does not check that `runID` belongs to the agent. A panic inside `run` leaves the action marked as executing in memory |

## Routines (F1-T09, 2026-11-23)

| Check | Result |
|---|---|
| `go test -race ./internal/routines -count=3` | ok, 13 tests: one execution per scheduled time; missed hourly runs coalesce to one execution at the earliest missed time; transient failure retries the same row (`attempt=2`) with no failure notification; exhausted retries notify once with the error and stop; quiet window postpones to the window end; `notify_on=news` silent without news; `notify_on=failure` silent on success; restart requeues the interrupted occurrence and reuses its record (two services over one store); pause and global pause stop execution; resume does not fire times missed while paused; `run_on` routes to this node only; cross-owner agent refused; unknown node refused |
| `go test ./internal/core ./internal/store -count=1` | ok: cron parsing rejects malformed fields; `Next` semantics (step, weekday, 7 = Sunday, OR of day fields, time zone, 29 February); a never-matching expression errors instead of looping; routine validation; occurrence ID is timezone-free; backoff schedule; quiet window across midnight; schema lists `routine_runs` and `routines` |
| `go test ./... -count=1`, `go vet ./...`, `gofmt -l .` | ok |
| Limits | No HTTP routes, no client events, no executor that creates an agent run (the daemon has no executor yet), `routine_finish` and event triggers not implemented. Routines run only in-process while the daemon is up; there is no cross-node handoff for `run_on=vps` |

## Memory (F1-T11, 2026-11-23)

| Check | Result |
|---|---|
| `go test ./internal/memory ./internal/conversations ./internal/store -count=1` | ok: create and validate (empty, too long, bad scope or kind, agent scope without agent, account scope with agent); context shows the memory with its header; empty context stays empty; budget keeps the most recent memories within 1500 tokens; edit and delete take effect at once; other owner gets `not_found`; purge keeps deleted rows inside 30 days and removes older ones; a run's prompt carries only active memories; schema lists `memories` |
| `go test -race ./internal/memory ./internal/conversations -count=1` | ok |
| `go test ./... -count=1`, `go vet ./...`, `gofmt -l .` | ok |
| Limits | No `memory_save` tool (D3) and no HTTP routes (F1-T13). `project` scope and `source_run_id` not implemented. No purge loop; purge runs when called. Ranking is by recency only |

## HTTP API and local token (F1-T13 partial, F1-T14 partial, 2026-11-23)

Environment: Go 1.27.4 linux/amd64.

| Check | Result |
|---|---|
| `go test ./internal/api ./internal/auth -count=1` | ok: no token and wrong token get 401 with `WWW-Authenticate`; `/healthz` 200 without token; agent create 201 with trimmed name and defaults `run_on=pc`, `approval_level=sensitive`; get, replace, delete 204, then 404 `not_found`; unknown field, missing token limit and trailing JSON get 400 `invalid_input`; memory create (account scope), agent scope with unknown agent gets 404, bad scope filter gets 400, patch, delete, empty list is `[]`; approvals empty list is `[]`, bad status gets 400, unknown approval decide gets 404, bad decision gets 400; token file is mode 0600, reused on the second call, refused when 0644 and when empty |
| `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .` | ok |
| Redocly lint `api/openapi.yaml --max-problems=0` | valid, no warnings (adding `createMemory` cleared the `MemoryInput` unused-component warning) |
| Smoke: built `calango`, temporary `CALANGO_DATA_DIR`, `CALANGO_ADDR=127.0.0.1:7399` | `/healthz` 200; `/v1/agents` 401 with no token and with a wrong token; create 201 and list 200 with the token; token file mode 600; process stopped after the check |
| Limits (as of this entry) | Only agents, memories and approvals were served; pairing, devices and usage were added later (see sections below). Owner and device were constants. The token was a file, not the vault (F1-T10); no rotation. The log line names the token file path, never its value |

## Device pairing (F1-T15 partial, 2026-11-23)

Environment: Go 1.27.4 linux/amd64.

| Check | Result |
|---|---|
| `go test ./internal/devices ./internal/api ./internal/store -count=1` | ok. Service: code is single use; expiry at the TTL boundary; wrong code refused; blank name and unknown platform refused without consuming the code; lowercase, hyphen and space normalized; device authenticates and `last_seen_at` is set; revoke stops the token at once and the list keeps revoked rows; scope copied from the code; empty and unknown token refused; revoke under another owner gives `not_found`; second revoke is a no-op. HTTP: pairing flow gives a working token and the device list leaks no token material; `read` scope reads, writes get 403 `forbidden`; `approve_only` lists approvals and gets 403 on agents and memories; revoked device gets 401; code reuse gets 401; the sixth `/v1/pair` attempt from one origin gets 429 `rate_limited`; `/v1/pairing-codes` gets 401 without token and 403 for a `read` device; unknown token gets 401; store schema test lists `devices` and `pairing_codes` |
| `go test ./... -count=1`, `go vet ./...`, `gofmt -l .` | ok |
| Redocly lint `api/openapi.yaml --max-problems=0` | valid, no warnings |
| Smoke: built `calango`, temporary `CALANGO_DATA_DIR`, `CALANGO_ADDR=127.0.0.1:7399` | `/healthz` 200; code issued with the local token (8 characters); `POST /v1/pair` without token returns a device with scope `full`; device token reads `/v1/agents` 200; device listed as `Phone`, `ios`, `full`, not revoked; code reuse 401; `DELETE /v1/devices/{id}` 204; revoked token 401; token file mode 600; process stopped |
| Limits | Rate limit keys on the TCP peer, or on the `X-Forwarded-For` client when the peer is listed in `CALANGO_TRUSTED_PROXIES` (Q19). Not verified behind a real proxy (no VPS yet). No QR code, no mobile app, no administrator password on the VPS. Desktop token is a file, not the vault (F1-T10), and has no rotation. Owner is single (`owner`). Device tokens have no expiry or rotation |
| Proxy-aware pairing limit (Q19, 2026-11-23) | `go test ./internal/api ./internal/config -count=1 -v` on the tests named Pairing and TrustedProxies: 7 api tests (4 existing pairing tests, 3 new origin tests) and 2 config tests pass. `gofmt -l` and `go vet` clean for `internal/config`, `internal/api`, `cmd/calango`. Not run behind a real reverse proxy |

## Usage endpoint (F1-T17 partial, F1-T13 partial, 2026-11-23)

Environment: Go 1.27.4 linux/amd64.

| Check | Result |
|---|---|
| `go test ./internal/usage ./internal/store ./internal/api -count=1 -run 'Usage\|Summary\|NewService'` | ok. Store: totals per agent with `to` excluding a later row; routine grouping with the empty key sorted first; unknown-cost rows counted, never summed as zero; another owner's rows never appear; invalid group gives `invalid_input`. Service: default group `agent` and owner passed through; bad group and `to` not after `from` rejected before the repository is read. HTTP: read scope 200 with `[]` (not `null`); approve_only 403; bad `group_by`, bad timestamp and inverted range 400 `invalid_input` |
| `go test ./... -count=1`, `go vet ./...`, `gofmt -l .` | ok |
| Redocly lint `api/openapi.yaml --max-problems=0` | valid, no warnings |
| Smoke: built `calango`, temporary `CALANGO_DATA_DIR`, `CALANGO_ADDR=127.0.0.1:7398` | `GET /v1/usage` with token 200 `[]`; `group_by=provider` 400; `to` before `from` 400; no token 401; `group_by=model` with `from` 200 `[]`; token file mode 600; process stopped |
| Limits | No known/estimated split in the response or the store query. No panel (desktop screens). Smoke returns empty lists because the binary has no HTTP route that creates usage; non-empty results are covered by store and service tests only. The `project` group always yields the empty key in V1 (no projects table) |

## Audit chain and AgenticOS attribution (2026-11-23)

Environment: Go 1.27.4 linux/amd64.

| Check | Result |
|---|---|
| `gofmt -l .` | ok, no files |
| `go vet ./...` | ok, no output |
| `go test ./... -count=1` | ok, 14 packages; `internal/audit` 8 tests, `internal/store` audit tests 4 (chain verifies, edit after dropping `audit_log_reject_update` is detected, truncation after dropping `audit_log_reject_delete` is detected, checkpoint rewind refused by trigger) |
| `git diff --check` | ok |
| Secret pattern scan over `internal`, `third_party`, `DEV`, `NOTICE`, `AGENTS.md` | no matches |
| Redocly lint | not run: no API contract change in this slice |

Pure cases in `internal/audit/chain_test.go`: edited entry, deleted middle entry (link break), truncated tail (checkpoint), legacy row without hash, key-order independence of the hash, malformed facts, empty chain. Reordering is caught by the link check; there is no dedicated reorder test.

Not verified: concurrent appends under load (`SQLITE_BUSY` on stale snapshots is by design and not exercised); the HTTP verify route (not built). The upstream scenarios were read from `verify_chain`, not run against the upstream Python tests.

## Vault and API token (F1-T10 partial, F1-T14 partial, 2026-11-23)

Environment: Go 1.27.4, linux/amd64; smoke data dirs under `/tmp`, removed afterwards.

| Check | Result |
|---|---|
| `gofmt -l .` | no files listed |
| `go vet ./...` | clean |
| `go test ./... -count=1` | 15 packages ok (includes `internal/vault`, `internal/auth`, `internal/config`, `internal/store`) |
| Cross-compile `./cmd/calango` | `windows/amd64`, `darwin/arm64`, `linux/arm64` build OK; the keyring backend compiles there but is not run |
| Smoke on the built binary, `CALANGO_VAULT_KEY` (random 32 bytes), legacy `api-token` file (0600) | First start logs "moved the api token … into the vault"; `/healthz` 200; legacy token on `/v1/agents` 200; wrong token 401; legacy file removed; token string not found in `calango.db`. Second start with the same key: token still 200 from the vault; no import line |
| `git diff --check` | clean |
| Secret pattern scan (`internal`, `third_party`, `cmd`, `DEV`, `NOTICE`, `AGENTS.md`) | no matches |

Limits:

- The OS keyring backend (`OSKeyring`) did not run on Windows, macOS or a Linux Secret Service. The smoke used `CALANGO_VAULT_KEY` on purpose, so no real entry was written to this machine's keyring.
- No vault audit events (Q28), no vault HTTP routes (Q29), no `api-token` rotation or revocation (Q30), no master-key rotation at startup (Q31). `LoadMasterKeys` supplies key version 1 only.
- Startup is stricter: without a master key the daemon does not start. Owner acknowledgement pending.

## Audit verify route and concurrent appends (F1-T18 partial, Q07, Q10, 2026-11-23)

| Check | Result |
|---|---|
| `go test ./internal/store -run TestConcurrentAuditAppends -count=3 -v` | PASS on 3 runs. Before the `_txlock=immediate` change the same test failed with `database is locked (SQLITE_BUSY)`. |
| `go test ./internal/api` (`audit_test.go`: first-break mapping, read scope, intact empty chain on a real store) | PASS |
| `gofmt -l .` | empty |
| `go vet ./...` | clean |
| `go test ./... -count=1` | all packages ok |
| `npx @redocly/cli@2.81.0 lint --extends minimal api/openapi.yaml --max-problems=0` | valid |

Limits:

- The HTTP break path uses a fake verifier; tamper cases over the real database are covered by `internal/audit` and `internal/store` tests, not over HTTP.
- `_txlock=immediate` applies to every store transaction. No write-latency measurement under load was taken.

## Vault audit events (F1-T10 partial, Q28, 2026-11-23)

| Check | Result |
|---|---|
| `go test ./internal/vault -count=1` | PASS, including `TestServiceAuditsEachActionWithNameOnly`, `TestServiceFailsClosedWhenAuditIsRefused`, `TestServiceRequiresAuditorAndIDSource` |
| `go test ./... -count=1` | 15 packages ok (`internal/auth` and `cmd/calango` build with the new `NewService` signature) |
| `gofmt -l .` | empty |
| `go vet ./...` | clean |

Limits:

- Tests use an in-memory audit recorder and a fixed id source. No event was checked in a real database file or over HTTP.
- In-process callers always record as actor `owner`.

## Vault HTTP routes (F1-T10 partial, Q29, 2026-11-23)

| Check | Result |
|---|---|
| `go test ./internal/api -run Vault -count=1 -v` | PASS, 7 tests: value absent from PUT, GET and audit rows; set replaces the same name; bad name and empty value 400 `invalid_input`; delete 204 then list `[]`, unknown delete 404; read and approve_only devices 403; `api-token` PUT and DELETE 409 `conflict`; `vault.set` and `vault.delete` audit rows with subject `vault:github` and no value |
| `go test ./... -count=1` | 15 packages ok |
| `gofmt -l .` | empty |
| `go vet ./...` | clean |
| `npx @redocly/cli@2.81.0 lint --extends minimal api/openapi.yaml --max-problems=0` | valid |
| `git diff --check` | clean |
| Forbidden-name scan (ADR 0006) over `DEV/`, `README.md`, `AGENTS.md`, `NOTICE` | no matches |

Limits:

- The routes were tested on the in-memory store (`OpenMemory`), not a database file. The binary was not started for this slice; the earlier smoke test predates the routes.
- No route returns a value and `Reveal` is not on `VaultStore`, so reveal is not reachable over HTTP. This is by design until a consumer needs it.
- `api-token` cannot be set or deleted through these routes until rotation (Q30) exists.

## Token rotation (F1-T14, Q30, 2026-11-23)

| Check | Result |
|---|---|
| `gofmt -l .` | clean |
| `go vet ./...` | clean |
| `go test ./internal/auth -count=1` | ok; includes `TestLocalTokenRotationSurvivesARestart`, `TestLocalTokenKeepsOldTokenWhenVaultWriteFails` |
| `go test ./internal/api -count=1` | ok; includes `TestRotateTokenRetiresTheOldToken`, `TestRotateTokenNeedsFullScope`, audit verify test using `newTestLocalToken` |
| `go test ./... -count=1` | 15 packages ok |
| `npx @redocly/cli@2.81.0 lint --extends minimal api/openapi.yaml --max-problems=0` | valid |
| Binary smoke: built `cmd/calango`, legacy `api-token` file imported, `POST /v1/auth/rotate-token` 200, old token 401, new token 200, both after a restart; no token in the logs | as stated |

Limits:

- The fail-closed test uses a closed store, so the vault write fails before the swap. It does not inject a failure in the middle of the write.
- A lost rotation response leaves the client with no token; no V1 route recovers it. Accepted by the owner on 2026-11-23.
- The smoke ran on Linux only; the keyring backends were not involved (key file used).

## Usage known/estimated split (F1-T17, Q18, 2026-11-23)

- `gofmt -l` on `internal/usage`, `internal/store`, `internal/api`: empty.
- `go vet ./...`: clean.
- `go test` on the usage, store and API packages, filter `Usage`: PASS.
- `go test ./... -count=1`: 15 packages ok.
- Redocly lint `api/openapi.yaml` (minimal, max-problems 0): valid.
- Limit: the split is covered by store and API tests only; the built binary was not smoke-tested after this change. The desktop panel is not built.

## Cross-compile after Q29 and Q30 (2026-11-23)

- `CGO_ENABLED=0 GOOS/GOARCH go build ./cmd/calango`: OK for windows/amd64, darwin/arm64, darwin/amd64, linux/amd64.
- Limit: a successful build does not run the keyring backends. The Windows Credential Manager, macOS Keychain and Linux Secret Service paths have not been exercised at runtime (Q13).

## Push and CI for commit dee8efa (2026-11-23)

- Gate before commit: `go vet ./...` clean; `go test ./... -count=1` 15 packages ok; `git status` clean after commit.
- Pushed `main` to `origin` (`6c62ceb..dee8efa`) on the owner's explicit request.
- GitHub CI run 45188079827 on `dee8efa`: success (build jobs for each target).

## Audit export and master-key rotation (Q08, Q31, 2026-11-23)

- Checks: `gofmt -l .` clean; `go vet ./...` clean; `go test ./... -count=1` 15 packages ok; `git diff --check` clean; Redocly lint of `api/openapi.yaml` (`minimal`, `--max-problems=0`) valid.
- Tests: `internal/config/vault_rotation_test.go` (7: settings read; default version 1; malformed version; both PREVIOUS sources; PREVIOUS without current; PREVIOUS at version 1; version 2 without explicit key); `internal/vault/bootstrap_test.go` (previous key opens the old version during rotation; PREVIOUS needs version 2; a version needs an explicit key); `internal/vault/rotate_audit_test.go` (each re-wrap audited with name only; nothing stored when the audit fails); `internal/store/audit_export_test.go`; `internal/api/audit_export_test.go` (4 tests).
- Throwaway probe on real SQLite (deleted after the run): set under v1, `Rotate` with v2 and previous v1, reveal ok, stored key version 2, one `vault.rewrap` row; v2-only `Rotate` = 0; v1-only refused with `vault: rotate "old-key": vault: no master key configured for this version: 1`.
- Binary smoke (`go build ./cmd/calango`, temporary data dir and key files, removed afterward), `127.0.0.1:7399`:
  1. v1 start: legacy `api-token` moved into the vault; `PUT /v1/vault/smoke-secret` → 200.
  2. v2 current, v1 previous: `vault: re-wrapped 2 secrets under master key version 2`; `GET /v1/vault` → 200.
  3. v2 only: no rewrap log; `GET /v1/vault` → 200; `GET /v1/audit/verify` → `{"ok":true,"entries_checked":8,"first_break":null}`; JSONL export has 2 `vault.rewrap` lines; CSV header and rows as specified.
  4. v1 only after rotation: exit 1, nothing served: `vault key rotation stopped after 0 secrets: vault: rotate "api-token": vault: no master key configured for this version: 2`.
  5. Grep of the logs for the secret value and the token: no match.
- Not checked: the export's first entry is a `vault.reveal` on `api-token` written at first start (legacy import); the import path was not inspected in this step. Docker not run; keyring backends not run on Windows or macOS.
- Limits: the export is buffered in memory; CSV has no checkpoint line, so tail truncation shows only in JSONL; CSV fields are not neutralised against spreadsheet formulas (fixed prefixes and JSON facts; low risk); the export itself is not audited.

## Codex CLI harness (F1-T05 partial, 2026-11-23)

- Tests with a fake `codex` shell script in `t.TempDir()` (no real Codex run; owner did not authorise a model call): `internal/harness/codex_test.go` (streams the final message and estimates usage; isolation flags present and the prompt absent from argv and sent on stdin; signed out → `ErrSignedOut` with `exec` not called; not installed → `ErrNotInstalled`; failing exec streams nothing and does not echo CLI output; cancellation returns `DeadlineExceeded` quickly; `NewCodex` requires a working directory) and `internal/harness/codex_gateway_test.go` (through `gateway.Call`: tokens recorded as estimated, cost unknown with no price, signed-out error reaches the caller).
- Finding: the cancellation test ran 30 s (grandchild `sleep` held the stdout pipe). Fixed with `cmd.WaitDelay = 2 * time.Second`; the test now takes about 2.3 s.
- Checks: `gofmt -l .` clean; `go vet ./...` clean; `go test ./... -count=1` 16 packages ok (harness and conversations among them); `git diff --check` clean.
- Not checked: a real `codex exec` run; the `--json` event schema; whether `~/.codex/AGENTS.md` loads under `--ignore-user-config`; the sandbox's effect on a real workspace.

## Maestro npm probe (F1-T16 prep, 2026-11-23)

- Isolated install with `--home-path` in a temp dir: `install --core-only` (exit 0), then `install --only codex` (exit 0). `verify --home-path` exit 0 with 20 Codex agents, 20 Codex prompts and 58 skills. The temp `HOME` received only `.config/orquestrador-maestro/telemetry.json`.
- Reproduced: `verify` after `--core-only` exits 1 with `Codex agents missing` and `Codex prompts missing`.
- Read-only commands with a temp `HOME`: `status --json`, `version`, `update --dry-run`, `rollback --list`.
- Caveat: the first `install --core-only` ran with the real `HOME`. Files that changed in that window under `~/.codex`, `~/.local/share` and `~/.local/state` were not attributed to Maestro. The later windows, run with a temp `HOME`, showed no writes to the real `HOME` from Maestro.
- Not checked: real install in the user's home; desktop plugin; skill sync; behaviour on Node below 20.19 (engines `>=20.19.0`; this machine has Node 22.23.4).
- Applied (temp `HOME`, `--home-path`, `ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1`): `update --only codex --non-interactive --skip-desktop-plugin --skip-community-skills --skip-skill-sync` exit 0; `verify` exit 0; `rollback --to <stamp>` exit 0; `verify` exit 0 after rollback.
- Failed without the skip flag: `update` ran `npm install -g @iapro/orquestrador-maestro-cli@latest --force`, exit 243 (EACCES renaming the root-owned `/usr/local` install). Global dirs keep their pre-probe mtimes (2026-10-31 and 2026-11-21).
- Failed: `version --check` exit 1, "npm retornou uma versão inválida" (npm returns `["1.1.1"]` for `view <pkg>@latest version --json`).

## 2026-11-23 — Conversations routes and daemon wiring (Q22)

- `gofmt -l cmd internal` empty; `go vet ./...` clean; `go test ./... -count=1` all packages ok, including `internal/api` conversations tests (`TestConversationsCreateListAndSendMessage`, `TestConversationRunStopAndRetry`).
- Smoke on the built daemon: `CALANGO_DATA_DIR=<tmp> CALANGO_ADDR=127.0.0.1:7399 CALANGO_VAULT_KEY=<base64>`; log `calango listening on 127.0.0.1:7399`; `GET /healthz` → `{"status":"ok"}`; `<tmp>/workspace` created.
- Not checked: `POST /v1/conversations/{id}/messages` with the real Codex CLI (no owner authorisation for a model call); `api/openapi.yaml` was checked only as YAML (python yaml parse), not by the repo's OpenAPI lint; steer is absent by design.
## 2026-11-23 — WebSocket `/v1/events` (Q23)

- Checks:
  - `gofmt -l cmd internal` empty; `go vet ./...` clean; `go test ./... -count=1` all 17 packages ok; `go test -race ./internal/events ./internal/api -run 'Events|Conversation'` ok.
  - `internal/events/hub_test.go`: owner filter, slow subscriber dropped, idempotent Close. `internal/api/events_test.go`: no token gives 401 on the upgrade; with the token, `message.created`, then `run.started` (status `running`), then `run.finished` (status `stopped`) after `POST /v1/runs/{id}/stop`.
  - Daemon smoke on the built binary: `CALANGO_ADDR=127.0.0.1:7398`; log `calango listening on 127.0.0.1:7398 (api token in the vault)`; `/healthz` 200 `{"status":"ok"}`; upgrade request without bearer returns 401.
  - Not checked: a real WebSocket client from Electron or mobile; `run.step` payload has no test of its own (only the publish path); slow-client close over a real network; a real Codex run emitting steps; the repo's OpenAPI lint.

## 2026-11-23 — Direct Anthropic provider (F1-T04)

- `gofmt -l cmd internal` empty; `go vet ./...` clean; `go test ./... -count=1` all 18 packages ok.
- `internal/anthropic/anthropic_test.go` against an `httptest` server (no real API call): streamed text and exact tokens; request headers (`x-api-key` from vault, `anthropic-version`) and body (`stream`, `model`, `max_tokens`, prompt); HTTP 401 error without the key; stream error keeps partial text; stream cut before `message_stop` is an error; missing key makes no request; bad base URL rejected; `gateway.Call` records provider/model/tokens in the ledger with unknown cost.
- Daemon smoke: build OK; `/healthz` 200; `/v1/conversations` without token 401; log `calango listening on 127.0.0.1:7398`.
- Not checked: a real Anthropic call (no owner authorisation); a real vault key flow through `Reveal` with the audit row; agents reaching this provider (not wired).

## 2026-11-23 — Provider chain per agent (Q32, ADR 0016)

- `gofmt -l cmd internal` empty; `go vet ./...` clean; `go test ./... -count=1` all 18 test packages ok (`cmd/calango` and `internal/providers` have no test files).
- Migration: `internal/store/provider_chain_migration_test.go` applies 0001–0009, seeds three legacy agents, applies 0010; asserts legacy columns gone, chains per agent, empty-model agent has no chain, three `default` accounts.
- Fallback: `internal/conversations/chain_test.go`: falls back to the next account before output; no fallback once output streamed; budget exhaustion never falls back.
- Accounts API: `internal/api/provider_accounts_test.go`: create 201 with key `anthropic/work` and setup text; duplicate 409; list shows defaults and `work`; account used by an agent → delete 409; spare account delete 204, then 404.
- Codex home: `internal/harness/codex_test.go` (fake `codex` records `CODEX_HOME`): account home overrides an inherited `CODEX_HOME`; `default` keeps the inherited value.
- Events: `internal/api/events_test.go` skips `run.step` frames before `run.finished` (a stopped run keeps its cancelled attempt step).
- OpenAPI: `api/openapi.yaml` parses (python yaml).
- Daemon smoke on the built binary (`go run ./cmd/calango`, `CALANGO_ADDR=127.0.0.1:7398`, fresh data dir): `/healthz` 200 `{"status":"ok"}`; `/v1/provider-accounts` 401 and `/v1/events` 401 without token; process stopped.
- Stale throwaway data dir `/tmp/calango-smoke-data` fails to start with a vault key mismatch; not used, not deleted.
- Not checked: a real Codex run or `codex login` with `CODEX_HOME`; a real Anthropic call (no owner authorisation); migration on a real legacy database file (the test seeds a fixture); Windows and macOS keyring.

## 2026-11-23 — Steer (Q22)

- Checks:
  - `gofmt -l cmd internal` empty; `go vet ./...` clean; `go test ./... -count=1` all 18 test packages ok.
  - `internal/conversations/steer_test.go`: queued steer stops the queued run and runs a continuation after it, whose prompt has the body and the guidance; running steer keeps the partial reply in the continuation prompt, a repeated key returns the same continuation, a different pending guidance is `ErrConflict`; steer on a finished run is `ErrConflict`, empty guidance `ErrInvalidInput`.
  - `internal/api/conversations_test.go`: steer without `Idempotency-Key` 400; 202 with `steer_of` and instructions; repeat returns the same run; unknown run 404.
  - Daemon smoke on the built binary (`CALANGO_ADDR=127.0.0.1:7398`, fresh data dir, `CALANGO_VAULT_KEY` unset): `/healthz` 200; `POST /v1/runs/run_x/steer` 401 and `/v1/events` 401 without token.
- Not checked: a real model call (no owner authorisation); the repo's OpenAPI lint; the steer route with a valid token on the daemon (needs an agent and a run; covered by the API test instead).

## 2026-11-23 — Routines fire agent runs (Q33, ADR 0017)

- Checks:
  - `gofmt -l cmd internal` empty; `go vet ./...` clean; `go test ./... -count=1` all 18 test packages ok (`cmd/calango` and `internal/providers` have no test files).
  - `internal/routines/control_test.go`: finish ends the routine and stops future occurrences; an ended routine rejects pause, run now and update with `ErrConflict`; finish on a failed attempt is ignored; the conversation survives a failed attempt; run now fires once as a manual occurrence on the next tick; a schedule update recomputes the next run.
  - `internal/routines/executor_test.go` (fake `Chat`): one conversation per routine; each attempt keyed `<OccurrenceID>#<Attempt>`; the control line is read; a missing control line defaults to news and keeps the routine alive; a failed run keeps the conversation for the retry.
  - `internal/api/routines_test.go`: route status codes and validation.
  - Daemon smoke on the built binary (`env -u CALANGO_VAULT_KEY`, `CALANGO_ADDR=127.0.0.1:7398`, fresh data dir, key file): `/healthz` 200; `GET /v1/routines` and `GET /v1/routines/x/runs` 401 without token; daemon stopped by PID.
- Not checked: a real model call from a routine (no owner authorisation); the repo's OpenAPI lint (Redocly); a routine run that waits for approval (Q15); VPS execution (`run_on=vps`); the `routine.fired` event over a real WebSocket client; restart while a run is in flight.

## 2026-11-23 — Maestro validation (Q24)

- Checks (npm 1.1.1, `/tmp/maestro-calango-probe`, `ORQUESTRADOR_MAESTRO_TELEMETRY=0`, `XDG_CONFIG_HOME=<home>/config`):
  - `npm install --prefix versions/1.1.1 @iapro/orquestrador-maestro-cli@1.1.1`: exit 0.
  - `install --home-path H --only codex --non-interactive`: exit 0. `verify --home-path H`: exit 0.
  - Marker check on the real `$HOME` (`find -newer`, npm cache excluded): nothing changed.
  - `ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1 update --home-path H --only codex --non-interactive`: exit 0; `verify`: exit 0.
  - `rollback --home-path H --list`: exit 0, one stamp. `status --json --project-path H`: exit 0, `jq -e .` valid.
- Not checked: `rollback --to` in this run (only in the earlier probe); Codex with `CODEX_HOME` loading the agents (needs a real session; not authorised); `version --check` (known broken). No Calango code path exists yet.

## 2026-11-23 — Approvals run ownership (Q15 slice)

- Checks: `gofmt -l internal/approvals` empty; `go vet ./internal/approvals/` clean; `go test ./internal/approvals/ -count=1` ok; `go test ./internal/api/ -count=1` ok.
- Not checked: a live approval flow over HTTP with a second agent's run; the owner-scope test for `GetRun` (it has no owner argument; `GetAgent` enforces owner before the run check).

## 2026-11-23 — Go minimum 1.23

- Checks (local toolchain go1.27.4): `gofmt -l cmd internal` empty; `go vet ./...` clean; `go test ./... -count=1` all 18 test packages ok.
- Not checked: a Go 1.23 toolchain itself, the CI jobs on `actions/setup-go` with `1.23`, and the `golang:1.23-alpine` image build. Those run in CI or Docker, not here.

## 2026-11-23 — MCP gateway per turn (Q20, F1-T12)

- Checks:
  - `gofmt -l cmd internal` empty; `go vet ./cmd/... ./internal/...` clean.
  - `go test ./... -count=1 -timeout 120s`: all 19 test packages ok (`cmd/calango` and `internal/providers` have no test files). One failure found and fixed: `internal/store` expected table list lacked `agent_tools` and `connectors`.
  - `go test -race` on `./internal/mcpgw ./internal/tools ./internal/api ./internal/harness`: ok.
  - `internal/tools/catalog_test.go`: `ReadOnly` from `readOnlyHint`; unreachable connector → `ErrUnavailable`; catalog owner-scoped (`ErrNotFound`).
  - `internal/mcpgw/gateway_test.go` (7 tests): token scope, approval gating, execution from snapshot.
  - `internal/harness/codex_test.go`: token appears only in the child environment, never in argv.
  - Daemon smoke on built binary (`env -u CALANGO_VAULT_KEY`, `CALANGO_ADDR=127.0.0.1:7399`, fresh data dir, key file): `/healthz` 200; `POST /mcp` with no token 401 `unauthorized`; `POST /mcp` with `Bearer abc` 401; `GET /v1/connectors` without token 401. Daemon stopped by PID.
- Not checked: an authenticated `/v1/connectors` round trip on the daemon (the local token is in the vault; not read for this smoke); a real Codex 0.171 run with `sandbox: workspace-write` reaching the loopback gateway and using the `mcp_servers` keys (needs owner authorisation for a model call); the OpenAPI lint (Redocly); Go 1.23 toolchain; CI.

## 2026-11-23 — Waiting approval and reconciliation (Q15, F1-T08)

- Checks:
  - `gofmt -l cmd internal` empty; `go vet ./...` clean; `go test ./... -count=1 -timeout 240s`: all 20 test packages ok (`cmd/calango` and `internal/providers` have no test files).
  - `go test -race` on `./internal/conversations ./internal/routines` with the WaitRun, Stop, retry, routine and executor tests: ok.
  - `internal/store/run_status_test.go`: a waiting run keeps its slot; a repeated move from running is a no-op; `waiting_approval` → `failed` is refused; interrupt ends a waiting run as failed and frees its slot.
  - `internal/mcpgw/gateway_test.go`: `TestWriteHoldsRunWhileWaitingForDecision` (waiting_approval while the write waits, running after approval); the timeout test checks the run is running again.
  - `internal/approvals/service_test.go`: `TestReconcileClosesApprovedWithoutReceiptAndNeverRetries` (second reconcile closes nothing; execute after reconcile is `ErrConflict`; one audit entry).
  - `internal/conversations/service_test.go`: `TestWaitRunHoldsWhileRunWaitsForApproval` (WaitRun does not return while waiting; returns done after the run resumes and finishes).
- Not checked: a live write approval over HTTP while a Codex run waits (needs owner authorisation for a model call); the approval card; stopping a run in `waiting_approval`; a daemon restart while a write waits; the mcpgw test and the reconcile test on Go 1.23 (local toolchain go1.27.4 only).

## 2026-11-23 — memory_save through the gateway (Q17, F1-T11)

- Checks:
  - `gofmt -l cmd internal` empty; `go vet ./...` clean; `go test ./... -count=1`: all 20 test packages ok (`cmd/calango` and `internal/providers` have no test files).
  - `go test ./internal/mcpgw/ ./internal/conversations/ ./internal/memory/ -count=1`: ok, run before the full suite.
- Not checked: a real Codex run calling `memory_save`; an authenticated daemon round trip; a saved memory appearing in the next run's prompt over MCP; the Go 1.23 toolchain; no REST route was added, so the OpenAPI lint has nothing new to check.

## 2026-11-23 — Desktop chat and workflow smoke

- `gofmt -d internal/conversations/service.go internal/conversations/service_test.go internal/store/conversations.go`: no diff.
- `go test ./internal/conversations ./internal/api -count=1`: both packages passed.
- `go build -o /tmp/calango-v1-smoke/bin/calango ./cmd/calango` and `go build -o bin/calango ./cmd/calango`: passed.
- `npm run typecheck -w apps/desktop` and `npm run build -w apps/desktop`: passed.
- `node scripts/generate-i18n.mjs --check`: passed; generated translations are current.
- Isolated Electron E2E on Linux, fresh `/tmp` data/profile and fake Codex executable: agent creation, visible pause/resume, conversation creation and displayed fake response, routine creation and pause/resume, and usage row all passed. The usage row reported 8 input tokens, 6 output tokens and one unknown-cost row.
- Isolated Chromium renderer smoke with mocked IPC/API: a simulated waiting run displayed the pending action/target; Approve, Always allow, and Decline each dispatched to the mock decision endpoint with HTTP 200. This verifies renderer wiring only, not the daemon approval path or action execution.
- Confirmed the daemon child's `codex` resolves to the fake executable after npm's `node_modules/.bin` entries are removed from `PATH`.
- No real model call, actual provider cost, Windows install, or real approval request/decision/action was exercised. The isolated profile/data directory was used; the default `~/.config/calango` database was not touched.
- Restarted Electron with the same isolated profile/data directory; `/healthz` returned 200 on `127.0.0.1:7317`; CDP port `9222` remained closed.

## 2026-11-23 — V1 TODO continuation

- `go test -race ./... -count=1 -timeout 300s && go vet ./...`: passed across all Go packages; vet clean.
- `npm run typecheck -w apps/desktop && node scripts/generate-i18n.mjs --check && npm run build -w apps/desktop`: passed. Vite transformed 43 modules and built the renderer. The generated tray catalog was current.
- `npm run test:main -w apps/desktop`: 2/2 tests passed (XDG path/fallback; autostart file create/remove and escaped command entry).
- Isolated Electron smoke with `/tmp` profile/data/key, a loopback-only daemon and isolated `XDG_CONFIG_HOME`: app opened; Settings showed no secret values; enabling startup created `autostart/calango.desktop`, disabling it removed the file, and no UI error remained.
- Isolated audit UI smoke: chain verified (`ok`, 23 entries); JSONL and CSV downloads completed (8,602 and 5,815 bytes).
- Site review at 1440×1000 desktop and 390×844 mobile, light/dark: `scrollWidth` matched viewport (390 px mobile, 1440 px desktop); no mobile horizontal overflow. Plan cards list phone/cloud/team capabilities still on the roadmap; owner copy review remains open.
- `npx @redocly/cli@2.81.0 lint --extends minimal --max-problems=0 api/openapi.yaml`: valid; one non-blocking warning.
- Not exercised: real model/provider cost, live external approval action, physical OS shortcut activation, tray-menu click, Windows/macOS runtime, Windows signing/install, Docker bridge. No production data or user Codex login used.

## 2026-11-23 — TODO status reconciliation

- `go test -race ./... -count=1 -timeout 300s` and `go vet ./...` passed earlier in this work session; no source changed after those checks.
- `npm run typecheck -w apps/desktop`, `node scripts/generate-i18n.mjs --check`, `npm run build -w apps/desktop`, and `npm run test:main -w apps/desktop` passed earlier; no source changed after those checks.
- `codex exec --help` ran locally without a model call. It confirms `--json` emits JSONL but does not document the event/usage schema; F1-T05 remains partial.
- Repository search found no Calango telemetry client/sender; `internal/maestro` sets `ORQUESTRADOR_MAESTRO_TELEMETRY=0` and private `XDG_CONFIG_HOME` for the CLI.
- The latest Redocly lint is valid with one non-blocking warning. GitHub run 45188079827 passed on `dee8efa`; it does not include current uncommitted changes.
- No real model call, external approval action, Docker image run, Windows/macOS runtime test, commit or push was performed.
- `go test ./internal/store -run TestPreMigrationSnapshotRestoresAndAppliesPendingMigration -count=1`: passed; `gofmt -d internal/store/backup_restore_test.go` produced no output.
- The new test applies all but the latest migration, seeds an agent, snapshots the closed database, applies the migration, restores the snapshot, then verifies the record after the current binary reapplies the migration. It does not test a previous binary or OS keyring recovery.

## 2026-11-23 — Desktop launch, checks and push

- `gofmt -l cmd internal`: empty after `gofmt -w internal/api/server.go`.
- `go vet ./...`: exit 0.
- `go test ./... -count=1`: 21 packages `ok`; `cmd/calango` and `internal/providers` have no test files.
- `npm run typecheck -w apps/desktop`: exit 0. `npm run build -w apps/desktop`: exit 0 (`bin/calango` built; Vite transformed 43 modules; `index-DAGH1Jkq.js` 242.19 kB, `index-CdleRBvq.css` 5.52 kB).
- `node --test apps/desktop/test/startup.test.cjs`: 2/2 pass.
- Daemon kill recovery (earlier this session): SIGKILL on the daemon; supervisor restarted it as PID 208776; `listening` logged at 15:05:15; `/healthz` 200 after restart.
- Desktop relaunch: `DISPLAY=:1 npm run desktop:start`. Log: `calango listening on 127.0.0.1:7399 (api token in the vault)`; `curl 127.0.0.1:7399/healthz` returned `{"status":"ok"}` (HTTP 200). Electron started under Wayland and logged `'--ozone-platform=wayland' is not compatible with Vulkan` (GPU warning). Log also shows `maestro automatic update failed` at start; cause not investigated.
- CDP was not enabled in this launch.
- Not verified: window visibility on the owner's screen (no screenshot or window-list tool on this machine); real model call; Windows; real approval action; Maestro update failure cause.

## 2026-11-23 — F1-T16 stable Codex home checks

- `gofmt -l internal/maestro`: empty. `go vet ./internal/maestro/... ./internal/conversations/... ./cmd/calango/...`: exit 0. `go build ./...`: exit 0.
- `go test ./internal/maestro/... ./internal/conversations/... ./cmd/calango/... -count=1`: `internal/maestro` ok, `internal/conversations` ok, `cmd/calango` has no test files.
- Throwaway CLI probe in `/tmp/mtest` (not in the repo; `@iapro/orquestrador-maestro-cli` 1.1.0 and 1.1.1 installed in `/tmp`, no owner files touched): reinstall into the same home and version swaps 1.1.0 → 1.1.1 → 1.1.0 kept `.codex/auth.json`; `verify` exit 0 each time; 38 skills; `~/.codex/AGENTS.md` mtime unchanged.
- Not exercised: a real Codex session on the stable home; installed agents and skills loading under it; the owner's `~/.codex`; the migration of earlier per-version logins.
- Changes are uncommitted in `internal/maestro/manager.go`, `internal/maestro/manager_test.go`, and the docs above. Not committed or pushed: AGENTS.md requires an explicit owner request.

## 2026-11-25 — F4-T03 integrated terminal (issue #84)

Environment: Go 1.23.12 linux/amd64, Node 24.22.1, Electron 45.4.1 (Node 24.22.3).

| Check | Result |
|---|---|
| `go test -race -count=1 ./internal/pty/` | ok: echo and cwd, environment without `CALANGO_VAULT_KEY`/`GITHUB_TOKEN` canaries, resize (`stty size`), cwd outside workspace and symlink escape refused, bad sizes, limit, owner/device scoping and busy, scrollback replay, terminate kills shell and background job, shell `exit 3` ends session and kills leftover job, `Close` kills all and refuses new, idle sweep only for detached sessions, start policy refusal |
| `go test -race -count=1 -run PTY ./internal/api/` | ok: 401 without/with wrong token (REST and WebSocket), 403 for read scope, 404 for another device, echo and resize over WebSocket, 409 second client, 400 invalid input, 429 limit, DELETE sends exit frame + close 1000 and kills background job, shell exit code in exit frame, malformed control frame closes 1003 and the session survives |
| `go test ./... -count=1`, `go vet ./...`, `gofmt -l cmd internal` | ok, ok, empty |
| `GOOS=windows go vet ./...`, `GOOS=darwin go vet ./...`; builds windows/darwin amd64+arm64, linux/arm64 | ok |
| staticcheck 2024.1.1 on `internal/pty`, `internal/api`, `cmd` | ok |
| `npm run contracts:check`; Redocly lint | ok; valid, 2 warnings `operation-2xx-response` (WebSocket routes answer 101, same as `/v1/events`) |
| `npm run typecheck`, `npm run test:main -w apps/desktop`, `npm run desktop:build`, Biome format/lint | ok (6/6 tests); terminal screen in its own chunk |
| Isolated daemon smoke (temporary data dir, port 7391) | `PtyBridge` from Electron's Node: create, attach with header, resize 120x40, `CALANGO_VAULT_KEY` empty in shell, `exit 4` → exit frame; audit shows `terminal.opened`/`terminal.closed`; POST without token 401; SIGTERM to the daemon killed the shell |
| Isolated Electron under Xvfb (temporary profile and data dir, port 7392) | Terminal screen: new session, typed command output rendered, fit size sent (31x119), end session; no console or CSP errors |
| Not verified | Windows and macOS at runtime; packaged app |
