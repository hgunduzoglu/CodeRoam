# Milestone 1 Foundation

## Objective

Complete Milestone 1 with a reproducible monorepo foundation: CI coverage, buildable Go
deployables, a bootable Flutter shell, PostgreSQL and Redis development infrastructure, generated
Protobuf contracts, and a narrow cryptography boundary ready for later pairing and Noise work.

## Scope

This plan covers foundation behavior only. It does not implement control-plane business modules,
agent pairing, Noise handshakes, relay routing, or workspace operations assigned to later
milestones.

## Current state

- Milestone 0 is complete and merged.
- The monorepo already contains the Flutter app, both WebViews, four Go deployables, database
  migrations, Dockerfiles, a Compose stack, Protobuf schemas, and generated Go/Dart contracts.
- CI currently validates Go and agent-guidance synchronization, but not Flutter, WebViews,
  Protobuf regeneration, or the local infrastructure smoke path.
- `packages/go/cryptox` now validates X25519 public-key encodings and exposes opaque static
  identity/provider contracts without exposing private-key bytes.
- Compose defines PostgreSQL, Redis, control-plane, worker, and relay, but readiness and migration
  behavior are not exercised by an automated smoke test.

## Design

Work proceeds in small independently verified slices. Each code slice adds no more than a few
functions or methods, includes focused tests, and is validated before the next slice begins.

The cryptography package establishes contracts only. Private identity material is the protected
asset. Storage and later protocol implementations cross explicit interfaces and may fail. The
foundation must never invent cryptographic primitives, fall back to weak randomness, log secrets,
or silently replace an identity. Noise XXpsk3 and IK implementations remain deferred to M3/M4.

CI will call repository-owned commands so local and hosted validation do not diverge. Integration
smoke checks will validate service readiness and migrations without adding M2 domain behavior.

## Milestones

1. Add the minimal cryptography/key-storage contracts and unit tests.
2. Make infrastructure readiness and migration behavior observable and integration-testable.
3. Add deterministic Protobuf regeneration and compatibility validation.
4. Expand CI to cover Go, Flutter, WebViews, Protobuf, and infrastructure smoke checks.
5. Verify clean-clone bootstrap/build behavior and document the M1 acceptance results.

## Progress

- [x] Inspect the existing M1 foundation and identify gaps.
- [x] Add cryptography boundary contracts and tests.
- [ ] Add infrastructure readiness and migration smoke coverage.
- [ ] Add Protobuf regeneration checks.
- [ ] Expand CI coverage.
- [ ] Run the complete applicable quality gate.
- [ ] Record the M1 decision and remaining manual checks.

## Decisions

- 2026-07-15: Reuse the existing monorepo and fill verified gaps rather than recreate foundation
  code.
- 2026-07-15: Keep Noise and concrete identity persistence out of M1; define only the contracts
  required to implement those safely in later milestones.
- 2026-07-15: Deliver in small test-backed slices, following the user's requested limit of roughly
  two or three functions per implementation turn.
- 2026-07-15: Do not expose frame-encryption cipher states in M1. Security review showed that a
  premature wrapper could permit shared directional state, concurrent nonce misuse, and optional
  authenticated metadata. M1 exposes only public-key validation and opaque identity ownership;
  M4 will introduce frame encryption together with canonical metadata and audited Noise state.

## Validation

Focused validation runs after each slice. The final gate will include:

```bash
make bootstrap
make proto
make fmt
make lint
make test
make build
docker compose -f deployments/compose/docker-compose.yml up --build --wait
make migrate
docker compose -f deployments/compose/docker-compose.yml down
```

Generated-code cleanliness and a repeated migration run will also be checked explicitly.

## Recovery and rollback

Each slice remains a focused diff. If a slice fails validation, revert only that slice's files and
retain the last passing foundation state. Local Compose resources are stopped with `make down`;
volumes are preserved unless a migration test explicitly uses an isolated disposable project.

## Open risks

- CI duration may require separating fast pull-request checks from the container smoke job.
- Flutter platform folders are locally generated and may require a CI-specific bootstrap path.
- Buf compatibility needs an explicit baseline strategy that works for both pull requests and the
  initial repository history.
- Docker may be unavailable in some local environments; CI must remain the authoritative
  infrastructure smoke environment when that occurs.
