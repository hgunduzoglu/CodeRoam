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
- [ ] Implement devices and revocation outbox behavior.
- [ ] Implement agents, environments, and projects.
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
- Session signing and relay ticket cryptography belong to M3/M4; M2 must not ship a placeholder
  token that could be mistaken for a secure production ticket.
