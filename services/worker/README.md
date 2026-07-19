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
