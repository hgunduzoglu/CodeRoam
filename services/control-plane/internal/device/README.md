# `device` module

This module is the only writer to the `device` PostgreSQL schema.

Device registration and revocation require an authenticated `auth.Actor`. The domain stores only an
initialized X25519 public key, never private identity material. Authorization requires the exact
owning actor and an active device; revocation is irreversible and idempotent in memory.
Copied device handles share one synchronized revocation state, so a retained copy cannot remain
active after another handle revokes it.

Persisted authorization runs inside the caller's existing `pgx.Tx`, queries by both canonical device
ID and authenticated owner ID, requires `revoked_at IS NULL`, and holds a shared row lock until the
caller commits or rolls back. A future session service must write ticket metadata in that same
bounded transaction, so either issuance commits before revocation or revocation wins and the check
fails. Stored pairing time, bounded name, platform, and X25519 public-key data are revalidated through
the device domain constructor. Missing, foreign, revoked, future, or corrupt rows all return the same
access-denied result. Authorization makes no mutation or outbox write and does not treat the
unresolved fingerprint field as pairing proof.

Persisted revocation re-reads and locks the row by both device ID and authenticated owner ID, uses a
server-owned clock, and commits `revoked_at` with one metadata-only `device.revoked.v1` outbox event
in the same PostgreSQL transaction. Missing, foreign-owned, and malformed-owner rows share the same
access-denied result. Repeated revocation preserves the first timestamp and emits no second event;
an update or outbox failure before commit rolls back the state change. A commit error has an unknown
outcome and must be retried: the retry re-reads the row, returns success without a second event if the
transaction committed, or performs the atomic revocation if it did not.

Paired-device listing is a transaction-owning, owner-scoped management read. It returns at most 100
records ordered by pairing time and ID, includes revoked devices so clients can explain their state,
and exposes the canonical fingerprint but never the static public key. Every stored identity and
timestamp is revalidated, including recomputing the fingerprint from the hidden key; one corrupt or
future row fails the whole read instead of returning a partial trust view. Listing does not establish
that a device is active: trust-sensitive callers must still use `Authorize`.

M3 migration version 2 takes an exclusive table lock and rejects malformed, noncanonical,
low-order, or duplicate legacy X25519 public keys before changing data. This prevents RFC 7748
input aliases from creating multiple durable identities for the same DH point. It backfills the
canonical fingerprint from each validated key and constrains all later key/fingerprint writes to
the same representation. A failed preflight or constraint installation rolls back both the
backfill and migration ledger entry. Migration version 3 adds the owner/pairing-time/ID index used
by the bounded paired-device listing query. Transport-provided timestamps or user identifiers must
not bypass the application service. Callers must bound and finish authorization transactions;
returning from `Authorize` does not release its row lock.
