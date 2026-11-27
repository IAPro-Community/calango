# Deployment — Calango on your VPS

Status: design. On 2026-11-25, the site and node images built and passed `/healthz` over a user-defined Docker bridge. The node database persisted in a named volume across container recreation. This smoke does not validate the Caddy/TLS Compose deployment, VPS security, or backup/restore.

## Model

```
Person's VPS                           clients
┌──────────────────────────┐          ┌─────────────┐
│ docker: calango          │◄──TLS────│ desktop     │
│  - core, routines        │          │ iOS         │
│  - browser (chromium)    │          │ Android     │
│  - data (SQLite in vol)  │          │ web         │
└──────────────────────────┘          └─────────────┘
```

The server is the only place where the agent runs, stores data and executes routines. Clients are remote controls: they do not run the agent; they only show state and send commands. That is why the agent stays alive when the laptop is off, and the phone works from anywhere.

It also works with the server on the computer itself (local mode). Same binary; only the address changes.

## Pairing

1. The person starts the container on the VPS.
2. The server shows a single-use pairing code (QR or text), valid for a few minutes.
3. The app reads the code, exchanges it for a device token and stores the token in the system keychain.
4. Each device can be revoked separately.

## Secure access (required)

- **Never expose the port in plain text.** Everything goes through TLS, via a reverse proxy (Caddy or Traefik with automatic certificates) or over a private network.
- **Behind a reverse proxy**, set `CALANGO_TRUSTED_PROXIES` to the proxy's address. Without it, every client behind the proxy shares one pairing rate-limit bucket.
- **Alternative without a public port**: a private network (Tailscale or WireGuard). Recommended as the default for anyone who does not want to expose the VPS.
- Strong administrator password on first access; no default credentials.
- Device tokens with scope and expiry; immediate revocation.
- Secrets (model keys, connector tokens) encrypted at rest, with the key kept outside the data volume.

## Push notifications (mobile)

A self-hosted server cannot talk directly to APNs (iOS) or FCM (Android) without Apple and Google credentials. Proposal: a small **push relay**, hosted by us, which receives a notice from the person's server and forwards it to their phone. The relay sees no content: it only sends "there is news", and the app fetches the content from the person's server. This is also a monetization point (see [MONETIZATION.md](MONETIZATION.md)).

## Updates

- Versioned image (`calango/calango:<version>`). Updating means changing the tag and restarting.
- Database migrations run at startup and have no down-migrations. Take an offline snapshot before upgrading.
- The local-node backup procedure, vault-key requirements and restore-before-rollback policy are in [BACKUP_RESTORE.md](BACKUP_RESTORE.md). Docker volume backup/restore remains unvalidated.

## Managed hosting (paid option)

For those who do not want to run a VPS, offer the same server hosted by us, with the same APIs. This corresponds to the Pro/Team plan in [MONETIZATION.md](MONETIZATION.md). The self-hosted version remains available, so the product does not depend on us to exist.

## Files

- `deploy/Dockerfile.node`: two-stage node build, minimal final image, non-root user. Site: `Dockerfile` at the root, nginx non-root.
- `deploy/docker-compose.yml`: example with a data volume and a TLS proxy.

## Container smoke validation

The site and node images were built and run on a disposable bridge network. The node image defaults `CALANGO_DATA_DIR` to `/data`; its directory is owned by UID/GID 65532 for the `nonroot` user. Named volumes preserve this ownership on first mount. Bind-mounted host directories must also be writable by UID/GID 65532.

Reproduce the smoke with Docker Engine:

```sh
docker build -t calango-site:smoke .
docker build -f deploy/Dockerfile.node -t calango-node:smoke .
docker network create --driver bridge calango-smoke
docker volume create calango-smoke-data
umask 077
printf 'CALANGO_VAULT_KEY=%s\n' "$(openssl rand -base64 32 | tr -d '\n')" > /tmp/calango-smoke.env
docker run -d --name calango-smoke-node --network calango-smoke \
  -p 127.0.0.1:17317:7317 -v calango-smoke-data:/data \
  --env-file /tmp/calango-smoke.env calango-node:smoke
docker run -d --name calango-smoke-site --network calango-smoke \
  -p 127.0.0.1:18088:8080 calango-site:smoke
curl --fail http://127.0.0.1:18088/healthz
curl --fail http://127.0.0.1:17317/healthz
docker rm -f calango-smoke-node
docker run -d --name calango-smoke-node --network calango-smoke \
  -p 127.0.0.1:17317:7317 -v calango-smoke-data:/data \
  --env-file /tmp/calango-smoke.env calango-node:smoke
curl --fail http://127.0.0.1:17317/healthz
docker rm -f calango-smoke-node calango-smoke-site
docker volume rm calango-smoke-data
docker network rm calango-smoke
rm -f /tmp/calango-smoke.env
```

The vault key must decode to 32 bytes; use a disposable key only. This manual smoke does not exercise `deploy/docker-compose.yml`, Caddy/TLS, or a public VPS.

## Pending

- Local pairing and device-token APIs are implemented; QR/mobile and VPS administrator-password flows remain pending (phases 2–3).
- Push relay: to be defined.
- Browser in the container (chromium) requires a larger image; evaluate a separate image.
