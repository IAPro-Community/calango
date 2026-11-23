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
| i18n catalog generation/drift | planned in F1-T26, together with the catalog creation |
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
