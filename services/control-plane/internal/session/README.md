# `session` module

This module is the only writer to the `session` PostgreSQL schema.

The session domain records only bounded authorization metadata for one authenticated owner, device,
agent, project, relay region, and server-owned start time. A relay region is a canonical lowercase
ASCII label selected by trusted server configuration; it is not accepted as user routing authority.

A `Session` is not a relay ticket and grants no access by itself. It contains no signature, nonce,
expiry claim, E2E key, pairing secret, source code, terminal data, prompt, or other engineering
payload. The application service must authorize the persisted device, exact agent, and agent-bound
project through their owning modules inside one bounded PostgreSQL transaction before this module
may persist the record. Ticket signing, relay validation, replay protection, and endpoint pairing
remain M3/M4 responsibilities and must not be replaced with an unsigned M2 token.

`Repository.Create` accepts only an existing `pgx.Tx`, inserts the validated metadata under a fixed
maximum deadline, and never begins, commits, or rolls back the caller's transaction. This allows the
application service to make device, agent, and project authorization plus session insertion one
atomic decision. Invalid or canceled input performs no SQL, duplicate session IDs return a typed
conflict, and rollback removes the inserted row.

`Service.Start` requires the caller to generate one canonical opaque session ID and reuse it for
every retry of the same request. The ID is an idempotency identifier, not an authorization
capability. Inside one bounded transaction, the service authorizes the active persisted device,
active persisted agent, and exact agent-bound project in that order, then calls
`Repository.CreateOrGet`. An existing ID is returned only when its owner, device, agent, project,
and server-selected region exactly match; foreign or mismatched reuse fails closed. A commit error
returns `ErrSessionCommitOutcomeUnknown`, so callers must retry the same ID and inputs instead of
creating a second session. Session insertion emits no outbox event because M2 creates no relay
credential or other external side effect.

M3 migration version 2 replaces the unused pairing-attempt starter shape with session-owned,
bounded bootstrap, candidate, claim, confirmation, and consumption metadata. Pre-M3 rows cannot be
authenticated because they contain no bootstrap-credential hash, so the forward migration deletes
them transactionally instead of upgrading them into trusted attempts. The schema stores only a
domain-separated 32-byte bootstrap-credential hash; the raw credential, pairing secret, Noise
plaintext, private keys, and engineering payloads never enter PostgreSQL.

Database constraints provide defense in depth for canonical IDs/fingerprints, 32-byte public keys
and channel bindings, the five-minute maximum lifetime, bounded failure count, complete optional
claim groups, matching endpoint bindings, and explicit `open`/`claimed`/`confirming`/`consumed`
state shapes. Application services remain responsible for recomputing fingerprints, authenticating
the bootstrap credential, enforcing expiry on every access, and performing state transitions under
a row lock; those rules must not be inferred from cross-schema SQL.

The open-attempt domain constructor rejects noncanonical identifiers, zero or malformed public
keys, unbounded/control/bidirectional/format-separator metadata, unsupported protocol versions,
invalid relay regions, zero or wrong-sized credential hashes, and PostgreSQL-canonical lifetimes
outside `(0, 5 minutes]`. It computes the canonical agent fingerprint from the copied public key
instead of accepting fingerprint text.

`CreatePairingAttempt` persists only validated constructor output in a caller-owned transaction.
`LockOpenPairingAttempt` samples the repository clock before and after `FOR UPDATE`, and returns the
same generic unavailable result for missing, future, expired, exhausted, non-open, noncanonical, or
partially populated rows. The caller retains the row lock until its transaction commits or rolls
back.

`HashPairingBootstrapCredential` hashes an exact 256-bit credential with a versioned domain and the
canonical pairing ID, so a stored digest cannot be substituted between attempts. The raw credential
remains caller-owned and is never persisted. `authenticateOpenPairingAttempt` locks and restores the
open attempt, compares the derived digest in constant time, and rechecks expiry after hashing. The
helper is package-private and reports a wrong, zero, or malformed credential as a normal rejected
outcome after incrementing the bounded failure count. The package-private pairing-attempt service
owns the transaction, commits that rejection through a short cancellation-independent context, and
only then returns the same unavailable result used by missing, expired, and exhausted attempts. An
ambiguous commit grants no authentication and returns a distinct reconciliation error. A successful
check is deliberately not exposed as an authorization capability; the confirmation slice must keep
verification and its typed mutation under this same row lock and transaction. Clock rollback,
cancellation before mutation, exhausted attempts, and expiry fail closed.
