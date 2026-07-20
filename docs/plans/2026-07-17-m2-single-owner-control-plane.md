# Milestone 2 Single-Owner Control Plane

## Objective

Implement the MVP control plane for authenticated single-owner resources: users, devices, agents,
environments, projects, sessions, and transactional outbox processing. Every resource belongs
directly to one user, and authorization fails closed when ownership or device/agent trust cannot be
proven.

## Scope

This plan covers the Go control-plane modular monolith, its PostgreSQL schemas and migrations, the
worker's outbox consumption, REST/JSON contracts with OpenAPI, focused integration tests, and the
minimum Flutter client integration required to exercise M2 behavior.

It does not implement organizations, teams, memberships, generic roles/policies, device or agent
bootstrap, pairing handshakes, relay routing, Noise sessions, workspace filesystem/PTY operations,
or GitHub integration. Those remain assigned to later milestones.

## Current state

- M0 and M1 are complete and merged.
- The control plane exposes public `/healthz` plus authenticated `/v1/projects` and `/v1/sessions`
  through the composed OIDC, repository, and application-service graph.
- Starter SQL runs through the transactional migration ledger, and the auth module now owns a
  PostgreSQL repository for validated users with typed duplicate and not-found failures. It also
  owns exact, case-sensitive OIDC issuer/subject bindings without treating email as identity.
- The auth application service accepts bounded opaque evidence only through an injected verifier,
  resolves verified OIDC claims through the exact auth-owned issuer/subject binding, re-resolves
  the user in PostgreSQL, and issues an opaque actor only after both identity checks match.
- The device module now validates mobile identity metadata and X25519 public keys, binds new devices
  to an authenticated actor, authorizes only an active owner-scoped persisted row, denies
  authorization after irreversible revocation, and can atomically persist revocation with a
  metadata-only outbox event.
- The workspace module now authorizes an active owner-scoped persisted agent inside a caller-owned
  bounded transaction, holds a shared row lock through that caller's commit or rollback, and can
  atomically persist owner-scoped revocation with a metadata-only outbox event. It also authorizes
  persisted project ownership only when the project environment references the exact requested
  agent, holding shared project/environment locks in that same caller-owned transaction.
- The outbox module now exposes closed metadata-only event kinds and can enqueue a fixed empty JSON
  event only inside a caller-owned PostgreSQL transaction.
- The session module now defines bounded owner/device/agent/project/region metadata without exposing
  an unsigned placeholder relay ticket or carrying secrets and engineering payloads. Its service
  authorizes the active device, active agent, and exact agent-bound project, then idempotently
  persists the metadata in one bounded transaction using a caller-stable opaque session ID.
- The control-plane runtime opens and verifies a bounded PostgreSQL pool before serving, drains HTTP
  requests during shutdown, and closes the pool after the server stops.
- The worker runtime owns a bounded PostgreSQL pool and drains retry-safe, duplicate-delivery-safe
  claim/handler/finish transactions until graceful cancellation.
- The worker runtime now drains the bounded outbox processor with the closed M2 revocation handler,
  retry-safe transaction ownership, and graceful pool shutdown. Relay-session termination remains
  an M4 replacement because M2 creates no relay session.
- Production authentication uses generic OIDC Authorization Code with PKCE: the mobile app has no
  client secret, the backend validates issuer, audience, signature, time claims, and exact subject,
  and registered device identity remains separate from account identity.
- The provider-neutral HTTP boundary accepts one Bearer credential as opaque verification evidence,
  exposes only a verified nonzero actor to handlers, and returns fixed credential-free JSON errors.
  The production composition root now owns one OIDC verifier/cache and activates the M2 routes only
  after every repository and service constructor succeeds.
- The authenticated `GET /v1/projects` and metadata-only `POST /v1/sessions` routes share one
  fail-closed runtime handler composition and an OpenAPI 3.1 contract.
- Flutter now has strict metadata models, a same-origin bounded HTTPS transport, a provider-neutral
  authenticated REST repository, touch project/session surfaces, generic native OIDC Authorization
  Code with PKCE, secure token persistence, and a lifecycle-owned production composition shell. The
  M2 build takes a pre-registered device ID as a non-secret bootstrap selector until M3 pairing
  supplies registered and pinned device identity.

## Design

The control plane remains one Go deployable with internal modules. Each module is the sole runtime
writer to its PostgreSQL schema and exposes narrow application interfaces to other in-process
modules. Cross-schema SQL may support read models only; authorization and trust decisions call the
owning module.

Transport handlers validate JSON and authenticated actor context, then call application services.
Domain constructors reject malformed or unbounded values before persistence. Repository methods
use context-aware queries and translate expected uniqueness/not-found conditions into domain
errors. Device/agent revocation and every durable external side effect write an outbox event in the
same PostgreSQL transaction as the state change.

Session issuance will require the same owning user across the requested project, active device, and
active agent. It will fail closed on missing, foreign-owned, or revoked resources. M2 persists only
short-lived session/ticket metadata; E2E keys, pairing secrets, source code, terminal data, prompts,
and engineering payloads never enter PostgreSQL, Redis, outbox payloads, or logs.

Redis remains limited to later ephemeral ticket/replay/routing state. M2 will not use it as an
authorization source or durable store.

## Milestones

1. Establish the auth-owned user domain contract and negative validation tests.
2. Add the PostgreSQL migration ledger/test harness and the approved database driver boundary.
3. Implement auth persistence, application service, authenticated actor boundary, and REST/OpenAPI
   behavior.
4. Implement device trust, persisted authorization, and revocation with atomic outbox publication;
   defer registration/listing to the M3 pairing contract.
5. Implement workspace agents, environments, and projects with explicit single-owner checks.
6. Implement session authorization and short-lived ticket metadata using owning-module interfaces.
7. Implement bounded, retry-safe, duplicate-safe outbox claiming and worker processing.
8. Add the minimum Flutter integration and complete end-to-end ownership/revocation acceptance.
9. Run the full repository gate, adversarial review, migration recovery checks, and document M2
   acceptance.

## Progress

- [x] Inspect the existing M2 schemas, deployables, and trust boundaries.
- [x] Add the auth user domain contract and tests.
- [x] Approve and add the PostgreSQL runtime dependency.
- [x] Add the migration ledger primitive and PostgreSQL integration harness.
- [x] Route the starter migrations through the ledger.
- [x] Add auth repository integration coverage.
- [x] Add the auth application service and fail-closed actor boundary.
- [x] Add provider-neutral authenticated REST/OpenAPI behavior.
- [x] Add exact auth-owned OIDC issuer/subject identity bindings.
- [x] Add the fail-closed verified-claims-to-local-user adapter.
- [x] Add bounded explicit OIDC verifier trust-anchor configuration.
- [x] Add signed OIDC ID-token verification with bounded shared JWKS retrieval.
- [x] Load required OIDC and relay trust inputs through bounded runtime configuration.
- [x] Wire the approved OIDC authentication adapter into the control-plane runtime.
- [x] Add the device identity and revocation domain contract.
- [x] Add the metadata-only transactional outbox enqueue primitive.
- [x] Implement persisted device revocation with atomic outbox behavior.
- [x] Add the persisted active-device authorization boundary.
- [x] Defer device registration/listing persistence until the M3 fingerprint and pairing contract.
- [x] Implement agent, environment, and project domains plus persisted authorization/read behavior.
- [x] Add the workspace agent identity and revocation domain boundary.
- [x] Add the persisted active-agent authorization boundary.
- [x] Implement persisted agent revocation with atomic outbox behavior.
- [x] Add the workspace environment ownership domain boundary.
- [x] Add the workspace project ownership and registered-root metadata domain boundary.
- [x] Add the persisted owner- and agent-bound project authorization boundary.
- [x] Add the bounded session metadata domain boundary.
- [x] Persist validated session metadata inside a caller-owned transaction.
- [x] Implement session authorization and retry-stable session metadata.
- [x] Implement worker outbox processing.
- [x] Integrate the provider-neutral Flutter M2 surface.
- [x] Activate the control-plane composition root after authentication approval.
- [x] Activate the mobile composition root after account authentication and device bootstrap.
- [x] Run full repository validation for the completed M2 scope.
- [ ] Record M2 acceptance after mobile composition and real-provider login are exercised.

## Decisions

- 2026-07-17: Deliver M2 as small module-owned vertical slices with focused negative and integration
  tests rather than introducing all control-plane modules in one change.
- 2026-07-17: Keep authorization in owning application interfaces. Cross-schema SQL must not decide
  user ownership, revocation, or trust.
- 2026-07-17: Defer provider-specific authentication until the bootstrap mechanism is explicitly
  approved. Domain and persistence slices will not store credentials or accept an unverified user
  identifier from a request header as authentication.
- 2026-07-17: Keep user identifiers opaque outside the auth module and parse their canonical
  encoding at the boundary. Preserve verified email local-part case while normalizing only the
  domain; provider-specific identity association must not be inferred from generic email casing.
- 2026-07-17: Pin `github.com/jackc/pgx/v5` at `v5.10.0`. Keep the reusable migration mechanism in
  the domain-neutral `postgresx` package and its deployment-owned ledger in `coderoam_meta`, outside
  every application module's writable schema.
- 2026-07-17: Apply each migration and its ledger record in one transaction, serialize runners with
  a PostgreSQL advisory transaction lock, and reject checksum or name drift for an applied
  scope/version.
- 2026-07-17: Keep the repository migration catalog as an explicit allow-list of module scopes and
  directories. Run it through the bounded Go migration command so local `psql` is not required and
  the database DSN remains in the environment rather than process arguments.
- 2026-07-17: Starter DDL is intentionally strict and contains no `IF NOT EXISTS`. The ledger is the
  only repeat-application mechanism; an empty ledger plus a pre-existing module object fails closed
  instead of silently adopting an unverifiable M1 schema.
- 2026-07-17: Keep auth persistence behind the auth module's repository. Bind every SQL value,
  translate known ID/email uniqueness conflicts into typed errors, and revalidate stored rows
  through the domain constructor so malformed or noncanonical identity data fails closed.
- 2026-07-17: Let the control-plane process own one bounded PostgreSQL pool. Startup requires a
  successful ping under a deadline; shutdown drains the HTTP server before closing the pool; DSNs
  and user records are never logged.
- 2026-07-17: Keep provider verification behind an `IdentityVerifier` supplied by the composition
  root. The application service validates bounded evidence, maps rejected/zero/missing/mismatched
  identities to one unauthenticated result, sanitizes dependency failures, and issues an actor only
  after the verified ID exactly matches a repository user. A zero actor yields no usable user ID.
- 2026-07-19: Keep HTTP authentication provider-neutral: accept exactly one RFC 6750 Bearer token as
  opaque verification evidence, call the auth service, and store only its nonzero actor in a private
  request context key. Reject duplicate/coalesced or malformed credentials before verification and
  return fixed JSON error categories without evidence or dependency details. Runtime composition
  still requires the approved identity provider/bootstrap adapter.
- 2026-07-20: Use generic OIDC Authorization Code with PKCE for production account authentication.
  Mobile is a public client with no embedded client secret; the backend validates the configured
  HTTPS issuer, audience, signature, and temporal claims before resolving the exact case-sensitive
  `(issuer, subject)` pair through auth-owned persistence. Never infer account identity from email,
  and keep registered device identity and trust independent from the OIDC account identifier.
- 2026-07-20: Keep signed-token verification separate from local identity resolution. The OIDC
  adapter accepts only already-verified issuer/subject claims, reconstructs the bounded exact
  identity, maps an unlinked identity to the same rejection as an invalid token, and preserves
  dependency failures for the auth service to sanitize as unavailable.
- 2026-07-20: Configure OIDC verification with one exact HTTPS issuer, exact bounded audience,
  explicit HTTPS JWKS URL, and one allow-listed asymmetric signing algorithm. Preserve the issuer's
  exact spelling as its OIDC identity namespace; reject invalid hosts and empty, zero, or
  out-of-range ports rather than normalizing trust anchors into a different identifier.
- 2026-07-20: Accept only compact signed ID tokens whose protected `typ` is exactly `JWT`; reject
  access-token and ambiguous JWT types before identity verification. Pin one configured asymmetric
  algorithm, require RSA keys of at least 2048 bits or the exact ECDSA curve for the selected
  algorithm, and require a structurally valid public signing key in every accepted JWKS response.
  Filter disallowed keys before the verifier can use them. Bound exact-URL JWKS reads and share the
  validated response through a URL-and-algorithm-keyed cache with a 30-second refresh interval so
  attacker-selected unknown key IDs cannot amplify provider traffic or starve a cold verifier.
  Consult that cache on every signature verification so even a locally known key must be refreshed
  within 30 seconds; failed refreshes clear stale key material and fail closed.
- 2026-07-20: Require `OIDC_ISSUER`, `OIDC_AUDIENCE`, `OIDC_JWKS_URL`,
  `OIDC_SIGNING_ALGORITHM`, `RELAY_REGION`, and `POSTGRES_DSN` at control-plane startup. Preserve
  OIDC trust-anchor spelling exactly for the auth-owned validators, trim only the non-identity DSN
  and listen address, and reject an unbounded, control-bearing, or invalid numeric listen address
  before it can reach logs or the HTTP server.
- 2026-07-20: Construct one process-lifetime OIDC verifier and validated JWKS cache, exact-identity
  auth repository/service, device/workspace repositories, and metadata-only session service before
  registering any M2 route. Keep `/healthz` public; route `/v1/projects` and `/v1/sessions` through
  the same fail-closed actor boundary. Any invalid trust input or dependency leaves the handler nil
  and prevents `ListenAndServe`.
- 2026-07-17: Require an authenticated actor to register or revoke a device. Accept only canonical
  opaque IDs, bounded names, explicit iOS/iPadOS/Android platforms, initialized X25519 public keys,
  and nonzero server-owned pairing times. Active-device authorization requires the exact owning
  actor; revocation checks ownership first, preserves its first timestamp, and has no reactivation
  path.
- 2026-07-17: Device value copies share a private synchronized revocation state so a retained handle
  cannot authorize after another copy revokes it. The persistence slice must still make every trust
  decision from committed repository state; process-local handles are not a durable authorization
  source.
- 2026-07-17: Reuse the existing domain-neutral `cryptox` public-key type in the control plane. Defer
  fingerprint encoding to the persistence/pairing contract so this M2 slice does not invent M3 wire
  semantics.
- 2026-07-18: Keep all `outbox.events` insert SQL inside the outbox module and require callers to pass
  an existing `pgx.Tx`; a pool cannot satisfy the enqueue API. The initial event contract accepts
  only closed `EventKind` values, generates cryptographically random event IDs, requires typed
  aggregate IDs, stores only a fixed empty JSON object, and returns a typed duplicate-ID error. This
  slice changes no schema and requires no backfill or compatibility rollout; rollback removes both
  the caller's state change and event.
- 2026-07-18: Persist device revocation by locking the active row with both device and authenticated
  owner IDs, deriving the timestamp from an injected server clock, and committing `revoked_at` plus
  one `device.revoked.v1` event in a nested caller-owned PostgreSQL transaction. Missing, foreign,
  and malformed-owner rows use the same access-denied result. Repeated revocation commits no change
  and emits no second event. Repository work has a fixed maximum deadline while preserving earlier
  caller deadlines. Pre-commit failures roll back atomically; commit errors have unknown outcome and
  require the same idempotent retry. This slice changes no schema; registration remains deferred
  rather than inventing the M3 public-key fingerprint encoding.
- 2026-07-18: Authorize a persisted device only inside the caller's existing `pgx.Tx`, querying the
  device-owned schema with canonical device and authenticated owner IDs plus `revoked_at IS NULL`
  and holding a shared row lock until the caller finishes the transaction. This lets the future
  session service write ticket metadata in the same bounded transaction, linearizing issuance with
  revocation. Revalidate bounded stored identity fields through the device domain constructor;
  missing, foreign, revoked, future-paired, and corrupt rows share one access-denied result. The
  method performs no mutation or outbox write and ignores fingerprint data until M3 defines its
  encoding and proof semantics.
- 2026-07-18: Establish the workspace-owned agent domain before persistence or pairing. Bind a
  canonical opaque agent ID, bounded name and software version, and initialized X25519 public key to
  an authenticated owner. Keep authorization fail-closed for zero/foreign actors and share private
  synchronized revocation state across value copies. Revocation is owner-only, irreversible, and
  idempotent, with server-owned creation/revocation times. Defer fingerprint encoding, bootstrap,
  pairing proof, key pinning, persistence, and relay/session behavior to their owning slices.
- 2026-07-18: Authorize a persisted agent only inside the caller's existing `pgx.Tx`, querying the
  workspace-owned schema by canonical agent and authenticated owner IDs with `revoked_at IS NULL`
  and holding `FOR SHARE` until the caller finishes. Revalidate bounded stored name, version,
  X25519 public-key bytes, and creation time through the agent domain; missing, foreign, revoked,
  future-created, or corrupt rows share one access-denied result. The query has a fixed maximum
  deadline, performs no mutation, and does not trust fingerprint or last-seen metadata. This slice
  changes no schema, index, backfill, outbox, or Redis state. Future session issuance must authorize
  device, agent, and project and persist ticket metadata in this exact bounded transaction, with
  rollback guaranteed on every exit.
- 2026-07-19: Persist agent revocation by locking the owner-scoped row with `FOR UPDATE`, deriving
  the timestamp from the server clock, and committing `revoked_at` plus one fixed-empty-payload
  `agent.revoked.v1` event in the same bounded PostgreSQL transaction. Missing, foreign, and
  malformed-owner rows share the access-denied result; future-created rows fail closed. Repeated
  calls preserve the first timestamp and emit no second event. Pre-commit failures roll back both
  writes, while a commit error has an unknown outcome and requires the same idempotent retry. The
  exclusive row lock linearizes with persisted authorization's shared lock. This slice changes no
  schema, index, backfill, Redis state, transport behavior, or worker processing.
- 2026-07-18: Bind each environment directly to one authenticated owner and one active agent owned
  by that actor. Validate canonical environment IDs plus bounded display/provider metadata without
  inventing a provider taxonomy; the provider label is not authorization input. Keep stable
  environment ownership distinct from current agent trust: agent revocation does not erase the
  owner's environment record, while future session issuance must still authorize the persisted
  agent and project separately. This slice changes no schema and exposes no transport behavior.
- 2026-07-18: Bind each project directly to an authenticated owner through an environment already
  owned by that actor. Validate canonical project IDs, bounded display metadata, a canonical
  absolute POSIX root narrower than `/`, and creation times no earlier than the environment. Treat
  the registered root only as control-plane metadata: M5 must still confine project-relative paths
  against traversal, symlinks, races, and filesystem changes at use time. Stable project ownership
  does not imply current agent trust or session access. Repository URL, last-opened metadata,
  persistence, transport behavior, and runtime filesystem authorization remain deferred to their
  owning contracts.
- 2026-07-19: Authorize a persisted project only inside the caller's existing `pgx.Tx`, requiring
  the project, environment, and exact requested agent to share the authenticated owner. Revalidate
  bounded environment/project metadata and creation ordering, reject future or corrupt rows, and
  hold `FOR SHARE` on the project and environment until the caller finishes. Keep stable project
  ownership separate from active-agent trust: the session service must call `AuthorizeAgent` first,
  then `AuthorizeProject`, and persist ticket metadata in that same bounded transaction. The method
  is read-only, ignores repository URL and last-opened metadata, and changes no schema, index,
  backfill, outbox, Redis, transport, or filesystem behavior.
- 2026-07-19: List at most 100 owner-scoped project summaries in deterministic newest-first order,
  exposing only opaque project/environment/agent IDs plus bounded display metadata and creation
  time. Listing represents stable ownership, so a revoked agent does not erase the project; session
  start still reauthorizes the active agent. Owner-constrained left joins make orphaned or
  cross-owner hierarchy corruption fail the whole read without exposing foreign metadata. Every
  stored field and finite timestamp is revalidated through the environment/project domains.
- 2026-07-19: Expose the project read model through authenticated `GET /v1/projects` with one strict
  optional `limit`, fixed JSON failures, and no project root/provider or other engineering metadata.
  Revalidate every summary and response count at the serialization boundary even though the owning
  repository already validates storage. Keep the OpenAPI 3.1 contract next to the deployable and
  lint it independently; do not mount the handler until a real identity verifier is composed.
- 2026-07-19: Define session metadata as one authenticated owner plus canonical session, device,
  agent, and project IDs, a bounded canonical server-selected relay-region label, and a server-owned
  start time. Keep every field private and expose no serialization or credential surface. A session
  domain value is not a relay ticket and contains no signature, nonce, expiry claim, E2E material,
  pairing secret, or engineering payload. The application layer must still perform all persisted
  authorization and storage in one transaction; M3/M4 retain ticket signing, replay protection,
  relay validation, and endpoint pairing.
- 2026-07-19: Expose session metadata to transport only through `MetadataFor` with the exact owning
  actor. The view contains canonical resource IDs, relay region, and start time only; foreign or zero
  actors receive nothing, and no ticket, credential, cryptographic material, or capability is added.
- 2026-07-19: Accept metadata-session starts through authenticated `POST /v1/sessions` only as one
  bounded JSON object containing each canonical caller-stable resource ID exactly once. Reject
  duplicate/case-alias fields, unknown fields, trailing data, and mismatched service results. Return
  an explicit `metadata-only` marker—not a ticket—and distinguish commit outcome unknown so clients
  retry the same ID and inputs. Keep the OpenAPI contract free of token/key/connection fields.
- 2026-07-19: Persist validated session metadata only through an existing `pgx.Tx`, under a fixed
  maximum deadline, without letting the repository begin or finish the transaction. Map the
  session primary-key conflict to a typed duplicate result, keep invalid/canceled calls SQL-free,
  and preserve rollback/commit visibility. Emit no outbox event because this slice creates no relay
  credential or external side effect. Keep `Create` as the low-level typed-conflict primitive while
  session issuance uses the retry-safe `CreateOrGet` path after authorization in the same transaction.
- 2026-07-19: Start a metadata-only session only after the owning device, active agent, and exact
  agent-bound project authorize the same actor in one bounded transaction. Require a caller-stable
  canonical opaque session ID so a retry never invents a second logical session. Insert or return an
  existing row only when owner, device, agent, project, and server-selected region match exactly;
  foreign, mismatched, ended, or corrupt reuse fails closed. Return a typed commit-outcome-unknown
  error on every commit failure and require retry with the same ID and inputs. Device and agent
  shared authorization locks remain held through metadata commit, linearizing issuance against
  exclusive revocation. This creates no ticket, credential, outbox event, Redis state, or payload.
- 2026-07-19: Establish the worker's closed outbox delivery boundary before adding SQL claiming.
  Accept only canonical opaque event and aggregate IDs, exact device/agent revocation kind pairs,
  an exactly empty JSON payload, a nonzero availability time, and a nonnegative attempt count.
  Reject unknown, mismatched, malformed, or payload-bearing rows without exposing their contents.
- 2026-07-19: Claim at most one due outbox row inside a caller-owned transaction using ordered
  `FOR UPDATE SKIP LOCKED`, so concurrent workers never handle the same locked delivery. Bound text
  in SQL before it crosses the process boundary and use the locked transaction-local row locator to
  quarantine malformed metadata without loading an oversized ID. Finish only through closed
  complete, fixed-delay retry, or permanent-discard outcomes; increment attempts without integer
  overflow and persist only fixed safe error classifications. The repository never owns commit or
  rollback, and the next processor slice must bound the complete claim/handler/finish transaction.
- 2026-07-19: Process one delivery per bounded transaction, holding the claimed row lock across an
  idempotent deadline-aware handler and the closed finish update. A handler timeout or retryable
  failure schedules a fixed-delay retry, a permanent failure or fifth failed attempt terminates the
  row with a fixed safe classification, and invalid metadata never reaches a handler. A crash after
  an external side effect but before commit intentionally redelivers the row, so handlers must be
  idempotent. Any commit error returns an outcome-unknown result; raw handler errors are neither
  persisted nor returned as delivery metadata.
- 2026-07-19: Wire the worker runtime to a verified bounded PostgreSQL pool and one immediate-drain
  processing loop with fixed-delay backoff after empty claims or failures. Keep runtime logs static
  and metadata-free, close the pool after cancellation, and fail startup on an invalid processing
  flag. M2 acknowledges the closed revocation event kinds because relay sessions do not exist; M4
  must replace that acknowledgement with active-session termination before enabling relay sessions.
- 2026-07-20: Compose project and metadata-session routes behind one actor-authentication boundary.
  Authentication runs before route-specific method rejection, typed-nil dependencies fail closed,
  and the composition still requires a real verifier from the runtime bootstrap.
- 2026-07-20: Keep the Flutter control-plane client provider-neutral and metadata-only. Pin one HTTPS
  origin, reject redirects and unsafe headers, bound authentication evidence, deadlines, JSON depth,
  and response bodies, preserve caller-stable session requests after ambiguous outcomes, and never
  expose a relay ticket or capability.
- 2026-07-20: Treat every post-dispatch session-start 5xx response as outcome-unknown before parsing
  its body. A gateway can replace the backend response after a successful commit, so canonical,
  unavailable, malformed, duplicate-key, or empty 5xx responses must all retry the exact same
  session, device, agent, and project IDs; known non-5xx rejections remain fixed failures.
- 2026-07-20: Let the Flutter composition shell own repositories, transports, controllers, and
  trust-bound routes. Changing origin, device, evidence provider, transport factory, or removing the
  shell immediately revokes old controllers and closes transport state, then removes stale routes
  after the current frame so late responses cannot reopen an old workspace.

## Validation

Each slice will run its focused unit and PostgreSQL integration tests before the applicable package
and repository gates. The final gate includes:

```bash
make fmt
make lint
make test
make build
make test-infrastructure
```

Database slices additionally verify forward migration, repeated startup, uniqueness conflicts,
transaction rollback, outbox atomicity, duplicate delivery, and recovery after handler failure.
Security-sensitive slices receive an adversarial review before commit.

2026-07-17 auth-domain slice validation passed: 13 focused tests under the race detector, focused
`go vet`, repository formatting/lint/tests, and adversarial review. Review findings about externally
constructible user identifiers and unsafe whole-email lowercasing were fixed and re-reviewed with no
remaining findings.

2026-07-17 migration-ledger primitive validation passed: unit/race tests, `go vet`, dependency
verification, `govulncheck`, ShellCheck, the smoke-script regression test, and real PostgreSQL 17
integration coverage for ordering, repeat application, checksum drift, transactional rollback, and
recovery. The full Compose infrastructure gate and applicable repository formatting, lint, test,
and build gates also passed.

2026-07-17 starter-migration activation validation passed: loader and runner unit/race tests,
focused `go vet`, ShellCheck, repeat execution against PostgreSQL 17, exact verification of all
eight module scope/version ledger entries, runner integration coverage, and the full Compose
infrastructure gate. Repository formatting, lint, test, command build, deployable build, and
container build gates also passed.

2026-07-17 strict-baseline validation passed: a static guard covers every module migration, focused
unit/race tests and `go vet` passed, and PostgreSQL 17 integration proved that a pre-existing
unledgered schema returns `duplicate_schema`, creates no new table, and records no ledger entry.
Fresh and repeated repository migration runs, the full Compose infrastructure gate, and repository
formatting, lint, and test gates still pass.

2026-07-17 auth-repository and runtime-pool slice validation passed: 16 control-plane and 22
`postgresx` tests under the race detector, focused `go vet`, module verification, `govulncheck`,
ShellCheck, repository formatting/lint/test/build, and the full Compose infrastructure gate.
PostgreSQL 17 coverage exercises round trips, typed uniqueness conflicts, missing users, corrupt
rows, and transaction rollback without persistent test writes. Adversarial review found no remaining
security or database-lifecycle issues and verified clean SIGTERM shutdown.

2026-07-17 auth-service slice validation passed: 40 auth tests and 41 control-plane tests under the
race detector, focused and module `go vet`, module verification, `govulncheck`, repository
formatting/lint/tests, and the full Compose infrastructure gate. Negative coverage rejects malformed
or oversized evidence, verifier rejection, zero and missing identities, repository mismatches,
unsanitized dependency failures, nil context, and zero actors. A medium-severity review finding that
wrapped cancellation errors could leak dependency details was fixed by returning canonical context
sentinels; adversarial re-review found no remaining issues.

2026-07-17 device-domain slice validation passed: 16 device tests and 57 control-plane tests under
the race detector, focused and module `go vet`, module verification, `govulncheck`, repository
formatting/lint/tests/build, the control-plane container build, and the full Compose infrastructure
gate. Negative coverage rejects zero actors, malformed or oversized metadata, unsupported platforms,
uninitialized public keys, zero pairing times, foreign revocation, invalid revocation times, zero
devices, and authorization by foreign or revoked actors. A medium-severity review finding that a
copied device could retain active state was fixed with shared synchronized revocation state plus
retained-copy and concurrent regression tests; adversarial re-review found no remaining issues.

2026-07-18 outbox-enqueue slice validation passed: six outbox unit cases and two opaque-ID tests
under the race detector, focused `go vet`, module verification, vulnerability scanning, ShellCheck,
Bash syntax validation, PostgreSQL 17 integration, and the full repository format/lint/test/build
gates. Coverage proves that rollback leaves no event, commit preserves exactly one metadata-only
`{}` event, duplicate IDs return the typed conflict, invalid or canceled boundaries do not enqueue,
and cleanup deletes only a successfully committed random test ID. Adversarial review found two
medium issues in the initial draft: free-form metadata could carry secret-shaped values, and cleanup
could delete a pre-existing fixed-ID row after a failed commit. Closed event kinds, internally
generated typed IDs, and post-commit cleanup resolved both; re-review found no remaining issues.

2026-07-18 persisted-device-revocation slice validation passed: 18 focused device cases and 65
control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, ShellCheck, Bash syntax validation, PostgreSQL 17 integration, and the full
repository format/lint/test/build gates. Integration coverage proves owner-only row locking,
single-event commit, repeat idempotency, foreign/missing/corrupt-owner denial, future-pairing
failure, outbox-failure rollback and recovery, and exact transaction-scoped cleanup. Adversarial
review found an unbounded owner-row lock wait and an inaccurate claim that commit errors always roll
back. A fixed repository operation deadline, held-lock timeout/retry regression, and documentation
of outcome-ambiguous commits resolved both; re-review found no remaining issues.

2026-07-18 persisted-device-authorization slice validation passed: 19 focused device cases and 66
control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, ShellCheck, Bash syntax validation, PostgreSQL 17 integration, and the full
repository format/lint/test/build gates. Integration coverage proves owner-only active-row access,
uniform missing/foreign/revoked/future/corrupt denial, no authorization mutation or event, bounded
lock-wait recovery, and `FOR SHARE` authorization versus `FOR UPDATE` revocation linearization.
Adversarial review found no actionable issue. The future session slice must still authorize and
persist ticket metadata in the exact same bounded transaction, with guaranteed rollback on every
exit, before session issuance can be considered race-safe.

2026-07-18 workspace-agent-domain slice validation passed: 19 focused workspace cases and 85
control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, and the full repository format/lint/test/build gates. Negative coverage
rejects zero/foreign actors, malformed IDs, empty/invalid/control-character/oversized names and
versions, uninitialized keys, and invalid timestamps. Copy and concurrency coverage proves that
revocation is owner-only, irreversible, idempotent, and visible to every retained value. Adversarial
review found no actionable issue and confirmed that fingerprint, pairing proof, key pinning,
persistence, and relay/session trust remain explicitly deferred.

2026-07-18 persisted-agent-authorization slice validation passed: 63 focused workspace cases and
129 control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, ShellCheck, Bash syntax and smoke-harness regression checks, PostgreSQL 17
integration, and the full repository format/lint/test/build and Compose infrastructure gates.
Integration coverage proves owner-only active-row access, uniform missing/foreign/revoked/future/
corrupt denial, no authorization mutation, bounded lock-wait recovery, and `FOR SHARE`
authorization versus persisted revocation ordering. Adversarial review found no actionable issue.
The future session service must still persist ticket metadata and finish or roll back on every exit
inside the exact transaction used for device, agent, and project authorization.

2026-07-19 persisted-agent-revocation slice validation passed: 64 focused workspace cases, six
outbox cases, and 130 control-plane tests under the race detector, focused and module `go vet`,
module verification, vulnerability scanning, ShellCheck, Bash syntax and smoke-harness regression
checks, PostgreSQL 17 integration, and the full repository format/lint/test/build and Compose
infrastructure gates. Integration coverage proves owner-only locking, exact metadata-only event
commit, repeat idempotency, uniform foreign/missing/corrupt-owner denial, future-created failure,
outbox-failure rollback and recovery, bounded row-lock timeout and retry, and safe retry after an
outcome-ambiguous commit without changing the first timestamp or adding an event. Adversarial review
found one Low documentation gap around ambiguous commit outcomes; the explicit retry contract and
its regression test resolved it, with no code-level security or correctness defect found.

2026-07-18 workspace-environment-domain slice validation passed: 39 focused workspace cases and 104
control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, and the full repository format/lint/test/build gates. Negative coverage
rejects zero/foreign actors, zero/foreign/revoked agents, malformed IDs, invalid or oversized
display/provider metadata, and invalid creation times. Lifecycle coverage proves stable owner access
after later agent revocation without treating that ownership check as current agent/session trust.
Adversarial review found one Low issue: creation could predate the linked agent. Rejecting strictly
pre-agent timestamps plus same-instant and one-nanosecond-before regression coverage resolved it;
re-review found no remaining issue.

2026-07-18 workspace-project-domain slice validation passed: 61 focused workspace cases and 127
control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, and the full repository format/lint/test/build gates. Negative coverage
rejects zero/foreign actors and environments, malformed project IDs, invalid or oversized display
metadata, relative/root/noncanonical/control-character/invalid-UTF-8/oversized root paths, and
invalid creation times. Lifecycle coverage proves stable project ownership after later agent
revocation without treating it as current agent or filesystem authority. Adversarial review found
no actionable issue and confirmed that filesystem confinement, repository credentials, persistence,
and session authorization remain explicitly deferred.

2026-07-19 persisted-project-authorization slice validation passed: 65 focused workspace cases and
131 control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, ShellCheck, Bash syntax and smoke-harness regression checks, PostgreSQL 17
integration, and the full repository format/lint/test/build and Compose infrastructure gates.
Integration coverage proves exact owner/agent/project binding, uniform foreign/wrong-agent/missing/
future/corrupt denial, read-only behavior, stable ownership after agent revocation, bounded lock
wait and retry, and shared project/environment locks through caller commit. Adversarial review found
one Low test-harness cleanup gap that could retain locks after an early assertion; immediate bounded
rollback cleanup for every acquired transaction resolved it, and re-review found no remaining
authorization, transaction, or cleanup issue.

2026-07-19 session-metadata-domain slice validation passed: 14 focused session cases and 145
control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, and the full repository format/lint/test/build gates. Negative coverage
rejects zero actors, malformed resource IDs, empty/oversized/noncanonical relay-region labels, and
zero start times while proving exact ownership and UTC normalization. Adversarial review found no
actionable issue and confirmed the type exposes no token, serialization, signing, nonce, expiry,
secret, or engineering-payload surface.

2026-07-19 session-metadata-persistence slice validation passed: 15 focused session cases and 146
control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, ShellCheck, Bash syntax and smoke-harness regression checks, PostgreSQL 17
integration, and the full repository format/lint/test/build and Compose infrastructure gates.
Integration coverage proves transaction-local visibility, rollback removal, exact metadata-only
commit, typed duplicate rejection, SQL-free invalid and canceled calls, bounded unique-key
contention, and successful retry. Adversarial review found one Low fixture-ownership gap; proving
each random committed ID absent before registering exact-ID cleanup resolved it, and re-review found
no remaining persistence, transaction-ownership, or cleanup issue.

2026-07-19 atomic-session-start slice validation passed: 41 focused session cases and 172
control-plane tests under the race detector, focused and module `go vet`, module verification,
vulnerability scanning, ShellCheck, Bash syntax and smoke-harness regression checks, PostgreSQL 17
integration, and the full repository format/lint/test/build and Compose infrastructure gates.
Integration coverage proves one ordered transaction across device, agent, exact project, and
session repositories; uniform denial and rollback; device and agent revocation linearization; exact
retry reconciliation; and one-row recovery when a real commit succeeds but reports an error.
Adversarial review found a Medium ambiguous-commit retry risk and a Low broad fixture-cleanup risk.
Caller-stable IDs, owner-scoped exact `CreateOrGet`, a typed unknown-outcome result, the committed-
but-error regression, and exact typed cleanup resolved both; re-review found no remaining issue.

2026-07-19 worker-outbox-domain slice validation passed: 10 worker cases under the race detector,
focused `go vet`, module verification, and clean formatting/diffs. Negative coverage rejects invalid
event or aggregate IDs, unknown or mismatched event/aggregate kinds, nonempty payloads, zero
availability times, and negative attempt counts. No persistence, external side effect, logging,
runtime dependency beyond the existing opaque-ID primitive, or engineering-payload surface was
introduced.

2026-07-19 worker-outbox-repository slice validation passed: 12 focused worker cases under the race
detector, focused `go vet`, module verification, ShellCheck, Bash syntax, PostgreSQL 17 integration,
and the full Compose infrastructure gate. Integration coverage proves due-time ordering, future-row
skipping, concurrent `SKIP LOCKED` claims, fixed-delay retry, successful re-delivery, completed and
invalid-event terminal outcomes, transaction rollback recovery, attempt/error metadata, and bounded
handling of a 4 KiB malformed event ID. The first run exposed an assertion that compared retry time
to the original availability time; correcting the expected retry fixture resolved the test defect.

2026-07-19 worker-outbox-processor slice validation passed: 29 focused worker cases under the race
detector, focused `go vet`, ShellCheck, Bash syntax, PostgreSQL 17 integration, and the full Compose
infrastructure gate. Coverage proves success, no-work rollback, invalid/permanent/exhausted terminal
outcomes, fixed-delay retry and later success, cooperative handler timeout, exact transaction reuse,
commit-outcome ambiguity, crash-after-side-effect rollback and duplicate redelivery, and a real
committed-but-error outcome that is not processed twice. Raw handler error text never enters the
database. Adversarial review found and fixed non-finite availability timestamps: `-infinity` could
otherwise poison the ordered claim queue, while `+infinity` could remain falsely pending forever.
PostgreSQL integration proves both are discarded without invoking a handler and that the next valid
event proceeds. The worker module is also tidy and independently testable with `GOWORK=off`.

2026-07-19 worker-runtime slice validation passed: focused runtime and processor packages under the
race detector, focused `go vet`, module verification, `govulncheck`, ShellCheck, Bash syntax, Docker
image build, and the full PostgreSQL 17 Compose gate. Unit coverage proves strict configuration,
pool close after cancellation, immediate draining, clean shutdown after processor failure, and the
closed M2 handler boundary. Runtime integration proves the enabled worker opens its own pool,
claims a real revocation event, persists one successful attempt, and shuts down cleanly; the Compose
worker remains explicitly disabled while fixture-owning integration packages run.

2026-07-19 HTTP-authentication-boundary slice validation passed: 67 focused auth/transport cases
under the race detector, focused `go vet`, and 197 control-plane cases under the race detector plus
full control-plane vet, `govulncheck`, and module verification. Adversarial review found and fixed
coalesced/ambiguous Bearer evidence, typed-nil dependency panics, and padding-only tokens. The final
boundary rejects those cases before verification, exposes no credential/provider detail, and never
lets a zero actor reach a protected handler.

2026-07-19 project-list slice validation passed: 72 focused workspace cases under the race detector,
focused `go vet`, ShellCheck, Bash syntax, and the full PostgreSQL 17 Compose infrastructure gate.
Integration coverage proves deterministic limits, owner isolation, revoked-agent visibility without
session authorization, corrupt root rejection, and fail-closed orphaned/cross-owner hierarchies.
Adversarial review's silent-inner-join omission finding was fixed with owner-constrained left joins.

2026-07-19 projects-REST slice validation passed: 125 focused transport/workspace cases under the
race detector, focused `go vet`, `git diff --check`, YAML parsing, and Redocly CLI 2.13.0 validation
with no OpenAPI errors or warnings. Tests cover authenticated owner propagation, strict/default
limits, duplicate and unknown queries, fixed error mapping, typed-nil dependencies, bounded response
counts, invalid summary rejection, and absence of root-path output. Final adversarial review found no
remaining issue after serialization-boundary revalidation was added.

2026-07-19 session-metadata-view slice validation passed: 42 focused session cases under the race
detector, focused `go vet`, and clean formatting/diffs. Owner, foreign-owner, and zero-value coverage
proves the view exposes only canonical metadata to the exact actor. Security review found no issue
and confirmed ticket, key, and authorization material remain outside the boundary.

2026-07-19 sessions-REST slice validation passed: 103 focused transport/session cases under the race
detector, focused `go vet`, `git diff --check`, and clean Redocly CLI 2.13.0 OpenAPI validation.
Coverage proves authentication, strict media/body bounds, exact once-only JSON fields, canonical
stable IDs, returned-ID reconciliation, fixed failures, outcome-unknown retry signaling, and a
metadata-only response with no ticket. Adversarial review's duplicate-key and missing-contract
findings were fixed; final re-review found no remaining issue.

2026-07-20 authenticated-handler-composition validation passed: 258 control-plane cases under the
race detector plus focused `go vet`. Coverage proves authentication precedes method rejection,
project/session routes share one trust boundary, and typed-nil dependencies fail closed. A missing
auth-before-method regression identified in review was added; final re-review found no remaining
issue.

2026-07-20 Flutter metadata-and-surface validation passed: strict opaque IDs and UTC timestamps,
bounded project/session metadata, immutable catalog selection, caller-stable ambiguous session
retry, touch catalog/session screens, and explicit metadata-only local-workspace handoff are covered
by unit and widget tests. Review findings about unbounded names and normalized invalid timestamps
were fixed and re-reviewed cleanly.

2026-07-20 Flutter transport-and-repository validation passed: focused transport/repository tests,
the full mobile suite, and `flutter analyze`. Coverage proves a pinned same-origin HTTPS boundary,
disabled redirects, immutable bounded bodies, total deadlines including late socket acquisition,
strict duplicate-aware successful JSON, fixed credential-free failures, and exact unknown-outcome
request retention for transport races and every post-dispatch 5xx. High and Medium review findings
across origin pinning, mutable data, late requests, ambiguous outcomes, duplicate keys, and proxy
503/504 retry identity were fixed; final re-reviews found no remaining issue. The controller-level
regression proves both attempts send all four identical IDs.

2026-07-20 Flutter-runtime-composition validation passed: 10 focused widget cases, 84 full mobile
tests, and `flutter analyze`. Trust-input rotation and shell replacement are covered while routes are
idle, in flight, outcome-unknown, metadata-ready, or displaying the local workspace. Old transport
state closes exactly once, stale routes are removed, and late completions cannot reopen them. Two
route-lifecycle findings were fixed; final security re-review was clean.

2026-07-20 completed-scope final validation passed: `make fmt`, `make lint`, `make test`, `make build`,
and `make test-infrastructure`. All Go modules formatted, vetted, and tested; Flutter analysis and 84
tests passed; both WebView packages type-checked, tested, and built; every Go deployable and container
image built; agent-skill mirrors were synchronized; and the PostgreSQL 17/Redis Compose smoke,
migration, repeat-start, worker, and health checks passed. Protocol contracts did not change, and no
M0 editor/terminal touch behavior changed, so protocol regeneration and physical-device touch-matrix
retesting were not applicable to this closure slice.

2026-07-20 OIDC-identity-binding slice validation passed: 54 focused auth cases normally and under
the race detector, focused `go vet`, ShellCheck, Bash syntax, the smoke-harness regression, and the
full PostgreSQL 17/Redis Compose infrastructure gate. Integration coverage proves exact
case-sensitive issuer/subject lookup, duplicate-rebind rejection, missing-user rollback recovery,
repeatable forward migration, and deterministic multi-version ledger ordering. The empty-query
issuer edge case found in adversarial review was fixed, and final security re-review was clean.

2026-07-20 OIDC-identity-adapter slice validation passed: 69 focused auth cases normally and under
the race detector, 287 full control-plane cases under the race detector, full control-plane
`go vet`, and `git diff --check`. Coverage proves exact verified issuer/subject lookup, uniform
invalid-token and unlinked-identity rejection, cancellation propagation, sanitized downstream
failure handling, and fail-closed plain/typed-nil composition. The typed-nil panic path found in
adversarial review was fixed, and final security re-review was clean.

2026-07-20 OIDC-verifier-config slice validation passed: 105 focused auth cases normally and under
the race detector, focused `go vet`, and `git diff --check`. Coverage proves bounded exact
issuer/audience/JWKS configuration, HTTPS-only endpoints, host and port validation, and an explicit
asymmetric-only algorithm allow-list. Missing-host and invalid-port findings from adversarial
review were fixed; URL size and UTF-8 limits are enforced before parsing.

2026-07-20 signed OIDC-ID-token validation passed: 178 focused auth cases and 396 full control-plane
cases under the race detector, focused and full `go vet`, `govulncheck ./...`, `go mod verify`, and
`git diff --check`. Real TLS JWKS coverage proves signature and exact issuer/audience/subject/time
validation, exact protected ID-token type, sanitized provider failures, structurally valid strong
algorithm-compatible public keys, removal of every disallowed key before verification, bounded
response I/O, and one validated URL-and-algorithm-keyed response shared across cold verifier caches.
Adversarial findings for access-token confusion, unknown-key refresh amplification, generic-JSON
misclassification, weak or mixed-strength key sets, ECDSA curve mismatch, per-instance cooldowns,
and cold-verifier starvation were fixed. A removed-key freshness bypass in the dependency's private
cache was eliminated by verifying through the shared freshness boundary on every token; rotation
coverage proves removed keys fail and replacement keys succeed after refresh, while a failed
refresh clears stale keys and remains unavailable without another cooldown fetch. Final security
re-review found no remaining actionable issue.

2026-07-20 control-plane runtime-config validation passed: 16 `cmd/api` cases under the race
detector, focused `go vet`, `docker compose config --quiet`, and `git diff --check`. Coverage proves
required OIDC/relay/database inputs, exact OIDC value retention, the explicit local listen default,
numeric port bounds, UTF-8 and control-character rejection, and no environment values in
configuration errors. Final security review found no actionable issue.

2026-07-20 control-plane runtime-composition validation passed: 34 focused `cmd/api` cases and 429
full control-plane cases under the race detector, focused and full `go vet`, `govulncheck ./...`,
`go mod verify`, ShellCheck, Bash syntax, `git diff --check`, and the full PostgreSQL 17/Redis Compose
infrastructure gate. Unit coverage proves no handler is returned for nil dependencies or invalid
OIDC/relay configuration. The new real TLS JWKS plus PostgreSQL integration path verifies an exact
linked identity, returns its owner-scoped project, starts and persists metadata-only session state,
and exercises the same runtime handler used by the deployable. Final security re-review found no
remaining blocking or medium issue.
The Compose gate rebuilds the production image and validates required environment parsing, process
startup, migrations, and public health. Authenticated traffic through the container remains a
manual acceptance check against the real registered provider because the repository intentionally
does not ship a private-key-bearing identity-provider fixture.

2026-07-20 mobile OIDC composition and automated M2 closure validation passed: 117 Flutter tests,
`flutter analyze`, Android debug APK build, iOS simulator debug app build, `make fmt`, `make lint`,
`make test`, `make build`, and the full PostgreSQL 17/Redis Compose infrastructure gate. Coverage
proves invalid build inputs expose only a fixed unavailable state, authentication must restore or
complete before the control-plane runtime exists, provider failures remain sanitized, and logout
removes and disposes authenticated transport state. The production entry point now owns the native
public-client OIDC flow, secure token lifecycle, registered-device bootstrap selector, and
authenticated M2 shell; final security review found no actionable P1/P2 issue. Protocol contracts
and M0 touch behavior did not change. The exact real-provider redirect, linked identity, active
device ownership, and authenticated physical-device route remain the manual acceptance boundary.

## Recovery and rollback

Code slices remain independent commits on the M2 branch. Forward migrations must be compatible
with the previously deployed binary or include an explicit staged rollout. Failed local database
tests use the isolated Compose project and remove disposable volumes. Runtime rollback never
deletes or rewrites durable user data automatically.

Any local database that ran the earlier pre-merge idempotent starter SQL or its ledger checksums
must be recreated before continuing. For the disposable M1 Compose database, stop the stack and
remove its volume with `docker compose -f deployments/compose/docker-compose.yml down --volumes`,
then rerun the migrations. This reset is only valid before durable M2 application data exists; it is
not a production rollback procedure.

## Open risks

- A real OIDC provider/client registration and linked bootstrap user are still required before
  production login and the container-level authenticated-route acceptance check can be exercised;
  Compose intentionally requires explicit trust anchors.
- Real-provider physical-device acceptance still requires a pre-registered active device ID owned
  by the linked user. M3 replaces this explicit build-time bootstrap selector with pairing and
  pinned device identity.
- Public-key fingerprint encoding must be fixed with the M3 pairing contract before device
  persistence is exposed to clients.
- Session signing and relay ticket cryptography belong to M3/M4; M2 must not ship a placeholder
  token that could be mistaken for a secure production ticket.
