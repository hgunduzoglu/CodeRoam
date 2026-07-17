# CodeRoam

**CodeRoam is an open-source, touch-first remote engineering workspace for coding, reviewing,
shipping, and operating software away from the desk.**

The mobile client runs on iOS, iPadOS, and Android. Build tools, language servers, terminals,
Git, and AI coding CLIs run inside a user-controlled remote environment through the CodeRoam
workspace agent. The client and agent communicate through CodeRoam's opaque relay using an
application-level end-to-end encrypted, multiplexed protocol.

## Product position

CodeRoam is not a desktop VS Code interface squeezed into a phone screen. It is a task-oriented,
touch-first engineering client:

- **Phone:** review, supervise, inspect, operate, and make focused edits.
- **Tablet:** focused development, terminal work, pull-request review, incidents, and AI supervision.
- **Tablet + keyboard/pointer:** extended coding sessions.

## Deployables

- `apps/mobile` — Flutter client.
- `services/control-plane` — Go modular-monolith REST API.
- `services/worker` — Go transactional-outbox and scheduled-job worker.
- `services/relay` — Go opaque regional data-plane relay.
- `services/agent` — Go workspace agent installed in remote environments.

## Core architecture

```text
Flutter client
      │  TLS WebSocket
      ▼
CodeRoam relay ── opaque encrypted frames only
      ▲
      │  outbound TLS WebSocket
Workspace agent
      │
      ├── filesystem
      ├── PTY
      ├── Git
      ├── LSP
      ├── Codex / Claude Code
      ├── logs and controlled runbooks
      └── localhost port forwarding
```

The runtime data path is **client ⇄ relay ⇄ agent**. SSH is not the normal runtime transport.
An SSH session may optionally help a user install the agent, but all product traffic uses the
CodeRoam relay and protocol.

## MVP boundaries

Included:

- Touch-first CodeMirror 6 editor and xterm.js terminal embedded in Flutter.
- User-controlled remote environments with an installable CodeRoam agent.
- Secure QR/manual pairing, pinned device/agent identities, revocation, reconnect, and resume.
- Remote files, PTY, Git, LSP, AI CLIs, in-app preview, GitHub, logs, and controlled runbooks.
- Limited encrypted **Offline Drafts** for selected text files.

Not included:

- Organizations, teams, enterprise RBAC, or a general policy engine.
- Managed development compute.
- Full VS Code extension compatibility.
- CRDTs, repository-wide offline synchronization, automatic conflict merging, offline Git, or
  multi-device offline merge.
- Public unauthenticated preview links.
- Unrestricted production shell or AI-triggered push/deploy/runbook actions.

## Start here

```bash
cp .env.example .env
make bootstrap
make test-go
```

Flutter platform projects are checked in. Regenerate them only when intentionally updating the
Flutter/native project scaffolding:

```bash
./scripts/bootstrap-mobile.sh
```

Install web dependencies:

```bash
npm install
```

Open the root folder or `coderoam.code-workspace` in VS Code.

## Documentation

- [`PRODUCT.md`](PRODUCT.md) — approved MVP scope.
- [`docs/development-spec.md`](docs/development-spec.md) — canonical implementation specification.
- [`docs/architecture.md`](docs/architecture.md) — system boundaries.
- [`docs/security.md`](docs/security.md) — trust, pairing, and security model.
- [`docs/protocol.md`](docs/protocol.md) — multiplexed data-plane rules.
- [`docs/touch-ux-spike.md`](docs/touch-ux-spike.md) — Milestone 0 acceptance plan.
- [`AGENTS.md`](AGENTS.md) — Codex and Claude Code engineering instructions.

## License

Apache License 2.0.
