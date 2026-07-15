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

# Engineering Principles

Apply these principles pragmatically. They are repository-wide requirements.

## Change Discipline

- Understand the existing implementation and applicable documentation before editing.
- Inspect all applicable `AGENTS.md` files before making changes.
- Preserve existing user changes. Never reset, discard, overwrite, or revert them unless explicitly requested.
- Make the smallest coherent change that fully addresses the task.
- Keep diffs focused and easy to review.
- Do not perform unrelated refactoring.
- Do not rename or move files without a concrete technical reason.
- Do not add speculative extension points or infrastructure for hypothetical requirements.
- Do not introduce new dependencies unless the task clearly requires them and the benefit is documented.
- Do not suppress analyzer, compiler, linter, or test failures without addressing and explaining the underlying issue.
- Do not commit, push, merge, amend commits, or modify pull requests unless explicitly requested.
- Do not claim work is complete unless the relevant validation commands have passed.

## DRY

DRY means avoiding duplicated knowledge, not mechanically eliminating every repeated line.

- Do not abstract incidental similarity.
- Prefer a small amount of obvious duplication over an incorrect shared abstraction.
- Extract shared behavior only when its responsibility, invariants, and variation are understood.
- Keep components separate when their semantics differ, even if their current implementation looks similar.
- Do not create generic `manager`, `handler`, `helper`, `utils`, `common`, or `base` abstractions without a clear domain responsibility.
- Do not create abstractions solely to reduce line count.

## SOLID

Apply SOLID without unnecessary ceremony.

- Keep responsibilities narrow at meaningful module boundaries.
- Separate presentation, transport, domain policy, persistence, and temporary test-harness behavior.
- Depend on small interfaces where a real architectural boundary or required test seam exists.
- Prefer composition over inheritance.
- Do not create an interface with only one implementation unless it establishes a meaningful boundary or enables necessary testing.
- Avoid classes and functions that merely forward calls without adding ownership, policy, validation, or transformation.
- Keep lifecycle and resource ownership explicit.

## Clean Code

- Use precise, domain-specific names.
- Prefer explicit and readable code over clever or compressed code.
- Keep control flow easy to follow.
- Return, propagate, and surface errors deliberately.
- Do not silently swallow failures.
- Comments must explain constraints, intent, platform behavior, or non-obvious decisions rather than restating the implementation.
- Remove dead code and obsolete comments.
- Avoid hidden mutable global state.
- Validate all data crossing process, network, serialization, Flutter-WebView, or other trust boundaries.
- Keep TypeScript strict.
- Avoid `any`, unchecked casts, and non-null assertions unless unavoidable and localized.
- Keep Dart null safety intact.
- Keep Go errors contextual and preserve error chains where appropriate.
- Never log secrets, pairing material, authentication data, complete editor documents, arbitrary terminal output, or other sensitive payloads.
- Logs should contain only the minimum diagnostic metadata required.

## Testing

- Test externally observable behavior and important invariants.
- Avoid tests coupled to private implementation details.
- Every bug fix should receive a regression test when reasonably practical.
- Keep tests deterministic.
- Do not introduce arbitrary sleeps to resolve races.
- Prefer explicit synchronization and controllable dependencies.
- Use dependency injection only where it establishes a meaningful boundary or necessary test seam.
- Do not instantiate native platform views in ordinary widget or unit tests.
- Separate automated verification from simulator and physical-device acceptance testing.
- Do not weaken or delete an existing test merely to make a change pass.

## Validation Discipline

- Inspect the repository's existing scripts before running or inventing commands.
- Use canonical repository commands rather than duplicating their behavior manually.
- Format changed files using the language's canonical formatter.
- Run the narrowest relevant checks first, followed by the applicable package or repository checks.
- Run checks only for affected areas unless repository policy explicitly requires broader validation.
- Do not routinely clear build caches or perform clean builds.
- Use clean builds only when stale generated assets, native state, or build caching is a plausible cause.
- When generated source is affected, use the repository's canonical generation workflow.
- Never edit generated files manually.

## Final Report

At the end of a task, report:

1. What was inspected.
2. Problems or risks found.
3. Changes made and the reason for each change.
4. Tests added or updated.
5. Validation commands run and their actual results.
6. Files changed.
7. Manual checks still required.
8. Remaining risks, limitations, or deferred work.
9. A suggested commit message.

Clearly distinguish:

- completed and validated work
- implemented but manually unverified work
- deferred work
- failed or unavailable checks

Never describe a task or milestone as complete unless its stated acceptance criteria have actually been satisfied.

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
