# CodeRoam MVP Development Specification

## 1. Purpose

CodeRoam is an open-source remote engineering workspace designed specifically for touch-first
phone and tablet use. The system lets a developer work with code, terminals, Git, language
servers, CI/CD, AI coding CLIs, logs, previews, and controlled production actions in a
user-controlled remote environment.

Remote execution is necessary but not differentiating by itself. CodeRoam's product differentiation
is the combination of:

1. A native Flutter product shell designed around touch and mobile task modes.
2. A secure provider-independent client/agent data plane.
3. Mobile-resilient reconnect and session-resume behavior.
4. Structured engineering workflows instead of a desktop IDE merely rendered on a smaller screen.
5. Limited, safe offline drafting rather than an over-promised synchronization engine.

## 2. Product principles

### 2.1 Touch-first, not desktop-shrunk

Phone and tablet layouts expose only the controls necessary for the current task. Editor, terminal,
review, preview, AI, and incident views are task modes rather than permanently visible desktop
panels.

Interactive targets follow platform minimums:

- iOS/iPadOS: at least 44 × 44 points.
- Android: at least 48 × 48 dp.

High-frequency actions remain within thumb reach on phones. A hardware keyboard or pointer enables
denser layouts and additional shortcuts but is not required for core navigation.

### 2.2 User-controlled compute

CodeRoam does not provide managed development machines in the MVP. The user installs the CodeRoam
agent in an environment they control or are permitted to use. Supported environments include
ordinary Linux servers and third-party development workspaces that allow the agent to run.

### 2.3 Relay is the core runtime path

The normal product path is always:

```text
Flutter client ⇄ CodeRoam relay ⇄ CodeRoam agent
```

The agent establishes an outbound connection, so no inbound workspace port is required. SSH may be
used manually to install the agent, but CodeRoam does not use SSH tunneling as its normal runtime
transport.

### 2.4 Relay opacity

The relay authenticates short-lived tickets, pairs endpoints, applies quotas and backpressure, and
forwards frames. It cannot decrypt engineering payloads. Source code, terminal output, LSP traffic,
AI streams, logs, and preview traffic are never persisted in relay storage, PostgreSQL, Redis, or
the worker.

### 2.5 Safe scope

The MVP has many users but no organization or team domain. Every resource has one owning user.
There is no enterprise RBAC hierarchy or general-purpose policy engine.

Offline support is limited to explicit Offline Drafts for selected text files. The MVP does not
implement a distributed filesystem, CRDT, repository sync, automatic merge, or offline Git.

## 3. Deployables

### 3.1 Flutter mobile client

Responsibilities:

- Authentication and device identity.
- Environment/project selection.
- Touch-first navigation and responsive phone/tablet layout.
- CodeMirror and xterm.js WebView bridges.
- Secure local key and session state.
- Multiplexed encrypted connection to the agent through the relay.
- Local rendering of input before acknowledgement.
- Limited encrypted Offline Drafts.
- Git, GitHub, AI, preview, logs, and runbook presentation.
- Biometric confirmation for sensitive actions.

The mobile client does not own server-side authorization. It does not execute build tools, Git,
language servers, or AI CLIs locally.

### 3.2 Control plane

A Go modular monolith exposing REST/JSON and OpenAPI.

Modules:

- `auth` — user identity and login/session metadata.
- `device` — device public identities, names, last-seen, and revocation.
- `workspace` — agents, environments, projects, and their ownership.
- `session` — short-lived relay tickets and session metadata.
- `preview` — preview sessions and preview-only invitations.
- `integration` — GitHub installation and webhook metadata.
- `runbook` — owner-defined runbooks and execution history.
- `outbox` — durable post-commit side effects.

Every module is the only writer to its PostgreSQL schema. Modules communicate through Go
application interfaces. Explicit cross-schema SQL is permitted only in read models and never for
authorization, device trust, pairing, runbook safety, or preview capability checks.

### 3.3 Worker

The worker consumes PostgreSQL transactional-outbox events and runs scheduled cleanup.

Examples:

- Revoke active relay sessions after a device or agent revocation event.
- Process GitHub installation/webhook work.
- Expire stale pairing attempts and preview invitations.
- Deliver non-sensitive notifications.
- Record terminal failure status for permanently failed jobs.

Jobs are idempotent, timeout-bounded, retry-safe, and observable.

### 3.4 Relay

The relay is a regional Go data-plane service.

Responsibilities:

- Accept outbound agent WebSocket connections.
- Accept client WebSocket connections.
- Validate short-lived signed connection tickets.
- Route pairing/session connections by opaque identifiers.
- Forward encrypted multiplexed frames.
- Apply maximum frame sizes, connection limits, flow control, and slow-consumer handling.
- Prioritize latency-sensitive channels.
- Support controlled drain and reconnect during deployment.

The relay does not:

- Parse application messages after the minimal outer routing envelope.
- Decrypt client-agent payloads.
- Persist payload frames.
- execute workspace commands.
- own project or user business rules.

Redis may hold short-lived relay instance presence, endpoint routing, ticket replay markers, and
rate-limit counters. It never stores application payloads or durable ownership.

### 3.5 Workspace agent

A signed Go binary running as a non-root remote operating-system user.

Responsibilities:

- Generate and protect the agent identity.
- Establish and maintain an outbound relay connection.
- Pair with approved devices.
- Constrain filesystem operations to registered project roots.
- Manage PTYs and process trees.
- Execute structured Git commands.
- Start and supervise language servers.
- Run Codex CLI and Claude Code CLI under explicit permission rules.
- Stream predefined logs.
- Forward selected localhost ports.
- Execute predefined, owner-approved runbooks.
- Enforce timeout, cancellation, output, path, and resource limits.

The agent never constructs shell commands by concatenating user input. Commands are represented by
an executable, argument array, working directory, sanitized environment, timeout, and output limit.

## 4. Touch UX spike — Milestone 0

The first engineering milestone validates the product's riskiest assumption before backend-heavy
work.

The spike contains:

```text
Flutter shell
├── CodeMirror 6 WebView with mock files
└── xterm.js WebView with simulated terminal output
```

No control plane, relay, or agent is required for this milestone.

### Editor acceptance

Test on a physical iPad, iPhone, Android tablet, and Android phone:

- Tap-to-place cursor accuracy.
- Long-press word selection.
- Native selection-handle dragging.
- Multi-line selection.
- Copy, cut, and paste.
- Undo and redo.
- IME composition and Turkish characters.
- Software keyboard open/close and viewport correction.
- Cursor remains visible above keyboard.
- Orientation and split-screen changes.
- Search and replace.
- Completion popup positioning.
- Diagnostics and code-action touch targets.
- 10,000+ line file scrolling.
- Hardware keyboard shortcuts.
- Pointer and trackpad selection.
- Flutter-to-WebView focus transitions.

### Terminal acceptance

- `Ctrl+C`, `Ctrl+D`, `Esc`, `Tab`, and arrows through the developer key row.
- Selection and copy/paste.
- Fast output with bounded scrollback.
- Resize while keyboard is open.
- Full-screen terminal mode.
- Hardware keyboard and pointer behavior.
- Flutter-to-WebView focus transitions.
- No duplicate input when reconnect-style simulated events occur.

A successful spike produces a short device matrix in `docs/ux-spike-results.md`, including failures
and required bridge/overlay changes.

## 5. Identity and pairing

### 5.1 Device identity

The client generates a device static identity on first launch.

- Private key material remains protected by iOS Keychain or Android Keystore.
- The control plane stores only the public key, fingerprint, platform, display name, and revocation
  metadata.
- Device revocation terminates active sessions and prevents new session tickets.

### 5.2 Agent identity

The agent generates a static identity on first start.

- Private key is stored with user-only filesystem permissions.
- Public key and fingerprint may be registered with the control plane.
- Agent revocation prevents routing and new session tickets.

### 5.3 Initial pairing

The initial pairing runs through the relay without trusting the relay with session secrets.

1. The agent generates its identity and opens an outbound pairing connection.
2. The agent requests an opaque pairing identifier.
3. The agent displays a QR code containing:
   - pairing identifier,
   - agent static public-key fingerprint,
   - a random 128-bit pairing secret,
   - protocol version and expiry.
4. Manual fallback uses an equivalent high-entropy base32 secret, not a six-digit PIN.
5. An authenticated mobile user scans or enters the payload.
6. The client obtains a short-lived pairing relay ticket.
7. Client and agent run `Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s`, using the pairing secret as the
   pre-shared key.
8. Both endpoints prove possession of their static identities, derive ephemeral session keys, and
   pin the peer static public key.
9. The agent proves pairing completion to the control plane; the agent becomes owned by the user.
10. The pairing identifier and secret become unusable immediately.

The pairing secret is never used as a long-term key. Pairing attempts are expiry-limited,
attempt-limited, rate-limited, and replay-protected.

### 5.4 Normal session handshake

After pairing:

1. The authenticated client requests a short-lived signed session ticket.
2. The control plane checks resource ownership, device status, agent status, and project access.
3. Client and agent connect to the relay with matching opaque session identifiers.
4. The relay validates tickets and pairs the endpoints.
5. The endpoints run `Noise_IK_25519_ChaChaPoly_BLAKE2s` using pinned static public keys.
6. All application frames are encrypted with derived session keys.
7. Key rotation occurs before sequence exhaustion and on resumed-session policy boundaries.

Outer TLS protects each endpoint-to-relay transport. Noise protects the application payload
end-to-end from the relay and control plane.

## 6. Multiplexed protocol

One encrypted logical session carries channels including:

- control and heartbeat,
- filesystem,
- editor document operations,
- PTY input/output,
- Git,
- LSP,
- AI CLI streams,
- log streams,
- port forwarding and preview.

Every frame includes:

- protocol version,
- session identifier,
- channel identifier,
- stream identifier,
- sequence number,
- acknowledgement watermark,
- flags,
- encrypted payload length.

### Priorities

1. Control, heartbeat, cancellation.
2. Terminal input and editor save acknowledgement.
3. Interactive filesystem and Git.
4. LSP request/response.
5. Terminal output.
6. AI and log streaming.
7. Preview and bulk transfer.

Per-channel queues and the whole session are bounded. Slow consumers cause channel-specific
backpressure or cancellation rather than unbounded memory growth.

### Resume

Endpoints maintain bounded replay state for resumable channels. A reconnect presents the last
acknowledged sequence watermark. The agent either resumes from retained state or returns a typed
`resume_unavailable` result so the client can refresh safely.

PTYs may survive transient network loss for a configured grace period. Filesystem saves remain
version/hash checked and atomic.

## 7. Remote filesystem and editor

Each project has one registered absolute root. Every request carries a project-relative path.

The agent must defend against:

- `..` traversal,
- absolute paths,
- symlink races and root escape,
- hard-link and special-file surprises,
- case-normalization differences,
- oversized files,
- path changes between validation and use.

File save flow:

1. Client sends project ID, path, expected remote version/hash, and new content or patch.
2. Agent resolves and opens the path through confinement-safe operations.
3. Agent rejects a stale expected version.
4. Agent writes to a temporary file in the destination directory.
5. Agent preserves approved permissions.
6. Agent fsyncs when configured and atomically renames.
7. Agent returns the new version/hash.

CodeMirror renders edits locally and sends ordered document operations. The protocol supports a
full-content resync when operation history is unavailable.

## 8. Terminal and processes

The agent creates a PTY with an explicit shell/profile chosen from trusted configuration.

- Input is a byte stream, not a shell command API.
- Structured one-shot actions use executable + args.
- PTY output and scrollback replay are bounded.
- Resize messages are coalesced.
- Cancellation terminates the process group.
- Disconnect grace period is explicit.
- Environment values are filtered; secret values are never returned in metadata.

The Flutter/xterm surface provides a context-aware developer key row.

## 9. Git and GitHub

Git executes in the workspace through structured agent operations.

MVP Git operations:

- status,
- branch list/create/switch,
- diff and hunk retrieval,
- stage/unstage,
- commit,
- fetch,
- pull with explicit strategy,
- push with explicit user confirmation.

Credentials remain in the workspace environment.

The GitHub App handles repository metadata, pull requests, review comments, Actions, deployment
status, and webhooks. GitHub is not used as a source-code data plane.

## 10. Language servers

The agent discovers and supervises configured language servers for:

- TypeScript/JavaScript,
- Go,
- Python,
- Rust.

The agent owns process lifecycle; the client owns UI presentation. LSP messages are streamed over a
dedicated channel with request cancellation, document-version checks, bounded queues, and restart
behavior.

## 11. AI CLI adapters

Codex CLI and Claude Code CLI execute in the remote environment. Provider credentials remain there
and are not stored in CodeRoam's control plane.

Default permission tiers:

- Read/search: allowed within the registered project root.
- File writes: require explicit or scoped approval.
- Command execution: requires explicit or scoped approval.
- Secrets and protected paths: denied.
- Git push: never autonomous.
- Deployment: never autonomous.
- Production runbook: never autonomous.

AI output is streamed but not centrally persisted.

## 12. Offline Drafts

Offline Drafts are intentionally not a synchronization engine.

Supported:

- Previously opened or explicitly selected text files.
- Encrypted local base content and draft content.
- Local syntax highlighting and editing.
- Base remote version/hash.
- Explicit reconnect/apply flow.
- Conflict warning when remote content changed.
- Keep remote, explicitly replace with local, or save local as a copy.

Not supported:

- repository-wide cache,
- background bidirectional sync,
- CRDT,
- automatic three-way merge,
- last-write-wins,
- offline rename/delete,
- offline Git,
- binary editing,
- multi-device merge.

Reconnect behavior:

- If `remote_hash == base_hash`, the client may apply the draft through the normal atomic save.
- If hashes differ, no automatic overwrite occurs. The user resolves the conflict explicitly.

Draft content stays encrypted on the device and is not uploaded to PostgreSQL or Redis.

## 13. Preview and port forwarding

A remote dev server may remain on `127.0.0.1:<port>`.

The agent opens a multiplexed port-forward channel. The client exposes a loopback endpoint to an
in-app WebView. No workspace port needs to be publicly exposed.

A preview invitation grants only a short-lived capability for one preview session and port. It
grants no filesystem, terminal, Git, AI, agent, or runbook access. Organization membership is not
required.

## 14. Logs and controlled runbooks

The agent streams only predefined log sources with bounded output and redaction rules.

Runbooks are owner-defined structured actions, not arbitrary production shell access. A runbook
requires:

- owning user,
- active non-revoked device,
- explicit reason,
- biometric confirmation for sensitive actions,
- timeout and output limits,
- predefined executable/args/template,
- execution history.

AI agents cannot execute production runbooks.

## 15. Persistence

PostgreSQL stores durable control-plane metadata:

- users,
- device public identities and revocation,
- agent public identities and ownership,
- environments and projects,
- session metadata,
- preview sessions/invitations,
- GitHub installation metadata,
- runbooks and execution history,
- transactional outbox.

Redis stores only ephemeral:

- relay endpoint presence,
- short-lived routing,
- replay markers,
- rate limits,
- short-lived ticket/session metadata.

Neither stores source code, terminal output, LSP payloads, AI streams, preview bodies, or Offline
Draft content.

## 16. Observability

Every deployable emits structured metadata:

- request/session correlation IDs,
- deployment version,
- latency and error category,
- connection and channel counts,
- bounded queue usage,
- reconnect/resume outcome,
- job attempts.

Telemetry must not include engineering payloads, prompts, terminal output, file contents, token
values, or raw environment variables.

## 17. Milestone execution

### M0 — Touch UX spike
Real-device validation for CodeMirror/xterm inside Flutter WebViews.

### M1 — Foundation
Monorepo, CI, Go services, Flutter shell, PostgreSQL/Redis, Protobuf envelope, and crypto interface.

### M2 — Single-owner control plane
Auth, devices, agents, environments, projects, sessions, outbox; no organization or policy engine.

### M3 — Agent bootstrap and pairing
Signed binary, agent identity, outbound registration, QR/manual pairing, pinning, and revocation.

### M4 — Relay and secure session
Ticket validation, endpoint pairing, Noise, opaque frames, flow control, reconnect, and resume.

### M5 — Workspace core
Constrained filesystem, CodeMirror bridge, PTY, xterm bridge, atomic save, and process lifecycle.

### M6 — Git and LSP
Structured Git and four language ecosystems.

### M7 — Offline Drafts
Selected text files, encrypted drafts, base-hash validation, and explicit conflict resolution.

### M8 — Preview
Encrypted localhost forwarding, in-app WebView, and preview-only invitation.

### M9 — GitHub and AI
PR/Actions/deployment metadata, Codex, Claude Code, and approvals.

### M10 — Operations
Logs, observability integrations, controlled runbooks, security review, and tablet acceptance.

## 18. Definition of done

A feature is complete only when:

- Observable behavior matches this specification.
- Ownership and data-plane boundaries remain intact.
- Focused unit/integration/negative tests pass.
- Reconnect and failure behavior are tested when applicable.
- Security-sensitive changes receive adversarial review.
- Generated protocol code is current.
- Documentation and durable agent learnings are updated.
- The applicable full quality gate passes.
