# CodeRoam worker

Transactional-outbox and scheduled-job worker.

The worker accepts only the closed metadata-only `device.revoked.v1` and `agent.revoked.v1` event
contracts. Event and aggregate IDs must be canonical opaque IDs, the event/aggregate kind pair must
match, the JSON payload must be exactly empty, and availability/attempt metadata must be valid.
Source code, terminal output, prompts, credentials, secret values, or other engineering payloads are
never valid worker input.
