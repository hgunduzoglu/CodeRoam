# `auth` module

This module is the only writer to the `auth` PostgreSQL schema.

Authentication evidence is attacker-controlled. Only the approved `IdentityVerifier` adapter may
translate it to a user ID, and the application service must resolve that ID through this module's
repository before issuing an `Actor`. Transport code must never construct an actor from a request
header or raw user ID.
