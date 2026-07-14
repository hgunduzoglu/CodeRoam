# CodeRoam — MVP Product Specification

## Vision

CodeRoam is an open-source, touch-first remote engineering workspace for coding, reviewing,
shipping, and operating software away from the desk.

The product connects an iOS, iPadOS, or Android client to a user-controlled remote environment.
Compilation, tests, language servers, Git, terminals, and AI coding CLIs execute remotely. The
CodeRoam relay is the normal runtime transport and remains opaque to application payloads.

## Primary use by device

### Phone

- Review changes and pull requests.
- Inspect builds, deployments, logs, and incidents.
- Supervise Codex or Claude Code.
- Run focused terminal commands.
- Make small, deliberate code edits.
- Execute predefined, controlled runbooks.

### Tablet

- Focused code editing.
- Terminal and Git workflows.
- Pull-request review.
- Dev-server preview.
- Incident response.
- AI-assisted development supervision.

### Tablet with keyboard and pointer

- Extended coding sessions.
- Multi-pane editor and terminal workflows.
- Shortcut-heavy navigation.
- More desktop-like information density without copying desktop VS Code UI.

## MVP IN

- Flutter client for iOS, iPadOS, and Android.
- CodeMirror 6 editor and xterm.js terminal embedded through WebView.
- A real-device touch UX spike before backend-heavy implementation.
- User-controlled remote environments with a signed, non-root CodeRoam agent.
- User-supplied servers and supported third-party environments where the agent can be installed.
- Multiple environments and projects, with one active project and retained per-project UI state.
- Go modular-monolith control plane.
- Separate Go worker, relay, and workspace-agent deployables.
- PostgreSQL durable metadata and transactional outbox.
- Redis ephemeral presence, routing, rate-limit, and short-lived state only.
- Relay-only runtime data path: client ⇄ relay ⇄ agent.
- TLS to the relay plus application-level E2E encryption between client and agent.
- QR/manual initial pairing and pinned device/agent identities.
- Protobuf envelopes over multiplexed WebSocket connections.
- Bounded flow control, channel priorities, acknowledgements, reconnect, and session resume.
- Remote filesystem, PTY, structured Git, LSP, logs, port forwarding, and dev preview.
- TypeScript/JavaScript, Go, Python, and Rust language intelligence.
- GitHub App integration for repositories, pull requests, reviews, Actions, and deployments.
- Codex CLI and Claude Code CLI adapters running inside the remote environment.
- Explicit AI approval boundaries; no autonomous push, deploy, or production runbook.
- Limited encrypted Offline Drafts for selected text files.
- Authenticated, short-lived, preview-only invitations.
- Sentry, Grafana, Loki, GitHub Actions, and predefined remote-log contexts.
- Owner-only controlled production runbooks with biometric confirmation, reason, timeout, and
  execution history.
- Open-source self-hosting manifests for control plane, worker, and relay.

## MVP OUT

- Organizations, teams, memberships, enterprise RBAC, or a general policy engine.
- Managed development compute.
- Full desktop IDE parity.
- Full VS Code extension-host or marketplace compatibility.
- Native Swift or Kotlin editor implementations.
- Runtime SSH tunneling as the primary data path.
- Local compilation, remote LSP, terminal, Git execution, or AI execution while offline.
- Repository-wide background synchronization.
- CRDTs, automatic three-way merge, last-write-wins, offline Git, offline rename/delete, or
  multi-device offline conflict resolution.
- Public unauthenticated previews.
- Unrestricted production shell in the runbook interface.
- AI-triggered push, deploy, or production action.
- A custom Git host, CI engine, observability store, or model provider.
- A second durable database, Kafka, NATS, MongoDB, or DynamoDB.
- Microservices for control-plane domain modules.
- GitLab integration in the initial MVP.

## Approved stack

| Area | Choice |
| --- | --- |
| Product name | CodeRoam |
| Client | Flutter |
| Editor | CodeMirror 6 in WebView |
| Terminal | xterm.js in WebView |
| Backend | Go |
| Control plane | Modular monolith |
| Durable jobs | Separate Go worker + PostgreSQL outbox |
| Data plane | Go relay + Go workspace agent |
| Durable database | PostgreSQL |
| Ephemeral state | Redis |
| Public API | REST/JSON + OpenAPI |
| Runtime path | Client ⇄ relay ⇄ agent |
| Data-plane contract | Protobuf multiplexed over WebSocket |
| E2E handshake | Noise XXpsk3 for initial pairing; Noise IK after key pinning |
| Workspace hosting | User-controlled or supported third-party environment |
| Offline model | Explicit encrypted Offline Drafts |

## Milestones

0. Touch UX spike on real iOS/iPadOS and Android devices.
1. Reproducible monorepo, CI, Flutter shell, Go deployables, and protocol skeleton.
2. Single-owner control plane: auth, devices, agents, environments, projects, sessions, and outbox.
3. Signed agent bootstrap, outbound registration, QR/manual pairing, key pinning, and revocation.
4. Opaque relay, E2E secure session, multiplexing, flow control, reconnect, and resume.
5. Remote filesystem, CodeMirror integration, PTY, and xterm.js integration.
6. Structured Git and TypeScript/JavaScript, Go, Python, and Rust LSP.
7. Limited Offline Drafts with base-hash validation and explicit conflict handling.
8. Encrypted localhost forwarding, in-app preview, and scoped preview invitations.
9. GitHub integration, Codex CLI, Claude Code CLI, and explicit approvals.
10. Logs, incidents, controlled runbooks, security review, and tablet acceptance testing.

A milestone is complete only when its observable acceptance behavior and tests pass.
