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
| Limits | Rate limit keys on the TCP `RemoteAddr`: behind a reverse proxy (VPS deployment) all clients share one bucket. No QR code, no mobile app, no administrator password on the VPS. Desktop token is a file, not the vault (F1-T10), and has no rotation. Owner is single (`owner`). Device tokens have no expiry or rotation |

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
- A lost rotation response leaves the client with no token; no V1 route recovers it. Owner decision pending.
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
