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

It does not implement organizations, teams, memberships, generic roles/policies, agent bootstrap,
pairing handshakes, relay routing, Noise sessions, workspace filesystem/PTY operations, or GitHub
integration. Those remain assigned to later milestones.

## Current state

- M0 and M1 are complete and merged.
- The control plane currently exposes only `/healthz`; no authentication provider or transport is
  wired at runtime.
- Starter SQL runs through the transactional migration ledger, and the auth module now owns a
  PostgreSQL repository for validated users with typed duplicate and not-found failures.
- The auth application service accepts bounded opaque evidence only through an injected verifier,
  re-resolves the verified user in PostgreSQL, and issues an opaque actor on exact identity match.
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
  an unsigned placeholder relay ticket or carrying secrets and engineering payloads, and can insert
  that metadata only inside a caller-owned PostgreSQL transaction.
- The control-plane runtime opens and verifies a bounded PostgreSQL pool before serving, drains HTTP
  requests during shutdown, and closes the pool after the server stops.
- The worker emits a heartbeat but does not claim or process outbox events.
- Authentication-provider/bootstrap behavior is not specified, so no provider-specific login flow
  will be invented inside an ownership slice.

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
4. Implement device registration/listing/revocation with atomic outbox publication.
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
- [ ] Wire an approved authentication adapter and REST/OpenAPI behavior.
- [x] Add the device identity and revocation domain contract.
- [x] Add the metadata-only transactional outbox enqueue primitive.
- [x] Implement persisted device revocation with atomic outbox behavior.
- [x] Add the persisted active-device authorization boundary.
- [ ] Implement device registration/listing persistence after the fingerprint contract is fixed.
- [ ] Implement agents, environments, and projects.
- [x] Add the workspace agent identity and revocation domain boundary.
- [x] Add the persisted active-agent authorization boundary.
- [x] Implement persisted agent revocation with atomic outbox behavior.
- [x] Add the workspace environment ownership domain boundary.
- [x] Add the workspace project ownership and registered-root metadata domain boundary.
- [x] Add the persisted owner- and agent-bound project authorization boundary.
- [x] Add the bounded session metadata domain boundary.
- [x] Persist validated session metadata inside a caller-owned transaction.
- [ ] Implement session authorization and ticket metadata.
- [ ] Implement worker outbox processing.
- [ ] Integrate the Flutter M2 surface.
- [ ] Run final validation and record M2 acceptance.

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
- 2026-07-19: Define session metadata as one authenticated owner plus canonical session, device,
  agent, and project IDs, a bounded canonical server-selected relay-region label, and a server-owned
  start time. Keep every field private and expose no serialization or credential surface. A session
  domain value is not a relay ticket and contains no signature, nonce, expiry claim, E2E material,
  pairing secret, or engineering payload. The application layer must still perform all persisted
  authorization and storage in one transaction; M3/M4 retain ticket signing, replay protection,
  relay validation, and endpoint pairing.
- 2026-07-19: Persist validated session metadata only through an existing `pgx.Tx`, under a fixed
  maximum deadline, without letting the repository begin or finish the transaction. Map the
  session primary-key conflict to a typed duplicate result, keep invalid/canceled calls SQL-free,
  and preserve rollback/commit visibility. Emit no outbox event because this slice creates no relay
  credential or external side effect. The next application-service slice must perform device,
  agent, and exact project authorization before calling `Create` in this same transaction.

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

- The authentication bootstrap/provider is not specified and requires an explicit product decision
  before a production login endpoint can be completed.
- Public-key fingerprint encoding must be fixed with the M3 pairing contract before device
  persistence is exposed to clients.
- Session signing and relay ticket cryptography belong to M3/M4; M2 must not ship a placeholder
  token that could be mistaken for a secure production ticket.
