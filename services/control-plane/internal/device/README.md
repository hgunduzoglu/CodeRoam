# `device` module

This module is the only writer to the `device` PostgreSQL schema.

Device registration and revocation require an authenticated `auth.Actor`. The domain stores only an
initialized X25519 public key, never private identity material. Authorization requires the exact
owning actor and an active device; revocation is irreversible and idempotent in memory.
Copied device handles share one synchronized revocation state, so a retained copy cannot remain
active after another handle revokes it.

Persistence, public-key fingerprint encoding, and the atomic revocation outbox event are separate
slices. Persisted authorization must re-read committed revocation state rather than trusting a
cached domain handle. Transport-provided timestamps or user identifiers must not bypass the
application service.
