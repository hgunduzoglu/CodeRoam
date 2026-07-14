# CodeRoam Code Review Standard

Lead with findings in this order:

1. Pairing, authentication, authorization, revocation, or ticket bypass.
2. Relay plaintext exposure, replay, unbounded buffers, or routing confusion.
3. Filesystem escape, command injection, secret exposure, or unsafe process lifecycle.
4. Data loss in file save or Offline Draft conflict behavior.
5. Protocol compatibility, ordering, ACK, flow-control, reconnect, or resume defects.
6. PostgreSQL ownership, transaction, or outbox errors.
7. Touch UX regressions involving selection, IME, keyboard, focus, or accessibility.
8. Race conditions, leaks, performance regressions, and missing negative tests.

Each finding includes severity, location, concrete failure/exploit path, smallest safe fix, and a
negative test. Do not add style-only noise.

Project checks:

- Runtime path remains client ⇄ relay ⇄ agent.
- Relay remains payload-opaque.
- No organization/policy scope enters MVP.
- Offline Drafts do not silently overwrite changed remote content.
- AI cannot push/deploy/run production actions.
- Engineering payloads and secrets are absent from storage/logging.
