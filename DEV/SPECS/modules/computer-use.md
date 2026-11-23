# Module: computer-use

Phase 5 · Package `internal/computer` (PC only)

## Goal
The agent operates desktop apps; the person controls the PC screen from their phone.

## Rules
- **Off by default.** Turned on per app, at the PC itself (never remotely).
- Actions: screen capture, click, keyboard, scroll, list windows.
- While active: a visible indicator on the PC and an always-visible stop button.
- Screen control from the phone (paid plan): video stream over the transport + remote input; must be turned on at the PC; audit of start and end.
- macOS requires Accessibility and Screen Recording permissions; the app guides the person through them.
- Sensitive apps (banking, password manager) blocked by default.

## Acceptance
- With the feature off, no computer tool appears to the agent.
- Stopping on the PC ends the remote session in less than 1 s.
