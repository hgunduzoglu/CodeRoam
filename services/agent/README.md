# CodeRoam workspace agent

Non-root remote workspace executor and outbound relay client.

The M3 identity provider keeps the agent's stable X25519 key in a dedicated `0700` directory and an
owner-only file. Identity creation is explicit, atomic, and exclusive; normal loading fails closed
for missing, corrupt, replaced, symlinked, or insecurely permissioned material. A failed load never
creates or rotates the identity.

Linux release builds resolve the configured physical prefix, walk and validate every ancestor from
the filesystem root, and anchor reads and publication to a no-follow directory descriptor. The
walk verifies ownership, rejects unsafe write permissions and POSIX ACLs, and permits only
root-owned sticky shared directories. macOS development builds apply the same checks with libc ACL
inspection and therefore require CGO. A fixed, fsynced pending record lets explicit initialization
recover the same key after a crash; invalid pending state blocks generation instead of rotating the
identity. Concurrent initialization uses a context-aware exclusive lock.

Linux release artifacts, checksum/provenance verification, and the non-root installation boundary
are documented in [`docs/agent-release.md`](../../docs/agent-release.md). The release pipeline never
publishes pull-request code and does not add autonomous update or privileged runtime behavior.
