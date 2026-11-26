# Security

## Assets

Model and connector keys, PC files, browser sessions, project code, conversations and memory, control of the PC screen.

## Threat model

| Threat | Defense | Where |
|---|---|---|
| Someone on the network accesses the node | every route requires a device token, including on the LAN; Tailscale; TLS on the VPS | `pairing-auth` |
| Stolen device token | scoped, time-limited token; per-device revocation; visible device list | `pairing-auth` |
| Agent runs something destructive | mandatory approval per action type; restricted working folder; Docker on the VPS | `approvals`, `tools-mcp` |
| Prompt injection (web, email, file) | external content enters marked as data; write tools always go through approval; the agent cannot approve itself | `harness`, `approvals` |
| Key leakage | operating system vault; the agent uses keys by reference and never sees the value; masking in logs and traces | `vault` |
| Uncontrolled spending | budget per agent and per routine, checked before each call | `gateway` |
| Integrated terminal abused (arbitrary commands as the node's user) | write scope and bearer header on every route, session bound to the creating device, unguessable id, shell chosen by the daemon, working folder limited to the daemon workspace, allowlisted environment (no `CALANGO_*` or tokens), 8-session limit, idle timeout, whole process tree killed on end and on shutdown, open/close audited without content | `ade` (`internal/pty`, ADR 0023) |
| Remote screen control abused | off by default; can only be turned on at the PC itself; visible notice while active; audit; ends with one click | `computer-use` |
| Forged license on the VPS node | Ed25519-signed key, validated offline; downgrades to free, never shuts down | `licensing-billing` |
| Push leaking content | relay only carries "there is news" + an opaque id | `push` |
| Malicious update | signed installers; Docker image pinned by digest; signed update channel | `desktop-electron`, `cloud` |

## Rules

1. No route without authentication, except `/healthz` and the start of pairing.
2. The agent never decides an approval. Only people approve.
3. Append-only audit of: approvals, remote actions, screen control, permission changes, paired devices.
4. A secret is never returned by the API after it is saved.
5. Network egress per agent may be restricted to a list of destinations (later phase).
