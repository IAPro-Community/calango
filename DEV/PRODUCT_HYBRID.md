# Hybrid product — PC, VPS and phone

Status: product proposal. None of this is implemented.

## Core idea

Calango is **one agent with several nodes**. Each installation is a node, and the person chooses where each thing runs:

- **PC node** (free): installed on the computer. Has access to local files, apps and the browser.
- **VPS node** (paid): the same installation, on the person's VPS, via Docker. Runs 24/7, executes routines and projects without depending on the PC being on.
- **Clients** (phone, desktop, web): do not run the agent. They talk to the account's nodes and show what is happening on any of them.

The account is the identity. The nodes belong to the person. What leaves the nodes for Calango's cloud is only the minimum needed to connect and bill.

## The three connection modes

| Path | When | Who sees the content |
|---|---|---|
| **Direct LAN** | phone and PC on the same network (mDNS) | only the devices |
| **Tunnel via relay** | outside the house; the PC opens an outbound connection to the relay, with no port opened on the router | nobody but the endpoints (end-to-end encryption) |
| **Direct VPS** | VPS with a domain and TLS | only the devices |

The tunnel is the key piece. The PC (or the VPS) opens an **outbound** connection to Calango's relay, and the phone connects to the relay. The relay only forwards encrypted bytes between the two. This works behind any router, with no configuration.

## Hybrid work

- **Where it runs**: each task or routine has a destination: PC, VPS or automatic. Automatic = the PC if it is online, otherwise the VPS (if the person has one). Without a VPS, the routine waits for the PC to come back.
- **Projects (ADE)**: the code lives in git. PC and VPS work in worktrees of the same repository and sync through `push`/`pull`. Do not invent a custom code sync.
- **Memory and conversations**: append-only event log per node, replicated between the account's nodes. Each record has its origin node, so there are no authorship conflicts.
- **Remote control of the PC from the phone**: the phone sends commands to the PC through the tunnel, and the PC executes them. A sensitive action still requires approval on the phone.

## Remote control of the PC (what the person pays for)

From the phone, with the PC on:
- see the state of agents, routines and tasks;
- approve or refuse sensitive actions;
- send tasks and messages;
- see the PC screen and control mouse and keyboard (explicit opt-in, with a notice on the PC screen and an audit record).

Remote screen control is the highest-risk feature. It is off by default and must be turned on at the PC.

## Security (non-negotiable)

- End-to-end encryption between the phone and the node. The key is created at pairing and stored in the device keychain. The relay cannot read content.
- Pairing with a single-use code, with per-device revocation.
- PC and VPS nodes never accept a connection without a valid device token, even on the LAN.
- Audit of every remote action, visible on the phone and on the PC.
- A sensitive write action always asks for approval, wherever it comes from.

## Monetization

The rule is: **what runs locally is free; what needs our infrastructure or server time is paid.** This way the free plan is genuinely useful and the paid plan has a clear reason.

### Plans

| | Free | Pro | Team |
|---|---|---|---|
| PC node | 1 | 1 | unlimited per member |
| VPS node (extension) | no | 1 | 1 per member |
| Phone on the LAN | yes | yes | yes |
| Relay tunnel: control and status | small daily quota | unlimited | unlimited |
| Screen and file control through the tunnel | no | yes | yes |
| Hybrid execution (PC ↔ VPS) | no | yes | yes |
| Routines on the VPS 24/7 | no | yes | yes |
| Phone push | no | yes | yes |
| Shared channels | no | no | yes |

### Add-ons (billed by usage)

1. **Model credits** (BYOK on free; purchased on Pro).
2. **Relay beyond the quota**: file and screen traffic, per GB.
3. **VPS managed by us**: the same VPS, without the person administering it.
4. **Storage** for files and history above the quota.
5. **Premium connectors** with a per-call cost.

### How to charge for the VPS extension

- The payer receives a **license key**. The VPS node's Docker image only accepts a valid key for the paid features.
- Validation is **signed offline**: the node checks the signature locally and queries the license server from time to time. If the internet goes down, there is a grace period. Nothing that is free stops working.
- Without a key, the VPS runs only what is free, or does not start at all. Decide this carefully: shutting down a person's production server for non-payment is bad for product trust. Preferable: downgrade to free mode and notify.

### Why the user pays

- Wants the agent running when the PC is off.
- Wants to control the PC from outside without configuring a router.
- Wants to work on projects on the VPS and the PC at the same time.
- Wants phone notifications even with everything closed.

### Costs Calango has (and charges for)

- Relay (tunnel traffic): billed by usage above the quota.
- Push (APNs/FCM via the relay): included in Pro, limited on Free.
- License server: cheap, included.
- Hosted models: credits with a markup.

## What to build first (order)

1. **PC node with pairing and LAN** (free; proves the product).
2. **Phone client** connected to the PC node on the LAN.
3. **Relay with outbound tunnel** and a free quota for control.
4. **License key and VPS node** (first paid item).
5. **Hybrid execution** (choice of destination and event replication).
6. **Screen control** through the tunnel (paid).

## Risks

- **The relay costs money and becomes a bottleneck**: limit it on the free plan from the start.
- **Remote PC control is an attack surface**: audit and opt-in are mandatory before release.
- **The VPS node code** is what the person runs; decide whether it is open source (with license by key) or source-available. Do not copy code from the reference repos.
- **App stores**: confirm that the remote-control app is approved in both stores, and that the subscription goes through their purchase systems (see [MONETIZATION.md](MONETIZATION.md)).
- **Memory synchronization between nodes**: the hardest part to get right. Start with simple replication and one "owner" node per record.

## Decision: transport for now is Tailscale

Decided on 2026-11-23: the initial phase uses **Tailscale** (WireGuard private network) as the tunnel, instead of an own relay.

- Phone, PC node and VPS node join the person's tailnet. End-to-end connection, no open port and no relay of ours.
- Implementation alternative: **tsnet** (Tailscale's Go library) embedded in `calango`, so the node joins the tailnet without the person installing a separate client. Confirm license and terms before embedding.
- Consequence for the business model: the tunnel is no longer billed for our traffic in this phase. Monetization becomes the VPS license, model credits, managed VPS and connectors. The relay quota waits until an own relay exists.
- User cost: a Tailscale account is required. Confirm the limits of their free plan for personal use before advertising "free".
- The own relay stays on the roadmap, for those who do not want to depend on third parties and for paid screen control.

Open questions:
- Is the VPS node open source or source-available?
- Does screen control go in Pro or only in Team?
