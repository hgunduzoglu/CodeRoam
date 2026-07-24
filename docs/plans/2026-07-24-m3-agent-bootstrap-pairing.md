# Milestone 3 Agent Bootstrap and Pairing

## Objective

Implement the first production-shaped trust establishment path between a signed CodeRoam agent and
an authenticated mobile device. M3 must create and persist stable X25519 identities, complete a
bounded Noise XXpsk3 pairing over an opaque pairing-only relay route, pin the recovered peer keys,
register both endpoints to one owner, and make revocation observable and enforceable.

M3 is complete only when a user can install a verifiable Linux agent artifact, start a one-use
pairing attempt, scan or manually enter the high-entropy secret on an authenticated iPhone, observe
the same pinned identities on both endpoints, restart both endpoints without identity replacement,
and revoke either endpoint so later authorization fails closed.

## Scope

This plan covers:

- signed Linux agent artifacts and documented provenance verification;
- explicit first-run agent and mobile X25519 identity creation with fail-closed restoration;
- one canonical public-key fingerprint encoding across Go, Dart, SQL, Protobuf, and UI;
- short-lived control-plane pairing attempts and purpose-bound relay tickets;
- a pairing-only TLS WebSocket route whose payloads remain opaque to the relay;
- cross-language `Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s` handshakes;
- two-sided confirmation and atomic single-owner device/agent registration;
- minimal paired-device and paired-agent listing and revocation surfaces;
- removal of the M2 build-time device selector after paired identity restoration is proven;
- migrations, compatibility, negative tests, failure injection, operational docs, and physical
  iPhone acceptance.

M3 does not implement normal IK sessions, reusable session tickets, multiplexed editor/terminal/LSP
channels, reconnect/resume, filesystem or process operations, managed compute, organizations,
automatic environment/project reassignment, agent self-update, or application engineering
payloads. Those boundaries remain assigned to M4 or later milestones.

## Current state

- M0, M1, and M2 are merged. M2 provides OIDC-authenticated single-owner users, devices, agents,
  environments, projects, metadata-only sessions, revocation primitives, and transactional outbox
  processing.
- The mobile app still receives a pre-registered `CODEROAM_DEVICE_ID` at build time. M3 must replace
  that selector with the restored paired device identity, without weakening M2 authorization during
  the rollout.
- The device and workspace tables already store 32-byte X25519 public keys and textual
  fingerprints. Their repositories authorize and revoke persisted rows, but registration, listing,
  and canonical fingerprint enforcement were intentionally deferred to M3.
- `session.pairing_attempts` is only a starter table. It has no owner claim, bootstrap credential,
  endpoint confirmation, or state-machine data and stores no pairing secret.
- The pairing Protobuf already carries a pairing ID, agent public key, fingerprint, pairing secret,
  protocol version, expiry, and completion keys. Relay ticket and common encrypted-frame messages
  also exist, but M3 must evolve them additively and define their security semantics before use.
- `packages/go/cryptox` validates 32-byte X25519 public keys and exposes an identity-provider
  boundary. Private identity persistence and canonical fingerprints are not implemented.
- The agent `pair` command currently prints placeholder material, and the agent runtime has no
  persistent identity or outbound pairing lifecycle.
- The relay exposes health behavior while its connection route remains a starter. It does not yet
  verify CodeRoam tickets, enforce route roles, or forward bounded opaque pairing frames.
- Flutter already has secure storage, native OIDC, cryptography, and WebSocket dependencies. No
  Noise or QR runtime dependency is approved for M3 yet.
- An M2 environment/project can reference the preseeded M2 agent. Pairing a new agent must not
  silently rebind that existing environment or project.

## Design

### Trust boundaries and attacker-controlled data

The authenticated user controls the mobile initiation. The control plane is authoritative for the
account-to-device and account-to-agent ownership records and for one-use attempt state. The relay is
trusted to enforce signed ticket metadata and bounds, but it is not trusted with the pairing secret,
Noise plaintext, or either private identity. The agent and mobile trust a peer only after the
XXpsk3 transcript proves knowledge of the one-use secret and yields the exact expected static public
key.

Attacker-controlled inputs include QR/manual text, pairing identifiers, public keys, fingerprints,
protocol versions, endpoint metadata, tickets, WebSocket frames, ordering, timing, reconnects, and
duplicate confirmation requests. Every boundary must reject unknown fields where ambiguity is
unsafe, enforce exact byte and text limits before allocation, use fixed deadlines, and return
credential-free errors. Pairing secrets, bootstrap credentials, private keys, OIDC tokens, Noise
payloads, and transcript material must never enter logs.

### Identity and canonical fingerprints

Agent and mobile identities use X25519 static key pairs. First creation is an explicit lifecycle
operation. Agent private material is written atomically with owner-only permissions and the agent
runs as a non-root user. Mobile private material is stored through Keychain/Keystore-backed secure
storage. A missing, truncated, malformed, or permission-invalid identity after initialization fails
closed; neither endpoint silently regenerates or replaces a pinned identity. Account logout removes
account credentials but does not implicitly rotate the device identity.

Fingerprint version 1 is:

```text
x25519-sha256:<64 lowercase hexadecimal characters>
```

The hexadecimal value is SHA-256 over the exact 32 raw public-key bytes. All producers compute this
value locally, all consumers recompute it before comparison, and no submitted fingerprint is
trusted as an independent fact. Database migrations backfill and constrain the existing columns to
this representation after a collision and malformed-key preflight.

### Pairing attempt and secret lifecycle

The agent creates a 128-bit random pairing ID and a 128-bit random pairing secret. The manual
representation is unpadded uppercase Base32 and therefore 26 characters; UI may group characters
for readability but parsing removes only the documented separators. The secret exists only in agent
memory, the locally rendered QR/manual value, and mobile memory during the attempt. It is never
submitted to the control plane or relay and is never stored in PostgreSQL, Redis, outbox payloads,
crash reports, or logs.

The unauthenticated agent bootstrap endpoint accepts only bounded agent metadata, the pairing ID,
the canonical agent public key/fingerprint, protocol version, and expiry. It is protected by
per-source and per-key rate limits. The control plane returns:

- a server-generated 256-bit bootstrap credential, of which only a domain-separated hash is
  persisted;
- a short-lived, purpose-bound signed agent pairing ticket; and
- the exact relay region and endpoint.

The attempt expires after approximately five minutes. Relay tickets live no longer than 60 seconds
and tolerate only a small explicit clock skew. A signed-in mobile claims the public attempt using
the pairing ID and QR-bound public metadata, registers its candidate device metadata and public key,
and receives the opposite-role client ticket. The claim does not prove possession of the pairing
secret; only the subsequent XXpsk3 handshake does.

### Ticket signing and pairing-only relay

The control plane signs exact serialized Protobuf ticket claims with an Ed25519 key held in its
secret store. Claims include key ID, ticket ID, route ID, purpose, role, endpoint ID, relay region,
protocol version, issued/not-before/expiry timestamps, and a one-use nonce. `PAIRING` and future
`SESSION` purposes are distinct and cannot be substituted. Relay deployments pin the current
verification key and may temporarily accept one previous key during an explicit rotation.

Tickets are sent in an authorization header, never in a URL or log field. The relay validates the
signature, purpose, region, role, time window, route, and replay nonce before upgrade. Redis holds
only bounded, expiring replay/routing state; it is not an ownership or durable authorization source.

M3 adds a pairing-only WebSocket path. A route admits exactly one agent and one mobile carrying
opposite-role tickets for the same pairing attempt. It forwards the three Noise XX messages in the
fixed direction sequence client-to-agent, agent-to-client, client-to-agent. Frames are opaque binary
payloads, individually bounded to 4 KiB, and the whole route expires within 30 seconds. A duplicate
role, extra frame, wrong direction, malformed frame type, slow consumer, oversize message, expired
ticket, nonce replay, or disconnect closes the route. M3 offers no reconnect or resume; the agent
starts a fresh attempt with a fresh secret after interruption.

### Noise XXpsk3 and peer binding

The mobile initiates and the agent responds using
`Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s`. The prologue domain-separates CodeRoam pairing version 1
and binds the protocol version, pairing ID, canonical agent fingerprint, and endpoint roles.
Authenticated payloads carry bounded CodeRoam pairing data only. The recovered static keys must
exactly match the QR-bound agent candidate and the signed-in mobile candidate submitted to the
control plane.

M3 uses the handshake to establish and confirm pins, then discards the Noise transport cipher
states. It does not send normal application frames over this route. IK session setup, transport
nonces, rekeying, channel multiplexing, flow control, replay/resume, and reconnect remain M4
responsibilities.

An interoperability gate precedes the production implementation. It must run official Noise test
vectors plus a deterministic Go-to-mobile XXpsk3 transcript, including wrong-PSK, wrong-key,
wrong-prologue, truncation, order, replay, and oversize failures. The current Dart package candidate
has limited adoption and does not expose an obvious XXpsk3 convenience path. The preferred
evaluation is a mature Go Noise implementation paired with a maintained native mobile core through
Flutter FFI, but no Noise, FFI, QR, or release-action runtime dependency is added without explicit
approval.

### Two-sided confirmation and atomic registration

After the handshake, each endpoint derives the same bounded, domain-separated channel-binding value
from the completed handshake hash and observed peer identities:

- the authenticated mobile submits its confirmation using OIDC;
- the agent submits its confirmation using the bootstrap credential.

The session module owns the attempt state and locks one attempt while validating expiry, version,
claimed owner, expected agent key, device candidate, confirmation values, and replay state. Final
completion calls narrow device-owned and workspace-owned registration interfaces inside the same
caller-owned PostgreSQL transaction. Authorization does not use cross-schema SQL. The transaction
either consumes the attempt and creates both owner-bound registrations or creates neither.

Retries with the same valid confirmation are stable. A missing peer confirmation leaves an expiring
pending attempt. A mismatch, replay, expired attempt, revoked candidate, foreign owner, or reused
bootstrap credential fails closed and creates no ownership. Local pins are persisted atomically
only after the endpoints observe the consumed attempt; an unknown network outcome is reconciled by
polling the same attempt and never by generating a new local identity.

An already-active device may pair another agent for the same owner only when its device ID, public
key, and canonical fingerprint all match the persisted row. A fingerprint or key collision is a
hard failure. A registered agent identity cannot move between owners. Revocation is irreversible
for that registration and never revives a row; re-pairing a revoked physical endpoint requires an
explicit new identity and registration.

### Persistence ownership and migrations

Each schema remains single-writer:

- `session` owns the pairing attempt, claim, bootstrap-credential hash, confirmation state, and
  consumption transition;
- `device` owns mobile identity registration, listing, and revocation;
- `workspace` owns agent identity registration, listing, and revocation;
- `auth` remains the only owner of OIDC-to-user identity;
- `outbox` remains the only writer of durable external-side-effect records.

The session migration adds the canonical agent candidate, bounded metadata, protocol version,
bootstrap-credential hash, claimed user/device candidate, endpoint confirmations, timestamps, and
explicit state needed by the attempt transition. It stores no pairing secret or Noise plaintext.
Device and workspace migrations backfill canonical fingerprints from the existing raw public keys,
preflight duplicates, and add equality/shape constraints. There are no cross-schema foreign keys.
Migrations are transactional and forward-compatible with the merged M2 binaries, which do not make
trust decisions from the fingerprint text.

Pairing completion does not require a durable external notification because both endpoints can
reconcile the consumed attempt. Existing revocation outbox events remain metadata-only. Expiry is an
authorization rule enforced on every access; physical cleanup may be lazy or scheduled through an
owner-approved path without turning the worker into a second schema writer.

### Mobile and agent lifecycle

The agent command explicitly initializes or loads its identity, creates one bounded attempt, renders
QR and manual material locally, connects outbound to the selected relay, completes XXpsk3, confirms
the attempt, persists the mobile pin after consumption, and exits or enters its future runtime
state. Signals, deadline expiry, and network failures erase in-memory secret material and close the
attempt without replacing the identity.

Flutter restores the stable device identity before authenticated project loading. The pairing
surface supports camera scanning and manual entry, displays the agent name and canonical
fingerprint for confirmation, rejects malformed/expired/unsupported payloads locally, performs the
pairing lifecycle, and stores the agent pin only after consumed status. Paired device/agent lists
show identity and revocation state without displaying secrets.

Once paired identity restoration and rollback compatibility are proven, the app stops requiring
`CODEROAM_DEVICE_ID`. During rollout, the old M2 selector may remain as a temporary compatibility
path but cannot override a conflicting paired identity.

Pairing a new agent creates an owner-bound agent registration only. It does not automatically
reassign the preseeded M2 environment/project. Environment attachment remains an explicit,
authorized action or controlled test fixture until its owning API is approved.

### Signed agent distribution

CI builds non-root Linux agent binaries for at least `linux/amd64` and `linux/arm64`, publishes
checksums, and attaches verifiable build provenance using an approved GitHub artifact-attestation
or Sigstore flow. External actions are pinned to immutable commit SHAs. Installation documentation
verifies the artifact before placing it under a dedicated OS user with owner-only identity storage.
M3 does not add autonomous update, privileged installation, or production runbook execution.

## Milestones

1. Freeze fingerprint, identifiers, expiry, size, and ordering rules; complete the cross-language
   XXpsk3 interoperability and dependency evaluation; obtain explicit dependency approval.
2. Evolve Protobuf contracts additively and implement purpose-bound Ed25519 ticket signing and
   verification with compatibility, malformed-input, and rotation tests.
3. Implement explicit agent and mobile identity initialization/restoration with permission,
   corruption, missing-key, logout, and no-silent-replacement tests.
4. Add forward-compatible session/device/workspace migrations and module-owned pairing,
   registration, listing, and revocation repository behavior.
5. Build verifiable Linux agent artifacts and implement the bounded agent bootstrap/attempt
   lifecycle without transmitting the pairing secret.
6. Implement the pairing-only relay route, ticket replay protection, direction/state enforcement,
   resource bounds, and failure cleanup.
7. Implement the cross-language XXpsk3 handshake, peer-key checks, two-sided channel binding, and
   retry-stable confirmation.
8. Add the mobile scan/manual pairing experience and lifecycle restoration while preserving the M2
   compatibility path.
9. Complete atomic single-owner registration, paired endpoint listing/revocation, and remove the
   required build-time device selector.
10. Run end-to-end failure injection, security review, full repository validation, signed-artifact
    verification, and physical iPhone acceptance before recording M3 closure.

Each implementation slice should remain reviewable: normally two or three production functions plus
their focused unit or integration tests. A slice must pass its narrow checks before the next slice
starts. Protocol generation, migrations, or cross-deployable behavior are split at explicit
compatibility seams rather than delivered as one large file replacement.

## Progress

- [x] Inspect the merged M2 implementation, applicable instructions, schemas, contracts, and trust
  boundaries.
- [x] Define the M3 ExecPlan and create `feat/m3-agent-bootstrap-pairing` from `origin/main`.
- [x] Implement and test the canonical X25519 fingerprint codec without adding a dependency.
- [ ] Complete the XXpsk3 Go/mobile interoperability spike and dependency risk report.
- [ ] Obtain explicit approval for Noise/mobile FFI, QR, and release-attestation dependencies.
- [ ] Add and regenerate the additive M3 Protobuf contracts.
- [ ] Implement purpose-bound Ed25519 pairing ticket signing and verification.
- [ ] Implement fail-closed agent identity creation and restoration.
- [ ] Implement fail-closed mobile identity creation and restoration.
- [ ] Add session-owned pairing-attempt migration and persistence.
- [ ] Backfill and constrain device/workspace canonical fingerprints.
- [ ] Implement device-owned and workspace-owned registration/listing boundaries.
- [ ] Implement signed agent artifact generation and verification documentation.
- [ ] Implement bounded agent bootstrap and outbound pairing lifecycle.
- [ ] Implement pairing-only relay admission, routing, replay, and cleanup.
- [ ] Implement cross-language XXpsk3 pairing and two-sided confirmation.
- [ ] Implement Flutter QR/manual pairing and retry/reconciliation UX.
- [ ] Complete atomic registration and revocation behavior.
- [ ] Remove the required M2 build-time device selector after compatibility verification.
- [ ] Run adversarial security review and address actionable findings.
- [ ] Pass the full repository, infrastructure, release, and physical-device acceptance gates.
- [ ] Record final M3 acceptance and all explicitly deferred coverage.

## Decisions

- 2026-07-24: Keep M3 limited to bootstrap, pairing, pinning, registration, listing, and revocation.
  Discard pairing transport cipher states after XXpsk3. Normal IK sessions, multiplexing, flow
  control, reconnect/resume, and application data remain M4.
- 2026-07-24: Use one canonical fingerprint:
  `x25519-sha256:` followed by lowercase hexadecimal SHA-256 of the exact 32-byte X25519 public key.
  Recompute it at every trust boundary and backfill persisted M2 rows before constraining them.
- 2026-07-24: Use a 128-bit random opaque pairing ID and a separate 128-bit random unpadded Base32
  secret. The secret never crosses the control-plane or relay boundary. Attempts expire in about
  five minutes, relay tickets in at most 60 seconds, and a live route in at most 30 seconds.
- 2026-07-24: Let the mobile initiate and the agent respond to
  `Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s`. Bind protocol version, pairing ID, agent fingerprint, and
  roles through a CodeRoam-specific prologue and reject any recovered peer key mismatch.
- 2026-07-24: Require an interoperability and dependency-approval gate before production Noise,
  Flutter FFI, QR, or release-action dependencies. Prefer established cryptographic implementations
  over implementing Noise primitives locally.
- 2026-07-24: Sign exact pairing ticket claims with Ed25519 and separate pairing purpose from future
  session purpose. Keep signing material only in the control plane and verification keys in the
  relay; keep ticket/replay state short-lived and metadata-only.
- 2026-07-24: Complete ownership only after both endpoints submit the same channel binding and the
  session-owned attempt is consumed in the same transaction as device-owned and workspace-owned
  registration. No cross-schema SQL decides authorization or trust.
- 2026-07-24: Persist local pins only after consumed status is observed. Treat commit/network
  ambiguity as a retry/reconciliation problem and never rotate identity automatically.
- 2026-07-24: Do not automatically attach a newly paired agent to the existing M2
  environment/project. That requires a separate explicit owner-authorized action.
- 2026-07-24: Ship verifiable Linux agent artifacts with checksums and approved provenance. Keep
  installation non-root and defer self-update.

## Validation

Run the narrowest relevant checks after every slice, then the owning module gate. Required M3
coverage includes:

- fingerprint golden vectors shared across Go, Dart, SQL backfill, Protobuf, and UI; reject wrong
  length, prefix, case, characters, hash, key type, and collision;
- `buf lint`, `buf breaking --against '.git#branch=main'`, canonical generation, clean generated
  diff, and mixed-version producer/consumer tests;
- official Noise vectors and deterministic Go-to-mobile XXpsk3 transcripts; reject wrong secret,
  peer key, prologue, role, version, order, replay, truncation, trailing data, oversize frames, and
  post-completion messages;
- signing/verification tests for exact claims, purpose separation, algorithm/key confusion,
  unknown/previous key IDs, expiry, not-before, region, role, route, nonce replay, and mutation;
- migration tests for forward application, repeat application, transaction rollback, canonical
  backfill, malformed legacy rows, duplicate fingerprints, and concurrent completion;
- repository/application tests for foreign owner, revoked endpoint, mismatched keys, expired
  attempt, missing confirmation, duplicate confirmation, unknown commit outcome, and idempotent
  reconciliation;
- relay tests for duplicate roles, mismatched tickets, slow consumers, disconnects, expired routes,
  bounded queues, cancellation, deadline cleanup, Redis loss, and credential-free logs;
- agent tests for permissions, atomic creation, first-run races, missing/corrupt identity, no silent
  replacement, signal cleanup, bounded retries, and zero secret leakage;
- mobile tests for secure-storage restoration, logout, biometric/platform failure where applicable,
  QR/manual parser bounds, retry states, cancellation, and no native-platform-view construction in
  ordinary unit tests;
- release tests for `linux/amd64` and `linux/arm64`, checksum verification, provenance verification,
  immutable action pins, and a non-root smoke install;
- race detector and focused fuzz/property tests for parsers, tickets, state machines, and frame
  boundaries;
- infrastructure smoke with PostgreSQL, Redis, control plane, relay, agent, and authenticated mobile
  integration; verify PostgreSQL/Redis/outbox/logs contain no pairing secret, private key, token,
  Noise plaintext, source, or terminal payload;
- `make bootstrap`, `make proto`, `make fmt`, `make lint`, `make test`, `make build`, the repository
  infrastructure gate, `govulncheck` for every Go module, `shellcheck`, and `actionlint`;
- an independent security review after implementation, with every actionable finding fixed or
  explicitly dispositioned before closure.

Physical acceptance uses the production-shaped ZITADEL, Cloud Run/control-plane, relay, and a signed
agent on user-controlled Linux:

1. Verify and install the signed artifact as a non-root user.
2. Start pairing and complete QR pairing from an iPhone.
3. Repeat with the manual secret path.
4. Confirm both sides display the same canonical fingerprints and paired ownership.
5. Restart the agent and app and confirm the identities and pins are unchanged.
6. Interrupt before each handshake message and before each confirmation; confirm expiry/retry
   creates no partial registration and never replaces identity.
7. Revoke the device, then the agent, and confirm later authorization/pairing attempts fail closed.
8. Inspect bounded operational metadata and confirm no sensitive material was logged or persisted.

Additional physical phone/tablet coverage is recorded explicitly at closure. It is not claimed from
an iPhone-only run.

## Recovery and rollback

- Deliver focused commits at protocol, migration, deployable, and client compatibility seams.
- Additive Protobuf fields/messages remain readable by M2 consumers; never reuse or renumber a
  field. Roll back producers before removing any compatibility path.
- Device/workspace fingerprint migrations are forward fixes. Old M2 binaries ignore fingerprint
  text, so application rollback remains possible after backfill. Do not restore noncanonical
  fingerprints merely to mimic old placeholder data.
- A control-plane or relay rollback leaves new attempts to expire harmlessly. Pairing credentials
  and tickets are short-lived and cannot become durable authorization.
- Rotate a compromised ticket signing key by deploying the new relay verification key before the
  control-plane signer, retaining only the explicitly bounded previous verification key window.
- A half-finished handshake or one-sided confirmation creates no device/agent registrations. Retried
  identical confirmations reconcile; mismatches remain denied.
- Never delete or regenerate a local identity as rollback. Preserve it, surface the failed state,
  and require an explicit operator/user rotation flow.
- Redis loss invalidates live pairing routes but cannot alter durable ownership. The agent creates a
  fresh attempt and secret.
- Keep the M2 build-time selector until restored pairing identity is proven in both upgrade and
  rollback tests. Remove it only in its own reviewable slice.
- Development Docker and cloud smoke resources must be disposable and cleaned up after validation;
  production changes remain manual and user-authorized.

## Open risks

- The mobile Noise implementation is not approved. The visible pure-Dart candidate has limited
  adoption and its direct XXpsk3 support needs proof. A maintained native core through Flutter FFI
  may be safer but adds build, audit, and platform complexity.
- Camera scanning/rendering and release provenance may add dependencies or external GitHub actions.
  Their exact packages and immutable versions require explicit approval before implementation.
- An unauthenticated bootstrap endpoint and pairing relay route create denial-of-service pressure.
  Rate limits, admission bounds, timeouts, queue caps, and metrics need failure-injection evidence.
- Keychain/Keystore accessibility, backup/restore, uninstall/reinstall, and device-lock behavior
  differ by platform and require physical-device validation.
- A newly paired agent is owner-registered but not automatically attached to the existing M2
  environment/project. The product must keep this state understandable until an explicit attachment
  flow is approved.
- The M3/M4 boundary must remain strict: no general session ticket, IK transport, reconnect/resume,
  or application frame may leak into the pairing route.
- Release provenance verification depends on external tooling and network availability; closure
  must distinguish locally verified artifacts from unavailable third-party checks.
- iPhone-only acceptance cannot prove Android or tablet platform behavior. Any deferred physical
  matrix must be stated explicitly rather than treated as passed.
