# CodeRoam — Agent Instructions

CodeRoam is an open-source, touch-first Flutter client with Go control-plane, worker, relay, and
workspace-agent deployables.

This repository supports OpenAI Codex and Anthropic Claude Code.

## Read first

1. `PRODUCT.md`
2. `docs/development-spec.md`
3. The nearest nested `AGENTS.md`
4. `docs/agent/code_review.md`
5. `PLANS.md` for broad work
6. `docs/agent/AGENT_COMPATIBILITY.md` before changing agent configuration

`AGENTS.md` files are canonical. Claude `CLAUDE.md` files import adjacent instructions.

## Product invariants

- CodeRoam is touch-first, not desktop VS Code rendered on a smaller screen.
- Milestone 0 validates CodeMirror/xterm touch behavior on physical devices.
- Runtime traffic is client ⇄ relay ⇄ agent from the first integrated data-plane milestone.
- SSH is not the normal runtime transport.
- Relay payloads remain application-level E2E encrypted and opaque.
- The MVP has single-owner resources and no organization, team, enterprise RBAC, or policy engine.
- Offline support is limited to explicit encrypted text-file drafts with base-hash checks.
- No CRDT, automatic merge, last-write-wins, repository sync, or offline Git in MVP.
- No managed development compute in MVP.

## Repository map

| Path                      | Owner                                       |
| ------------------------- | ------------------------------------------- |
| `apps/mobile/`            | Flutter touch-first client and local drafts |
| `services/control-plane/` | Go modular-monolith control API             |
| `services/worker/`        | Outbox and scheduled durable jobs           |
| `services/relay/`         | Opaque encrypted data-plane relay           |
| `services/agent/`         | Remote workspace executor                   |
| `protocol/proto/`         | Authoritative Protobuf contracts            |
| `protocol/gen/`           | Generated contracts; never edit manually    |
| `webview/editor/`         | CodeMirror surface and typed bridge         |
| `webview/terminal/`       | xterm.js surface and typed bridge           |

## Mandatory start

```bash
pwd
git status --short
git branch --show-current
```

- Do not edit on `main`.
- Read current implementation and tests.
- Identify the owner and nearest instructions.
- Use an ExecPlan for work spanning three deployables, protocol/security changes, migrations, or
  multiple independently verifiable milestones.

## Architecture rules

- The control plane is a modular monolith.
- Every module is the only writer to its PostgreSQL schema.
- In-process modules communicate through Go interfaces.
- Cross-schema SQL is read-model only and never decides authorization or trust.
- Durable external side effects use PostgreSQL transactional outbox.
- Redis contains ephemeral state only.
- Engineering payloads never enter PostgreSQL, Redis, worker jobs, or logs.
- Client-to-control-plane uses REST/JSON.
- Client and agent connect independently to the relay over TLS WebSockets.
- Client-agent application frames use Protobuf and E2E Noise encryption.
- Initial pairing uses XXpsk3 and a high-entropy QR/manual secret.
- Normal sessions use IK and pinned static identities.
- Every frame, queue, replay buffer, process, output, and deadline is bounded.

## Commands

```bash
cp .env.example .env
make bootstrap
make bootstrap-mobile
npm install
make proto
make fmt
make lint
make test
make build
make up
make down
```

Go modules are tested individually through `make test-go`; do not replace it with `go test ./...`
from repository root.

If skills change:

```bash
python3 scripts/sync-agent-skills.py
python3 scripts/sync-agent-skills.py --check
```

## Coding and security rules

- No global `utils`, `helpers`, `common`, or shared domain-model package.
- Shared packages contain domain-neutral technical primitives only.
- Never concatenate user input into a shell command.
- Filesystem requests stay inside registered project roots despite symlinks or races.
- Never bypass device/agent revocation, pairing, biometric, preview, or runbook checks.
- Never log source code, terminal output, prompts, credentials, tokens, secret values, or raw
  environment variables.
- Never let AI push, deploy, or execute production runbooks autonomously.
- Never edit generated protocol files manually.
- Ask before adding a runtime dependency or changing approved scope/stack/protocol.

## Specialist agents

Use only when specialization or parallelism materially helps:

- `architect`
- `mobile-engineer`
- `control-plane-engineer`
- `data-plane-engineer`
- `integration-tester`
- `security-reviewer`

Avoid agents editing the same contract, migration, or file concurrently. The parent owns
integration and final verification.

## Skills

Canonical sources live in `.agents/skills/`; Claude mirrors are generated in `.claude/skills/`.

- `$touch-ux-spike`
- `$implement-vertical-slice`
- `$change-data-plane-protocol`
- `$security-sensitive-change`
- `$database-change`
- `$verify-repository`
- `$regenerate-protocol`

## Completion

A task is complete only when behavior, focused tests, full applicable gates, generated files,
security invariants, docs, and any ExecPlan are consistent. Report skipped checks honestly.

Do not commit, push, force-push, or open a pull request unless explicitly requested.
