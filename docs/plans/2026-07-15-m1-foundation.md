# Milestone 1 Foundation

## Objective

Complete Milestone 1 with a reproducible monorepo foundation: CI coverage, buildable Go
deployables, a bootable Flutter shell, PostgreSQL and Redis development infrastructure, generated
Protobuf contracts, and a narrow cryptography boundary ready for later pairing and Noise work.

## Scope

This plan covers foundation behavior only. It does not implement control-plane business modules,
agent pairing, Noise handshakes, relay routing, or workspace operations assigned to later
milestones.

## Outcome

Milestone 1 acceptance passed on 2026-07-17. The repository now has reproducible local and hosted
checks, buildable foundation deployables and Flutter/WebView surfaces, health-tested development
infrastructure, deterministic Protobuf generation, and the reviewed cryptography boundary required
for later pairing and Noise milestones.

## Current state

- Milestone 0 is complete and merged.
- The monorepo already contains the Flutter app, both WebViews, four Go deployables, database
  migrations, Dockerfiles, a Compose stack, Protobuf schemas, and generated Go/Dart contracts.
- CI validates Go, Flutter, both WebViews, Protobuf compatibility/regeneration, infrastructure
  readiness/migrations, and agent-guidance synchronization in independent bounded jobs.
- `packages/go/cryptox` now validates X25519 public-key encodings and exposes opaque static
  identity/provider contracts without exposing private-key bytes.
- Compose defines PostgreSQL, Redis, control-plane, worker, and relay with health-gated startup;
  automated smoke coverage verifies readiness, repeated migrations, service health, and cleanup.

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
- [x] Add infrastructure readiness and migration smoke coverage. PostgreSQL/Redis health-gated
      startup, repeated migrations, service health, worker lifecycle, and cleanup are validated.
- [x] Add Protobuf regeneration checks. The repository-owned check lints schemas, compares them
      with `main`, regenerates Go and Dart code, and rejects generated drift.
- [x] Expand CI coverage. Independent bounded jobs now validate Go, Flutter, both WebViews,
      Protobuf compatibility/regeneration, infrastructure readiness/migrations, and agent guidance.
- [x] Run the complete applicable quality gate.
- [x] Record the M1 decision and remaining manual checks.

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
- 2026-07-15: Keep parsed public keys opaque, and separate identity loading from explicit atomic
  creation. Normal startup must fail closed instead of regenerating identity after missing or
  unreadable storage, because replacement would break peer pinning. The opaque key's zero value
  cannot be encoded as usable material; callers must obtain initialized keys through parsing.
- 2026-07-15: Make public-key values deliberately non-comparable and provide fail-closed equality.
  This prevents two uninitialized zero values from accidentally satisfying a peer-pinning check.
- 2026-07-15: Gate local service startup on PostgreSQL and Redis health. Application containers
  remain distroless; their HTTP readiness will be checked externally by the integration smoke
  runner instead of adding shell tooling to production images.
- 2026-07-16: Use `.git#branch=main` as the default Buf compatibility baseline, with an environment
  override for CI or unusual local Git layouts. Keep compatibility and generated-drift checks in a
  repository-owned command so local and hosted validation use the same behavior.
- 2026-07-17: Give each CI concern its own timeout-bounded job and call repository-owned check
  targets. Pin Node 24, Flutter 3.41.5, Dart 3.11.3, and Buf 1.47.2; use the maintained Buf action in
  setup-only mode so validation policy remains in `make proto-check` and CI does not publish schemas.
- 2026-07-17: Accept Milestone 1 after the complete local gate, a clean detached-worktree bootstrap
  and build, generated/format cleanliness, infrastructure cleanup, and all hosted CI jobs passed.
  Pairing, concrete identity storage, Noise handshakes, and encrypted frame state remain assigned
  to M3/M4 rather than being pulled into the foundation.

## Validation

Focused validation runs after each slice. The final gate will include:

```bash
make bootstrap
make proto
make fmt
make lint
make test
make build
make test-infrastructure
```

Generated-code cleanliness and a repeated migration run will also be checked explicitly.

2026-07-16 infrastructure-slice validation passed: ShellCheck and Bash syntax, Compose config,
health-gated startup, two migration runs, control-plane and relay health, worker lifecycle, cleanup,
`make fmt`, `make lint`, `make test`, and `make build`.

2026-07-16 protocol-slice validation passed: Buf lint and compatibility against `main`, clean Go
and Dart regeneration, ShellCheck and Bash syntax, the protocol-check regression test, all Go
consumer tests, Flutter analysis, and all Flutter tests.

2026-07-17 CI-slice local validation passed: actionlint 1.7.12, WebView formatting/lint/tests/build,
Flutter lockfile/format/analyze/tests and Android debug build, protocol regression/compatibility/
regeneration, and the complete isolated infrastructure smoke test. All hosted GitHub Actions jobs
subsequently passed.

2026-07-17 final M1 validation passed: `make bootstrap`, `make proto`, `make fmt`, `make lint`,
`make test`, `make build`, and `make test-infrastructure`. Regeneration and formatting left the
tracked tree clean. A detached worktree with no inherited dependencies or build products also
passed bootstrap, `npm ci` (zero reported vulnerabilities), protocol generation, WebView checks,
Flutter checks and Android debug build, all Go/WebView/container builds, and a final clean status.
The disposable Compose project left no containers or named volume behind.

## Recovery and rollback

Each slice remains a focused diff. If a slice fails validation, revert only that slice's files and
retain the last passing foundation state. Local Compose resources are stopped with `make down`;
volumes are preserved unless a migration test explicitly uses an isolated disposable project.

## Open risks

- The Android debug build succeeds but its dependency toolchain emits deprecation warnings for
  Java 8 source/target compatibility. This is non-blocking for M1 and should be revisited during a
  deliberate Flutter/Android dependency upgrade rather than mixed into foundation acceptance.
