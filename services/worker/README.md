# CodeRoam worker

Transactional-outbox and scheduled-job worker.

The worker accepts only the closed metadata-only `device.revoked.v1` and `agent.revoked.v1` event
contracts. Event and aggregate IDs must be canonical opaque IDs, the event/aggregate kind pair must
match, the JSON payload must be exactly empty, and availability/attempt metadata must be valid.
Source code, terminal output, prompts, credentials, secret values, or other engineering payloads are
never valid worker input.

The outbox repository claims at most one due row through a caller-owned `pgx.Tx` using
`FOR UPDATE SKIP LOCKED`. Text fields are bounded in SQL before crossing the process boundary, and
the transaction-local row locator lets malformed rows be discarded without loading an unbounded
primary key. `Finish` accepts only closed completed, retry, or discard outcomes; increments attempts
without overflowing PostgreSQL `integer`; stores only fixed safe failure classifications; and never
commits or rolls back the caller's transaction. Retry scheduling uses a fixed bounded delay.

`Processor.ProcessNext` owns one bounded claim/handler/finish transaction. Handlers receive only the
closed event kind and opaque aggregate ID, must honor the supplied deadline, and must be idempotent:
a crash after an external side effect but before the database commit deliberately causes duplicate
delivery. Retryable failures are delayed, permanent failures and exhausted attempts are terminal,
and commit errors return an explicit unknown-outcome result. Handler error text is never persisted.
The runtime opens and verifies a bounded PostgreSQL pool, drains available events one transaction at
a time, backs off after an empty claim or failure, and closes the pool after graceful cancellation.
`WORKER_PROCESSING_ENABLED=false` is reserved for isolated database-test orchestration; unset or
`true` enables processing, and other values fail startup. M2 acknowledges revocations because relay
sessions do not exist yet. M4 must replace that acknowledgement with active-session termination
before it enables relay sessions.
