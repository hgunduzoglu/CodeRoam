# `outbox` module

This module is the only writer to the `outbox` PostgreSQL schema.

`Enqueue` requires the caller's existing `pgx.Tx`, so a domain state change and its durable event can
commit or roll back together. The initial contract exposes a closed event-kind allowlist, generates
each event ID cryptographically, requires a typed aggregate ID, and writes a fixed empty JSON object.
It exposes no free-form event type or payload field. Callers must still pass only domain-owned IDs,
never raw transport data, credentials, prompts, source, or terminal data.

Duplicate event IDs return a typed error and leave the PostgreSQL transaction aborted, so callers
must roll it back. New event kinds or typed payloads require an explicit reviewed constructor;
claiming, retries, and completion bookkeeping are separate slices and must remain bounded and
idempotent.
