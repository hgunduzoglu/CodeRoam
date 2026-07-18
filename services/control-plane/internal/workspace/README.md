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
