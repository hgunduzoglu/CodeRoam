# `device` module

This module is the only writer to the `device` PostgreSQL schema.

Device registration and revocation require an authenticated `auth.Actor`. The domain stores only an
initialized X25519 public key, never private identity material. Authorization requires the exact
owning actor and an active device; revocation is irreversible and idempotent in memory.
Copied device handles share one synchronized revocation state, so a retained copy cannot remain
active after another handle revokes it.

Persisted revocation re-reads and locks the row by both device ID and authenticated owner ID, uses a
server-owned clock, and commits `revoked_at` with one metadata-only `device.revoked.v1` outbox event
in the same PostgreSQL transaction. Missing, foreign-owned, and malformed-owner rows share the same
access-denied result. Repeated revocation preserves the first timestamp and emits no second event;
an update or outbox failure before commit rolls back the state change. A commit error has an unknown
outcome and must be retried: the retry re-reads the row, returns success without a second event if the
transaction committed, or performs the atomic revocation if it did not.

Device creation/listing persistence and public-key fingerprint encoding remain separate slices.
Fingerprint encoding must be fixed with the M3 pairing contract before registration is exposed.
Persisted authorization must re-read committed revocation state rather than trusting a cached domain
handle. Transport-provided timestamps or user identifiers must not bypass the application service.
