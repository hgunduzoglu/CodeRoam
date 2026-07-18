# `workspace` module

This module is the only writer to the `workspace` PostgreSQL schema.

The agent domain binds a canonical opaque agent ID, bounded display name and software version, and
an initialized X25519 public key to an authenticated owning actor. New in-memory agents authorize
only that owner while active. Revocation is owner-only, irreversible, idempotent, and shared by
retained value copies so an old handle cannot authorize after revocation.

Creation and revocation times are server-owned inputs; transport-provided timestamps or user IDs
must not reach the constructor as authority. Persisted agent authorization re-reads committed
workspace state inside the caller's existing bounded PostgreSQL transaction, rejects future-created
or revoked agents, and holds a shared row lock until that caller commits or rolls back.

Persisted agent revocation re-reads and locks the row by both agent ID and authenticated owner ID,
uses a server-owned clock, and commits `revoked_at` with one metadata-only `agent.revoked.v1` outbox
event containing `{}` in the same PostgreSQL transaction. Missing, foreign-owned, and
malformed-owner rows share the same access-denied result. Repeated revocation preserves the first
timestamp and emits no second event; an update or outbox failure before commit rolls back the state
change. A commit error has an unknown outcome and must be retried: the retry returns success without
a second event if the transaction committed, or performs the atomic revocation if it did not.

The persistence boundary ignores fingerprint and last-seen metadata until their M3 contracts are
defined; it does not treat a size-valid public key as pairing proof. Persisted authorization does
not begin, commit, or roll back the caller's transaction. Future session issuance must authorize
the persisted device, agent, and project and write ticket metadata inside that exact transaction.

The environment domain binds a canonical opaque environment ID and bounded display/provider
metadata to an authenticated owner and an active agent already owned by that actor. The provider
label is metadata only and never decides authorization. New environments cannot reference a zero,
foreign-owned, or revoked agent.

Environment ownership remains stable if the linked agent is revoked later. `Environment.OwnedBy`
proves only the single-owner relationship; it is not an agent-status or session-authorization check.
Future persisted creation must re-read committed agent ownership/status, and session issuance must
separately authorize the current agent and project in one bounded transaction.

The project domain binds a canonical opaque project ID, bounded display name, and one canonical
absolute POSIX root-path value to an environment already owned by the authenticated actor. The
filesystem root itself is rejected because a project registration must narrow authority. Project
creation cannot predate its environment.

The registered root is control-plane metadata, not proof that a path is safe to access. It does not
resolve symlinks, defend against filesystem races, or authorize a runtime request. The M5 agent
must confine project-relative paths beneath the persisted root using filesystem-safe operations.
Project ownership remains stable after agent revocation, while future session issuance must still
authorize the persisted project and active agent separately in one bounded transaction.

Optional repository URL and last-opened metadata remain deferred until their update and credential
handling contracts are defined.
