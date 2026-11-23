# Calango competitive map

Review of public sources: **2026-11-23**. Scope: products that cover at least two of these jobs: a personal assistant that executes tasks, local execution or execution on a machine controlled by the user, phone access, recurring automation, computer/browser use, agent-driven development, approvals, or collaboration between agents.

## Method and limits

- I prioritized official pages, documentation and repositories maintained by the product itself. Each name below links to the source consulted.
- The table compares publicly declared capabilities; it is not a practical test of quality, security, price, availability by country or terms of use. A capability not mentioned means **not verified**, not necessarily absent.
- "Direct" indicates overlap with the core proposition; "adjacent" indicates competition for a specific flow or for the user's budget.
- There is no closed list of all competitors in a market that changes weekly. This review covers the visible products that meet the criterion above and leaves the monitoring list explicit.

## Comparable products

| Product | Type | Public capabilities that intersect with Calango | Reading for the product |
|---|---|---|---|
| [OpenClaw](https://docs.openclaw.ai/) | Direct · self-hosted | Own gateway on a computer or server, many messaging channels, mobile nodes, cron, browser, skills and plugins. | Largest overlap in the always-available, self-hosted assistant. "Runs at home" no longer differentiates on its own. |
| [Hermes Agent](https://hermes-agent.nousresearch.com/docs) | Direct · self-hosted | CLI and desktop, local/Docker/SSH/serverless backends, dozens of channels, persistent memory, cron, bots, tools and MCP. | Competes on execution flexibility, memory, automation and channel variety. |
| [Skales](https://github.com/skalesapp/skales) | Direct · desktop/mobile | Declares desktop and mobile support, persistent agents, tasks, browser/computer use, voice, MCP, local models and a mobile relay. | Very high overlap; watch the boundary between local execution, VM and relay, and the pairing experience. |
| [Nova Personal Agent](https://github.com/deepnovacore/NovaPersonalAgent) | Direct · local/multi-device | Declares a Space/Host/Client architecture, companion apps, E2E sync, scheduled tasks, computer use and a timeline with evidence. | The emerging comparable closest to the idea of a personal identity crossing devices. Verify maturity by release; do not extrapolate from the README. |
| [vBot](https://github.com/Vironnimo/vbot) | Direct · self-hosted | Persistent agents, desktop/web/terminal/channels, cron, voice, projects, subagents, run resumption, permission rules and secrets. | Shows that memory, channels and scheduled tasks are already a basic expectation in this segment. |
| [MindRoom](https://github.com/mindroom-ai/mindroom) | Direct · personal and teams | Personal and shared agents, tools/MCP, local memory, paired computer, web/desktop/mobile clients and approvals. | Close competitor for "a chat with people and agents"; strong in integrations and workspace experience. |
| [agentes.chat](https://github.com/agentes-chat/agentes-chat) | Direct · teams | Self-hosted chat, channels/DMs, agents, MCP, budget, memory, routines and approvals; web, desktop and mobile clients. | Strong substitute in Team. Calango should keep the individual and their nodes at the center before growing into teams. |
| [Perplexity Personal Computer](https://www.perplexity.ai/en-GB/hub/products/portable-computer) | Direct · local agent | The page describes installing a local harness, local models and isolated execution on compatible machines. | Validates demand for locally assisted execution; availability and required hardware must be verified before any cost comparison. |
| [Manus](https://www.manus.im/features) and [Cloud Computer](https://www.manus.im/features/cloud-computer) | Direct · hosted + desktop | Mobile app, browser operator, recurring tasks, persistent cloud computer and an option to use a local computer. | Competes on the outcome "delegate from anywhere and keep working". Calango must demonstrate data control and node choice in a real flow. |
| [Claude Cowork/Code — Dispatch and computer use](https://claude.com/resources/articles/dispatch-and-computer-use) | Direct · model ecosystem | Work with the computer and applications, commands from the phone, and conversation continuity across surfaces. | Strong substitute for general work; dependent on the vendor's ecosystem and execution policy. |
| [ChatGPT Work](https://help.openai.com/en/articles/20001275-chatgpt-work-and-codex) and [scheduled tasks](https://help.openai.com/en/articles/10291617-scheduled-tasks-in-chatgpt) | Direct · hosted + desktop | Scheduled or event-triggered tasks, computer use on the desktop, cloud browser and mobile access. | Competitor for daily use and execution. The boundary between local and cloud files should be described as a flow and a data boundary, not as a slogan. |
| [Google Gemini — computer use on Mac](https://blog.google/innovation-and-ai/products/gemini-app/computer-use-mac/) | Direct · hosted desktop | Automates files, apps and the web in an isolated window, with preview and permission per app/action. | Visible control and granular permissions have become a competitive requirement. |
| [Meta Muse](https://about.fb.com/news/2026/09/introducing-muse-personal-ai-agent) | Adjacent · consumer assistant | Personal agent for everyday tasks inside Meta products and consumer channels. | Competes for attention and distribution; equivalence with self-hosted execution or ADE has not been verified. |
| [Goose](https://block.github.io/goose/) and [Interpreter Workstation](https://github.com/openinterpreter/interpreter-workstation) | Direct · local workstation | BYO models, MCP tools, recipes/skills, desktop and local execution; Workstation also describes local/remote endpoints and computer control. | Close competitors for technical users who value model choice and local data. |
| [OpenHands](https://hub.openhands.dev/) | Adjacent · development | Engineering agents in a local VM, cloud sandbox or VPC, with issue triage, QA and recurring work. | Reference for isolation and ADE automation; does not replace a multi-node personal assistant on its own. |
| [Cursor Cloud Agents](https://prod.cursor.com/help/ai-features/background-agents) and [mobile](https://prod.cursor.com/docs/cloud-agent/mobile) | Adjacent · development | Agents in VMs, access by web/mobile, automations, and remote control of a session whose tools still run on the PC. | Strong in ADE and local/cloud continuity; continuously review the boundary of context, credentials and execution. |
| [Devin](https://app.devin.ai/) | Adjacent · development | Cloud coding agents, recurring tasks, PRs, browser tests and video evidence. | Raises the expectation that a coding task delivers reviewable proof, not just a closing text. |
| [Replit Agent](https://replit.com/products/mobile) | Adjacent · software creation | Create, iterate and publish apps from the phone and continue on the desktop. | Competitor for "build with an agent" and mobile; execution and hosting are centered on their service. |
| [n8n](https://n8n.io/) and [Zapier Agents](https://zapier.com/blog/best-ai-agents/) | Adjacent · automation | Self-hostable or hosted workflows, large-scale connectors, observation of executions and human approvals. | Competitors in cross-app automation. Calango must combine natural language with deterministic rules, history and approvals without trying to recreate hundreds of connectors in V1. |
| [Home Assistant Assist](https://www.home-assistant.io/voice_control) | Adjacent · local voice | Local voice assistant and home control, with local or external model options. | Reference for voice privacy and local execution; the product domain is different. |

## What changed in Calango's thesis

1. **PC + server + phone is no longer a rare combination.** OpenClaw, Manus, Cursor and others already describe parts of this flow. Do not announce it as an exclusive advantage.
2. **The opportunity is coherence between the parts.** The same run must keep identity, history, budget, policy, approval and result when moving between the person's PC and their VPS. V1 still has to prove this claim in execution.
3. **Trust is verifiable behavior.** Approving the exact payload, preventing changes after approval, showing execution state and receipts, and allowing interruption/revocation are product criteria. A "safe" label does not replace these contracts.
4. **Cost must distinguish known, estimated and unknown.** Recording unknown as zero creates a false impression of free cost; the interface and the gateway must preserve the uncertainty state.
5. **The ADE must deliver evidence.** Diff, commands run, tests and artifacts must be reviewable, following the expectation that cloud coding agents have already created.
6. **Do not compete on connector count in V1.** Prioritize MCP with per-agent permissions, a reliable experience for the chosen connectors, and a clear path to add integrations.

## Recommendations that enter the V1 TODO

- Phase 1: persist run transitions; recover state after reconnection; do not duplicate a run on resend; show failure and estimated/actual/unknown cost.
- Phase 1: bind approval to an immutable canonical payload and verify that binding immediately before the action; store the receipt separately from the decision.
- Phase 1: identify each scheduled occurrence and make retries idempotent; do not promise "exactly once" for external effects without support from the destination.
- Phase 3: apply replication events and cursor advancement in the same transaction; node commands carry an idempotent ID and an explicit owner.
- Phase 4: deliver diff, commands and test results tied to the run; merge and publication remain subject to approval.
- Cross-cutting: guided first run, node/connector/model diagnostics, backup and restore demonstrated, export and deletion of personal data.

These recommendations refine criteria that already exist; they do not remove decisions D2–D6 nor authorize choosing the harness, price, relay or update policy on the owner's behalf. Engineering actions already taken in this cycle are in `DEV/WORKLOG.md`.

## Monitoring

Revalidate this matrix every quarter, before freezing V1 and whenever a core Calango capability changes. Update capabilities, links and availability status; do not automatically carry competitors' features into the roadmap.
