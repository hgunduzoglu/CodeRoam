# `workspace` module

This module is the only writer to the `workspace` PostgreSQL schema.

The agent domain binds a canonical opaque agent ID, bounded display name and software version, and
an initialized X25519 public key to an authenticated owning actor. New in-memory agents authorize
only that owner while active. Revocation is owner-only, irreversible, idempotent, and shared by
retained value copies so an old handle cannot authorize after revocation.

Creation and revocation times are server-owned inputs; transport-provided timestamps or user IDs
must not reach the constructor as authority. Persisted authorization must later re-read committed
workspace state and reject future-created or revoked agents rather than trusting an in-memory
handle.

This domain boundary does not define fingerprint encoding, signed-agent bootstrap, pairing proof,
key pinning, persistence, last-seen updates, or relay/session behavior. Those require their owning
M2 persistence and M3/M4 security slices.

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
