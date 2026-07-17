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
- The control plane currently exposes only `/healthz`; no application modules are wired at runtime.
- Starter SQL creates the module-owned schemas and tables, but there are no repositories,
  transaction boundaries, migration ledger, or database integration tests.
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
- [ ] Approve and add the PostgreSQL runtime dependency.
- [ ] Add migration/repository integration coverage.
- [ ] Implement auth and authenticated actor behavior.
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

## Recovery and rollback

Code slices remain independent commits on the M2 branch. Forward migrations must be compatible
with the previously deployed binary or include an explicit staged rollout. Failed local database
tests use the isolated Compose project and remove disposable volumes. Runtime rollback never
deletes or rewrites durable user data automatically.

## Open risks

- The authentication bootstrap/provider is not specified and requires an explicit product decision
  before a production login endpoint can be completed.
- PostgreSQL access requires a runtime driver dependency; repository policy requires approval before
  it is added.
- Existing starter migrations use `IF NOT EXISTS` without a migration ledger, so future schema
  evolution needs a versioned, failure-aware migration path before table changes proceed.
- Session signing and relay ticket cryptography belong to M3/M4; M2 must not ship a placeholder
  token that could be mistaken for a secure production ticket.
