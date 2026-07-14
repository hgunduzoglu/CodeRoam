# CodeRoam Worker Guidance

- Consume outbox and scheduled cleanup; do not become a second domain owner.
- Jobs are idempotent, bounded, retry-safe, and observable.
- Revocation, expiry, GitHub, and notification work must not contain engineering payloads.
- Test duplicate delivery, crash-after-side-effect, timeout, and permanent failure.
