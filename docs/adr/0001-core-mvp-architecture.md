# ADR 0001 — Core CodeRoam MVP Architecture

- **Status:** Accepted
- **Date:** 2026-07-14

## Decision

CodeRoam uses:

- Flutter with CodeMirror 6 and xterm.js WebViews.
- Go modular-monolith control plane.
- Separate Go worker, opaque relay, and workspace agent.
- PostgreSQL plus transactional outbox.
- Redis for ephemeral state only.
- Relay-only runtime transport.
- Noise XXpsk3 initial pairing and Noise IK normal sessions.
- Single-owner resources without organizations or enterprise policy.
- Limited Offline Drafts rather than advanced synchronization.
- A mandatory real-device touch UX spike as Milestone 0.

## Consequences

- Relay and secure-session work remain first-class MVP architecture.
- Touch behavior is validated before large backend investment.
- Enterprise organization/RBAC complexity is deferred.
- Offline data-loss risk is controlled through explicit conflicts rather than automatic merge.
