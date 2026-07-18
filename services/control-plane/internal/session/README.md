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
