# Deployment — Calango on your VPS

Status: design. The Dockerfiles were built on 2026-11-22 and the health checks passed with `--network host` (see `VERIFY.md`). Port mapping through bridge networking remains unvalidated. In this session the sandbox blocked access to the Docker daemon, so the images were not revalidated.

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

## Pending

- Local pairing and device-token APIs are implemented; QR/mobile and VPS administrator-password flows remain pending (phases 2–3).
- Push relay: to be defined.
- Browser in the container (chromium) requires a larger image; evaluate a separate image.
