# CodeRoam Architecture

## System view

```text
                         REST/JSON
Flutter client ─────────────────────────► Control plane
      │                                        │
      │                                        ├── PostgreSQL
      │                                        ├── Redis (ephemeral)
      │                                        └── Outbox → Worker
      │
      │ encrypted multiplexed WebSocket
      ▼
Regional relay  ◄──────────────────────── Workspace agent
 opaque frames          outbound WebSocket      │
                                               workspace
```

## Boundary rules

- The relay is the only normal runtime transport.
- The relay cannot decrypt application payloads.
- The control plane is metadata/control only.
- Engineering payloads bypass PostgreSQL, Redis, and the worker.
- The control plane is a modular monolith with schema ownership.
- The agent is the remote executor.
- The Flutter client is touch-first presentation and orchestration.
- Every durable resource has one owning user; no organization model exists in the MVP.
- Offline support is explicit text-file drafts, not filesystem synchronization.

## Control-plane modules

```text
auth
device
workspace
session
preview
integration
runbook
outbox
```

A module may expose application interfaces. It may not write another module's tables.
